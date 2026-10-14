package contentcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"syscall"
	"time"

	"github.com/hostrepute/reputecast/internal/store"
)

// StoreBlocklist reads URI blocklist results cached in link_domain_listings.
// Deliverability lookups (DNS, HostRepute) write that table; content checks
// only read it, so they stay fast.
type StoreBlocklist struct {
	Q store.Querier
}

// Listings implements Blocklist.
func (b StoreBlocklist) Listings(ctx context.Context, domains []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(domains) == 0 {
		return out, nil
	}
	rows, err := b.Q.ContentCheckLinkListings(ctx, domains)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if len(r.Lists) > 0 {
			out[r.Domain] = r.Lists
		}
	}
	return out, nil
}

// StaticBlocklist is a fixed map, for tests and installs without lookups.
type StaticBlocklist map[string][]string

// Listings implements Blocklist.
func (b StaticBlocklist) Listings(_ context.Context, domains []string) (map[string][]string, error) {
	out := map[string][]string{}
	for _, d := range domains {
		if l := b[d]; len(l) > 0 {
			out[d] = l
		}
	}
	return out, nil
}

// HTTPProber requests links with HEAD (GET when HEAD isn't allowed), with
// short timeouts and an SSRF guard: it never connects to loopback, private,
// link-local or otherwise internal addresses, also after redirects.
type HTTPProber struct {
	Client *http.Client
	// MaxLinks caps how many links one run requests.
	MaxLinks int
	// Concurrency is how many requests run at once.
	Concurrency int

	mu    sync.Mutex
	cache map[string]cachedProbe
}

type cachedProbe struct {
	res ProbeResult
	at  time.Time
}

// ErrBlockedAddress is returned when a link resolves to an internal address.
var ErrBlockedAddress = errors.New("the link points at an internal address")

// PublicOnly is a net.Dialer Control function that refuses non-public IPs.
func PublicOnly(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() ||
		cgnat.Contains(ip) || ip.Is4() && ip.As4()[0] == 0 {
		return ErrBlockedAddress
	}
	return nil
}

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// NewHTTPProber returns a prober with the SSRF guard on its dialer.
func NewHTTPProber() *HTTPProber {
	dialer := &net.Dialer{Timeout: 4 * time.Second, Control: PublicOnly}
	tr := &http.Transport{DialContext: dialer.DialContext, TLSHandshakeTimeout: 4 * time.Second, ResponseHeaderTimeout: 5 * time.Second,
		MaxIdleConns: 20, IdleConnTimeout: 30 * time.Second, Proxy: nil}
	return &HTTPProber{
		Client: &http.Client{Transport: tr, Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		}},
		MaxLinks: 40, Concurrency: 4,
	}
}

// Probe implements Prober. Results are cached for ten minutes.
func (p *HTTPProber) Probe(ctx context.Context, urls []string) map[string]ProbeResult {
	out := map[string]ProbeResult{}
	if p.MaxLinks > 0 && len(urls) > p.MaxLinks {
		urls = urls[:p.MaxLinks]
	}
	conc := p.Concurrency
	if conc <= 0 {
		conc = 4
	}
	var mu sync.Mutex
	sem := make(chan struct{}, conc)
	var wg sync.WaitGroup
	for _, u := range urls {
		if r, ok := p.cached(u); ok {
			out[u] = r
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(u string) {
			defer wg.Done()
			defer func() { <-sem }()
			r := p.one(ctx, u)
			p.store(u, r)
			mu.Lock()
			out[u] = r
			mu.Unlock()
		}(u)
	}
	wg.Wait()
	return out
}

func (p *HTTPProber) one(ctx context.Context, u string) ProbeResult {
	do := func(method string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "ReputeCast link check (+https://reputecast.com)")
		if method == http.MethodGet {
			req.Header.Set("Range", "bytes=0-0")
		}
		return p.Client.Do(req)
	}
	start := time.Now()
	resp, err := do(http.MethodHead)
	if err == nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented || resp.StatusCode == http.StatusForbidden) {
		_ = resp.Body.Close()
		start = time.Now()
		resp, err = do(http.MethodGet)
	}
	millis := int(time.Since(start) / time.Millisecond)
	if err != nil {
		if errors.Is(err, ErrBlockedAddress) {
			return ProbeResult{Broken: true, Error: ErrBlockedAddress.Error()}
		}
		return ProbeResult{Broken: true, Error: fmt.Sprintf("no response: %v", err), Millis: millis}
	}
	_ = resp.Body.Close()
	broken := resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone || resp.StatusCode >= 500
	out := ProbeResult{Status: resp.StatusCode, Broken: broken, Millis: millis}
	if resp.Request != nil && resp.Request.URL != nil && resp.Request.URL.String() != u {
		out.Final = resp.Request.URL.String()
	}
	return out
}

func (p *HTTPProber) cached(u string) (ProbeResult, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := p.cache[u]
	if !ok || time.Since(c.at) > 10*time.Minute {
		return ProbeResult{}, false
	}
	return c.res, true
}

func (p *HTTPProber) store(u string, r ProbeResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cache == nil {
		p.cache = map[string]cachedProbe{}
	}
	if len(p.cache) > 5000 {
		p.cache = map[string]cachedProbe{}
	}
	p.cache[u] = cachedProbe{res: r, at: time.Now()}
}
