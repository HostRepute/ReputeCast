package publicpages

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// Previews of a post on the web for the app (ComposerPublishToWeb):
// "Preview on web" as each kind of reader, from the post as it is now,
// published or not; and the approval review's web preview. Inert, like
// every preview: nothing submits and nothing is counted.

// WebsitePostPreviewInput says whom to preview a post for.
type WebsitePostPreviewInput struct {
	As   string `json:"as" enum:"visitor,visitor_used,free,free_used,paid" doc:"A visitor (first metered read, or the meter used up), a free member (or the meter used up), a paid member"`
	Tier string `json:"tier,omitempty" maxLength:"64" doc:"A paid member's tier (the highest by default)"`
}

// WebsitePostPreviewPath is a post to preview (exported: Huma reads
// embedded path structs only when their type is exported).
type WebsitePostPreviewPath struct {
	api.WorkspaceInput
	Post string `path:"post" format:"uuid" doc:"The post's id"`
}

type postPreviewInput struct {
	WebsitePostPreviewPath
	Body WebsitePostPreviewInput
}

type homePreviewInput struct {
	api.WorkspaceInput
	Body website.WebsiteHomeNavInput
}

// registerSitePreviews adds the post preview operation.
func registerSitePreviews(h huma.API, p *Handler) {
	api.Post(h, api.Op{ID: "preview-website-post", Path: "/workspaces/{workspace}/website/posts/{post}/preview",
		Summary: "Preview a post on the web as a visitor, a free member or a paid member", Tags: []string{"Website"}, Access: api.Workspace(authz.WebsiteView)},
		func(ctx context.Context, in *postPreviewInput) (*api.Body[PublicPagePreview], error) {
			id, err := uuid.Parse(in.Post)
			if err != nil {
				return nil, api.NotFound("Post")
			}
			out, err := p.PostPreview(ctx, api.WorkspaceFrom(ctx), id, in.Body.As, in.Body.Tier)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-website-home", Path: "/workspaces/{workspace}/website/home-nav/preview",
		Summary: "Preview the home page with unsaved Home & navigation settings, as a visitor sees it", Tags: []string{"Website"}, Access: api.Workspace(authz.WebsiteView)},
		func(ctx context.Context, in *homePreviewInput) (*api.Body[PublicPagePreview], error) {
			out, err := p.HomePreview(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
}

// HomePreview renders the home page as a visitor sees it, with unsaved
// Home & navigation settings (the tab's live preview). Nothing is saved.
func (h *Handler) HomePreview(ctx context.Context, ws *store.Workspace, in website.WebsiteHomeNavInput) (PublicPagePreview, error) {
	if h.Website == nil {
		return PublicPagePreview{}, api.Err(http.StatusServiceUnavailable, "unavailable", "The website isn't available on this server.")
	}
	ctx, err := h.Website.WithHomeNavDraft(ctx, ws, in)
	if err != nil {
		return PublicPagePreview{}, err
	}
	s, err := h.previewSite(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	// The footer's links are the branding's, changed in the same tab.
	s.set.Branding.PoweredBy = in.PoweredBy
	s.set.Branding.PrivacyURL = strings.TrimSpace(in.PrivacyURL)
	code := h.defaultLang(ctx, s)
	v, err := h.siteViewIn(ctx, s, pageLang{Code: code, Labels: pagelabels.For(code, nil)})
	if err != nil {
		return PublicPagePreview{}, err
	}
	body, err := h.homeBody(ctx, s, v, "")
	if err != nil {
		return PublicPagePreview{}, err
	}
	v.Body = body
	v.mark("home", v.Frame)
	v.Title = v.Frame.Title
	v.Description = v.Frame.Description
	out, err := h.siteHTML("home", v)
	if err != nil {
		return PublicPagePreview{}, err
	}
	return PublicPagePreview{HTML: string(out), URL: s.canonPath()}, nil
}

// PostPreview renders a post's page as one kind of reader, in the
// workspace's default language.
func (h *Handler) PostPreview(ctx context.Context, ws *store.Workspace, id uuid.UUID, as, tier string) (PublicPagePreview, error) {
	if h.Website == nil {
		return PublicPagePreview{}, api.Err(http.StatusServiceUnavailable, "unavailable", "The website isn't available on this server.")
	}
	s, err := h.previewSite(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	code := h.defaultLang(ctx, s)
	pl := pageLang{Code: code, Labels: pagelabels.For(code, nil)}
	v, err := h.siteViewIn(ctx, s, pl)
	if err != nil {
		return PublicPagePreview{}, err
	}
	p, pv, reader, err := h.Website.PreviewPost(ctx, ws, id, as, tier, v.labels.Lang())
	if err != nil {
		return PublicPagePreview{}, err
	}
	if reader.Member != nil {
		name := "Sam Rivera"
		v.Member = &siteMember{Name: name, First: "Sam", Initials: initials(name), Color: avatarColor(name), Email: "sam@example.com"}
	}
	body, err := h.postBody(ctx, s, v, p, pv, reader, nil)
	if err != nil {
		return PublicPagePreview{}, err
	}
	v.Body = body
	h.postMeta(s, v, p, pv)
	out, err := h.siteHTML("post", v)
	if err != nil {
		return PublicPagePreview{}, err
	}
	return PublicPagePreview{HTML: string(out), URL: s.canonPath("posts", p.Slug), Contact: v.memberEmail()}, nil
}

// memberEmail is the previewed member's address ("" for a visitor).
func (v *siteView) memberEmail() string {
	if v.Member == nil {
		return ""
	}
	return v.Member.Email
}

// ApprovalPreview is the approval review's web preview: the post whole,
// as a reader holding its tier reads it.
func (h *Handler) ApprovalPreview(ctx context.Context, ws *store.Workspace, postID uuid.UUID) (string, error) {
	out, err := h.PostPreview(ctx, ws, postID, website.AsPaid, "")
	return out.HTML, err
}
