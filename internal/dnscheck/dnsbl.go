package dnscheck

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Asset kinds checked against blocklists.
const (
	KindIP     = "ip"
	KindDomain = "domain"
)

// Blocklist is a DNS-based blocklist (DNSBL for IPs, URI/domain list for
// domains).
type Blocklist struct {
	Name string `json:"name"`
	Zone string `json:"zone"`
	// Kind is KindIP or KindDomain: what the list is queried with.
	Kind string `json:"kind"`
	// Impact on inbox placement when listed: high, medium or low.
	Impact string `json:"impact"`
	// Rank orders lists of the same impact (lower first).
	Rank int `json:"rank"`
	// DelistURL is the list's lookup and removal page.
	DelistURL string `json:"delistUrl"`
	// classify maps an answer to listed, or to an error when the list
	// refused the query (rate limits, public resolvers).
	classify func(netip.Addr) (bool, error)
}

// ErrRefused means the list answered with an error code instead of a
// listing, usually because the query came through a public resolver.
var ErrRefused = errors.New("dnscheck: blocklist refused the query")

// Builtin is the set ReputeCast checks itself when HostRepute isn't
// connected: eight major public lists, checked daily. (SORBS, still on some
// lists of "major DNSBLs", shut down in 2024 and is replaced by Mailspike.)
var Builtin = []Blocklist{
	{Name: "Spamhaus ZEN", Zone: "zen.spamhaus.org", Kind: KindIP, Impact: "high", Rank: 1, DelistURL: "https://check.spamhaus.org/", classify: spamhaus},
	{Name: "Spamhaus DBL", Zone: "dbl.spamhaus.org", Kind: KindDomain, Impact: "high", Rank: 2, DelistURL: "https://check.spamhaus.org/", classify: spamhaus},
	{Name: "SURBL", Zone: "multi.surbl.org", Kind: KindDomain, Impact: "high", Rank: 3, DelistURL: "https://surbl.org/surbl-analysis", classify: refusedOn(127, 0, 0, 1)},
	{Name: "URIBL", Zone: "multi.uribl.com", Kind: KindDomain, Impact: "high", Rank: 4, DelistURL: "https://admin.uribl.com/", classify: refusedOn(127, 0, 0, 1)},
	{Name: "Barracuda BRBL", Zone: "b.barracudacentral.org", Kind: KindIP, Impact: "high", Rank: 5, DelistURL: "https://www.barracudacentral.org/rbl/removal-request"},
	{Name: "SpamCop", Zone: "bl.spamcop.net", Kind: KindIP, Impact: "medium", Rank: 6, DelistURL: "https://www.spamcop.net/bl.shtml"},
	{Name: "Mailspike", Zone: "bl.mailspike.net", Kind: KindIP, Impact: "medium", Rank: 7, DelistURL: "https://mailspike.org/anubis/lookup.html"},
	{Name: "PSBL", Zone: "psbl.surriel.com", Kind: KindIP, Impact: "low", Rank: 8, DelistURL: "https://psbl.org/"},
}

// spamhaus treats 127.255.255.x as an error (public resolver, rate limit,
// typo in the query) and 127.0.1.255 on the DBL as "you sent an IP".
func spamhaus(a netip.Addr) (bool, error) {
	b := a.As4()
	if b[0] != 127 {
		return false, fmt.Errorf("%w: unexpected answer %s", ErrRefused, a)
	}
	if b[1] == 255 && b[2] == 255 {
		return false, fmt.Errorf("%w: Spamhaus answered %s (use your own resolver, not a public one)", ErrRefused, a)
	}
	if b[1] == 0 && b[2] == 1 && b[3] == 255 {
		return false, fmt.Errorf("%w: invalid query", ErrRefused)
	}
	return true, nil
}

// refusedOn returns a classifier for lists that answer one fixed address when
// they refuse a query (SURBL and URIBL answer 127.0.0.1).
func refusedOn(a, b, c, d byte) func(netip.Addr) (bool, error) {
	refused := netip.AddrFrom4([4]byte{a, b, c, d})
	return func(ip netip.Addr) (bool, error) {
		if ip == refused {
			return false, fmt.Errorf("%w: the list refused queries from this resolver", ErrRefused)
		}
		return ip.As4()[0] == 127, nil
	}
}

func defaultClassify(a netip.Addr) (bool, error) {
	if !a.Is4() || a.As4()[0] != 127 {
		return false, fmt.Errorf("%w: unexpected answer %s", ErrRefused, a)
	}
	return true, nil
}

// QueryName returns the DNS name to look up asset on the list: reversed
// octets (or nibbles for IPv6) for IPs, the domain itself for domains.
func (b Blocklist) QueryName(asset string) (string, error) {
	if b.Kind == KindIP {
		ip, err := netip.ParseAddr(asset)
		if err != nil {
			return "", fmt.Errorf("%s is not an IP address", asset)
		}
		rev, err := dns.ReverseAddr(ip.Unmap().String())
		if err != nil {
			return "", err
		}
		rev = strings.TrimSuffix(rev, ".")
		rev = strings.TrimSuffix(strings.TrimSuffix(rev, ".in-addr.arpa"), ".ip6.arpa")
		return rev + "." + b.Zone, nil
	}
	d := strings.ToLower(strings.TrimSuffix(asset, "."))
	if _, err := netip.ParseAddr(d); err == nil || d == "" {
		return "", fmt.Errorf("%s is not a domain", asset)
	}
	return d + "." + b.Zone, nil
}

// Listing is the result of one asset on one list.
type Listing struct {
	List   Blocklist `json:"list"`
	Asset  string    `json:"asset"`
	Listed bool      `json:"listed"`
	// Codes are the list's answers ("127.0.0.2"); the code says why.
	Codes []string `json:"codes,omitempty"`
	// Reason is the list's TXT explanation, when it publishes one.
	Reason string `json:"reason,omitempty"`
	Err    error  `json:"-"`
}

// Lookup checks one asset on one list. Not listed is a NXDOMAIN/NODATA
// answer; errors are returned in Listing.Err and never count as listed.
func Lookup(ctx context.Context, r Resolver, list Blocklist, asset string) Listing {
	out := Listing{List: list, Asset: asset}
	name, err := list.QueryName(asset)
	if err != nil {
		out.Err = err
		return out
	}
	addrs, err := r.A(ctx, name)
	if err != nil {
		if !IsNotFound(err) {
			out.Err = err
		}
		return out
	}
	classify := list.classify
	if classify == nil {
		classify = defaultClassify
	}
	for _, a := range addrs {
		listed, cerr := classify(a)
		if cerr != nil {
			out.Err = cerr
			continue
		}
		if listed {
			out.Listed = true
			out.Codes = append(out.Codes, a.String())
		}
	}
	if out.Listed {
		out.Err = nil
		if txts, err := r.TXT(ctx, name); err == nil && len(txts) > 0 {
			out.Reason = txts[0]
		}
	}
	return out
}

// Asset is something checked against blocklists.
type Asset struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // KindIP or KindDomain
}

// Applies reports whether the list checks this kind of asset.
func (b Blocklist) Applies(a Asset) bool { return b.Kind == a.Kind }

// Pairs returns the (list, asset) lookups for assets on lists.
func Pairs(lists []Blocklist, assets []Asset) [][2]int {
	var out [][2]int
	for li, l := range lists {
		for ai, a := range assets {
			if l.Applies(a) {
				out = append(out, [2]int{li, ai})
			}
		}
	}
	return out
}

// CheckAll looks up every asset on every list that applies, at most
// concurrency at a time, each with its own timeout. progress (optional) is
// called after each answer with the number done and the total.
func CheckAll(ctx context.Context, r Resolver, lists []Blocklist, assets []Asset, concurrency int, timeout time.Duration, progress func(done, total int)) []Listing {
	pairs := Pairs(lists, assets)
	out := make([]Listing, len(pairs))
	if concurrency <= 0 {
		concurrency = 8
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	for i, p := range pairs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, list Blocklist, asset string) {
			defer wg.Done()
			defer func() { <-sem }()
			lctx, cancel := context.WithTimeout(ctx, timeout)
			out[i] = Lookup(lctx, r, list, asset)
			cancel()
			if progress != nil {
				mu.Lock()
				done++
				progress(done, len(pairs))
				mu.Unlock()
			}
		}(i, lists[p[0]], assets[p[1]].Name)
	}
	wg.Wait()
	return out
}
