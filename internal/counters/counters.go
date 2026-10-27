// Package counters keeps a campaign's report counters (campaign_counters,
// campaign_hourly and the clicks of campaign_links) in step with its
// messages (review AN-3).
//
// A counter is a function of message states, not a tally of events: a
// message counts as delivered when it has a delivery and no hard bounce, as
// soft-bounced while a soft bounce isn't followed by a delivery, as deferred
// once however often it was deferred, and its opens fall in exactly one of
// three buckets (a person's, a privacy proxy's, a scanner's). So counters
// change only when a message's state does: a redelivered webhook, a soft
// bounce followed by a delivery or a machine open followed by a person's
// move each message from one bucket to another instead of counting twice,
// and the counters can always be rebuilt from the stored states and events
// (Rebuild).
//
// Writers move a message with a statement that returns its state before and
// after (CountersMoveDelivery, CountersMoveEngagement), give both states to
// a Ledger and Flush it at the end of their transaction. Flush applies the
// changes in one lock order: campaign_counters rows, then campaign_hourly,
// then campaign_links, each in key order, always after the message rows the
// writer locked. Writers lock messages in id order (sort a batch before
// applying it), so two writers never wait for each other in a cycle.
package counters

import (
	"time"

	"github.com/google/uuid"
)

// Version of the counting rules. Campaigns rebuilt with an older version
// are rebuilt again by the rebuild sweep.
const Version = 1

// MaxHour is the last hour campaign_hourly keeps (30 days).
const MaxHour = 720

// State is what decides where a message counts.
type State struct {
	Status          string
	SentAt          *time.Time
	DeliveredAt     *time.Time
	DeferredAt      *time.Time
	SoftBouncedAt   *time.Time
	OpenedAt        *time.Time
	HumanOpenedAt   *time.Time
	PrivacyOpenedAt *time.Time
	ClickedAt       *time.Time
	UnsubscribedAt  *time.Time
}

// Flags are the counters a message counts in. The rules are the same as
// CountersRebuildCells' (internal/store/queries/counters.sql).
type Flags struct {
	// Sent: the provider took the message (a provider that refuses it
	// afterwards takes it back out: it was never attempted).
	Sent bool
	// Delivered: a delivery event, and no hard bounce since; a complaint
	// proves delivery too.
	Delivered bool
	// Deferred: deferred at least once (once per message).
	Deferred bool
	// HardBounced and Complained follow the status.
	HardBounced bool
	Complained  bool
	// SoftBounced: a soft bounce that no delivery, hard bounce or
	// complaint followed.
	SoftBounced bool
	// Rejected: the provider refused it, when it was handed over or later.
	Rejected bool
	// Opens fall in one bucket: a person's (a click counts), else a privacy
	// proxy's (whether a person read it can't be measured), else a
	// scanner's.
	OpenedHuman   bool
	OpenedPrivacy bool
	OpenedMachine bool
	Clicked       bool
	Unsubscribed  bool
}

// Flags returns the counters a state counts in.
func (s State) Flags() Flags {
	var f Flags
	f.Rejected = s.Status == "rejected"
	f.Sent = s.SentAt != nil && !f.Rejected
	f.HardBounced = s.Status == "bounced"
	f.Complained = s.Status == "complained"
	f.Delivered = f.Complained || (s.DeliveredAt != nil && !f.HardBounced && !f.Rejected)
	f.SoftBounced = s.SoftBouncedAt != nil && !f.Delivered && !f.HardBounced && !f.Complained && !f.Rejected
	f.Deferred = s.DeferredAt != nil
	f.OpenedHuman = s.HumanOpenedAt != nil
	f.OpenedPrivacy = !f.OpenedHuman && s.PrivacyOpenedAt != nil
	f.OpenedMachine = !f.OpenedHuman && !f.OpenedPrivacy && s.OpenedAt != nil
	f.Clicked = s.ClickedAt != nil
	f.Unsubscribed = s.UnsubscribedAt != nil
	return f
}

// Delta is a change of one counter cell (or the counters of a message).
type Delta struct {
	Sent, Delivered, Deferred, HardBounced, SoftBounced, Complained, Rejected int64
	OpenedHuman, OpenedPrivacy, OpenedMachine, Clicked, Unsubscribed          int64
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// Diff is what moving a message from one state to another changes.
func Diff(before, after State) Delta {
	a, b := before.Flags(), after.Flags()
	return Delta{
		Sent: b2i(b.Sent) - b2i(a.Sent), Delivered: b2i(b.Delivered) - b2i(a.Delivered), Deferred: b2i(b.Deferred) - b2i(a.Deferred),
		HardBounced: b2i(b.HardBounced) - b2i(a.HardBounced), SoftBounced: b2i(b.SoftBounced) - b2i(a.SoftBounced),
		Complained: b2i(b.Complained) - b2i(a.Complained), Rejected: b2i(b.Rejected) - b2i(a.Rejected),
		OpenedHuman: b2i(b.OpenedHuman) - b2i(a.OpenedHuman), OpenedPrivacy: b2i(b.OpenedPrivacy) - b2i(a.OpenedPrivacy),
		OpenedMachine: b2i(b.OpenedMachine) - b2i(a.OpenedMachine), Clicked: b2i(b.Clicked) - b2i(a.Clicked),
		Unsubscribed: b2i(b.Unsubscribed) - b2i(a.Unsubscribed),
	}
}

// Zero reports whether the delta changes nothing.
func (d Delta) Zero() bool { return d == Delta{} }

func (d *Delta) add(o Delta) {
	d.Sent += o.Sent
	d.Delivered += o.Delivered
	d.Deferred += o.Deferred
	d.HardBounced += o.HardBounced
	d.SoftBounced += o.SoftBounced
	d.Complained += o.Complained
	d.Rejected += o.Rejected
	d.OpenedHuman += o.OpenedHuman
	d.OpenedPrivacy += o.OpenedPrivacy
	d.OpenedMachine += o.OpenedMachine
	d.Clicked += o.Clicked
	d.Unsubscribed += o.Unsubscribed
}

// Hours are the hours after the send started in which a state's
// engagement counts in campaign_hourly (nil: it doesn't count there).
type Hours struct {
	Human, Privacy, Machine, Clicked *int
}

// HourOf is the hour after start an event happened in, as campaign_hourly
// keeps it (0 to MaxHour).
func HourOf(start, at time.Time) int {
	h := int(at.Sub(start).Hours())
	return min(max(h, 0), MaxHour)
}

// HoursOf places a state's engagement in the hours after start.
func (s State) HoursOf(start time.Time) Hours {
	f := s.Flags()
	at := func(t *time.Time) *int {
		if t == nil {
			return nil
		}
		h := HourOf(start, *t)
		return &h
	}
	var h Hours
	switch {
	case f.OpenedHuman:
		h.Human = at(s.HumanOpenedAt)
	case f.OpenedPrivacy:
		h.Privacy = at(s.PrivacyOpenedAt)
	case f.OpenedMachine:
		h.Machine = at(s.OpenedAt)
	}
	if f.Clicked {
		h.Clicked = at(s.ClickedAt)
	}
	return h
}

// Cell is one campaign_counters row: a campaign's messages through one
// sending provider, to one mailbox provider, in one A/B variant.
type Cell struct {
	Campaign uuid.UUID
	Provider uuid.UUID
	Mailbox  string
	Variant  string
}
