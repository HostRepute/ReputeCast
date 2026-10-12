// Package dnscheck answers the DNS questions deliverability depends on:
// SPF, DKIM, DMARC (with policy discovery), BIMI and MAIL FROM alignment for
// sending domains, forward-confirmed reverse DNS for IPs, and DNSBL listings
// for both.
//
// Everything goes through the small Resolver interface. Client implements it
// with miekg/dns v1 (see docs/TECH_STACK.md: switch to codeberg.org/miekg/dns
// v2 here once it reaches 1.0); Zone is an in-memory resolver for tests and
// the demo. Checks are plain functions over a Resolver, so the deliverability
// checks and the sending-domain verification share one implementation.
package dnscheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Resolver looks up DNS records. Names may omit the trailing dot.
// Implementations return an error wrapping ErrNotFound when the name does not
// exist (NXDOMAIN) or has no record of the asked type (NODATA).
type Resolver interface {
	// TXT returns one string per TXT record (multi-string records joined).
	TXT(ctx context.Context, name string) ([]string, error)
	// CNAME returns the target of the CNAME at name, without the trailing dot.
	CNAME(ctx context.Context, name string) (string, error)
	MX(ctx context.Context, name string) ([]MX, error)
	A(ctx context.Context, name string) ([]netip.Addr, error)
	AAAA(ctx context.Context, name string) ([]netip.Addr, error)
	// PTR returns the reverse DNS names of ip, without trailing dots.
	PTR(ctx context.Context, ip netip.Addr) ([]string, error)
}

// MX is one mail exchanger.
type MX struct {
	Host string `json:"host"`
	Pref uint16 `json:"pref"`
}

// Errors returned by resolvers.
var (
	// ErrNotFound is wrapped by ErrNXDomain and ErrNoData.
	ErrNotFound = errors.New("dnscheck: not found")
	// ErrNXDomain means the name does not exist.
	ErrNXDomain = fmt.Errorf("%w: no such domain", ErrNotFound)
	// ErrNoData means the name exists but has no record of the asked type.
	ErrNoData = fmt.Errorf("%w: no record of that type", ErrNotFound)
	// ErrTemporary means no server gave a usable answer (timeout, SERVFAIL).
	ErrTemporary = errors.New("dnscheck: temporary failure")
)

// IsNotFound reports whether err means "no such record".
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// Options configures Client.
type Options struct {
	// Servers are resolvers as host or host:port. Empty means the system's
	// resolvers from /etc/resolv.conf. Public resolvers (1.1.1.1, 8.8.8.8)
	// work for authentication checks but are refused by Spamhaus and URIBL,
	// so DNSBL lookups need your own recursive resolver.
	Servers []string
	// Timeout per query and server (default 4 s).
	Timeout time.Duration
}

// Client is the miekg/dns implementation of Resolver: UDP with EDNS0, TCP
// when an answer is truncated, and the next server on errors.
type Client struct {
	servers []string
	udp     *dns.Client
	tcp     *dns.Client
}

// NewResolver returns a Client for the configured servers, or the system
// resolver of the Go standard library when no server is configured and
// /etc/resolv.conf is unavailable (Windows).
func NewResolver(o Options) Resolver {
	c, err := NewClient(o)
	if err != nil {
		return stdResolver{r: net.DefaultResolver}
	}
	return c
}

// NewClient returns a miekg/dns client for the given servers.
func NewClient(o Options) (*Client, error) {
	servers := make([]string, 0, len(o.Servers))
	for _, s := range o.Servers {
		if s = strings.TrimSpace(s); s != "" {
			servers = append(servers, withPort(s))
		}
	}
	if len(servers) == 0 {
		cfg, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			return nil, fmt.Errorf("dnscheck: no resolvers configured: %w", err)
		}
		for _, s := range cfg.Servers {
			servers = append(servers, net.JoinHostPort(s, cfg.Port))
		}
	}
	if len(servers) == 0 {
		return nil, errors.New("dnscheck: no resolvers configured")
	}
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	return &Client{
		servers: servers,
		udp:     &dns.Client{Net: "udp", Timeout: timeout, UDPSize: 1232},
		tcp:     &dns.Client{Net: "tcp", Timeout: timeout},
	}, nil
}

func withPort(s string) string {
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	return net.JoinHostPort(strings.Trim(s, "[]"), "53")
}

// query asks each server in turn until one answers authoritatively for the
// question (NOERROR or NXDOMAIN).
func (c *Client) query(ctx context.Context, name string, qtype uint16) ([]dns.RR, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), qtype)
	m.RecursionDesired = true
	m.SetEdns0(1232, false)
	var lastErr error
	for _, server := range c.servers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		in, _, err := c.udp.ExchangeContext(ctx, m, server)
		if err == nil && in != nil && in.Truncated {
			in, _, err = c.tcp.ExchangeContext(ctx, m, server)
		}
		if err != nil {
			lastErr = err
			continue
		}
		switch in.Rcode {
		case dns.RcodeSuccess:
			var out []dns.RR
			for _, rr := range in.Answer {
				if rr.Header().Rrtype == qtype {
					out = append(out, rr)
				}
			}
			if len(out) == 0 {
				return nil, fmt.Errorf("%s %s: %w", name, dns.TypeToString[qtype], ErrNoData)
			}
			return out, nil
		case dns.RcodeNameError:
			return nil, fmt.Errorf("%s %s: %w", name, dns.TypeToString[qtype], ErrNXDomain)
		default:
			lastErr = fmt.Errorf("%s answered %s", server, dns.RcodeToString[in.Rcode])
		}
	}
	return nil, fmt.Errorf("%s %s: %w: %v", name, dns.TypeToString[qtype], ErrTemporary, lastErr)
}

// TXT implements Resolver.
func (c *Client) TXT(ctx context.Context, name string) ([]string, error) {
	rrs, err := c.query(ctx, name, dns.TypeTXT)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rrs))
	for _, rr := range rrs {
		out = append(out, strings.Join(rr.(*dns.TXT).Txt, ""))
	}
	return out, nil
}

// CNAME implements Resolver.
func (c *Client) CNAME(ctx context.Context, name string) (string, error) {
	rrs, err := c.query(ctx, name, dns.TypeCNAME)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(rrs[0].(*dns.CNAME).Target, "."), nil
}

// MX implements Resolver.
func (c *Client) MX(ctx context.Context, name string) ([]MX, error) {
	rrs, err := c.query(ctx, name, dns.TypeMX)
	if err != nil {
		return nil, err
	}
	out := make([]MX, 0, len(rrs))
	for _, rr := range rrs {
		mx := rr.(*dns.MX)
		out = append(out, MX{Host: strings.TrimSuffix(mx.Mx, "."), Pref: mx.Preference})
	}
	return out, nil
}

// A implements Resolver.
func (c *Client) A(ctx context.Context, name string) ([]netip.Addr, error) {
	rrs, err := c.query(ctx, name, dns.TypeA)
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(rrs))
	for _, rr := range rrs {
		if a, ok := netip.AddrFromSlice(rr.(*dns.A).A.To4()); ok {
			out = append(out, a)
		}
	}
	return out, nil
}

// AAAA implements Resolver.
func (c *Client) AAAA(ctx context.Context, name string) ([]netip.Addr, error) {
	rrs, err := c.query(ctx, name, dns.TypeAAAA)
	if err != nil {
		return nil, err
	}
	out := make([]netip.Addr, 0, len(rrs))
	for _, rr := range rrs {
		if a, ok := netip.AddrFromSlice(rr.(*dns.AAAA).AAAA); ok {
			out = append(out, a)
		}
	}
	return out, nil
}

// PTR implements Resolver.
func (c *Client) PTR(ctx context.Context, ip netip.Addr) ([]string, error) {
	rev, err := dns.ReverseAddr(ip.String())
	if err != nil {
		return nil, err
	}
	rrs, err := c.query(ctx, rev, dns.TypePTR)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rrs))
	for _, rr := range rrs {
		out = append(out, strings.TrimSuffix(rr.(*dns.PTR).Ptr, "."))
	}
	return out, nil
}

// stdResolver adapts net.Resolver for platforms without /etc/resolv.conf.
type stdResolver struct{ r *net.Resolver }

func mapStdErr(err error) error {
	var de *net.DNSError
	if errors.As(err, &de) {
		if de.IsNotFound {
			return fmt.Errorf("%s: %w", de.Name, ErrNotFound)
		}
		return fmt.Errorf("%s: %w: %v", de.Name, ErrTemporary, de.Err)
	}
	return err
}

func (s stdResolver) TXT(ctx context.Context, name string) ([]string, error) {
	out, err := s.r.LookupTXT(ctx, name)
	return out, mapStdErr(err)
}

func (s stdResolver) CNAME(ctx context.Context, name string) (string, error) {
	out, err := s.r.LookupCNAME(ctx, name)
	if err != nil {
		return "", mapStdErr(err)
	}
	out = strings.TrimSuffix(out, ".")
	if strings.EqualFold(out, strings.TrimSuffix(name, ".")) {
		return "", fmt.Errorf("%s CNAME: %w", name, ErrNoData)
	}
	return out, nil
}

func (s stdResolver) MX(ctx context.Context, name string) ([]MX, error) {
	mxs, err := s.r.LookupMX(ctx, name)
	if err != nil {
		return nil, mapStdErr(err)
	}
	out := make([]MX, 0, len(mxs))
	for _, m := range mxs {
		out = append(out, MX{Host: strings.TrimSuffix(m.Host, "."), Pref: m.Pref})
	}
	return out, nil
}

func (s stdResolver) addrs(ctx context.Context, network, name string) ([]netip.Addr, error) {
	ips, err := s.r.LookupNetIP(ctx, network, name)
	if err != nil {
		return nil, mapStdErr(err)
	}
	return ips, nil
}

func (s stdResolver) A(ctx context.Context, name string) ([]netip.Addr, error) {
	return s.addrs(ctx, "ip4", name)
}

func (s stdResolver) AAAA(ctx context.Context, name string) ([]netip.Addr, error) {
	return s.addrs(ctx, "ip6", name)
}

func (s stdResolver) PTR(ctx context.Context, ip netip.Addr) ([]string, error) {
	names, err := s.r.LookupAddr(ctx, ip.String())
	if err != nil {
		return nil, mapStdErr(err)
	}
	for i := range names {
		names[i] = strings.TrimSuffix(names[i], ".")
	}
	return names, nil
}
