package readiness

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/sending/engine"
)

// SR-7: the size is measured on the HTML the send renders, with the links
// rewritten to click redirects and the open pixel added.
func TestTrackedSize(t *testing.T) {
	urls := publicurl.New("https://news.acme.io")
	html := `<html><body><p>Hi {{ first_name | default: "there" }}</p>` + strings.Repeat(`<a href="https://acme.io/blog/post">Read</a>`, 20) +
		`<a href="{{ unsubscribe_url }}">Unsubscribe</a></body></html>`
	both := trackedSize(html, engine.Tracking{Opens: true, Clicks: true}, urls, "acme")
	if both <= len(html)+20*60 {
		t.Fatalf("20 tracked links and a pixel add more than that: %d → %d", len(html), both)
	}
	opens := trackedSize(html, engine.Tracking{Opens: true}, urls, "acme")
	if opens >= both || opens <= len(html) {
		t.Fatalf("pixel only: %d (both %d, html %d)", opens, both, len(html))
	}
	if n := trackedSize(html, engine.Tracking{}, urls, "acme"); n != 0 {
		t.Fatalf("nothing tracked: %d", n)
	}
	if n := trackedSize(`<p>{% if x %}</p>`, engine.Tracking{Opens: true, Clicks: true}, urls, "acme"); n != 0 {
		t.Fatalf("broken Liquid: %d", n)
	}
}

// The personalization fields an email prints: a use without a fallback
// anywhere means the field has none; unknown names and system links aren't
// fields.
func TestFieldUses(t *testing.T) {
	known := map[string]bool{"first_name": true, "company": true}
	got := fieldUses(known, `Hi {{ first_name | default: "there" }}`, `<p>{{ contact.first_name }} at {{ company | default: "your team" }}</p>
<a href="{{ unsubscribe_url }}">x</a> {{ frist_name }}`)
	if len(got) != 2 || got[0].Field != "first_name" || got[0].HasFallback || got[0].Fallback != "there" || got[1].Field != "company" || !got[1].HasFallback {
		t.Fatalf("uses: %+v", got)
	}
}

// A tracking domain resolves, doesn't (missing) or DNS doesn't answer.
func TestResolves(t *testing.T) {
	z := dnscheck.NewZone()
	z.Add("go.news.acme.io", "CNAME", "news.acme.io")
	z.Add("news.acme.io", "A", "203.0.113.10")
	z.Add("v6.acme.io", "AAAA", "2001:db8::1")
	z.Fail("broken.acme.io")
	ctx := context.Background()
	for host, want := range map[string]string{"news.acme.io": "ok", "v6.acme.io": "ok", "nowhere.acme.io": "missing"} {
		if got := resolves(ctx, z, host); got != want {
			t.Errorf("%s: %s, want %s", host, got, want)
		}
	}
	if got := resolves(ctx, z, "broken.acme.io"); got == "ok" || got == "missing" {
		t.Errorf("a failing lookup is DNS trouble: %s", got)
	}
}

func TestDMARCMode(t *testing.T) {
	if dmarcMode("v=DMARC1; p=reject; aspf=s; adkim=r", "aspf") != "s" || dmarcMode("v=DMARC1; p=none", "aspf") != "r" || dmarcMode("v=DMARC1; ADKIM = S", "adkim") != "s" {
		t.Fatal("alignment modes")
	}
}
