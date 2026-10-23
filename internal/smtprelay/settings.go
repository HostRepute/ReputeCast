package smtprelay

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// SMTPRelayCheck is one status check.
type SMTPRelayCheck struct {
	Key    string `json:"key" enum:"dns,starttls,tls,certificate"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail" example:"reputecast.acme.io points to 203.0.113.40"`
	// Port is the port a port check connected to.
	Port int `json:"port,omitempty"`
}

// storedCheck is a check as smtp_relay.status keeps it: its English detail
// and the message it came from, so it reads in each reader's language.
type storedCheck struct {
	SMTPRelayCheck
	Text i18n.Text `json:"text,omitzero"`
}

// storedStatus is smtp_relay.status: the last checks with their messages.
type storedStatus struct {
	CheckedAt *time.Time    `json:"checkedAt,omitempty"`
	Checks    []storedCheck `json:"checks"`
	From      string        `json:"from,omitempty"`
}

// SMTPRelayStatus is the last status check.
type SMTPRelayStatus struct {
	CheckedAt *time.Time       `json:"checkedAt,omitempty"`
	Checks    []SMTPRelayCheck `json:"checks"`
	// From says where the port checks ran: this server, connecting to the
	// hostname's address (no outside service is asked).
	From string `json:"from,omitempty"`
}

// SMTPRelayCertificate is the certificate the relay serves.
type SMTPRelayCertificate struct {
	NotAfter time.Time `json:"notAfter"`
	Issuer   string    `json:"issuer" example:"Let's Encrypt"`
}

// SMTPRelaySettings is the SMTP relay section of Instance settings.
type SMTPRelaySettings struct {
	Enabled      bool                  `json:"enabled"`
	Hostname     string                `json:"hostname" example:"reputecast.acme.io"`
	StartTLSPort int                   `json:"starttlsPort" example:"587"`
	TLSPort      int                   `json:"tlsPort" example:"465"`
	CertSource   string                `json:"certSource" enum:"acme,files"`
	CertFile     string                `json:"certFile"`
	KeyFile      string                `json:"keyFile"`
	Certificate  *SMTPRelayCertificate `json:"certificate,omitempty"`
	// RenewError is why the last certificate request or load failed.
	RenewError    string     `json:"renewError,omitempty"`
	RenewFailedAt *time.Time `json:"renewFailedAt,omitempty"`
	EnabledBy     string     `json:"enabledBy,omitempty"`
	EnabledAt     *time.Time `json:"enabledAt,omitempty"`
	// UnreachableSince is set while a port check fails.
	UnreachableSince *time.Time      `json:"unreachableSince,omitempty"`
	Status           SMTPRelayStatus `json:"status"`
	// SuggestedHostname is the public URL's host.
	SuggestedHostname string `json:"suggestedHostname"`
}

// SMTPRelayInput saves the relay's settings.
type SMTPRelayInput struct {
	Enabled      bool   `json:"enabled"`
	Hostname     string `json:"hostname" maxLength:"253"`
	StartTLSPort int    `json:"starttlsPort" minimum:"1" maximum:"65535" default:"587"`
	TLSPort      int    `json:"tlsPort" minimum:"1" maximum:"65535" default:"465"`
	CertSource   string `json:"certSource" enum:"acme,files" default:"acme"`
	CertFile     string `json:"certFile,omitempty" maxLength:"1024" doc:"PEM certificate chain (files)"`
	KeyFile      string `json:"keyFile,omitempty" maxLength:"1024" doc:"PEM private key (files)"`
}

// PublicURL returns the instance's public URL (for the suggested hostname).
type PublicURL func(context.Context) string

// Settings returns the relay's settings and last status.
func (r *Relay) Settings(ctx context.Context, publicURL string) (SMTPRelaySettings, error) {
	row, err := r.Q.RelaySettings(ctx)
	if err != nil {
		return SMTPRelaySettings{}, err
	}
	out := SMTPRelaySettings{Enabled: row.Enabled, Hostname: row.Hostname, StartTLSPort: int(row.StarttlsPort), TLSPort: int(row.TlsPort),
		CertSource: row.CertSource, CertFile: row.CertFile, KeyFile: row.KeyFile, RenewError: row.RenewError, RenewFailedAt: row.RenewFailedAt,
		EnabledBy: row.EnabledByName, EnabledAt: row.EnabledAt, UnreachableSince: row.UnreachableSince, Status: SMTPRelayStatus{Checks: []SMTPRelayCheck{}}}
	if row.CertNotAfter != nil {
		out.Certificate = &SMTPRelayCertificate{NotAfter: *row.CertNotAfter, Issuer: row.CertIssuer}
	}
	var st storedStatus
	_ = json.Unmarshal(row.Status, &st)
	out.Status.CheckedAt, out.Status.From = st.CheckedAt, st.From
	l := i18n.LangFrom(ctx)
	for _, c := range st.Checks {
		v := c.SMTPRelayCheck
		v.Detail = detailIn(l, c)
		out.Status.Checks = append(out.Status.Checks, v)
	}
	if u, err := url.Parse(publicURL); err == nil {
		out.SuggestedHostname = u.Hostname()
	}
	return out, nil
}

// Save changes the relay's settings: audited as turned on, off or changed.
func (r *Relay) Save(ctx context.Context, in SMTPRelayInput, publicURL string) (SMTPRelaySettings, error) {
	cur, err := r.Q.RelaySettings(ctx)
	if err != nil {
		return SMTPRelaySettings{}, err
	}
	host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(in.Hostname), "."))
	if in.StartTLSPort == 0 {
		in.StartTLSPort = 587
	}
	if in.TLSPort == 0 {
		in.TLSPort = 465
	}
	if in.CertSource == "" {
		in.CertSource = "acme"
	}
	if in.Enabled && host == "" {
		return SMTPRelaySettings{}, api.Invalid("body.hostname", say(ctx, "problem.hostnameNeeded", nil))
	}
	if host != "" && !validHostname(host) {
		return SMTPRelaySettings{}, api.Invalid("body.hostname", say(ctx, "problem.hostnameFormat", nil))
	}
	if in.StartTLSPort == in.TLSPort {
		return SMTPRelaySettings{}, api.Invalid("body.tlsPort", say(ctx, "problem.portsDiffer", nil))
	}
	if in.CertSource == "files" {
		if strings.TrimSpace(in.CertFile) == "" || strings.TrimSpace(in.KeyFile) == "" {
			return SMTPRelaySettings{}, api.Invalid("body.certFile", say(ctx, "problem.certFiles", nil))
		}
		if _, err := tls.LoadX509KeyPair(in.CertFile, in.KeyFile); err != nil {
			return SMTPRelaySettings{}, api.Invalid("body.certFile", say(ctx, "problem.certUnreadable", i18n.Args{"error": err.Error()}))
		}
	}
	p := api.PrincipalFrom(ctx)
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		u := p.UserID
		by = &u
	}
	enabledBy, enabledAt := cur.EnabledBy, cur.EnabledAt
	if in.Enabled && !cur.Enabled {
		now := r.now()
		enabledBy, enabledAt = by, &now
	}
	action := "smtp_relay.updated"
	switch {
	case in.Enabled && !cur.Enabled:
		action = "smtp_relay.enabled"
	case !in.Enabled && cur.Enabled:
		action = "smtp_relay.disabled"
	}
	before := map[string]any{"enabled": cur.Enabled, "hostname": cur.Hostname, "starttls_port": cur.StarttlsPort, "tls_port": cur.TlsPort, "cert_source": cur.CertSource}
	after := map[string]any{"enabled": in.Enabled, "hostname": host, "starttls_port": in.StartTLSPort, "tls_port": in.TLSPort, "cert_source": in.CertSource}
	err = db.Tx(ctx, r.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RelayUpdate(ctx, store.RelayUpdateParams{Enabled: in.Enabled, Hostname: host, StarttlsPort: int32(in.StartTLSPort), TlsPort: int32(in.TLSPort),
			CertSource: in.CertSource, CertFile: strings.TrimSpace(in.CertFile), KeyFile: strings.TrimSpace(in.KeyFile), EnabledBy: enabledBy, EnabledAt: enabledAt,
			UpdatedBy: by, UpdatedAt: r.now()}); err != nil {
			return err
		}
		summary := "Changed the SMTP relay"
		switch action {
		case "smtp_relay.enabled":
			summary = fmt.Sprintf("Turned the SMTP relay on at %s (ports %d and %d)", host, in.StartTLSPort, in.TLSPort)
		case "smtp_relay.disabled":
			summary = "Turned the SMTP relay off"
		}
		return r.Audit.LogWith(ctx, q, audit.Event{Action: action, TargetType: "smtp_relay", TargetName: firstNonEmpty(host, cur.Hostname, "SMTP relay"),
			Before: before, After: after, Metadata: map[string]any{"summary": summary}})
	})
	if err != nil {
		return SMTPRelaySettings{}, err
	}
	// A relay running in this process picks the change up at once; other
	// web processes within a poll.
	r.Wake()
	return r.Settings(ctx, publicURL)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func validHostname(h string) bool {
	if len(h) > 253 || !strings.Contains(h, ".") {
		return h == "localhost"
	}
	for _, label := range strings.Split(h, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, c := range label {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	return true
}

// Check runs the status checks: the hostname resolves, both ports answer
// at its address (connecting from this server), and the certificate is
// valid. While a port doesn't answer, instance admins are told once.
func (r *Relay) Check(ctx context.Context, publicURL string) (SMTPRelaySettings, error) {
	row, err := r.Q.RelaySettings(ctx)
	if err != nil {
		return SMTPRelaySettings{}, err
	}
	cfg := configOf(row)
	now := r.now()
	st := storedStatus{CheckedAt: &now, Checks: []storedCheck{}, From: "this server"}
	if cfg.Hostname == "" {
		st.Checks = append(st.Checks, check("dns", false, 0, "check.noHostname", nil))
	} else {
		st.Checks = append(st.Checks, r.checks(ctx, cfg)...)
	}
	reachable := true
	for _, c := range st.Checks {
		if (c.Key == "starttls" || c.Key == "tls") && !c.OK {
			reachable = false
		}
	}
	since := row.UnreachableSince
	newly := false
	switch {
	case !cfg.Enabled || reachable:
		since = nil
	case since == nil:
		since, newly = &now, true
	}
	b, _ := json.Marshal(st)
	err = db.Tx(ctx, r.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RelaySetStatus(ctx, store.RelaySetStatusParams{Status: b, UnreachableSince: since}); err != nil {
			return err
		}
		if !newly {
			return nil
		}
		admins, err := q.InstanceAdminAdminNames(ctx)
		if err != nil {
			return err
		}
		// The first port that failed, in the notification's words.
		body := i18n.Args{"why": "unreachable", "port": "", "host": cfg.Hostname, "error": ""}
		for _, c := range st.Checks {
			if (c.Key == "starttls" || c.Key == "tls") && !c.OK {
				for k, v := range c.Text.Args {
					body[k] = v
				}
				if c.Text.ID == msgs.M("check.portUnresolved").ID() {
					body["why"] = "unresolved"
				}
				break
			}
		}
		for _, a := range admins {
			if err := notify.ToUser(ctx, tx, a.ID, notify.Notification{Kind: notify.KindSMTPRelayUnreachable, Severity: "bad",
				TitleText: msgs.M("notify.title").Keep(nil), BodyText: msgs.M("notify.body").Keep(body), Link: "/admin/instance#smtp-relay",
				Data: map[string]any{"hostname": cfg.Hostname}}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return SMTPRelaySettings{}, err
	}
	return r.Settings(ctx, publicURL)
}

func (r *Relay) checks(ctx context.Context, cfg Config) []storedCheck {
	res := r.Resolver
	if res == nil {
		res = net.DefaultResolver
	}
	timeout := r.DialTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	out := []storedCheck{}
	lctx, cancel := context.WithTimeout(ctx, timeout)
	addrs, err := res.LookupHost(lctx, cfg.Hostname)
	cancel()
	host := i18n.Args{"host": cfg.Hostname}
	if err != nil || len(addrs) == 0 {
		msg := "check.unresolved"
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			msg = "check.unresolvedAdd"
		}
		out = append(out, check("dns", false, 0, msg, host))
		for _, k := range []struct {
			key  string
			port int
		}{{"starttls", cfg.StartTLSPort}, {"tls", cfg.TLSPort}} {
			out = append(out, check(k.key, false, k.port, "check.portUnresolved", i18n.Args{"port": portArg(k.port), "host": cfg.Hostname}))
		}
		out = append(out, check("certificate", false, 0, "check.certUnresolved", host))
		return out
	}
	out = append(out, check("dns", true, 0, "check.resolves", i18n.Args{"host": cfg.Hostname, "addresses": strings.Join(addrs, ", ")}))
	// Each port must answer at one of the hostname's addresses.
	ip := addrs[0]
	dialAny := func(port int, useTLS bool) error {
		var err error
		for _, a := range addrs {
			if err = dialOne(ctx, a, cfg.Hostname, port, useTLS, timeout); err == nil {
				ip = a
				return nil
			}
		}
		return err
	}
	for _, k := range []struct {
		key    string
		port   int
		useTLS bool
	}{{"starttls", cfg.StartTLSPort, false}, {"tls", cfg.TLSPort, true}} {
		if err := dialAny(k.port, k.useTLS); err != nil {
			out = append(out, check(k.key, false, k.port, "check.portUnreachable", i18n.Args{"port": portArg(k.port), "host": cfg.Hostname, "error": shortNetErr(err)}))
			continue
		}
		out = append(out, check(k.key, true, k.port, "check.portReachable", i18n.Args{"port": portArg(k.port)}))
	}
	out = append(out, r.certCheck(ctx, cfg, ip, timeout))
	return out
}

// dialOne connects to one address of the relay and reads its greeting.
func dialOne(ctx context.Context, ip, hostname string, port int, useTLS bool, timeout time.Duration) error {
	d := &net.Dialer{Timeout: timeout}
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	var conn net.Conn
	var err error
	if useTLS {
		td := &tls.Dialer{NetDialer: d, Config: &tls.Config{ServerName: hostname, InsecureSkipVerify: true}} //nolint:gosec // a reachability check; the certificate check verifies
		conn, err = td.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "220") {
		return fmt.Errorf("answered %q instead of 220", strings.TrimSpace(line))
	}
	return nil
}

func (r *Relay) certCheck(ctx context.Context, cfg Config, ip string, timeout time.Duration) storedCheck {
	tc := &tls.Config{ServerName: cfg.Hostname}
	if r.RootCAs != nil {
		tc.RootCAs = r.RootCAs
	}
	td := &tls.Dialer{NetDialer: &net.Dialer{Timeout: timeout}, Config: tc}
	raw, err := td.DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(cfg.TLSPort)))
	if err != nil {
		return check("certificate", false, 0, "check.certUnverified", i18n.Args{"error": shortNetErr(err)})
	}
	conn := raw.(*tls.Conn)
	defer func() { _ = conn.Close() }()
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return check("certificate", false, 0, "check.noCert", nil)
	}
	na := certs[0].NotAfter
	// The English keeps its day-month-year date; other languages write the
	// day in the reader's region format.
	if !na.After(r.now()) {
		c := check("certificate", false, 0, "check.certExpired", i18n.Args{"at": dateArg(na)})
		c.Detail = "The certificate expired on " + na.UTC().Format("2 Jan 2006")
		return c
	}
	c := check("certificate", true, 0, "check.certValid", i18n.Args{"at": dateArg(na)})
	c.Detail = "Certificate valid until " + na.UTC().Format("2 Jan 2006")
	return c
}

func shortNetErr(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 && i+2 < len(s) {
		s = s[i+2:]
	}
	return clip(s, 160)
}

// CheckArgs runs the relay's status checks.
type CheckArgs struct{}

// Kind implements river.JobArgs.
func (CheckArgs) Kind() string { return "smtp_relay.check" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (CheckArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Minute}}
}

// CheckWorker runs CheckArgs while the relay is on.
type CheckWorker struct {
	river.WorkerDefaults[CheckArgs]
	R         *Relay
	PublicURL func(context.Context) string
}

// Work implements river.Worker.
func (w *CheckWorker) Work(ctx context.Context, _ *river.Job[CheckArgs]) error {
	row, err := w.R.Q.RelaySettings(ctx)
	if err != nil || !row.Enabled || isDemo(row.Status) {
		return err
	}
	_, err = w.R.Check(ctx, w.PublicURL(ctx))
	return err
}

// RegisterJobs adds the relay's periodic status check.
func RegisterJobs(reg *jobs.Registry, r *Relay, publicURL func(context.Context) string) {
	river.AddWorker(reg.Workers, &CheckWorker{R: r, PublicURL: publicURL})
	reg.Every(10*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return CheckArgs{}, nil }, false)
}

// Register adds Instance admin › SMTP relay.
func Register(h huma.API, r *Relay, publicURL func(context.Context) string) {
	tags := []string{"Instance admin"}
	api.Get(h, api.Op{ID: "get-smtp-relay", Path: "/admin/smtp-relay", Summary: "The SMTP relay: on or off, hostname, ports, certificate and the last status check",
		Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[SMTPRelaySettings], error) {
			out, err := r.Settings(ctx, publicURL(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-smtp-relay", Path: "/admin/smtp-relay", Summary: "Turn the SMTP relay on or off, or change its hostname, ports or certificate",
		Description: "Audited. Web processes start or stop the listeners within 30 seconds.", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct{ Body SMTPRelayInput }) (*api.Body[SMTPRelaySettings], error) {
			out, err := r.Save(ctx, in.Body, publicURL(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-smtp-relay", Path: "/admin/smtp-relay/check", Summary: "Run the status checks: DNS, both ports and the certificate",
		Description: "The ports are checked from this server by connecting to the hostname's address.", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[SMTPRelaySettings], error) {
			out, err := r.Check(ctx, publicURL(ctx))
			return api.OK(out), err
		})
}
