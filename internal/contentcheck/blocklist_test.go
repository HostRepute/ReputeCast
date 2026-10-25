package contentcheck_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/store"
)

// cache is link_domain_listings in memory.
type cache struct {
	store.Querier
	rows   map[string]store.LinkDomainListing
	writes int
}

func (c *cache) ContentCheckLinkListings(_ context.Context, domains []string) ([]store.LinkDomainListing, error) {
	out := []store.LinkDomainListing{}
	for _, d := range domains {
		if r, ok := c.rows[d]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (c *cache) ContentCheckUpsertLinkListing(_ context.Context, p store.ContentCheckUpsertLinkListingParams) error {
	c.writes++
	c.rows[p.Domain] = store.LinkDomainListing{Domain: p.Domain, Lists: p.Lists, Source: p.Source, CheckedAt: p.CheckedAt}
	return nil
}

var lookupNow = time.Date(2026, 10, 13, 13, 40, 0, 0, time.UTC)

func listsZone() *dnscheck.Zone {
	z := dnscheck.NewZone()
	// Spamhaus DBL and SURBL both list go.acme-promo.link; SURBL alone lists bad.example.
	z.Add("go.acme-promo.link.dbl.spamhaus.org", "A", "127.0.1.2")
	z.Add("go.acme-promo.link.multi.surbl.org", "A", "127.0.0.8")
	z.Add("bad.example.multi.surbl.org", "A", "127.0.0.16")
	// Everything else is NXDOMAIN: not listed.
	return z
}

func live(c *cache) *contentcheck.LiveBlocklist {
	return &contentcheck.LiveBlocklist{Q: c, DNS: listsZone(), Now: func() time.Time { return lookupNow }}
}

// SR-1: link domains are looked up live on Spamhaus DBL and SURBL and kept
// in link_domain_listings; a failed lookup is unknown and isn't kept; a
// fresh answer is reused, a stale one looked up again.
func TestLiveBlocklist(t *testing.T) {
	ctx := context.Background()
	c := &cache{rows: map[string]store.LinkDomainListing{}}
	b := live(c)
	rep, err := b.Check(ctx, []string{"go.acme-promo.link", "acme.io", "bad.example", "203.0.113.25", "ACME.io"})
	if err != nil {
		t.Fatal(err)
	}
	if got := rep.Listed["go.acme-promo.link"]; !slices.Equal(got, []string{"Spamhaus DBL", "SURBL"}) {
		t.Errorf("go.acme-promo.link: %v", got)
	}
	if got := rep.Listed["bad.example"]; !slices.Equal(got, []string{"SURBL"}) {
		t.Errorf("bad.example: %v", got)
	}
	if _, ok := rep.Listed["acme.io"]; ok || len(rep.Unknown) != 0 {
		t.Errorf("acme.io is clean: %+v", rep)
	}
	if rep.Domains != 3 || !slices.Equal(rep.Lists, []string{"Spamhaus DBL", "SURBL"}) {
		t.Errorf("domains %d (an IP and a duplicate aren't looked up), lists %v", rep.Domains, rep.Lists)
	}
	if row := c.rows["acme.io"]; row.Source != "dns" || len(row.Lists) != 0 || !row.CheckedAt.Equal(lookupNow) {
		t.Errorf("clean answer kept: %+v", row)
	}
	if row := c.rows["go.acme-promo.link"]; !slices.Equal(row.Lists, []string{"Spamhaus DBL", "SURBL"}) {
		t.Errorf("listing kept: %+v", row)
	}

	// Fresh answers are reused: a resolver that fails isn't asked.
	failing := dnscheck.NewZone()
	for _, n := range []string{"acme.io.dbl.spamhaus.org", "acme.io.multi.surbl.org"} {
		failing.Fail(n)
	}
	c.writes = 0
	b2 := &contentcheck.LiveBlocklist{Q: c, DNS: failing, Now: func() time.Time { return lookupNow.Add(30 * time.Minute) }}
	rep, _ = b2.Check(ctx, []string{"acme.io", "go.acme-promo.link"})
	if len(rep.Unknown) != 0 || len(rep.Listed["go.acme-promo.link"]) != 2 || c.writes != 0 {
		t.Errorf("cached answers: %+v, %d writes", rep, c.writes)
	}

	// An hour later they're looked up again; DNS fails: unknown, not clean,
	// and the old answer isn't overwritten.
	b2.Now = func() time.Time { return lookupNow.Add(2 * time.Hour) }
	rep, _ = b2.Check(ctx, []string{"acme.io"})
	if rep.Unknown["acme.io"] != contentcheck.TroubleOther || c.writes != 0 {
		t.Errorf("failed lookup: %+v, %d writes", rep, c.writes)
	}

	// A list that refuses the resolver (Spamhaus answers 127.255.255.254
	// to public resolvers) is unknown too.
	refusing := dnscheck.NewZone()
	refusing.Add("news.example.dbl.spamhaus.org", "A", "127.255.255.254")
	refusing.Add("news.example.multi.surbl.org", "A", "127.0.0.1")
	b3 := &contentcheck.LiveBlocklist{Q: &cache{rows: map[string]store.LinkDomainListing{}}, DNS: refusing, Now: func() time.Time { return lookupNow }}
	rep, _ = b3.Check(ctx, []string{"news.example"})
	if rep.Unknown["news.example"] != contentcheck.TroubleRefused {
		t.Errorf("refused: %+v", rep)
	}
}

// SR-1: the composer's checks flag a listed link domain and say which
// domains couldn't be checked.
func TestLinkChecksUseLiveLookups(t *testing.T) {
	ctx := context.Background()
	failing := listsZone()
	failing.Fail("docs.unreachable.example.dbl.spamhaus.org").Fail("docs.unreachable.example.multi.surbl.org")
	failing.Fail("unreachable.example.dbl.spamhaus.org").Fail("unreachable.example.multi.surbl.org")
	c := &contentcheck.Checker{Blocklist: &contentcheck.LiveBlocklist{Q: &cache{rows: map[string]store.LinkDomainListing{}}, DNS: failing, Now: func() time.Time { return lookupNow }}}
	r, err := c.Check(ctx, contentcheck.Input{Subject: "Flash sale", HTML: `<p>Acme Inc, 500 Market St, San Francisco, CA 94105</p>
		<a href="https://go.acme-promo.link/sale">Claim your discount</a> <a href="https://docs.unreachable.example/a">Docs</a>
		<a href="{{ unsubscribe_url }}">Unsubscribe</a>`})
	if err != nil {
		t.Fatal(err)
	}
	var listed, unknown *contentcheck.ContentCheckIssue
	for i, is := range r.Issues {
		switch is.ID {
		case "links.blocklist":
			listed = &r.Issues[i]
		case "links.blocklist_unknown":
			unknown = &r.Issues[i]
		}
	}
	if listed == nil || listed.Severity != contentcheck.Block {
		t.Fatalf("listed link domain: %+v", r.Issues)
	}
	if unknown == nil || unknown.Severity != contentcheck.Info || unknown.Title != "Couldn't check 2 link domains on Spamhaus DBL and SURBL" ||
		!strings.Contains(unknown.Detail, "isn't the same as clean") {
		t.Fatalf("unknown domains: %+v", unknown)
	}
	if r.LinkDomains != 4 || len(r.LinkDomainsUnknown) != 2 || r.LinkDomainsUnknown[0].Domain != "docs.unreachable.example" {
		t.Errorf("report: %d domains, unknown %+v", r.LinkDomains, r.LinkDomainsUnknown)
	}
}
