package campaigns_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

// Review R3.U2 (PR-13): a campaign can be written before any provider is
// connected. A new draft, the composer, saving, the preview and Send
// Readiness all work in a workspace without one; only the send is locked,
// by the route check, whose fix leads to routing.
func TestWriteBeforeAProvider(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "founder@example.com", Name: "Founder"})
	w := env.CreateWorkspace("Fresh Letter", owner.ID)
	c := env.Client()
	c.SignIn("founder@example.com", "")
	base := "/workspaces/" + w.Slug
	var providers int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM providers WHERE workspace_id = $1`, w.ID).Scan(&providers); err != nil || providers != 0 {
		t.Fatalf("providers: %d %v", providers, err)
	}

	// New campaign isn't locked, and the composer opens with no route.
	var d detail
	c.Do("POST", base+"/campaigns", map[string]any{}).MustStatus(t, 201).JSON(t, &d)
	if d.Status != "draft" || d.Route.Source != "none" || len(d.Route.Shares) != 0 || !d.Access.Editable || !d.Access.CanEdit {
		t.Fatalf("a draft without a provider: %+v", d)
	}
	c.Do("GET", base+"/composer", nil).MustStatus(t, 200)

	// Writing: saving a version and previewing it.
	mjml := `<mjml><mj-body><mj-section><mj-column><mj-text>Hi {{ first_name | default: "there" }}, our first issue.</mj-text></mj-column></mj-section></mj-body></mjml>`
	var saved struct{ Version int }
	c.Do("PUT", base+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": d.Version, "subject": "Our first issue", "mjml": mjml}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Version != d.Version+1 {
		t.Fatalf("saved version %d after %d", saved.Version, d.Version)
	}
	var got detail
	c.Do("GET", base+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if got.Subject != "Our first issue" || !strings.Contains(got.MJML, "our first issue") {
		t.Fatalf("the draft as saved: %+v", got)
	}
	c.Do("POST", base+"/campaigns/"+d.ID+"/preview", map[string]any{"sample": "no_first_name"}).MustStatus(t, 200)

	// Send Readiness runs, and its route check is what locks the send.
	run := readinesstest.Run(t, env, c, w.Slug, d.ID)
	var route *readiness.ReadinessCheck
	for i, ch := range run.Checks {
		if ch.Key == "routing.health" {
			route = &run.Checks[i]
		}
	}
	if run.Verdict != readiness.VerdictBlocked || route == nil || route.Status != "block" || route.Title != "Broadcasts has no sending provider" {
		t.Fatalf("readiness without a provider: %s, route %+v", run.Verdict, route)
	}
	if len(route.Fixes) != 1 || route.Fixes[0].Label != "Set up routing" || !strings.HasSuffix(route.Fixes[0].Href, "/providers/routing") {
		t.Fatalf("the route check's fix: %+v", route.Fixes)
	}
	p := c.Do("POST", base+"/campaigns/"+d.ID+"/send", map[string]any{"runId": run.ID}).MustStatus(t, 409).Problem(t)
	if p["code"] != "readiness_blocked" {
		t.Fatalf("send without a provider: %v", p)
	}
	var status string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM campaigns WHERE id = $1`, d.ID).Scan(&status); err != nil || status != "draft" {
		t.Fatalf("after the refused send: %s %v", status, err)
	}
}
