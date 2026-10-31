package engine_test

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testStreamHeartbeats: the live stream sends a snapshot at once, then a
// heartbeat event while nothing changes, so the monitor can tell a quiet
// send from a connection a proxy is holding (review GS-14: an
// EventSource never sees comments).
func testStreamHeartbeats(t *testing.T, f *sendtest.Fixture) {
	f.AddContacts(sendtest.ContactSpec{N: 10, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{})
	old := f.E.StreamHeartbeat
	f.E.StreamHeartbeat = 100 * time.Millisecond
	t.Cleanup(func() { f.E.StreamHeartbeat = old })
	c := f.Env.Client()
	c.SignInDemo("aisha@acme.io")
	ctx, cancel := context.WithTimeout(f.Ctx, 30*time.Second)
	defer cancel()
	resp, err := c.Stream(ctx, "/workspaces/acme-newsletter/campaigns/"+f.Campaign.String()+"/live/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status %d, %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var events []string
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		ev, ok := strings.CutPrefix(sc.Text(), "event: ")
		if !ok {
			continue
		}
		events = append(events, ev)
		if len(events) == 1 && ev != "snapshot" {
			t.Fatalf("first event %q, want the snapshot", ev)
		}
		if ev == "heartbeat" {
			return
		}
	}
	t.Fatalf("events %v before the stream ended (%v), want a snapshot then a heartbeat", events, sc.Err())
}
