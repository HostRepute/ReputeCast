package tracking_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// TestProviderTracking: when a workspace keeps its sending provider's
// tracking, the opens and clicks the provider's webhook reports count
// like ReputeCast's own (review AN-4): classified (people, Apple Mail's
// privacy proxy, scanners), clicks matched to the email's links and
// settled first, redeliveries counted once, and the counters rebuild to
// the same numbers.
func TestProviderTracking(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "Provider tracking")
	f.Exec(`UPDATE providers SET tracking = jsonb_set(coalesce(tracking, '{}'::jsonb), '{mode}', '"provider"') WHERE workspace_id = $1`, f.WS)
	ids := f.HandedOver(sendtest.ContactSpec{N: 8, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	tr, ok := f.E.Recorder.(*tracking.Service)
	if !ok {
		t.Fatalf("recorder %T", f.E.Recorder)
	}
	ctx := context.Background()
	arrived := f.Start.Add(time.Minute)
	deliver(t, f, arrived, ids...)
	urls := map[int]string{}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT position, url FROM campaign_links WHERE campaign_id = $1 ORDER BY position`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var pos int
		var u string
		if err := rows.Scan(&pos, &u); err != nil {
			t.Fatal(err)
		}
		urls[pos] = u
	}
	rows.Close()
	if len(urls) < 3 {
		t.Fatalf("the test email has %d links, want at least 3", len(urls))
	}
	// The provider's webhook, through the normalizer as production gets it.
	report := func(evs ...providers.Event) {
		t.Helper()
		if err := sending.Tx(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
			for _, ev := range evs {
				var pid uuid.UUID
				if err := tx.QueryRow(f.Ctx, `SELECT provider_id FROM messages WHERE id = $1`, *ev.MessageID).Scan(&pid); err != nil {
					return err
				}
				p, err := f.Env.App.Q.ProvGetAny(f.Ctx, pid)
				if err != nil {
					return err
				}
				if _, err := f.E.Normalizer.ApplyAll(f.Ctx, tx, p, nil, "webhook", ev.OccurredAt, []providers.Event{ev}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	open := func(id uuid.UUID, at time.Time, ua, ip string) providers.Event {
		mid := id
		return providers.Event{Type: providers.EventOpen, MessageID: &mid, OccurredAt: at, OriginalType: "Open", UserAgent: ua, IP: ip}
	}
	click := func(id uuid.UUID, at time.Time, url, ua string) providers.Event {
		mid := id
		return providers.Event{Type: providers.EventClick, MessageID: &mid, OccurredAt: at, OriginalType: "Click", URL: url, UserAgent: ua, IP: "198.51.100.20"}
	}
	settle := func(at time.Time) {
		t.Helper()
		if _, err := tr.SettleEngagement(ctx, at); err != nil {
			t.Fatal(err)
		}
	}
	t0 := arrived.Add(20 * time.Minute)

	// A person's open counts, once even when the provider sends it again.
	report(open(ids[0], t0, chrome, "198.51.100.10"))
	report(open(ids[0], t0, chrome, "198.51.100.10"))
	if c := f.Counts(); c["opened_human"] != 1 {
		t.Fatalf("a person's open reported by the provider: %v", c)
	}
	if n := f.Count(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'open'`, ids[0]); n != 1 {
		t.Fatalf("%d open events after a redelivery, want 1", n)
	}
	// Apple Mail's privacy proxy and a scanner, apart from people.
	report(open(ids[1], arrived.Add(3*time.Second), "Mozilla/5.0", "17.58.1.1"))
	report(open(ids[2], arrived.Add(5*time.Second), "Barracuda Sentinel (EE)", "203.0.113.9"))
	// A provider that doesn't pass the user agent on: not a machine's sign.
	report(open(ids[3], t0, "", ""))
	if c := f.Counts(); c["opened_human"] != 2 || c["opened_privacy"] != 1 || c["opened_machine"] != 1 {
		t.Fatalf("opens by class: %v, want 2 human, 1 privacy-protected, 1 machine", c)
	}
	var class string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT class FROM delivery_events WHERE message_id = $1 AND type = 'open'`, ids[1]).Scan(&class); err != nil || class != "privacy" {
		t.Fatalf("the privacy proxy's open is stored as %q (%v)", class, err)
	}

	// A person's click: matched to the email's link, counted once it settles.
	report(click(ids[4], t0, urls[1], chrome))
	if f.Counts()["clicked"] != 0 {
		t.Fatal("a provider-reported click counted before it settled")
	}
	settle(t0.Add(61 * time.Second))
	if c := f.Counts(); c["clicked"] != 1 || c["opened_human"] != 3 {
		t.Fatalf("a person's click: %v, want 1 click and its human open", c)
	}
	if linkCount(t, f, 1, "clicks") != 1 || linkCount(t, f, 1, "unique") != 1 {
		t.Fatal("the click isn't on its link")
	}
	// A scanner following every link within seconds: bot clicks.
	report(click(ids[5], t0, urls[0], chrome), click(ids[5], t0.Add(time.Second), urls[1], chrome), click(ids[5], t0.Add(2*time.Second), urls[2], chrome))
	settle(t0.Add(3 * time.Minute))
	if c := f.Counts(); c["clicked"] != 1 {
		t.Fatalf("a burst counted as a person's: %v", c)
	}
	if linkCount(t, f, 0, "bot") != 1 || linkCount(t, f, 2, "bot") != 1 {
		t.Fatal("the burst's clicks aren't among the bot clicks")
	}
	// A link that isn't one of the email's (the provider's own, say): the
	// person clicked, the click map can't place it.
	report(click(ids[6], t0, "https://example.org/not-in-the-email", chrome))
	settle(t0.Add(5 * time.Minute))
	if c := f.Counts(); c["clicked"] != 2 {
		t.Fatalf("a click on another link: %v, want 2 clicks", c)
	}
	if n := f.Count(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'click' AND link = $2 AND class = 'human'`, ids[6], tracking.NoLink); n != 1 {
		t.Fatal("the click on another link isn't stored as a person's, off the map")
	}

	// With ReputeCast's tracking, the provider's opens are logged nowhere
	// and count for nothing (ReputeCast's pixel counted them already).
	f.Exec(`UPDATE providers SET tracking = jsonb_set(coalesce(tracking, '{}'::jsonb), '{mode}', '"reputecast"') WHERE workspace_id = $1`, f.WS)
	before := f.Counts()
	report(open(ids[7], t0, chrome, "198.51.100.30"))
	if c := f.Counts(); c["opened_human"] != before["opened_human"] {
		t.Fatalf("a provider open counted with ReputeCast's tracking: %v", c)
	}

	// The counters rebuild from the stored events to the same numbers.
	before = f.Counts()
	links := f.Count(`SELECT coalesce(sum(clicks * 1000000 + unique_clicks * 1000 + bot_clicks), 0) FROM campaign_links WHERE campaign_id = $1`, f.Campaign)
	res, err := counters.Rebuild(ctx, f.Env.App.Pool, f.Campaign, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !res.Rebuilt {
		t.Fatalf("not rebuilt: %+v", res)
	}
	after := f.Counts()
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("rebuilt %s = %d, counted %d", k, after[k], v)
		}
	}
	if got := f.Count(`SELECT coalesce(sum(clicks * 1000000 + unique_clicks * 1000 + bot_clicks), 0) FROM campaign_links WHERE campaign_id = $1`, f.Campaign); got != links {
		t.Fatalf("rebuilt link clicks %d, counted %d", got, links)
	}
}

// TestLinkPosition: a provider-reported click finds its link, personalized
// or not.
func TestLinkPosition(t *testing.T) {
	links := map[int]string{
		0: "https://acme.io/changelog",
		1: "https://docs.acme.io/flags?ref={{ contact.id }}&utm_source=newsletter",
		2: "https://acme.io/changelog",
		3: "{{ campaign.url }}",
	}
	for _, c := range []struct {
		clicked string
		want    int
	}{
		{"https://acme.io/changelog", 0},
		{" https://acme.io/changelog ", 0},
		{"https://docs.acme.io/flags?ref=0192-abc&utm_source=newsletter", 1},
		{"https://docs.acme.io/flags?ref=x&utm_source=other", 3},
		{"", tracking.NoLink},
	} {
		if got := tracking.LinkPosition(links, c.clicked); got != c.want {
			t.Errorf("%q: link %d, want %d", c.clicked, got, c.want)
		}
	}
	if got := tracking.LinkPosition(map[int]string{0: "https://a.example/x"}, "https://b.example/"); got != tracking.NoLink {
		t.Errorf("an unknown link: %d", got)
	}
}
