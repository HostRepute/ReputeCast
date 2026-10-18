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

	// The screens behind it, over HTTP (their inputs embed the program's
	// and the link's paths): results, milestones, referrers and one of them,
	// the review queue, the settings' impact and a partner's request log.
	p := ws + "/grow/referrals/" + o.ProgramID
	var res struct {
		Enough bool
		Funnel struct{ Visits, Counted int64 }
		Top    []struct{ LinkID, Name string }
	}
	c.Do("GET", p+"/results?period=all&grouping=week", nil).MustStatus(t, 200).JSON(t, &res)
	if !res.Enough || res.Funnel.Visits != 2940 || res.Funnel.Counted != 139 || len(res.Top) == 0 || res.Top[0].Name != "Omar Haddad" {
		t.Fatalf("results: %+v", res)
	}
	var ms struct{ Items []struct{ ID string } }
	c.Do("GET", p+"/milestones", nil).MustStatus(t, 200).JSON(t, &ms)
	if len(ms.Items) != 3 {
		t.Fatalf("milestones: %+v", ms)
	}
	c.Do("GET", ws+"/grow/milestones/"+ms.Items[0].ID, nil).MustStatus(t, 200)
	var people struct {
		Items []struct{ LinkID, Name string }
	}
	c.Do("GET", p+"/referrers?q=sam.rivera", nil).MustStatus(t, 200).JSON(t, &people)
	if len(people.Items) != 1 || people.Items[0].Name != "Sam Rivera" {
		t.Fatalf("referrers: %+v", people)
	}
	var panel struct {
		Code      string
		Referrals []struct{ Number int }
	}
	c.Do("GET", p+"/referrers/"+people.Items[0].LinkID, nil).MustStatus(t, 200).JSON(t, &panel)
	if panel.Code != "K7Q2MX" || len(panel.Referrals) != 4 {
		t.Fatalf("Sam's panel: %+v", panel)
	}
	var queue struct{ Held int64 }
	c.Do("GET", p+"/review", nil).MustStatus(t, 200).JSON(t, &queue)
	if queue.Held != 9 {
		t.Fatalf("review: %+v", queue)
	}
	c.Do("POST", p+"/settings/impact", map[string]any{"holdDays": 14, "sameIp": true, "disposable": true, "left": true, "numbered": true,
		"burst": map[string]any{"on": true, "threshold": 5}, "sharedIp": map[string]any{"on": true, "threshold": 3}}).MustStatus(t, 200)
	var recs struct {
		Items []struct{ Name, LinkID string }
	}
	c.Do("GET", ws+"/grow/recommendations", nil).MustStatus(t, 200).JSON(t, &recs)
	if len(recs.Items) != 3 || recs.Items[0].LinkID == "" {
		t.Fatalf("recommendations: %+v", recs)
	}
	c.Do("GET", ws+"/grow/partner-links/"+recs.Items[0].LinkID+"/requests", nil).MustStatus(t, 200)

	// Aisha (Viewer) reads the overview; she can't decide or edit.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/grow/overview", nil).MustStatus(t, 200)
	v.Do("POST", ws+"/grow/referrals/"+o.ProgramID+"/off", nil).MustStatus(t, 403)
}
