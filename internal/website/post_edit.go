package website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// WebsitePostInput saves a post's settings: the Web tab of the composer.
type WebsitePostInput struct {
	Title   string `json:"title" maxLength:"200"`
	Slug    string `json:"slug" maxLength:"120"`
	Excerpt string `json:"excerpt" maxLength:"400"`
	// UseSubject and UsePreheader go back to the campaign's subject and
	// preheader (posts from campaigns).
	UseSubject   bool   `json:"useSubject,omitempty"`
	UsePreheader bool   `json:"usePreheader,omitempty"`
	CoverURL     string `json:"coverUrl" maxLength:"1000"`
	CoverAlt     string `json:"coverAlt" maxLength:"300"`
	// The byline: a member, or a written name.
	BylineUserID string `json:"bylineUserId,omitempty" maxLength:"64"`
	BylineName   string `json:"bylineName,omitempty" maxLength:"120"`
	// Tags are tag ids, in order (up to 5).
	Tags       []string       `json:"tags" maxItems:"5"`
	Visibility string         `json:"visibility" enum:"public,members,paid"`
	Tiers      []string       `json:"tiers" maxItems:"10" doc:"A paid post's tiers; empty for every paid tier"`
	NeverFree  bool           `json:"neverFree"`
	Featured   bool           `json:"featured"`
	SEO        WebsitePostSEO `json:"seo"`
	// WebOn is a campaign's "Publish to <domain>" switch.
	WebOn bool `json:"webOn"`
	// When a post goes live: with the campaign's send, at a time, or now
	// (a sent campaign; web-only posts publish with Publish).
	PublishMode string     `json:"publishMode" enum:"with_send,at,now"`
	PublishAt   *time.Time `json:"publishAt,omitempty"`
	PublishTZ   string     `json:"publishTz,omitempty" maxLength:"64"`
}

// WebsitePostCreate starts a web-only post.
type WebsitePostCreate struct {
	Title string `json:"title,omitempty" maxLength:"200"`
	// MJML is the post's starting content (empty: the composer's gallery).
	MJML string `json:"mjml,omitempty" maxLength:"2000000"`
}

// WebsitePostContent is an autosave or a save of a web-only post's content.
type WebsitePostContent struct {
	MJML string `json:"mjml" maxLength:"2000000"`
	// BaseVersion is the version the edit started from: another save since
	// then is a conflict (409) unless Force.
	BaseVersion int    `json:"baseVersion" minimum:"0"`
	Force       bool   `json:"force,omitempty"`
	Source      string `json:"source,omitempty" enum:"autosave,manual,import,template,restore,"`
	Label       string `json:"label,omitempty" maxLength:"80"`
}

// WebsitePostSaved is a saved version.
type WebsitePostSaved struct {
	Version int       `json:"version"`
	SavedAt time.Time `json:"savedAt"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// slugify makes an address from a title: lowercase letters, digits and
// hyphens.
func slugify(s string) string {
	out := forms.Slugify(s, 120)
	out = strings.Join(strings.FieldsFunc(out, func(r rune) bool { return r == '-' }), "-")
	if len(out) > 120 {
		out = strings.Trim(out[:120], "-")
	}
	return out
}

// freeSlug is a slug no other post holds or held, from base.
func (s *Service) freeSlug(ctx context.Context, ws *store.Workspace, id uuid.UUID, base string) string {
	base = slugify(base)
	if base == "" {
		base = "post"
	}
	slug := base
	for i := 2; i < 1000; i++ {
		taken, err := s.Q.WebsitePostSlugTaken(ctx, store.WebsitePostSlugTakenParams{WorkspaceID: ws.ID, Slug: slug, ID: id})
		if err != nil || !taken {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return slug
}

// CreatePost starts a web-only post in the composer's post mode, with the
// workspace's default visibility, its writer as the byline.
func (s *Service) CreatePost(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsitePostCreate) (WebsitePost, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsitePost{}, err
	}
	id := newID()
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = tr(ctx, "post.untitled", nil)
	}
	now := s.now()
	slug := s.freeSlug(ctx, ws, id, title)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WebsitePostInsert(ctx, store.WebsitePostInsertParams{ID: id, WorkspaceID: ws.ID, Kind: KindWeb, WebOn: true, Status: PostDraft,
			PublishMode: WhenNow, Title: title, Slug: slug, Visibility: c.row.DefaultVisibility, Tiers: []uuid.UUID{}, BylineUserID: userID(p),
			Mjml: in.MJML, CreatedBy: userID(p), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.created", TargetType: "website_post", TargetID: id.String(),
			TargetName: title, After: map[string]any{"kind": KindWeb, "slug": slug, "visibility": c.row.DefaultVisibility}})
	})
	if err != nil {
		return WebsitePost{}, err
	}
	return s.Post(ctx, ws, p, id)
}

// UpdatePost saves a post's settings. A changed address keeps the old one
// as a redirect once the post was published.
func (s *Service) UpdatePost(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsitePostInput) (WebsitePost, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Post")
		}
		if err != nil {
			return err
		}
		return s.applySettings(ctx, q, ws, p, cur, in)
	})
	if err != nil {
		return WebsitePost{}, err
	}
	s.changed(ctx, ws, p, id)
	return s.Post(ctx, ws, p, id)
}

// changed tells the approvals a post changed: requests for it as it was
// no longer cover it.
func (s *Service) changed(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) {
	a := s.approvals()
	if a == nil {
		return
	}
	pr, err := s.getPost(ctx, s.Q, ws, id)
	if err != nil {
		return
	}
	cur := pr.WebsitePost
	src, err := s.sourceMJML(ctx, ws, cur)
	if err != nil {
		return
	}
	if err := a.PostChanged(ctx, ws, p, id, cur.Title, contentHash(cur, src)); err != nil {
		s.log().WarnContext(ctx, "invalidate post approvals", "post", id, "err", err)
	}
}

// applySettings validates and writes a post's settings, its tags and
// whether it's featured, with their audit entries.
func (s *Service) applySettings(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, cur store.WebsitePost, in WebsitePostInput) error {
	next := cur
	title := strings.TrimSpace(in.Title)
	if utf8.RuneCountInString(title) > 200 {
		return api.Invalid("body.title", tr(ctx, "post.titleLong", nil))
	}
	next.Title, next.TitleEdited = title, true
	if cur.Kind == KindCampaign && (in.UseSubject || title == "") {
		next.TitleEdited = false
		next.Title = s.campaignTitle(ctx, q, cur)
	}
	if next.Title == "" {
		return api.Invalid("body.title", tr(ctx, "post.titleRequired", nil))
	}
	excerpt := strings.Join(strings.Fields(in.Excerpt), " ")
	if utf8.RuneCountInString(excerpt) > 200 {
		return api.Invalid("body.excerpt", tr(ctx, "post.excerptLong", nil))
	}
	next.Excerpt, next.ExcerptEdited = excerpt, true
	if cur.Kind == KindCampaign && (in.UsePreheader || excerpt == "") {
		next.ExcerptEdited = false
		next.Excerpt = s.campaignExcerpt(ctx, q, cur)
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugify(next.Title)
	}
	if !slugRe.MatchString(slug) || len(slug) > 120 {
		return api.Invalid("body.slug", tr(ctx, "post.slugInvalid", nil))
	}
	if slug != cur.Slug {
		taken, err := q.WebsitePostSlugTaken(ctx, store.WebsitePostSlugTakenParams{WorkspaceID: ws.ID, Slug: slug, ID: cur.ID})
		if err != nil {
			return err
		}
		if taken {
			return api.Err(http.StatusConflict, "slug_taken", tr(ctx, "post.slugTaken", map[string]any{"path": "/posts/" + slug})).
				WithMeta("suggestion", s.freeSlug(ctx, ws, cur.ID, slug+"-"+slugTail(next.Title, slug)))
		}
	}
	next.Slug = slug
	next.CoverUrl, next.CoverAlt = strings.TrimSpace(in.CoverURL), strings.TrimSpace(in.CoverAlt)
	if next.CoverUrl != "" && !strings.HasPrefix(next.CoverUrl, "https://") && !strings.HasPrefix(next.CoverUrl, "http://") && !strings.HasPrefix(next.CoverUrl, "/media/") {
		return api.Invalid("body.coverUrl", tr(ctx, "post.coverInvalid", nil))
	}
	next.BylineUserID, next.BylineName = nil, strings.TrimSpace(in.BylineName)
	if in.BylineUserID != "" {
		uid, err := uuid.Parse(in.BylineUserID)
		if err != nil {
			return api.Invalid("body.bylineUserId", tr(ctx, "post.bylineUnknown", nil))
		}
		members, err := q.ApprovalMembers(ctx, ws.ID)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(members, func(m store.ApprovalMembersRow) bool { return m.ID == uid }) {
			return api.Invalid("body.bylineUserId", tr(ctx, "post.bylineUnknown", nil))
		}
		next.BylineUserID, next.BylineName = &uid, ""
	}
	if !gate.ValidLevel(in.Visibility) {
		return api.Invalid("body.visibility", tr(ctx, "post.visibilityInvalid", nil))
	}
	next.Visibility = in.Visibility
	next.Tiers = []uuid.UUID{}
	if in.Visibility == gate.Paid {
		tiers, err := s.tiers(ctx, q, ws)
		if err != nil {
			return err
		}
		for _, raw := range in.Tiers {
			tid, err := uuid.Parse(raw)
			if err != nil || !slices.ContainsFunc(tiers, func(t gate.Tier) bool { return t.ID == tid }) {
				return api.Invalid("body.tiers", tr(ctx, "post.tierUnknown", nil))
			}
			if !slices.Contains(next.Tiers, tid) {
				next.Tiers = append(next.Tiers, tid)
			}
		}
	}
	next.NeverFree = in.NeverFree && in.Visibility != gate.Public
	next.SeoTitle, next.SeoDescription = strings.TrimSpace(in.SEO.Title), strings.TrimSpace(in.SEO.Description)
	next.SocialImage, next.CanonicalUrl, next.HideFromSearch = strings.TrimSpace(in.SEO.SocialImage), strings.TrimSpace(in.SEO.CanonicalURL), in.SEO.HideFromSearch
	if next.CanonicalUrl != "" && !strings.HasPrefix(next.CanonicalUrl, "https://") && !strings.HasPrefix(next.CanonicalUrl, "http://") {
		return api.Invalid("body.seo.canonicalUrl", tr(ctx, "post.canonicalInvalid", nil))
	}
	if cur.Kind == KindCampaign {
		next.WebOn = in.WebOn
		switch in.PublishMode {
		case WhenWithSend, WhenAt, WhenNow:
			next.PublishMode = in.PublishMode
		default:
			next.PublishMode = WhenWithSend
		}
	} else if in.PublishMode == WhenAt || in.PublishMode == WhenNow {
		next.PublishMode = in.PublishMode
	}
	next.PublishAt, next.PublishTz = nil, strings.TrimSpace(in.PublishTZ)
	if next.PublishMode == WhenAt {
		if in.PublishAt == nil {
			return api.Invalid("body.publishAt", tr(ctx, "post.timeRequired", nil))
		}
		at := in.PublishAt.UTC()
		next.PublishAt = &at
	}
	if next.PublishTz == "" {
		next.PublishTz = ws.Timezone
	}
	now := s.now()
	if err := q.WebsitePostSaveFields(ctx, store.WebsitePostSaveFieldsParams{WorkspaceID: ws.ID, ID: cur.ID, WebOn: next.WebOn, PublishMode: next.PublishMode,
		PublishAt: next.PublishAt, PublishTz: next.PublishTz, Title: next.Title, TitleEdited: next.TitleEdited, Slug: next.Slug, Excerpt: next.Excerpt,
		ExcerptEdited: next.ExcerptEdited, CoverUrl: next.CoverUrl, CoverAlt: next.CoverAlt, BylineUserID: next.BylineUserID, BylineName: next.BylineName,
		Visibility: next.Visibility, Tiers: next.Tiers, NeverFree: next.NeverFree, SeoTitle: next.SeoTitle, SeoDescription: next.SeoDescription,
		SocialImage: next.SocialImage, CanonicalUrl: next.CanonicalUrl, HideFromSearch: next.HideFromSearch, UpdatedBy: userID(p), At: now}); err != nil {
		return err
	}
	// A scheduled post that moved or lost its time goes back to draft.
	if cur.Status == PostScheduled && (next.PublishMode != WhenAt || next.PublishAt == nil) {
		if err := q.WebsitePostToDraft(ctx, store.WebsitePostToDraftParams{WorkspaceID: ws.ID, ID: cur.ID, UpdatedBy: userID(p), At: now}); err != nil {
			return err
		}
	}
	tagIDs, err := s.setTags(ctx, q, ws, cur.ID, in.Tags)
	if err != nil {
		return err
	}
	if err := s.setFeatured(ctx, q, ws, p, cur.ID, in.Featured); err != nil {
		return err
	}
	// The old address redirects once the post went out.
	if next.Slug != cur.Slug && cur.FirstPublishedAt != nil {
		if err := q.WebsiteRedirectInsert(ctx, store.WebsiteRedirectInsertParams{WorkspaceID: ws.ID, Path: "posts/" + cur.Slug, PostID: cur.ID, At: now}); err != nil {
			return err
		}
	}
	return s.auditSettings(ctx, q, ws, cur, next, tagIDs)
}

// auditSettings writes a post's settings changes: its address (with the
// redirect) and its visibility apart, the rest together.
func (s *Service) auditSettings(ctx context.Context, q *store.Queries, ws *store.Workspace, cur, next store.WebsitePost, tags []uuid.UUID) error {
	log := func(action string, before, after map[string]any, meta map[string]any) error {
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "website_post", TargetID: cur.ID.String(),
			TargetName: next.Title, Before: before, After: after, Metadata: meta})
	}
	if next.Slug != cur.Slug {
		meta := map[string]any{}
		if cur.FirstPublishedAt != nil {
			meta["redirect"] = "/posts/" + cur.Slug + " → /posts/" + next.Slug
		}
		if err := log("post.slug_changed", map[string]any{"slug": cur.Slug}, map[string]any{"slug": next.Slug}, meta); err != nil {
			return err
		}
	}
	if next.Visibility != cur.Visibility || !slices.Equal(next.Tiers, cur.Tiers) || next.NeverFree != cur.NeverFree {
		if err := log("post.visibility_changed", map[string]any{"visibility": cur.Visibility, "tiers": cur.Tiers, "never_free": cur.NeverFree},
			map[string]any{"visibility": next.Visibility, "tiers": next.Tiers, "never_free": next.NeverFree}, nil); err != nil {
			return err
		}
	}
	before, after := map[string]any{}, map[string]any{}
	diff := func(k string, a, b any) {
		if fmt.Sprint(a) != fmt.Sprint(b) {
			before[k], after[k] = a, b
		}
	}
	diff("title", cur.Title, next.Title)
	diff("excerpt", cur.Excerpt, next.Excerpt)
	diff("cover", cur.CoverUrl, next.CoverUrl)
	diff("cover_alt", cur.CoverAlt, next.CoverAlt)
	diff("byline", fmt.Sprint(cur.BylineUserID, cur.BylineName), fmt.Sprint(next.BylineUserID, next.BylineName))
	diff("web_on", cur.WebOn, next.WebOn)
	diff("publish_mode", cur.PublishMode, next.PublishMode)
	diff("publish_at", cur.PublishAt, next.PublishAt)
	diff("seo", []any{cur.SeoTitle, cur.SeoDescription, cur.SocialImage, cur.CanonicalUrl, cur.HideFromSearch},
		[]any{next.SeoTitle, next.SeoDescription, next.SocialImage, next.CanonicalUrl, next.HideFromSearch})
	if len(after) == 0 {
		return nil
	}
	return log("post.updated", before, after, map[string]any{"tags": len(tags)})
}

// setTags gives a post its tags in order (up to 5).
func (s *Service) setTags(ctx context.Context, q *store.Queries, ws *store.Workspace, post uuid.UUID, ids []string) ([]uuid.UUID, error) {
	if len(ids) > MaxTags {
		return nil, api.Invalid("body.tags", tr(ctx, "post.tooManyTags", nil))
	}
	out := []uuid.UUID{}
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, api.Invalid("body.tags", tr(ctx, "tag.notFound", nil))
		}
		if _, err := q.WebsiteTagGet(ctx, store.WebsiteTagGetParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return nil, api.Invalid("body.tags", tr(ctx, "tag.notFound", nil))
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	if err := q.WebsitePostClearTags(ctx, post); err != nil {
		return nil, err
	}
	for i, id := range out {
		if err := q.WebsitePostAddTag(ctx, store.WebsitePostAddTagParams{PostID: post, TagID: id, Position: int16(i)}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// setFeatured features a post on the home page (up to 3) or stops.
func (s *Service) setFeatured(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, post uuid.UUID, on bool) error {
	c, err := s.lock(ctx, q, ws)
	if err != nil {
		return err
	}
	has := slices.Contains(c.row.Featured, post)
	if on == has {
		return nil
	}
	if on {
		if len(c.row.Featured) >= MaxFeatured {
			return api.Err(http.StatusConflict, "featured_full", tr(ctx, "post.featuredFull", nil))
		}
		c.row.Featured = append(c.row.Featured, post)
	} else {
		c.row.Featured = slices.DeleteFunc(c.row.Featured, func(id uuid.UUID) bool { return id == post })
	}
	return s.save(ctx, q, ws, c, userID(p))
}

// SavePostContent saves a web-only post's content (autosave, Save, a
// template or a restored version).
func (s *Service) SavePostContent(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsitePostContent) (WebsitePostSaved, error) {
	var out WebsitePostSaved
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Post")
		}
		if err != nil {
			return err
		}
		if cur.Kind != KindWeb {
			return api.Conflict(tr(ctx, "post.contentIsCampaign", nil))
		}
		if !in.Force && in.BaseVersion != 0 && in.BaseVersion != int(cur.Version) {
			return api.Err(http.StatusConflict, "conflict", tr(ctx, "post.contentConflict", nil)).WithMeta("version", cur.Version)
		}
		now := s.now()
		next := cur.Version + 1
		var autosaved *time.Time
		if in.Source == "autosave" || in.Source == "" {
			autosaved = &now
		}
		if err := q.WebsitePostSaveContent(ctx, store.WebsitePostSaveContentParams{WorkspaceID: ws.ID, ID: id, Mjml: in.MJML, Version: next,
			AutosavedAt: autosaved, UpdatedBy: userID(p), At: now}); err != nil {
			return err
		}
		// Every manual save and every tenth autosave keeps a version.
		if in.Source != "autosave" || next%10 == 0 {
			if err := q.WebsitePostVersionInsert(ctx, store.WebsitePostVersionInsertParams{PostID: id, Version: next, Title: cur.Title, Mjml: in.MJML,
				SavedBy: userID(p), At: now, Label: strings.TrimSpace(in.Label)}); err != nil {
				return err
			}
		}
		out = WebsitePostSaved{Version: int(next), SavedAt: now}
		return nil
	})
	if err == nil {
		s.changed(ctx, ws, p, id)
	}
	return out, err
}

// DeletePost deletes a post: it leaves the site (its address answers 410).
func (s *Service) DeletePost(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Post")
		}
		if err != nil {
			return err
		}
		if cur.FirstPublishedAt != nil && !p.Can(authz.WebsitePublish) {
			return api.Forbidden(authz.WebsitePublish, s.helpers(ctx, ws, authz.WebsitePublish))
		}
		if err := q.WebsitePostDelete(ctx, store.WebsitePostDeleteParams{WorkspaceID: ws.ID, ID: id, UpdatedBy: userID(p), At: s.now()}); err != nil {
			return err
		}
		if err := s.setFeatured(ctx, q, ws, p, id, false); err != nil {
			return err
		}
		if cur.Status == PostPublished {
			if err := s.emitPost(ctx, tx, ws, cur, "post.unpublished", "deleted"); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "post.deleted", TargetType: "website_post", TargetID: id.String(),
			TargetName: cur.Title, Before: map[string]any{"status": cur.Status, "slug": cur.Slug}})
	})
}

// helpers are the members who hold a permission ("Ask Sofia Rossi or…").
func (s *Service) helpers(ctx context.Context, ws *store.Workspace, perm authz.Permission) []string {
	rows, err := s.Q.ApprovalMembers(ctx, ws.ID)
	if err != nil {
		return []string{}
	}
	out := []string{}
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			out = append(out, r.Name)
		}
	}
	return out
}

// campaignTitle is the title a campaign's post gets: its subject with the
// personalization's fallbacks, else its name.
func (s *Service) campaignTitle(ctx context.Context, q *store.Queries, p store.WebsitePost) string {
	if p.CampaignID == nil {
		return p.Title
	}
	var subject, name string
	if err := s.Pool.QueryRow(ctx, `SELECT subject, name FROM campaigns WHERE id = $1`, *p.CampaignID).Scan(&subject, &name); err != nil {
		return p.Title
	}
	if t := forms.RenderTitle(subject); t != "" {
		return t
	}
	return name
}

// campaignExcerpt is a campaign's preheader as an excerpt (up to 200 characters).
func (s *Service) campaignExcerpt(ctx context.Context, q *store.Queries, p store.WebsitePost) string {
	if p.CampaignID == nil {
		return p.Excerpt
	}
	var pre, text string
	if err := s.Pool.QueryRow(ctx, `SELECT preheader, plain_text FROM campaigns WHERE id = $1`, *p.CampaignID).Scan(&pre, &text); err != nil {
		return p.Excerpt
	}
	ex := forms.RenderTitle(pre)
	if ex == "" {
		ex = forms.Excerpt(text, 180)
	}
	if utf8.RuneCountInString(ex) > 200 {
		ex = forms.Excerpt(ex, 195)
	}
	return ex
}
