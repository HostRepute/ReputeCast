package publicpages

import "net/http"

// The pages of the site that are still to come answer 404 until they're
// built (design expansion phase 7, X18).

func (h *Handler) siteStub(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	h.notFound(w, r, s)
}

func (h *Handler) sitePost(w http.ResponseWriter, r *http.Request)       { h.siteStub(w, r) }
func (h *Handler) sitePostSignup(w http.ResponseWriter, r *http.Request) { h.siteStub(w, r) }
func (h *Handler) siteTag(w http.ResponseWriter, r *http.Request)        { h.siteStub(w, r) }
func (h *Handler) siteTagFeed(w http.ResponseWriter, r *http.Request)    { h.siteStub(w, r) }
func (h *Handler) siteFeed(w http.ResponseWriter, r *http.Request)       { h.siteStub(w, r) }
func (h *Handler) signinPage(w http.ResponseWriter, r *http.Request)     { h.siteStub(w, r) }
func (h *Handler) signinPost(w http.ResponseWriter, r *http.Request)     { h.siteStub(w, r) }
func (h *Handler) signinCode(w http.ResponseWriter, r *http.Request)     { h.siteStub(w, r) }
func (h *Handler) signinLink(w http.ResponseWriter, r *http.Request)     { h.siteStub(w, r) }
func (h *Handler) signinLinkPost(w http.ResponseWriter, r *http.Request) { h.siteStub(w, r) }
func (h *Handler) signout(w http.ResponseWriter, r *http.Request)        { h.siteStub(w, r) }
func (h *Handler) accountPage(w http.ResponseWriter, r *http.Request)    { h.siteStub(w, r) }
func (h *Handler) accountPost(w http.ResponseWriter, r *http.Request)    { h.siteStub(w, r) }

// siteFail is the site's error page.
func (h *Handler) siteFail(w http.ResponseWriter, r *http.Request, s *site, err error) {
	h.serverError(w, r, s, err)
}
