package transactional_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// clock is the test's time, shared by the API and the outbox.
type clock struct{ ns atomic.Int64 }

func (c *clock) now() time.Time      { return time.Unix(0, c.ns.Load()) }
func (c *clock) set(t time.Time)     { c.ns.Store(t.UnixNano()) }
func (c *clock) add(d time.Duration) { c.ns.Add(int64(d)) }

// sendEnv is the demo workspace ready to send transactional mail: a
// verified Transactional sender on notify.acme.io, the route Postmark then
// Amazon SES, and password-reset published. The demo providers deliver to
// an SMTP sink standing in for the development mail catcher.
type sendEnv struct {
	t     *testing.T
	env   *apptest.Env
	ctx   context.Context
	ws    uuid.UUID
	clock *clock
	owner *apptest.Client // Linh: every permission
	sofia *apptest.Client // Editor: edit and publish, no send
	sink  *apptest.SMTPSink
	prov  map[string]uuid.UUID
}

func newSendEnv(t *testing.T) *sendEnv {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	s := &sendEnv{t: t, env: env, ctx: context.Background(), clock: &clock{}, prov: map[string]uuid.UUID{}}
	s.clock.set(time.Now().Truncate(time.Second))
	env.App.Transactional().Now = s.clock.now
	env.App.SendingEngine().Now = s.clock.now
	s.sink = apptest.NewSMTPSink(t)
	env.App.TransactionalOutbox().DevSMTP = s.sink.Addr
	s.scan(`SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`, &s.ws)
	rows, err := env.App.Pool.Query(s.ctx, `SELECT adapter, id FROM providers WHERE workspace_id = $1`, s.ws)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a string
		var id uuid.UUID
		_ = rows.Scan(&a, &id)
		s.prov[a] = id
	}
	rows.Close()
	dom := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO sending_domains (id, workspace_id, domain, status, transactional) VALUES ($1, $2, 'notify.acme.io', 'verified', true)`, dom, s.ws)
	s.exec(`INSERT INTO senders (id, workspace_id, domain_id, name, email, reply_to) VALUES ($1, $2, $3, 'Acme Cloud', 'no-reply@notify.acme.io', 'support@acme.io')`,
		uuid.Must(uuid.NewV7()), s.ws, dom)
	s.owner = env.Client()
	s.owner.SignInDemo("linh@acme.io")
	s.sofia = env.Client()
	s.sofia.SignInDemo("sofia@acme.io")
	s.route(s.prov["postmark"], s.prov["ses"])
	s.publish("password-reset", "Password reset", "password-reset")
	return s
}

func (s *sendEnv) exec(sql string, args ...any) {
	s.t.Helper()
	if _, err := s.env.App.Pool.Exec(s.ctx, sql, args...); err != nil {
		s.t.Fatalf("%s: %v", sql, err)
	}
}

func (s *sendEnv) scan(sql string, dest ...any) {
	s.t.Helper()
	if err := s.env.App.Pool.QueryRow(s.ctx, sql).Scan(dest...); err != nil {
		s.t.Fatalf("%s: %v", sql, err)
	}
}

func (s *sendEnv) one(sql string, dest any, args ...any) {
	s.t.Helper()
	if err := s.env.App.Pool.QueryRow(s.ctx, sql, args...).Scan(dest); err != nil {
		s.t.Fatalf("%s: %v", sql, err)
	}
}

func (s *sendEnv) count(sql string, args ...any) int {
	s.t.Helper()
	var n int
	s.one(sql, &n, args...)
	return n
}

// route sets the Transactional route: primary, then fallback (automatic).
func (s *sendEnv) route(primary, fallback uuid.UUID) {
	s.t.Helper()
	var c struct {
		Streams []struct {
			Stream string
			Splits []struct {
				ProviderID string
				Weight     int
			}
			Fallback   string
			Failover   string
			Conditions []struct {
				Field      string
				Values     []string
				ProviderID string
				Note       string
			}
		}
	}
	s.owner.Do("GET", "/workspaces/acme-newsletter/routing", nil).MustStatus(s.t, 200).JSON(s.t, &c)
	streams := []map[string]any{}
	for _, st := range c.Streams {
		in := map[string]any{"stream": st.Stream, "fallback": st.Fallback, "failover": st.Failover}
		splits, conds := []map[string]any{}, []map[string]any{}
		for _, sp := range st.Splits {
			splits = append(splits, map[string]any{"providerId": sp.ProviderID, "weight": sp.Weight})
		}
		for _, cd := range st.Conditions {
			conds = append(conds, map[string]any{"field": cd.Field, "values": cd.Values, "providerId": cd.ProviderID, "note": cd.Note})
		}
		if st.Stream == "transactional" {
			splits = []map[string]any{{"providerId": primary.String(), "weight": 100}}
			in["fallback"], in["failover"] = fallback.String(), "automatic"
		}
		in["splits"], in["conditions"] = splits, conds
		streams = append(streams, in)
	}
	s.owner.Do("PUT", "/workspaces/acme-newsletter/routing", map[string]any{"streams": streams, "acknowledgeBulkTerms": true}).MustStatus(s.t, 200)
}

// publish creates a template from a starter and publishes it.
func (s *sendEnv) publish(key, name, starter string) {
	s.t.Helper()
	var tpl ttemplate
	s.sofia.Do("POST", base+"/templates", map[string]any{"name": name, "key": key, "starter": starter}).MustStatus(s.t, 201).JSON(s.t, &tpl)
	s.sofia.Do("POST", base+"/templates/"+key+"/publish", map[string]any{"baseRevision": tpl.DraftRevision}).MustStatus(s.t, 200)
}

// key creates an API key; transactional is its send-only block.
func (s *sendEnv) key(name string, transactional map[string]any) (string, string) {
	s.t.Helper()
	var created struct {
		Key struct {
			ID            string
			Transactional *struct{ Preset, Mode string }
		}
		Secret string
	}
	s.owner.Do("POST", "/workspaces/acme-newsletter/api-keys", map[string]any{"name": name, "permissions": []string{"transactional:send"}, "transactional": transactional}).
		MustStatus(s.t, 201).JSON(s.t, &created)
	if created.Key.Transactional == nil {
		s.t.Fatalf("key %s isn't send-only", name)
	}
	return created.Key.ID, created.Secret
}

// call sends a request with a key and an idempotency key.
func (s *sendEnv) call(secret, path, idem string, body any) *apptest.Response {
	s.t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		s.t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, s.env.Server.URL+"/api/v1"+base+path, bytes.NewReader(b))
	if err != nil {
		s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	return &apptest.Response{Status: resp.StatusCode, Header: resp.Header, Body: raw}
}

type sendResult struct {
	ID     string
	Status string
	SendAt *time.Time `json:"send_at"`
	Test   bool
	Detail string
}

func resetBody(to string) map[string]any {
	return map[string]any{"template": "password-reset", "to": map[string]any{"email": to, "name": "Sam Rivera"},
		"data": map[string]any{"first_name": "Sam", "reset_url": "https://app.acme.io/reset/tok_9f8e7d6c5b4a", "expires_in": "30 minutes"},
		"tags": []string{"auth"}, "metadata": map[string]string{"user_id": "u_1842"}}
}

// step runs the outbox at the test's time.
func (s *sendEnv) step() int {
	s.t.Helper()
	n, err := s.env.App.TransactionalOutbox().Step(s.ctx, s.clock.now())
	if err != nil {
		s.t.Fatalf("outbox: %v", err)
	}
	return n
}

func (s *sendEnv) status(id string) string {
	s.t.Helper()
	var st string
	s.one(`SELECT status FROM messages WHERE id = $1`, &st, id)
	return st
}

// webhook subscribes an endpoint to every message.* event.
func (s *sendEnv) webhook() {
	s.exec(`INSERT INTO webhooks (id, workspace_id, name, url, events, secret_enc) VALUES ($1, $2, 'App', 'https://app.acme.io/hooks/reputecast', '{message.*}', '\x00')`,
		uuid.Must(uuid.NewV7()), s.ws)
}

// events lists the message.* webhook events queued for a message.
func (s *sendEnv) events(id string) []string {
	s.t.Helper()
	rows, err := s.env.App.Pool.Query(s.ctx, `SELECT event FROM webhook_deliveries WHERE payload->'data'->>'message_id' = $1 ORDER BY created_at, event`, id)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var e string
		_ = rows.Scan(&e)
		out = append(out, e)
	}
	return out
}

// unfold undoes quoted-printable soft line breaks and folded headers so a
// test can look for a value in a received message.
func unfold(data string) string {
	return strings.NewReplacer("=\r\n", "", "=3D", "=", "\r\n ", "", "\r\n\t", "").Replace(data)
}

func TestSendAndDeliver(t *testing.T) {
	s := newSendEnv(t)
	s.webhook()
	keyID, secret := s.key("Production", map[string]any{"preset": "send_only"})
	if !strings.HasPrefix(secret, "rc_live_") {
		t.Fatalf("secret %q", secret)
	}

	var res sendResult
	s.call(secret, "/messages", "reset-u_1842-1", resetBody("sam.rivera@outlook.com")).MustStatus(t, 202).JSON(t, &res)
	if res.Status != "queued" || res.ID == "" || res.Test {
		t.Fatalf("send: %+v", res)
	}
	// The log keeps the call without its sensitive value; the sealed
	// payload waits for the outbox.
	var data, route string
	var keyName, source string
	s.one(`SELECT data::text FROM transactional_messages WHERE message_id = $1`, &data, res.ID)
	s.one(`SELECT route->>'provider_name' FROM transactional_messages WHERE message_id = $1`, &route, res.ID)
	s.one(`SELECT key_name FROM transactional_messages WHERE message_id = $1`, &keyName, res.ID)
	s.one(`SELECT source FROM transactional_messages WHERE message_id = $1`, &source, res.ID)
	if strings.Contains(data, "tok_9f8e7d6c5b4a") || !strings.Contains(data, "••••••") || route != "Postmark" || keyName != "Production" || source != "api" {
		t.Fatalf("log: data %s route %s key %s source %s", data, route, keyName, source)
	}
	var html string
	s.one(`SELECT html FROM transactional_message_content WHERE message_id = $1`, &html, res.ID)
	if strings.Contains(html, "tok_9f8e7d6c5b4a") || !strings.Contains(html, "••••••") {
		t.Fatal("stored content carries the sensitive value")
	}
	if n := s.count(`SELECT count(*) FROM transactional_outbox WHERE message_id = $1`, res.ID); n != 1 {
		t.Fatalf("outbox %d", n)
	}
	if n := s.count(`SELECT count(*) FROM river_job WHERE kind = 'sending.transactional'`); n != 1 {
		t.Fatalf("outbox wake jobs %d", n)
	}

	// A retry with the same key gets the same answer and sends nothing.
	var again sendResult
	s.call(secret, "/messages", "reset-u_1842-1", resetBody("sam.rivera@outlook.com")).MustStatus(t, 202).JSON(t, &again)
	if again.ID != res.ID {
		t.Fatalf("retry made %s, first %s", again.ID, res.ID)
	}
	// The same key with another body is refused.
	p := s.call(secret, "/messages", "reset-u_1842-1", resetBody("other@outlook.com")).MustStatus(t, 409).Problem(t)
	if p["code"] != "idempotency_key_reused" {
		t.Fatalf("reused: %v", p)
	}
	if n := s.count(`SELECT count(*) FROM transactional_messages WHERE key_id = $1`, keyID); n != 1 {
		t.Fatalf("messages %d", n)
	}

	// The outbox hands it to Postmark (the sink here) and the mail catcher's
	// answer is its delivery.
	if n := s.step(); n != 1 {
		t.Fatalf("sent %d", n)
	}
	if st := s.status(res.ID); st != "delivered" {
		t.Fatalf("status %s", st)
	}
	if n := s.count(`SELECT count(*) FROM transactional_outbox WHERE message_id = $1`, res.ID); n != 0 {
		t.Fatal("the sealed payload outlived the hand-off")
	}
	msgs := s.sink.Messages()
	if len(msgs) != 1 || msgs[0].From != "no-reply@notify.acme.io" || len(msgs[0].To) != 1 || msgs[0].To[0] != "sam.rivera@outlook.com" {
		t.Fatalf("sink: %+v", msgs)
	}
	body := unfold(msgs[0].Data)
	for _, want := range []string{"tok_9f8e7d6c5b4a", "Subject: Reset your", "Feedback-ID: password-reset:acme-newsletter:transactional:reputecast", "support@acme.io"} {
		if !strings.Contains(body, want) {
			t.Fatalf("delivered message lacks %q:\n%s", want, body[:min(len(body), 2000)])
		}
	}
	if strings.Contains(body, "List-Unsubscribe:") {
		t.Fatal("an Essential email carries List-Unsubscribe")
	}
	if ev := s.events(res.ID); strings.Join(ev, ",") != "message.delivered,message.sent" && strings.Join(ev, ",") != "message.sent,message.delivered" {
		t.Fatalf("webhook events %v", ev)
	}
	var payload map[string]any
	var raw []byte
	s.one(`SELECT payload FROM webhook_deliveries WHERE payload->'data'->>'message_id' = $1 AND event = 'message.sent'`, &raw, res.ID)
	_ = json.Unmarshal(raw, &payload)
	d, _ := payload["data"].(map[string]any)
	if d["template"] != "password-reset" || fmt.Sprint(d["tags"]) != "[auth]" || fmt.Sprint(d["metadata"]) != "map[user_id:u_1842]" {
		t.Fatalf("message.sent data %v", d)
	}
	// API sends aren't audited.
	if n := s.count(`SELECT count(*) FROM audit_events WHERE action LIKE 'transactional.message%'`); n != 0 {
		t.Fatalf("api sends audited: %d", n)
	}
}

func TestSendErrors(t *testing.T) {
	s := newSendEnv(t)
	s.webhook()
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	code := func(r *apptest.Response, status int, want string) map[string]any {
		t.Helper()
		p := r.MustStatus(t, status).Problem(t)
		if p["code"] != want {
			t.Fatalf("code %v, want %s: %v", p["code"], want, p)
		}
		return p
	}

	b := resetBody("sam.rivera@outlook.com")
	b["template"] = "no-such-template"
	code(s.call(secret, "/messages", "", b), 422, "template_not_found")

	// A draft with no published version.
	s.sofia.Do("POST", base+"/templates", map[string]any{"name": "Sign-in link", "key": "magic-link", "starter": "magic-link"}).MustStatus(t, 201)
	b = resetBody("sam.rivera@outlook.com")
	b["template"] = "magic-link"
	code(s.call(secret, "/messages", "", b), 422, "template_not_published")

	b = resetBody("sam.rivera@outlook.com")
	delete(b["data"].(map[string]any), "reset_url")
	p := code(s.call(secret, "/messages", "", b), 422, "missing_variables")
	if fmt.Sprint(p["missing"]) != "[reset_url]" || p["title"] != "Missing variables" {
		t.Fatalf("missing: %v", p)
	}

	code(s.call(secret, "/messages", "", resetBody("sam.rivera@")), 422, "invalid_recipient")
	b = resetBody("sam.rivera@outlook.com")
	b["cc"] = []map[string]any{{"email": "not an address"}}
	code(s.call(secret, "/messages", "", b), 422, "invalid_recipient")

	b = resetBody("sam.rivera@outlook.com")
	b["from"] = "ceo@acme.io"
	p = code(s.call(secret, "/messages", "", b), 422, "sender_not_verified")
	if !strings.Contains(p["detail"].(string), "ceo@acme.io") {
		t.Fatalf("sender: %v", p)
	}

	// send_at is at most 72 hours ahead.
	b = resetBody("sam.rivera@outlook.com")
	b["send_at"] = s.clock.now().Add(73 * time.Hour).Format(time.RFC3339)
	s.call(secret, "/messages", "", b).MustStatus(t, 422)

	// Suppressed: a hard bounce stops transactional mail too, and the call
	// says why and since when; an unsubscribe doesn't.
	since := s.clock.now().Add(-72 * time.Hour)
	s.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source, original_code, detail, created_at) VALUES ($1, $2, 'gone@outlook.com', 'hard_bounce', 'webhook', '550 5.1.1', 'Mailbox does not exist', $3)`,
		uuid.Must(uuid.NewV7()), s.ws, since)
	s.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source, created_at) VALUES ($1, $2, 'left@outlook.com', 'unsubscribe', 'unsubscribe_link', $3)`,
		uuid.Must(uuid.NewV7()), s.ws, since)
	p = code(s.call(secret, "/messages", "sup-1", resetBody("gone@outlook.com")), 422, "recipient_suppressed")
	meta, _ := p["meta"].(map[string]any)
	// The date is written in the workspace's timezone, not the test's.
	var tz string
	s.scan(`SELECT timezone FROM workspaces WHERE slug = 'acme-newsletter'`, &tz)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatal(err)
	}
	if meta["reason"] != "hard_bounce" || meta["messageId"] == nil || !strings.Contains(p["detail"].(string), "hard bounce on "+since.In(loc).Format("2 Jan 2006")) {
		t.Fatalf("suppressed: %v", p)
	}
	// The refusal is in the log, and a retry answers the same.
	if st := s.status(meta["messageId"].(string)); st != "rejected" {
		t.Fatalf("suppressed status %s", st)
	}
	if ev := s.events(meta["messageId"].(string)); len(ev) != 1 || ev[0] != "message.rejected" {
		t.Fatalf("rejected events %v", ev)
	}
	p2 := code(s.call(secret, "/messages", "sup-1", resetBody("gone@outlook.com")), 422, "recipient_suppressed")
	if p2["meta"].(map[string]any)["messageId"] != meta["messageId"] {
		t.Fatal("retry of a refused call made another message")
	}
	var ok sendResult
	s.call(secret, "/messages", "", resetBody("left@outlook.com")).MustStatus(t, 202).JSON(t, &ok)
	if ok.Status != "queued" {
		t.Fatalf("unsubscribed: %+v", ok)
	}

	// Suppressed copies are left out, the message still goes.
	b = resetBody("sam.rivera@outlook.com")
	b["bcc"] = []map[string]any{{"email": "gone@outlook.com"}}
	s.call(secret, "/messages", "", b).MustStatus(t, 202).JSON(t, &ok)
	var dropped string
	s.one(`SELECT reject_detail->>'dropped' FROM transactional_messages WHERE message_id = $1`, &dropped, ok.ID)
	if !strings.Contains(dropped, "gone@outlook.com") {
		t.Fatalf("dropped %q", dropped)
	}

	// No route: nothing is queued, and the answer says where to set one.
	s.exec(`DELETE FROM routing_rules WHERE workspace_id = $1 AND stream = 'transactional'`, s.ws)
	p = code(s.call(secret, "/messages", "", resetBody("sam.rivera@outlook.com")), 503, "no_healthy_route")
	if !strings.Contains(p["detail"].(string), "Routing") {
		t.Fatalf("no route: %v", p)
	}
	s.route(s.prov["postmark"], s.prov["ses"])
	// Both providers down: 503 with Retry-After.
	s.exec(`UPDATE providers SET status = 'down' WHERE id = ANY($1)`, []uuid.UUID{s.prov["postmark"], s.prov["ses"]})
	r := s.call(secret, "/messages", "", resetBody("sam.rivera@outlook.com"))
	p = code(r, 503, "no_healthy_route")
	if r.Header.Get("Retry-After") == "" || !strings.Contains(p["detail"].(string), "Postmark and Amazon SES are down") {
		t.Fatalf("down: %v %v", r.Header, p)
	}
	// Postmark down: the message goes straight to the fallback.
	s.exec(`UPDATE providers SET status = 'healthy' WHERE id = $1`, s.prov["ses"])
	var fo sendResult
	s.call(secret, "/messages", "", resetBody("sam.rivera@outlook.com")).MustStatus(t, 202).JSON(t, &fo)
	var prov string
	var failedOver bool
	s.one(`SELECT p.name FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.id = $1`, &prov, fo.ID)
	s.one(`SELECT coalesce((route->>'failed_over')::boolean, false) FROM transactional_messages WHERE message_id = $1`, &failedOver, fo.ID)
	if prov != "Amazon SES" || !failedOver {
		t.Fatalf("failover at accept: %s %v", prov, failedOver)
	}
	s.exec(`UPDATE providers SET status = 'healthy' WHERE id = $1`, s.prov["postmark"])

	// Rejected calls are kept for the guide and the publish check.
	if n := s.count(`SELECT count(*) FROM transactional_rejected_calls WHERE workspace_id = $1`, s.ws); n < 7 {
		t.Fatalf("rejected calls %d", n)
	}
}

func TestSendAttachments(t *testing.T) {
	s := newSendEnv(t)
	_, secret := s.key("Billing", map[string]any{"preset": "send_only"})
	pdf := []byte("%PDF-1.7\n% Acme Cloud invoice INV-2026-0418\n")
	b := resetBody("ops@contoso.example")
	b["cc"] = []map[string]any{{"email": "billing@contoso.example", "name": "Contoso Billing"}}
	b["attachments"] = []map[string]any{{"filename": "INV-2026-0418.pdf", "content": base64.StdEncoding.EncodeToString(pdf)}}
	var res sendResult
	s.call(secret, "/messages", "", b).MustStatus(t, 202).JSON(t, &res)
	var meta string
	s.one(`SELECT attachments::text FROM transactional_messages WHERE message_id = $1`, &meta, res.ID)
	if !strings.Contains(meta, "INV-2026-0418.pdf") || !strings.Contains(meta, "application/pdf") || strings.Contains(meta, "PDF-1.7") {
		t.Fatalf("attachment meta %s", meta)
	}
	s.step()
	msgs := s.sink.Messages()
	if len(msgs) != 1 || len(msgs[0].To) != 2 {
		t.Fatalf("sink: %+v", msgs)
	}
	if !strings.Contains(msgs[0].Data, "INV-2026-0418.pdf") || !strings.Contains(msgs[0].Data, "Cc: \"Contoso Billing\" <billing@contoso.example>") {
		t.Fatalf("delivered: %s", msgs[0].Data[:min(2000, len(msgs[0].Data))])
	}

	// Too large for Postmark's 10 MB: the answer names the limit.
	big := make([]byte, 8<<20)
	b["attachments"] = []map[string]any{{"filename": "export.csv", "content": base64.StdEncoding.EncodeToString(big)}}
	p := s.call(secret, "/messages", "", b).MustStatus(t, 422).Problem(t)
	if p["code"] != "attachments_too_large" || !strings.Contains(p["detail"].(string), "10 MB (Postmark)") {
		t.Fatalf("too large: %v", p)
	}
	// Not base64.
	b["attachments"] = []map[string]any{{"filename": "a.pdf", "content": "%%%"}}
	s.call(secret, "/messages", "", b).MustStatus(t, 422)
}

func TestSendScheduled(t *testing.T) {
	s := newSendEnv(t)
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	at := s.clock.now().Add(2 * time.Hour)
	b := resetBody("sam.rivera@outlook.com")
	b["send_at"] = at.Format(time.RFC3339)
	var res sendResult
	s.call(secret, "/messages", "", b).MustStatus(t, 202).JSON(t, &res)
	if res.Status != "scheduled" || res.SendAt == nil || !res.SendAt.Equal(at) {
		t.Fatalf("scheduled: %+v", res)
	}
	if n := s.step(); n != 0 {
		t.Fatalf("sent early: %d", n)
	}
	s.clock.add(2*time.Hour + time.Second)
	if n := s.step(); n != 1 || s.status(res.ID) != "delivered" {
		t.Fatalf("scheduled send: %d %s", n, s.status(res.ID))
	}
}

func TestSendBatch(t *testing.T) {
	s := newSendEnv(t)
	keyID, secret := s.key("Production", map[string]any{"preset": "send_only"})
	s.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source) VALUES ($1, $2, 'gone@outlook.com', 'complaint', 'webhook')`, uuid.Must(uuid.NewV7()), s.ws)
	batch := map[string]any{"messages": []any{resetBody("a@contoso.example"), resetBody("not-an-address"), resetBody("gone@outlook.com"), resetBody("b@contoso.example")}}
	var out struct {
		Results []struct {
			Index  int
			ID     string
			Status string
			Error  map[string]any
		}
		Accepted, Failed int
	}
	s.call(secret, "/messages/batch", "batch-1", batch).MustStatus(t, 200).JSON(t, &out)
	if out.Accepted != 2 || out.Failed != 2 || len(out.Results) != 4 {
		t.Fatalf("batch: %+v", out)
	}
	if out.Results[0].Status != "queued" || out.Results[1].Error["code"] != "invalid_recipient" || out.Results[2].Error["code"] != "recipient_suppressed" || out.Results[3].Status != "queued" {
		t.Fatalf("results: %+v", out.Results)
	}
	// The retry answers the same and sends nothing more.
	var again struct{ Results []struct{ ID string } }
	s.call(secret, "/messages/batch", "batch-1", batch).MustStatus(t, 200).JSON(t, &again)
	if again.Results[0].ID != out.Results[0].ID {
		t.Fatal("batch retry sent again")
	}
	if n := s.count(`SELECT count(*) FROM transactional_messages WHERE key_id = $1`, keyID); n != 3 {
		t.Fatalf("messages %d", n)
	}
	if n := s.step(); n != 2 {
		t.Fatalf("sent %d", n)
	}
	// More than 100 is refused whole.
	many := []any{}
	for i := range 101 {
		many = append(many, resetBody(fmt.Sprintf("u%d@contoso.example", i)))
	}
	s.call(secret, "/messages/batch", "", map[string]any{"messages": many}).MustStatus(t, 422)
}

func TestSendKeysAndPeople(t *testing.T) {
	s := newSendEnv(t)
	s.publish("trial-ending", "Trial ends in 3 days", "trial-ending")

	// A send-only key holds transactional:send and nothing else.
	s.owner.Do("POST", "/workspaces/acme-newsletter/api-keys", map[string]any{"name": "Too much", "permissions": []string{"transactional:send", "campaigns:view"},
		"transactional": map[string]any{"preset": "send_only"}}).MustStatus(t, 422)
	// Live keys have no test addresses.
	s.owner.Do("POST", "/workspaces/acme-newsletter/api-keys", map[string]any{"name": "Odd", "permissions": []string{"transactional:send"},
		"transactional": map[string]any{"preset": "send_only", "testAddresses": []string{"qa@acme.io"}}}).MustStatus(t, 422)

	// Limited to one template.
	_, onlyTrial := s.key("Trials", map[string]any{"preset": "send_only", "templates": []string{"trial-ending"}})
	p := s.call(onlyTrial, "/messages", "", resetBody("sam.rivera@outlook.com")).MustStatus(t, 403).Problem(t)
	if p["code"] != "template_not_allowed" || !strings.Contains(p["detail"].(string), "trial-ending") {
		t.Fatalf("not allowed: %v", p)
	}

	// Test mode delivers to members and allowed patterns only.
	_, test := s.key("Staging", map[string]any{"preset": "send_only", "mode": "test", "testAddresses": []string{"qa+*@acme.io"}})
	if !strings.HasPrefix(test, "rc_test_") {
		t.Fatalf("test key %q", test)
	}
	var r sendResult
	s.call(test, "/messages", "", resetBody("qa+signup@acme.io")).MustStatus(t, 202).JSON(t, &r)
	if r.Status != "queued" || !r.Test {
		t.Fatalf("allowed pattern: %+v", r)
	}
	s.call(test, "/messages", "", resetBody("sofia@acme.io")).MustStatus(t, 202).JSON(t, &r)
	if r.Status != "queued" || !r.Test {
		t.Fatalf("member: %+v", r)
	}
	s.call(test, "/messages", "", resetBody("customer@gmail.com")).MustStatus(t, 202).JSON(t, &r)
	if r.Status != "not_delivered" || !r.Test || !strings.Contains(r.Detail, "customer@gmail.com") {
		t.Fatalf("not allowed in test mode: %+v", r)
	}
	if st := s.status(r.ID); st != "cancelled" {
		t.Fatalf("test mode status %s", st)
	}
	if n := s.step(); n != 2 {
		t.Fatalf("test mode sent %d", n)
	}

	// An address that turned an Optional category off.
	var cat string
	s.one(`SELECT c.id::text FROM transactional_categories c JOIN transactional_templates t ON t.category_id = c.id WHERE t.key = 'trial-ending'`, &cat)
	s.exec(`INSERT INTO transactional_opt_outs (workspace_id, email, category_id, opted_out_at, source) VALUES ($1, 'sam.rivera@outlook.com', $2, $3, 'preference_center')`,
		s.ws, cat, s.clock.now().Add(-24*time.Hour))
	trial := map[string]any{"template": "trial-ending", "to": map[string]any{"email": "sam.rivera@outlook.com"},
		"data": map[string]any{"first_name": "Sam", "days_left": 3, "upgrade_url": "https://app.acme.io/billing"}}
	_, live := s.key("Production", map[string]any{"preset": "send_only"})
	p = s.call(live, "/messages", "", trial).MustStatus(t, 422).Problem(t)
	if p["code"] != "recipient_suppressed" || p["meta"].(map[string]any)["reason"] != "opted_out" || !strings.Contains(p["detail"].(string), "turned off") {
		t.Fatalf("opted out: %v", p)
	}
	// Optional emails carry the one-click unsubscribe headers.
	trial["to"] = map[string]any{"email": "ops@contoso.example"}
	s.call(live, "/messages", "", trial).MustStatus(t, 202).JSON(t, &r)
	before := len(s.sink.Messages())
	s.step()
	msgs := s.sink.Messages()
	if len(msgs) != before+1 || !strings.Contains(msgs[len(msgs)-1].Data, "List-Unsubscribe-Post: List-Unsubscribe=One-Click") {
		last := ""
		if len(msgs) > 0 {
			last = msgs[len(msgs)-1].Data
		}
		t.Fatalf("optional headers missing: %d messages (before %d), status %s:\n%s", len(msgs), before, s.status(r.ID), last[:min(len(last), 1500)])
	}

	// People: Viewers and Editors can't send; MCP agents never send.
	viewer := s.env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("POST", base+"/messages", resetBody("sam.rivera@outlook.com")).MustStatus(t, 403)
	s.sofia.Do("POST", base+"/messages", resetBody("sam.rivera@outlook.com")).MustStatus(t, 403)
	var agent struct{ Secret string }
	s.owner.Do("POST", "/workspaces/acme-newsletter/mcp-agents", map[string]any{"name": "Claude", "permissions": []string{"transactional:send"}}).MustStatus(t, 201).JSON(t, &agent)
	p = s.call(agent.Secret, "/messages", "", resetBody("sam.rivera@outlook.com")).MustStatus(t, 403).Problem(t)
	if p["code"] != "agents_never_send" {
		t.Fatalf("agent: %v", p)
	}
	// The Owner sends from a session (the first-send guide does).
	s.owner.Do("POST", base+"/messages", resetBody("linh@acme.io")).MustStatus(t, 202)
}

func TestSendRateLimit(t *testing.T) {
	s := newSendEnv(t)
	_, secret := s.key("Slow", map[string]any{"preset": "send_only", "ratePerSecond": 2})
	s.call(secret, "/messages", "", resetBody("a@contoso.example")).MustStatus(t, 202)
	s.call(secret, "/messages", "", resetBody("b@contoso.example")).MustStatus(t, 202)
	r := s.call(secret, "/messages", "", resetBody("c@contoso.example"))
	p := r.MustStatus(t, 429).Problem(t)
	if p["code"] != "rate_limited" || r.Header.Get("Retry-After") != "1" || !strings.Contains(p["detail"].(string), "2 messages per second") {
		t.Fatalf("rate limited: %v %v", r.Header, p)
	}
	s.clock.add(time.Second)
	s.call(secret, "/messages", "", resetBody("c@contoso.example")).MustStatus(t, 202)
}

// TestOutboxFailover: temporary errors are retried for ten minutes, then
// the message goes to the route's fallback; when every provider gave up it
// fails, with message.failed.
func TestOutboxFailover(t *testing.T) {
	s := newSendEnv(t)
	s.webhook()
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	// Nothing listens: every attempt fails with a connection error.
	ln, err := (&net.ListenConfig{}).Listen(s.ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().String()
	_ = ln.Close()
	s.env.App.TransactionalOutbox().DevSMTP = closed

	var res sendResult
	s.call(secret, "/messages", "", resetBody("sam.rivera@outlook.com")).MustStatus(t, 202).JSON(t, &res)
	s.step()
	var next *time.Time
	s.one(`SELECT next_attempt_at FROM messages WHERE id = $1`, &next, res.ID)
	if s.status(res.ID) != "queued" || next == nil || !next.After(s.clock.now()) {
		t.Fatalf("retry: %s %v", s.status(res.ID), next)
	}
	// After ten minutes Postmark gives up; Amazon SES takes over.
	s.clock.add(11 * time.Minute)
	s.step()
	var prov string
	s.one(`SELECT p.adapter FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.id = $1`, &prov, res.ID)
	var from *uuid.UUID
	s.one(`SELECT failed_over_from FROM transactional_messages WHERE message_id = $1`, &from, res.ID)
	if prov != "ses" || from == nil || *from != s.prov["postmark"] || s.status(res.ID) != "queued" {
		t.Fatalf("failover: %s from %v status %s", prov, from, s.status(res.ID))
	}
	s.step()
	s.clock.add(11 * time.Minute)
	s.step()
	if st := s.status(res.ID); st != "failed" {
		t.Fatalf("status %s", st)
	}
	if n := s.count(`SELECT count(*) FROM transactional_outbox WHERE message_id = $1`, res.ID); n != 0 {
		t.Fatal("a failed message kept its payload")
	}
	var attempts int
	s.one(`SELECT jsonb_array_length(attempts) FROM transactional_messages WHERE message_id = $1`, &attempts, res.ID)
	if attempts < 4 {
		t.Fatalf("attempts %d", attempts)
	}
	if ev := s.events(res.ID); len(ev) != 1 || ev[0] != "message.failed" {
		t.Fatalf("events %v", ev)
	}

	// A provider marked down sends its queue to the fallback at once. (The
	// failed attempts above marked both providers down.)
	s.env.App.TransactionalOutbox().DevSMTP = s.sink.Addr
	s.exec(`UPDATE providers SET status = 'healthy' WHERE id = ANY($1)`, []uuid.UUID{s.prov["postmark"], s.prov["ses"]})
	s.call(secret, "/messages", "", resetBody("ops@contoso.example")).MustStatus(t, 202).JSON(t, &res)
	s.exec(`UPDATE providers SET status = 'down' WHERE id = $1`, s.prov["postmark"])
	s.step()
	s.step()
	s.one(`SELECT p.adapter FROM messages m JOIN providers p ON p.id = m.provider_id WHERE m.id = $1`, &prov, res.ID)
	if prov != "ses" || s.status(res.ID) != "delivered" {
		t.Fatalf("down failover: %s %s", prov, s.status(res.ID))
	}
}
