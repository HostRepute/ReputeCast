package systemmail_test

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/systemmail"
)

const ws = "/workspaces/acme-newsletter"

func TestRender(t *testing.T) {
	team := systemmail.Brand{Name: "Acme ReputeCast", Accent: "#0F766E", Footer: "Sent by Acme ReputeCast · reputecast.acme.io"}
	r, err := systemmail.Render("password_reset", team, "Reset your password", "Hi Aisha,\n\nReset it here: https://reputecast.acme.io/reset/abc.\n<script>x</script>", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<a href="https://reputecast.acme.io/reset/abc" style="color:#0F766E">https://reputecast.acme.io/reset/abc</a>.`,
		`&lt;script&gt;x&lt;/script&gt;`, "Acme ReputeCast</td>", "someone asked to reset the password", "Sent by Acme ReputeCast · reputecast.acme.io",
		`<title>Reset your password</title>`, `>Hi Aisha,</div>`,
	} {
		if !strings.Contains(r.HTML, want) {
			t.Errorf("HTML is missing %q", want)
		}
	}
	if strings.Contains(r.HTML, "<script>") {
		t.Error("text wasn't escaped")
	}
	if !strings.HasPrefix(r.Text, "Hi Aisha,\n\nReset it here: https://reputecast.acme.io/reset/abc.") || !strings.HasSuffix(r.Text, "<script>x</script>\n\n—\nYou get this email because someone asked to reset the password of this account. If it wasn't you, ignore it.\nSent by Acme ReputeCast · reputecast.acme.io\n") {
		t.Errorf("text: %q", r.Text)
	}

	// Subscriber mail carries the publisher's brand; an HTML body gets a text version.
	pub := systemmail.Brand{Name: "Acme Newsletter", Accent: "#E11D48", Footer: "Acme Newsletter · 1 Market St"}
	r, err = systemmail.Render("double_opt_in", pub, "Confirm your subscription", "", `<p>Confirm: <a href="https://acme.io/p/c/1">Yes, subscribe me</a></p>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.HTML, "background:#E11D48") || !strings.Contains(r.Text, "Yes, subscribe me: https://acme.io/p/c/1") || strings.Contains(r.HTML, "#0F766E") {
		t.Errorf("subscriber mail: %s\n%s", r.HTML, r.Text)
	}
	r, _ = systemmail.Render("something_new", team, "S", "Body", "")
	if !strings.Contains(r.Text, "This is an automated message from Acme ReputeCast.") {
		t.Errorf("default reason: %q", r.Text)
	}
	r, _ = systemmail.Render("notification", team, "S", "Body", "")
	if !strings.HasSuffix(r.Text, "Body\n\n—\nSent by Acme ReputeCast · reputecast.acme.io\n") {
		t.Errorf("notification footer: %q", r.Text)
	}
}

// The demo's sample providers can't send outside development.
func TestDemoProvidersUnavailableOutsideDevelopment(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	wsID := workspaceID(t, env)
	if env.App.Mail.SystemAvailable(ctx, &wsID) || env.App.Mail.SystemAvailable(ctx, nil) {
		t.Fatal("demo providers count as available")
	}
	if _, err := env.App.Mail.SendSystem(ctx, mailer.Message{To: "aisha@acme.io", Subject: "Reset", Text: "x", Kind: "password_reset"}); !errors.Is(err, mailer.ErrNoSystemProvider) {
		t.Fatalf("err = %v", err)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE kind = 'password_reset'`).Scan(&n)
	if n != 0 {
		t.Fatal("a message was recorded")
	}
}

// connectSMTP connects an SMTP provider on sink through the connect flow
// and gives it the System stream in role ("primary" or "fallback").
func connectSMTP(t *testing.T, c *apptest.Client, sink *apptest.SMTPSink, role string) string {
	t.Helper()
	host, port, _ := net.SplitHostPort(sink.Addr)
	n, _ := strconv.Atoi(port)
	var st struct{ ID, ProviderID string }
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "smtp"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID
	c.Do("POST", base+"/connection", map[string]any{"smtp": map[string]any{"host": host, "port": n, "tls": "none", "username": "acme", "password": "secret"}}).MustStatus(t, 200).JSON(t, &st)
	if st.ProviderID == "" {
		t.Fatal("not connected")
	}
	c.Do("PUT", base+"/streams", map[string]any{"streams": []map[string]any{{"stream": "system", "enabled": true, "role": role}}}).MustStatus(t, 200)
	c.Do("POST", base+"/finish", map[string]any{"pullSuppressions": false, "startEmpty": false}).MustStatus(t, 200)
	return st.ProviderID
}

func TestSystemMailThroughTheSystemRoute(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	primary, fallback := apptest.NewSMTPSink(t), apptest.NewSMTPSink(t)
	pid := connectSMTP(t, c, primary, "primary")
	fid := connectSMTP(t, c, fallback, "fallback")
	var pname, fname string
	_ = env.App.Pool.QueryRow(ctx, `SELECT name FROM providers WHERE id = $1`, pid).Scan(&pname)
	_ = env.App.Pool.QueryRow(ctx, `SELECT name FROM providers WHERE id = $1`, fid).Scan(&fname)
	wsID := workspaceID(t, env)
	if !env.App.Mail.SystemAvailable(ctx, &wsID) || !env.App.Mail.SystemAvailable(ctx, nil) {
		t.Fatal("system mail unavailable")
	}

	// A password reset (no workspace) goes out through the recipient's workspace.
	got, err := env.App.Mail.SendSystem(ctx, mailer.Message{To: "aisha@acme.io", Subject: "Reset your ReputeCast password",
		Text: "Reset it here: https://reputecast.acme.io/reset/abc", Kind: "password_reset"})
	if err != nil || got != pname {
		t.Fatalf("sent via %q: %v", got, err)
	}
	msgs := primary.Messages()
	if len(msgs) != 1 || msgs[0].From != "no-reply@news.acme.io" || msgs[0].To[0] != "aisha@acme.io" {
		t.Fatalf("primary got %+v", msgs)
	}
	data := msgs[0].Data
	for _, want := range []string{"Subject: Reset your ReputeCast password", "Reply-To: <support@acme.io>", "Auto-Submitted: auto-generated", "X-ReputeCast-Kind: password_reset", "multipart/alternative"} {
		if !strings.Contains(data, want) {
			t.Errorf("message is missing %q", want)
		}
	}
	var status, kind, stream string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status, kind, stream FROM messages WHERE email = 'aisha@acme.io' AND provider_id = $1`, pid).Scan(&status, &kind, &stream); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" || kind != "password_reset" || stream != "system" {
		t.Fatalf("recorded %s/%s/%s", status, kind, stream)
	}

	// Team invitations (W03) say which provider sends, then send through it.
	var opts struct {
		CanEmail       bool
		SystemProvider string
		Roles          []struct{ ID, Key string }
	}
	c.Do("GET", ws+"/invitations/options", nil).MustStatus(t, 200).JSON(t, &opts)
	if !opts.CanEmail || opts.SystemProvider != pname {
		t.Fatalf("invite options: %+v", opts)
	}
	role := ""
	for _, r := range opts.Roles {
		if r.Key == "viewer" {
			role = r.ID
		}
	}
	var inv struct {
		Emailed  bool
		SentVia  string
		Fallback string
	}
	c.Do("POST", ws+"/invitations", map[string]any{"emails": []string{"priya@acme.io"}, "roleId": role}).MustStatus(t, 201).JSON(t, &inv)
	if !inv.Emailed || inv.SentVia != pname || inv.Fallback != "" {
		t.Fatalf("invite: %+v", inv)
	}
	if msgs = primary.Messages(); len(msgs) != 2 || msgs[1].To[0] != "priya@acme.io" || !strings.Contains(msgs[1].Data, "X-ReputeCast-Kind: invite") {
		t.Fatalf("invite mail: %+v", msgs)
	}

	// The primary refuses: the fallback takes the message.
	primary.Reject = true
	got, err = env.App.Mail.SendSystem(ctx, mailer.Message{WorkspaceID: &wsID, To: "marcus@acme.io", Subject: "A comment for you", Text: "x", Kind: "notification"})
	if err != nil || got != fname {
		t.Fatalf("fallback: %q %v", got, err)
	}
	if len(fallback.Messages()) != 1 {
		t.Fatal("the fallback didn't get the message")
	}
	var failed, sent int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'failed' AND provider_id = $1), count(*) FILTER (WHERE status = 'delivered' AND provider_id = $2)
		FROM messages WHERE email = 'marcus@acme.io' AND kind = 'notification'`, pid, fid).Scan(&failed, &sent)
	if failed != 1 || sent != 1 {
		t.Fatalf("failed %d, delivered %d", failed, sent)
	}
	// Both refuse: the caller gets the error.
	fallback.Reject = true
	if _, err := env.App.Mail.SendSystem(ctx, mailer.Message{WorkspaceID: &wsID, To: "marcus@acme.io", Subject: "S", Text: "x", Kind: "notification"}); err == nil {
		t.Fatal("no error when every provider refused")
	}
}

// In development the demo providers deliver to the mail catcher and record
// the message as delivered by that provider.
func TestDemoProviderDeliversToDevSMTP(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	sink := apptest.NewSMTPSink(t)
	n := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q}
	m := &systemmail.Mailer{Q: env.App.Q,
		Deliver: &systemmail.Deliverer{Pool: env.App.Pool, Q: env.App.Q, DevSMTP: sink.Addr,
			ApplyEvent: func(ctx context.Context, tx pgx.Tx, p store.Provider, ev providers.Event) error {
				_, err := n.Apply(ctx, tx, p, nil, "api", time.Now(), ev)
				return err
			}},
		Instance: func(context.Context) (string, string) { return "Acme ReputeCast", "https://reputecast.acme.io" }}
	got, err := m.SendSystem(ctx, mailer.Message{To: "aisha@acme.io", Subject: "Reset your ReputeCast password", Text: "Reset it: https://reputecast.acme.io/reset/abc", Kind: "password_reset"})
	if err != nil || got != "Postmark" {
		t.Fatalf("sent via %q: %v", got, err)
	}
	msgs := sink.Messages()
	if len(msgs) != 1 || !strings.Contains(msgs[0].Data, "Sent by Acme ReputeCast") || !strings.Contains(msgs[0].Data, "From: \"Acme accounts\" <no-reply@news.acme.io>") {
		t.Fatalf("caught: %+v", msgs)
	}
	var status, original string
	if err := env.App.Pool.QueryRow(ctx, `SELECT m.status, e.original_type FROM messages m JOIN delivery_events e ON e.message_id = m.id
		JOIN providers p ON p.id = m.provider_id WHERE m.kind = 'password_reset' AND p.adapter = 'postmark'`).Scan(&status, &original); err != nil {
		t.Fatal(err)
	}
	if status != "delivered" || original != "Delivered to Mailpit (development)" {
		t.Fatalf("recorded %s / %s", status, original)
	}
}

func workspaceID(t *testing.T, env *apptest.Env) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
