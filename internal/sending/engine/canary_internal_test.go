package engine

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/sending/rules"
)

// TestSeedGateNeedsArrivals: the seed placement gate passes only once
// copies arrived and none is pending; a test where nothing arrived says
// nothing about placement, so it keeps collecting instead of passing.
func TestSeedGateNeedsArrivals(t *testing.T) {
	cs := canaryState{SeedCheck: "check", Seeds: 10, Gates: rules.DefaultCanary.Gates}
	state := func(p *SeedPlacement) string {
		t.Helper()
		cs.SeedResult = p
		g := canaryGates(cs, tallies{}, typed{})
		last := g[len(g)-1]
		if last.Key != "seeds" {
			t.Fatalf("last gate %q, want seeds", last.Key)
		}
		return last.State
	}
	for _, c := range []struct {
		name string
		p    *SeedPlacement
		want string
	}{
		{"not read yet", nil, "collecting"},
		{"still arriving", &SeedPlacement{Inbox: 5, Pending: 5}, "collecting"},
		{"none arrived", &SeedPlacement{Missing: 10}, "collecting"},
		{"arrived", &SeedPlacement{Inbox: 8, Promotions: 2}, "passing"},
		{"some missing", &SeedPlacement{Inbox: 8, Missing: 2}, "passing"},
		{"spam over the gate", &SeedPlacement{Inbox: 5, Spam: 3, Pending: 2}, "failing"},
	} {
		if got := state(c.p); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}
