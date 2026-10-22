package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReviewSendInput confirms a send from Review & send.
type ReviewSendInput struct {
	// RunID is the Send Readiness run the person reviewed; the send is
	// refused when the checks ran again or the campaign changed since.
	RunID string `json:"runId" format:"uuid"`
	// Schedule replaces the saved schedule (omit to use it).
	Schedule *readiness.ReadinessSchedule `json:"schedule,omitempty"`
	// Confirm is the recipient count, typed for large sends.
	Confirm string `json:"confirm,omitempty" maxLength:"20"`
	// Acknowledged lists the keys of the warnings the person confirmed.
	Acknowledged []string `json:"acknowledged,omitempty" maxItems:"60"`
}

// ApprovalBypassInput is an Owner's emergency bypass of the approval policy.
type ApprovalBypassInput struct {
	ReviewSendInput
	Reason   string `json:"reason" minLength:"1" maxLength:"1000"`
	Password string `json:"password,omitempty" maxLength:"500"`
}

// ReviewSendResult says what happened.
type ReviewSendResult struct {
	Status      string     `json:"status" enum:"scheduled,sending"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	Recipients  int64      `json:"recipients"`
}

type sendMode struct {
	approval *store.ApprovalRequest
	bypass   string
}

// Send schedules or starts a campaign for a member holding campaigns:send.
func (s *Service) Send(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, in ReviewSendInput) (ReviewSendResult, error) {
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return ReviewSendResult{}, err
	}
	return s.send(ctx, ws, p, c, in, sendMode{})
}

// ScheduleApproved schedules an approved campaign, as approved (the
// snapshot's schedule). Its author can do it without campaigns:send.
func (s *Service) ScheduleApproved(ctx context.Context, ws *store.Workspace, p *authz.Principal, requestID uuid.UUID, in ReviewSendInput) (ReviewSendResult, error) {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: requestID})
	if err != nil {
		return ReviewSendResult{}, mapErr(ctx, err)
	}
	if r.CampaignID == nil {
		return ReviewSendResult{}, api.Conflict(say(ctx, mCampaignOnly, nil))
	}
	if (r.RequestedBy == nil || *r.RequestedBy != p.UserID) && !p.Can(authz.CampaignsSend) {
		return ReviewSendResult{}, api.Forbidden(authz.CampaignsSend, nil)
	}
	if r.Status == "pending" || r.Status == "approved" {
		if changes, err := s.revalidate(ctx, ws, *r.CampaignID); err != nil {
			return ReviewSendResult{}, err
		} else if len(changes) > 0 {
			return ReviewSendResult{}, s.noLongerCovers(ctx, changes)
		}
	}
	open, err := s.open(ctx, *r.CampaignID)
	if err != nil {
		return ReviewSendResult{}, err
	}
	if open == nil || open.ID != r.ID || open.Status != "approved" {
		return ReviewSendResult{}, api.Conflict(say(ctx, mNotApproved, nil))
	}
	c, err := s.Readiness.Campaign(ctx, ws, *r.CampaignID)
	if err != nil {
		return ReviewSendResult{}, err
	}
	snap := ParseSnapshot(open.Snapshot)
	in.Schedule = nil
	view := readiness.ScheduleView(snap.Schedule, ws.Timezone)
	in.Schedule = &view
	return s.send(ctx, ws, p, c, in, sendMode{approval: open})
}

// Bypass sends without approval: Owners only, with a reason, when the
// policy allows it. It's logged and every approver is told.
func (s *Service) Bypass(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, in ApprovalBypassInput) (ReviewSendResult, error) {
	if p.RoleKey != authz.RoleOwner {
		return ReviewSendResult{}, api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, mBypassOwnerOnly, nil))
	}
	if !ParsePolicy(ws.ApprovalPolicy).EmergencyBypass {
		return ReviewSendResult{}, api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, mBypassOff, nil))
	}
	reason := strings.TrimSpace(in.Reason)
	if len([]rune(reason)) < 10 {
		return ReviewSendResult{}, api.Invalid("body.reason", say(ctx, mBypassReason, nil))
	}
	u, err := s.Q.GetUser(ctx, p.UserID)
	if err != nil {
		return ReviewSendResult{}, err
	}
	if u.PasswordHash != nil {
		ok, err := authn.VerifyPassword(in.Password, *u.PasswordHash)
		if err != nil || !ok {
			return ReviewSendResult{}, api.Invalid("body.password", say(ctx, mWrongPassword, nil))
		}
	}
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return ReviewSendResult{}, err
	}
	return s.send(ctx, ws, p, c, in.ReviewSendInput, sendMode{bypass: reason})
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Service) send(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow, in ReviewSendInput, mode sendMode) (ReviewSendResult, error) {
	switch c.Status {
	case "draft", "approval", "approved":
	case "scheduled":
		return ReviewSendResult{}, api.Conflict(say(ctx, mAlreadyScheduled, nil))
	default:
		return ReviewSendResult{}, api.Conflict(say(ctx, mAlreadySending, nil))
	}
	// The campaign as the person reviewed it, before a new schedule.
	reviewed := c
	sched := readiness.ParseSchedule(c.Schedule)
	if in.Schedule != nil {
		next, _, err := in.Schedule.Resolve("body.schedule")
		if err != nil {
			return ReviewSendResult{}, err
		}
		sched = next
	}
	now := s.now()
	var at *time.Time
	if sched.Mode != "now" {
		t := sched.StartAt(now)
		if t.Before(now.Add(time.Minute)) {
			return ReviewSendResult{}, api.Invalid("body.schedule.date", say(ctx, mPickFuture, nil))
		}
		at = &t
	}
	// Compared as values: Postgres hands jsonb back reformatted.
	if sched != readiness.ParseSchedule(c.Schedule) && mode.approval == nil {
		sb, _ := json.Marshal(sched)
		if err := s.Q.ApprovalScheduleCampaign(ctx, store.ApprovalScheduleCampaignParams{Schedule: sb, ScheduledAt: nil, UpdatedBy: &p.UserID, Now: now, WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return ReviewSendResult{}, err
		}
		s.CampaignChanged(ctx, ws, c.ID)
		var err error
		if c, err = s.Readiness.Campaign(ctx, ws, c.ID); err != nil {
			return ReviewSendResult{}, err
		}
	}
	// The run the person reviewed must still describe the campaign.
	run, err := s.Readiness.Latest(ctx, c.ID)
	if err != nil {
		return ReviewSendResult{}, err
	}
	if run == nil || (in.RunID != "" && run.ID != in.RunID) {
		return ReviewSendResult{}, api.Err(http.StatusConflict, "readiness_changed", say(ctx, mReadinessChanged, nil))
	}
	if run.Fingerprint != s.Readiness.CurrentFingerprint(ctx, ws, reviewed) {
		return ReviewSendResult{}, api.Err(http.StatusConflict, "readiness_stale", say(ctx, mReadinessStale, nil))
	}
	if run.Verdict == readiness.VerdictBlocked {
		return ReviewSendResult{}, api.Err(http.StatusConflict, "readiness_blocked", say(ctx, mSendBlocked, i18n.Args{"count": run.Counts.Blocks}))
	}
	if run.Incomplete && run.UnknownAcceptedAt == nil {
		return ReviewSendResult{}, api.Err(http.StatusConflict, "readiness_incomplete", say(ctx, mReadinessIncomplete, nil))
	}
	missing := []string{}
	for _, ch := range run.Checks {
		if ch.Status == readiness.Warn && !slices.Contains(in.Acknowledged, ch.Key) {
			missing = append(missing, ch.Title)
		}
	}
	if len(missing) > 0 {
		return ReviewSendResult{}, api.Invalid("body.acknowledged", say(ctx, mConfirmWarnings, i18n.Args{"warnings": strings.Join(missing, "; ")}))
	}
	recipients := run.Estimates.Audience.Final
	// Large sends are confirmed by typing the count; a bypass is confirmed
	// by the written reason and the Owner's password instead.
	if recipients >= LargeSend && mode.bypass == "" && digits(in.Confirm) != fmt.Sprint(recipients) {
		return ReviewSendResult{}, api.Invalid("body.confirm", say(ctx, mTypeCount, i18n.Args{"count": fmt.Sprint(recipients)}))
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return ReviewSendResult{}, err
	}
	var approval *store.ApprovalRequest
	switch {
	case mode.bypass != "":
	case mode.approval != nil:
		approval = mode.approval
	default:
		author := memberByID(ms, &p.UserID)
		if author == nil {
			return ReviewSendResult{}, api.Forbidden(authz.CampaignsSend, nil)
		}
		facts, err := s.ruleFacts(ctx, ws, c, run, *author)
		if err != nil {
			return ReviewSendResult{}, err
		}
		if rules := ParsePolicy(ws.ApprovalPolicy).Evaluate(facts); len(rules) > 0 {
			if changes, err := s.revalidate(ctx, ws, c.ID); err != nil {
				return ReviewSendResult{}, err
			} else if len(changes) > 0 {
				return ReviewSendResult{}, s.noLongerCovers(ctx, changes)
			}
			open, err := s.open(ctx, c.ID)
			if err != nil {
				return ReviewSendResult{}, err
			}
			if open == nil || open.Status != "approved" {
				return ReviewSendResult{}, approvalRequired(ctx, rules)
			}
			approval = open
		}
	}
	// Sending now: Send Readiness runs again as the send starts.
	if sched.Mode == "now" {
		fresh, err := s.Readiness.Run(ctx, ws, c.ID, readiness.RunOptions{Trigger: readiness.TriggerSendTime, By: &p.UserID})
		if err != nil && !errors.Is(err, readiness.ErrRunning) {
			return ReviewSendResult{}, err
		}
		if err == nil && fresh.Verdict == readiness.VerdictBlocked {
			return ReviewSendResult{}, api.Err(http.StatusConflict, "readiness_blocked", say(ctx, mBlockedAtStart, i18n.Args{"reason": firstBlockIn(langOf(ctx), fresh)}))
		}
		if err == nil {
			// The engine starts on this run instead of running the checks again.
			ctx = sending.WithRecheck(ctx, c.ID, sending.Recheck{Verdict: fresh.Verdict, Throttles: throttlesOf(fresh),
				Audience: fresh.Estimates.Audience.Final, RunID: fresh.ID})
		}
	}
	if err := s.Q.ApprovalScheduleCampaign(ctx, store.ApprovalScheduleCampaignParams{Schedule: c.Schedule, ScheduledAt: at, UpdatedBy: &p.UserID, Now: now, WorkspaceID: ws.ID, ID: c.ID}); err != nil {
		return ReviewSendResult{}, err
	}
	if err := s.starter().Start(ctx, c.ID, p.UserID); err != nil {
		// Nothing was scheduled: the campaign keeps its time as it was.
		_ = s.Q.ApprovalScheduleCampaign(ctx, store.ApprovalScheduleCampaignParams{Schedule: c.Schedule, ScheduledAt: c.ScheduledAt, UpdatedBy: c.UpdatedBy, Now: c.UpdatedAt,
			WorkspaceID: ws.ID, ID: c.ID})
		switch {
		case errors.Is(err, sending.ErrUnavailable):
			return ReviewSendResult{}, api.Err(http.StatusServiceUnavailable, "sending_unavailable", say(ctx, mSendingUnavailable, nil))
		case errors.Is(err, sending.ErrNotReady):
			msg := say(ctx, mBlockedAsStarted, nil)
			var nr *sending.NotReadyError
			if errors.As(err, &nr) && nr.Reason != "" {
				msg = say(ctx, mBlockedAsStartedReason, i18n.Args{"reason": strings.TrimSuffix(gapIn(langOf(ctx), nr.Reason), ".")})
			}
			return ReviewSendResult{}, api.Err(http.StatusConflict, "readiness_blocked", msg)
		}
		return ReviewSendResult{}, err
	}
	out := ReviewSendResult{Status: "sending", Recipients: recipients}
	if at != nil {
		out.Status, out.ScheduledAt = "scheduled", at
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		meta := map[string]any{"recipients": recipients, "run_id": run.ID, "schedule": ScheduleText(c.Schedule, state.Location(ws.Timezone))}
		if approval != nil {
			meta["approval_id"] = approval.ID.String()
		}
		if at != nil {
			if err := q.ApprovalMarkScheduled(ctx, store.ApprovalMarkScheduledParams{ScheduledAt: at, Now: now, ID: c.ID}); err != nil {
				return err
			}
			ref := ""
			if approval != nil {
				ref = approval.ID.String()
			}
			if err := emitScheduled(ctx, tx, ws, c.ID, c.Name, *at, recipients, ref); err != nil {
				return err
			}
		} else if err := q.ApprovalMarkSending(ctx, store.ApprovalMarkSendingParams{StartedBy: &p.UserID, Now: now, ID: c.ID}); err != nil {
			return err
		}
		if err := q.ApprovalResolveHold(ctx, store.ApprovalResolveHoldParams{ResolvedAt: &now, Resolution: "resumed", ResolvedBy: &p.UserID, CampaignID: c.ID}); err != nil {
			return err
		}
		// The confirmed run describes the campaign at its new time.
		if fp := s.Readiness.CurrentFingerprint(ctx, ws, c); fp != run.Fingerprint {
			if rid, err := uuid.Parse(run.ID); err == nil {
				if err := q.ReadinessSetFingerprint(ctx, store.ReadinessSetFingerprintParams{Fingerprint: fp, ID: rid}); err != nil {
					return err
				}
			}
		}
		if mode.bypass != "" {
			if err := s.recordBypass(ctx, tx, ws, p, c, run, mode.bypass, ms); err != nil {
				return err
			}
			meta["bypass"] = true
		}
		action := "campaign.scheduled"
		if at == nil {
			action = "campaign.send_started"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": out.Status, "scheduled_at": at}, Metadata: meta})
	})
	return out, err
}

func firstBlock(run readiness.ReadinessRun) string { return firstBlockIn(i18n.Default, run) }

// firstBlockIn is the first blocking check's title, as Send Readiness wrote
// it, or "a check failed" in the reader's language.
func firstBlockIn(l i18n.Lang, run readiness.ReadinessRun) string {
	for _, ch := range run.Checks {
		if ch.Status == readiness.Block {
			return ch.FlatTitle
		}
	}
	return mGapCheckFailed.Text(l, nil)
}

// recordBypass stores the bypass with the snapshot, logs it and tells every approver.
func (s *Service) recordBypass(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, c store.ReadinessCampaignRow, run *readiness.ReadinessRun, reason string, ms []member) error {
	q := store.New(tx)
	author := memberByID(ms, &p.UserID)
	if author == nil {
		return api.Forbidden(authz.CampaignsSend, nil)
	}
	facts, err := s.ruleFacts(ctx, ws, c, run, *author)
	if err != nil {
		return err
	}
	rules := ParsePolicy(ws.ApprovalPolicy).Evaluate(facts)
	snap, err := s.buildSnapshot(ctx, ws, c, run, rules, *author)
	if err != nil {
		return err
	}
	sb, _ := json.Marshal(snap)
	id := uuid.Must(uuid.NewV7())
	version := c.Version
	runID, _ := uuid.Parse(run.ID)
	if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "campaign", CampaignID: &c.ID, RequestedBy: &p.UserID,
		RequestedAt: s.now(), Status: "bypassed", MatchedRules: ruleKeys(rules), ApprovalsRequired: 0, Snapshot: sb, CampaignVersion: &version, ReadinessRunID: &runID,
		BypassReason: reason}); err != nil {
		return err
	}
	others := []string{}
	for _, a := range approversOf(ms) {
		if a.ID != p.UserID {
			others = append(others, a.Name)
		}
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.approval_bypassed", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
		Reason: reason, After: map[string]any{"recipients": snap.AudienceCount, "rules": ruleKeys(rules), "version": c.Version, "subject": c.Subject, "route": snap.SplitLabel,
			"schedule": ScheduleText(c.Schedule, state.Location(ws.Timezone))},
		Metadata: map[string]any{"request_id": id.String(), "notified": others, "summary": fmt.Sprintf("%s recipients · reason: %s", readiness.Count(snap.AudienceCount), reason)}}); err != nil {
		return err
	}
	n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalDecided, Severity: "warn", Title: "Sent without approval",
		Summary: fmt.Sprintf("%s used the emergency bypass for “%s” · %s recipients", author.Name, c.Name, readiness.Count(snap.AudienceCount)),
		Body:    fmt.Sprintf("%s used the emergency bypass for “%s” · %s recipients. Reason: %s", author.Name, c.Name, readiness.Count(snap.AudienceCount), reason),
		Link:    "/w/" + ws.Slug + "/campaigns/" + c.ID.String() + "/review", Data: map[string]any{"request_id": id.String(), "campaign_id": c.ID.String()}}
	for _, a := range approversOf(ms) {
		if a.ID == p.UserID {
			continue
		}
		if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
			return err
		}
	}
	return nil
}

// Unschedule takes a scheduled campaign back. An approved campaign keeps its
// approval (it can be scheduled again as approved).
func (s *Service) Unschedule(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) error {
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return err
	}
	if c.Status != "scheduled" {
		return api.Conflict(say(ctx, mNotScheduled, nil))
	}
	if err := s.starter().Cancel(ctx, c.ID, p.UserID); err != nil && !errors.Is(err, sending.ErrUnavailable) {
		return err
	}
	status := "draft"
	if open, err := s.open(ctx, c.ID); err == nil && open != nil && open.Status == "approved" {
		status = "approved"
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalUnschedule(ctx, store.ApprovalUnscheduleParams{Status: status, UpdatedBy: &p.UserID, Now: now, WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return err
		}
		if err := q.ApprovalResolveHold(ctx, store.ApprovalResolveHoldParams{ResolvedAt: &now, Resolution: "cancelled", ResolvedBy: &p.UserID, CampaignID: c.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.unscheduled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Before: map[string]any{"status": "scheduled", "scheduled_at": c.ScheduledAt}, After: map[string]any{"status": status}})
	})
}

// CheckSendTime is the recheck when a send starts (add-on 3.8): Send
// Readiness runs again and the approval must still cover the campaign. It
// returns an error wrapping sending.ErrNotReady when the send must not
// start; the campaign is then held and its author and approvers are told.
// The sending engine calls it before the first batch of a scheduled send.
func (s *Service) CheckSendTime(ctx context.Context, wsID, campaignID uuid.UUID) error {
	w, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	ws := &w
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return err
	}
	run, err := s.Readiness.Run(ctx, ws, campaignID, readiness.RunOptions{Trigger: readiness.TriggerSendTime})
	if errors.Is(err, readiness.ErrRunning) {
		latest, lerr := s.Readiness.Latest(ctx, campaignID)
		if lerr != nil || latest == nil || s.now().Sub(latest.RanAt) > 10*time.Minute {
			return fmt.Errorf("%w: %s", sending.ErrNotReady, english(mGapRunning, nil))
		}
		run = *latest
	} else if err != nil {
		return err
	}
	if run.Verdict == readiness.VerdictBlocked {
		reason := firstBlock(run)
		if err := s.hold(ctx, ws, c, &run, reason); err != nil {
			return err
		}
		return fmt.Errorf("%w: %s", sending.ErrNotReady, reason)
	}
	reason, err := s.approvalGap(ctx, ws, campaignID)
	if err != nil {
		return err
	}
	if reason != "" {
		if err := s.hold(ctx, ws, c, &run, reason); err != nil {
			return err
		}
		return fmt.Errorf("%w: %s", sending.ErrNotReady, reason)
	}
	now := s.now()
	return s.Q.ApprovalResolveHold(ctx, store.ApprovalResolveHoldParams{ResolvedAt: &now, Resolution: "auto_recovered", CampaignID: campaignID})
}

// approvalGap says why the campaign's approval doesn't let it send now
// ("approval is still pending", "the approval no longer covers the
// campaign", "an approver asked for changes"…), or "" when it does, an
// Owner bypassed it or the send never needed one. An approval that stopped
// covering the campaign is invalidated on the way.
func (s *Service) approvalGap(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (string, error) {
	last, err := s.Q.ApprovalLatestForCampaign(ctx, &campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if last.Status == "pending" || last.Status == "approved" {
		changes, err := s.Revalidate(ctx, ws, campaignID)
		if err != nil {
			return "", err
		}
		if len(changes) > 0 {
			last.Status = "invalidated"
		}
	}
	switch last.Status {
	case "pending":
		return english(mGapPending, nil), nil
	case "invalidated":
		return english(mGapInvalidated, nil), nil
	}
	settled := ""
	if m, ok := gapMsgs[last.Status]; ok {
		settled = english(m, nil)
	}
	if settled == "" {
		return "", nil
	}
	// Only a request made for the scheduled send holds it: an older one was
	// settled before someone scheduled the campaign without needing it.
	if sched, err := s.Q.ApprovalLastAudit(ctx, store.ApprovalLastAuditParams{WorkspaceID: &ws.ID, Action: "campaign.scheduled", TargetID: campaignID.String()}); err == nil {
		if last.RequestedAt.After(sched.OccurredAt) {
			return settled, nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return "", nil
}

// SendRecheck is Send Readiness as a send starts, in the terms the sending
// engine asks for (its send-time recheck): the verdict, what blocks and
// the messages per second Send Readiness set per mailbox provider.
type SendRecheck struct {
	// Verdict is "ready", "warnings" or "blocked".
	Verdict string
	// Blocking are the titles of the blocking checks, and why the approval
	// doesn't cover the send when it doesn't.
	Blocking []string
	// Throttles are messages per second per mailbox provider ("outlook": 5);
	// nil keeps the engine's defaults.
	Throttles map[string]float64
	// Audience is the run's final recipient count, RunID the run.
	Audience int64
	RunID    string
}

// throttlesOf is a run's per-mailbox-provider rates; nil when it sets none.
func throttlesOf(run readiness.ReadinessRun) map[string]float64 {
	var out map[string]float64
	for _, t := range run.Estimates.Throttling {
		if t.Rate > 0 {
			if out == nil {
				out = map[string]float64{}
			}
			out[t.Key] = t.Rate
		}
	}
	return out
}

// SendRecheck runs Send Readiness for a send that is starting (the run is
// recorded with the send_time trigger) and checks that the approval, when
// the send needed one, still covers the campaign. Unlike CheckSendTime it
// holds nothing and tells nobody: the sending engine does both when the
// verdict is Blocked, and the review page reads the blocks from this run.
func (s *Service) SendRecheck(ctx context.Context, campaignID uuid.UUID) (SendRecheck, error) {
	wsID, err := s.Q.ReadinessCampaignWorkspace(ctx, campaignID)
	if err != nil {
		return SendRecheck{}, err
	}
	w, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return SendRecheck{}, err
	}
	ws := &w
	out := SendRecheck{Blocking: []string{}}
	run, err := s.Readiness.Run(ctx, ws, campaignID, readiness.RunOptions{Trigger: readiness.TriggerSendTime})
	running := errors.Is(err, readiness.ErrRunning)
	if err != nil && !running {
		return SendRecheck{}, err
	}
	if running {
		// Someone started the checks a moment ago: use that run if it's fresh.
		latest, lerr := s.Readiness.Latest(ctx, campaignID)
		if lerr != nil {
			return SendRecheck{}, lerr
		}
		if latest == nil || s.now().Sub(latest.RanAt) > 10*time.Minute {
			out.Verdict = readiness.VerdictBlocked
			out.Blocking = append(out.Blocking, english(mGapRunning, nil))
			return out, nil
		}
		run = *latest
	}
	out.Verdict, out.Throttles, out.Audience, out.RunID = run.Verdict, throttlesOf(run), run.Estimates.Audience.Final, run.ID
	for _, ch := range run.Checks {
		if ch.Status == readiness.Block {
			out.Blocking = append(out.Blocking, firstNonEmptyStr(ch.FlatTitle, ch.Title))
		}
	}
	reason, err := s.approvalGap(ctx, ws, campaignID)
	if err != nil {
		return SendRecheck{}, err
	}
	if reason != "" {
		out.Verdict = readiness.VerdictBlocked
		out.Blocking = append(out.Blocking, strings.ToUpper(reason[:1])+reason[1:])
	}
	return out, nil
}

// HoldDetail is campaign_incidents.detail of a held send. Holds recorded
// here carry the run and its blocking checks; the sending engine's carry
// only the titles of what blocked (Blocking).
type HoldDetail struct {
	RunID    string                     `json:"run_id"`
	Reason   string                     `json:"reason"`
	Blocks   []readiness.ReadinessCheck `json:"blocks"`
	Blocking []string                   `json:"blocking,omitempty"`
}

// heldView is a held send as Review & send and the approval show it. When
// the hold doesn't carry its checks (the sending engine recorded it), they
// come from the campaign's latest run, which is the send-time recheck.
func (s *Service) heldView(ctx context.Context, h store.CampaignIncident) *ReviewHeld {
	var d HoldDetail
	_ = json.Unmarshal(h.Detail, &d)
	out := &ReviewHeld{Since: h.TrippedAt, Reason: gapIn(langOf(ctx), d.Reason), Held: h.Held, Blocks: d.Blocks}
	if len(out.Blocks) == 0 || out.Held == 0 {
		if out.Blocks == nil {
			out.Blocks = []readiness.ReadinessCheck{}
		}
		if run, err := s.Readiness.Latest(ctx, h.CampaignID); err == nil && run != nil && run.Verdict == readiness.VerdictBlocked {
			if len(out.Blocks) == 0 {
				for _, ch := range run.Checks {
					if ch.Status == readiness.Block {
						out.Blocks = append(out.Blocks, ch)
					}
				}
			}
			if out.Held == 0 {
				out.Held = run.Estimates.Audience.Final
			}
		}
	}
	if out.Reason == "" {
		switch {
		case len(d.Blocking) > 0:
			out.Reason = gapIn(langOf(ctx), d.Blocking[0])
		case len(out.Blocks) > 0:
			out.Reason = firstNonEmptyStr(out.Blocks[0].FlatTitle, out.Blocks[0].Title)
		}
	}
	return out
}

// hold records that a send didn't start and tells the author and approvers.
func (s *Service) hold(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, run *readiness.ReadinessRun, reason string) error {
	if _, err := s.Q.ApprovalOpenHold(ctx, c.ID); err == nil {
		return nil // already held and told
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return err
	}
	detail := HoldDetail{Reason: reason, Blocks: []readiness.ReadinessCheck{}}
	var held int64
	if run != nil {
		detail.RunID = run.ID
		held = run.Estimates.Audience.Final
		for _, ch := range run.Checks {
			if ch.Status == readiness.Block {
				detail.Blocks = append(detail.Blocks, ch)
			}
		}
	}
	db1, _ := json.Marshal(detail)
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalInsertHold(ctx, store.ApprovalInsertHoldParams{ID: uuid.Must(uuid.NewV7()), CampaignID: c.ID, Held: held, Detail: db1, TrippedAt: now}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_held", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: reason, Metadata: map[string]any{"held": held, "run_id": detail.RunID}}); err != nil {
			return err
		}
		if err := emitHeld(ctx, tx, ws, c.ID, c.Name, reason, held); err != nil {
			return err
		}
		targets := []uuid.UUID{}
		if last, err := q.ApprovalLatestForCampaign(ctx, &c.ID); err == nil && last.RequestedBy != nil {
			targets = append(targets, *last.RequestedBy)
		} else if c.UpdatedBy != nil {
			targets = append(targets, *c.UpdatedBy)
		}
		for _, a := range approversOf(ms) {
			targets = append(targets, a.ID)
		}
		at := now.In(state.Location(ws.Timezone)).Format("15:04")
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSendBlocked, Severity: "bad", Title: "Send held",
			Summary: fmt.Sprintf("“%s” didn't start at %s: %s", c.Name, at, reason),
			Body:    fmt.Sprintf("“%s” didn't start at %s: %s. It stays held until a recheck passes.", c.Name, at, reason),
			Link:    "/w/" + ws.Slug + "/campaigns/" + c.ID.String() + "/review", Data: map[string]any{"campaign_id": c.ID.String()}}
		seen := map[uuid.UUID]bool{}
		for _, id := range targets {
			if seen[id] {
				continue
			}
			seen[id] = true
			if err := notify.ToUser(ctx, tx, id, n); err != nil {
				return err
			}
		}
		return nil
	})
}
