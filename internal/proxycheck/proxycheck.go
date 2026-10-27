// Package proxycheck notices requests that reach ReputeCast through a
// reverse proxy it doesn't trust, and tells the instance's admins how to
// fix it (review AN-8).
//
// Behind a proxy ReputeCast doesn't trust, every request carries the
// proxy's address instead of the reader's (server.RealIP ignores the
// forwarding headers of untrusted peers, as it must). Opens and clicks
// then all look like they come from one address: the bot filter can't
// tell Apple's privacy proxy or a mail scanner's network from a person,
// and rate limits treat every reader as one. The fix is configuration:
// http.trusted_proxies (REPUTECAST_HTTP__TRUSTED_PROXIES) lists the proxy.
package proxycheck

import (
	"context"
	"embed"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The setting that fixes it, as the configuration file and the
// environment name it.
const (
	Setting = "http.trusted_proxies"
	Env     = "REPUTECAST_HTTP__TRUSTED_PROXIES"
)

const (
	// minRequests: a proxy's requests before admins are told.
	minRequests = 20
	// flushEvery: how often the counts are written.
	flushEvery = time.Minute
	// publicShare and publicMin: a public address counts as the
	// deployment's proxy (a CDN in front of ReputeCast) only when most
	// requests in a window came through untrusted proxies; a reader's
	// office proxy adding its own header must not trigger it.
	publicShare = 0.8
	publicMin   = 100
	// remindAfter: admins are told again while it lasts.
	remindAfter = 7 * 24 * time.Hour
	// forgetAfter: a proxy not seen for this long is forgotten.
	forgetAfter = 30 * 24 * time.Hour
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "proxycheck")

// Detector watches requests for forwarding headers from peers that aren't
// trusted. It is safe for concurrent use.
type Detector struct {
	Q       *store.Queries
	Log     *slog.Logger
	Trusted []netip.Prefix
	Now     func() time.Time

	mu      sync.Mutex
	window  time.Time
	total   int64
	proxied int64
	peers   map[netip.Addr]*seen
}

type seen struct {
	header string
	n      int64
	last   time.Time
}

func (d *Detector) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// Middleware notes each request; it runs before server.RealIP rewrites the
// address of trusted proxies' requests.
func (d *Detector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.Observe(r)
		next.ServeHTTP(w, r)
	})
}

// Observe notes a request and, once a window is over, writes what it saw.
func (d *Detector) Observe(r *http.Request) {
	header := forwardingHeader(r)
	now := d.now()
	d.mu.Lock()
	if d.window.IsZero() {
		d.window = now
	}
	d.total++
	if header != "" {
		if peer, ok := peerAddr(r.RemoteAddr); ok && !trusted(peer, d.Trusted) {
			d.proxied++
			if d.peers == nil {
				d.peers = map[netip.Addr]*seen{}
			}
			s := d.peers[peer]
			if s == nil {
				s = &seen{}
				d.peers[peer] = s
			}
			s.header, s.last = header, now
			s.n++
		}
	}
	var batch []Seen
	if now.Sub(d.window) >= flushEvery {
		batch = d.take()
		d.window = now
	}
	d.mu.Unlock()
	if len(batch) > 0 && d.Q != nil {
		go d.write(batch)
	}
}

// Seen is a proxy's requests in a window.
type Seen struct {
	Peer     netip.Addr
	Header   string
	Requests int64
	At       time.Time
}

// take returns the window's deployment proxies and starts a new window
// (callers hold mu): private and loopback peers, and public ones when most
// requests came through untrusted proxies.
func (d *Detector) take() []Seen {
	var out []Seen
	mostly := d.total >= publicMin && float64(d.proxied) >= publicShare*float64(d.total)
	for peer, s := range d.peers {
		if private(peer) || mostly {
			out = append(out, Seen{Peer: peer, Header: s.header, Requests: s.n, At: s.last})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Peer.Less(out[j].Peer) })
	d.peers, d.total, d.proxied = nil, 0, 0
	return out
}

// Flush writes what the current window saw now (tests, shutdown).
func (d *Detector) Flush(ctx context.Context) error {
	d.mu.Lock()
	batch := d.take()
	d.window = d.now()
	d.mu.Unlock()
	return d.store(ctx, batch)
}

func (d *Detector) write(batch []Seen) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.store(ctx, batch); err != nil && d.Log != nil {
		d.Log.Warn("untrusted proxies not recorded", "err", err)
	}
}

func (d *Detector) store(ctx context.Context, batch []Seen) error {
	for _, s := range batch {
		if err := d.Q.ProxySeen(ctx, store.ProxySeenParams{Peer: s.Peer, Header: s.Header, Requests: s.Requests, SeenAt: s.At}); err != nil {
			return err
		}
	}
	return nil
}

// forwardingHeader names the forwarding header a request carries, or "".
func forwardingHeader(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-IP", "Forwarded"} {
		if strings.TrimSpace(r.Header.Get(h)) != "" {
			return h
		}
	}
	return ""
}

func peerAddr(hostport string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return ip.Unmap(), true
}

func trusted(ip netip.Addr, nets []netip.Prefix) bool {
	for _, p := range nets {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// cgnat is the shared address space some load balancers use.
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// private: loopback, private, link-local and shared addresses, where a
// reverse proxy in front of ReputeCast usually is.
func private(ip netip.Addr) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || cgnat.Contains(ip)
}
