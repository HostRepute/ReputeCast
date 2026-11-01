package transactional

import (
	"context"
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
