package customhost

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRequestHost(t *testing.T) {
	for in, want := range map[string]string{
		"news.acme.io":        "news.acme.io",
		"News.Acme.IO:443":    "news.acme.io",
		"news.acme.io.":       "news.acme.io",
		"news.acme.io.:8443":  "news.acme.io",
		" news.acme.io ":      "news.acme.io",
		"[2001:db8::1]:443":   "2001:db8::1",
		"127.0.0.1:8080":      "127.0.0.1",
		"":                    "",
		"reputecast.example.": "reputecast.example",
	} {
		if got := RequestHost(in); got != want {
			t.Errorf("RequestHost(%q) = %q, want %q", in, got, want)
		}
	}
}

// The instance's own host is nobody's, and no Hosts at all holds nothing:
// both carry every workspace's links without asking the database (Q is
// nil here, so a query would panic).
func TestHostsWithoutTheDatabase(t *testing.T) {
	ctx := context.Background()
	ws := uuid.Must(uuid.NewV7())
	var none *Hosts
	if ok, err := none.Carries(ctx, "news.acme.io", ws); !ok || err != nil {
		t.Fatalf("no Hosts: %v, %v", ok, err)
	}
	h := &Hosts{Instance: func() string { return "https://ReputeCast.example.com/" }}
	for _, host := range []string{"reputecast.example.com", "REPUTECAST.example.com:443", "reputecast.example.com.", ""} {
		if ok, err := h.Carries(ctx, host, ws); !ok || err != nil {
			t.Errorf("%q: %v, %v", host, ok, err)
		}
		if ok, err := h.CarriesSlug(ctx, host, "acme-newsletter"); !ok || err != nil {
			t.Errorf("%q by slug: %v, %v", host, ok, err)
		}
	}
}
