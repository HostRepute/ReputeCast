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
//
// Anyone on the internet can send forwarding headers, and a trusted
// address can make up readers' addresses (rate limits, the audit log, the
// bot filter). So only an address on a private network is ever named as
// one to add, and only once it has behaved like a proxy for a while:
// nearly all its requests carried a forwarding header, minute after
// minute. A NAT in front of ReputeCast passes on the headers of the few
// clients that send one, and a burst through it names nobody. A public
// address that forwarded most of the traffic for a long time may be a CDN
// or load balancer: admins are asked to check whose it is and, if it is
// theirs, to add the ranges its provider publishes (security review).
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
	// privateShare: a reverse proxy adds a forwarding header to nearly all
	// of its own requests, its checks of /healthz, /readyz and /metrics
	// aside. A window counts for a private address only when at least this
	// share of its requests carried one; a NAT (Docker's userland proxy, a
	// load balancer's SNAT) passes on the headers of the few clients that
	// send one.
	privateShare = 0.95
	// publicShare and publicMin: a window counts for a public address (a
	// CDN or load balancer in front of ReputeCast) only when most of its
	// requests came through untrusted public addresses; a reader's office
	// proxy adding its own header must not count.
	publicShare = 0.8
	publicMin   = 100
	// privateWindows and publicWindows: the windows that counted an
	// address needs, with no gap of streakGap between two of them, before
	// admins hear of it: ten minutes for a private address, an hour of most
	// of the traffic for a public one. A burst names nobody.
	privateWindows = 10
	publicWindows  = 60
	streakGap      = 24 * time.Hour
	// maxPeers: the addresses a window counts, at most maxPublic of them
	// public; as many are followed across windows. A sender that rotates
	// its address (an IPv6 /64 has plenty) can't grow the counts, the
	// table or the notifications.
	maxPeers  = 32
	maxPublic = 8
	// maxTold and maxToldDaily: the proxies admins are told about in one
	// run and in a day; the others wait for a later run.
	maxTold      = 3
	maxToldDaily = 10
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

	mu     sync.Mutex
	window time.Time
	// total: the window's requests; viaPublic: those of them an untrusted
	// public address forwarded.
	total, viaPublic int64
	// peers: the window's untrusted peers that forwarded a request, and
	// private ones that sent one without a header; publicPeers of them
	// are public.
	peers       map[netip.Addr]*seen
	publicPeers int
	// proxies and cdn: the private and public addresses whose windows
	// counted, followed across windows.
	proxies, cdn map[netip.Addr]*streak
}

type seen struct {
	header string
	n      int64 // requests with a forwarding header
	plain  int64 // a private peer's requests without one, checks aside
	last   time.Time
}

// streak is an address's windows that counted, and their requests not
// written yet.
type streak struct {
	windows  int
	requests int64
	last     time.Time
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
	peer, ok := peerAddr(r.RemoteAddr)
	// Requests straight from the internet don't matter. A private peer's
	// requests without a header do: they tell a NAT from a proxy.
	ok = ok && !trusted(peer, d.Trusted) && (header != "" || private(peer) && !check(r))
	now := d.now()
	d.mu.Lock()
	if d.window.IsZero() {
		d.window = now
	}
	d.total++
	if ok {
		d.count(peer, header, now)
	}
	var batch []Seen
	if now.Sub(d.window) >= flushEvery {
		batch = d.take(now)
		d.window = now
	}
	d.mu.Unlock()
	if len(batch) > 0 && d.Q != nil {
		go d.write(batch)
	}
}

// count notes a request an untrusted peer forwarded, or a private peer
// sent without a header (callers hold mu). A window keeps maxPeers
// addresses, maxPublic of them public; the requests of the others only
// count towards the window's share.
func (d *Detector) count(peer netip.Addr, header string, now time.Time) {
	public := !private(peer)
	if public {
		d.viaPublic++ // only forwarded requests of public peers come here
	}
	s := d.peers[peer]
	if s == nil {
		if len(d.peers) >= maxPeers || public && d.publicPeers >= maxPublic {
			return
		}
		if d.peers == nil {
			d.peers = map[netip.Addr]*seen{}
		}
		s = &seen{}
		d.peers[peer] = s
		if public {
			d.publicPeers++
		}
	}
	if header == "" {
		s.plain++
		return
	}
	s.header, s.last = header, now
	s.n++
}

// Seen is a proxy's requests in a window.
type Seen struct {
	Peer     netip.Addr
	Header   string
	Requests int64
	At       time.Time
}

// take returns the window's deployment proxies and starts a new window
// (callers hold mu): the addresses whose windows counted for long enough.
// A private address's window counts when nearly all its requests carried
// a forwarding header (privateWindows of them), a public address's when
// most of the window's traffic came through untrusted public addresses
// (publicWindows).
func (d *Detector) take(now time.Time) []Seen {
	if d.proxies == nil {
		d.proxies, d.cdn = map[netip.Addr]*streak{}, map[netip.Addr]*streak{}
	}
	for _, streaks := range []map[netip.Addr]*streak{d.proxies, d.cdn} {
		for peer, c := range streaks {
			if now.Sub(c.last) >= streakGap {
				delete(streaks, peer)
			}
		}
	}
	var out []Seen
	mostly := d.total >= publicMin && float64(d.viaPublic) >= publicShare*float64(d.total)
	for peer, s := range d.peers {
		var n int64
		switch {
		case s.n == 0: // requests without a header only
		case private(peer):
			if float64(s.n) >= privateShare*float64(s.n+s.plain) {
				n = qualify(d.proxies, maxPeers, peer, s.n, now, privateWindows)
			}
		case mostly:
			n = qualify(d.cdn, maxPublic, peer, s.n, now, publicWindows)
		}
		if n > 0 {
			out = append(out, Seen{Peer: peer, Header: s.header, Requests: n, At: s.last})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Peer.Less(out[j].Peer) })
	d.peers, d.publicPeers, d.total, d.viaPublic = nil, 0, 0, 0
	return out
}

// qualify adds a window that counted, with its n forwarded requests, to
// peer's streak in streaks (which follows at most limit addresses), and
// returns the requests to write once the streak has need windows: all of
// the streak's the first time, the window's after that; 0 before.
func qualify(streaks map[netip.Addr]*streak, limit int, peer netip.Addr, n int64, now time.Time, need int) int64 {
	c := streaks[peer]
	if c == nil {
		if len(streaks) >= limit {
			return 0
		}
		c = &streak{}
		streaks[peer] = c
	}
	c.windows++
	c.last = now
	c.requests += n
	if c.windows < need {
		return 0
	}
	n, c.requests = c.requests, 0
	return n
}

// Flush writes what the current window saw now (tests, shutdown).
func (d *Detector) Flush(ctx context.Context) error {
	d.mu.Lock()
	now := d.now()
	batch := d.take(now)
	d.window = now
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

// check: ReputeCast's health checks and metrics, which a proxy or an
// orchestrator asks for itself, without a forwarding header.
func check(r *http.Request) bool {
	if r.URL == nil {
		return false
	}
	switch r.URL.Path {
	case "/healthz", "/readyz", "/metrics":
		return true
	}
	return false
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

// private: loopback (127.0.0.0/8, ::1), private (RFC 1918, fc00::/7),
// link-local (169.254.0.0/16, fe80::/10) and shared (100.64.0.0/10)
// addresses, where a reverse proxy in front of ReputeCast usually is, and
// the only addresses admins are ever told to add.
func private(ip netip.Addr) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || cgnat.Contains(ip)
}
