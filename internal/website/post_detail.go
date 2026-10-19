package website

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// WebsitePostSEO are a post's search and social settings.
type WebsitePostSEO struct {
	Title          string `json:"title" maxLength:"120"`
	Description    string `json:"description" maxLength:"300"`
	SocialImage    string `json:"socialImage" maxLength:"500"`
	CanonicalURL   string `json:"canonicalUrl" maxLength:"500"`
	HideFromSearch bool   `json:"hideFromSearch"`
}

// WebsiteHiddenBlock is a block left out of a post's web version.
type WebsiteHiddenBlock struct {
	// ID is the block's Visibility id (the composer selects it).
	ID string `json:"id,omitempty"`
	// Name is the block's own name; Kind a well-known block (view_online,
	// footer, referral, header) or block.
	Name      string `json:"name,omitempty"`
	Kind      string `json:"kind" enum:"view_online,footer,referral,header,block"`
	Reason    string `json:"reason" enum:"email_only,conditional,hidden"`
	Condition string `json:"condition,omitempty"`
	Line      int    `json:"line"`
}

// WebsiteFallback is a personalization token the web shows with its fallback.
type WebsiteFallback struct {
	Token    string `json:"token"`
	Fallback string `json:"fallback"`
	// Source is the token as written with the word before it ("Hi {{
	// first_name | default: "there" }}"), Sample how the web reads it.
	Source string `json:"source"`
	Sample string `json:"sample"`
}

// WebsiteWebCheck is one of a post's web checks.
type WebsiteWebCheck struct {
	Key    string `json:"key" enum:"address,cover,tokens,hidden"`
	Status string `json:"status" enum:"pass,warn,info,block"`
	// Address: the path; Taken: who holds it; Suggestion: a free slug.
	Path       string               `json:"path,omitempty"`
	Taken      *WebsiteSlugOwner    `json:"taken,omitempty"`
	Suggestion string               `json:"suggestion,omitempty"`
	Fallbacks  []WebsiteFallback    `json:"fallbacks"`
	Hidden     []WebsiteHiddenBlock `json:"hidden"`
}

// WebsiteSlugOwner is the post that holds, or once held, an address.
type WebsiteSlugOwner struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Status string    `json:"status"`
	Since  time.Time `json:"since"`
}

// WebsitePublishState says whether the person may publish the post as it is.
type WebsitePublishState struct {
	// CanPublish: holds website:publish and nothing blocks (or the post
	// is approved as it is).
	CanPublish bool `json:"canPublish"`
	// NeedsApproval: Publish becomes Request approval, for Rule.
	NeedsApproval bool     `json:"needsApproval"`
	Rule          string   `json:"rule,omitempty" enum:"author_lacks_send,post_publish,"`
	Approvers     []string `json:"approvers"`
	// Pending is an open request for this post.
	Pending     *WebsitePostApproval `json:"pending,omitempty"`
	ApprovedID  string               `json:"approvedId,omitempty"`
	Blocked     bool                 `json:"blocked"`
	BlockReason string               `json:"blockReason,omitempty"`
}

// WebsitePost is one post: its settings, its content (web-only posts),
// its web checks and whether the person may publish it.
type WebsitePost struct {
	WebsitePostRow
	CoverURL      string         `json:"coverUrl"`
	CoverAlt      string         `json:"coverAlt"`
	TitleEdited   bool           `json:"titleEdited"`
	ExcerptEdited bool           `json:"excerptEdited"`
	WebOn         bool           `json:"webOn"`
	SEO           WebsitePostSEO `json:"seo"`
	// MJML is a web-only post's content; Version its saved version and
	// PublishedVersion the one live on the site (lower: unpublished changes).
	MJML             string     `json:"mjml"`
	Version          int        `json:"version"`
	PublishedVersion int        `json:"publishedVersion"`
	AutosavedAt      *time.Time `json:"autosavedAt,omitempty"`
	// Base is the address before the slug ("news.acme.io/posts/").
	Base           string              `json:"base"`
	Words          int                 `json:"words"`
	ReadingMinutes int                 `json:"readingMinutes"`
	Checks         []WebsiteWebCheck   `json:"checks"`
	Publish        WebsitePublishState `json:"publish"`
	// Bylines are the workspace's members a byline can name.
	Bylines []WebsiteByline `json:"bylines"`
}

// Post returns one post.
func (s *Service) Post(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WebsitePost, error) {
	pr, err := s.getPost(ctx, s.Q, ws, id)
	if err != nil {
		return WebsitePost{}, err
	}
	return s.detail(ctx, ws, p, pr)
}

func (s *Service) detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, pr postRow) (WebsitePost, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsitePost{}, err
	}
	tiers, err := s.tiers(ctx, s.Q, ws)
	if err != nil {
		return WebsitePost{}, err
	}
	now := s.now()
	until := day(now, location(ws))
	rows, err := s.rows(ctx, ws, []uuid.UUID{pr.WebsitePost.ID}, until.AddDate(0, 0, -29), until)
	if err != nil {
		return WebsitePost{}, err
	}
	row := s.row(ws, c, tiers, pr)
	if len(rows) == 1 {
		row = rows[0]
	}
	post := pr.WebsitePost
	out := WebsitePost{WebsitePostRow: row, CoverURL: post.CoverUrl, CoverAlt: post.CoverAlt, TitleEdited: post.TitleEdited,
		ExcerptEdited: post.ExcerptEdited, WebOn: post.WebOn, MJML: post.Mjml, Version: int(post.Version), PublishedVersion: int(post.PublishedVersion),
		AutosavedAt: post.AutosavedAt, Base: strings.TrimSuffix(s.sitePath(ctx, ws, "posts", "x"), "x"),
		SEO: WebsitePostSEO{Title: post.SeoTitle, Description: post.SeoDescription, SocialImage: post.SocialImage, CanonicalURL: post.CanonicalUrl,
			HideFromSearch: post.HideFromSearch}}
	out.Base = hostOnly(out.Base)
	src, err := s.sourceMJML(ctx, ws, post)
	if err != nil {
		return out, err
	}
	res, rerr := render.Render(src, s.renderOptions(ctx, ws, post))
	if rerr == nil {
		out.Words = res.Words
		out.HasMarker, out.MarkerAfter = res.HasMarker, res.MarkerAfter
	}
	out.ReadingMinutes = render.ReadingMinutes(out.Words)
	out.Checks = s.checks(ctx, ws, post, res, rerr == nil)
	out.Publish = s.publishState(ctx, ws, p, pr, out.Checks)
	out.Bylines = []WebsiteByline{}
	if members, err := s.Q.ApprovalMembers(ctx, ws.ID); err == nil {
		for _, m := range members {
			out.Bylines = append(out.Bylines, WebsiteByline{UserID: m.ID.String(), Name: m.Name, Color: m.AvatarColor})
		}
	}
	return out, nil
}

// hostOnly drops the scheme of an address: "news.acme.io/posts/".
func hostOnly(u string) string {
	u = strings.TrimPrefix(u, "https://")
	return strings.TrimPrefix(u, "http://")
}

// sourceMJML is a post's content: its own (web-only) or its campaign's.
func (s *Service) sourceMJML(ctx context.Context, ws *store.Workspace, p store.WebsitePost) (string, error) {
	if p.Kind == KindWeb || p.CampaignID == nil {
		return p.Mjml, nil
	}
	var mjml string
	err := s.Pool.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE id = $1 AND workspace_id = $2`, *p.CampaignID, ws.ID).Scan(&mjml)
	return mjml, err
}

// renderOptions are the web's links and the post's title and cover.
func (s *Service) renderOptions(ctx context.Context, ws *store.Workspace, p store.WebsitePost) render.Options {
	o := render.Options{Title: p.Title, CoverSrc: p.CoverUrl, PublicURL: s.URLs.Base(), Links: map[string]string{}}
	if s.Forms != nil {
		if set, err := s.Forms.LoadSettings(ctx, ws); err == nil {
			l := s.Forms.Links(ws, set)
			o.Links["preferences_url"] = l.PreferencesRequest()
			o.Links["unsubscribe_url"] = l.PreferencesRequest()
			o.Links["view_in_browser_url"] = l.Path("posts", p.Slug)
		}
	}
	return o
}

// checks are a post's web checks: its address, its cover's alt text, the
// tokens that show their fallback, the blocks left out on the web.
func (s *Service) checks(ctx context.Context, ws *store.Workspace, p store.WebsitePost, res render.Result, rendered bool) []WebsiteWebCheck {
	out := []WebsiteWebCheck{}
	addr := WebsiteWebCheck{Key: "address", Status: "pass", Path: "/posts/" + p.Slug, Fallbacks: []WebsiteFallback{}, Hidden: []WebsiteHiddenBlock{}}
	if owner, err := s.Q.WebsitePostSlugOwner(ctx, store.WebsitePostSlugOwnerParams{WorkspaceID: ws.ID, ID: p.ID, Slug: p.Slug}); err == nil {
		addr.Status = "block"
		addr.Taken = &WebsiteSlugOwner{ID: owner.ID.String(), Title: owner.Title, Status: owner.Status, Since: owner.Since}
		addr.Suggestion = s.freeSlug(ctx, ws, p.ID, p.Slug+"-"+slugTail(p.Title, p.Slug))
	}
	out = append(out, addr)
	cover := WebsiteWebCheck{Key: "cover", Status: "pass", Fallbacks: []WebsiteFallback{}, Hidden: []WebsiteHiddenBlock{}}
	if p.CoverUrl != "" && strings.TrimSpace(p.CoverAlt) == "" {
		cover.Status = "warn"
	}
	if p.CoverUrl == "" {
		cover.Status = "info"
	}
	out = append(out, cover)
	tokens := WebsiteWebCheck{Key: "tokens", Status: "pass", Fallbacks: []WebsiteFallback{}, Hidden: []WebsiteHiddenBlock{}}
	hidden := WebsiteWebCheck{Key: "hidden", Status: "pass", Fallbacks: []WebsiteFallback{}, Hidden: []WebsiteHiddenBlock{}}
	if rendered {
		for _, f := range res.Fallbacks {
			tokens.Fallbacks = append(tokens.Fallbacks, WebsiteFallback{Token: f.Token, Fallback: f.Fallback, Source: f.Source, Sample: f.Sample})
		}
		if len(tokens.Fallbacks) > 0 {
			tokens.Status = "warn"
		}
		for _, h := range res.Hidden {
			hidden.Hidden = append(hidden.Hidden, WebsiteHiddenBlock{ID: h.ID, Name: h.Name, Kind: h.Kind, Reason: h.Reason, Condition: h.Condition, Line: h.Line})
		}
		if len(hidden.Hidden) > 0 {
			hidden.Status = "info"
		}
	}
	return append(out, tokens, hidden)
}

// slugTail is what a taken slug gets to be told apart: the rest of the
// title after the slug's words ("feature-flags" + "at-scale").
func slugTail(title, slug string) string {
	full := slugify(title)
	if strings.HasPrefix(full, slug+"-") {
		return strings.TrimPrefix(full, slug+"-")
	}
	return "2"
}

// PostHash is what an approval of a post covers, for a post and its
// content (the demo's request for Daniel's post).
func PostHash(p store.WebsitePost, src string) string { return contentHash(p, src) }

// contentHash covers what an approval of a post covers: its content and
// title, its address, its visibility and when it goes live.
func contentHash(p store.WebsitePost, src string) string {
	h := sha256.New()
	b, _ := json.Marshal([]any{src, p.Title, p.Excerpt, p.CoverUrl, p.Slug, p.Visibility, p.Tiers, p.NeverFree, p.PublishMode, p.PublishAt})
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil)[:16])
}

// CheckText is a web check in words, as Review & send lists it.
func CheckText(ctx context.Context, c WebsiteWebCheck) (title, detail string) {
	switch c.Key {
	case "address":
		if c.Status == "block" {
			detail = c.Path
			if c.Taken != nil {
				detail = tr(ctx, "check.address.takenBy", map[string]any{"path": c.Path, "title": c.Taken.Title, "suggestion": c.Suggestion})
			}
			return tr(ctx, "check.address.block", nil), detail
		}
		return tr(ctx, "check.address.pass", nil), c.Path
	case "cover":
		switch c.Status {
		case "warn":
			return tr(ctx, "check.cover.warn", nil), tr(ctx, "check.cover.warnDetail", nil)
		case "info":
			return tr(ctx, "check.cover.info", nil), tr(ctx, "check.cover.infoDetail", nil)
		}
		return tr(ctx, "check.cover.pass", nil), ""
	case "tokens":
		if len(c.Fallbacks) == 0 {
			return tr(ctx, "check.tokens.pass", nil), ""
		}
		parts := []string{}
		for _, f := range c.Fallbacks {
			parts = append(parts, tr(ctx, "check.tokens.sample", map[string]any{"source": f.Source, "sample": f.Sample}))
		}
		return tr(ctx, "check.tokens.warn", map[string]any{"count": len(c.Fallbacks)}), strings.Join(parts, " · ")
	case "hidden":
		if len(c.Hidden) == 0 {
			return tr(ctx, "check.hidden.pass", nil), ""
		}
		names := []string{}
		for _, h := range c.Hidden {
			n := h.Name
			if n == "" {
				n = tr(ctx, "check.hidden.kind."+h.Kind, nil)
			}
			names = append(names, n)
		}
		return tr(ctx, "check.hidden.info", map[string]any{"count": len(c.Hidden)}), strings.Join(names, ", ")
	}
	return c.Key, ""
}
