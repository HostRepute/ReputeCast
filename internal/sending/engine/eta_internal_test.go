package engine

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// When only a slow route is left (review GS-8), the send's ETA is that
// route's: 1,000 Gmail recipients on Postal at 2 /s take 500 s, about 8
// minutes, though SES sent at 14 /s a moment ago. SES has nothing left, so
// it has no "done about"; Gmail is done when Postal is.
func TestETAOfTheSlowRouteLeft(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 20, 0, 0, time.UTC)
	ses, postal := uuid.New(), uuid.New()
	s := &LiveSnapshot{
		Providers: []LiveProvider{{ID: ses.String(), Short: "SES"}, {ID: postal.String(), Short: "Postal"}},
		Mailboxes: []LiveMailbox{{Key: "gmail"}, {Key: "outlook"}},
	}
	queue := map[queueKey]queueCell{{postal, "gmail"}: {queued: 1000}}
	eta := liveETAs(s, queue, nil, nil, nil, map[uuid.UUID]float64{ses: 14, postal: 2}, nil, now)
	if eta != 500*time.Second {
		t.Fatalf("ETA %v, want 500 s at Postal's 2 /s", eta)
	}
	if s.Providers[0].DoneAt != nil {
		t.Fatalf("SES done at %v, but nothing is left on it", s.Providers[0].DoneAt)
	}
	if d := s.Providers[1].DoneAt; d == nil || !d.Equal(now.Add(500*time.Second)) {
		t.Fatalf("Postal done at %v, want %v", d, now.Add(500*time.Second))
	}
	if d := s.Mailboxes[0].DoneAt; d == nil || !d.Equal(now.Add(500*time.Second)) {
		t.Fatalf("Gmail done at %v, want with Postal", d)
	}
	if s.Mailboxes[1].DoneAt != nil {
		t.Fatal("Outlook has a done time with nothing left")
	}
	// A throttled mailbox provider on the slow route doesn't make it
	// faster: 400 Outlook at its 5 /s throttle among the 1,000 still take
	// Postal's 500 s.
	queue = map[queueKey]queueCell{{postal, "gmail"}: {queued: 600}, {postal, "outlook"}: {queued: 400}}
	if eta := liveETAs(s, queue, nil, nil, map[string]float64{"outlook": 5}, map[uuid.UUID]float64{ses: 14, postal: 2}, nil, now); eta != 500*time.Second {
		t.Fatalf("ETA %v with Outlook throttled, want Postal's 500 s", eta)
	}
}
