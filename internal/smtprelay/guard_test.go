package smtprelay

import (
	"bufio"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// TestGuardBlocksRepeatedFailedLogins: five failed logins within ten
// minutes block the address for fifteen; a success forgets the failures
// (review 2026-11 TX-6).
func TestGuardBlocksRepeatedFailedLogins(t *testing.T) {
	g := newGuard()
	ip, other := netip.MustParseAddr("203.0.113.9"), netip.MustParseAddr("198.51.100.4")
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	for i := range AuthFailures - 1 {
		if g.failed(ip, now.Add(time.Duration(i)*time.Minute)) {
			t.Fatalf("blocked after %d failures", i+1)
		}
	}
	if !g.failed(ip, now.Add(4*time.Minute)) {
		t.Fatal("the fifth failure should block")
	}
	if w := g.blocked(ip, now.Add(5*time.Minute)); w != 14*time.Minute {
		t.Fatalf("blocked for %s, want 14m", w)
	}
	if g.blocked(other, now) != 0 {
		t.Fatal("another address isn't blocked")
	}
	if g.blocked(ip, now.Add(4*time.Minute+AuthBlock)) != 0 {
		t.Fatal("the block ends")
	}
	// Failures spread over more than the window don't add up.
	for i := range AuthFailures {
		if g.failed(other, now.Add(time.Duration(i)*(AuthFailureWindow/2+time.Second))) && i < AuthFailures-1 {
			t.Fatalf("spread failures blocked at %d", i)
		}
	}
	g.succeeded(other)
	if g.blocked(other, now.Add(time.Hour)) != 0 {
		t.Fatal("a success forgets")
	}
}

// TestLimitListenerCapsConnections: connections over the per-address cap
// are refused with 421 before any SMTP; closing one frees its slot.
func TestLimitListenerCapsConnections(t *testing.T) {
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	g := newGuard()
	g.maxPerIP = 2
	l := limitListener{Listener: raw, g: g}
	accepted := make(chan net.Conn, 4)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			accepted <- c
		}
	}()
	dial := func() net.Conn {
		t.Helper()
		c, err := net.Dial("tcp", raw.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	a, b := dial(), dial()
	defer a.Close()
	defer b.Close()
	first, second := <-accepted, <-accepted
	third := dial()
	defer third.Close()
	_ = third.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, _ := bufio.NewReader(third).ReadString('\n')
	if !strings.HasPrefix(line, "421 4.7.0") {
		t.Fatalf("over the cap: %q", line)
	}
	_ = first.Close()
	fourth := dial()
	defer fourth.Close()
	select {
	case c := <-accepted:
		_ = c.Close()
	case <-time.After(5 * time.Second):
		t.Fatal("a freed slot takes a new connection")
	}
	_ = second.Close()
}
