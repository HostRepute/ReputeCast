package webhookauth

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

// maxTokens bounds the memory a Replay uses (about 64 bytes a token).
const maxTokens = 250_000

// Replay remembers, for Window, which payload each signed one-time token
// came with, for schemes whose signature covers a token but not the
// payload (Mailgun's): a token is honored again with the same payload (a
// redelivery, which ingest's dedupe drops) but never with another, so a
// captured signature can't carry a forged event. Call Check only once the
// signature verified, so only the provider's own tokens are remembered.
//
// It lives in one process and remembers at most maxTokens tokens; the
// timestamp window still bounds what it doesn't cover.
type Replay struct {
	Window time.Duration

	mu    sync.Mutex
	seen  map[[16]byte]seenToken
	sweep time.Time
}

type seenToken struct {
	payload [16]byte
	until   time.Time
}

// Check records token with payload, or fails with an error wrapping
// providers.ErrBadSignature when the token was seen with another payload.
func (r *Replay) Check(token string, payload []byte, now time.Time) error {
	ts, ps := sha256.Sum256([]byte(token)), sha256.Sum256(payload)
	var t, p [16]byte
	copy(t[:], ts[:])
	copy(p[:], ps[:])
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen == nil {
		r.seen = map[[16]byte]seenToken{}
	}
	if now.After(r.sweep) {
		for k, v := range r.seen {
			if now.After(v.until) {
				delete(r.seen, k)
			}
		}
		r.sweep = now.Add(5 * time.Minute)
	}
	if s, ok := r.seen[t]; ok && !now.After(s.until) {
		if s.payload != p {
			return fmt.Errorf("%w: the signature was already used with another payload", providers.ErrBadSignature)
		}
		return nil
	}
	if len(r.seen) < maxTokens {
		r.seen[t] = seenToken{payload: p, until: now.Add(r.Window)}
	}
	return nil
}
