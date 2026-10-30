package mailbox

import (
	"context"
	"errors"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/store"
)

// Resolver is the part of net.Resolver the MX check uses.
type Resolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// MXResult is what a domain's DNS says.
type MXResult struct {
	HasMX bool
	// Hosting names the mailbox hosting behind an "other" domain
	// ("Google Workspace"), from the MX host.
	Hosting string
	// Checked is false when DNS gave no definite answer (a timeout or a
	// server failure): HasMX is then a benign "yes" and Hosting unknown.
	Checked bool
}

// MXChecker tells whether a domain accepts mail and who hosts it, caching
// answers in mx_cache for a week so imports and the classifier don't hit
// DNS again for the same domain.
type MXChecker struct {
	Q        *store.Queries
	Resolver Resolver
	TTL      time.Duration
	// Timeout bounds one lookup; Parallel bounds concurrent lookups.
	Timeout  time.Duration
	Parallel int
}

// wellKnown domains always accept mail; no lookup needed.
var wellKnown = map[string]bool{
	"gmail.com": true, "googlemail.com": true, "outlook.com": true, "hotmail.com": true, "live.com": true, "msn.com": true,
	"yahoo.com": true, "ymail.com": true, "aol.com": true, "icloud.com": true, "me.com": true, "mac.com": true,
	"protonmail.com": true, "proton.me": true, "fastmail.com": true, "gmx.com": true, "gmx.de": true, "web.de": true,
}

// Check returns, for each domain, whether it can receive mail. Temporary
// DNS failures count as "yes": an address is only rejected for a definite
// answer (NXDOMAIN, or no MX and no address records).
func (m *MXChecker) Check(ctx context.Context, domains []string) (map[string]MXResult, error) {
	out := make(map[string]MXResult, len(domains))
	var todo []string
	for _, d := range domains {
		if wellKnown[d] || emailaddr.MailboxProvider(d) != emailaddr.Other {
			out[d] = MXResult{HasMX: true, Checked: true}
			continue
		}
		todo = append(todo, d)
	}
	if len(todo) == 0 {
		return out, nil
	}
	ttl := m.ttl()
	cached, err := m.Q.GetMXCache(ctx, store.GetMXCacheParams{Domains: todo, FreshAfter: time.Now().Add(-ttl)})
	if err != nil {
		return nil, err
	}
	for _, c := range cached {
		out[c.Domain] = MXResult{HasMX: c.HasMx, Hosting: c.Provider, Checked: true}
	}
	var missing []string
	for _, d := range todo {
		if _, ok := out[d]; !ok {
			missing = append(missing, d)
		}
	}
	sort.Strings(missing)
	if len(missing) == 0 || m.Resolver == nil {
		for _, d := range missing {
			out[d] = MXResult{HasMX: true}
		}
		return out, nil
	}
	parallel := m.Parallel
	if parallel <= 0 {
		parallel = 16
	}
	timeout := m.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)
	for _, d := range missing {
		wg.Add(1)
		sem <- struct{}{}
		go func(domain string) {
			defer wg.Done()
			defer func() { <-sem }()
			res, definite := m.lookup(ctx, domain, timeout)
			res.Checked = definite
			mu.Lock()
			out[domain] = res
			mu.Unlock()
			if definite {
				_ = m.Q.PutMXCache(context.WithoutCancel(ctx), store.PutMXCacheParams{Domain: domain, HasMx: res.HasMX, Provider: res.Hosting})
			}
		}(d)
	}
	wg.Wait()
	return out, nil
}

// lookup asks DNS; definite is false for timeouts and server failures.
func (m *MXChecker) lookup(ctx context.Context, domain string, timeout time.Duration) (MXResult, bool) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	mx, err := m.Resolver.LookupMX(ctx, domain)
	if err == nil && len(mx) > 0 {
		// A null MX ("." with preference 0, RFC 7505) means no mail.
		if len(mx) == 1 && (mx[0].Host == "." || mx[0].Host == "") {
			return MXResult{HasMX: false}, true
		}
		return MXResult{HasMX: true, Hosting: emailaddr.HostingFromMX(mx[0].Host)}, true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && !dnsErr.IsNotFound {
		return MXResult{HasMX: true}, false
	}
	// No MX: the domain's A/AAAA record is the implicit mail server (RFC 5321).
	if addrs, err := m.Resolver.LookupHost(ctx, domain); err == nil && len(addrs) > 0 {
		return MXResult{HasMX: true}, true
	} else if errors.As(err, &dnsErr) && !dnsErr.IsNotFound {
		return MXResult{HasMX: true}, false
	}
	return MXResult{HasMX: false}, true
}

// ttl is how long a cached answer stays fresh.
func (m *MXChecker) ttl() time.Duration {
	if m.TTL > 0 {
		return m.TTL
	}
	return 7 * 24 * time.Hour
}
