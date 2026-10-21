package overview_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	// The Campaigns board's two drafts plus the approval sample data:
	// November Product Update (waiting for approval) and two of Daniel's
	// drafts from the approval inbox's history, the draft blog.acme.io made
	// from a post (integrations hub) and the accessibility sample's Weekly
	// Digest #143. With texts (X21, brief §4), Daniel Park's text draft
	// "Summit: doors open" is one more.
	if n.Drafts != 8 {
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

	// Subscriber growth and tiers, from the audience domain (Main board).
	var a struct {
		Total, Subscribers, Net int64
		NetPercent              float64
		Days                    int
		Series                  []struct {
			Day         string
			Subscribers int64
		}
		Lists []struct {
			Name          string
			Contacts, Net int64
		}
		Tiers []struct {
			Tier, Description string
			Count             int64
			Pct               float64
		}
		Policy string
	}
	c.Do("GET", base+"/audience", nil).MustStatus(t, 200).JSON(t, &a)
	// The boards' 48,213, with Acme Summit 2026 (X21, brief §8.12): 338 new
	// contacts, and 1,240 registrations on its private list in the last 30 days.
	if a.Days != 30 || a.Total != 48213+338 || a.Subscribers != 48213+1240 || a.Net != 1284+1240 || a.NetPercent != 5.4 ||
		a.Policy != "Sunset contacts get one re-engagement email, then stop receiving broadcasts after 30 days." {
		t.Fatalf("audience: %+v", a)
	}
	if len(a.Series) != 31 || a.Series[0].Subscribers != 46929 || a.Series[30].Subscribers != 48213+1240 {
		t.Fatalf("growth line: %+v", a.Series)
	}
	wantLists := []string{"Weekly Digest 31988 812", "Product Updates 14110 391", "Beta Program 2115 81", "Acme Summit 2026 1240 1240"}
	for i, l := range a.Lists {
		if i >= len(wantLists) || fmt.Sprintf("%s %d %d", l.Name, l.Contacts, l.Net) != wantLists[i] {
			t.Fatalf("lists: %+v", a.Lists)
		}
	}
	wantTiers := []string{"active 29750 61.3 opened in the last 30 days", "cooling 9884 20.4 last open 31 to 90 days ago",
		"at_risk 5302 10.9 last open 91 to 180 days ago", "sunset 3615 7.4 no open in 180 days"}
	for i, tr := range a.Tiers {
		if i >= len(wantTiers) || fmt.Sprintf("%s %d %.1f %s", tr.Tier, tr.Count, tr.Pct, tr.Description) != wantTiers[i] {
			t.Fatalf("tiers: %+v", a.Tiers)
		}
	}

	var r struct {
		Items []struct{ Name, Status string }
	}
	c.Do("GET", base+"/recent-campaigns", nil).MustStatus(t, 200).JSON(t, &r)
	// The approval inbox's history adds the Beta Program launch invite (sent
	// Oct 6), which the Main board doesn't list, and the recurring digest its
	// issue #2 (sent Oct 10): recent campaigns include sent issues. Texts are
	// campaigns too (X21): "Summit: 8 days to go" went out on Oct 18.
	if len(r.Items) != 4 || r.Items[0].Name != "Summit: 8 days to go" || r.Items[1].Name != "October Product Update" || r.Items[3].Name != "Engineering blog roundup #2" {
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
