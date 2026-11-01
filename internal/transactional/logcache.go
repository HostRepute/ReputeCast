package transactional

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// countsTTL is how long the log's status counts are reused: the live poll
// asks every 10 seconds for newer messages only, and the numbers above the
// log move once a minute (review 2026-11 TX-3).
const countsTTL = time.Minute

// countsCache keeps the log's status counts per workspace and filter set.
type countsCache struct {
	mu      sync.Mutex
	entries map[string]countsEntry
}

type countsEntry struct {
	counts TransactionalLogCounts
	at     time.Time
}

// countsKey is a filter set's key; the status filter doesn't change the
// counts, which are per status.
func countsKey(ws uuid.UUID, f LogFilter) string {
	return strings.Join([]string{ws.String(), f.Range, f.From, f.To, f.Template, f.Provider, f.Key, strings.ToLower(strings.TrimSpace(f.Query))}, "\x00")
}

func (c *countsCache) get(key string, now time.Time) (TransactionalLogCounts, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || now.Sub(e.at) >= countsTTL {
		return TransactionalLogCounts{}, false
	}
	return e.counts, true
}

func (c *countsCache) put(key string, counts TransactionalLogCounts, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]countsEntry{}
	}
	if len(c.entries) >= 5000 {
		// Expired entries go first; a cache this full of fresh ones starts over.
		for k, e := range c.entries {
			if now.Sub(e.at) >= countsTTL {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= 5000 {
			c.entries = map[string]countsEntry{}
		}
	}
	c.entries[key] = countsEntry{counts: counts, at: now}
}
