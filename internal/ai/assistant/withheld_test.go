package assistant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/assistant"
)

// lose takes the member's reading of the demo draft away: their role keeps
// ai:use only, or the draft is deleted.
func (d *demoFixture) lose(how string) {
	d.t.Helper()
	ctx := context.Background()
	switch how {
	case "permission":
		var role uuid.UUID
		if err := d.env.App.Pool.QueryRow(ctx, `INSERT INTO roles (id, workspace_id, key, name, builtin, permissions) VALUES ($1, $2, 'chat-only', 'Chat only', false, ARRAY['ai:use'])
			RETURNING id`, uuid.Must(uuid.NewV7()), d.ws.ID).Scan(&role); err != nil {
			d.t.Fatal(err)
		}
		if _, err := d.env.App.Pool.Exec(ctx, `UPDATE workspace_members SET role_id = $1 WHERE workspace_id = $2 AND user_id = $3`, role, d.ws.ID, d.member); err != nil {
			d.t.Fatal(err)
		}
	case "deleted":
		if _, err := d.env.App.Pool.Exec(ctx, `UPDATE campaigns SET deleted_at = now() WHERE id = $1`, d.draft); err != nil {
			d.t.Fatal(err)
		}
	}
}

// run sends a message and runs its turn.
func (d *demoFixture) run(thread uuid.UUID, text string) uuid.UUID {
	d.t.Helper()
	turn := d.send(d.mc, thread, text).Turn.ID
	if err := d.worker().ExecuteTurn(context.Background(), d.job(turn)); err != nil {
		d.t.Fatal(err)
	}
	return turn
}

// An answer that quoted a campaign is withheld once its owner can no
// longer read the campaign (their role changed, or it was deleted): the
// model reads a placeholder instead, and get-ai-thread, a snapshot and the
// events sent again on resume show it without its text. An answer from
// before the campaign was read stays.
func TestAssistantWithholdsAnswersFromLostSources(t *testing.T) {
	for _, how := range []string{"permission", "deleted"} {
		t.Run(how, func(t *testing.T) {
			d := newDemoFixture(t)
			th := d.thread(d.mc, map[string]any{})
			const before = "Before reading anything: hello."
			d.streams.then(says(before))
			d.run(th.ID, "Hi")
			campaign := d.attach(d.mc, th.ID, "campaign", d.draft)
			quote := "The subject reads: " + d.draftSubject
			d.streams.then(calls("read_attached", `{"handle":"`+campaign.String()+`"}`), says(quote))
			quoted := d.run(th.ID, "What does the draft say?")
			if got := d.get(d.mc, th.ID); !strings.Contains(mustJSON(t, got), d.draftSubject) {
				t.Fatal("the answer didn't quote the draft")
			}

			d.lose(how)
			d.streams.then(says("Noted."))
			d.run(th.ID, "And now?")
			bodies := d.streams.streamed(t)
			if body := mustJSON(t, bodies[len(bodies)-1]); strings.Contains(body, d.draftSubject) || !strings.Contains(body, before) ||
				!strings.Contains(body, "An earlier answer here used something the person can no longer read") {
				t.Fatalf("the next request: %s", body)
			}

			got := d.get(d.mc, th.ID)
			var withheld, kept, row int
			for _, m := range got.Messages {
				switch {
				case m.TurnID != nil && *m.TurnID == quoted && m.Role == "assistant":
					if !m.Withheld || m.Text != "" {
						t.Fatalf("the quoting answer: %+v", m)
					}
					withheld++
				case m.Text == before:
					if m.Withheld {
						t.Fatalf("the earlier answer: %+v", m)
					}
					kept++
				case m.Role == "tool":
					for _, r := range m.Tools {
						if r.Name == "read_attached" && (r.OK || r.Code != "no_longer_available" || r.Label != "") {
							t.Fatalf("the read's row: %+v", r)
						}
						row++
					}
				}
			}
			if withheld != 1 || kept != 1 || row != 1 || strings.Contains(mustJSON(t, got), d.draftSubject) {
				t.Fatalf("the thread: %s", mustJSON(t, got))
			}

			// A snapshot, and every event sent again from the first.
			s := d.openStream(d.mc, th.ID, "")
			if ev := s.next(t, 10*time.Second); ev.Event != "snapshot" || strings.Contains(string(ev.Data), d.draftSubject) || !strings.Contains(string(ev.Data), `"withheld":true`) {
				t.Fatalf("the snapshot: %s %s", ev.Event, ev.Data)
			}
			s.close()
			s = d.openStream(d.mc, th.ID, "0")
			last, sawWithheld, sawRow := strconv.FormatInt(got.EventSeq, 10), false, false
			for {
				ev := s.next(t, 10*time.Second)
				if strings.Contains(string(ev.Data), d.draftSubject) {
					t.Fatalf("event %s (%s) sent the quote again: %s", ev.ID, ev.Event, ev.Data)
				}
				if ev.Event == "message" && strings.Contains(string(ev.Data), `"withheld":true`) {
					sawWithheld = true
				}
				if ev.Event == "tool" && strings.Contains(string(ev.Data), `"read_attached"`) {
					var e assistant.AIThreadToolEvent
					if err := json.Unmarshal(ev.Data, &e); err != nil || e.Tool.OK || e.Tool.Code != "no_longer_available" || e.Tool.Label != "" {
						t.Fatalf("the read's row sent again: %s", ev.Data)
					}
					sawRow = true
				}
				if ev.ID == last {
					break
				}
			}
			if !sawWithheld || !sawRow {
				t.Fatalf("the quoting answer's message event withheld %v, the read's row %v", sawWithheld, sawRow)
			}
		})
	}
}

// An answer stopped mid-stream, or left by a worker that died and saved
// by the sweep, is saved without its request: it counts everything the
// thread read before, so it is withheld once that can't be read.
func TestAssistantStoppedAnswerKeepsSources(t *testing.T) {
	for _, how := range []string{"stopped", "swept"} {
		t.Run(how, func(t *testing.T) { stoppedAnswerKeepsSources(t, how) })
	}
}

func stoppedAnswerKeepsSources(t *testing.T, how string) {
	d := newDemoFixture(t)
	th := d.thread(d.mc, map[string]any{})
	d.readOnce(th)
	turn := d.send(d.mc, th.ID, "Tell me more").Turn.ID
	args := d.job(turn)
	pieces := func() bool {
		return d.count(`SELECT count(*) FROM ai_assistant_events WHERE turn_id = $1 AND kind = 'delta'`, turn) > 0
	}
	if how == "stopped" {
		d.streams.set(drip(d.draftSubject+" ", 50*time.Millisecond, 200))
		done := make(chan error, 1)
		go func() { done <- d.worker().ExecuteTurn(context.Background(), args) }()
		waitFor(t, "a piece of the answer", pieces)
		d.mc.Do(http.MethodPost, demoAI+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 200)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	} else {
		release := make(chan struct{})
		defer close(release)
		d.streams.set(held(release, d.draftSubject+" "))
		pool := d.otherPool()
		ctx, kill := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- d.workerOn(pool).ExecuteTurn(ctx, args) }()
		waitFor(t, "a piece of the answer", pieces)
		pool.Close()
		kill()
		<-done
		sweeper := d.worker()
		sweeper.Now = func() time.Time { return time.Now().Add(2 * time.Minute) }
		if n, err := sweeper.Sweep(context.Background()); err != nil || n != 1 {
			t.Fatalf("the sweep: %d, %v", n, err)
		}
	}
	var stopped *assistant.AIThreadMessage
	got := d.get(d.mc, th.ID)
	for i, m := range got.Messages {
		if m.TurnID != nil && *m.TurnID == turn && m.Role == "assistant" {
			stopped = &got.Messages[i]
		}
	}
	if stopped == nil || !stopped.Partial || stopped.Withheld {
		t.Fatalf("the stopped answer before: %+v", stopped)
	}
	d.lose("deleted")
	got = d.get(d.mc, th.ID)
	for _, m := range got.Messages {
		if m.ID == stopped.ID && (!m.Withheld || m.Text != "") {
			t.Fatalf("the stopped answer after: %+v", m)
		}
	}
	if strings.Contains(mustJSON(t, got), d.draftSubject) {
		t.Fatalf("the thread: %s", mustJSON(t, got))
	}
}

// An answer in flight may use anything the thread read: once its owner
// can't read that, get-ai-thread leaves its text out (withheld), and its
// pieces sent again on resume are empty.
func TestAssistantAnswerInFlightWithheld(t *testing.T) {
	d := newDemoFixture(t)
	th := d.thread(d.mc, map[string]any{})
	d.readOnce(th)
	turn := d.send(d.mc, th.ID, "Tell me more").Turn.ID
	release := make(chan struct{})
	d.streams.set(held(release, d.draftSubject+" "))
	done := make(chan error, 1)
	args := d.job(turn)
	go func() { done <- d.worker().ExecuteTurn(context.Background(), args) }()
	defer func() { close(release); <-done }()
	waitFor(t, "a piece of the answer", func() bool {
		return d.count(`SELECT count(*) FROM ai_assistant_events WHERE turn_id = $1 AND kind = 'delta'`, turn) > 0
	})
	if got := d.get(d.mc, th.ID); got.Turn.Withheld || !strings.Contains(got.Turn.Text, d.draftSubject) {
		t.Fatalf("the answer in flight before: %+v", got.Turn)
	}
	d.lose("permission")
	got := d.get(d.mc, th.ID)
	if !got.Turn.Withheld || got.Turn.Text != "" || strings.Contains(mustJSON(t, got), d.draftSubject) {
		t.Fatalf("the thread after: %s", mustJSON(t, got))
	}
	s := d.openStream(d.mc, th.ID, "0")
	for last := strconv.FormatInt(got.EventSeq, 10); ; {
		ev := s.next(t, 10*time.Second)
		if strings.Contains(string(ev.Data), d.draftSubject) {
			t.Fatalf("event %s (%s) sent the answer in flight again: %s", ev.ID, ev.Event, ev.Data)
		}
		if ev.ID == last {
			break
		}
	}
}
