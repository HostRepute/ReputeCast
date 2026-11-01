package smtprelay

import (
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"
)

// Abuse limits of the relay (review 2026-11 TX-6): each web node caps its
// SMTP connections, in all and per address, and an address that keeps
// failing to log in is refused for a while.
const (
	MaxConnections      = 500
	MaxConnectionsPerIP = 20
	// AuthFailures failed logins within AuthFailureWindow block the address
	// for AuthBlock.
	AuthFailures      = 5
	AuthFailureWindow = 10 * time.Minute
	AuthBlock         = 15 * time.Minute
)

// guard counts connections and failed logins per address.
type guard struct {
	mu    sync.Mutex
	total int
	conns map[netip.Addr]int
	fails map[netip.Addr]*failures
	// limits, overridable in tests.
	maxTotal, maxPerIP int
}

type failures struct {
	n            int
	first, until time.Time
}

func newGuard() *guard {
	return &guard{conns: map[netip.Addr]int{}, fails: map[netip.Addr]*failures{}, maxTotal: MaxConnections, maxPerIP: MaxConnectionsPerIP}
}

// open takes a connection slot for ip, or says the relay or the address is
// full.
func (g *guard) open(ip netip.Addr) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.total >= g.maxTotal || g.conns[ip] >= g.maxPerIP {
		return false
	}
	g.total++
	g.conns[ip]++
	return true
}

func (g *guard) close(ip netip.Addr) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.total--
	if g.conns[ip]--; g.conns[ip] <= 0 {
		delete(g.conns, ip)
	}
}

// blocked is how long ip still can't log in (0: it can).
func (g *guard) blocked(ip netip.Addr, now time.Time) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	f := g.fails[ip]
	if f == nil || !now.Before(f.until) {
		return 0
	}
	return f.until.Sub(now)
}

// failed counts a failed login; the AuthFailures-th within the window
// blocks the address, and reports true.
func (g *guard) failed(ip netip.Addr, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.fails) > 100_000 {
		// Many addresses at once: keep the blocked ones.
		for k, f := range g.fails {
			if !now.Before(f.until) {
				delete(g.fails, k)
			}
		}
	}
	f := g.fails[ip]
	if f == nil || now.Sub(f.first) > AuthFailureWindow {
		f = &failures{first: now}
		g.fails[ip] = f
	}
	f.n++
	if f.n >= AuthFailures {
		f.until, f.n, f.first = now.Add(AuthBlock), 0, now
		return true
	}
	return false
}

// succeeded forgets an address's failures.
func (g *guard) succeeded(ip netip.Addr) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.fails, ip)
}

// limitListener refuses connections over the guard's caps with 421 before
// any SMTP is spoken.
type limitListener struct {
	net.Listener
	g *guard
}

func (l limitListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		ip := remoteIP(c)
		if l.g.open(ip) {
			return &guardedConn{Conn: c, g: l.g, ip: ip}, nil
		}
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, _ = fmt.Fprint(c, "421 4.7.0 Too many connections, try again later\r\n")
		_ = c.Close()
	}
}

type guardedConn struct {
	net.Conn
	g    *guard
	ip   netip.Addr
	once sync.Once
}

func (c *guardedConn) Close() error {
	c.once.Do(func() { c.g.close(c.ip) })
	return c.Conn.Close()
}

func remoteIP(c net.Conn) netip.Addr {
	if ap, err := netip.ParseAddrPort(c.RemoteAddr().String()); err == nil {
		return ap.Addr().Unmap()
	}
	return netip.Addr{}
}
