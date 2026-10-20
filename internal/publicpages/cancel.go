package publicpages

// The hosted cancel page (design expansion phase 6, X17): a paid member's
// personal cancel link shows what happens, an optional reason and the
// retention offer they're eligible for; "Continue to cancel" opens Stripe's
// portal on the subscription's cancel step, and the portal returns here.
//
//	/p/{workspace}/cancel/{token}         the page; POST takes the answer
//	/p/{workspace}/cancel/{token}?done=1  back from the portal

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/offers"
)

// cancelBody is the cancel page.
type cancelBody struct {
	offers.CancelView
	Action      string
	NotYou      string
	Preferences string
	TermsURL    string
	// The expired link's form asks for a new link by email.
	RequestAction string
	RequestToken  string
}

func (h *Handler) cancelBody(ctx context.Context, s *site, v offers.CancelView, token string) cancelBody {
	b := cancelBody{CancelView: v, Action: earnLink(s.links, "cancel/"+token), NotYou: s.links.PreferencesRequest(), Preferences: s.links.PreferencesRequest(),
		RequestAction: s.links.PreferencesRequest(), RequestToken: h.Forms.FormToken("prefs:"+s.ws.ID.String(), h.now())}
	if h.Earn != nil {
		b.TermsURL = h.Earn.PageSettingsOf(ctx, s.ws.ID).TermsURL
	}
	return b
}

func (h *Handler) renderCancel(w http.ResponseWriter, r *http.Request, s *site, b cancelBody, status int) {
	vw := h.earnView(s, h.earnTitle(s, b.Heading), b)
	// Continue to cancel goes to Stripe's portal.
	h.render(w, r, s, "cancel", status, vw, csp{formAction: []string{"https://billing.stripe.com"}}, true)
}

func (h *Handler) cancelPage(w http.ResponseWriter, r *http.Request) {
	if h.Offers == nil {
		h.cancelPortal(w, r)
		return
	}
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	v, err := h.Offers.CancelPage(r.Context(), s.ws, token, r.URL.Query().Get("done") == "1")
	switch {
	case errors.Is(err, earn.ErrLinkInvalid):
		h.notFound(w, r, s)
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	}
	status := http.StatusOK
	if v.State == "expired" {
		status = http.StatusGone
	}
	h.renderCancel(w, r, s, h.cancelBody(r.Context(), s, v, token), status)
}

func (h *Handler) cancelPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Offers == nil || !h.parseForm(w, r, 8<<10) {
		h.notFound(w, r, s)
		return
	}
	token := chi.URLParam(r, "token")
	f := r.PostForm
	back := h.Earn.PageURL(r.Context(), s.ws, "cancel/"+token+"?done=1")
	res, err := h.Offers.CancelDecide(r.Context(), s.ws, token, f.Get("visit"), f.Get("reason"), f.Get("decision"), back)
	switch {
	case errors.Is(err, earn.ErrLinkInvalid):
		h.notFound(w, r, s)
		return
	case errors.Is(err, earn.ErrUnavailable):
		h.earnMessage(w, r, s, http.StatusServiceUnavailable, earn.Text("page.unavailableTitle", nil), earn.Text("portal.unavailableText", nil))
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	}
	if res.Redirect != "" {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		h.toProvider(w, r, s, res.Redirect)
		return
	}
	h.renderCancel(w, r, s, h.cancelBody(r.Context(), s, *res.View, token), http.StatusOK)
}
