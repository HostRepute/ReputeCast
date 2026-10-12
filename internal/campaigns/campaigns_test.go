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
			Name      string `json:"name"`
			Status    string `json:"status"`
			Blocked   bool   `json:"blocked"`
			Audience  string `json:"audience"`
			Notes     []string
			Routes    []struct{ Short string; Percent float64; Sent int64 }
			Readiness *struct{ Verdict string; Issues int64 }
		} `json:"items"`
		Total  int64 `json:"total"`
		Counts struct{ All, Drafts, Scheduled, Sending, Sent int64 }
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns?pageSize=9", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total != 48 || list.Counts.All != 48 || list.Counts.Drafts != 2 || list.Counts.Scheduled != 1 || list.Counts.Sent != 45 {
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
	if !list.Items[1].Blocked || list.Items[1].Audience != "All subscribers" {
		t.Fatalf("blocked draft: %+v", list.Items[1])
	}
	var oct = list.Items[3]
	if oct.Name != "October Product Update" || len(oct.Notes) == 0 || oct.Notes[0] != "A/B subject, winner B" {
		t.Fatalf("A/B note: %+v", oct)
	}

	c.Do("GET", "/workspaces/acme-newsletter/campaigns?status=drafts", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total != 2 {
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
	if sum.Campaigns != 6 || sum.Sent < 180000 {
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
