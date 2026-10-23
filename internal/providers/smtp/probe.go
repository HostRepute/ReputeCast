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

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/smtp/bouncebox"
	"github.com/hostrepute/reputecast/internal/providers/smtp/dsn"
)

// This file is the SMTP side of "Connect a sending provider": the connection
// and bounce-mailbox tests.

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
		ce := &providers.CallError{Op: "SMTP connect"}
		ce.Message, ce.Text = say(mNoHost, nil)
		return out, ce
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
		ce := &providers.CallError{Op: "SMTP connect", Temporary: true, Cause: err}
		ce.Message, ce.Text = dialMessage(err)
		return out, ce
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

	// "STARTTLS, TLS 1.3 · AUTH PLAIN accepted"
	summary := i18n.Args{"tls": "none", "auth": "none"}
	switch p.s.TLS {
	case "tls":
		summary["tls"], summary["version"] = "implicit", tlsVersion(tlsState)
	case "none":
	default:
		if !starttls {
			ce := &providers.CallError{Op: "STARTTLS"}
			ce.Message, ce.Text = say(mNoStartTLS, i18n.Args{"host": p.s.Host, "port": strconv.Itoa(p.s.Port)})
			return out, ce
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return out, &providers.CallError{Op: "STARTTLS", Message: err.Error(), Temporary: true, Cause: err}
		}
		st, _ := c.TLSConnectionState()
		tlsState = &st
		summary["tls"], summary["version"] = "starttls", tlsVersion(tlsState)
	}
	if p.s.Username != "" && p.s.Auth != "none" {
		_, mechs := c.Extension("AUTH")
		mech := pickMechanism(p.s.Auth, mechs)
		if mech == "" {
			ce := &providers.CallError{Op: "SMTP AUTH"}
			ce.Message, ce.Text = say(mNoLogin, i18n.Args{"host": p.s.Host, "offered": strings.TrimSpace(mechs)})
			return out, ce
		}
		var a smtp.Auth = plainAuth{user: p.s.Username, pass: p.c.Password}
		if mech == "LOGIN" {
			a = loginAuth{user: p.s.Username, pass: p.c.Password}
		}
		if err := c.Auth(a); err != nil {
			return out, &providers.CallError{Op: "SMTP AUTH", Status: 535, Message: err.Error(), Auth: true, Cause: err}
		}
		summary["auth"], summary["mechanism"] = "accepted", mech
		out.Auth = strings.ToLower(mech)
	}
	_ = c.Quit()
	out.Summary, out.SummaryText = say(mSummary, summary)
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

// dialMessage turns a dial error into the words people search for, with
// the message they come from (none for an error it doesn't know).
func dialMessage(err error) (string, i18n.Text) {
	low := strings.ToLower(err.Error())
	switch {
	case strings.Contains(low, "connection refused"):
		return say(mRefused, nil)
	case strings.Contains(low, "no such host"):
		return say(mNotFound, nil)
	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline"):
		return say(mTimedOut, nil)
	}
	return err.Error(), i18n.Text{}
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
		ce := &providers.CallError{Op: "bounce mailbox"}
		ce.Message, ce.Text = say(mMailboxMissing, nil)
		return providers.MailboxReport{}, ce
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
			ce := &providers.CallError{Op: cfg.Label() + " LOGIN", Auth: true, Cause: err}
			ce.Message, ce.Text = say(mMailboxAuth, nil)
			return rep, ce
		}
		ce := &providers.CallError{Op: cfg.Label() + " connect", Temporary: true, Cause: err}
		ce.Message, ce.Text = dialMessage(err)
		return rep, ce
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
