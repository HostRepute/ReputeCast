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
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// Campaigns on channels other than email (SMS in phase 8; push and
// WhatsApp after it) go through the same approvals: the workspace's rules,
// requests with a snapshot of what they cover, invalidation when that
// changes, decisions, comments and the inbox. Their own Review & send and
// checks belong to the channel's domain, which describes a campaign here.

// ChannelCampaign is a channel campaign as the approval flow sees it.
type ChannelCampaign struct {
	Name    string
	Status  string
	Version int
	// Facts are what the rules look at; the author's fields are filled here.
	Facts RuleFacts
	// Snapshot is what a request covers; Current the same, for revalidation.
	Snapshot Snapshot
	Current  Current
	// Blocks counts blocking checks: a blocked campaign isn't sent for
	// approval.
	Blocks int
	// SendPermission is the channel's own send permission ("sms:send").
	SendPermission authz.Permission
}

// ChannelCampaigns describes campaigns of one channel; the channel's domain
// implements it.
type ChannelCampaigns interface {
	DescribeForApproval(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (ChannelCampaign, error)
}

// channelOf is a campaign's channel ("email" when unknown).
func (s *Service) channelOf(ctx context.Context, campaignID uuid.UUID) string {
	var ch string
	if err := s.Pool.QueryRow(ctx, `SELECT channel FROM campaigns WHERE id = $1`, campaignID).Scan(&ch); err != nil || ch == "" {
		return "email"
	}
	return ch
}

// channelHook is the describer of a campaign's channel, nil for email.
func (s *Service) channelHook(ctx context.Context, campaignID uuid.UUID) ChannelCampaigns {
	ch := s.channelOf(ctx, campaignID)
	if ch == "email" || s.Channels == nil {
		return nil
	}
	return s.Channels[ch]
}

// channelFacts fills the author's part of a channel campaign's facts.
func channelFacts(d ChannelCampaign, author member, now time.Time) RuleFacts {
	f := d.Facts
	f.AuthorName, f.AuthorRole = author.Name, author.RoleName
	f.AuthorCanSend = author.Perms.Has(authz.CampaignsSend) && (d.SendPermission == "" || author.Perms.Has(d.SendPermission))
	if !author.Perms.Has(authz.CampaignsSend) {
		f.SendPermission = string(authz.CampaignsSend)
	} else if d.SendPermission != "" && !author.Perms.Has(d.SendPermission) {
		f.SendPermission = string(d.SendPermission)
	}
	f.Now = now
	return f
}

// requestChannel asks for approval of a channel campaign.
func (s *Service) requestChannel(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, hook ChannelCampaigns, in ApprovalRequestInput) (uuid.UUID, error) {
	d, err := hook.DescribeForApproval(ctx, ws, campaignID)
	if err != nil {
		return uuid.Nil, err
	}
	switch d.Status {
	case "draft", "approval", "approved", "scheduled":
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
			return uuid.Nil, api.Conflict(say(ctx, mAlreadyApproved, nil))
		}
	}
	if d.Blocks > 0 {
		return uuid.Nil, api.Err(http.StatusConflict, "readiness_blocked", say(ctx, mChecksBlocked, i18n.Args{"count": d.Blocks}))
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
	facts := channelFacts(d, *author, s.now())
	rules := policy.Evaluate(facts)
	if len(rules) == 0 {
		if !facts.AuthorCanSend {
			perm := authz.Permission(facts.SendPermission)
			return uuid.Nil, api.Forbidden(perm, approverNames(ms, nil, perm))
		}
		return uuid.Nil, api.Err(http.StatusConflict, "approval_not_required", say(ctx, mNotRequired, nil))
	}
	approvers := []member{}
	for _, a := range approversOf(ms) {
		if a.ID != p.UserID {
			approvers = append(approvers, a)
		}
	}
	if len(approvers) < policy.ApprovalsRequired {
		return uuid.Nil, api.Err(http.StatusConflict, "no_approvers", say(ctx, mNoApprovers, i18n.Args{"needed": policy.ApprovalsRequired, "have": len(approvers)}))
	}
	snap := d.Snapshot
	snap.Rules, snap.AuthorRole = rules, author.RoleName
	return s.insertChannelRequest(ctx, ws, campaignID, d, snap, rules, p.UserID, strings.TrimSpace(in.Note), nil, policy.ApprovalsRequired, author.Name, approvers)
}

// insertChannelRequest writes a channel campaign's request, moves the
// campaign to approval, audits it and tells the approvers.
func (s *Service) insertChannelRequest(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, d ChannelCampaign, snap Snapshot, rules []ApprovalRuleMatch,
	by uuid.UUID, note string, previous *uuid.UUID, required int, authorName string, approvers []member) (uuid.UUID, error) {
	sb, _ := json.Marshal(snap)
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	version := int32(d.Version)
	if previous == nil {
		if last, err := s.Q.ApprovalLatestForCampaign(ctx, &campaignID); err == nil && last.Status != "pending" && last.Status != "approved" {
			previous = &last.ID
		}
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "campaign", CampaignID: &campaignID,
			RequestedBy: &by, RequestedAt: now, Note: note, Status: "pending", MatchedRules: ruleKeys(rules), ApprovalsRequired: int32(required),
			Snapshot: sb, CampaignVersion: &version, PreviousID: previous}); err != nil {
			if strings.Contains(err.Error(), "approval_requests_open_campaign") {
				return api.Conflict(say(ctx, mRequestedForCampaign, nil))
			}
			return err
		}
		if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: "approval", Now: now, ID: campaignID, FromStatuses: []string{"draft", "approved"}}); err != nil {
			return err
		}
		summary := fmt.Sprintf("%s recipients · %s", readiness.Count(snap.AudienceCount), strings.ReplaceAll(rulesLine(rules), " · ", ", "))
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.requested", TargetType: "campaign", TargetID: campaignID.String(), TargetName: d.Name,
			After:    map[string]any{"status": "pending", "rules": ruleKeys(rules), "version": d.Version, "audience": snap.AudienceCount, "channel": snap.Channel},
			Metadata: map[string]any{"summary": summary, "detail": readiness.Count(snap.AudienceCount) + " recipients", "request_id": id.String(), "permission": "campaigns:edit"}}); err != nil {
			return err
		}
		if err := emitRequested(ctx, tx, ws, id, "campaign", campaignID, d.Name, authorName, ruleKeys(rules), snap.AudienceCount); err != nil {
			return err
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested",
			Summary: fmt.Sprintf("%s asked you to approve “%s” · %s recipients", authorName, d.Name, readiness.Count(snap.AudienceCount)),
			Body: fmt.Sprintf("%s asked for approval: “%s” · %s recipients · rules: %s", authorName, d.Name, readiness.Count(snap.AudienceCount),
				strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(rulesLine(rules), "rules: "), "rule: "), " · ", ", ")),
			Link: "/w/" + ws.Slug + "/approvals/" + id.String(), Data: map[string]any{"request_id": id.String(), "campaign_id": campaignID.String()}}
		for _, a := range approvers {
			if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

// rerequestChannel opens a new request for a channel campaign as it
// stands, after an invalidated one.
func (s *Service) rerequestChannel(ctx context.Context, ws *store.Workspace, prev store.ApprovalGetRow, hook ChannelCampaigns) (uuid.UUID, error) {
	if open, err := s.open(ctx, *prev.CampaignID); err != nil {
		return uuid.Nil, err
	} else if open != nil {
		return open.ID, nil
	}
	d, err := hook.DescribeForApproval(ctx, ws, *prev.CampaignID)
	if err != nil {
		return uuid.Nil, err
	}
	if d.Blocks > 0 {
		return uuid.Nil, api.Err(http.StatusConflict, "readiness_blocked", say(ctx, mChecksBlockedNew, nil))
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	author := memberByID(ms, prev.RequestedBy)
	if author == nil {
		return uuid.Nil, api.Conflict(say(ctx, mAuthorGone, nil))
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	rules := policy.Evaluate(channelFacts(d, *author, s.now()))
	if len(rules) == 0 {
		rules = ParseSnapshot(prev.Snapshot).Rules
	}
	snap := d.Snapshot
	snap.Rules, snap.AuthorRole = rules, author.RoleName
	prevID := prev.ID
	return s.insertChannelRequest(ctx, ws, *prev.CampaignID, d, snap, rules, *prev.RequestedBy, prev.Note, &prevID, policy.ApprovalsRequired, author.Name, nil)
}

// ChannelGate checks the approval rules for a channel campaign sent by p.
// It returns the approved request that covers the campaign as it stands,
// nil when no rule matches, or a problem: approval_required or
// approval_invalidated.
func (s *Service) ChannelGate(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (*store.ApprovalRequest, error) {
	hook := s.channelHook(ctx, campaignID)
	if hook == nil {
		return nil, errors.New("approvals: not a channel campaign")
	}
	d, err := hook.DescribeForApproval(ctx, ws, campaignID)
	if err != nil {
		return nil, err
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return nil, err
	}
	author := memberByID(ms, &p.UserID)
	if author == nil {
		return nil, api.Forbidden(authz.CampaignsSend, nil)
	}
	rules := ParsePolicy(ws.ApprovalPolicy).Evaluate(channelFacts(d, *author, s.now()))
	if len(rules) == 0 {
		return nil, nil
	}
	if changes, err := s.revalidate(ctx, ws, campaignID); err != nil {
		return nil, err
	} else if len(changes) > 0 {
		return nil, s.noLongerCovers(ctx, changes)
	}
	open, err := s.open(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if open == nil || open.Status != "approved" {
		return nil, approvalRequired(ctx, rules)
	}
	return open, nil
}

// ChannelReview is the approval side of a channel campaign's Review & send.
func (s *Service) ChannelReview(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (ReviewApproval, error) {
	if _, err := s.Revalidate(ctx, ws, campaignID); err != nil {
		return ReviewApproval{}, err
	}
	hook := s.channelHook(ctx, campaignID)
	if hook == nil {
		return ReviewApproval{}, errors.New("approvals: not a channel campaign")
	}
	d, err := hook.DescribeForApproval(ctx, ws, campaignID)
	if err != nil {
		return ReviewApproval{}, err
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return ReviewApproval{}, err
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	out := ReviewApproval{Rules: []ApprovalRuleMatch{}, Approvers: []ApprovalPerson{}, Needed: policy.ApprovalsRequired, PolicyRules: policy.RuleLabelsIn(langOf(ctx)),
		MedianWaitMinutes: s.medianWait(ctx, ws)}
	for _, a := range approversOf(ms) {
		out.Approvers = append(out.Approvers, s.person(langOf(ctx), a))
	}
	if me := memberByID(ms, &p.UserID); me != nil {
		out.Rules = policy.Evaluate(channelFacts(d, *me, s.now()))
		out.Required = len(out.Rules) > 0
	}
	var req *store.ApprovalRequest
	if open, err := s.open(ctx, campaignID); err != nil {
		return out, err
	} else if open != nil {
		req = open
	} else if last, err := s.Q.ApprovalLatestForCampaign(ctx, &campaignID); err == nil {
		r := store.ApprovalRequest{ID: last.ID, WorkspaceID: last.WorkspaceID, SubjectType: last.SubjectType, CampaignID: last.CampaignID, RequestedBy: last.RequestedBy,
			RequestedAt: last.RequestedAt, Note: last.Note, Status: last.Status, MatchedRules: last.MatchedRules, ApprovalsRequired: last.ApprovalsRequired,
			Snapshot: last.Snapshot, InvalidatedAt: last.InvalidatedAt, InvalidatedReason: last.InvalidatedReason, Changes: last.Changes, BypassReason: last.BypassReason,
			DecidedAt: last.DecidedAt, CampaignVersion: last.CampaignVersion, DecidedBy: last.DecidedBy, InvalidatedBy: last.InvalidatedBy, RemindedAt: last.RemindedAt,
			PreviousID: last.PreviousID, InvalidatedFrom: last.InvalidatedFrom, InvalidatedReasonText: last.InvalidatedReasonText}
		req = &r
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if req != nil {
		v, err := s.requestView(ctx, ws, *req, ms)
		if err != nil {
			return out, err
		}
		out.Request = &v
		out.Valid = req.Status == "approved"
		if req.Status == "pending" || req.Status == "approved" {
			if snap := ParseSnapshot(req.Snapshot); len(snap.Rules) > 0 {
				out.Rules = snap.Rules
				out.Required = true
			}
			out.Needed = int(req.ApprovalsRequired)
		}
	}
	switch {
	case !p.Can(authz.CampaignsEdit):
		out.RequestReason = say(ctx, mReasonNeedsEdit, nil)
	case !out.Required:
		out.RequestReason = say(ctx, mReasonNoRule, nil)
	case d.Blocks > 0:
		out.RequestReason = say(ctx, mReasonFixBlocks, nil)
	case req != nil && req.Status == "pending":
		out.RequestReason = say(ctx, mReasonRequested, nil)
	case req != nil && req.Status == "approved":
		out.RequestReason = say(ctx, mReasonApproved, nil)
	default:
		others := 0
		for _, a := range approversOf(ms) {
			if a.ID != p.UserID {
				others++
			}
		}
		if others < policy.ApprovalsRequired {
			out.RequestReason = say(ctx, mReasonNoOtherApprover, nil)
		} else {
			out.CanRequest = true
		}
	}
	out.Rules = localRules(langOf(ctx), out.Rules)
	return out, nil
}
