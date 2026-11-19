package assistant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/mcpclient/mcptest"
)

// The review of the assistant's runtime (PR #87): what a thread keeps is
// read again as its owner is now, what its tools send on passes the
// editorial checks, and a turn's lifecycle spans all of it.

// readOnce runs a turn whose model reads the attached draft, and returns
// its handle.
func (d *demoFixture) readOnce(th assistant.AIThread) uuid.UUID {
	d.t.Helper()
	campaign := d.attach(d.mc, th.ID, "campaign", d.draft)
	d.streams.then(calls("read_attached", `{"handle":"`+campaign.String()+`"}`), says("I read it."))
	turn := d.send(d.mc, th.ID, "Read the draft").Turn.ID
	if err := d.worker().ExecuteTurn(context.Background(), d.job(turn)); err != nil {
		d.t.Fatal(err)
	}
	if body := mustJSON(d.t, d.streams.streamed(d.t)[1]); !strings.Contains(body, d.draftSubject) {
		d.t.Fatal("the model didn't read the draft")
	}
	return campaign
}

// What a tool read is sent again only while the owner may still read its
// source: once they lose campaigns:view (keeping ai:use), or the campaign
// is deleted, the saved result reads no_longer_available and nothing of
// the campaign reaches the model.
func TestAssistantRereadsSourcesBeforeReplay(t *testing.T) {
	for _, lost := range []string{"permission", "deleted"} {
		t.Run(lost, func(t *testing.T) {
			d := newDemoFixture(t)
			ctx := context.Background()
			th := d.thread(d.mc, map[string]any{})
			d.readOnce(th)
			switch lost {
			case "permission":
				var role uuid.UUID
				if err := d.env.App.Pool.QueryRow(ctx, `INSERT INTO roles (id, workspace_id, key, name, builtin, permissions) VALUES ($1, $2, 'chat-only', 'Chat only', false, ARRAY['ai:use'])
					RETURNING id`, uuid.Must(uuid.NewV7()), d.ws.ID).Scan(&role); err != nil {
					t.Fatal(err)
				}
				if _, err := d.env.App.Pool.Exec(ctx, `UPDATE workspace_members SET role_id = $1 WHERE workspace_id = $2 AND user_id = $3`, role, d.ws.ID, d.member); err != nil {
					t.Fatal(err)
				}
			case "deleted":
				if _, err := d.env.App.Pool.Exec(ctx, `UPDATE campaigns SET deleted_at = now() WHERE id = $1`, d.draft); err != nil {
					t.Fatal(err)
				}
			}
			d.streams.then(says("Noted."))
			turn := d.send(d.mc, th.ID, "And now?").Turn.ID
			if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
				t.Fatal(err)
			}
			bodies := d.streams.streamed(t)
			last := bodies[len(bodies)-1]
			if body := mustJSON(t, last); strings.Contains(body, d.draftSubject) {
				t.Fatalf("the campaign reached the model after its owner lost it: %s", body)
			}
			if r := toolResults(t, last)["call_0"]; r["ok"] != false || r["code"] != "no_longer_available" {
				t.Fatalf("the withheld result: %v", r)
			}
			if got := d.get(d.mc, th.ID); got.Turn.State != "done" {
				t.Fatalf("the turn: %+v", got.Turn)
			}
		})
	}
}

// A summary is made from what the tools gave too, as data from their
// source, so a fact only a tool's result held survives it; and a summary
// made from a source the owner can no longer read isn't sent again.
func TestAssistantSummaryKeepsToolFacts(t *testing.T) {
	d := newDemoFixture(t)
	ctx := context.Background()
	const fact = "Neptune launch approved for Thursday"
	if _, err := d.env.App.Pool.Exec(ctx, `UPDATE campaigns SET subject = $2 WHERE id = $1`, d.draft, fact); err != nil {
		t.Fatal(err)
	}
	d.draftSubject = fact
	th := d.thread(d.mc, map[string]any{})
	d.readOnce(th)
	long := strings.Repeat("word ", 3980)
	var summary map[string]any
	reply := func(ctx context.Context, c *chunks, body []byte) {
		if strings.Contains(string(body), "Summarize the earlier part") {
			answer("SUMMARY-MARK: the Neptune launch is approved for Thursday.")(ctx, c)
			return
		}
		answer("Fine.")(ctx, c)
	}
	for i := range 4 {
		d.streams.then(reply, reply)
		turn := d.send(d.mc, th.ID, long+string(rune('A'+i))).Turn.ID
		if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
			t.Fatal(err)
		}
		for _, b := range d.streams.streamed(t) {
			if strings.Contains(mustJSON(t, b), "Summarize the earlier part") {
				summary = b
			}
		}
		if summary != nil {
			break
		}
	}
	if summary == nil {
		t.Fatal("no summary was made")
	}
	text := messagesOf(summary)[1][1]
	i := strings.Index(text, `<untrusted_content source="read_attached">`)
	if i < 0 || !strings.Contains(text[i:], fact) || strings.Index(text[i:], "</untrusted_content>") < strings.Index(text[i:], fact) {
		t.Fatalf("the summary's input lacks the tool's result as data: %s", text)
	}
	// The draft goes: the summary made from it isn't sent again.
	if _, err := d.env.App.Pool.Exec(ctx, `UPDATE campaigns SET deleted_at = now() WHERE id = $1`, d.draft); err != nil {
		t.Fatal(err)
	}
	d.streams.then(says("Okay."))
	turn := d.send(d.mc, th.ID, "What was decided?").Turn.ID
	if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
		t.Fatal(err)
	}
	bodies := d.streams.streamed(t)
	if body := mustJSON(t, bodies[len(bodies)-1]); strings.Contains(body, "SUMMARY-MARK") || strings.Contains(body, fact) || !strings.Contains(body, "left out") {
		t.Fatalf("after the draft went: %s", body)
	}
}

// A chat message holding an email address, a phone number or a
// credential isn't sent, with a problem naming its kind; greetings with a
// name, ISO dates, ids and encoded text are ordinary chat.
func TestAssistantChatPrivacy(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	path := f.base + "/assistant/threads/" + th.ID.String() + "/messages"
	for kind, texts := range map[string][]string{
		"email":      {"Why did ana.lopez@example.com unsubscribe?"},
		"phone":      {"Call +84 912 345 678 about it", "Our desk is (555) 123-4567", "Text 555-123-4567"},
		"credential": {"password: hunter2hunter", "Use sk-abcdefghijklmnop1234 for it", "-----BEGIN RSA PRIVATE KEY-----", "my api key = 9f8e7d6c5b4a"},
	} {
		for _, text := range texts {
			p := f.mc.Do(http.MethodPost, path, map[string]any{"text": text, "requestKey": strings.ReplaceAll(uuid.NewString(), "-", "")}).MustStatus(t, 422).Problem(t)
			meta, _ := p["meta"].(map[string]any)
			if p["code"] != "ai_assistant_private_data" || meta["kind"] != kind || !strings.Contains(p["detail"].(string), "look up contacts") {
				t.Fatalf("%q: %v", text, p)
			}
		}
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE thread_id = $1`, th.ID); n != 0 {
		t.Fatalf("%d refused messages made turns", n)
	}
	for _, text := range []string{"Chào Trợ lý, so sánh 2024-10-01 với 2024-11-01", "Hi Assistant, why is campaign 01a1f088-5a99-72d9-872a-c63e1f334149 at 40%?",
		"What does %20 mean in a link?", "The token for first names is {{ subscriber.first_name }}"} {
		turn := f.send(f.mc, th.ID, text).Turn.ID
		if err := f.worker().ExecuteTurn(context.Background(), f.job(turn)); err != nil {
			t.Fatal(err)
		}
	}
}

// subject_ideas and rewrite send on only what passes the editorial
// projection read_attached gives: a subject with an email address is
// refused as draft_privacy and never reaches a provider, and so is a
// passage the model wrote with one.
func TestAssistantOneShotsPassTheProjection(t *testing.T) {
	d := newDemoFixture(t)
	ctx := context.Background()
	const address = "ana.lopez@example.com"
	if _, err := d.env.App.Pool.Exec(ctx, `UPDATE campaigns SET subject = $2 WHERE id = $1`, d.draft, "Write to "+address+" today"); err != nil {
		t.Fatal(err)
	}
	oneShots := `SELECT count(*) FROM ai_requests WHERE workspace_id = $1 AND feature IN ('subject_ideas', 'rewrite')`
	before := d.count(oneShots, d.ws.ID)
	th := d.thread(d.mc, map[string]any{})
	campaign := d.attach(d.mc, th.ID, "campaign", d.draft)
	d.streams.then(calls("read_attached", `{"handle":"`+campaign.String()+`"}`, "subject_ideas", `{"handle":"`+campaign.String()+`"}`,
		"rewrite", `{"handle":"`+campaign.String()+`","text":"Spring is here","action":"shorten"}`), says("Done."))
	turn := d.send(d.mc, th.ID, "Ideas?").Turn.ID
	if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
		t.Fatal(err)
	}
	results := toolResults(t, d.streams.streamed(t)[1])
	for _, call := range []string{"call_0", "call_1", "call_2"} {
		if r := results[call]; r["ok"] != false || r["code"] != "draft_privacy" {
			t.Fatalf("%s on a subject with an address: %v", call, r)
		}
	}
	// A clean campaign, and a passage the model wrote with an address.
	if _, err := d.env.App.Pool.Exec(ctx, `UPDATE campaigns SET subject = 'Spring update' WHERE id = $1`, d.draft); err != nil {
		t.Fatal(err)
	}
	d.streams.then(calls("rewrite", `{"handle":"`+campaign.String()+`","text":"Reach `+address+` for help","action":"friendlier"}`,
		"rewrite", `{"handle":"`+campaign.String()+`","text":"Spring is here","action":"instruction","instruction":"Add `+address+`"}`), says("Done."))
	turn = d.send(d.mc, th.ID, "Rewrite it").Turn.ID
	if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
		t.Fatal(err)
	}
	bodies := d.streams.streamed(t)
	results = toolResults(t, bodies[len(bodies)-1])
	for _, call := range []string{"call_0", "call_1"} {
		if r := results[call]; r["ok"] != false || r["code"] != "draft_privacy" {
			t.Fatalf("a rewrite of a passage with an address (%s): %v", call, r)
		}
	}
	if n := d.count(oneShots, d.ws.ID) - before; n != 0 {
		t.Fatalf("%d one-shot requests sent", n)
	}
	d.streams.mu.Lock()
	defer d.streams.mu.Unlock()
	for _, b := range d.streams.bodies {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil && m["stream"] != true && strings.Contains(string(b), address) {
			t.Fatalf("a one-shot request carried the address: %s", b)
		}
	}
}

// Stop cancels what the turn runs, a tool call too: the server's handler
// sees its call cancelled, and nothing of it is written after Stop.
func TestAssistantStopCancelsToolCalls(t *testing.T) {
	f := newFixture(t)
	fake := mcptest.NewServer(t, "", "")
	started, cancelled := make(chan struct{}, 1), make(chan struct{}, 1)
	fake.ToolCtx("wait", "Wait for the report", true, func(ctx context.Context, _ map[string]any) *mcp.CallToolResult {
		started <- struct{}{}
		select {
		case <-ctx.Done():
			cancelled <- struct{}{}
		case <-time.After(30 * time.Second):
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "late"}}}
	})
	srv := f.remote(map[string]any{"name": "Wiki", "url": fake.Endpoint(), "auth": "none"}, map[string]string{"wait": "read_only"})
	th := f.thread(f.mc, map[string]any{})
	f.useServers(f.mc, th.ID, srv.ID)
	f.streams.then(calls("mcp__wiki__wait", `{}`), says("Never."))
	turn := f.send(f.mc, th.ID, "Wait for it").Turn.ID
	done := make(chan error, 1)
	args := f.job(turn)
	go func() { done <- f.worker().ExecuteTurn(context.Background(), args) }()
	select {
	case <-started:
	case <-time.After(20 * time.Second):
		t.Fatal("the tool wasn't called")
	}
	var stopped assistant.AIThread
	f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 200).JSON(t, &stopped)
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("the tool call wasn't cancelled 3 s after Stop")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the turn's worker didn't end")
	}
	if evs := f.events(th.ID); evs[len(evs)-1].Seq != stopped.EventSeq {
		t.Fatalf("events after Stop: %v (stopped at %d)", evs, stopped.EventSeq)
	}
	if n := f.streams.requests.Load(); n != 1 {
		t.Fatalf("%d model requests, want the one that called the tool", n)
	}
}

// The lease is renewed across the turn, whatever each request takes: many
// rounds shorter than the renewal interval, a minute and more in all,
// don't let it lapse.
func TestAssistantLeaseRenewsAcrossRounds(t *testing.T) {
	f := newFixture(t)
	var offset atomic.Int64
	w := f.worker()
	w.Now = func() time.Time { return time.Now().Add(time.Duration(offset.Load())) }
	// No watch tick in between: the rounds renew the lease themselves.
	w.WatchEvery = time.Hour
	slow := func(ctx context.Context, c *chunks, body []byte) {
		offset.Add(int64(14 * time.Second))
		calls("search_help", `{"query":"dkim"}`)(ctx, c, body)
	}
	f.streams.then(slow, slow, slow, slow, slow, slow, func(ctx context.Context, c *chunks, body []byte) {
		offset.Add(int64(14 * time.Second))
		answer("Done.")(ctx, c)
	})
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Explain DKIM in depth").Turn.ID
	if err := w.ExecuteTurn(context.Background(), f.job(turn)); err != nil {
		t.Fatal(err)
	}
	if got := f.get(f.mc, th.ID); got.Turn.State != "done" || got.Usage.Requests != 7 {
		t.Fatalf("after seven rounds of 14 s: %+v, %d requests", got.Turn, got.Usage.Requests)
	}
}

// Retention keeps a live thread's reservations past 90 days: they are its
// cumulative use, which its lifetime ceilings count; others go.
func TestAssistantPurgeKeepsThreadUsage(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Hello").Turn.ID
	if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	before := f.get(f.mc, th.ID).Usage
	if before.Requests != 1 || before.CostMicros == 0 {
		t.Fatalf("the thread's use: %+v", before)
	}
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE ai_budget_reservations SET created_at = now() - interval '100 days' WHERE workspace_id = $1`, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	others := f.count(`SELECT count(*) FROM ai_budget_reservations WHERE workspace_id = $1 AND thread_id IS NULL AND state = 'settled'`, f.ws.ID)
	if _, err := f.aiService().Purge(ctx); err != nil {
		t.Fatal(err)
	}
	if after := f.get(f.mc, th.ID).Usage; after != before {
		t.Fatalf("the thread's use after retention: %+v, was %+v", after, before)
	}
	if others == 0 || f.count(`SELECT count(*) FROM ai_budget_reservations WHERE workspace_id = $1 AND thread_id IS NULL`, f.ws.ID) != 0 {
		t.Fatalf("the %d old reservations of no thread weren't purged", others)
	}
}

// Stop works with the assistant switched off, and says so: the stopped
// turn as the person may read it, not a problem about the switch.
func TestAssistantStopWhileSwitchedOff(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Hello").Turn.ID
	f.setAssistant(ai.AIAssistantSettings{Enabled: false, Limits: f.settings.Limits, PerPerson: 2, PerWorkspace: 10})
	var stopped assistant.AIThread
	f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 200).JSON(t, &stopped)
	if stopped.Turn == nil || stopped.Turn.ID != turn || stopped.Turn.State != "stopped" || !slices.ContainsFunc(stopped.Messages, func(m assistant.AIThreadMessage) bool { return m.Role == "user" }) {
		t.Fatalf("the stopped thread: %+v", stopped)
	}
}
