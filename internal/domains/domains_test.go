package domains_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const ws = "/workspaces/acme-newsletter"

type cellT struct{ Text, Tone string }

type overviewT struct {
	Senders []struct {
		ID, Name, Email, ReplyTo, Note, Domain string
		DefaultFor                             []string
		Pending                                bool
		Verification                           []struct{ ProviderID, Status, Text string }
	}
	Domains []struct {
		ID, Domain, Note                             string
		UsedBy                                       []string
		DKIM, SPF, DMARC, ReturnPath, Tracking, BIMI cellT
		Senders                                      int
	}
	Providers []struct{ ID, Name, Short string }
	Defaults  struct {
		DefaultSenderID, ReplyHandling, ReplyTo, FooterAddress, TrackingDomain string
		TrackingOptions                                                        []struct {
			Domain   string
			Verified bool
		}
	}
	CanManage bool
}

func auditCount(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSendersOverview(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var ov overviewT
	c.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &ov)
	if len(ov.Providers) != 3 || ov.Providers[0].Short != "SES" || ov.Providers[1].Short != "Postmark" || ov.Providers[2].Short != "Postal" {
		t.Fatalf("provider columns: %+v", ov.Providers)
	}
	by := map[string]int{}
	for i, s := range ov.Senders {
		by[s.Email] = i
	}
	if len(ov.Senders) != 6 {
		t.Fatalf("senders: %d", len(ov.Senders))
	}
	digest := ov.Senders[by["digest@news.acme.io"]]
	if digest.Name != "Acme Newsletter" || digest.ReplyTo != "hello@acme.io" || len(digest.DefaultFor) != 1 || digest.DefaultFor[0] != "broadcast" {
		t.Fatalf("digest: %+v", digest)
	}
	for _, v := range digest.Verification {
		if v.Text != "Verified" {
			t.Fatalf("digest verification: %+v", digest.Verification)
		}
	}
	noReply := ov.Senders[by["no-reply@news.acme.io"]]
	if noReply.DefaultFor[0] != "system" || noReply.Verification[2].Text != "Not used" || noReply.Verification[0].Text != "Verified" {
		t.Fatalf("no-reply: %+v", noReply)
	}
	marcus := ov.Senders[by["marcus@acme.io"]]
	if marcus.Note != "personal sends" || !marcus.Pending || marcus.Verification[0].Text != "Pending · acme.io DKIM" || marcus.Verification[1].Text != "Not set up" {
		t.Fatalf("marcus: %+v", marcus)
	}
	if len(ov.Domains) != 2 {
		t.Fatalf("domains: %+v", ov.Domains)
	}
	news := ov.Domains[0]
	if news.Domain != "news.acme.io" || news.DKIM.Text != "3 of 3 providers" || news.SPF.Text != "pass · -all" || news.DMARC.Text != "p=quarantine" ||
		news.Tracking.Text != "go.news.acme.io" || news.BIMI.Text != "Not set" || news.ReturnPath.Text != "bounce. · pm-bounces. · rp." ||
		strings.Join(news.UsedBy, " · ") != "SES · Postmark · Postal" || news.Note != "primary sending domain" {
		t.Fatalf("news.acme.io: %+v", news)
	}
	acme := ov.Domains[1]
	if acme.DKIM.Text != "Pending CNAME 2 of 3" || acme.DKIM.Tone != "warn" || acme.ReturnPath.Text != "Default" || strings.Join(acme.UsedBy, ",") != "SES" {
		t.Fatalf("acme.io: %+v", acme)
	}
	if ov.Defaults.DefaultSenderID != digest.ID || ov.Defaults.ReplyTo != "hello@acme.io" || !strings.HasPrefix(ov.Defaults.FooterAddress, "Acme Inc") ||
		ov.Defaults.TrackingDomain != "go.news.acme.io" || len(ov.Defaults.TrackingOptions) != 1 || !ov.Defaults.TrackingOptions[0].Verified || !ov.CanManage {
		t.Fatalf("defaults: %+v", ov.Defaults)
	}

	// Viewers read; they can't change anything.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	var vov overviewT
	v.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &vov)
	if vov.CanManage {
		t.Fatal("viewer canManage")
	}
	v.Do("POST", ws+"/senders", map[string]any{"name": "X", "email": "x@news.acme.io"}).MustStatus(t, 403)
	v.Do("POST", ws+"/domains", map[string]any{"domain": "example.test"}).MustStatus(t, 403)
}

func TestSenderLifecycle(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var created struct {
		ID           string
		Verification []struct{ Status, Text string }
		DefaultFor   []string
	}
	c.Do("POST", ws+"/senders", map[string]any{"name": "Acme Events", "email": "Events@News.Acme.io", "replyTo": "events@acme.io"}).MustStatus(t, 201).JSON(t, &created)
	if len(created.Verification) != 3 || created.Verification[0].Text != "Verified" {
		t.Fatalf("new sender: %+v", created)
	}
	// Unknown domain, duplicate, bad address.
	r := c.Do("POST", ws+"/senders", map[string]any{"name": "X", "email": "x@unknown.example"}).MustStatus(t, 422)
	if p := r.Problem(t); p["meta"].(map[string]any)["domain"] != "unknown.example" {
		t.Fatalf("unknown domain problem: %v", p)
	}
	c.Do("POST", ws+"/senders", map[string]any{"name": "X", "email": "events@news.acme.io"}).MustStatus(t, 409)
	c.Do("POST", ws+"/senders", map[string]any{"name": "X", "email": "not an address"}).MustStatus(t, 422)

	var ov overviewT
	c.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &ov)
	postal := ov.Providers[2].ID
	var upd struct {
		Name         string
		DefaultFor   []string
		Verification []struct{ Status, Text string }
	}
	c.Do("PATCH", ws+"/senders/"+created.ID, map[string]any{"name": "Acme Events team", "replyTo": "events@acme.io", "defaultFor": []string{"automation"}, "notUsedWith": []string{postal}}).
		MustStatus(t, 200).JSON(t, &upd)
	if upd.Name != "Acme Events team" || len(upd.DefaultFor) != 1 || upd.Verification[2].Text != "Not used" {
		t.Fatalf("updated: %+v", upd)
	}
	c.Do("PATCH", ws+"/senders/"+created.ID, map[string]any{"name": "X", "email": "other@news.acme.io"}).MustStatus(t, 422)
	// hello@ lost the Automations default.
	c.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &ov)
	for _, s := range ov.Senders {
		if s.Email == "hello@news.acme.io" && len(s.DefaultFor) != 0 {
			t.Fatalf("hello@ still default: %+v", s)
		}
	}
	c.Do("DELETE", ws+"/senders/"+created.ID, nil).MustStatus(t, 204)
	if auditCount(t, env, "sender.create") != 1 || auditCount(t, env, "sender.update") != 1 || auditCount(t, env, "sender.delete") != 1 {
		t.Fatal("audit entries")
	}
	// The Content Writer can't manage senders.
	w := env.Client()
	w.SignInDemo("daniel@acme.io")
	w.Do("PATCH", ws+"/senders/"+ov.Senders[0].ID, map[string]any{"name": "X"}).MustStatus(t, 403)
}

func TestSenderDefaults(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var ov overviewT
	c.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &ov)
	product := ""
	for _, s := range ov.Senders {
		if s.Email == "product@news.acme.io" {
			product = s.ID
		}
	}
	var def struct{ DefaultSenderID, FooterAddress, TrackingDomain string }
	c.Do("PUT", ws+"/senders/defaults", map[string]any{"defaultSenderId": product, "replyHandling": "forward", "replyTo": "hello@acme.io",
		"footerAddress": "Acme Inc · 1 Main St", "trackingDomain": ""}).MustStatus(t, 200).JSON(t, &def)
	if def.DefaultSenderID != product || def.FooterAddress != "Acme Inc · 1 Main St" || def.TrackingDomain != "" {
		t.Fatalf("defaults: %+v", def)
	}
	var defaultSender string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT default_sender_id::text FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&defaultSender); err != nil || defaultSender != product {
		t.Fatalf("workspace default sender: %s %v", defaultSender, err)
	}
	c.Do("PUT", ws+"/senders/defaults", map[string]any{"replyHandling": "inbox", "footerAddress": ""}).MustStatus(t, 422)
	c.Do("PUT", ws+"/senders/defaults", map[string]any{"replyHandling": "forward", "footerAddress": "", "trackingDomain": "go.unknown.example"}).MustStatus(t, 422)
	if auditCount(t, env, "senders.defaults_update") != 1 {
		t.Fatal("audit")
	}
}

func TestDomainLifecycle(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var created struct {
		Domain struct {
			ID, Domain, Note string
			Records          []struct{ Purpose, Provider, Status, Label, Name string }
			Counts           struct{ Verified, Total int }
			Providers        []struct{ Short string }
			DMARCTip         string
		}
	}
	c.Do("POST", ws+"/domains", map[string]any{"domain": "Mail.Example.test", "note": "events"}).MustStatus(t, 201).JSON(t, &created)
	d := created.Domain
	if d.Domain != "mail.example.test" || d.Note != "events" || len(d.Providers) != 3 || d.Counts.Total == 0 || d.Counts.Verified != 0 {
		t.Fatalf("created: %+v", d)
	}
	dmarc := false
	for _, r := range d.Records {
		if r.Purpose == "dmarc" {
			dmarc = r.Status == "missing" && r.Name == "_dmarc.mail.example.test"
		}
	}
	if !dmarc || !strings.Contains(d.DMARCTip, "ReputeCast suggested `p=quarantine`") {
		t.Fatalf("dmarc: %+v / %s", d.Records, d.DMARCTip)
	}
	c.Do("POST", ws+"/domains", map[string]any{"domain": "mail.example.test"}).MustStatus(t, 409)
	c.Do("POST", ws+"/domains", map[string]any{"domain": "not a domain"}).MustStatus(t, 422)

	// One provider's records (plus the domain-wide ones).
	var ov overviewT
	c.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &ov)
	ses := ov.Providers[0].ID
	var only struct {
		Records []struct{ Provider, Purpose string }
	}
	c.Do("GET", ws+"/domains/"+d.ID+"?provider="+ses, nil).MustStatus(t, 200).JSON(t, &only)
	for _, r := range only.Records {
		if r.Provider != "Amazon SES" && r.Purpose != "dmarc" {
			t.Fatalf("filtered records: %+v", only.Records)
		}
	}
	var patched struct {
		Note     string
		Tracking struct{ Domain, Status string }
	}
	c.Do("PATCH", ws+"/domains/"+d.ID, map[string]any{"note": "event invitations", "trackingDomain": "go.mail.example.test"}).MustStatus(t, 200).JSON(t, &patched)
	if patched.Note != "event invitations" || patched.Tracking.Domain != "go.mail.example.test" || patched.Tracking.Status != "missing" {
		t.Fatalf("patched: %+v", patched)
	}
	c.Do("PATCH", ws+"/domains/"+d.ID, map[string]any{"trackingDomain": "go.other.example"}).MustStatus(t, 422)
	c.Do("DELETE", ws+"/domains/"+d.ID, nil).MustStatus(t, 204)
	c.Do("DELETE", ws+"/domains/"+ov.Domains[0].ID, nil).MustStatus(t, 409) // senders use news.acme.io

	// A check against the demo DNS keeps news.acme.io verified.
	var checked struct {
		Status string
		Counts struct{ Verified, Total int }
		DNS    string `json:"dnsHost"`
	}
	c.Do("POST", ws+"/domains/"+ov.Domains[0].ID+"/check", nil).MustStatus(t, 200).JSON(t, &checked)
	if checked.Status != "verified" || checked.Counts.Verified != checked.Counts.Total || checked.DNS != "Cloudflare" {
		t.Fatalf("checked: %+v", checked)
	}
	for _, a := range []string{"domain.create", "domain.update", "domain.delete", "domain.check"} {
		if auditCount(t, env, a) == 0 {
			t.Fatalf("no audit entry %s", a)
		}
	}
}
