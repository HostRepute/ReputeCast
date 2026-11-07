package customhost

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/store"
)

// The token is the one the pages' custom domains always had: TXT records
// published before it moved here still prove their host.
func TestToken(t *testing.T) {
	ws := uuid.MustParse("01923456-789a-7bcd-8ef0-123456789abc")
	if got := Token(ws, "news.acme.io"); got != "reputecast-verify=22f58eb03409609184493665" {
		t.Fatalf("token %q", got)
	}
	if Token(ws, "go.acme.io") == Token(ws, "news.acme.io") || Token(uuid.Must(uuid.NewV7()), "news.acme.io") == Token(ws, "news.acme.io") {
		t.Fatal("a token proves another host or another workspace's")
	}
	if got := TXTName("news.acme.io"); got != "_reputecast.news.acme.io" {
		t.Fatalf("TXT name %q", got)
	}
}

func TestTakenBefore(t *testing.T) {
	held := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	before, after := held.Add(-time.Hour), held.Add(time.Hour)
	for _, c := range []struct {
		name  string
		claim store.CustomHostClaimsRow
		held  *time.Time
		want  bool
	}{
		{"any use while the host isn't verified here", store.CustomHostClaimsRow{Verified: false}, nil, true},
		{"proven before", store.CustomHostClaimsRow{Verified: true, Since: &before}, &held, true},
		{"proven at the same moment", store.CustomHostClaimsRow{Verified: true, Since: &held}, &held, true},
		{"proven when nobody recorded", store.CustomHostClaimsRow{Verified: true}, &held, true},
		{"proven later", store.CustomHostClaimsRow{Verified: true, Since: &after}, &held, false},
		{"never proven", store.CustomHostClaimsRow{Verified: false, Since: &before}, &held, false},
	} {
		if got := TakenBefore(c.claim, c.held); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestUnderTheInstance(t *testing.T) {
	for host, want := range map[string]bool{
		"acme.reputecast.example.com":    true,
		"go.acme.reputecast.example.com": true,
		"reputecast.example.com.evil.io": false,
		"xreputecast.example.com":        false,
		"news.acme.io":                   false,
	} {
		if got := under(host, "reputecast.example.com"); got != want {
			t.Errorf("%s: %v, want %v", host, got, want)
		}
	}
	if under("acme.example.com", "") {
		t.Error("everything is under an instance without a host")
	}
	if Records.TXTOnly() || Never.TXTOnly() || !InUse.TXTOnly() || !UnderInstance.TXTOnly() {
		t.Error("TXTOnly")
	}
}

func TestHasToken(t *testing.T) {
	ctx := context.Background()
	ws, other := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	zone := dnscheck.NewZone()
	zone.Add("_reputecast.go.acme.io", "TXT", Token(other, "go.acme.io"))
	zone.Add("_reputecast.go.acme.io", "TXT", Token(ws, "go.acme.io"))
	zone.Add("_reputecast.t.acme.io", "TXT", Token(other, "t.acme.io"))
	zone.Fail("_reputecast.down.acme.io")
	for _, c := range []struct {
		host            string
		found, answered bool
	}{
		{"go.acme.io", true, true}, // one record per workspace that uses the host
		{"t.acme.io", false, true},
		{"none.acme.io", false, true},
		{"down.acme.io", false, false},
	} {
		if found, answered := HasToken(ctx, zone, ws, c.host); found != c.found || answered != c.answered {
			t.Errorf("%s: found %v answered %v, want %v %v", c.host, found, answered, c.found, c.answered)
		}
	}
}
