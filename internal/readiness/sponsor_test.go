package readiness_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// The Sponsor block in the Content group (ComposerEarnBlocks): a waiting
// creative blocks the issue with the booking to open and the block to
// remove; a stale block asks to be updated; without a booking the block is
// left out; sponsored content is always labelled.
func TestSponsorBlockChecks(t *testing.T) {
	due := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)
	f := digest()
	f.Sponsor = readiness.SponsorFacts{HasBlock: true, State: "waiting", Sponsor: "Northwind Analytics", Issue: 143, BookingID: "b143", CreativeDue: &due, Labelled: true}
	res := readiness.Assemble(f)
	checks := byKey(res)
	c, ok := checks["content.sponsor.creative"]
	if !ok || c.Status != readiness.Block || c.Title != "Sponsor block: Northwind Analytics’ creative is missing" ||
		!strings.HasPrefix(c.Detail, "Due Fri 16 Oct.") || !hasFix(c, readiness.FixSponsorRemove) {
		t.Fatalf("waiting creative: %+v", c)
	}
	if c.Fixes[0].Kind != "link" || c.Fixes[0].Href != "/w/acme-newsletter/earn/sponsors?booking=b143" {
		t.Errorf("open booking: %+v", c.Fixes[0])
	}
	if l := checks["content.sponsor.label"]; l.Status != readiness.Pass || l.Title != "Sponsored content is labelled" {
		t.Errorf("label: %+v", l)
	}
	if res.Verdict != readiness.VerdictBlocked {
		t.Errorf("verdict %s, want blocked", res.Verdict)
	}

	// The creative arrived after the email was saved.
	f.Sponsor = readiness.SponsorFacts{HasBlock: true, State: "ready", Sponsor: "Northwind Analytics", Issue: 143, BookingID: "b143", Stale: true, Labelled: true}
	if c := byKey(readiness.Assemble(f))["content.sponsor.stale"]; c.Status != readiness.Block || !hasFix(c, readiness.FixSponsorRefresh) {
		t.Errorf("stale: %+v", c)
	}
	// Up to date and labelled: nothing holds the issue.
	f.Sponsor.Stale = false
	res = readiness.Assemble(f)
	for _, c := range res.Checks {
		if strings.HasPrefix(c.Key, "content.sponsor") && c.Status != readiness.Pass {
			t.Errorf("ready block: %+v", c)
		}
	}
	// The label was edited away.
	f.Sponsor.Labelled = false
	if c := byKey(readiness.Assemble(f))["content.sponsor.label"]; c.Status != readiness.Block || !hasFix(c, readiness.FixSponsorRefresh) {
		t.Errorf("unlabelled: %+v", c)
	}
	// No booking: the block is left out of the email.
	f.Sponsor = readiness.SponsorFacts{HasBlock: true, State: "none", Issue: 144, Labelled: true}
	checks = byKey(readiness.Assemble(f))
	if c := checks["content.sponsor.none"]; c.Status != readiness.Pass || !strings.Contains(c.Title, "#144") {
		t.Errorf("no booking: %+v", c)
	}
	if _, ok := checks["content.sponsor.label"]; ok {
		t.Error("a left-out block needs no label check")
	}
	// A booking canceled after the creative went into the email.
	f.Sponsor.Stale = true
	if c := byKey(readiness.Assemble(f))["content.sponsor.stale"]; c.Status != readiness.Block || !hasFix(c, readiness.FixSponsorRefresh) || !hasFix(c, readiness.FixSponsorRemove) {
		t.Errorf("canceled booking: %+v", c)
	}
	// No Sponsor block, no checks.
	f.Sponsor = readiness.SponsorFacts{}
	for _, c := range readiness.Assemble(f).Checks {
		if strings.HasPrefix(c.Key, "content.sponsor") {
			t.Errorf("check without a block: %+v", c)
		}
	}
}
