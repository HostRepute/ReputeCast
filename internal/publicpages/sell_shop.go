package publicpages

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	pagesvc "github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/website"
)

// The shop (/products), the default product page (/products/{slug}),
// the Stripe handoff and the thank-you page.

// shopBody is the shop.
type shopBody struct {
	Title, Intro string
	Cards        template.HTML
}

// productBody is the default product page.
type productBody struct {
	ShopHref, ShopText string
	Name, Short        string
	Long               []string
	Meta               string
	Cover              template.HTML
	Box                template.HTML
	Notice             template.HTML
	BuyBar             *buyBarView
}

// shopPage is /products.
func (h *Handler) shopPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Pages == nil || h.Products == nil {
		h.siteNotFound(w, r, s)
		return
	}
	ctx := r.Context()
	if on, err := h.Pages.ShopOn(ctx, s.ws); err != nil || !on {
		h.siteNotFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	h.sellView(v, s)
	list, err := h.Products.ShopProducts(ctx, s.ws)
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	title, intro := h.Pages.ShopText(ctx, s.ws)
	if title == "" {
		title = v.S("shop.title")
	}
	if intro == "" {
		intro = v.S("shop.intro", "publisher", s.set.Branding.PublisherName)
	}
	pr := &pageRender{h: h, ctx: ctx, s: s, v: v, source: "shop", now: h.now()}
	v.Body = shopBody{Title: title, Intro: intro, Cards: pr.shopCards(list, "")}
	v.Title = v.T("site.title", "page", title, "site", v.Frame.Title)
	v.Description = intro
	v.Canonical = s.canonPath("products")
	v.mark("shop", v.Frame)
	if r.Method == http.MethodGet && h.Website != nil {
		h.Website.CountView(ctx, s.ws, r, website.PagePage, uuid.Nil)
	}
	h.renderSite(w, r, s, "sell-shop", http.StatusOK, v)
}

// appsFor names the apps a file opens in ("Excel, Numbers or Google Sheets").
func appsFor(v *siteView, kind string) string {
	switch strings.ToLower(kind) {
	case "xlsx", "csv":
		return v.S("shop.apps.sheet")
	case "docx":
		return v.S("shop.apps.doc")
	case "pptx":
		return v.S("shop.apps.slides")
	case "epub":
		return v.S("shop.apps.ebook")
	}
	return ""
}

// productPage is a product's default page: used when it has no sales page.
func (h *Handler) productPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Products == nil {
		h.siteNotFound(w, r, s)
		return
	}
	ctx := r.Context()
	p, moved, err := h.Products.ProductBySlug(ctx, s.ws, chi.URLParam(r, "slug"), false)
	switch {
	case moved != "":
		http.Redirect(w, r, s.path("products", moved), http.StatusMovedPermanently) //nolint:gosec // G710: a path on this site
		return
	case errors.Is(err, products.ErrNotForSale):
		h.siteNotFound(w, r, s)
		return
	case err != nil:
		h.siteFail(w, r, s, err)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	h.sellView(v, s)
	body, buyBar := h.productBody(r, s, v, p, false)
	if r.URL.Query().Get("checkout") == "cancelled" {
		body.Notice = v.SH("buy.cancelled")
		v.private = true
	}
	body.BuyBar = buyBar
	v.Body = body
	v.Title = v.T("site.title", "page", p.Live.Name, "site", v.Frame.Title)
	v.Description = p.Live.ShortDesc
	v.Canonical = s.canonPath("products", p.Live.Slug)
	v.OG = &og{Type: "product", SiteName: v.Frame.Title, Title: p.Live.Name, Description: p.Live.ShortDesc, URL: v.Canonical, Image: h.mediaURL(p.Live.CoverURL)}
	if r.Method == http.MethodGet && h.Website != nil {
		h.Website.CountView(ctx, s.ws, r, website.PagePage, uuid.Nil)
	}
	h.renderSite(w, r, s, "sell-product", http.StatusOK, v)
}

// productBody builds the default product page (the builder previews a
// draft's too).
func (h *Handler) productBody(r *http.Request, s *site, v *siteView, p products.PublicProduct, preview bool) (productBody, *buyBarView) {
	pr := &pageRender{h: h, ctx: r.Context(), s: s, v: v, source: "product_page", pageURL: s.path("products", p.Live.Slug), now: h.now(), preview: preview,
		products: map[uuid.UUID]products.PublicProduct{p.ID: p}}
	if h.Products != nil {
		pr.merchant, _ = h.Products.Merchant(r.Context(), s.ws)
	}
	node := &html.Node{Type: html.ElementNode, Data: pagesvc.ElProduct, Attr: []html.Attribute{{Key: "product", Val: p.ID.String()}, {Key: "id", Val: "buy"}, {Key: "price", Val: "1"}, {Key: "offer", Val: "1"}}}
	b := productBody{ShopHref: s.path("products"), ShopText: v.S("shop.crumb"), Name: p.Live.Name, Short: p.Live.ShortDesc,
		Cover: template.HTML(coverImg(p, "lg")), Box: pr.buyBox(node, p)} //nolint:gosec // escaped by coverImg
	for _, para := range strings.Split(p.Live.LongDesc, "\n") {
		if t := strings.TrimSpace(para); t != "" {
			b.Long = append(b.Long, t)
		}
	}
	if p.Files.Count > 0 {
		file := pr.fileLine(p.Files)
		if apps := appsFor(v, p.Files.Kind); apps != "" {
			b.Meta = v.S("shop.fileMeta", "file", file, "apps", apps)
		} else {
			b.Meta = file
		}
	}
	return b, pr.buyBar
}

// handoffBody is "Taking you to Stripe…".
type handoffBody struct {
	Text         template.HTML
	Name, Price  string
	Action       string
	Fields       map[string]string
	Back         string
	Error        string
	Continue     bool
	ContinueText string
}

// buyHandoff answers a Buy button: "Taking you to Stripe…", which posts on
// to create the Checkout session (pages.js does it at once; without
// scripts, "Continue to Stripe" does the same).
func (h *Handler) buyHandoff(w http.ResponseWriter, r *http.Request) {
	h.handoff(w, r, false)
}

// buyGo creates the Checkout session and sends the buyer to Stripe.
func (h *Handler) buyGo(w http.ResponseWriter, r *http.Request) {
	h.handoff(w, r, true)
}

func (h *Handler) handoff(w http.ResponseWriter, r *http.Request, start bool) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Products == nil || h.Pages == nil {
		h.siteNotFound(w, r, s)
		return
	}
	if !h.parseForm(w, r, 16<<10) {
		l := i18n.For(i18n.English)
		if loc, ok := i18n.ParseLocale(h.visitorLang(nil, r, s, nil).Code); ok {
			l = i18n.For(loc)
		}
		http.Error(w, sellMsgs.M("shop.formUnreadable").Text(l, nil), http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "product"))
	if err != nil {
		h.siteNotFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	h.sellView(v, s)
	noStore(w, v)
	v.Robots = "noindex, nofollow"
	v.Title = v.T("site.title", "page", v.S("handoff.title"), "site", v.Frame.Title)
	m, _ := h.Products.PublicProducts(ctx, s.ws, []uuid.UUID{id}, false)
	p, found := m[id]
	merchant, _ := h.Products.Merchant(ctx, s.ws)
	// Where the Buy button was: a page, the shop or the product's page.
	src := r.PostForm.Get("_src")
	back, thanks, label := s.path("products"), "", ""
	var pageID *uuid.UUID
	if pid, err := uuid.Parse(r.PostForm.Get("_page")); err == nil {
		if lp, err := h.Pages.Draft(ctx, s.ws, pid); err == nil {
			if res, err := h.Pages.Resolve(ctx, s.ws, lp.Slug); err == nil && res.Page != nil && res.Page.ID == pid {
				pageID = &pid
				back = s.path(res.Page.Slug)
				thanks = s.path(res.Page.Slug, "thanks")
			}
		}
	}
	if found && pageID == nil {
		if src == "product_page" {
			back = s.path("products", p.Live.Slug)
		}
		thanks = s.path("products", p.Live.Slug, "thanks")
	}
	if src != "page" && src != "shop" && src != "product_page" {
		src = "page"
	}
	if pageID == nil && src == "page" {
		src = "product_page"
	}
	label = strings.TrimPrefix(strings.TrimPrefix(back, "https://"), "http://")
	body := handoffBody{Action: s.path("buy", id.String(), "go"), Back: back, ContinueText: v.S("handoff.continue"),
		Fields: map[string]string{"offer": r.PostForm.Get("offer"), "_page": r.PostForm.Get("_page"), "_src": src}}
	if !found || !merchant.Connected {
		body.Error = v.S("handoff.closed")
		v.Body = body
		h.renderSite(w, r, s, "sell-handoff", http.StatusConflict, v)
		return
	}
	accountName := merchant.Account
	if accountName == "" {
		accountName = s.set.Branding.PublisherName
	}
	body.Text = v.SH("handoff.text", "merchant", accountName)
	body.Name = p.Live.Name
	full := money(p.Live.PriceCents, p.Currency)
	if !strings.Contains(full, ".") {
		full += ".00"
	}
	switch {
	case p.Live.PriceKind == products.PricePWYW:
		body.Price = v.S("handoff.pwyw", "min", money(p.Live.MinCents, p.Currency))
	case merchant.Tax:
		body.Price = v.S("handoff.price", "price", full)
	default:
		body.Price = v.S("handoff.priceNoTax", "price", full)
	}
	body.Continue = true
	if !start {
		v.Body = body
		h.renderSite(w, r, s, "sell-handoff", http.StatusOK, v)
		return
	}
	in := products.CheckoutStart{ProductID: id, Ticked: r.PostForm.Get("offer") == "1", Source: src, PageID: pageID, PageURL: back, PageLabel: label,
		ThanksURL: thanks + "?session_id={CHECKOUT_SESSION_ID}"}
	if ip := clientIP(r); ip != nil {
		in.IP = ip.String()
	}
	if v.Session != nil {
		in.Email = v.Session.Email
		cid := v.Session.ContactID
		in.ContactID = &cid
	}
	hand, err := h.Products.StartCheckout(ctx, s.ws, in)
	if err != nil {
		var p *api.Problem
		body.Error = v.S("handoff.failed")
		if errors.Is(err, products.ErrNotForSale) {
			body.Error = v.S("handoff.closed")
			body.Continue = false
		} else if !errors.As(err, &p) {
			h.log().ErrorContext(ctx, "checkout", "err", err)
		}
		v.Body = body
		h.renderSite(w, r, s, "sell-handoff", http.StatusServiceUnavailable, v)
		return
	}
	http.Redirect(w, r, hand.URL, http.StatusSeeOther) //nolint:gosec // G710: Stripe's Checkout address for this session
}

// thanksBody is the thank-you page.
type thanksBody struct {
	Test, Emailed, Offer template.HTML
	Name, Files          string
	Download             string
	Receipt              string
	Waiting, Unknown     bool
	Refresh              string
}

// customThanks is a page's thank-you page (/{slug}/thanks).
func (h *Handler) customThanks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Pages == nil {
		h.siteNotFound(w, r, s)
		return
	}
	res, err := h.Pages.Resolve(r.Context(), s.ws, chi.URLParam(r, "slug"))
	if err != nil || res.Page == nil {
		h.siteNotFound(w, r, s)
		return
	}
	h.thanks(w, r, s, res.Page.Settings.HideHeader)
}

// productThanks is a product page's thank-you page.
func (h *Handler) productThanks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	h.thanks(w, r, s, false)
}

// thanks shows the download of the order a returning buyer paid for.
func (h *Handler) thanks(w http.ResponseWriter, r *http.Request, s *site, focus bool) {
	if h.Products == nil {
		h.siteNotFound(w, r, s)
		return
	}
	ctx := r.Context()
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	h.sellView(v, s)
	noStore(w, v)
	v.Robots = "noindex, nofollow"
	v.Focus = focus
	v.Title = v.T("site.title", "page", v.S("thanks.title"), "site", v.Frame.Title)
	session := r.URL.Query().Get("session_id")
	tv, ok, err := h.Products.ThanksFor(ctx, s.ws, session)
	if err != nil {
		h.log().WarnContext(ctx, "thank-you page", "err", err)
	}
	b := thanksBody{Refresh: r.URL.RequestURI()}
	if !ok {
		b.Unknown = session == ""
		b.Waiting = session != ""
		v.Body = b
		h.renderSite(w, r, s, "sell-thanks", http.StatusOK, v)
		return
	}
	loc := location(s.ws)
	b.Name = tv.ProductName
	b.Files = pageFileLine(v, tv.Files)
	b.Download = tv.DownloadURL
	b.Emailed = v.SH("thanks.emailed", "email", tv.EmailMasked, "date", v.shortDate(tv.ExpiresAt.In(loc)), "count", int(tv.Downloads))
	if tv.OfferPending && tv.OfferList != "" {
		b.Offer = template.HTML(template.HTMLEscapeString(v.S("thanks.offer", "list", tv.OfferList))) //nolint:gosec // escaped
	}
	merchant, _ := h.Products.Merchant(ctx, s.ws)
	if tv.Test {
		name := merchant.Account
		if name == "" {
			name = s.set.Branding.PublisherName
		}
		b.Test = v.SH("thanks.test", "merchant", name)
		b.Receipt = v.S("thanks.receiptTest", "order", tv.Order.Number)
	} else {
		b.Receipt = v.S("thanks.receipt", "order", tv.Order.Number)
	}
	if v.Frame.On && v.Frame.SigninOn {
		b.Receipt += " " + v.S("thanks.lost", "account", strings.TrimPrefix(strings.TrimPrefix(s.path("account"), "https://"), "http://"))
	}
	v.Body = b
	h.renderSite(w, r, s, "sell-thanks", http.StatusOK, v)
}

// pageFileLine writes a product's files with the pages' words.
func pageFileLine(v *siteView, f products.ProductFilesSummary) string {
	if f.Count == 0 {
		return ""
	}
	if f.Pages > 0 && f.Count == 1 {
		return v.S("dl.filePages", "kind", f.Kind, "count", f.Pages, "size", v.size(f.Bytes))
	}
	return v.S("dl.file", "kind", f.Kind, "size", v.size(f.Bytes))
}
