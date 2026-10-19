package pages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Publishing: Publish (or Publish changes) puts the draft live at its
// address now, or at a time; Unpublish takes it off (its address answers
// 410, or sends readers elsewhere); Undo takes a publish back within ten
// seconds. A changed address keeps the old one as a redirect. With "Show
// in navigation" the header links to the page while it's live.

// PagePublishInput publishes a page now or at a time.
type PagePublishInput struct {
	When string     `json:"when" enum:"now,at"`
	At   *time.Time `json:"at,omitempty"`
	TZ   string     `json:"tz,omitempty" maxLength:"64"`
}

// PageUnpublishInput takes a page off the site.
type PageUnpublishInput struct {
	Reason string `json:"reason,omitempty" maxLength:"300"`
	// RedirectTo is a path on the site readers are sent to; empty answers 410.
	RedirectTo string `json:"redirectTo,omitempty" maxLength:"300"`
}

// PageApprovalInput asks for approval to publish.
type PageApprovalInput struct {
	Note string `json:"note,omitempty" maxLength:"1000"`
}

// ApprovalRequest is what an approval to publish a page covers: the page
// as it is (one hash of its content, address, settings and time).
type ApprovalRequest struct {
	PageID     uuid.UUID
	Name       string
	Slug       string
	Path       string
	Visibility string
	Template   string
	When       string
	PublishAt  *time.Time
	Hash       string
}

// ApprovalDecision is what the approval rules say about publishing.
type ApprovalDecision struct {
	Allowed       bool
	NeedsApproval bool
	Rule          string
	Reason        string
	Approvers     []string
	// Approved is the approval that covers the page as it is.
	Approved    string
	Pending     string
	RequestedBy string
	RequestedAt *time.Time
}

// Approvals applies the workspace's approval rules to pages
// (internal/approvals implements it).
type Approvals interface {
	PageDecision(ctx context.Context, ws *store.Workspace, p *authz.Principal, req ApprovalRequest) (ApprovalDecision, error)
	RequestPage(ctx context.Context, ws *store.Workspace, p *authz.Principal, req ApprovalRequest, note string) (uuid.UUID, error)
}

// approvalRequest is the approval a page needs as it is.
func approvalRequest(pg store.Page, when string, at *time.Time) ApprovalRequest {
	h := sha256.New()
	for _, part := range []string{pg.Name, pg.Slug, pg.Html, string(settingsOf(pg).json()), when} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	if at != nil {
		h.Write([]byte(at.UTC().Format(time.RFC3339)))
	}
	return ApprovalRequest{PageID: pg.ID, Name: pg.Name, Slug: pg.Slug, Path: "/" + pg.Slug, Visibility: pg.Visibility, Template: pg.Template,
		When: when, PublishAt: at, Hash: hex.EncodeToString(h.Sum(nil))[:32]}
}

// decision asks the rules whether p may publish the page now.
func (s *Service) decision(ctx context.Context, ws *store.Workspace, p *authz.Principal, pg store.Page, when string, at *time.Time) (ApprovalDecision, error) {
	if s.Approvals == nil {
		return ApprovalDecision{Allowed: p.Can(authz.WebsitePublish)}, nil
	}
	return s.Approvals.PageDecision(ctx, ws, p, approvalRequest(pg, when, at))
}

// publishState says what p may do with a page.
func (s *Service) publishState(ctx context.Context, ws *store.Workspace, p *authz.Principal, pg store.Page, checks []PageCheck) PagePublishState {
	out := PagePublishState{CanEdit: p.Can(authz.WebsiteEdit), CanManage: p.Can(authz.WebsiteManage), Approvers: []string{}, Publishers: []string{}}
	blocked := false
	for _, c := range checks {
		if c.Status == CheckBlock {
			blocked = true
		}
	}
	d, err := s.decision(ctx, ws, p, pg, "now", nil)
	if err != nil {
		s.log().WarnContext(ctx, "page approval decision", "err", err)
	}
	out.NeedsApproval, out.Rule, out.Reason = d.NeedsApproval, d.Rule, d.Reason
	if d.Approvers != nil {
		out.Approvers = d.Approvers
	}
	if d.Pending != "" && d.RequestedAt != nil {
		out.Pending = &PageApproval{ID: d.Pending, RequestedBy: d.RequestedBy, RequestedAt: *d.RequestedAt}
	}
	out.CanPublish = d.Allowed && !blocked
	if !p.Can(authz.WebsitePublish) {
		out.Publishers = s.helpers(ctx, ws, authz.WebsitePublish, 3, false)
	}
	return out
}

// Publish publishes a page's draft now, or schedules it.
func (s *Service) Publish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PagePublishInput) (PageDetail, error) {
	if in.When == "" {
		in.When = "now"
	}
	if in.When == "at" {
		if in.At == nil || !in.At.After(s.now()) {
			return PageDetail{}, api.Invalid("body.at", word(ctx, "page.atPast", nil))
		}
		if in.TZ == "" {
			in.TZ = ws.Timezone
		}
		if _, err := time.LoadLocation(in.TZ); err != nil {
			return PageDetail{}, api.Invalid("body.tz", word(ctx, "page.tzInvalid", nil))
		}
	} else {
		in.At, in.TZ = nil, ""
	}
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return PageDetail{}, err
	}
	checks, _, err := s.checks(ctx, ws, pg)
	if err != nil {
		return PageDetail{}, err
	}
	var failing []string
	for _, c := range checks {
		if c.Status == CheckBlock {
			failing = append(failing, c.Key)
		}
	}
	if len(failing) > 0 {
		return PageDetail{}, api.Err(http.StatusConflict, "checks_failed", word(ctx, "page.checksFailed", map[string]any{"count": len(failing)})).WithMeta("checks", failing)
	}
	d, err := s.decision(ctx, ws, p, pg, in.When, in.At)
	if err != nil {
		return PageDetail{}, err
	}
	if !d.Allowed {
		if d.NeedsApproval {
			return PageDetail{}, api.Err(http.StatusConflict, "approval_required", d.Reason).WithMeta("rule", d.Rule, "approvers", d.Approvers)
		}
		return PageDetail{}, api.Forbidden(authz.WebsitePublish, s.helpers(ctx, ws, authz.WebsitePublish, 3, false))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if cur.Version != pg.Version {
			// Saved meanwhile: the checks ran on an older version.
			return api.Conflict(word(ctx, "page.changedMeanwhile", nil))
		}
		if in.When == "at" {
			if err := q.PageSchedule(ctx, store.PageScheduleParams{ID: cur.ID, Status: StatusScheduled, PublishAt: in.At, PublishTz: in.TZ, At: s.now(), Actor: actor(p)}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.scheduled", TargetType: "page", TargetID: cur.ID.String(), TargetName: cur.Name,
				Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": StatusScheduled, "publish_at": in.At, "tz": in.TZ, "version": cur.Version},
				Metadata: approvalMeta(d)})
		}
		return s.publishNow(ctx, tx, ws, cur, actor(p), nil, d.Approved)
	})
	if err != nil {
		return PageDetail{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

func approvalMeta(d ApprovalDecision) map[string]any {
	if d.Approved == "" {
		return nil
	}
	return map[string]any{"approval": d.Approved}
}

// publishNow puts a page's draft live, in tx.
func (s *Service) publishNow(ctx context.Context, tx pgx.Tx, ws *store.Workspace, cur store.Page, by *uuid.UUID, system *audit.Actor, approval string) error {
	q := store.New(tx)
	now := s.now()
	first := cur.FirstPublishedAt == nil
	if err := q.PagePublish(ctx, store.PagePublishParams{ID: cur.ID, LiveVersion: &cur.Version, LiveSlug: &cur.Slug, At: now, Actor: by}); err != nil {
		return err
	}
	if err := q.PageVersionPublished(ctx, store.PageVersionPublishedParams{PageID: cur.ID, Version: cur.Version, At: now, Actor: by}); err != nil {
		return err
	}
	// The old address keeps working as a redirect; the new one is the
	// page's own.
	from := ""
	if cur.LiveSlug != nil && *cur.LiveSlug != cur.Slug {
		from = *cur.LiveSlug
		if err := q.PageRedirectInsert(ctx, store.PageRedirectInsertParams{WorkspaceID: ws.ID, Slug: from, PageID: cur.ID, At: now}); err != nil {
			return err
		}
	}
	if err := q.PageRedirectDelete(ctx, store.PageRedirectDeleteParams{WorkspaceID: ws.ID, Slug: cur.Slug}); err != nil {
		return err
	}
	if s.Website != nil {
		label := cur.NavLabel
		if _, err := s.Website.SetPageLink(ctx, q, ws, by, from, cur.Slug, label, cur.ShowInNav); err != nil {
			return err
		}
	}
	action := "page.published"
	if !first {
		action = "page.changes_published"
	}
	before := map[string]any{"status": cur.Status}
	if cur.LiveVersion != nil {
		before["version"] = *cur.LiveVersion
	}
	meta := map[string]any{}
	if approval != "" {
		meta["approval"] = approval
	}
	if from != "" {
		meta["redirect_from"] = "/" + from
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Actor: system, Action: action, TargetType: "page", TargetID: cur.ID.String(), TargetName: cur.Name,
		Before: before, After: map[string]any{"status": StatusPublished, "version": cur.Version, "path": "/" + cur.Slug, "visibility": cur.Visibility}, Metadata: meta})
}

// Unpublish takes a page off the site.
func (s *Service) Unpublish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PageUnpublishInput) (PageDetail, error) {
	if !p.Can(authz.WebsitePublish) {
		return PageDetail{}, api.Forbidden(authz.WebsitePublish, s.helpers(ctx, ws, authz.WebsitePublish, 3, false))
	}
	in.RedirectTo = strings.TrimSpace(in.RedirectTo)
	if in.RedirectTo != "" && (!strings.HasPrefix(in.RedirectTo, "/") || strings.HasPrefix(in.RedirectTo, "//")) {
		return PageDetail{}, api.Invalid("body.redirectTo", word(ctx, "page.redirectInvalid", nil))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		switch cur.Status {
		case StatusScheduled:
			// A schedule taken back: the page stays as it was.
			status := StatusDraft
			if cur.LiveVersion != nil {
				status = StatusPublished
			}
			if err := q.PageSchedule(ctx, store.PageScheduleParams{ID: cur.ID, Status: status, At: s.now(), Actor: actor(p)}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.unscheduled", TargetType: "page", TargetID: cur.ID.String(), TargetName: cur.Name,
				Before: map[string]any{"status": StatusScheduled, "publish_at": cur.PublishAt}, After: map[string]any{"status": status}})
		case StatusPublished:
		default:
			return api.Conflict(word(ctx, "page.notPublished", nil))
		}
		if err := q.PageUnpublish(ctx, store.PageUnpublishParams{ID: cur.ID, At: s.now(), Actor: actor(p), Reason: strings.TrimSpace(in.Reason), RedirectTo: in.RedirectTo}); err != nil {
			return err
		}
		if s.Website != nil && cur.LiveSlug != nil {
			if _, err := s.Website.SetPageLink(ctx, q, ws, actor(p), "", *cur.LiveSlug, "", false); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.unpublished", TargetType: "page", TargetID: cur.ID.String(), TargetName: cur.Name,
			Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": StatusUnpublished, "redirect_to": in.RedirectTo, "reason": in.Reason}})
	})
	if err != nil {
		return PageDetail{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

// Undo takes a publish back within ten seconds ("Undo" on the toast): the
// version live before comes back, or the page goes back to its draft.
func (s *Service) Undo(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PageDetail, error) {
	if !p.Can(authz.WebsitePublish) {
		return PageDetail{}, api.Forbidden(authz.WebsitePublish, s.helpers(ctx, ws, authz.WebsitePublish, 3, false))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if cur.Status != StatusPublished || cur.PublishedAt == nil || s.now().Sub(*cur.PublishedAt) > 15*time.Second || cur.LiveVersion == nil {
			return api.Conflict(word(ctx, "page.undoLate", nil))
		}
		prev, err := q.PagePreviousPublished(ctx, store.PagePreviousPublishedParams{PageID: cur.ID, Version: *cur.LiveVersion})
		params := store.PageRestoreLiveParams{ID: cur.ID, Status: StatusDraft, At: s.now()}
		if err == nil {
			params.Status, params.LiveVersion, params.LiveSlug = StatusPublished, &prev.Version, &prev.Slug
			params.PublishedAt, params.PublishedBy, params.FirstPublishedAt = prev.PublishedAt, prev.PublishedBy, cur.FirstPublishedAt
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := q.PageRestoreLive(ctx, params); err != nil {
			return err
		}
		if s.Website != nil {
			on := params.Status == StatusPublished && parseSettings(prev.Settings).ShowInNav
			slug, label := cur.Slug, ""
			if params.LiveSlug != nil {
				slug, label = *params.LiveSlug, parseSettings(prev.Settings).NavLabel
			}
			if _, err := s.Website.SetPageLink(ctx, q, ws, actor(p), cur.Slug, slug, label, on); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.publish_undone", TargetType: "page", TargetID: cur.ID.String(), TargetName: cur.Name,
			Before: map[string]any{"status": StatusPublished, "version": *cur.LiveVersion}, After: map[string]any{"status": params.Status, "version": params.LiveVersion}})
	})
	if err != nil {
		return PageDetail{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

// RequestApproval asks the workspace's approvers to approve publishing the
// page as it is.
func (s *Service) RequestApproval(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PageApprovalInput) (PageDetail, error) {
	if s.Approvals == nil {
		return PageDetail{}, api.Conflict(word(ctx, "page.noApproval", nil))
	}
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return PageDetail{}, err
	}
	when, at := "now", (*time.Time)(nil)
	if pg.Status == StatusScheduled && pg.PublishAt != nil {
		when, at = "at", pg.PublishAt
	}
	if _, err := s.Approvals.RequestPage(ctx, ws, p, approvalRequest(pg, when, at), in.Note); err != nil {
		return PageDetail{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

// PublishApproved publishes a page once its approval came (the approvals
// domain calls it), unless it changed since.
func (s *Service) PublishApproved(ctx context.Context, ws *store.Workspace, id uuid.UUID, approvalID, approver string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lockPage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if cur.Status == StatusScheduled && cur.PublishAt != nil && cur.PublishAt.After(s.now()) {
			// It goes live at its time (PublishDue).
			return nil
		}
		return s.publishNow(ctx, tx, ws, cur, nil, &audit.Actor{Type: authz.ActorSystem, Name: "Approved by " + approver}, approvalID)
	})
}

// PublishDue publishes the scheduled pages whose time came.
func (s *Service) PublishDue(ctx context.Context) (int, error) {
	now := s.now()
	due, err := s.Q.PagesDue(ctx, &now)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range due {
		ws, err := s.Q.GetWorkspace(ctx, d.WorkspaceID)
		if err != nil {
			continue
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			cur, err := s.lockPage(ctx, q, &ws, d.ID)
			if err != nil {
				return err
			}
			if cur.Status != StatusScheduled || cur.PublishAt == nil || cur.PublishAt.After(s.now()) {
				return nil
			}
			n++
			return s.publishNow(ctx, tx, &ws, cur, nil, &audit.Actor{Type: authz.ActorSystem, Name: "Schedule"}, "")
		})
		if err != nil {
			s.log().ErrorContext(ctx, "scheduled page", "page", d.ID, "err", err)
		}
	}
	return n, nil
}
