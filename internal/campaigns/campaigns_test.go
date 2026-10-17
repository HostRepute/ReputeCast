package campaigns_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestListCampaigns(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var list struct {
		Items []struct {
			Name     string `json:"name"`
			Status   string `json:"status"`
			Blocked  bool   `json:"blocked"`
			Audience string `json:"audience"`
			Notes    []string
			Routes   []struct {
				Short   string
				Percent float64
				Sent    int64
			}
			Readiness *struct {
				Verdict string
				Issues  int64
			}
		} `json:"items"`
		Total  int64 `json:"total"`
		Counts struct{ All, Drafts, Scheduled, Sending, Sent int64 }
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns?pageSize=9", nil).MustStatus(t, 200).JSON(t, &list)
	// 48 campaigns of the Campaigns board plus the approval sample data:
	// November Product Update (waiting for approval), two of Daniel's drafts
	// and the Beta Program launch invite (sent). The integrations hub adds
	// the draft blog.acme.io made from a post (Scheduled rollouts are here),
	// and the accessibility sample its draft, Weekly Digest #143. The
	// survey's email, Beta Program: October check-in, is sent.
	if list.Total != 55 || list.Counts.All != 55 || list.Counts.Drafts != 7 || list.Counts.Scheduled != 1 || list.Counts.Sent != 47 {
		t.Fatalf("counts: total %d %+v", list.Total, list.Counts)
	}
	if len(list.Items) != 9 {
		t.Fatalf("page size: %d", len(list.Items))
	}
	first := list.Items[0]
	if first.Name != "Weekly Digest #142" || first.Status != "scheduled" || first.Readiness == nil || first.Readiness.Issues != 2 {
		t.Fatalf("scheduled campaign first: %+v", first)
	}
	if len(first.Routes) != 2 || first.Routes[0].Short != "SES" || first.Routes[0].Percent != 90 {
		t.Fatalf("route split: %+v", first.Routes)
	}
	if list.Items[1].Name != "November Product Update" || list.Items[1].Status != "approval" {
		t.Fatalf("waiting for approval: %+v", list.Items[1])
	}
	// Drafts follow their last save (Weekly Digest #143's is relative to the
	// clock), so the drafts below are found by name.
	var post, flash, oct = list.Items[0], list.Items[0], list.Items[0]
	for _, it := range list.Items {
		switch it.Name {
		case "Scheduled rollouts are here":
			post = it
		case "Flash sale: 48 hours only":
			flash = it
		case "October Product Update":
			oct = it
		}
	}
	// The integrations hub's post draft.
	if post.Name != "Scheduled rollouts are here" || post.Status != "draft" {
		t.Fatalf("post draft: %+v", post)
	}
	// "3 blocking issues", as on the board and on Review & send: its two
	// warnings aren't counted.
	if !flash.Blocked || flash.Audience != "All subscribers" || flash.Readiness == nil || flash.Readiness.Issues != 3 {
		t.Fatalf("blocked draft: %+v", flash)
	}
	if oct.Name != "October Product Update" || len(oct.Notes) == 0 || oct.Notes[0] != "A/B subject, winner B" {
		t.Fatalf("A/B note: %+v", oct)
	}

	c.Do("GET", "/workspaces/acme-newsletter/campaigns?status=drafts", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total != 7 {
		t.Fatalf("drafts: %d", list.Total)
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns?q=flash", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total != 1 || list.Items[0].Name != "Flash sale: 48 hours only" {
		t.Fatalf("search: %+v", list.Items)
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns?status=nope", nil).MustStatus(t, 422)

	var sum struct {
		Campaigns int64
		Sent      int64
		CostBy    []struct{ Short string }
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/summary", nil).MustStatus(t, 200).JSON(t, &sum)
	// Six on the board plus the Beta Program launch invite of the approval
	// history and the survey's email, Beta Program: October check-in.
	if sum.Campaigns != 8 || sum.Sent < 180000 {
		t.Fatalf("summary: %+v", sum)
	}

	// Viewers can see campaigns; people outside the workspace can't.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", "/workspaces/acme-newsletter/campaigns", nil).MustStatus(t, 200)
	out := env.Client()
	out.SignInDemo("linh@acme.io")
	out.Do("GET", "/workspaces/does-not-exist/campaigns", nil).MustStatus(t, 404)
}
