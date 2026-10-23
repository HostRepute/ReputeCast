package publicpages

// Landing and sales pages, the shop, checkout and downloads (design
// expansion phase 7, M29: PublicSalesPage, PublicPagesStates). Pages are
// the site's pages (site.go's frame and theme) with static/pages.css on
// top; focused landing pages keep the header to the logo. Blocks that show
// live data (products and their Buy buttons, signup forms, countdowns,
// event details) are drawn on every request (sell_blocks.go).
//
//	/{slug}                     a published page (sell_page.go)
//	/{slug}/form                its signup form's posts
//	/{slug}/thanks              back from Stripe: the download
//	/products, /products/{slug} the shop and the default product page
//	/buy/{product}              "Taking you to Stripe…", then Checkout
//	/d/{token}                  the download page and its files
//	/l/{token}                  a lead magnet's download (the confirmation page)

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/products"
)

//go:embed sell/messages/*.json
var sellFS embed.FS

var sellMsgs = func() *i18n.Catalog {
	sub, err := fs.Sub(sellFS, "sell")
	if err != nil {
		panic(err)
	}
	return i18n.MustLoad(sub, "sell")
}()

// The pages' templates, rendered in the site's frame.
func init() {
	sitePages = append(sitePages, "sell-page", "sell-gate", "sell-shop", "sell-product", "sell-handoff", "sell-thanks", "sell-download", "sell-registered")
}

// sellRootPaths are the addresses on a custom domain. Custom pages answer
// at /{slug}; the second segments are the pages' own.
var sellRootPaths = []string{"/products", "/products/*", "/buy/*", "/d/*", "/l/*", "/account/purchases/*",
	"/{slug}", "/{slug}/form", "/{slug}/thanks", "/{slug}/event.ics"}

// sellRoutes mounts the pages.
func (h *Handler) sellRoutes(r chi.Router) {
	r.Get("/products", h.shopPage)
	r.Get("/products/{slug}", h.productPage)
	r.Get("/products/{slug}/thanks", h.productThanks)
	r.Post("/buy/{product}", h.buyHandoff)
	r.Post("/buy/{product}/go", h.buyGo)
	r.Get("/d/{token}", h.downloadPage)
	r.Head("/d/{token}", h.headOnly)
	r.Get("/d/{token}/{file}", h.downloadFile)
	r.Post("/d/{token}/new-link", h.downloadNewLink)
	r.Get("/l/{token}", h.leadMagnetFile)
	r.Post("/account/purchases/{order}", h.accountNewLink)
	r.Get("/{slug}", h.customPage)
	r.Head("/{slug}", h.customPage)
	r.Post("/{slug}/form", h.customForm)
	r.Get("/{slug}/thanks", h.customThanks)
	r.Get("/{slug}/event.ics", h.eventICS)
}

// S is one of the pages' words as plain text.
func (v *siteView) S(key string, kv ...any) string {
	args, _ := labelArgs(kv)
	return sellMsgs.M(key).Text(v.words.L, args)
}

// size is a file's size as the page's region writes it ("4.2 MB", "4,2
// MB").
func (v *siteView) size(n int64) string { return products.HumanSizeIn(v.words.L, n) }

// fileText is a product's files in the page's words: "PDF, 86 pages", "ZIP,
// 38 MB" (products.FileText's); long adds a PDF's size ("PDF, 86 pages, 4.2
// MB", products.FileTextLong's).
func (v *siteView) fileText(f products.ProductFilesSummary, long bool) string {
	switch {
	case f.Count == 0:
		return ""
	case f.Count == 1 && f.Pages > 0 && long:
		return v.S("files.kindPagesSize", "kind", f.Kind, "count", f.Pages, "size", v.size(f.Bytes))
	case f.Count == 1 && f.Pages > 0:
		return v.S("files.kindPages", "kind", f.Kind, "count", f.Pages)
	}
	return v.S("files.kindSize", "kind", f.Kind, "size", v.size(f.Bytes))
}

// SH is one of the pages' sentences with markup (<b>).
func (v *siteView) SH(key string, kv ...any) template.HTML {
	args, _ := labelArgs(kv)
	tags := i18n.Tags{"b": func(inner string) string { return "<b>" + inner + "</b>" }}
	return template.HTML(sellMsgs.M(key).HTML(v.words.L, args, tags)) //nolint:gosec // the formatter escapes the text and the values
}

// sellView prepares the site's frame for one of the pages: its stylesheet
// and script.
func (h *Handler) sellView(v *siteView, s *site) {
	v.Styles = append(v.Styles, s.links.Static("pages.css?v="+pagesVersion()))
	if !v.Preview {
		v.Scripts = append(v.Scripts, s.links.Static("pages.js?v="+pagesJSVersion()))
	}
}

var pagesVersion = sync.OnceValue(func() string { return assetHash("static/pages.css") })
var pagesJSVersion = sync.OnceValue(func() string { return assetHash("static/pages.js") })

func assetHash(name string) string {
	data, err := staticFS.ReadFile(name)
	if err != nil {
		return "0"
	}
	return strings.TrimSpace(hashString(data))
}

// noStore keeps a page out of every cache (it holds a personal token or
// an order).
func noStore(w http.ResponseWriter, v *siteView) {
	v.private, v.secret = true, true
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
}
