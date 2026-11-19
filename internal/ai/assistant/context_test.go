package assistant_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/assistant"
)

// messagesOf are a request's messages, as role and text.
func messagesOf(body map[string]any) [][2]string {
	out := [][2]string{}
	msgs, _ := body["messages"].([]any)
	for _, m := range msgs {
		mm := m.(map[string]any)
		text, _ := mm["content"].(string)
		out = append(out, [2]string{mm["role"].(string), text})
	}
	return out
}

// A thread's requests keep their beginning from one to the next (for the
// providers' prompt caches): the same instruction and tools, then the
// history; what changes between requests, the screen and the attached
// resources, comes last.
func TestAssistantRequestPrefixIsStable(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{"screen": "campaign.report"})
	for _, text := range []string{"First question", "Second question"} {
		turn := f.send(f.mc, th.ID, text).Turn.ID
		if err := f.worker().ExecuteTurn(context.Background(), f.job(turn)); err != nil {
			t.Fatal(err)
		}
	}
	b := f.streams.streamed(t)
	first, second := messagesOf(b[0]), messagesOf(b[1])
	if first[0][0] != "system" || first[0] != second[0] || strings.Contains(first[0][1], "<screen>") {
		t.Fatalf("the instruction: %q, then %q", first[0][1], second[0][1])
	}
	if !slices.Equal(toolNames(b[0]), toolNames(b[1])) {
		t.Fatalf("tools: %v, then %v", toolNames(b[0]), toolNames(b[1]))
	}
	if !slices.Equal(first[:len(first)-1], second[:len(first)-1]) {
		t.Fatalf("the second request doesn't begin with the first:\n%v\n%v", first, second)
	}
	for _, m := range [][][2]string{first, second} {
		if last := m[len(m)-1]; last[0] != "user" || !strings.Contains(last[1], "<screen>campaign.report</screen>") {
			t.Fatalf("the last message: %v", last)
		}
	}
}

// Past 60% of the model's window (32k tokens for one the price list
// doesn't know), earlier messages are summarized in a step of their own,
// with its own reservation; the answer's request has the summary instead
// of them, and the thread shows where it was made.
func TestAssistantSummarizes(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	long := strings.Repeat("word ", 3980)
	var last string
	for i := range 4 {
		turn := f.send(f.mc, th.ID, long+string(rune('A'+i))).Turn.ID
		if err := f.worker().ExecuteTurn(context.Background(), f.job(turn)); err != nil {
			t.Fatal(err)
		}
		last = turn.String()
	}
	b := f.streams.streamed(t)
	if len(b) != 5 {
		t.Fatalf("%d requests, want 3 answers, a summary and an answer", len(b))
	}
	sum, ans := messagesOf(b[3]), messagesOf(b[4])
	if !strings.HasPrefix(sum[0][1], "Summarize the earlier part") || len(toolNames(b[3])) != 0 {
		t.Fatalf("the summary's request: %v", sum[0])
	}
	longs := 0
	for _, m := range ans {
		if strings.Contains(m[1], long) {
			longs++
		}
	}
	if !strings.Contains(ans[1][1], "<summary_of_earlier_messages>") || longs != 1 {
		t.Fatalf("the answer's request: %d long messages, second %q", longs, ans[1][1][:min(80, len(ans[1][1]))])
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_steps s JOIN ai_budget_reservations b ON b.id = s.id WHERE s.turn_id = $1::uuid AND s.kind = 'summary'`, last); n != 1 {
		t.Fatalf("%d summary steps with a reservation", n)
	}
	after := f.get(f.mc, th.ID)
	if !slices.ContainsFunc(after.Messages, func(m assistant.AIThreadMessage) bool { return m.Role == "summary" }) {
		t.Fatal("no summary marker in the thread")
	}
}

// contentGone fails unless nothing of a thread's content is left: no
// sealed row of it, and the sentinel nowhere in plain text.
func (f *fixture) contentGone(thread, sentinel string) {
	f.t.Helper()
	for _, q := range []string{
		`SELECT count(*) FROM ai_assistant_messages WHERE thread_id = $1::uuid`,
		`SELECT count(*) FROM ai_assistant_contexts WHERE thread_id = $1::uuid`,
		`SELECT count(*) FROM ai_assistant_events WHERE thread_id = $1::uuid`,
		`SELECT count(*) FROM ai_assistant_threads WHERE id = $1::uuid AND title_sealed IS NOT NULL`,
	} {
		if n := f.count(q, thread); n != 0 {
			f.t.Fatalf("%d rows left: %s", n, q)
		}
	}
	for _, q := range []string{
		`SELECT count(*) FROM audit_events WHERE row_to_json(audit_events.*)::text LIKE '%' || $1 || '%'`,
		`SELECT count(*) FROM ai_requests WHERE row_to_json(ai_requests.*)::text LIKE '%' || $1 || '%'`,
		`SELECT count(*) FROM river_job WHERE args::text LIKE '%' || $1 || '%'`,
		`SELECT count(*) FROM notifications WHERE row_to_json(notifications.*)::text LIKE '%' || $1 || '%'`,
	} {
		if n := f.count(q, sentinel); n != 0 {
			f.t.Fatalf("the sentinel in plain text: %s", q)
		}
	}
}

// Retention: stream events go 24 hours after their turn; the content 30
// days after the last activity, leaving a tombstone with the usage; the
// tombstone 90 days later. Deleting a thread removes its content at once.
func TestAssistantRetention(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	const sentinel = "Retention sentinel 4d2e"
	th := f.thread(f.mc, map[string]any{"title": sentinel})
	turn := f.send(f.mc, th.ID, sentinel+" question").Turn.ID
	if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	gone := f.thread(f.mc, map[string]any{"title": sentinel + " too"})
	f.send(f.mc, gone.ID, sentinel+" again")
	f.mc.Do(http.MethodDelete, f.base+"/assistant/threads/"+gone.ID.String(), nil).MustStatus(t, 204)
	f.contentGone(gone.ID.String(), "4d2e again")
	start := time.Now()
	at := func(d time.Duration) *assistant.Service {
		s := f.worker()
		s.Now = func() time.Time { return start.Add(d) }
		return s
	}
	if r, err := at(25 * time.Hour).Purge(ctx); err != nil || r.Events == 0 || r.Expired != 0 {
		t.Fatalf("a day later: %+v %v", r, err)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_events WHERE thread_id = $1`, th.ID); n != 0 || len(f.get(f.mc, th.ID).Messages) != 2 {
		t.Fatalf("%d events a day after the turn ended", n)
	}
	if r, err := at(31 * 24 * time.Hour).Purge(ctx); err != nil || r.Expired != 1 {
		t.Fatalf("31 days later: %+v %v", r, err)
	}
	after := f.get(f.mc, th.ID)
	if after.State != "expired" || after.Title != "" || len(after.Messages) != 0 || after.Usage.Requests != 1 {
		t.Fatalf("the tombstone: %+v", after)
	}
	// With its word: four hex digits alone can be part of an id.
	f.contentGone(th.ID.String(), "sentinel 4d2e")
	if r, err := at(122 * 24 * time.Hour).Purge(ctx); err != nil || r.Tombstones != 1 {
		t.Fatalf("90 days after that: %+v %v", r, err)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_threads WHERE id = $1`, th.ID); n != 0 {
		t.Fatal("the tombstone is still there")
	}
}

// A worker that checks in after retention removed its thread's content
// restores nothing: its turn ended with the content, so its last batches
// and its answer are dropped, and its request is still accounted.
func TestAssistantLateCheckpointAfterRetention(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "A slow answer please").Turn.ID
	release := make(chan struct{})
	f.streams.set(held(release, "Before ", "after ", "the purge."))
	done := make(chan error, 1)
	args := f.job(turn)
	go func() { done <- f.worker().ExecuteTurn(ctx, args) }()
	f.waitKind(th.ID, "delta")
	purger := f.worker()
	purger.Now = func() time.Time { return time.Now().Add(31 * 24 * time.Hour) }
	if r, err := purger.Purge(ctx); err != nil || r.Expired != 1 {
		t.Fatalf("the purge: %+v %v", r, err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	f.contentGone(th.ID.String(), "the purge")
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = $1 AND state = 'failed' AND reason = 'expired'`, turn); n != 1 {
		t.Fatal("the turn didn't end with the content")
	}
	if n := f.count(`SELECT count(*) FROM ai_requests r JOIN ai_assistant_steps s ON s.id = r.id WHERE s.turn_id = $1 AND r.outcome = 'cancelled'`, turn); n != 1 {
		t.Fatal("the late request isn't accounted")
	}
}

// Removing a member, or a member leaving, deletes their threads in that
// workspace with all their content; deactivation keeps them.
func TestAssistantThreadsGoWithTheMember(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	const sentinel = "Member sentinel 77ab"
	mine := f.thread(f.mc, map[string]any{"title": sentinel})
	turn := f.send(f.mc, mine.ID, sentinel+" question").Turn.ID
	if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	theirs := f.thread(f.cc, map[string]any{"title": "Colleague's"})
	var member string
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT id::text FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, f.ws.ID, f.member).Scan(&member); err != nil {
		t.Fatal(err)
	}
	ws := "/workspaces/" + f.ws.Slug
	f.oc.Do(http.MethodPost, ws+"/members/"+member+"/deactivate", map[string]any{}).MustStatus(t, 200)
	if n := f.count(`SELECT count(*) FROM ai_assistant_messages WHERE thread_id = $1`, mine.ID); n != 2 {
		t.Fatalf("deactivation left %d messages, want 2", n)
	}
	// A notice of a call waiting in one of their threads goes with them;
	// their other notices stay.
	for _, kind := range []string{"ai_assistant_confirmation", "ai_usage"} {
		if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, title, data) VALUES ($1, $2, $3, $4, 'Waiting', $5)`,
			uuid.New(), f.member, f.ws.ID, kind, `{"thread_id":"`+mine.ID.String()+`"}`); err != nil {
			t.Fatal(err)
		}
	}
	f.oc.Do(http.MethodDelete, ws+"/members/"+member, nil).MustStatus(t, 204)
	if n := f.count(`SELECT count(*) FROM ai_assistant_threads WHERE workspace_id = $1 AND user_id = $2`, f.ws.ID, f.member); n != 0 {
		t.Fatalf("%d threads of the removed member", n)
	}
	if n := f.count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'ai_assistant_confirmation'`, f.member); n != 0 {
		t.Fatal("the removed member's confirmation notice is kept")
	}
	if n := f.count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'ai_usage'`, f.member); n != 1 {
		t.Fatal("the removal took another notice")
	}
	f.contentGone(mine.ID.String(), "sentinel 77ab")
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'member.removed' AND (after->>'assistant_threads_removed')::int = 1`, f.ws.ID); n != 1 {
		t.Fatal("the removal doesn't say the threads went")
	}
	f.cc.Do(http.MethodPost, ws+"/leave", nil).MustStatus(t, 204)
	if n := f.count(`SELECT count(*) FROM ai_assistant_threads WHERE id = $1`, theirs.ID); n != 0 {
		t.Fatal("the thread of a member who left is still there")
	}
}
