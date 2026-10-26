package proxycheck

import (
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

func TestDetectorTakesDeploymentProxies(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	d := &Detector{Trusted: []netip.Prefix{netip.MustParsePrefix("10.0.0.9/32")}, Now: func() time.Time { return now }}
	see := func(peer, header string, n int) {
		for range n {
			r := httptest.NewRequest("GET", "/t/o/x", nil)
			r.RemoteAddr = peer + ":4567"
			if header != "" {
				r.Header.Set(header, "198.51.100.7")
			}
			d.Observe(r)
		}
	}
	// A proxy on the private network, not trusted; a trusted one; readers
	// straight from the internet; a reader's office proxy adding its header.
	see("10.0.0.5", "X-Forwarded-For", 30)
	see("10.0.0.9", "X-Forwarded-For", 30)
	see("203.0.113.40", "", 60)
	see("192.0.2.77", "X-Real-IP", 3)
	d.mu.Lock()
	got := d.take()
	d.mu.Unlock()
	if len(got) != 1 || got[0].Peer != netip.MustParseAddr("10.0.0.5") || got[0].Requests != 30 || got[0].Header != "X-Forwarded-For" {
		t.Fatalf("taken: %+v", got)
	}

	// A CDN in front of ReputeCast: public addresses, but almost every
	// request comes through them.
	see("198.18.10.1", "X-Forwarded-For", 70)
	see("198.18.10.2", "Forwarded", 60)
	see("203.0.113.40", "", 10)
	d.mu.Lock()
	got = d.take()
	d.mu.Unlock()
	if len(got) != 2 || got[0].Peer != netip.MustParseAddr("198.18.10.1") || got[1].Header != "Forwarded" {
		t.Fatalf("a CDN's addresses: %+v", got)
	}

	// IPv4 behind an IPv6 listener, loopback.
	see("[::ffff:127.0.0.1]", "X-Forwarded-For", 2)
	d.mu.Lock()
	got = d.take()
	d.mu.Unlock()
	if len(got) != 1 || got[0].Peer != netip.MustParseAddr("127.0.0.1") {
		t.Fatalf("loopback: %+v", got)
	}
}
