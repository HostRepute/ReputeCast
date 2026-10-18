package publicpages

// The publication website (design expansion phase 7, M28): its pages share
// one frame (templates/site/frame.html: the header with the navigation,
// Subscribe and Sign in or the member's initials, the footer) and one
// stylesheet (static/site.css). A page of the site:
//
//	v, ok := h.siteViewFor(w, r, s)      // the frame, the reader, the language
//	v.Title, v.Body = …                  // its metadata and its data
//	h.renderSite(w, r, s, "post", http.StatusOK, v)
//
// with templates/site/<page>.html defining "site-main". Later phases add
// their pages (landing and sales pages, the shop) the same way and gate
// them with internal/website/gate (website.Service.Reader and Rules).
//
// Pages for a signed-in reader are never cached (Cache-Control: private,
// no-store); the others may be.

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"hash/fnv"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/templates"
	"github.com/hostrepute/reputecast/internal/website"
)

//go:embed templates/site/*.html
var siteFS embed.FS

// sitePages are the site's templates, each rendered in frame.html.
var sitePages = []string{"home", "post", "tag", "archive", "gone", "fail", "signin", "account"}

// siteState holds the site's parsed templates.
type siteState struct {
	once sync.Once
	tmpl map[string]*template.Template
	err  error
}

func (h *Handler) siteTemplates() (map[string]*template.Template, error) {
	h.siteTmpl.once.Do(func() {
		funcs := template.FuncMap{
			"add": func(a, b int) int { return a + b },
			"attr": func(s string) template.HTMLAttr {
				return template.HTMLAttr(s) //nolint:gosec // only the site's own class names go through attr
			},
		}
		h.siteTmpl.tmpl = map[string]*template.Template{}
		for _, p := range sitePages {
			t, err := template.New("site").Funcs(funcs).ParseFS(siteFS, "templates/site/frame.html", "templates/site/"+p+".html")
			if err != nil {
				h.siteTmpl.err = fmt.Errorf("publicpages: site template %s: %w", p, err)
				return
			}
			h.siteTmpl.tmpl[p] = t
		}
	})
	return h.siteTmpl.tmpl, h.siteTmpl.err
}

// siteHrefs are the frame's links.
type siteHrefs struct {
	Home, Subscribe, Signin, Account, Archive, Feed, Preferences, Privacy, Self string
}

// siteNavLink is a header link with its address.
type siteNavLink struct {
	Label, Href string
	Current     bool
}

// siteMember is the signed-in reader in the header.
type siteMember struct {
	Name, First, Initials, Color, Email string
}

// siteBrand is the publisher's mark.
type siteBrand struct {
	Initial, Logo string
}

// siteFeed is a feed the page links to.
type siteFeed struct{ Title, Href string }

// siteView is what frame.html renders around a page.
type siteView struct {
	Title       string
	Description string
	Canonical   string
	Robots      string
	OG          *og
	JSONLD      []any
	Feeds       []siteFeed
	CSS         string
	Theme       string
	Icon        string
	Scripts     []string
	Lang        string
	Dir         string
	Alternates  []alternate
	Menu        []langLink
	Preview     bool
	Brand       siteBrand
	Frame       website.SiteFrame
	Nav         []siteNavLink
	Href        siteHrefs
	Member      *siteMember
	// Session is the member's session (nil for a visitor).
	Session   *website.Session
	PoweredBy bool
	Landing   string
	Year      int
	Owner     string
	// Toast is a one-line notice at the bottom ("You're signed in as …").
	Toast string
	Body  any

	words  website.Words
	labels pagelabels.Set
	// private: the page is for a signed-in reader or holds a personal
	// token; it is never cached.
	private bool
	// current is the key of the header link of this page ("tag:engineering").
	current string
}

// T is one of the site's words as plain text: {{.T "site.signIn"}},
// {{.T "site.homeLabel" "title" .Frame.Title}}.
func (v *siteView) T(key string, kv ...any) string {
	args, _ := labelArgs(kv)
	return v.words.Text(key, args)
}

// H is a sentence of the site with markup: <b> in bold, <link> linked to
// the "href" argument.
func (v *siteView) H(key string, kv ...any) template.HTML {
	args, href := labelArgs(kv)
	tags := i18n.Tags{
		"b": func(inner string) string { return "<b>" + inner + "</b>" },
		"link": func(inner string) string {
			if href == "" {
				return inner
			}
			return `<a href="` + template.HTMLEscapeString(href) + `" class="st-lnk">` + inner + `</a>`
		},
	}
	return template.HTML(v.words.HTML(key, args, tags)) //nolint:gosec // the formatter escapes the text and the values
}

// L is a built-in page label (pagelabels) as plain text.
func (v *siteView) L(key string, kv ...any) string {
	args, _ := labelArgs(kv)
	return v.labels.Text(key, args)
}

// path is an address on the site: "/posts/x" on the custom domain,
// "/p/<workspace>/posts/x" on the instance.
func (s *site) path(parts ...string) string {
	if len(parts) == 0 {
		if s.links.Custom {
			return s.links.Origin() + "/"
		}
		return s.links.Path() + "/"
	}
	return s.links.Path(parts...)
}

// canonPath is the preferred address of a page of the site.
func (s *site) canonPath(parts ...string) string {
	if len(parts) == 0 {
		if s.canon.Custom {
			return s.canon.Origin() + "/"
		}
		return s.canon.Path() + "/"
	}
	return s.canon.Path(parts...)
}

// sitePath turns a frame path ("/tag/engineering") into an address.
func (s *site) sitePath(p string) string {
	p = strings.Trim(p, "/")
	if p == "" {
		return s.path()
	}
	return s.path(strings.Split(p, "/")...)
}

func cookieValue(r *http.Request, name string) string {
	if c, err := r.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}

// session is the signed-in member of a request (nil for a visitor).
func (h *Handler) session(r *http.Request, s *site) *website.Session {
	if h.Website == nil || s.preview {
		return nil
	}
	sess, err := h.Website.SessionFor(r.Context(), s.ws, cookieValue(r, website.SessionCookie))
	if err != nil {
		h.log().WarnContext(r.Context(), "website session", "err", err)
		return nil
	}
	return sess
}

// avatarColors are the avatar fills of the site, picked by name.
var avatarColors = []string{"c-ok", "c-pink", "c-info", "c-warn", "c-purple", "c-sky"}

// avatarColor picks a stable colour for a name.
func avatarColor(name string) string {
	f := fnv.New32a()
	_, _ = f.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
	return avatarColors[int(f.Sum32()%uint32(len(avatarColors)))]
}

// initials are "SR" for Sam Rivera, "S" for sam@….
func initials(name string) string {
	name = strings.TrimSpace(name)
	if at := strings.IndexByte(name, '@'); at > 0 {
		name = name[:at]
	}
	f := strings.FieldsFunc(name, func(r rune) bool { return r == ' ' || r == '.' || r == '_' || r == '-' })
	out := ""
	for i, w := range f {
		if i == 0 || i == len(f)-1 {
			for _, r := range w {
				out += strings.ToUpper(string(r))
				break
			}
		}
		if len(f) == 1 {
			break
		}
	}
	if out == "" {
		return "?"
	}
	return out
}

// owner is the publisher's legal name for the footer's copyright: the
// first part of the postal address ("Acme Inc · 500 Market St…"), else
// the publisher's name.
func owner(s *site) string {
	addr := templates.WorkspaceAddress(s.ws)
	for _, sep := range []string{" · ", ", ", "\n"} {
		if i := strings.Index(addr, sep); i > 0 {
			addr = addr[:i]
			break
		}
	}
	if a := strings.TrimSpace(addr); a != "" && len(a) <= 80 {
		return a
	}
	return s.set.Branding.PublisherName
}

// siteViewFor builds the frame of a page for a request: the site's header
// and footer, the reader, the page's language. ok is false when it
// answered already (an error).
func (h *Handler) siteViewFor(w http.ResponseWriter, r *http.Request, s *site) (*siteView, bool) {
	ctx := r.Context()
	pl := h.visitorLang(w, r, s, nil)
	v, err := h.siteViewIn(ctx, s, pl)
	if err != nil {
		h.serverError(w, r, s, err)
		return nil, false
	}
	v.Menu = h.menu(r, s, pl.Code, nil)
	v.Href.Self = r.URL.EscapedPath()
	if sess := h.session(r, s); sess != nil && v.Frame.On {
		v.Session = sess
		v.private = true
		name := sess.Name()
		v.Member = &siteMember{Name: name, First: sess.FirstOrName(), Initials: initials(name), Color: avatarColor(name), Email: sess.Email}
	}
	return v, true
}

// siteViewIn is the frame in a language, without a request (previews).
func (h *Handler) siteViewIn(ctx context.Context, s *site, pl pageLang) (*siteView, error) {
	if h.Website == nil {
		return nil, fmt.Errorf("publicpages: the website isn't wired")
	}
	frame, err := h.Website.Frame(ctx, s.ws, s.set)
	if err != nil {
		return nil, err
	}
	b := s.set.Branding
	v := &siteView{
		Robots: "index, follow", CSS: s.links.Static("site.css?v=" + h.siteVersion()), Theme: s.links.Theme(themeVersion(s.set)),
		Lang: pl.Code, Dir: pagelabels.Dir(pl.Code), Preview: s.preview, Brand: siteBrand{Initial: b.Initial()}, Frame: frame,
		PoweredBy: b.PoweredBy, Landing: h.LandingURL, Year: h.now().In(location(s.ws)).Year(), Owner: owner(s),
		words: website.WordsIn(pl.Labels.Lang()), labels: pl.Labels,
	}
	if b.Logo != nil {
		v.Brand.Logo = s.links.Logo(b.Logo.Hash)
		v.Icon = v.Brand.Logo
	}
	if !frame.Index {
		v.Robots = "noindex, follow"
	}
	v.Href = siteHrefs{Home: s.path(), Subscribe: s.links.Subscribe(""), Signin: s.path("signin"), Account: s.path("account"),
		Archive: s.links.Archive(), Feed: s.path("feed"), Preferences: s.links.PreferencesRequest(), Privacy: b.PrivacyURL}
	if s.copy != nil {
		v.Href.Privacy = s.copy.PrivacyURL(pl.Code, v.Href.Privacy)
	} else if pc := h.copyOf(ctx, s); pc != nil {
		v.Href.Privacy = pc.PrivacyURL(pl.Code, v.Href.Privacy)
	}
	for _, l := range frame.Nav {
		href := l.URL
		if href == "" {
			href = s.sitePath(l.Path)
		}
		v.Nav = append(v.Nav, siteNavLink{Label: l.Label, Href: href})
	}
	return v, nil
}

// mark sets the header link of the current page (aria-current).
func (v *siteView) mark(key string, frame website.SiteFrame) {
	v.current = key
	for i, l := range frame.Nav {
		if i < len(v.Nav) && l.Key == key && key != "" {
			v.Nav[i].Current = true
		}
	}
}

var siteCSSVersion = sync.OnceValue(func() string {
	data, err := staticFS.ReadFile("static/site.css")
	if err != nil {
		return "0"
	}
	f := fnv.New64a()
	_, _ = f.Write(data)
	return strconv.FormatUint(f.Sum64(), 36)
})

func (h *Handler) siteVersion() string { return siteCSSVersion() }

// renderSite writes a page of the site with status.
func (h *Handler) renderSite(w http.ResponseWriter, r *http.Request, s *site, page string, status int, v *siteView) {
	out, err := h.siteHTML(page, v)
	if err != nil {
		h.log().ErrorContext(r.Context(), "render site page", "page", page, "err", err)
		http.Error(w, "This page couldn't be shown. Try again in a minute.", http.StatusInternalServerError)
		return
	}
	policy := csp{fonts: h.fontOrigins(s)}
	pageHeaders(w, policy, false)
	hd := w.Header()
	hd.Add("Vary", "Accept-Language")
	hd.Add("Vary", "Cookie")
	if v.private {
		// A signed-in reader's page: never cached, never indexed as theirs.
		hd.Set("Cache-Control", "private, no-store")
	} else if hd.Get("Cache-Control") == "" {
		hd.Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(out) //nolint:gosec // html/template output
	}
}

// siteHTML renders a page of the site.
func (h *Handler) siteHTML(page string, v *siteView) ([]byte, error) {
	tmpl, err := h.siteTemplates()
	if err != nil {
		return nil, err
	}
	t := tmpl[page]
	if t == nil {
		return nil, fmt.Errorf("publicpages: no site template %q", page)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "site", v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// siteRootPaths are the site's addresses on a custom domain (the archive
// and the subscribe page are the public pages' own).
var siteRootPaths = []string{"/posts/*", "/tag/*", "/feed", "/signin", "/signin/*", "/signout", "/account", "/account/*"}

// siteRoutes mounts the website's pages.
func (h *Handler) siteRoutes(r chi.Router) {
	r.Get("/posts/{slug}", h.sitePost)
	r.Head("/posts/{slug}", h.sitePost)
	r.Post("/posts/{slug}", h.sitePostSignup)
	r.Get("/tag/{slug}", h.siteTag)
	r.Get("/tag/{slug}/feed", h.siteTagFeed)
	r.Get("/feed", h.siteFeed)
	r.Get("/signin", h.signinPage)
	r.Post("/signin", h.signinPost)
	r.Post("/signin/code", h.signinCode)
	r.Get("/signin/link/{token}", h.signinLink)
	r.Head("/signin/link/{token}", h.headOnly)
	r.Post("/signin/link/{token}", h.signinLinkPost)
	r.Post("/signout", h.signout)
	r.Get("/account", h.accountPage)
	r.Post("/account/{action}", h.accountPost)
}

// siteOn reports whether the website serves a workspace's pages.
func (h *Handler) siteOn(ctx context.Context, s *site) bool {
	if h.Website == nil {
		return false
	}
	on, err := h.Website.IsOn(ctx, s.ws)
	if err != nil {
		h.log().WarnContext(ctx, "website status", "err", err)
	}
	return on
}
