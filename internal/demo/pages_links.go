package demo

import (
	"net/url"
	"strings"
)

// PagesLinkStatus answers the page builder's link check for the demo's
// addresses without the network: Acme's own sites answer, except the
// beta terms the "Join the Beta Program" draft points at, which are gone
// (PageBuilderStates: "A link doesn't work"). ok is false for any other
// address.
func PagesLinkStatus(raw string) (int, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return 0, false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if host != "acme.io" && !strings.HasSuffix(host, ".acme.io") && host != "acme.example" {
		return 0, false
	}
	if host == "acme.io" && strings.TrimRight(u.Path, "/") == "/beta-terms" {
		return 404, true
	}
	return 200, true
}
