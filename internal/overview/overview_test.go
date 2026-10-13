package overview_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

func TestOverviewCards(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	base := "/workspaces/acme-newsletter/overview"

	var h struct {
		State, Label, Domain string
		Score                int
		Reasons              []struct{ Key, Tone, Text string }
	}
	c.Do("GET", base+"/sender-health", nil).MustStatus(t, 200).JSON(t, &h)
	if h.State != "ready" || h.Score != 92 || h.Label != "Good" || h.Domain != "news.acme.io" || len(h.Reasons) != 5 {
		t.Fatalf("sender health: %+v", h)
	}
	if h.Reasons[3].Tone != "warn" || h.Reasons[3].Text != "Postal IP 203.0.113.25 is listed on 1 of 84 sources (UCEPROTECT L3)" {
		t.Fatalf("IP reason: %+v", h.Reasons[3])
	}

	var n struct {
		Drafts   int64
		Campaign *struct {
			Name            string
			Recipients      int64
			Cost            float64
			DurationMinutes *float64
			Routes          []struct {
				Short   string
				Percent float64
			}
			Readiness *struct {
				Verdict string
				Issues  []struct{ Status, Title string }
			}
		}
	}
	c.Do("GET", base+"/next-campaign", nil).MustStatus(t, 200).JSON(t, &n)
	nc := n.Campaign
	if nc == nil || nc.Name != "Weekly Digest #142" || nc.Recipients != 31604 || nc.Cost != 2.84 || nc.DurationMinutes == nil || *nc.DurationMinutes != 34 ||
		len(nc.Routes) != 2 || nc.Routes[0].Short != "SES" || nc.Routes[0].Percent != 90 || nc.Readiness == nil || nc.Readiness.Verdict != "warnings" || len(nc.Readiness.Issues) != 2 {
		t.Fatalf("next campaign: %+v", nc)
	}
	if n.Drafts != 2 {
		t.Fatalf("drafts: %d", n.Drafts)
	}

	var p struct {
		Connected int
		Providers []struct {
			DisplayName, Detail string
			Card                struct {
				Name, Status string
				Quota        struct {
					Kind      string
					Max, Used int64
				}
				Cost struct{ ToDate float64 }
			}
		}
	}
	c.Do("GET", base+"/providers", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Connected != 3 || len(p.Providers) != 3 {
		t.Fatalf("providers: %+v", p)
	}
	ses, postal := p.Providers[0], p.Providers[2]
	if ses.DisplayName != "Amazon SES" || ses.Detail != "us-east-1 · Broadcasts, Automations" || ses.Card.Quota.Max != 50000 || ses.Card.Cost.ToDate != 14.62 {
		t.Fatalf("SES card: %+v", ses)
	}
	if postal.DisplayName != "Postal (self-hosted)" || postal.Detail != "mta.acme.io · 203.0.113.25 dedicated · 10% of Broadcasts" || postal.Card.Status != "warming_up" {
		t.Fatalf("Postal card: %+v", postal)
	}

	var a struct {
		Total  int64
		Days   int
		Series []struct{ Day string }
		Lists  []struct{ Name string }
		Tiers  []struct{ Tier, Description string }
		Policy string
	}
	c.Do("GET", base+"/audience", nil).MustStatus(t, 200).JSON(t, &a)
	if a.Days != 30 || len(a.Series) != 31 || len(a.Lists) != 3 || len(a.Tiers) != 4 || a.Tiers[0].Description != "opened in the last 30 days" ||
		a.Policy != "Sunset contacts get one re-engagement email, then stop receiving broadcasts after 30 days." {
		t.Fatalf("audience: %+v", a)
	}

	var r struct {
		Items []struct{ Name, Status string }
	}
	c.Do("GET", base+"/recent-campaigns", nil).MustStatus(t, 200).JSON(t, &r)
	if len(r.Items) != 4 || r.Items[0].Name != "October Product Update" || r.Items[3].Name != "Weekly Digest #140" {
		t.Fatalf("recent: %+v", r.Items)
	}

	// Viewers see every card; a Content Writer only the campaign cards.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	for _, card := range []string{"/sender-health", "/next-campaign", "/providers", "/audience", "/recent-campaigns"} {
		v.Do("GET", base+card, nil).MustStatus(t, 200)
	}
	// A role with campaigns only sees the campaign cards.
	ctx := context.Background()
	ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	role, err := env.App.Q.CreateRole(ctx, store.CreateRoleParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Key: "campaigns-only", Name: "Campaigns only",
		Permissions: authz.NewSet(authz.CampaignsView).Strings()})
	if err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "intern@acme.io", Name: "Intern"})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	d := env.Client()
	d.SignIn("intern@acme.io", "")
	d.Do("GET", base+"/next-campaign", nil).MustStatus(t, 200)
	d.Do("GET", base+"/recent-campaigns", nil).MustStatus(t, 200)
	if pr := d.Do("GET", base+"/providers", nil).MustStatus(t, 403).Problem(t); pr["permission"] != "providers:view" {
		t.Fatalf("providers denied: %v", pr)
	}
	d.Do("GET", base+"/sender-health", nil).MustStatus(t, 403)
	d.Do("GET", base+"/audience", nil).MustStatus(t, 403)
}

func TestOverviewFirstRun(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "founder@example.com", Name: "Founder"})
	ws := env.CreateWorkspace("Fresh Letter", owner.ID)
	c := env.Client()
	c.SignIn("founder@example.com", "correct horse battery staple")
	base := "/workspaces/" + ws.Slug + "/overview"
	var p struct{ Connected int }
	c.Do("GET", base+"/providers", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Connected != 0 {
		t.Fatalf("connected: %d", p.Connected)
	}
	var h struct {
		State   string
		Reasons []any
	}
	c.Do("GET", base+"/sender-health", nil).MustStatus(t, 200).JSON(t, &h)
	if h.State != "pending" || len(h.Reasons) != 0 {
		t.Fatalf("health: %+v", h)
	}
	var n struct{ Campaign any }
	c.Do("GET", base+"/next-campaign", nil).MustStatus(t, 200).JSON(t, &n)
	if n.Campaign != nil {
		t.Fatalf("next: %+v", n)
	}
	var r struct{ Items []any }
	c.Do("GET", base+"/recent-campaigns", nil).MustStatus(t, 200).JSON(t, &r)
	if len(r.Items) != 0 {
		t.Fatalf("recent: %+v", r)
	}
}
