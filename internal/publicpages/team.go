package publicpages

// The team and pause pages (M65, G-38 and G-39). Their words come from
// internal/earn, in the page's language; the branding is the publisher's.
//
//	/p/{workspace}/upgrade/{tier}/team    a tier's team page; POST opens Stripe Checkout
//	/p/{workspace}/team/{token}           the buyer's manage page (a week); POST does one action
//	/p/{workspace}/team/join/{token}      an invite (two weeks); POST accepts it
//	/p/{workspace}/resume/{token}         a paused member's "Resume now"; POST resumes
//
// Every POST checks that it comes from the page itself (Origin and
// Sec-Fetch-Site) and is limited per link and per address; accepting an
// invite is a POST, so link scanners can't.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// crossOrigin refuses a state-changing request another site made a
// browser send (the Origin header, else Sec-Fetch-Site).
var crossOrigin = http.NewCrossOriginProtection()

// teamRoutes adds the team and pause pages to a workspace's page routes.
func (h *Handler) teamRoutes(r chi.Router) {
	r.Get("/upgrade/{tier}/team", h.teamPage)
	r.Head("/upgrade/{tier}/team", h.headOnly)
	r.Post("/upgrade/{tier}/team", h.teamCheckout)
	r.Get("/team/join/{token}", h.teamJoin)
	r.Head("/team/join/{token}", h.headOnly)
	r.Post("/team/join/{token}", h.teamAccept)
	r.Get("/team/{token}", h.teamManage)
	r.Head("/team/{token}", h.headOnly)
	r.Post("/team/{token}", h.teamAction)
	r.Get("/resume/{token}", h.resumePage)
	r.Head("/resume/{token}", h.headOnly)
	r.Post("/resume/{token}", h.resumePost)
}

// teamBody is a tier's team page.
type teamBody struct {
	View       earn.TeamView
	Action     string
	Token      string
	Test       string
	MonthHref  string
	YearHref   string
	BackHref   string
	TermsURL   string
	TermsLabel string
}

// teamManageBody is a team's manage page.
type teamManageBody struct {
	View       earn.TeamManageView
	Action     string
	Page       string
	TermsURL   string
	TermsLabel string
}

// teamJoinBody is an invite's page.
type teamJoinBody struct {
	View       earn.TeamJoinView
	Action     string
	TermsURL   string
	TermsLabel string
}

// resumeBody is a paused member's resume page.
type resumeBody struct {
	View        earn.ResumeView
	Action      string
	Preferences string
	BackLabel   string
	TermsURL    string
	TermsLabel  string
}

// samePage refuses a POST another site's page sent (403).
func (h *Handler) samePage(w http.ResponseWriter, r *http.Request) bool {
	if err := crossOrigin.Check(r); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return false
	}
	return true
}

// allowLink limits a link's POSTs per link and per address (an address
// sharing a link with others still gets its own budget).
func (h *Handler) allowLink(r *http.Request, kind, token string, perMinute int) bool {
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	sum := sha256.Sum256([]byte(token))
	return h.Forms.AllowPerMinute(kind+"-ip:"+ip, perMinute*2) && h.Forms.AllowPerMinute(kind+"-link:"+hex.EncodeToString(sum[:8]), perMinute)
}

// termsLabel is the footer's "Terms of sale" in the page's language.
func termsLabel(r *http.Request) string {
	return earn.TextIn(i18n.LangFrom(r.Context()), "page.terms", nil)
}

func (h *Handler) termsOf(r *http.Request, s *site) string {
	if h.Earn == nil {
		return ""
	}
	return h.Earn.PageSettingsOf(r.Context(), s.ws.ID).TermsURL
}

func (h *Handler) teamPage(w http.ResponseWriter, r *http.Request) {
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
	seats, _ := strconv.Atoi(q.Get("seats"))
	req := earn.TeamRequest{Tier: chi.URLParam(r, "tier"), Period: q.Get("period"), Seats: seats, Canceled: q.Get("canceled") == "1", TestToken: q.Get("test"),
		Token: q.Get("r"), Lang: pl.Labels.Lang()}
	v, err := h.Earn.TeamPage(r.Context(), s.ws, req)
	if errors.Is(err, earn.ErrUnavailable) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.renderTeam(w, r, s, v, req, http.StatusOK, pl)
}

func (h *Handler) renderTeam(w http.ResponseWriter, r *http.Request, s *site, v earn.TeamView, req earn.TeamRequest, status int, pl pageLang) {
	page := "upgrade/" + url.PathEscape(v.TierSlug) + "/team"
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
	back := earnLink(s.links, "upgrade")
	if req.Token != "" || req.TestToken != "" {
		q := url.Values{}
		if req.Token != "" {
			q.Set("r", req.Token)
		}
		if req.TestToken != "" {
			q.Set("test", req.TestToken)
		}
		back += "?" + q.Encode()
	}
	body := teamBody{View: v, Action: earnLink(s.links, page), Token: req.Token, Test: req.TestToken, MonthHref: period("month"), YearHref: period("year"),
		BackHref: back, TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r)}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), body, pl)
	vw.Description = v.Intro
	vw.Canonical = earnLink(s.canon, page)
	if req.Token == "" && req.TestToken == "" {
		vw.Robots = "index, follow"
	}
	vw.Scripts = []string{s.links.Static("team.js?v=" + h.version)}
	h.render(w, r, s, "team", status, vw, csp{formAction: []string{"https://checkout.stripe.com"}}, req.Token != "" || req.TestToken != "")
}

func (h *Handler) teamCheckout(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil || !h.parseForm(w, r, 16<<10) {
		h.notFound(w, r, s)
		return
	}
	if !h.samePage(w, r) {
		return
	}
	pl, r := h.earnLang(w, r, s)
	l := pl.Labels.Lang()
	f := r.PostForm
	tier := chi.URLParam(r, "tier")
	req := earn.TeamCheckoutRequest{Tier: tier, Period: f.Get("period"), Seats: f.Get("seats"), Team: f.Get("team"), Email: f.Get("email"), Token: f.Get("r"),
		TestToken: f.Get("test"), Lang: l}
	page := earn.TeamRequest{Tier: tier, Period: req.Period, TestToken: req.TestToken, Token: req.Token, Lang: l, SeatsText: req.Seats, Team: req.Team, Email: req.Email}
	page.Seats, _ = strconv.Atoi(req.Seats)
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	if f.Get("website") != "" || !h.Forms.AllowPerMinute("team-checkout-ip:"+ip, 10) {
		h.teamAgain(w, r, s, page, pl, http.StatusTooManyRequests, true)
		return
	}
	to, err := h.Earn.StartTeamCheckout(r.Context(), s.ws, req)
	var form *earn.TeamFormError
	switch {
	case errors.As(err, &form):
		page.Errors = form.Fields
		h.teamAgain(w, r, s, page, pl, http.StatusUnprocessableEntity, false)
		return
	case err != nil:
		if !errors.Is(err, earn.ErrUnavailable) {
			h.log().ErrorContext(r.Context(), "team checkout", "err", err)
		}
		h.teamAgain(w, r, s, page, pl, http.StatusServiceUnavailable, true)
		return
	}
	h.toProvider(w, r, s, to)
}

// teamAgain is the team page once more after its form: with the fields'
// errors, or with "isn't available right now".
func (h *Handler) teamAgain(w http.ResponseWriter, r *http.Request, s *site, req earn.TeamRequest, pl pageLang, status int, unavailable bool) {
	v, err := h.Earn.TeamPage(r.Context(), s.ws, req)
	if errors.Is(err, earn.ErrUnavailable) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if unavailable {
		v.Notice = &earn.PageNotice{Tone: "bad", Title: v.Words["unavailableTitle"], Text: v.Words["unavailableText"]}
	}
	h.renderTeam(w, r, s, v, req, status, pl)
}

func (h *Handler) teamManage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil {
		h.notFound(w, r, s)
		return
	}
	token := chi.URLParam(r, "token")
	pl, r := h.earnLinkLang(w, r, s, token, earn.TokenGroupManage)
	v, err := h.Earn.TeamManage(r.Context(), s.ws, token, pl.Labels.Lang(), r.URL.Query().Get("remove"))
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.renderTeamManage(w, r, s, token, v, earn.TeamPageStatus(v.State), pl)
}

func (h *Handler) renderTeamManage(w http.ResponseWriter, r *http.Request, s *site, token string, v earn.TeamManageView, status int, pl pageLang) {
	page := earnLink(s.links, "team/"+token)
	body := teamManageBody{View: v, Action: page, Page: page, TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r)}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), body, pl)
	vw.Scripts = []string{s.links.Static("team.js?v=" + h.version)}
	h.render(w, r, s, "teammanage", status, vw, csp{formAction: []string{"https://billing.stripe.com"}}, true)
}

// teamAction does one thing the manage page asks: invite, resend, remove,
// change the seats or open billing.
func (h *Handler) teamAction(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil || !h.parseForm(w, r, 32<<10) {
		h.notFound(w, r, s)
		return
	}
	if !h.samePage(w, r) {
		return
	}
	token := chi.URLParam(r, "token")
	pl, r := h.earnLinkLang(w, r, s, token, earn.TokenGroupManage)
	l := pl.Labels.Lang()
	if !h.allowLink(r, "team", token, 10) {
		h.earnMessage(w, r, s, pl, http.StatusTooManyRequests, "page.unavailableTitle", "team.word.unavailableText")
		return
	}
	ctx := r.Context()
	f := r.PostForm
	var (
		res earn.TeamResult
		err error
	)
	switch f.Get("action") {
	case "invite":
		res, err = h.Earn.TeamInvite(ctx, s.ws, token, f.Get("emails"), l)
	case "resend":
		res, err = h.Earn.TeamResend(ctx, s.ws, token, f.Get("seat"), l)
	case "remove":
		res, err = h.Earn.TeamRemove(ctx, s.ws, token, f.Get("seat"), l)
	case "seats":
		res, err = h.Earn.TeamSetSeats(ctx, s.ws, token, f.Get("seats"), l)
	case "billing":
		res, err = h.Earn.TeamBilling(ctx, s.ws, token, h.Earn.PageURL(ctx, s.ws, "team/"+token), l)
	default:
		var v earn.TeamManageView
		v, err = h.Earn.TeamManage(ctx, s.ws, token, l, "")
		res = earn.TeamResult{View: v}
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if res.Redirect != "" {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		h.toProvider(w, r, s, res.Redirect)
		return
	}
	h.renderTeamManage(w, r, s, token, res.View, earn.TeamPageStatus(res.View.State), pl)
}

func (h *Handler) teamJoin(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil {
		h.notFound(w, r, s)
		return
	}
	token := chi.URLParam(r, "token")
	pl, r := h.earnLang(w, r, s)
	v, err := h.Earn.TeamJoin(r.Context(), s.ws, token, pl.Labels.Lang())
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.renderTeamJoin(w, r, s, token, v, pl)
}

func (h *Handler) renderTeamJoin(w http.ResponseWriter, r *http.Request, s *site, token string, v earn.TeamJoinView, pl pageLang) {
	body := teamJoinBody{View: v, Action: earnLink(s.links, "team/join/"+token), TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r)}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), body, pl)
	h.render(w, r, s, "teamjoin", earn.TeamPageStatus(v.State), vw, csp{}, true)
}

func (h *Handler) teamAccept(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil || !h.parseForm(w, r, 4<<10) {
		h.notFound(w, r, s)
		return
	}
	if !h.samePage(w, r) {
		return
	}
	token := chi.URLParam(r, "token")
	pl, r := h.earnLang(w, r, s)
	if !h.allowLink(r, "team-join", token, 5) {
		h.earnMessage(w, r, s, pl, http.StatusTooManyRequests, "page.unavailableTitle", "team.word.unavailableText")
		return
	}
	in := earn.JoinInput{UserAgent: r.UserAgent()}
	if a := clientIP(r); a != nil {
		in.IP = a.String()
	}
	v, err := h.Earn.AcceptTeamInvite(r.Context(), s.ws, token, in, pl.Labels.Lang())
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.renderTeamJoin(w, r, s, token, v, pl)
}

func (h *Handler) resumePage(w http.ResponseWriter, r *http.Request) {
	h.resume(w, r, false)
}

func (h *Handler) resumePost(w http.ResponseWriter, r *http.Request) {
	h.resume(w, r, true)
}

// resume is a paused member's "Resume now" page; post resumes.
func (h *Handler) resume(w http.ResponseWriter, r *http.Request, post bool) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Earn == nil || (post && !h.parseForm(w, r, 4<<10)) {
		h.notFound(w, r, s)
		return
	}
	if post && !h.samePage(w, r) {
		return
	}
	token := chi.URLParam(r, "token")
	pl, r := h.earnLinkLang(w, r, s, token, earn.TokenBilling)
	l := pl.Labels.Lang()
	var (
		v   earn.ResumeView
		err error
	)
	if post {
		if !h.allowLink(r, "resume", token, 5) {
			h.earnMessage(w, r, s, pl, http.StatusTooManyRequests, "page.unavailableTitle", "team.word.unavailableText")
			return
		}
		v, err = h.Earn.ResumeFromLink(r.Context(), s.ws, token, l)
	} else {
		v, err = h.Earn.ResumePage(r.Context(), s.ws, token, l)
	}
	switch {
	case errors.Is(err, earn.ErrBadLink):
		h.notFound(w, r, s)
		return
	case errors.Is(err, earn.ErrLinkExpired):
		h.linkExpired(w, r, s, pl)
		return
	case errors.Is(err, earn.ErrUnavailable):
		h.earnMessage(w, r, s, pl, http.StatusServiceUnavailable, "page.unavailableTitle", "portal.unavailableText")
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	}
	body := resumeBody{View: v, Action: earnLink(s.links, "resume/"+token), Preferences: s.links.PreferencesRequest(),
		BackLabel: earn.TextIn(l, "portal.preferences", nil), TermsURL: h.termsOf(r, s), TermsLabel: termsLabel(r)}
	vw := h.earnView(r, s, h.earnTitle(s, v.Heading), body, pl)
	h.render(w, r, s, "resume", http.StatusOK, vw, csp{}, true)
}
