package publicpages

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	pagesvc "github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/store"
)

// The page builder's canvas and previews (PageBuilder): the site's
// stylesheets, header and footer around the blocks, the live blocks drawn
// as readers would see them (inert: nothing submits), and the whole page
// for Preview. The product editor previews a product's default page.

// PageCanvas is what the page builder's canvas loads around the blocks.
type PageCanvas struct {
	// Styles are the site's stylesheets, in order (site, pages, theme).
	Styles []string `json:"styles"`
	// Header and Footer are the site's frame, inert ("Site header · theme").
	Header string `json:"header"`
	// FocusHeader is the header of a page that hides the navigation.
	FocusHeader string `json:"focusHeader"`
	Footer      string `json:"footer"`
	// Lang and Dir are the page's language.
	Lang string `json:"lang"`
	Dir  string `json:"dir"`
}

// PageRenderInput are live blocks to draw (the custom elements' HTML,
// <rc-product …></rc-product>), or a whole page.
type PageRenderInput struct {
	Blocks []string `json:"blocks,omitempty" maxItems:"100"`
	HTML   string   `json:"html,omitempty" maxLength:"2097152"`
}

// PageRendered is the drawn blocks, in the order asked, or the page.
type PageRendered struct {
	Blocks []string `json:"blocks"`
	HTML   string   `json:"html,omitempty"`
}

// PagePreviewInput previews a page: its saved draft, or the HTML and
// settings being edited.
type PagePreviewInput struct {
	HTML       string `json:"html,omitempty" maxLength:"2097152"`
	HideHeader *bool  `json:"hideHeader,omitempty"`
	Name       string `json:"name,omitempty" maxLength:"120"`
	// Live previews what readers see now (the published version).
	Live bool `json:"live,omitempty"`
}

type pageCanvasInput struct {
	pagesvc.PagePath
}

type pageRenderInput struct {
	pagesvc.PagePath
	Body PageRenderInput
}

type pagePreviewInput struct {
	pagesvc.PagePath
	Body PagePreviewInput
}

// ProductPreviewPath is a product to preview (exported for Huma).
type ProductPreviewPath struct {
	api.WorkspaceInput
	Product string `path:"product" format:"uuid"`
}

// RegisterSellAPI adds the page builder's canvas and preview operations.
func RegisterSellAPI(h huma.API, p *Handler) {
	tags := []string{"Pages"}
	base := "/workspaces/{workspace}/pages/{page}"
	view := api.Workspace(authz.WebsiteView)
	api.Get(h, api.Op{ID: "get-page-canvas", Path: base + "/canvas", Summary: "The site's stylesheets, header and footer for the page builder's canvas", Tags: tags, Access: view},
		func(ctx context.Context, in *pageCanvasInput) (*api.Body[PageCanvas], error) {
			out, err := p.Canvas(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "render-page-blocks", Path: base + "/render", Summary: "Draw live blocks (products, forms, countdowns, events) as readers see them, inert",
		Tags: tags, Access: view, MaxBodyBytes: pagesvc.MaxHTML + 64<<10},
		func(ctx context.Context, in *pageRenderInput) (*api.Body[PageRendered], error) {
			out, err := p.RenderBlocks(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Page), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-page", Path: base + "/preview", Summary: "Preview a page in the site's frame: the draft, the blocks being edited, or what's live",
		Tags: tags, Access: view, MaxBodyBytes: pagesvc.MaxHTML + 64<<10},
		func(ctx context.Context, in *pagePreviewInput) (*api.Body[PublicPagePreview], error) {
			out, err := p.PagePreview(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Page), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "preview-product-page", Path: "/workspaces/{workspace}/products/{product}/preview",
		Summary: "Preview a product's default page (used when it has no sales page)", Tags: []string{"Products"}, Access: api.Workspace(authz.ProductsView)},
		func(ctx context.Context, in *ProductPreviewPath) (*api.Body[PublicPagePreview], error) {
			id, err := uuid.Parse(in.Product)
			if err != nil {
				return nil, appNotFound(ctx, "product")
			}
			out, err := p.ProductPreview(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
}

// previewView is the site's frame for a preview, in the default language.
func (h *Handler) previewView(ctx context.Context, ws *store.Workspace) (*site, *siteView, error) {
	if h.Website == nil {
		return nil, nil, appUnavailable(ctx, "siteUnavailable")
	}
	s, err := h.previewSite(ctx, ws)
	if err != nil {
		return nil, nil, err
	}
	code := h.defaultLang(ctx, s)
	v, err := h.siteViewIn(ctx, s, pageLang{Code: code, Labels: pagelabels.For(code, nil)})
	if err != nil {
		return nil, nil, err
	}
	h.sellView(v, s)
	return s, v, nil
}

// Canvas is the frame the page builder draws the blocks in.
func (h *Handler) Canvas(ctx context.Context, ws *store.Workspace) (PageCanvas, error) {
	s, v, err := h.previewView(ctx, ws)
	if err != nil {
		return PageCanvas{}, err
	}
	tmpl, err := h.siteTemplates()
	if err != nil {
		return PageCanvas{}, err
	}
	t := tmpl["sell-page"]
	part := func(name string, focus bool) (string, error) {
		v.Focus = focus
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, name, v); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	out := PageCanvas{Styles: []string{v.CSS}, Lang: v.Lang, Dir: v.Dir}
	out.Styles = append(out.Styles, v.Styles...)
	out.Styles = append(out.Styles, v.Theme)
	if out.Header, err = part("site-header", false); err != nil {
		return out, err
	}
	if out.FocusHeader, err = part("site-header", true); err != nil {
		return out, err
	}
	if out.Footer, err = part("site-footer", false); err != nil {
		return out, err
	}
	_ = s
	return out, nil
}

// RenderBlocks draws a page's live blocks for the builder: inert, with
// drafts and placeholders shown.
func (h *Handler) RenderBlocks(ctx context.Context, ws *store.Workspace, pageID uuid.UUID, in PageRenderInput) (PageRendered, error) {
	s, v, err := h.previewView(ctx, ws)
	if err != nil {
		return PageRendered{}, err
	}
	if h.Pages != nil {
		if _, err := h.Pages.Draft(ctx, ws, pageID); err != nil {
			return PageRendered{}, err
		}
	}
	out := PageRendered{Blocks: make([]string, 0, len(in.Blocks))}
	pr := &pageRender{h: h, ctx: ctx, s: s, v: v, pageID: pageID, source: "page", preview: true, now: h.now()}
	for _, b := range in.Blocks {
		src := pagesvc.Sanitize(b)
		if !strings.HasPrefix(src, "<rc-") {
			out.Blocks = append(out.Blocks, "")
			continue
		}
		html, err := pr.render(src)
		if err != nil {
			return out, err
		}
		out.Blocks = append(out.Blocks, string(html))
	}
	if in.HTML != "" {
		html, err := pr.render(pagesvc.Sanitize(in.HTML))
		if err != nil {
			return out, err
		}
		out.HTML = string(html)
	}
	return out, nil
}

// PagePreview renders a whole page in the site's frame, inert.
func (h *Handler) PagePreview(ctx context.Context, ws *store.Workspace, pageID uuid.UUID, in PagePreviewInput) (PublicPagePreview, error) {
	if h.Pages == nil {
		return PublicPagePreview{}, appNotFound(ctx, "page")
	}
	lp, err := h.Pages.Draft(ctx, ws, pageID)
	if err != nil {
		return PublicPagePreview{}, err
	}
	if in.Live {
		res, err := h.Pages.Resolve(ctx, ws, lp.Slug)
		if err == nil && res.Page != nil && res.Page.ID == pageID {
			lp = res.Page
		}
	}
	src := lp.HTML
	if strings.TrimSpace(in.HTML) != "" {
		src = pagesvc.Sanitize(in.HTML)
	}
	if in.Name != "" {
		lp.Name = in.Name
	}
	if in.HideHeader != nil {
		lp.Settings.HideHeader = *in.HideHeader
	}
	s, v, err := h.previewView(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	h.pageMeta(s, v, lp)
	v.Focus = lp.Settings.HideHeader
	v.mark("page:"+lp.Slug, v.Frame)
	pr := &pageRender{h: h, ctx: ctx, s: s, v: v, pageID: lp.ID, pageURL: s.path(lp.Slug), source: "page", preview: true, now: h.now()}
	body, err := pr.render(src)
	if err != nil {
		return PublicPagePreview{}, err
	}
	v.Body = pageBody{HTML: body, BuyBar: pr.buyBar}
	out, err := h.siteHTML("sell-page", v)
	if err != nil {
		return PublicPagePreview{}, err
	}
	return PublicPagePreview{HTML: string(out), URL: s.canonPath(lp.Slug)}, nil
}

// ProductPreview renders a product's default page, its draft included.
func (h *Handler) ProductPreview(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PublicPagePreview, error) {
	if h.Products == nil {
		return PublicPagePreview{}, appNotFound(ctx, "product")
	}
	m, err := h.Products.PublicProducts(ctx, ws, []uuid.UUID{id}, true)
	if err != nil {
		return PublicPagePreview{}, err
	}
	p, ok := m[id]
	if !ok {
		return PublicPagePreview{}, appNotFound(ctx, "product")
	}
	s, v, err := h.previewView(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	r, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	body, bar := h.productBody(r, s, v, p, true)
	body.BuyBar = bar
	v.Body = body
	v.Title = v.T("site.title", "page", p.Live.Name, "site", v.Frame.Title)
	out, err := h.siteHTML("sell-product", v)
	if err != nil {
		return PublicPagePreview{}, err
	}
	return PublicPagePreview{HTML: string(out), URL: s.canonPath("products", p.Live.Slug)}, nil
}
