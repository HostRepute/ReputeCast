package readiness_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// An Upgrade block that still offers an offer that ended holds the send,
// with the way to the block in the composer (which puts the normal price
// back).
func TestUpgradeOfferEnded(t *testing.T) {
	f := digest()
	ended := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	f.UpgradeOffers = []readiness.EndedUpgradeOffer{{Code: "FALL26", Name: "FALL26", EndedOn: &ended, Path: []int{4}}}
	res := readiness.Assemble(f)
	c, ok := byKey(res)["content.upgrade_offer.4"]
	if !ok || c.Status != readiness.Block || c.Title != "Upgrade block: FALL26 ended on Sat 31 Oct, and the block still offers it" ||
		len(c.Fixes) != 1 || c.Fixes[0].Kind != "link" || !strings.Contains(c.Fixes[0].Href, "/campaigns/c1/edit?") || !strings.Contains(c.Fixes[0].Href, "select=4") {
		t.Fatalf("ended offer: %+v", c)
	}
	if res.Verdict != readiness.VerdictBlocked {
		t.Errorf("verdict %s", res.Verdict)
	}
	f.UpgradeOffers = nil
	for _, c := range readiness.Assemble(f).Checks {
		if strings.HasPrefix(c.Key, "content.upgrade_offer") {
			t.Errorf("check without an ended offer: %+v", c)
		}
	}
}
