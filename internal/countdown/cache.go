package countdown

import (
	"container/list"
	"context"
	"sync"
)

// lru is a bounded map that forgets the least recently used entry first.
type lru[K comparable, V any] struct {
	mu  sync.Mutex
	max int
	ll  *list.List
	m   map[K]*list.Element
}

type lruEntry[K comparable, V any] struct {
	k K
	v V
}

func newLRU[K comparable, V any](max int) *lru[K, V] {
	return &lru[K, V]{max: max, ll: list.New(), m: make(map[K]*list.Element, max)}
}

func (c *lru[K, V]) get(k K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[k]; ok {
		c.ll.MoveToFront(e)
		return e.Value.(*lruEntry[K, V]).v, true
	}
	var zero V
	return zero, false
}

func (c *lru[K, V]) put(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.m[k]; ok {
		e.Value.(*lruEntry[K, V]).v = v
		c.ll.MoveToFront(e)
		return
	}
	c.m[k] = c.ll.PushFront(&lruEntry[K, V]{k: k, v: v})
	if c.ll.Len() > c.max {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.m, last.Value.(*lruEntry[K, V]).k)
	}
}

// flight runs one call per key at a time: callers that ask for a key
// already being computed wait for that result (or their context).
type flight[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]*flightCall[V]
}

type flightCall[V any] struct {
	done chan struct{}
	v    V
	err  error
}

func (g *flight[K, V]) do(ctx context.Context, k K, fn func() (V, error)) (V, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = map[K]*flightCall[V]{}
	}
	if c, ok := g.m[k]; ok {
		g.mu.Unlock()
		select {
		case <-c.done:
			return c.v, c.err
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		}
	}
	c := &flightCall[V]{done: make(chan struct{})}
	g.m[k] = c
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		delete(g.m, k)
		g.mu.Unlock()
		close(c.done)
	}()
	c.v, c.err = fn()
	return c.v, c.err
}
