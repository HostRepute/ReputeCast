package approvals

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// Approvals for publishing posts of the publication website (design
// expansion phase 7, publish-sell brief §4 Approvals):
//
//   - The optional workspace rule "Publishing a post or page needs
//     approval", off by default.
//   - The rule "The author lacks the send permission" also covers
//     website:publish: an author without it gets Request approval in place
//     of Publish.
//   - Approvers are the workspace's approvers (campaigns:approve), never
//     the author. The approval covers the post as it is: its content,
//     address, visibility and publish time (one hash). A change after it
//     needs a new one; once approved the post goes live by itself, now or at
//     its time ("publish when approved").
//   - A campaign's post that goes live with the send needs none of its
//     own: the campaign's approval covers its web settings.

// WebsitePosts is the part of the website the approvals need: publishing a
// post once it is approved, and its web preview for the review.
type WebsitePosts interface {
	PublishApproved(ctx context.Context, ws *store.Workspace, postID uuid.UUID, approvalID string, approverName string) error
}

// ApprovalWebsiteSnapshot is what a post publish request covers, as the
// approval review shows it.
type ApprovalWebsiteSnapshot struct {
	PostID     string `json:"postId"`
	Title      string `json:"title"`
	Kind       string `json:"kind" enum:"campaign,web"`
	Slug       string `json:"slug"`
	Path       string `json:"path"`
	Visibility string `json:"visibility" enum:"public,members,paid"`
	// VisibilityLabel says it in words ("Members", "Paid · all tiers").
	VisibilityLabel string `json:"visibilityLabel"`
	// When the post goes live once approved: now (when approved), at a
	// time, or with the campaign's send.
	When      string     `json:"when" enum:"now,at,with_send"`
	PublishAt *time.Time `json:"publishAt,omitempty"`
	Excerpt   string     `json:"excerpt"`
	// Published is set once the approved post went live.
	Published bool `json:"published"`
}

// WebsiteGate implements website.Approvals with the workspace's rules.
type WebsiteGate struct{ S *Service }

var _ website.Approvals = WebsiteGate{}

// postRule is the rule that makes p ask for approval to publish a post
// ("" when none applies).
func postRule(policy ApprovalPolicy, p *authz.Principal, req website.PostApprovalRequest) string {
	if req.Kind == website.KindCampaign && req.When == website.WhenWithSend {
		// The campaign's approval covers its web settings.
		return ""
	}
	switch {
	case policy.PostPublish:
		return RulePostPublish
	case policy.AuthorLacksSend && !p.Can(authz.WebsitePublish):
		return RuleAuthorLacksSend
	}
	return ""
}

// postRuleMatch is the matched rule as the inbox shows it.
func postRuleMatch(rule string, p *authz.Principal, role string) ApprovalRuleMatch {
	if rule == RuleAuthorLacksSend {
		return ApprovalRuleMatch{Key: RuleAuthorLacksSend, Label: "The author lacks the send permission (website:publish)", Short: "author lacks publish",
			Badge: "Author lacks publish", Author: p.Name, Role: role}
	}
	return ApprovalRuleMatch{Key: RulePostPublish, Label: "Publishing a post or page needs approval", Short: "post publish", Badge: "Post publish"}
}

// approverNames are the approvers who may decide on p's post.
func (g WebsiteGate) approverNames(ctx context.Context, q *store.Queries, wsID uuid.UUID, p *authz.Principal) ([]string, error) {
	ms, err := g.S.members(ctx, q, wsID)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, a := range approversOf(ms) {
		if p == nil || a.ID != p.UserID {
			out = append(out, a.Name)
		}
	}
	return out, nil
}

// PostDecision implements website.Approvals.
func (g WebsiteGate) PostDecision(ctx context.Context, ws *store.Workspace, p *authz.Principal, req website.PostApprovalRequest) (website.PostApprovalDecision, error) {
	policy := ParsePolicy(ws.ApprovalPolicy)
	rule := postRule(policy, p, req)
	if rule == "" {
		ok := p.Can(authz.WebsitePublish) || (req.Kind == website.KindCampaign && req.When == website.WhenWithSend)
		return website.PostApprovalDecision{Allowed: ok}, nil
	}
	approvers, err := g.approverNames(ctx, g.S.Q, ws.ID, p)
	if err != nil {
		return website.PostApprovalDecision{}, err
	}
	out := website.PostApprovalDecision{Rule: rule, Approvers: approvers}
	if id, err := g.S.Q.PostApprovalForHash(ctx, store.PostApprovalForHashParams{PostID: &req.PostID, DraftHash: req.Hash}); err == nil {
		// Approved as it is: it goes ahead (the approver's decision is the
		// authority, so the author doesn't need website:publish).
		out.Allowed, out.Approved = true, id.String()
		return out, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out.NeedsApproval = true
	who := i18n.Args{"approvers": i18n.OrList(langOf(ctx).Locale, approvers)}
	switch rule {
	case RuleAuthorLacksSend:
		out.Reason = say(ctx, mPublishLacksPublish, who)
	default:
		out.Reason = say(ctx, mPublishPostRule, who)
	}
	if pend, err := g.S.Q.PostApprovalPending(ctx, &req.PostID); err == nil {
		out.Pending, out.RequestedBy = pend.ID.String(), pend.RequestedByName
		at := pend.RequestedAt
		out.RequestedAt = &at
		if pend.DraftHash == req.Hash {
			out.Reason = say(ctx, mPublishWaiting, who)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if len(approvers) == 0 {
		out.Reason = say(ctx, mPublishPostNoApprovers, nil)
	}
	return out, nil
}

// RequestPost implements website.Approvals: it asks the approvers to
// approve publishing the post as it is. A pending request for an older
// version is replaced.
func (g WebsiteGate) RequestPost(ctx context.Context, ws *store.Workspace, p *authz.Principal, req website.PostApprovalRequest, note string) (uuid.UUID, error) {
	s := g.S
	if !p.IsHuman() {
		return uuid.Nil, api.Err(403, api.CodePermissionDenied, say(ctx, mPostPeopleOnly, nil))
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	rule := postRule(policy, p, req)
	if rule == "" {
		return uuid.Nil, api.Conflict(say(ctx, mPostNoNeed, nil))
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	approvers := []member{}
	for _, a := range approversOf(ms) {
		if a.ID != p.UserID {
			approvers = append(approvers, a)
		}
	}
	if len(approvers) == 0 {
		return uuid.Nil, api.Conflict(say(ctx, mPostNoApprovers, nil))
	}
	role := ""
	if m := memberByID(ms, &p.UserID); m != nil {
		role = m.RoleName
	}
	match := postRuleMatch(rule, p, role)
	snap := Snapshot{DraftHash: req.Hash, Subject: req.Title, Preheader: req.Excerpt, Rules: []ApprovalRuleMatch{match}, AuthorRole: role,
		Website: &ApprovalWebsiteSnapshot{PostID: req.PostID.String(), Title: req.Title, Kind: req.Kind, Slug: req.Slug, Path: req.Path, Visibility: req.Visibility,
			VisibilityLabel: req.VisibilityLabel, When: req.When, PublishAt: req.PublishAt, Excerpt: req.Excerpt}}
	note = strings.TrimSpace(note)
	id := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if err := q.PostApprovalCancelPending(ctx, store.PostApprovalCancelPendingParams{PostID: &req.PostID, Now: &now}); err != nil {
			return err
		}
		if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "website_post", WebsitePostID: &req.PostID,
			RequestedBy: &p.UserID, RequestedAt: now, Note: note, Status: "pending", MatchedRules: []string{rule},
			ApprovalsRequired: int32(policy.ApprovalsRequired), Snapshot: snapJSON(snap)}); err != nil {
			return err
		}
		if err := emitRequested(ctx, tx, ws, id, "website_post", req.PostID, req.Title, p.Name, []string{rule}, 0); err != nil {
			return err
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested",
			Summary: fmt.Sprintf("%s asked you to approve publishing “%s”", p.Name, req.Title),
			Body:    fmt.Sprintf("%s asked to publish the post “%s” at %s%s", p.Name, req.Title, req.Path, noteSuffix(note)),
			Link:    "/w/" + ws.Slug + "/approvals/" + id.String(), Data: map[string]any{"request_id": id.String(), "website_post_id": req.PostID.String()}}
		for _, a := range approvers {
			if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.publish_approval_requested", TargetType: "website_post", TargetID: req.PostID.String(),
			TargetName: req.Title, After: map[string]any{"status": "pending", "path": req.Path, "visibility": req.Visibility, "when": req.When, "note": note},
			Metadata: map[string]any{"request_id": id.String(), "approvers": orNames(names(approvers)), "rule": rule}})
	})
	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, api.Conflict(say(ctx, mPostRace, nil))
		}
		return uuid.Nil, err
	}
	return id, nil
}

// PostPublished implements website.Approvals: the approved request a
// publish relied on remembers it went out.
func (g WebsiteGate) PostPublished(ctx context.Context, ws *store.Workspace, postID uuid.UUID, hash string) error {
	return g.S.Q.PostApprovalMarkPublished(ctx, store.PostApprovalMarkPublishedParams{PostID: &postID, DraftHash: hash})
}

// PostChanged implements website.Approvals: requests for the post as it
// was are invalidated, and their authors told.
func (g WebsiteGate) PostChanged(ctx context.Context, ws *store.Workspace, p *authz.Principal, postID uuid.UUID, title, hash string) error {
	s := g.S
	now := s.now()
	rows, err := s.Q.PostApprovalInvalidate(ctx, store.PostApprovalInvalidateParams{PostID: &postID, Now: &now, DraftHash: hash,
		Reason: "The post changed after the request: its content, address, visibility or publish time."})
	if err != nil || len(rows) == 0 {
		return err
	}
	for _, r := range rows {
		if r.RequestedBy == nil || (p != nil && *r.RequestedBy == p.UserID) {
			continue
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalInvalidated, Severity: "warn", Title: "Approval invalidated",
			Summary: fmt.Sprintf("“%s” changed after the request", title),
			Body:    fmt.Sprintf("The post “%s” changed after its approval request, so the request no longer covers it. Ask for approval again.", title),
			Link:    "/w/" + ws.Slug + "/approvals/" + r.ID.String(), Data: map[string]any{"request_id": r.ID.String(), "website_post_id": postID.String()}}
		if err := notify.ToUser(ctx, s.Pool, *r.RequestedBy, n); err != nil {
			return err
		}
	}
	return nil
}

// websiteDecisionAudit names a decision on a post publish request.
func websiteDecisionAudit(postID uuid.UUID, decision string) (targetType, targetID, action string) {
	action = map[string]string{DecisionApprove: "post.publish_approved", DecisionRequestChanges: "post.publish_changes_requested",
		DecisionReject: "post.publish_rejected"}[decision]
	return "website_post", postID.String(), action
}

// websiteDecisionNote points the author at the post.
func websiteDecisionNote(ws *store.Workspace, postID uuid.UUID, snap Snapshot, status, summary, body string) (link, newSummary, newBody string) {
	link = "/w/" + ws.Slug + "/forms/website/posts/" + postID.String()
	if status == "approved" && snap.Website != nil {
		switch snap.Website.When {
		case website.WhenAt:
			summary += " · it goes live at its time"
		default:
			summary += " · it's live now"
		}
		body += ". A change to the post needs a new approval."
	}
	return link, summary, body
}
