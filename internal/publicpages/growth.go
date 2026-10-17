package publicpages

import "net/http"

// Growth sources (audience-insights brief §3.4): the hosted pages post back
// where the visitor was (the page's address with its UTM parameters) and
// came from, as the embed script does.

// visitedURL is a hosted page's address as the visitor opened it, query
// included.
func visitedURL(r *http.Request, page string) string {
	if q := r.URL.RawQuery; q != "" {
		return clipURL(page + "?" + q)
	}
	return clipURL(page)
}

// clipURL keeps a posted address to a sane length.
func clipURL(s string) string {
	if len(s) > 2000 {
		return s[:2000]
	}
	return s
}
