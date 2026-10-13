package dnscheck

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// Zone is an in-memory Resolver: tests describe the DNS they need, and the
// demo answers for its sample domains without touching the network. Names
// are case-insensitive. A name with no records of any type is NXDOMAIN; a
// name with other records is NODATA. CNAMEs are followed for other types.
type Zone struct {
	mu      sync.RWMutex
	records map[string]map[string][]string // name → type → values
	// Fail makes every lookup of a name fail temporarily (SERVFAIL).
	fail map[string]bool
}

// NewZone returns an empty zone.
func NewZone() *Zone {
	return &Zone{records: map[string]map[string][]string{}, fail: map[string]bool{}}
}

func zkey(name string) string { return strings.ToLower(strings.TrimSuffix(name, ".")) }

// Add adds a record: Add("news.acme.io", "TXT", "v=spf1 -all"). For MX the
// value is "10 host"; for PTR the name is the IP address ("203.0.113.25").
func (z *Zone) Add(name, typ, value string) *Zone {
	z.mu.Lock()
	defer z.mu.Unlock()
	k := zkey(name)
	if z.records[k] == nil {
		z.records[k] = map[string][]string{}
	}
	t := strings.ToUpper(typ)
	z.records[k][t] = append(z.records[k][t], value)
	return z
}

// Fail makes lookups of name return ErrTemporary.
func (z *Zone) Fail(name string) *Zone {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.fail[zkey(name)] = true
	return z
}

// Has reports whether the zone holds any record for name.
func (z *Zone) Has(name string) bool {
	z.mu.RLock()
	defer z.mu.RUnlock()
	_, ok := z.records[zkey(name)]
	return ok
}

func (z *Zone) lookup(name, typ string) ([]string, error) {
	z.mu.RLock()
	defer z.mu.RUnlock()
	k := zkey(name)
	for range 8 { // follow at most 8 CNAMEs
		if z.fail[k] {
			return nil, fmt.Errorf("%s %s: %w: SERVFAIL", name, typ, ErrTemporary)
		}
		recs, ok := z.records[k]
		if !ok {
			return nil, fmt.Errorf("%s %s: %w", name, typ, ErrNXDomain)
		}
		if v := recs[typ]; len(v) > 0 {
			return append([]string(nil), v...), nil
		}
		if typ != "CNAME" && len(recs["CNAME"]) > 0 {
			k = zkey(recs["CNAME"][0])
			continue
		}
		return nil, fmt.Errorf("%s %s: %w", name, typ, ErrNoData)
	}
	return nil, fmt.Errorf("%s %s: %w: CNAME loop", name, typ, ErrTemporary)
}

// TXT implements Resolver.
func (z *Zone) TXT(_ context.Context, name string) ([]string, error) { return z.lookup(name, "TXT") }

// CNAME implements Resolver.
func (z *Zone) CNAME(_ context.Context, name string) (string, error) {
	v, err := z.lookup(name, "CNAME")
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(v[0], "."), nil
}

// MX implements Resolver.
func (z *Zone) MX(_ context.Context, name string) ([]MX, error) {
	v, err := z.lookup(name, "MX")
	if err != nil {
		return nil, err
	}
	out := make([]MX, 0, len(v))
	for _, s := range v {
		var mx MX
		var host string
		if _, err := fmt.Sscanf(s, "%d %s", &mx.Pref, &host); err != nil {
			return nil, fmt.Errorf("zone: bad MX %q", s)
		}
		mx.Host = strings.TrimSuffix(host, ".")
		out = append(out, mx)
	}
	return out, nil
}

func (z *Zone) addrs(name, typ string) ([]netip.Addr, error) {
	v, err := z.lookup(name, typ)
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(v))
	for _, s := range v {
		a, err := netip.ParseAddr(s)
		if err != nil {
			return nil, fmt.Errorf("zone: bad %s %q", typ, s)
		}
		out = append(out, a)
	}
	return out, nil
}

// A implements Resolver.
func (z *Zone) A(_ context.Context, name string) ([]netip.Addr, error) { return z.addrs(name, "A") }

// AAAA implements Resolver.
func (z *Zone) AAAA(_ context.Context, name string) ([]netip.Addr, error) {
	return z.addrs(name, "AAAA")
}

// PTR implements Resolver. PTR records are added under the IP address or its
// in-addr.arpa / ip6.arpa name.
func (z *Zone) PTR(_ context.Context, ip netip.Addr) ([]string, error) {
	v, err := z.lookup(ip.String(), "PTR")
	if IsNotFound(err) {
		if rev, rerr := dns.ReverseAddr(ip.String()); rerr == nil {
			v, err = z.lookup(rev, "PTR")
		}
	}
	if err != nil {
		return nil, err
	}
	for i := range v {
		v[i] = strings.TrimSuffix(v[i], ".")
	}
	return v, nil
}

// Overlay answers names the zone knows (and names under its suffixes) from
// the zone and everything else from fallback. The demo uses it so the Acme
// sample domains check clean without real DNS.
func Overlay(z *Zone, suffixes []string, fallback Resolver) Resolver {
	return overlay{z: z, suffixes: suffixes, fallback: fallback}
}

type overlay struct {
	z        *Zone
	suffixes []string
	fallback Resolver
}

func (o overlay) owns(name string) bool {
	k := zkey(name)
	if o.z.Has(k) {
		return true
	}
	for _, s := range o.suffixes {
		s = zkey(s)
		if k == s || strings.HasSuffix(k, "."+s) {
			return true
		}
	}
	return false
}

func (o overlay) TXT(ctx context.Context, name string) ([]string, error) {
	if o.owns(name) {
		return o.z.TXT(ctx, name)
	}
	return o.fallback.TXT(ctx, name)
}

func (o overlay) CNAME(ctx context.Context, name string) (string, error) {
	if o.owns(name) {
		return o.z.CNAME(ctx, name)
	}
	return o.fallback.CNAME(ctx, name)
}

func (o overlay) MX(ctx context.Context, name string) ([]MX, error) {
	if o.owns(name) {
		return o.z.MX(ctx, name)
	}
	return o.fallback.MX(ctx, name)
}

func (o overlay) A(ctx context.Context, name string) ([]netip.Addr, error) {
	if o.owns(name) {
		return o.z.A(ctx, name)
	}
	return o.fallback.A(ctx, name)
}

func (o overlay) AAAA(ctx context.Context, name string) ([]netip.Addr, error) {
	if o.owns(name) {
		return o.z.AAAA(ctx, name)
	}
	return o.fallback.AAAA(ctx, name)
}

func (o overlay) PTR(ctx context.Context, ip netip.Addr) ([]string, error) {
	if o.owns(ip.String()) {
		return o.z.PTR(ctx, ip)
	}
	return o.fallback.PTR(ctx, ip)
}
