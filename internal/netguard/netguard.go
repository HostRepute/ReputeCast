// Package netguard makes outbound HTTP requests to user-supplied URLs
// (outgoing webhooks, the audit stream) safe from server-side request
// forgery: the address is checked after DNS resolution, at connect time, so
// a hostname can't point at loopback, private, link-local or metadata
// addresses, and redirects are never followed. Private networks can be
// allowed explicitly for self-hosted setups (webhooks.allow_private_networks).
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// ErrBlocked is returned when a URL resolves to a forbidden address.
var ErrBlocked = errors.New("address not allowed")

// blocked ranges that are never reachable, even when private networks are allowed.
var alwaysBlocked = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), // link-local, cloud metadata (169.254.169.254)
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("224.0.0.0/4"), // multicast
	netip.MustParsePrefix("ff00::/8"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("255.255.255.255/32"),
	netip.MustParsePrefix("100.100.100.200/32"), // Alibaba metadata
	netip.MustParsePrefix("fd00:ec2::254/128"),  // AWS IMDS over IPv6
}

// private ranges, reachable only when explicitly allowed.
var privateRanges = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("100.64.0.0/10"), // carrier-grade NAT
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("::/128"),
}

var nat64 = netip.MustParsePrefix("64:ff9b::/96")

// Policy decides which addresses may be dialed.
type Policy struct {
	// AllowPrivate allows loopback and RFC 1918 addresses (self-hosted receivers).
	AllowPrivate bool
}

// Check reports whether ip may be dialed.
func (p Policy) Check(ip netip.Addr) error {
	ip = ip.Unmap()
	if nat64.Contains(ip) {
		// NAT64 addresses carry an IPv4 address in their last 32 bits.
		b := ip.As16()
		ip = netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	if !ip.IsValid() || ip.IsUnspecified() {
		return fmt.Errorf("%w: %s", ErrBlocked, ip)
	}
	for _, r := range alwaysBlocked {
		if r.Contains(ip) {
			return fmt.Errorf("%w: %s is a link-local, metadata or multicast address", ErrBlocked, ip)
		}
	}
	if !p.AllowPrivate {
		for _, r := range privateRanges {
			if r.Contains(ip) {
				return fmt.Errorf("%w: %s is a private or loopback address", ErrBlocked, ip)
			}
		}
	}
	return nil
}

// ValidateURL checks the scheme and host of a URL before it is stored. The
// address itself is checked again at every connection.
func (p Policy) ValidateURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return nil, errors.New("use a full URL, like https://api.example.com/hooks/reputecast")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, errors.New("use an https:// URL")
	}
	if u.User != nil {
		return nil, errors.New("put credentials in the signing secret, not in the URL")
	}
	host := u.Hostname()
	if ip, err := netip.ParseAddr(host); err == nil {
		if err := p.Check(ip); err != nil {
			return nil, err
		}
	} else if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") || strings.HasSuffix(strings.ToLower(host), ".internal") {
		if !p.AllowPrivate {
			return nil, fmt.Errorf("%w: %s is a local hostname", ErrBlocked, host)
		}
	}
	return u, nil
}

// Dialer returns a dialer that refuses forbidden addresses at connect time,
// for raw connections (SMTP, IMAP, POP3) and for Client.
func (p Policy) Dialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip, err := netip.ParseAddr(host)
			if err != nil {
				return fmt.Errorf("%w: %s", ErrBlocked, host)
			}
			return p.Check(ip)
		},
	}
}

// Client returns an HTTP client that refuses forbidden addresses at connect
// time and never follows redirects.
func (p Policy) Client(timeout time.Duration) *http.Client {
	dialer := p.Dialer(5 * time.Second)
	tr := &http.Transport{
		Proxy:                 nil, // a proxy would bypass the address check
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// IsBlocked reports whether err comes from the address policy.
func IsBlocked(err error) bool { return errors.Is(err, ErrBlocked) }

// Describe turns a transport error into a short sentence for delivery logs.
func Describe(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	var dnsErr *net.DNSError
	switch {
	case IsBlocked(err):
		msg := err.Error()
		if i := strings.Index(msg, ErrBlocked.Error()+": "); i >= 0 {
			msg = msg[i+len(ErrBlocked.Error())+2:]
		}
		return "Blocked: " + msg
	case errors.As(err, &dnsErr):
		return "DNS lookup failed for " + dnsErr.Name
	case errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout"):
		return "Timed out"
	case strings.Contains(err.Error(), "connection refused"):
		return "Connection refused"
	}
	msg := err.Error()
	if len(msg) > 200 {
		msg = msg[:200] + "…"
	}
	return msg
}
