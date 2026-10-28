package routing_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/routing"
)

// R3.U4c: the Campaign tab simulates what the send does on its day, as the
// audience build and the send loop decide it. A warming provider sends only
// its day's engagement tiers (the others go through a warm provider of
// their rule) and no more than its cap, per mailbox provider when its plan
// splits the cap; what's over goes through a warm provider when the plan
// allows overflow, else it waits for the next day's cap, and all of it
// waits while the plan is paused. A provider that can't take mail now sends
// its share through the stream's fallback when the policy and the
// fallback's checks allow it, and holds it otherwise.
func TestCampaignSimulationWarmupAndFailover(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	wsRow, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	id := func(adapter string) uuid.UUID {
		t.Helper()
		var pid uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = $2 AND disconnected_at IS NULL`, wsRow.ID, adapter).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		return pid
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	postal, ses := id("postal"), id("ses")
	allTiers := []string{"active", "cooling", "at_risk", "sunset", ""}
	next := time.Now().Add(36 * time.Hour).Truncate(time.Hour)
	wc := routing.WarmCap{Cap: 1_000_000, Left: 1_000_000, Day: 10, Days: 21, Tiers: allTiers, NextDay: next}
	svc := &routing.Service{Pool: env.App.Pool, Q: env.App.Q, WarmCap: func(_ context.Context, pid uuid.UUID, _ time.Time) (routing.WarmCap, bool) {
		return wc, pid == postal
	}}
	run := func() (routing.RouteCampaignResult, routing.RouteCampaignWarmup, routing.RouteCampaignShare) {
		t.Helper()
		r, err := svc.SimulateCampaign(ctx, &wsRow, routing.RouteCampaignInput{})
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Warmup) != 1 || r.Warmup[0].ProviderID != postal.String() {
			t.Fatalf("warm-up %+v", r.Warmup)
		}
		for _, sh := range r.Shares {
			if sh.ProviderID == postal.String() {
				return r, r.Warmup[0], sh
			}
		}
		t.Fatalf("no Postal share: %+v", r.Shares)
		return r, routing.RouteCampaignWarmup{}, routing.RouteCampaignShare{}
	}

	// Room for everyone: nothing over, nothing diverted.
	_, w, share := run()
	all := share.Count
	if all == 0 || w.Over != 0 || w.Tier != 0 || w.Resumes != nil || w.Spread || w.Day != 10 || w.Cap != 1_000_000 {
		t.Fatalf("plenty of room: share %d, warm-up %+v", all, w)
	}

	// Only the Active tier on this day: the others go through SES, the
	// largest warm share of the split, and don't count toward the cap.
	wc.Tiers = []string{"active"}
	wc.Left = 0
	_, w, _ = run()
	if w.Tier <= 0 || w.Tier >= all || w.TierTo != "Amazon SES" || w.Over != all-w.Tier {
		t.Fatalf("tiers: %d of %d to %q, over %d", w.Tier, all, w.TierTo, w.Over)
	}
	wc.Tiers = allTiers

	// Over the cap without overflow: held until the next day's cap.
	wc.Left = all - 100
	_, w, _ = run()
	if w.Over != 100 || w.To != "" || w.Resumes == nil || !w.Resumes.Equal(next) || w.Left != all-100 {
		t.Fatalf("held at the cap: %+v", w)
	}
	// With overflow: SES takes them, and nothing waits.
	wc.Overflow = true
	_, w, _ = run()
	if w.Over != 100 || w.To != "Amazon SES" || w.Resumes != nil {
		t.Fatalf("overflow: %+v", w)
	}
	wc.Overflow = false

	// The cap split by mailbox provider: Gmail's part is small, so Gmail's
	// recipients go over it while the whole cap has room.
	wc.Left = all + 1000
	parts := map[string]int64{"gmail": 10, "outlook": 1_000_000, "yahoo": 1_000_000, "icloud": 1_000_000, "other": 1_000_000}
	wc.MailboxCaps, wc.MailboxLeft = parts, parts
	_, w, _ = run()
	var gmail routing.RouteCampaignMailboxCap
	var sumOver, sumRecipients int64
	for _, mc := range w.MailboxCaps {
		if mc.Mailbox == "gmail" {
			gmail = mc
		}
		sumOver += mc.Over
		sumRecipients += mc.Recipients
	}
	if !w.Spread || len(w.MailboxCaps) != 5 || gmail.Recipients <= 10 || gmail.Over != gmail.Recipients-10 || w.Over != sumOver || sumRecipients != all {
		t.Fatalf("per mailbox provider: %+v", w)
	}
	wc.MailboxCaps, wc.MailboxLeft = nil, nil

	// A paused plan holds the whole share, until someone resumes it; with
	// overflow on, the share goes through SES instead, as the send does.
	wc.Paused, wc.Left = true, 0
	_, w, _ = run()
	if !w.Paused || w.Over != all || w.Resumes != nil || w.To != "" {
		t.Fatalf("paused: %+v", w)
	}
	wc.Overflow = true
	_, w, _ = run()
	if !w.Paused || w.Over != all || w.Resumes != nil || w.To != "Amazon SES" {
		t.Fatalf("paused with overflow: %+v", w)
	}
	wc.Paused, wc.Left, wc.Overflow = false, 1_000_000, false

	// Postal down: with "pause and ask" its share waits for a decision.
	exec(`UPDATE providers SET status = 'down' WHERE id = $1`, postal)
	_, _, share = run()
	if d := share.Down; d == nil || d.Status != "down" || d.Held != all || d.Moved != 0 || d.Reason != "pause_and_ask" {
		t.Fatalf("down, pause and ask: %+v", share.Down)
	}
	// Automatic failover without a fallback: still waits.
	exec(`INSERT INTO stream_policies (workspace_id, stream, failover) VALUES ($1, 'broadcast', 'automatic')
		ON CONFLICT (workspace_id, stream) DO UPDATE SET failover = EXCLUDED.failover`, wsRow.ID)
	exec(`UPDATE routing_rules SET fallback_provider_id = NULL WHERE workspace_id = $1 AND stream = 'broadcast'`, wsRow.ID)
	_, _, share = run()
	if d := share.Down; d == nil || d.Held != all || d.Reason != "no_fallback" {
		t.Fatalf("down, no fallback: %+v", share.Down)
	}
	// With SES as the fallback, the share goes there.
	exec(`UPDATE routing_rules SET fallback_provider_id = $2 WHERE workspace_id = $1 AND stream = 'broadcast' AND name = ''`, wsRow.ID, ses)
	r, _, share := run()
	if d := share.Down; d == nil || d.Moved != all || d.To != "Amazon SES" || d.Held != 0 {
		t.Fatalf("down, automatic to SES: %+v", share.Down)
	}
	for _, sh := range r.Shares {
		if sh.ProviderID == ses.String() && sh.Down != nil {
			t.Fatal("SES is up: its share doesn't move")
		}
	}
	// A fallback that fails its checks takes nothing: the share waits, and
	// the tester says which check failed.
	exec(`UPDATE providers SET status = 'sandbox' WHERE id = $1`, ses)
	_, _, share = run()
	if d := share.Down; d == nil || d.Held != all || d.Reason != "checks" || len(d.Checks) == 0 {
		t.Fatalf("down, the fallback in the sandbox: %+v", share.Down)
	}

	// Through the API, viewers see the same fields.
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	var api struct {
		Warmup []struct{ ProviderID string }
		Shares []struct {
			ProviderID string
			Down       *struct{ Reason string }
		}
	}
	viewer.Do("POST", ws+"/routing/test-campaign", map[string]any{}).MustStatus(t, 200).JSON(t, &api)
	found := false
	for _, sh := range api.Shares {
		if sh.ProviderID == postal.String() && sh.Down != nil && sh.Down.Reason == "checks" {
			found = true
		}
	}
	if !found || len(api.Warmup) != 1 {
		t.Fatalf("API: %+v", api)
	}
}

// R3.U4a: the impact of unsaved routing covers every stream, not only a
// Broadcasts split: each changed stream's recent recipients (or, without
// any, the contacts' addresses) run through the saved and the unsaved
// routing, and the preview says how many move and where most of them go.
func TestRoutingImpactForEveryStream(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	wsRow, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	provider := func(adapter string) (string, string) {
		t.Helper()
		var id uuid.UUID
		var name string
		if err := env.App.Pool.QueryRow(ctx, `SELECT id, name FROM providers WHERE workspace_id = $1 AND adapter = $2 AND disconnected_at IS NULL`, wsRow.ID, adapter).Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		return id.String(), name
	}
	ses, sesName := provider("ses")
	postmark, postmarkName := provider("postmark")
	postal, _ := provider("postal")
	svc := &routing.Service{Pool: env.App.Pool, Q: env.App.Q}
	impact := func(streams ...routing.RoutingStreamInput) map[string]routing.RouteStreamImpact {
		t.Helper()
		r, err := svc.SimulateCampaign(ctx, &wsRow, routing.RouteCampaignInput{Draft: &routing.RoutingSaveInput{Streams: streams}})
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]routing.RouteStreamImpact{}
		for _, im := range r.Impact {
			out[im.Stream] = im
		}
		return out
	}
	only := func(id string) []routing.RoutingSplit { return []routing.RoutingSplit{{ProviderID: id, Weight: 100}} }
	none := []routing.RoutingCondition{}

	// The demo sends Automations through SES (Postmark the fallback) and
	// System through Postmark (SES the fallback). Swapping Automations moves
	// every recipient to Postmark; a new System fallback moves nobody.
	got := impact(
		routing.RoutingStreamInput{Stream: "automation", Splits: only(postmark), Fallback: ses, Failover: "automatic", Conditions: none},
		routing.RoutingStreamInput{Stream: "system", Splits: only(postmark), Fallback: postal, Failover: "automatic", Conditions: none},
	)
	a, ok := got["automation"]
	if !ok || a.Addresses == 0 || a.Moved != a.Addresses || a.To != postmarkName || a.From != sesName {
		t.Fatalf("Automations to Postmark: %+v", got)
	}
	if sys, ok := got["system"]; !ok || sys.Addresses == 0 || sys.Moved != 0 {
		t.Fatalf("System's fallback only: %+v", got)
	}
	if _, ok := got["transactional"]; ok {
		t.Fatalf("Transactional didn't change: %+v", got)
	}
	// The stream's recipients of the last 30 days are the basis when it
	// mailed anyone, else the contacts' addresses.
	var recent int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE workspace_id = $1 AND stream = 'automation' AND campaign_id IS NULL
		AND queued_at >= now() - interval '30 days'`, wsRow.ID).Scan(&recent); err != nil {
		t.Fatal(err)
	}
	if want := map[bool]string{true: "recent", false: "contacts"}[recent > 0]; a.Basis != want {
		t.Fatalf("basis %q with %d recent automation emails", a.Basis, recent)
	}

	// Half of Automations to Postmark: some move, not all.
	got = impact(routing.RoutingStreamInput{Stream: "automation", Splits: []routing.RoutingSplit{{ProviderID: ses, Weight: 50}, {ProviderID: postmark, Weight: 50}},
		Fallback: postmark, Failover: "automatic", Conditions: none})
	if h := got["automation"]; h.Moved == 0 || h.Moved == h.Addresses || h.To != postmarkName {
		t.Fatalf("a 50/50 split: %+v", h)
	}

	// The saved routing as the draft: nothing to preview.
	if got = impact(routing.RoutingStreamInput{Stream: "automation", Splits: only(ses), Fallback: postmark, Failover: "automatic", Conditions: none}); len(got) != 0 {
		t.Fatalf("no change: %+v", got)
	}
}
