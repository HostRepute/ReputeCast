package forms

import "time"

// HourlyLeft reports whether key's bucket of n an hour has one left,
// without taking it: a form counts only what it accepts (a sponsor's
// request, M65), checked before and taken after (HourlyTake).
func (s *Service) HourlyLeft(key string, n int) bool {
	l, now := s.rates(), s.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[key]
	return b == nil || b.lim.TokensAt(now) >= 1
}

// HourlyTake takes one from key's bucket of n an hour.
func (s *Service) HourlyTake(key string, n int) {
	if n <= 0 {
		return
	}
	s.rates().take(key, time.Hour/time.Duration(n), n, s.now())
}
