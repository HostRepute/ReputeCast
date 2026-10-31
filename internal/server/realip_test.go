package server

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

// TestRealIP: the client's address comes from the forwarding headers only
// when a trusted proxy sent them, read from the right, so a value the
// client wrote itself never wins over the hop our proxy added (security
// review: HAProxy's option forwardfor adds its own header line).
func TestRealIP(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	for _, c := range []struct {
		name, peer string
		xff        []string // header lines, in order
		realIP     string
		want       string
	}{
		{name: "no proxy", peer: "198.51.100.7:4567", want: "198.51.100.7:4567"},
		{name: "an untrusted peer's header", peer: "198.51.100.7:4567", xff: []string{"203.0.113.9"}, want: "198.51.100.7:4567"},
		{name: "a trusted proxy", peer: "10.0.0.5:4567", xff: []string{"203.0.113.9"}, want: "203.0.113.9:0"},
		{name: "a spoofed left-most value", peer: "10.0.0.5:4567", xff: []string{"6.6.6.6, 203.0.113.9"}, want: "203.0.113.9:0"},
		{name: "trusted hops are skipped", peer: "10.0.0.5:4567", xff: []string{"6.6.6.6, 203.0.113.9, 10.0.0.7"}, want: "203.0.113.9:0"},
		{name: "the proxy adds a line of its own", peer: "10.0.0.5:4567", xff: []string{"6.6.6.6", "203.0.113.9"}, want: "203.0.113.9:0"},
		{name: "lines and lists", peer: "10.0.0.5:4567", xff: []string{"6.6.6.6, 7.7.7.7", "203.0.113.9, 10.0.0.7"}, want: "203.0.113.9:0"},
		{name: "a spoofed line behind trusted hops", peer: "10.0.0.5:4567", xff: []string{"6.6.6.6", "203.0.113.9", "10.0.0.7"}, want: "203.0.113.9:0"},
		{name: "only trusted hops", peer: "10.0.0.5:4567", xff: []string{"10.0.0.8", "10.0.0.7"}, want: "10.0.0.8:0"},
		{name: "a hop that isn't an address", peer: "10.0.0.5:4567", xff: []string{"203.0.113.9", "unknown"}, want: "10.0.0.5:4567"},
		{name: "X-Real-IP", peer: "10.0.0.5:4567", realIP: "203.0.113.9", want: "203.0.113.9:0"},
		{name: "IPv4 behind an IPv6 listener", peer: "[::ffff:10.0.0.5]:4567", xff: []string{"6.6.6.6", "203.0.113.9"}, want: "203.0.113.9:0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var got string
			h := RealIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.RemoteAddr }))
			r := httptest.NewRequest(http.MethodGet, "/t/o/x", nil)
			r.RemoteAddr = c.peer
			for _, line := range c.xff {
				r.Header.Add("X-Forwarded-For", line)
			}
			if c.realIP != "" {
				r.Header.Set("X-Real-IP", c.realIP)
			}
			h.ServeHTTP(httptest.NewRecorder(), r)
			if got != c.want {
				t.Errorf("RemoteAddr = %q, want %q", got, c.want)
			}
		})
	}
}
