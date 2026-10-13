package automations_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
)

// clock is a controllable "now" for the engine.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func engine(env *apptest.Env, clk *clock) *automations.Service {
	return &automations.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Log: env.App.Log, Now: clk.now}
}

func listID(t *testing.T, env *apptest.Env, name string) string {
	t.Helper()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM lists WHERE name = $1`, name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id.String()
}

// publishFlow creates an automation with g and publishes it.
func publishFlow(t *testing.T, c *apptest.Client, name string, g automations.AutomationGraph) string {
	t.Helper()
	var created struct{ ID string }
	c.Do("POST", wsPath, map[string]any{"name": name}).MustStatus(t, 201).JSON(t, &created)
	var saved struct {
		DraftVersion int
		Issues       []struct{ Code, Message string }
	}
	c.Do("PUT", wsPath+"/"+created.ID+"/draft", map[string]any{"graph": g, "revision": 0}).MustStatus(t, 200).JSON(t, &saved)
	if len(saved.Issues) > 0 {
		t.Fatalf("flow %q has issues: %+v", name, saved.Issues)
	}
	c.Do("POST", wsPath+"/"+created.ID+"/publish", map[string]any{"version": saved.DraftVersion, "migration": "finish"}).MustStatus(t, 200)
	return created.ID
}

func newEmail(t *testing.T, c *apptest.Client, name string) string {
	t.Helper()
	var e struct{ ID string }
	c.Do("POST", "/workspaces/acme-newsletter/automation-emails", map[string]any{"name": name, "subject": name}).MustStatus(t, 201).JSON(t, &e)
	return e.ID
}

func addContact(t *testing.T, env *apptest.Env, email string, attrs map[string]any) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	a, _ := json.Marshal(attrs)
	if attrs == nil {
		a = []byte(`{}`)
	}
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO contacts (id, workspace_id, email, first_name, timezone, attributes)
		SELECT $1, id, $2, 'Test', 'America/New_York', $3 FROM workspaces WHERE slug = 'acme-newsletter'`, id, email, a); err != nil {
		t.Fatal(err)
	}
	return id
}

func run(t *testing.T, env *apptest.Env, flow string, contact uuid.UUID) (node, status string, state map[string]any) {
	t.Helper()
	var raw []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT node_id, status, state FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, flow, contact).Scan(&node, &status, &raw); err != nil {
		t.Fatalf("run of %s: %v", contact, err)
	}
	_ = json.Unmarshal(raw, &state)
	return node, status, state
}

func node(id, typ string, d automations.AutomationNodeData) automations.AutomationNode {
	return automations.AutomationNode{ID: id, Type: typ, Data: d}
}

func link(src, dst, path string) automations.AutomationEdge {
	return automations.AutomationEdge{ID: src + "-" + dst, Source: src, Target: dst, Data: automations.AutomationEdgeData{Path: path}}
}

func TestEngineRunsAFlowEndToEnd(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	svc := engine(env, clk)
	emailID := newEmail(t, c, `Test · step 1 "Hello"`)
	product := listID(t, env, "Product Updates")
	flow := publishFlow(t, c, "Test flow", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, automations.AutomationNodeData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product, ConfirmedOnly: true}}),
			node("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: emailID, EmailVersion: 1, Subject: "Hello"}}),
			node("b", automations.NodeBranch, automations.AutomationNodeData{Condition: &automations.AutomationCondition{Kind: "opened", Step: "m", Within: &automations.AutomationDuration{Amount: 1, Unit: "hours"}, HumanOnly: true}}),
			node("g", automations.NodeGoal, automations.AutomationNodeData{}),
			node("cold", automations.NodeList, automations.AutomationNodeData{Action: &automations.AutomationListAction{Kind: "add_tag", Tag: "cold"}}),
		},
		Edges: []automations.AutomationEdge{link("t", "m", ""), link("m", "b", ""), link("b", "g", "yes"), link("b", "cold", "no")},
	})
	opener := addContact(t, env, "opener@example.com", nil)
	quiet := addContact(t, env, "quiet@example.com", nil)
	pending := addContact(t, env, "pending@example.com", nil)
	for _, m := range []struct {
		c      uuid.UUID
		status string
	}{{opener, "subscribed"}, {quiet, "subscribed"}, {pending, "pending"}} {
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at) VALUES ($1,$2,$3,'form:test',now())`, product, m.c, m.status); err != nil {
			t.Fatal(err)
		}
	}
	started, err := svc.Dispatch(ctx)
	if err != nil || started != 2 {
		t.Fatalf("dispatch started %d (double opt-in pending contacts wait): %v", started, err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	nodeID, status, state := run(t, env, flow, opener)
	if nodeID != "b" || status != "waiting" || state["wait"] != "branch" || state["message"] == nil {
		t.Fatalf("opener should wait at the branch: %s %s %v", nodeID, status, state)
	}
	var provider string
	if err := env.App.Pool.QueryRow(ctx, `SELECT p.adapter FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.id = $1 AND m.status = 'queued' AND m.stream = 'automation'`, state["message"]).Scan(&provider); err != nil || provider != "ses" {
		t.Fatalf("queued message on the Automations route: %q %v", provider, err)
	}
	// The opener opens (a human open); the scan wakes the branch.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, contact_id, type, occurred_at, machine)
		SELECT $1, workspace_id, id, contact_id, 'open', now(), false FROM messages WHERE id = $2`, uuid.Must(uuid.NewV7()), state["message"]); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if nodeID, status, _ = run(t, env, flow, opener); nodeID != "g" || status != "goal" {
		t.Fatalf("opener reached %s %s", nodeID, status)
	}
	if _, status, _ = run(t, env, flow, quiet); status != "waiting" {
		t.Fatalf("quiet contact should still wait, is %s", status)
	}
	// An hour later the window closes: the "No" path tags the contact.
	clk.advance(61 * time.Minute)
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if nodeID, status, _ = run(t, env, flow, quiet); nodeID != "cold" || status != "completed" {
		t.Fatalf("quiet contact at %s %s", nodeID, status)
	}
	var tags []string
	if err := env.App.Pool.QueryRow(ctx, `SELECT tags FROM contacts WHERE id = $1`, quiet).Scan(&tags); err != nil || !slices.Contains(tags, "cold") {
		t.Fatalf("tags %v %v", tags, err)
	}
	// Stats per step.
	var st struct {
		Entered int64
		Steps   []struct {
			NodeID            string
			Entered, Sent     int64
			GoalReached       int64
			Exited, Completed int64
			Paths             []struct {
				Path  string
				Count int64
			}
		}
	}
	c.Do("GET", wsPath+"/"+flow+"/stats", nil).MustStatus(t, 200).JSON(t, &st)
	got := map[string]string{}
	for _, s := range st.Steps {
		b, _ := json.Marshal(s)
		got[s.NodeID] = string(b)
	}
	if st.Entered != 2 || !strings.Contains(got["m"], `"Sent":2`) || !strings.Contains(got["b"], `{"Path":"no","Count":1},{"Path":"yes","Count":1}`) ||
		!strings.Contains(got["g"], `"GoalReached":1`) || !strings.Contains(got["cold"], `"Exited":1`) {
		t.Fatalf("stats %d %v", st.Entered, got)
	}
	// Re-entry is "once": joining again doesn't restart the flow.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE list_memberships SET status = 'unsubscribed' WHERE contact_id = $1; `, opener); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE list_memberships SET status = 'subscribed' WHERE contact_id = $1`, opener); err != nil {
		t.Fatal(err)
	}
	if started, err := svc.Dispatch(ctx); err != nil || started != 0 {
		t.Fatalf("re-entry started %d: %v", started, err)
	}
}

func TestEnginePausedFlowsHoldRuns(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	svc := engine(env, clk)
	// Every Welcome series run waits; move the clock past all their timers.
	clk.advance(15 * 24 * time.Hour)
	id := welcomeID(t, c)
	c.Do("POST", wsPath+"/"+id+"/pause", map[string]any{"reason": "testing"}).MustStatus(t, 200)
	n, err := svc.Tick(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var moved int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND status NOT IN ('waiting')`, id).Scan(&moved); err != nil || moved != 0 {
		t.Fatalf("paused flow moved %d runs (tick processed %d): %v", moved, n, err)
	}
	c.Do("POST", wsPath+"/"+id+"/resume", nil).MustStatus(t, 200)
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	var stillWaiting int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND status = 'waiting' AND next_at < $2`, id, clk.now()).Scan(&stillWaiting); err != nil || stillWaiting != 0 {
		t.Fatalf("%d runs still waiting on past timers: %v", stillWaiting, err)
	}
}

func TestWebhookStepsAreSignedLoggedAndNeverBlock(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	var mu sync.Mutex
	var bodies [][]byte
	var sigs []string
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, b)
		sigs = append(sigs, r.Header.Get(automations.SignatureHeader))
		code := status
		mu.Unlock()
		w.WriteHeader(code)
	}))
	defer srv.Close()
	flow := publishFlow(t, c, "Webhook flow", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, automations.AutomationNodeData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerAPIEvent, Event: "trial.started", Reentry: "always"}}),
			node("w", automations.NodeWebhook, automations.AutomationNodeData{Webhook: &automations.AutomationWebhook{Name: "crm", URL: "https://hooks.example.com/crm", Method: "POST"}}),
			node("x", automations.NodeExit, automations.AutomationNodeData{}),
		},
		Edges: []automations.AutomationEdge{link("t", "w", ""), link("w", "x", "")},
	})
	const secret = "whsec_test_0123456789abcdef"
	c.Do("PUT", wsPath+"/"+flow+"/steps/w/secret", map[string]any{"secret": secret}).MustStatus(t, 204)
	// Point the live step at the test server (validation refuses loopback URLs).
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automation_versions SET graph = replace(graph::text, 'https://hooks.example.com/crm', $2)::jsonb WHERE automation_id = $1`, flow, srv.URL+"/crm"); err != nil {
		t.Fatal(err)
	}
	contact := addContact(t, env, "webhook@example.com", map[string]any{"plan": "pro"})
	trusted := engine(env, clk)
	trusted.HTTP = srv.Client()
	call := func(svc *automations.Service) {
		t.Helper()
		clk.advance(time.Minute)
		c.Do("POST", wsPath+"/events", map[string]any{"event": "trial.started", "contactId": contact.String()}).MustStatus(t, 202)
		if _, err := svc.Dispatch(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Tick(ctx); err != nil {
			t.Fatal(err)
		}
		var runID uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM automation_runs WHERE automation_id = $1 AND status = 'waiting' AND node_id = 'w'`, flow).Scan(&runID); err != nil {
			t.Fatalf("run waiting at the webhook: %v", err)
		}
		if err := svc.RunWebhook(ctx, automations.WebhookArgs{RunID: runID, AutomationID: uuid.MustParse(flow), Version: 1, NodeID: "w"}, 1, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Tick(ctx); err != nil {
			t.Fatal(err)
		}
		var st string
		if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM automation_runs WHERE id = $1`, runID).Scan(&st); err != nil || st != "completed" {
			t.Fatalf("run after the webhook: %s %v", st, err)
		}
	}
	call(trusted)
	mu.Lock()
	if len(bodies) != 1 {
		t.Fatalf("calls %d", len(bodies))
	}
	var payload struct {
		Type    string
		Contact struct {
			Email  string
			Fields map[string]any
		}
	}
	_ = json.Unmarshal(bodies[0], &payload)
	ts, v1, _ := strings.Cut(strings.TrimPrefix(sigs[0], "t="), ",v1=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(bodies[0])
	if payload.Type != "automation.step" || payload.Contact.Email != "webhook@example.com" || payload.Contact.Fields["plan"] != "pro" || v1 != hex.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("payload %s signature %q", bodies[0], sigs[0])
	}
	status = http.StatusNotFound
	mu.Unlock()
	// A failing endpoint is logged and alerts, and the contact moves on.
	call(trusted)
	var calls struct {
		Calls []struct {
			StatusCode int
			Error      string
		}
	}
	c.Do("GET", wsPath+"/"+flow+"/steps/w/calls", nil).MustStatus(t, 200).JSON(t, &calls)
	if len(calls.Calls) != 2 || calls.Calls[0].StatusCode != 404 || calls.Calls[1].StatusCode != 200 {
		t.Fatalf("calls %+v", calls.Calls)
	}
	var l list
	c.Do("GET", wsPath, nil).MustStatus(t, 200).JSON(t, &l)
	found := false
	for _, a := range l.Alerts {
		found = found || (a.Kind == "step_failing" && strings.Contains(a.Title, `"Webhook flow" webhook crm returns 404`))
	}
	if !found {
		t.Fatalf("alerts %+v", l.Alerts)
	}
	// The default client refuses loopback: the call is blocked, never sent.
	mu.Lock()
	before := len(bodies)
	mu.Unlock()
	call(engine(env, clk))
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != before {
		t.Fatalf("a loopback URL was called")
	}
	c.Do("GET", wsPath+"/"+flow+"/steps/w/calls", nil).MustStatus(t, 200).JSON(t, &calls)
	if !strings.HasPrefix(calls.Calls[0].Error, "blocked:") {
		t.Fatalf("blocked call %+v", calls.Calls[0])
	}
}

func TestGoalsAreDetectedAnywhereInTheFlow(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	var contact uuid.UUID
	var flow string
	if err := env.App.Pool.QueryRow(ctx, `SELECT r.contact_id, r.automation_id::text FROM automation_runs r JOIN automations a ON a.id = r.automation_id
		WHERE a.name = 'Onboarding · new workspace' AND r.status = 'waiting' LIMIT 1`).Scan(&contact, &flow); err != nil {
		t.Fatal(err)
	}
	c.Do("POST", wsPath+"/events", map[string]any{"event": "teammate.invited", "contactId": contact.String()}).MustStatus(t, 202)
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if node, status, _ := run(t, env, flow, contact); node != "n14" || status != "goal" {
		t.Fatalf("goal not reached: %s %s", node, status)
	}
}

func TestDateFieldTriggers(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Now().In(loc)
	clk := &clock{t: time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)}
	svc := engine(env, clk)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO custom_fields (id, workspace_id, key, label, type) SELECT $1, id, 'trial_ends_at', 'Trial ends', 'date' FROM workspaces WHERE slug = 'acme-newsletter'`, uuid.Must(uuid.NewV7())); err != nil {
		t.Fatal(err)
	}
	emailID := newEmail(t, c, `Trial "Ends in 3 days"`)
	flow := publishFlow(t, c, "Trial ending", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, automations.AutomationNodeData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerDateField, Field: "trial_ends_at", OffsetDays: -3, At: "09:00"}}),
			node("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: emailID, EmailVersion: 1, Subject: "Your trial ends in 3 days"}}),
		},
		Edges: []automations.AutomationEdge{link("t", "m", "")},
	})
	due := addContact(t, env, "due@example.com", map[string]any{"trial_ends_at": clk.now().AddDate(0, 0, 3).Format("2006-01-02")})
	notDue := addContact(t, env, "later@example.com", map[string]any{"trial_ends_at": clk.now().AddDate(0, 0, 5).Format("2006-01-02")})
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil { // twice: one event per contact per day
		t.Fatal(err)
	}
	if started, err := svc.Dispatch(ctx); err != nil || started != 1 {
		t.Fatalf("started %d: %v", started, err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if _, status, _ := run(t, env, flow, due); status != "completed" {
		t.Fatalf("due contact: %s", status)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM automation_runs WHERE contact_id = $1`, notDue).Scan(&n)
	if n != 0 {
		t.Fatalf("contact whose date isn't due entered the flow")
	}
}
