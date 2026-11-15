package website

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// ProposalRevision covers settings as well as content: changing a title or
// visibility does not increment the composer's content version. It is local
// concurrency metadata, not content sent to the model.
func ProposalRevision(post store.WebsitePost) string {
	raw, _ := json.Marshal(post)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

// AcceptDraftIn changes only editorial content on an exact, unpublished
// web-only revision. Publication settings, visibility and addresses stay as
// the person set them. Decision and history share the caller's transaction.
func (s *Service) AcceptDraftIn(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, target uuid.UUID, base int64, snapshot, title, excerpt, src string, origin audit.ProposalOrigin) (uuid.UUID, error) {
	if !p.IsHuman() || p.WorkspaceID != ws.ID || !p.Can(authz.WebsiteEdit) {
		return uuid.Nil, api.Forbidden(authz.WebsiteEdit, nil)
	}
	q := store.New(tx)
	cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: target})
	if err != nil {
		return uuid.Nil, api.MapError(err, "Post")
	}
	if cur.Kind != KindWeb || cur.Status != PostDraft || cur.FirstPublishedAt != nil || int64(cur.Version) != base || ProposalRevision(cur) != snapshot {
		return uuid.Nil, api.Err(409, "ai_draft_changed", tr(ctx, "post.contentConflict", nil))
	}
	if err := checkProposalContent(ctx, title, excerpt, src); err != nil {
		return uuid.Nil, err
	}
	now := s.now()
	if err := q.WebsitePostAcceptDraft(ctx, store.WebsitePostAcceptDraftParams{WorkspaceID: ws.ID, ID: target,
		Title: strings.TrimSpace(title), Excerpt: excerpt, Mjml: src, UpdatedBy: &p.UserID, At: now}); err != nil {
		return uuid.Nil, err
	}
	if err := q.WebsitePostVersionInsert(ctx, store.WebsitePostVersionInsertParams{PostID: target, Version: cur.Version + 1,
		Title: strings.TrimSpace(title), Mjml: src, SavedBy: &p.UserID, At: now, Label: origin.Label()}); err != nil {
		return uuid.Nil, err
	}
	meta := origin.Metadata()
	meta["before_version"], meta["version"] = base, cur.Version+1
	if err := s.Audit.LogWith(audit.WithPrincipal(ctx, p), q, audit.Event{WorkspaceID: &ws.ID, Action: "post.content.save", TargetType: "website_post", TargetID: target.String(),
		Metadata: meta}); err != nil {
		return uuid.Nil, err
	}
	cur.Title, cur.Excerpt, cur.Mjml = strings.TrimSpace(title), excerpt, src
	db.AfterCommit(tx, func() {
		if approvals := s.approvals(); approvals != nil {
			if err := approvals.PostChanged(ctx, ws, p, target, cur.Title, contentHash(cur, src)); err != nil {
				s.log().WarnContext(ctx, "invalidate post approvals", "post", target, "err", err)
			}
		}
	})
	return target, nil
}

func checkProposalContent(ctx context.Context, title, excerpt, src string) error {
	if strings.TrimSpace(title) == "" {
		return api.Invalid("body.title", tr(ctx, "post.titleRequired", nil))
	}
	if utf8.RuneCountInString(title) > 200 {
		return api.Invalid("body.title", tr(ctx, "post.titleLong", nil))
	}
	if utf8.RuneCountInString(excerpt) > 200 {
		return api.Invalid("body.excerpt", tr(ctx, "post.excerptLong", nil))
	}
	if _, _, issues := mjml.Checked(src, ""); issues != nil {
		return api.Invalid("body.mjml", tr(ctx, "post.invalidDraft", nil))
	}
	if _, err := render.Render(src, render.Options{Title: title}); err != nil {
		return api.Invalid("body.mjml", tr(ctx, "post.invalidDraft", nil))
	}
	return nil
}

// KeepDraftIn belongs to the human bundle decision. It creates a web-only
// draft and its first content version, never rendered public content,
// publication dates, schedules, approval requests or notification jobs.
func (s *Service) KeepDraftIn(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, title, excerpt, src string, origin audit.ProposalOrigin) (uuid.UUID, error) {
	if !p.IsHuman() || p.WorkspaceID != ws.ID || !p.Can(authz.WebsiteEdit) {
		return uuid.Nil, api.Forbidden(authz.WebsiteEdit, nil)
	}
	title = strings.TrimSpace(title)
	if err := checkProposalContent(ctx, title, excerpt, src); err != nil {
		return uuid.Nil, err
	}
	q := store.New(tx)
	visibility := "public"
	settings, err := q.WebsiteSettingsGet(ctx, ws.ID)
	if err == nil {
		visibility = settings.DefaultVisibility
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	id, now := newID(), s.now()
	// A unique suffix avoids a check-then-insert slug race between bundles.
	// No website settings need creating or changing to keep a post draft.
	slug := slugify(title)
	if slug == "" {
		slug = "post"
	}
	slug = strings.TrimRight(slug[:min(len(slug), 80)], "-") + "-" + id.String()
	if err := q.WebsitePostInsert(ctx, store.WebsitePostInsertParams{ID: id, WorkspaceID: ws.ID, Kind: KindWeb,
		WebOn: true, Status: PostDraft, PublishMode: WhenNow, Title: title, TitleEdited: true, Slug: slug,
		Excerpt: excerpt, ExcerptEdited: true, Visibility: visibility, Tiers: []uuid.UUID{}, BylineUserID: userID(p),
		Mjml: src, CreatedBy: userID(p), At: now}); err != nil {
		return uuid.Nil, err
	}
	if err := q.WebsitePostVersionInsert(ctx, store.WebsitePostVersionInsertParams{PostID: id, Version: 1, Title: title, Mjml: src,
		SavedBy: userID(p), At: now, Label: origin.Label()}); err != nil {
		return uuid.Nil, err
	}
	err = s.Audit.LogWith(audit.WithPrincipal(ctx, p), q, audit.Event{WorkspaceID: &ws.ID, Action: "post.created", TargetType: "website_post", TargetID: id.String(),
		Metadata: origin.Metadata()})
	return id, err
}
