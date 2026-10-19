package readiness_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// Grow's rows join the Content group (SendReadiness board, phase 5): a
// passing Referral block row, and a warning that needs acknowledging when
// the program is off.
func TestAssembleGrowRows(t *testing.T) {
	f := digest()
	f.Grow = []readiness.GrowRow{
		{Key: "content.referral_block", Status: readiness.Pass, Title: "Referral block: shown to 31,604 of 31,604 recipients", Label: "Referral block"},
		{Key: "content.recommendations_block", Status: readiness.Warn, Title: "No recommendation can show right now. The Recommendations block will be hidden.",
			Label: "Recommendations block", Ack: "I understand the Recommendations block won't show.",
			Links: []readiness.GrowLink{{Label: "Open recommendations", Href: "/grow/recommendations"}}},
	}
	r := readiness.Assemble(f)
	found := map[string]readiness.ReadinessCheck{}
	for _, c := range r.Checks {
		found[c.Key] = c
	}
	ref, ok := found["content.referral_block"]
	if !ok || ref.Group != readiness.GroupContent || ref.Status != readiness.Pass {
		t.Fatalf("referral row: %+v", ref)
	}
	recs := found["content.recommendations_block"]
	if recs.Status != readiness.Warn || recs.Ack == "" || len(recs.Fixes) != 1 || recs.Fixes[0].Href != "/w/acme-newsletter/grow/recommendations" {
		t.Fatalf("recommendations row: %+v", recs)
	}
	if r.Counts.Warnings < 3 {
		t.Fatalf("the warning doesn't count: %+v", r.Counts)
	}
}
