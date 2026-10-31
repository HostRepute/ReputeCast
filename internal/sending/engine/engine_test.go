package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

func TestSending(t *testing.T) {
	s := sendtest.NewSuite(t)
	t.Run("EndToEnd", func(t *testing.T) { testEndToEnd(t, s.Fixture(t, "End to end")) })
	t.Run("AutoPauseOutlookComplaints", func(t *testing.T) { testAutoPause(t, s.Fixture(t, "Auto-pause")) })
	t.Run("PostalOutageHoldsShare", func(t *testing.T) { testOutage(t, s.Fixture(t, "Outage")) })
	t.Run("PauseResumeCancel", func(t *testing.T) { testPauseResumeCancel(t, s.Fixture(t, "Pause")) })
	t.Run("ScheduledStart", func(t *testing.T) { testScheduled(t, s.Fixture(t, "Scheduled")) })
	t.Run("HeldSend", func(t *testing.T) { testHeld(t, s.Fixture(t, "Held")) })
	t.Run("API", func(t *testing.T) { testAPI(t, s.Fixture(t, "API")) })
	t.Run("Tracking", func(t *testing.T) { testTracking(t, s.Fixture(t, "Tracking")) })
	t.Run("DemoSendReset", func(t *testing.T) { testDemoReset(t, s.Fixture(t, "Demo reset")) })
	t.Run("BestTime", func(t *testing.T) { testBestTime(t, s.Fixture(t, "Best time")) })
	t.Run("AudienceFromSeveralLists", func(t *testing.T) { testSeveralLists(t, s.Fixture(t, "Several lists")) })
	t.Run("Languages", func(t *testing.T) { testLanguages(t, s.Fixture(t, "Languages")) })
	t.Run("AutomationLanguages", func(t *testing.T) { testAutomationLanguages(t, s.Fixture(t, "Automation languages")) })
	t.Run("MailboxFromMX", func(t *testing.T) { testMailboxFromMX(t, s.Fixture(t, "Mailbox from MX")) })
}

// testBestTime sends at each recipient's best time: a contact who usually
// opens at a given hour gets the send then (within 24 hours), one without
// opens at the start; a send scheduled that way waits for its time.
func testBestTime(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 6, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	hour := (f.Start.UTC().Hour() + 3) % 24
	// Three of them opened at that hour on several days (one machine fetch
	// at another hour doesn't count).
	f.Exec(`INSERT INTO delivery_events (id, workspace_id, contact_id, type, occurred_at, machine)
		SELECT gen_random_uuid(), c.workspace_id, c.id, 'open', date_trunc('day', $2::timestamptz) - (d || ' days')::interval + make_interval(hours => $3) + interval '7 minutes', false
		FROM (SELECT c.* FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1 ORDER BY c.id LIMIT 3) c, generate_series(1, 4) d`,
		f.List, f.Start, hour)
	f.Exec(`INSERT INTO delivery_events (id, workspace_id, contact_id, type, occurred_at, machine)
		SELECT gen_random_uuid(), c.workspace_id, c.id, 'open', $2::timestamptz - interval '2 days', true
		FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1`, f.List, f.Start)

	later := f.Start.Add(36 * time.Hour)
	f.Exec(`UPDATE campaigns SET schedule = $2 WHERE id = $1`, f.Campaign, fmt.Sprintf(`{"mode":"best_time","at":%q,"timezone":"UTC"}`, later.Format(time.RFC3339)))
	if err := f.E.Start(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if f.Status() != "scheduled" {
		t.Fatalf("a best-time send for later is %s, want scheduled", f.Status())
	}
	f.StartSend(engine.StartOptions{})
	want := f.Start.UTC().Truncate(24 * time.Hour).Add(time.Duration(hour) * time.Hour)
	if want.Before(f.Start) {
		want = want.Add(24 * time.Hour)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND next_attempt_at = $2`, f.Campaign, want); n != 3 {
		f.Dump()
		t.Fatalf("%d recipients due at their usual hour %v, want 3", n, want)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND next_attempt_at IS NULL`, f.Campaign); n != 3 {
		t.Fatalf("%d recipients without opens go at the start, want 3", n)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.FinishesAt == nil || !snap.FinishesAt.Equal(want) {
		t.Fatalf("the monitor's finish time %v, want %v (the last recipient's hour)", snap.FinishesAt, want)
	}
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, want.Sub(f.Start)+time.Minute); err != nil || !done {
		t.Fatalf("the send didn't finish at the last recipient's hour: done=%v err=%v", done, err)
	}
}

// testDemoReset runs the demo command's send, then starts it again with
// --reset: what the first send added outside the campaign (provider daily
// stats, suppressions from its bounces, the bounced contacts' status) is
// taken back, so each run starts from the same demo.
func testDemoReset(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 2000, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	stats := func() int64 {
		return f.Count(`SELECT coalesce(sum(d.sent + d.delivered + d.bounced + d.complained), 0) FROM provider_daily_stats d
			JOIN providers p ON p.id = d.provider_id WHERE p.workspace_id = $1`, f.WS)
	}
	suppressed := func() int64 { return f.Count(`SELECT count(*) FROM suppressions WHERE workspace_id = $1`, f.WS) }
	inactive := func() int64 {
		return f.Count(`SELECT count(*) FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1 AND c.status <> 'active'`, f.List)
	}
	baseStats, baseSuppressed := stats(), suppressed()
	// Without a canary, so three minutes send to everyone (bounces included).
	res, err := f.E.DemoSend(f.Ctx, engine.DemoSendOptions{Campaign: "Demo reset", Elapsed: 3 * time.Minute, Canary: engine.CanaryOff})
	if err != nil {
		t.Fatal(err)
	}
	if res.Recipients != 2000 || res.Sent == 0 {
		t.Fatalf("demo send: %d recipients, %d sent", res.Recipients, res.Sent)
	}
	if stats() <= baseStats {
		t.Fatal("the demo send didn't count in its providers' daily stats")
	}
	bounced := inactive()
	if bounced == 0 || suppressed() != baseSuppressed+bounced {
		t.Fatalf("%d contacts bounced or complained, suppressions went from %d to %d", bounced, baseSuppressed, suppressed())
	}
	if _, err := f.E.DemoSend(f.Ctx, engine.DemoSendOptions{Campaign: "Demo reset", Reset: true}); err != nil {
		t.Fatal(err)
	}
	if got := stats(); got != baseStats {
		t.Fatalf("provider daily stats %d after the reset, want %d", got, baseStats)
	}
	if got := suppressed(); got != baseSuppressed {
		t.Fatalf("%d suppressions after the reset, want %d", got, baseSuppressed)
	}
	if n := inactive(); n != 0 {
		t.Fatalf("%d contacts still bounced or complained after the reset", n)
	}
	if n := f.Count(`SELECT recipients FROM campaign_sends WHERE campaign_id = $1`, f.Campaign); n != 2000 {
		t.Fatalf("the send started again for %d recipients, want all 2,000", n)
	}
	if err := f.E.Cancel(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
}

func testEndToEnd(t *testing.T, f *sendtest.Fixture) {
	f.AddContacts(
		sendtest.ContactSpec{N: 400, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 150, Mailbox: "outlook", Domain: "outlook.com", Engaged: true},
		sendtest.ContactSpec{N: 60, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
		sendtest.ContactSpec{N: 50, Mailbox: "icloud", Domain: "icloud.com", Engaged: true},
		sendtest.ContactSpec{N: 40, Mailbox: "other", Domain: "example.org", Engaged: true},
		sendtest.ContactSpec{N: 5, Mailbox: "gmail", Domain: "gmail.com", Suppress: true},
		sendtest.ContactSpec{N: 3, Mailbox: "gmail", Domain: "gmail.com", Status: "unsubscribed"},
		sendtest.ContactSpec{N: 2, Mailbox: "gmail", Domain: "gmail.com", Paused: true},
		sendtest.ContactSpec{N: 4, Mailbox: "other", Domain: "example.org", Role: true},
		// Left out as Send Readiness counts them: aliases of contacts already
		// in the audience, role addresses known by their name, and contacts
		// whose Broadcasts the sunset policy stopped.
		sendtest.ContactSpec{N: 3, Mailbox: "gmail", Domain: "gmail.com", Alias: true},
		sendtest.ContactSpec{N: 2, Mailbox: "other", Domain: "example.net", RolePattern: true},
		sendtest.ContactSpec{N: 2, Mailbox: "gmail", Domain: "gmail.com", Stopped: true},
	)
	f.StartSend(engine.StartOptions{ABWindow: 2 * time.Minute})

	if s := f.Status(); s != "sending" {
		t.Fatalf("status %q after start, want sending", s)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 700 {
		t.Fatalf("%d messages queued, want 700 (suppressed, unsubscribed, paused, role, duplicate and stopped contacts left out)", n)
	}
	var excluded engine.Excluded
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT excluded FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(raw, &excluded)
	if excluded.Suppressed != 5 || excluded.Unsubscribed != 3 || excluded.Paused != 2 || excluded.Role != 6 || excluded.Duplicates != 3 || excluded.Sunset != 2 {
		t.Fatalf("excluded %+v, want 5 suppressed, 3 unsubscribed, 2 paused, 6 role addresses, 3 duplicates, 2 stopped", excluded)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND email LIKE '%+alias@%'`, f.Campaign); n != 0 {
		t.Fatalf("%d aliases queued; the older contact keeps the mailbox", n)
	}
	// No condition matches news.acme.io's mail: every mailbox provider
	// follows the 90/10 split (strict ones get their own lower caps from
	// Postal's warm-up instead, PR-8).
	if n := f.Count(`SELECT count(*) FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.campaign_id = $1 AND m.mailbox_provider IN ('outlook','icloud') AND p.adapter = 'postal'`, f.Campaign); n == 0 {
		t.Fatal("no Outlook or iCloud recipients on Postal's share")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND route_note LIKE 'Postal share of the 90/10 split, warm-up day %'`, f.Campaign); n == 0 {
		t.Fatal("no route notes on the Postal share")
	}
	if n := f.Count(`SELECT coalesce(sum(planned), 0) FROM campaign_routes WHERE campaign_id = $1`, f.Campaign); n != 700 {
		t.Fatalf("routes plan %d, want 700", n)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_links WHERE campaign_id = $1`, f.Campaign); n == 0 {
		t.Fatal("no tracked links indexed")
	}
	// The A/B sample goes first; the rest waits for the winner.
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held'`, f.Campaign); n == 0 || n >= 700 {
		t.Fatalf("%d held for the A/B winner", n)
	}

	_, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 25*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("send didn't finish: done=%v status=%s", done, f.Status())
	}
	if n := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign); n != 700 {
		t.Fatalf("counters say %d sent, want 700", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); n != 700 {
		t.Fatalf("%d messages have sent_at, want 700", n)
	}
	if n := f.Count(`SELECT count(DISTINCT provider_message_id) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 700 {
		t.Fatalf("%d distinct provider message ids, want 700 (nothing sent twice)", n)
	}
	if n := f.Count(`SELECT coalesce(sum(delivered), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign); n < 650 {
		t.Fatalf("only %d delivered", n)
	}
	if n := f.Count(`SELECT coalesce(sum(opened_human + opened_machine), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign); n == 0 {
		t.Fatal("no opens recorded")
	}
	if n := f.Count(`SELECT count(*) FROM delivery_events WHERE campaign_id = $1 AND type = 'open' AND machine`, f.Campaign); n == 0 {
		t.Fatal("no machine opens classified (Apple Mail Privacy Protection)")
	}
	var winner string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT ab->>'winner' FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&winner); err != nil || winner == "" {
		t.Fatalf("A/B winner %q: %v", winner, err)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('campaign.send_started', 'campaign.completed', 'campaign.ab_winner_picked')`, f.Campaign.String()); n != 3 {
		t.Fatalf("%d audit entries for start, A/B winner and finish, want 3", n)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'send_finished' AND data->>'campaignId' = $2`, f.Sofia, f.Campaign.String()); n != 1 {
		t.Fatalf("%d send_finished notifications for the starter", n)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.State != "finished" || snap.Sent != 700 || snap.StartedBy != "Sofia Rossi" {
		t.Fatalf("snapshot %s sent=%d by %q", snap.State, snap.Sent, snap.StartedBy)
	}
	if len(snap.Mailboxes) != 5 || len(snap.Providers) != 2 {
		t.Fatalf("snapshot rows: %d mailbox, %d provider", len(snap.Mailboxes), len(snap.Providers))
	}
	for _, ev := range snap.Events {
		if ev.Address == "" || len(ev.Address) > 40 || !containsDots(ev.Address) {
			t.Fatalf("event address not masked: %q", ev.Address)
		}
	}
}

func containsDots(s string) bool {
	for _, r := range s {
		if r == '·' {
			return true
		}
	}
	return false
}

func testAutoPause(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		// Outlook sends at 5 /s, so 2,500 take over eight minutes: complaints
		// arrive one to six minutes after delivery and trip the threshold
		// while most of the share is still queued.
		sendtest.ContactSpec{N: 2500, Mailbox: "outlook", Domain: "hotmail.com", Tier: "at_risk", Source: "import:mailchimp"},
		sendtest.ContactSpec{N: 600, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{Scenario: "outlook-complaints"})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 20*time.Minute); err != nil {
		t.Fatal(err)
	}
	var kind, scope, mailbox, metric string
	var value, limit float64
	var held int64
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT kind, scope, mailbox_provider, metric, value, threshold, held FROM campaign_incidents WHERE campaign_id = $1 ORDER BY tripped_at LIMIT 1`, f.Campaign).
		Scan(&kind, &scope, &mailbox, &metric, &value, &limit, &held); err != nil {
		f.Dump()
		t.Fatalf("no incident: %v", err)
	}
	if kind != "threshold" || scope != "mailbox_provider" || mailbox != "outlook" || metric != "complaint" || value <= limit {
		f.Dump()
		t.Fatalf("incident %s %s %s %s %.2f/%.2f", kind, scope, mailbox, metric, value, limit)
	}
	if held == 0 {
		t.Fatal("nothing held")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'gmail' AND status IN ('queued','held')`, f.Campaign); n != 0 {
		f.Dump()
		t.Fatalf("%d Gmail recipients left; the rest of the campaign should have finished", n)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident == nil || snap.Incident.Detail.Changes == nil || len(snap.Incident.Detail.Options) != 3 {
		t.Fatalf("snapshot incident %+v", snap.Incident)
	}
	if len(snap.Incident.Detail.Causes) == 0 || snap.Incident.Detail.Note == "" {
		t.Fatal("incident has no likely causes or note")
	}
	checkAutoPauseLanguages(t, f, snap)
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.auto_paused'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d auto-pause audit entries", n)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was told about the auto-pause")
	}
	res, err := f.E.ResumeScope(f.Ctx, f.Campaign, f.Sofia, engine.ResumeRequest{Mailbox: "outlook", WithChanges: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Excluded == 0 || res.Rate == 0 {
		t.Fatalf("resume with changes excluded %d at %.0f /s", res.Excluded, res.Rate)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND resolution = 'resumed_with_changes'`, f.Campaign); n != 1 {
		t.Fatal("incident not resolved as resumed with changes")
	}
	_, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(20*time.Minute), 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		f.Dump()
		t.Fatalf("send didn't finish after resuming: %s", f.Status())
	}
}

func testOutage(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 3000, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{Scenario: "postal-outage"})
	// The scenario takes Postal down 14 minutes into the send; start the
	// clock 13 minutes in.
	f.Exec(`UPDATE campaigns SET started_at = $2 WHERE id = $1`, f.Campaign, f.Start.Add(-13*time.Minute))
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 8*time.Minute); err != nil {
		t.Fatal(err)
	}
	var routeStatus string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT cr.status FROM campaign_routes cr JOIN providers p ON p.id = cr.provider_id WHERE cr.campaign_id = $1 AND p.adapter = 'postal'`, f.Campaign).Scan(&routeStatus); err != nil {
		t.Fatal(err)
	}
	if routeStatus != "held" {
		f.Dump()
		t.Fatalf("Postal route is %s, want held", routeStatus)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident == nil || snap.Incident.Kind != "provider_outage" || snap.Incident.Detail.ErrorKind != "connection errors" {
		t.Fatalf("incident %+v", snap.Incident)
	}
	d := snap.Incident.Detail
	if d.Move == nil || d.Move.TargetShort != "SES" || d.Policy != "pause_and_ask" || len(d.Log) == 0 || d.Remaining == 0 {
		t.Fatalf("outage detail %+v", d)
	}
	checkOutageLanguages(t, f, snap)
	// With only the held share left, the send stays on to test Postal by
	// itself until someone decides (review GS-3).
	if f.Status() != "sending" || snap.State != "partial" {
		f.Dump()
		t.Fatalf("status %s (%s); with only the held share left the send keeps testing Postal", f.Status(), snap.State)
	}
	for _, p := range snap.Providers {
		if p.Adapter == "postal" && (!p.Probing || p.NextProbe == nil) {
			t.Fatalf("the held Postal share isn't testing its provider: %+v", p)
		}
	}
	moved, err := f.E.MoveShare(f.Ctx, f.Campaign, f.Sofia, f.ProviderID("postal"), f.ProviderID("ses"))
	if err != nil {
		t.Fatal(err)
	}
	if moved != d.Remaining {
		t.Fatalf("moved %d, the incident held %d", moved, d.Remaining)
	}
	_, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(8*time.Minute), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("send didn't finish after the move: done=%v %s", done, f.Status())
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('campaign.share_held', 'campaign.share_moved')`, f.Campaign.String()); n != 2 {
		t.Fatalf("%d audit entries for hold and move, want 2", n)
	}
}

func testPauseResumeCancel(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 600, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := f.E.Pause(f.Ctx, f.Campaign, f.Sofia, "Checking a typo"); err != nil {
		t.Fatal(err)
	}
	sentAtPause := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign)
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(10*time.Second), 30*time.Second); err != nil || !done {
		t.Fatalf("a paused send kept running: done=%v err=%v", done, err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); n != sentAtPause {
		t.Fatalf("%d sent while paused", n-sentAtPause)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.State != "paused" || snap.PausedBy != "Sofia Rossi" || snap.PauseReason != "Checking a typo" {
		t.Fatalf("paused snapshot %s by %q (%q)", snap.State, snap.PausedBy, snap.PauseReason)
	}
	if err := f.E.Resume(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(40*time.Second), 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := f.E.Cancel(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if f.Status() != "cancelled" {
		t.Fatalf("status %s after cancel", f.Status())
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status IN ('queued', 'held')`, f.Campaign); n != 0 {
		t.Fatalf("%d messages still queued after cancel", n)
	}
	for _, action := range []string{"campaign.paused", "campaign.resumed", "campaign.cancelled"} {
		if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = $2`, f.Campaign.String(), action); n != 1 {
			t.Fatalf("%d %s audit entries", n, action)
		}
	}
	if err := f.E.Pause(f.Ctx, f.Campaign, f.Sofia, ""); err == nil {
		t.Fatal("pausing a cancelled send must fail")
	}
}

func testScheduled(t *testing.T, f *sendtest.Fixture) {
	f.AddContacts(sendtest.ContactSpec{N: 20, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	at := f.Start.Add(48 * time.Hour)
	f.Exec(`UPDATE campaigns SET schedule = $2 WHERE id = $1`, f.Campaign, fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, at.Format(time.RFC3339)))
	if err := f.E.Start(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if f.Status() != "scheduled" {
		t.Fatalf("status %s, want scheduled", f.Status())
	}
	if n := f.Count(`SELECT count(*) FROM river_job WHERE kind = 'sending.start' AND args->>'campaign_id' = $1 AND scheduled_at = $2`, f.Campaign.String(), at); n != 1 {
		t.Fatalf("%d start jobs at the scheduled time", n)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.scheduled'`, f.Campaign.String()); n != 1 {
		t.Fatal("scheduling wasn't audited")
	}
	if err := f.E.Cancel(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if f.Status() != "draft" {
		t.Fatalf("unscheduling left %s, want draft", f.Status())
	}
	if n := f.Count(`SELECT count(*) FROM campaigns WHERE id = $1 AND scheduled_at IS NULL`, f.Campaign); n != 1 {
		t.Fatal("unscheduling kept the time")
	}
	// An approved campaign stays approved when it's taken back (it can be
	// scheduled again as approved), as Review & send's Unschedule does.
	f.Exec(`INSERT INTO approval_requests (id, workspace_id, campaign_id, requested_by, status, snapshot) VALUES ($1, $2, $3, $4, 'approved', '{}')`,
		uuid.Must(uuid.NewV7()), f.WS, f.Campaign, f.Sofia)
	if err := f.E.Start(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if err := f.E.Cancel(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if f.Status() != "approved" {
		t.Fatalf("unscheduling an approved campaign left %s, want approved", f.Status())
	}
}

// fixedRecheck is a send-time recheck with a set answer.
type fixedRecheck struct{ rc sending.Recheck }

func (r fixedRecheck) Recheck(context.Context, uuid.UUID) (sending.Recheck, error) { return r.rc, nil }

// countedRecheck answers like fixedRecheck and counts the calls.
type countedRecheck struct {
	rc    sending.Recheck
	calls *int
}

func (r countedRecheck) Recheck(context.Context, uuid.UUID) (sending.Recheck, error) {
	*r.calls++
	return r.rc, nil
}

// testHeld: a scheduled send whose send-time recheck turns Blocked is held
// the way Review & send shows it. It stays scheduled, nothing is sent, the
// author and the approvers are told once, the watchdog leaves it to Review
// & send's minute job, and it starts (the hold ends) once a recheck passes.
func testHeld(t *testing.T, f *sendtest.Fixture) {
	saved := f.E.Readiness
	defer func() { f.E.Readiness = saved }()
	f.AddContacts(sendtest.ContactSpec{N: 20, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		// A stopped contact and its newer alias: the mailbox still gets the send.
		sendtest.ContactSpec{N: 1, Mailbox: "gmail", Domain: "gmail.com", Engaged: true, StoppedPair: true})
	hook := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO webhooks (id, workspace_id, url, events, secret_enc, demo) VALUES ($1, $2, 'https://hooks.example.test/w12', '{campaign.*}', '\x00', true)`, hook, f.WS)
	at := f.Start.Add(-time.Minute)
	f.Exec(`UPDATE campaigns SET status = 'scheduled', scheduled_at = $2, schedule = $3, updated_by = $4 WHERE id = $1`, f.Campaign, at,
		fmt.Sprintf(`{"mode":"at","at":%q,"timezone":"America/New_York"}`, at.Format(time.RFC3339)), f.Sofia)

	blocked := sending.Recheck{Verdict: "blocked", Blocking: []string{"Sender domain fails DMARC", "Approval is still pending"}, Audience: 21, RunID: "run-7"}
	f.E.Readiness = fixedRecheck{blocked}
	held := func() {
		t.Helper()
		err := f.E.StartWith(f.Ctx, f.Campaign, engine.StartOptions{By: f.Sofia, Now: true, Scheduled: true})
		var nr *sending.NotReadyError
		if !errors.Is(err, sending.ErrNotReady) || !errors.As(err, &nr) || nr.Reason != "Sender domain fails DMARC" {
			t.Fatalf("start: %v, want ErrNotReady with the first block", err)
		}
		if f.Status() != "scheduled" {
			t.Fatalf("held send is %s, want scheduled", f.Status())
		}
	}
	held()
	var raw []byte
	var heldCount int64
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT detail, held FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'recheck_blocked' AND resolved_at IS NULL`, f.Campaign).Scan(&raw, &heldCount); err != nil {
		t.Fatalf("no open hold: %v", err)
	}
	var d struct {
		Reason   string   `json:"reason"`
		Blocking []string `json:"blocking"`
		RunID    string   `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &d); err != nil || d.Reason != "Sender domain fails DMARC" || len(d.Blocking) != 2 || d.RunID != "run-7" {
		t.Fatalf("hold detail %s: %v", raw, err)
	}
	// The hold counts the recipients the recheck counted, not zero.
	if heldCount != 21 {
		t.Fatalf("hold holds %d recipients, want the recheck's 21", heldCount)
	}
	told := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_blocked' AND data->>'campaign_id' = $1`, f.Campaign.String())
	if told == 0 || f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_blocked' AND user_id = $1 AND title = 'Send held' AND data->>'campaign_id' = $2`, f.Sofia, f.Campaign.String()) != 1 {
		t.Fatalf("%d people told, and the author should be one of them", told)
	}
	// The next recheck is Blocked too: still one hold, nobody told twice.
	held()
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'recheck_blocked'`, f.Campaign); n != 1 {
		t.Fatalf("%d holds, want 1", n)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_blocked' AND data->>'campaign_id' = $1`, f.Campaign.String()); n != told {
		t.Fatalf("people told again: %d, then %d", told, n)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.send_held'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d send_held audit entries, want 1", n)
	}
	now := f.Start
	due, err := f.E.Q.SendingDueScheduled(f.Ctx, &now)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range due {
		if id == f.Campaign {
			t.Fatal("the watchdog restarts a held send; Review & send's minute job rechecks it")
		}
	}
	// A recheck passes: Review & send's minute job starts it, with no one
	// behind it. It hands over the recheck it ran, and the engine doesn't
	// run Send Readiness again (which would block here).
	calls := 0
	f.E.Readiness = countedRecheck{blocked, &calls}
	ctx := sending.WithRecheck(f.Ctx, f.Campaign, sending.Recheck{Verdict: "warnings"})
	if err := f.E.Start(ctx, f.Campaign, uuid.Nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("Send Readiness ran %d more times after the caller's recheck", calls)
	}
	if f.Status() != "sending" {
		t.Fatalf("status %s after the recheck passed, want sending", f.Status())
	}
	var resolution string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT resolution FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'recheck_blocked'`, f.Campaign).Scan(&resolution); err != nil || resolution != "auto_recovered" {
		t.Fatalf("hold resolution %q: %v", resolution, err)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.send_started' AND actor_type = 'system'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d system send_started entries, want 1", n)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_sends WHERE campaign_id = $1 AND started_by = $2 AND started_by_name = 'Sofia Rossi'`, f.Campaign, f.Sofia); n != 1 {
		t.Fatal("the send isn't started on behalf of who scheduled it")
	}
	if err := f.E.DrivePrepare(f.Ctx, f.Campaign, f.Start); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 21 {
		t.Fatalf("%d recipients, want 21", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND email LIKE '%+pair@%'`, f.Campaign); n != 1 {
		t.Fatal("the stopped contact's alias should get the send for their mailbox")
	}
	// The workspace's endpoints heard about both.
	for event, n := range map[string]int64{"campaign.paused": 1, "campaign.started": 1} {
		if got := f.Count(`SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND event = $2 AND payload->'data'->>'campaign_id' = $3`, hook, event, f.Campaign.String()); got != n {
			t.Fatalf("%d %s deliveries, want %d", got, event, n)
		}
	}
	if err := f.E.Cancel(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if got := f.Count(`SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND event = 'campaign.completed' AND payload->'data'->>'status' = 'cancelled'`, hook); got != 1 {
		t.Fatalf("%d campaign.completed deliveries for the cancelled send, want 1", got)
	}
}

// testSeveralLists builds an audience from two overlapping lists over more
// than one page: every member gets the send once.
func testSeveralLists(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 3500, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	second := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, 'Send test second list')`, second, f.WS)
	// The second list: the last 1,000 members (who leave the first list)
	// and 1,000 from the middle (who stay on both).
	f.Exec(`WITH moved AS (
		    DELETE FROM list_memberships WHERE list_id = $1 AND contact_id IN (SELECT contact_id FROM list_memberships WHERE list_id = $1 ORDER BY contact_id DESC LIMIT 1000)
		    RETURNING contact_id
		)
		INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at) SELECT $2, contact_id, 'subscribed', now() FROM moved`, f.List, second)
	f.Exec(`INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at)
		SELECT $2, contact_id, 'subscribed', now() FROM list_memberships WHERE list_id = $1 ORDER BY contact_id OFFSET 1000 LIMIT 1000`, f.List, second)
	if n := f.Count(`SELECT count(*) FROM list_memberships WHERE list_id = $1`, f.List); n != 2500 {
		t.Fatalf("first list has %d members", n)
	}
	f.Exec(`UPDATE campaigns SET audience = $2 WHERE id = $1`, f.Campaign, fmt.Sprintf(`{"lists":[%q,%q]}`, f.List, second))
	f.StartSend(engine.StartOptions{})
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 3500 {
		t.Fatalf("%d recipients, want 3,500", n)
	}
	if n := f.Count(`SELECT count(DISTINCT contact_id) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 3500 {
		t.Fatalf("%d distinct recipients, want 3,500 (each member once)", n)
	}
	if err := f.E.Cancel(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
}

// testLanguages gives each recipient the language version for them: their
// own when it is translated, otherwise the fallback (English, the source),
// with the reason recorded on the message.
func testLanguages(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.Exec(`INSERT INTO workspace_languages (workspace_id, code, page_copy_status) VALUES ($1, 'es', 'live'), ($1, 'de', 'live'), ($1, 'vi', 'not_set_up')
		ON CONFLICT (workspace_id, code) DO UPDATE SET removed_at = NULL`, f.WS)
	f.AddContacts(sendtest.ContactSpec{N: 6, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	for i, lang := range []string{"es-MX", "de-AT", "", "pt-BR", "vi", "en"} {
		f.Exec(`UPDATE contacts SET language = $2 WHERE id = (SELECT c.id FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1 ORDER BY c.email OFFSET $3 LIMIT 1)`,
			f.List, lang, i)
	}
	// Each version is the source translated: its own headline and docs link,
	// and the footer every version needs (Send Readiness checks each one).
	add := func(lang, status, subject string) {
		f.Exec(`INSERT INTO content_translations (id, workspace_id, owner_kind, owner_id, language, status, subject, mjml, html, plain_text)
			SELECT gen_random_uuid(), $1, 'campaign', c.id, $3, $4, $5::text,
			       replace(replace(c.mjml, 'Three launches, one deprecation', $5::text), 'https://docs.acme.io/flags', 'https://acme.io/' || $3::text),
			       replace(replace(c.html, 'Three launches, one deprecation', $5::text), 'https://docs.acme.io/flags', 'https://acme.io/' || $3::text),
			       c.plain_text
			FROM campaigns c WHERE c.id = $2`,
			f.WS, f.Campaign, lang, status, subject)
	}
	add("es", "translated", "Hola")
	add("de", "translated", "Hallo")
	add("vi", "draft", "Xin chào")
	f.StartSend(engine.StartOptions{})
	type row struct{ lang, contact, reason string }
	want := map[row]int64{{"es", "es", "match"}: 1, {"de", "de", "match"}: 1, {"en", "", "no_language"}: 1, {"en", "pt", "not_offered"}: 1, {"en", "vi", "draft"}: 1, {"en", "en", "match"}: 1}
	for r, n := range want {
		if got := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND language = $2 AND contact_language = $3 AND language_reason = $4`, f.Campaign, r.lang, r.contact, r.reason); got != n {
			f.Dump()
			rows, _ := f.Env.App.Pool.Query(f.Ctx, `SELECT m.email, m.language, m.contact_language, m.language_reason, c.language FROM messages m JOIN contacts c ON c.id = m.contact_id WHERE m.campaign_id = $1`, f.Campaign)
			for rows.Next() {
				var e, l, cl, re, raw string
				_ = rows.Scan(&e, &l, &cl, &re, &raw)
				t.Logf("message %s: %q %q %q (contact %q)", e, l, cl, re, raw)
			}
			rows.Close()
			t.Fatalf("%+v: %d messages, want %d", r, got, n)
		}
	}
	// Each version's links are numbered after the source's.
	if n := f.Count(`SELECT count(*) FROM campaign_links WHERE campaign_id = $1 AND url IN ('https://acme.io/es', 'https://acme.io/de')`, f.Campaign); n != 2 {
		t.Fatalf("%d language version links indexed, want 2", n)
	}
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 10*time.Minute); err != nil || !done {
		t.Fatalf("the send didn't finish: done=%v err=%v", done, err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); n != 6 {
		t.Fatalf("%d sent, want 6", n)
	}
}
