package webhookauth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

// maxTokens bounds the memory a Replay uses (about 64 bytes a token).
const maxTokens = 250_000

// ErrReplayed is a signed token used again with another payload: a
// captured signature carrying a forged event. It wraps
// providers.ErrBadSignature.
var ErrReplayed = fmt.Errorf("%w: the signature was already used with another payload", providers.ErrBadSignature)

// Replay ties each signed one-time token to the payload it first came with,
// for schemes whose signature covers a token but not the payload
// (Mailgun's): a token is honored again with the same payload (a
// redelivery, which ingest's dedupe drops) but never with another, so a
// captured signature can't carry a forged event. Call Check only once the
// signature verified, so only the provider's own tokens are kept.
//
// The ties are kept in a providers.TokenStore (ReputeCast's database), where
// every process and every restart sees them, for as long as a signature
// with the token can be accepted. Memory only remembers what this process
// saw, to answer again without asking the store. Without a store (unit
// tests) memory is all there is, and once it is full new tokens are
// refused rather than let through untied.
type Replay struct {
	// Window is how old a signed timestamp may be (Within).
	Window time.Duration

	mu    sync.Mutex
	seen  map[[16]byte]seenToken
	sweep time.Time
}

type seenToken struct {
	payload [16]byte
	until   time.Time
}

// Check ties token to payload within scope (the adapter), or fails with
// ErrReplayed when the token is tied to another payload. The tie lasts as
// long as a signature with the token can be accepted: Within's window,
// signed up to Tolerance ahead. A store that can't be reached fails with an
// error wrapping providers.ErrUnavailable: the provider is asked to deliver
// again, rather than the event stored unchecked.
func (r *Replay) Check(ctx context.Context, store providers.TokenStore, scope, token string, payload []byte, now time.Time) error {
	th, ph := sha256.Sum256([]byte(scope+"\x00"+token)), sha256.Sum256(payload)
	t, p := short(th), short(ph)
	until := now.Add(max(r.Window, Tolerance) + Tolerance)
	r.mu.Lock()
	if bound, ok := r.recall(t, now); ok {
		r.mu.Unlock()
		return tiedTo(bound, p)
	}
	if store == nil {
		defer r.mu.Unlock()
		if !r.keep(t, p, until) {
			r.forget(now)
			if !r.keep(t, p, until) {
				return fmt.Errorf("%w: too many webhook tokens to remember", providers.ErrUnavailable)
			}
		}
		return nil
	}
	r.mu.Unlock()
	bound, err := store.BindToken(ctx, scope, th, ph, now, until)
	if err != nil {
		return fmt.Errorf("%w: webhook tokens: %w", providers.ErrUnavailable, err)
	}
	r.mu.Lock()
	// With memory full, the store answers the next time too.
	r.keep(t, short(bound), until)
	r.mu.Unlock()
	return tiedTo(short(bound), p)
}

func short(h [32]byte) [16]byte {
	var s [16]byte
	copy(s[:], h[:])
	return s
}

func tiedTo(bound, payload [16]byte) error {
	if bound != payload {
		return ErrReplayed
	}
	return nil
}

// recall returns the payload a token is tied to in memory, forgetting the
// tokens past their time every five minutes. The caller holds r.mu.
func (r *Replay) recall(t [16]byte, now time.Time) ([16]byte, bool) {
	if r.seen == nil {
		r.seen = map[[16]byte]seenToken{}
	}
	if now.After(r.sweep) {
		r.forget(now)
	}
	s, ok := r.seen[t]
	if !ok || now.After(s.until) {
		return [16]byte{}, false
	}
	return s.payload, true
}

// keep remembers the payload a token is tied to, unless memory is full.
// The caller holds r.mu.
func (r *Replay) keep(t, payload [16]byte, until time.Time) bool {
	if len(r.seen) >= maxTokens {
		return false
	}
	r.seen[t] = seenToken{payload: payload, until: until}
	return true
}

// forget drops the tokens past their time. The caller holds r.mu.
func (r *Replay) forget(now time.Time) {
	for k, v := range r.seen {
		if now.After(v.until) {
			delete(r.seen, k)
		}
	}
	r.sweep = now.Add(5 * time.Minute)
}
