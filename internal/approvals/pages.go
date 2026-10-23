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
	"github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Approvals for publishing landing and sales pages (design expansion phase
// 7, M29), as for posts (website.go):
//
//   - The workspace rule "Publishing a post or page needs approval" covers
//     pages, and "The author lacks the send permission" covers
//     website:publish: an author without it gets Request approval in place
//     of Publish.
//   - Approvers are the workspace's approvers (campaigns:approve), never the
//     author. An approval covers the page as it is (one hash of its content,
//     address, settings and publish time); a change after it needs a new
//     one. Once approved, the page goes live by itself, now or at its time.

// PagesPublisher publishes a page once its approval is granted
// (internal/pages).
type PagesPublisher interface {
	PublishApproved(ctx context.Context, ws *store.Workspace, pageID uuid.UUID, approvalID, approver, hash string) error
}

// ApprovalPageSnapshot is what a page publish request covers, as the
// approval review shows it.
type ApprovalPageSnapshot struct {
	PageID   string `json:"pageId"`
	Name     string `json:"name"`
	Template string `json:"template"`
	Slug     string `json:"slug"`
	Path     string `json:"path"`
	// Visibility is who can read the page; VisibilityLabel says it in words
	// ("Members", "Paid").
	Visibility      string `json:"visibility" enum:"public,members,paid"`
	VisibilityLabel string `json:"visibilityLabel"`
	// When the page goes live once approved: now (when approved) or at a
	// time.
	When      string     `json:"when" enum:"now,at"`
	PublishAt *time.Time `json:"publishAt,omitempty"`
	// Published is set once the approved page went live.
	Published bool `json:"published"`
}

// PagesGate implements pages.Approvals with the workspace's rules.
type PagesGate struct{ S *Service }

var (
	_ pages.Approvals     = PagesGate{}
	_ pages.ApprovalHooks = PagesGate{}
)

// pageRule is the rule that makes p ask for approval to publish a page (""
// when none applies).
func pageRule(policy ApprovalPolicy, p *authz.Principal) string {
	switch {
	case policy.PostPublish:
		return RulePostPublish
	case policy.AuthorLacksSend && !p.Can(authz.WebsitePublish):
		return RuleAuthorLacksSend
	}
	return ""
}

// pageVisibilityLabel says who can read a page: Members, Paid, Public.
func pageVisibilityLabel(l i18n.Lang, v string) string {
	return mVisibility.Text(l, i18n.Args{"visibility": v})
}

// PageDecision implements pages.Approvals.
func (g PagesGate) PageDecision(ctx context.Context, ws *store.Workspace, p *authz.Principal, req pages.ApprovalRequest) (pages.ApprovalDecision, error) {
	policy := ParsePolicy(ws.ApprovalPolicy)
	rule := pageRule(policy, p)
	if rule == "" {
		return pages.ApprovalDecision{Allowed: p.Can(authz.WebsitePublish)}, nil
	}
	approvers, err := WebsiteGate(g).approverNames(ctx, g.S.Q, ws.ID, p)
	if err != nil {
		return pages.ApprovalDecision{}, err
	}
	out := pages.ApprovalDecision{Rule: rule, Approvers: approvers}
	if id, err := g.S.Q.PageApprovalForHash(ctx, store.PageApprovalForHashParams{PageID: &req.PageID, DraftHash: req.Hash}); err == nil {
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
		out.Reason = say(ctx, mPublishPageRule, who)
	}
	if pend, err := g.S.Q.PageApprovalPending(ctx, &req.PageID); err == nil {
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
		out.Reason = say(ctx, mPublishPageNoApprovers, nil)
	}
	return out, nil
}

// RequestPage implements pages.Approvals: it asks the approvers to approve
// publishing the page as it is. A pending request for an older version is
// replaced.
func (g PagesGate) RequestPage(ctx context.Context, ws *store.Workspace, p *authz.Principal, req pages.ApprovalRequest, note string) (uuid.UUID, error) {
	s := g.S
	if !p.IsHuman() {
		return uuid.Nil, api.Err(403, api.CodePermissionDenied, say(ctx, mPagePeopleOnly, nil))
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	rule := pageRule(policy, p)
	if rule == "" {
		return uuid.Nil, api.Conflict(say(ctx, mPageNoNeed, nil))
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
		return uuid.Nil, api.Conflict(say(ctx, mPageNoApprovers, nil))
	}
	role := ""
	if m := memberByID(ms, &p.UserID); m != nil {
		role = m.RoleName
	}
	rm := postRuleMatch(rule, p, role)
	if rule == RulePostPublish {
		rm = match(RulePostPublish, mRulePost, mRulePageShort, mRulePageBadge, nil)
	}
	snap := Snapshot{DraftHash: req.Hash, Subject: req.Name, Rules: []ApprovalRuleMatch{rm}, AuthorRole: role,
		Page: &ApprovalPageSnapshot{PageID: req.PageID.String(), Name: req.Name, Template: req.Template, Slug: req.Slug, Path: req.Path,
			Visibility: req.Visibility, VisibilityLabel: pageVisibilityLabel(i18n.Default, req.Visibility), When: req.When, PublishAt: req.PublishAt}}
	note = strings.TrimSpace(note)
	id := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if err := q.PageApprovalCancelPending(ctx, store.PageApprovalCancelPendingParams{PageID: &req.PageID, Now: &now}); err != nil {
			return err
		}
		if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "website_page", WebsitePageID: &req.PageID,
			RequestedBy: &p.UserID, RequestedAt: now, Note: note, Status: "pending", MatchedRules: []string{rule},
			ApprovalsRequired: int32(policy.ApprovalsRequired), Snapshot: snapJSON(snap)}); err != nil {
			return err
		}
		if err := emitRequested(ctx, tx, ws, id, "website_page", req.PageID, req.Name, p.Name, []string{rule}, 0); err != nil {
			return err
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested",
			Summary: fmt.Sprintf("%s asked you to approve publishing “%s”", p.Name, req.Name),
			Body:    fmt.Sprintf("%s asked to publish the page “%s” at %s%s", p.Name, req.Name, req.Path, noteSuffix(note)),
			Link:    "/w/" + ws.Slug + "/approvals/" + id.String(), Data: map[string]any{"request_id": id.String(), "website_page_id": req.PageID.String()},
			TitleText:   mNotifyRequestedTitle.Keep(nil),
			SummaryText: mNotifyPublishSummary.Keep(i18n.Args{"author": p.Name, "name": req.Name}),
			BodyText:    mNotifyPageBody.Keep(i18n.Args{"author": p.Name, "name": req.Name, "path": req.Path, "hasNote": yesNo(note != ""), "note": note})}
		for _, a := range approvers {
			if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.publish_approval_requested", TargetType: "page", TargetID: req.PageID.String(),
			TargetName: req.Name, After: map[string]any{"status": "pending", "path": req.Path, "visibility": req.Visibility, "when": req.When, "note": note},
			Metadata: map[string]any{"request_id": id.String(), "approvers": orNames(names(approvers)), "rule": rule}})
	})
	if err != nil {
		if isUniqueViolation(err) {
			return uuid.Nil, api.Conflict(say(ctx, mPageRace, nil))
		}
		return uuid.Nil, err
	}
	return id, nil
}

// PagePublished implements pages.ApprovalHooks: the approved request a
// publish relied on remembers it went out.
func (g PagesGate) PagePublished(ctx context.Context, ws *store.Workspace, pageID uuid.UUID, hash string) error {
	return g.S.Q.PageApprovalMarkPublished(ctx, store.PageApprovalMarkPublishedParams{PageID: &pageID, DraftHash: hash})
}

// PageChanged implements pages.ApprovalHooks: requests for the page as it
// was are invalidated, and their authors told.
func (g PagesGate) PageChanged(ctx context.Context, ws *store.Workspace, p *authz.Principal, pageID uuid.UUID, name, hash string) error {
	s := g.S
	now := s.now()
	rows, err := s.Q.PageApprovalInvalidate(ctx, store.PageApprovalInvalidateParams{PageID: &pageID, Now: &now, DraftHash: hash,
		Reason: english(mInvalidPage, nil), ReasonText: textJSON(mInvalidPage.Keep(nil))})
	if err != nil || len(rows) == 0 {
		return err
	}
	for _, r := range rows {
		if r.RequestedBy == nil || (p != nil && *r.RequestedBy == p.UserID) {
			continue
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalInvalidated, Severity: "warn", Title: "Approval invalidated",
			Summary: fmt.Sprintf("“%s” changed after the request", name),
			Body:    fmt.Sprintf("The page “%s” changed after its approval request, so the request no longer covers it. Ask for approval again.", name),
			Link:    "/w/" + ws.Slug + "/approvals/" + r.ID.String(), Data: map[string]any{"request_id": r.ID.String(), "website_page_id": pageID.String()},
			TitleText:   mNotifyInvalidatedTitle.Keep(nil),
			SummaryText: mNotifyChangedSummary.Keep(i18n.Args{"name": name}),
			BodyText:    mNotifyPageChangedBody.Keep(i18n.Args{"name": name})}
		if err := notify.ToUser(ctx, s.Pool, *r.RequestedBy, n); err != nil {
			return err
		}
	}
	return nil
}

// pageDecisionAudit names a decision on a page publish request.
func pageDecisionAudit(pageID uuid.UUID, decision string) (targetType, targetID, action string) {
	action = map[string]string{DecisionApprove: "page.publish_approved", DecisionRequestChanges: "page.publish_changes_requested",
		DecisionReject: "page.publish_rejected"}[decision]
	return "page", pageID.String(), action
}

// pageDecisionNote points the author at the page in the builder.
func pageDecisionNote(ws *store.Workspace, pageID uuid.UUID, snap Snapshot, status, summary, body string) (link, newSummary, newBody string) {
	link = "/w/" + ws.Slug + "/forms/pages/" + pageID.String()
	if status == "approved" && snap.Page != nil {
		if snap.Page.When == "at" {
			summary += " · it goes live at its time"
		} else {
			summary += " · it's live now"
		}
		body += ". A change to the page needs a new approval."
	}
	return link, summary, body
}
