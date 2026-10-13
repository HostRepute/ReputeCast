package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// hub computes one campaign's live snapshot at most once a second while
// anyone watches, and fans it out to every watcher in this process.
type hub struct {
	ws, campaign uuid.UUID
	mu           sync.Mutex
	subs         map[chan LiveSnapshot]struct{}
	last         *LiveSnapshot
	lastBody     []byte
	idleSince    time.Time
}

// pollEvery is how often a watched campaign's snapshot is recomputed.
const pollEvery = time.Second

// Subscribe streams a campaign's snapshots: the current one at once, then
// every change. cancel stops the stream.
func (e *Engine) Subscribe(ctx context.Context, ws, campaign uuid.UUID) (<-chan LiveSnapshot, func()) {
	ch := make(chan LiveSnapshot, 4)
	e.mu.Lock()
	if e.hubs == nil {
		e.hubs = map[uuid.UUID]*hub{}
	}
	h := e.hubs[campaign]
	start := false
	if h == nil {
		h = &hub{ws: ws, campaign: campaign, subs: map[chan LiveSnapshot]struct{}{}}
		e.hubs[campaign] = h
		start = true
	}
	e.mu.Unlock()
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	if h.last != nil {
		select {
		case ch <- *h.last:
		default:
		}
	}
	h.mu.Unlock()
	if start {
		go e.runHub(h)
	}
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs, ch)
			if len(h.subs) == 0 {
				h.idleSince = time.Now()
			}
			h.mu.Unlock()
		})
	}
}

// runHub polls while the hub has watchers, then retires after a while
// without any.
func (e *Engine) runHub(h *hub) {
	t := time.NewTicker(pollEvery)
	defer t.Stop()
	e.pollHub(h)
	for range t.C {
		h.mu.Lock()
		idle := len(h.subs) == 0 && !h.idleSince.IsZero() && time.Since(h.idleSince) > 30*time.Second
		h.mu.Unlock()
		if idle {
			e.mu.Lock()
			h.mu.Lock()
			if len(h.subs) == 0 {
				delete(e.hubs, h.campaign)
				h.mu.Unlock()
				e.mu.Unlock()
				return
			}
			h.mu.Unlock()
			e.mu.Unlock()
		}
		e.pollHub(h)
	}
}

func (e *Engine) pollHub(h *hub) {
	h.mu.Lock()
	watching := len(h.subs) > 0
	h.mu.Unlock()
	if !watching && h.last != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := e.Snapshot(ctx, h.ws, h.campaign)
	if err != nil {
		e.log().Warn("live snapshot", "campaign", h.campaign, "err", err)
		return
	}
	body := comparable(s)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.last != nil && bytes.Equal(body, h.lastBody) && time.Since(h.last.At) < 15*time.Second {
		return
	}
	h.last, h.lastBody = &s, body
	for ch := range h.subs {
		select {
		case ch <- s:
		default:
			// A slow watcher misses an intermediate snapshot; the next one
			// is complete anyway.
		}
	}
}

// comparable is a snapshot without its timestamps, to detect changes.
func comparable(s LiveSnapshot) []byte {
	s.At, s.Version = time.Time{}, 0
	s.ETASeconds, s.FinishesAt = nil, nil
	b, _ := json.Marshal(s)
	return b
}
