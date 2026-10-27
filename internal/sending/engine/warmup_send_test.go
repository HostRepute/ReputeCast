package engine_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Warm-up in the send loop (review PR-8, C5): a warming provider sends no
// more than today's cap, per mailbox provider when its plan says so, most
// engaged first; past the cap the rest goes through a warm provider when
// the plan allows it, or waits as a hold with choices.
func TestWarmupInTheSendLoop(t *testing.T) {
	s := sendtest.NewSuite(t)
	// The overflow scenario runs first: it counts on Postal having sent
	// nothing yet today.
	t.Run("OverflowToWarmProvider", func(t *testing.T) { testWarmupOverflow(t, s.Fixture(t, "Warm-up overflow")) })
	t.Run("CapHoldsTheShare", func(t *testing.T) { testWarmupCapHold(t, s.Fixture(t, "Warm-up cap")) })
}

// warmupFixture sends the fixture's list through Postal alone (SES is the
// rule's fallback), without an A/B test, with today's cap left at 60.
func warmupFixture(t *testing.T, f *sendtest.Fixture, overflow, spread bool) (postal, ses uuid.UUID, plan string, sentBefore int64) {
	t.Helper()
	postal, ses = f.ProviderID("postal"), f.ProviderID("ses")
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.Exec(`INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id)
		VALUES ($1, $2, 'broadcast', 1, '', $3, $4, $5)`, uuid.Must(uuid.NewV7()), f.WS,
		fmt.Sprintf(`{"lists":[%q]}`, f.List), fmt.Sprintf(`[{"provider_id":%q,"weight":100}]`, postal), ses)
	a, ok, err := warmup.AllowanceFor(f.Ctx, f.Env.App.Q, postal, f.Start, nil)
	if err != nil || !ok {
		t.Fatalf("Postal's warm-up: %v %v", ok, err)
	}
	f.Exec(`UPDATE warmup_plans SET held_cap = $2, overflow = $3, spread_mailbox = $4, engaged_first = true WHERE id = $1`, a.PlanID, a.Sent+60, overflow, spread)
	return postal, ses, a.PlanID.String(), a.Sent
}

func testWarmupOverflow(t *testing.T, f *sendtest.Fixture) {
	postal, ses, plan, before := warmupFixture(t, f, true, true)
	if before != 0 {
		t.Fatalf("Postal already sent %d today", before)
	}
	f.AddContacts(
		sendtest.ContactSpec{N: 50, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 30, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	// The audience build keeps 60 on Postal (today's cap) and sends the
	// rest through SES.
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, postal); n != 60 {
		f.Dump()
		t.Fatalf("%d on Postal after the audience build, want 60", n)
	}
	_, done := f.Drive(20 * time.Minute)
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("send didn't finish: %v %s", done, f.Status())
	}
	// Gmail's part of the cap is 40% (24), Yahoo's 15% (9): the rest of
	// Postal's share went through SES instead of waiting.
	sentOn := func(p uuid.UUID, mailbox string) int64 {
		return f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND mailbox_provider = $3 AND sent_at IS NOT NULL`, f.Campaign, p, mailbox)
	}
	if g, y := sentOn(postal, "gmail"), sentOn(postal, "yahoo"); g != 24 || y != 9 {
		f.Dump()
		t.Fatalf("Postal sent %d Gmail and %d Yahoo, want 24 and 9", g, y)
	}
	if n := sentOn(ses, "gmail") + sentOn(ses, "yahoo"); n != 47 {
		t.Fatalf("SES sent %d, want 47", n)
	}
	if n := f.Count(`SELECT overflowed FROM warmup_plans WHERE id = $1`, plan); n != 27 {
		t.Fatalf("over the cap today: %d, want 27", n)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND metric = 'warmup_cap'`, f.Campaign); n != 0 {
		t.Fatal("an overflow isn't a hold")
	}
}

func testWarmupCapHold(t *testing.T, f *sendtest.Fixture) {
	postal, ses, plan, before := warmupFixture(t, f, false, false)
	f.AddContacts(
		sendtest.ContactSpec{N: 40, Mailbox: "gmail", Domain: "gmail.com", Tier: "cooling", Engaged: true},
		sendtest.ContactSpec{N: 50, Mailbox: "gmail", Domain: "gmail.com", Tier: "active", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, postal); n != 90 {
		f.Dump()
		t.Fatalf("%d on Postal; without overflow the share stays there", n)
	}
	at, done := f.Drive(10 * time.Minute)
	if done {
		f.Dump()
		t.Fatal("the send finished past the cap")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND sent_at IS NOT NULL`, f.Campaign, postal); n != 60 {
		f.Dump()
		t.Fatalf("Postal sent %d, today's cap leaves 60 (%d sent before)", n, before)
	}
	// Most engaged first: every Active contact went out before Cooling.
	if n := f.Count(`SELECT count(*) FROM messages m JOIN contacts c ON c.id = m.contact_id WHERE m.campaign_id = $1 AND m.sent_at IS NOT NULL AND c.tier = 'active'`, f.Campaign); n != 50 {
		t.Fatalf("%d Active contacts sent of 60, want all 50 first", n)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_routes WHERE campaign_id = $1 AND provider_id = $2 AND status = 'held' AND status_detail LIKE '%"reason":"warmup_cap"%'`, f.Campaign, postal); n != 1 {
		f.Dump()
		t.Fatal("Postal's share isn't held at the cap")
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'throttled' AND metric = 'warmup_cap' AND held = 30 AND resolved_at IS NULL`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatal("no warm-up cap hold for 30 recipients")
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE data->>'reason' = 'warmup_cap' AND data->>'campaignId' = $1`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was told about the hold")
	}
	if s := f.Status(); s != "sending" {
		t.Fatalf("campaign %s; a warm-up hold keeps it sending", s)
	}

	// Moving the held share sends it through SES.
	owner := f.Env.Client()
	owner.SignInDemo("linh@acme.io")
	var res warmup.WarmupHoldDecisionResult
	owner.Do("POST", "/workspaces/acme-newsletter/warmup/"+plan+"/held/"+f.Campaign.String(), map[string]any{"action": "move", "targetProviderId": ses.String()}).
		MustStatus(t, 200).JSON(t, &res)
	if res.Moved != 30 {
		t.Fatalf("moved %d", res.Moved)
	}
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, at, 10*time.Minute); err != nil || !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("after the move: done=%v status=%s err=%v", done, f.Status(), err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND sent_at IS NOT NULL`, f.Campaign, ses); n != 30 {
		t.Fatalf("SES sent %d of the held 30", n)
	}
}
