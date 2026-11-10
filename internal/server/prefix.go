package server

import (
	"net/netip"
	"strings"
)

// ParsePrefix reads an address or a network as a person writes it:
// 203.0.113.7 (a /32), 203.0.113.0/24, 2001:db8::/48. An IPv4 address
// written as IPv6 (::ffff:203.0.113.7) is the IPv4 one, and host bits are
// cleared (203.0.113.9/24 is 203.0.113.0/24). An address with a zone
// (fe80::1%eth0) and an IPv4-mapped network wider than /96 are refused:
// neither names a network a caller can come from. API keys' allowed IPs and
// the tracking networks read addresses with it.
func ParsePrefix(s string) (netip.Prefix, bool) {
	s = strings.TrimSpace(s)
	var p netip.Prefix
	if !strings.Contains(s, "/") {
		a, err := netip.ParseAddr(s)
		if err != nil || a.Zone() != "" {
			return netip.Prefix{}, false
		}
		a = a.Unmap()
		p = netip.PrefixFrom(a, a.BitLen())
	} else {
		var err error
		if p, err = netip.ParsePrefix(s); err != nil {
			return netip.Prefix{}, false
		}
		if p.Addr().Is4In6() {
			if p.Bits() < 96 {
				return netip.Prefix{}, false
			}
			p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
		}
	}
	p = p.Masked()
	if !p.IsValid() {
		return netip.Prefix{}, false
	}
	return p, true
}

// AllowedFrom reports whether a key limited to the networks in list may be
// called from ip: from anywhere when the list is empty, and never from an
// address ReputeCast doesn't know (it fails closed).
func AllowedFrom(list []netip.Prefix, ip *netip.Addr) bool {
	if len(list) == 0 {
		return true
	}
	if ip == nil || !ip.IsValid() {
		return false
	}
	a := ip.Unmap()
	for _, p := range list {
		if p.Contains(a) {
			return true
		}
	}
	return false
}
