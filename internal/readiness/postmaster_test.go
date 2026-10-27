package readiness_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// SR-12: with Gmail Postmaster data for the domain, the bulk-sender check
// is Google's compliance status on its latest day, beside ReputeCast's
// checks of the same rules for Yahoo; without it, ReputeCast's checks
// stand alone and say so.

func postmasterDay(f readiness.Facts, daysAgo int) time.Time {
	d := f.Now.In(f.Location).AddDate(0, 0, -daysAgo)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func withPostmaster(f readiness.Facts, daysAgo int, dmarc, tls float64) readiness.Facts {
	f.Compliance.Gmail = &readiness.PostmasterFacts{Domain: "news.acme.io", Day: postmasterDay(f, daysAgo), SpamRate: f64(0.05),
		SPF: f64(100), DKIM: f64(100), DMARC: f64(dmarc), TLS: f64(tls)}
	f.Compliance.PostmasterConnected = true
	return f
}

// The board: "Gmail Postmaster: Compliant (yesterday)".
func TestPostmasterCompliant(t *testing.T) {
	res := stored(t, readiness.Assemble(withPostmaster(digest(), 1, 100, 100)))
	c := byKey(res)["compliance.bulk_sender"]
	if c.Status != readiness.Pass || c.Title != "Gmail Postmaster: Compliant (yesterday)" ||
		c.Detail != "Google reports compliance a day late. Yahoo has no such report, so ReputeCast checks the same rules itself: DMARC aligned, one-click unsubscribe, complaint rate 0.04% under the 0.3% limit." {
		t.Fatalf("compliant: %q · %q", c.Title, c.Detail)
	}
	vi := res.In(i18n.For(i18n.Vietnamese))
	for _, ch := range vi.Checks {
		if ch.Key == "compliance.bulk_sender" && (ch.Title != "Gmail Postmaster: Tuân thủ (hôm qua)" || !strings.Contains(ch.Detail, "tỷ lệ khiếu nại 0,04% dưới giới hạn 0,3%")) {
			t.Errorf("in Vietnamese: %q · %q", ch.Title, ch.Detail)
		}
	}
}

// Short of the rules: Needs work, naming what Google measured (Warn).
func TestPostmasterNeedsWork(t *testing.T) {
	f := withPostmaster(digest(), 1, 96.2, 97)
	c := byKey(stored(t, readiness.Assemble(f)))["compliance.bulk_sender"]
	if c.Status != readiness.Warn || c.Title != "Gmail Postmaster: Needs work (yesterday)" || c.Ack == "" || len(c.Fixes) == 0 ||
		!strings.HasPrefix(c.Detail, "Google measured DMARC 96.2% and TLS 97% for news.acme.io, short of its bulk-sender rules.") {
		t.Fatalf("needs work: %+v", c)
	}
	// An older report is dated.
	f = withPostmaster(digest(), 3, 96.2, 100)
	c = byKey(stored(t, readiness.Assemble(f)))["compliance.bulk_sender"]
	if strings.Contains(c.Title, "yesterday") || !strings.HasPrefix(c.Title, "Gmail Postmaster: Needs work (") {
		t.Errorf("an older day: %q", c.Title)
	}
	// Over Google's spam limit the spam rate blocks, and Postmaster's
	// status says why too.
	f = withPostmaster(digest(), 1, 100, 100)
	f.Compliance.Gmail.SpamRate = f64(0.35)
	c = byKey(stored(t, readiness.Assemble(f)))["compliance.bulk_sender"]
	if c.Status != readiness.Block || !strings.Contains(c.Title, "Gmail spam rate 0.35%") || !strings.HasPrefix(c.Detail, "Google measured a spam rate of 0.35% for news.acme.io") {
		t.Errorf("spam over the limit: %+v", c)
	}
}

// Not connected: ReputeCast's own checks, and a way to connect it; a
// connected workspace without data for the domain says so.
func TestPostmasterNotConnected(t *testing.T) {
	f := digest()
	c := byKey(stored(t, readiness.Assemble(f)))["compliance.bulk_sender"]
	if c.Status != readiness.Pass || c.Title != "Bulk-sender requirements met" || !strings.HasSuffix(c.Detail, " · Gmail Postmaster not connected") ||
		len(c.Fixes) != 1 || c.Fixes[0].Label != "Connect Postmaster" || c.Fixes[0].Href != "/w/acme-newsletter/deliverability/connect" {
		t.Fatalf("not connected: %+v", c)
	}
	f.Compliance.PostmasterConnected = true
	c = byKey(stored(t, readiness.Assemble(f)))["compliance.bulk_sender"]
	if c.Status != readiness.Pass || !strings.HasSuffix(c.Detail, " · no Gmail Postmaster data for news.acme.io yet") || len(c.Fixes) != 0 {
		t.Fatalf("no data for the domain: %+v", c)
	}
}
