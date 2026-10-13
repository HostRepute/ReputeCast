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
	"github.com/hostrepute/reputecast/internal/notify"
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

// StateError explains why an action isn't possible now.
type StateError struct{ Msg string }

func (e *StateError) Error() string { return e.Msg }
func (e *StateError) Unwrap() error { return ErrState }

func stateErr(format string, a ...any) error { return &StateError{Msg: fmt.Sprintf(format, a...)} }

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
}

// schedule is campaigns.schedule.
type schedule struct {
	Mode     string     `json:"mode"`
	At       *time.Time `json:"at"`
	Timezone string     `json:"timezone"`
}

var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// Start implements Starter: it schedules the send when the campaign is set
// for a later time, and otherwise reruns Send Readiness and starts it.
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
		return stateErr("%q is already sending.", c.Name)
	default:
		return stateErr("%q was already sent or cancelled; duplicate it to send it again.", c.Name)
	}
	var sc schedule
	_ = json.Unmarshal(c.Schedule, &sc)
	if !o.Now && sc.Mode == "at" && sc.At != nil && sc.At.After(now.Add(5*time.Second)) {
		return e.schedule(ctx, c, *sc.At, o)
	}
	rc, err := e.readiness().Recheck(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("send readiness recheck: %w", err)
	}
	if rc.Verdict == "blocked" {
		if err := e.blocked(ctx, c, rc, now); err != nil {
			return err
		}
		return sending.ErrNotReady
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
			return stateErr("%q is %s.", c.Name, c.Status)
		}
		throttles := DefaultThrottles
		if rc.Throttles != nil {
			throttles = rc.Throttles
		}
		tb, _ := json.Marshal(throttles)
		th, _ := json.Marshal(ParseThresholds(c.Thresholds))
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
		by, byName := e.actorRef(ctx, q, o.By)
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
		if err := e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_start", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "sending", "verdict": rc.Verdict}, Actor: e.actorFor(ctx, by, byName)}); err != nil {
			return err
		}
		return e.insertTx(ctx, tx, PrepareArgs{CampaignID: c.ID}, nil)
	})
}

// workspaceOf finds a campaign's workspace (Start is called with ids only).
func (e *Engine) workspaceOf(ctx context.Context, id uuid.UUID) uuid.UUID {
	var ws uuid.UUID
	_ = e.Pool.QueryRow(ctx, `SELECT workspace_id FROM campaigns WHERE id = $1`, id).Scan(&ws)
	return ws
}

// schedule records a later start and enqueues the job that starts it.
func (e *Engine) schedule(ctx context.Context, c store.Campaign, at time.Time, o StartOptions) error {
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SendingMarkScheduled(ctx, store.SendingMarkScheduledParams{ID: c.ID, At: at}); err != nil {
			return err
		}
		by, byName := e.actorRef(ctx, q, o.By)
		if err := e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_schedule", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "scheduled", "at": at}, Actor: e.actorFor(ctx, by, byName)}); err != nil {
			return err
		}
		args := StartArgs{CampaignID: c.ID, At: at}
		if by != nil {
			args.By = *by
		}
		return e.insertTx(ctx, tx, args, &river.InsertOpts{ScheduledAt: at})
	})
}

// blocked records a send-time recheck that turned Blocked: the send doesn't
// start, a scheduled campaign goes back to draft, and the author and the
// approvers are told.
func (e *Engine) blocked(ctx context.Context, c store.Campaign, rc sending.Recheck, now time.Time) error {
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if c.Status == "scheduled" {
			if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "draft", At: now}); err != nil {
				return err
			}
		}
		detail, _ := json.Marshal(map[string]any{"blocking": rc.Blocking})
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: newID(), CampaignID: c.ID, Kind: "recheck_blocked", Scope: "campaign",
			Metric: "readiness", Detail: detail, TrippedAt: now}); err != nil {
			return err
		}
		body := "Send Readiness turned Blocked when the send was due to start, so nothing was sent."
		if len(rc.Blocking) > 0 {
			body += " " + strings.Join(rc.Blocking, "; ") + "."
		}
		ws, err := q.GetWorkspace(ctx, c.WorkspaceID)
		if err != nil {
			return err
		}
		n := notify.Notification{Kind: notify.KindSendBlocked, Severity: "bad", Title: fmt.Sprintf("%q didn't start: Send Readiness is blocked", c.Name), Body: body,
			Link: "/w/" + ws.Slug + "/campaigns/" + c.ID.String() + "/review", Data: map[string]any{"campaignId": c.ID.String()}}
		notified := map[uuid.UUID]bool{}
		for _, u := range []*uuid.UUID{c.CreatedBy, c.UpdatedBy} {
			if u != nil && !notified[*u] {
				notified[*u] = true
				wsID := c.WorkspaceID
				n.WorkspaceID = &wsID
				if err := notify.ToUser(ctx, tx, *u, n); err != nil {
					return err
				}
			}
		}
		approvers, err := notify.MembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsApprove)
		if err != nil {
			return err
		}
		for _, u := range approvers {
			if !notified[u] {
				notified[u] = true
				wsID := c.WorkspaceID
				n.WorkspaceID = &wsID
				if err := notify.ToUser(ctx, tx, u, n); err != nil {
					return err
				}
			}
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_blocked", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Result: audit.Denied, Reason: body, Actor: systemActor})
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
			return stateErr("%q isn't sending (it is %s).", c.Name, c.Status)
		}
		if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "paused", At: now}); err != nil {
			return err
		}
		ref, name := e.actorRef(ctx, q, by)
		if err := q.SendingSetPaused(ctx, store.SendingSetPausedParams{CampaignID: c.ID, PausedAt: &now, PausedByName: firstNonEmpty(name, "System"), PauseReason: clip(strings.TrimSpace(reason), 500)}); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_pause", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
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
			return stateErr("%q isn't paused (it is %s).", c.Name, c.Status)
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
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_resume", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
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

// Cancel implements Starter: nothing more is sent. A scheduled campaign goes
// back to draft; a running one is cancelled and what is left is marked not
// sent.
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
			if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "draft", At: now}); err != nil {
				return err
			}
			return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_unschedule", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"status": "scheduled"}, After: map[string]any{"status": "draft"}, Actor: e.actorFor(ctx, ref, name)})
		case "sending", "paused":
		default:
			return stateErr("%q isn't sending (it is %s).", c.Name, c.Status)
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
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_cancel", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "cancelled", "not_sent": n}, Actor: e.actorFor(ctx, ref, name)})
	})
}

// kick makes sure a campaign's send loop runs.
func (e *Engine) kick(ctx context.Context, id uuid.UUID) {
	e.insert(ctx, RunArgs{CampaignID: id}, nil)
}
