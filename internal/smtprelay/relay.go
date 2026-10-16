// Package smtprelay is the SMTP relay: apps that only speak SMTP send
// transactional mail through it instead of the HTTP API. It is off until
// an instance admin turns it on (Instance admin › Instance settings), and
// listens on 587 with STARTTLS and on 465 with TLS. Credentials are SMTP
// credentials, send-only keys of a workspace: the username is the key's ID,
// the password its secret. The From address must be a verified
// Transactional sender. Mail is logged as "Raw (SMTP)" and goes through the
// workspace's Transactional route, suppression rules and log.
//
// The server is github.com/emersion/go-smtp: a maintained, dependency-light
// SMTP server library (go-sasl and go-message, already in the module graph
// through go-imap) with STARTTLS, implicit TLS, AUTH PLAIN and graceful
// shutdown, used by Maddy and other mail servers. Writing an SMTP server
// from net/textproto would duplicate that work and its edge cases.
//
// The certificate comes from Let's Encrypt (golang.org/x/crypto's autocert,
// with its cache sealed in Postgres so every web node shares it and any of
// them answers the HTTP-01 challenge) or from files an operator manages.
package smtprelay

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Limits of the relay.
const (
	// MaxMessageBytes is the largest message the relay accepts; the route's
	// providers may accept less (the reply then says so).
	MaxMessageBytes = 40 << 20
	DefaultPoll     = 30 * time.Second
	shutdownGrace   = 15 * time.Second
)

// Sender authenticates SMTP credentials and queues their mail (the
// transactional domain).
type Sender interface {
	AuthenticateSMTP(ctx context.Context, username, password string, ip *netip.Addr) (transactional.SMTPCredential, error)
	SendRaw(ctx context.Context, cred transactional.SMTPCredential, m transactional.RawMessage) ([]transactional.RawResult, error)
}

// Relay runs the SMTP relay's listeners in a web process.
type Relay struct {
	Pool   *pgxpool.Pool
	Q      *store.Queries
	Box    *secrets.Box
	Audit  *audit.Logger
	Log    *slog.Logger
	Sender Sender
	Now    func() time.Time
	// ListenHost is the address the listeners bind to ("" for every
	// interface).
	ListenHost string
	// Poll is how often a running relay reads its settings again.
	Poll time.Duration
	// Resolver and DialTimeout serve the status checks.
	Resolver    *net.Resolver
	DialTimeout time.Duration
	// RootCAs verify the certificate in the status checks (nil: the
	// system's roots).
	RootCAs *x509.CertPool

	mu      sync.Mutex
	running *servers
	acme    *acmeSource
	wake    chan struct{}
	// sessions counts open SMTP sessions, so a stop lets them finish.
	sessions atomic.Int64
}

func (r *Relay) wakeCh() chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wake == nil {
		r.wake = make(chan struct{}, 1)
	}
	return r.wake
}

// Wake makes a running relay read its settings now (after a save).
func (r *Relay) Wake() {
	select {
	case r.wakeCh() <- struct{}{}:
	default:
	}
}

// Stop stops the listeners at once (tests; Run stops them when its
// context ends).
func (r *Relay) Stop() { r.stop(context.Background()) }

func (r *Relay) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func (r *Relay) log() *slog.Logger {
	if r.Log != nil {
		return r.Log
	}
	return slog.Default()
}

// Config is how the relay listens.
type Config struct {
	Enabled      bool
	Hostname     string
	StartTLSPort int
	TLSPort      int
	CertSource   string // acme, files
	CertFile     string
	KeyFile      string
}

func configOf(row store.RelaySettingsRow) Config {
	return Config{Enabled: row.Enabled, Hostname: row.Hostname, StartTLSPort: int(row.StarttlsPort), TLSPort: int(row.TlsPort),
		CertSource: row.CertSource, CertFile: row.CertFile, KeyFile: row.KeyFile}
}

// isDemo reports whether the settings are the demo's (internal/demo): on,
// at reputecast.acme.io with passing checks, but nothing listens and the
// periodic check leaves them be, until an instance admin saves the
// settings.
func isDemo(status []byte) bool {
	var v struct {
		Demo bool `json:"demo"`
	}
	_ = json.Unmarshal(status, &v)
	return v.Demo
}

// servers are the two listeners of a running relay.
type servers struct {
	cfg       Config
	starttls  *smtp.Server
	tls       *smtp.Server
	certs     certSource
	listeners []net.Listener
}

// Run keeps the relay's listeners in line with its settings until ctx
// ends, then stops them gracefully (sessions get a few seconds to finish).
func (r *Relay) Run(ctx context.Context) error {
	poll := r.Poll
	if poll <= 0 {
		poll = DefaultPoll
	}
	t := time.NewTicker(poll)
	defer t.Stop()
	wake := r.wakeCh()
	last := ""
	for {
		// The same error (a port taken, say) is logged once, not every poll.
		err := r.Reconcile(ctx)
		switch {
		case err != nil && ctx.Err() == nil && err.Error() != last:
			r.log().WarnContext(ctx, "smtp relay", "err", err)
			last = err.Error()
		case err == nil:
			last = ""
		}
		select {
		case <-ctx.Done():
			r.stop(context.Background())
			return nil
		case <-t.C:
		case <-wake:
		}
	}
}

// Reconcile starts, restarts or stops the listeners to match the saved
// settings.
func (r *Relay) Reconcile(ctx context.Context) error {
	row, err := r.Q.RelaySettings(ctx)
	if err != nil {
		return err
	}
	cfg := configOf(row)
	r.mu.Lock()
	cur := r.running
	r.mu.Unlock()
	want := cfg.Enabled && cfg.Hostname != "" && !isDemo(row.Status)
	if cur != nil && (!want || cur.cfg != cfg) {
		r.stop(ctx)
		cur = nil
	}
	if !want || cur != nil {
		return nil
	}
	return r.start(ctx, cfg)
}

// Listening reports the configuration the relay listens with, if any.
func (r *Relay) Listening() (Config, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running == nil {
		return Config{}, false
	}
	return r.running.cfg, true
}

func (r *Relay) start(ctx context.Context, cfg Config) error {
	var certs certSource
	switch cfg.CertSource {
	case "files":
		certs = &fileSource{certFile: cfg.CertFile, keyFile: cfg.KeyFile, relay: r}
	default:
		a := newACMESource(r, cfg.Hostname)
		certs = a
		r.mu.Lock()
		r.acme = a
		r.mu.Unlock()
	}
	tlsCfg := &tls.Config{GetCertificate: certs.GetCertificate, MinVersion: tls.VersionTLS12}
	be := &backend{relay: r}
	mk := func(port int) *smtp.Server {
		s := smtp.NewServer(be)
		s.Addr = net.JoinHostPort(r.ListenHost, strconv.Itoa(port))
		s.Domain = cfg.Hostname
		s.TLSConfig = tlsCfg
		s.MaxMessageBytes = MaxMessageBytes
		s.MaxRecipients = transactional.MaxRawRecipients
		s.ReadTimeout, s.WriteTimeout = 5*time.Minute, 5*time.Minute
		s.AllowInsecureAuth = false
		s.EnableSMTPUTF8 = false
		s.ErrorLog = slogLogger{r.log()}
		return s
	}
	st := &servers{cfg: cfg, starttls: mk(cfg.StartTLSPort), tls: mk(cfg.TLSPort), certs: certs}
	l1, err := (&net.ListenConfig{}).Listen(ctx, "tcp", st.starttls.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", st.starttls.Addr, err)
	}
	l2, err := (&net.ListenConfig{}).Listen(ctx, "tcp", st.tls.Addr)
	if err != nil {
		_ = l1.Close()
		return fmt.Errorf("listen on %s: %w", st.tls.Addr, err)
	}
	st.listeners = []net.Listener{l1, l2}
	go func() { _ = st.starttls.Serve(l1) }()
	go func() { _ = st.tls.Serve(tls.NewListener(l2, tlsCfg)) }()
	r.mu.Lock()
	r.running = st
	r.mu.Unlock()
	r.log().InfoContext(ctx, "smtp relay listening", "hostname", cfg.Hostname, "starttls", st.starttls.Addr, "tls", st.tls.Addr, "certificate", cfg.CertSource)
	return nil
}

func (r *Relay) stop(ctx context.Context) {
	r.mu.Lock()
	st := r.running
	r.running = nil
	r.mu.Unlock()
	if st == nil {
		return
	}
	// No new connections; sessions get a few seconds to finish their
	// message, then the rest are closed. (go-smtp's Shutdown would wait
	// on its own WaitGroup while its accept loop may still Add to it.)
	for _, l := range st.listeners {
		_ = l.Close()
	}
	deadline := time.Now().Add(shutdownGrace)
	for r.sessions.Load() > 0 && time.Now().Before(deadline) && ctx.Err() == nil {
		time.Sleep(50 * time.Millisecond)
	}
	for _, srv := range []*smtp.Server{st.starttls, st.tls} {
		if err := srv.Close(); err != nil && !errors.Is(err, smtp.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			r.log().Debug("smtp relay close", "err", err)
		}
	}
	r.log().Info("smtp relay stopped")
}

type slogLogger struct{ l *slog.Logger }

func (s slogLogger) Printf(format string, v ...interface{}) {
	s.l.Debug("smtp relay", "msg", fmt.Sprintf(format, v...))
}
func (s slogLogger) Println(v ...interface{}) { s.l.Debug("smtp relay", "msg", fmt.Sprint(v...)) }

// backend opens a session per connection.
type backend struct{ relay *Relay }

// NewSession implements smtp.Backend.
func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	b.relay.sessions.Add(1)
	s := &session{relay: b.relay, conn: c}
	if ap, err := netip.ParseAddrPort(c.Conn().RemoteAddr().String()); err == nil {
		ip := ap.Addr().Unmap()
		s.ip = &ip
	}
	return s, nil
}

// session is one SMTP connection.
type session struct {
	relay  *Relay
	conn   *smtp.Conn
	ip     *netip.Addr
	cred   *transactional.SMTPCredential
	rcpts  []string
	closed atomic.Bool
}

var _ smtp.AuthSession = (*session)(nil)

// AuthMechanisms implements smtp.AuthSession.
func (s *session) AuthMechanisms() []string { return []string{sasl.Plain, sasl.Login} }

// Auth implements smtp.AuthSession.
func (s *session) Auth(mech string) (sasl.Server, error) {
	check := func(username, password string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cred, err := s.relay.Sender.AuthenticateSMTP(ctx, username, password, s.ip)
		if err != nil {
			return authError(err)
		}
		s.cred = &cred
		return nil
	}
	switch mech {
	case sasl.Plain:
		return sasl.NewPlainServer(func(_, username, password string) error { return check(username, password) }), nil
	case sasl.Login:
		return &loginServer{check: check}, nil
	}
	return nil, smtp.ErrAuthUnknownMechanism
}

func authError(err error) error {
	switch {
	case errors.Is(err, transactional.ErrSMTPAuthFailed), errors.Is(err, transactional.ErrSMTPNotSMTPKey), errors.Is(err, transactional.ErrSMTPKeyRevoked),
		errors.Is(err, transactional.ErrSMTPKeyExpired), errors.Is(err, transactional.ErrSMTPCannotSend), errors.Is(err, transactional.ErrSMTPWorkspaceGone),
		errors.Is(err, transactional.ErrSMTPIntegrationKey):
		return &smtp.SMTPError{Code: 535, EnhancedCode: smtp.EnhancedCode{5, 7, 8}, Message: "Authentication failed: " + err.Error()}
	}
	return &smtp.SMTPError{Code: 454, EnhancedCode: smtp.EnhancedCode{4, 7, 0}, Message: "Temporary authentication failure, try again"}
}

// Mail implements smtp.Session.
func (s *session) Mail(_ string, _ *smtp.MailOptions) error {
	if s.cred == nil {
		return smtp.ErrAuthRequired
	}
	return nil
}

// Rcpt implements smtp.Session.
func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	if s.cred == nil {
		return smtp.ErrAuthRequired
	}
	addr := strings.Trim(strings.TrimSpace(to), "<>")
	if at := strings.LastIndexByte(addr, '@'); at < 1 || at == len(addr)-1 || strings.ContainsAny(addr, " ,;") {
		return &smtp.SMTPError{Code: 501, EnhancedCode: smtp.EnhancedCode{5, 1, 3}, Message: "“" + clip(addr, 80) + "” isn't a valid email address"}
	}
	if len(s.rcpts) >= transactional.MaxRawRecipients {
		return &smtp.SMTPError{Code: 452, EnhancedCode: smtp.EnhancedCode{4, 5, 3}, Message: fmt.Sprintf("Too many recipients: at most %d per message", transactional.MaxRawRecipients)}
	}
	s.rcpts = append(s.rcpts, strings.ToLower(addr))
	return nil
}

// Data implements smtp.Session.
func (s *session) Data(r io.Reader) error {
	if s.cred == nil {
		_, _ = io.Copy(io.Discard, r)
		return smtp.ErrAuthRequired
	}
	raw, err := io.ReadAll(io.LimitReader(r, MaxMessageBytes+1))
	if err != nil {
		return &smtp.SMTPError{Code: 451, EnhancedCode: smtp.EnhancedCode{4, 3, 0}, Message: "The message couldn't be read, try again"}
	}
	if len(raw) > MaxMessageBytes {
		_, _ = io.Copy(io.Discard, r)
		return &smtp.SMTPError{Code: 552, EnhancedCode: smtp.EnhancedCode{5, 3, 4}, Message: fmt.Sprintf("The message is larger than %d MB", MaxMessageBytes>>20)}
	}
	m, err := parseMessage(bytes.NewReader(raw))
	if err != nil {
		return &smtp.SMTPError{Code: 554, EnhancedCode: smtp.EnhancedCode{5, 6, 0}, Message: "The message couldn't be parsed: " + clip(err.Error(), 200)}
	}
	m.Recipients, m.IP = s.rcpts, s.ip
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	results, err := s.relay.Sender.SendRaw(ctx, *s.cred, m)
	if err != nil {
		return replyFor(err)
	}
	return resultReply(results)
}

// Reset implements smtp.Session.
func (s *session) Reset() { s.rcpts = nil }

// Logout implements smtp.Session.
func (s *session) Logout() error {
	if s.closed.CompareAndSwap(false, true) {
		s.relay.sessions.Add(-1)
	}
	return nil
}

// replyFor turns a refusal of the whole message into an SMTP reply that
// says why.
func replyFor(err error) error {
	var p *transactional.Problem
	if !errors.As(err, &p) {
		return &smtp.SMTPError{Code: 451, EnhancedCode: smtp.EnhancedCode{4, 3, 0}, Message: "Something went wrong on our side; nothing was sent, try again"}
	}
	msg := oneLine(p.Detail)
	switch p.Code {
	case transactional.CodeSenderNotVerified:
		return &smtp.SMTPError{Code: 550, EnhancedCode: smtp.EnhancedCode{5, 7, 1}, Message: msg}
	case transactional.CodeRateLimited:
		return &smtp.SMTPError{Code: 451, EnhancedCode: smtp.EnhancedCode{4, 7, 1}, Message: msg}
	case transactional.CodeNoHealthyRoute:
		return &smtp.SMTPError{Code: 451, EnhancedCode: smtp.EnhancedCode{4, 4, 3}, Message: msg}
	case transactional.CodeAttachmentsTooLarge:
		return &smtp.SMTPError{Code: 552, EnhancedCode: smtp.EnhancedCode{5, 3, 4}, Message: msg}
	case transactional.CodeTemplateNotAllowed:
		return &smtp.SMTPError{Code: 550, EnhancedCode: smtp.EnhancedCode{5, 7, 1}, Message: msg}
	case transactional.CodeInvalidRecipient:
		return &smtp.SMTPError{Code: 550, EnhancedCode: smtp.EnhancedCode{5, 1, 3}, Message: msg}
	}
	if msg == "" {
		msg = p.Title
	}
	return &smtp.SMTPError{Code: 550, EnhancedCode: smtp.EnhancedCode{5, 6, 0}, Message: msg}
}

// resultReply answers DATA: 250 when at least one recipient was queued,
// 550 when every one was stopped (suppressed or opted out).
func resultReply(results []transactional.RawResult) error {
	var queued, stopped []transactional.RawResult
	for _, r := range results {
		if r.Status == "rejected" {
			stopped = append(stopped, r)
		} else {
			queued = append(queued, r)
		}
	}
	if len(queued) == 0 && len(stopped) > 0 {
		return &smtp.SMTPError{Code: 550, EnhancedCode: smtp.EnhancedCode{5, 7, 1}, Message: oneLine(stopped[0].Detail)}
	}
	// A 250 reply's text is the Data error's message when it's an
	// SMTPError with code 250; go-smtp writes it as the final reply.
	ids := make([]string, 0, len(queued))
	for _, r := range queued {
		ids = append(ids, r.ID)
	}
	msg := "Queued as " + strings.Join(ids, ", ")
	if len(stopped) > 0 {
		msg += fmt.Sprintf("; %d not sent: %s", len(stopped), oneLine(stopped[0].Detail))
	}
	return &smtp.SMTPError{Code: 250, EnhancedCode: smtp.EnhancedCode{2, 0, 0}, Message: clip(msg, 400)}
}

func oneLine(s string) string {
	return clip(strings.Join(strings.Fields(s), " "), 400)
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// loginServer is the server side of AUTH LOGIN (draft-murchison-sasl-login),
// which go-sasl only implements for clients.
type loginServer struct {
	check    func(username, password string) error
	step     int
	username string
}

// Next implements sasl.Server.
func (l *loginServer) Next(response []byte) ([]byte, bool, error) {
	switch l.step {
	case 0:
		l.step = 1
		if response == nil {
			return []byte("Username:"), false, nil
		}
		l.username = string(response)
		l.step = 2
		return []byte("Password:"), false, nil
	case 1:
		l.username = string(response)
		l.step = 2
		return []byte("Password:"), false, nil
	case 2:
		l.step = 3
		return nil, true, l.check(l.username, string(response))
	}
	return nil, true, sasl.ErrUnexpectedClientResponse
}
