package transactional

import (
	"math"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// keyLimiter enforces each key's send rate (requests per second, set on
// the key; 50 by default). Each process keeps its own buckets, so several
// web nodes behind a load balancer together allow up to their number
// times the rate.
type keyLimiter struct {
	mu      sync.Mutex
	buckets map[uuid.UUID]*keyBucket
}

type keyBucket struct {
	lim  *rate.Limiter
	rate int
	used time.Time
}

// allow takes n requests' worth from the key's bucket. When it is empty it
// returns how long to wait.
func (l *keyLimiter) allow(key uuid.UUID, perSecond, n int, now time.Time) (bool, time.Duration) {
	if perSecond <= 0 {
		perSecond = 50
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.buckets == nil {
		l.buckets = map[uuid.UUID]*keyBucket{}
	}
	b := l.buckets[key]
	if b == nil || b.rate != perSecond {
		b = &keyBucket{lim: rate.NewLimiter(rate.Limit(perSecond), perSecond), rate: perSecond}
		l.buckets[key] = b
	}
	b.used = now
	if len(l.buckets) > 10000 {
		for k, v := range l.buckets {
			if now.Sub(v.used) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
	}
	if n > perSecond {
		n = perSecond // a batch larger than a second's worth waits for a full bucket
	}
	r := b.lim.ReserveN(now, n)
	if !r.OK() {
		return false, time.Second
	}
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now)
		return false, d
	}
	return true, 0
}

func retrySeconds(d time.Duration) int { return int(math.Ceil(d.Seconds())) }

func invalidDetail(location, message string) *huma.ErrorDetail {
	return &huma.ErrorDetail{Location: location, Message: message}
}
