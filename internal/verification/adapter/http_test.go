package adapter_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// A failure stops the addresses after it, never the check of one before it
// that is still waiting for its answer: the service answers (and charges
// for) that check whatever order the parallel checks reach it in, and a
// job keeps the answers before the failure.
func TestOneByOneKeepsChecksBeforeAFailure(t *testing.T) {
	failed := make(chan struct{})
	var calls atomic.Int32
	answers, err := adapter.OneByOne(context.Background(), []string{"a@x.test", "b@x.test", "c@x.test"}, 2,
		func(ctx context.Context, email string) (adapter.Answer, error) {
			calls.Add(1)
			switch email {
			case "a@x.test":
				// The answer comes after b's check failed.
				<-failed
				select {
				case <-ctx.Done():
					return adapter.Answer{}, ctx.Err()
				case <-time.After(50 * time.Millisecond):
					return adapter.Answer{Email: email, Result: adapter.Valid}, nil
				}
			case "b@x.test":
				close(failed)
				return adapter.Answer{}, &adapter.CallError{Kind: adapter.ErrNoCredits, Message: "Insufficient balance"}
			}
			return adapter.Answer{Email: email, Result: adapter.Valid}, nil
		})
	if !errors.Is(err, adapter.ErrNoCredits) {
		t.Fatalf("error %v, want the service's", err)
	}
	if len(answers) != 1 || answers[0].Email != "a@x.test" {
		t.Fatalf("answers %+v, want a's", answers)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("%d checks, want 2: c comes after the failure", n)
	}
}
