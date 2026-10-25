package readiness_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// digestDomains is the digest fixture's reputation over its domains: the
// sending domain and its parent (Deliverability watches both), SES's and
// Postal's return paths and the tracking domain (looked up live).
func digestDomains() readiness.Facts {
	f := digest()
	f.Reputation.Domains = []readiness.DomainAssetFacts{
		{Domain: "news.acme.io", Role: readiness.DomainSending},
		{Domain: "acme.io", Role: readiness.DomainParent},
		{Domain: "bounce.news.acme.io", Role: readiness.DomainReturnPath, Providers: []string{"SES"}, Live: true},
		{Domain: "rp.news.acme.io", Role: readiness.DomainReturnPath, Providers: []string{"Postal"}, Live: true},
		{Domain: "links.acme.io", Role: readiness.DomainTracking, Live: true},
	}
	return f
}

// SR-2: sender reputation covers the parent domain, the return-path domains
// and the tracking domain: acme.io listed shows on news.acme.io's review.
func TestReputationCoversEveryDomain(t *testing.T) {
	f := digestDomains()
	c := byKey(readiness.Assemble(f))["reputation.domain_blocklists"]
	if c.Status != readiness.Pass || c.Title != "news.acme.io and acme.io are clean on 84 of 84 domain blocklists" ||
		!strings.Contains(c.Detail, "bounce.news.acme.io (SES return path), rp.news.acme.io (Postal return path) and links.acme.io (tracking domain): clean on Spamhaus DBL and SURBL, looked up live.") {
		t.Errorf("clean: %s %q %q", c.Status, c.Title, c.Detail)
	}
	if c.Short != "news.acme.io and acme.io clean on 84 of 84 blocklists" {
		t.Errorf("short: %q", c.Short)
	}

	// acme.io on Spamhaus DBL (Deliverability's listing of the parent domain).
	f.Reputation.Listings = append(f.Reputation.Listings, readiness.ListingFacts{Asset: "acme.io", Type: "domain", Source: "Spamhaus DBL", Impact: "high"})
	res := readiness.Assemble(f)
	c = byKey(res)["reputation.domain_blocklists"]
	if c.Status != readiness.Block || c.Title != "acme.io, the parent domain of news.acme.io, is listed on Spamhaus DBL" ||
		!strings.HasPrefix(c.Detail, "Mailbox providers judge news.acme.io by its parent domain's reputation too.") {
		t.Errorf("parent listed: %s %q %q", c.Status, c.Title, c.Detail)
	}
	if res.Verdict != readiness.VerdictBlocked {
		t.Errorf("verdict %s", res.Verdict)
	}

	// A return path on a list the run looked it up on.
	f = digestDomains()
	f.Reputation.Domains[2].Lists = []string{"SURBL"}
	c = byKey(readiness.Assemble(f))["reputation.domain_blocklists"]
	if c.Status != readiness.Block || c.Title != "SES return-path domain bounce.news.acme.io is listed on SURBL" || c.Ack == "" {
		t.Errorf("return path listed: %s %q", c.Status, c.Title)
	}

	// The tracking domain, and a second listing named in the detail.
	f.Reputation.Domains[4].Lists = []string{"Spamhaus DBL"}
	c = byKey(readiness.Assemble(f))["reputation.domain_blocklists"]
	if !strings.Contains(c.Detail, "1 more domain is listed too: links.acme.io.") {
		t.Errorf("second listing: %q", c.Detail)
	}
	f.Reputation.Domains[2].Lists = nil
	c = byKey(readiness.Assemble(f))["reputation.domain_blocklists"]
	if c.Title != "The tracking domain links.acme.io is listed on Spamhaus DBL" {
		t.Errorf("tracking domain listed: %q", c.Title)
	}

	// A live lookup that failed is unknown, not clean.
	f = digestDomains()
	f.Reputation.Domains[4].Unknown = "timeout"
	c = byKey(readiness.Assemble(f))["reputation.domain_blocklists"]
	if c.Status != readiness.Unknown || c.Title != "Couldn't check links.acme.io on Spamhaus DBL and SURBL" || c.Note != "timed out" {
		t.Errorf("unknown: %s %q %q", c.Status, c.Title, c.Note)
	}
}
