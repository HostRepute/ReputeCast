package reports

import (
	"sync"
	"time"
)

// Limiter is a fixed-window rate limiter per client address for the public
// report endpoint: a cap on requests, and a much lower cap on lookups of
// links that don't exist, so guessing tokens is pointless.
type Limiter struct {
	// Requests per Window per address.
	Requests int
	// Misses (unknown or revoked links) per MissWindow per address.
	Misses     int
	Window     time.Duration
	MissWindow time.Duration

	mu    sync.Mutex
	hits  map[string]*window
	miss  map[string]*window
	sweep time.Time
}

type window struct {
	start time.Time
	n     int
}

// NewLimiter returns the default limits: 60 requests a minute and 20
// unknown links per 10 minutes per address.
func NewLimiter() *Limiter {
	return &Limiter{Requests: 60, Misses: 20, Window: time.Minute, MissWindow: 10 * time.Minute}
}

func (l *Limiter) init(now time.Time) {
	if l.hits == nil {
		l.hits, l.miss = map[string]*window{}, map[string]*window{}
	}
	if now.Sub(l.sweep) < time.Minute {
		return
	}
	l.sweep = now
	for k, w := range l.hits {
		if now.Sub(w.start) >= l.Window {
			delete(l.hits, k)
		}
	}
	for k, w := range l.miss {
		if now.Sub(w.start) >= l.MissWindow {
			delete(l.miss, k)
		}
	}
}

func bump(m map[string]*window, key string, now time.Time, size time.Duration) int {
	w := m[key]
	if w == nil || now.Sub(w.start) >= size {
		w = &window{start: now}
		m[key] = w
	}
	w.n++
	return w.n
}

// Allow counts a request from key and reports whether it may proceed: it
// is refused over the request cap, or when the address already looked up
// too many unknown links.
func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.init(now)
	if w := l.miss[key]; w != nil && now.Sub(w.start) < l.MissWindow && w.n >= l.Misses {
		return false
	}
	return bump(l.hits, key, now, l.Window) <= l.Requests
}

// Miss records a lookup of an unknown, revoked or expired link.
func (l *Limiter) Miss(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.init(now)
	bump(l.miss, key, now, l.MissWindow)
}
