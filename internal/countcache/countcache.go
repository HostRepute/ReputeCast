// Package countcache keeps the audience counts that read every contact of a
// workspace for a short while (review 2026-11 OP-12): the Audience header's
// counts (about 300 ms on a million contacts) and a segment's count (about
// a second for two conditions).
//
// Only counts that took MinCost or longer are kept: a quick count is
// cheaper to run again than to serve stale, so small workspaces always see
// fresh numbers. An entry lives for TTL. A change this process makes to a workspace's
// contacts, lists or suppressions drops the workspace's entries at once
// (Changed, wired to the audit log, which every such change writes); a
// change another process makes shows within TTL. Concurrent misses for the
// same entry run one count and share its result.
package countcache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultTTL is how long a count is served without being counted again.
const DefaultTTL = time.Minute

// DefaultMax bounds the entries kept, across workspaces.
const DefaultMax = 4096

// SlowCount is the MinCost the app uses: a quarter of a second, about what
// the Audience header's counts take on a million contacts.
const SlowCount = 250 * time.Millisecond

// Cache is safe for concurrent use. The zero value isn't: use New.
type Cache struct {
	// MinCost is the least time a count must have taken to be kept; zero
	// keeps every count. Set it before the cache is used.
	MinCost time.Duration

	ttl time.Duration
	max int
	now func() time.Time

	mu sync.Mutex
	// gen is each workspace's generation: Changed bumps it, so an entry
	// counted before a change is never stored after it.
	gen      map[uuid.UUID]uint64
	entries  map[key]entry
	inflight map[key]*call
}

type key struct {
	ws   uuid.UUID
	name string
}

type entry struct {
	val any
	at  time.Time
	gen uint64
}

type call struct {
	done chan struct{}
	val  any
	at   time.Time
	err  error
}

// New returns a cache; ttl ≤ 0 means DefaultTTL, max ≤ 0 DefaultMax, and a
// nil now the wall clock.
func New(ttl time.Duration, max int, now func() time.Time) *Cache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if max <= 0 {
		max = DefaultMax
	}
	if now == nil {
		now = time.Now
	}
	return &Cache{ttl: ttl, max: max, now: now, gen: map[uuid.UUID]uint64{}, entries: map[key]entry{}, inflight: map[key]*call{}}
}

// Changed drops a workspace's counts: the next read counts again.
func (c *Cache) Changed(ws uuid.UUID) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen[ws]++
	for k := range c.entries {
		if k.ws == ws {
			delete(c.entries, k)
		}
	}
}

// Get returns the workspace's count called name, counting it with load when
// it isn't kept or is older than the TTL. at is when the value was counted.
// A nil cache always loads.
func Get[T any](ctx context.Context, c *Cache, ws uuid.UUID, name string, load func(context.Context) (T, error)) (val T, at time.Time, err error) {
	if c == nil {
		v, err := load(ctx)
		return v, time.Now(), err
	}
	k := key{ws, name}
	c.mu.Lock()
	if e, ok := c.entries[k]; ok && c.now().Sub(e.at) < c.ttl {
		c.mu.Unlock()
		return e.val.(T), e.at, nil
	}
	if cl, ok := c.inflight[k]; ok {
		c.mu.Unlock()
		select {
		case <-cl.done:
		case <-ctx.Done():
			return val, at, ctx.Err()
		}
		if cl.err != nil {
			return val, at, cl.err
		}
		return cl.val.(T), cl.at, nil
	}
	cl := &call{done: make(chan struct{})}
	c.inflight[k] = cl
	gen := c.gen[ws]
	c.mu.Unlock()

	started := c.now()
	v, err := load(ctx)
	c.mu.Lock()
	delete(c.inflight, k)
	cl.val, cl.at, cl.err = v, started, err
	if err == nil && c.gen[ws] == gen && c.now().Sub(started) >= c.MinCost {
		c.store(k, entry{val: v, at: started, gen: gen})
	}
	c.mu.Unlock()
	close(cl.done)
	return v, started, err
}

// Put keeps a value counted elsewhere (a recount the person asked for).
func Put[T any](c *Cache, ws uuid.UUID, name string, val T) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store(key{ws, name}, entry{val: val, at: c.now(), gen: c.gen[ws]})
}

// store adds an entry, making room by dropping expired entries and then
// the oldest. Called with mu held.
func (c *Cache) store(k key, e entry) {
	if _, ok := c.entries[k]; !ok && len(c.entries) >= c.max {
		now := c.now()
		var oldest key
		var oldestAt time.Time
		for ek, ev := range c.entries {
			if now.Sub(ev.at) >= c.ttl {
				delete(c.entries, ek)
				continue
			}
			if oldestAt.IsZero() || ev.at.Before(oldestAt) {
				oldest, oldestAt = ek, ev.at
			}
		}
		if len(c.entries) >= c.max {
			delete(c.entries, oldest)
		}
	}
	c.entries[k] = e
}

// Len is the number of entries kept (tests and metrics).
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// audiencePrefixes are the audit actions that change who is in a
// workspace's audience, its lists, fields, tiers or suppressions.
var audiencePrefixes = []string{"contact", "list", "import.", "suppression", "field.", "audience.", "sunset_policy."}

// AudienceChange reports whether an audit action changes what the counts
// count ("contacts.erase", "list.delete", "import.finish"…).
func AudienceChange(action string) bool {
	for _, p := range audiencePrefixes {
		if strings.HasPrefix(action, p) {
			return true
		}
	}
	return false
}
