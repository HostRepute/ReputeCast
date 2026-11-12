package inbound_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/inbound"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/providers/smtp/bouncebox"
	"github.com/hostrepute/reputecast/internal/store"
)

const wsPath = "/workspaces/acme-newsletter"

// The simulator is a test adapter: the app's build doesn't register it.
func init() {
	if _, ok := providers.Lookup(simulator.Adapter); !ok {
		providers.Register(simulator.Descriptor())
	}
}

// fixture is the demo workspace receiving through a simulator route on
// in.news.acme.io (a subdomain of its verified news.acme.io).
type fixture struct {
	t      *testing.T
	env    *apptest.Env
	c      *apptest.Client
	ctx    context.Context
	ws     store.Workspace
	route  store.InboundRoute
	secret string
	token  string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	f := &fixture{t: t, env: env, c: env.Client(), ctx: context.Background()}
	f.c.SignInDemo("linh@acme.io")
	ws, err := env.App.Q.GetWorkspaceBySlug(f.ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	f.ws = ws
	prov := uuid.Must(uuid.NewV7())
	f.exec(`INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1, $2, 'simulator', 'Simulator', 'test', '{}', NULL, $3, 'healthy', '', 'shared', '{}', '{}', '{}', '{}', '{}', '{}', now())`,
		prov, ws.ID, "sim-"+prov.String())
	var view inbound.InboundRouteView
	f.c.Do("PUT", wsPath+"/inbound", map[string]any{"kind": "provider", "providerId": prov.String(), "domain": "in.news.acme.io"}).MustStatus(t, 200).JSON(t, &view)
	if view.Status != "pending" || view.Domain != "in.news.acme.io" || view.Addresses.DMARC != "dmarc@in.news.acme.io" || view.Provider == nil {
		t.Fatalf("route: %+v", view)
	}
	f.route, err = env.App.Q.InboundRouteGet(f.ctx, ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if f.secret, err = env.App.Secrets.OpenString(f.route.SecretSealed, "inbound_route:"+f.route.ID.String()+":secret"); err != nil {
		t.Fatal(err)
	}
	if f.token, err = env.App.Secrets.OpenString(f.route.HookTokenSealed, "inbound_route:"+f.route.ID.String()+":hook_token"); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.env.App.Pool.Exec(f.ctx, sql, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) count(sql string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.env.App.Pool.QueryRow(f.ctx, sql, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f *fixture) address(letter byte, msg uuid.UUID) string {
	return inbound.Box{Domain: f.route.Domain, Local: f.route.MailboxLocal}.Address(f.env.App.Links, letter, f.ws.ID, msg)
}

// post sends a raw message to the hook as the simulator would.
func (f *fixture) post(raw string, signed bool) int {
	f.t.Helper()
	req, _ := http.NewRequestWithContext(f.ctx, http.MethodPost, f.env.Server.URL+"/hooks/inbound/"+f.token, strings.NewReader(raw))
	req.Header.Set("Content-Type", "message/rfc822")
	sig := simulator.SignInbound(f.secret, []byte(raw))
	if !signed {
		sig = simulator.SignInbound("not-the-secret", []byte(raw))
	}
	req.Header.Set(simulator.InboundSignatureHeader, sig)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		f.t.Fatal(err)
	}
	_ = res.Body.Close()
	return res.StatusCode
}

// process runs the inbound.process job for what's waiting.
func (f *fixture) process() {
	f.t.Helper()
	rows, err := f.env.App.Pool.Query(f.ctx, `SELECT id FROM inbound_raw WHERE processed_at IS NULL ORDER BY received_at`)
	if err != nil {
		f.t.Fatal(err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := f.env.App.Inbound().Process(f.ctx, id); err != nil {
			f.t.Fatalf("process %s: %v", id, err)
		}
	}
}

func message(from, to, subject, id, extra, body string) string {
	return "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMessage-ID: <" + id + ">\r\nDate: " +
		time.Now().UTC().Format(time.RFC1123Z) + "\r\n" + extra + "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body + "\r\n"
}

// ours is a campaign message of the demo to an active contact.
type ours struct {
	id, contact, campaign uuid.UUID
	email                 string
}

func (f *fixture) ourMessage(campaign string, skip int) ours {
	f.t.Helper()
	var o ours
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT m.id, m.contact_id, m.campaign_id, m.email FROM messages m
		JOIN campaigns c ON c.id = m.campaign_id JOIN contacts ct ON ct.id = m.contact_id
		WHERE c.name = $1 AND ct.status = 'active' AND m.unsubscribed_at IS NULL AND m.status IN ('delivered', 'sent')
		ORDER BY m.id OFFSET $2 LIMIT 1`, campaign, skip).Scan(&o.id, &o.contact, &o.campaign, &o.email); err != nil {
		f.t.Fatal(err)
	}
	return o
}

type replyList struct {
	Items []struct {
		ID, Kind, Subject, ContactID string
		Source                       *struct{ Type, ID, Name string }
		AutoReply, Spam              bool
	}
	Counts struct{ Open, Done, Auto, Spam, Other, Unsubscribes int64 }
}

func (f *fixture) replies(view string) replyList {
	f.t.Helper()
	var l replyList
	f.c.Do("GET", wsPath+"/replies?view="+view+"&limit=100", nil).MustStatus(f.t, 200).JSON(f.t, &l)
	return l
}

func (l replyList) find(subject string) int {
	for i, it := range l.Items {
		if it.Subject == subject {
			return i
		}
	}
	return -1
}

// M58 G-15–G-17, G-20: a reply posted to the hook lands in Audience ›
// Replies with its contact and campaign once processed; a bad signature is
// refused and stored for diagnosis only; a forged tag is other mail; the
// same Message-ID twice is one row; the inbox's writes are audited and
// email.replied reaches a webhook endpoint. Viewers can't read replies.
func TestReplies(t *testing.T) {
	f := setup(t)
	hook := uuid.Must(uuid.NewV7())
	f.exec(`INSERT INTO webhooks (id, workspace_id, name, url, events, secret_enc, enabled, created_by, created_at, updated_at)
		SELECT $1, $2, 'Replies', 'https://hooks.example.com/replies', ARRAY['email.replied'], '\x00'::bytea, true, u.id, now(), now()
		FROM users u WHERE u.email = 'linh@acme.io'`, hook, f.ws.ID)
	o := f.ourMessage("October Product Update", 0)
	reply := message("Sam Rivera <"+o.email+">", f.address(inbound.LetterReply, o.id), "Re: October Product Update", "sam-1@outlook.com", "",
		"Loved the new routing rules.\n\n> On Tue, Acme wrote:\n> October Product Update")

	if st := f.post(reply, false); st != http.StatusUnauthorized {
		t.Fatalf("bad signature: %d", st)
	}
	if n := f.count(`SELECT count(*) FROM inbound_raw WHERE NOT signature_ok AND length(body) <= 2048`); n != 1 {
		t.Fatalf("unverified rows kept: %d", n)
	}
	if st := f.post(reply, true); st != http.StatusOK {
		t.Fatalf("hook: %d", st)
	}
	if st := f.post(reply, true); st != http.StatusOK {
		t.Fatalf("hook again: %d", st)
	}
	forged := message("Mallory <m@evil.example>", "r-"+strings.Repeat("a", 39)+"@in.news.acme.io", "Forged", "forged-1@evil.example", "", "Hello")
	if st := f.post(forged, true); st != http.StatusOK {
		t.Fatalf("forged: %d", st)
	}
	auto := message("Out <"+o.email+">", f.address(inbound.LetterReply, o.id), "Automatic reply: October Product Update", "auto-1@outlook.com",
		"Auto-Submitted: auto-replied\r\n", "I'm away.")
	if st := f.post(auto, true); st != http.StatusOK {
		t.Fatalf("auto-reply: %d", st)
	}
	f.process()

	open := f.replies("open")
	i := open.find("Re: October Product Update")
	if i < 0 {
		t.Fatalf("the reply isn't listed: %+v", open)
	}
	it := open.Items[i]
	if it.ContactID != o.contact.String() || it.Source == nil || it.Source.Type != "campaign" || it.Source.ID != o.campaign.String() || it.Source.Name != "October Product Update" {
		t.Fatalf("reply: %+v", it)
	}
	if n := f.count(`SELECT count(*) FROM inbound_messages WHERE header_message_id = 'sam-1@outlook.com'`); n != 1 {
		t.Fatalf("the same Message-ID twice made %d rows", n)
	}
	other := f.replies("other")
	if j := other.find("Forged"); j < 0 || other.Items[j].ContactID != "" {
		t.Fatalf("forged tag: %+v", other)
	}
	if a := f.replies("auto"); a.find("Automatic reply: October Product Update") < 0 {
		t.Fatalf("auto-reply: %+v", a)
	}
	var route inbound.InboundRouteView
	f.c.Do("GET", wsPath+"/inbound", nil).MustStatus(t, 200).JSON(t, &route)
	if route.Status != "receiving" || route.LastReceivedAt == nil {
		t.Fatalf("route after its first message: %+v", route)
	}
	// email.replied goes to the endpoint for the reply and the auto-reply,
	// not for other mail.
	if n := f.count(`SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND event = 'email.replied'`, hook); n != 2 {
		t.Fatalf("email.replied deliveries: %d", n)
	}
	if n := f.count(`SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND payload -> 'data' ->> 'contact_id' = $2`, hook, o.contact.String()); n != 2 {
		t.Fatalf("email.replied carries the contact: %d", n)
	}

	// Detail, then Mark done (audited).
	var d inbound.ReplyDetail
	f.c.Do("GET", wsPath+"/replies/"+it.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Contact == nil || d.Contact.Email != o.email || !strings.HasPrefix(d.Text, "Loved the new routing rules.") || !d.CanUnsubscribe {
		t.Fatalf("detail: %+v", d)
	}
	var res inbound.ReplyStatusResult
	f.c.Do("POST", wsPath+"/replies/status", map[string]any{"ids": []string{it.ID}, "status": "done"}).MustStatus(t, 200).JSON(t, &res)
	if res.Updated != 1 || f.replies("done").find("Re: October Product Update") < 0 {
		t.Fatalf("mark done: %+v", res)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE action = 'reply.status_updated' AND workspace_id = $1`, f.ws.ID); n != 1 {
		t.Fatalf("status audits: %d", n)
	}
	// Unsubscribe contact from the reply (audited, consent source reply).
	f.c.Do("POST", wsPath+"/replies/"+it.ID+"/unsubscribe-contact", nil).MustStatus(t, 200)
	if n := f.count(`SELECT count(*) FROM audit_events WHERE action = 'reply.contact_unsubscribed' AND workspace_id = $1`, f.ws.ID); n != 1 {
		t.Fatalf("unsubscribe audits: %d", n)
	}
	if n := f.count(`SELECT count(*) FROM suppressions WHERE workspace_id = $1 AND lower(email) = lower($2) AND reason = 'unsubscribe' AND source = 'reply'`, f.ws.ID, o.email); n != 1 {
		t.Fatalf("reply unsubscribe suppressions: %d", n)
	}
	// The route's own writes are audited too.
	if n := f.count(`SELECT count(*) FROM audit_events WHERE action = 'inbound.route_saved' AND workspace_id = $1`, f.ws.ID); n != 1 {
		t.Fatalf("route audits: %d", n)
	}

	// A Viewer can't read replies; a Content Writer can't either.
	for _, who := range []string{"aisha@acme.io", "daniel@acme.io"} {
		v := f.env.Client()
		v.SignInDemo(who)
		v.Do("GET", wsPath+"/replies", nil).MustStatus(t, 403)
		v.Do("PUT", wsPath+"/inbound", map[string]any{"kind": "provider"}).MustStatus(t, 403)
	}
	// The contact's timeline says they replied.
	var tl struct {
		Items []struct{ ID, Type, Subject string }
	}
	f.c.Do("GET", wsPath+"/contacts/"+o.contact.String()+"/activity?limit=100", nil).MustStatus(t, 200).JSON(t, &tl)
	found := false
	for _, x := range tl.Items {
		if x.Type == "replied" && x.ID == it.ID && x.Subject == "October Product Update" {
			found = true
		}
	}
	if !found {
		t.Fatalf("timeline: %+v", tl.Items)
	}
	// Removing the route keeps the replies and sets reply handling back.
	f.exec(`UPDATE workspaces SET sending_defaults = jsonb_set(coalesce(sending_defaults, '{}'), '{reply_handling}', '"inbox"') WHERE id = $1`, f.ws.ID)
	f.c.Do("DELETE", wsPath+"/inbound", nil).MustStatus(t, 204)
	if n := f.count(`SELECT count(*) FROM inbound_messages WHERE workspace_id = $1`, f.ws.ID); n == 0 {
		t.Fatal("removing the route removed the replies")
	}
	if n := f.count(`SELECT count(*) FROM workspaces WHERE id = $1 AND sending_defaults ->> 'reply_handling' = 'forward'`, f.ws.ID); n != 1 {
		t.Fatal("reply handling didn't go back to forwarding")
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE action = 'inbound.route_removed' AND workspace_id = $1`, f.ws.ID); n != 1 {
		t.Fatalf("remove audits: %d", n)
	}
}

// M58 G-16: an email to a message's u- address unsubscribes its contact as
// the one-click link would (suppression, consent source mailto, the
// campaign's unsubscribe count), and later sends name the u- address in
// List-Unsubscribe while the route receives.
func TestMailtoUnsubscribe(t *testing.T) {
	f := setup(t)
	o := f.ourMessage("Weekly Digest #141", 0)
	before := f.count(`SELECT coalesce(sum(unsubscribed), 0)::int FROM campaign_counters WHERE campaign_id = $1`, o.campaign)
	listed := f.replies("unsubscribes").Counts.Unsubscribes
	if st := f.post(message("<"+o.email+">", f.address(inbound.LetterUnsubscribe, o.id), "unsubscribe", "u-1@gmail.com", "", "unsubscribe"), true); st != 200 {
		t.Fatalf("hook: %d", st)
	}
	f.process()
	if n := f.count(`SELECT count(*) FROM suppressions WHERE workspace_id = $1 AND lower(email) = lower($2) AND reason = 'unsubscribe' AND source = 'mailto'`, f.ws.ID, o.email); n != 1 {
		t.Fatalf("suppressions: %d", n)
	}
	if n := f.count(`SELECT count(*) FROM contacts WHERE id = $1 AND status = 'unsubscribed'`, o.contact); n != 1 {
		t.Fatal("the contact is still subscribed")
	}
	if after := f.count(`SELECT coalesce(sum(unsubscribed), 0)::int FROM campaign_counters WHERE campaign_id = $1`, o.campaign); after != before+1 {
		t.Fatalf("campaign unsubscribes %d → %d", before, after)
	}
	if l := f.replies("unsubscribes"); l.Counts.Unsubscribes != listed+1 || l.find("unsubscribe") < 0 {
		t.Fatalf("unsubscribes view: %+v", l)
	}
	// The route receives now: sends use its u- address.
	st, err := inbound.StateOf(f.ctx, f.env.App.Q, f.ws.ID)
	if err != nil || !st.Receiving {
		t.Fatalf("state: %+v %v", st, err)
	}
	if got := f.env.App.SendingEngine().Inbound; got == nil {
		t.Fatal("the engine doesn't read the route")
	} else if box, err := got(f.ctx, f.ws.ID); err != nil || box == nil || !box.Receiving || box.Domain != "in.news.acme.io" {
		t.Fatalf("engine box: %+v %v", box, err)
	}
}

// M58 G-18: a person's reply to a flow's email ends the contact's run in a
// flow that stops on reply (exited, replied); an auto-reply doesn't.
func TestStopOnReply(t *testing.T) {
	f := setup(t)
	var run, flow, contact, msg uuid.UUID
	var version int32
	var email string
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT r.id, r.automation_id, r.contact_id, r.version, ct.email FROM automation_runs r
		JOIN automations a ON a.id = r.automation_id JOIN contacts ct ON ct.id = r.contact_id
		WHERE a.name = 'Welcome series' AND r.status = 'waiting' ORDER BY r.id LIMIT 1`).Scan(&run, &flow, &contact, &version, &email); err != nil {
		t.Fatal(err)
	}
	// The flow's email to that contact: one of the flow's sent messages, as theirs.
	msg = uuid.Must(uuid.NewV7())
	f.exec(`INSERT INTO messages SELECT (jsonb_populate_record(m, jsonb_build_object('id', $1::uuid, 'contact_id', $2::uuid, 'email', $3::text,
		'provider_message_id', NULL))).* FROM messages m WHERE m.automation_id = $4 ORDER BY m.id LIMIT 1`, msg, contact, email, flow)
	var raw []byte
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT graph FROM automation_versions WHERE automation_id = $1 AND version = $2`, flow, version).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	g, err := automations.ParseGraph(raw)
	if err != nil {
		t.Fatal(err)
	}
	g.Trigger().Data.Trigger.ExitOnReply = true
	f.exec(`UPDATE automation_versions SET graph = $3 WHERE automation_id = $1 AND version = $2`, flow, version, g.MustJSON())
	status := func() (string, string) {
		var s, r string
		if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT status, coalesce(exit_reason, '') FROM automation_runs WHERE id = $1`, run).Scan(&s, &r); err != nil {
			t.Fatal(err)
		}
		return s, r
	}
	to := f.address(inbound.LetterReply, msg)
	f.post(message("<"+email+">", to, "Automatic reply: Welcome", "ooo-1@x.example", "Auto-Submitted: auto-replied\r\n", "Away"), true)
	f.process()
	if s, _ := status(); s != "waiting" {
		t.Fatalf("an auto-reply ended the run: %s", s)
	}
	f.post(message("<"+email+">", to, "Re: Welcome", "human-1@x.example", "", "Thanks!"), true)
	f.process()
	if s, r := status(); s != "exited" || r != automations.ExitReplied {
		t.Fatalf("after a reply: %s %s", s, r)
	}
}

// M58 G-19: a DMARC aggregate report sent to dmarc@ is stored with via
// 'inbound'; an ARF report at fbl@ becomes one complaint event for the
// original message.
func TestReports(t *testing.T) {
	f := setup(t)
	end := time.Now().Unix()
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<feedback><report_metadata><org_name>google.com</org_name><email>noreply-dmarc-support@google.com</email><report_id>in-1</report_id>
<date_range><begin>%d</begin><end>%d</end></date_range></report_metadata>
<policy_published><domain>news.acme.io</domain><adkim>r</adkim><aspf>r</aspf><p>none</p><pct>100</pct></policy_published>
<record><row><source_ip>192.0.2.10</source_ip><count>12</count><policy_evaluated><disposition>none</disposition><dkim>pass</dkim><spf>pass</spf></policy_evaluated></row>
<identifiers><header_from>news.acme.io</header_from></identifiers><auth_results><spf><domain>news.acme.io</domain><result>pass</result></spf></auth_results></record>
</feedback>`, end-86400, end)
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write([]byte(xml))
	_ = zw.Close()
	dmarc := "From: noreply-dmarc-support@google.com\r\nTo: dmarc@in.news.acme.io\r\nSubject: Report domain: news.acme.io\r\nMessage-ID: <dmarc-1@google.com>\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=x\r\n\r\n--x\r\nContent-Type: text/plain\r\n\r\nReport\r\n" +
		"--x\r\nContent-Type: application/gzip\r\nContent-Disposition: attachment; filename=\"google.com!news.acme.io!1!2.xml.gz\"\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
		base64.StdEncoding.EncodeToString(gz.Bytes()) + "\r\n--x--\r\n"
	if st := f.post(dmarc, true); st != 200 {
		t.Fatalf("dmarc: %d", st)
	}

	o := f.ourMessage("Weekly Digest #141", 1)
	var provider uuid.UUID
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT provider_id FROM messages WHERE id = $1`, o.id).Scan(&provider); err != nil {
		t.Fatal(err)
	}
	arf := "From: fbl@mail.example\r\nTo: fbl@in.news.acme.io\r\nSubject: Abuse report\r\nMessage-ID: <arf-1@mail.example>\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: multipart/report; report-type=feedback-report; boundary=r\r\n\r\n" +
		"--r\r\nContent-Type: text/plain\r\n\r\nThis is an abuse report.\r\n" +
		"--r\r\nContent-Type: message/feedback-report\r\n\r\nFeedback-Type: abuse\r\nUser-Agent: Example-FBL/1.0\r\nVersion: 1\r\nOriginal-Rcpt-To: " + o.email + "\r\n\r\n" +
		"--r\r\nContent-Type: message/rfc822\r\n\r\nFrom: digest@news.acme.io\r\nTo: " + o.email + "\r\nSubject: Weekly Digest #141\r\nMessage-ID: <" + o.id.String() + "@news.acme.io>\r\n" +
		"X-ReputeCast-Message: " + o.id.String() + "\r\n\r\nHello\r\n--r--\r\n"
	if st := f.post(arf, true); st != 200 {
		t.Fatalf("arf: %d", st)
	}
	f.process()
	if n := f.count(`SELECT count(*) FROM dmarc_reports WHERE workspace_id = $1 AND via = 'inbound' AND report_id = 'in-1'`, f.ws.ID); n != 1 {
		t.Fatalf("dmarc reports via inbound: %d", n)
	}
	if n := f.count(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'complaint'`, o.id); n != 1 {
		t.Fatalf("complaint events: %d", n)
	}
	if n := f.count(`SELECT count(*) FROM provider_raw_events WHERE provider_id = $1 AND source = 'inbound'`, provider); n != 1 {
		t.Fatalf("raw events from inbound: %d", n)
	}
	// Reports aren't replies.
	l := f.replies("open")
	if l.find("Abuse report") >= 0 || l.find("Report domain: news.acme.io") >= 0 {
		t.Fatalf("reports listed as replies: %+v", l)
	}
}

// fakeBox is a mailbox with messages.
type fakeBox struct {
	msgs  []bouncebox.Message
	acked []string
}

func (b *fakeBox) Fetch(context.Context, int) ([]bouncebox.Message, error) { return b.msgs, nil }
func (b *fakeBox) Ack(_ context.Context, ids []string) error {
	b.acked = append(b.acked, ids...)
	return nil
}
func (b *fakeBox) Close() error { return nil }

// M58 G-15: a mailbox route reads its mailbox every minute, marks what it
// stored, and replies to its plus addresses land in the inbox; a mailbox
// that refuses the password makes the route failing and tells the people
// who manage domains once.
func TestMailboxRoute(t *testing.T) {
	f := setup(t)
	var view inbound.InboundRouteView
	in := map[string]any{"kind": "mailbox", "mailbox": map[string]any{"address": "replies@acme.io", "protocol": "imap", "host": "imap.acme.io", "port": 993,
		"security": "tls", "username": "replies@acme.io", "password": "secret-1"}}
	f.c.Do("PUT", wsPath+"/inbound", in).MustStatus(t, 200).JSON(t, &view)
	if view.Kind != "mailbox" || view.Mailbox == nil || !view.Mailbox.PasswordSet || view.Addresses.DMARC != "replies+dmarc@acme.io" {
		t.Fatalf("mailbox route: %+v", view)
	}
	if strings.Contains(string(f.c.Do("GET", wsPath+"/inbound", nil).Body), "secret-1") {
		t.Fatal("the password came back")
	}
	// A local part over 22 characters is refused.
	long := map[string]any{"kind": "mailbox", "mailbox": map[string]any{"address": strings.Repeat("r", 23) + "@acme.io", "protocol": "imap", "host": "imap.acme.io",
		"security": "tls", "username": "u", "password": "p"}}
	f.c.Do("PUT", wsPath+"/inbound", long).MustStatus(t, 422)

	route, err := f.env.App.Q.InboundRouteGet(f.ctx, f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.route = route
	o := f.ourMessage("October Product Update", 2)
	box := &fakeBox{msgs: []bouncebox.Message{{ID: "1", Raw: []byte(message("<"+o.email+">", f.address(inbound.LetterReply, o.id), "Re: from the mailbox", "mb-1@x.example", "", "Hi"))}}}
	s := f.env.App.Inbound()
	s.OpenBox = func(context.Context, bouncebox.Config) (bouncebox.Box, error) { return box, nil }
	if n, err := s.Poll(f.ctx, route); err != nil || n != 1 || len(box.acked) != 1 {
		t.Fatalf("poll: %d %v %v", n, err, box.acked)
	}
	f.process()
	if l := f.replies("open"); l.find("Re: from the mailbox") < 0 {
		t.Fatalf("mailbox reply: %+v", l)
	}
	s.OpenBox = func(context.Context, bouncebox.Config) (bouncebox.Box, error) { return nil, bouncebox.ErrAuth }
	for range 2 {
		_, _ = s.Poll(f.ctx, route)
	}
	f.c.Do("GET", wsPath+"/inbound", nil).MustStatus(t, 200).JSON(t, &view)
	if view.Status != "failing" || !strings.Contains(view.StatusDetail, "sign in") {
		t.Fatalf("failing mailbox: %+v", view)
	}
	if n := f.count(`SELECT count(*) FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE n.kind = 'inbound_failing' AND n.workspace_id = $1 AND u.email = 'linh@acme.io'`, f.ws.ID); n != 1 {
		t.Fatalf("failing notices to the Owner: %d", n)
	}
}
