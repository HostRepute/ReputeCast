package manage

import (
	"encoding/json"
	"net/netip"
	"testing"
)

func TestMaskEmail(t *testing.T) {
	for in, want := range map[string]string{
		"r.wong@outlook.com": "r.w···@outlook.com",
		"ab@x.io":            "a···@x.io",
		"not-an-email":       "not-an-email",
	} {
		if got := MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlaceholderSafeKeepsTemplatesParseable(t *testing.T) {
	ok := `{"to": "{{ recipient }}", "headers": {{ headers_json }}, "n": 1}`
	if !json.Valid([]byte(placeholderSafe(ok))) {
		t.Fatalf("placeholderSafe(%s) = %s", ok, placeholderSafe(ok))
	}
	if json.Valid([]byte(placeholderSafe(`{"to": {{ recipient }`))) {
		t.Fatal("a broken template passed")
	}
}

func TestParseEndpoint(t *testing.T) {
	m, u, err := parseEndpoint("post https://api.example-esp.com/v1/messages")
	if err != nil || m != "POST" || u != "https://api.example-esp.com/v1/messages" {
		t.Fatalf("%s %s %v", m, u, err)
	}
	if _, _, err := parseEndpoint("GET https://x.io"); err == nil {
		t.Fatal("GET accepted")
	}
	if _, _, err := parseEndpoint("POST http://x.io"); err == nil {
		t.Fatal("plain http accepted")
	}
}

func TestPublicIP(t *testing.T) {
	for addr, want := range map[string]bool{"203.0.113.25": true, "8.8.8.8": true, "127.0.0.1": false, "10.1.2.3": false, "169.254.169.254": false, "::1": false, "fd00::1": false} {
		if got := PublicIP(netip.MustParseAddr(addr)); got != want {
			t.Errorf("PublicIP(%s) = %v", addr, got)
		}
	}
}

func TestDemoManifest(t *testing.T) {
	if !isDemoManifest(json.RawMessage(`{"demo": true}`)) || isDemoManifest(json.RawMessage(`{"name": "x"}`)) {
		t.Fatal("demo manifest detection")
	}
}
