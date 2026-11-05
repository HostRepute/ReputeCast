package webhookauth_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

func TestReplayBindsTokenToPayload(t *testing.T) {
	ctx := context.Background()
	r := &webhookauth.Replay{Window: time.Hour}
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	if err := r.Check(ctx, nil, "mailgun", "tok-1", []byte(`{"event":"delivered"}`), now); err != nil {
		t.Fatal(err)
	}
	if err := r.Check(ctx, nil, "mailgun", "tok-1", []byte(`{"event":"delivered"}`), now.Add(time.Minute)); err != nil {
		t.Fatalf("a redelivery of the same event is honored: %v", err)
	}
	if err := r.Check(ctx, nil, "mailgun", "tok-1", []byte(`{"event":"complained"}`), now.Add(2*time.Minute)); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a captured signature can't carry another event: %v", err)
	}
	if err := r.Check(ctx, nil, "mailgun", "tok-2", []byte(`{"event":"complained"}`), now); err != nil {
		t.Fatalf("other tokens are independent: %v", err)
	}
	// After the window the timestamp check rejects the request anyway.
	if err := r.Check(ctx, nil, "mailgun", "tok-1", []byte(`{"event":"complained"}`), now.Add(2*time.Hour)); err != nil {
		t.Fatalf("tokens are forgotten after the window: %v", err)
	}
}

// tokenStore stands for the database every process shares.
type tokenStore struct {
	mu    sync.Mutex
	ties  map[[32]byte]tie
	calls int
	err   error
}

type tie struct {
	payload [32]byte
	until   time.Time
}

func (s *tokenStore) BindToken(_ context.Context, _ string, token, payload [32]byte, now, expires time.Time) ([32]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return [32]byte{}, s.err
	}
	if b, ok := s.ties[token]; ok && !now.After(b.until) {
		return b.payload, nil
	}
	s.ties[token] = tie{payload: payload, until: expires}
	return payload, nil
}

// Every process, and every restart, shares the ties through the store: a
// token one of them saw is refused with another payload by the others, for
// as long as its signature can be accepted.
func TestReplaySharesTiesThroughTheStore(t *testing.T) {
	ctx := context.Background()
	store := &tokenStore{ties: map[[32]byte]tie{}}
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	web, worker := &webhookauth.Replay{Window: 12 * time.Hour}, &webhookauth.Replay{Window: 12 * time.Hour}
	if err := web.Check(ctx, store, "mailgun", "tok-1", []byte(`{"event":"delivered"}`), now); err != nil {
		t.Fatal(err)
	}
	if err := worker.Check(ctx, store, "mailgun", "tok-1", []byte(`{"event":"failed"}`), now.Add(time.Minute)); !errors.Is(err, webhookauth.ErrReplayed) ||
		!errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("another process: %v", err)
	}
	if err := worker.Check(ctx, store, "mailgun", "tok-1", []byte(`{"event":"delivered"}`), now.Add(time.Minute)); err != nil {
		t.Fatalf("a redelivery in another process: %v", err)
	}
	// What a process saw, it answers from memory.
	calls := store.calls
	if err := web.Check(ctx, store, "mailgun", "tok-1", []byte(`{"event":"failed"}`), now.Add(2*time.Minute)); !errors.Is(err, webhookauth.ErrReplayed) || store.calls != calls {
		t.Fatalf("from memory: %v, %d more calls", err, store.calls-calls)
	}
	// A signature is accepted for 12 hours, and signed up to 5 minutes ahead.
	restarted := &webhookauth.Replay{Window: 12 * time.Hour}
	if err := restarted.Check(ctx, store, "mailgun", "tok-1", []byte(`{"event":"failed"}`), now.Add(12*time.Hour+4*time.Minute)); !errors.Is(err, webhookauth.ErrReplayed) {
		t.Fatalf("after a restart, while the signature is accepted: %v", err)
	}
	// The same token of another adapter is another token.
	if err := restarted.Check(ctx, store, "other", "tok-1", []byte(`{"event":"failed"}`), now.Add(3*time.Minute)); err != nil {
		t.Fatalf("another scope: %v", err)
	}
	// A store that can't be reached asks the provider to deliver again:
	// nothing goes through untied.
	store.err = errors.New("database gone")
	err := restarted.Check(ctx, store, "mailgun", "tok-2", []byte(`{"event":"delivered"}`), now)
	if !errors.Is(err, providers.ErrUnavailable) || errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("store unreachable: %v", err)
	}
}

// Without a store, a full memory refuses new tokens rather than letting
// them through untied, until the old ones are past their time.
func TestReplayWithoutStoreRefusesWhenFull(t *testing.T) {
	ctx := context.Background()
	r := &webhookauth.Replay{Window: time.Hour}
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	var err error
	for i := 0; err == nil && i < 1_000_000; i++ {
		err = r.Check(ctx, nil, "mailgun", strconv.Itoa(i), []byte(`{}`), now)
	}
	if !errors.Is(err, providers.ErrUnavailable) || errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a full memory: %v", err)
	}
	if err := r.Check(ctx, nil, "mailgun", "later", []byte(`{}`), now.Add(2*time.Hour)); err != nil {
		t.Fatalf("once the old tokens are past their time: %v", err)
	}
}
