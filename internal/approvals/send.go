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

	"github.com/hostrepute/reputecast/internal/access"
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
	// Canary is the canary choice (review GS-10, C1): auto lets Traffic
	// rules decide, on runs one, off sends to everyone at once; without one
	// the campaign keeps the choice it has.
	Canary string `json:"canary,omitempty" enum:"auto,on,off,"`
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
		// A held send is confirmed again from Review & send, on the fresh
		// checks and count (SR-3); a send waiting for its time isn't.
		if _, err := s.Q.ApprovalOpenHold(ctx, c.ID); errors.Is(err, pgx.ErrNoRows) {
			return ReviewSendResult{}, api.Conflict(say(ctx, mAlreadyScheduled, nil))
		} else if err != nil {
			return ReviewSendResult{}, err
		}
	default:
		return ReviewSendResult{}, api.Conflict(say(ctx, mAlreadySending, nil))
	}
	// A scoped member sends only within their lists, whoever built the audience.
	if a := readiness.ParseAudience(c.Audience); !access.AudienceInScope(p.Scope, access.ParseLists(a.Lists), len(a.Segments)) {
		return ReviewSendResult{}, api.Err(http.StatusForbidden, api.CodePermissionDenied, access.AudienceScopeProblem(i18n.LangFrom(ctx)))
	}
	// The campaign as the person reviewed it, before a new schedule.
	reviewed := c
	sched := readiness.ParseSchedule(c.Schedule)
	if in.Schedule != nil {
		next, _, err := in.Schedule.Resolve(ctx, "body.schedule")
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
	// A warning dismissed in the composer, with a reason, is acknowledged (CO-14).
	if err := s.Readiness.Dismissed(ctx, ws, c.ID, run); err != nil {
		return ReviewSendResult{}, err
	}
	acknowledged := append(slices.Clone(in.Acknowledged), readiness.DismissedKeys(*run)...)
	missing := []string{}
	for _, ch := range run.Checks {
		if ch.Status == readiness.Warn && !slices.Contains(acknowledged, ch.Key) {
			missing = append(missing, ch.Title)
		}
	}
	if len(missing) > 0 {
		return ReviewSendResult{}, api.Invalid("body.acknowledged", say(ctx, mConfirmWarnings, i18n.Args{"warnings": strings.Join(missing, "; ")}))
	}
	recipients := run.Estimates.Audience.Final
	// Only an unusual send is confirmed by typing the count (SR-13); a
	// bypass is confirmed by the written reason and the Owner's password.
	if mode.bypass == "" && len(s.typedReasons(ctx, c, *run)) > 0 && digits(in.Confirm) != fmt.Sprint(recipients) {
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
	// Sending now: the run the person just confirmed is the send-time
	// check, so it must be recent. Nothing runs inside the request (SR-8):
	// an older run is run again in the background, and the person sends
	// when it finishes.
	if sched.Mode == "now" {
		if age := now.Sub(run.RanAt); age > SendNowWithin {
			if _, err := s.Readiness.Start(ctx, ws, c.ID, readiness.RunOptions{Trigger: readiness.TriggerManual, By: &p.UserID}); err != nil && !errors.Is(err, readiness.ErrRunning) {
				return ReviewSendResult{}, err
			}
			return ReviewSendResult{}, api.Err(http.StatusConflict, "readiness_old", say(ctx, mReadinessOld, i18n.Args{"minutes": int(age.Minutes())}))
		}
		// The engine starts on this run instead of running the checks again.
		ctx = sending.WithRecheck(ctx, c.ID, sending.Recheck{Verdict: run.Verdict, Throttles: throttlesOf(*run),
			Audience: run.Estimates.Audience.Final, RunID: run.ID})
	}
	if err := s.Q.ApprovalScheduleCampaign(ctx, store.ApprovalScheduleCampaignParams{Schedule: c.Schedule, ScheduledAt: at, UpdatedBy: &p.UserID, Now: now, WorkspaceID: ws.ID, ID: c.ID}); err != nil {
		return ReviewSendResult{}, err
	}
	// The canary choice goes with the start (C1): the engine keeps it on the
	// campaign, so a scheduled send starts with it at its time; without one
	// the campaign keeps its own.
	ctx = sending.WithCanary(ctx, c.ID, in.Canary)
	if err := s.starter().Start(ctx, c.ID, p.UserID); err != nil {
		// Nothing was scheduled: the campaign keeps its time as it was.
		_ = s.Q.ApprovalScheduleCampaign(ctx, store.ApprovalScheduleCampaignParams{Schedule: c.Schedule, ScheduledAt: c.ScheduledAt, UpdatedBy: c.UpdatedBy, Now: c.UpdatedAt,
			WorkspaceID: ws.ID, ID: c.ID})
		return ReviewSendResult{}, startProblem(ctx, err)
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
		// What the person confirmed: the send-time recheck holds the send
		// on anything they didn't (SR-3).
		if err := s.Readiness.Confirm(ctx, q, ws, c, *run, p.UserID, acknowledged, at, now); err != nil {
			return err
		}
		// Review & send shows the choice confirmed with the send.
		if in.Canary != "" {
			if err := s.Readiness.SetCanary(ctx, q, c.ID, in.Canary); err != nil {
				return err
			}
			meta["canary"] = in.Canary
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

// typedReasons are why the sender types the recipient count (SR-13): what
// Send Readiness found unusual about the send, and an audience that
// changed by more than 10% since it was confirmed (a held send), in the
// request's language. None for a routine send.
func (s *Service) typedReasons(ctx context.Context, c store.ReadinessCampaignRow, run readiness.ReadinessRun) []string {
	out := []string{}
	if cf := run.In(langOf(ctx)).Estimates.Confirm; cf != nil && cf.Typed {
		out = append(out, cf.Reasons...)
	}
	if c.Status != "scheduled" {
		return out
	}
	h, err := s.Q.ApprovalOpenHold(ctx, c.ID)
	if err != nil {
		return out
	}
	var d HoldDetail
	if json.Unmarshal(h.Detail, &d) != nil {
		return out
	}
	for _, x := range d.Holds {
		if x.Kind == readiness.HoldAudience {
			out = append(out, x.Text.In(langOf(ctx)))
		}
	}
	return out
}

// SendNowWithin is how recent the run a person confirms must be for a send
// that starts now: it is that send's send-time check.
const SendNowWithin = 15 * time.Minute

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
		Link:    "/w/" + ws.Slug + "/campaigns/" + c.ID.String() + "/review", Data: map[string]any{"request_id": id.String(), "campaign_id": c.ID.String()},
		TitleText:   mNotifyBypassTitle.Keep(nil),
		SummaryText: mNotifyBypassSummary.Keep(i18n.Args{"author": author.Name, "name": c.Name, "count": snap.AudienceCount}),
		BodyText:    mNotifyBypassBody.Keep(i18n.Args{"author": author.Name, "name": c.Name, "count": snap.AudienceCount, "reason": reason})}
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
		if err := s.Readiness.Forget(ctx, q, c.ID); err != nil {
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
	check, err := s.sendTimeCheck(ctx, ws, campaignID, run)
	if err != nil {
		return err
	}
	if check.Held() {
		reason := check.Holds[0].English()
		if check.Holds[0].Kind == readiness.HoldBlocked {
			reason = firstBlock(run)
		}
		if err := s.hold(ctx, ws, c, &run, reason, &check); err != nil {
			return err
		}
		return fmt.Errorf("%w: %s", sending.ErrNotReady, reason)
	}
	reason, err := s.approvalGap(ctx, ws, campaignID)
	if err != nil {
		return err
	}
	if reason != "" {
		if err := s.hold(ctx, ws, c, &run, reason, nil); err != nil {
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
	// What nobody confirmed holds the send too (SR-3): new warnings and
	// unknown checks, the audience changed by more than 10%, the content,
	// sender or route edited after scheduling.
	check, err := s.sendTimeCheck(ctx, ws, campaignID, run)
	if err != nil {
		return SendRecheck{}, err
	}
	for _, h := range check.Holds {
		if h.Kind != readiness.HoldBlocked {
			out.Verdict = readiness.VerdictBlocked
			out.Blocking = append(out.Blocking, h.English())
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
// here carry the run, its blocking checks and why the send-time recheck held
// it (Holds, with the confirmed and the fresh count); the sending engine's
// carry only the titles of what blocked (Blocking).
type HoldDetail struct {
	RunID     string                     `json:"run_id"`
	Reason    string                     `json:"reason"`
	Blocks    []readiness.ReadinessCheck `json:"blocks"`
	Blocking  []string                   `json:"blocking,omitempty"`
	Holds     []readiness.SendHold       `json:"holds,omitempty"`
	Audience  int64                      `json:"audience,omitempty"`
	Confirmed int64                      `json:"confirmed,omitempty"`
}

// sendTimeCheck compares a fresh run and the campaign as it stands with
// what the person confirmed when they scheduled it (SR-3, C3).
func (s *Service) sendTimeCheck(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, run readiness.ReadinessRun) (readiness.SendTimeCheck, error) {
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return readiness.SendTimeCheck{}, err
	}
	conf, err := s.Readiness.Confirmed(ctx, campaignID)
	if err != nil {
		return readiness.SendTimeCheck{}, err
	}
	now := s.Readiness.Digests(ctx, ws, c)
	// Approvers who approved the campaign as it is after it was scheduled
	// confirmed its content, sender and route: edits they approved don't
	// hold the send.
	if conf != nil && conf.Digests != now {
		if last, err := s.Q.ApprovalLatestForCampaign(ctx, &campaignID); err == nil && last.Status == "approved" && last.DecidedAt != nil &&
			last.DecidedAt.After(conf.ConfirmedAt) {
			if changes, err := s.revalidate(ctx, ws, campaignID); err == nil && len(changes) == 0 {
				approved := *conf
				approved.Digests = now
				conf = &approved
			}
		}
	}
	// Warnings dismissed in the composer since are acknowledged too (CO-14).
	if err := s.Readiness.Dismissed(ctx, ws, campaignID, &run); err != nil {
		return readiness.SendTimeCheck{}, err
	}
	return readiness.CheckSendTime(run, conf, now), nil
}

// heldView is a held send as Review & send and the approval show it. When
// the hold doesn't carry its checks (the sending engine recorded it), they
// come from the campaign's latest run, which is the send-time recheck.
func (s *Service) heldView(ctx context.Context, h store.CampaignIncident) *ReviewHeld {
	var d HoldDetail
	_ = json.Unmarshal(h.Detail, &d)
	out := &ReviewHeld{Since: h.TrippedAt, Reason: gapIn(langOf(ctx), d.Reason), Held: h.Held, Blocks: d.Blocks, Holds: []ReviewHoldReason{}}
	s.holdReasons(ctx, h, d, out)
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
	// The blocks read in the reader's language; a reason that is a block's
	// title (as the run stored it) reads as that block does.
	l := i18n.LangFrom(ctx)
	read := readiness.InChecks(out.Blocks, l)
	for i, b := range out.Blocks {
		switch out.Reason {
		case b.FlatTitle:
			out.Reason = read[i].FlatTitle
		case b.Title:
			out.Reason = read[i].Title
		default:
			continue
		}
		break
	}
	out.Blocks = read
	return out
}

// holdReasons fills in why the send-time recheck held a send, the counts
// to compare and when the hold expires. A hold the sending engine recorded
// carries only titles: its reasons are worked out again from the latest run
// (the send-time recheck) and what was confirmed.
func (s *Service) holdReasons(ctx context.Context, h store.CampaignIncident, d HoldDetail, out *ReviewHeld) {
	l := i18n.LangFrom(ctx)
	wsID, err := s.Q.ReadinessCampaignWorkspace(ctx, h.CampaignID)
	if err != nil {
		return
	}
	w, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return
	}
	c, err := s.Readiness.Campaign(ctx, &w, h.CampaignID)
	if err != nil {
		return
	}
	if c.ScheduledAt != nil {
		at := c.ScheduledAt.Add(readiness.HeldExpiry)
		out.ExpiresAt = &at
	}
	conf, _ := s.Readiness.Confirmed(ctx, h.CampaignID)
	holds, audience, confirmed := d.Holds, d.Audience, d.Confirmed
	latest, _ := s.Readiness.Latest(ctx, h.CampaignID)
	if len(holds) == 0 && latest != nil {
		check := readiness.CheckSendTime(*latest, conf, s.Readiness.Digests(ctx, &w, c))
		holds, audience, confirmed = check.Holds, check.Audience, check.Confirmed
	}
	for _, x := range holds {
		out.Holds = append(out.Holds, ReviewHoldReason{Kind: x.Kind, Text: x.Text.In(l)})
	}
	if len(out.Holds) > 0 && out.Holds[0].Kind != readiness.HoldBlocked {
		out.Reason = out.Holds[0].Text
	}
	if conf != nil && confirmed > 0 {
		out.Reviewed = &ReviewCount{At: conf.ConfirmedAt, Recipients: confirmed}
	}
	if latest != nil && audience > 0 {
		out.Rechecked = &ReviewCount{At: latest.RanAt, Recipients: audience}
	}
}

// hold records that a send didn't start and tells the author and approvers;
// check says why the send-time recheck held it (nil for other reasons).
func (s *Service) hold(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, run *readiness.ReadinessRun, reason string, check *readiness.SendTimeCheck) error {
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
	if check != nil {
		detail.Holds, detail.Audience, detail.Confirmed = check.Holds, check.Audience, check.Confirmed
	}
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
		n.TitleText = mNotifyHeldTitle.Keep(nil)
		for _, id := range targets {
			if seen[id] {
				continue
			}
			seen[id] = true
			// The time and the reason are written in the language of the member it goes to.
			rl := readerLang(ctx, q, id)
			why := gapIn(rl, reason)
			if check != nil && len(check.Holds) > 0 {
				why = check.Holds[0].Text.In(rl)
			}
			args := i18n.Args{"name": c.Name, "time": rl.Region.Time(now.In(state.Location(ws.Timezone)), ""), "reason": why}
			n.SummaryText, n.BodyText = mNotifyHeldSummary.Keep(args), mNotifyHeldBody.Keep(args)
			if err := notify.ToUser(ctx, tx, id, n); err != nil {
				return err
			}
		}
		return nil
	})
}

// startProblem is why the engine couldn't start a send, as the API answers
// it in the request's language.
func startProblem(ctx context.Context, err error) error {
	var ex sending.Explained
	switch {
	case errors.Is(err, sending.ErrUnavailable):
		return api.Err(http.StatusServiceUnavailable, "sending_unavailable", say(ctx, mSendingUnavailable, nil))
	case errors.Is(err, sending.ErrNotReady):
		msg := say(ctx, mBlockedAsStarted, nil)
		var nr *sending.NotReadyError
		if errors.As(err, &nr) && nr.Reason != "" {
			msg = say(ctx, mBlockedAsStartedReason, i18n.Args{"reason": strings.TrimSuffix(gapIn(langOf(ctx), nr.Reason), ".")})
		}
		return api.Err(http.StatusConflict, "readiness_blocked", msg)
	case errors.As(err, &ex):
		return api.Conflict(ex.In(langOf(ctx)))
	}
	return err
}
