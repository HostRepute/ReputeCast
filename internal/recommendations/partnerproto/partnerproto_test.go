package partnerproto

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCodes(t *testing.T) {
	for range 50 {
		c := NewCode()
		if n, ok := NormalizeCode(strings.ToLower(strings.ReplaceAll(c, "-", " "))); !ok || n != c {
			t.Fatalf("normalize %q: %q %v", c, n, ok)
		}
	}
	for _, bad := range []string{"", "H3RT-6QWN", "H3RT-6QWN-8ZKB-AAAA", "H3RT-6QWN-8ZK1", "H3RT-6QWN-8ZKO"} {
		if _, ok := NormalizeCode(bad); ok {
			t.Fatalf("%q read as a code", bad)
		}
	}
	base, code, err := ParseCodeURL(" https://reputecast.opsweekly.example/p/h3rt-6qwn-8zkb/ ")
	if err != nil || base != "https://reputecast.opsweekly.example" || code != "H3RT-6QWN-8ZKB" {
		t.Fatalf("parse: %q %q %v", base, code, err)
	}
	if base, _, err := ParseCodeURL("https://example.com/reputecast/p/H3RT-6QWN-8ZKB"); err != nil || base != "https://example.com/reputecast" {
		t.Fatalf("an instance under a path: %q %v", base, err)
	}
	for _, bad := range []string{"https://fabrikam.example/newsletter", "H3RT-6QWN-8ZKB", "ftp://x.example/p/H3RT-6QWN-8ZKB", "https://u:p@x.example/p/H3RT-6QWN-8ZKB"} {
		if _, _, err := ParseCodeURL(bad); !errors.Is(err, ErrNotACode) {
			t.Fatalf("%q: %v", bad, err)
		}
	}
	if CodeURL("https://a.example/", "H3RT-6QWN-8ZKB") != "https://a.example/p/H3RT-6QWN-8ZKB" {
		t.Fatal("code URL")
	}
}

func TestKeyAgreement(t *testing.T) {
	a, _ := NewKeyPair()
	b, _ := NewKeyPair()
	sa, err := a.Secret(b.Public(), ConnectSalt("H3RT-6QWN-8ZKB"), "rec", "partner")
	if err != nil {
		t.Fatal(err)
	}
	sb, _ := b.Secret(a.Public(), ConnectSalt("H3RT-6QWN-8ZKB"), "rec", "partner")
	if sa != sb || !strings.HasPrefix(sa, "rcps_") {
		t.Fatalf("secrets differ: %s %s", sa, sb)
	}
	other, _ := b.Secret(a.Public(), ConnectSalt("AAAA-BBBB-CCCC"), "rec", "partner")
	swapped, _ := b.Secret(a.Public(), ConnectSalt("H3RT-6QWN-8ZKB"), "partner", "rec")
	if other == sa || swapped == sa {
		t.Fatal("the secret isn't bound to the code and the links")
	}
	if _, err := a.Secret("not-a-key", nil, "", ""); err == nil {
		t.Fatal("a bad key was accepted")
	}
}

func TestSignatures(t *testing.T) {
	now := time.Unix(1_760_000_000, 0)
	body := []byte(`{"protocol":1,"request":"r1"}`)
	path := Prefix + "/links/l1/subscribe"
	h := SignatureHeader([]string{"new", "old"}, now.Unix(), "POST", path, body)
	if !strings.HasPrefix(h, "v1=") || strings.Count(h, "v1=") != 2 {
		t.Fatalf("header: %s", h)
	}
	ts := "1760000000"
	if err := Verify([]string{"old"}, ts, h, "POST", path, body, now.Add(time.Minute)); err != nil {
		t.Fatalf("the previous secret: %v", err)
	}
	if err := Verify([]string{"new"}, ts, h, "POST", path, body, now.Add(6*time.Minute)); !errors.Is(err, ErrStaleTimestamp) {
		t.Fatalf("stale: %v", err)
	}
	if err := Verify([]string{"new"}, ts, h, "POST", Prefix+"/links/l1/state", body, now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("another endpoint: %v", err)
	}
	if err := Verify([]string{"new"}, ts, h, "POST", path, []byte(`{"protocol":1,"request":"r2"}`), now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("another body: %v", err)
	}
	if err := Verify([]string{"new"}, "", h, "POST", path, body, now); !errors.Is(err, ErrNoSignature) {
		t.Fatalf("unsigned: %v", err)
	}
	// A response signs its status in place of the method.
	resp := SignatureHeader([]string{"new"}, now.Unix(), ResponseMethod(200), path, body)
	if err := Verify([]string{"new"}, ts, resp, "POST", path, body, now); err == nil {
		t.Fatal("a response signature passed as a request's")
	}
}
