package webhookauth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

func TestReplayBindsTokenToPayload(t *testing.T) {
	r := &webhookauth.Replay{Window: time.Hour}
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	if err := r.Check("tok-1", []byte(`{"event":"delivered"}`), now); err != nil {
		t.Fatal(err)
	}
	if err := r.Check("tok-1", []byte(`{"event":"delivered"}`), now.Add(time.Minute)); err != nil {
		t.Fatalf("a redelivery of the same event is honored: %v", err)
	}
	if err := r.Check("tok-1", []byte(`{"event":"complained"}`), now.Add(2*time.Minute)); !errors.Is(err, providers.ErrBadSignature) {
		t.Fatalf("a captured signature can't carry another event: %v", err)
	}
	if err := r.Check("tok-2", []byte(`{"event":"complained"}`), now); err != nil {
		t.Fatalf("other tokens are independent: %v", err)
	}
	// After the window the timestamp check rejects the request anyway.
	if err := r.Check("tok-1", []byte(`{"event":"complained"}`), now.Add(2*time.Hour)); err != nil {
		t.Fatalf("tokens are forgotten after the window: %v", err)
	}
}
