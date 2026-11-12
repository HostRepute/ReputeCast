package publicpages

import (
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/website/importer"
)

// Old addresses (M61, G-26): on the site's own custom domain, the
// addresses a site imported from Substack, Ghost or WordPress had keep
// working. A post's old address answers 301 to the post; the old feed
// addresses answer 301 to /feed, and /posts/<slug>/ to /posts/<slug>.
// Anything else on the domain is the site's own 404 page, not the web
// app's. Landing pages (/{slug}) win over old addresses: only a path
// nothing on the site answers is looked up.

// oldFeeds are the feed addresses Substack, Ghost and WordPress serve.
var oldFeeds = map[string]bool{"/feed/": true, "/rss": true, "/rss/": true, "/feed.xml": true, "/rss.xml": true, "/feed/rss2": true, "/feed/atom": true}

// siteLegacy answers an old address of the site with a redirect, and
// reports whether it did.
func (h *Handler) siteLegacy(w http.ResponseWriter, r *http.Request, s *site) bool {
	if h.Website == nil || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		return false
	}
	if _, onDomain := r.Context().Value(hostKey{}).(string); !onDomain {
		return false
	}
	p := r.URL.Path
	switch {
	case oldFeeds[strings.ToLower(p)]:
		redirectPermanent(w, r, s.sitePath("feed"), "")
		return true
	case strings.HasPrefix(p, "/posts/") && strings.HasSuffix(p, "/") && strings.Count(p, "/") == 3 && len(p) > len("/posts//"):
		redirectPermanent(w, r, s.sitePath(strings.TrimSuffix(p, "/")), r.URL.RawQuery)
		return true
	}
	key := importer.LegacyKey(p, r.URL.RawQuery)
	if key == "" {
		return false
	}
	slug, err := h.Website.LegacyPost(r.Context(), s.ws, key)
	if err != nil {
		h.log().WarnContext(r.Context(), "old address", "path", p, "err", err)
		return false
	}
	if slug == "" {
		return false
	}
	redirectPermanent(w, r, s.sitePath("posts/"+slug), "")
	return true
}

// redirectPermanent answers 301 to a path of the site.
func redirectPermanent(w http.ResponseWriter, r *http.Request, to, rawQuery string) {
	if rawQuery != "" {
		to += "?" + rawQuery
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.Redirect(w, r, to, http.StatusMovedPermanently) //nolint:gosec // G710: a path on this site
}

// rootNotFound answers a path nothing on a custom domain answers: an old
// address of the site, or the site's 404 page.
func (h *Handler) rootNotFound(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	h.siteNotFound(w, r, s)
}

// SiteFallback serves what no route matched: on a workspace's verified
// custom domain the site answers (an old address, or its 404 page); any
// other host gets next (the web app).
func (h *Handler) SiteFallback(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.Forms == nil {
			next.ServeHTTP(w, r)
			return
		}
		if slug := h.Forms.WorkspaceForHost(r.Context(), r.Host); slug != "" {
			h.rootNotFound(w, r.WithContext(withHost(r.Context(), slug)))
			return
		}
		next.ServeHTTP(w, r)
	})
}
