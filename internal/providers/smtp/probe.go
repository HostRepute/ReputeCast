package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/smtp/bouncebox"
	"github.com/hostrepute/reputecast/internal/providers/smtp/dsn"
)

// This file is the SMTP side of "Connect a sending provider": a connection
// test that reports the greeting, the extensions the server offers, the TLS
// version and whether it accepted the credentials, and a bounce-mailbox test
// that counts waiting reports without marking or deleting anything.

var (
	_ providers.ConnectionTester    = (*Provider)(nil)
	_ providers.BounceMailboxTester = (*Provider)(nil)
)

// recorder keeps what the server says before TLS starts, for the transcript.
type recorder struct {
	net.Conn
	mu      sync.Mutex
	on      bool
	read    bytes.Buffer
	written bytes.Buffer
}

func (r *recorder) Read(b []byte) (int, error) {
	n, err := r.Conn.Read(b)
	r.mu.Lock()
	if r.on && n > 0 {
		r.read.Write(b[:n])
	}
	r.mu.Unlock()
	return n, err
}

func (r *recorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	if r.on {
		r.written.Write(b)
	}
	r.mu.Unlock()
	return r.Conn.Write(b)
}

func (r *recorder) stop() {
	r.mu.Lock()
	r.on = false
	r.mu.Unlock()
}

func lines(b []byte) []string {
	out := []string{}
	for _, l := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// TestConnection implements providers.ConnectionTester: connect, EHLO,
// TLS (implicit or STARTTLS), AUTH, QUIT. Nothing is sent.
func (p *Provider) TestConnection(ctx context.Context) (providers.ConnectionReport, error) {
	out := providers.ConnectionReport{Transcript: []string{}}
	if strings.TrimSpace(p.s.Host) == "" {
		return out, &providers.CallError{Op: "SMTP connect", Message: "Enter the SMTP host."}
	}
	addr := net.JoinHostPort(p.s.Host, strconv.Itoa(p.s.Port))
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	var raw net.Conn
	var err error
	if p.dial != nil {
		raw, err = p.dial(ctx, "tcp", addr)
	} else {
		raw, err = (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return out, &providers.CallError{Op: "SMTP connect", Message: dialMessage(err), Temporary: true, Cause: err}
	}
	defer func() { _ = raw.Close() }()
	_ = raw.SetDeadline(deadline)
	tlsCfg := &tls.Config{ServerName: p.s.Host, MinVersion: tls.VersionTLS12}
	conn := raw
	var tlsState *tls.ConnectionState
	if p.s.TLS == "tls" {
		tc := tls.Client(raw, tlsCfg)
		if err := tc.HandshakeContext(ctx); err != nil {
			return out, &providers.CallError{Op: "TLS handshake", Message: err.Error(), Temporary: true, Cause: err}
		}
		st := tc.ConnectionState()
		tlsState = &st
		conn = tc
	}
	rec := &recorder{Conn: conn, on: true}
	c, err := smtp.NewClient(rec, p.s.Host)
	if err != nil {
		return out, &providers.CallError{Op: "SMTP greeting", Message: err.Error(), Temporary: true, Cause: err}
	}
	defer func() { _ = c.Close() }()
	helo := p.s.HELO
	if helo == "" {
		helo = "localhost"
	}
	greeting := lines(rec.read.Bytes())
	if len(greeting) > 0 {
		out.Transcript = append(out.Transcript, greeting[0])
	}
	rec.read.Reset()
	if err := c.Hello(helo); err != nil {
		return out, &providers.CallError{Op: "SMTP EHLO", Message: err.Error(), Temporary: true, Cause: err}
	}
	// Ask for the extensions: Hello is lazy, the first Extension call sends EHLO.
	starttls, _ := c.Extension("STARTTLS")
	out.Transcript = append(out.Transcript, "EHLO "+helo)
	reply := lines(rec.read.Bytes())
	if len(reply) > 1 {
		reply = reply[1:] // the first line repeats the server name
	}
	out.Transcript = append(out.Transcript, reply...)
	rec.stop()

	tlsWord := "No TLS"
	switch p.s.TLS {
	case "tls":
		tlsWord = "Implicit TLS, " + tlsVersion(tlsState)
	case "none":
	default:
		if !starttls {
			return out, &providers.CallError{Op: "STARTTLS", Message: fmt.Sprintf("%s doesn't offer STARTTLS on port %d. Use implicit TLS on 465, or turn encryption off only on a private network.", p.s.Host, p.s.Port)}
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return out, &providers.CallError{Op: "STARTTLS", Message: err.Error(), Temporary: true, Cause: err}
		}
		st, _ := c.TLSConnectionState()
		tlsState = &st
		tlsWord = "STARTTLS, " + tlsVersion(tlsState)
	}
	authWord := "no authentication"
	if p.s.Username != "" && p.s.Auth != "none" {
		_, mechs := c.Extension("AUTH")
		mech := pickMechanism(p.s.Auth, mechs)
		if mech == "" {
			return out, &providers.CallError{Op: "SMTP AUTH", Message: fmt.Sprintf("%s doesn't offer a login method ReputeCast supports (offered: %s).", p.s.Host, strings.TrimSpace(mechs))}
		}
		var a smtp.Auth = plainAuth{user: p.s.Username, pass: p.c.Password}
		if mech == "LOGIN" {
			a = loginAuth{user: p.s.Username, pass: p.c.Password}
		}
		if err := c.Auth(a); err != nil {
			return out, &providers.CallError{Op: "SMTP AUTH", Status: 535, Message: err.Error(), Auth: true, Cause: err}
		}
		authWord = "AUTH " + mech + " accepted"
		out.Auth = strings.ToLower(mech)
	}
	_ = c.Quit()
	out.Summary = tlsWord + " · " + authWord
	return out, nil
}

func tlsVersion(st *tls.ConnectionState) string {
	if st == nil {
		return "TLS"
	}
	switch st.Version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	}
	return tls.VersionName(st.Version)
}

func pickMechanism(want, offered string) string {
	have := map[string]bool{}
	for _, m := range strings.Fields(strings.ToUpper(offered)) {
		have[m] = true
	}
	switch want {
	case "plain":
		if have["PLAIN"] {
			return "PLAIN"
		}
		return ""
	case "login":
		if have["LOGIN"] {
			return "LOGIN"
		}
		return ""
	}
	if have["PLAIN"] {
		return "PLAIN"
	}
	if have["LOGIN"] {
		return "LOGIN"
	}
	return ""
}

// dialMessage turns a dial error into the words people search for.
func dialMessage(err error) string {
	low := strings.ToLower(err.Error())
	switch {
	case strings.Contains(low, "connection refused"):
		return "ECONNREFUSED · nothing listens on that port"
	case strings.Contains(low, "no such host"):
		return "ENOTFOUND · the host name doesn't resolve"
	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline"):
		return "ETIMEDOUT · no answer within 15 s (a firewall may block the port)"
	}
	return err.Error()
}

// plainAuth is AUTH PLAIN without net/smtp's TLS requirement: the person
// chose the encryption setting, and STARTTLS is the default.
type plainAuth struct{ user, pass string }

func (a plainAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.user + "\x00" + a.pass), nil
}

func (a plainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}

// loginAuth is AUTH LOGIN.
type loginAuth struct{ user, pass string }

func (a loginAuth) Start(*smtp.ServerInfo) (string, []byte, error) { return "LOGIN", nil, nil }

func (a loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(fromServer))) {
	case "username:", "user name:":
		return []byte(a.user), nil
	case "password:":
		return []byte(a.pass), nil
	}
	return nil, fmt.Errorf("unexpected server challenge %q", fromServer)
}

// TestBounceMailbox implements providers.BounceMailboxTester: log in, count
// unprocessed messages and the ones that parse as bounce or complaint
// reports. Messages are read with PEEK and never acknowledged.
func (p *Provider) TestBounceMailbox(ctx context.Context) (providers.MailboxReport, error) {
	mb := p.s.BounceMailbox
	if mb == nil || mb.Host == "" {
		return providers.MailboxReport{}, errors.New("no bounce mailbox is set up yet: enter its server and login")
	}
	cfg := *mb
	if cfg.Timeout == 0 {
		cfg.Timeout = 20 * time.Second
	}
	rep := providers.MailboxReport{Protocol: cfg.Label(), Folder: cfg.Folder}
	if rep.Folder == "" {
		rep.Folder = "INBOX"
	}
	box, err := bouncebox.Open(ctx, cfg)
	if err != nil {
		if errors.Is(err, bouncebox.ErrAuth) {
			return rep, &providers.CallError{Op: cfg.Label() + " LOGIN", Message: "The mailbox rejected the user name or password.", Auth: true, Cause: err}
		}
		return rep, &providers.CallError{Op: cfg.Label() + " connect", Message: dialMessage(err), Temporary: true, Cause: err}
	}
	defer func() { _ = box.Close() }()
	msgs, err := box.Fetch(ctx, 200)
	if err != nil {
		return rep, &providers.CallError{Op: cfg.Label() + " fetch", Message: err.Error(), Temporary: true, Cause: err}
	}
	rep.Messages = len(msgs)
	for _, m := range msgs {
		if _, err := dsn.Parse(m.Raw); err == nil {
			rep.Reports++
		}
	}
	return rep, nil
}
