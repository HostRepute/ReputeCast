package smtprelay_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const ws = "/workspaces/acme-newsletter"

// selfSigned writes a certificate for localhost and returns its files and
// a pool that trusts it.
func selfSigned(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(90 * 24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: true}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "relay.crt"), filepath.Join(dir, "relay.key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0o600); err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return certFile, keyFile, pool
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

type relayEnv struct {
	t        *testing.T
	env      *apptest.Env
	owner    *apptest.Client
	ws       uuid.UUID
	pool     *x509.CertPool
	p587     int
	p465     int
	keyID    string
	secret   string
	sink     *apptest.SMTPSink
	billing  uuid.UUID
	postmark uuid.UUID
	// stopRelay stops the relay's supervisor and its listeners.
	stopRelay func()
}

func (e *relayEnv) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
}

func (e *relayEnv) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func newRelayEnv(t *testing.T) *relayEnv {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	e := &relayEnv{t: t, env: env, owner: env.Client(), sink: apptest.NewSMTPSink(t)}
	e.owner.SignInDemo("linh@acme.io")
	ctx := context.Background()
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&e.ws); err != nil {
		t.Fatal(err)
	}
	var ses uuid.UUID
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = 'postmark'`, e.ws).Scan(&e.postmark)
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = 'ses'`, e.ws).Scan(&ses)
	dom := uuid.Must(uuid.NewV7())
	e.exec(`INSERT INTO sending_domains (id, workspace_id, domain, status, transactional) VALUES ($1, $2, 'notify.acme.io', 'verified', true)`, dom, e.ws)
	e.exec(`INSERT INTO senders (id, workspace_id, domain_id, name, email, reply_to) VALUES ($1, $2, $3, 'Acme Billing', 'billing@notify.acme.io', 'billing@acme.io')`,
		uuid.Must(uuid.NewV7()), e.ws, dom)
	e.exec(`INSERT INTO routing_rules (id, workspace_id, stream, priority, name, splits, fallback_provider_id) VALUES ($1, $2, 'transactional', 100, 'Default', $3, $4)`,
		uuid.Must(uuid.NewV7()), e.ws, fmt.Sprintf(`[{"provider_id":%q,"weight":100}]`, e.postmark), ses)
	e.billing = uuid.Must(uuid.NewV7())
	e.exec(`INSERT INTO transactional_categories (id, workspace_id, name, kind, description) VALUES ($1, $2, 'Billing', 'essential', 'Invoices and receipts')`, e.billing, e.ws)
	var created struct {
		Key    struct{ ID string }
		Secret string
	}
	e.owner.Do("POST", ws+"/api-keys", map[string]any{"name": "Billing service (SMTP)", "permissions": []string{"transactional:send"},
		"transactional": map[string]any{"preset": "smtp", "categories": []string{e.billing.String()}}}).MustStatus(t, 201).JSON(t, &created)
	e.keyID, e.secret = created.Key.ID, created.Secret
	if !strings.HasPrefix(e.secret, "rc_smtp_") {
		t.Fatalf("secret %q", e.secret)
	}
	// Turn the relay on, with a certificate from files.
	certFile, keyFile, pool := selfSigned(t)
	e.pool = pool
	e.p587, e.p465 = freePort(t), freePort(t)
	relay := env.App.SMTPRelay()
	relay.ListenHost, relay.RootCAs, relay.Poll = "127.0.0.1", pool, 100*time.Millisecond
	env.App.TransactionalOutbox().DevSMTP = e.sink.Addr
	rctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { _ = env.App.RunSMTPRelay(rctx); close(done) }()
	var once sync.Once
	e.stopRelay = func() { once.Do(func() { cancel(); <-done }) }
	t.Cleanup(e.stopRelay)
	e.owner.Do("PUT", "/admin/smtp-relay", map[string]any{"enabled": true, "hostname": "localhost", "starttlsPort": e.p587, "tlsPort": e.p465,
		"certSource": "files", "certFile": certFile, "keyFile": keyFile}).MustStatus(t, 200)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(e.p587), 200*time.Millisecond); err == nil {
			_ = c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the relay didn't start")
		}
		time.Sleep(50 * time.Millisecond)
	}
	return e
}

func (e *relayEnv) client(implicit bool) *smtp.Client {
	e.t.Helper()
	tc := &tls.Config{ServerName: "localhost", RootCAs: e.pool, MinVersion: tls.VersionTLS12}
	var c *smtp.Client
	var err error
	if implicit {
		c, err = smtp.DialTLS("127.0.0.1:"+strconv.Itoa(e.p465), tc)
	} else {
		c, err = smtp.DialStartTLS("127.0.0.1:"+strconv.Itoa(e.p587), tc)
	}
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { _ = c.Close() })
	return c
}

func receipt(from, to, extra string) string {
	return "From: Acme Billing <" + from + ">\r\nTo: " + to + "\r\nSubject: Your receipt #4821\r\n" + extra +
		"MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=b1\r\n\r\n" +
		"--b1\r\nContent-Type: multipart/alternative; boundary=b2\r\n\r\n" +
		"--b2\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nThanks for your payment of $49.00.\r\n" +
		"--b2\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<p>Thanks for your payment of <b>$49.00</b>.</p>\r\n--b2--\r\n" +
		"--b1\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"receipt-4821.pdf\"\r\nContent-Transfer-Encoding: base64\r\n\r\nJVBERi0xLjcKJSBBY21l\r\n--b1--\r\n"
}

func TestSMTPRelay(t *testing.T) {
	e := newRelayEnv(t)
	if n := e.count(`SELECT count(*) FROM audit_events WHERE action = 'smtp_relay.enabled'`); n != 1 {
		t.Fatalf("enabled audited %d", n)
	}

	// No AUTH before STARTTLS on 587.
	plain, err := smtp.Dial("127.0.0.1:" + strconv.Itoa(e.p587))
	if err != nil {
		t.Fatal(err)
	}
	if err := plain.Auth(sasl.NewPlainClient("", e.keyID, e.secret)); err == nil {
		t.Fatal("AUTH worked without TLS")
	}
	_ = plain.Close()

	// A wrong password, and an API key, are refused.
	c := e.client(false)
	if err := c.Auth(sasl.NewPlainClient("", e.keyID, "rc_smtp_wrong")); err == nil || !strings.Contains(err.Error(), "535") {
		t.Fatalf("wrong password: %v", err)
	}
	var api struct {
		Key    struct{ ID string }
		Secret string
	}
	e.owner.Do("POST", ws+"/api-keys", map[string]any{"name": "API", "permissions": []string{"transactional:send"}, "transactional": map[string]any{"preset": "send_only"}}).
		MustStatus(t, 201).JSON(t, &api)
	c = e.client(false)
	if err := c.Auth(sasl.NewPlainClient("", api.Key.ID, api.Secret)); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("API key over SMTP: %v", err)
	}

	// A receipt: logged as Raw (SMTP), routed, delivered.
	c = e.client(false)
	if err := c.Auth(sasl.NewPlainClient("", e.keyID, e.secret)); err != nil {
		t.Fatal(err)
	}
	msg := receipt("billing@notify.acme.io", "finance@northwind.example", "X-ReputeCast-Tags: receipt, billing\r\nX-ReputeCast-Category: Billing\r\n")
	if err := c.SendMail("billing@notify.acme.io", []string{"finance@northwind.example"}, strings.NewReader(msg)); err != nil {
		t.Fatalf("send: %v", err)
	}
	var id, source, key, subject, cat, tags, files string
	if err := e.env.App.Pool.QueryRow(context.Background(), `SELECT message_id::text, source, template_key, subject, category_name, array_to_string(tags, ','), attachments::text
		FROM transactional_messages WHERE workspace_id = $1 AND source = 'smtp'`, e.ws).Scan(&id, &source, &key, &subject, &cat, &tags, &files); err != nil {
		t.Fatal(err)
	}
	if key != "" || subject != "Your receipt #4821" || cat != "Billing" || tags != "receipt,billing" || !strings.Contains(files, "receipt-4821.pdf") {
		t.Fatalf("logged: %s %s %s %s %s", key, subject, cat, tags, files)
	}
	if _, err := e.env.App.TransactionalOutbox().Step(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	var status string
	_ = e.env.App.Pool.QueryRow(context.Background(), `SELECT status FROM messages WHERE id = $1`, id).Scan(&status)
	got := e.sink.Messages()
	if status != "delivered" || len(got) != 1 || !strings.Contains(got[0].Data, "receipt-4821.pdf") || !strings.Contains(got[0].Data, "Your receipt #4821") {
		t.Fatalf("delivered: %s %d", status, len(got))
	}

	// The From must be a verified Transactional sender.
	c = e.client(true) // port 465, TLS from the start; AUTH LOGIN
	if err := c.Auth(sasl.NewLoginClient(e.keyID, e.secret)); err != nil {
		t.Fatal(err)
	}
	err = c.SendMail("ceo@acme.io", []string{"x@contoso.example"}, strings.NewReader(receipt("ceo@acme.io", "x@contoso.example", "")))
	if err == nil || !strings.Contains(err.Error(), "550") || !strings.Contains(err.Error(), "isn't a verified Transactional sender") {
		t.Fatalf("unverified From: %v", err)
	}

	// A suppressed recipient alone is refused; with another, the other goes.
	e.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source) VALUES ($1, $2, 'gone@outlook.com', 'hard_bounce', 'webhook')`, uuid.Must(uuid.NewV7()), e.ws)
	c = e.client(false)
	if err := c.Auth(sasl.NewPlainClient("", e.keyID, e.secret)); err != nil {
		t.Fatal(err)
	}
	err = c.SendMail("billing@notify.acme.io", []string{"gone@outlook.com"}, strings.NewReader(receipt("billing@notify.acme.io", "gone@outlook.com", "")))
	if err == nil || !strings.Contains(err.Error(), "suppression list") {
		t.Fatalf("suppressed: %v", err)
	}
	if err := c.SendMail("billing@notify.acme.io", []string{"gone@outlook.com", "ap@northwind.example"},
		strings.NewReader(receipt("billing@notify.acme.io", "ap@northwind.example", ""))); err != nil {
		t.Fatalf("mixed: %v", err)
	}
	if n := e.count(`SELECT count(*) FROM transactional_messages WHERE workspace_id = $1 AND source = 'smtp' AND reject_reason = 'suppressed'`, e.ws); n != 2 {
		t.Fatalf("rejected logged: %d", n)
	}

	// A category the credential can't send.
	e.exec(`INSERT INTO transactional_categories (id, workspace_id, name, kind) VALUES ($1, $2, 'Reminders', 'optional')`, uuid.Must(uuid.NewV7()), e.ws)
	err = c.SendMail("billing@notify.acme.io", []string{"y@contoso.example"}, strings.NewReader(receipt("billing@notify.acme.io", "y@contoso.example", "X-ReputeCast-Category: Reminders\r\n")))
	if err == nil || !strings.Contains(err.Error(), "can only send Billing") {
		t.Fatalf("category: %v", err)
	}

	// The status checks: DNS, both ports and the certificate.
	var st struct {
		Status struct {
			Checks []struct {
				Key    string
				OK     bool
				Detail string
			}
		}
		Certificate *struct{ NotAfter time.Time }
	}
	e.owner.Do("POST", "/admin/smtp-relay/check", nil).MustStatus(t, 200).JSON(t, &st)
	if len(st.Status.Checks) != 4 || st.Certificate == nil {
		t.Fatalf("checks: %+v", st)
	}
	for _, ch := range st.Status.Checks {
		if !ch.OK {
			t.Fatalf("check %s: %s", ch.Key, ch.Detail)
		}
	}
	// Workspace members see the relay on the Setup tab.
	var setup struct {
		Relay struct {
			Enabled bool
			State   string
			Admins  []string
		}
	}
	e.owner.Do("GET", ws+"/transactional/setup", nil).MustStatus(t, 200).JSON(t, &setup)
	if !setup.Relay.Enabled || setup.Relay.State != "ok" || len(setup.Relay.Admins) == 0 {
		t.Fatalf("setup relay: %+v", setup.Relay)
	}

	// Turned off: the listeners stop, the change is audited.
	e.owner.Do("PUT", "/admin/smtp-relay", map[string]any{"enabled": false, "hostname": "localhost", "starttlsPort": e.p587, "tlsPort": e.p465, "certSource": "acme"}).MustStatus(t, 200)
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(e.p587), 200*time.Millisecond)
		if err != nil {
			break
		}
		_ = c.Close()
		if time.Now().After(deadline) {
			t.Fatal("the relay kept listening")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if n := e.count(`SELECT count(*) FROM audit_events WHERE action = 'smtp_relay.disabled'`); n != 1 {
		t.Fatalf("disabled audited %d", n)
	}
	// Viewers and workspace owners who aren't instance admins can't change it.
	aisha := e.env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("PUT", "/admin/smtp-relay", map[string]any{"enabled": true, "hostname": "localhost"}).MustStatus(t, 403)
}

func TestSMTPRelayUnreachable(t *testing.T) {
	e := newRelayEnv(t)
	// The relay is on but no process listens any more: the check tells the
	// instance admins once.
	e.stopRelay()
	var st struct {
		UnreachableSince *time.Time
		Status           struct {
			Checks []struct {
				Key string
				OK  bool
			}
		}
	}
	for i := 0; i < 2; i++ {
		e.owner.Do("POST", "/admin/smtp-relay/check", nil).MustStatus(t, 200).JSON(t, &st)
	}
	if st.UnreachableSince == nil {
		t.Fatalf("not unreachable: %+v", st)
	}
	var linh uuid.UUID
	if err := e.env.App.Pool.QueryRow(context.Background(), `SELECT id FROM users WHERE email = 'linh@acme.io'`).Scan(&linh); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT count(*) FROM notifications WHERE kind = 'smtp_relay_unreachable' AND user_id = $1`, linh); n != 1 {
		t.Fatalf("instance admins told %d times", n)
	}
}
