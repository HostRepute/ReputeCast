package demo

import "github.com/hostrepute/reputecast/internal/dnscheck"

// ReadinessLinkDNS answers the URI blocklist lookups of link domains in the
// demo and in tests (SR-1): go.acme-promo.link, the flash sale's "Claim
// your discount" link, is on Spamhaus DBL and SURBL, as the boards draw it,
// and every other domain is clean. No lookup reaches the real lists, whose
// operators refuse lookups through public resolvers anyway.
func ReadinessLinkDNS(r dnscheck.Resolver) dnscheck.Resolver {
	z := dnscheck.NewZone()
	z.Add("go.acme-promo.link.dbl.spamhaus.org", "A", "127.0.1.2")
	z.Add("go.acme-promo.link.multi.surbl.org", "A", "127.0.0.8")
	return dnscheck.Overlay(z, []string{"dbl.spamhaus.org", "multi.surbl.org"}, r)
}
