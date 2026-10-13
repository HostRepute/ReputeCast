package remote

import (
	"sync"
	"time"
)

// Circuit breaker settings: after BreakerThreshold adapter failures in a
// row, calls fail at once for BreakerCooldown (doubling up to
// BreakerMaxCooldown while probes keep failing).
const (
	BreakerThreshold   = 5
	BreakerCooldown    = 30 * time.Second
	BreakerMaxCooldown = 5 * time.Minute
)

// Breaker states.
const (
	BreakerClosed   = "closed"
	BreakerOpen     = "open"
	BreakerHalfOpen = "half_open"
)

// breaker stops calls to an adapter process that keeps failing, so a dead
// or hanging adapter can't hold up sends for its timeout on every call. One
// breaker serves every connection made through an installed adapter.
type breaker struct {
	mu        sync.Mutex
	now       func() time.Time
	failures  int
	cooldown  time.Duration
	openUntil time.Time
	probing   bool
	lastError string
}

func newBreaker(now func() time.Time) *breaker {
	if now == nil {
		now = time.Now
	}
	return &breaker{now: now}
}

// allow reports whether a call may go out. When the breaker is open it
// returns false and when it reopens; after the pause it lets exactly one
// probe through (half-open) until that probe reports back.
func (b *breaker) allow() (bool, time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failures < BreakerThreshold {
		return true, time.Time{}
	}
	if b.now().Before(b.openUntil) || b.probing {
		return false, b.openUntil
	}
	b.probing = true
	return true, time.Time{}
}

// success closes the breaker.
func (b *breaker) success() {
	b.mu.Lock()
	b.failures, b.cooldown, b.probing, b.openUntil, b.lastError = 0, 0, false, time.Time{}, ""
	b.mu.Unlock()
}

// failure counts an adapter failure and opens the breaker at the threshold.
func (b *breaker) failure(reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	b.lastError = reason
	if b.failures < BreakerThreshold {
		return
	}
	switch {
	case b.cooldown == 0:
		b.cooldown = BreakerCooldown
	case b.probing:
		b.cooldown = min(b.cooldown*2, BreakerMaxCooldown)
	}
	b.probing = false
	b.openUntil = b.now().Add(b.cooldown)
}

// release ends a probe that neither succeeded nor failed (the caller gave up).
func (b *breaker) release() {
	b.mu.Lock()
	b.probing = false
	b.mu.Unlock()
}

// state describes the breaker for the runtime view.
func (b *breaker) state() (string, time.Time, string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case b.failures < BreakerThreshold:
		return BreakerClosed, time.Time{}, b.lastError
	case b.now().Before(b.openUntil):
		return BreakerOpen, b.openUntil, b.lastError
	}
	return BreakerHalfOpen, time.Time{}, b.lastError
}
