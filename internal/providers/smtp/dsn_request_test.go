package smtp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/smtp"
	"github.com/hostrepute/reputecast/internal/providers/smtp/dsn"
)

// fakeServer is a minimal SMTP server on loopback that records the
// envelope commands it gets; dsn says whether it announces DSN (RFC 3461).
type fakeServer struct {
	addr string
	mu   sync.Mutex
	cmds []string
}

func startFake(t *testing.T, announceDSN bool) *fakeServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	s := &fakeServer{addr: ln.Addr().String()}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(c, announceDSN)
		}
	}()
	return s
}

func (s *fakeServer) serve(c net.Conn, announceDSN bool) {
	defer func() { _ = c.Close() }()
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	r := bufio.NewReader(c)
	say := func(l string) { _, _ = c.Write([]byte(l + "\r\n")) }
	say("220 fake.test ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		up := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(up, "EHLO"):
			if announceDSN {
				say("250-fake.test")
				say("250-8BITMIME")
				say("250 DSN")
			} else {
				say("250-fake.test")
				say("250 8BITMIME")
			}
		case strings.HasPrefix(up, "HELO"):
			say("250 fake.test")
		case strings.HasPrefix(up, "MAIL FROM"), strings.HasPrefix(up, "RCPT TO"):
			s.mu.Lock()
			s.cmds = append(s.cmds, line)
			s.mu.Unlock()
			say("250 OK")
		case up == "DATA":
			say("354 go ahead")
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
			}
			say("250 2.0.0 Ok: queued")
		case up == "QUIT":
			say("221 bye")
			return
		default: // RSET, NOOP
			say("250 OK")
		}
	}
}

func (s *fakeServer) envelope() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.cmds...)
}

func sendThrough(t *testing.T, s *fakeServer) {
	t.Helper()
	host, port, _ := net.SplitHostPort(s.addr)
	settings, _ := json.Marshal(map[string]any{"host": host, "port": atoi(port), "tls": "none", "return_path": "bounces@news.acme.io"})
	p, err := smtp.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{}`), WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Capabilities().InfersDelivery {
		t.Fatal("SMTP doesn't infer delivery from the server's acceptance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := p.Send(ctx, []providers.Message{contracttest.Message(0)})
	if err != nil || len(res) != 1 || res[0].Err != nil {
		t.Fatalf("send: %v %+v", err, res)
	}
}

// Delivery status notifications are asked for where the server announces
// DSN (review GS-7): success, failure and delay, headers only.
func TestRequestsDSNWhereSupported(t *testing.T) {
	s := startFake(t, true)
	sendThrough(t, s)
	env := strings.Join(s.envelope(), "\n")
	if !strings.Contains(env, "RET=HDRS") || !strings.Contains(env, "NOTIFY=SUCCESS,FAILURE,DELAY") {
		t.Fatalf("envelope without the DSN request:\n%s", env)
	}
	plain := startFake(t, false)
	sendThrough(t, plain)
	if env := strings.Join(plain.envelope(), "\n"); strings.Contains(env, "RET=") || strings.Contains(env, "NOTIFY=") {
		t.Fatalf("DSN parameters sent to a server that doesn't announce DSN:\n%s", env)
	}
}

// A success notification confirms a delivery already counted at
// acceptance: it maps to a delivered event the ingest counts once per
// message.
func TestSuccessDSN(t *testing.T) {
	body, err := os.ReadFile("dsn/testdata/postfix-success.eml")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := dsn.Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	evs := smtp.Events(rep)
	if len(evs) != 1 || evs[0].Type != providers.EventDelivered || evs[0].Recipient != "a.k@example.org" {
		t.Fatalf("events %+v", evs)
	}
}
