package server

import (
	"net/netip"
	"testing"
)

func TestParsePrefix(t *testing.T) {
	for in, want := range map[string]string{
		"203.0.113.7":            "203.0.113.7/32",
		" 203.0.113.7 ":          "203.0.113.7/32",
		"::ffff:203.0.113.7":     "203.0.113.7/32",
		"::ffff:203.0.113.0/120": "203.0.113.0/24",
		"2001:db8::/48":          "2001:db8::/48",
		"2001:db8::1":            "2001:db8::1/128",
		"203.0.113.9/24":         "203.0.113.0/24",
		"2001:db8:1:2:3::9/64":   "2001:db8:1:2::/64",
	} {
		p, ok := ParsePrefix(in)
		if !ok || p.String() != want {
			t.Errorf("ParsePrefix(%q) = %v, %v; want %s", in, p, ok, want)
		}
	}
	for _, in := range []string{"", "not-an-ip", "203.0.113.0/33", "203.0.113", "::ffff:203.0.113.0/90", "fe80::1%eth0", "fe80::1%eth0/64", "203.0.113.7/"} {
		if p, ok := ParsePrefix(in); ok {
			t.Errorf("ParsePrefix(%q) = %v, want refused", in, p)
		}
	}
}

func TestAllowedFrom(t *testing.T) {
	list := []netip.Prefix{netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("2001:db8::/48")}
	addr := func(s string) *netip.Addr { a := netip.MustParseAddr(s); return &a }
	for _, c := range []struct {
		list []netip.Prefix
		ip   *netip.Addr
		want bool
	}{
		{nil, nil, true},
		{nil, addr("198.51.100.4"), true},
		{list, nil, false},
		{list, &netip.Addr{}, false},
		{list, addr("203.0.113.200"), true},
		{list, addr("::ffff:203.0.113.200"), true},
		{list, addr("2001:db8:0:5::1"), true},
		{list, addr("198.51.100.4"), false},
		{list, addr("2001:db9::1"), false},
	} {
		if got := AllowedFrom(c.list, c.ip); got != c.want {
			t.Errorf("AllowedFrom(%v, %v) = %v, want %v", c.list, c.ip, got, c.want)
		}
	}
}
