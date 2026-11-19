package assistant

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// A group of 400 threads notifies in payloads under Postgres's 8,000
// bytes, which the hub reads back as every thread.
func TestDeltaGroupNotifications(t *testing.T) {
	var last []store.AIAssistantDeltaSeqsRow
	var want []uuid.UUID
	for i := range 400 {
		id := uuid.Must(uuid.NewV7())
		last = append(last, store.AIAssistantDeltaSeqsRow{ID: id, EventSeq: int64(1000000 + i)})
		want = append(want, id)
	}
	payloads := notifications(slices.Clone(last))
	var got []uuid.UUID
	for _, p := range payloads {
		if len(p) > notifyMax {
			t.Fatalf("a payload of %d bytes", len(p))
		}
		got = append(got, notified(p)...)
	}
	if len(payloads) < 3 || len(got) != len(want) {
		t.Fatalf("%d payloads naming %d threads", len(payloads), len(got))
	}
	for _, id := range want {
		if !slices.Contains(got, id) {
			t.Fatalf("%s not notified", id)
		}
	}
	if one := notified(want[0].String() + ":7"); !slices.Equal(one, want[:1]) {
		t.Fatalf("a single event's notification: %v", one)
	}
}
