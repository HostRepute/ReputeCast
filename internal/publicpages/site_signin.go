package publicpages

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/website"
)

// Signing in and signing up on the website (PublicSiteStates): no
// passwords, a link and a 6-digit code by email. Opening a link shows a
// page with Continue (site.js continues by itself): link scanners that
// fetch the address don't use it up. A sign-in link's page never sends
// its address as a referrer.

// FlashCookie carries a one-time toast to the next page of the site.
const FlashCookie = "rc_flash"

// cookiePath is the path the site's cookies are scoped to.
func (s *site) cookiePath() string {
	if s.links.Custom {
		return "/"
	}
	return "/p/" + url.PathEscape(s.ws.Slug) + "/"
}

func (s *site) secure(r *http.Request) bool {
	return r.TLS != nil || strings.HasPrefix(s.links.Origin(), "https://")
}

// setCookie sets one of the site's first-party cookies (0 maxAge: for the
// browser session; negative: removes it).
func (h *Handler) setCookie(w http.ResponseWriter, r *http.Request, s *site, name, value string, maxAge time.Duration) {
	c := &http.Cookie{Name: name, Value: value, Path: s.cookiePath(), HttpOnly: true, Secure: s.secure(r), SameSite: http.SameSiteLaxMode} //nolint:gosec // G124: Secure follows the site's origin, plain http only in development
	switch {
	case maxAge < 0:
		c.MaxAge = -1
		c.Value = ""
	case maxAge > 0:
		c.MaxAge = int(maxAge / time.Second)
		c.Expires = h.now().Add(maxAge)
	}
	http.SetCookie(w, c)
}

// browserCookie is the browser's sign-in cookie, made when it has none:
// a code is typed in the browser that asked for it.
func (h *Handler) browserCookie(w http.ResponseWriter, r *http.Request, s *site) string {
	if v := cookieValue(r, website.BrowserCookie); website.BrowserKey(v) != "" {
		return v
	}
	v := website.Token()
	h.setCookie(w, r, s, website.BrowserCookie, v, 24*time.Hour)
	return v
}

// flash sets the toast of the next page.
func (h *Handler) flash(w http.ResponseWriter, r *http.Request, s *site, kind string) {
	h.setCookie(w, r, s, FlashCookie, kind, 2*time.Minute)
}

// takeFlash reads and clears the toast for this page.
func (h *Handler) takeFlash(w http.ResponseWriter, r *http.Request, s *site, v *siteView) {
	kind := cookieValue(r, FlashCookie)
	if kind == "" {
		return
	}
	h.setCookie(w, r, s, FlashCookie, "", -1)
	email := ""
	if v.Member != nil {
		email = v.Member.Email
	}
	switch kind {
	case "signed_in":
		v.Toast = v.T("signin.signedIn", "email", email)
	case "welcome":
		v.Toast = v.T("signin.welcome", "email", email)
	case "email_changed":
		v.Toast = v.T("signin.emailChanged", "email", email)
	case "email_taken":
		v.Toast = v.T("signin.emailTaken")
	case "signed_out":
		v.Toast = v.T("signin.signedOut")
	case "everywhere":
		v.Toast = v.T("signin.signedOutEverywhere")
	case "name":
		v.Toast = v.T("account.nameSaved")
	case "newsletters":
		v.Toast = v.T("account.newslettersSaved")
	case "unsubscribed":
		v.Toast = v.T("account.unsubscribed")
	case "purchase_link":
		// A purchase's new download link (sell_account.go).
		v.Toast = v.S("account.sent", "email", products.MaskEmail(email))
	}
	if v.Toast != "" {
		v.private = true
	}
}

// digitView is a box of the code.
type digitView struct{ Name, Label string }

// signinBody is the sign-in page in one of its states.
type signinBody struct {
	Mode    string
	Heading string
	Lead    template.HTML
	Error   string
	// The email form.
	Action, ReturnTo, Email, EmailLabel, Continue string
	NotMember, SubscribeHref, SubscribeText       string
	// The code form.
	CodeAction, CodeLegend, Submit string
	Digits                         []digitView
	CodeError                      string
	ResendAction, ResendLabel      string
	ResendIn                       int
	ResendInLabel                  string
	OtherHref, OtherText           string
	// A link's page: Continue posts the link; Resend sends a new one.
	FinishAction string
	Auto         bool
	Resend       *resendView
	Actions      []linkAction
	Note         string
	// Turnstile.
	SiteKey string
	Lists   []string
}

// resendView is "Send a new link": the address again, posted to /signin.
type resendView struct {
	Action, Email, ReturnTo, Label string
	Primary                        bool
}

// signinView is the frame of a sign-in page: private, never indexed.
func (h *Handler) signinView(w http.ResponseWriter, r *http.Request, s *site, title string, b signinBody) (*siteView, bool) {
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return nil, false
	}
	v.Title = v.T("site.title", "page", title, "site", v.Frame.Title)
	v.Robots = "noindex, nofollow"
	v.private, v.secret = true, true
	v.Body = b
	return v, true
}

func (h *Handler) renderSignin(w http.ResponseWriter, r *http.Request, s *site, status int, title string, b signinBody) {
	v, ok := h.signinView(w, r, s, title, b)
	if !ok {
		return
	}
	if b.SiteKey != "" {
		v.Scripts = append(v.Scripts, forms.TurnstileScript)
		v.cspScripts, v.cspFrames = []string{forms.TurnstileOrigin}, []string{forms.TurnstileOrigin}
	}
	h.renderSite(w, r, s, "signin", status, v)
}

// signinForm is the email form.
func (h *Handler) signinForm(r *http.Request, s *site, v *siteView, back, email, problem string) signinBody {
	return signinBody{Mode: "form", Heading: v.T("signin.title"), Lead: template.HTML(template.HTMLEscapeString(v.T("signin.lead"))), //nolint:gosec // escaped
		Action: s.path("signin"), ReturnTo: back, Email: email, EmailLabel: v.T("signin.email"), Continue: v.T("signin.continue"), Error: problem,
		NotMember: v.T("signin.notMember"), SubscribeHref: s.links.Subscribe(""), SubscribeText: v.T("signin.subscribeFree")}
}

// returnOf is where a sign-in leads back to: the ?return= or posted path
// when it is a page of this site, else the home page.
func (h *Handler) returnOf(s *site, raw string) string {
	p := website.SafeReturn(raw)
	home := s.returnPath(s.path())
	if p == "" {
		return home
	}
	base := strings.TrimSuffix(home, "/")
	if base != "" && !strings.HasPrefix(p, base+"/") && p != base {
		return home
	}
	return p
}

// signinPage is /signin.
func (h *Handler) signinPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.siteOn(r.Context(), s) {
		h.siteNotFound(w, r, s)
		return
	}
	back := h.returnOf(s, r.URL.Query().Get("return"))
	if sess := h.session(r, s); sess != nil {
		http.Redirect(w, r, back, http.StatusSeeOther) //nolint:gosec // G710: returnOf keeps only paths on this site
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	if !v.Frame.SigninOn {
		h.renderSignin(w, r, s, http.StatusOK, v.T("signin.title"), signinBody{Mode: "message", Heading: v.T("signin.off"),
			Lead: template.HTML(template.HTMLEscapeString(v.T("signin.offText"))), Actions: []linkAction{{Label: v.T("signin.subscribeFree"), Href: s.links.Subscribe(""), Primary: true}}}) //nolint:gosec // escaped
		return
	}
	h.renderSignin(w, r, s, http.StatusOK, v.T("signin.title"), h.signinForm(r, s, v, back, "", ""))
}

// signinPost sends a sign-in link and a code.
func (h *Handler) signinPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.parseForm(w, r, 8<<10) {
		h.siteNotFound(w, r, s)
		return
	}
	if r.PostForm.Get("list_mode") == "join" {
		// The Turnstile step of a sign-up from a page of the site.
		h.join(w, r, s, h.returnOf(s, r.PostForm.Get("return")), func(*signupProblem) { h.signinPage(w, r) })
		return
	}
	ctx := r.Context()
	back := h.returnOf(s, r.PostForm.Get("return"))
	email := strings.TrimSpace(r.PostForm.Get("email"))
	browser := h.browserCookie(w, r, s)
	ip := clientIP(r)
	remote := ""
	if ip != nil {
		remote = ip.String()
	}
	out, err := h.Website.RequestSignin(ctx, s.ws, website.SigninInput{Email: email, ReturnTo: back, Browser: browser, IP: ip,
		Turnstile: r.PostForm.Get("cf-turnstile-response"), Verify: h.Forms.TurnstileVerifier(ctx, s.set, remote)})
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	switch out.Status {
	case website.SigninOff:
		h.signinPage(w, r)
	case website.SigninInvalid:
		h.renderSignin(w, r, s, http.StatusUnprocessableEntity, v.T("signin.title"), h.signinForm(r, s, v, back, email, out.Message))
	case website.SigninLimited:
		h.signinLimited(w, r, s, out, back)
	case website.SigninChallenge:
		h.signinChallenge(w, r, s, back, email, nil)
	default:
		h.renderSignin(w, r, s, http.StatusOK, v.T("signin.checkTitle"), h.checkBody(s, v, out.Email, back, ""))
	}
}

// checkBody is "Check your email" with the code form.
func (h *Handler) checkBody(s *site, v *siteView, email, back, codeError string) signinBody {
	b := signinBody{Mode: "check", Heading: v.T("signin.checkTitle"), Lead: v.H("signin.checkText", "email", email), CodeAction: s.path("signin", "code"),
		CodeLegend: v.T("signin.otherDevice"), Submit: v.T("signin.submit"), CodeError: codeError, ReturnTo: back, Email: email,
		ResendAction: s.path("signin"), ResendLabel: v.T("signin.resend"), ResendIn: int(website.ResendAfter / time.Second),
		ResendInLabel: v.T("signin.resendIn", "time", "1:00"), OtherHref: s.path("signin") + "?" + url.Values{"return": {back}}.Encode(), OtherText: v.T("signin.otherEmail")}
	for i := 1; i <= 6; i++ {
		b.Digits = append(b.Digits, digitView{Name: fmt.Sprintf("d%d", i), Label: v.T("signin.digit", "n", i)})
	}
	if codeError != "" {
		b.ResendIn = 0
	}
	return b
}

// signinLimited is "Too many links".
func (h *Handler) signinLimited(w http.ResponseWriter, r *http.Request, s *site, out website.SigninOutcome, back string) {
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	mins := int(out.RetryIn.Round(time.Minute) / time.Minute)
	text := v.T("signin.limitIP")
	if out.Email != "" && out.RetryIn > 0 && out.RetryIn < time.Hour+time.Minute {
		text = v.T("signin.limitText", "minutes", max(mins, 1))
	}
	b := signinBody{Mode: "message", Heading: v.T("signin.limitTitle"), Error: text, Lead: template.HTML(template.HTMLEscapeString(v.T("signin.limitNote")))} //nolint:gosec // escaped
	label := v.T("signin.home")
	if strings.Contains(back, "/posts/") {
		label = v.T("signin.backToPost")
	}
	b.Actions = []linkAction{{Label: label, Href: back}}
	w.Header().Set("Retry-After", fmt.Sprint(int(max(out.RetryIn, time.Minute)/time.Second)))
	h.renderSignin(w, r, s, http.StatusTooManyRequests, v.T("signin.limitTitle"), b)
}

// signinChallenge asks Turnstile whether the reader is a person, then
// posts the same request again.
func (h *Handler) signinChallenge(w http.ResponseWriter, r *http.Request, s *site, back, email string, lists []uuid.UUID) {
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	b := signinBody{Mode: "challenge", Heading: v.T("signin.challengeTitle"), Lead: template.HTML(template.HTMLEscapeString(v.T("signin.challengeText"))), //nolint:gosec // escaped
		Action: s.path("signin"), ReturnTo: back, Email: email, Continue: v.T("signin.continue"), SiteKey: s.set.TurnstileSiteKey()}
	for _, l := range lists {
		b.Lists = append(b.Lists, l.String())
	}
	if lists != nil || r.PostForm.Get("list_mode") == "join" {
		b.Mode = "challenge_join"
	}
	h.renderSignin(w, r, s, http.StatusOK, v.T("signin.challengeTitle"), b)
}

// joinSent is "Check your email" after a sign-up.
func (h *Handler) joinSent(w http.ResponseWriter, r *http.Request, s *site, email, page string) {
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	b := signinBody{Mode: "message", Heading: v.T("join.checkTitle"), Lead: v.H("join.checkText", "email", email)}
	label := v.T("signin.home")
	if strings.Contains(page, "/posts/") {
		label = v.T("signin.backToPost")
	}
	b.Actions = []linkAction{{Label: label, Href: page}}
	h.renderSignin(w, r, s, http.StatusOK, v.T("join.checkTitle"), b)
}

// signinCode checks the code typed in this browser.
func (h *Handler) signinCode(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.parseForm(w, r, 8<<10) {
		h.siteNotFound(w, r, s)
		return
	}
	code := r.PostForm.Get("code")
	if code == "" {
		for i := 1; i <= 6; i++ {
			code += r.PostForm.Get(fmt.Sprintf("d%d", i))
		}
	}
	back := h.returnOf(s, r.PostForm.Get("return"))
	out, err := h.Website.SigninWithCode(r.Context(), s.ws, cookieValue(r, website.BrowserCookie), code, h.evidence(r, "website"))
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	if out.Status == website.CodeOK && out.Done != nil {
		h.signedIn(w, r, s, out.Done)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	email := out.Email
	if email == "" {
		email = r.PostForm.Get("email")
	}
	if out.Status == website.CodeWrong {
		h.renderSignin(w, r, s, http.StatusUnprocessableEntity, v.T("signin.checkTitle"),
			h.checkBody(s, v, email, back, v.T("signin.codeWrong", "left", out.TriesLeft)))
		return
	}
	b := signinBody{Mode: "message", Heading: v.T("signin.expiredTitle"), Lead: template.HTML(template.HTMLEscapeString(v.T("signin.codeExpired")))} //nolint:gosec // escaped
	if email != "" {
		b.Resend = &resendView{Action: s.path("signin"), Email: email, ReturnTo: back, Label: v.T("signin.newLink"), Primary: true}
	}
	b.Actions = []linkAction{{Label: v.T("signin.otherEmail"), Href: s.path("signin") + "?" + url.Values{"return": {back}}.Encode()}}
	h.renderSignin(w, r, s, http.StatusGone, v.T("signin.expiredTitle"), b)
}

// signedIn sets the session's cookie and goes back where the reader was.
func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request, s *site, done *website.SigninDone) {
	h.setCookie(w, r, s, website.SessionCookie, done.Token, time.Until(done.ExpiresAt))
	h.setCookie(w, r, s, website.MeterCookie, "", -1)
	kind := "signed_in"
	switch {
	case done.Purpose == website.PurposeSignup:
		kind = "welcome"
	case done.Purpose == website.PurposeEmail && done.Taken:
		kind = "email_taken"
	case done.Purpose == website.PurposeEmail:
		kind = "email_changed"
	}
	h.flash(w, r, s, kind)
	to := h.returnOf(s, done.ReturnTo)
	if done.Purpose == website.PurposeEmail {
		to = s.returnPath(s.path("account"))
	}
	http.Redirect(w, r, to, http.StatusSeeOther) //nolint:gosec // G710: returnOf keeps only paths on this site
}

// signinLink is the page a sign-in link opens: nothing is used until
// Continue (site.js continues by itself in a browser).
func (h *Handler) signinLink(w http.ResponseWriter, r *http.Request) {
	h.linkPage(w, r, false)
}

// signinLinkPost uses the link.
func (h *Handler) signinLinkPost(w http.ResponseWriter, r *http.Request) {
	h.linkPage(w, r, true)
}

func (h *Handler) linkPage(w http.ResponseWriter, r *http.Request, use bool) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil {
		h.siteNotFound(w, r, s)
		return
	}
	ctx := r.Context()
	token := chi.URLParam(r, "token")
	browser := h.browserCookie(w, r, s)
	var st website.LinkState
	var err error
	if use {
		var done *website.SigninDone
		st, done, err = h.Website.SigninWithLink(ctx, s.ws, token, browser, h.evidence(r, "website"))
		if err == nil && done != nil {
			h.signedIn(w, r, s, done)
			return
		}
	} else {
		st, err = h.Website.Link(ctx, s.ws, token, browser)
	}
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	back := h.returnOf(s, st.ReturnTo)
	resend := &resendView{Action: s.path("signin"), Email: st.Email, ReturnTo: back, Label: v.T("signin.newLink"), Primary: true}
	other := linkAction{Label: v.T("signin.otherEmail"), Href: s.path("signin") + "?" + url.Values{"return": {back}}.Encode()}
	switch st.Status {
	case website.LinkOK:
		b := signinBody{Mode: "finish", Heading: v.T("signin.finishTitle"), Lead: v.H("signin.finishText", "email", st.Email),
			FinishAction: s.path("signin", "link", token), Continue: v.T("signin.continue"), Auto: true}
		if !strings.Contains(back, "/posts/") {
			b.Lead = v.H("signin.finishTextHome", "email", st.Email)
		}
		switch st.Purpose {
		case website.PurposeSignup:
			b.Heading, b.Lead = v.T("signin.finishSignup"), v.H("signin.finishSignupText", "email", st.Email)
		case website.PurposeEmail:
			b.Heading, b.Lead, b.Auto = v.T("signin.finishEmail"), v.H("signin.finishEmailText", "email", st.Email), false
		}
		h.renderSignin(w, r, s, http.StatusOK, b.Heading, b)
	case website.LinkExpired:
		b := signinBody{Mode: "message", Heading: v.T("signin.expiredTitle"), Lead: template.HTML(template.HTMLEscapeString( //nolint:gosec // escaped
			v.T("signin.expiredText", "minutes", st.Minutes, "email", st.Email))), Actions: []linkAction{other}}
		if st.Purpose != website.PurposeEmail {
			b.Resend = resend
		}
		h.renderSignin(w, r, s, http.StatusGone, b.Heading, b)
	case website.LinkUsed:
		resend.Label = v.T("signin.usedNew")
		b := signinBody{Mode: "message", Heading: v.T("signin.usedTitle"), Lead: template.HTML(template.HTMLEscapeString(v.T("signin.usedText"))), //nolint:gosec // escaped
			Resend: resend, Note: v.T("signin.usedNote")}
		h.renderSignin(w, r, s, http.StatusGone, b.Heading, b)
	case website.LinkReplaced:
		b := signinBody{Mode: "message", Heading: v.T("signin.replacedTitle"), Lead: template.HTML(template.HTMLEscapeString(v.T("signin.replacedText"))), //nolint:gosec // escaped
			Resend: resend, Actions: []linkAction{other}}
		h.renderSignin(w, r, s, http.StatusGone, b.Heading, b)
	default:
		b := signinBody{Mode: "message", Heading: v.T("signin.unknownTitle"), Lead: template.HTML(template.HTMLEscapeString(v.T("signin.unknownText"))), //nolint:gosec // escaped
			Actions: []linkAction{{Label: v.T("signin.newLink"), Href: s.path("signin"), Primary: true}}}
		h.renderSignin(w, r, s, http.StatusNotFound, b.Heading, b)
	}
}

// signout ends the reader's session on this browser.
func (h *Handler) signout(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website != nil {
		if sess := h.session(r, s); sess != nil {
			if err := h.Website.SignOut(r.Context(), s.ws, sess); err != nil {
				h.siteFail(w, r, s, err)
				return
			}
		}
	}
	h.setCookie(w, r, s, website.SessionCookie, "", -1)
	h.flash(w, r, s, "signed_out")
	http.Redirect(w, r, s.path(), http.StatusSeeOther) //nolint:gosec // G710: the site's home page
}
