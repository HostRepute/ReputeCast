package ai

import (
	"slices"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// TestReportNumbers: a summary's numbers are filled in from their metrics
// in the reader's format, and numbers written otherwise are checked
// against the report's (review AN-6).
func TestReportNumbers(t *testing.T) {
	nums := reportNumbers{
		"delivered":     {Value: 47548, Kind: kindCount},
		"humanOpenRate": {Value: 41.2, Kind: kindRate},
		"complaintRate": {Value: 0.04, Kind: kindSmallRate},
		"bounceRate":    {Value: 0.9925, Kind: kindRate},
		"ab.confidence": {Value: 98, Kind: kindShare},
		"ab.lift":       {Value: 0.8, Kind: kindPoints},
	}
	// Each rate has the decimals the report shows it with: Outlook's
	// bounces as 1.0%, complaints as 0.04%.
	text := "{delivered} delivered, {humanOpenRate} opened, {bounceRate} bounced, {complaintRate} complained, {ab.confidence} confidence, {ab.lift} points ahead, {opensSoFar}."
	en, unknown := fill(text, nums, i18n.Standard)
	if en != "47,548 delivered, 41.2% opened, 1.0% bounced, 0.04% complained, 98% confidence, 0.8 points ahead, {opensSoFar}." || !slices.Equal(unknown, []string{"{opensSoFar}"}) {
		t.Fatalf("English: %q %v", en, unknown)
	}
	vi, _ := fill(text, nums, i18n.Region("vi-VN"))
	if vi != "47.548 delivered, 41,2% opened, 1,0% bounced, 0,04% complained, 98% confidence, 0,8 points ahead, {opensSoFar}." {
		t.Fatalf("Vietnamese: %q", vi)
	}
	for _, c := range []struct {
		text   string
		region i18n.Region
		want   []string
	}{
		{"47.548 người, 41,2% đã mở", i18n.Region("vi-VN"), nil},
		{"47,548 people, 41.2% opened", i18n.Standard, nil},
		{"Outlook bounced 0.99% (1.0%)", i18n.Standard, nil},
		{"47,548 people, 41.3% opened, 48,000 sent", i18n.Standard, []string{"41.3%", "48,000"}},
		{"In 2026, 3 links drew 98% of the clicks over 72 hours.", i18n.Standard, nil},
		{"Blocked with 550 5.7.1 (S3150).", i18n.Standard, nil},
	} {
		got := uncheckedNumbers(c.text, nums, c.region, "Postal 550 5.7.1 (Outlook S3150)")
		if !slices.Equal(got, c.want) {
			t.Errorf("%q: unchecked %v, want %v", c.text, got, c.want)
		}
	}
}

// TestPageOf: a model gets a link's host and path, not its query.
func TestPageOf(t *testing.T) {
	for in, want := range map[string]string{
		"https://docs.acme.io/flags/?utm_source=reputecast#top": "docs.acme.io/flags",
		"https://acme.io":   "acme.io",
		"not a url":         "not a url",
		"mailto:hi@acme.io": "mailto:hi@acme.io",
	} {
		if got := pageOf(in); got != want {
			t.Errorf("pageOf(%q) = %q, want %q", in, got, want)
		}
	}
}
