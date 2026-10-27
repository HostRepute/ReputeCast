package tracking_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/tracking"
)

const chrome = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"

// deliver applies a delivery event through the webhook normalizer.
func deliver(t *testing.T, f *sendtest.Fixture, at time.Time, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		var pid uuid.UUID
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT provider_id FROM messages WHERE id = $1`, id).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		p, err := f.Env.App.Q.ProvGetAny(f.Ctx, pid)
		if err != nil {
			t.Fatal(err)
		}
		mid := id
		if err := sending.Tx(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
			_, err := f.E.Normalizer.Apply(f.Ctx, tx, p, nil, "webhook", at, providers.Event{Type: providers.EventDelivered, MessageID: &mid, OccurredAt: at, OriginalCode: "250 2.0.0"})
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func linkCount(t *testing.T, f *sendtest.Fixture, position int, col string) int64 {
	t.Helper()
	var clicks, unique, bot int64
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT clicks, unique_clicks, bot_clicks FROM campaign_links WHERE campaign_id = $1 AND position = $2`,
		f.Campaign, position).Scan(&clicks, &unique, &bot); err != nil {
		t.Fatal(err)
	}
	return map[string]int64{"clicks": clicks, "unique": unique, "bot": bot}[col]
}

// TestBotClicks: clicks settle before they count, so a scanner's clicks
// (a burst over every link, the hidden link, a click on arrival, an
// address on the instance's scanner list) never count as a person's, and
// nothing has to be taken back (review AN-2).
func TestBotClicks(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "Bot clicks")
	ids := f.HandedOver(sendtest.ContactSpec{N: 6, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	if n := f.Count(`SELECT count(*) FROM campaign_links WHERE campaign_id = $1`, f.Campaign); n < 4 {
		t.Fatalf("the test email has %d tracked links, want at least 4", n)
	}
	tr, ok := f.E.Recorder.(*tracking.Service)
	if !ok {
		t.Fatalf("recorder %T", f.E.Recorder)
	}
	ctx := context.Background()
	arrived := f.Start.Add(time.Minute)
	deliver(t, f, arrived, ids[0], ids[1], ids[2], ids[4])
	t0 := f.Start.Add(30 * time.Minute)
	click := func(id uuid.UUID, link int, at time.Time, ip string) sending.Engagement {
		return sending.Engagement{MessageID: id, Click: true, Link: link, UserAgent: chrome, IP: ip, Method: "GET", At: at}
	}
	record := func(evs ...sending.Engagement) {
		t.Helper()
		if err := tr.RecordEngagement(ctx, evs); err != nil {
			t.Fatal(err)
		}
	}
	settle := func(at time.Time) {
		t.Helper()
		if _, err := tr.SettleEngagement(ctx, at); err != nil {
			t.Fatal(err)
		}
	}

	// A person's click waits to settle, then counts once.
	record(click(ids[0], 0, t0, "198.51.100.10"))
	if c := f.Counts(); c["clicked"] != 0 || c["opened_human"] != 0 {
		t.Fatalf("a click counted before it settled: %v", c)
	}
	settle(t0.Add(30 * time.Second))
	if f.Counts()["clicked"] != 0 {
		t.Fatal("a click settled before its minute was up")
	}
	settle(t0.Add(61 * time.Second))
	if c := f.Counts(); c["clicked"] != 1 || c["opened_human"] != 1 {
		t.Fatalf("a person's click: %v, want 1 click and 1 human open", c)
	}
	if linkCount(t, f, 0, "clicks") != 1 || linkCount(t, f, 0, "unique") != 1 {
		t.Fatal("the link's clicks don't include the person's click")
	}

	// A scanner following every link within seconds: a burst.
	record(click(ids[1], 0, t0, "198.51.100.11"), click(ids[1], 1, t0.Add(time.Second), "198.51.100.11"), click(ids[1], 2, t0.Add(2*time.Second), "198.51.100.11"))
	settle(t0.Add(2 * time.Minute))
	if c := f.Counts(); c["clicked"] != 1 || c["opened_human"] != 1 {
		t.Fatalf("a burst counted as a person's: %v", c)
	}
	for _, link := range []int{0, 1, 2} {
		if linkCount(t, f, link, "bot") != 1 {
			t.Fatalf("link %d: the burst's click isn't among the bot clicks", link)
		}
	}

	// The hidden link: the click next to it was the same scanner's.
	record(click(ids[2], 3, t0, "198.51.100.12"), click(ids[2], tracking.HoneypotLink, t0.Add(5*time.Second), "198.51.100.12"))
	settle(t0.Add(3 * time.Minute))
	if f.Counts()["clicked"] != 1 || linkCount(t, f, 3, "bot") != 1 {
		t.Fatal("a click next to the hidden link counted as a person's")
	}

	// Windows count from delivery: delivered late, clicked on arrival.
	late := t0.Add(10 * time.Minute)
	deliver(t, f, late, ids[3])
	record(click(ids[3], 0, late.Add(3*time.Second), "198.51.100.13"))
	if n := f.Count(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'click' AND class = 'bot'`, ids[3]); n != 1 {
		t.Fatal("a click 3 s after a late delivery wasn't a scanner's")
	}

	// A network an instance admin adds to the scanner list.
	admin := f.Env.Client()
	admin.SignInDemo("linh@acme.io")
	admin.Do("POST", "/admin/tracking-networks", map[string]any{"cidr": "198.18.0.0/15", "name": "Our mail gateway", "class": "scanner"}).MustStatus(t, http.StatusCreated)
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE action = 'tracking_network.create' AND target_id = '198.18.0.0/15'`); n != 1 {
		t.Fatalf("%d audit entries for the new network", n)
	}
	admin.Do("POST", "/admin/tracking-networks", map[string]any{"cidr": "10.0.0.0/4", "name": "Too wide", "class": "scanner"}).MustStatus(t, http.StatusUnprocessableEntity)
	viewer := f.Env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("POST", "/admin/tracking-networks", map[string]any{"cidr": "198.20.0.0/16", "name": "x", "class": "scanner"}).MustStatus(t, http.StatusForbidden)
	record(click(ids[4], 1, arrived.Add(40*time.Second), "198.18.4.4"))
	var reason string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT diagnostic FROM delivery_events WHERE message_id = $1 AND type = 'click'`, ids[4]).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != "Link scanner (Our mail gateway)" {
		t.Fatalf("a click from the listed gateway on arrival: %q", reason)
	}
	admin.Do("DELETE", "/admin/tracking-networks?cidr=198.18.0.0/15", nil).MustStatus(t, http.StatusNoContent)
	admin.Do("DELETE", "/admin/tracking-networks?cidr=17.0.0.0/8", nil).MustStatus(t, http.StatusNotFound)

	// The hidden link in an email: a request to it is a bot click.
	var contact uuid.UUID
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT contact_id FROM messages WHERE id = $1`, ids[5]).Scan(&contact); err != nil {
		t.Fatal(err)
	}
	token := f.Env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Honeypot, WorkspaceID: f.WS, MessageID: ids[5], ContactID: contact})
	resp, err := http.Get(f.Env.Server.URL + "/t/h/" + token) //nolint:noctx // a test request
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("hidden link: %d", resp.StatusCode)
	}
	deadline := time.Now().Add(5 * time.Second)
	for f.Count(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'click' AND class = 'bot' AND link = -1`, ids[5]) != 1 {
		if time.Now().After(deadline) {
			t.Fatal("the hidden link's click wasn't recorded as a bot's")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// The counters can be rebuilt from these events: the same numbers.
	before := f.Counts()
	links := f.Count(`SELECT coalesce(sum(clicks * 1000000 + unique_clicks * 1000 + bot_clicks), 0) FROM campaign_links WHERE campaign_id = $1`, f.Campaign)
	if _, err := counters.Rebuild(ctx, f.Env.App.Pool, f.Campaign, time.Now()); err != nil {
		t.Fatal(err)
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
