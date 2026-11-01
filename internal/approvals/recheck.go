package approvals

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// recheckWorkers is how many open requests a batch re-check checks at once.
const recheckWorkers = 4

// recheck re-checks the open requests of these campaigns, a few at a time,
// sharing what they have in common: campaigns that send to the same audience
// or route count it once (review 2026-11 AP-3). Each check stands alone, so
// one that fails leaves the others; failures are returned in no set order.
func (s *Service) recheck(ctx context.Context, ws *store.Workspace, campaigns []uuid.UUID) []error {
	ctx = withCheckMemo(ctx)
	sem := make(chan struct{}, recheckWorkers)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, id := range campaigns {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := s.Revalidate(ctx, ws, id); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return errs
}

// checkMemo keeps a batch re-check's audience counts and routes, so requests
// that share them compute them once. It lives as long as one batch.
type checkMemo struct {
	mu       sync.Mutex
	audience map[string]*memoEntry[readiness.AudienceFacts]
	route    map[string]*memoEntry[readiness.RouteConfig]
}

type memoEntry[T any] struct {
	once sync.Once
	v    T
	err  error
}

type checkMemoKey struct{}

func withCheckMemo(ctx context.Context) context.Context {
	if _, ok := ctx.Value(checkMemoKey{}).(*checkMemo); ok {
		return ctx
	}
	return context.WithValue(ctx, checkMemoKey{}, &checkMemo{
		audience: map[string]*memoEntry[readiness.AudienceFacts]{},
		route:    map[string]*memoEntry[readiness.RouteConfig]{},
	})
}

// memoized returns the entry for key, made on first use; the caller fills it once.
func memoized[T any](m *checkMemo, entries map[string]*memoEntry[T], key string) *memoEntry[T] {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := entries[key]
	if e == nil {
		e = &memoEntry[T]{}
		entries[key] = e
	}
	return e
}

// audienceFacts counts a stored audience, once per batch re-check.
func (s *Service) audienceFacts(ctx context.Context, ws *store.Workspace, raw json.RawMessage) (readiness.AudienceFacts, error) {
	m, ok := ctx.Value(checkMemoKey{}).(*checkMemo)
	if !ok {
		return s.Readiness.Audience(ctx, ws, raw)
	}
	e := memoized(m, m.audience, ws.ID.String()+"\x00"+string(raw))
	e.once.Do(func() { e.v, e.err = s.Readiness.Audience(ctx, ws, raw) })
	return e.v, e.err
}

// routeConfig reads a campaign's routing, once per batch re-check.
func (s *Service) routeConfig(ctx context.Context, ws *store.Workspace, override json.RawMessage) (readiness.RouteConfig, error) {
	m, ok := ctx.Value(checkMemoKey{}).(*checkMemo)
	if !ok {
		return s.Readiness.Route(ctx, ws, override)
	}
	e := memoized(m, m.route, ws.ID.String()+"\x00"+string(override))
	e.once.Do(func() { e.v, e.err = s.Readiness.Route(ctx, ws, override) })
	return e.v, e.err
}
