package assistant_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/mcpclient/mcptest"
)

// lastConfirmation is the confirmation of a thread's latest confirmation
// event, as its streams read it.
func (f *fixture) lastConfirmation(thread uuid.UUID) assistant.AIThreadConfirmation {
	f.t.Helper()
	var seq int64
	var sealed []byte
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT seq, payload_sealed FROM ai_assistant_events WHERE thread_id = $1 AND kind = 'confirmation'
		ORDER BY seq DESC LIMIT 1`, thread).Scan(&seq, &sealed); err != nil {
		f.t.Fatal(err)
	}
	raw, err := f.env.App.Secrets.Open(sealed, fmt.Sprintf("ai_assistant_event:%s:%d", thread, seq))
	if err != nil {
		f.t.Fatal(err)
	}
	var e assistant.AIThreadConfirmationEvent
	if err := json.Unmarshal(raw, &e); err != nil {
		f.t.Fatal(err)
	}
	return e.Confirmation
}

// waitingCall starts a turn in a thread that stops at a writing tool's
// call, and returns the thread, the turn and the call.
func (f *fixture) waitingCall(thread uuid.UUID, title string) (uuid.UUID, assistant.AIThreadConfirmation) {
	f.t.Helper()
	f.streams.then(calls("mcp__linear__create_issue", `{"title":"`+title+`"}`))
	turn := f.send(f.mc, thread, "File an issue: "+title).Turn.ID
	f.execute(f.job(turn))
	th := f.get(f.mc, thread)
	if th.Turn.State != "waiting" || th.Confirmation == nil {
		f.t.Fatalf("not waiting: %+v", th.Turn)
	}
	return turn, *th.Confirmation
}

// A call waiting for its owner is in their notifications: the server and
// the tool, never the arguments, linking to the thread. The notice goes
// once the call is decided or stopped, or the thread is deleted; a person
// who turned the event off isn't notified.
func TestAssistantConfirmationNotice(t *testing.T) {
	f := newFixture(t)
	fake := mcptest.NewServer(t, "", "")
	fake.Tool("create_issue", "Create an issue", false, mcptest.Text("Created ENG-1"))
	srv := f.remote(map[string]any{"name": "Linear", "url": fake.Endpoint(), "auth": "none"}, map[string]string{"create_issue": "writes"})
	th := f.thread(f.mc, map[string]any{})
	f.useServers(f.mc, th.ID, srv.ID)
	notices := func() int {
		return f.count(`SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'ai_assistant_confirmation'`, f.member)
	}
	_, c := f.waitingCall(th.ID, "Pixel alpha")
	var link, body, data string
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT link, body, data::text FROM notifications WHERE user_id = $1 AND kind = 'ai_assistant_confirmation'
		AND in_app AND workspace_id = $2`, f.member, f.ws.ID).Scan(&link, &body, &data); err != nil {
		t.Fatal(err)
	}
	var d map[string]any
	_ = json.Unmarshal([]byte(data), &d)
	if link != "/w/"+f.ws.Slug+"/assistant/"+th.ID.String() || !strings.Contains(body, "Linear · create_issue") || d["thread_id"] != th.ID.String() ||
		d["confirmation_id"] != c.ID.String() || d["server"] != "Linear" || d["tool"] != "create_issue" {
		t.Fatalf("the notice: %s %q %s", link, body, data)
	}
	if n := f.count(`SELECT count(*) FROM notifications WHERE row_to_json(notifications.*)::text LIKE '%Pixel alpha%'`); n != 0 {
		t.Fatal("a notice has the call's arguments")
	}
	if n := f.count(`SELECT count(*) FROM notifications WHERE kind = 'ai_assistant_confirmation' AND user_id <> $1`, f.member); n != 0 {
		t.Fatal("someone else was told")
	}
	f.decide(f.mc, th.ID, c.ID, "allow", 200)
	if notices() != 0 {
		t.Fatal("the notice stayed after the decision")
	}
	f.streams.then(says("Created it."))
	f.execute(f.latestJob(c.TurnID))
	turn, _ := f.waitingCall(th.ID, "Pixel beta")
	if notices() != 1 {
		t.Fatal("the second call isn't in the notifications")
	}
	f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 200)
	if notices() != 0 {
		t.Fatal("the notice stayed after Stop")
	}
	f.waitingCall(th.ID, "Pixel gamma")
	f.mc.Do(http.MethodDelete, f.base+"/assistant/threads/"+th.ID.String(), nil).MustStatus(t, 204)
	if notices() != 0 {
		t.Fatal("the notice stayed after the thread was deleted")
	}
	// The event turned off: no notice.
	if _, err := f.env.App.Pool.Exec(context.Background(), `INSERT INTO notification_preferences (user_id, event_type, in_app, email) VALUES ($1, 'ai_assistant_confirmation', false, false)`,
		f.member); err != nil {
		t.Fatal(err)
	}
	other := f.thread(f.mc, map[string]any{})
	f.useServers(f.mc, other.ID, srv.ID)
	f.waitingCall(other.ID, "Pixel delta")
	if notices() != 0 {
		t.Fatal("a person who turned the event off was notified")
	}
	if n := len(fake.Calls("create_issue")); n != 1 {
		t.Fatalf("%d calls, want the allowed one", n)
	}
}

// A call that waits a day for its owner expires: the sweep ends its turn
// as confirmation_expired, nothing is sent, nothing is held, its streams
// read it, the notice goes, the system's audit says so; deciding it later
// is refused, and the thread goes on.
func TestAssistantConfirmationExpires(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	fake := mcptest.NewServer(t, "", "")
	fake.Tool("create_issue", "Create an issue", false, mcptest.Text("Created"))
	srv := f.remote(map[string]any{"name": "Linear", "url": fake.Endpoint(), "auth": "none"}, map[string]string{"create_issue": "writes"})
	th := f.thread(f.mc, map[string]any{})
	f.useServers(f.mc, th.ID, srv.ID)
	turn, c := f.waitingCall(th.ID, "Late")
	age := func(d string) {
		if _, err := f.env.App.Pool.Exec(ctx, `UPDATE ai_assistant_confirmations SET created_at = now() - $2::interval WHERE id = $1`, c.ID, d); err != nil {
			t.Fatal(err)
		}
	}
	age("23 hours")
	if _, err := f.worker().Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if got := f.get(f.mc, th.ID); got.Turn.State != "waiting" || got.Confirmation == nil {
		t.Fatalf("a call waiting 23 hours: %+v", got.Turn)
	}
	age("25 hours")
	if _, err := f.worker().Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	got := f.get(f.mc, th.ID)
	if got.Turn.ID != turn || got.Turn.State != "failed" || got.Turn.Reason != "confirmation_expired" || got.Confirmation != nil {
		t.Fatalf("after a day: %+v %+v", got.Turn, got.Confirmation)
	}
	if e := f.lastConfirmation(th.ID); e.ID != c.ID || e.State != "expired" {
		t.Fatalf("the streams read %+v", e)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_confirmations WHERE id = $1 AND state = 'expired' AND decision IS NULL`, c.ID); n != 1 {
		t.Fatal("the call isn't expired")
	}
	if n := f.count(`SELECT count(*) FROM ai_budget_reservations b JOIN ai_assistant_steps s ON s.id = b.id WHERE s.turn_id = $1 AND b.state = 'reserved'`, turn) +
		f.count(`SELECT count(*) FROM ai_assistant_slots WHERE turn_id = $1`, turn); n != 0 {
		t.Fatal("the expired turn holds a reservation or a slot")
	}
	if n := f.count(`SELECT count(*) FROM notifications WHERE kind = 'ai_assistant_confirmation'`); n != 0 {
		t.Fatal("the notice stayed")
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE action = 'ai_assistant_confirmation.expired' AND actor_type = 'system' AND metadata->>'confirmation_id' = $1`, c.ID.String()); n != 1 {
		t.Fatal("the expiry isn't audited")
	}
	if p := f.decide(f.mc, th.ID, c.ID, "allow", 409).Problem(t); p["code"] != "ai_assistant_confirmation_closed" {
		t.Fatalf("allowing an expired call: %v", p)
	}
	if _, err := f.worker().Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE action = 'ai_assistant_confirmation.expired'`); n != 1 {
		t.Fatal("the sweep expired the call twice")
	}
	f.streams.then(says("Still here."))
	next := f.send(f.mc, th.ID, "Anything else?").Turn.ID
	f.execute(f.job(next))
	if got := f.get(f.mc, th.ID); got.Turn.State != "done" {
		t.Fatalf("the next turn: %+v", got.Turn)
	}
	if len(fake.Calls("create_issue")) != 0 {
		t.Fatal("the expired call was sent")
	}
}

// Why a thread's model can't answer has its own reason, for the turn and
// for the next message, saying who can fix it: the account disconnected,
// the tool check no longer passed after a key was replaced, the model
// removed, personal keys turned off; a person's own model says it is
// theirs to fix.
func TestAssistantModelProblems(t *testing.T) {
	cases := []struct {
		name, reason, code, says string
		personal                 bool
		change                   func(f *fixture, account, model uuid.UUID)
	}{
		{name: "disconnected", reason: "account_disconnected", code: "ai_account_disconnected", says: "Someone who can manage integrations can connect it again",
			change: func(f *fixture, _, _ uuid.UUID) {
				f.oc.Do(http.MethodDelete, f.base+"/connection", nil).MustStatus(f.t, 200)
			}},
		{name: "key replaced", reason: "tools_unchecked", code: "ai_tools_unchecked", says: "Someone who can manage integrations can test its tools",
			change: func(f *fixture, account, _ uuid.UUID) {
				f.oc.Do(http.MethodPut, f.base+"/accounts/"+account.String()+"/key", map[string]any{"apiKey": "sk-replaced-key"}).MustStatus(f.t, 200)
			}},
		{name: "model removed", reason: "model_gone", code: "ai_assistant_model_gone", says: "model was removed",
			change: func(f *fixture, account, model uuid.UUID) {
				f.oc.Do(http.MethodDelete, f.base+"/accounts/"+account.String()+"/models/"+model.String(), nil).MustStatus(f.t, 200)
			}},
		{name: "personal keys off", reason: "personal_keys_off", code: "ai_personal_keys_off", says: "Personal keys were turned off", personal: true,
			change: func(f *fixture, _, _ uuid.UUID) {
				f.oc.Do(http.MethodPut, f.base+"/personal-keys", map[string]any{"enabled": false, "capUsd": 10}).MustStatus(f.t, 200)
			}},
		{name: "own key replaced", reason: "tools_unchecked", code: "ai_tools_unchecked", says: "Your own model hasn't passed", personal: true,
			change: func(f *fixture, account, _ uuid.UUID) {
				f.mc.Do(http.MethodPut, f.base+"/personal/accounts/"+account.String()+"/key", map[string]any{"apiKey": "sk-own-replaced"}).MustStatus(f.t, 200)
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			account, model := f.accountID, f.modelID
			if c.personal {
				f.oc.Do(http.MethodPut, f.base+"/personal-keys", map[string]any{"enabled": true, "capUsd": 10}).MustStatus(t, 200)
				var own ai.AIPersonalSettings
				f.mc.Do(http.MethodPost, f.base+"/personal/accounts", map[string]any{"provider": "compatible", "baseUrl": f.model.URL, "model": "own-model",
					"inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200).JSON(t, &own)
				account, model = own.Accounts[0].ID, own.Accounts[0].Models[0].ID
				f.passTools(f.mc, "/personal/accounts/"+account.String()+"/models/"+model.String())
			} else if c.reason == "model_gone" {
				var s ai.AISettings
				f.oc.Do(http.MethodPost, f.base+"/accounts/"+account.String()+"/models", map[string]any{"model": "model-b", "inputPrice": 1, "outputPrice": 2}).
					MustStatus(t, 200).JSON(t, &s)
				model = s.Accounts[0].Models[1].ID
				f.passTools(f.oc, "/accounts/"+account.String()+"/models/"+model.String())
			}
			th := f.thread(f.mc, map[string]any{"modelId": model})
			turn := f.send(f.mc, th.ID, "Hello").Turn.ID
			c.change(f, account, model)
			f.execute(f.job(turn))
			if got := f.get(f.mc, th.ID).Turn; got.State != "failed" || got.Reason != c.reason {
				t.Fatalf("the turn: %s %q, want failed %q", got.State, got.Reason, c.reason)
			}
			p := f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/messages", map[string]any{"text": "Again", "requestKey": "again-1"}).
				MustStatus(t, 409).Problem(t)
			if p["code"] != c.code || !strings.Contains(fmt.Sprint(p["detail"]), c.says) {
				t.Fatalf("the next message: %v, want %s saying %q", p, c.code, c.says)
			}
		})
	}
}
