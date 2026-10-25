package readiness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// sampleZone is news.acme.io as SES and Postal need it (the digest fixture's
// providers): Easy DKIM CNAMEs and a custom MAIL FROM for SES, a DKIM key and
// a return path for Postal, DMARC p=quarantine on acme.io.
func sampleZone() (*dnscheck.Zone, []dnscheck.Requirement) {
	key := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA" + strings.Repeat("u3Zq8xN2pL7vR4kT9wY6bH1mC5", 13) + "IDAQAB"
	z := dnscheck.NewZone()
	z.Add("news.acme.io", "TXT", "v=spf1 include:amazonses.com ip4:203.0.113.25 -all")
	z.Add("amazonses.com", "TXT", "v=spf1 ip4:54.240.0.0/18 -all")
	z.Add("_dmarc.acme.io", "TXT", "v=DMARC1; p=quarantine; sp=quarantine")
	reqs := []dnscheck.Requirement{}
	for _, tok := range []string{"7kq2n3x", "p9m4ab2", "z1c8e7h"} {
		z.Add(tok+"._domainkey.news.acme.io", "CNAME", tok+".dkim.amazonses.com")
		z.Add(tok+".dkim.amazonses.com", "TXT", "p="+key)
		reqs = append(reqs, dnscheck.Requirement{ProviderID: "p-ses", Purpose: "dkim", Type: "CNAME", Name: tok + "._domainkey.news.acme.io", Value: tok + ".dkim.amazonses.com"})
	}
	z.Add("bounce.news.acme.io", "MX", "10 feedback-smtp.us-east-1.amazonses.com")
	z.Add("bounce.news.acme.io", "TXT", "v=spf1 include:amazonses.com ~all")
	reqs = append(reqs,
		dnscheck.Requirement{ProviderID: "p-ses", Purpose: "return_path", Type: "MX", Name: "bounce.news.acme.io", Value: "10 feedback-smtp.us-east-1.amazonses.com"},
		dnscheck.Requirement{ProviderID: "p-ses", Purpose: "spf", Type: "TXT", Name: "bounce.news.acme.io", Value: "v=spf1 include:amazonses.com ~all"})
	z.Add("postal-2026._domainkey.news.acme.io", "TXT", "v=DKIM1; p="+key)
	z.Add("rp.news.acme.io", "MX", "10 mta.acme.io")
	z.Add("rp.news.acme.io", "TXT", "v=spf1 ip4:203.0.113.25 -all")
	reqs = append(reqs,
		dnscheck.Requirement{ProviderID: "p-postal", Purpose: "dkim", Type: "TXT", Name: "postal-2026._domainkey.news.acme.io", Value: "v=DKIM1; p=" + key},
		dnscheck.Requirement{ProviderID: "p-postal", Purpose: "return_path", Type: "MX", Name: "rp.news.acme.io", Value: "10 mta.acme.io"})
	return z, reqs
}

// authOf is the digest fixture with news.acme.io checked through r.
func authOf(t *testing.T, r dnscheck.Resolver, reqs []dnscheck.Requirement) (readiness.Facts, dnscheck.DomainAuth) {
	t.Helper()
	f := digest()
	auth := dnscheck.CheckDomain(context.Background(), r, "news.acme.io", reqs)
	checked := f.Auth.CheckedAt
	f.Auth = readiness.AuthFactsOf("news.acme.io", auth, nil, f.Route.Shares)
	f.Auth.CheckedAt = checked
	return f, auth
}

// SR-6: a DNS lookup that fails (a timeout, SERVFAIL) is Unknown, never a
// Block saying the record is missing or invalid; the run looks the domain
// up again, and records DNS answers for again pass.
func TestDNSLookupErrorIsUnknown(t *testing.T) {
	ctx := context.Background()
	z, reqs := sampleZone()
	if f, auth := authOf(t, z, reqs); readiness.LookupFailed(auth) {
		t.Fatalf("working DNS has lookup errors: %+v", auth)
	} else if res := readiness.Assemble(f); res.Counts.Blocks != 0 || res.Counts.Unknown != 0 {
		t.Fatalf("working DNS: %+v", res.Counts)
	}

	// Deliverability checked the domain while its resolver failed for the
	// DMARC record and SES's DKIM keys.
	z.Fail("_dmarc.news.acme.io").Fail("_dmarc.acme.io")
	z.Fail("7kq2n3x._domainkey.news.acme.io")
	f, stored := authOf(t, z, reqs)
	if !readiness.LookupFailed(stored) {
		t.Fatal("the failed lookups aren't reported")
	}
	res := readiness.Assemble(f)
	checks := byKey(res)
	for key, record := range map[string]string{"auth.dmarc": "DMARC", "auth.dkim": "DKIM"} {
		c := checks[key]
		if c.Status != readiness.Unknown {
			t.Errorf("%s: %s %q, want unknown", key, c.Status, c.Title)
		}
		if strings.Contains(strings.ToLower(c.Title), "invalid") || strings.Contains(c.Title, "isn't verified") || !strings.Contains(c.Title, "Couldn't look up the "+record) {
			t.Errorf("%s says the record is wrong: %q", key, c.Title)
		}
		if !strings.Contains(c.Detail, "SERVFAIL") || !strings.Contains(c.Detail, "send-time recheck") {
			t.Errorf("%s detail: %q", key, c.Detail)
		}
		if c.Note != "DNS error" {
			t.Errorf("%s note: %q", key, c.Note)
		}
	}
	if checks["auth.dkim"].Title != "Couldn't look up the DKIM records of SES" {
		t.Errorf("DKIM names the provider whose keys DNS didn't answer for: %q", checks["auth.dkim"].Title)
	}
	for _, c := range res.Checks {
		if c.Group == readiness.GroupAuthentication && c.Status == readiness.Block {
			t.Errorf("a DNS error blocks: %s %q", c.Key, c.Title)
		}
	}
	if !res.Incomplete || res.Verdict == readiness.VerdictReady || res.Problem != "Couldn't look up the DKIM records of SES." {
		t.Errorf("verdict %s, incomplete %v, problem %q", res.Verdict, res.Incomplete, res.Problem)
	}

	// The run looks the domain up again: still failing, still unknown…
	again := readiness.Relookup(ctx, z, "news.acme.io", stored)
	if !readiness.LookupFailed(again) {
		t.Fatal("still failing DNS reads as answered")
	}
	// …and once DNS answers, the records pass.
	ok, _ := sampleZone()
	fresh := readiness.Relookup(ctx, ok, "news.acme.io", stored)
	if readiness.LookupFailed(fresh) || fresh.DMARC.Status != dnscheck.StatusPass {
		t.Fatalf("relookup with working DNS: %+v", fresh.DMARC)
	}
	f.Auth = readiness.AuthFactsOf("news.acme.io", fresh, nil, f.Route.Shares)
	if c := byKey(readiness.Assemble(f))["auth.dmarc"]; c.Status != readiness.Pass {
		t.Errorf("DMARC after the relookup: %s %q", c.Status, c.Title)
	}

	// A record DNS answers is missing still blocks.
	missing, reqs := sampleZone()
	gone := dnscheck.NewZone()
	for _, name := range []string{"news.acme.io", "amazonses.com"} {
		txt, _ := missing.TXT(ctx, name)
		for _, v := range txt {
			gone.Add(name, "TXT", v)
		}
	}
	f, _ = authOf(t, gone, reqs)
	if c := byKey(readiness.Assemble(f))["auth.dmarc"]; c.Status != readiness.Block || !strings.HasPrefix(c.Title, "No DMARC record") {
		t.Errorf("a missing DMARC record: %s %q", c.Status, c.Title)
	}
}

// SR-7: an invalid BIMI record is a warning (no logo shows), not a pass that
// says BIMI isn't required, and not a block.
func TestInvalidBIMIWarns(t *testing.T) {
	z, reqs := sampleZone()
	z.Add("default._bimi.news.acme.io", "TXT", "v=BIMI1; l=http://acme.io/logo.svg")
	f, _ := authOf(t, z, reqs)
	c := byKey(readiness.Assemble(f))["auth.bimi"]
	if c.Status != readiness.Warn || c.Title != "The BIMI record of news.acme.io is invalid" || !strings.Contains(c.Detail, "https") || c.Ack == "" {
		t.Errorf("invalid BIMI: %s %q %q ack %q", c.Status, c.Title, c.Detail, c.Ack)
	}
}
