package publicpages

// Grow's public pages (design expansion phase 5; brief §5.9–5.11, 5.14):
// a referrer's hub, a referral link's invite page, the leaderboard, reward
// downloads and the recommendations page, plus the confirmation page's
// "While you're here" and the preference center's "Invite friends". They
// work without JavaScript (a script only adds Copy). Their words are
// English: Grow's labels aren't in the page languages yet.

import (
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/recommendations"
	"github.com/hostrepute/reputecast/internal/referrals"
)

//go:embed messages/*.json
var growFS embed.FS

var growMsgs = i18n.MustLoad(growFS, "publicgrow")

var growLang = i18n.For(i18n.English)

// gt is a Grow label as text.
func gt(key string, kv ...any) string {
	args, _ := labelArgs(kv)
	return growMsgs.M(key).Text(growLang, args)
}

// gh is a Grow label as HTML: <b> bold, <link> linked to "href".
func gh(key string, kv ...any) template.HTML {
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
	return template.HTML(growMsgs.M(key).HTML(growLang, args, tags)) //nolint:gosec // the formatter escapes the label and its values
}

// G is a Grow label for the templates: {{$.G "hub.heading"}}.
func (v *view) G(key string, kv ...any) string { return gt(key, kv...) }

// GH is a Grow label as HTML for the templates.
func (v *view) GH(key string, kv ...any) template.HTML { return gh(key, kv...) }

// growView is a Grow page: in English, without a language menu.
func (h *Handler) growView(s *site, title string, body any) *view {
	v := h.view(s, title, body)
	v.Lang, v.Dir, v.labels = "en", "ltr", pagelabels.English()
	return v
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
	v := h.growView(s, title, body)
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
	v := h.growView(s, heading+" · "+s.set.Branding.PublisherName, messageBody{Heading: heading, Text: text, Actions: actions})
	h.render(w, r, s, "message", status, v, csp{}, true)
}

// shortDate is "19 Oct" (and "19 Oct 2025" for another year).
func shortDate(t time.Time, now time.Time) string {
	if t.Year() != now.Year() {
		return t.Format("2 Jan 2006")
	}
	return t.Format("2 Jan")
}

// fileDesc is "PDF, 18 MB".
func fileDesc(contentType string, size int64) string {
	kind := strings.ToUpper(strings.TrimPrefix(strings.TrimPrefix(contentType, "application/"), "image/"))
	if kind == "JPEG" {
		kind = "JPG"
	}
	if size <= 0 {
		return kind
	}
	mb := float64(size) / (1 << 20)
	switch {
	case mb >= 10:
		return fmt.Sprintf("%s, %d MB", kind, int(math.Round(mb)))
	case mb >= 1:
		return fmt.Sprintf("%s, %.1f MB", kind, mb)
	}
	return fmt.Sprintf("%s, %d KB", kind, max(int(math.Round(float64(size)/1024)), 1))
}

// ordinal is "15th".
func ordinal(n int) string {
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

// articled is "the Acme Field Guide" for "The Acme Field Guide".
func articled(name string) string {
	if strings.HasPrefix(name, "The ") {
		return "the " + name[4:]
	}
	return name
}
