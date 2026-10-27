package engine_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// canaryFixture is a send of 400 (300 engaged Gmail readers, 100 Yahoo
// readers who never opened) without the seed inboxes: the seeds' own
// placement test is internal/placement's.
func canaryFixture(t *testing.T, f *sendtest.Fixture) {
	t.Helper()
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.Exec(`INSERT INTO traffic_rules (workspace_id, canary) VALUES ($1, '{"seeds":false}')
		ON CONFLICT (workspace_id) DO UPDATE SET canary = traffic_rules.canary || '{"seeds":false}'`, f.WS)
	t.Cleanup(func() { f.Exec(`UPDATE traffic_rules SET canary = canary - 'seeds' WHERE workspace_id = $1`, f.WS) })
	f.AddContacts(
		sendtest.ContactSpec{N: 300, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 100, Mailbox: "yahoo", Domain: "yahoo.com"},
	)
}

// canaryState reads the send's canary.
func canaryState(t *testing.T, f *sendtest.Fixture) map[string]any {
	t.Helper()
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT canary FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	_ = json.Unmarshal(raw, &c)
	return c
}

// testCanaryPasses: a canary chosen in Review & send takes the most
// engaged 5% first and holds everyone else; after the watch every gate
// passes and the send goes on to everyone by itself (review GS-10).
func testCanaryPasses(t *testing.T, f *sendtest.Fixture) {
	canaryFixture(t, f)
	f.StartSend(engine.StartOptions{Canary: engine.CanaryOn})
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND canary`, f.Campaign); n != 20 {
		f.Dump()
		t.Fatalf("%d canary recipients, want 5%% of 400", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND canary AND mailbox_provider <> 'gmail'`, f.Campaign); n != 0 {
		t.Fatalf("%d canary recipients outside the engaged Gmail readers: the most engaged go first", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held' AND hold_reason = 'canary'`, f.Campaign); n != 380 {
		t.Fatalf("%d held for the canary, want 380", n)
	}
	at, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); n != 20 {
		f.Dump()
		t.Fatalf("%d sent while the canary is watched, want its 20", n)
	}
	setClock(t, f, at)
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	c := snap.Canary
	if c == nil || c.Phase != "watching" || c.Size != 20 || c.Queued != 380 || c.Until == nil || !c.Chosen || len(c.Gates) != 4 {
		t.Fatalf("canary %+v", c)
	}
	if u := c.Until.Sub(*c.SentAt); u != 20*time.Minute {
		t.Fatalf("watched %v after the last canary message, want the 20 minutes Traffic rules set", u)
	}
	// What the providers still retry after a deferral, in all and per
	// mailbox provider (the canary's tiles and table; review GS-14).
	var retrying, deferred int64
	for _, m := range snap.Mailboxes {
		retrying += m.Retrying
		deferred += m.Deferred
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'deferred'`, f.Campaign); snap.Retrying != n || retrying != n {
		t.Fatalf("retrying %d (%d by mailbox provider), want the %d deferred messages", snap.Retrying, retrying, n)
	}
	if deferred != snap.Totals.Deferred {
		t.Fatalf("deferred %d by mailbox provider, %d in all", deferred, snap.Totals.Deferred)
	}
	// "Watch 10 more minutes" moves the end; it's audited.
	if err := f.E.ExtendCanary(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if c := canaryState(t, f); c["extended"] != 10.0 {
		t.Fatalf("canary %v after watching longer", c)
	}
	at, done, err := f.E.Drive(f.Ctx, f.Campaign, at, 45*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("done %v, status %s: the canary should pass and everyone get it", done, f.Status())
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); n != 400 {
		t.Fatalf("%d sent, want everyone", n)
	}
	if c := canaryState(t, f); c["phase"] != "passed" {
		t.Fatalf("canary %v, want passed", c)
	}
	// The main send went out after the watch: nothing before it ended.
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND NOT canary AND sent_at < $2`, f.Campaign, f.Start.Add(30*time.Minute)); n != 0 {
		t.Fatalf("%d main-send messages before the watch ended", n)
	}
	for _, action := range []string{"campaign.canary_passed", "campaign.canary_extended"} {
		if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = $2`, f.Campaign.String(), action); n != 1 {
			t.Fatalf("%d %s entries, want 1", n, action)
		}
	}
	_ = at
}

// testCanaryHolds: a gate that fails holds the send and asks: an incident
// with stop and continue, the people who can send told; stopping cancels
// the recipients it held and keeps the canary's results.
func testCanaryHolds(t *testing.T, f *sendtest.Fixture) {
	canaryFixture(t, f)
	f.StartSend(engine.StartOptions{Canary: engine.CanaryOn})
	at, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// 8 of the 20 canary messages bounced: 40% against the 1% gate.
	f.Exec(`UPDATE campaign_counters SET hard_bounced = hard_bounced + 8 WHERE campaign_id = $1 AND mailbox_provider = 'gmail'
		AND provider_id = (SELECT provider_id FROM campaign_counters WHERE campaign_id = $1 AND mailbox_provider = 'gmail' ORDER BY sent DESC LIMIT 1)`, f.Campaign)
	at, _, err = f.E.Drive(f.Ctx, f.Campaign, at, 25*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if f.Status() != "paused" {
		f.Dump()
		t.Fatalf("status %s, want paused by the failed gate", f.Status())
	}
	if c := canaryState(t, f); c["phase"] != "held" {
		t.Fatalf("canary %v, want held", c)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'canary' AND metric = 'bounce' AND resolved_at IS NULL`, f.Campaign); n != 1 {
		t.Fatalf("%d canary incidents, want the bounce gate's", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND NOT canary AND sent_at IS NOT NULL`, f.Campaign); n != 0 {
		t.Fatalf("%d main-send messages went out after the gate failed", n)
	}
	setClock(t, f, at)
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident == nil || snap.Incident.Kind != "canary" || len(snap.Incident.Detail.Options) != 2 {
		t.Fatalf("decision %+v", snap.Incident)
	}
	// A plain resume would leave the held recipients waiting: the canary
	// decides.
	if err := f.E.Resume(f.Ctx, f.Campaign, f.Sofia); err == nil {
		t.Fatal("resuming a send whose canary failed a gate should ask for the canary's decision")
	}
	if err := f.E.StopCanary(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if f.Status() != "cancelled" {
		t.Fatalf("status %s after stopping, want cancelled", f.Status())
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'cancelled' AND NOT canary`, f.Campaign); n != 380 {
		t.Fatalf("%d cancelled, want the 380 the canary held", n)
	}
	if c := canaryState(t, f); c["phase"] != "stopped" {
		t.Fatalf("canary %v, want stopped", c)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'canary' AND resolution = 'stopped'`, f.Campaign); n != 1 {
		t.Fatal("the canary's incident isn't resolved as stopped")
	}
}

// testCanaryChoices: off sends to everyone at once; auto runs a canary only
// when Traffic rules' reasons apply and the audience is large enough;
// Review & send's choice is kept for a scheduled start; continuing early
// releases everyone at once.
func testCanaryChoices(t *testing.T, f *sendtest.Fixture) {
	canaryFixture(t, f)
	plan, err := f.E.CanaryPlanFor(f.Ctx, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	// Never mailed from news.acme.io, but 400 is under the 1,000 an
	// automatic canary needs.
	if plan.Auto || plan.MinAudience != 1000 || plan.Percent != 5 {
		t.Fatalf("plan %+v", plan)
	}
	f.StartSend(engine.StartOptions{Canary: engine.CanaryOn})
	var canary string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT canary FROM campaigns WHERE id = $1`, f.Campaign).Scan(&canary); err != nil || canary != "on" {
		t.Fatalf("campaign canary %q (%v), want the choice kept", canary, err)
	}
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := f.E.ContinueCanary(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if c := canaryState(t, f); c["phase"] != "continued" || c["decidedBy"] == "" {
		t.Fatalf("canary %v, want continued by Sofia", c)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held'`, f.Campaign); n != 0 {
		t.Fatalf("%d still held after continuing", n)
	}
	if err := f.E.ExtendCanary(f.Ctx, f.Campaign, f.Sofia); err == nil {
		t.Fatal("watching longer after continuing should be refused")
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.canary_continued'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d continue entries, want 1", n)
	}
}
