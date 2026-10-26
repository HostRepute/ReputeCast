package engine

import (
	"testing"

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
	rows := liveThresholds(DefaultThresholds, ts, guardState{}, nil)
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
		{3, 1000, "collecting"}, // 0.30% but 3 events
		{2, 300, "collecting"},  // over, on 300 delivered
		{0, 0, "na"},            // nothing delivered
	} {
		r := thresholdRow("mailbox_complaint", c.events, c.base, minEvents, minSampleScoped, 0.10)
		if r.State != c.want {
			t.Errorf("%d in %d: %s, want %s", c.events, c.base, r.State, c.want)
		}
	}
}
