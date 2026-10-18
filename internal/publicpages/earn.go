package publicpages

// The Earn pages (design expansion phase 6, X16): the upgrade page with its
// tiers and Stripe Checkout, the welcome page after checkout, and the
// billing portal links of member emails and the preference center. Their
// words come from internal/earn (English until the pages are translated);
// the branding is the publisher's.
//
//	/p/{workspace}/upgrade[/{tier}]       the upgrade page (a hidden tier's own link)
//	/p/{workspace}/upgrade/checkout       POST: opens Stripe Checkout
//	/p/{workspace}/upgrade/welcome        after checkout (?session=)
//	/p/{workspace}/billing/{token}        a paid member's billing portal
//	/p/{workspace}/cancel/{token}         the portal's cancel step (the cancel page comes with offers)

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/pagelabels"
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
	r.Get("/cancel/{token}", h.cancelPortal)
	r.Head("/cancel/{token}", h.headOnly)
}

// earnRootPaths are the Earn pages on a verified custom domain.
var earnRootPaths = []string{"/upgrade", "/upgrade/*", "/billing/*", "/cancel/*"}

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

// earnView is a view of an Earn page: its words are English.
func (h *Handler) earnView(s *site, title string, body any) *view {
	v := h.view(s, title, body)
	v.Lang, v.Dir, v.labels = "en", "ltr", pagelabels.For("en", nil)
	return v
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
	q := r.URL.Query()
	req := earn.UpgradeRequest{Token: q.Get("r"), Tier: chi.URLParam(r, "tier"), Period: q.Get("period"), Canceled: q.Get("canceled") == "1",
		TestToken: q.Get("test"), Lang: earn.PageLang}
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
	h.renderUpgrade(w, r, s, v, req, http.StatusOK)
}

func (h *Handler) renderUpgrade(w http.ResponseWriter, r *http.Request, s *site, v earn.UpgradeView, req earn.UpgradeRequest, status int) {
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
		YearHref: period("year"), ArchiveHref: s.links.Archive(), TermsURL: v.TermsURL, Upgrading: earn.Text("page.upgrading", nil), Retry: r.URL.String()}
	vw := h.earnView(s, h.earnTitle(s, v.Headline), body)
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
	req := earn.CheckoutRequest{Tier: r.PostForm.Get("tier"), Period: r.PostForm.Get("period"), Token: r.PostForm.Get("r"), TestToken: r.PostForm.Get("test")}
	to, err := h.Earn.StartCheckout(r.Context(), s.ws, req)
	if err != nil {
		if !errors.Is(err, earn.ErrUnavailable) {
			h.log().ErrorContext(r.Context(), "upgrade checkout", "err", err)
		}
		up := earn.UpgradeRequest{Token: req.Token, Period: req.Period, TestToken: req.TestToken, Lang: earn.PageLang}
		v, verr := h.Earn.Upgrade(r.Context(), s.ws, up)
		if verr != nil {
			h.serverError(w, r, s, verr)
			return
		}
		v.Notice = &earn.PageNotice{Tone: "bad", Title: earn.Text("page.unavailableTitle", nil), Text: earn.Text("page.unavailableText", nil), Retry: true}
		h.renderUpgrade(w, r, s, v, up, http.StatusServiceUnavailable)
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
	v, err := h.Earn.Welcome(r.Context(), s.ws, r.URL.Query().Get("session"), earn.PageLang)
	if errors.Is(err, earn.ErrUnavailable) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	vw := h.earnView(s, h.earnTitle(s, v.Heading), welcomeBody{WelcomeView: v})
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
	to, err := h.Earn.Portal(r.Context(), s.ws, chi.URLParam(r, "token"), kind)
	switch {
	case errors.Is(err, earn.ErrBadLink):
		h.notFound(w, r, s)
		return
	case errors.Is(err, earn.ErrNoSubscription):
		h.earnMessage(w, r, s, http.StatusOK, earn.Text("portal.noSubTitle", nil), earn.Text("portal.noSubText", nil))
		return
	case errors.Is(err, earn.ErrUnavailable):
		h.earnMessage(w, r, s, http.StatusServiceUnavailable, earn.Text("page.unavailableTitle", nil), earn.Text("portal.unavailableText", nil))
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

func (h *Handler) earnMessage(w http.ResponseWriter, r *http.Request, s *site, status int, heading, text string) {
	pl := h.fixedLang(r.Context(), s, "en", nil)
	h.messageIn(w, r, s, pl, status, h.earnTitle(s, heading), messageBody{Heading: heading, Text: text,
		Actions: []action{{Label: earn.Text("portal.preferences", nil), Href: s.links.PreferencesRequest()}}}, true)
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
		uv, _ := data.(earn.UpgradeView)
		v = h.earnView(s, h.earnTitle(s, uv.Headline), upgradeBody{View: uv, Action: "#", ArchiveHref: s.links.Archive(), TermsURL: uv.TermsURL, Upgrading: earn.Text("page.upgrading", nil)})
	default:
		return "", errors.New("publicpages: no Earn page " + page)
	}
	v.Preview = true
	out, err := h.html(page, v)
	return string(out), err
}
