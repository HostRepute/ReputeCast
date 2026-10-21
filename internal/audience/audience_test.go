package audience_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const ws = "/workspaces/acme-newsletter"

// demoEnv seeds the Acme demo once for every subtest: the numbers below are
// the ones the design boards show.
func demoEnv(t *testing.T) (*apptest.Env, *apptest.Client) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	return env, c
}

// audits counts allowed audit events with an action.
func audits(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND result = 'allowed'`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAudienceDemo(t *testing.T) {
	env, c := demoEnv(t)
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")

	t.Run("summary and contacts match the boards", func(t *testing.T) {
		var sum struct {
			Contacts int64
			Lists    []struct{ Name string }
		}
		c.Do(http.MethodGet, ws+"/audience", nil).MustStatus(t, 200).JSON(t, &sum)
		// The boards' 48,213 and SMS's (X21) 338 new Acme Summit 2026
		// registrants, all Active.
		if sum.Contacts != 48213+338 {
			t.Fatalf("contacts %d, want 48,213 + 338", sum.Contacts)
		}
		var page struct {
			Total int64
			Items []struct {
				Email string
				Tier  string
			}
		}
		c.Do(http.MethodGet, ws+"/contacts?tier=active&pageSize=10", nil).MustStatus(t, 200).JSON(t, &page)
		if page.Total != 29412+338 {
			t.Fatalf("active %d, want 29,412 + 338", page.Total)
		}
		for _, it := range page.Items {
			if it.Tier != "active" {
				t.Fatalf("%s is %s in the active filter", it.Email, it.Tier)
			}
		}
		c.Do(http.MethodGet, ws+"/contacts?q=sam.rivera", nil).MustStatus(t, 200).JSON(t, &page)
		if page.Total != 1 || page.Items[0].Email != "sam.rivera@outlook.com" {
			t.Fatalf("search: %+v", page)
		}
	})

	t.Run("profile and activity resolve the path parameters", func(t *testing.T) {
		var page struct {
			Items []struct{ ID string }
		}
		c.Do(http.MethodGet, ws+"/contacts?q=sam.rivera", nil).MustStatus(t, 200).JSON(t, &page)
		id := page.Items[0].ID
		var p struct {
			Email string
			Tags  []string
		}
		c.Do(http.MethodGet, ws+"/contacts/"+id, nil).MustStatus(t, 200).JSON(t, &p)
		if p.Email != "sam.rivera@outlook.com" {
			t.Fatalf("profile %+v", p)
		}
		var act struct {
			Items []struct{ Type string }
			Total int
		}
		c.Do(http.MethodGet, ws+"/contacts/"+id+"/activity?limit=8", nil).MustStatus(t, 200).JSON(t, &act)
		if len(act.Items) != 8 || act.Total < 8 {
			t.Fatalf("activity: %d items of %d", len(act.Items), act.Total)
		}
		before := audits(t, env, "contact.update")
		c.Do(http.MethodPatch, ws+"/contacts/"+id, map[string]any{"tags": append(p.Tags, "vip")}).MustStatus(t, 200)
		if audits(t, env, "contact.update") != before+1 {
			t.Fatal("updating a contact wrote no audit entry")
		}
		viewer.Do(http.MethodPatch, ws+"/contacts/"+id, map[string]any{"tags": []string{}}).MustStatus(t, 403)
	})

	t.Run("list health", func(t *testing.T) {
		var h struct {
			Tiers []struct {
				Tier  string
				Count int64
			}
			Trend      []struct{ Day string }
			PolicyWeek int
			ThisWeek   struct{ Entering, Stopping int64 }
			Rows       []struct {
				Kind, Name string
				Contacts   int64
			}
		}
		c.Do(http.MethodGet, ws+"/audience/health", nil).MustStatus(t, 200).JSON(t, &h)
		want := map[string]int64{"active": 29412 + 338, "cooling": 9884, "at_risk": 5302, "sunset": 3615} // with SMS's (X21) new registrants
		for _, tc := range h.Tiers {
			if tc.Count != want[tc.Tier] {
				t.Errorf("%s: %d, want %d", tc.Tier, tc.Count, want[tc.Tier])
			}
		}
		if len(h.Trend) != 12 || h.PolicyWeek < 0 {
			t.Errorf("trend %d columns, policy week %d", len(h.Trend), h.PolicyWeek)
		}
		for _, w := range h.Trend {
			if len(w.Day) != len("2006-01-02") {
				t.Errorf("trend day %q is not a calendar day", w.Day)
			}
		}
		if h.ThisWeek.Entering != 1204 || h.ThisWeek.Stopping != 893 {
			t.Errorf("this week: %+v, want 1,204 entering and 893 stopping", h.ThisWeek)
		}
		var view bool
		for _, r := range h.Rows {
			view = view || (r.Kind == "view" && r.Contacts == 2118)
		}
		if !view {
			t.Errorf("the followed Mailchimp view is missing: %+v", h.Rows)
		}
		policy := map[string]any{"enabled": true, "activeDays": 30, "atRiskDays": 91, "sunsetDays": 181, "countMachineOpens": false,
			"reengage": true, "stopBroadcasts": true, "stopAfterDays": 30, "deleteAfter": false, "deleteAfterMonths": 12}
		viewer.Do(http.MethodPut, ws+"/audience/sunset-policy", policy).MustStatus(t, 403)
		policy["atRiskDays"] = 20
		c.Do(http.MethodPut, ws+"/audience/sunset-policy", policy).MustStatus(t, 422)
	})

	t.Run("suppression list", func(t *testing.T) {
		var l struct {
			Total, Matching int64
			Counts          struct{ Total, HardBounce, Complaint, Unsubscribe, Manual int64 }
			Items           []struct {
				ID, Address string
				Group       bool
				Rows        int64
			}
			Sync []struct {
				Name   string
				Local  bool
				Pulled int64
			}
			Sources []struct{ Value, Label string }
		}
		c.Do(http.MethodGet, ws+"/suppressions", nil).MustStatus(t, 200).JSON(t, &l)
		// The board's 1,874, billing@northwind.example's role entry
		// (Transactional) and the 287 invalid addresses the Sunset tier's
		// verification suppressed (X14).
		if l.Counts.Total != 2162 || l.Counts.HardBounce != 1402 || l.Counts.Complaint != 96 || l.Counts.Unsubscribe != 331 || l.Counts.Manual != 45 {
			t.Fatalf("counts %+v", l.Counts)
		}
		// The Brevo import is one row, after the newer suppressions.
		var group bool
		for page := 1; page <= 30 && !group; page++ {
			var pg struct {
				Items []struct {
					Group bool
					Rows  int64
				}
			}
			c.Do(http.MethodGet, ws+"/suppressions?pageSize=100&page="+strconv.Itoa(page), nil).MustStatus(t, 200).JSON(t, &pg)
			for _, it := range pg.Items {
				group = group || (it.Group && it.Rows == 1033)
			}
			if len(pg.Items) == 0 {
				break
			}
		}
		if !group {
			t.Error("the Brevo import rows are not grouped")
		}
		for _, s := range l.Sync {
			if s.Name == "Postal" && !s.Local {
				t.Error("Postal has no suppression API and should be local")
			}
		}
		for _, s := range l.Sources {
			if s.Value == "preference_center" || s.Value == "one_click" {
				t.Errorf("source filter lists %s; the board lists providers, imports and Manual", s.Label)
			}
		}

		viewer.Do(http.MethodPost, ws+"/suppressions", map[string]any{"emails": []string{"x@example.com"}}).MustStatus(t, 403)
		var add struct {
			Added, Already int
			Invalid        []string
		}
		c.Do(http.MethodPost, ws+"/suppressions", map[string]any{"emails": []string{"New.Person@Example.com", "not-an-address", "new.person@example.com"}, "note": "asked by phone"}).
			MustStatus(t, 200).JSON(t, &add)
		if add.Added != 1 || len(add.Invalid) != 1 {
			t.Fatalf("add: %+v", add)
		}
		if audits(t, env, "suppression.add") < 1 {
			t.Fatal("adding a suppression wrote no audit entry")
		}
		c.Do(http.MethodGet, ws+"/suppressions?q=new.person", nil).MustStatus(t, 200).JSON(t, &l)
		if len(l.Items) != 1 {
			t.Fatalf("search found %d", len(l.Items))
		}
		id := l.Items[0].ID
		c.Do(http.MethodPost, ws+"/suppressions/"+id+"/remove", map[string]any{"reason": "x"}).MustStatus(t, 422)
		c.Do(http.MethodPost, ws+"/suppressions/"+id+"/remove", map[string]any{"reason": "added by mistake"}).MustStatus(t, 204)
		if audits(t, env, "suppression.remove") != 1 {
			t.Fatal("removing a suppression wrote no audit entry")
		}
		viewer.Do(http.MethodGet, ws+"/suppressions/export", nil).MustStatus(t, 403)
		r := c.Do(http.MethodGet, ws+"/suppressions/export", nil).MustStatus(t, 200)
		if lines := strings.Count(string(r.Body), "\n"); lines != 2162+1 {
			t.Fatalf("export has %d lines, want header + 2,162", lines)
		}
	})

	t.Run("bulk actions report what failed", func(t *testing.T) {
		var page struct {
			Items []struct {
				ID, Email string
			}
		}
		c.Do(http.MethodGet, ws+"/contacts?pageSize=3", nil).MustStatus(t, 200).JSON(t, &page)
		ids := []string{}
		for _, it := range page.Items {
			ids = append(ids, it.ID)
		}
		var lists struct {
			Items []struct{ ID, Name string }
		}
		c.Do(http.MethodGet, ws+"/lists", nil).MustStatus(t, 200).JSON(t, &lists)
		var res struct{ Done, Failed int }
		c.Do(http.MethodPost, ws+"/contacts/bulk", map[string]any{"action": "add_tag", "ids": ids, "tag": "q4-review"}).MustStatus(t, 200).JSON(t, &res)
		if res.Done != len(ids) || res.Failed != 0 {
			t.Fatalf("bulk tag: %+v", res)
		}
		viewer.Do(http.MethodPost, ws+"/contacts/bulk", map[string]any{"action": "add_tag", "ids": ids, "tag": "x"}).MustStatus(t, 403)
	})

	t.Run("exports are logged and need the export permission", func(t *testing.T) {
		before := audits(t, env, "contacts.export")
		r := c.Do(http.MethodPost, ws+"/contacts/export", map[string]any{"allMatching": true, "filter": map[string]any{"tier": "sunset"}}).MustStatus(t, 200)
		if got := strings.Count(string(r.Body), "\n"); got != 3615+1 {
			t.Fatalf("export has %d lines, want header + 3,615", got)
		}
		if audits(t, env, "contacts.export") != before+1 {
			t.Fatal("export wrote no audit entry")
		}
		viewer.Do(http.MethodPost, ws+"/contacts/export", map[string]any{}).MustStatus(t, 403)
	})

	t.Run("another workspace's ids are not found", func(t *testing.T) {
		var page struct {
			Items []struct{ ID string }
		}
		c.Do(http.MethodGet, "/workspaces/acme-engineering-blog/contacts?pageSize=1", nil).MustStatus(t, 200).JSON(t, &page)
		if len(page.Items) == 0 {
			t.Fatal("the engineering blog has no contacts")
		}
		c.Do(http.MethodGet, fmt.Sprintf("%s/contacts/%s", ws, page.Items[0].ID), nil).MustStatus(t, 404)
	})
}
