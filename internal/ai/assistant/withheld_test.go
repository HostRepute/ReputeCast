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

// An answer carries what its request carried, and a request carries the
// titles of what is attached: an answer that only repeats a title, without
// reading the resource, is withheld like a quote once its owner can no
// longer read the resource, and so is an answer stopped mid-stream. An
// answer from before the resource was attached stays.
func TestAssistantWithholdsAnswersThatRepeatATitle(t *testing.T) {
	for _, how := range []string{"permission", "deleted", "stopped"} {
		t.Run(how, func(t *testing.T) {
			d := newDemoFixture(t)
			const title = "Zephyr pricing leak, do not share"
			if _, err := d.env.App.Pool.Exec(context.Background(), `UPDATE campaigns SET name = $2 WHERE id = $1`, d.draft, title); err != nil {
				t.Fatal(err)
			}
			th := d.thread(d.mc, map[string]any{})
			const before = "Before anything was attached: hello."
			d.streams.then(says(before))
			d.run(th.ID, "Hi")
			d.attach(d.mc, th.ID, "campaign", d.draft)
			echo := "You attached " + title + "."
			var turn uuid.UUID
			if how == "stopped" {
				turn = d.send(d.mc, th.ID, "What did I attach?").Turn.ID
				d.streams.set(drip(echo+" ", 50*time.Millisecond, 200))
				done := make(chan error, 1)
				args := d.job(turn)
				go func() { done <- d.worker().ExecuteTurn(context.Background(), args) }()
				waitFor(t, "a piece of the answer", func() bool {
					return d.count(`SELECT count(*) FROM ai_assistant_events WHERE turn_id = $1 AND kind = 'delta'`, turn) > 0
				})
				d.mc.Do(http.MethodPost, demoAI+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 200)
				if err := <-done; err != nil {
					t.Fatal(err)
				}
				d.streams.set(nil)
			} else {
				d.streams.then(says(echo))
				turn = d.run(th.ID, "What did I attach?")
			}
			if body := mustJSON(t, d.streams.streamed(t)[1]); !strings.Contains(body, title) {
				t.Fatal("the request didn't carry the title")
			}
			if got := d.get(d.mc, th.ID); !strings.Contains(mustJSON(t, got), "You attached "+title) {
				t.Fatalf("the answer didn't repeat the title: %s", mustJSON(t, got))
			}

			lose := how
			if how == "stopped" {
				lose = "permission"
			}
			d.lose(lose)
			d.streams.then(says("Noted."))
			d.run(th.ID, "And now?")
			bodies := d.streams.streamed(t)
			if body := mustJSON(t, bodies[len(bodies)-1]); strings.Contains(body, title) || !strings.Contains(body, before) {
				t.Fatalf("the next request: %s", body)
			}
			got := d.get(d.mc, th.ID)
			var withheld, kept int
			for _, m := range got.Messages {
				switch {
				case m.TurnID != nil && *m.TurnID == turn && m.Role == "assistant":
					if !m.Withheld || m.Text != "" {
						t.Fatalf("the answer that repeated the title: %+v", m)
					}
					withheld++
				case m.Text == before:
					if m.Withheld {
						t.Fatalf("the earlier answer: %+v", m)
					}
					kept++
				}
			}
			if withheld != 1 || kept != 1 || strings.Contains(mustJSON(t, got), title) {
				t.Fatalf("the thread: %s", mustJSON(t, got))
			}
			s := d.openStream(d.mc, th.ID, "")
			if ev := s.next(t, 10*time.Second); ev.Event != "snapshot" || strings.Contains(string(ev.Data), title) {
				t.Fatalf("the snapshot: %s %s", ev.Event, ev.Data)
			}
			s.close()
			s = d.openStream(d.mc, th.ID, "0")
			for last := strconv.FormatInt(got.EventSeq, 10); ; {
				ev := s.next(t, 10*time.Second)
				if strings.Contains(string(ev.Data), title) {
					t.Fatalf("event %s (%s) sent the title again: %s", ev.ID, ev.Event, ev.Data)
				}
				if ev.ID == last {
					break
				}
			}
		})
	}
}

// A stream stops sending an answer's pieces soon after its owner loses
// something the answer may have used (a resource the thread read, or one
// attached whose title the request carried), while they keep ai:use: the
// pieces that follow arrive empty, which the panel shows as withheld, and
// the answer's message is withheld. A thread that used nothing keeps
// streaming through the same role change.
func TestAssistantLiveAnswerStopsWhenASourceIsLost(t *testing.T) {
	for _, used := range []string{"read", "title", "nothing"} {
		t.Run(used, func(t *testing.T) {
			d := newDemoFixture(t)
			svc := d.env.App.Assistant()
			old := svc.AccessEvery
			svc.AccessEvery = 100 * time.Millisecond
			t.Cleanup(func() { svc.AccessEvery = old })
			th := d.thread(d.mc, map[string]any{})
			switch used {
			case "read":
				d.readOnce(th)
			case "title":
				d.attach(d.mc, th.ID, "campaign", d.draft)
			}
			cursor := strconv.FormatInt(d.get(d.mc, th.ID).EventSeq, 10)
			s := d.openStream(d.mc, th.ID, cursor)
			// read is the stream's next event. The server may end a stream
			// before the turn ends, and its reader then resumes from the last
			// event it read, as the panel does (on CI one ended right after
			// the owner's role changed, even for the thread that used
			// nothing). This test checks what the reader is sent, so it
			// resumes too; a problem event fails it.
			last, resumed := cursor, 0
			read := func() sseEvent {
				t.Helper()
				for {
					ev, ok := s.nextOrEnd(t, 10*time.Second)
					if !ok {
						if resumed++; resumed > 3 {
							t.Fatalf("the stream ended %d times", resumed)
						}
						t.Logf("the stream ended after event %s; resuming from it", last)
						s = d.openStream(d.mc, th.ID, last)
						continue
					}
					if ev.Event == "problem" {
						t.Fatalf("the stream sent a problem: %s", ev.Data)
					}
					if ev.ID != "" {
						last = ev.ID
					}
					return ev
				}
			}
			const first, second = "The first piece. ", "The second piece, written after access changed."
			release := make(chan struct{})
			d.streams.set(held(release, first, second))
			turn := d.send(d.mc, th.ID, "Tell me more").Turn.ID
			done := make(chan error, 1)
			args := d.job(turn)
			go func() { done <- d.worker().ExecuteTurn(context.Background(), args) }()
			released := false
			defer func() {
				if !released {
					close(release)
				}
				<-done
			}()
			for {
				ev := read()
				if ev.Event == "delta" && strings.Contains(string(ev.Data), first) {
					break
				}
			}
			d.lose("permission")
			time.Sleep(5 * svc.AccessEvery)
			close(release)
			released = true
			var empty, sawSecond bool
			var message *assistant.AIThreadMessageEvent
			for message == nil {
				ev := read()
				switch ev.Event {
				case "delta":
					var e assistant.AIThreadDeltaEvent
					if err := json.Unmarshal(ev.Data, &e); err != nil {
						t.Fatal(err)
					}
					empty = empty || e.Text == ""
					sawSecond = sawSecond || strings.Contains(e.Text, "second piece")
				case "message":
					var e assistant.AIThreadMessageEvent
					if err := json.Unmarshal(ev.Data, &e); err != nil {
						t.Fatal(err)
					}
					if e.Message.Role == "assistant" {
						message = &e
					}
				}
			}
			if used == "nothing" {
				if !sawSecond || empty || message.Message.Withheld {
					t.Fatalf("the answer that used nothing: second piece %v, an empty piece %v, its message %+v", sawSecond, empty, message.Message)
				}
				return
			}
			if sawSecond || !empty || !message.Message.Withheld || message.Message.Text != "" {
				t.Fatalf("after access changed: second piece %v, an empty piece %v, the message %+v", sawSecond, empty, message.Message)
			}
		})
	}
}

// An answer that repeats an earlier answer carries what that answer
// carried: after the resource whose title they repeat is removed from the
// thread (it is no longer attached, so the request doesn't list it) and its
// owner then loses it, both answers are withheld.
func TestAssistantWithholdsAnswersThatRepeatAnEarlierAnswer(t *testing.T) {
	d := newDemoFixture(t)
	const title = "Zephyr pricing leak, do not share"
	if _, err := d.env.App.Pool.Exec(context.Background(), `UPDATE campaigns SET name = $2 WHERE id = $1`, d.draft, title); err != nil {
		t.Fatal(err)
	}
	th := d.thread(d.mc, map[string]any{})
	handle := d.attach(d.mc, th.ID, "campaign", d.draft)
	d.streams.then(says("You attached " + title + "."))
	first := d.run(th.ID, "What did I attach?")
	d.mc.Do(http.MethodDelete, demoAI+"/assistant/threads/"+th.ID.String()+"/contexts/"+handle.String(), nil).MustStatus(t, 200)
	d.streams.then(says("Earlier you attached " + title + "."))
	second := d.run(th.ID, "Remind me what it was?")
	if body := mustJSON(t, d.streams.streamed(t)[1]); strings.Contains(body, "<attached>") || !strings.Contains(body, "You attached "+title) {
		t.Fatalf("the second request: %s", body)
	}

	d.lose("permission")
	got := d.get(d.mc, th.ID)
	for _, m := range got.Messages {
		if m.Role == "assistant" && m.TurnID != nil && (*m.TurnID == first || *m.TurnID == second) && (!m.Withheld || m.Text != "") {
			t.Fatalf("an answer that repeated the title: %+v", m)
		}
	}
	if strings.Contains(mustJSON(t, got), title) {
		t.Fatalf("the thread: %s", mustJSON(t, got))
	}
	d.streams.then(says("Noted."))
	d.run(th.ID, "And now?")
	bodies := d.streams.streamed(t)
	if body := mustJSON(t, bodies[len(bodies)-1]); strings.Contains(body, title) {
		t.Fatalf("the next request: %s", body)
	}
}
