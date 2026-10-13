package forms

import (
	"encoding/base64"
	"encoding/binary"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// --- timestamp tokens ----------------------------------------------------------

// FormToken signs when a form was rendered, for one form. It is the CSRF
// token of the hosted page and the timing check of every form: a
// submission needs a token at least MinSeconds old and at most a day old.
// It is sealed with the instance key, so it can't be forged or moved to
// another form.
func (s *Service) FormToken(formKey string, at time.Time) string {
	if s.Box == nil {
		return ""
	}
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(at.UnixMilli()))
	sealed, err := s.Box.Seal(b[:], "form_token:"+formKey)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(sealed)
}

// tokenAge opens a token and returns how long ago it was issued.
func (s *Service) tokenAge(formKey, token string, now time.Time) (time.Duration, bool) {
	if s.Box == nil || token == "" || len(token) > 200 {
		return 0, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, false
	}
	pt, err := s.Box.Open(raw, "form_token:"+formKey)
	if err != nil || len(pt) != 8 {
		return 0, false
	}
	issued := time.UnixMilli(int64(binary.BigEndian.Uint64(pt)))
	age := now.Sub(issued)
	if age < -time.Minute || age > 24*time.Hour {
		return 0, false
	}
	return age, true
}

// formKey identifies a form in tokens: its public id, or the workspace's
// built-in hosted form.
func formKey(c Config) string {
	if c.Builtin {
		return "builtin:" + c.WorkspaceID.String()
	}
	return c.PublicID
}

// --- origins ---------------------------------------------------------------------

// OriginHost reads the host a browser request came from: the Origin header,
// else the Referer. Empty when neither is sent (not a browser form post).
func OriginHost(origin, referer string) string {
	for _, raw := range []string{origin, referer} {
		if raw == "" || raw == "null" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			continue
		}
		return strings.ToLower(u.Host)
	}
	return ""
}

// OriginAllowed reports whether a form accepts submissions from host: its
// allowed domains, the instance and the custom domain. Ports must match
// when an allowed entry names one.
func OriginAllowed(host string, allowed, own []string) bool {
	if host == "" {
		return false
	}
	h := strings.ToLower(host)
	bare := h
	if i := strings.LastIndexByte(h, ':'); i > 0 && !strings.Contains(h[i:], "]") {
		bare = h[:i]
	}
	match := func(list []string) bool {
		return slices.ContainsFunc(list, func(a string) bool {
			a = strings.ToLower(a)
			if strings.Contains(a, ":") {
				return a == h
			}
			return a == bare
		})
	}
	return match(own) || match(allowed)
}

// --- rate limits -----------------------------------------------------------------

// limiter keeps a token bucket per key (client address, form) in this
// process. Idle buckets are dropped after ten minutes.
type limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	lim  *rate.Limiter
	seen time.Time
}

func (s *Service) rates() *limiter {
	s.limitsOnce.Do(func() { s.limits = &limiter{buckets: map[string]*bucket{}} })
	return s.limits
}

// allow takes one token from key's bucket of perMinute tokens a minute.
func (l *limiter) allow(key string, perMinute int, now time.Time) bool {
	if perMinute <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) > time.Minute || len(l.buckets) > 200_000 {
		for k, b := range l.buckets {
			if now.Sub(b.seen) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		if len(l.buckets) > 200_000 {
			l.buckets = map[string]*bucket{}
		}
		l.lastSweep = now
	}
	b := l.buckets[key]
	if b == nil {
		b = &bucket{lim: rate.NewLimiter(rate.Limit(float64(perMinute)/60), perMinute)}
		l.buckets[key] = b
	}
	b.seen = now
	return b.lim.AllowN(now, 1)
}

// Rate limits on public submissions beyond the per-IP limit of each form.
const (
	formSubmitsPerMinute = 120 // one form, every address
	prefLinksPerMinute   = 3   // preference links asked from one address
)

// isBot reports user agents that are crawlers or link previews, which
// shouldn't count as form views.
func isBot(ua string) bool {
	ua = strings.ToLower(ua)
	if ua == "" {
		return true
	}
	for _, s := range []string{"bot", "crawler", "spider", "slurp", "preview", "headless", "curl/", "wget/", "python-requests", "go-http-client", "facebookexternalhit", "monitor"} {
		if strings.Contains(ua, s) {
			return true
		}
	}
	return false
}

// Allow takes one request from key's bucket, which refills once every
// interval (one preference link per address every ten minutes).
func (s *Service) Allow(key string, every time.Duration) bool {
	l := s.rates()
	now := s.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[key]
	if b == nil {
		b = &bucket{lim: rate.NewLimiter(rate.Every(every), 1)}
		l.buckets[key] = b
	}
	b.seen = now
	return b.lim.AllowN(now, 1)
}

// AllowPerMinute takes one request from key's bucket of n a minute.
func (s *Service) AllowPerMinute(key string, n int) bool {
	return s.rates().allow(key, n, s.now())
}
