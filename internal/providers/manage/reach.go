package manage

// Draft (PR-9): webhook reachability is proven from outside, never by
// ReputeCast calling its own URL. What ReputeCast can tell by itself is
// whether the address could be reached at all (a public name and address);
// the provider proves the rest by delivering (a test event on request, the
// SNS handshake, or any event).

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// AddressProblem is why no provider can reach a webhook address: "" when
// nothing is wrong with it as far as DNS tells.
type AddressProblem struct {
	// Kind: local (localhost and the like), private (a private address),
	// dns (the name doesn't resolve) or invalid.
	Kind string
	Host string
	IP   string
}

// CheckAddress looks at a webhook address without calling it.
func CheckAddress(ctx context.Context, raw string) (AddressProblem, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return AddressProblem{Kind: "invalid"}, false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") ||
		strings.HasSuffix(host, ".lan") {
		return AddressProblem{Kind: "local", Host: host}, false
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		if !PublicIP(ip) {
			return AddressProblem{Kind: "private", Host: host, IP: ip.String()}, false
		}
		return AddressProblem{}, true
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	addrs, err := net.DefaultResolver.LookupNetIP(cctx, "ip", host)
	cancel()
	if err != nil || len(addrs) == 0 {
		return AddressProblem{Kind: "dns", Host: host}, false
	}
	for _, a := range addrs {
		if PublicIP(a) {
			return AddressProblem{}, true
		}
	}
	return AddressProblem{Kind: "private", Host: host, IP: addrs[0].Unmap().String()}, false
}
