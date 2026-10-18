// Package seedtest runs an in-process IMAP server for placement tests: seed
// mailboxes with an inbox, a spam folder marked \Junk (as the real
// providers mark theirs) and a trash, password and XOAUTH2 sign-in, and
// Gmail's X-GM-RAW category search for the tabs. It is built on go-imap's
// own server (imapserver, imapmemserver); the Gmail search, which that
// server doesn't speak, is answered by a thin proxy in front of it.
package seedtest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/emersion/go-sasl"
)

// Server is a running fake IMAP server.
type Server struct {
	// Host and Port to connect to (plain TCP: Config.InsecureIMAP).
	Host string
	Port int

	mem   *imapmemserver.Server
	srv   *imapserver.Server
	ln    net.Listener
	gmail bool

	mu        sync.Mutex
	users     map[string]*imapmemserver.User
	passwords map[string]string
	tokens    map[string]string // access token → user
	// categories of Gmail inbox messages: user → uid → category ("promotions")
	categories map[string]map[uint32]string
	// refuse makes sign-ins of a user fail (a revoked password).
	refuse map[string]bool
	logins []string
}

// Options configures the server.
type Options struct {
	// Gmail advertises X-GM-EXT-1, so clients read tabs.
	Gmail bool
}

// New starts a server; it stops with the test.
func New(t testing.TB, o Options) *Server {
	t.Helper()
	s := &Server{
		mem: imapmemserver.New(), users: map[string]*imapmemserver.User{}, passwords: map[string]string{},
		tokens: map[string]string{}, categories: map[string]map[uint32]string{}, refuse: map[string]bool{},
	}
	s.gmail = o.Gmail
	caps := imap.CapSet{
		imap.CapIMAP4rev1: {}, imap.CapSpecialUse: {}, imap.CapListExtended: {}, imap.CapUIDPlus: {}, imap.CapMove: {},
	}
	s.srv = imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return &session{Session: s.mem.NewSession(), s: s}, nil, nil
		},
		Caps:         caps,
		InsecureAuth: true,
		Logger:       nopLogger{},
	})
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s.ln = ln
	addr := ln.Addr().(*net.TCPAddr)
	s.Host, s.Port = "127.0.0.1", addr.Port
	go s.accept()
	t.Cleanup(func() { _ = ln.Close(); _ = s.srv.Close() })
	return s
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...interface{}) {}

// Mailbox kinds a seed gets.
const (
	Inbox = "INBOX"
	Trash = "Trash"
)

// AddUser creates a seed with its spam folder (named as the provider names
// it, marked \Junk) and a trash folder.
func (s *Server) AddUser(username, password, spam string) {
	u := imapmemserver.NewUser(username, password)
	_ = u.Create(Inbox, nil)
	_ = u.Create(spam, &imap.CreateOptions{SpecialUse: []imap.MailboxAttr{imap.MailboxAttrJunk}})
	trash := Trash
	if strings.HasPrefix(spam, "[Gmail]/") {
		trash = "[Gmail]/Trash"
	}
	_ = u.Create(trash, &imap.CreateOptions{SpecialUse: []imap.MailboxAttr{imap.MailboxAttrTrash}})
	_ = u.Create("Sent", &imap.CreateOptions{SpecialUse: []imap.MailboxAttr{imap.MailboxAttrSent}})
	s.mem.AddUser(u)
	s.mu.Lock()
	s.users[username] = u
	s.passwords[username] = password
	s.mu.Unlock()
}

// AllowToken accepts an XOAUTH2 access token for a user.
func (s *Server) AllowToken(token, username string) {
	s.mu.Lock()
	s.tokens[token] = username
	s.mu.Unlock()
}

// Refuse makes every sign-in of a user fail from now on.
func (s *Server) Refuse(username string) {
	s.mu.Lock()
	s.refuse[username] = true
	s.mu.Unlock()
}

// Logins returns the users that signed in, in order.
func (s *Server) Logins() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.logins...)
}

type literal struct {
	*bytes.Reader
	n int64
}

func (l literal) Size() int64 { return l.n }

// Deliver puts a message in a folder as received at at. category is the
// Gmail tab of an inbox message ("promotions", "updates"; "" for Primary).
func (s *Server) Deliver(username, mailbox string, raw []byte, at time.Time, category string) error {
	s.mu.Lock()
	u := s.users[username]
	s.mu.Unlock()
	if u == nil {
		return fmt.Errorf("seedtest: no user %s", username)
	}
	data, err := u.Append(mailbox, literal{bytes.NewReader(raw), int64(len(raw))}, &imap.AppendOptions{Time: at})
	if err != nil {
		return err
	}
	if category != "" {
		s.mu.Lock()
		if s.categories[username] == nil {
			s.categories[username] = map[uint32]string{}
		}
		s.categories[username][uint32(data.UID)] = category
		s.mu.Unlock()
	}
	return nil
}

// Count returns how many messages a folder holds.
func (s *Server) Count(username, mailbox string) int {
	s.mu.Lock()
	u := s.users[username]
	s.mu.Unlock()
	if u == nil {
		return 0
	}
	st, err := u.Status(mailbox, &imap.StatusOptions{NumMessages: true})
	if err != nil || st.NumMessages == nil {
		return 0
	}
	return int(*st.NumMessages)
}

// session adds XOAUTH2, refusal and MOVE to the in-memory session.
type session struct {
	imapserver.Session
	s *Server
}

func (ss *session) Login(username, password string) error {
	ss.s.mu.Lock()
	refused := ss.s.refuse[username]
	ss.s.mu.Unlock()
	if refused {
		return imapserver.ErrAuthFailed
	}
	if err := ss.Session.Login(username, password); err != nil {
		return err
	}
	ss.s.mu.Lock()
	ss.s.logins = append(ss.s.logins, username)
	ss.s.mu.Unlock()
	return nil
}

func (ss *session) Move(w *imapserver.MoveWriter, numSet imap.NumSet, dest string) error {
	return ss.Session.(imapserver.SessionMove).Move(w, numSet, dest)
}

func (ss *session) AuthenticateMechanisms() []string { return []string{"PLAIN", "XOAUTH2"} }

func (ss *session) Authenticate(mech string) (sasl.Server, error) {
	switch mech {
	case "PLAIN":
		return sasl.NewPlainServer(func(_, username, password string) error { return ss.Login(username, password) }), nil
	case "XOAUTH2":
		return &xoauth2Server{ss: ss}, nil
	}
	return nil, imapserver.ErrAuthFailed
}

type xoauth2Server struct {
	ss *session
}

func (x *xoauth2Server) Next(resp []byte) ([]byte, bool, error) {
	user, token := parseXOAUTH2(resp)
	x.ss.s.mu.Lock()
	owner, ok := x.ss.s.tokens[token]
	pw := x.ss.s.passwords[owner]
	x.ss.s.mu.Unlock()
	if !ok || !strings.EqualFold(owner, user) {
		return nil, true, imapserver.ErrAuthFailed
	}
	return nil, true, x.ss.Login(owner, pw)
}

func parseXOAUTH2(b []byte) (user, token string) {
	for _, part := range strings.Split(string(b), "\x01") {
		switch {
		case strings.HasPrefix(part, "user="):
			user = strings.TrimPrefix(part, "user=")
		case strings.HasPrefix(part, "auth=Bearer "):
			token = strings.TrimPrefix(part, "auth=Bearer ")
		}
	}
	return user, token
}

// accept runs each connection through the proxy that answers Gmail's
// category search and passes everything else to the IMAP server.
func (s *Server) accept() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		inner, outer := net.Pipe()
		go func() { _ = s.srv.Serve(&oneConn{c: inner}) }()
		go s.proxy(c, outer)
	}
}

var (
	gmRaw    = regexp.MustCompile(`(?i)^(\S+) UID SEARCH UID (\S+) X-GM-RAW "category:(\w+)"$`)
	loginRe  = regexp.MustCompile(`(?i)^\S+ LOGIN "?([^" ]+)"?`)
	authRe   = regexp.MustCompile(`(?i)^\S+ AUTHENTICATE XOAUTH2 (\S+)`)
	literalN = regexp.MustCompile(`\{(\d+)\+?\}$`)
)

func (s *Server) proxy(client, server net.Conn) {
	defer func() { _ = client.Close(); _ = server.Close() }()
	var wmu sync.Mutex
	write := func(b []byte) {
		wmu.Lock()
		_, _ = client.Write(b)
		wmu.Unlock()
	}
	go func() {
		// go-imap's server only advertises the capabilities it knows: add
		// Gmail's extension to the CAPABILITY lines it writes.
		sr := bufio.NewReader(server)
		for {
			line, err := sr.ReadString('\n')
			if line != "" {
				if s.gmail && strings.Contains(line, "CAPABILITY IMAP4rev1") {
					line = strings.Replace(line, "CAPABILITY IMAP4rev1", "CAPABILITY IMAP4rev1 X-GM-EXT-1", 1)
				}
				write([]byte(line))
			}
			if err != nil {
				_ = client.Close()
				return
			}
		}
	}()
	r := bufio.NewReader(client)
	user := ""
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		trim := strings.TrimRight(line, "\r\n")
		if m := loginRe.FindStringSubmatch(trim); m != nil {
			user = m[1]
		}
		if m := authRe.FindStringSubmatch(trim); m != nil {
			if b, err := base64.StdEncoding.DecodeString(m[1]); err == nil {
				user, _ = parseXOAUTH2(b)
			}
		}
		if m := gmRaw.FindStringSubmatch(trim); m != nil {
			write([]byte(s.searchCategory(user, m[2], strings.ToLower(m[3]), m[1])))
			continue
		}
		if _, err := server.Write([]byte(line)); err != nil {
			return
		}
		if m := literalN.FindStringSubmatch(trim); m != nil {
			n, _ := strconv.Atoi(m[1])
			if _, err := io.CopyN(server, r, int64(n)); err != nil {
				return
			}
		}
	}
}

func (s *Server) searchCategory(user, set, category, tag string) string {
	s.mu.Lock()
	cats := s.categories[user]
	s.mu.Unlock()
	var hits []string
	for _, f := range strings.Split(set, ",") {
		n, err := strconv.ParseUint(f, 10, 32)
		if err != nil {
			continue
		}
		if cats[uint32(n)] == category {
			hits = append(hits, f)
		}
	}
	out := "* SEARCH"
	if len(hits) > 0 {
		out += " " + strings.Join(hits, " ")
	}
	return out + "\r\n" + tag + " OK SEARCH completed\r\n"
}

// oneConn is a listener that hands out one connection.
type oneConn struct {
	c    net.Conn
	once sync.Once
	done chan struct{}
	mu   sync.Mutex
}

func (l *oneConn) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.done == nil {
		l.done = make(chan struct{})
	}
	done := l.done
	l.mu.Unlock()
	var c net.Conn
	l.once.Do(func() { c = l.c })
	if c != nil {
		return c, nil
	}
	<-done
	return nil, net.ErrClosed
}

func (l *oneConn) Close() error {
	l.mu.Lock()
	if l.done == nil {
		l.done = make(chan struct{})
	}
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	l.mu.Unlock()
	return nil
}

func (l *oneConn) Addr() net.Addr { return l.c.LocalAddr() }
