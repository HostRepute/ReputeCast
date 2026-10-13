// Package publicpages serves the pages subscribers see (spec 5.12, 7.12),
// server-rendered with html/template under the layout of internal/publicurl:
//
//	/p/{workspace}/subscribe[/{form}]      hosted subscribe page (GET, POST)
//	/p/{workspace}/confirm/{token}         double opt-in confirmation
//	/p/{workspace}/preferences[/{token}]   preference center (or ask for a link)
//	/p/{workspace}/unsubscribe/{token}     unsubscribe page; POST is RFC 8058 one-click
//	/p/{workspace}/view/{token}            an email in the browser
//	/p/{workspace}/archive[/{issue}]       public archive, archive.rss, sitemap.xml
//	/p/{workspace}/forms/{id}.js           embed script, /session and /submit
//	/p/_/…                                 stylesheet, fonts, scripts
//
// A workspace's verified custom domain serves the same pages at its root
// (news.acme.io/subscribe). The pages work without JavaScript and cookies,
// load no third-party code (Cloudflare Turnstile only on a challenge page),
// run under a strict CSP without inline scripts or styles, and carry full
// metadata; every page reached through a signed link is noindex.
package publicpages

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// Handler serves the public pages.
type Handler struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Forms    *forms.Service
	Audience *audience.Service
	Tokens   *linktoken.Signer
	URLs     publicurl.Builder
	Log      *slog.Logger
	// SPA answers root paths on hosts that aren't a custom domain, as before.
	SPA http.Handler
	// LandingURL is where "Sent with ReputeCast" points.
	LandingURL string
	Now        func() time.Time

	tmpl    map[string]*template.Template
	version string
}

func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *Handler) log() *slog.Logger {
	if h.Log != nil {
		return h.Log
	}
	return slog.Default()
}

// pages are the templates, each rendered inside layout.html.
var pages = []string{"subscribe", "confirm", "preferences", "request", "unsubscribe", "done", "message", "archive", "challenge"}

// Init parses the templates; Routes calls it.
func (h *Handler) Init() error {
	if h.tmpl != nil {
		return nil
	}
	funcs := template.FuncMap{"add": func(a, b int) int { return a + b }}
	h.tmpl = map[string]*template.Template{}
	for _, p := range pages {
		t, err := template.New("layout.html").Funcs(funcs).ParseFS(templateFS, "templates/layout.html", "templates/"+p+".html")
		if err != nil {
			return fmt.Errorf("publicpages: template %s: %w", p, err)
		}
		h.tmpl[p] = t
	}
	css, _ := fs.ReadFile(staticFS, "static/public.css")
	sum := sha256.Sum256(css)
	h.version = hex.EncodeToString(sum[:6])
	if h.LandingURL == "" {
		h.LandingURL = "https://reputecast.com"
	}
	return nil
}

// Routes mounts the public pages, their assets and the custom-domain root.
func (h *Handler) Routes(r chi.Router) {
	if err := h.Init(); err != nil {
		panic(err)
	}
	r.Get("/p/_/fonts/{file}", h.font)
	r.Get("/p/_/tls-check", h.tlsCheck)
	r.Get("/p/_/{file}", h.static)
	r.Route("/p/{ws}", func(r chi.Router) { h.pageRoutes(r) })
	// On a verified custom domain the same pages answer at the root.
	root := chi.NewRouter()
	h.pageRoutes(root)
	root.Get("/", h.home)
	root.Get("/robots.txt", h.robots)
	custom := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := h.Forms.WorkspaceForHost(r.Context(), r.Host)
		if slug == "" {
			if h.SPA != nil {
				h.SPA.ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
			return
		}
		root.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), hostKey{}, slug)))
	})
	for _, p := range []string{"/", "/robots.txt", "/subscribe", "/subscribe/*", "/confirm/*", "/preferences", "/preferences/*",
		"/unsubscribe/*", "/view/*", "/archive", "/archive/*", "/archive.rss", "/sitemap.xml"} {
		r.Handle(p, custom)
	}
}

func (h *Handler) pageRoutes(r chi.Router) {
	r.Get("/theme.css", h.theme)
	r.Get("/logo", h.logo)
	r.Post("/turnstile-error", h.turnstileError)
	r.Get("/subscribe", h.subscribePage)
	r.Post("/subscribe", h.subscribePost)
	r.Get("/subscribe/challenge", h.challengePage)
	r.Post("/subscribe/challenge", h.challengePost)
	r.Get("/subscribe/{form}", h.subscribePage)
	r.Post("/subscribe/{form}", h.subscribePost)
	r.Get("/confirm/{token}", h.confirm)
	r.Head("/confirm/{token}", h.headOnly)
	r.Get("/preferences", h.requestPage)
	r.Post("/preferences", h.requestPost)
	r.Get("/preferences/{token}", h.preferencesPage)
	r.Post("/preferences/{token}", h.preferencesPost)
	r.Get("/unsubscribe/{token}", h.unsubscribePage)
	r.Head("/unsubscribe/{token}", h.headOnly)
	r.Post("/unsubscribe/{token}", h.unsubscribePost)
	r.Get("/view/{token}", h.viewOnline)
	r.Get("/archive", h.archive)
	r.Get("/archive.rss", h.feed)
	r.Get("/archive/{issue}", h.issue)
	r.Get("/sitemap.xml", h.sitemap)
	r.Get("/forms/{script}", h.formScript)
	r.Get("/forms/{id}/session", h.formSession)
	r.Options("/forms/{id}/session", h.formPreflight)
	r.Post("/forms/{id}/submit", h.formSubmit)
	r.Options("/forms/{id}/submit", h.formPreflight)
}

type hostKey struct{}

// --- the request's workspace ------------------------------------------------------

// site is the workspace a request is for and how to link inside it.
type site struct {
	ws    *store.Workspace
	set   *forms.Settings
	links forms.PageLinks
	// canon are the preferred links (the custom domain once verified).
	canon forms.PageLinks
	// preview renders a page for the app: its links carry no working token.
	preview bool
}

var errNoSite = errors.New("publicpages: no such workspace")

func (h *Handler) site(r *http.Request) (*site, error) {
	ctx := r.Context()
	slug, viaDomain := ctx.Value(hostKey{}).(string)
	if !viaDomain {
		slug = chi.URLParam(r, "ws")
	}
	if slug == "" {
		return nil, errNoSite
	}
	w, err := h.Q.GetWorkspaceBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNoSite
	}
	if err != nil {
		return nil, err
	}
	set, err := h.Forms.LoadSettings(ctx, &w)
	if err != nil {
		return nil, err
	}
	s := &site{ws: &w, set: set, canon: h.Forms.Links(&w, set)}
	host := strings.ToLower(r.Host)
	switch {
	case viaDomain:
		s.links = forms.LinksOn(schemeOf(h.URLs)+host, w.Slug, true)
	case set.CustomDomain() != "" && host == set.CustomDomain():
		s.links = forms.LinksOn(schemeOf(h.URLs)+host, w.Slug, false)
	default:
		s.links = forms.LinksOn(h.URLs.Base(), w.Slug, false)
	}
	return s, nil
}

func schemeOf(b publicurl.Builder) string {
	if strings.HasPrefix(b.Base(), "http://") {
		return "http://"
	}
	return "https://"
}

// claims verifies a signed link of kind for the request's workspace.
func (h *Handler) claims(s *site, token string, kind linktoken.Kind) (linktoken.Claims, bool) {
	if h.Tokens == nil || token == "" || len(token) > 200 {
		return linktoken.Claims{}, false
	}
	c, err := h.Tokens.Verify(token, kind)
	if err != nil || c.WorkspaceID != s.ws.ID || c.ContactID == uuid.Nil {
		return linktoken.Claims{}, false
	}
	return c, true
}

// previewToken stands in for signed links on previews: a preview shows a
// subscriber's page to the team (Viewers included), and a working link
// would let them change that subscriber's lists. It never verifies.
const previewToken = "preview"

func (h *Handler) sign(s *site, kind linktoken.Kind, contact, message uuid.UUID) string {
	if s.preview {
		return previewToken
	}
	return h.Tokens.Sign(linktoken.Claims{Kind: kind, WorkspaceID: s.ws.ID, ContactID: contact, MessageID: message})
}

// --- rendering ----------------------------------------------------------------------

// og is the Open Graph and Twitter card of a page.
type og struct {
	Type        string
	SiteName    string
	Title       string
	Description string
	URL         string
	Image       string
	Published   string
}

// view is what layout.html renders around a page.
type view struct {
	Title       string
	Description string
	Canonical   string
	Robots      string
	OG          *og
	JSONLD      any
	Feed        string
	FeedTitle   string
	CSS         string
	Theme       string
	Scripts     []string
	Brand       brandView
	Links       forms.PageLinks
	Landing     string
	PoweredBy   bool
	Privacy     string
	// Preview renders an in-app preview: forms don't submit.
	Preview bool
	Body    any
}

type brandView struct {
	Name    string
	Initial string
	Logo    string
}

func (h *Handler) view(s *site, title string, body any) *view {
	b := s.set.Branding
	v := &view{
		Title: title, Robots: "noindex, nofollow", CSS: s.links.Static("public.css?v=" + h.version),
		Theme: s.links.Theme(themeVersion(s.set)), Brand: brandView{Name: b.PublisherName, Initial: b.Initial()},
		Links: s.links, Landing: h.LandingURL, PoweredBy: b.PoweredBy, Privacy: b.PrivacyURL, Body: body,
	}
	if b.Logo != nil {
		v.Brand.Logo = s.links.Logo(b.Logo.Hash)
	}
	return v
}

func themeVersion(set *forms.Settings) string {
	return fmt.Sprintf("%x", set.UpdatedAt().UnixMilli())
}

// csp is the Content-Security-Policy of the pages: same-origin assets only,
// no inline script or style, forms posting to the page's own origin.
type csp struct {
	scripts    []string
	frames     []string
	fonts      []string
	formAction []string
	ancestors  []string
}

func (c csp) String() string {
	join := func(base string, extra []string) string {
		if len(extra) == 0 {
			return base
		}
		return base + " " + strings.Join(extra, " ")
	}
	ancestors := "'none'"
	if len(c.ancestors) > 0 {
		ancestors = strings.Join(c.ancestors, " ")
	}
	parts := []string{
		"default-src 'none'",
		join("script-src 'self'", c.scripts),
		join("style-src 'self'", c.fonts),
		join("font-src 'self'", c.fonts),
		"img-src 'self' https: data:",
		"connect-src 'self'",
		join("form-action 'self'", c.formAction),
		"frame-ancestors " + ancestors,
		"base-uri 'none'",
		"manifest-src 'self'",
	}
	if len(c.frames) > 0 {
		parts = append(parts, "frame-src "+strings.Join(c.frames, " "))
	}
	return strings.Join(parts, "; ")
}

// pageHeaders sets the headers every public page sends.
func pageHeaders(w http.ResponseWriter, policy csp, private bool) {
	hd := w.Header()
	hd.Set("Content-Type", "text/html; charset=utf-8")
	hd.Set("Content-Security-Policy", policy.String())
	hd.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	if len(policy.ancestors) == 0 {
		hd.Set("X-Frame-Options", "DENY")
	} else {
		hd.Del("X-Frame-Options")
	}
	hd.Set("Cross-Origin-Opener-Policy", "same-origin")
	if private {
		// Signed links: never cached, never indexed, never sent as a referrer.
		hd.Set("Cache-Control", "no-store")
		hd.Set("X-Robots-Tag", "noindex, nofollow")
		hd.Set("Referrer-Policy", "no-referrer")
	}
}

func (h *Handler) fontOrigins(s *site) []string {
	if o := s.set.Branding.FontOrigin(); o != "" {
		return []string{o}
	}
	return nil
}

// html renders a page inside the layout.
func (h *Handler) html(page string, v *view) ([]byte, error) {
	t := h.tmpl[page]
	if t == nil {
		return nil, fmt.Errorf("publicpages: no template %q", page)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// render writes a page with status.
func (h *Handler) render(w http.ResponseWriter, r *http.Request, s *site, page string, status int, v *view, policy csp, private bool) {
	policy.fonts = append(policy.fonts, h.fontOrigins(s)...)
	out, err := h.html(page, v)
	if err != nil {
		h.log().ErrorContext(r.Context(), "render public page", "page", page, "err", err)
		http.Error(w, "This page couldn't be shown. Try again in a minute.", http.StatusInternalServerError)
		return
	}
	buf := bytes.NewBuffer(out)
	pageHeaders(w, policy, private)
	if !private && w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-cache")
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf.Bytes()) //nolint:gosec // html/template output
	}
}

// messageBody is a page with one message (link not valid, form closed…).
type messageBody struct {
	Icon      string // "tick", "mail" or ""
	Heading   string
	Text      string
	Email     string
	TextAfter string
	// Actions are buttons; the first is primary.
	Actions []action
	Footer  string
}

type action struct {
	Label string
	Href  string
}

func (h *Handler) message(w http.ResponseWriter, r *http.Request, s *site, status int, title string, body messageBody, private bool) {
	v := h.view(s, title, body)
	if !private && status == http.StatusOK {
		v.Robots = "noindex, follow"
	}
	h.render(w, r, s, "message", status, v, csp{}, private)
}

// notFound answers a broken or forged link without saying who it was for.
func (h *Handler) notFound(w http.ResponseWriter, r *http.Request, s *site) {
	if s == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Robots-Tag", "noindex")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("This page doesn't exist.\n"))
		return
	}
	h.message(w, r, s, http.StatusNotFound, "Link not valid · "+s.set.Branding.PublisherName, messageBody{
		Heading: "This link doesn't work",
		Text:    "It may be cut short or out of date. Copy the whole link from the email, or manage your subscription from any recent issue.",
		Actions: []action{{Label: "Subscribe page", Href: s.links.Subscribe("")}},
	}, true)
}

func (h *Handler) serverError(w http.ResponseWriter, r *http.Request, s *site, err error) {
	h.log().ErrorContext(r.Context(), "public page", "path", r.URL.Path, "err", err)
	if s == nil {
		http.Error(w, "Something went wrong on our side. Nothing was changed; try again in a minute.", http.StatusInternalServerError)
		return
	}
	h.message(w, r, s, http.StatusInternalServerError, "Something went wrong · "+s.set.Branding.PublisherName, messageBody{
		Heading: "Something went wrong on our side",
		Text:    "Nothing was changed. Try again in a minute; if it keeps happening, reply to any email from " + s.set.Branding.PublisherName + ".",
	}, true)
}

func (h *Handler) siteOr404(w http.ResponseWriter, r *http.Request) (*site, bool) {
	s, err := h.site(r)
	if errors.Is(err, errNoSite) {
		h.notFound(w, r, nil)
		return nil, false
	}
	if err != nil {
		h.serverError(w, r, nil, err)
		return nil, false
	}
	return s, true
}

func (h *Handler) headOnly(w http.ResponseWriter, _ *http.Request) {
	// Link scanners check links with HEAD: answer without acting on them.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.WriteHeader(http.StatusOK)
}

// clientIP is the request's (proxy-resolved) address.
func clientIP(r *http.Request) *netip.Addr { return server.ClientIP(r) }

// --- assets ---------------------------------------------------------------------------

func (h *Handler) static(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "file")
	var data []byte
	ctype := ""
	switch name {
	case "public.css":
		data, _ = fs.ReadFile(staticFS, "static/public.css")
		ctype = "text/css; charset=utf-8"
	case "embed.css":
		data, ctype = forms.EmbedCSS(), "text/css; charset=utf-8"
		w.Header().Set("Access-Control-Allow-Origin", "*")
	case "issue.css":
		data, _ = fs.ReadFile(staticFS, "static/issue.css")
		ctype = "text/css; charset=utf-8"
	case "challenge.js":
		data, _ = fs.ReadFile(staticFS, "static/challenge.js")
		ctype = "text/javascript; charset=utf-8"
	default:
		http.NotFound(w, r)
		return
	}
	assetHeaders(w, r, ctype)
	http.ServeContent(w, r, name, staticTime, bytes.NewReader(data))
}

var staticTime = time.Now()

func assetHeaders(w http.ResponseWriter, r *http.Request, ctype string) {
	hd := w.Header()
	hd.Set("Content-Type", ctype)
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cross-Origin-Resource-Policy", "cross-origin")
	if r.URL.Query().Get("v") != "" {
		hd.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		hd.Set("Cache-Control", "public, max-age=3600")
	}
}

func (h *Handler) font(w http.ResponseWriter, r *http.Request) {
	name := path.Base(chi.URLParam(r, "file"))
	if !strings.HasSuffix(name, ".woff2") {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(staticFS, "static/fonts/"+name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", "font/woff2")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cache-Control", "public, max-age=31536000, immutable")
	hd.Set("Access-Control-Allow-Origin", "*")
	http.ServeContent(w, r, name, staticTime, bytes.NewReader(data))
}

// theme is the workspace's colours and font as CSS variables.
func (h *Handler) theme(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	b := s.set.Branding
	p := b.Palette
	var css strings.Builder
	css.WriteString(b.FontFace())
	fmt.Fprintf(&css, ":root{--pub-accent:%s;--pub-hover:%s;--pub-soft:%s;--pub-text:%s;--pub-on:%s;--pub-ground:%s;--pub-font:%s}\n",
		p.Accent, p.Hover, p.Soft, p.Text, p.OnAccent, p.Background, b.FontStack())
	assetHeaders(w, r, "text/css; charset=utf-8")
	http.ServeContent(w, r, "theme.css", s.set.UpdatedAt(), strings.NewReader(css.String()))
}

func (h *Handler) logo(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	data, ctype, at, err := h.Forms.Logo(r.Context(), s.ws)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	assetHeaders(w, r, ctype)
	// SVGs are only ever images here: nothing in them may run.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	http.ServeContent(w, r, "logo", at, bytes.NewReader(data))
}

func (h *Handler) turnstileError(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	ip := "unknown"
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	if h.Forms.AllowPerMinute("turnstile-error:"+ip, 3) && h.parseForm(w, r, 4<<10) {
		if err := h.Forms.ReportTurnstileError(r.Context(), s.ws, r.PostForm.Get("challenge"), r.PostForm.Get("code")); err != nil {
			h.log().WarnContext(r.Context(), "record turnstile error", "err", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// tlsCheck answers on-demand TLS "ask" requests (Caddy): 200 for verified
// custom domains, 404 otherwise, so certificates are only issued for them.
func (h *Handler) tlsCheck(w http.ResponseWriter, r *http.Request) {
	if h.Forms.HostVerified(r.Context(), r.URL.Query().Get("domain")) {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// home is "/" on a custom domain: the subscribe page.
func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	http.Redirect(w, r, s.links.Subscribe(""), http.StatusFound)
}
