package recommendations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/recommendations"
	pp "github.com/hostrepute/reputecast/internal/recommendations/partnerproto"
	"github.com/hostrepute/reputecast/internal/store"
)

// instance is one ReputeCast in the test: its app, a workspace with one
// list, its owner signed in, and a recommendations service on its database
// (the jobs' side).
type instance struct {
	t      *testing.T
	env    *apptest.Env
	ws     store.Workspace
	list   uuid.UUID
	owner  *apptest.Client
	s      *recommendations.Service
	prefix string
}

func newInstance(t *testing.T, name, list, owner string) *instance {
	t.Helper()
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: owner, Name: strings.Split(owner, "@")[0]})
	in := &instance{t: t, env: env, ws: env.CreateWorkspace(name, u.ID), list: uuid.Must(uuid.NewV7())}
	in.exec(`INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, $3)`, in.list, in.ws.ID, list)
	in.owner = env.Client()
	in.owner.SignInDemo(owner)
	in.prefix = "/workspaces/" + in.ws.Slug + "/grow"
	in.s = &recommendations.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Links: env.App.Links,
		Instance: func() string { return env.Server.URL }, AllowPrivate: func() bool { return true }}
	return in
}

func (in *instance) exec(sql string, args ...any) {
	in.t.Helper()
	if _, err := in.env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		in.t.Fatalf("%s: %v", sql, err)
	}
}

func (in *instance) count(sql string, args ...any) int {
	in.t.Helper()
	var n int
	if err := in.env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		in.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func (in *instance) view() recommendations.RecommendationsView {
	in.t.Helper()
	var v recommendations.RecommendationsView
	in.owner.Do("GET", in.prefix+"/recommendations", nil).MustStatus(in.t, 200).JSON(in.t, &v)
	return v
}

// signed sends a signed partner message straight to an instance, as the
// other side would.
func signed(t *testing.T, to *instance, link, action string, secret string, body any, ts time.Time) (int, []byte) {
	t.Helper()
	raw, _ := json.Marshal(body)
	path := pp.Prefix + "/links/" + link + "/" + action
	req, _ := http.NewRequest(http.MethodPost, to.env.Server.URL+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(pp.HeaderTimestamp, strconv.FormatInt(ts.Unix(), 10))
	req.Header.Set(pp.HeaderSignature, pp.SignatureHeader([]string{secret}, ts.Unix(), http.MethodPost, path, raw))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// Partner links between two instances (brief §3.9): codes, signatures,
// replay protection, the daily limit, confirmations, pausing, rotating the
// secret and ending.
func TestPartnerLinks(t *testing.T) {
	ctx := context.Background()
	acme := newInstance(t, "Acme Weekly Digest", "Weekly Digest", "linh@acme.test")
	ops := newInstance(t, "Ops Weekly", "Ops Weekly", "marcus@ops.test")

	// Ops Weekly creates a code for Acme: shown once, never logged.
	var code recommendations.PartnerCodeCreated
	ops.owner.Do("POST", ops.prefix+"/partner-codes", map[string]any{"label": "Acme", "listId": ops.list.String(), "dailyLimit": 2}).
		MustStatus(t, 201).JSON(t, &code)
	if !strings.HasPrefix(code.URL, ops.env.Server.URL+"/p/") || len(code.Code) != 14 {
		t.Fatalf("code: %+v", code)
	}
	if n := ops.count(`SELECT count(*) FROM audit_events WHERE action = 'partner.code_created' AND (coalesce(after::text, '') || coalesce(metadata::text, '')) LIKE '%'||$1||'%'`, code.Code); n != 0 {
		t.Fatal("the partner code is in the audit log")
	}
	if ops.count(`SELECT count(*) FROM partner_codes WHERE code_hash = $1`, pp.HashCode(code.Code)) != 1 {
		t.Fatal("only the code's hash is kept")
	}
	// A code opened in a browser explains itself.
	resp, err := http.Get(code.URL)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("code page: %v %v", resp, err)
	}
	_ = resp.Body.Close()

	// Acme pastes it: the profile first, then Connect.
	var prev recommendations.PartnerPreview
	acme.owner.Do("POST", acme.prefix+"/partners/preview", map[string]any{"code": code.URL}).MustStatus(t, 200).JSON(t, &prev)
	if prev.Name != "Ops Weekly" || prev.List != "Ops Weekly" || prev.DailyLimit != 2 || !prev.DoubleOptIn {
		t.Fatalf("preview: %+v", prev)
	}
	r := acme.owner.Do("POST", acme.prefix+"/partners/preview", map[string]any{"code": "https://fabrikam.example/newsletter"}).MustStatus(t, 422)
	if p := r.Problem(t); !strings.Contains(p["detail"].(string), "not a partner code") {
		t.Fatalf("a subscribe page: %v", p)
	}
	var item recommendations.RecommendationItem
	acme.owner.Do("POST", acme.prefix+"/partners/connect", map[string]any{"code": code.URL, "showConfirm": true, "showBlock": true}).MustStatus(t, 201).JSON(t, &item)
	if item.Kind != recommendations.KindPartner || item.State != "connected" || item.Hidden || item.DailyLimit != 2 {
		t.Fatalf("connected: %+v", item)
	}
	// The code works once, and says who used it.
	r = acme.owner.Do("POST", acme.prefix+"/partners/preview", map[string]any{"code": code.URL}).MustStatus(t, 422)
	if p := r.Problem(t); !strings.Contains(p["detail"].(string), "linh used this code") {
		t.Fatalf("used code: %v", p)
	}
	ov := ops.view()
	if len(ov.RecommendingYou) != 1 || ov.RecommendingYou[0].Name != "Acme Weekly Digest" || ov.RecommendingYou[0].Label != "Acme" {
		t.Fatalf("recommending you: %+v", ov.RecommendingYou)
	}
	opsLink := ov.RecommendingYou[0].LinkID
	acmeLink := item.LinkID
	// Both sides keep the secret sealed, and agree on it.
	var sealedA, sealedB []byte
	_ = acme.env.App.Pool.QueryRow(ctx, `SELECT secret FROM partner_links WHERE id = $1`, uuid.MustParse(acmeLink)).Scan(&sealedA)
	_ = ops.env.App.Pool.QueryRow(ctx, `SELECT secret FROM partner_links WHERE id = $1`, uuid.MustParse(opsLink)).Scan(&sealedB)
	secretA, errA := acme.env.App.Secrets.OpenString(sealedA, "partner_link:"+acmeLink+":secret")
	secretB, errB := ops.env.App.Secrets.OpenString(sealedB, "partner_link:"+opsLink+":secret")
	if errA != nil || errB != nil || secretA != secretB || !strings.HasPrefix(secretA, "rcps_") || bytes.Contains(sealedA, []byte(secretA)) {
		t.Fatalf("secrets: %v %v, equal %v", errA, errB, secretA == secretB)
	}

	// A reader presses Subscribe on Acme's confirmation page.
	recID := uuid.MustParse(item.ID)
	out, err := acme.s.RequestSubscribe(ctx, &acme.ws, recID, "sam.rivera@outlook.com", "confirm")
	if err != nil || out != recommendations.OutcomeAccepted {
		t.Fatalf("subscribe: %s %v", out, err)
	}
	var contact uuid.UUID
	var source, status string
	if err := ops.env.App.Pool.QueryRow(ctx, `SELECT c.id, c.source, m.status FROM contacts c JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $2
		WHERE c.workspace_id = $1 AND c.email = 'sam.rivera@outlook.com'`, ops.ws.ID, ops.list).Scan(&contact, &source, &status); err != nil {
		t.Fatal(err)
	}
	if source != "partner:"+opsLink || status != "pending" {
		t.Fatalf("partner signup: source %q, %s", source, status)
	}
	if ops.count(`SELECT count(*) FROM partner_requests_in WHERE contact_id = $1 AND asked AND optin_status = 'queued' AND source_name = 'Acme Weekly Digest'`, contact) != 1 {
		t.Fatal("no confirmation email queued")
	}
	// Acme keeps no address once the partner accepted.
	if acme.count(`SELECT count(*) FROM partner_requests_out WHERE email IS NOT NULL`) != 0 {
		t.Fatal("the reader's address stayed on the recommending side")
	}

	// Replays: the same request id gets the same answer and nothing more;
	// an old timestamp or another secret is refused.
	req := pp.SubscribeRequest{Protocol: 1, Request: "req-1", Email: "jordan@fabrikam.example", Source: "Acme Weekly Digest", At: time.Now()}
	for i := 0; i < 2; i++ {
		st, body := signed(t, ops, opsLink, "subscribe", secretA, req, time.Now())
		if st != 200 || !strings.Contains(string(body), `"accepted"`) {
			t.Fatalf("subscribe %d: %d %s", i, st, body)
		}
	}
	if n := ops.count(`SELECT count(*) FROM partner_requests_in WHERE request_id = 'req-1'`); n != 1 {
		t.Fatalf("replayed request recorded %d times", n)
	}
	if st, body := signed(t, ops, opsLink, "subscribe", secretA, pp.SubscribeRequest{Protocol: 1, Request: "req-2", Email: "x@example.com", At: time.Now()},
		time.Now().Add(-10*time.Minute)); st != 401 || !strings.Contains(string(body), pp.ErrSignatureRejected) {
		t.Fatalf("stale: %d %s", st, body)
	}
	if st, _ := signed(t, ops, opsLink, "subscribe", "rcps_wrong", pp.SubscribeRequest{Protocol: 1, Request: "req-3", Email: "x@example.com", At: time.Now()},
		time.Now()); st != 401 {
		t.Fatalf("wrong secret: %d", st)
	}

	// The daily limit (2): req-1 and Sam took it. The next reader is
	// refused, and Acme hides Ops Weekly until 00:00 UTC.
	out, _ = acme.s.RequestSubscribe(ctx, &acme.ws, recID, "kim@woodgrove.example", "page")
	if out != recommendations.OutcomeLimit {
		t.Fatalf("over the limit: %s", out)
	}
	it := acme.view().Items[0]
	if it.State != "limit" || !it.Hidden || it.LimitUntil == nil {
		t.Fatalf("at the limit: %+v", it)
	}
	acme.exec(`UPDATE partner_links SET limit_until = NULL`)

	// Sam confirms on Ops Weekly: Acme counts it, without learning who.
	tok := ops.env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Confirm, WorkspaceID: ops.ws.ID, ContactID: contact})
	ops.env.Client().Do("GET", "/p/"+ops.ws.Slug+"/confirm/"+tok, nil).MustStatus(t, 200)
	if n, err := ops.s.SendNotices(ctx); err != nil || n != 1 {
		t.Fatalf("notices: %d %v", n, err)
	}
	if n := acme.count(`SELECT coalesce(sum(confirmed), 0)::int FROM recommendation_daily WHERE recommendation_id = $1`, recID); n != 1 {
		t.Fatalf("confirmed on Acme: %d", n)
	}
	if n := acme.count(`SELECT count(*) FROM partner_requests_out WHERE confirmed_at IS NOT NULL`); n != 1 {
		t.Fatalf("confirmed requests: %d", n)
	}

	// Ops Weekly pauses: Acme hides it; then resumes.
	ops.owner.Do("POST", ops.prefix+"/partner-links/"+opsLink+"/pause", nil).MustStatus(t, 204)
	if it := acme.view().Items[0]; it.State != "partner_paused" || !it.Hidden {
		t.Fatalf("paused by the partner: %+v", it)
	}
	if out, _ := acme.s.RequestSubscribe(ctx, &acme.ws, recID, "ava@northwind.example", "confirm"); out != recommendations.OutcomeUnavailable {
		t.Fatalf("subscribe while paused: %s", out)
	}
	acme.owner.Do("POST", acme.prefix+"/partner-links/"+acmeLink+"/resume", nil).MustStatus(t, 409)
	ops.owner.Do("POST", ops.prefix+"/partner-links/"+opsLink+"/resume", nil).MustStatus(t, 204)
	if it := acme.view().Items[0]; it.State != "connected" {
		t.Fatalf("resumed: %+v", it)
	}

	// Rotating the secret: both sides switch, the old one works 24 hours.
	ops.exec(`UPDATE partner_links SET daily_limit = 50 WHERE id = $1`, uuid.MustParse(opsLink))
	acme.owner.Do("POST", acme.prefix+"/partner-links/"+acmeLink+"/rotate", nil).MustStatus(t, 204)
	_ = acme.env.App.Pool.QueryRow(ctx, `SELECT secret FROM partner_links WHERE id = $1`, uuid.MustParse(acmeLink)).Scan(&sealedA)
	_ = ops.env.App.Pool.QueryRow(ctx, `SELECT secret FROM partner_links WHERE id = $1`, uuid.MustParse(opsLink)).Scan(&sealedB)
	newA, _ := acme.env.App.Secrets.OpenString(sealedA, "partner_link:"+acmeLink+":secret")
	newB, _ := ops.env.App.Secrets.OpenString(sealedB, "partner_link:"+opsLink+":secret")
	if newA != newB || newA == secretA {
		t.Fatal("the rotated secrets don't match")
	}
	if st, _ := signed(t, ops, opsLink, "subscribe", secretA, pp.SubscribeRequest{Protocol: 1, Request: "req-old", Email: "old@example.com", At: time.Now()}, time.Now()); st != 200 {
		t.Fatalf("the previous secret within 24 hours: %d", st)
	}
	if out, _ := acme.s.RequestSubscribe(ctx, &acme.ws, recID, "noor@tailspin.example", "page"); out != recommendations.OutcomeAccepted {
		t.Fatalf("after the rotation: %s", out)
	}

	// Ending: both sides drop it; Acme's recommendation hides.
	acme.owner.Do("POST", acme.prefix+"/partner-links/"+acmeLink+"/end", nil).MustStatus(t, 204)
	if len(ops.view().RecommendingYou) != 0 {
		t.Fatal("the ended partner is still recommending")
	}
	if it := acme.view().Items[0]; it.State != "ended" || !it.Hidden {
		t.Fatalf("ended: %+v", it)
	}
	if st, _ := signed(t, ops, opsLink, "subscribe", newA, pp.SubscribeRequest{Protocol: 1, Request: "req-end", Email: "x@example.com", At: time.Now()}, time.Now()); st != 410 {
		t.Fatalf("a request after the end: %d", st)
	}
	audits := acme.count(`SELECT count(*) FROM audit_events WHERE action IN ('partner.link_connected', 'partner.secret_rotated', 'partner.link_ended')`)
	if audits != 3 {
		t.Fatalf("Acme's partner audit entries: %d", audits)
	}

	// A partner that lost the secret: Acme's next request is rejected, the
	// link turns invalid and members with grow:connect hear about it.
	var code2 recommendations.PartnerCodeCreated
	ops.owner.Do("POST", ops.prefix+"/partner-codes", map[string]any{"label": "Acme again", "listId": ops.list.String(), "dailyLimit": 50}).MustStatus(t, 201).JSON(t, &code2)
	acme.owner.Do("POST", acme.prefix+"/partners/connect", map[string]any{"code": code2.URL, "showConfirm": true, "showBlock": false}).MustStatus(t, 201).JSON(t, &item)
	var second uuid.UUID
	_ = ops.env.App.Pool.QueryRow(ctx, `SELECT id FROM partner_links WHERE status = 'connected'`).Scan(&second)
	lost, _ := ops.env.App.Secrets.SealString("rcps_lost", "partner_link:"+second.String()+":secret")
	ops.exec(`UPDATE partner_links SET secret = $1 WHERE id = $2`, lost, second)
	if out, _ := acme.s.RequestSubscribe(ctx, &acme.ws, uuid.MustParse(item.ID), "lee@contoso.example", "confirm"); out != recommendations.OutcomeUnavailable {
		t.Fatalf("rejected signature: %s", out)
	}
	it = acme.view().Items[1]
	if it.State != "invalid" || it.InvalidReason != "signature_rejected" || !it.Hidden {
		t.Fatalf("invalid: %+v", it)
	}
	if acme.count(`SELECT count(*) FROM notifications WHERE kind = 'partner_link_problem'`) == 0 {
		t.Fatal("nobody was told about the invalid link")
	}
}

// A partner instance at a private address is refused, unless private
// networks are allowed (development); a code for an instance that doesn't
// answer says so.
func TestPartnerCodeProblems(t *testing.T) {
	ctx := context.Background()
	acme := newInstance(t, "Acme Weekly Digest", "Weekly Digest", "linh@acme.test")
	strict := *acme.s
	strict.AllowPrivate = func() bool { return false }
	_, err := strict.PreviewPartner(ctx, &acme.ws, "https://10.0.4.12/p/H3RT-6QWN-8ZKB")
	if err == nil || !strings.Contains(err.Error(), "10.0.4.12 is a private address") {
		t.Fatalf("private: %v", err)
	}
	_, err = acme.s.PreviewPartner(ctx, &acme.ws, "http://127.0.0.1:1/p/H3RT-6QWN-8ZKB")
	if err == nil || !strings.Contains(err.Error(), "didn't answer over HTTPS") {
		t.Fatalf("unreachable: %v", err)
	}
	// An instance without partner links (no /partner/v1 there).
	_, err = acme.s.PreviewPartner(ctx, &acme.ws, acme.env.Server.URL+"/nothing/p/H3RT-6QWN-8ZKB")
	if err == nil || !strings.Contains(err.Error(), "without partner links") {
		t.Fatalf("unsupported: %v", err)
	}
	// An expired code.
	var code recommendations.PartnerCodeCreated
	acme.owner.Do("POST", acme.prefix+"/partner-codes", map[string]any{"label": "Old", "listId": acme.list.String(), "dailyLimit": 5}).MustStatus(t, 201).JSON(t, &code)
	acme.exec(`UPDATE partner_codes SET expires_at = now() - interval '1 day'`)
	other := newInstance(t, "Fabrikam", "Engineering Letter", "pat@fabrikam.test")
	_, err = other.s.PreviewPartner(ctx, &other.ws, code.URL)
	if err == nil || !strings.Contains(err.Error(), "This code expired on") {
		t.Fatalf("expired: %v", err)
	}
}
