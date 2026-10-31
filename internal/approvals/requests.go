package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// Decisions.
const (
	DecisionApprove        = "approve"
	DecisionRequestChanges = "request_changes"
	DecisionReject         = "reject"
)

// ApprovalRequestInput asks for approval.
type ApprovalRequestInput struct {
	Note string `json:"note,omitempty" maxLength:"2000" doc:"A note for approvers"`
}

// ApprovalDecisionInput is an approver's decision.
type ApprovalDecisionInput struct {
	Decision string `json:"decision" enum:"approve,request_changes,reject"`
	Comment  string `json:"comment,omitempty" maxLength:"2000"`
}

// ApprovalCommentInput adds to the discussion.
type ApprovalCommentInput struct {
	Body string `json:"body" minLength:"1" maxLength:"4000"`
}

// facts gathers what the rules look at for a campaign sent by author.
func (s *Service) ruleFacts(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, run *readiness.ReadinessRun, author member) (RuleFacts, error) {
	f := RuleFacts{AuthorCanSend: author.Perms.Has(authz.CampaignsSend), AuthorName: author.Name, AuthorRole: author.RoleName, Now: s.now()}
	if run != nil {
		f.Recipients = run.Estimates.Audience.Final
		f.Verdict = run.Verdict
		f.Warnings = run.Counts.Warnings
		provs, err := s.Q.ProvListActive(ctx, ws.ID)
		if err != nil {
			return f, err
		}
		connected := map[string]time.Time{}
		for _, p := range provs {
			connected[p.ID.String()] = p.ConnectedAt
		}
		for _, sh := range run.Estimates.Route.Shares {
			f.Shares = append(f.Shares, RuleShare{Short: sh.Short, Recipients: sh.Recipients, WarmingUp: sh.Status == state.WarmingUp, WarmupDay: sh.WarmupDay,
				WarmupDays: sh.WarmupDays, ConnectedAt: connected[sh.ProviderID]})
		}
	}
	lists, err := s.Q.ApprovalAudienceLists(ctx, ws.ID)
	if err != nil {
		return f, err
	}
	def := readiness.ParseAudience(c.Audience)
	for _, l := range lists {
		if !l.RequiresApproval {
			continue
		}
		for _, id := range def.Lists {
			if id == l.ID.String() {
				f.ApprovalLists = append(f.ApprovalLists, l.Name)
			}
		}
	}
	return f, nil
}

// freshRun returns the latest run when it still describes the campaign.
// When it doesn't, the checks are queued to run again (in the background:
// no readiness work happens inside a request) and the request is refused
// until they finish.
func (s *Service) freshRun(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, trigger string, by *uuid.UUID) (*readiness.ReadinessRun, error) {
	run, err := s.Readiness.Latest(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	if run != nil && run.Fingerprint == s.Readiness.CurrentFingerprint(ctx, ws, c) {
		return run, nil
	}
	if _, err := s.Readiness.Start(ctx, ws, c.ID, readiness.RunOptions{Trigger: trigger, By: by}); err != nil && !errors.Is(err, readiness.ErrRunning) {
		return nil, err
	}
	return nil, api.Err(http.StatusConflict, "readiness_running", say(ctx, mReadinessRunning, nil))
}

// buildSnapshot records what a request covers.
func (s *Service) buildSnapshot(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow, run *readiness.ReadinessRun, rules []ApprovalRuleMatch, author member) (Snapshot, error) {
	rc, err := s.Readiness.Route(ctx, ws, c.RouteOverride)
	if err != nil {
		return Snapshot{}, err
	}
	snap := Snapshot{Version: int(c.Version), ContentHash: c.ContentHash, Subject: c.Subject, Preheader: c.Preheader, From: fromLine(c.SenderName, c.SenderEmail),
		Links: Links(c.Mjml, c.Html), Sections: Sections(c.Mjml), Audience: c.Audience, Schedule: c.Schedule, Rules: rules, AuthorRole: author.RoleName,
		RouteKey: rc.Key, SplitLabel: rc.StreamLabel, Route: []SnapshotShare{}, AudienceLabels: []string{}}
	if c.SenderID != nil {
		snap.SenderID = c.SenderID.String()
	}
	if s.Languages != nil {
		if snap.Languages, err = s.Languages.CampaignSnapshot(ctx, s.Q, c.ID); err != nil {
			return Snapshot{}, err
		}
	}
	ws2 := *ws
	snap.Thresholds = readiness.ParseThresholds(c.Thresholds, ws2.SendThresholds)
	if a, err := s.Readiness.Audience(ctx, ws, c.Audience); err == nil && !a.Empty {
		snap.AudienceBaseline = a.Final
	}
	if run != nil {
		e := run.Estimates
		snap.AudienceCount, snap.AudienceLabels, snap.Suppressed, snap.Role = e.Audience.Final, e.Audience.Lists, e.Audience.Suppressed, e.Audience.Role
		for _, ch := range run.Checks {
			if ch.Key == "audience.bounced" && ch.Status != readiness.Pass {
				snap.Bounced = 1
			}
		}
		snap.Minutes, snap.Cost = e.Route.TotalMinutes, e.Route.TotalCost
		for i, sh := range e.Route.Shares {
			snap.Route = append(snap.Route, SnapshotShare{ProviderID: sh.ProviderID, Name: sh.Name, Short: sh.Short, Label: sh.Label, Adapter: sh.Adapter, Percent: sh.Percent,
				Recipients: sh.Recipients, Cost: sh.Cost, PriceLabel: sh.PriceLabel, Status: sh.Status, StatusLabel: sh.StatusLabel, WarmupDay: sh.WarmupDay, WarmupDays: sh.WarmupDays,
				StatusText: e.ShareStatus(i)})
		}
		snap.Readiness = SnapshotReadiness{RunID: run.ID, Verdict: run.Verdict, Passed: run.Counts.Passed, Warnings: run.Counts.Warnings, Blocks: run.Counts.Blocks,
			Total: run.Counts.Total, RanAt: run.RanAt, Issues: []readiness.ReadinessCheck{}}
		for _, ch := range run.Checks {
			if ch.Status != readiness.Pass {
				snap.Readiness.Issues = append(snap.Readiness.Issues, ch)
			}
		}
	}
	return snap, nil
}

// RequestApproval asks the workspace's approvers to approve a send.
func (s *Service) RequestApproval(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, in ApprovalRequestInput) (uuid.UUID, error) {
	if hook := s.channelHook(ctx, campaignID); hook != nil {
		return s.requestChannel(ctx, ws, p, campaignID, hook, in)
	}
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return uuid.Nil, err
	}
	switch c.Status {
	case "draft", "approval", "approved":
	case "scheduled":
		// The time stays reserved while approval is asked for again; the
		// send-time recheck keeps it from starting until it is approved.
	default:
		return uuid.Nil, api.Conflict(say(ctx, mAlreadySending, nil))
	}
	if open, err := s.open(ctx, campaignID); err != nil {
		return uuid.Nil, err
	} else if open != nil {
		if open.Status == "pending" {
			return uuid.Nil, api.Conflict(say(ctx, mPendingRequest, nil))
		}
		if changes, err := s.Revalidate(ctx, ws, campaignID); err != nil {
			return uuid.Nil, err
		} else if len(changes) == 0 {
			if c.Status == "scheduled" {
				return uuid.Nil, api.Conflict(say(ctx, mApprovedScheduled, nil))
			}
			return uuid.Nil, api.Conflict(say(ctx, mAlreadyApproved, nil))
		}
	}
	run, err := s.freshRun(ctx, ws, c, readiness.TriggerApproval, &p.UserID)
	if err != nil {
		return uuid.Nil, err
	}
	if run.Verdict == readiness.VerdictBlocked {
		return uuid.Nil, api.Err(http.StatusConflict, "readiness_blocked", say(ctx, mReadinessBlocked, i18n.Args{"count": run.Counts.Blocks}))
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	author := memberByID(ms, &p.UserID)
	if author == nil {
		return uuid.Nil, api.Forbidden(authz.CampaignsEdit, nil)
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	facts, err := s.ruleFacts(ctx, ws, c, run, *author)
	if err != nil {
		return uuid.Nil, err
	}
	rules := policy.Evaluate(facts)
	if len(rules) == 0 {
		if !author.Perms.Has(authz.CampaignsSend) {
			return uuid.Nil, api.Forbidden(authz.CampaignsSend, approverNames(ms, nil, authz.CampaignsSend))
		}
		return uuid.Nil, api.Err(http.StatusConflict, "approval_not_required", say(ctx, mNotRequired, nil))
	}
	approvers := []member{}
	for _, a := range approversOf(ms) {
		if a.ID != p.UserID {
			approvers = append(approvers, a)
		}
	}
	if len(approvers) < requiredFor(policy, rules) {
		return uuid.Nil, api.Err(http.StatusConflict, "no_approvers", say(ctx, mNoApprovers, i18n.Args{"needed": policy.ApprovalsRequired, "have": len(approvers)}))
	}
	if err := s.needClientApprover(ctx, ws, rules, approvers); err != nil {
		return uuid.Nil, err
	}
	snap, err := s.buildSnapshot(ctx, ws, c, run, rules, *author)
	if err != nil {
		return uuid.Nil, err
	}
	sb, _ := json.Marshal(snap)
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	note := strings.TrimSpace(in.Note)
	var previous *uuid.UUID
	if last, err := s.Q.ApprovalLatestForCampaign(ctx, &campaignID); err == nil && last.Status != "pending" && last.Status != "approved" {
		previous = &last.ID
	}
	runID, _ := uuid.Parse(run.ID)
	version := c.Version
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "campaign", CampaignID: &campaignID,
			RequestedBy: &p.UserID, RequestedAt: now, Note: note, Status: "pending", MatchedRules: ruleKeys(rules), ApprovalsRequired: int32(requiredFor(policy, rules)),
			Snapshot: sb, CampaignVersion: &version, ReadinessRunID: &runID, PreviousID: previous}); err != nil {
			if strings.Contains(err.Error(), "approval_requests_open_campaign") {
				return api.Conflict(say(ctx, mRequestedForCampaign, nil))
			}
			return err
		}
		if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: "approval", Now: now, ID: campaignID, FromStatuses: []string{"draft", "approved"}}); err != nil {
			return err
		}
		summary := fmt.Sprintf("%s recipients · %s", readiness.Count(snap.AudienceCount), strings.ReplaceAll(rulesLine(rules), " · ", ", "))
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.requested", TargetType: "campaign", TargetID: campaignID.String(), TargetName: c.Name,
			After:    map[string]any{"status": "pending", "rules": ruleKeys(rules), "version": c.Version, "audience": snap.AudienceCount},
			Metadata: map[string]any{"summary": summary, "detail": readiness.Count(snap.AudienceCount) + " recipients", "request_id": id.String(), "permission": "campaigns:edit"}}); err != nil {
			return err
		}
		if err := emitRequested(ctx, tx, ws, id, "campaign", campaignID, c.Name, author.Name, ruleKeys(rules), snap.AudienceCount); err != nil {
			return err
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested",
			Summary: fmt.Sprintf("%s asked you to approve “%s” · %s recipients", author.Name, c.Name, readiness.Count(snap.AudienceCount)),
			Body: fmt.Sprintf("%s asked for approval: “%s” · %s recipients · rules: %s", author.Name, c.Name, readiness.Count(snap.AudienceCount),
				strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(rulesLine(rules), "rules: "), "rule: "), " · ", ", ")),
			Link: "/w/" + ws.Slug + "/approvals/" + id.String(), Data: map[string]any{"request_id": id.String(), "campaign_id": campaignID.String()}}
		n.TitleText = mNotifyRequestedTitle.Keep(nil)
		n.SummaryText = mNotifyRequestedSummary.Keep(i18n.Args{"author": author.Name, "name": c.Name, "count": snap.AudienceCount})
		for _, a := range approvers {
			n.BodyText = mNotifyRequestedBody.Keep(i18n.Args{"author": author.Name, "name": c.Name, "count": snap.AudienceCount,
				"rules": ruleList(readerLang(ctx, q, a.ID), rules)})
			if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

func approverNames(ms []member, except *uuid.UUID, perm authz.Permission) []string {
	out := []string{}
	for _, m := range ms {
		if m.Perms.Has(perm) && (except == nil || m.ID != *except) {
			out = append(out, m.Name)
			if len(out) == 2 {
				break
			}
		}
	}
	return out
}

// Decide records an approver's decision. Approving an invalidated request
// approves the campaign as it stands now (a new request with the diff).
func (s *Service) Decide(ctx context.Context, ws *store.Workspace, p *authz.Principal, requestID uuid.UUID, in ApprovalDecisionInput) (uuid.UUID, error) {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: requestID})
	if err != nil {
		return uuid.Nil, mapErr(ctx, err)
	}
	if r.RequestedBy != nil && *r.RequestedBy == p.UserID {
		return uuid.Nil, api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, mOwnCampaign, nil))
	}
	if r.SubjectType == "campaign" && r.CampaignID != nil {
		if r.Status == "pending" || r.Status == "approved" {
			if changes, err := s.Revalidate(ctx, ws, *r.CampaignID); err != nil {
				return uuid.Nil, err
			} else if len(changes) > 0 {
				r.Status = "invalidated"
			}
		}
		if r.Status == "invalidated" {
			// Decide on the campaign as it stands: a new request follows this one.
			next, err := s.rerequest(ctx, ws, r)
			if err != nil {
				return uuid.Nil, err
			}
			requestID = next
		}
	}
	comment := strings.TrimSpace(in.Comment)
	var notifyAfter func()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		locked, err := q.ApprovalLock(ctx, store.ApprovalLockParams{WorkspaceID: ws.ID, ID: requestID})
		if err != nil {
			return mapErr(ctx, err)
		}
		if locked.Status != "pending" {
			return api.Conflict(say(ctx, mRequestIsAlready, i18n.Args{"status": statusSelect(locked.Status)}))
		}
		decisions, err := q.ApprovalDecisions(ctx, []uuid.UUID{requestID})
		if err != nil {
			return err
		}
		given := 0
		approved := []uuid.UUID{p.UserID}
		for _, d := range decisions {
			if d.ApproverID != nil && *d.ApproverID == p.UserID {
				return api.Conflict(say(ctx, mAlreadyDecided, nil))
			}
			if d.Decision == DecisionApprove {
				given++
				if d.ApproverID != nil {
					approved = append(approved, *d.ApproverID)
				}
			}
		}
		// Client sign-off: approvals from agency staff don't count toward it.
		signedOff, err := signoffMet(ctx, q, locked.MatchedRules, approved)
		if err != nil {
			return err
		}
		now := s.now()
		if err := q.ApprovalInsertDecision(ctx, store.ApprovalInsertDecisionParams{ID: uuid.Must(uuid.NewV7()), RequestID: requestID, ApproverID: &p.UserID, Decision: in.Decision,
			Comment: comment, CreatedAt: now}); err != nil {
			return err
		}
		status := "pending"
		switch in.Decision {
		case DecisionApprove:
			if given+1 >= int(locked.ApprovalsRequired) && signedOff {
				status = "approved"
			}
		case DecisionRequestChanges:
			status = "changes_requested"
		case DecisionReject:
			status = "rejected"
		}
		if status != "pending" {
			if err := q.ApprovalSetStatus(ctx, store.ApprovalSetStatusParams{Status: status, DecidedAt: &now, DecidedBy: &p.UserID, ID: requestID}); err != nil {
				return err
			}
		}
		name := r.SubjectName
		action := map[string]string{DecisionApprove: "campaign.approved", DecisionRequestChanges: "campaign.changes_requested", DecisionReject: "campaign.approval_rejected"}[in.Decision]
		targetType, targetID := "campaign", ""
		if locked.CampaignID != nil {
			targetID = locked.CampaignID.String()
			campaignStatus := map[string]string{"approved": "approved", "changes_requested": "draft", "rejected": "draft"}[status]
			if campaignStatus != "" {
				if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: campaignStatus, Now: now, ID: *locked.CampaignID, FromStatuses: []string{"approval"}}); err != nil {
					return err
				}
			}
		}
		if locked.SubjectType == "automation" && locked.AutomationID != nil {
			targetType, targetID = "automation", locked.AutomationID.String()
			action = map[string]string{DecisionApprove: "automation.publish_approved", DecisionRequestChanges: "automation.publish_changes_requested", DecisionReject: "automation.publish_rejected"}[in.Decision]
		}
		if locked.SubjectType == "transactional_template" && locked.TransactionalTemplateID != nil {
			targetType, targetID, action = transactionalDecisionAudit(*locked.TransactionalTemplateID, in.Decision)
		}
		if locked.SubjectType == "website_post" && locked.WebsitePostID != nil {
			targetType, targetID, action = websiteDecisionAudit(*locked.WebsitePostID, in.Decision)
		}
		if locked.SubjectType == "website_page" && locked.WebsitePageID != nil {
			targetType, targetID, action = pageDecisionAudit(*locked.WebsitePageID, in.Decision)
		}
		meta := map[string]any{"request_id": requestID.String(), "approvals": fmt.Sprintf("%d of %d", min(given+1, int(locked.ApprovalsRequired)), locked.ApprovalsRequired), "permission": "campaigns:approve"}
		if in.Decision == DecisionApprove && !signedOff {
			// An agency's staff approved: recorded, but client sign-off still waits.
			meta["client_signoff"] = "waiting for a Client approver"
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: targetType, TargetID: targetID, TargetName: name,
			Before: map[string]any{"status": "pending"}, After: map[string]any{"status": status, "comment": comment}, Metadata: meta}); err != nil {
			return err
		}
		if status != "pending" {
			if err := emitDecided(ctx, tx, ws, locked, name, status, p.Name, comment); err != nil {
				return err
			}
		}
		if status == "pending" || locked.RequestedBy == nil {
			return nil
		}
		approver, _ := s.Q.GetUser(ctx, p.UserID)
		title, sev, verb := "Approval given", "ok", "approved"
		switch status {
		case "changes_requested":
			title, sev, verb = "Changes requested", "warn", "asked for changes to"
		case "rejected":
			title, sev, verb = "Approval rejected", "bad", "rejected"
		}
		summary := fmt.Sprintf("%s %s “%s”", approver.Name, verb, name)
		if status == "approved" && locked.SubjectType == "campaign" {
			snap := ParseSnapshot(locked.Snapshot)
			if t := ScheduleText(snap.Schedule, state.Location(ws.Timezone)); t != "Send now" {
				summary += " · scheduled " + t
			}
		}
		body := summary
		if comment != "" {
			body += ": “" + comment + "”"
		}
		link := "/w/" + ws.Slug + "/approvals/" + requestID.String()
		if locked.CampaignID != nil {
			link = "/w/" + ws.Slug + "/campaigns/" + locked.CampaignID.String() + "/review"
		}
		if locked.TransactionalTemplateID != nil {
			link, summary, body = transactionalDecisionNote(ws, *locked.TransactionalTemplateID, ParseSnapshot(locked.Snapshot), status, summary, body)
		}
		if locked.WebsitePostID != nil {
			link, summary, body = websiteDecisionNote(ws, *locked.WebsitePostID, ParseSnapshot(locked.Snapshot), status, summary, body)
		}
		if locked.WebsitePageID != nil {
			link, summary, body = pageDecisionNote(ws, *locked.WebsitePageID, ParseSnapshot(locked.Snapshot), status, summary, body)
		}
		kept := decisionNoteArgs(ctx, q, ws, locked, status, approver.Name, name, comment)
		if err := notify.ToUser(ctx, tx, *locked.RequestedBy, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalDecided, Severity: sev, Title: title,
			Summary: summary, Body: body, Link: link, Data: map[string]any{"request_id": requestID.String(), "decision": in.Decision},
			TitleText: mNotifyDecidedTitle.Keep(i18n.Args{"status": status}), SummaryText: mNotifyDecidedSummary.Keep(kept), BodyText: mNotifyDecidedBody.Keep(kept)}); err != nil {
			return err
		}
		if status == "approved" && locked.SubjectType == "website_post" && locked.WebsitePostID != nil && s.Website != nil {
			post, approverName := *locked.WebsitePostID, approver.Name
			notifyAfter = func() {
				if w := s.Website(); w != nil {
					if err := w.PublishApproved(context.WithoutCancel(ctx), ws, post, requestID.String(), approverName); err != nil && s.Log != nil {
						s.Log.WarnContext(ctx, "publish the approved post", "post", post, "err", err)
					}
				}
			}
		}
		if status == "approved" && locked.SubjectType == "website_page" && locked.WebsitePageID != nil && s.Pages != nil {
			page, approverName, hash := *locked.WebsitePageID, approver.Name, ParseSnapshot(locked.Snapshot).DraftHash
			notifyAfter = func() {
				if pp := s.Pages(); pp != nil {
					if err := pp.PublishApproved(context.WithoutCancel(ctx), ws, page, requestID.String(), approverName, hash); err != nil && s.Log != nil {
						s.Log.WarnContext(ctx, "publish the approved page", "page", page, "err", err)
					}
				}
			}
		}
		if status == "approved" && locked.SubjectType == "automation" && locked.AutomationID != nil {
			auto, version, migration, by := *locked.AutomationID, ParseSnapshot(locked.Snapshot).Automation, "", *locked.RequestedBy
			if version != nil {
				migration = version.Migration
				notifyAfter = func() { s.completePublish(context.WithoutCancel(ctx), ws, by, auto, version.Version, migration) }
			}
		}
		return nil
	})
	if err == nil && notifyAfter != nil {
		notifyAfter()
	}
	return requestID, err
}

// rerequest opens a new request for the campaign as it stands, following an
// invalidated one (approvers then decide on the new version with the diff).
func (s *Service) rerequest(ctx context.Context, ws *store.Workspace, prev store.ApprovalGetRow) (uuid.UUID, error) {
	if prev.CampaignID == nil || prev.RequestedBy == nil {
		return uuid.Nil, api.Conflict(say(ctx, mCantRenew, nil))
	}
	if hook := s.channelHook(ctx, *prev.CampaignID); hook != nil {
		return s.rerequestChannel(ctx, ws, prev, hook)
	}
	c, err := s.Readiness.Campaign(ctx, ws, *prev.CampaignID)
	if err != nil {
		return uuid.Nil, err
	}
	if open, err := s.open(ctx, c.ID); err != nil {
		return uuid.Nil, err
	} else if open != nil {
		return open.ID, nil
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	author := memberByID(ms, prev.RequestedBy)
	if author == nil {
		return uuid.Nil, api.Conflict(say(ctx, mAuthorGone, nil))
	}
	run, err := s.freshRun(ctx, ws, c, readiness.TriggerApproval, nil)
	if err != nil {
		return uuid.Nil, err
	}
	if run.Verdict == readiness.VerdictBlocked {
		return uuid.Nil, api.Err(http.StatusConflict, "readiness_blocked", say(ctx, mReadinessBlockedNew, nil))
	}
	facts, err := s.ruleFacts(ctx, ws, c, run, *author)
	if err != nil {
		return uuid.Nil, err
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	rules := policy.Evaluate(facts)
	if len(rules) == 0 {
		rules = ParseSnapshot(prev.Snapshot).Rules
	}
	snap, err := s.buildSnapshot(ctx, ws, c, run, rules, *author)
	if err != nil {
		return uuid.Nil, err
	}
	sb, _ := json.Marshal(snap)
	id := uuid.Must(uuid.NewV7())
	runID, _ := uuid.Parse(run.ID)
	version := c.Version
	prevID := prev.ID
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "campaign", CampaignID: &c.ID,
			RequestedBy: prev.RequestedBy, RequestedAt: now, Note: prev.Note, Status: "pending", MatchedRules: ruleKeys(rules), ApprovalsRequired: int32(policy.ApprovalsRequired),
			Snapshot: sb, CampaignVersion: &version, ReadinessRunID: &runID, PreviousID: &prevID}); err != nil {
			return err
		}
		if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: "approval", Now: now, ID: c.ID, FromStatuses: []string{"draft", "approved"}}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.requested", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			After:    map[string]any{"status": "pending", "version": c.Version, "follows": prevID.String()},
			Metadata: map[string]any{"summary": fmt.Sprintf("v%d after the approval of v%d was invalidated", c.Version, ParseSnapshot(prev.Snapshot).Version), "request_id": id.String()}}); err != nil {
			return err
		}
		return emitRequested(ctx, tx, ws, id, "campaign", c.ID, c.Name, author.Name, ruleKeys(rules), snap.AudienceCount)
	})
	return id, err
}

// Comment adds to a request's discussion (author and approvers).
func (s *Service) Comment(ctx context.Context, ws *store.Workspace, p *authz.Principal, requestID uuid.UUID, in ApprovalCommentInput) error {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: requestID})
	if err != nil {
		return mapErr(ctx, err)
	}
	if !canSee(p, r.RequestedBy) {
		return api.Forbidden(authz.CampaignsApprove, nil)
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return api.Invalid("body.body", say(ctx, mWriteComment, nil))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalInsertComment(ctx, store.ApprovalInsertCommentParams{ID: uuid.Must(uuid.NewV7()), RequestID: requestID, AuthorID: &p.UserID, Body: body, CreatedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.commented", TargetType: "approval_request", TargetID: requestID.String(), TargetName: r.SubjectName,
			After: map[string]any{"comment": body}})
	})
}

// canSee: approvers see every request; others only their own.
func canSee(p *authz.Principal, requester *uuid.UUID) bool {
	return p.Can(authz.CampaignsApprove) || (requester != nil && *requester == p.UserID)
}

// Cancel withdraws a request (its author, or an Owner).
func (s *Service) Cancel(ctx context.Context, ws *store.Workspace, p *authz.Principal, requestID uuid.UUID) error {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: requestID})
	if err != nil {
		return mapErr(ctx, err)
	}
	if (r.RequestedBy == nil || *r.RequestedBy != p.UserID) && p.RoleKey != authz.RoleOwner {
		return api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, mCancelNotAllowed, nil))
	}
	if r.Status != "pending" && r.Status != "approved" {
		return api.Conflict(say(ctx, mRequestIsAlready, i18n.Args{"status": statusSelect(r.Status)}))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if err := q.ApprovalSetStatus(ctx, store.ApprovalSetStatusParams{Status: "cancelled", DecidedAt: &now, DecidedBy: &p.UserID, ID: requestID}); err != nil {
			return err
		}
		if r.CampaignID != nil {
			if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: "draft", Now: now, ID: *r.CampaignID, FromStatuses: []string{"approval", "approved"}}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.cancelled", TargetType: "approval_request", TargetID: requestID.String(), TargetName: r.SubjectName,
			Before: map[string]any{"status": r.Status}, After: map[string]any{"status": "cancelled"}})
	})
}

// RemindEvery is how often an author can remind the approvers.
const RemindEvery = time.Hour

// Remind notifies the approvers of a pending request again.
func (s *Service) Remind(ctx context.Context, ws *store.Workspace, p *authz.Principal, requestID uuid.UUID) error {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: requestID})
	if err != nil {
		return mapErr(ctx, err)
	}
	if r.RequestedBy == nil || *r.RequestedBy != p.UserID {
		return api.Err(http.StatusForbidden, api.CodePermissionDenied, say(ctx, mRemindNotAllowed, nil))
	}
	if r.Status != "pending" {
		return api.Conflict(say(ctx, mRemindPendingOnly, nil))
	}
	now := s.now()
	if r.RemindedAt != nil && now.Sub(*r.RemindedAt) < RemindEvery {
		next := clock(langOf(ctx), r.RemindedAt.Add(RemindEvery).In(state.Location(ws.Timezone)))
		return api.Err(http.StatusTooManyRequests, api.CodeRateLimited, say(ctx, mRemindedRecently, i18n.Args{"time": next}))
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return err
	}
	snap := ParseSnapshot(r.Snapshot)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalSetReminded(ctx, store.ApprovalSetRemindedParams{RemindedAt: &now, ID: requestID}); err != nil {
			return err
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested · reminder",
			Summary: fmt.Sprintf("%s is waiting for your approval of “%s” · %s recipients", r.RequestedByName, r.SubjectName, readiness.Count(snap.AudienceCount)),
			Body:    fmt.Sprintf("%s is still waiting for approval of “%s” · %s recipients.", r.RequestedByName, r.SubjectName, readiness.Count(snap.AudienceCount)),
			Link:    "/w/" + ws.Slug + "/approvals/" + requestID.String(), Data: map[string]any{"request_id": requestID.String()},
			TitleText:   mNotifyReminderTitle.Keep(nil),
			SummaryText: mNotifyReminderSummary.Keep(i18n.Args{"author": r.RequestedByName, "name": r.SubjectName, "count": snap.AudienceCount}),
			BodyText:    mNotifyReminderBody.Keep(i18n.Args{"author": r.RequestedByName, "name": r.SubjectName, "count": snap.AudienceCount})}
		for _, a := range approversOf(ms) {
			if a.ID == p.UserID {
				continue
			}
			if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.reminded", TargetType: "approval_request", TargetID: requestID.String(), TargetName: r.SubjectName})
	})
}

// Restore puts the campaign back as the snapshot had it (the version, the
// audience, the route and the schedule), which keeps the approval.
func (s *Service) Restore(ctx context.Context, ws *store.Workspace, p *authz.Principal, requestID uuid.UUID) error {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: requestID})
	if err != nil {
		return mapErr(ctx, err)
	}
	if r.Status != "invalidated" || r.CampaignID == nil {
		return api.Conflict(say(ctx, mRestoreInvalidatedOnly, nil))
	}
	if !canSee(p, r.RequestedBy) {
		return api.Forbidden(authz.CampaignsApprove, nil)
	}
	if open, err := s.open(ctx, *r.CampaignID); err != nil {
		return err
	} else if open != nil {
		return api.Conflict(say(ctx, mNewerRequest, nil))
	}
	snap := ParseSnapshot(r.Snapshot)
	c, err := s.Readiness.Campaign(ctx, ws, *r.CampaignID)
	if err != nil {
		return err
	}
	if c.ContentHash != snap.ContentHash {
		if s.Campaigns == nil {
			return api.Conflict(say(ctx, mRestoreUnavailable, nil))
		}
		if _, err := s.Campaigns.Restore(ctx, ws, p, c.ID, snap.Version, campaigns.ComposerRestoreVersion{BaseVersion: int(c.Version)}); err != nil {
			return err
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if !sameAudience(snap.Audience, c.Audience) && len(snap.Audience) > 0 {
			if err := q.ReadinessSetAudience(ctx, store.ReadinessSetAudienceParams{Audience: snap.Audience, AudienceCount: &snap.AudienceCount, UpdatedBy: &p.UserID, Now: now, WorkspaceID: ws.ID, ID: c.ID}); err != nil {
				return err
			}
		}
		if !sameSchedule(snap.Schedule, c.Schedule) && len(snap.Schedule) > 0 {
			if err := q.ReadinessSetSendSettings(ctx, store.ReadinessSetSendSettingsParams{Thresholds: c.Thresholds, Schedule: snap.Schedule, UpdatedBy: &p.UserID, Now: now, WorkspaceID: ws.ID, ID: c.ID}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	c, err = s.Readiness.Campaign(ctx, ws, *r.CampaignID)
	if err != nil {
		return err
	}
	cur, err := s.current(ctx, ws, c)
	if err != nil {
		return err
	}
	if changes := diff(snap, cur, state.Location(ws.Timezone)); len(changes) > 0 {
		return api.Err(http.StatusConflict, "restore_incomplete", say(ctx, mRestoreIncomplete, i18n.Args{"reason": render(langOf(ctx), reasonOf(changes))}))
	}
	status := r.InvalidatedFrom
	if status != "pending" && status != "approved" {
		status = "approved"
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalReopen(ctx, store.ApprovalReopenParams{Status: status, ID: requestID}); err != nil {
			return err
		}
		campaignStatus := map[string]string{"approved": "approved", "pending": "approval"}[status]
		if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: campaignStatus, Now: s.now(), ID: c.ID, FromStatuses: []string{"draft"}}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.restored", TargetType: "approval_request", TargetID: requestID.String(), TargetName: c.Name,
			Before: map[string]any{"status": "invalidated"}, After: map[string]any{"status": status, "version": snap.Version}})
	})
}
