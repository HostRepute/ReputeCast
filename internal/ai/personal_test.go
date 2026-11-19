package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/jobs"
)

// Personal keys (assistant brief D4): a member's own accounts answer only
// them, in their own monthly cap, while managers allow them.

// personal is a registry workspace with a member (an Editor: ai:use, no
// integrations:manage) signed in on mc.
type personal struct {
	*registry
	member uuid.UUID
	mc     *apptest.Client
}

func newPersonal(t *testing.T, first *modelServer, connect map[string]any) *personal {
	r := newRegistry(t, first, connect)
	u := r.env.CreateUser(apptest.UserOpts{Email: "member@registry.test", Name: "Registry Member"})
	r.env.AddMember(r.ws, u.ID, authz.RoleEditor)
	mc := r.env.Client()
	mc.SignInDemo(u.Email)
	return &personal{registry: r, member: u.ID, mc: mc}
}

// enable turns personal keys on with a cap, as a manager.
func (p *personal) enable(capUSD float64) {
	p.t.Helper()
	p.c.Do(http.MethodPut, p.base+"/personal-keys", map[string]any{"enabled": true, "capUsd": capUSD}).MustStatus(p.t, 200)
}

// addOwn adds the member's own account on srv with one model.
func (p *personal) addOwn(url, key string, input, output float64) (account, model uuid.UUID) {
	p.t.Helper()
	var out ai.AIPersonalSettings
	p.mc.Do(http.MethodPost, p.base+"/personal/accounts", map[string]any{"provider": "compatible", "baseUrl": url, "apiKey": key, "model": "own-model",
		"inputPrice": input, "outputPrice": output}).MustStatus(p.t, 200).JSON(p.t, &out)
	if len(out.Accounts) != 1 || len(out.Accounts[0].Models) != 1 || out.Accounts[0].Primary || out.Accounts[0].Key.Set != (key != "") {
		p.t.Fatalf("the member's accounts: %+v", out.Accounts)
	}
	return out.Accounts[0].ID, out.Accounts[0].Models[0].ID
}

func (p *personal) pick(feature string, model uuid.UUID) {
	p.t.Helper()
	p.mc.Do(http.MethodPut, p.base+"/personal/features/"+feature+"/model", map[string]any{"modelId": model}).MustStatus(p.t, 200)
}

func draftBy(c *apptest.Client, base string) *apptest.Response {
	return c.Do(http.MethodPost, base+"/drafts/campaigns", ai.AICampaignBriefInput{SharingReviewed: true, Brief: "Announce easier exports.", Name: "Exports", Language: "en"})
}

// async runs a request in the background.
func async(do func() *apptest.Response) chan *apptest.Response {
	done := make(chan *apptest.Response, 1)
	go func() { done <- do() }()
	return done
}

func await(t *testing.T, done chan *apptest.Response, what string) *apptest.Response {
	t.Helper()
	select {
	case res := <-done:
		return res
	case <-time.After(20 * time.Second):
		t.Fatalf("%s waited", what)
		return nil
	}
}

// Off by default: no personal account can be added, and turning the switch
// off stops the next reservation of a personal request without deleting
// the account.
func TestAIPersonalKeysSwitch(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, nil)
	if res := p.mc.Do(http.MethodPost, p.base+"/personal/accounts", map[string]any{"provider": "compatible", "baseUrl": mine.srv.URL, "model": "own-model",
		"inputPrice": 1, "outputPrice": 2}).MustStatus(t, 409).Problem(t); res["code"] != "ai_personal_keys_off" {
		t.Fatalf("adding a personal key while off: %v", res)
	}
	if keys, _ := mine.got(); len(keys) != 0 {
		t.Fatal("a personal account was tested while personal keys are off")
	}
	p.enable(10)
	_, model := p.addOwn(mine.srv.URL, "", 1, 2)
	p.pick("draft", model)
	draftBy(p.mc, p.base).MustStatus(t, 200)

	// A request prepared while on, whose reservation comes after the switch
	// went off (without any other change), isn't sent.
	ctx := context.Background()
	tx, err := p.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM ai_workspace_settings WHERE workspace_id = $1 FOR UPDATE`, p.ws.ID); err != nil {
		t.Fatal(err)
	}
	done := async(func() *apptest.Response { return draftBy(p.mc, p.base) })
	p.waitForLock(1)
	if _, err := tx.Exec(ctx, `UPDATE ai_workspace_settings SET personal_keys = false WHERE workspace_id = $1`, p.ws.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if res := await(t, done, "the prepared request").MustStatus(t, 409).Problem(t); res["code"] != "ai_personal_keys_off" {
		t.Fatalf("a personal request after the switch went off: %v", res)
	}
	if _, models := mine.got(); len(models) != 2 {
		t.Fatalf("requests on the personal key: %v", models)
	}
	// Through the API too; the account stays.
	p.enable(10)
	p.c.Do(http.MethodPut, p.base+"/personal-keys", map[string]any{"enabled": false, "capUsd": 10}).MustStatus(t, 200)
	if res := draftBy(p.mc, p.base).MustStatus(t, 409).Problem(t); res["code"] != "ai_personal_keys_off" {
		t.Fatalf("a personal request while off: %v", res)
	}
	var own ai.AIPersonalSettings
	p.mc.Do(http.MethodGet, p.base+"/personal", nil).MustStatus(t, 200).JSON(t, &own)
	if own.Enabled || own.CanAdd || len(own.Accounts) != 1 || len(own.Choices) != 1 {
		t.Fatalf("personal settings while off: %+v", own)
	}
	var actions []string
	if err := pgxCollect(ctx, p.registry, `SELECT action FROM audit_events WHERE workspace_id = $1 AND action LIKE 'ai\_personal\_keys.%' ORDER BY occurred_at, id`, &actions, p.ws.ID); err != nil {
		t.Fatal(err)
	}
	// The SQL change above isn't the API's: on, on again, off.
	if !slices.Equal(actions, []string{"ai_personal_keys.enabled", "ai_personal_keys.enabled", "ai_personal_keys.disabled"}) {
		t.Fatalf("the switch's audit entries: %v", actions)
	}
}

func pgxCollect(ctx context.Context, r *registry, query string, into *[]string, args ...any) error {
	rows, err := r.env.App.Pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	*into = out
	return err
}

// A personal request counts against its owner's cap only: the workspace's
// usage doesn't move, the workspace's cap doesn't stop it, and the
// person's cap does.
func TestAIPersonalBudget(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, map[string]any{"capKind": "usd", "capUsd": 25})
	p.enable(5)
	_, model := p.addOwn(mine.srv.URL, "", 1, 2)
	p.pick("draft", model)
	var before ai.AISettings
	p.c.Do(http.MethodGet, p.base, nil).MustStatus(t, 200).JSON(t, &before)
	var proposal ai.AICampaignProposal
	draftBy(p.mc, p.base).MustStatus(t, 200).JSON(t, &proposal)
	if _, models := mine.got(); !slices.Equal(models, []string{"own-model", "own-model"}) {
		t.Fatalf("the personal server got %v", models)
	}
	var after ai.AISettings
	p.c.Do(http.MethodGet, p.base, nil).MustStatus(t, 200).JSON(t, &after)
	if after.Usage.Requests != before.Usage.Requests || after.Usage.CostMicros != before.Usage.CostMicros || after.Usage.ReservedCostMicros != 0 {
		t.Fatalf("the workspace's usage moved: %+v → %+v", before.Usage, after.Usage)
	}
	// The connection's test and the draft, each 113 in and 29 out at $1/$2.
	const each = 113*1 + 29*2
	if len(after.Personal.People) != 1 || after.Personal.People[0].UserID != p.member || after.Personal.People[0].Requests != 2 || after.Personal.People[0].CostMicros != 2*each ||
		len(after.Personal.People[0].Accounts) != 1 || !slices.Equal(after.Personal.People[0].Accounts[0].Models, []string{"own-model"}) {
		t.Fatalf("the manager's view of personal use: %+v", after.Personal.People)
	}
	var own ai.AIPersonalSettings
	p.mc.Do(http.MethodGet, p.base+"/personal", nil).MustStatus(t, 200).JSON(t, &own)
	if own.Usage.Requests != 2 || own.Usage.CostMicros != 2*each || len(own.People) != 0 {
		t.Fatalf("the member's own use: %+v", own)
	}
	ctx := context.Background()
	var reservedPersonal, loggedPersonal bool
	var owner *uuid.UUID
	if err := p.env.App.Pool.QueryRow(ctx, `SELECT b.personal, q.personal, q.user_id FROM ai_budget_reservations b JOIN ai_requests q ON q.id = b.id WHERE b.id = $1`, proposal.ID).
		Scan(&reservedPersonal, &loggedPersonal, &owner); err != nil {
		t.Fatal(err)
	}
	if !reservedPersonal || !loggedPersonal || owner == nil || *owner != p.member {
		t.Fatalf("the draft's reservation and log row: personal %v/%v, user %v", reservedPersonal, loggedPersonal, owner)
	}
	var activity ai.AIActivityList
	p.c.Do(http.MethodGet, p.base+"/activity", nil).MustStatus(t, 200).JSON(t, &activity)
	for _, row := range activity.Items {
		if (row.UserID == p.member.String()) != row.Personal {
			t.Fatalf("the activity log's personal flag: %+v", row)
		}
	}
	// The member asked only on their own key: the log's filter names them,
	// and filtering by them finds their requests.
	if !slices.Contains(activity.Members, ai.AIActivityMember{ID: p.member.String(), Name: "Registry Member"}) {
		t.Fatalf("the activity log's members: %+v", activity.Members)
	}
	var theirs ai.AIActivityList
	p.c.Do(http.MethodGet, p.base+"/activity?member="+p.member.String(), nil).MustStatus(t, 200).JSON(t, &theirs)
	if len(theirs.Items) != 2 {
		t.Fatalf("the member's activity: %+v", theirs.Items)
	}

	// The workspace's cap used up: the member's own key still answers.
	spent := uuid.Must(uuid.NewV7())
	p.exec(`INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, cost_micros, outcome, created_at)
		VALUES ($1, $2, $3, 'rewrite', 'compatible', 'model-a', 25000000, 'accepted', now())`, spent, p.ws.ID, p.owner)
	if res := draftBy(p.c, p.base).MustStatus(t, 409).Problem(t); res["code"] != "ai_cap_reached" {
		t.Fatalf("the workspace at its cap: %v", res)
	}
	draftBy(p.mc, p.base).MustStatus(t, 200)
	// The person's cap used up: their key stops, the workspace's doesn't.
	p.exec(`DELETE FROM ai_requests WHERE id = $1`, spent)
	p.exec(`INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, cost_micros, outcome, created_at, personal)
		VALUES ($1, $2, $3, 'rewrite', 'compatible', 'own-model', 5000000, 'accepted', now(), true)`, uuid.Must(uuid.NewV7()), p.ws.ID, p.member)
	sent, _ := mine.got()
	if res := draftBy(p.mc, p.base).MustStatus(t, 409).Problem(t); res["code"] != "ai_personal_cap_reached" {
		t.Fatalf("the person at their cap: %v", res)
	}
	if now, _ := mine.got(); len(now) != len(sent) {
		t.Fatal("a request past the person's cap was sent")
	}
	draftBy(p.c, p.base).MustStatus(t, 200)
}

// Status follows each feature's own model and budget, as the requests do:
// the member's draft on their own key is ready with the workspace's cap used
// up, stops at their own cap while the workspace is ready, and is ready with
// no workspace model connected. The agent follows the model a new run of
// theirs takes, and its switch.
func TestAIStatusFollowsPersonalBudget(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, map[string]any{"capKind": "usd", "capUsd": 25})
	p.enable(5)
	p.c.Do(http.MethodPut, p.base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	account, model := p.addOwn(mine.srv.URL, "", 1, 2)
	p.pick("draft", model)
	var agent ai.AIStatusFeature
	status := func() (st ai.AIStatus, draft, rewrite ai.AIStatusFeature) {
		t.Helper()
		p.mc.Do(http.MethodGet, p.base+"/status", nil).MustStatus(t, 200).JSON(t, &st)
		agent = ai.AIStatusFeature{}
		for _, f := range st.Features {
			switch f.Key {
			case "draft":
				draft = f
			case "rewrite":
				rewrite = f
			case "agent":
				agent = f
			}
		}
		return st, draft, rewrite
	}

	// The workspace's cap used up: the draft, in the member's own budget
	// (so far the add's test, 113 in and 29 out at $1/$2), is still ready.
	spent := uuid.Must(uuid.NewV7())
	p.exec(`INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, cost_micros, outcome, created_at)
		VALUES ($1, $2, $3, 'rewrite', 'compatible', 'model-a', 25000000, 'accepted', now())`, spent, p.ws.ID, p.owner)
	st, draft, rewrite := status()
	if st.State != "cap_reached" || st.Usage.State != "reached" || rewrite.State != "cap_reached" || rewrite.Personal || rewrite.Cap.USD != 25 {
		t.Fatalf("the workspace at its cap: %+v, rewrite %+v", st, rewrite)
	}
	if draft.State != "ready" || !draft.On || !draft.Personal || draft.ModelName != "own-model" || draft.Prices.Input != 1 || draft.Prices.Output != 2 ||
		draft.Cap.Kind != "usd" || draft.Cap.USD != 5 || draft.Usage.State != "ok" || draft.Usage.Requests != 1 || draft.Usage.CostMicros != 113*1+29*2 {
		t.Fatalf("the member's draft with the workspace at its cap: %+v", draft)
	}
	// The agent's runs go to the workspace's model: its cap, reached.
	if agent.Key != "agent" || agent.State != "cap_reached" || !agent.On || agent.Personal || agent.ModelID == nil || *agent.ModelID != *rewrite.ModelID ||
		agent.ModelName != rewrite.ModelName || agent.Cap.USD != 25 || agent.Prices.Input != rewrite.Prices.Input {
		t.Fatalf("the agent with the workspace at its cap: %+v", agent)
	}
	draftBy(p.mc, p.base).MustStatus(t, 200)

	// The member's cap used up: their draft stops, the workspace is ready.
	p.exec(`DELETE FROM ai_requests WHERE id = $1`, spent)
	own := uuid.Must(uuid.NewV7())
	p.exec(`INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, cost_micros, outcome, created_at, personal)
		VALUES ($1, $2, $3, 'rewrite', 'compatible', 'own-model', 5000000, 'accepted', now(), true)`, own, p.ws.ID, p.member)
	st, draft, rewrite = status()
	if st.State != "ready" || rewrite.State != "ready" || draft.State != "cap_reached" || draft.Usage.State != "reached" || agent.State != "ready" {
		t.Fatalf("the member at their cap: %s, rewrite %+v, draft %+v, agent %+v", st.State, rewrite, draft, agent)
	}
	if res := draftBy(p.mc, p.base).MustStatus(t, 409).Problem(t); res["code"] != "ai_personal_cap_reached" {
		t.Fatalf("the member's draft at their cap: %v", res)
	}

	// No workspace model connected: only the member's own draft is ready.
	p.exec(`DELETE FROM ai_requests WHERE id = $1`, own)
	p.c.Do(http.MethodDelete, p.base+"/connection", nil).MustStatus(t, 200)
	st, draft, rewrite = status()
	if st.State != "not_configured" || rewrite.State != "not_configured" || rewrite.On || draft.State != "ready" || !draft.On || !draft.Personal {
		t.Fatalf("without a workspace model: %s, rewrite %+v, draft %+v", st.State, rewrite, draft)
	}
	draftBy(p.mc, p.base).MustStatus(t, 200)

	// The agent on the member's own model once its tool check passed: its
	// state, model, prices, cap and use are the draft's; it's off with the
	// agent's switch.
	if agent.State != "not_configured" || agent.On {
		t.Fatalf("the agent without a workspace model: %+v", agent)
	}
	var tools ai.AIAgentCapability
	p.mc.Do(http.MethodPost, p.base+"/personal/accounts/"+account.String()+"/models/"+model.String()+"/test-tools", nil).MustStatus(t, 200).JSON(t, &tools)
	if tools.State != "supported" {
		t.Fatalf("the personal tool check: %+v", tools)
	}
	p.pick("agent", model)
	st, draft, _ = status()
	if agent.State != "ready" || !agent.On || !agent.Available || !agent.Personal || agent.ModelID == nil || *agent.ModelID != model || agent.ModelName != "own-model" ||
		agent.Prices.Input != 1 || agent.Prices.Output != 2 || agent.Cap.USD != 5 || agent.Usage.Requests != draft.Usage.Requests || agent.Usage.CostMicros != draft.Usage.CostMicros {
		t.Fatalf("the agent on the member's model: %+v, draft %+v", agent, draft)
	}
	p.c.Do(http.MethodPut, p.base+"/agent/settings", ai.AIAgentSettings{Enabled: false, Limits: st.Agent.Limits, Revision: st.Agent.Revision}).MustStatus(t, 204)
	if status(); agent.On || agent.State != "ready" {
		t.Fatalf("the agent switched off: %+v", agent)
	}
}

// Two requests on one person's key race that person's cap: of two tests
// that each fit alone, one reserves and the other is refused before it is
// sent. (The workspace has no cap.)
func TestAIPersonalReservationsSerialize(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	defer mine.free()
	p := newPersonal(t, first, nil)
	// At $1,000 per million tokens a test reserves about $4.40; the add's
	// test used $0.14 of the $6.00.
	p.enable(6)
	account, model := p.addOwn(mine.srv.URL, "", 1000, 1000)
	mine.hold.Store(true)
	ctx := context.Background()
	tx, err := p.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM ai_workspace_settings WHERE workspace_id = $1 FOR UPDATE`, p.ws.ID); err != nil {
		t.Fatal(err)
	}
	path := p.base + "/personal/accounts/" + account.String() + "/models/" + model.String() + "/test"
	done := make(chan *apptest.Response, 2)
	for range 2 {
		go func() { done <- p.mc.Do(http.MethodPost, path, nil) }()
	}
	p.waitForLock(2)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if res := await(t, done, "both tests").MustStatus(t, 409).Problem(t); res["code"] != "ai_reservation_cap" {
		t.Fatalf("the losing test: %v", res)
	}
	mine.free()
	await(t, done, "the winning test").MustStatus(t, 200)
	if keys, _ := mine.got(); len(keys) != 2 {
		t.Fatalf("%d requests reached the personal server, want the add's test and one more", len(keys))
	}
}

// A personal request and a workspace request don't wait for each other's
// budget: each budget has its own lock.
func TestAIPersonalBudgetLocks(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, map[string]any{"capKind": "usd", "capUsd": 25})
	p.enable(10)
	_, model := p.addOwn(mine.srv.URL, "", 1, 2)
	p.pick("draft", model)
	ctx := context.Background()
	hold := func(key string) pgx.Tx {
		t.Helper()
		tx, err := p.env.App.Pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
			t.Fatal(err)
		}
		return tx
	}
	for _, c := range []struct {
		name         string
		lock         string
		free, waits  *apptest.Client
		freeModelSrv *modelServer
	}{
		{"the workspace's budget", "ai_budget:" + p.ws.ID.String(), p.mc, p.c, mine},
		{"the person's budget", "ai_budget:" + p.ws.ID.String() + ":" + p.member.String(), p.c, p.mc, first},
	} {
		tx := hold(c.lock)
		// Reserved and accounted while the other budget's lock is held.
		before, _ := c.freeModelSrv.got()
		await(t, async(func() *apptest.Response { return draftBy(c.free, p.base) }), "a request on another budget than "+c.name).MustStatus(t, 200)
		if after, _ := c.freeModelSrv.got(); len(after) != len(before)+1 {
			t.Fatalf("with %s locked, the other request wasn't sent", c.name)
		}
		done := async(func() *apptest.Response { return draftBy(c.waits, p.base) })
		p.waitForLock(1)
		select {
		case res := <-done:
			t.Fatalf("a request on %s didn't wait for its lock: %d", c.name, res.Status)
		default:
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		await(t, done, "the request on "+c.name).MustStatus(t, 200)
	}
}

// Someone else's personal account isn't there, even for a manager: no
// operation on it, no test with it, no feature default on it. Managers see
// it exists and may remove it.
func TestAIPersonalKeyIsTheOwners(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, nil)
	p.enable(10)
	const sentinel = "sk-sentinel-personal-4b8e0d"
	account, model := p.addOwn(mine.srv.URL, sentinel, 1, 2)
	other := p.env.CreateUser(apptest.UserOpts{Email: "colleague@registry.test", Name: "Colleague"})
	p.env.AddMember(p.ws, other.ID, authz.RoleEditor)
	oc := p.env.Client()
	oc.SignInDemo(other.Email)
	a, m := "/accounts/"+account.String(), "/accounts/"+account.String()+"/models/"+model.String()
	personalOps := []struct {
		method, path string
		body         any
	}{
		{http.MethodDelete, "/personal" + a, nil},
		{http.MethodPut, "/personal" + a + "/key", map[string]any{"apiKey": "sk-not-sent"}},
		{http.MethodPost, "/personal" + a + "/models", map[string]any{"model": "other-model", "inputPrice": 1, "outputPrice": 1}},
		{http.MethodPut, "/personal" + m + "/prices", map[string]any{"input": 9, "output": 9}},
		{http.MethodDelete, "/personal" + m, nil},
		{http.MethodPost, "/personal" + m + "/test", nil},
		{http.MethodPost, "/personal" + m + "/test-tools", nil},
		{http.MethodPut, "/personal/features/draft/model", map[string]any{"modelId": model}},
	}
	var bodies []string
	for _, c := range []*apptest.Client{p.c, oc} {
		for _, op := range personalOps {
			res := c.Do(op.method, p.base+op.path, op.body).MustStatus(t, 404)
			bodies = append(bodies, string(res.Body))
		}
	}
	// The workspace's operations don't reach it either.
	for _, op := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/test", map[string]any{"modelId": model}},
		{http.MethodPost, "/test", map[string]any{"accountId": account}},
		{http.MethodPost, "/models", map[string]any{"provider": "compatible", "baseUrl": mine.srv.URL, "accountId": account}},
		{http.MethodPut, a + "/key", map[string]any{"apiKey": "sk-not-sent"}},
		{http.MethodPost, a + "/models", map[string]any{"model": "other-model", "inputPrice": 1, "outputPrice": 1}},
		{http.MethodPost, m + "/test-tools", nil},
		{http.MethodDelete, a, nil},
	} {
		p.c.Do(op.method, p.base+op.path, op.body).MustStatus(t, 404)
	}
	// A personal model is never a feature's default.
	if res := p.c.Do(http.MethodPut, p.base+"/features/draft/model", map[string]any{"modelId": model}).MustStatus(t, 409).Problem(t); res["code"] != "ai_personal_model" {
		t.Fatalf("a personal model as a feature default: %v", res)
	}
	if keys, _ := mine.got(); len(keys) != 1 {
		t.Fatalf("the personal server got %d requests, want the add's test only", len(keys))
	}
	// Managers see who has it, never the key; the colleague doesn't.
	var s ai.AISettings
	res := p.c.Do(http.MethodGet, p.base, nil).MustStatus(t, 200)
	bodies = append(bodies, string(res.Body))
	res.JSON(t, &s)
	if len(s.Accounts) != 1 || len(s.Personal.Accounts) != 0 || len(s.Personal.People) != 1 || s.Personal.People[0].Accounts[0].ID != account ||
		s.Personal.People[0].Accounts[0].Status != "connected" || s.Personal.People[0].Name != "Registry Member" {
		t.Fatalf("the manager's settings: %+v / %+v", s.Accounts, s.Personal)
	}
	for _, d := range s.Defaults {
		if d.ModelID == model {
			t.Fatalf("a personal model became a default: %+v", s.Defaults)
		}
	}
	var theirs ai.AIPersonalSettings
	oc.Do(http.MethodGet, p.base+"/personal", nil).MustStatus(t, 200).JSON(t, &theirs)
	if len(theirs.Accounts) != 0 || len(theirs.People) != 0 {
		t.Fatalf("a colleague sees: %+v", theirs)
	}
	// Offboarding: a manager removes it.
	res = p.c.Do(http.MethodDelete, p.base+"/personal-keys/accounts/"+account.String(), nil).MustStatus(t, 200)
	bodies = append(bodies, string(res.Body))
	p.c.Do(http.MethodDelete, p.base+"/personal-keys/accounts/"+p.account.String(), nil).MustStatus(t, 404)
	var removed int
	if err := p.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'ai_account.remove'
		AND metadata->>'personal' = 'true' AND metadata->>'owner' = $2`, p.ws.ID, p.member.String()).Scan(&removed); err != nil || removed != 1 {
		t.Fatalf("the removal's audit entry: %d %v", removed, err)
	}
	assertNoKey(t, p.registry, bodies, sentinel)
}

// Deactivating or removing a member deletes their personal keys: their
// requests are refused, and they add the key again to use the account.
func TestAIPersonalKeysLeaveWithTheMember(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, nil)
	p.enable(10)
	account, model := p.addOwn(mine.srv.URL, "sk-sentinel-member-1c9a", 1, 2)
	p.pick("draft", model)
	ctx := context.Background()
	var memberID uuid.UUID
	if err := p.env.App.Pool.QueryRow(ctx, `SELECT id FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, p.ws.ID, p.member).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	team := "/workspaces/" + p.ws.Slug + "/members/" + memberID.String()
	keyGone := func(what string) {
		t.Helper()
		var sealed []byte
		var disconnected *time.Time
		if err := p.env.App.Pool.QueryRow(ctx, `SELECT key_sealed, disconnected_at FROM ai_accounts WHERE id = $1`, account).Scan(&sealed, &disconnected); err != nil {
			t.Fatal(err)
		}
		if len(sealed) != 0 || disconnected == nil {
			t.Fatalf("%s: the personal key is still there", what)
		}
	}
	p.c.Do(http.MethodPost, team+"/deactivate", map[string]any{}).MustStatus(t, 200)
	keyGone("deactivated")
	var n int
	if err := p.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'member.deactivated' AND after->>'personal_ai_keys_removed' = '1'`, p.ws.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("the deactivation's audit entry: %d %v", n, err)
	}
	p.c.Do(http.MethodPost, team+"/reactivate", nil).MustStatus(t, 204)
	p.mc.SignInDemo("member@registry.test")
	sent, _ := mine.got()
	if res := draftBy(p.mc, p.base).MustStatus(t, 409).Problem(t); res["code"] != "ai_not_configured" || !strings.Contains(res["detail"].(string), "no key") {
		t.Fatalf("a request on the removed key: %v", res)
	}
	if now, _ := mine.got(); len(now) != len(sent) {
		t.Fatal("a request on the removed key was sent")
	}
	// Back with a new key, it works again.
	p.mc.Do(http.MethodPut, p.base+"/personal/accounts/"+account.String()+"/key", map[string]any{"apiKey": "sk-sentinel-member-again-77"}).MustStatus(t, 200)
	draftBy(p.mc, p.base).MustStatus(t, 200)
	p.c.Do(http.MethodDelete, team, nil).MustStatus(t, 204)
	keyGone("removed")
}

// The owner's agent runs use their own model only once it passed the tool
// check, and a run stays on it after they pick another.
func TestAIPersonalAgentRun(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, map[string]any{"model": "agent-a", "inputPrice": 3, "outputPrice": 7})
	p.c.Do(http.MethodPut, p.base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	p.enable(10)
	account, model := p.addOwn(mine.srv.URL, "", 3, 7)
	if res := p.mc.Do(http.MethodPut, p.base+"/personal/features/agent/model", map[string]any{"modelId": model}).MustStatus(t, 409).Problem(t); res["code"] != "ai_tools_unchecked" {
		t.Fatalf("the agent on an unchecked personal model: %v", res)
	}
	var tools ai.AIAgentCapability
	p.mc.Do(http.MethodPost, p.base+"/personal/accounts/"+account.String()+"/models/"+model.String()+"/test-tools", nil).MustStatus(t, 200).JSON(t, &tools)
	if tools.State != "supported" {
		t.Fatalf("the personal tool check: %+v", tools)
	}
	p.pick("agent", model)
	var run ai.AIAgentRunView
	p.mc.Do(http.MethodPost, p.base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Announce easier exports.", Language: "en", RequestKey: "personal-1"}).MustStatus(t, 200).JSON(t, &run)
	// Back to the workspace's model for new runs; this one stays.
	p.mc.Do(http.MethodDelete, p.base+"/personal/features/agent/model", nil).MustStatus(t, 200)
	env := p.env
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	ctx := context.Background()
	before, _ := first.got()
	if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: p.ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
		t.Fatal(err)
	}
	var runModel, reserved *uuid.UUID
	var personalHold bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT r.model_id, b.model_id, b.personal FROM ai_agent_runs r JOIN ai_budget_reservations b ON b.run_id = r.id
		WHERE r.id = $1 ORDER BY b.created_at DESC LIMIT 1`, run.ID).Scan(&runModel, &reserved, &personalHold); err != nil {
		t.Fatal(err)
	}
	if runModel == nil || *runModel != model || reserved == nil || *reserved != model || !personalHold {
		t.Fatalf("the run on %v, reserved on %v (personal %v); want %s", runModel, reserved, personalHold, model)
	}
	if after, _ := first.got(); len(after) != len(before) {
		t.Fatal("the run's request went to the workspace's model")
	}
	if _, models := mine.got(); models[len(models)-1] != "own-model" {
		t.Fatalf("the personal server got %v", models)
	}
}

// usageless answers like srv, without the usage once drop is set: a
// request whose cost is unknown.
func usageless(t *testing.T, srv *modelServer) (*httptest.Server, *atomic.Bool) {
	drop := &atomic.Bool{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		srv.srv.Config.Handler.ServeHTTP(rec, r)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if drop.Load() {
			delete(body, "usage")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(s.Close)
	return s, drop
}

// A hold on a personal key whose usage is unknown is its owner's to
// reconcile, in their own budget and under their own lock, once: nobody
// else sees or settles it, and the workspace's ledger doesn't move.
func TestAIPersonalBudgetHolds(t *testing.T) {
	first, inner := newModelServer(t), newModelServer(t)
	srv, drop := usageless(t, inner)
	p := newPersonal(t, first, map[string]any{"capKind": "usd", "capUsd": 25})
	p.enable(10)
	_, model := p.addOwn(srv.URL, "", 1, 2)
	p.pick("draft", model)
	drop.Store(true)
	if res := draftBy(p.mc, p.base).MustStatus(t, 409).Problem(t); res["code"] != "ai_usage_unknown" {
		t.Fatalf("a personal request without usage: %v", res)
	}
	drop.Store(false)
	var holds ai.AIBudgetHolds
	p.mc.Do(http.MethodGet, p.base+"/personal/budget-holds", nil).MustStatus(t, 200).JSON(t, &holds)
	if len(holds.Items) != 1 || holds.Items[0].Model != "own-model" {
		t.Fatalf("the member's holds: %+v", holds)
	}
	id := holds.Items[0].ID
	path := p.base + "/personal/budget-holds/" + id.String() + "/reconcile"
	counts := ai.AIBudgetReconcileInput{InputTokens: 17, OutputTokens: 11, Confirmed: true}

	// Not the workspace's, a manager's nor a colleague's.
	p.c.Do(http.MethodGet, p.base+"/budget-holds", nil).MustStatus(t, 200).JSON(t, &holds)
	if len(holds.Items) != 0 {
		t.Fatalf("the workspace's holds list a personal one: %+v", holds)
	}
	colleague := p.env.CreateUser(apptest.UserOpts{Email: "colleague@registry.test", Name: "Colleague"})
	p.env.AddMember(p.ws, colleague.ID, authz.RoleEditor)
	oc := p.env.Client()
	oc.SignInDemo(colleague.Email)
	oc.Do(http.MethodGet, p.base+"/personal/budget-holds", nil).MustStatus(t, 200).JSON(t, &holds)
	if len(holds.Items) != 0 {
		t.Fatalf("a colleague sees: %+v", holds)
	}
	oc.Do(http.MethodPost, path, counts).MustStatus(t, 404)
	p.c.Do(http.MethodPost, path, counts).MustStatus(t, 404)
	p.c.Do(http.MethodPost, p.base+"/budget-holds/"+id.String()+"/reconcile", counts).MustStatus(t, 404)
	if res := p.mc.Do(http.MethodPost, path, counts).MustStatus(t, 409).Problem(t); res["code"] != "ai_reconcile_wait" {
		t.Fatalf("reconciling while the answer may still come: %v", res)
	}
	p.exec(`UPDATE ai_budget_reservations SET created_at = now() - interval '11 minutes' WHERE id = $1`, id)

	var wsBefore, wsAfter ai.AISettings
	var before, after ai.AIPersonalSettings
	p.c.Do(http.MethodGet, p.base, nil).MustStatus(t, 200).JSON(t, &wsBefore)
	p.mc.Do(http.MethodGet, p.base+"/personal", nil).MustStatus(t, 200).JSON(t, &before)
	if before.Usage.UncertainRequests != 1 || before.Usage.ReservedCostMicros == 0 || wsBefore.Usage.UncertainRequests != 0 || wsBefore.Usage.ReservedCostMicros != 0 {
		t.Fatalf("the hold before: the member's %+v, the workspace's %+v", before.Usage, wsBefore.Usage)
	}
	// Settled under the person's budget lock: the workspace's, held
	// meanwhile, doesn't stop it.
	ctx := context.Background()
	tx, err := p.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "ai_budget:"+p.ws.ID.String()); err != nil {
		t.Fatal(err)
	}
	await(t, async(func() *apptest.Response { return p.mc.Do(http.MethodPost, path, counts) }), "the reconcile").MustStatus(t, 204)
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	// Once: the same counts again change nothing, others are refused.
	p.mc.Do(http.MethodPost, path, counts).MustStatus(t, 204)
	other := counts
	other.OutputTokens++
	if res := p.mc.Do(http.MethodPost, path, other).MustStatus(t, 409).Problem(t); res["code"] != "ai_reconcile_changed" {
		t.Fatalf("reconciling again with other counts: %v", res)
	}

	const reconciled = 17*1 + 11*2
	p.mc.Do(http.MethodGet, p.base+"/personal", nil).MustStatus(t, 200).JSON(t, &after)
	if after.Usage.UncertainRequests != 0 || after.Usage.ReservedCostMicros != 0 || after.Usage.CostMicros != before.Usage.CostMicros+reconciled || after.Usage.Requests != before.Usage.Requests {
		t.Fatalf("the member's use: %+v → %+v", before.Usage, after.Usage)
	}
	p.c.Do(http.MethodGet, p.base, nil).MustStatus(t, 200).JSON(t, &wsAfter)
	if wsAfter.Usage != wsBefore.Usage {
		t.Fatalf("the workspace's use moved: %+v → %+v", wsBefore.Usage, wsAfter.Usage)
	}
	var personal, known bool
	var cost int64
	var state string
	if err := p.env.App.Pool.QueryRow(ctx, `SELECT q.personal, q.usage_known, q.cost_micros, b.state FROM ai_requests q JOIN ai_budget_reservations b ON b.id = q.id WHERE q.id = $1`, id).
		Scan(&personal, &known, &cost, &state); err != nil {
		t.Fatal(err)
	}
	if !personal || !known || cost != reconciled || state != "settled" {
		t.Fatalf("the reconciled request: personal %v, known %v, cost %d, %s", personal, known, cost, state)
	}
	var audits int
	if err := p.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'ai.usage_reconciled' AND target_id = $2
		AND metadata->>'personal' = 'true'`, p.ws.ID, id.String()).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("the reconcile's audit entries: %d %v", audits, err)
	}
	p.mc.Do(http.MethodGet, p.base+"/personal/budget-holds", nil).MustStatus(t, 200).JSON(t, &holds)
	if len(holds.Items) != 0 {
		t.Fatalf("a settled hold is still listed: %+v", holds)
	}
}

// rejecting answers like srv until reject is set, then refuses every key,
// repeating it in its error as some providers do.
func rejecting(t *testing.T, srv *modelServer) (*httptest.Server, *atomic.Bool) {
	reject := &atomic.Bool{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reject.Load() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "Incorrect API key provided: " + strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "),
				"type": "invalid_request_error", "code": "invalid_api_key"}})
			return
		}
		srv.srv.Config.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)
	return s, reject
}

// A personal key never comes back nor is kept readable: not in responses,
// audit entries, the activity log, notifications, jobs or logs, also when
// the provider refuses it. Only its owner is told it was refused.
func TestAIPersonalKeyNotShown(t *testing.T) {
	first, inner := newModelServer(t), newModelServer(t)
	srv, reject := rejecting(t, inner)
	p := newPersonal(t, first, nil)
	p.enable(10)
	var logs bytes.Buffer
	env := p.env
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true },
		Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})), BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	who := &authz.Principal{Type: authz.ActorUser, ID: p.member, UserID: p.member, WorkspaceID: p.ws.ID}
	ctx := audit.WithPrincipal(context.Background(), who)
	const key, next = "sk-sentinel-own-6d21", "sk-sentinel-own-next-a07f"
	var bodies []string
	keep := func(v any) {
		raw, _ := json.Marshal(v)
		bodies = append(bodies, string(raw))
	}
	out, err := s.AddPersonalAccount(ctx, &p.ws, who, ai.AIAccountInput{Provider: "compatible", BaseURL: srv.URL, APIKey: key, Model: "own-model", InputPrice: ptr(1.0), OutputPrice: ptr(2.0)})
	if err != nil {
		t.Fatal(err)
	}
	keep(out)
	account, model := out.Accounts[0].ID, out.Accounts[0].Models[0].ID
	replaced, test, err := s.ReplacePersonalKey(ctx, &p.ws, who, account, ai.AIKeyInput{APIKey: next})
	if err != nil {
		t.Fatal(err)
	}
	keep(replaced)
	keep(test)
	reject.Store(true)
	res, err := s.TestPersonalModel(ctx, &p.ws, who, account, model)
	if err != nil || res.OK || res.ErrorKind != "auth" {
		t.Fatalf("a refused personal key's test: %+v %v", res, err)
	}
	keep(res)
	if keys, _ := inner.got(); !slices.Equal(keys, []string{key, next}) {
		t.Fatalf("keys that were answered: %v", keys)
	}
	for _, c := range []*apptest.Client{p.c, p.mc} {
		for _, path := range []string{"", "/personal", "/activity"} {
			if c == p.mc && path != "/personal" {
				continue
			}
			bodies = append(bodies, string(c.Do(http.MethodGet, p.base+path, nil).MustStatus(t, 200).Body))
		}
	}
	for _, k := range []string{key, next} {
		if strings.Contains(logs.String(), k) {
			t.Fatalf("the log has the key: %s", logs.String())
		}
	}
	assertNoKey(t, p.registry, bodies, key, next)
	var mineN, managers int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE user_id = $2), count(*) FILTER (WHERE user_id <> $2) FROM notifications WHERE workspace_id = $1 AND kind = 'ai_key_rejected'`,
		p.ws.ID, p.member).Scan(&mineN, &managers); err != nil || mineN != 1 || managers != 0 {
		t.Fatalf("key-rejected notifications: the owner %d, others %d (%v)", mineN, managers, err)
	}
}

// Personal keys' operations need ai:use, the switch and the cap
// integrations:manage: a Viewer gets no personal key, an Editor and a
// Content Writer don't set the switch, and see only their own accounts.
func TestAIPersonalPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	srv := newModelServer(t)
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	clients := map[string]*apptest.Client{}
	for _, who := range []string{"sofia@acme.io", "daniel@acme.io", "aisha@acme.io"} {
		c := env.Client()
		c.SignInDemo(who)
		clients[who] = c
	}
	base := ws + "/ai"
	for _, who := range []string{"sofia@acme.io", "daniel@acme.io"} {
		clients[who].Do(http.MethodPut, base+"/personal-keys", map[string]any{"enabled": true, "capUsd": 10}).MustStatus(t, 403)
	}
	var s ai.AISettings
	linh.Do(http.MethodPut, base+"/personal-keys", map[string]any{"enabled": true, "capUsd": 12.5}).MustStatus(t, 200).JSON(t, &s)
	if !s.Personal.Enabled || s.Personal.CapUSD != 12.5 || s.Personal.CapSetBy == "" {
		t.Fatalf("after turning personal keys on: %+v", s.Personal)
	}
	add := map[string]any{"provider": "compatible", "baseUrl": srv.srv.URL, "model": "own-model", "inputPrice": 1, "outputPrice": 2}
	aisha := clients["aisha@acme.io"]
	aisha.Do(http.MethodGet, base+"/personal", nil).MustStatus(t, 403)
	aisha.Do(http.MethodPost, base+"/personal/accounts", add).MustStatus(t, 403)
	aisha.Do(http.MethodPut, base+"/personal/features/draft/model", map[string]any{"modelId": uuid.Must(uuid.NewV7())}).MustStatus(t, 403)
	var own ai.AIPersonalSettings
	clients["sofia@acme.io"].Do(http.MethodPost, base+"/personal/accounts", add).MustStatus(t, 200).JSON(t, &own)
	if len(own.Accounts) != 1 || !own.CanAdd || own.Usage.Requests != 1 {
		t.Fatalf("Sofia's personal settings: %+v", own)
	}
	account := own.Accounts[0].ID.String()
	daniel := clients["daniel@acme.io"]
	var his ai.AIPersonalSettings
	daniel.Do(http.MethodGet, base+"/personal", nil).MustStatus(t, 200).JSON(t, &his)
	if len(his.Accounts) != 0 || len(his.People) != 0 {
		t.Fatalf("Daniel sees: %+v", his)
	}
	daniel.Do(http.MethodDelete, base+"/personal/accounts/"+account, nil).MustStatus(t, 404)
	for _, who := range []string{"sofia@acme.io", "daniel@acme.io"} {
		clients[who].Do(http.MethodDelete, base+"/personal-keys/accounts/"+account, nil).MustStatus(t, 403)
		var view ai.AISettings
		clients[who].Do(http.MethodGet, base, nil).MustStatus(t, 200).JSON(t, &view)
		if len(view.Personal.People) != 0 {
			t.Fatalf("%s sees who has personal keys: %+v", who, view.Personal.People)
		}
	}
	linh.Do(http.MethodGet, base, nil).MustStatus(t, 200).JSON(t, &s)
	if len(s.Personal.People) != 1 || s.Personal.People[0].Name != "Sofia Rossi" || len(s.Personal.People[0].Accounts) != 1 {
		t.Fatalf("the manager's list of personal keys: %+v", s.Personal.People)
	}
	linh.Do(http.MethodDelete, base+"/personal-keys/accounts/"+account, nil).MustStatus(t, 200)
}
