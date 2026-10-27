package campaigns

import (
	"testing"
	"time"
)

// CO-10: review links are looked up at most 60 times a minute per client
// address, and 20 unknown links in 10 minutes block the address for the
// rest of those 10 minutes, so guessing tokens is pointless.
func TestReviewLimiter(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	const a, b = "203.0.113.7", "203.0.113.8"
	for i := range reviewRequestsPerMinute {
		if !reviewAllow(a, now) {
			t.Fatalf("request %d refused", i+1)
		}
	}
	if reviewAllow(a, now) {
		t.Fatal("request 61 in a minute allowed")
	}
	if !reviewAllow(b, now) {
		t.Fatal("another address is limited on its own")
	}
	if !reviewAllow(a, now.Add(time.Minute)) {
		t.Fatal("the next minute starts over")
	}

	for range reviewMissesPer10Min {
		reviewMiss(b, now)
	}
	if reviewAllow(b, now.Add(5*time.Minute)) {
		t.Fatal("20 unknown links should block the address")
	}
	if !reviewAllow(b, now.Add(10*time.Minute)) {
		t.Fatal("the block lasts 10 minutes")
	}
}
