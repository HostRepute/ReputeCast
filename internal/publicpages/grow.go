package publicpages

// Grow's public pages (design expansion phase 5; brief §5.9–5.11, 5.14):
// a referrer's hub, a referral link's invite page, the leaderboard, reward
// downloads and the recommendations page, plus the confirmation page's
// "While you're here" and the preference center's "Invite friends". They
// work without JavaScript (a script only adds Copy). They read in the
// page's language like the other public pages: the menu's pick, then the
// reader's own language on a page reached from their email (the hub), else
// the browser's, then the default.

import (
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/recommendations"
	"github.com/hostrepute/reputecast/internal/referrals"
)

//go:embed messages/*.json
var growFS embed.FS

var growMsgs = i18n.MustLoad(growFS, "publicgrow")

// growText is a Grow label as text in a language.
func growText(l i18n.Lang, key string, kv ...any) string {
	args, _ := labelArgs(kv)
	return growMsgs.M(key).Text(l, args)
}

// growHTML is a Grow label as HTML in a language: <b> bold, <link> linked
// to "href".
func growHTML(l i18n.Lang, key string, kv ...any) template.HTML {
	args, href := labelArgs(kv)
	tags := i18n.Tags{
		"b": func(inner string) string { return "<strong>" + inner + "</strong>" },
		"link": func(inner string) string {
			if href == "" {
				return inner
			}
			return `<a href="` + template.HTMLEscapeString(href) + `">` + inner + `</a>`
		},
	}
	return template.HTML(growMsgs.M(key).HTML(l, args, tags)) //nolint:gosec // the formatter escapes the label and its values
}

// gt is a Grow label as text in the page's language.
func (s *site) gt(key string, kv ...any) string { return growText(s.lang(), key, kv...) }

// gh is a Grow label as HTML in the page's language.
func (s *site) gh(key string, kv ...any) template.HTML { return growHTML(s.lang(), key, kv...) }

// G is a Grow label for the templates: {{$.G "hub.heading"}}.
func (v *view) G(key string, kv ...any) string { return growText(v.labels.Lang(), key, kv...) }

// GH is a Grow label as HTML for the templates.
func (v *view) GH(key string, kv ...any) template.HTML { return growHTML(v.labels.Lang(), key, kv...) }

// growEnglish reports whether a Grow label is written in English for the
// page's language (it isn't translated): English grammar the code applies
// (ordinals, "the" before a reward's name) holds only then.
func (s *site) growEnglish(key string) bool {
	l := s.lang().Locale
	if l == i18n.English || l == i18n.Pseudo || l == "" {
		return true
	}
	return slices.Contains(growMsgs.Missing(l), key)
}

// growView is a Grow page in the page's language, with the footer's
// language menu (none without a request: the app's previews).
func (h *Handler) growView(r *http.Request, s *site, title string, body any) *view {
	v := h.view(s, title, body)
	h.setLang(v, r, s, s.page(), nil)
	return v
}

// linkLang picks the language of a page reached from a signed link (Grow's,
// a poll's or a survey's): the email's or the reader's own when the link
// names them, else the visitor's.
func (h *Handler) linkLang(w http.ResponseWriter, r *http.Request, s *site, c linktoken.Claims) pageLang {
	if c.ContactID != uuid.Nil || c.MessageID != uuid.Nil {
		return h.messageLang(w, r, s, c)
	}
	return h.visitorLang(w, r, s, nil)
}

// growRoutes are the Grow pages, on the instance (/p/{ws}/…) and on a
// workspace's custom domain (at the root).
func (h *Handler) growRoutes(r chi.Router) {
	r.Get("/join/{code}", h.joinPage)
	r.Post("/join/{code}", h.joinPost)
	r.Head("/join/{code}", h.headOnly)
	r.Get("/referrals", h.hubRequestPage)
	r.Post("/referrals", h.hubRequestPost)
	r.Get("/referrals/{token}", h.hubPage)
	r.Head("/referrals/{token}", h.headOnly)
	r.Post("/referrals/{token}/send", h.hubSend)
	r.Post("/referrals/{token}/privacy", h.hubPrivacy)
	r.Post("/referrals/{token}/address/{reward}", h.hubAddress)
	r.Get("/referrals/{token}/leaderboard", h.boardPage)
	r.Get("/leaderboard", h.boardPage)
	r.Get("/rewards/{token}", h.rewardDownload)
	r.Head("/rewards/{token}", h.headOnly)
	r.Get("/recommends", h.recommendsPage)
	r.Post("/recommends", h.recommendsPost)
	r.Get("/recommends/visit/{id}", h.recommendsVisit)
	r.Get("/recommends/{token}", h.recommendsPage)
	r.Head("/recommends/{token}", h.headOnly)
	r.Post("/recommends/{token}", h.recommendsPost)
	r.Post("/confirm/{token}", h.confirmRecommend)
}

// growPaths are the Grow pages a custom domain answers at its root.
var growPaths = []string{"/join/*", "/referrals", "/referrals/*", "/leaderboard", "/rewards/*", "/recommends", "/recommends/*"}

func (h *Handler) referrals() *referrals.Service {
	if h.Referrals == nil {
		return nil
	}
	return h.Referrals()
}

func (h *Handler) recommendations() *recommendations.Service {
	if h.Recommendations == nil {
		return nil
	}
	return h.Recommendations()
}

// growPage renders a Grow page (private: signed links, never cached or
// indexed).
func (h *Handler) growPage(w http.ResponseWriter, r *http.Request, s *site, page string, status int, title string, body any, private bool) {
	v := h.growView(r, s, title, body)
	if page == "grow-hub" {
		// Copy for the link; the page works without it.
		v.Scripts = append(v.Scripts, s.links.Static("grow.js?v="+h.version))
	}
	if !private && status == http.StatusOK {
		v.Robots = "noindex, follow"
	}
	h.render(w, r, s, page, status, v, csp{}, private)
}

// growMessage is a one-message Grow page.
func (h *Handler) growMessage(w http.ResponseWriter, r *http.Request, s *site, status int, heading, text string, actions []action) {
	v := h.growView(r, s, heading+" · "+s.set.Branding.PublisherName, messageBody{Heading: heading, Text: text, Actions: actions})
	h.render(w, r, s, "message", status, v, csp{}, true)
}

// shortDate is "19 Oct" (and "19 Oct 2025" for another year), in the
// page's region format ("19 thg 10").
func (s *site) shortDate(t time.Time, now time.Time) string {
	l := s.lang()
	if l.Region != "" && l.Region != i18n.Standard {
		if t.Year() != now.Year() {
			return l.Region.Date(t)
		}
		return l.Region.DayMonth(t)
	}
	if t.Year() != now.Year() {
		return t.Format("2 Jan 2006")
	}
	return t.Format("2 Jan")
}

// fileDesc is "PDF, 18 MB", its size in the page's region format.
func (s *site) fileDesc(contentType string, size int64) string {
	kind := strings.ToUpper(strings.TrimPrefix(strings.TrimPrefix(contentType, "application/"), "image/"))
	if kind == "JPEG" {
		kind = "JPG"
	}
	if size <= 0 {
		return kind
	}
	r := s.lang().Region
	mb := float64(size) / (1 << 20)
	switch {
	case mb >= 10:
		return kind + ", " + r.Int(int64(math.Round(mb))) + " MB"
	case mb >= 1:
		return kind + ", " + r.Decimal(mb, 1) + " MB"
	}
	return kind + ", " + r.Int(int64(max(int(math.Round(float64(size)/1024)), 1))) + " KB"
}

// ordinal is a rank: "15th" in English, the number where a translation
// says the rest ("Hạng 15").
func (s *site) ordinal(n int, key string) string {
	if !s.growEnglish(key) {
		return s.lang().Region.Int(int64(n))
	}
	suffix := "th"
	switch {
	case n%100 >= 11 && n%100 <= 13:
	case n%10 == 1:
		suffix = "st"
	case n%10 == 2:
		suffix = "nd"
	case n%10 == 3:
		suffix = "rd"
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// articled is "the Acme Field Guide" for "The Acme Field Guide" inside an
// English sentence; a translation keeps the name as it is.
func (s *site) articled(name, key string) string {
	if strings.HasPrefix(name, "The ") && s.growEnglish(key) {
		return "the " + name[4:]
	}
	return name
}
