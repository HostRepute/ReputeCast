// Package tracker is the site tracker (M62 G-28): a small script,
// /track/v1/rc.js, that a publisher adds to their own sites and stores
// (never ReputeCast's pages, which stay cookie-less), the endpoint its
// events go to, who a visitor is (only ever from a signed click token, a
// form signup's signed token or a signed identify call), the sites'
// settings, the contact timeline's lines, two segment conditions' data and
// the retention.
//
//	GET  /track/v1/rc.js                 the script, for every site
//	POST /p/{ws}/track/{site}/e          a site's events (and OPTIONS)
//
// Events go through a group-commit writer (writer.go): a request is
// answered as soon as its events are queued, and a batch of them is
// written every 200 ms or 500 events in one transaction.
package tracker

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/customhost"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// LinkKind is the token a form signup hands to the tracker on the same
// page (reputecast('link', …)): its contact, valid ten minutes
// (linktoken.UntilMinute in Link).
const LinkKind linktoken.Kind = 232

// LinkValid is how long a form signup's link token works.
const LinkValid = 10 * time.Minute

// Event kinds (tracker_events.kind).
const (
	KindPage    int16 = 1
	KindProduct int16 = 2
	KindCart    int16 = 3
	KindCustom  int16 = 4
)

// Who a visitor was linked by (tracker_visitors.identified_by).
const (
	ByClick    = "click"
	ByForm     = "form"
	ByIdentify = "identify"
)

// Retention.
const (
	// EventDays is how long events are kept (or the workspace's event
	// detail retention when it is 30 days).
	EventDays = 90
	// BackfillDays is how much of a visitor's past joins the contact they
	// are linked to.
	BackfillDays = 30
	// AnonymousDays: a visitor never linked to a contact goes this long
	// after they were last seen, with their events.
	AnonymousDays = 30
	// InterestDays: a contact's interest in a product goes this long after
	// the last view or add-to-cart.
	InterestDays = 180
	// RefusalDays: refused origins are counted this long.
	RefusalDays = 7
)

// Service is the site tracker.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Links *linktoken.Signer
	URLs  publicurl.Builder
	Log   *slog.Logger
	Now   func() time.Time
	// Hosts are the hosts workspaces hold (custom domains, the publication
	// website): ReputeCast's own pages, which are never tracked.
	Hosts *customhost.Hosts
	// Carries reports whether the host a request came to may carry the
	// workspace's endpoints (customhost.Hosts.CarriesSlug); nil carries
	// every workspace's.
	Carries func(r *http.Request, slug string) bool
	// Development allows http://localhost origins.
	Development func() bool
	// Writers is how many batches are written at once (default 2).
	Writers int

	once     sync.Once
	w        *writer
	sites    cache[collectSite]
	products cache[productHit]
	origins  cache[[]string]
	limits   limiter
	refused  refusals
	// refusing serializes the stores of refused origins.
	refusing sync.Mutex
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func (s *Service) development() bool { return s.Development != nil && s.Development() }

// cache is a small map of values that expire, bounded in size.
type cache[V any] struct {
	mu sync.Mutex
	m  map[string]cached[V]
}

type cached[V any] struct {
	v     V
	until time.Time
}

// cacheMax bounds a cache; past it the cache starts over.
const cacheMax = 50_000

func (c *cache[V]) get(key string, now time.Time) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || now.After(e.until) {
		var zero V
		return zero, false
	}
	return e.v, true
}

func (c *cache[V]) put(key string, v V, until time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil || len(c.m) >= cacheMax {
		c.m = map[string]cached[V]{}
	}
	c.m[key] = cached[V]{v: v, until: until}
}

func (c *cache[V]) drop(match func(key string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		if match(k) {
			delete(c.m, k)
		}
	}
}

// Close waits until the events queued so far are written (tests, and a
// process that stops).
func (s *Service) Close(ctx context.Context) error {
	return s.start().flush(ctx)
}
