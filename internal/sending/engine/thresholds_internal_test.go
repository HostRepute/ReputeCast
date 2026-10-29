package engine

import (
	"cmp"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Complaint rates count only mail delivered where mailbox providers report
// complaints: Gmail's deliveries don't dilute Outlook's complaints (review
// F-2, GS-6).
func TestComplaintsCountWhereReported(t *testing.T) {
	pid := uuid.New()
	ts := tallyRows([]store.SendingCountersRow{
		{ProviderID: pid, MailboxProvider: "gmail", Sent: 9000, Delivered: 8900},
		{ProviderID: pid, MailboxProvider: "icloud", Sent: 1000, Delivered: 990},
		{ProviderID: pid, MailboxProvider: "outlook", Sent: 4100, Delivered: 4000, Complained: 7},
		{ProviderID: pid, MailboxProvider: "yahoo", Sent: 1800, Delivered: 1700, Complained: 1},
	})
	if ts.total.reported != 5700 {
		t.Fatalf("reported %d, want Outlook's and Yahoo's 5,700", ts.total.reported)
	}
	if got := round2(ts.total.complaintRate()); got != 0.14 {
		t.Fatalf("complaint rate %.2f, want 0.14 (8 of 5,700)", got)
	}
	if ts.mailbox["gmail"].reported != 0 {
		t.Fatal("Gmail's deliveries count as reported")
	}
	rows := liveThresholds(DefaultThresholds, ts, guardState{}, nil, typed{})
	byKey := map[string]LiveThreshold{}
	for _, r := range rows {
		byKey[r.Key] = r
	}
	c := byKey["complaint"]
	if c.Base != 5700 || c.Events != 8 || len(c.Parts) != 2 {
		t.Fatalf("complaint row %+v", c)
	}
	m := byKey["mailbox_complaint"]
	if m.Mailbox != "outlook" || m.State != "collecting" && m.State != "tripped" {
		t.Fatalf("mailbox complaint row %+v, want Outlook over its limit", m)
	}
}

// A threshold's state: watching, close to the limit, collecting evidence
// while it can't trip yet, tripped.
func TestThresholdState(t *testing.T) {
	for _, c := range []struct {
		events, base int64
		want         string
	}{
		{2, 4049, "ok"},         // 0.05% of 0.10%
		{6, 6446, "near"},       // 0.09%: close to the limit
		{7, 3980, "tripped"},    // 0.18% with enough evidence
		{3, 1000, "tripped"},    // 0.30%: 3 events, but clearly over (review R2.L7c)
		{3, 1284, "collecting"}, // 0.23% on 3 events: not clearly over yet
		{2, 300, "collecting"},  // over, on 300 delivered, but only 2 events
		{10, 300, "tripped"},    // 3.3% of a small send
		{0, 0, "na"},            // nothing delivered
	} {
		r := thresholdRow("mailbox_complaint", c.events, c.base, minEvents, minSampleScoped, 0.10)
		if r.State != c.want || !r.AnyVolume {
			t.Errorf("%d in %d: %s (any volume %v), want %s", c.events, c.base, r.State, r.AnyVolume, c.want)
		}
	}
	// Bounces of every kind together keep the floors alone: a few unknown
	// users say nothing about how mailbox providers see the sender.
	if r := thresholdRow("bounce", 30, 300, minEvents, minSampleCampaign, 2); r.State != "collecting" || r.AnyVolume {
		t.Errorf("30 bounces in 300 sent: %s (any volume %v), want collecting", r.State, r.AnyVolume)
	}
}

// Small sends (review R2.L7c): complaints and block/policy bounces trip at
// any volume once clearly over their limit (the lower end of the rate's
// one-sided 95% Wilson interval above it, from 3 events); other rates keep
// the floors.
func TestClearlyOverAtAnyVolume(t *testing.T) {
	for _, c := range []struct {
		events, base int64
		limit        float64
		want         bool
	}{
		{3, 300, 0.10, true},   // 1%: its interval starts at 0.40%
		{10, 300, 0.10, true},  // 3.3%
		{2, 300, 0.10, false},  // too few events, whatever the rate
		{3, 1284, 0.10, false}, // 0.23%: its interval reaches down to 0.09%
		{3, 1900, 0.10, false}, // 0.16%
		{3, 200, 0.5, true},    // block/policy: 1.5% of 200 sent
		{3, 500, 0.5, false},   // 0.6%: too close to tell
		{4, 300, 0.5, true},    // 1.3%
		{0, 0, 0.10, false},
	} {
		if got := clearlyOver(c.events, c.base, c.limit); got != c.want {
			t.Errorf("%d in %d against %.2f%%: %v (lower end %.3f%%), want %v", c.events, c.base, c.limit, got, wilsonLower(c.events, c.base), c.want)
		}
	}
	// The floors' rule still trips first at large volumes, and says so.
	if ok, clear := exceeds("mailbox_complaint", 7, 3980, minEvents, minSampleScoped, 0.10); !ok || clear {
		t.Errorf("7 in 3,980: %v clear %v, want the rate rule", ok, clear)
	}
	if ok, clear := exceeds("mailbox_complaint", 3, 300, minEvents, minSampleScoped, 0.10); !ok || !clear {
		t.Errorf("3 in 300: %v clear %v, want clearly over", ok, clear)
	}
	if ok, _ := exceeds("deferral", 30, 300, minDeferrals, minSampleScoped, 5); ok {
		t.Error("deferrals tripped at any volume")
	}
}

// The complaint watch alerts once for the same complaints (review R2.L7c):
// the whole send leaves out a mailbox provider over its own limit, or
// that alerted before, so a small send's complaints at Outlook alert for
// Outlook alone; complaints spread thin over mailbox providers that can't
// trip on their own alert for the whole send.
func TestWatchAlertsOncePerComplaint(t *testing.T) {
	pid := uuid.New()
	th := Thresholds{Complaint: 0.1, MailboxComplaint: 0.1}
	scopes := func(trips []watchTrip) []string {
		var out []string
		for _, tr := range trips {
			out = append(out, cmp.Or(tr.mailbox, "campaign"))
		}
		return out
	}
	small := tallyRows([]store.SendingCountersRow{
		{ProviderID: pid, MailboxProvider: "outlook", Sent: 300, Delivered: 300, Complained: 3},
		{ProviderID: pid, MailboxProvider: "yahoo", Sent: 200, Delivered: 200},
	})
	if got := scopes(watchTrips(th, small, nil)); !slices.Equal(got, []string{"outlook"}) {
		t.Errorf("3 complaints at Outlook alert for %v, want Outlook alone", got)
	}
	// Outlook alerted on an earlier run: still nothing for the whole send.
	if got := scopes(watchTrips(th, small, map[string]bool{"outlook": true})); !slices.Equal(got, []string{"outlook"}) {
		t.Errorf("after Outlook's alert: %v, want Outlook's trip alone (already told)", got)
	}
	spread := tallyRows([]store.SendingCountersRow{
		{ProviderID: pid, MailboxProvider: "outlook", Sent: 300, Delivered: 300, Complained: 2},
		{ProviderID: pid, MailboxProvider: "yahoo", Sent: 200, Delivered: 200, Complained: 2},
	})
	if got := scopes(watchTrips(th, spread, nil)); !slices.Equal(got, []string{"campaign"}) {
		t.Errorf("2 + 2 complaints in 500 alert for %v, want the whole send", got)
	}
}

// A burst of block/policy bounces at one mailbox provider pauses it with
// far less evidence than unknown users need (bytype.go): 3 within 10
// minutes, clearly over the limit among what was sent to it then; not at
// "Other", and not in the window after a person resumed it.
func TestBlockBurst(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC)
	ty := typed{recentBlock: map[string]int64{"outlook": 3, "gmail": 3, "other": 9}, sent: map[int]map[string]int64{blockBurstMinutes: {"outlook": 180, "gmail": 9000, "other": 200}}}
	if b, s, ok := ty.blockBurst("outlook", 0.5, time.Time{}, now); !ok || b != 3 || s != 180 {
		t.Errorf("Outlook: %d blocks in %d sent, burst %v; want a burst", b, s, ok)
	}
	if _, _, ok := ty.blockBurst("gmail", 0.5, time.Time{}, now); ok {
		t.Error("3 blocks among 9,000 sent to Gmail in 10 minutes made a burst")
	}
	if _, _, ok := ty.blockBurst("other", 0.5, time.Time{}, now); ok {
		t.Error(`"Other" made a burst`)
	}
	if _, _, ok := ty.blockBurst("outlook", 0.5, now.Add(-5*time.Minute), now); ok {
		t.Error("a burst within the window after a resume")
	}
	if _, _, ok := ty.blockBurst("outlook", 0.5, now.Add(-11*time.Minute), now); !ok {
		t.Error("no burst a whole window after a resume")
	}
}
