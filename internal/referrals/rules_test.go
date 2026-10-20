package referrals

import (
	"net/netip"
	"strings"
	"testing"
)

func TestNewCode(t *testing.T) {
	seen := map[string]bool{}
	for range 2000 {
		c := NewCode()
		if len(c) != CodeLength {
			t.Fatalf("code %q has %d characters", c, len(c))
		}
		if strings.ContainsAny(c, "01OIL") {
			t.Fatalf("code %q has a look-alike", c)
		}
		if _, ok := NormalizeCode(c); !ok {
			t.Fatalf("code %q isn't a valid code", c)
		}
		seen[c] = true
	}
	if len(seen) < 1990 {
		t.Fatalf("only %d distinct codes in 2000", len(seen))
	}
}

func TestNormalizeCode(t *testing.T) {
	for in, want := range map[string]string{"K7Q2MX": "K7Q2MX", " k7q2mx ": "K7Q2MX", "K7Q2MX/": "K7Q2MX"} {
		if got, ok := NormalizeCode(in); !ok || got != want {
			t.Errorf("NormalizeCode(%q) = %q, %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "K7Q2M", "K7Q2MXX", "K7Q2M0", "K7Q2MO", "K7Q2M1", "K7Q2MI", "K7Q2ML", "K7Q-MX"} {
		if _, ok := NormalizeCode(in); ok {
			t.Errorf("NormalizeCode(%q) accepted", in)
		}
	}
}

func TestSameMailbox(t *testing.T) {
	same := [][2]string{
		{"sam.rivera@outlook.com", "Sam.Rivera@Outlook.com"},
		{"sam.rivera@outlook.com", "sam.rivera+friends@outlook.com"},
		{"sam.rivera@gmail.com", "samrivera@gmail.com"},
		{"s.a.m.rivera+x@googlemail.com", "samrivera@gmail.com"},
	}
	for _, p := range same {
		if !SameMailbox(p[0], p[1]) {
			t.Errorf("%s and %s should be the same mailbox", p[0], p[1])
		}
	}
	different := [][2]string{
		{"sam.rivera@outlook.com", "samrivera@outlook.com"}, // dots matter outside Gmail
		{"sam@acme.io", "sam@acme.com"},
		{"", ""},
	}
	for _, p := range different {
		if SameMailbox(p[0], p[1]) {
			t.Errorf("%s and %s should be different mailboxes", p[0], p[1])
		}
	}
}

func TestNumberedKey(t *testing.T) {
	if a, b := NumberedKey("ben.t01@outlook.com"), NumberedKey("ben.t06@outlook.com"); a == "" || a != b {
		t.Fatalf("numbered addresses differ: %q %q", a, b)
	}
	if NumberedKey("ben.t01@outlook.com") == NumberedKey("ben.t01@gmail.com") {
		t.Fatal("different domains share a key")
	}
	if NumberedKey("a.adeyemi@contoso.example") != "" {
		t.Fatal("an address without digits has a key")
	}
	if NumberedKey("t.okoro@contoso.example") == NumberedKey("f.nwosu@contoso.example") && NumberedKey("t.okoro@contoso.example") != "" {
		t.Fatal("different names share a key")
	}
	if got := NumberedKey("j2smith77+x@gmail.com"); got != "j#smith#@gmail.com" {
		t.Fatalf("NumberedKey = %q", got)
	}
}

func TestNetworkOf(t *testing.T) {
	for in, want := range map[string]string{
		"203.0.113.77":          "203.0.113.0/24",
		"::ffff:198.51.100.12":  "198.51.100.0/24",
		"2001:db8:1234:5678::1": "2001:db8:1234::/48",
	} {
		if got := NetworkOf(netip.MustParseAddr(in)); got != want {
			t.Errorf("NetworkOf(%s) = %s, want %s", in, got, want)
		}
	}
	if NetworkOf(netip.Addr{}) != "" {
		t.Error("an invalid address has a network")
	}
}

func TestChannelOf(t *testing.T) {
	for in, want := range map[string]string{"": "copy", "X": "x", "twitter": "x", "linkedin": "linkedin", "WhatsApp": "whatsapp", "email": "email", "telegram": "other"} {
		if got := ChannelOf(in); got != want {
			t.Errorf("ChannelOf(%q) = %q, want %q", in, got, want)
		}
	}
}
