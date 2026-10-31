package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/store"
)

// DemoSendOptions start a demo send (development only).
type DemoSendOptions struct {
	Workspace string
	Campaign  string
	Scenario  string
	// Elapsed fast-forwards the send: it starts this long ago and the part
	// it would have sent is sent at once on a virtual clock, with back-dated
	// times, so the monitor opens mid-send.
	Elapsed time.Duration
	// ABWindow is the A/B test's window for the demo (the campaign's own
	// window, hours, would hold the rest of the audience that long).
	ABWindow time.Duration
	// Reset clears an earlier send of the campaign first.
	Reset bool
	// Canary is the canary choice (auto, on, off); "" runs one in the
	// normal scenario, as on the LiveMonitor and LiveMonitorCanary boards,
	// and none in the others.
	Canary string
}

// DemoSendResult describes the started send.
type DemoSendResult struct {
	CampaignID uuid.UUID
	Name       string
	Recipients int64
	Sent       int64
	Path       string
}

// DemoABWindow lets the A/B sample of a demo send finish before the winner
// goes to the rest (about 7 minutes for Weekly Digest #142).
const DemoABWindow = 7 * time.Minute

// DemoSend starts a campaign's send through the demo transport as Sofia
// Rossi, with Marcus Chen's approval recorded (Oct 10 16:20 on the boards),
// so the live monitor's boards are reproduced with real moving data. It
// refuses unless every provider the Broadcasts route uses is a demo
// provider: it never sends real mail.
func (e *Engine) DemoSend(ctx context.Context, o DemoSendOptions) (DemoSendResult, error) {
	if o.Scenario == "" {
		o.Scenario = simulator.ScenarioNormal
	}
	if !slices.Contains(simulator.Scenarios, o.Scenario) {
		return DemoSendResult{}, fmt.Errorf("unknown scenario %q (use %s)", o.Scenario, strings.Join(simulator.Scenarios, ", "))
	}
	if o.Workspace == "" {
		o.Workspace = "acme-newsletter"
	}
	if o.ABWindow <= 0 {
		o.ABWindow = DemoABWindow
	}
	var ws store.Workspace
	if err := e.Pool.QueryRow(ctx, `SELECT id, slug, timezone FROM workspaces WHERE slug = $1 AND deleted_at IS NULL`, o.Workspace).Scan(&ws.ID, &ws.Slug, &ws.Timezone); err != nil {
		return DemoSendResult{}, fmt.Errorf("workspace %q not found: run `reputecast seed` first", o.Workspace)
	}
	var c store.Campaign
	if err := e.Pool.QueryRow(ctx, `SELECT id, name, status FROM campaigns WHERE workspace_id = $1 AND lower(name) = lower($2) AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`,
		ws.ID, strings.TrimSpace(o.Campaign)).Scan(&c.ID, &c.Name, &c.Status); err != nil {
		return DemoSendResult{}, fmt.Errorf("campaign %q not found in %s", o.Campaign, o.Workspace)
	}
	provs, err := e.Q.SendingWorkspaceProviders(ctx, ws.ID)
	if err != nil {
		return DemoSendResult{}, err
	}
	for _, p := range provs {
		if !manage.IsDemo(p) {
			return DemoSendResult{}, fmt.Errorf("%s is a real provider; demo sends only run in a workspace whose providers are all demo providers", p.Name)
		}
	}
	reset := false
	switch c.Status {
	case "draft", "approval", "approved", "scheduled":
	default:
		if !o.Reset {
			return DemoSendResult{}, fmt.Errorf("%q is %s; pass --reset to clear that send and start again", c.Name, c.Status)
		}
		// Stop the earlier send before clearing it: the server's send loop
		// sees the campaign isn't sending and lets go of it (below), and
		// the demo transport has nothing more to report for it.
		if _, err := e.Pool.Exec(ctx, `UPDATE campaigns SET status = 'cancelled' WHERE id = $1`, c.ID); err != nil {
			return DemoSendResult{}, err
		}
		if _, err := e.Pool.Exec(ctx, `DELETE FROM sending_sim_events WHERE campaign_id = $1`, c.ID); err != nil {
			return DemoSendResult{}, err
		}
		reset = true
	}
	people := map[string]uuid.UUID{}
	rows, err := e.Pool.Query(ctx, `SELECT email, id FROM users WHERE email IN ('sofia@acme.io', 'marcus@acme.io', 'linh@acme.io')`)
	if err != nil {
		return DemoSendResult{}, err
	}
	for rows.Next() {
		var email string
		var id uuid.UUID
		if err := rows.Scan(&email, &id); err != nil {
			rows.Close()
			return DemoSendResult{}, err
		}
		people[email] = id
	}
	rows.Close()
	sofia, marcus, linh := people["sofia@acme.io"], people["marcus@acme.io"], people["linh@acme.io"]
	if sofia == uuid.Nil || marcus == uuid.Nil {
		return DemoSendResult{}, errors.New("the demo team (Sofia Rossi, Marcus Chen) is missing: run `reputecast seed` first")
	}
	now := e.now()
	approvedAt := designTime(now, 2, 16, 20)
	start := now.Add(-o.Elapsed)
	// The command builds the audience and fast-forwards the send itself on the
	// virtual clock, holding the send's lock throughout, so the server's worker
	// (woken by the start) waits until it's done. A worker still running an
	// earlier send lets go within seconds of seeing it stopped.
	var unlock func()
	for wait := time.Now().Add(90 * time.Second); ; {
		if unlock, err = e.lockSend(ctx, c.ID); err == nil {
			break
		}
		if !errors.Is(err, errBusy) || time.Now().After(wait) {
			return DemoSendResult{}, fmt.Errorf("%q is being sent by another process; try again in a minute", c.Name)
		}
		select {
		case <-ctx.Done():
			return DemoSendResult{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	if reset {
		// Opens and clicks the demo transport handed to the tracker just
		// before it stopped are written in the background: let them land,
		// then clear everything.
		select {
		case <-ctx.Done():
			unlock()
			return DemoSendResult{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		if err := e.resetDemo(ctx, c.ID); err != nil {
			unlock()
			return DemoSendResult{}, err
		}
	}
	prev := e.Now
	e.Now = func() time.Time { return start }
	canary := o.Canary
	if canary == "" {
		canary = CanaryOff
		if o.Scenario == "" || o.Scenario == "normal" {
			canary = CanaryOn
		}
	}
	err = e.StartWith(ctx, c.ID, StartOptions{By: sofia, Now: true, Scenario: o.Scenario, ABWindow: o.ABWindow, ApprovedBy: &marcus, ApprovedAt: &approvedAt,
		ScheduledBy: &linh, Canary: canary})
	e.Now = prev
	if err == nil {
		err = e.prepare(ctx, c.ID, start)
	}
	if err == nil && o.Elapsed > 0 {
		ids := make([]uuid.UUID, 0, len(provs))
		for _, p := range provs {
			ids = append(ids, p.ID)
		}
		t, end := start, now
		for {
			// The fast-forward replays seconds that already passed: free the
			// demo providers' rate windows over them first, or a real-time
			// send in those seconds would leave it no room.
			if err = e.Q.SendingClearWindows(ctx, store.SendingClearWindowsParams{ProviderIds: ids, FromSec: t.Unix() - 1, ToSec: end.Unix() + 1}); err != nil {
				break
			}
			if err = e.Q.SendingClearMailboxWindows(ctx, store.SendingClearMailboxWindowsParams{WorkspaceID: ws.ID, FromSec: t.Unix() - 10, ToSec: end.Unix() + 1}); err != nil {
				break
			}
			var done bool
			if t, done, err = e.drive(ctx, c.ID, t, end.Sub(t)); err != nil || done {
				break
			}
			// Fast-forwarding took real time too: catch up with it, so the
			// server's worker carries on from now without a gap.
			if e.now().Sub(end) < 2*time.Second {
				break
			}
			end = e.now()
		}
	}
	unlock()
	if err != nil {
		return DemoSendResult{}, err
	}
	e.kick(ctx, c.ID)
	e.insert(ctx, SimArgs{}, nil)
	out := DemoSendResult{CampaignID: c.ID, Name: c.Name, Path: liveLink(ws.Slug, c.ID)}
	_ = e.Pool.QueryRow(ctx, `SELECT recipients FROM campaign_sends WHERE campaign_id = $1`, c.ID).Scan(&out.Recipients)
	_ = e.Pool.QueryRow(ctx, `SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, c.ID).Scan(&out.Sent)
	return out, nil
}

// designTime is hour:minute (New York) daysAgo days before the demo's
// "today", as the demo seeder dates the boards' events.
func designTime(now time.Time, daysAgo, hour, minute int) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	n := now.In(loc).Add(-11 * time.Hour)
	return time.Date(n.Year(), n.Month(), n.Day(), hour, minute, 0, 0, loc).AddDate(0, 0, -daysAgo)
}

// resetDemo clears an earlier demo send of a campaign so it can start again.
func (e *Engine) resetDemo(ctx context.Context, id uuid.UUID) error {
	e.forgetTemplates(id)
	e.mu.Lock()
	for k := range e.calls {
		if k.campaign == id {
			delete(e.calls, k)
		}
	}
	e.mu.Unlock()
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		for _, q := range []string{
			// Take back what the earlier send added outside the campaign, so a
			// scenario runs again from the same demo: the providers' daily
			// counts (Postal's warm-up cap counts them) and the suppressions,
			// with the contacts' bounced or complained status, that its bounces
			// and complaints created. Opens on contacts, call counts and costs stay.
			`UPDATE provider_daily_stats d SET sent = greatest(d.sent - x.n, 0)
			 FROM (SELECT provider_id, sent_at::date AS day, count(*) AS n FROM messages
			       WHERE campaign_id = $1 AND sent_at IS NOT NULL AND provider_id IS NOT NULL GROUP BY 1, 2) x
			 WHERE d.provider_id = x.provider_id AND d.day = x.day`,
			`UPDATE provider_daily_stats d SET delivered = greatest(d.delivered - x.delivered, 0),
			     bounced = greatest(d.bounced - x.bounced, 0), complained = greatest(d.complained - x.complained, 0),
			     delivered_reporting = greatest(d.delivered_reporting - x.reporting, 0)
			 FROM (SELECT provider_id, occurred_at::date AS day, count(*) FILTER (WHERE type = 'delivered') AS delivered,
			              count(*) FILTER (WHERE type = 'delivered' AND mailbox_provider NOT IN ('gmail', 'icloud')) AS reporting,
			              count(*) FILTER (WHERE type = 'hard_bounce') AS bounced, count(*) FILTER (WHERE type = 'complaint') AS complained
			       FROM delivery_events WHERE campaign_id = $1 AND provider_id IS NOT NULL GROUP BY 1, 2) x
			 WHERE d.provider_id = x.provider_id AND d.day = x.day`,
			`WITH ev AS (
			     SELECT workspace_id, recipient, provider_id, occurred_at, type FROM delivery_events
			     WHERE campaign_id = $1 AND type IN ('hard_bounce', 'complaint') AND recipient <> ''
			 ), gone AS (
			     DELETE FROM suppressions s USING ev
			     WHERE s.workspace_id = ev.workspace_id AND s.email = ev.recipient AND s.source = 'webhook'
			       AND s.source_provider_id = ev.provider_id AND s.created_at = ev.occurred_at
			       AND s.reason = CASE ev.type WHEN 'hard_bounce' THEN 'hard_bounce' ELSE 'complaint' END
			     RETURNING s.workspace_id, s.email, s.reason
			 )
			 UPDATE contacts c SET status = 'active', updated_at = now()
			 FROM gone WHERE c.workspace_id = gone.workspace_id AND c.email = gone.email
			   AND c.status = CASE gone.reason WHEN 'hard_bounce' THEN 'bounced' ELSE 'complained' END`,
			// Moving a held share noted it in the warming provider's plan (and,
			// before warm-up plans moved by counted days, shifted it by a day).
			`WITH shifted AS (
			     DELETE FROM warmup_events ev USING warmup_plans p, campaigns c
			     WHERE c.id = $1 AND c.started_at IS NOT NULL AND ev.plan_id = p.id
			       AND p.provider_id IN (SELECT provider_id FROM campaign_routes WHERE campaign_id = $1)
			       AND ev.kind = 'settings' AND ev.title IN ('Schedule shifted by a day.', 'Share moved during an outage.') AND ev.occurred_at >= c.started_at
			     RETURNING ev.plan_id, ev.title
			 )
			 UPDATE warmup_plans p SET start_date = p.start_date - 1
			 FROM (SELECT DISTINCT plan_id FROM shifted WHERE title = 'Schedule shifted by a day.') s WHERE p.id = s.plan_id`,
			`DELETE FROM sending_sim_events WHERE campaign_id = $1`,
			`DELETE FROM delivery_events WHERE campaign_id = $1`,
			`DELETE FROM tracking_link_clicks WHERE message_id IN (SELECT id FROM messages WHERE campaign_id = $1)`,
			`DELETE FROM messages WHERE campaign_id = $1`,
			`DELETE FROM campaign_counters WHERE campaign_id = $1`,
			`DELETE FROM campaign_hourly WHERE campaign_id = $1`,
			`DELETE FROM campaign_routes WHERE campaign_id = $1`,
			`DELETE FROM campaign_incidents WHERE campaign_id = $1`,
			`DELETE FROM campaign_links WHERE campaign_id = $1`,
			`DELETE FROM campaign_sends WHERE campaign_id = $1`,
			`DELETE FROM incidents WHERE campaign_id = $1`,
			// An earlier send decided the A/B test on the campaign itself.
			`UPDATE campaigns SET ab_test = ab_test - 'winner' WHERE id = $1 AND ab_test IS NOT NULL`,
			`UPDATE campaigns SET status = 'scheduled', started_at = NULL, started_by = NULL, completed_at = NULL, updated_at = now() WHERE id = $1`,
		} {
			if _, err := tx.Exec(ctx, q, id); err != nil {
				return err
			}
		}
		return nil
	})
}
