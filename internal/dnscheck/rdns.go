package dnscheck

import (
	"context"
	"net/netip"
	"strings"
)

// ReverseDNS is the forward-confirmed reverse DNS (FCrDNS) of an IP: its PTR
// name, and whether that name resolves back to the same IP. Mailbox
// providers expect sending IPs to pass this check.
type ReverseDNS struct {
	IP     string `json:"ip"`
	PTR    string `json:"ptr,omitempty"`
	Status string `json:"status"` // pass, fail (doesn't resolve back), missing, error
	// Matches is true when PTR resolves back to IP.
	Matches bool   `json:"matches"`
	Error   string `json:"error,omitempty"`
}

// CheckReverseDNS looks up the PTR of ip and confirms it forward.
func CheckReverseDNS(ctx context.Context, r Resolver, ip netip.Addr) ReverseDNS {
	out := ReverseDNS{IP: ip.String()}
	names, err := r.PTR(ctx, ip)
	if err != nil {
		out.Status, out.Error = statusFor(err), err.Error()
		return out
	}
	if len(names) == 0 {
		out.Status = StatusMissing
		return out
	}
	out.PTR = strings.ToLower(names[0])
	for _, name := range names {
		var addrs []netip.Addr
		if ip.Is4() {
			addrs, err = r.A(ctx, name)
		} else {
			addrs, err = r.AAAA(ctx, name)
		}
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if a == ip.Unmap() || a == ip {
				out.PTR, out.Matches, out.Status = strings.ToLower(name), true, StatusPass
				return out
			}
		}
	}
	out.Status = StatusFail
	return out
}
