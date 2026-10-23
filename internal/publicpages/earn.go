package publicpages

// The Earn pages (design expansion phase 6, X16): the upgrade page with its
// tiers and Stripe Checkout, the welcome page after checkout, and the
// billing portal links of member emails and the preference center. Their
// words come from internal/earn, in the page's language like every public
// page (the menu's pick, then the browser's languages for a page anyone
// can open, the reader's own for a personal link, then the default); the
// branding is the publisher's.
//
//	/p/{workspace}/upgrade[/{tier}]       the upgrade page (a hidden tier's own link)
//	/p/{workspace}/upgrade/checkout       POST: opens Stripe Checkout
//	/p/{workspace}/upgrade/welcome        after checkout (?session=)
//	/p/{workspace}/billing/{token}        a paid member's billing portal
//	/p/{workspace}/cancel/{token}         the portal's cancel step (the cancel page comes with offers)
//	/p/{workspace}/gift…                  the gift pages (gift.go, X17)

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// earnRoutes adds the Earn pages to a workspace's page routes.
func (h *Handler) earnRoutes(r chi.Router) {
	r.Get("/upgrade", h.upgradePage)
	r.Head("/upgrade", h.headOnly)
	r.Post("/upgrade/checkout", h.upgradeCheckout)
	r.Get("/upgrade/welcome", h.upgradeWelcome)
	r.Get("/upgrade/{tier}", h.upgradePage)
	r.Get("/billing/{token}", h.billingPortal)
	r.Head("/billing/{token}", h.headOnly)
	r.Get("/cancel/{token}", h.cancelPage)
	r.Post("/cancel/{token}", h.cancelPost)
	r.Head("/cancel/{token}", h.headOnly)
	// The gift pages (X17, gift.go).
	h.giftRoutes(r)
}

// earnRootPaths are the Earn pages on a verified custom domain.
var earnRootPaths = []string{"/upgrade", "/upgrade/*", "/billing/*", "/cancel/*", "/gift", "/gift/*"}

// upgradeBody is the upgrade page.
type upgradeBody struct {
	View        earn.UpgradeView
	Action      string
	Token       string
	Test        string
	MonthHref   string
	YearHref    string
	ArchiveHref string
	TermsURL    string
	Upgrading   string
	Retry       string
	Footer      string
}

// welcomeBody is the welcome page after checkout.
type welcomeBody struct {
	earn.WelcomeView
	Footer string
}

// earnView is a view of an Earn page in its language, with the footer's
// language menu (none without a request: the app's previews).
func (h *Handler) earnView(r *http.Request, s *site, title string, body any, pl pageLang) *view {
	v := h.view(s, title, body)
	h.setLang(v, r, s, pl, nil)
	return v
}

// earnLang is the language of an Earn page anyone can open (the menu's
// pick, then the browser's languages, then the default), and the request
// with it in its context: internal/earn, gifts and offers write the page's
// words, dates and prices in the reader's language.
func (h *Handler) earnLang(w http.ResponseWriter, r *http.Request, s *site) (pageLang, *http.Request) {
	pl := h.visitorLang(w, r, s, nil)
	return pl, r.WithContext(i18n.WithLang(r.Context(), pl.Labels.Lang()))
}

// earnLinkLang is the language of an Earn page reached from a personal
// link in a member email (billing, cancel): the menu's pick, then the
// reader's language, then the fallback.
func (h *Handler) earnLinkLang(w http.ResponseWriter, r *http.Request, s *site, token string, kind linktoken.Kind) (pageLang, *http.Request) {
	pl := h.visitorLang(w, r, s, nil)
	if c, ok := h.claims(s, token, kind); ok {
		pl = h.messageLang(w, r, s, c)
	}
	return pl, r.WithContext(i18n.WithLang(r.Context(), pl.Labels.Lang()))
}

func (h *Handler) earnTitle(s *site, heading string) string {
	return heading + " · " + s.set.Branding.PublisherName
}

func (h *Handler) upgradePage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil {
		h.notFound(w, r, s)
		return
	}
	pl, r := h.earnLang(w, r, s)
	q := r.URL.Query()
	req := earn.UpgradeRequest{Token: q.Get("r"), Tier: chi.URLParam(r, "tier"), Period: q.Get("period"), Canceled: q.Get("canceled") == "1",
		TestToken: q.Get("test"), Lang: pl.Labels.Lang(), Offer: q.Get("offer")}
	v, err := h.Earn.Upgrade(r.Context(), s.ws, req)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if r.Method == http.MethodGet && req.TestToken == "" {
		h.Earn.Visit(r.Context(), s.ws, "upgrade")
		if req.Token != "" {
			h.Earn.UpgradeClicked(r.Context(), s.ws, req.Token)
		}
	}
	h.renderUpgrade(w, r, s, v, req, http.StatusOK, pl)
}

func (h *Handler) renderUpgrade(w http.ResponseWriter, r *http.Request, s *site, v earn.UpgradeView, req earn.UpgradeRequest, status int, pl pageLang) {
	page := "upgrade"
	if req.Tier != "" {
		page += "/" + url.PathEscape(req.Tier)
	}
	period := func(p string) string {
		q := url.Values{}
		q.Set("period", p)
		if req.Token != "" {
			q.Set("r", req.Token)
		}
		if req.TestToken != "" {
			q.Set("test", req.TestToken)
		}
		return earnLink(s.links, page) + "?" + q.Encode()
	}
	body := upgradeBody{View: v, Action: earnLink(s.links, "upgrade/checkout"), Token: req.Token, Test: req.TestToken, MonthHref: period("month"),
		YearHref: period("year"), ArchiveHref: s.links.Archive(), TermsURL: v.TermsURL, Upgrading: earn.TextIn(v.Lang, "page.upgrading", nil), Retry: r.URL.String()}
	vw := h.earnView(r, s, h.earnTitle(s, v.Headline), body, pl)
	vw.Canonical = earnLink(s.canon, "upgrade")
	if req.Token == "" && req.TestToken == "" {
		vw.Robots = "index, follow"
	}
	// Checkout is Stripe's page: the form's redirect goes there.
	h.render(w, r, s, "upgrade", status, vw, csp{formAction: []string{"https://checkout.stripe.com"}}, req.Token != "" || req.TestToken != "")
}

// earnLink is an Earn page's address in a workspace's links.
func earnLink(l forms.PageLinks, path string) string {
	home := strings.TrimSuffix(l.Home(), "subscribe")
	return home + path
}

func (h *Handler) upgradeCheckout(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil || !h.parseForm(w, r, 1<<14) {
		h.notFound(w, r, s)
		return
	}
	req := earn.CheckoutRequest{Tier: r.PostForm.Get("tier"), Period: r.PostForm.Get("period"), Token: r.PostForm.Get("r"), TestToken: r.PostForm.Get("test"),
		Offer: r.PostForm.Get("offer")}
	to, err := h.Earn.StartCheckout(r.Context(), s.ws, req)
	if err != nil {
		if !errors.Is(err, earn.ErrUnavailable) {
			h.log().ErrorContext(r.Context(), "upgrade checkout", "err", err)
		}
		pl, r := h.earnLang(w, r, s)
		l := pl.Labels.Lang()
		up := earn.UpgradeRequest{Token: req.Token, Period: req.Period, TestToken: req.TestToken, Lang: l, Offer: req.Offer}
		v, verr := h.Earn.Upgrade(r.Context(), s.ws, up)
		if verr != nil {
			h.serverError(w, r, s, verr)
			return
		}
		v.Notice = &earn.PageNotice{Tone: "bad", Title: earn.TextIn(l, "page.unavailableTitle", nil), Text: earn.TextIn(l, "page.unavailableText", nil), Retry: true}
		h.renderUpgrade(w, r, s, v, up, http.StatusServiceUnavailable, pl)
		return
	}
	h.toProvider(w, r, s, to)
}

func (h *Handler) upgradeWelcome(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil {
		h.notFound(w, r, s)
		return
	}
	pl, r := h.earnLang(w, r, s)
	v, err := h.Earn.Welcome(r.Context(), s.ws, r.URL.Query().Get("session"), pl.Labels.Lang())
	if errors.Is(err, earn.ErrUnavailable) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), welcomeBody{WelcomeView: v}, pl)
	h.render(w, r, s, "welcome", http.StatusOK, vw, csp{}, true)
}

func (h *Handler) billingPortal(w http.ResponseWriter, r *http.Request) {
	h.portal(w, r, earn.TokenBilling)
}

func (h *Handler) cancelPortal(w http.ResponseWriter, r *http.Request) {
	h.portal(w, r, earn.TokenCancel)
}

// portal sends a paid member to Stripe's billing portal (its cancel step
// for the cancel link).
func (h *Handler) portal(w http.ResponseWriter, r *http.Request, kind linktoken.Kind) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil {
		h.notFound(w, r, s)
		return
	}
	token := chi.URLParam(r, "token")
	to, err := h.Earn.Portal(r.Context(), s.ws, token, kind)
	switch {
	case errors.Is(err, earn.ErrBadLink):
		h.notFound(w, r, s)
		return
	case errors.Is(err, earn.ErrNoSubscription):
		pl, r := h.earnLinkLang(w, r, s, token, kind)
		h.earnMessage(w, r, s, pl, http.StatusOK, "portal.noSubTitle", "portal.noSubText")
		return
	case errors.Is(err, earn.ErrUnavailable):
		pl, r := h.earnLinkLang(w, r, s, token, kind)
		h.earnMessage(w, r, s, pl, http.StatusServiceUnavailable, "page.unavailableTitle", "portal.unavailableText")
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	h.toProvider(w, r, s, to)
}

// toProvider sends the reader to the payment provider's page: the address
// comes from the provider's API (a checkout or portal session), never from
// the request, and must be an absolute https address.
func (h *Handler) toProvider(w http.ResponseWriter, r *http.Request, s *site, to string) {
	u, err := url.Parse(to)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		h.serverError(w, r, s, errors.New("publicpages: the payment provider's address isn't https"))
		return
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther) //nolint:gosec // G710: the provider's session address, checked above
}

// earnMessage is a one-message Earn page in its language: heading and text
// are internal/earn's messages.
func (h *Handler) earnMessage(w http.ResponseWriter, r *http.Request, s *site, pl pageLang, status int, heading, text string) {
	l := pl.Labels.Lang()
	head := earn.TextIn(l, heading, nil)
	h.messageIn(w, r, s, pl, status, h.earnTitle(s, head), messageBody{Heading: head, Text: earn.TextIn(l, text, nil),
		Actions: []action{{Label: earn.TextIn(l, "portal.preferences", nil), Href: s.links.PreferencesRequest()}}}, true)
}

// EarnRender renders an Earn page for the app (the upgrade page editor's
// preview): inert, with the instance's assets.
func (h *Handler) EarnRender(ctx context.Context, ws *store.Workspace, page string, data any) (string, error) {
	if err := h.Init(); err != nil {
		return "", err
	}
	set, err := h.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return "", err
	}
	s := &site{ws: ws, set: set, links: forms.LinksOn(h.URLs.Base(), ws.Slug, false), canon: h.Forms.Links(ws, set), preview: true}
	var v *view
	switch page {
	case "upgrade":
		// The page in the language its view was written in (the workspace's
		// default language).
		uv, _ := data.(earn.UpgradeView)
		pl := h.fixedLang(ctx, s, string(uv.Lang.Locale), func(string) bool { return true })
		v = h.earnView(nil, s, h.earnTitle(s, uv.Headline), upgradeBody{View: uv, Action: "#", ArchiveHref: s.links.Archive(), TermsURL: uv.TermsURL,
			Upgrading: earn.TextIn(uv.Lang, "page.upgrading", nil)}, pl)
	default:
		return "", errors.New("publicpages: no Earn page " + page)
	}
	v.Preview = true
	out, err := h.html(page, v)
	return string(out), err
}
