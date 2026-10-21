package automations_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
)

const wsPath = "/workspaces/acme-newsletter/automations"

type listItem struct {
	ID                 string
	Name               string
	Description        string
	Status             string
	LiveVersion        *int
	DraftVersion       int
	Trigger            string
	Steps              int
	Entered            *int64
	InFlow             *int64
	Completed          *float64
	HumanOpens         *float64
	PausedBy           string
	PauseReason        string
	UnpublishedChanges bool
	Goal               *struct {
		Label   string
		Percent *float64
	}
}

type list struct {
	Items  []listItem
	Counts struct{ All, Live, Paused, Drafts int64 }
	Route  struct {
		Label   string
		Healthy bool
	}
	Alerts []struct {
		Kind, Title, Detail string
	}
}

func signedIn(t *testing.T, email string) (*apptest.Env, *apptest.Client) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo(email)
	return env, c
}

func findItem(t *testing.T, l list, name string) listItem {
	t.Helper()
	for _, it := range l.Items {
		if it.Name == name {
			return it
		}
	}
	t.Fatalf("no automation %q in %+v", name, l.Items)
	return listItem{}
}

func TestListAndSummaryMatchTheBoard(t *testing.T) {
	_, c := signedIn(t, "linh@acme.io")
	var l list
	c.Do("GET", wsPath, nil).MustStatus(t, 200).JSON(t, &l)
	// Contact scoring (X13) adds the live "Sales-ready handoff", and SMS
	// (X21, brief §4) the live "Acme Summit reminders".
	if l.Counts.All != 7 || l.Counts.Live != 5 || l.Counts.Paused != 1 || l.Counts.Drafts != 1 {
		t.Fatalf("counts %+v", l.Counts)
	}
	var got []string
	for _, it := range l.Items {
		if it.Name != "Sales-ready handoff" && it.Name != "Acme Summit reminders" {
			got = append(got, it.Name)
		}
	}
	if strings.Join(got, "|") != "Welcome series|Onboarding · new workspace|Re-engagement|Re-permission · Mailchimp import|Beta invite" {
		t.Fatalf("order %v", got)
	}
	w := findItem(t, l, "Welcome series")
	if w.Status != "live" || *w.LiveVersion != 3 || w.DraftVersion != 4 || w.Steps != 11 || *w.Entered != 1284 || *w.InFlow != 612 ||
		w.Trigger != "Joined list · Weekly Digest" || !w.UnpublishedChanges || w.Goal == nil || w.Goal.Label != "Clicked docs" || *w.Goal.Percent != 24.3 || *w.Completed != 62.0 {
		t.Fatalf("welcome row %+v goal %+v completed %v", w, w.Goal, *w.Completed)
	}
	o := findItem(t, l, "Onboarding · new workspace")
	if o.Steps != 14 || *o.Entered != 906 || *o.InFlow != 488 || o.Trigger != "API event · workspace.created" || o.Goal.Label != "Invited a teammate" || *o.Goal.Percent != 38.0 {
		t.Fatalf("onboarding row %+v %+v", o, o.Goal)
	}
	r := findItem(t, l, "Re-engagement")
	if *r.Entered != 4811 || *r.InFlow != 832 || *r.Goal.Percent != 18.4 || *r.Completed != 83.0 || *r.HumanOpens != 14.2 {
		t.Fatalf("re-engagement row %+v completed %v opens %v", r, *r.Completed, *r.HumanOpens)
	}
	d := findItem(t, l, "Re-permission · Mailchimp import")
	if d.Status != "draft" || d.Entered != nil || d.InFlow != nil || d.Goal.Label != "Confirmed opt-in" || d.Goal.Percent != nil || d.Trigger != "Tag added · mailchimp-pending" {
		t.Fatalf("draft row %+v", d)
	}
	b := findItem(t, l, "Beta invite")
	if b.Status != "paused" || b.PausedBy != "Marcus Chen" || b.PauseReason != "beta is full" || *b.InFlow != 0 || *b.Entered != 214 || b.Trigger != "Form submitted · beta-apply" || *b.HumanOpens != 63.1 {
		t.Fatalf("paused row %+v", b)
	}
	if l.Route.Label != "SES, failover Postmark" || !l.Route.Healthy {
		t.Fatalf("route %+v", l.Route)
	}
	// The Slack webhook is failing, but the draft removes it: no card.
	if len(l.Alerts) != 0 {
		t.Fatalf("alerts %+v", l.Alerts)
	}
	c.Do("GET", wsPath+"?status=drafts", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 1 || l.Items[0].Status != "draft" {
		t.Fatalf("drafts filter: %+v", l.Items)
	}
	c.Do("GET", wsPath+"?status=nope", nil).MustStatus(t, 422)

	var sum struct {
		InFlow, LiveCount, Sent, Goals int64
		GoalsBy                        []struct {
			Name  string
			Count int64
		}
		SentBy []struct {
			Short    string
			Sent     int64
			Fallback bool
		}
	}
	c.Do("GET", wsPath+"/summary", nil).MustStatus(t, 200).JSON(t, &sum)
	if sum.InFlow != 1932 || sum.LiveCount != 5 || sum.Goals != 604 || len(sum.GoalsBy) != 3 || sum.GoalsBy[0].Name != "welcome" || sum.GoalsBy[0].Count != 312 {
		t.Fatalf("summary %+v", sum)
	}
	if len(sum.SentBy) == 0 || sum.SentBy[0].Short != "SES" {
		t.Fatalf("sent by %+v", sum.SentBy)
	}
}

// The AutomationsStates › Error board: SES down, the fallback's token
// rejected, Onboarding's emails held; and the Slack webhook card once the
// draft no longer removes the step.
func TestAlertsForHeldSendingAndFailingSteps(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	for _, q := range []string{
		`UPDATE providers SET status = 'down', health = health || jsonb_build_object('down', jsonb_build_object('since', to_jsonb(now() - interval '1 hour'), 'reason', 'outage')) WHERE adapter = 'ses'`,
		`UPDATE providers SET status = 'down', health = health || jsonb_build_object('down', jsonb_build_object('since', to_jsonb(now() - interval '1 hour'), 'reason', 'auth'), 'lastError', jsonb_build_object('status', 401)) WHERE adapter = 'postmark'`,
		`INSERT INTO automation_alerts (id, automation_id, kind, node_id, detail, failures, first_at, last_at)
		 SELECT gen_random_uuid(), id, 'sending_held', '', '{"queued": 23}', 23, now(), now() FROM automations WHERE name = 'Onboarding · new workspace'`,
		`UPDATE automations SET draft_version = live_version WHERE name = 'Welcome series'`,
	} {
		if _, err := env.App.Pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	var l struct {
		Route  struct{ Healthy bool }
		Alerts []struct {
			Kind, Title, Detail string
			Actions             []struct{ Label, Kind, NodeID string }
		}
	}
	c.Do("GET", wsPath, nil).MustStatus(t, 200).JSON(t, &l)
	if l.Route.Healthy || len(l.Alerts) != 2 {
		t.Fatalf("route healthy %v, alerts %+v", l.Route.Healthy, l.Alerts)
	}
	held, failing := l.Alerts[0], l.Alerts[1]
	if held.Kind != "sending_held" || held.Title != `"Onboarding" paused: no healthy provider on the Automations route` ||
		!strings.HasPrefix(held.Detail, "SES has been Down since ") ||
		!strings.Contains(held.Detail, "and Postmark, the fallback, returned 401: its API key or token was rejected. 23 emails are queued") {
		t.Fatalf("held alert %+v", held)
	}
	if len(held.Actions) != 2 || held.Actions[0].Label != "Fix Postmark token" || held.Actions[1].Label != "Add Postal as second fallback" {
		t.Fatalf("held actions %+v", held.Actions)
	}
	if failing.Kind != "step_failing" || failing.Title != `"Welcome series" webhook notify-slack returns 404` ||
		!strings.HasPrefix(failing.Detail, "1,284 calls failed since ") || failing.Actions[0].Label != "Open the step" || failing.Actions[0].NodeID != "w1" {
		t.Fatalf("failing alert %+v", failing)
	}
}

type detail struct {
	ID               string
	Name             string
	Status           string
	LiveVersion      *int
	DraftVersion     int
	Revision         int64
	Changes          int
	InFlow           int64
	LiveNodeIDs      []string
	PublishedNodeIDs []string
	Issues           []struct{ Code, NodeID string }
	Graph            automations.AutomationGraph
}

func welcomeID(t *testing.T, c *apptest.Client) string {
	t.Helper()
	var l list
	c.Do("GET", wsPath, nil).MustStatus(t, 200).JSON(t, &l)
	return findItem(t, l, "Welcome series").ID
}

func TestDetailStatsAndPublishPreview(t *testing.T) {
	_, c := signedIn(t, "linh@acme.io")
	id := welcomeID(t, c)
	var d detail
	c.Do("GET", wsPath+"/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	if d.DraftVersion != 4 || *d.LiveVersion != 3 || d.Changes != 3 || len(d.Issues) != 0 || len(d.Graph.Nodes) != 13 || len(d.LiveNodeIDs) != 11 || d.InFlow != 612 {
		t.Fatalf("detail %+v", d)
	}
	for _, n := range []string{"n7", "n8", "n9"} {
		for _, p := range d.PublishedNodeIDs {
			if p == n {
				t.Fatalf("%s is new in v4 and was never published", n)
			}
		}
	}
	var st struct {
		Entered int64
		Steps   []struct {
			NodeID                          string
			Entered, Sent, OpenedHuman      int64
			Delivered, Clicked, GoalReached int64
			Paths                           []struct {
				Path  string
				Count int64
			}
			Providers []struct {
				Short    string
				Sent     int64
				Fallback bool
			}
		}
	}
	c.Do("GET", wsPath+"/"+id+"/stats?days=30", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Entered != 1284 {
		t.Fatalf("entered %d", st.Entered)
	}
	for _, s := range st.Steps {
		switch s.NodeID {
		case "n3":
			if s.Sent != 1276 || len(s.Providers) != 2 || s.Providers[0].Short != "SES" || s.Providers[0].Sent != 1266 || !s.Providers[1].Fallback || s.Providers[1].Sent != 10 {
				t.Fatalf("step 1 %+v", s)
			}
		case "n4":
			if len(s.Paths) != 2 || s.Paths[0].Path != "no" || s.Paths[0].Count != 498 || s.Paths[1].Count != 778 {
				t.Fatalf("branch %+v", s.Paths)
			}
		case "n10":
			if s.GoalReached != 312 {
				t.Fatalf("goal %+v", s)
			}
		}
	}

	var p struct {
		FromVersion    *int
		ToVersion      int
		Changes        int
		InFlow         int64
		FinishEstimate string
		MoveImpact     string
		CanPublish     bool
		Items          []struct{ Kind, Title, Detail string }
		Checks         []struct{ Status, Label string }
	}
	c.Do("GET", wsPath+"/"+id+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &p)
	if *p.FromVersion != 3 || p.ToVersion != 4 || p.Changes != 3 || p.InFlow != 612 || !p.CanPublish {
		t.Fatalf("preview %+v", p)
	}
	if p.Items[0].Title != `A/B split after "Step 2", 50 / 50` || p.Items[2].Title != `Webhook "notify-slack" after the trigger` ||
		!strings.Contains(p.Items[2].Detail, "was returning 404 since") || !strings.Contains(p.Items[2].Detail, "1,284 failed calls") ||
		p.Items[3].Title != `Trigger, step 1, branch, "No" path, goal and exit` {
		t.Fatalf("items %+v", p.Items)
	}
	if p.MoveImpact != "612 contacts waiting before step 2 would get the 2-day delay; 0 contacts are affected by the removed webhook." {
		t.Fatalf("move impact %q", p.MoveImpact)
	}
	want := []string{"Automations route healthy · SES", "5 emails pass content checks", "No step references a deleted list or tag", "Branch has both paths connected"}
	if len(p.Checks) < 4 {
		t.Fatalf("checks %+v", p.Checks)
	}
	for i, w := range want {
		if p.Checks[i].Label != w || p.Checks[i].Status != "ok" {
			t.Fatalf("check %d: %+v, want %q", i, p.Checks[i], w)
		}
	}
}

func TestPublishMovesContactsAndAudits(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	id := welcomeID(t, c)
	var res struct{ Status string }
	// Stale version is refused.
	c.Do("POST", wsPath+"/"+id+"/publish", map[string]any{"version": 3, "migration": "move"}).MustStatus(t, 409)
	c.Do("POST", wsPath+"/"+id+"/publish", map[string]any{"version": 4, "migration": "move"}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "published" {
		t.Fatalf("publish %+v", res)
	}
	var d detail
	c.Do("GET", wsPath+"/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	if *d.LiveVersion != 4 || d.DraftVersion != 4 || d.Changes != 0 || d.Status != "live" {
		t.Fatalf("after publish %+v", d)
	}
	ctx := context.Background()
	var onOld, atDelay int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE version <> 4), count(*) FILTER (WHERE node_id = 'n5' AND next_at > node_entered_at + interval '47 hours')
		FROM automation_runs r JOIN automations a ON a.id = r.automation_id WHERE a.name = 'Welcome series' AND r.status IN ('active','waiting')`).Scan(&onOld, &atDelay); err != nil {
		t.Fatal(err)
	}
	if onOld != 0 || atDelay != 440 {
		t.Fatalf("moved runs: %d still on old versions, %d waiting 2 days at step 2's delay", onOld, atDelay)
	}
	var versions struct {
		Versions []struct {
			Version         int
			Status, Summary string
		}
	}
	c.Do("GET", wsPath+"/"+id+"/versions", nil).MustStatus(t, 200).JSON(t, &versions)
	if versions.Versions[0].Status != "live" || versions.Versions[1].Status != "retired" || !strings.Contains(versions.Versions[0].Summary, "added A/B split") {
		t.Fatalf("versions %+v", versions.Versions)
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'automation.publish' AND target_id = $1`, id).Scan(&n); err != nil || n != 1 {
		t.Fatalf("publish audit entries: %d %v", n, err)
	}
	// The failing webhook went away with v4.
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM automation_alerts WHERE resolved_at IS NULL AND node_id = 'w1'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("open webhook alerts: %d %v", n, err)
	}
	// Nothing left to publish.
	c.Do("POST", wsPath+"/"+id+"/publish", map[string]any{"version": 4, "migration": "finish"}).MustStatus(t, 409)
}

func TestPublishFinishKeepsOldVersion(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	id := welcomeID(t, c)
	c.Do("POST", wsPath+"/"+id+"/publish", map[string]any{"version": 4, "migration": "finish"}).MustStatus(t, 200)
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND version = 3 AND status IN ('active','waiting')`, id).Scan(&n); err != nil || n != 612 {
		t.Fatalf("contacts finishing on v3: %d %v", n, err)
	}
	var versions struct{ Versions []struct{ Status string } }
	c.Do("GET", wsPath+"/"+id+"/versions", nil).MustStatus(t, 200).JSON(t, &versions)
	if versions.Versions[1].Status != "retiring" {
		t.Fatalf("v3 should be retiring: %+v", versions.Versions)
	}
}

func TestCreateEditValidateAndConflicts(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	var created struct{ ID string }
	c.Do("POST", wsPath, map[string]any{"template": "welcome-series"}).MustStatus(t, 201).JSON(t, &created)
	var d detail
	c.Do("GET", wsPath+"/"+created.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Name != "Welcome series" || d.Status != "draft" || len(d.Graph.Nodes) != 7 || len(d.Issues) != 0 {
		t.Fatalf("from template: %+v", d)
	}
	// Remove the "No" path: saved, but reported.
	g := d.Graph
	var edges []automations.AutomationEdge
	for _, e := range g.Edges {
		if e.Data.Path != automations.PathNo {
			edges = append(edges, e)
		}
	}
	g.Edges = edges
	var saved struct {
		Revision int64
		Issues   []struct{ Code string }
	}
	c.Do("PUT", wsPath+"/"+created.ID+"/draft", map[string]any{"graph": g, "revision": d.Revision}).MustStatus(t, 200).JSON(t, &saved)
	codes := map[string]bool{}
	for _, i := range saved.Issues {
		codes[i.Code] = true
	}
	if !codes["missing_path"] || !codes["orphan"] {
		t.Fatalf("issues %+v", saved.Issues)
	}
	// Someone else's revision.
	c.Do("PUT", wsPath+"/"+created.ID+"/draft", map[string]any{"graph": g, "revision": d.Revision}).MustStatus(t, 409)
	// Publishing an invalid draft is refused.
	c.Do("POST", wsPath+"/"+created.ID+"/publish", map[string]any{"version": 1, "migration": "finish"}).MustStatus(t, 422)
	// Restore the template's graph and publish.
	c.Do("PUT", wsPath+"/"+created.ID+"/draft", map[string]any{"graph": d.Graph, "revision": saved.Revision}).MustStatus(t, 200)
	var res struct{ Status string }
	c.Do("POST", wsPath+"/"+created.ID+"/publish", map[string]any{"version": 1, "migration": "finish"}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "published" {
		t.Fatalf("first publish %+v", res)
	}
	// Rename, duplicate, delete.
	c.Do("PATCH", wsPath+"/"+created.ID, map[string]any{"name": "Welcome · spring", "description": "3 emails"}).MustStatus(t, 200)
	var dup struct{ ID string }
	c.Do("POST", wsPath+"/"+created.ID+"/duplicate", nil).MustStatus(t, 201).JSON(t, &dup)
	c.Do("GET", wsPath+"/"+dup.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Name != "Copy of Welcome · spring" || d.Status != "draft" {
		t.Fatalf("duplicate %+v", d)
	}
	c.Do("DELETE", wsPath+"/"+dup.ID, nil).MustStatus(t, 204)
	c.Do("GET", wsPath+"/"+dup.ID, nil).MustStatus(t, 404)
	c.Do("GET", wsPath+"/not-a-uuid", nil).MustStatus(t, 404)
	var actions []string
	rows, err := env.App.Pool.Query(context.Background(), `SELECT action FROM audit_events WHERE action LIKE 'automation.%' ORDER BY occurred_at`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		actions = append(actions, a)
	}
	rows.Close()
	for _, want := range []string{"automation.create", "automation.draft_update", "automation.publish", "automation.update", "automation.duplicate", "automation.delete"} {
		found := false
		for _, a := range actions {
			found = found || a == want
		}
		if !found {
			t.Errorf("no %s audit entry in %v", want, actions)
		}
	}
}

func TestPermissions(t *testing.T) {
	env, viewer := signedIn(t, "aisha@acme.io")
	viewer.Do("GET", wsPath, nil).MustStatus(t, 200)
	viewer.Do("POST", wsPath, map[string]any{"template": "onboarding"}).MustStatus(t, 403)
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	id := welcomeID(t, owner)
	r := viewer.Do("POST", wsPath+"/"+id+"/publish", map[string]any{"version": 4, "migration": "finish"}).MustStatus(t, 403)
	if p := r.Problem(t); p["permission"] != "automations:publish" {
		t.Fatalf("problem %+v", p)
	}
	viewer.Do("POST", wsPath+"/"+id+"/pause", map[string]any{}).MustStatus(t, 403)
	viewer.Do("PUT", wsPath+"/"+id+"/draft", map[string]any{"graph": map[string]any{"nodes": []any{}, "edges": []any{}}, "revision": 0}).MustStatus(t, 403)
	viewer.Do("POST", wsPath+"/events", map[string]any{"event": "workspace.created", "email": "maya.chen0@example.com"}).MustStatus(t, 403)
	// The Content Writer can view automations but not edit them.
	writer := env.Client()
	writer.SignInDemo("daniel@acme.io")
	writer.Do("GET", wsPath+"/"+id, nil).MustStatus(t, 200)
	writer.Do("PATCH", wsPath+"/"+id, map[string]any{"name": "x", "description": ""}).MustStatus(t, 403)
}

func TestPauseResume(t *testing.T) {
	_, c := signedIn(t, "linh@acme.io")
	id := welcomeID(t, c)
	var d struct {
		Status, PausedBy, PauseReason string
	}
	c.Do("POST", wsPath+"/"+id+"/pause", map[string]any{"reason": "fixing the Slack hook"}).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "paused" || d.PausedBy != "Linh Nguyen" || d.PauseReason != "fixing the Slack hook" {
		t.Fatalf("paused %+v", d)
	}
	c.Do("POST", wsPath+"/"+id+"/pause", map[string]any{}).MustStatus(t, 409)
	var r struct{ Status, PausedBy, PauseReason string }
	c.Do("POST", wsPath+"/"+id+"/resume", nil).MustStatus(t, 200).JSON(t, &r)
	d = r
	if d.Status != "live" || d.PausedBy != "" {
		t.Fatalf("resumed %+v", d)
	}
}

func TestWebhookSecretIsWriteOnly(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	var l list
	c.Do("GET", wsPath, nil).MustStatus(t, 200).JSON(t, &l)
	id := findItem(t, l, "Onboarding · new workspace").ID
	const secret = "whsec_3kF9ab0c1d2e3f4g5h6"
	c.Do("PUT", wsPath+"/"+id+"/steps/n5/secret", map[string]any{"secret": "short"}).MustStatus(t, 422)
	c.Do("PUT", wsPath+"/"+id+"/steps/n5/secret", map[string]any{"secret": secret}).MustStatus(t, 204)
	c.Do("PUT", wsPath+"/"+id+"/steps/n4/secret", map[string]any{"secret": secret}).MustStatus(t, 404)
	r := c.Do("GET", wsPath+"/"+id, nil).MustStatus(t, 200)
	if strings.Contains(string(r.Body), secret) || !strings.Contains(string(r.Body), `"hasSecret":true`) {
		t.Fatalf("secret leaked or not marked: %s", r.Body)
	}
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'automation.step_secret_set' AND NOT (coalesce(metadata::text,'') LIKE '%whsec%')`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("secret audit: %d %v", n, err)
	}
}

func TestSendEvent(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	var email string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT email FROM contacts ORDER BY created_at LIMIT 1`).Scan(&email); err != nil {
		t.Fatal(err)
	}
	var res struct {
		Accepted bool
		Flows    int
	}
	c.Do("POST", wsPath+"/events", map[string]any{"event": "workspace.created", "email": email, "properties": map[string]any{"plan": "pro"}}).MustStatus(t, 202).JSON(t, &res)
	if !res.Accepted || res.Flows != 1 {
		t.Fatalf("event %+v", res)
	}
	c.Do("POST", wsPath+"/events", map[string]any{"event": "workspace.created", "email": "nobody@example.net"}).MustStatus(t, 404)
	c.Do("POST", wsPath+"/events", map[string]any{"event": "bad event!", "email": email}).MustStatus(t, 422)
}

func TestTemplatesAndOptions(t *testing.T) {
	_, c := signedIn(t, "aisha@acme.io")
	var tpl struct {
		Templates []struct {
			Key   string
			Steps int
		}
	}
	c.Do("GET", wsPath+"/templates", nil).MustStatus(t, 200).JSON(t, &tpl)
	if len(tpl.Templates) != 3 || tpl.Templates[0].Key != "welcome-series" || tpl.Templates[0].Steps != 7 {
		t.Fatalf("templates %+v", tpl)
	}
	var opts struct {
		Lists     []struct{ Name string }
		Emails    []struct{ Name string }
		Providers []struct{ Short string }
	}
	c.Do("GET", wsPath+"/options", nil).MustStatus(t, 200).JSON(t, &opts)
	// The three lists of the board, and SMS's (X21, brief §4) Acme Summit 2026.
	if len(opts.Lists) != 4 || len(opts.Emails) < 5 || len(opts.Providers) != 3 {
		t.Fatalf("options %+v", opts)
	}
}
