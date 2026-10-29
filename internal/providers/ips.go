package providers

import (
	"context"
	"net/netip"
	"slices"
	"strings"
)

// SendingIPs is what a provider's API says about the IPs an account's mail
// leaves from.
type SendingIPs struct {
	// Dedicated is true when the account sends from IPs of its own, leased
	// or dedicated to it, and false on the provider's shared pool, whose IPs
	// it doesn't list: they change from message to message and are kept
	// warm by every customer's mail.
	Dedicated bool
	// IPs are the dedicated IPs as the provider lists them; none on a
	// shared pool.
	IPs []string
}

// IPReporter is implemented by adapters whose API lists the dedicated IPs an
// account sends from (Capabilities.ReportsIPs says so in the catalog).
// ReputeCast asks when a provider is connected, when its connection step is
// repeated and every day, and keeps providers.ip_mode and ips from the
// answer: the canary's new-IP trigger, Send Readiness's IP blocklist rows,
// Deliverability and warm-up read them.
//
// A provider that can't be asked (an outage, refused credentials, a key
// without the scope) is an error, never an empty list: ReputeCast keeps what
// it knew. Adapters whose API can't list an account's IPs don't implement
// it, and people set those IPs by hand on Sending providers.
type IPReporter interface {
	SendingIPs(ctx context.Context) (SendingIPs, error)
}

// MaxSendingIPs is how many sending IPs a person can list by hand for one
// provider.
const MaxSendingIPs = 64

// ParseIP reads one sending IP: an IPv4 or IPv6 address, without a zone or
// a prefix length. An IPv4-mapped IPv6 address reads as the IPv4 address.
func ParseIP(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil || a.Zone() != "" {
		return netip.Addr{}, false
	}
	return a.Unmap(), true
}

// CleanIPs returns the addresses among ips written the canonical way, in
// address order (IPv4 first) and each once, as providers.ips keeps them,
// and the entries that aren't addresses, as given.
func CleanIPs(ips []string) (clean, bad []string) {
	addrs := make([]netip.Addr, 0, len(ips))
	for _, s := range ips {
		if a, ok := ParseIP(s); ok {
			addrs = append(addrs, a)
		} else {
			bad = append(bad, s)
		}
	}
	slices.SortFunc(addrs, netip.Addr.Compare)
	addrs = slices.Compact(addrs)
	clean = make([]string, len(addrs))
	for i, a := range addrs {
		clean[i] = a.String()
	}
	return clean, bad
}
