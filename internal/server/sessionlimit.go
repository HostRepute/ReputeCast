package server

import (
	"sync"
	"time"
)

// SessionLimiter limits the API requests of each signed-in browser session
// (review 2026-11 OP-6). It counts in one-minute windows and weighs the
// previous window by how much of it still overlaps the last minute, like
// the per-key limiter. Each web replica counts on its own, so behind a load
// balancer a session gets up to the limit on every replica.
type SessionLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	start     time.Time
	cur, prev map[string]int
}

// NewSessionLimiter allows perMinute requests per session.
func NewSessionLimiter(perMinute int) *SessionLimiter {
	return &SessionLimiter{limit: perMinute, window: time.Minute, now: time.Now, cur: map[string]int{}, prev: map[string]int{}}
}

// Allow counts one request of the session, or says how long to wait.
func (l *SessionLimiter) Allow(session string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	start := now.Truncate(l.window)
	switch {
	case start.Equal(l.start):
	case start.Sub(l.start) == l.window:
		l.prev, l.cur, l.start = l.cur, map[string]int{}, start
	default:
		l.prev, l.cur, l.start = map[string]int{}, map[string]int{}, start
	}
	left := float64(l.window-now.Sub(start)) / float64(l.window)
	rate := float64(l.prev[session])*left + float64(l.cur[session])
	if rate+1 > float64(l.limit) {
		return false, l.window - now.Sub(start)
	}
	l.cur[session]++
	return true, 0
}
