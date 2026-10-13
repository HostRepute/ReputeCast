package apptest

import (
	"bufio"
	"net"
	"strings"
	"sync"
	"testing"
)

// SMTPSink is a minimal SMTP server for tests: it accepts EHLO, AUTH PLAIN
// or LOGIN (any credentials unless Reject is set), MAIL, RCPT and DATA, and
// keeps every message it received.
type SMTPSink struct {
	Addr string
	// Reject makes AUTH fail with 535.
	Reject bool

	mu       sync.Mutex
	messages []SinkMessage
	ln       net.Listener
}

// SinkMessage is one message the sink received.
type SinkMessage struct {
	From string
	To   []string
	Data string
}

// NewSMTPSink starts a sink on a free local port; it stops with the test.
func NewSMTPSink(t testing.TB) *SMTPSink {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &SMTPSink{Addr: ln.Addr().String(), ln: ln}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

// Messages returns what the sink received so far.
func (s *SMTPSink) Messages() []SinkMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]SinkMessage(nil), s.messages...)
}

func (s *SMTPSink) serve() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(c)
	}
}

func (s *SMTPSink) handle(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	w := func(line string) { _, _ = c.Write([]byte(line + "\r\n")) }
	w("220 sink.test ESMTP Sink")
	var msg SinkMessage
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			w("250-sink.test")
			w("250-AUTH PLAIN LOGIN")
			w("250 SIZE 52428800")
		case strings.HasPrefix(cmd, "AUTH"):
			if s.Reject {
				w("535 5.7.8 Authentication credentials invalid")
				continue
			}
			if strings.HasPrefix(cmd, "AUTH LOGIN") {
				w("334 VXNlcm5hbWU6")
				_, _ = r.ReadString('\n')
				w("334 UGFzc3dvcmQ6")
				_, _ = r.ReadString('\n')
			}
			w("235 2.7.0 Authentication successful")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			msg = SinkMessage{From: strings.Trim(strings.TrimSpace(line)[10:], "<> ")}
			w("250 2.1.0 Ok")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			msg.To = append(msg.To, strings.Trim(strings.TrimSpace(line)[8:], "<> "))
			w("250 2.1.5 Ok")
		case cmd == "DATA":
			w("354 End data with <CR><LF>.<CR><LF>")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" || l == ".\n" {
					break
				}
				b.WriteString(l)
			}
			msg.Data = b.String()
			s.mu.Lock()
			s.messages = append(s.messages, msg)
			s.mu.Unlock()
			w("250 2.0.0 Ok: queued")
		case cmd == "RSET", cmd == "NOOP":
			w("250 2.0.0 Ok")
		case cmd == "QUIT":
			w("221 2.0.0 Bye")
			return
		default:
			w("502 5.5.2 Command not recognized")
		}
	}
}
