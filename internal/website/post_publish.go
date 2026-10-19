package website

import (
	"context"
	"encoding/json"
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
	"github.com/hostrepute/reputecast/internal/webhooks"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// The Website webhook events.
func init() {
	webhooks.Events = append(webhooks.Events,
		webhooks.WebhookEventInfo{Name: "post.published", Group: "Website", Description: "A post went live on the website, by a person, at its time or with its campaign's send."},
		webhooks.WebhookEventInfo{Name: "post.unpublished", Group: "Website", Description: "A post was taken off the website or deleted."},
	)
}

// WebsitePublishInput publishes a post now or at a time.
type WebsitePublishInput struct {
	When string     `json:"when" enum:"now,at"`
	At   *time.Time `json:"at,omitempty"`
	TZ   string     `json:"tz,omitempty" maxLength:"64"`
}

// WebsiteUnpublishInput takes a post off the site.
type WebsiteUnpublishInput struct {
	Reason string `json:"reason,omitempty" maxLength:"300"`
	// RedirectTo is a path on the site the post's address sends readers
	// to ("/posts/…", "/archive"); empty answers 410.
	RedirectTo string `json:"redirectTo,omitempty" maxLength:"300"`
}

// WebsiteApprovalInput asks for approval to publish a post.
type WebsiteApprovalInput struct {
	Note string `json:"note,omitempty" maxLength:"1000"`
}

// errApproval is what a person who needs approval to publish gets.
func approvalProblem(ctx context.Context, d PostApprovalDecision) *api.Problem {
	return api.Err(http.StatusConflict, "approval_required", d.Reason).WithMeta("rule", d.Rule, "approvers", d.Approvers)
}

// publishState says whether p may publish a post as it is.
func (s *Service) publishState(ctx context.Context, ws *store.Workspace, p *authz.Principal, pr postRow, checks []WebsiteWebCheck) WebsitePublishState {
	out := WebsitePublishState{Approvers: []string{}}
	for _, c := range checks {
		if c.Status == "block" {
			out.Blocked = true
			out.BlockReason = c.Key
		}
	}
	d, err := s.decision(ctx, ws, p, pr.WebsitePost)
	if err != nil {
		s.log().WarnContext(ctx, "post approval decision", "err", err)
	}
	out.NeedsApproval, out.Rule, out.Approvers, out.ApprovedID = d.NeedsApproval, d.Rule, d.Approvers, d.Approved
	if out.Approvers == nil {
		out.Approvers = s.approverNames(ctx, ws)
	}
	if pend := s.openApprovals(ctx, ws, []uuid.UUID{pr.WebsitePost.ID})[pr.WebsitePost.ID]; pend != nil {
		out.Pending = pend
	}
	out.CanPublish = d.Allowed && !out.Blocked && p.Can(authz.WebsitePublish)
	return out
}

// decision asks the approval rules whether p may publish a post as it is.
func (s *Service) decision(ctx context.Context, ws *store.Workspace, p *authz.Principal, post store.WebsitePost) (PostApprovalDecision, error) {
	a := s.approvals()
	if a == nil {
		if p.Can(authz.WebsitePublish) {
			return PostApprovalDecision{Allowed: true}, nil
		}
		return PostApprovalDecision{}, nil
	}
	req, err := s.approvalRequest(ctx, ws, post)
	if err != nil {
		return PostApprovalDecision{}, err
	}
	return a.PostDecision(ctx, ws, p, req)
}

// approvalRequest is what an approval of a post covers.
func (s *Service) approvalRequest(ctx context.Context, ws *store.Workspace, post store.WebsitePost) (PostApprovalRequest, error) {
	src, err := s.sourceMJML(ctx, ws, post)
	if err != nil {
		return PostApprovalRequest{}, err
	}
	when := post.PublishMode
	if post.Kind == KindWeb && when == WhenWithSend {
		when = WhenNow
	}
	return PostApprovalRequest{PostID: post.ID, Title: post.Title, Kind: post.Kind, Slug: post.Slug, Path: "/posts/" + post.Slug,
		Visibility: post.Visibility, When: when, PublishAt: post.PublishAt, Hash: contentHash(post, src), Excerpt: post.Excerpt}, nil
}

// Publish publishes a post now, or schedules it.
func (s *Service) Publish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsitePublishInput) (WebsitePost, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.load(ctx, q, ws)
		if err != nil {
			return err
		}
		if !c.On() {
			return api.Conflict(tr(ctx, "site.off", nil))
		}
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Post")
		}
		if err != nil {
			return err
		}
		if cur.Status == PostPublished {
			return api.Conflict(tr(ctx, "post.alreadyPublished", nil))
		}
		if cur.Kind == KindCampaign && in.When == WhenNow {
			// A campaign's post goes live now only once the campaign went out.
			var started *time.Time
			_ = tx.QueryRow(ctx, `SELECT started_at FROM campaigns WHERE id = $1`, cur.CampaignID).Scan(&started)
			if started == nil {
				return api.Conflict(tr(ctx, "post.notSentYet", nil))
			}
		}
		if taken, err := q.WebsitePostSlugTaken(ctx, store.WebsitePostSlugTakenParams{WorkspaceID: ws.ID, Slug: cur.Slug, ID: cur.ID}); err != nil {
			return err
		} else if taken {
			return api.Err(http.StatusConflict, "slug_taken", tr(ctx, "post.slugTaken", map[string]any{"path": "/posts/" + cur.Slug}))
		}
		if in.When == WhenAt {
			if in.At == nil || !in.At.After(s.now()) {
				return api.Invalid("body.at", tr(ctx, "post.timePast", nil))
			}
			at := in.At.UTC()
			cur.PublishMode, cur.PublishAt = WhenAt, &at
			cur.PublishTz = strings.TrimSpace(in.TZ)
			if cur.PublishTz == "" {
				cur.PublishTz = ws.Timezone
			}
		}
		d, err := s.decision(ctx, ws, p, cur)
		if err != nil {
			return err
		}
		if d.NeedsApproval || !d.Allowed {
			if d.NeedsApproval {
				return approvalProblem(ctx, d)
			}
			return api.Forbidden(authz.WebsitePublish, s.helpers(ctx, ws, authz.WebsitePublish))
		}
		if in.When == WhenAt {
			if err := q.WebsitePostSchedule(ctx, store.WebsitePostScheduleParams{WorkspaceID: ws.ID, ID: id, PublishMode: WhenAt, PublishAt: cur.PublishAt,
				PublishTz: cur.PublishTz, UpdatedBy: userID(p), At: s.now()}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.scheduled", TargetType: "website_post", TargetID: id.String(),
				TargetName: cur.Title, Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": PostScheduled, "publish_at": cur.PublishAt,
					"visibility": cur.Visibility, "slug": cur.Slug}})
		}
		return s.publishNow(ctx, tx, ws, cur, userID(p), nil, d.Approved)
	})
	if err != nil {
		return WebsitePost{}, err
	}
	return s.Post(ctx, ws, p, id)
}

// publishNow puts a post on the site: the content it goes out with is
// drawn once, its address and the old archive address lead to it.
// by is the person (nil: the system, at its time or with the send).
func (s *Service) publishNow(ctx context.Context, tx pgx.Tx, ws *store.Workspace, cur store.WebsitePost, by *uuid.UUID, system map[string]any, approvalID string) error {
	q := store.New(tx)
	now := s.now()
	at := now
	if system != nil {
		if t, ok := system["at"].(time.Time); ok && !t.IsZero() {
			at = t
		}
	}
	if err := s.snapshot(ctx, q, ws, cur); err != nil {
		return err
	}
	if err := q.WebsitePostPublish(ctx, store.WebsitePostPublishParams{WorkspaceID: ws.ID, ID: cur.ID, PublishedBy: by, At: at}); err != nil {
		return err
	}
	if cur.CampaignID != nil {
		// The archive's address of the issue leads to the post.
		if ai, err := q.ArchiveIssueGet(ctx, store.ArchiveIssueGetParams{WorkspaceID: ws.ID, CampaignID: *cur.CampaignID}); err == nil {
			if err := q.WebsiteRedirectInsert(ctx, store.WebsiteRedirectInsertParams{WorkspaceID: ws.ID, Path: "archive/" + ai.Slug, PostID: cur.ID, At: now}); err != nil {
				return err
			}
		}
	}
	ev := audit.Event{WorkspaceID: &ws.ID, Action: "post.published", TargetType: "website_post", TargetID: cur.ID.String(), TargetName: cur.Title,
		Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": PostPublished, "slug": cur.Slug, "visibility": cur.Visibility, "tiers": cur.Tiers}}
	if approvalID != "" {
		ev.Metadata = map[string]any{"approval": approvalID}
	}
	if system != nil {
		name, _ := system["name"].(string)
		ev.Actor = &audit.Actor{Type: authz.ActorSystem, Name: name}
		meta := map[string]any{}
		for k, v := range system {
			if k != "at" && k != "name" {
				meta[k] = v
			}
		}
		ev.Metadata = meta
	}
	if err := s.Audit.LogWith(ctx, q, ev); err != nil {
		return err
	}
	cur.Status = PostPublished
	cur.PublishedAt = &at
	if err := s.emitPost(ctx, tx, ws, cur, "post.published", ""); err != nil {
		return err
	}
	if a := s.approvals(); a != nil && approvalID != "" {
		src, err := s.sourceMJML(ctx, ws, cur)
		if err == nil {
			if err := a.PostPublished(ctx, ws, cur.ID, contentHash(cur, src)); err != nil {
				s.log().WarnContext(ctx, "settle post approval", "err", err)
			}
		}
	}
	return nil
}

// snapshot draws the web version of the content a post goes out with.
func (s *Service) snapshot(ctx context.Context, q *store.Queries, ws *store.Workspace, cur store.WebsitePost) error {
	src, err := s.sourceMJML(ctx, ws, cur)
	if err != nil {
		return err
	}
	res, err := render.Render(src, s.renderOptions(ctx, ws, cur))
	if err != nil {
		// An MJML that doesn't parse leaves the post with its excerpt.
		s.log().WarnContext(ctx, "render post", "post", cur.ID, "err", err)
		res = render.Result{Hidden: []render.Hidden{}, Fallbacks: []render.Fallback{}, Paid: []render.Paid{}}
	}
	notes, _ := json.Marshal(renderNotes{Hidden: res.Hidden, Fallbacks: res.Fallbacks})
	paid, _ := json.Marshal(res.Paid)
	return q.WebsitePostSetRendered(ctx, store.WebsitePostSetRenderedParams{ID: cur.ID, PublishedMjml: src, PublishedVersion: cur.Version,
		BodyHtml: res.Body, PreviewHtml: res.Preview, HasMarker: res.HasMarker, MarkerAfter: int32(res.MarkerAfter), BodyText: res.Text,
		PreviewText: res.PreviewText, Words: int32(res.Words), RenderNotes: notes, PaidBlocks: paid, RenderedHash: contentHash(cur, src)})
}

// Unpublish takes a post off the site: the home page, tags, the archive,
// feeds and the sitemap. Its address answers 410, or redirects.
func (s *Service) Unpublish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsiteUnpublishInput) (WebsitePost, error) {
	redirect := strings.TrimSpace(in.RedirectTo)
	if redirect != "" && (!strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") || strings.ContainsAny(redirect, " \\")) {
		return WebsitePost{}, api.Invalid("body.redirectTo", tr(ctx, "post.redirectInvalid", nil))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Post")
		}
		if err != nil {
			return err
		}
		if cur.Status != PostPublished && cur.Status != PostScheduled {
			return api.Conflict(tr(ctx, "post.notPublished", nil))
		}
		if redirect == "/posts/"+cur.Slug {
			return api.Invalid("body.redirectTo", tr(ctx, "post.redirectSelf", nil))
		}
		if cur.Status == PostScheduled {
			// A scheduled post that never went out goes back to draft.
			if err := q.WebsitePostToDraft(ctx, store.WebsitePostToDraftParams{WorkspaceID: ws.ID, ID: id, UpdatedBy: userID(p), At: s.now()}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.unscheduled", TargetType: "website_post", TargetID: id.String(),
				TargetName: cur.Title, Before: map[string]any{"status": PostScheduled, "publish_at": cur.PublishAt}, After: map[string]any{"status": PostDraft}})
		}
		reason := strings.TrimSpace(in.Reason)
		if err := q.WebsitePostUnpublish(ctx, store.WebsitePostUnpublishParams{WorkspaceID: ws.ID, ID: id, UnpublishedBy: userID(p), Reason: reason,
			RedirectTo: redirect, At: s.now()}); err != nil {
			return err
		}
		if err := s.setFeatured(ctx, q, ws, p, id, false); err != nil {
			return err
		}
		if err := s.emitPost(ctx, tx, ws, cur, "post.unpublished", "unpublished"); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.unpublished", TargetType: "website_post", TargetID: id.String(),
			TargetName: cur.Title, Before: map[string]any{"status": PostPublished}, After: map[string]any{"status": PostUnpublished, "reason": reason,
				"redirect_to": redirect}})
	})
	if err != nil {
		return WebsitePost{}, err
	}
	return s.Post(ctx, ws, p, id)
}

// UndoPublish takes back a publish within 10 seconds ("Undo" on the
// toast): the post goes back to where it was.
func (s *Service) UndoPublish(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WebsitePost, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Post")
		}
		if err != nil {
			return err
		}
		if cur.Status != PostPublished || cur.PublishedAt == nil || s.now().Sub(*cur.PublishedAt) > 15*time.Second {
			return api.Conflict(tr(ctx, "post.undoLate", nil))
		}
		if _, err := tx.Exec(ctx, `UPDATE website_posts SET status = 'draft', published_at = NULL, published_by = NULL,
			first_published_at = CASE WHEN first_published_at = published_at THEN NULL ELSE first_published_at END, updated_at = $2
			WHERE id = $1`, id, s.now()); err != nil {
			return err
		}
		if err := s.emitPost(ctx, tx, ws, cur, "post.unpublished", "undone"); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.publish_undone", TargetType: "website_post", TargetID: id.String(),
			TargetName: cur.Title, Before: map[string]any{"status": PostPublished}, After: map[string]any{"status": PostDraft}})
	})
	if err != nil {
		return WebsitePost{}, err
	}
	return s.Post(ctx, ws, p, id)
}

// SetRedirect changes where an unpublished post's address sends readers.
func (s *Service) SetRedirect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsiteUnpublishInput) (WebsitePost, error) {
	redirect := strings.TrimSpace(in.RedirectTo)
	if redirect != "" && (!strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") || strings.ContainsAny(redirect, " \\")) {
		return WebsitePost{}, api.Invalid("body.redirectTo", tr(ctx, "post.redirectInvalid", nil))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Post")
		}
		if err != nil {
			return err
		}
		if cur.Status != PostUnpublished {
			return api.Conflict(tr(ctx, "post.notUnpublished", nil))
		}
		if err := q.WebsitePostSetRedirect(ctx, store.WebsitePostSetRedirectParams{WorkspaceID: ws.ID, ID: id, RedirectTo: redirect, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.redirect_changed", TargetType: "website_post", TargetID: id.String(),
			TargetName: cur.Title, Before: map[string]any{"redirect_to": cur.RedirectTo}, After: map[string]any{"redirect_to": redirect}})
	})
	if err != nil {
		return WebsitePost{}, err
	}
	return s.Post(ctx, ws, p, id)
}

// RequestApproval asks the workspace's approvers to approve publishing a
// post as it is.
func (s *Service) RequestApproval(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsiteApprovalInput) (WebsitePost, error) {
	a := s.approvals()
	if a == nil {
		return WebsitePost{}, api.Conflict(tr(ctx, "post.noApprovals", nil))
	}
	pr, err := s.getPost(ctx, s.Q, ws, id)
	if err != nil {
		return WebsitePost{}, err
	}
	req, err := s.approvalRequest(ctx, ws, pr.WebsitePost)
	if err != nil {
		return WebsitePost{}, err
	}
	if _, err := a.RequestPost(ctx, ws, p, req, strings.TrimSpace(in.Note)); err != nil {
		return WebsitePost{}, err
	}
	return s.Post(ctx, ws, p, id)
}

// PublishApproved goes ahead with a post once its approval came: now, or
// at its time. The approver's decision is the authority.
func (s *Service) PublishApproved(ctx context.Context, ws *store.Workspace, postID uuid.UUID, approvalID string, approverName string) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: postID})
		if err != nil {
			return err
		}
		if cur.Status == PostPublished {
			return nil
		}
		if cur.Kind == KindCampaign && cur.PublishMode == WhenWithSend {
			// It goes live with its campaign's send.
			return nil
		}
		if cur.PublishMode == WhenAt && cur.PublishAt != nil && cur.PublishAt.After(s.now()) {
			return q.WebsitePostSchedule(ctx, store.WebsitePostScheduleParams{WorkspaceID: ws.ID, ID: postID, PublishMode: WhenAt, PublishAt: cur.PublishAt,
				PublishTz: cur.PublishTz, At: s.now()})
		}
		return s.publishNow(ctx, tx, ws, cur, nil, map[string]any{"name": "ReputeCast", "approved_by": approverName, "approval": approvalID}, approvalID)
	})
}

// PublishDue publishes the posts whose time came: scheduled at a time, or
// set to go live with their campaign's send once it started. The minute
// job runs it.
func (s *Service) PublishDue(ctx context.Context) (int, error) {
	due, err := s.Q.WebsitePostsDue(ctx, s.now())
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
			cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: d.ID})
			if err != nil {
				return err
			}
			if cur.Status == PostPublished || cur.Status == PostUnpublished || !cur.WebOn {
				return nil
			}
			system := map[string]any{"name": "ReputeCast"}
			if cur.PublishMode == WhenWithSend && cur.CampaignID != nil {
				var started *time.Time
				var name string
				_ = tx.QueryRow(ctx, `SELECT started_at, name FROM campaigns WHERE id = $1`, *cur.CampaignID).Scan(&started, &name)
				if started != nil {
					system["at"] = *started
				}
				system["source"], system["campaign"] = "with the send", name
			} else {
				system["source"] = "scheduled"
			}
			return s.publishNow(ctx, tx, &ws, cur, cur.UpdatedBy, system, "")
		})
		if err != nil {
			s.log().WarnContext(ctx, "publish due post", "post", d.ID, "err", err)
			continue
		}
		n++
	}
	return n, nil
}

// emitPost queues a Website webhook event of a post.
func (s *Service) emitPost(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p store.WebsitePost, event, why string) error {
	data := map[string]any{"post_id": p.ID.String(), "title": p.Title, "slug": p.Slug, "url": s.postURL(ws, p.Slug), "kind": p.Kind,
		"visibility": p.Visibility}
	if p.CampaignID != nil {
		data["campaign_id"] = p.CampaignID.String()
	}
	if p.PublishedAt != nil {
		data["published_at"] = p.PublishedAt.UTC()
	}
	if why != "" {
		data["reason"] = why
	}
	_, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, event, data)
	return err
}
