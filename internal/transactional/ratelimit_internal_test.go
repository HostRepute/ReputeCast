package transactional

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// TestTakeTokensSharedAcrossNodes: two web nodes take from one bucket per
// key, so together they send at the key's rate, not twice it (review
// 2026-11 TX-4).
func TestTakeTokensSharedAcrossNodes(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	a, b := &Service{Pool: pool, Q: store.New(pool)}, &Service{Pool: pool, Q: store.New(pool)}
	key, now := uuid.New(), time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	take := func(s *Service, n int, at time.Time) (bool, time.Duration) {
		t.Helper()
		ok, wait, err := s.takeTokens(ctx, key, 5, n, at)
		if err != nil {
			t.Fatal(err)
		}
		return ok, wait
	}
	if ok, _ := take(a, 3, now); !ok {
		t.Fatal("node A takes 3 of 5")
	}
	if ok, _ := take(b, 2, now); !ok {
		t.Fatal("node B takes the other 2")
	}
	ok, wait := take(b, 1, now)
	if ok || wait != 200*time.Millisecond {
		t.Fatalf("the bucket is empty for both nodes: ok %v, wait %s", ok, wait)
	}
	if ok, _ := take(a, 1, now); ok {
		t.Fatal("node A sees the same empty bucket")
	}
	// A second later it is full again, never more than a second's worth.
	if ok, _ := take(a, 5, now.Add(10*time.Second)); !ok {
		t.Fatal("refilled")
	}
	if ok, _ := take(b, 1, now.Add(10*time.Second)); ok {
		t.Fatal("no more than the rate")
	}
	// A node whose clock is behind adds nothing.
	if ok, _ := take(b, 1, now.Add(9*time.Second)); ok {
		t.Fatal("a clock behind doesn't refill")
	}
	// A batch larger than the rate waits for a full bucket.
	if ok, _ := take(a, 50, now.Add(20*time.Second)); !ok {
		t.Fatal("a big batch takes a full bucket")
	}
	// Idle buckets go after an hour.
	if n, err := a.Q.TxnRateBucketsPurge(ctx, now.Add(2*time.Hour)); err != nil || n != 1 {
		t.Fatalf("purge: %d %v", n, err)
	}
}

// TestTakeTokensConcurrently: senders that take at the same moment get
// exactly the bucket's tokens between them, neither more nor fewer, with
// each take a single statement (review 2026-11 OP-13).
func TestTakeTokensConcurrently(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	s := &Service{Pool: pool, Q: store.New(pool)}
	key, now := uuid.New(), time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	const rate, senders = 50, 200
	var mu sync.Mutex
	took, refused := 0, 0
	var wg sync.WaitGroup
	for range senders {
		wg.Go(func() {
			ok, wait, err := s.takeTokens(ctx, key, rate, 1, now)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				t.Error(err)
			case ok:
				took++
			case wait != time.Second/rate:
				t.Errorf("an empty bucket waits a fiftieth of a second, not %s", wait)
			default:
				refused++
			}
		})
	}
	wg.Wait()
	if took != rate || refused != senders-rate {
		t.Fatalf("%d took and %d were refused; want %d and %d", took, refused, rate, senders-rate)
	}
	// Half a second later half the bucket is back.
	n := 0
	for range rate {
		if ok, _, _ := s.takeTokens(ctx, key, rate, 1, now.Add(500*time.Millisecond)); ok {
			n++
		}
	}
	if n != rate/2 {
		t.Fatalf("half a second refilled %d tokens, want %d", n, rate/2)
	}
}
