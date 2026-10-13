package engine

import (
	"context"
	"encoding/json"
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
	switch c.Status {
	case "draft", "approval", "approved", "scheduled":
	default:
		if !o.Reset {
			return DemoSendResult{}, fmt.Errorf("%q is %s; pass --reset to clear that send and start again", c.Name, c.Status)
		}
		if err := e.resetDemo(ctx, c.ID); err != nil {
			return DemoSendResult{}, err
		}
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
	if _, err := e.Q.SendingApproval(ctx, &c.ID); errors.Is(err, pgx.ErrNoRows) {
		if err := e.recordDemoApproval(ctx, ws.ID, c.ID, sofia, marcus, approvedAt); err != nil {
			return DemoSendResult{}, err
		}
	}
	start := now.Add(-o.Elapsed)
	// The command builds the audience and fast-forwards the send itself, on
	// the virtual clock: it holds the send's lock throughout, so the
	// server's worker (woken by the start) waits until it's done.
	// A worker still running an earlier send of it (just reset) lets go
	// within seconds.
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
	prev := e.Now
	e.Now = func() time.Time { return start }
	err = e.StartWith(ctx, c.ID, StartOptions{By: sofia, Now: true, Scenario: o.Scenario, ABWindow: o.ABWindow, ApprovedBy: &marcus, ApprovedAt: &approvedAt, ScheduledBy: &linh})
	e.Now = prev
	if err == nil {
		err = e.prepare(ctx, c.ID, start)
	}
	if err == nil && o.Elapsed > 0 {
		// The fast-forward replays seconds that already passed: free the
		// demo providers' rate windows over them first, or a real-time send
		// in those seconds would leave it no room.
		ids := make([]uuid.UUID, 0, len(provs))
		for _, p := range provs {
			ids = append(ids, p.ID)
		}
		err = e.Q.SendingClearWindows(ctx, store.SendingClearWindowsParams{ProviderIds: ids, FromSec: start.Unix() - 1, ToSec: now.Unix() + 1})
		if err == nil {
			_, _, err = e.drive(ctx, c.ID, start, o.Elapsed)
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

// recordDemoApproval records the approval the boards show (Marcus Chen
// approving Sofia Rossi's request) when the approvals domain has none.
func (e *Engine) recordDemoApproval(ctx context.Context, ws, campaign, sofia, marcus uuid.UUID, at time.Time) error {
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		id := newID()
		snap, _ := json.Marshal(map[string]any{"source": "demo send"})
		if _, err := tx.Exec(ctx, `INSERT INTO approval_requests (id, workspace_id, subject_type, campaign_id, requested_by, requested_at, status, matched_rules, approvals_required, snapshot, decided_at)
			VALUES ($1, $2, 'campaign', $3, $4, $5, 'approved', ARRAY['audience_size','warming_provider'], 1, $6, $7)`, id, ws, campaign, sofia, at.Add(-40*time.Minute), snap, at); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO approval_decisions (id, request_id, approver_id, decision, comment, created_at) VALUES ($1, $2, $3, 'approve', '', $4)`, newID(), id, marcus, at)
		return err
	})
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
			`UPDATE campaigns SET status = 'scheduled', started_at = NULL, started_by = NULL, completed_at = NULL, updated_at = now() WHERE id = $1`,
		} {
			if _, err := tx.Exec(ctx, q, id); err != nil {
				return err
			}
		}
		return nil
	})
}
