package publicpages

// The gift pages (design expansion phase 6, X17): the gift page sells a
// tier for a fixed period with one payment through the provider's
// checkout; the page after paying says when the gift arrives; the redeem
// page lets the recipient start it without a card.
//
//	/p/{workspace}/gift                  the gift page; POST opens checkout
//	/p/{workspace}/gift/thanks           after checkout (?session=)
//	/p/{workspace}/gift/redeem/{code}    the redeem page; POST redeems

import (
	"errors"
	"html"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/gifts"
)

// giftRoutes adds the gift pages to a workspace's page routes.
func (h *Handler) giftRoutes(r chi.Router) {
	r.Get("/gift", h.giftPage)
	r.Head("/gift", h.headOnly)
	r.Post("/gift", h.giftBuy)
	r.Get("/gift/thanks", h.giftThanks)
	r.Get("/gift/redeem/{code}", h.redeemPage)
	r.Head("/gift/redeem/{code}", h.headOnly)
	r.Post("/gift/redeem/{code}", h.redeemPost)
}

// giftBody is the gift page.
type giftBody struct {
	gifts.PageView
	Action   string
	TermsURL string
	// TextHTML is the off page's text with the support address linked.
	TextHTML template.HTML
}

// giftDoneBody is the page after paying.
type giftDoneBody struct {
	gifts.ThanksView
	TermsURL   string
	TermsLabel string
}

// redeemBody is the redeem page.
type redeemBody struct {
	gifts.RedeemView
	Action      string
	Preferences string
	TermsURL    string
	TermsLabel  string
	TextHTML    template.HTML
}

// mailtoHTML escapes a sentence and links the address in it.
func mailtoHTML(text, email string) template.HTML {
	esc := html.EscapeString(text)
	if email == "" {
		return template.HTML(esc) //nolint:gosec // escaped above
	}
	e := html.EscapeString(email)
	link := `<a href="mailto:` + e + `">` + e + `</a>`
	return template.HTML(strings.Replace(esc, e, link, 1)) //nolint:gosec // escaped above; the address is escaped too
}

func (h *Handler) giftPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Gifts == nil {
		h.notFound(w, r, s)
		return
	}
	pl, r := h.earnLang(w, r, s)
	v, err := h.Gifts.Page(r.Context(), s.ws, gifts.BuyInput{}, r.URL.Query().Get("canceled") == "1")
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if r.Method == http.MethodGet && h.Earn != nil && v.State == "form" {
		h.Earn.Visit(r.Context(), s.ws, "gift")
	}
	h.renderGift(w, r, s, v, http.StatusOK, pl)
}

func (h *Handler) renderGift(w http.ResponseWriter, r *http.Request, s *site, v gifts.PageView, status int, pl pageLang) {
	b := giftBody{PageView: v, Action: earnLink(s.links, "gift"), TermsURL: h.giftTerms(r, s)}
	if v.State == "off" {
		b.TextHTML = mailtoHTML(v.Text, v.Support)
	}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), b, pl)
	vw.Canonical = earnLink(s.canon, "gift")
	vw.Scripts = []string{s.links.Static("gift.js?v=" + h.version)}
	// Paying is Stripe's page: the form's redirect goes there.
	h.render(w, r, s, "gift", status, vw, csp{formAction: []string{"https://checkout.stripe.com"}}, v.State != "form")
}

// giftTerms is the terms of sale's address.
func (h *Handler) giftTerms(r *http.Request, s *site) string {
	if h.Earn == nil {
		return ""
	}
	return h.Earn.PageSettingsOf(r.Context(), s.ws.ID).TermsURL
}

func (h *Handler) giftBuy(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Gifts == nil || !h.parseForm(w, r, 8<<10) {
		h.notFound(w, r, s)
		return
	}
	pl, r := h.earnLang(w, r, s)
	f := r.PostForm
	in := gifts.BuyInput{Option: f.Get("option"), To: f.Get("to"), RecipientName: f.Get("recipient_name"), RecipientEmail: f.Get("recipient_email"),
		When: f.Get("when"), Date: f.Get("date"), Time: f.Get("time"), Timezone: f.Get("tz"), Message: f.Get("message"), BuyerName: f.Get("buyer_name"),
		BuyerEmail: f.Get("buyer_email")}
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	// Bots fill the invisible field, and nobody buys ten gifts a minute.
	if f.Get("website") != "" || !h.Forms.AllowPerMinute("gift-ip:"+ip, 10) {
		v, err := h.Gifts.Page(r.Context(), s.ws, in, false)
		if err != nil {
			h.serverError(w, r, s, err)
			return
		}
		w.Header().Set("Retry-After", "60")
		v.Alert = earn.TextIn(pl.Labels.Lang(), "page.unavailableText", nil)
		h.renderGift(w, r, s, v, http.StatusTooManyRequests, pl)
		return
	}
	to, v, err := h.Gifts.Buy(r.Context(), s.ws, in)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if v != nil {
		status := http.StatusUnprocessableEntity
		if v.State == "off" {
			status = http.StatusOK
		}
		h.renderGift(w, r, s, *v, status, pl)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	h.toProvider(w, r, s, to)
}

func (h *Handler) giftThanks(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Gifts == nil {
		h.notFound(w, r, s)
		return
	}
	pl, r := h.earnLang(w, r, s)
	v, err := h.Gifts.Thanks(r.Context(), s.ws, r.URL.Query().Get("session"))
	if errors.Is(err, gifts.ErrUnavailable) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), giftDoneBody{ThanksView: v, TermsURL: h.giftTerms(r, s), TermsLabel: gifts.TextIn(pl.Labels.Lang(), "page.terms", nil)}, pl)
	h.render(w, r, s, "giftdone", http.StatusOK, vw, csp{}, true)
}

func (h *Handler) redeemPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Gifts == nil {
		h.notFound(w, r, s)
		return
	}
	pl, r := h.earnLang(w, r, s)
	code := chi.URLParam(r, "code")
	v, err := h.Gifts.RedeemPage(r.Context(), s.ws, code)
	if errors.Is(err, gifts.ErrNoGift) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.renderRedeem(w, r, s, v, code, http.StatusOK, pl)
}

func (h *Handler) renderRedeem(w http.ResponseWriter, r *http.Request, s *site, v gifts.RedeemView, code string, status int, pl pageLang) {
	b := redeemBody{RedeemView: v, Action: earnLink(s.links, "gift/redeem/"+gifts.NormalizeCode(code)), Preferences: s.links.PreferencesRequest(),
		TermsURL: h.giftTerms(r, s), TermsLabel: gifts.TextIn(pl.Labels.Lang(), "page.terms", nil)}
	switch v.State {
	case "used", "expired", "refunded":
		b.TextHTML = mailtoHTML(v.Text, v.Support)
	}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), b, pl)
	if v.State == "have" {
		vw.Scripts = []string{s.links.Static("gift.js?v=" + h.version)}
	}
	// A gift link is personal: never indexed, never cached.
	w.Header().Set("Referrer-Policy", "no-referrer")
	h.render(w, r, s, "redeem", status, vw, csp{}, true)
}

func (h *Handler) redeemPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Gifts == nil || !h.parseForm(w, r, 4<<10) {
		h.notFound(w, r, s)
		return
	}
	pl, r := h.earnLang(w, r, s)
	code := chi.URLParam(r, "code")
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	if !h.Forms.AllowPerMinute("gift-redeem-ip:"+ip, 10) {
		v, err := h.Gifts.RedeemPage(r.Context(), s.ws, code)
		if err != nil {
			h.notFound(w, r, s)
			return
		}
		w.Header().Set("Retry-After", "60")
		v.Error = earn.TextIn(pl.Labels.Lang(), "page.unavailableText", nil)
		h.renderRedeem(w, r, s, v, code, http.StatusTooManyRequests, pl)
		return
	}
	v, err := h.Gifts.Redeem(r.Context(), s.ws, code, gifts.RedeemInput{Email: r.PostForm.Get("email"), IP: clientIP(r), UserAgent: r.UserAgent()})
	if errors.Is(err, gifts.ErrNoGift) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	status := http.StatusOK
	if v.Error != "" {
		status = http.StatusUnprocessableEntity
	}
	h.renderRedeem(w, r, s, v, code, status, pl)
}
