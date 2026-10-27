package counters

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func at(min int) *time.Time {
	t := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC).Add(time.Duration(min) * time.Minute)
	return &t
}

func TestFlags(t *testing.T) {
	tests := []struct {
		name string
		s    State
		want Flags
	}{
		{"queued", State{Status: "queued"}, Flags{}},
		{"sent", State{Status: "sent", SentAt: at(0)}, Flags{Sent: true}},
		{"delivered", State{Status: "delivered", SentAt: at(0), DeliveredAt: at(1)}, Flags{Sent: true, Delivered: true}},
		{"deferred, then delivered", State{Status: "delivered", SentAt: at(0), DeferredAt: at(1), DeliveredAt: at(6)},
			Flags{Sent: true, Delivered: true, Deferred: true}},
		{"soft bounce waiting", State{Status: "deferred", SentAt: at(0), SoftBouncedAt: at(2)}, Flags{Sent: true, SoftBounced: true}},
		{"soft bounce, then delivered", State{Status: "delivered", SentAt: at(0), SoftBouncedAt: at(2), DeliveredAt: at(30)},
			Flags{Sent: true, Delivered: true}},
		{"soft bounce, then hard", State{Status: "bounced", SentAt: at(0), SoftBouncedAt: at(2)}, Flags{Sent: true, HardBounced: true}},
		{"delivered, then hard bounce", State{Status: "bounced", SentAt: at(0), DeliveredAt: at(1)}, Flags{Sent: true, HardBounced: true}},
		{"complaint proves delivery", State{Status: "complained", SentAt: at(0)}, Flags{Sent: true, Delivered: true, Complained: true}},
		{"rejected when handed over", State{Status: "rejected"}, Flags{Rejected: true}},
		{"rejected afterwards", State{Status: "rejected", SentAt: at(0)}, Flags{Rejected: true}},
		{"machine open", State{Status: "delivered", SentAt: at(0), DeliveredAt: at(1), OpenedAt: at(1)},
			Flags{Sent: true, Delivered: true, OpenedMachine: true}},
		{"privacy prefetch", State{Status: "delivered", SentAt: at(0), DeliveredAt: at(1), OpenedAt: at(1), PrivacyOpenedAt: at(1)},
			Flags{Sent: true, Delivered: true, OpenedPrivacy: true}},
		{"prefetch, then a person", State{Status: "delivered", SentAt: at(0), DeliveredAt: at(1), OpenedAt: at(1), PrivacyOpenedAt: at(1), HumanOpenedAt: at(40)},
			Flags{Sent: true, Delivered: true, OpenedHuman: true}},
		{"click", State{Status: "delivered", SentAt: at(0), DeliveredAt: at(1), OpenedAt: at(9), HumanOpenedAt: at(9), ClickedAt: at(9)},
			Flags{Sent: true, Delivered: true, OpenedHuman: true, Clicked: true}},
		{"unsubscribed", State{Status: "delivered", SentAt: at(0), DeliveredAt: at(1), UnsubscribedAt: at(50)},
			Flags{Sent: true, Delivered: true, Unsubscribed: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.Flags(); got != tc.want {
				t.Fatalf("flags = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestDiffOnlyOnTransitions(t *testing.T) {
	sent := State{Status: "sent", SentAt: at(0)}
	soft := State{Status: "deferred", SentAt: at(0), SoftBouncedAt: at(2)}
	delivered := State{Status: "delivered", SentAt: at(0), SoftBouncedAt: at(2), DeliveredAt: at(30)}
	if d := Diff(sent, soft); d != (Delta{SoftBounced: 1}) {
		t.Fatalf("soft bounce: %+v", d)
	}
	if d := Diff(soft, delivered); d != (Delta{SoftBounced: -1, Delivered: 1}) {
		t.Fatalf("delivered after a soft bounce: %+v", d)
	}
	// The same delivery reported again changes nothing.
	if d := Diff(delivered, delivered); !d.Zero() {
		t.Fatalf("a repeated delivery counted: %+v", d)
	}
	// A late rejection takes the message back out of sent.
	if d := Diff(sent, State{Status: "rejected", SentAt: at(0)}); d != (Delta{Sent: -1, Rejected: 1}) {
		t.Fatalf("late rejection: %+v", d)
	}
	// A second deferral of a deferred message counts nothing.
	def := State{Status: "deferred", SentAt: at(0), DeferredAt: at(1)}
	if d := Diff(def, def); !d.Zero() {
		t.Fatalf("second deferral counted: %+v", d)
	}
}

func TestLedgerMovesHours(t *testing.T) {
	start := at(0)
	cell := Cell{Campaign: uuid.Must(uuid.NewV7()), Provider: uuid.Must(uuid.NewV7()), Mailbox: "icloud"}
	base := State{Status: "delivered", SentAt: at(0), DeliveredAt: at(1)}
	prefetch := base
	prefetch.OpenedAt, prefetch.PrivacyOpenedAt = at(1), at(1)
	person := prefetch
	person.HumanOpenedAt = at(125) // hour 2
	var l Ledger
	l.Move(cell, start, base, prefetch)
	l.Move(cell, start, prefetch, person)
	d := l.cells[cell]
	if d == nil || *d != (Delta{OpenedHuman: 1}) {
		t.Fatalf("cell = %+v, want one human open", d)
	}
	if h := l.hours[hourKey{cell.Campaign, 0}]; h == nil || *h != (hourDelta{}) {
		t.Fatalf("hour 0 = %+v, want the prefetch taken back out", h)
	}
	if h := l.hours[hourKey{cell.Campaign, 2}]; h == nil || h.human != 1 {
		t.Fatalf("hour 2 = %+v, want one human open", h)
	}
}

func TestHourOf(t *testing.T) {
	start := *at(0)
	for _, tc := range []struct {
		at   time.Time
		want int
	}{{start.Add(-time.Minute), 0}, {start.Add(59 * time.Minute), 0}, {start.Add(61 * time.Minute), 1}, {start.Add(800 * time.Hour), MaxHour}} {
		if got := HourOf(start, tc.at); got != tc.want {
			t.Errorf("HourOf(%v) = %d, want %d", tc.at.Sub(start), got, tc.want)
		}
	}
}
