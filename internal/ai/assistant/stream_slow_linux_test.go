//go:build linux

package assistant_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

// A client that stops reading is cut once an event waits on it for
// SlowClient, so it never holds the stream back; it resumes from the last
// event it read and gets the rest, nothing twice and nothing missing.
func TestAssistantSlowClientIsCut(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	// Before this app's first stream.
	f.env.App.Assistant().SlowClient = 300 * time.Millisecond
	th := f.thread(f.mc, map[string]any{})
	token, hash := authn.NewToken()
	if _, err := f.env.App.Q.CreateSession(ctx, store.CreateSessionParams{ID: uuid.Must(uuid.NewV7()), UserID: f.member, TokenHash: hash, AuthMethod: "test",
		MfaVerified: true, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	// A small receive buffer: once it is full, the server's writes wait.
	dialer := &net.Dialer{Control: func(_, _ string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) { serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4096) }); err != nil {
			return err
		}
		return serr
	}}
	hc := &http.Client{Transport: &http.Transport{DialContext: dialer.DialContext, DisableCompression: true}}
	// The app behind a listener whose connections have a small send buffer
	// too: the kernel otherwise grows it to megabytes (more on some CI
	// runners), and the whole answer would fit without a write waiting.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: f.env.Server.Config.Handler, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(smallSendBuffer{ln}) }()
	t.Cleanup(func() { _ = srv.Close() })
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+ln.Addr().String()+"/api/v1"+f.base+"/assistant/threads/"+th.ID.String()+"/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: f.env.App.Cfg.Auth.CookieName, Value: token})
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	r := bufio.NewReader(resp.Body)
	if ev, err := readEvent(r); err != nil || ev.Event != "snapshot" {
		t.Fatalf("first %s, %v", ev.Event, err)
	}
	// A long answer, written while the client doesn't read.
	pieces := make([]string, 500)
	for i := range pieces {
		pieces[i] = strings.Repeat("x", 2047) + "\n"
	}
	f.streams.set(answer(pieces...))
	turn := f.send(f.mc, th.ID, "A long answer, please").Turn.ID
	if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)
	got := []sseEvent{}
	cut := false
	for {
		ev, err := readEvent(r)
		if err != nil {
			cut = true
			break
		}
		got = append(got, ev)
		if ev.Event == "turn" && bytes.Contains(ev.Data, []byte(`"state":"done"`)) {
			break
		}
	}
	if !cut || len(got) == 0 {
		t.Fatalf("the slow client read %d events and wasn't cut", len(got))
	}
	s := f.openStream(f.mc, th.ID, got[len(got)-1].ID)
	all := append(got, s.until(t, "done")...)
	consecutive(t, ids(t, all), 1)
	var text strings.Builder
	for _, ev := range all {
		if ev.Event != "delta" {
			continue
		}
		var d assistant.AIThreadDeltaEvent
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			t.Fatal(err)
		}
		text.WriteString(d.Text)
	}
	if text.String() != strings.Join(pieces, "") {
		t.Fatalf("the deltas make %d bytes, want %d", text.Len(), len(strings.Join(pieces, "")))
	}
}

// smallSendBuffer accepts connections with a 4 KB send buffer.
type smallSendBuffer struct{ net.Listener }

func (l smallSendBuffer) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		if tc, ok := c.(*net.TCPConn); ok {
			_ = tc.SetWriteBuffer(4096)
		}
	}
	return c, err
}
