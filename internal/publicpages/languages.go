package publicpages

import (
	"context"
	"html"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/pagelabels"
)

// The public pages in the visitor's language (design expansion M17, phase
// 2 brief §3.6). A page reached from an email (the preference center,
// unsubscribe, the confirmation page) shows the contact's language; the
// others the language picked in the page's language menu, then the
// browser's languages, then the default. Only a language whose page copy
// is Live is shown: until then its visitors get the fallback's pages,
// whole. Every page sets lang and dir from the language it shows and lists
// the Live languages in their own names in its footer, as plain links that
// work without JavaScript.

// langParam picks a language from the menu; langCookie remembers the pick
// for the rest of the visit (a session cookie the visitor asked for).
const (
	langParam  = "lang"
	langCookie = "rc_lang"
)

// pageLang is the language a page shows.
type pageLang struct {
	Code   string
	Labels pagelabels.Set
	// picked is true when the visitor chose it in the language menu.
	picked bool
}

// copyOf is the pages' copy in every language, read once per request. A
// failure leaves the pages in their own language.
func (h *Handler) copyOf(ctx context.Context, s *site) *forms.PagesCopy {
	if s.copy == nil {
		pc, err := h.Forms.PagesCopyOf(ctx, s.ws, s.set)
		if err != nil {
			h.log().WarnContext(ctx, "load page copy", "err", err)
			return nil
		}
		s.copy = pc
	}
	return s.copy
}

// fixedLang is a page in a language asked for by name (the app's
// previews): that language when the pages are Live in it, else their own.
func (h *Handler) fixedLang(ctx context.Context, s *site, code string, live func(string) bool) pageLang {
	pc := h.copyOf(ctx, s)
	if pc == nil {
		return s.use(pageLang{Code: s.set.CopyLanguage(), Labels: pagelabels.For(s.set.CopyLanguage(), nil)})
	}
	if live == nil {
		live = pc.Live
	}
	code = languages.Base(code)
	if code == "" || !live(code) {
		code = pc.CopyLanguage
	}
	return s.use(pageLang{Code: code, Labels: pc.Labels(code)})
}

// picked is the language the visitor chose in the menu: ?lang=, else the
// cookie the choice set.
func picked(r *http.Request) string {
	if v := strings.TrimSpace(r.URL.Query().Get(langParam)); v != "" {
		return v
	}
	if c, err := r.Cookie(langCookie); err == nil {
		return c.Value
	}
	return ""
}

// chooseLang picks the language of a page. wants are the visitor's
// languages after the menu's pick: the contact's language on pages
// reached from an email, else the browser's. live says whether a language
// can be shown (the pages, and the form on a page with one).
func (h *Handler) chooseLang(w http.ResponseWriter, r *http.Request, s *site, live func(string) bool, wants ...string) pageLang {
	pc := h.copyOf(r.Context(), s)
	if pc == nil {
		return s.use(pageLang{Code: s.set.CopyLanguage(), Labels: pagelabels.For(s.set.CopyLanguage(), nil)})
	}
	if live == nil {
		live = pc.Live
	}
	pick := languages.Base(picked(r))
	if pick != "" && pc.Languages.Offers(pick) && live(pick) {
		h.rememberLang(w, r, s, pick)
		return s.use(pageLang{Code: pick, Labels: pc.Labels(pick), picked: true})
	}
	code := pc.Choose(live, wants...)
	return s.use(pageLang{Code: code, Labels: pc.Labels(code)})
}

// visitorLang is the language of a page anyone can open: the menu's pick,
// then the browser's languages, then the default.
func (h *Handler) visitorLang(w http.ResponseWriter, r *http.Request, s *site, live func(string) bool) pageLang {
	return h.chooseLang(w, r, s, live, audience.BrowserLanguages(r.Header.Get("Accept-Language"))...)
}

// contactLang is the language of a page reached from an email: the menu's
// pick, then the contact's language (then the fallback).
func (h *Handler) contactLang(w http.ResponseWriter, r *http.Request, s *site, contactLanguage string) pageLang {
	pl := h.chooseLang(w, r, s, nil, contactLanguage)
	if pl.picked {
		return pl
	}
	if pc := h.copyOf(r.Context(), s); pc != nil {
		if code := languages.Base(contactLanguage); code == "" || !pc.Languages.Offers(code) {
			// No language on the contact, or one the workspace doesn't
			// offer: the fallback, as their emails.
			code = pc.Fallback()
			return s.use(pageLang{Code: code, Labels: pc.Labels(code)})
		}
	}
	return pl
}

// rememberLang keeps a language picked in the menu for the rest of the
// visit. The cookie holds only the language code.
func (h *Handler) rememberLang(w http.ResponseWriter, r *http.Request, s *site, code string) {
	if w == nil || r.URL.Query().Get(langParam) == "" || s.preview {
		return
	}
	if c, err := r.Cookie(langCookie); err == nil && c.Value == code {
		return
	}
	path := "/"
	if !s.links.Custom {
		path = "/p/" + url.PathEscape(s.ws.Slug) + "/"
	}
	secure := r.TLS != nil || strings.HasPrefix(s.links.Origin(), "https://")
	http.SetCookie(w, &http.Cookie{Name: langCookie, Value: code, Path: path, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}) //nolint:gosec // G124: Secure follows the pages' origin, plain http only in development.
}

// langLink is one language of the footer's menu.
type langLink struct {
	Code, Name, Href string
	Current          bool
}

// menu is the footer's language menu: the Live languages in their own
// names, each linking to this page in it. Nil with fewer than two.
func (h *Handler) menu(r *http.Request, s *site, current string, live func(string) bool) []langLink {
	pc := h.copyOf(r.Context(), s)
	if pc == nil || s.preview {
		return nil
	}
	if live == nil {
		live = pc.Live
	}
	var out []langLink
	for _, code := range pc.LiveLanguages() {
		if !live(code) {
			continue
		}
		l, _ := languages.Lookup(code)
		name := l.Native
		if name == "" {
			name = code
		}
		out = append(out, langLink{Code: code, Name: name, Href: withLang(r, code), Current: code == current})
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

// globe is the language menu's icon.
const globe = `<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` +
	`<circle cx="8" cy="8" r="6.2"></circle><path d="M1.8 8h12.4M8 1.8c1.8 1.7 2.7 3.8 2.7 6.2S9.8 12.5 8 14.2C6.2 12.5 5.3 10.4 5.3 8S6.2 3.5 8 1.8z"></path></svg>`

// alternates are a page's addresses in each Live language for search
// engines (hreflang): the page with ?lang=, and the page itself as
// x-default. None with fewer than two.
func (h *Handler) alternates(ctx context.Context, s *site, href string, live func(string) bool) []alternate {
	pc := h.copyOf(ctx, s)
	if pc == nil || s.preview || href == "" {
		return nil
	}
	if live == nil {
		live = pc.Live
	}
	var out []alternate
	for _, code := range pc.LiveLanguages() {
		if live(code) {
			out = append(out, alternate{Lang: code, Href: withParam(href, langParam, code)})
		}
	}
	if len(out) < 2 {
		return nil
	}
	return append(out, alternate{Lang: "x-default", Href: href})
}

// withParam is an address with one query parameter set.
func withParam(href, key, value string) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

// defaultLang is the workspace's default language.
func (h *Handler) defaultLang(ctx context.Context, s *site) string {
	if pc := h.copyOf(ctx, s); pc != nil && pc.Languages.Default != "" {
		return pc.Languages.Default
	}
	return s.set.CopyLanguage()
}

// withLang is the request's own address with ?lang= set: the same page in
// another language. The pages answering a Turnstile check have no address
// of their own: theirs offer the form again.
func withLang(r *http.Request, code string) string {
	path, q := r.URL.EscapedPath(), r.URL.Query()
	if strings.HasSuffix(path, "/subscribe/challenge") && q.Get("c") == "" {
		path = strings.TrimSuffix(path, "/challenge")
	}
	q.Set(langParam, code)
	return path + "?" + q.Encode()
}

// setLang shows a page in a language: its lang and dir, its labels and
// the footer's language menu (none without a request: the app's previews).
func (h *Handler) setLang(v *view, r *http.Request, s *site, pl pageLang, live func(string) bool) {
	h.setLangCtx(v, nil, s, pl)
	if r != nil {
		v.Menu = h.menu(r, s, pl.Code, live)
	}
}

func (h *Handler) setLangCtx(v *view, ctx context.Context, s *site, pl pageLang) {
	s.use(pl)
	v.Lang, v.Dir, v.labels = pl.Code, pagelabels.Dir(pl.Code), pl.Labels
	if s.copy != nil {
		v.Privacy = s.copy.PrivacyURL(pl.Code, v.Privacy)
	} else if ctx != nil {
		if pc := h.copyOf(ctx, s); pc != nil {
			v.Privacy = pc.PrivacyURL(pl.Code, v.Privacy)
		}
	}
}

// T is a built-in label as plain text, for the templates: {{$.T "key"}},
// {{$.T "key" "publisher" .Name}}.
func (v *view) T(key string, kv ...any) string {
	args, _ := labelArgs(kv)
	return v.labels.Text(key, args)
}

// H is a built-in label as HTML: its <b> text in bold and its <link> text
// linked to the "href" argument.
func (v *view) H(key string, kv ...any) template.HTML {
	args, href := labelArgs(kv)
	return labelHTML(v.labels, key, args, href)
}

func labelHTML(l pagelabels.Set, key string, args i18n.Args, href string) template.HTML {
	tags := i18n.Tags{
		"b": func(inner string) string { return "<strong>" + inner + "</strong>" },
		"link": func(inner string) string {
			if href == "" {
				return inner
			}
			return `<a href="` + html.EscapeString(href) + `">` + inner + `</a>`
		},
	}
	return template.HTML(l.HTML(key, args, tags)) //nolint:gosec // the label's text and values are escaped by the formatter
}

// labelArgs reads name, value pairs; "href" is the address of a <link>.
func labelArgs(kv []any) (i18n.Args, string) {
	args := i18n.Args{}
	href := ""
	for i := 0; i+1 < len(kv); i += 2 {
		name, _ := kv[i].(string)
		if name == "href" {
			href, _ = kv[i+1].(string)
			continue
		}
		args[name] = kv[i+1]
	}
	return args, href
}
