package proxycheck

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// fixture is a detector on a clock the test moves.
type fixture struct {
	d   *Detector
	now time.Time
}

func newFixture(trusted ...string) *fixture {
	f := &fixture{now: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)}
	f.d = &Detector{Now: func() time.Time { return f.now }}
	for _, p := range trusted {
		f.d.Trusted = append(f.d.Trusted, netip.MustParsePrefix(p))
	}
	return f
}

// see sends n requests from peer, with a forwarding header unless header
// is "".
func (f *fixture) see(peer, header string, n int) { f.get("/t/o/x", peer, header, n) }

// get sends n requests for path from peer, with a forwarding header unless
// header is "".
func (f *fixture) get(path, peer, header string, n int) {
	for range n {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.RemoteAddr = peer + ":4567"
		if header != "" {
			r.Header.Set(header, "198.51.100.7")
		}
		f.d.Observe(r)
	}
}

// take ends the window a minute later and returns what the detector takes.
func (f *fixture) take() []Seen {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	f.now = f.now.Add(flushEvery)
	got := f.d.take(f.now)
	f.d.window = f.now
	return got
}

func TestDetectorTakesDeploymentProxies(t *testing.T) {
	f := newFixture("10.0.0.9/32")
	see, take := f.see, f.take
	// A proxy on the private network, not trusted, that checks /healthz
	// itself; a trusted one; readers straight from the internet; a reader's
	// office proxy adding its header. It is taken after ten minutes of it,
	// with all of their requests.
	var got []Seen
	for i := 1; i <= privateWindows; i++ {
		see("10.0.0.5", "X-Forwarded-For", 30)
		f.get("/healthz", "10.0.0.5", "", 6)
		see("10.0.0.9", "X-Forwarded-For", 30)
		see("203.0.113.40", "", 60)
		see("192.0.2.77", "X-Real-IP", 3)
		if got = take(); i < privateWindows && len(got) != 0 {
			t.Fatalf("after %d windows: %+v", i, got)
		}
	}
	if len(got) != 1 || got[0].Peer != netip.MustParseAddr("10.0.0.5") || got[0].Requests != 30*privateWindows || got[0].Header != "X-Forwarded-For" {
		t.Fatalf("taken: %+v", got)
	}
	see("10.0.0.5", "X-Forwarded-For", 30)
	if got = take(); len(got) != 1 || got[0].Requests != 30 {
		t.Fatalf("the next window: %+v", got)
	}

	// A CDN in front of ReputeCast: public addresses, but almost every
	// request comes through them. Anyone can send forwarding headers, so
	// they are taken only once they carried most of the traffic for an hour.
	for i := 1; i <= publicWindows; i++ {
		see("198.18.10.1", "X-Forwarded-For", 70)
		see("198.18.10.2", "Forwarded", 60)
		see("203.0.113.40", "", 10)
		if got = take(); i < publicWindows && len(got) != 0 {
			t.Fatalf("a CDN's addresses after %d windows: %+v", i, got)
		}
	}
	if len(got) != 2 || got[0].Peer != netip.MustParseAddr("198.18.10.1") || got[1].Header != "Forwarded" {
		t.Fatalf("a CDN's addresses: %+v", got)
	}

	// IPv4 behind an IPv6 listener, loopback.
	for range privateWindows {
		see("[::ffff:127.0.0.1]", "X-Forwarded-For", 2)
		got = take()
	}
	if len(got) != 1 || got[0].Peer != netip.MustParseAddr("127.0.0.1") || got[0].Requests != 2*privateWindows {
		t.Fatalf("loopback: %+v", got)
	}
}

// TestDetectorPrivateBurst: a private address is taken for a proxy only
// when nearly all its requests carry a forwarding header, minute after
// minute. A burst through a NAT names nobody, nor does a NAT that passes
// on the headers of a few of its clients (security review).
func TestDetectorPrivateBurst(t *testing.T) {
	f := newFixture()
	gateway := netip.MustParseAddr("172.17.0.1")
	// A hundred requests with X-Forwarded-For through Docker's gateway.
	f.see("172.17.0.1", "X-Forwarded-For", 100)
	if got, c := f.take(), f.d.proxies[gateway]; len(got) != 0 || c == nil || c.windows != 1 {
		t.Fatalf("a burst: %+v, %+v", got, c)
	}
	// Again the next day: it starts over.
	f.now = f.now.Add(streakGap)
	f.take()
	f.see("172.17.0.1", "X-Forwarded-For", 100)
	if got, c := f.take(), f.d.proxies[gateway]; len(got) != 0 || c == nil || c.windows != 1 {
		t.Fatalf("the next day: %+v, %+v", got, c)
	}

	// A NAT whose clients mostly send no header, for hours: no window
	// counts, below 95% of its requests with a header.
	for i := range 6 * privateWindows {
		f.see("172.18.0.1", "", 6)
		f.see("172.18.0.1", "X-Forwarded-For", 94)
		f.see("172.19.0.1", "", 40)
		f.see("172.19.0.1", "X-Forwarded-For", 60)
		f.see("172.20.0.1", "", 5)
		f.see("172.20.0.1", "X-Forwarded-For", 95)
		got := f.take()
		if i+1 < privateWindows && len(got) != 0 || i+1 >= privateWindows && (len(got) != 1 || got[0].Peer != netip.MustParseAddr("172.20.0.1")) {
			t.Fatalf("window %d: %+v", i+1, got)
		}
	}
	if f.d.proxies[netip.MustParseAddr("172.18.0.1")] != nil || f.d.proxies[netip.MustParseAddr("172.19.0.1")] != nil {
		t.Fatalf("a NAT's windows counted: %+v", f.d.proxies)
	}
}

// TestDetectorPublicSender: anyone can send forwarding headers. A public
// address that sends a burst of them to an instance with little traffic
// is never taken for its proxy, and a sender that rotates its address
// grows nothing (security review).
func TestDetectorPublicSender(t *testing.T) {
	f := newFixture()
	// A hundred requests a minute with X-Forwarded-For are most of a window.
	for i := 1; i < publicWindows; i++ {
		f.see("203.0.113.66", "X-Forwarded-For", 100)
		if got := f.take(); len(got) != 0 {
			t.Fatalf("window %d: %+v", i, got)
		}
	}
	// After a day without, it starts over.
	f.now = f.now.Add(streakGap)
	f.take()
	f.see("203.0.113.66", "X-Forwarded-For", 100)
	if got, c := f.take(), f.d.cdn[netip.MustParseAddr("203.0.113.66")]; len(got) != 0 || c == nil || c.windows != 1 {
		t.Fatalf("after a day's gap: %+v, %+v", got, c)
	}

	// A sender that rotates its address: a window keeps maxPublic of them
	// and follows only maxPublic across windows.
	f = newFixture()
	for w := range 3 {
		for i := range 500 {
			f.see(fmt.Sprintf("[2001:db8::%x:%x]", w, i), "X-Forwarded-For", 1)
		}
		if len(f.d.peers) != maxPublic || f.d.viaPublic != 500 {
			t.Fatalf("window %d keeps %d addresses of %d requests", w, len(f.d.peers), f.d.viaPublic)
		}
		if got := f.take(); len(got) != 0 || len(f.d.cdn) != maxPublic {
			t.Fatalf("window %d: %+v, %d followed", w, got, len(f.d.cdn))
		}
	}
	// Addresses on a private network: a window keeps maxPeers of them and
	// as many are followed.
	for w := range privateWindows {
		for i := range 100 {
			f.see(fmt.Sprintf("[fd00::%x:%x]", w, i), "X-Forwarded-For", 1)
		}
		if len(f.d.peers) != maxPeers {
			t.Fatalf("window %d keeps %d private addresses", w, len(f.d.peers))
		}
		if got := f.take(); len(got) != 0 || len(f.d.proxies) != maxPeers {
			t.Fatalf("window %d: %+v, %d followed", w, got, len(f.d.proxies))
		}
	}
}

func TestPrivate(t *testing.T) {
	for addr, want := range map[string]bool{
		"10.1.2.3": true, "172.16.0.1": true, "192.168.1.1": true, "100.64.0.1": true, "100.127.255.254": true,
		"127.0.0.1": true, "::1": true, "169.254.1.1": true, "fe80::1": true, "fc00::1": true, "fd12:3456::1": true,
		"172.32.0.1": false, "100.128.0.1": false, "198.18.10.1": false, "203.0.113.66": false, "8.8.8.8": false,
		"2001:db8::1": false, "2606:4700::1": false,
	} {
		if got := private(netip.MustParseAddr(addr)); got != want {
			t.Errorf("private(%s) = %v", addr, got)
		}
	}
}

// TestMessages: every message is in Vietnamese, and neither language asks
// admins to add a public address.
func TestMessages(t *testing.T) {
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("not in Vietnamese: %v", missing)
	}
	args := i18n.Args{"peer": "203.0.113.66", "header": "X-Forwarded-For", "setting": Setting, "env": Env}
	for l, add := range map[i18n.Locale]string{i18n.English: "Add 203.0.113.66", i18n.Vietnamese: "thêm 203.0.113.66"} {
		body := msgs.M("notify.publicBody").Text(i18n.For(l), args)
		if strings.Contains(body, add) || !strings.Contains(body, "203.0.113.66") || !strings.Contains(body, "CDN") || !strings.Contains(body, Setting) {
			t.Errorf("%s: %s", l, body)
		}
	}
}
