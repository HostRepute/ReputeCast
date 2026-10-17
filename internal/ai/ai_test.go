package ai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// fakeModel is an OpenAI-compatible server (what Ollama, vLLM and LM Studio
// speak) that answers each feature's prompt and records every request, so
// the tests can read exactly what left the instance.
type fakeModel struct {
	mu       sync.Mutex
	bodies   []string
	mode     string // "", "overloaded", "rate_limit", "auth", "refuse"
	srv      *httptest.Server
	requests int
}

func newFakeModel(t *testing.T) *fakeModel {
	f := &fakeModel{}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeModel) set(mode string) {
	f.mu.Lock()
	f.mode = mode
	f.mu.Unlock()
}

func (f *fakeModel) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.bodies...)
}

func (f *fakeModel) reset() {
	f.mu.Lock()
	f.bodies = nil
	f.mu.Unlock()
}

func (f *fakeModel) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.bodies = append(f.bodies, string(body))
	f.requests++
	mode := f.mode
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/v1/models" {
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"llama3.3:70b","object":"model"},{"id":"gemma3:27b","object":"model"}]}`))
		return
	}
	switch mode {
	case "overloaded":
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"server busy, please try again.","type":"api_error"}}`))
		return
	case "rate_limit":
		w.Header().Set("Retry-After", "20")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"too many requests","type":"rate_limit_exceeded","code":"rate_limit_exceeded"}}`))
		return
	case "auth":
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","code":"invalid_api_key"}}`))
		return
	}
	text := string(body)
	var answer string
	switch {
	case mode == "refuse":
		answer = `{"refusal":"I can't write a subject that pretends to be a reply to an email the reader never sent."}`
	case strings.Contains(text, "Reply with the word Ready."):
		answer = "Ready."
	case strings.Contains(text, "five different subject lines"):
		answer = `{"subjects":["Scheduled rollouts are here","Plan a flag change for 2 a.m.","Rollouts, exports and the v1 API","You won't believe how fast exports got","Weekly Digest #143: rollouts, exports"]}`
	case strings.Contains(text, "Make this text shorter"):
		// The model changed the token's default: it must come back as written.
		answer = `{"text":"Hi {{ first_name | default: \"you\" }}, flag changes can now go out on a schedule. <a href=\"https://evil.example/x\">See how</a>."}`
	case strings.Contains(text, "Translate this email"):
		answer = translateAnswer(text)
	case strings.Contains(text, "alt text for this image"):
		answer = `{"alt":"The flag editor's schedule picker, with a rollout set for 02:00 in New York."}`
	case strings.Contains(text, "link's text doesn't say"):
		answer = `{"text":"Read the v2 migration guide"}`
	case strings.Contains(text, "Draft the conditions"):
		if strings.Contains(text, "phone") {
			answer = `{"nothing":"There's no condition for “opened on a phone”."}`
		} else {
			answer = `{"groups":[{"match":"all","conditions":[{"field":"clicked","op":"in_last","amount":30,"unit":"days"},{"field":"last_open","op":"not_in_last","amount":14,"unit":"days"},{"field":"tier","op":"is","value":"Active"},{"field":"shoe_size","op":"is","value":"9"}]}],"note":"There's no condition for a specific link, so I used “Clicked any email”."}`
		}
	case strings.Contains(text, "plain-language summary"):
		answer = `{"summary":"October Product Update reached 47,548 inboxes, 99.1% of the 47,980 sent. 41.2% opened it as people. 112 people unsubscribed."}`
	default:
		answer = `{"text":"ok"}`
	}
	res := map[string]any{"id": "chatcmpl-1", "object": "chat.completion", "model": "llama3.3:70b",
		"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": answer}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": 900, "completion_tokens": 120, "total_tokens": 1020}}
	_ = json.NewEncoder(w).Encode(res)
}

// translateAnswer translates every segment by prefixing "ES ", and drops
// the first token it sees, as a careless model would.
func translateAnswer(prompt string) string {
	var req struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	_ = json.Unmarshal([]byte(prompt), &req)
	user := req.Messages[len(req.Messages)-1].Content
	start := strings.Index(user, "<segments>\n")
	end := strings.Index(user, "\n</segments>")
	var segs []struct{ ID, Text string }
	if start >= 0 && end > start {
		_ = json.Unmarshal([]byte(user[start+len("<segments>\n"):end]), &segs)
	}
	out := []map[string]string{}
	dropped := false
	for _, s := range segs {
		t := "ES " + s.Text
		if !dropped && strings.Contains(t, "{{ first_name") {
			t = strings.Replace(t, `{{ first_name | default: "there" }}`, "amigo", 1)
			dropped = true
		}
		out = append(out, map[string]string{"id": s.ID, "text": t})
	}
	b, _ := json.Marshal(map[string]any{"subject": "ES subject", "preheader": "ES preheader", "segments": out})
	return string(b)
}

type settings struct {
	Connected    bool
	Disconnected bool
	Connection   *struct {
		Provider, Model, ModelName, Status string
		Key                                struct{ Set bool }
		Cap                                struct {
			Kind string
			USD  float64
		}
		Prices struct{ Input, Output float64 }
		Demo   bool
	}
	Usage struct {
		CostMicros, InputTokens, OutputTokens, Requests int64
		State                                           string
	}
	ByFeature []struct {
		Feature    string
		CostMicros int64
	}
	ByMember []struct {
		Name       string
		CostMicros int64
	}
	Features []struct {
		Key       string
		On        bool
		Available bool
	}
	CanManage    bool
	Helpers      []string
	AIUseRoles   []string
	WithoutAIUse []struct{ Name, Role string }
}

const ws = "/workspaces/acme-newsletter"

// connectFake swaps the demo's sample model for the fake server.
func connectFake(t *testing.T, c *apptest.Client, f *fakeModel, capUSD float64) {
	t.Helper()
	c.Do(http.MethodDelete, ws+"/ai/connection", nil).MustStatus(t, 200)
	c.Do(http.MethodPut, ws+"/ai/connection", map[string]any{
		"provider": "compatible", "label": "Ollama on the office network", "baseUrl": f.srv.URL + "/v1", "model": "gemma3:27b",
		"capKind": "usd", "capUsd": capUSD, "inputPrice": 2, "outputPrice": 10,
	}).MustStatus(t, 200)
}

func TestAIDemoSettings(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var s settings
	c.Do(http.MethodGet, ws+"/ai", nil).MustStatus(t, 200).JSON(t, &s)
	if !s.Connected || s.Connection == nil || s.Connection.Provider != "anthropic" || s.Connection.ModelName != "Claude Sonnet 5.5" || !s.Connection.Key.Set || !s.Connection.Demo {
		t.Fatalf("connection: %+v", s.Connection)
	}
	if s.Connection.Cap.USD != 25 || s.Connection.Prices.Input != 2 || s.Connection.Prices.Output != 10 {
		t.Fatalf("cap and prices: %+v", s.Connection)
	}
	// The brief's month: $7.84 from 1.62M input and 460K output tokens.
	if s.Usage.CostMicros != 7_840_000 || s.Usage.InputTokens != 1_620_000 || s.Usage.OutputTokens != 460_000 || s.Usage.Requests != 412 || s.Usage.State != "ok" {
		t.Fatalf("usage: %+v", s.Usage)
	}
	want := map[string]int64{"subject_ideas": 910_000, "rewrite": 1_640_000, "translate": 3_860_000, "alt_text": 420_000, "accessibility_fix": 190_000, "segment_draft": 240_000, "report_summary": 580_000}
	for _, f := range s.ByFeature {
		if want[f.Feature] != f.CostMicros {
			t.Fatalf("by feature %s: %d", f.Feature, f.CostMicros)
		}
	}
	if len(s.ByMember) != 4 || s.ByMember[0].Name != "Sofia Rossi" || s.ByMember[0].CostMicros != 4_970_000 || s.ByMember[3].Name != "Marcus Chen" || s.ByMember[3].CostMicros != 380_000 {
		t.Fatalf("by member: %+v", s.ByMember)
	}
	if strings.Join(s.AIUseRoles, ", ") != "Owner, Admin, Editor, Content Writer" || len(s.WithoutAIUse) != 1 || s.WithoutAIUse[0].Name != "Aisha Bello" {
		t.Fatalf("ai:use holders: %v %+v", s.AIUseRoles, s.WithoutAIUse)
	}
	// The activity log: the board's six latest rows, never the content.
	var log struct {
		Items []struct {
			Member, Feature, Action, On, Outcome, Detail string
			Tokens                                       int64
		}
		MonthRequests int64
	}
	c.Do(http.MethodGet, ws+"/ai/activity?limit=6", nil).MustStatus(t, 200).JSON(t, &log)
	if log.MonthRequests != 412 || len(log.Items) != 6 {
		t.Fatalf("log: %d rows of %d", len(log.Items), log.MonthRequests)
	}
	first, last := log.Items[0], log.Items[5]
	if first.Member != "Daniel Park" || first.Feature != "subject_ideas" || first.On != "Weekly Digest #143" || first.Tokens != 1240 || first.Outcome != "accepted" || first.Detail != "1 of 5" {
		t.Fatalf("latest row: %+v", first)
	}
	if last.Feature != "rewrite" || last.Action != "instruction" || last.On != "Weekly Digest #143 subject" || last.Outcome != "refused" || last.Tokens != 610 {
		t.Fatalf("sixth row: %+v", last)
	}
	// A Viewer sees the settings (integrations:view) but can't ask.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do(http.MethodGet, ws+"/ai", nil).MustStatus(t, 200)
	aisha.Do(http.MethodGet, ws+"/ai/status", nil).MustStatus(t, 403)
	aisha.Do(http.MethodPut, ws+"/ai/cap", map[string]any{"kind": "usd", "usd": 50}).MustStatus(t, 403)
	// The Content Writer uses the assistant; changing it needs integrations:manage.
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	var st struct {
		State, ModelName string
		CanManage        bool
		Helpers          []string
	}
	daniel.Do(http.MethodGet, ws+"/ai/status", nil).MustStatus(t, 200).JSON(t, &st)
	if st.State != "ready" || st.ModelName != "Claude Sonnet 5.5" || st.CanManage || len(st.Helpers) == 0 {
		t.Fatalf("status: %+v", st)
	}
	daniel.Do(http.MethodPut, ws+"/ai/features/translate", map[string]any{"on": false}).MustStatus(t, 403)
	// The demo's model answers in tests and development, from recordings.
	var kept struct {
		Kept *struct {
			Summary, KeptBy, ModelName string
		}
	}
	c.Do(http.MethodGet, ws+"/campaigns/"+campaignID(t, env, "October Product Update")+"/ai-summary", nil).MustStatus(t, 200).JSON(t, &kept)
	if kept.Kept == nil || kept.Kept.KeptBy != "Linh Nguyen" || !strings.HasPrefix(kept.Kept.Summary, "October Product Update reached 47,548 inboxes") || kept.Kept.ModelName != "Claude Sonnet 5.5" {
		t.Fatalf("kept summary: %+v", kept.Kept)
	}
	// The catalog's AI models group: Anthropic connected with this month's use.
	var hub struct {
		Groups []struct {
			Key     string
			Entries []struct {
				Key, Status string
				Facts       struct {
					AI *struct {
						ModelName  string
						CostMicros int64
						CapUSD     float64
						Connected  string
					}
				}
			}
		}
	}
	c.Do(http.MethodGet, ws+"/integrations/catalog", nil).MustStatus(t, 200).JSON(t, &hub)
	found := false
	for _, g := range hub.Groups {
		if g.Key != "ai" {
			continue
		}
		found = true
		if len(g.Entries) != 5 || g.Entries[0].Key != "anthropic" || g.Entries[0].Status != "connected" || g.Entries[0].Facts.AI == nil || g.Entries[0].Facts.AI.CostMicros != 7_840_000 ||
			g.Entries[1].Status != "not_connected" || g.Entries[1].Facts.AI == nil || g.Entries[1].Facts.AI.Connected != "Anthropic" {
			t.Fatalf("AI models group: %+v", g.Entries)
		}
	}
	if !found {
		t.Fatal("the catalog has no AI models group")
	}
}

func campaignID(t *testing.T, env *apptest.Env, name string) string {
	t.Helper()
	var id string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT c.id::text FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = 'acme-newsletter' AND c.name = $1`, name).Scan(&id); err != nil {
		t.Fatalf("campaign %s: %v", name, err)
	}
	return id
}

// The features send what they say and never contact data: no address, name
// or field value of any contact is in what the model receives, tokens go as
// written, and a token or link the model changed comes back from the source.
func TestAINeverSendsContactData(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	f := newFakeModel(t)
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	connectFake(t, c, f, 25)
	f.reset()

	digest := campaignID(t, env, "Weekly Digest #143")
	var mj, subject string
	if err := env.App.Pool.QueryRow(ctx, `SELECT mjml, subject FROM campaigns WHERE id = $1`, digest).Scan(&mj, &subject); err != nil {
		t.Fatal(err)
	}
	// Subject ideas, each checked by the subject heuristics.
	var ideas struct {
		RequestID string
		Ideas     []struct {
			Subject, Verdict, Phrase string
			Heuristics               []string
		}
		Sent struct {
			ProviderName string
			Characters   int
		}
	}
	c.Do(http.MethodPost, ws+"/ai/subject-ideas", map[string]any{"campaignId": digest, "subject": subject, "preheader": "Plan a flag change", "mjml": mj}).MustStatus(t, 200).JSON(t, &ideas)
	if len(ideas.Ideas) != 5 || ideas.Ideas[3].Verdict != "warn" || ideas.Ideas[3].Phrase != "you won't believe" || ideas.Ideas[0].Verdict != "pass" || ideas.Sent.Characters == 0 {
		t.Fatalf("ideas: %+v", ideas)
	}
	c.Do(http.MethodPost, ws+"/ai/requests/"+ideas.RequestID+"/outcome", map[string]any{"outcome": "accepted", "detail": "1 of 5"}).MustStatus(t, 204)

	// Shorten: the token and the link come back as the source has them.
	src := `Hi {{ first_name | default: "there" }}, you can now schedule flag changes ahead of time, in any timezone. <a href="https://docs.acme.io/scheduled-rollouts">Click here</a> to see how it works.`
	var rw struct {
		HTML                          string
		TokensRestored, LinksRestored int
		Tokens                        []string
		WordsBefore, WordsAfter       int
	}
	c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": src}).MustStatus(t, 200).JSON(t, &rw)
	if !strings.Contains(rw.HTML, `{{ first_name | default: "there" }}`) || !strings.Contains(rw.HTML, `href="https://docs.acme.io/scheduled-rollouts"`) || rw.TokensRestored != 1 || rw.LinksRestored != 1 ||
		len(rw.Tokens) != 1 || rw.Tokens[0] != "first_name" || rw.WordsAfter >= rw.WordsBefore {
		t.Fatalf("rewrite: %+v", rw)
	}

	// A reputation question isn't sent at all.
	before := len(f.sent())
	var rep struct {
		RequestID  string
		Reputation *struct{ HostRepute bool }
	}
	c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "instruction", "instruction": "Why do our emails land in spam at Outlook? Rewrite this so they don't.", "html": src}).
		MustStatus(t, 200).JSON(t, &rep)
	if rep.Reputation == nil || !rep.Reputation.HostRepute || rep.RequestID != "" || len(f.sent()) != before {
		t.Fatalf("reputation question: %+v, %d requests", rep, len(f.sent())-before)
	}

	// Translate: block by block; the dropped token is restored.
	var tr struct {
		RequestID      string
		SourceVersion  int
		TokensRestored int
		Blocks         []struct {
			Index          int
			TokensRestored int
			Segments       []struct {
				Path       []int
				Kind       string
				Suggestion string
			}
		}
		SubjectSuggestion string
	}
	c.Do(http.MethodPost, ws+"/ai/translate", map[string]any{"campaignId": digest, "language": "de"}).MustStatus(t, 200).JSON(t, &tr)
	if tr.SubjectSuggestion != "ES subject" || tr.TokensRestored < 1 || len(tr.Blocks) < 3 {
		t.Fatalf("translation: %+v", tr)
	}
	restored := false
	for _, b := range tr.Blocks {
		for _, s := range b.Segments {
			if strings.Contains(s.Suggestion, `{{ first_name | default: "there" }}`) && b.TokensRestored > 0 {
				restored = true
			}
		}
	}
	if !restored {
		t.Fatalf("the dropped token wasn't restored: %+v", tr.Blocks)
	}

	// Alt text: the image goes, resized, with the text around it.
	var alt struct {
		Text             string
		Short, NoImageOf bool
	}
	c.Do(http.MethodPost, ws+"/ai/alt-text", map[string]any{"campaignId": digest, "src": firstImage(t, env, mj), "context": "Scheduled rollouts are here"}).MustStatus(t, 200).JSON(t, &alt)
	if alt.Text == "" || !alt.Short || !alt.NoImageOf {
		t.Fatalf("alt text: %+v", alt)
	}
	// Link text for an accessibility issue.
	var lt struct {
		Text      string
		SaysWhere bool
	}
	c.Do(http.MethodPost, ws+"/ai/link-text", map[string]any{"campaignId": digest, "text": "https://docs.acme.io/migrate-v2", "href": "https://docs.acme.io/migrate-v2",
		"context": "The v1 API stops answering on 30 November. Migration guide: https://docs.acme.io/migrate-v2"}).MustStatus(t, 200).JSON(t, &lt)
	if lt.Text != "Read the v2 migration guide" || !lt.SaysWhere {
		t.Fatalf("link text: %+v", lt)
	}
	// Describe the segment: the builder's conditions only, counted.
	var draft struct {
		Added  int
		Note   string
		Count  *struct{ Count, Total int64 }
		Groups []struct {
			Conditions []struct{ Field, Mark string }
		}
		Nothing string
	}
	c.Do(http.MethodPost, ws+"/ai/segment-draft", map[string]any{"description": "Clicked something in the last 30 days but haven't opened anything in two weeks, active only", "current": map[string]any{"groups": []any{}}}).
		MustStatus(t, 200).JSON(t, &draft)
	if draft.Added != 3 || draft.Count == nil || draft.Count.Total == 0 || !strings.Contains(draft.Note, "Clicked any email") || len(draft.Groups) != 1 || len(draft.Groups[0].Conditions) != 3 {
		t.Fatalf("segment draft: %+v", draft)
	}
	c.Do(http.MethodPost, ws+"/ai/segment-draft", map[string]any{"description": "People who opened on a phone last week", "current": map[string]any{"groups": []any{}}}).
		MustStatus(t, 200).JSON(t, &draft)
	if draft.Nothing == "" {
		t.Fatalf("nothing drafted: %+v", draft)
	}
	// A report summary: aggregate numbers only.
	var sum struct {
		Summary      string
		NumbersMatch bool
		NoCauses     bool
	}
	c.Do(http.MethodPost, ws+"/ai/report-summary", map[string]any{"campaignId": campaignID(t, env, "October Product Update")}).MustStatus(t, 200).JSON(t, &sum)
	if !strings.HasPrefix(sum.Summary, "October Product Update reached") || !sum.NoCauses {
		t.Fatalf("summary: %+v", sum)
	}

	// Nothing that left the instance names a contact: no address, no name,
	// no field value. Tokens went as written.
	sent := strings.Join(f.sent(), "\n")
	rows, err := env.App.Pool.Query(ctx, `SELECT c.email, coalesce(c.first_name, ''), coalesce(c.last_name, ''), coalesce(c.attributes::text, '{}')
		FROM contacts c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = 'acme-newsletter' ORDER BY c.created_at LIMIT 400`)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for rows.Next() {
		var email, first, last, attrs string
		if err := rows.Scan(&email, &first, &last, &attrs); err != nil {
			t.Fatal(err)
		}
		checked++
		if strings.Contains(sent, email) {
			t.Fatalf("a contact's address left the instance: %s", email)
		}
		if full := strings.TrimSpace(first + " " + last); len(full) > 6 && strings.Contains(sent, full) {
			t.Fatalf("a contact's name left the instance: %s", full)
		}
		var values map[string]any
		_ = json.Unmarshal([]byte(attrs), &values)
		for k, v := range values {
			if s, ok := v.(string); ok && len(s) > 12 && strings.Contains(sent, s) {
				t.Fatalf("a contact's %s value left the instance: %s", k, s)
			}
		}
	}
	rows.Close()
	if checked < 50 {
		t.Fatalf("checked only %d contacts", checked)
	}
	if !strings.Contains(sent, `{{ first_name | default: \"there\" }}`) {
		t.Fatal("tokens weren't sent as written")
	}
	// The log has each request, never its content.
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM ai_requests r JOIN workspaces w ON w.id = r.workspace_id WHERE w.slug = 'acme-newsletter' AND r.provider = 'compatible'`).Scan(&n)
	if n != 8 {
		t.Fatalf("activity log rows: %d", n)
	}
}

// firstImage returns the address of an email's first image.
func firstImage(t *testing.T, env *apptest.Env, mj string) string {
	t.Helper()
	i := strings.Index(mj, `src="`)
	if i < 0 {
		t.Fatal("no image in the email")
	}
	rest := mj[i+5:]
	return env.Server.URL + rest[:strings.Index(rest, `"`)]
}

// Connecting needs integrations:manage; the key is sealed, never returned;
// every change is audited; the cap stops requests and notifies once at 80%
// and once at the cap; errors change nothing.
func TestAIConnectionCapAndErrors(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	f := newFakeModel(t)
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	digest := campaignID(t, env, "Weekly Digest #143")

	// One model per workspace: another provider waits for a disconnect.
	c.Do(http.MethodPut, ws+"/ai/connection", map[string]any{"provider": "openai", "apiKey": "sk-test-other", "model": "gpt-5", "capKind": "usd", "capUsd": 25}).MustStatus(t, 409)
	// Test connection with the fake server, before saving.
	var test struct {
		OK          bool
		Answer      string
		PrivateHost string
		PrivateIP   string
	}
	c.Do(http.MethodPost, ws+"/ai/test", map[string]any{"provider": "compatible", "baseUrl": f.srv.URL + "/v1", "model": "gemma3:27b"}).MustStatus(t, 200).JSON(t, &test)
	if !test.OK || test.Answer != "Ready." || test.PrivateIP != "127.0.0.1" {
		t.Fatalf("test: %+v", test)
	}
	var models struct {
		Models []struct{ ID string }
	}
	c.Do(http.MethodPost, ws+"/ai/models", map[string]any{"provider": "compatible", "baseUrl": f.srv.URL + "/v1"}).MustStatus(t, 200).JSON(t, &models)
	if len(models.Models) != 2 || models.Models[0].ID != "llama3.3:70b" {
		t.Fatalf("models: %+v", models)
	}
	c.Do(http.MethodDelete, ws+"/ai/connection", nil).MustStatus(t, 200)
	key := "sk-local-secret-key-0042"
	r := c.Do(http.MethodPut, ws+"/ai/connection", map[string]any{"provider": "compatible", "baseUrl": f.srv.URL + "/v1", "apiKey": key, "model": "llama3.3:70b", "capKind": "usd", "capUsd": 10,
		"inputPrice": 2, "outputPrice": 10, "features": []string{"subject_ideas", "rewrite", "translate", "alt_text", "accessibility_fix", "segment_draft", "report_summary"}})
	r.MustStatus(t, 200)
	if strings.Contains(string(r.Body), key) {
		t.Fatal("the key came back")
	}
	var s settings
	r.JSON(t, &s)
	for _, ft := range s.Features {
		if ft.Key == "alt_text" && (ft.On || ft.Available) {
			t.Fatalf("a text-only model can't do alt text: %+v", ft)
		}
	}
	var stored []byte
	_ = env.App.Pool.QueryRow(ctx, `SELECT key_sealed FROM ai_connections c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = 'acme-newsletter'`).Scan(&stored)
	if len(stored) == 0 || strings.Contains(string(stored), key) {
		t.Fatal("the key isn't sealed")
	}
	// Alt text needs a model that reads images.
	c.Do(http.MethodPost, ws+"/ai/alt-text", map[string]any{"campaignId": digest, "src": "https://example.com/a.png"}).MustStatus(t, 409)

	// A feature switched off says so; the others keep working.
	c.Do(http.MethodPut, ws+"/ai/features/translate", map[string]any{"on": false}).MustStatus(t, 200)
	p := c.Do(http.MethodPost, ws+"/ai/translate", map[string]any{"campaignId": digest, "language": "de"}).MustStatus(t, 409).Problem(t)
	if p["code"] != "ai_feature_off" || !strings.Contains(p["detail"].(string), "Translation with AI is turned off for this workspace") {
		t.Fatalf("feature off: %v", p)
	}
	c.Do(http.MethodPut, ws+"/ai/features/translate", map[string]any{"on": true}).MustStatus(t, 200)

	// Overloaded twice: an error that says so, nothing changed, logged.
	f.set("overloaded")
	p = c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": "Hi there, this is a long paragraph to shorten."}).MustStatus(t, 502).Problem(t)
	if !strings.Contains(p["detail"].(string), "returned an error (503, ") || !strings.Contains(p["detail"].(string), "twice. Nothing was changed.") {
		t.Fatalf("overloaded: %v", p)
	}
	// Rate limited: try again in 20 s.
	f.set("rate_limit")
	r = c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": "Hi there, this is a long paragraph to shorten."})
	// 503, so the app doesn't read it as its own API's rate limit.
	p = r.MustStatus(t, 503).Problem(t)
	if p["code"] != "ai_rate_limited" || !strings.Contains(p["detail"].(string), "Try again in 20 s") {
		t.Fatalf("rate limit: %v", p)
	}
	// A refusal is an answer, in the model's words, and changes nothing.
	f.set("refuse")
	var refused struct{ Refusal string }
	c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "instruction", "instruction": "Make the subject look like a reply to them, start with Re:", "html": "Scheduled rollouts", "field": "subject"}).
		MustStatus(t, 200).JSON(t, &refused)
	if !strings.HasPrefix(refused.Refusal, "I can't write a subject") {
		t.Fatalf("refusal: %+v", refused)
	}
	f.set("")

	// The cap: $10.00, with the demo's $7.84 this month. Each fake answer
	// costs 900×2 + 120×10 micros.
	notices := func() int {
		var n int
		_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'ai_usage'`).Scan(&n)
		return n
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO ai_requests (id, workspace_id, feature, provider, model, cost_micros, outcome, created_at)
		SELECT gen_random_uuid(), w.id, 'rewrite', 'compatible', 'llama3.3:70b', 795000, 'accepted', now() FROM workspaces w WHERE w.slug = 'acme-newsletter'`); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": "Hi there, this is a long paragraph to shorten."}).MustStatus(t, 200)
	if n := notices(); n == 0 {
		t.Fatal("no 80% notice")
	}
	first := notices()
	c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": "Hi there, this is a long paragraph to shorten."}).MustStatus(t, 200)
	if notices() != first {
		t.Fatal("the 80% notice went out twice")
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_requests SET cost_micros = 2200000 WHERE cost_micros = 795000`); err != nil {
		t.Fatal(err)
	}
	f.reset()
	p = c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": "Hi there, this is a long paragraph to shorten."}).MustStatus(t, 409).Problem(t)
	if p["code"] != "ai_cap_reached" || !strings.Contains(p["detail"].(string), "This month's AI budget ($10.00) is used up.") || len(f.sent()) != 0 {
		t.Fatalf("cap reached: %v (%d sent)", p, len(f.sent()))
	}
	c.Do(http.MethodPut, ws+"/ai/cap", map[string]any{"kind": "usd", "usd": 50}).MustStatus(t, 200)

	// A rejected key: the connection says so, and the members who can replace
	// it are told once.
	f.set("auth")
	c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": "Hi there, this is a long paragraph to shorten."}).MustStatus(t, 502)
	f.reset()
	p = c.Do(http.MethodPost, ws+"/ai/rewrite", map[string]any{"campaignId": digest, "action": "shorten", "html": "Hi there, this is a long paragraph to shorten."}).MustStatus(t, 409).Problem(t)
	if p["code"] != "ai_key_rejected" || len(f.sent()) != 0 {
		t.Fatalf("key rejected: %v", p)
	}
	var keyNotes int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(DISTINCT user_id) FROM notifications WHERE kind = 'ai_key_rejected'`).Scan(&keyNotes)
	if keyNotes == 0 {
		t.Fatal("nobody was told about the rejected key")
	}
	f.set("")
	var rk struct {
		Settings settings
		Test     struct{ OK bool }
	}
	c.Do(http.MethodPut, ws+"/ai/key", map[string]any{"apiKey": "sk-local-new-key"}).MustStatus(t, 200).JSON(t, &rk)
	if !rk.Test.OK || rk.Settings.Connection.Status != "connected" {
		t.Fatalf("replaced key: %+v", rk)
	}

	// The configuration's changes are audited, without the key.
	var actions []string
	rows, err := env.App.Pool.Query(ctx, `SELECT action, coalesce(after::text, '') || coalesce(metadata::text, '') FROM audit_events WHERE action LIKE 'ai.%' AND occurred_at > now() - interval '1 hour' ORDER BY occurred_at`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a, body string
		_ = rows.Scan(&a, &body)
		if strings.Contains(body, key) || strings.Contains(body, "sk-local-new-key") {
			t.Fatalf("a key in the audit log: %s %s", a, body)
		}
		actions = append(actions, a)
	}
	rows.Close()
	for _, want := range []string{"ai.connection_removed", "ai.connection_created", "ai.feature_disabled", "ai.feature_enabled", "ai.cap_updated", "ai.key_replaced"} {
		found := false
		for _, a := range actions {
			if a == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("no %s in %v", want, actions)
		}
	}
	// Each entry has its details line (SettingsAudit).
	for action, want := range map[string]string{
		"ai.connection_created": "model llama3.3:70b · key added (write-only) · cap $10.00 a month · 6 of 7 features on",
		"ai.feature_disabled":   "Translate a language version turned off",
		"ai.cap_updated":        "cap $10.00 a month → $50.00 a month",
		"ai.key_replaced":       "key replaced (write-only) · test passed",
		"ai.connection_removed": "key deleted · the model, the cap and the prices are kept for the next connection",
	} {
		var got string
		_ = env.App.Pool.QueryRow(ctx, `SELECT metadata->>'summary' FROM audit_events WHERE action = $1 AND occurred_at > now() - interval '1 hour' ORDER BY occurred_at DESC LIMIT 1`, action).Scan(&got)
		if got != want {
			t.Fatalf("%s: %q, want %q", action, got, want)
		}
	}
	// Requests aren't audited: the activity log is their record.
	var audited int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action LIKE 'ai.%' AND action NOT IN ('ai.connection_removed','ai.connection_created','ai.connection_updated','ai.feature_disabled','ai.feature_enabled','ai.cap_updated','ai.prices_updated','ai.key_replaced','ai.model_changed')`).Scan(&audited)
	if audited != 0 {
		t.Fatalf("%d AI requests in the audit log", audited)
	}

	// API keys can't use the assistant: ai:use is for people.
	c.Do(http.MethodPost, ws+"/api-keys", map[string]any{"name": "AI key", "permissions": []string{"ai:use"}}).MustStatus(t, 403)
	var created struct{ Secret string }
	c.Do(http.MethodPost, ws+"/api-keys", map[string]any{"name": "Reports", "permissions": []string{"reports:view", "integrations:view"}}).MustStatus(t, 201).JSON(t, &created)
	bot := env.Client()
	bot.Bearer = created.Secret
	bot.Do(http.MethodGet, ws+"/ai/status", nil).MustStatus(t, 403)
	bot.Do(http.MethodPost, ws+"/ai/report-summary", map[string]any{"campaignId": campaignID(t, env, "October Product Update")}).MustStatus(t, 403)
	_ = time.Second
}

// An accepted translation writes the language version: every block
// accepted marks it translated, "Translated by AI · not reviewed" until an
// edit or "Mark as reviewed"; a summary kept on a report stays off
// read-only links.
func TestAITranslationAndSummary(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("sofia@acme.io")
	digest := campaignID(t, env, "Weekly Digest #143")
	// Vietnamese: a workspace language #143 has no version of.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO workspace_languages (workspace_id, code) SELECT id, 'fr' FROM workspaces WHERE slug = 'acme-newsletter' ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var tr struct {
		RequestID           string
		SourceVersion       int
		Status              string
		SubjectSuggestion   string
		PreheaderSuggestion string
		Blocks              []struct {
			Index    int
			Segments []struct {
				Path       []int
				Kind       string
				Suggestion string
			}
		}
	}
	c.Do(http.MethodPost, ws+"/ai/translate", map[string]any{"campaignId": digest, "language": "fr"}).MustStatus(t, 200).JSON(t, &tr)
	if tr.Status != "missing" || len(tr.Blocks) == 0 || tr.RequestID == "" {
		t.Fatalf("translation: %+v", tr)
	}
	blocks := []map[string]any{}
	for _, b := range tr.Blocks {
		segs := []map[string]any{}
		for _, s := range b.Segments {
			segs = append(segs, map[string]any{"path": s.Path, "kind": s.Kind, "text": "FR " + s.Suggestion})
		}
		blocks = append(blocks, map[string]any{"index": b.Index, "segments": segs})
	}
	items := len(tr.Blocks) + 2
	var acc struct {
		Status   string
		Version  int
		Accepted int
	}
	c.Do(http.MethodPost, ws+"/campaigns/"+digest+"/languages/fr/ai-translation", map[string]any{"requestId": tr.RequestID, "sourceVersion": tr.SourceVersion,
		"subject": "Déploiements programmés", "preheader": tr.PreheaderSuggestion, "blocks": blocks, "items": items}).MustStatus(t, 200).JSON(t, &acc)
	if acc.Status != "up_to_date" || acc.Accepted != items {
		t.Fatalf("accepted: %+v", acc)
	}
	var marks struct {
		Items []struct {
			Language, Model string
			Reviewed        bool
		}
	}
	c.Do(http.MethodGet, ws+"/campaigns/"+digest+"/ai-translations", nil).MustStatus(t, 200).JSON(t, &marks)
	if len(marks.Items) != 1 || marks.Items[0].Language != "fr" || marks.Items[0].Reviewed || marks.Items[0].Model != "Claude Sonnet 5.5" {
		t.Fatalf("marks: %+v", marks)
	}
	var subj, mj, status string
	if err := env.App.Pool.QueryRow(ctx, `SELECT subject, mjml, status FROM content_translations WHERE owner_id = $1 AND language = 'fr'`, digest).Scan(&subj, &mj, &status); err != nil {
		t.Fatal(err)
	}
	if subj != "Déploiements programmés" || status != "translated" || !strings.Contains(mj, "FR ") || !strings.Contains(mj, `{{ first_name | default: "there" }}`) {
		t.Fatalf("version: %s %s", subj, status)
	}
	// The same suggestion can't be accepted again after the source changed.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET version = version + 1 WHERE id = $1`, digest); err == nil {
		c.Do(http.MethodPost, ws+"/campaigns/"+digest+"/languages/fr/ai-translation", map[string]any{"requestId": tr.RequestID, "sourceVersion": tr.SourceVersion,
			"blocks": blocks, "items": items}).MustStatus(t, 409)
	}
	c.Do(http.MethodPost, ws+"/campaigns/"+digest+"/languages/fr/ai-reviewed", nil).MustStatus(t, 204)
	c.Do(http.MethodGet, ws+"/campaigns/"+digest+"/ai-translations", nil).MustStatus(t, 200).JSON(t, &marks)
	if len(marks.Items) != 1 || !marks.Items[0].Reviewed {
		t.Fatalf("reviewed: %+v", marks)
	}

	// A report summary from the demo's model, kept, then on the report only.
	oct := campaignID(t, env, "October Product Update")
	var sum struct {
		RequestID    string
		Summary      string
		NumbersMatch bool
	}
	c.Do(http.MethodPost, ws+"/ai/report-summary", map[string]any{"campaignId": oct}).MustStatus(t, 200).JSON(t, &sum)
	if !strings.HasPrefix(sum.Summary, "October Product Update reached") || !sum.NumbersMatch {
		t.Fatalf("summary: %+v", sum)
	}
	var kept struct {
		Kept *struct{ Summary, KeptBy string }
	}
	c.Do(http.MethodPut, ws+"/campaigns/"+oct+"/ai-summary", map[string]any{"requestId": sum.RequestID, "summary": sum.Summary}).MustStatus(t, 200).JSON(t, &kept)
	if kept.Kept == nil || kept.Kept.KeptBy != "Sofia Rossi" {
		t.Fatalf("kept: %+v", kept)
	}
	var outcome string
	_ = env.App.Pool.QueryRow(ctx, `SELECT outcome FROM ai_requests WHERE id = $1`, sum.RequestID).Scan(&outcome)
	if outcome != "kept" {
		t.Fatalf("outcome: %s", outcome)
	}
	// Someone else's suggestion can't be kept.
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	daniel.Do(http.MethodPut, ws+"/campaigns/"+oct+"/ai-summary", map[string]any{"requestId": sum.RequestID, "summary": "x"}).MustStatus(t, 409)
	// Read-only links never show it: the shared report has no summary.
	var link struct{ URL, Token string }
	c.Do(http.MethodPost, ws+"/campaigns/"+oct+"/report/shares", map[string]any{}).JSON(t, &link)
	if link.URL != "" {
		token := link.URL[strings.LastIndex(link.URL, "/")+1:]
		r := env.Client().Do(http.MethodGet, "/shared-reports/"+token, nil)
		if strings.Contains(string(r.Body), "Summary by AI") || strings.Contains(string(r.Body), sum.Summary[:40]) {
			t.Fatal("the read-only link shows the kept summary")
		}
	}
	c.Do(http.MethodDelete, ws+"/campaigns/"+oct+"/ai-summary", nil).MustStatus(t, 204)
	var gone struct{ Kept *struct{ Summary string } }
	c.Do(http.MethodGet, ws+"/campaigns/"+oct+"/ai-summary", nil).MustStatus(t, 200).JSON(t, &gone)
	if gone.Kept != nil {
		t.Fatal("the summary is still kept")
	}
}

// The log keeps 90 days.
func TestAIRetention(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_requests SET created_at = now() - interval '91 days' WHERE id IN (SELECT id FROM ai_requests ORDER BY created_at LIMIT 10)`); err != nil {
		t.Fatal(err)
	}
	svc := aiService(env)
	n, err := svc.Purge(ctx)
	if err != nil || n != 10 {
		t.Fatalf("purged %d: %v", n, err)
	}
}
