package netguard

import (
	"net/netip"
	"testing"
)

func TestCheck(t *testing.T) {
	strict, open := Policy{}, Policy{AllowPrivate: true}
	cases := []struct {
		ip           string
		strict, open bool // allowed?
	}{
		{"93.184.216.34", true, true},
		{"2606:2800:220:1:248:1893:25c8:1946", true, true},
		{"127.0.0.1", false, true},
		{"10.1.2.3", false, true},
		{"172.20.0.5", false, true},
		{"192.168.1.10", false, true},
		{"100.64.0.1", false, true},
		{"::1", false, true},
		{"fd12:3456::1", false, true},
		{"169.254.169.254", false, false}, // cloud metadata, always blocked
		{"fe80::1", false, false},
		{"0.0.0.0", false, false},
		{"224.0.0.1", false, false},
		{"::ffff:127.0.0.1", false, true},  // IPv4-mapped loopback
		{"64:ff9b::a00:1", false, true},    // NAT64 to 10.0.0.1
		{"64:ff9b::5db8:d822", true, true}, // NAT64 to a public address
	}
	for _, c := range cases {
		ip := netip.MustParseAddr(c.ip)
		if got := strict.Check(ip) == nil; got != c.strict {
			t.Errorf("strict %s: allowed=%v want %v", c.ip, got, c.strict)
		}
		if got := open.Check(ip) == nil; got != c.open {
			t.Errorf("private allowed %s: allowed=%v want %v", c.ip, got, c.open)
		}
	}
}

func TestValidateURL(t *testing.T) {
	p := Policy{}
	for _, bad := range []string{"ftp://example.com/x", "https://user:pw@example.com/", "https://127.0.0.1/hook", "http://localhost:8080/", "https://169.254.169.254/latest", "not a url"} {
		if _, err := p.ValidateURL(bad); err == nil {
			t.Errorf("%s should be rejected", bad)
		}
	}
	if _, err := p.ValidateURL("https://api.example.com/hooks/reputecast"); err != nil {
		t.Errorf("public https: %v", err)
	}
	if _, err := (Policy{AllowPrivate: true}).ValidateURL("http://127.0.0.1:9000/hook"); err != nil {
		t.Errorf("private allowed: %v", err)
	}
}
