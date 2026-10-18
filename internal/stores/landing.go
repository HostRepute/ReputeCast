package stores

import (
	"net/url"
	"strings"
)

// ClickTokenParam is the query parameter that carries ReputeCast's click
// token on links to a store (phase 6's click token, `rc_click`; the phase
// 7 brief's `rc` is read too).
const ClickTokenParam = "rc_click"

// legacyClickTokenParam is the shorter name the phase 7 brief uses.
const legacyClickTokenParam = "rc"

// ParseLanding reads a landing page (a path with its query, or a full URL)
// into its UTM parameters and click token.
func ParseLanding(landing string) (UTM, string) {
	landing = strings.TrimSpace(landing)
	if landing == "" {
		return UTM{}, ""
	}
	q := ""
	if u, err := url.Parse(landing); err == nil {
		q = u.RawQuery
	} else if _, after, ok := strings.Cut(landing, "?"); ok {
		q = after
	}
	if i := strings.IndexByte(q, '#'); i >= 0 {
		q = q[:i]
	}
	v, _ := url.ParseQuery(q)
	utm := UTM{
		Source:   v.Get("utm_source"),
		Medium:   v.Get("utm_medium"),
		Campaign: v.Get("utm_campaign"),
		Content:  v.Get("utm_content"),
		Term:     v.Get("utm_term"),
	}
	token := v.Get(ClickTokenParam)
	if token == "" {
		token = v.Get(legacyClickTokenParam)
	}
	return utm, token
}

// StripLanding keeps a landing page's path and query without the click
// token, so what is stored can't be replayed as a token.
func StripLanding(landing string) string {
	u, err := url.Parse(strings.TrimSpace(landing))
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Del(ClickTokenParam)
	q.Del(legacyClickTokenParam)
	u.RawQuery = q.Encode()
	u.Fragment = ""
	if u.Host != "" {
		return u.String()
	}
	out := u.Path
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}
