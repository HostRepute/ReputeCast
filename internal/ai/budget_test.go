package ai_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestAIImageReservation(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@vision.test"})
	ws := env.CreateWorkspace("Image reservations", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	campaign := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO campaigns(id,workspace_id,name) VALUES($1,$2,'Image draft')`, campaign, ws.ID); err != nil {
		t.Fatal(err)
	}
	// An ordinary, detailed 1,024px PNG exceeds the text-context limit
	// when encoded as base64. A small flat PNG exercises the opposite:
	// compression must not make a vision reservation effectively free.
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	random := rand.New(rand.NewPCG(17, 43))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = byte(random.Uint32()), byte(random.Uint32()), byte(random.Uint32()), 255
	}
	var detailed, flat bytes.Buffer
	if err := png.Encode(&detailed, img); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&flat, image.NewGray(image.Rect(0, 0, 1024, 1024))); err != nil {
		t.Fatal(err)
	}
	if detailed.Len() < 1_000_000 || flat.Len() > 10_000 {
		t.Fatal("image fixtures do not distinguish binary size from visual size")
	}
	images := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		if r.URL.Path == "/flat.png" {
			_, _ = w.Write(flat.Bytes())
		} else {
			_, _ = w.Write(detailed.Bytes())
		}
	}))
	defer images.Close()
	var seen atomic.Int32
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "data:image/png;base64,") {
			seen.Add(1)
			var held int64
			if err := env.App.Pool.QueryRow(r.Context(), `SELECT input_tokens FROM ai_budget_reservations WHERE workspace_id=$1 AND feature='alt_text' AND state='reserved'`, ws.ID).Scan(&held); err != nil || held < 65_536 {
				t.Errorf("image was not reserved before the request: %d %v", held, err)
			}
			if seen.Load() == 1 && !bytes.Contains(body, []byte(base64.StdEncoding.EncodeToString(detailed.Bytes()))) {
				t.Error("budget preparation removed or changed the actual image")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": `{"alt":"A detailed pattern of coloured pixels."}`}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 3107, "completion_tokens": 13},
		})
	}))
	defer model.Close()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "gemma3:27b", "capKind": "usd", "capUsd": 25, "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200)
	for _, path := range []string{"/detailed.png", "/flat.png"} {
		var result ai.AIAltText
		c.Do("POST", base+"/alt-text", map[string]any{"campaignId": campaign.String(), "src": images.URL + path}).MustStatus(t, 200).JSON(t, &result)
		if result.Text != "A detailed pattern of coloured pixels." || result.Sent.InputTokens != 3107 || result.Sent.CostMicros != 3133 {
			t.Fatalf("image output or measured billing: %+v", result)
		}
	}
	// A token cap must still block the smallest image before model I/O.
	c.Do("PUT", base+"/cap", map[string]any{"kind": "tokens", "tokens": 20000}).MustStatus(t, 200)
	c.Do("POST", base+"/alt-text", map[string]any{"campaignId": campaign.String(), "src": images.URL + "/flat.png"}).MustStatus(t, 409)
	if seen.Load() != 2 {
		t.Fatalf("expected two permitted image requests, got %d", seen.Load())
	}
}

func TestAIReservations(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@budget.test"})
	other := env.CreateUser(apptest.UserOpts{Email: "editor@budget.test"})
	ws := env.CreateWorkspace("Budget", owner.ID)
	env.AddMember(ws, other.ID, "editor")
	a, b := env.Client(), env.Client()
	a.SignInDemo(owner.Email)
	b.SignInDemo(other.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	var mode atomic.Int32
	var requests atomic.Int32
	entered, release := make(chan struct{}, 1), make(chan struct{})
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		requests.Add(1)
		if mode.Load() == 1 {
			entered <- struct{}{}
			<-release
		}
		answer, _ := json.Marshal(ai.AIDraftEmail{Subject: "Export update", Preheader: "What changed", MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Exports are easier.</mj-text></mj-column></mj-section></mj-body></mjml>`})
		out := map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": string(answer)}, "finish_reason": "stop"}}}
		if mode.Load() != 2 {
			out["usage"] = map[string]int{"prompt_tokens": 113, "completion_tokens": 29}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		model.Close()
	}()
	a.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "budget-fixture", "capKind": "tokens", "capTokens": 10000, "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	in := ai.AICampaignBriefInput{SharingReviewed: true, Brief: "Announce easier exports.", Name: "Exports", Language: "en"}
	mode.Store(1)
	finished := make(chan *apptest.Response, 1)
	go func() {
		finished <- a.Do("POST", base+"/drafts/campaigns", in)
	}()
	select {
	case <-entered:
	case result := <-finished:
		result.MustStatus(t, 200)
		t.Fatal("first request returned without calling the model")
	case <-time.After(10 * time.Second):
		t.Fatal("first request did not reach the model")
	}
	before := requests.Load()
	b.Do("POST", base+"/drafts/campaigns", in).MustStatus(t, 409)
	a.Do("POST", base+"/test", map[string]any{}).MustStatus(t, 409)
	if requests.Load() != before {
		t.Fatal("another draft or a connection test spent the reserved monthly budget")
	}
	close(release)
	(<-finished).MustStatus(t, 200)
	var cost, input, output int64
	var state string
	ctx := context.Background()
	if err := env.App.Pool.QueryRow(ctx, `SELECT state, input_tokens, output_tokens, cost_micros FROM ai_budget_reservations WHERE workspace_id = $1 AND feature = 'draft'`, ws.ID).Scan(&state, &input, &output, &cost); err != nil {
		t.Fatal(err)
	}
	if state != "settled" || input != 113 || output != 29 || cost != 542 {
		t.Fatalf("settlement: %s %d %d %d", state, input, output, cost)
	}
	// Missing usage retains the whole reservation; an apparently valid
	// answer cannot become a reviewable proposal billed as zero.
	mode.Store(2)
	a.Do("POST", base+"/drafts/campaigns", in).MustStatus(t, 409)
	var held int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT input_tokens + output_tokens FROM ai_budget_reservations WHERE workspace_id = $1 AND state = 'uncertain'`, ws.ID).Scan(&held); err != nil || held != 9716 {
		t.Fatalf("unknown usage reservation: %d %v", held, err)
	}
	before = requests.Load()
	b.Do("POST", base+"/drafts/campaigns", in).MustStatus(t, 409)
	if requests.Load() != before {
		t.Fatal("unknown usage was silently released")
	}
	var settings ai.AISettings
	a.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &settings)
	// Both the connection probe and the draft used 113 + 29 tokens.
	if settings.Usage.UncertainRequests != 1 || settings.Usage.ReservedTokens != 9716 || settings.Usage.Tokens != 284 {
		t.Fatalf("usage must distinguish measured and reserved: %+v", settings.Usage)
	}
	// A model change with unknown prices must not inherit the old prices.
	a.Do("PUT", base+"/cap", map[string]any{"kind": "none"}).MustStatus(t, 200)
	a.Do("PUT", base+"/model", map[string]any{"model": "unknown-model"}).MustStatus(t, 200)
	before = requests.Load()
	a.Do("POST", base+"/drafts/campaigns", in).MustStatus(t, 409)
	if requests.Load() != before {
		t.Fatal("unpriced model was called")
	}
	// Zero is a valid configured price, distinct from an absent price.
	a.Do("PUT", base+"/prices", map[string]any{"input": 0, "output": 0}).MustStatus(t, 200)
	mode.Store(0)
	a.Do("POST", base+"/drafts/campaigns", in).MustStatus(t, 200)

	// Reconciliation binds the original prices, not today's free model.
	// A non-manager cannot release the hold, nor can a manager change it
	// while its response might still arrive. Retrying a decision is safe.
	var holds ai.AIBudgetHolds
	a.Do("GET", base+"/budget-holds", nil).MustStatus(t, 200).JSON(t, &holds)
	if len(holds.Items) != 1 {
		t.Fatalf("holds: %+v", holds)
	}
	id := holds.Items[0].ID
	path := base + "/budget-holds/" + id.String() + "/reconcile"
	counts := ai.AIBudgetReconcileInput{InputTokens: 17, OutputTokens: 11, Confirmed: true}
	b.Do("POST", path, counts).MustStatus(t, 403)
	a.Do("POST", path, counts).MustStatus(t, 409)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_budget_reservations SET created_at = now() - interval '11 minutes' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	a.Do("POST", path, counts).MustStatus(t, 204)
	a.Do("POST", path, counts).MustStatus(t, 204)
	counts.OutputTokens++
	a.Do("POST", path, counts).MustStatus(t, 409)
	if err := env.App.Pool.QueryRow(ctx, `SELECT cost_micros FROM ai_requests WHERE id = $1 AND usage_known`, id).Scan(&cost); err != nil || cost != 128 {
		t.Fatalf("reconciled with original prices: %d %v", cost, err)
	}
	var audits int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'ai.usage_reconciled'`, ws.ID).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("reconciliation audit: %d %v", audits, err)
	}
	a.Do("GET", base+"/budget-holds", nil).MustStatus(t, 200).JSON(t, &holds)
	if len(holds.Items) != 0 {
		t.Fatal("settled hold still blocks the budget")
	}
}

func TestAIConnectionTestBudget(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@probe.test"})
	manager := env.CreateUser(apptest.UserOpts{Email: "manager@probe.test"})
	ws := env.CreateWorkspace("Connection probes", owner.ID)
	env.AddMember(ws, manager.ID, "viewer")
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	role := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO roles(id,workspace_id,key,name,permissions) VALUES($1,$2,'model-manager','Model manager',$3)`, role, ws.ID, []string{"integrations:view", "integrations:manage"})
	exec(`UPDATE workspace_members SET role_id=$1 WHERE workspace_id=$2 AND user_id=$3`, role, ws.ID, manager.ID)
	c := env.Client()
	c.SignInDemo(manager.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	var calls atomic.Int32
	var mode atomic.Int32
	entered, release := make(chan struct{}, 1), make(chan struct{})
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		calls.Add(1)
		if mode.Load() == 1 {
			entered <- struct{}{}
			<-release
		}
		if mode.Load() == 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"overloaded"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Ready."},"finish_reason":"stop"}],"usage":{"prompt_tokens":113,"completion_tokens":7}}`))
	}))
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		model.Close()
	}()
	in := map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "probe-model"}
	c.Do("POST", base+"/test", in).MustStatus(t, 409)
	if calls.Load() != 0 {
		t.Fatal("unknown price reached the provider")
	}
	in["inputPrice"], in["outputPrice"] = 3, 7
	// A manager without ai:use may run only the fixed onboarding probe.
	c.Do("POST", base+"/test", in).MustStatus(t, 200)
	var s ai.AISettings
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &s)
	if s.Connected || s.Usage.Requests != 1 || s.Usage.Tokens != 120 || s.Usage.CostMicros != 388 {
		t.Fatalf("onboarding usage missing: %+v", s.Usage)
	}
	c.Do("POST", base+"/drafts/campaigns", ai.AICampaignBriefInput{SharingReviewed: true, Brief: "A useful update", Name: "News", Language: "en"}).MustStatus(t, 403)
	in["capKind"], in["capTokens"] = "tokens", 10000
	c.Do("PUT", base+"/connection", in).MustStatus(t, 200)
	c.Do("POST", base+"/test", map[string]any{}).MustStatus(t, 200)
	c.Do("PUT", base+"/key", map[string]any{"apiKey": "local-fixture-key"}).MustStatus(t, 200)
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &s)
	if s.Usage.Requests != 4 || s.Usage.Tokens != 480 || s.Usage.CostMicros != 1552 {
		t.Fatalf("connect, saved test and replacement must all count: %+v", s.Usage)
	}
	// An in-flight ordinary request leaves no room for a test. Disconnection
	// and candidate prices cannot make that outstanding spend disappear.
	exec(`INSERT INTO ai_budget_reservations(id,workspace_id,feature,provider,model,pricing_revision,input_price,output_price,state,input_tokens,output_tokens,cost_micros,created_at)
		VALUES($1,$2,'agent','compatible','probe-model','fixture',3,7,'reserved',9000,520,30640,now())`, uuid.Must(uuid.NewV7()), ws.ID)
	before := calls.Load()
	c.Do("POST", base+"/test", map[string]any{}).MustStatus(t, 409)
	c.Do("PUT", base+"/key", map[string]any{"apiKey": "other-local-key"}).MustStatus(t, 409)
	c.Do("DELETE", base+"/connection", nil).MustStatus(t, 200)
	c.Do("POST", base+"/test", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "probe-model", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 409)
	c.Do("PUT", base+"/connection", in).MustStatus(t, 409)
	if calls.Load() != before {
		t.Fatal("a connection operation bypassed a monthly reservation")
	}
	exec(`DELETE FROM ai_budget_reservations WHERE workspace_id=$1 AND pricing_revision='fixture'`, ws.ID)
	c.Do("PUT", base+"/connection", in).MustStatus(t, 200)
	// A late successful key test cannot restore a setting edited meanwhile.
	mode.Store(1)
	finished := make(chan *apptest.Response, 1)
	go func() { finished <- c.Do("PUT", base+"/key", map[string]any{"apiKey": "late-local-key"}) }()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("key probe did not reach provider")
	}
	c.Do("PUT", base+"/cap", map[string]any{"kind": "tokens", "tokens": 20000}).MustStatus(t, 200)
	close(release)
	(<-finished).MustStatus(t, 409)
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &s)
	if s.Connection.Cap.Tokens != 20000 || s.Usage.Requests != 6 {
		t.Fatalf("late key probe lost current settings or its charge: %+v", s)
	}
	// A 503 may have been billed. Do not retry or release its reservation.
	mode.Store(2)
	before = calls.Load()
	c.Do("POST", base+"/test", map[string]any{}).MustStatus(t, 200)
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &s)
	if calls.Load() != before+1 || s.Usage.UncertainRequests != 1 || s.Usage.ReservedTokens == 0 {
		t.Fatalf("failed probe retried or lost usage uncertainty: calls=%d usage=%+v", calls.Load()-before, s.Usage)
	}
}
