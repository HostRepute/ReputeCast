package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/riverqueue/river"
)

func TestInterruptedJobsSnoozeAndFailedOnesRetrySoon(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var snooze *river.JobSnoozeError
	if err := interrupted(ctx, fmt.Errorf("claim: %w", context.Canceled)); !errors.As(err, &snooze) || snooze.Duration != 0 {
		t.Fatalf("a job cut short by shutdown must snooze to run again at once, got %v", err)
	}
	boom := errors.New("boom")
	if err := interrupted(context.Background(), boom); !errors.Is(err, boom) {
		t.Fatalf("a real failure stays a failure, got %v", err)
	}
	if err := interrupted(ctx, nil); err != nil {
		t.Fatalf("nothing to report, got %v", err)
	}
	for attempt, most := range map[int]time.Duration{1: 2 * time.Second, 3: 6 * time.Second, 40: 30 * time.Second} {
		if d := time.Until(quickRetry(attempt)); d <= 0 || d > most+time.Second {
			t.Errorf("attempt %d retries in %v, want about %v", attempt, d, most)
		}
	}
}
