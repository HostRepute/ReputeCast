package readiness_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// techRows are a check's technical details by label.
func techRows(c readiness.ReadinessCheck) map[string]string {
	out := map[string]string{}
	for _, r := range c.Technical {
		out[r.Label] = r.Value
	}
	return out
}

// R1.U2: one plain sentence per row, and what it rests on behind
// "Technical details": the DNS records of authentication, the unsubscribe
// headers as the send writes them, what each link answered and the
// provider's error of a route that's down.
func TestTechnicalDetailsOutOfSentences(t *testing.T) {
	f := digest()
	f.Auth.DMARCDomain = "acme.io"
	f.Compliance.Mailto = "hello@acme.io"
	res := stored(t, readiness.Assemble(f))

	// SPF and DMARC: the records as dig answers them, not as the detail.
	spf := check(t, res, "auth.spf")
	if spf.Status != readiness.Pass || spf.Detail != "" || spf.Mono ||
		strings.Join(spf.Raw, "\n") != "$ dig +short news.acme.io TXT\n\"v=spf1 include:amazonses.com include:spf.mta.acme.io -all\"" {
		t.Fatalf("SPF: %+v", spf)
	}
	dmarc := check(t, res, "auth.dmarc")
	if dmarc.Status != readiness.Pass || dmarc.Detail != "" || strings.Join(dmarc.Raw, "\n") != "$ dig +short _dmarc.acme.io TXT\n\"v=DMARC1; p=quarantine; adkim=s\"" {
		t.Fatalf("DMARC: %+v", dmarc)
	}
	// Each provider's DKIM keys and return path.
	if c := check(t, res, "auth.dkim"); c.Detail != "" || techRows(c)["SES"] != "3 Easy DKIM CNAMEs verified" || techRows(c)["Postal"] != "selector postal-2026 verified" {
		t.Fatalf("DKIM: %+v", c)
	}
	if c := check(t, res, "auth.mail_from"); c.Detail != "" || techRows(c)["SES"] != "bounce.news.acme.io" || techRows(c)["Postal"] != "rp.news.acme.io" {
		t.Fatalf("MAIL FROM: %+v", c)
	}
	// The one-click unsubscribe headers as sent.
	lu := check(t, res, "compliance.list_unsubscribe")
	if lu.Status != readiness.Pass || lu.Detail != "" || lu.Mono ||
		strings.Join(lu.Raw, "\n") != "List-Unsubscribe: <https://news.acme.io/u/{token}>, <mailto:hello@acme.io?subject=unsubscribe%20{token}>\nList-Unsubscribe-Post: List-Unsubscribe=One-Click" {
		t.Fatalf("one-click: %+v", lu)
	}

	// A failing SPF record: what to add, in a sentence; the record in the
	// details.
	f.Auth.SPF[1] = readiness.ProviderAuthFacts{Short: "Postal", DetailKind: readiness.AuthSPFAdd, Value: "include:spf.mta.acme.io", Name: "news.acme.io"}
	f.Auth.SPFRecord = "v=spf1 include:amazonses.com -all"
	c := check(t, stored(t, readiness.Assemble(f)), "auth.spf")
	if c.Status != readiness.Block || c.Detail != "Postal: add include:spf.mta.acme.io to the SPF record of news.acme.io." ||
		strings.Join(c.Raw, "\n") != "$ dig +short news.acme.io TXT\n\"v=spf1 include:amazonses.com -all\"" {
		t.Fatalf("failing SPF: %+v", c)
	}

	// A lookup the DNS server failed: the sentence says so, the details name
	// the answer.
	f = digest()
	f.Auth.DMARCStatus, f.Auth.DMARCError = "error", readiness.TroubleServFail
	c = check(t, stored(t, readiness.Assemble(f)), "auth.dmarc")
	if c.Status != readiness.Unknown || strings.Contains(c.Detail, "SERVFAIL") || !strings.HasPrefix(c.Detail, "The DNS server failed, ") || techRows(c)["DNS answer"] != "SERVFAIL" {
		t.Fatalf("SERVFAIL: %+v", c)
	}

	// Links: what each one answered.
	f = digest()
	f.Content.Report.LinkList = append(links("https://acme.io/blog"),
		contentcheck.ContentCheckLink{URL: "https://acme.io/old", Host: "acme.io", Kind: "link", Path: []int{1}, Probed: true, Status: 404, Broken: true},
		contentcheck.ContentCheckLink{URL: "https://down.acme.io/x", Host: "down.acme.io", Kind: "link", Path: []int{2}, Probed: true, Broken: true})
	c = check(t, stored(t, readiness.Assemble(f)), "links.respond")
	if c.Status != readiness.Warn || strings.Contains(c.Detail, "404") || !strings.Contains(c.Detail, "acme.io/old returns an error") ||
		techRows(c)["acme.io/old"] != "HTTP 404" || techRows(c)["down.acme.io/x"] != "No response" {
		t.Fatalf("broken links: %+v", c)
	}

	// A route that's down: the provider's error.
	f = digest()
	down := now.Add(-48 * time.Minute)
	f.Route.Shares[1].Status, f.Route.Shares[1].Available, f.Route.Shares[1].DownSince = state.Down, false, &down
	f.Route.Shares[1].Down = &readiness.DownFacts{Since: down, Who: "mta.acme.io", Code: "ECONNREFUSED"}
	routed := stored(t, readiness.Assemble(f))
	c = check(t, routed, "routing.health")
	if strings.Contains(c.Detail, "ECONNREFUSED") || !strings.HasPrefix(c.Detail, "mta.acme.io has refused connections since ") || techRows(c)["Error"] != "ECONNREFUSED" {
		t.Fatalf("down: %+v", c)
	}
	// An outage kept as English (an older run's facts) splits the same way.
	f.Route.Shares[1].Down = nil
	f.Route.Shares[1].DownDetail = "mta.acme.io has refused connections since 08:52 (ECONNREFUSED on port 25 and 443)"
	c = check(t, stored(t, readiness.Assemble(f)), "routing.health")
	if !strings.HasPrefix(c.Detail, "mta.acme.io has refused connections since 08:52. ") || techRows(c)["Error"] != "ECONNREFUSED on port 25 and 443" {
		t.Fatalf("down, kept as English: %+v", c)
	}

	// The labels read in Vietnamese.
	vi := routed.In(i18n.For(i18n.Vietnamese))
	if r := techRows(check(t, vi, "routing.health")); r["Lỗi"] != "ECONNREFUSED" {
		t.Errorf("in Vietnamese: %+v", r)
	}
}
