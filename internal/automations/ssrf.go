package automations

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall"
	"time"

	"github.com/hostrepute/reputecast/internal/netguard"
)

// Webhook steps call URLs that workspace members type in, so the HTTP client
// refuses anything that isn't a public address: the check runs on the IP
// actually dialed (after DNS, on every redirect), which also defeats DNS
// rebinding.

// ErrBlockedAddress is returned when a webhook would reach a private,
// loopback, link-local or otherwise internal address.
var ErrBlockedAddress = errors.New("the address is private or internal")

// blockedPrefixes are what webhook steps refuse beyond netguard's ranges
// (private, loopback, link-local, metadata, multicast and IPv4 tunnels):
// documentation and benchmarking ranges, the 6to4 relay, discard, the old
// site-local range, and NAT64's well-known prefix even to a public host.
var blockedPrefixes = func() []netip.Prefix {
	var out []netip.Prefix
	for _, s := range []string{
		"192.0.2.0/24", "192.88.99.0/24", "198.51.100.0/24", "203.0.113.0/24",
		"64:ff9b::/96", "100::/64", "2001:db8::/32", "fec0::/10",
	} {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}()

// blockedIP reports whether ip is anything but a public unicast address.
func blockedIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return true
	}
	ip = ip.Unmap()
	if (netguard.Policy{}).Check(ip) != nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || !ip.IsGlobalUnicast() {
		return true
	}
	for _, p := range blockedPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// blockedHost rejects hosts that are internal by their name or literal address.
func blockedHost(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
	if h == "" {
		return true
	}
	if ip, err := netip.ParseAddr(h); err == nil {
		return blockedIP(ip)
	}
	if h == "localhost" {
		return true
	}
	for _, suffix := range []string{".localhost", ".local", ".internal", ".lan", ".home.arpa", ".intranet", ".corp"} {
		if strings.HasSuffix(h, suffix) {
			return true
		}
	}
	// Single-label names resolve through search domains to internal hosts.
	return !strings.Contains(h, ".")
}

// safeDialControl runs after DNS resolution, right before connecting.
func safeDialControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || blockedIP(ip) {
		return fmt.Errorf("%w (%s)", ErrBlockedAddress, host)
	}
	return nil
}

// NewWebhookClient returns the HTTP client webhook steps use: public
// addresses only, short timeouts, at most 3 redirects, no proxy from the
// environment (a proxy would hide the destination from the check).
func NewWebhookClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second, Control: safeDialControl}
	tr := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("stopped after 3 redirects")
			}
			if req.URL.Scheme != "https" && req.URL.Scheme != "http" {
				return fmt.Errorf("redirect to %s:// isn't allowed", req.URL.Scheme)
			}
			if blockedHost(req.URL.Hostname()) {
				return fmt.Errorf("%w (%s)", ErrBlockedAddress, req.URL.Hostname())
			}
			return nil
		},
	}
}
