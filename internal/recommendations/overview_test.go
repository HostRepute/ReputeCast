package recommendations_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Grow's screens on the demo (GrowOverview, Referrals, Recommendations):
// the overview's numbers and Needs you, the results the program card reads,
// and what a Viewer may see.
func TestGrowOverview(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	const ws = "/api/v1/workspaces/acme-newsletter"

	var o struct {
		ListName, ProgramID, ProgramStatus string
		Recommends, RecommendedBy          int
		Tiles                              struct {
			FromGrow, NewSubscribers, Visits, Confirmed, PartnerSignups, RewardsSent int64
			Partners                                                                 []string
			Rewards                                                                  []struct {
				Milestone int
				Name      string
				Count     int64
			}
		}
		NeedsYou []struct {
			Kind, Partner, Reason, Reward string
			Count                         int64
			People                        []struct {
				Name  string
				Count int64
			}
			Expires []string
		}
		Weeks []struct {
			Other, Referrals, Partners int64
			Partial                    bool
		}
	}
	c.Do("GET", ws+"/grow/overview", nil).MustStatus(t, 200).JSON(t, &o)
	if o.ListName != "Weekly Digest" || o.ProgramID == "" || o.ProgramStatus != "on" || o.Recommends != 3 || o.RecommendedBy != 1 {
		t.Fatalf("overview: %+v", o)
	}
	tl := o.Tiles
	if tl.Visits != 2940 || tl.Confirmed != 171 || tl.PartnerSignups != 102 || tl.FromGrow != 273 || tl.NewSubscribers <= tl.FromGrow ||
		len(tl.Partners) != 1 || tl.Partners[0] != "Ops Weekly" {
		t.Fatalf("tiles: %+v", tl)
	}
	if tl.RewardsSent == 0 || len(tl.Rewards) == 0 || tl.Rewards[0].Milestone != 1 {
		t.Fatalf("rewards: %+v", tl)
	}
	// The weeks add up to the tiles' period and more (the first week starts
	// on a Monday before it); the last one is still going.
	var refs, partners int64
	for _, w := range o.Weeks {
		refs += w.Referrals
		partners += w.Partners
	}
	if len(o.Weeks) < 5 || refs < tl.Confirmed || partners < tl.PartnerSignups || !o.Weeks[len(o.Weeks)-1].Partial {
		t.Fatalf("weeks: %+v", o.Weeks)
	}
	// Needs you: held referrals by referrer, the invalid partner link and
	// the Insider coupons nobody claimed.
	kinds := map[string]int{}
	for i, n := range o.NeedsYou {
		kinds[n.Kind] = i
	}
	held, ok := kinds["held"]
	if !ok || o.NeedsYou[held].Count != 9 || len(o.NeedsYou[held].People) != 2 || o.NeedsYou[held].People[0].Name != "Alex Moreau" || o.NeedsYou[held].People[0].Count != 6 {
		t.Fatalf("held: %+v", o.NeedsYou)
	}
	inv, ok := kinds["partner_invalid"]
	if !ok || o.NeedsYou[inv].Partner != "The Deliverability Desk" || o.NeedsYou[inv].Reason != "signature_rejected" {
		t.Fatalf("partner: %+v", o.NeedsYou)
	}
	un, ok := kinds["unclaimed"]
	if !ok || o.NeedsYou[un].Count != 2 || len(o.NeedsYou[un].Expires) != 2 {
		t.Fatalf("unclaimed: %+v", o.NeedsYou)
	}

	// Aisha (Viewer) reads the overview; she can't decide or edit.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/grow/overview", nil).MustStatus(t, 200)
	v.Do("POST", ws+"/grow/referrals/"+o.ProgramID+"/off", nil).MustStatus(t, 403)
}
