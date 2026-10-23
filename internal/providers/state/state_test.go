package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

func TestBulkAllowed(t *testing.T) {
	for _, c := range []struct {
		snap Snapshot
		want bool
	}{
		{Snapshot{Capabilities: providers.Capabilities{Bulk: providers.BulkAllowed}}, true},
		{Snapshot{Capabilities: providers.Capabilities{Bulk: providers.BulkYourPolicy}}, true},
		{Snapshot{Capabilities: providers.Capabilities{Bulk: providers.BulkBroadcastStreams}}, false},
		{Snapshot{Capabilities: providers.Capabilities{Bulk: providers.BulkBroadcastStreams}, BroadcastStream: true}, true},
		{Snapshot{Capabilities: providers.Capabilities{Bulk: providers.BulkForbidden}}, false},
	} {
		if got := c.snap.BulkAllowed(); got != c.want {
			t.Errorf("%+v: %v", c.snap, got)
		}
	}
}

func TestWarmupDayCountsCalendarDays(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	start := time.Date(2026, 10, 5, 0, 0, 0, 0, ny)
	if d := WarmupDay(start, time.Date(2026, 10, 13, 23, 30, 0, 0, ny), ny); d != 9 {
		t.Fatalf("day = %d", d)
	}
	// Across the DST change the count stays on whole days.
	if d := WarmupDay(start, time.Date(2026, 11, 2, 9, 0, 0, 0, ny), ny); d != 29 {
		t.Fatalf("day after DST = %d", d)
	}
}

func TestReputationOfIsFreshOnly(t *testing.T) {
	now := time.Now()
	h := Health{Reputation: &Reputation{Bounce: 0.2, Complaint: 0.01, CheckedAt: now.Add(-time.Hour)}}
	if r := ReputationOf(h, now); r == nil || r.Complaint != 0.01 {
		t.Fatalf("fresh reputation %+v", r)
	}
	h.Reputation.CheckedAt = now.Add(-3 * 24 * time.Hour)
	if ReputationOf(h, now) != nil {
		t.Fatal("stale reputation used")
	}
}

func TestDecodersHaveDefaults(t *testing.T) {
	p := store.Provider{Adapter: "nope"}
	if l := LimitsOf(p); l.Bounce.Limit != 5 || l.Complaint.Limit != 0.1 {
		t.Fatalf("limits %+v", l)
	}
	if pr := PricingOf(p); pr.Model != "metered" {
		t.Fatalf("pricing %+v", pr)
	}
	if tr := TrackingOf(p); tr.Mode != "reputecast" {
		t.Fatalf("tracking %+v", tr)
	}
	p.Health = json.RawMessage(`{"vdm": true, "quota": {"kind": "rolling_24h", "max": 50000, "used": 2140}}`)
	if h := HealthOf(p); !h.VDM || h.Quota.Max != 50000 {
		t.Fatalf("health %+v", h)
	}
	if Short("Amazon SES", "ses") != "SES" || Short("Postal (self-hosted)", "postal") != "Postal" || Color("mailgun") != "other" {
		t.Fatal("display helpers")
	}
}

func TestDetailInReadsTheHealthChecksWords(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	for detail, want := range map[string]string{
		DetailDown:           "Ngừng hoạt động",
		DetailDownAuth:       "Ngừng hoạt động · xác thực thất bại",
		DetailWebhooksSilent: "Webhook im lặng",
		"Approved":           "Approved", // the provider's own word for the account
		"":                   "",
	} {
		if got := DetailIn(detail, vi); got != want {
			t.Errorf("%q: got %q, want %q", detail, got, want)
		}
	}
	if got := DetailIn(DetailDownAuth, i18n.For(i18n.English)); got != DetailDownAuth {
		t.Errorf("English: %q", got)
	}
}
