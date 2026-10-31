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
