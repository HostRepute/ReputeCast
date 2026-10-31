package contentcheck

import (
	"context"
	"net/netip"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/store"
)

// URIListNames are the URI blocklists link domains are looked up on.
var URIListNames = []string{"Spamhaus DBL", "SURBL"}

// URILists are the URI blocklists link domains are looked up on, from the
// built-in lists: Spamhaus DBL and SURBL.
func URILists() []dnscheck.Blocklist {
	out := []dnscheck.Blocklist{}
	for _, l := range dnscheck.Builtin {
		if slices.Contains(URIListNames, l.Name) {
			out = append(out, l)
		}
	}
	return out
}

// BlocklistReport is what a live lookup of link domains found.
type BlocklistReport struct {
	// Listed maps a listed domain to the lists it is on.
	Listed map[string][]string
	// Unknown maps a domain that couldn't be checked to why (a Trouble*
	// kind): DNS didn't answer, or the list refused the resolver.
	Unknown map[string]string
	// Lists names the lists the domains were looked up on.
	Lists []string
	// Domains is how many domains were checked (cached or live).
	Domains int
}

// BlocklistChecker is a Blocklist that also says which domains it couldn't
// check: a failed lookup is unknown, never clean.
type BlocklistChecker interface {
	Blocklist
	Check(ctx context.Context, domains []string) (BlocklistReport, error)
}

// Kinds of lookup trouble (BlocklistReport.Unknown).
const (
	TroubleTimeout = "timeout" // no answer in time
	TroubleRefused = "refused" // the list refused the query (public resolver, rate limit)
	TroubleOther   = "other"   // no usable answer
)

// LiveBlocklist looks link domains up on Spamhaus DBL and SURBL over DNS and
// keeps the answers in link_domain_listings, shared by every workspace: an
// answer younger than MaxAge is reused, an older one is looked up again. A
// lookup that fails is unknown and isn't kept. The composer's inline checks
// and Send Readiness both use it.
type LiveBlocklist struct {
	Q   store.Querier
	DNS dnscheck.Resolver
	// Lists default to URILists.
	Lists []dnscheck.Blocklist
	// MaxAge is how long an answer is reused (default an hour).
	MaxAge time.Duration
	// Budget bounds the live lookups of one check (default 30 s): the
	// composer, which checks as people type, uses a short one.
	Budget time.Duration
	Now    func() time.Time
}

func (b *LiveBlocklist) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

// Listings implements Blocklist.
func (b *LiveBlocklist) Listings(ctx context.Context, domains []string) (map[string][]string, error) {
	r, err := b.Check(ctx, domains)
	return r.Listed, err
}

// Check implements BlocklistChecker.
func (b *LiveBlocklist) Check(ctx context.Context, domains []string) (BlocklistReport, error) {
	lists := b.Lists
	if len(lists) == 0 {
		lists = URILists()
	}
	out := BlocklistReport{Listed: map[string][]string{}, Unknown: map[string]string{}, Lists: []string{}}
	for _, l := range lists {
		out.Lists = append(out.Lists, l.Name)
	}
	todo := []string{}
	seen := map[string]bool{}
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(d), "."))
		if d == "" || seen[d] || !strings.Contains(d, ".") {
			continue
		}
		if _, err := netip.ParseAddr(d); err == nil {
			continue // an IP address isn't on a domain list
		}
		seen[d] = true
		todo = append(todo, d)
	}
	sort.Strings(todo)
	out.Domains = len(todo)
	if len(todo) == 0 {
		return out, nil
	}
	maxAge := b.MaxAge
	if maxAge <= 0 {
		maxAge = time.Hour
	}
	now := b.now()
	cached, err := b.Q.ContentCheckLinkListings(ctx, todo)
	if err != nil {
		return out, err
	}
	fresh := map[string]bool{}
	for _, row := range cached {
		if now.Sub(row.CheckedAt) > maxAge {
			continue
		}
		fresh[row.Domain] = true
		if len(row.Lists) > 0 {
			out.Listed[row.Domain] = row.Lists
		}
	}
	live := []string{}
	for _, d := range todo {
		if !fresh[d] {
			live = append(live, d)
		}
	}
	if len(live) == 0 {
		return out, nil
	}
	if b.DNS == nil {
		for _, d := range live {
			out.Unknown[d] = TroubleOther
		}
		return out, nil
	}
	budget := b.Budget
	if budget <= 0 {
		budget = 30 * time.Second
	}
	lctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	assets := make([]dnscheck.Asset, 0, len(live))
	for _, d := range live {
		assets = append(assets, dnscheck.Asset{Name: d, Kind: dnscheck.KindDomain})
	}
	report := ProgressFrom(ctx)
	results := dnscheck.CheckAll(lctx, b.DNS, lists, assets, 8, 5*time.Second, func(done, total int) {
		report(StageLinkDomains, done/len(lists), len(live))
	})
	type answer struct {
		lists   []string
		trouble string
	}
	byDomain := map[string]*answer{}
	for _, d := range live {
		byDomain[d] = &answer{}
	}
	for _, r := range results {
		a := byDomain[r.Asset]
		switch {
		case r.Listed:
			a.lists = append(a.lists, r.List.Name)
		case r.Err != nil && a.trouble == "":
			a.trouble = lookupTrouble(lctx, r.Err)
		}
	}
	for _, d := range live {
		a := byDomain[d]
		// A domain on one list is listed even if another list didn't answer.
		if len(a.lists) == 0 && a.trouble != "" {
			out.Unknown[d] = a.trouble
			continue
		}
		if len(a.lists) > 0 {
			out.Listed[d] = a.lists
		}
		if err := b.Q.ContentCheckUpsertLinkListing(ctx, store.ContentCheckUpsertLinkListingParams{Domain: d, Lists: orEmpty(a.lists), Source: "dns", CheckedAt: now}); err != nil {
			return out, err
		}
	}
	return out, nil
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// lookupTrouble names why a list lookup failed.
func lookupTrouble(ctx context.Context, err error) string {
	e := strings.ToLower(err.Error())
	switch {
	case strings.Contains(e, "refused"):
		return TroubleRefused
	case ctx.Err() != nil || strings.Contains(e, "timeout") || strings.Contains(e, "deadline") || strings.Contains(e, "timed out"):
		return TroubleTimeout
	}
	return TroubleOther
}

// Stages a check reports progress on (WithProgress).
const (
	StageLinks       = "links"        // requesting each link
	StageLinkDomains = "link_domains" // looking link domains up on URI blocklists
)

type progressKey struct{}

// Progress is told how far a stage of a check got: done of total.
type Progress func(stage string, done, total int)

// WithProgress has the checks run with ctx report their progress (Send
// Readiness shows "5 of 8 domains").
func WithProgress(ctx context.Context, p Progress) context.Context {
	return context.WithValue(ctx, progressKey{}, p)
}

// ProgressFrom is the progress observer of ctx, or one that ignores it.
func ProgressFrom(ctx context.Context) Progress {
	if p, ok := ctx.Value(progressKey{}).(Progress); ok && p != nil {
		return p
	}
	return func(string, int, int) {}
}
