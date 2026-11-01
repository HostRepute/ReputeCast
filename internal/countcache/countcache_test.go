package countcache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/countcache"
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestKeptForTheTTLAndDroppedOnChange(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}
	c := countcache.New(time.Minute, 0, clk.now)
	ws, other := uuid.New(), uuid.New()
	loads := 0
	load := func(context.Context) (int, error) { loads++; return loads, nil }
	get := func(w uuid.UUID) int {
		t.Helper()
		v, _, err := countcache.Get(ctx, c, w, "n", load)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if get(ws) != 1 || get(ws) != 1 {
		t.Fatal("a second read within the TTL counts again")
	}
	clk.add(59 * time.Second)
	if get(ws) != 1 {
		t.Fatal("kept for under a minute")
	}
	clk.add(time.Second)
	if get(ws) != 2 {
		t.Fatal("not counted again after the TTL")
	}
	if get(other) != 3 {
		t.Fatal("workspaces share a count")
	}
	c.Changed(ws)
	if get(ws) != 4 || get(other) != 3 {
		t.Fatal("a change drops its own workspace's counts only")
	}
}

func TestOneCountForConcurrentReads(t *testing.T) {
	ctx := context.Background()
	c := countcache.New(0, 0, nil)
	ws := uuid.New()
	var loads atomic.Int32
	release := make(chan struct{})
	load := func(context.Context) (int, error) {
		loads.Add(1)
		<-release
		return 42, nil
	}
	var wg sync.WaitGroup
	got := make([]int, 20)
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[i], _, _ = countcache.Get(ctx, c, ws, "n", load)
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if loads.Load() != 1 {
		t.Fatalf("%d counts for concurrent reads", loads.Load())
	}
	for _, v := range got {
		if v != 42 {
			t.Fatalf("a reader got %d", v)
		}
	}
}

func TestAChangeDuringACountIsntHidden(t *testing.T) {
	ctx := context.Background()
	c := countcache.New(0, 0, nil)
	ws := uuid.New()
	n := 0
	_, _, _ = countcache.Get(ctx, c, ws, "n", func(context.Context) (int, error) {
		n++
		c.Changed(ws) // a contact changed while the count ran
		return n, nil
	})
	v, _, _ := countcache.Get(ctx, c, ws, "n", func(context.Context) (int, error) { n++; return n, nil })
	if v != 2 {
		t.Fatal("a count that began before a change was kept")
	}
}

func TestErrorsAreNotKeptAndRoomIsBounded(t *testing.T) {
	ctx := context.Background()
	c := countcache.New(0, 3, nil)
	ws := uuid.New()
	boom := errors.New("timeout")
	if _, _, err := countcache.Get(ctx, c, ws, "n", func(context.Context) (int, error) { return 0, boom }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if c.Len() != 0 {
		t.Fatal("a failed count was kept")
	}
	for _, k := range []string{"a", "b", "c", "d", "e"} {
		_, _, _ = countcache.Get(ctx, c, ws, k, func(context.Context) (int, error) { return 1, nil })
	}
	if c.Len() != 3 {
		t.Fatalf("kept %d entries, bound 3", c.Len())
	}
	var nilCache *countcache.Cache
	if v, _, err := countcache.Get(ctx, nilCache, ws, "n", func(context.Context) (int, error) { return 7, nil }); v != 7 || err != nil {
		t.Fatal("a nil cache loads")
	}
	nilCache.Changed(ws)
}

func TestAudienceChange(t *testing.T) {
	for _, a := range []string{"contact.create", "contacts.erase", "list.delete", "import.finish", "suppression.add", "field.delete", "audience.tiers_recompute"} {
		if !countcache.AudienceChange(a) {
			t.Errorf("%s changes the audience", a)
		}
	}
	for _, a := range []string{"campaign.rescheduled", "webhook.created", "segment.update"} {
		if countcache.AudienceChange(a) {
			t.Errorf("%s doesn't change the audience", a)
		}
	}
}

func TestQuickCountsAreNotKept(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}
	c := countcache.New(time.Minute, 0, clk.now)
	c.MinCost = 250 * time.Millisecond
	ws := uuid.New()
	loads := 0
	quick := func(context.Context) (int, error) { loads++; return loads, nil }
	slow := func(context.Context) (int, error) { loads++; clk.add(300 * time.Millisecond); return loads, nil }
	_, _, _ = countcache.Get(ctx, c, ws, "quick", quick)
	if v, _, _ := countcache.Get(ctx, c, ws, "quick", quick); v != 2 {
		t.Fatal("a quick count was kept")
	}
	_, _, _ = countcache.Get(ctx, c, ws, "slow", slow)
	if v, _, _ := countcache.Get(ctx, c, ws, "slow", slow); v != 3 {
		t.Fatalf("a slow count wasn't kept: %d", v)
	}
}
