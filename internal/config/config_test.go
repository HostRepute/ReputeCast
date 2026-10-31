package config

import "testing"

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
