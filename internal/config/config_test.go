package config

import (
	"net/netip"
	"strings"
	"testing"
)

// PR-9: REPUTECAST_PUBLIC_URL is the public URL's name everywhere; the
// earlier REPUTECAST_HTTP__PUBLIC_URL still works on its own.
func TestPublicURLName(t *testing.T) {
	t.Setenv("REPUTECAST_PUBLIC_URL", "https://news.example.com")
	c, err := Load("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTP.PublicURL != "https://news.example.com" {
		t.Errorf("REPUTECAST_PUBLIC_URL: got %q", c.HTTP.PublicURL)
	}

	t.Setenv("REPUTECAST_PUBLIC_URL", "")
	t.Setenv("REPUTECAST_HTTP__PUBLIC_URL", "https://old.example.com")
	if c, err = Load("", nil); err != nil {
		t.Fatal(err)
	}
	if c.HTTP.PublicURL != "https://old.example.com" {
		t.Errorf("REPUTECAST_HTTP__PUBLIC_URL alone: got %q", c.HTTP.PublicURL)
	}

	t.Setenv("REPUTECAST_PUBLIC_URL", "https://news.example.com")
	if c, err = Load("", nil); err != nil {
		t.Fatal(err)
	}
	if c.HTTP.PublicURL != "https://news.example.com" {
		t.Errorf("both set: the documented name wins, got %q", c.HTTP.PublicURL)
	}
}

// DP-7: /metrics has its own listener by default, and a bearer token for the
// main listener must be long enough to guess-proof.
func TestMetricsDefaults(t *testing.T) {
	c, err := Load("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Metrics.Enabled || c.Metrics.Addr != ":9464" || c.Metrics.Token != "" {
		t.Fatalf("metrics defaults = %+v", c.Metrics)
	}
	t.Setenv("REPUTECAST_METRICS__TOKEN", "short")
	c, err = Load("", nil)
	if err == nil {
		err = c.Validate()
	}
	if err == nil {
		t.Fatal("a 5-character metrics token passed validation")
	}
	t.Setenv("REPUTECAST_METRICS__TOKEN", "0123456789abcdef")
	c, err = Load("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("a 16-character token: %v", err)
	}
}

// M64 G-37: platforms set PORT; REPUTECAST_HTTP__ADDR still wins, and a
// PORT that isn't a port number is refused.
func TestPortFromPlatform(t *testing.T) {
	t.Setenv("PORT", "9000")
	c, err := Load("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTP.Addr != ":9000" {
		t.Errorf("PORT=9000 without REPUTECAST_HTTP__ADDR: addr %q, want :9000", c.HTTP.Addr)
	}
	t.Setenv("REPUTECAST_HTTP__ADDR", ":8181")
	if c, err = Load("", nil); err != nil {
		t.Fatal(err)
	}
	if c.HTTP.Addr != ":8181" {
		t.Errorf("REPUTECAST_HTTP__ADDR set: addr %q, want :8181", c.HTTP.Addr)
	}
	t.Setenv("REPUTECAST_HTTP__ADDR", "")
	t.Setenv("PORT", "http")
	if _, err := Load("", nil); err == nil {
		t.Error("PORT=http loaded")
	}
}

// M64 G-37: "private" trusts the networks in front of a platform's router,
// never a public address.
func TestTrustedProxiesPrivate(t *testing.T) {
	t.Setenv("REPUTECAST_HTTP__TRUSTED_PROXIES", "private,203.0.113.7,nonsense")
	c, err := Load("", nil)
	if err != nil {
		t.Fatal(err)
	}
	nets, invalid := ParseTrustedProxies(c.HTTP.TrustedProxies)
	trusted := func(s string) bool {
		ip := netip.MustParseAddr(s)
		for _, p := range nets {
			if p.Contains(ip) {
				return true
			}
		}
		return false
	}
	for _, ip := range []string{"10.1.2.3", "172.20.0.5", "192.168.1.1", "100.64.3.4", "127.0.0.1", "fd12::1", "::1", "203.0.113.7"} {
		if !trusted(ip) {
			t.Errorf("%s isn't trusted", ip)
		}
	}
	for _, ip := range []string{"8.8.8.8", "172.32.0.1", "2001:db8::1", "203.0.113.8"} {
		if trusted(ip) {
			t.Errorf("%s is trusted", ip)
		}
	}
	if len(invalid) != 1 || invalid[0] != "nonsense" {
		t.Errorf("invalid = %v, want [nonsense]", invalid)
	}
}

// M64 G-35: the storage backend is local by default; s3 needs a bucket and
// takes keys in pairs.
func TestStorageConfig(t *testing.T) {
	c, err := Load("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Storage.Backend != StorageLocal {
		t.Fatalf("default backend %q", c.Storage.Backend)
	}
	t.Setenv("REPUTECAST_STORAGE__BACKEND", "s3")
	if _, err := Load("", nil); err == nil || !strings.Contains(err.Error(), "storage.s3.bucket") {
		t.Errorf("s3 without a bucket: %v", err)
	}
	t.Setenv("REPUTECAST_STORAGE__S3__BUCKET", "media")
	t.Setenv("REPUTECAST_STORAGE__S3__ACCESS_KEY_ID", "AKIA")
	if _, err := Load("", nil); err == nil || !strings.Contains(err.Error(), "go together") {
		t.Errorf("an access key without its secret: %v", err)
	}
	t.Setenv("REPUTECAST_STORAGE__S3__SECRET_ACCESS_KEY", "secret")
	t.Setenv("REPUTECAST_STORAGE__S3__ENDPOINT", "http://minio:9000")
	t.Setenv("REPUTECAST_STORAGE__S3__PATH_STYLE", "true")
	if c, err = Load("", nil); err != nil {
		t.Fatal(err)
	}
	if c.Storage.S3.Bucket != "media" || c.Storage.S3.Endpoint != "http://minio:9000" || !c.Storage.S3.PathStyle {
		t.Errorf("s3 config = %+v", c.Storage.S3)
	}
	t.Setenv("REPUTECAST_STORAGE__BACKEND", "gcs")
	if _, err := Load("", nil); err == nil {
		t.Error("backend gcs loaded")
	}
}
