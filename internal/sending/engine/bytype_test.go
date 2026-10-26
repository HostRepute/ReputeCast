package engine_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testTypedLevels: the thresholds by bounce type and the deferral levels
// act with their own actions (review GS-11): Yahoo deferring 20% over the
// last 10 minutes slows it to half rate (a notice, nothing paused); Outlook
// rejecting 1.2% for policy pauses Outlook alone; Yahoo deferring 45% then
// pauses it; with Gmail still sending, unknown users over 5% of the whole
// send pause the send.
func testTypedLevels(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: 2000, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
		sendtest.ContactSpec{N: 2000, Mailbox: "outlook", Domain: "hotmail.com", Engaged: true},
		sendtest.ContactSpec{N: 2000, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	at, _ := f.Drive(3 * time.Minute)
	setClock(t, f, at)
	// eventsFor adds events for a share of what was sent to a mailbox
	// provider, a minute ago.
	eventsFor := func(mbp, typ, class string, pct int) {
		t.Helper()
		f.Exec(`INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, provider_id, type, occurred_at, mailbox_provider, original_code, bounce_class)
			SELECT gen_random_uuid(), m.workspace_id, m.id, m.campaign_id, m.provider_id, $3, $4, $2, '421 4.7.0', $5
			FROM messages m WHERE m.campaign_id = $1 AND m.mailbox_provider = $2 AND m.sent_at IS NOT NULL
			  AND NOT EXISTS (SELECT 1 FROM delivery_events d WHERE d.message_id = m.id AND d.type = $3)
			ORDER BY m.id LIMIT (SELECT count(*) * $6 / 100 FROM messages WHERE campaign_id = $1 AND mailbox_provider = $2 AND sent_at IS NOT NULL)`,
			f.Campaign, mbp, typ, at.Add(-time.Minute), class, pct)
	}
	drive := func(d time.Duration) {
		t.Helper()
		var err error
		if at, _, err = f.E.Drive(f.Ctx, f.Campaign, at, d); err != nil {
			t.Fatal(err)
		}
		setClock(t, f, at)
	}
	incidents := func(metric string) int64 {
		return f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'threshold' AND metric = $2`, f.Campaign, metric)
	}

	eventsFor("yahoo", "deferred", "", 20)
	drive(15 * time.Second)
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT throttles FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var th map[string]float64
	_ = json.Unmarshal(raw, &th)
	if th["yahoo"] != 2 {
		f.Dump()
		t.Fatalf("throttles %s, want Yahoo slowed from 4 to 2 /s", raw)
	}
	if incidents("deferral") != 1 || f.Status() != "sending" {
		t.Fatalf("%d deferral incidents, campaign %s: want one notice, still sending", incidents("deferral"), f.Status())
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident != nil {
		t.Fatalf("a slowdown is a notice, not a decision: %+v", snap.Incident)
	}

	eventsFor("outlook", "hard_bounce", "policy_block", 2)
	drive(15 * time.Second)
	if n := f.Count(`SELECT count(*) FROM campaign_sends WHERE campaign_id = $1 AND holds ? 'outlook'`, f.Campaign); n != 1 || incidents("block_bounce") != 1 {
		f.Dump()
		t.Fatal("Outlook isn't held for its block/policy bounces")
	}

	eventsFor("yahoo", "deferred", "", 50)
	drive(15 * time.Second)
	if n := f.Count(`SELECT count(*) FROM campaign_sends WHERE campaign_id = $1 AND holds ? 'yahoo'`, f.Campaign); n != 1 || incidents("deferral") != 2 {
		f.Dump()
		t.Fatal("Yahoo isn't paused at the second deferral level")
	}

	if s := f.Status(); s != "sending" {
		t.Fatalf("campaign %s, want Gmail still sending", s)
	}
	eventsFor("outlook", "hard_bounce", "mailbox_unknown", 20)
	eventsFor("yahoo", "hard_bounce", "mailbox_unknown", 20)
	eventsFor("gmail", "hard_bounce", "mailbox_unknown", 20)
	drive(15 * time.Second)
	if s := f.Status(); s != "paused" || incidents("unknown_user") != 1 {
		f.Dump()
		t.Fatalf("campaign %s, want paused for its unknown users", s)
	}
}
