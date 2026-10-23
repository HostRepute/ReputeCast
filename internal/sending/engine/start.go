package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// Errors returned by the engine's actions.
var (
	// ErrNotFound: no such campaign.
	ErrNotFound = errors.New("campaign not found")
	// ErrState: the campaign isn't in a state that allows the action.
	ErrState = errors.New("the campaign can't do that in its current state")
)

// StartOptions tune a start beyond the Starter contract (the demo command).
type StartOptions struct {
	// By is who starts the send.
	By uuid.UUID
	// Now starts right away even when the campaign is scheduled later.
	Now bool
	// Scenario is the demo transport's scenario (demo providers only).
	Scenario string
	// ABWindow replaces the A/B test's duration (demo sends).
	ABWindow time.Duration
	// ApprovedBy/ApprovedAt record the approval the send goes out under
	// when the approvals domain has none on file.
	ApprovedBy *uuid.UUID
	ApprovedAt *time.Time
	// ScheduledBy is who scheduled it (defaults to who last edited it).
	ScheduledBy *uuid.UUID
	// Scheduled is the scheduled start at its time (the start job): the
	// system starts it, on behalf of who scheduled it.
	Scheduled bool
}

// schedule is campaigns.schedule.
type schedule struct {
	Mode     string     `json:"mode"`
	At       *time.Time `json:"at"`
	Timezone string     `json:"timezone"`
}

var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// systemFor is the system actor of a campaign's start and completion.
func (e *Engine) systemFor(ctx context.Context, campaignID uuid.UUID) *audit.Actor {
	if e.SystemActor != nil {
		if a := e.SystemActor(ctx, campaignID); a != nil {
			return a
		}
	}
	return systemActor
}

// Start implements Starter: it schedules the send when the campaign is set
// for a later time, and otherwise reruns Send Readiness and starts it.
// When the recheck turns Blocked nothing is sent and the error wraps
// sending.ErrNotReady; a scheduled send is also held (see hold).
func (e *Engine) Start(ctx context.Context, campaignID, startedBy uuid.UUID) error {
	return e.StartWith(ctx, campaignID, StartOptions{By: startedBy})
}

// StartWith is Start with options.
func (e *Engine) StartWith(ctx context.Context, campaignID uuid.UUID, o StartOptions) error {
	now := e.now()
	c, err := e.Q.SendingCampaignInWorkspace(ctx, store.SendingCampaignInWorkspaceParams{WorkspaceID: e.workspaceOf(ctx, campaignID), ID: campaignID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	switch c.Status {
	case "draft", "approval", "approved", "scheduled":
	case "sending", "paused":
		return stateErr("state.alreadySending", campaignArgs(c.Name, c.Status))
	default:
		return stateErr("state.alreadySent", campaignArgs(c.Name, c.Status))
	}
	// Texts (and later push and WhatsApp) have their own sending
	// (internal/texting): an email send never takes them.
	if c.Channel != "" && c.Channel != "email" {
		return stateErr("state.notEmail", campaignArgs(c.Name, c.Status))
	}
	var sc schedule
	_ = json.Unmarshal(c.Schedule, &sc)
	// "best_time" starts at the time too, then gives each recipient their
	// usual open hour within the next 24 hours (the audience build).
	if !o.Now && (sc.Mode == "at" || sc.Mode == "best_time") && sc.At != nil && sc.At.After(now.Add(5*time.Second)) {
		return e.schedule(ctx, c, *sc.At, o)
	}
	rc, checked := sending.RecheckFrom(ctx, campaignID)
	if !checked {
		if rc, err = e.readiness().Recheck(ctx, campaignID); err != nil {
			return fmt.Errorf("send readiness recheck: %w", err)
		}
	}
	if rc.Verdict == "blocked" {
		if c.Status == "scheduled" {
			if err := e.hold(ctx, c, rc, now); err != nil {
				return err
			}
		}
		return &sending.NotReadyError{Reason: blockReason(rc), Blocking: rc.Blocking}
	}
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if err != nil {
			return err
		}
		switch c.Status {
		case "draft", "approval", "approved", "scheduled":
		default:
			return stateErr("state.is", campaignArgs(c.Name, c.Status))
		}
		defaults, err := q.SendingWorkspaceDefaults(ctx, c.WorkspaceID)
		if err != nil {
			return err
		}
		tb, _ := json.Marshal(sendThrottles(defaults.SendingDefaults, rc.Throttles))
		th, _ := json.Marshal(ParseThresholds(c.Thresholds, defaults.SendThresholds))
		approvedBy, approvedAt := o.ApprovedBy, o.ApprovedAt
		if approvedBy == nil {
			if ap, err := q.SendingApproval(ctx, &c.ID); err == nil {
				approvedBy, approvedAt = ap.ApproverID, &ap.ApprovedAt
			}
		}
		scheduledBy := o.ScheduledBy
		if scheduledBy == nil && c.Status == "scheduled" {
			scheduledBy = c.UpdatedBy
			if scheduledBy == nil {
				scheduledBy = c.CreatedBy
			}
		}
		// A scheduled send starts at its time, or once a held one passes its
		// recheck (Review & send's minute job starts it with no one behind
		// it): the system starts it on behalf of who scheduled it.
		scheduled := o.Scheduled || (o.By == uuid.Nil && c.Status == "scheduled")
		startBy := o.By
		if startBy == uuid.Nil && scheduled && scheduledBy != nil {
			startBy = *scheduledBy
		}
		by, byName := e.actorRef(ctx, q, startBy)
		if err := q.SendingMarkStarted(ctx, store.SendingMarkStartedParams{ID: c.ID, At: now, StartedBy: by}); err != nil {
			return err
		}
		if err := q.SendingInsertSend(ctx, store.SendingInsertSendParams{
			CampaignID: c.ID, WorkspaceID: c.WorkspaceID, StartedBy: by, StartedByName: byName, ApprovedBy: approvedBy, ApprovedAt: approvedAt,
			ScheduledBy: scheduledBy, Throttles: tb, Thresholds: th, Scenario: o.Scenario, At: now,
		}); err != nil {
			return err
		}
		if o.ABWindow > 0 {
			ab, _ := json.Marshal(abState{Window: int64(o.ABWindow / time.Second)})
			if err := q.SendingSetAB(ctx, store.SendingSetABParams{CampaignID: c.ID, Ab: ab, At: now}); err != nil {
				return err
			}
		}
		// The recheck passed: a hold from an earlier one ends.
		resolution, resolvedBy := "resumed", by
		if scheduled {
			resolution, resolvedBy = "auto_recovered", nil
		}
		if err := resolveHolds(ctx, q, c.ID, resolution, resolvedBy, now); err != nil {
			return err
		}
		if records(ctx) {
			actor, meta := e.actorFor(ctx, by, byName), map[string]any{"verdict": rc.Verdict}
			if scheduled {
				actor = e.systemFor(ctx, c.ID)
				meta["scheduled_by"] = byName
				meta["summary"] = "Scheduled send started · scheduled by " + firstNonEmpty(byName, "a member")
				meta["source"], meta["via"] = "send worker", "schedule"
			}
			if err := e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_started", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "sending"}, Actor: actor, Metadata: meta}); err != nil {
				return err
			}
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.started", campaignData(c.ID, c.Name, "started_by", firstNonEmpty(byName, "System"),
			"scheduled", scheduled, "started_at", now.UTC())); err != nil {
			return err
		}
		return e.insertTx(ctx, tx, PrepareArgs{CampaignID: c.ID}, nil)
	})
}

// blockReason is the first thing that blocks a send.
func blockReason(rc sending.Recheck) string {
	for _, b := range rc.Blocking {
		if strings.TrimSpace(b) != "" {
			return b
		}
	}
	return "a check failed"
}

// resolveHolds ends a campaign's open holds (the send-time recheck turned
// Blocked earlier).
func resolveHolds(ctx context.Context, q *store.Queries, campaignID uuid.UUID, resolution string, by *uuid.UUID, now time.Time) error {
	open, err := q.SendingOpenIncidents(ctx, campaignID)
	if err != nil {
		return err
	}
	for _, in := range open {
		if in.Kind != "recheck_blocked" {
			continue
		}
		if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: in.ID, At: &now, Resolution: resolution, ResolvedBy: by}); err != nil {
			return err
		}
	}
	return nil
}

// workspaceOf finds a campaign's workspace (Start is called with ids only).
func (e *Engine) workspaceOf(ctx context.Context, id uuid.UUID) uuid.UUID {
	var ws uuid.UUID
	_ = e.Pool.QueryRow(ctx, `SELECT workspace_id FROM campaigns WHERE id = $1`, id).Scan(&ws)
	return ws
}

// schedule records a later start and enqueues the job that starts it.
// Review & send schedules through Start and records it (campaign.scheduled
// in the audit log and as a webhook event); the engine records it only
// when no request is behind it.
func (e *Engine) schedule(ctx context.Context, c store.Campaign, at time.Time, o StartOptions) error {
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SendingMarkScheduled(ctx, store.SendingMarkScheduledParams{ID: c.ID, At: at}); err != nil {
			return err
		}
		by, byName := e.actorRef(ctx, q, o.By)
		if records(ctx) {
			if err := e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.scheduled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "scheduled", "scheduled_at": at}, Actor: e.actorFor(ctx, by, byName)}); err != nil {
				return err
			}
		}
		args := StartArgs{CampaignID: c.ID, At: at}
		if by != nil {
			args.By = *by
		}
		return e.insertTx(ctx, tx, args, &river.InsertOpts{ScheduledAt: at})
	})
}

// hold records a scheduled send whose send-time recheck turned Blocked, in
// the form Review & send shows it (approvals.HoldDetail). The campaign stays
// scheduled and nothing is sent. The author and the approvers are told once:
// Review & send's checks 15 minutes before may have held it already. Its
// minute job rechecks a held send every 15 minutes and starts it once a
// recheck passes.
func (e *Engine) hold(ctx context.Context, c store.Campaign, rc sending.Recheck, now time.Time) error {
	reason := blockReason(rc)
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.SendingCampaignForUpdate(ctx, c.ID); err != nil {
			return err
		}
		open, err := q.SendingOpenIncidents(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, in := range open {
			if in.Kind == "recheck_blocked" {
				return nil // already held, and people were told
			}
		}
		ws, err := q.GetWorkspace(ctx, c.WorkspaceID)
		if err != nil {
			return err
		}
		blocking := rc.Blocking
		if blocking == nil {
			blocking = []string{}
		}
		d := map[string]any{"reason": reason, "blocking": blocking}
		if rc.RunID != "" {
			d["run_id"] = rc.RunID
		}
		detail, _ := json.Marshal(d)
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: newID(), CampaignID: c.ID, Kind: "recheck_blocked", Scope: "campaign",
			Metric: "readiness", Held: rc.Audience, Detail: detail, TrippedAt: now}); err != nil {
			return err
		}
		if err := e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_held", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: systemActor, Reason: reason, Metadata: map[string]any{"blocking": blocking, "source": "send worker", "via": "send-time recheck"}}); err != nil {
			return err
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.paused", campaignData(c.ID, c.Name, "paused_by", "send_readiness", "reason", reason)); err != nil {
			return err
		}
		// The author (who asked for approval, or who scheduled it) and the approvers.
		targets := []uuid.UUID{}
		if last, err := q.ApprovalLatestForCampaign(ctx, &c.ID); err == nil && last.RequestedBy != nil {
			targets = append(targets, *last.RequestedBy)
		} else if c.UpdatedBy != nil {
			targets = append(targets, *c.UpdatedBy)
		} else if c.CreatedBy != nil {
			targets = append(targets, *c.CreatedBy)
		}
		approvers, err := notify.MembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsApprove)
		if err != nil {
			return err
		}
		targets = append(targets, approvers...)
		// The time in the workspace's zone; the reason is Send Readiness's
		// own words.
		words := i18n.Args{"name": c.Name, "at": now.In(state.Location(ws.Timezone)).Format(time.RFC3339), "reason": reason}
		wsID := c.WorkspaceID
		n := notify.Notification{WorkspaceID: &wsID, Kind: notify.KindSendBlocked, Severity: "bad", TitleText: keep("notify.heldTitle", nil),
			SummaryText: keep("notify.heldSummary", words), BodyText: keep("notify.heldBody", words),
			Link: "/w/" + ws.Slug + "/campaigns/" + c.ID.String() + "/review", Data: map[string]any{"campaign_id": c.ID.String()}}
		told := map[uuid.UUID]bool{}
		for _, u := range targets {
			if told[u] {
				continue
			}
			told[u] = true
			if err := notify.ToUser(ctx, tx, u, n); err != nil {
				return err
			}
		}
		return nil
	})
}

// actorRef returns a user id usable in user foreign keys (nil for keys,
// agents and the system) and a display name.
func (e *Engine) actorRef(ctx context.Context, q *store.Queries, id uuid.UUID) (*uuid.UUID, string) {
	if p := audit.PrincipalFrom(ctx); p != nil && p.ID == id && p.Type != authz.ActorUser {
		if p.UserID != uuid.Nil {
			u := p.UserID
			return &u, p.Name
		}
		return nil, p.Name
	}
	if id == uuid.Nil {
		return nil, ""
	}
	u, err := q.GetUser(ctx, id)
	if err != nil {
		return nil, ""
	}
	return &u.ID, u.Name
}

// actorFor is the audit actor: the request's principal when there is one,
// otherwise the user, otherwise the system.
func (e *Engine) actorFor(ctx context.Context, by *uuid.UUID, name string) *audit.Actor {
	if audit.PrincipalFrom(ctx) != nil {
		return nil
	}
	if by != nil {
		return &audit.Actor{Type: authz.ActorUser, ID: by, Name: name}
	}
	return systemActor
}

// Pause implements Starter: everything stops after the batches in flight.
func (e *Engine) Pause(ctx context.Context, campaignID, by uuid.UUID, reason string) error {
	now := e.now()
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		switch c.Status {
		case "paused":
			return nil
		case "sending":
		default:
			return stateErr("state.notSending", campaignArgs(c.Name, c.Status))
		}
		if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "paused", At: now}); err != nil {
			return err
		}
		ref, name := e.actorRef(ctx, q, by)
		if err := q.SendingSetPaused(ctx, store.SendingSetPausedParams{CampaignID: c.ID, PausedAt: &now, PausedByName: firstNonEmpty(name, "System"), PauseReason: clip(strings.TrimSpace(reason), 500)}); err != nil {
			return err
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.paused", campaignData(c.ID, c.Name, "paused_by", firstNonEmpty(name, "System"), "reason", strings.TrimSpace(reason), "scope", "campaign")); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.paused", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Reason: strings.TrimSpace(reason), Before: map[string]any{"status": "sending"}, After: map[string]any{"status": "paused"}, Actor: e.actorFor(ctx, ref, name)})
	})
}

// Resume implements Starter: a paused campaign sends again, and every
// mailbox provider or route on hold is released.
func (e *Engine) Resume(ctx context.Context, campaignID, by uuid.UUID) error {
	now := e.now()
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if c.Status != "paused" && c.Status != "sending" {
			return stateErr("state.notPaused", campaignArgs(c.Name, c.Status))
		}
		ref, name := e.actorRef(ctx, q, by)
		if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "sending", At: now}); err != nil {
			return err
		}
		released, err := e.releaseAll(ctx, q, c.ID, ref, now)
		if err != nil {
			return err
		}
		if err := q.SendingSetPaused(ctx, store.SendingSetPausedParams{CampaignID: c.ID}); err != nil {
			return err
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.resumed", campaignData(c.ID, c.Name, "resumed_by", firstNonEmpty(name, "System"), "scope", "campaign")); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.resumed", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "sending"}, Metadata: map[string]any{"released": released}, Actor: e.actorFor(ctx, ref, name)})
	})
	if err == nil {
		e.kick(ctx, campaignID)
	}
	return err
}

// releaseAll lifts every hold of a campaign and resolves the incidents.
func (e *Engine) releaseAll(ctx context.Context, q *store.Queries, id uuid.UUID, by *uuid.UUID, now time.Time) ([]string, error) {
	released := []string{}
	snd, err := q.SendingGetSendForUpdate(ctx, id)
	if err == nil {
		holds := parseHolds(snd.Holds)
		for k := range holds {
			released = append(released, k)
		}
		if len(holds) > 0 {
			if err := q.SendingSetHolds(ctx, store.SendingSetHoldsParams{CampaignID: id, Holds: []byte(`{}`), At: now}); err != nil {
				return nil, err
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	routes, err := q.SendingRoutes(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, r := range routes {
		if r.Status == "held" || r.Status == "paused" {
			if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: id, ProviderID: r.ProviderID, Status: "sending", StatusDetail: `{"decision":"resume"}`}); err != nil {
				return nil, err
			}
			e.resetCalls(routeKey{id, r.ProviderID})
			released = append(released, r.ProviderID.String())
		}
	}
	open, err := q.SendingOpenIncidents(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, in := range open {
		if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: in.ID, At: &now, Resolution: "resumed", ResolvedBy: by}); err != nil {
			return nil, err
		}
	}
	return released, nil
}

// Cancel implements Starter: nothing more is sent. A scheduled campaign is
// taken back, as Review & send's Unschedule does: to approved when its
// approval still stands (it can be scheduled again as approved), otherwise
// to draft, and a hold ends. A running one is cancelled and what is left is
// marked not sent.
func (e *Engine) Cancel(ctx context.Context, campaignID, by uuid.UUID) error {
	now := e.now()
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		ref, name := e.actorRef(ctx, q, by)
		switch c.Status {
		case "scheduled":
			status := "draft"
			if r, err := q.ApprovalOpenForCampaign(ctx, &c.ID); err == nil && r.Status == "approved" {
				status = "approved"
			} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if err := q.SendingUnschedule(ctx, store.SendingUnscheduleParams{ID: c.ID, Status: status, UpdatedBy: ref, At: now}); err != nil {
				return err
			}
			if err := resolveHolds(ctx, q, c.ID, "cancelled", ref, now); err != nil {
				return err
			}
			if !records(ctx) {
				return nil
			}
			return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.unscheduled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"status": "scheduled", "scheduled_at": c.ScheduledAt}, After: map[string]any{"status": status}, Actor: e.actorFor(ctx, ref, name)})
		case "sending", "paused":
		default:
			return stateErr("state.notSending", campaignArgs(c.Name, c.Status))
		}
		who := name
		if who == "" {
			who = "a member"
		}
		cid := c.ID
		n, err := q.SendingCancelRemaining(ctx, store.SendingCancelRemainingParams{CampaignID: &cid, Reason: "Cancelled by " + who})
		if err != nil {
			return err
		}
		if err := q.SendingMarkCompleted(ctx, store.SendingMarkCompletedParams{ID: c.ID, Status: "cancelled", At: now}); err != nil {
			return err
		}
		if err := q.SendingFinishRoutes(ctx, store.SendingFinishRoutesParams{CampaignID: c.ID, Status: "cancelled"}); err != nil {
			return err
		}
		if err := q.SendingSetDeliveryP50(ctx, c.ID); err != nil {
			return err
		}
		if err := q.SendingSetPhase(ctx, store.SendingSetPhaseParams{CampaignID: c.ID, Phase: "cancelled", FinishedAt: &now, At: now}); err != nil {
			return err
		}
		open, err := q.SendingOpenIncidents(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, in := range open {
			if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: in.ID, At: &now, Resolution: "cancelled", ResolvedBy: ref}); err != nil {
				return err
			}
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.completed", campaignData(c.ID, c.Name, "status", "cancelled", "cancelled_by", who, "not_sent", n)); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.cancelled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "cancelled", "not_sent": n}, Actor: e.actorFor(ctx, ref, name),
			Metadata: map[string]any{"summary": countText(n) + " not sent"}})
	})
}

// kick makes sure a campaign's send loop runs.
func (e *Engine) kick(ctx context.Context, id uuid.UUID) {
	e.insert(ctx, RunArgs{CampaignID: id}, nil)
}
