package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// Recurring digests (design expansion, phase 3): every issue is a
// campaign, and the approval rules are checked for each one. When a rule
// matches, the request is made at preparation in the digest owner's name
// (nobody presses Send: issues go out on the owner's authority).

// ErrNoApprovers means no one other than the owner can approve the issue.
var ErrNoApprovers = errors.New("no approver other than the owner")

// IssueRequest describes the digest issue a request is for.
type IssueRequest struct {
	Digest string
	Number int
	// Held is true when Send Readiness blocked the issue at preparation:
	// approvers are told it waits for the block to be fixed.
	Held bool
	// Actor is who the audit entry names ("System · Engineering blog
	// roundup, turned on by Sofia Rossi").
	Actor *audit.Actor
}

// RequestForIssue asks the approvers to approve a digest issue, in its
// owner's name. It returns uuid.Nil when no rule matches (the issue needs
// no approval) and ErrNoApprovers when only the owner could approve it.
func (s *Service) RequestForIssue(ctx context.Context, ws *store.Workspace, ownerID, campaignID uuid.UUID, in IssueRequest) (uuid.UUID, error) {
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return uuid.Nil, err
	}
	if open, err := s.open(ctx, campaignID); err != nil {
		return uuid.Nil, err
	} else if open != nil {
		return open.ID, nil
	}
	run, err := s.Readiness.Latest(ctx, campaignID)
	if err != nil {
		return uuid.Nil, err
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	author := memberByID(ms, &ownerID)
	if author == nil {
		return uuid.Nil, fmt.Errorf("the digest's owner isn't a member of the workspace")
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	facts, err := s.ruleFacts(ctx, ws, c, run, *author)
	if err != nil {
		return uuid.Nil, err
	}
	rules := policy.Evaluate(facts)
	if len(rules) == 0 {
		return uuid.Nil, nil
	}
	approvers := []member{}
	for _, a := range approversOf(ms) {
		if a.ID != ownerID {
			approvers = append(approvers, a)
		}
	}
	if len(approvers) < policy.ApprovalsRequired {
		return uuid.Nil, ErrNoApprovers
	}
	snap, err := s.buildSnapshot(ctx, ws, c, run, rules, *author)
	if err != nil {
		return uuid.Nil, err
	}
	sb, _ := json.Marshal(snap)
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	version := c.Version
	var runID *uuid.UUID
	if run != nil {
		if rid, err := uuid.Parse(run.ID); err == nil {
			runID = &rid
		}
	}
	var previous *uuid.UUID
	if last, err := s.Q.ApprovalLatestForCampaign(ctx, &campaignID); err == nil && last.Status != "pending" && last.Status != "approved" {
		previous = &last.ID
	}
	short := strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(rulesLine(rules), "rules: "), "rule: "), " · ", ", ")
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "campaign", CampaignID: &campaignID,
			RequestedBy: &ownerID, RequestedAt: now, Note: "", Status: "pending", MatchedRules: ruleKeys(rules), ApprovalsRequired: int32(policy.ApprovalsRequired),
			Snapshot: sb, CampaignVersion: &version, ReadinessRunID: runID, PreviousID: previous}); err != nil {
			return err
		}
		if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: "approval", Now: now, ID: campaignID, FromStatuses: []string{"draft", "approved"}}); err != nil {
			return err
		}
		summary := fmt.Sprintf("%s recipients · %s", readiness.Count(snap.AudienceCount), strings.ReplaceAll(rulesLine(rules), " · ", ", "))
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.requested", TargetType: "campaign", TargetID: campaignID.String(), TargetName: c.Name,
			Actor: in.Actor,
			After: map[string]any{"status": "pending", "rules": ruleKeys(rules), "version": c.Version, "audience": snap.AudienceCount},
			Metadata: map[string]any{"summary": summary, "detail": readiness.Count(snap.AudienceCount) + " recipients", "request_id": id.String(),
				"requested_for": author.Name, "digest": in.Digest, "issue": in.Number}}); err != nil {
			return err
		}
		if err := emitRequested(ctx, tx, ws, id, "campaign", campaignID, c.Name, author.Name, ruleKeys(rules), snap.AudienceCount); err != nil {
			return err
		}
		body := fmt.Sprintf("%s's digest: %s · %s recipients · rules: %s.", author.Name, c.Name, readiness.Count(snap.AudienceCount), short)
		if in.Held {
			body += " It is held until the block is fixed."
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested", Body: body,
			Summary: fmt.Sprintf("%s's digest: %s · %s recipients", author.Name, c.Name, readiness.Count(snap.AudienceCount)),
			Link:    "/w/" + ws.Slug + "/approvals/" + id.String(), Data: map[string]any{"request_id": id.String(), "campaign_id": campaignID.String(), "digest": in.Digest}}
		n.TitleText = mNotifyRequestedTitle.Keep(nil)
		n.SummaryText = mNotifyDigestSummary.Keep(i18n.Args{"owner": author.Name, "name": c.Name, "count": snap.AudienceCount})
		held := "no"
		if in.Held {
			held = "yes"
		}
		for _, a := range approvers {
			n.BodyText = mNotifyDigestBody.Keep(i18n.Args{"owner": author.Name, "name": c.Name, "count": snap.AudienceCount,
				"rules": ruleList(readerLang(ctx, q, a.ID), rules), "held": held})
			if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

// IssueRulesFacts are the facts the approval rules read for a digest's
// issues before one is prepared (the setup screen says up front which
// rules every issue will match).
func IssueRulesFacts(recipients int64, shares []RuleShare, ownerCanSend bool, ownerName, ownerRole string, now time.Time) RuleFacts {
	return RuleFacts{Recipients: recipients, Shares: shares, AuthorCanSend: ownerCanSend, AuthorName: ownerName, AuthorRole: ownerRole, Now: now,
		Verdict: readiness.VerdictReady}
}
