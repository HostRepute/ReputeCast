package demo_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestTransactionalDemo checks Acme Cloud's transactional mail against the
// Transactional boards (design expansion brief, §4).
func TestTransactionalDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	base := "/workspaces/acme-newsletter/transactional"

	// The templates over 7 days: sent, delivered, bounced, median, version.
	var list struct {
		Items []struct {
			Key          string
			LiveVersion  int
			DraftChanges bool
			DraftVersion int
			Attachments  bool
			Stats        struct {
				Sent, Delivered, Bounced, Deferred, Complained int64
				DeliveryRate, BounceRate                       float64
				MedianMs                                       int64
			}
		}
	}
	c.Do("GET", base+"/templates?period=7d", nil).MustStatus(t, 200).JSON(t, &list)
	// Earn's eight member emails (phase 6) are in the list too, in their
	// Memberships category; the boards' numbers are Acme Cloud's six.
	items := list.Items[:0]
	for _, it := range list.Items {
		if !strings.HasPrefix(it.Key, "member-") && !strings.HasPrefix(it.Key, "gift-") {
			items = append(items, it)
		}
	}
	list.Items = items
	type row struct {
		version         int
		sent            int64
		delivered       string // as the board rounds it
		bounced         int64
		median          string
		draft, attached bool
	}
	want := map[string]row{
		"password-reset": {3, 3212, "99.9", 3, "1.4", false, false},
		"magic-link":     {2, 6881, "99.8", 12, "1.2", false, false},
		"welcome":        {5, 1904, "99.6", 7, "1.9", false, false},
		"invoice-paid":   {4, 412, "100.0", 0, "2.3", false, true},
		"trial-ending":   {1, 71, "98.6", 1, "2.0", true, false},
		"usage-summary":  {6, 1240, "99.4", 4, "1.6", false, false},
	}
	if len(list.Items) != len(want) {
		t.Fatalf("templates: %+v", list.Items)
	}
	for _, it := range list.Items {
		w, ok := want[it.Key]
		got := row{it.LiveVersion, it.Stats.Sent, fmt1(it.Stats.DeliveryRate), it.Stats.Bounced, fmt1(float64(it.Stats.MedianMs) / 1000), it.DraftChanges, it.Attachments}
		if !ok || got != w {
			t.Fatalf("%s: got %+v, want %+v", it.Key, got, w)
		}
		if it.Key == "trial-ending" && it.DraftVersion != 2 {
			t.Fatalf("trial-ending's draft is v%d", it.DraftVersion)
		}
		if it.Key == "usage-summary" && it.Stats.Complained != 1 {
			t.Fatalf("usage-summary complaints: %d", it.Stats.Complained)
		}
	}

	// The header: 13,720 from the templates and the SMTP relay's receipt.
	var ov struct {
		Stats struct {
			Sent, Delivered, Bounced, Deferred, Complained, Rejected, Suppressed, OptedOut, Keys int64
			MedianMs                                                                             int64
			SMTP                                                                                 bool
			ComplainedOn                                                                         []string
		}
		Route struct {
			Primary, Fallback *struct{ Name string }
			Policy            string
			Priority          bool
		}
		Alerts    []struct{ Kind string }
		Incidents []struct {
			Kind     string
			Messages int64
			MedianMs *int64
			Value    float64
			Domains  []struct{ Domain string }
		}
	}
	c.Do("GET", base+"/overview?period=7d", nil).MustStatus(t, 200).JSON(t, &ov)
	st := ov.Stats
	if st.Sent != 13721 || st.Delivered != 13688 || st.Bounced != 27 || st.Deferred != 6 || st.Complained != 1 || st.Rejected != 12 ||
		st.Suppressed != 10 || st.OptedOut != 2 || st.Keys != 2 || !st.SMTP || fmt1(float64(st.MedianMs)/1000) != "1.3" ||
		len(st.ComplainedOn) != 1 || st.ComplainedOn[0] != "usage-summary" {
		t.Fatalf("overview: %+v", st)
	}
	if ov.Route.Primary == nil || ov.Route.Primary.Name != "Postmark" || ov.Route.Fallback == nil || ov.Route.Fallback.Name != "Amazon SES" ||
		ov.Route.Policy != "automatic" || !ov.Route.Priority || len(ov.Alerts) != 0 {
		t.Fatalf("route %+v alerts %+v", ov.Route, ov.Alerts)
	}
	if len(ov.Incidents) != 2 || ov.Incidents[0].Kind != "failover" || ov.Incidents[0].Messages != 37 || ov.Incidents[0].MedianMs == nil || *ov.Incidents[0].MedianMs != 3100 ||
		ov.Incidents[1].Kind != "bounce_rate" || ov.Incidents[1].Value != 2.4 || len(ov.Incidents[1].Domains) == 0 || ov.Incidents[1].Domains[0].Domain != "gmial.com" {
		t.Fatalf("incidents: %+v", ov.Incidents)
	}

	// The message log, last 24 hours: its chips and its latest rows.
	var log struct {
		Counts struct{ All, Delivered, Deferred, Bounced, Complained, Rejected, Failed, Test int64 }
		Items  []struct{ Email, Status string }
	}
	c.Do("GET", base+"/messages?range=24h&limit=10", nil).MustStatus(t, 200).JSON(t, &log)
	if n := log.Counts; n.All != 1986 || n.Delivered != 1951 || n.Deferred != 2 || n.Bounced != 4 || n.Complained != 0 || n.Rejected != 3 || n.Failed != 0 || n.Test != 26 {
		t.Fatalf("counts: %+v", log.Counts)
	}
	rows := []string{}
	for _, it := range log.Items {
		rows = append(rows, it.Email+" "+it.Status)
	}
	wantRows := "sam.rivera@outlook.com delivered, jordan@example.org rejected, billing@northwind.example delivered, priya.shah@example.net delivered, " +
		"ops@contoso.example bounced, kim.tran@example.com rejected, qa+signup@acme.io test, lena@example.com test, " +
		"finance@northwind.example delivered, alex.morgan@example.com delivered"
	if strings.Join(rows, ", ") != wantRows {
		t.Fatalf("latest rows:\n%s", strings.Join(rows, "\n"))
	}

	// Sam Rivera's password reset.
	var msg struct {
		Status, ProviderMessageID, IdempotencyKey string
		DeliveryMs                                int64
		Template                                  struct{ Version int }
		Timeline                                  []struct{ Kind string }
	}
	c.Do("GET", base+"/messages/0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44", nil).MustStatus(t, 200).JSON(t, &msg)
	kinds := []string{}
	for _, e := range msg.Timeline {
		kinds = append(kinds, e.Kind)
	}
	if msg.Status != "delivered" || msg.DeliveryMs != 1310 || msg.ProviderMessageID != "7e1f3c2a-91d4-4b70-a6f2-0c8e55d1b9a3" || msg.IdempotencyKey != "pwreset-7f3a91" ||
		msg.Template.Version != 3 || strings.Join(kinds, ",") != "accepted,rendered,routed,handed,delivered" {
		t.Fatalf("Sam's message: %+v", msg)
	}
	c.Do("GET", base+"/messages/0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44/content", nil).MustStatus(t, 200)

	// Lucía Fernández's reset went out in Spanish, her contact's language
	// (the languages brief): the log's Language column and the detail's line.
	var lucia struct {
		Items []struct {
			ID, Email, Language, Subject string
			ContactID                    string
		}
	}
	c.Do("GET", base+"/messages?range=24h&q=lucia.fernandez", nil).MustStatus(t, 200).JSON(t, &lucia)
	if len(lucia.Items) != 1 || lucia.Items[0].Language != "es" || lucia.Items[0].ContactID == "" || lucia.Items[0].Subject != "Restablece tu contraseña de Acme Cloud" {
		t.Fatalf("Lucía's message: %+v", lucia.Items)
	}
	var lang struct {
		LanguageInfo *struct{ Sent, Contact, Reason string }
	}
	c.Do("GET", base+"/messages/"+lucia.Items[0].ID, nil).MustStatus(t, 200).JSON(t, &lang)
	if lang.LanguageInfo == nil || lang.LanguageInfo.Sent != "es" || lang.LanguageInfo.Contact != "es" || lang.LanguageInfo.Reason != "match" {
		t.Fatalf("Lucía's language: %+v", lang.LanguageInfo)
	}

	// Categories and their opt-outs.
	var cats struct {
		Items []struct {
			Name, Kind string
			Templates  int
			OptedOut   int64
		}
	}
	c.Do("GET", base+"/categories", nil).MustStatus(t, 200).JSON(t, &cats)
	got := []string{}
	for _, it := range cats.Items {
		got = append(got, it.Name+":"+it.Kind+":"+itoa(it.OptedOut))
	}
	// Memberships holds Earn's member emails (phase 6).
	if strings.Join(got, " ") != "Account:essential:0 Billing:essential:0 Reminders:optional:2 Product updates:optional:31 Memberships:essential:0" {
		t.Fatalf("categories: %v", got)
	}

	// Setup: the keys that can send, the verified senders, the relay, and
	// "Up to 10 MB … (Postmark's limit; SES allows 40 MB)".
	var setup struct {
		Keys    []struct{ Name, Preset, Mode string }
		Senders []struct{ Email string }
		Relay   struct {
			Enabled bool
			State   string
		}
		Limits struct {
			RouteLimitBytes    int64
			RouteLimitProvider string
			RouteLimitOthers   []struct {
				Provider string
				Bytes    int64
			}
		}
	}
	c.Do("GET", base+"/setup", nil).MustStatus(t, 200).JSON(t, &setup)
	keys := []string{}
	for _, k := range setup.Keys {
		keys = append(keys, k.Name+":"+k.Preset+":"+k.Mode)
	}
	if strings.Join(keys, ", ") != "Acme Cloud · production:send_only:live, Acme Cloud · staging:send_only:test, Billing service (SMTP):smtp:live" ||
		len(setup.Senders) != 2 || !setup.Relay.Enabled || setup.Relay.State != "ok" {
		t.Fatalf("setup: %+v", setup)
	}
	if l := setup.Limits; l.RouteLimitBytes != 10<<20 || l.RouteLimitProvider != "Postmark" || len(l.RouteLimitOthers) != 1 ||
		l.RouteLimitOthers[0].Provider != "SES" || l.RouteLimitOthers[0].Bytes != 40<<20 {
		t.Fatalf("setup limits: %+v", l)
	}

	// The demo's relay is on with passing checks, but nothing listens
	// until an instance admin saves its settings.
	relay := env.App.SMTPRelay()
	if err := relay.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := relay.Listening(); ok {
		t.Fatal("the demo's relay listens")
	}
	c.Do("PUT", "/admin/smtp-relay", map[string]any{"enabled": false, "hostname": "reputecast.acme.io", "starttlsPort": 587, "tlsPort": 465, "certSource": "acme"}).MustStatus(t, 200)
	var demo bool
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT coalesce((status->>'demo')::boolean, false) FROM smtp_relay`).Scan(&demo); err != nil || demo {
		t.Fatalf("saving kept the demo flag: %v", err)
	}
}

func fmt1(v float64) string { return fmt.Sprintf("%.1f", v) }

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
