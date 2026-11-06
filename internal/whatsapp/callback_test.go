package whatsapp_test

// The callback URL: Meta's webhook at /hooks/whatsapp/{workspace}/{token},
// the old address without a token kept only for accounts Meta hasn't
// called there yet, a new token on demand, and the token kept out of logs.

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/whatsapp"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

// overview reads Sending providers › WhatsApp as the owner.
func (f *fixture) overview(t *testing.T) whatsapp.WhatsAppOverview {
	t.Helper()
	var ov whatsapp.WhatsAppOverview
	f.owner.Do(http.MethodGet, f.path+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	return ov
}

// exec changes the account's row as an earlier release left it.
func (f *fixture) exec(t *testing.T, sql string) {
	t.Helper()
	if _, err := f.env.App.Pool.Exec(context.Background(), sql+` WHERE workspace_id = $1`, f.ws.ID); err != nil {
		t.Fatal(err)
	}
}

// templateDelivery is a delivery signed with the app secret.
func templateDelivery() ([]byte, string) {
	return metatest.TemplateWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, "1290033512244412", "summit_venue_change", "en_US", "APPROVED", "NONE", "")
}

func (f *fixture) stored(t *testing.T) int64 {
	t.Helper()
	return one[int64](t, f.env, `SELECT count(*) FROM whatsapp_raw_events WHERE workspace_id = $1`, f.ws.ID)
}

// A new account's webhook is its callback URL: the workspace's slug alone
// never reaches it, and a wrong token is refused before the signature is
// looked at, counted on the account and not stored.
func TestWhatsAppCallbackURL(t *testing.T) {
	f := newFixture(t)
	ov := f.connect(t)
	old := "/hooks/whatsapp/" + f.ws.Slug
	token := strings.TrimPrefix(f.callback, old+"/")
	if ov.Account.Status != "connected" || ov.Account.OldAddress || ov.Account.VerifiedAt == nil {
		t.Fatalf("after the verification request at the callback URL: %+v", ov.Account)
	}

	// The old address: neither Meta's verification request nor a signed
	// delivery gets through.
	if code, _ := f.verify(t, old, ov.Account.VerifyToken, "7"); code != http.StatusForbidden {
		t.Fatalf("verification at the old address: %d", code)
	}
	body, sig := templateDelivery()
	if st := f.postTo(t, old, body, sig); st != http.StatusNotFound {
		t.Fatalf("delivery to the old address: %d", st)
	}
	// A wrong token, however close, even with a valid signature.
	wrong := []string{old + "/" + token[:len(token)-1], old + "/" + strings.ToUpper(token), old + "/" + token + "x"}
	for _, path := range wrong {
		if st := f.postTo(t, path, body, sig); st != http.StatusNotFound {
			t.Fatalf("delivery to a wrong token: %d", st)
		}
		if code, _ := f.verify(t, path, ov.Account.VerifyToken, "7"); code != http.StatusForbidden {
			t.Fatalf("verification at a wrong token: %d", code)
		}
	}
	// The token is checked before the signature: a wrong token without a
	// signature is a wrong address, not a refused signature.
	if st := f.postTo(t, wrong[0], []byte("not even JSON"), ""); st != http.StatusNotFound {
		t.Fatalf("unsigned delivery to a wrong token: %d", st)
	}
	a := f.overview(t).Account
	if a.WrongAddressDeliveries != 5 || a.WrongAddressAt == nil || a.RejectedDeliveries != 0 || f.stored(t) != 0 {
		t.Fatalf("refused for their address: %d (signature %d), %d stored", a.WrongAddressDeliveries, a.RejectedDeliveries, f.stored(t))
	}

	// The callback URL takes it; a bad signature there is still refused.
	if st := f.post(t, body, meta.Sign("not-the-secret", body)); st != http.StatusUnauthorized {
		t.Fatalf("bad signature at the callback URL: %d", st)
	}
	if st := f.hook(t, body, sig); st != http.StatusOK {
		t.Fatalf("delivery to the callback URL: %d", st)
	}
	if s := one[string](t, f.env, `SELECT status FROM whatsapp_templates WHERE workspace_id = $1 AND name = 'summit_venue_change'`, f.ws.ID); s != "approved" || f.stored(t) != 1 {
		t.Fatalf("template status: %s, %d stored", s, f.stored(t))
	}
	if a := f.overview(t).Account; a.RejectedDeliveries != 1 || a.WrongAddressDeliveries != 5 {
		t.Fatalf("counts: %+v", a)
	}
}

// An account connected before callback URLs had tokens keeps the old
// address until Meta first calls its callback URL: it gets its token the
// first time it's read, and once Meta verifies the new URL, or signs a
// delivery to it, the old address refuses its deliveries.
func TestWhatsAppCallbackURLMovesAnOlderAccount(t *testing.T) {
	f := newFixture(t)
	ov := f.connect(t)
	old := "/hooks/whatsapp/" + f.ws.Slug
	// As an earlier release left it: no token, the old address open.
	f.exec(t, `UPDATE whatsapp_accounts SET callback_token_hash = NULL, callback_token_enc = NULL, callback_moved_at = NULL`)
	body, sig := templateDelivery()
	if st := f.post(t, body, sig); st != http.StatusNotFound {
		t.Fatalf("a token the account doesn't have: %d", st)
	}
	if code, _ := f.verify(t, old, ov.Account.VerifyToken, "1"); code != http.StatusOK {
		t.Fatalf("verification at the old address: %d", code)
	}
	if st := f.postTo(t, old, body, sig); st != http.StatusOK || f.stored(t) != 1 {
		t.Fatalf("delivery to the old address: %d, %d stored", st, f.stored(t))
	}
	if st := f.postTo(t, old, body, meta.Sign("not-the-secret", body)); st != http.StatusUnauthorized {
		t.Fatalf("bad signature at the old address: %d", st)
	}

	// Read, it gets its token (the system's doing, logged) and keeps it.
	a := f.overview(t).Account
	if a.WebhookURL == "" || a.WebhookURL == ov.Account.WebhookURL || !a.OldAddress || a.VerifiedAt == nil || a.Status != "connected" {
		t.Fatalf("read after the upgrade: %+v", a)
	}
	if again := f.overview(t).Account; again.WebhookURL != a.WebhookURL {
		t.Fatalf("a second read made another token: %q, then %q", a.WebhookURL, again.WebhookURL)
	}
	if n := f.actions(t)["whatsapp.callback_created"]; n != 1 {
		t.Fatalf("callback_created entries: %d", n)
	}
	callback := pathOf(t, a.WebhookURL)
	// Nothing moves it but Meta: a bad signature at the callback URL, or
	// the old address in use, leaves the old address open.
	if st := f.postTo(t, callback, body, meta.Sign("not-the-secret", body)); st != http.StatusUnauthorized {
		t.Fatalf("bad signature at the callback URL: %d", st)
	}
	if st := f.postTo(t, old, body, sig); st != http.StatusOK || !f.overview(t).Account.OldAddress {
		t.Fatalf("the old address after a refused delivery to the new one: %d", st)
	}

	// Meta verifies the callback URL: moved, logged as the system's doing,
	// and the old address refuses the account from then on.
	if code, body := f.verify(t, callback, a.VerifyToken, "2"); code != http.StatusOK || body != "2" {
		t.Fatalf("verification at the callback URL: %d %q", code, body)
	}
	if a := f.overview(t).Account; a.OldAddress {
		t.Fatalf("not moved after Meta's verification request: %+v", a)
	}
	if code, _ := f.verify(t, old, a.VerifyToken, "3"); code != http.StatusForbidden {
		t.Fatalf("verification at the old address once moved: %d", code)
	}
	before := f.stored(t)
	if st := f.postTo(t, old, body, sig); st != http.StatusNotFound || f.stored(t) != before {
		t.Fatalf("delivery to the old address once moved: %d", st)
	}
	if a := f.overview(t).Account; a.WrongAddressDeliveries != 2 {
		t.Fatalf("refused for their address: %d", a.WrongAddressDeliveries)
	}
	if st := f.postTo(t, callback, body, sig); st != http.StatusOK {
		t.Fatalf("delivery to the callback URL: %d", st)
	}

	// A delivery with a valid signature moves it too.
	f.exec(t, `UPDATE whatsapp_accounts SET callback_moved_at = NULL`)
	if st := f.postTo(t, callback, body, sig); st != http.StatusOK || f.overview(t).Account.OldAddress {
		t.Fatalf("not moved by a signed delivery: %d", st)
	}
	if st := f.postTo(t, old, body, sig); st != http.StatusNotFound {
		t.Fatalf("delivery to the old address once moved: %d", st)
	}
	if n := f.actions(t)["whatsapp.callback_moved"]; n != 2 {
		t.Fatalf("callback_moved entries: %d", n)
	}
	var who, via string
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT actor_type, metadata->>'via' FROM audit_events WHERE workspace_id = $1
		AND action = 'whatsapp.callback_moved' ORDER BY occurred_at DESC LIMIT 1`, f.ws.ID).Scan(&who, &via); err != nil {
		t.Fatal(err)
	}
	if who != "system" || via != "delivery" {
		t.Fatalf("moved by %s, via %s", who, via)
	}
}

// A new callback URL stops the current one at once, and the account waits
// for Meta's verification request at the new one. Only people who can
// manage providers make one.
func TestWhatsAppCallbackURLRotates(t *testing.T) {
	f := newFixture(t)
	ov := f.connect(t)
	previous := f.callback
	var got whatsapp.WhatsAppOverview
	f.owner.Do(http.MethodPost, f.path+"/whatsapp/account/callback-url/rotate", nil).MustStatus(t, 200).JSON(t, &got)
	a := got.Account
	if a.WebhookURL == "" || a.WebhookURL == ov.Account.WebhookURL || a.Status != "waiting" || a.VerifiedAt != nil || a.OldAddress {
		t.Fatalf("after a new callback URL: %+v", a)
	}
	callback := pathOf(t, a.WebhookURL)

	// The previous URL: refused at once.
	body, sig := templateDelivery()
	if st := f.postTo(t, previous, body, sig); st != http.StatusNotFound {
		t.Fatalf("delivery to the previous URL: %d", st)
	}
	if code, _ := f.verify(t, previous, a.VerifyToken, "4"); code != http.StatusForbidden {
		t.Fatalf("verification at the previous URL: %d", code)
	}
	if f.stored(t) != 0 {
		t.Fatal("a delivery to the previous URL was stored")
	}
	// Campaigns wait for Meta's verification request, as when the account
	// connected.
	if a := f.overview(t).Account; a.Status != "waiting" || a.WrongAddressDeliveries != 1 {
		t.Fatalf("waiting: %+v", a)
	}
	if code, _ := f.verify(t, callback, a.VerifyToken, "5"); code != http.StatusOK {
		t.Fatalf("verification at the new URL: %d", code)
	}
	if st := f.postTo(t, callback, body, sig); st != http.StatusOK {
		t.Fatalf("delivery to the new URL: %d", st)
	}
	if a := f.overview(t).Account; a.Status != "connected" || a.VerifiedAt == nil {
		t.Fatalf("after Meta verified the new URL: %+v", a)
	}

	// Logged, without either token.
	if n := f.actions(t)["whatsapp.callback_rotated"]; n != 1 {
		t.Fatalf("callback_rotated entries: %d", n)
	}
	for _, path := range []string{previous, callback} {
		token := path[strings.LastIndex(path, "/")+1:]
		if n := one[int64](t, f.env, `SELECT count(*) FROM audit_events e WHERE workspace_id = $1 AND e::text LIKE '%' || $2 || '%'`, f.ws.ID, token); n != 0 {
			t.Fatalf("a callback token is in %d audit entries", n)
		}
	}

	// A viewer can't make one, and neither can a key that manages
	// providers, which never sees the URL either: it's for people to paste
	// in Meta.
	viewer := f.env.CreateUser(apptest.UserOpts{Email: "viewer@x22wa.test", Name: "Vic Viewer"})
	f.env.AddMember(f.ws, viewer.ID, "viewer")
	vc := f.env.Client()
	vc.SignInDemo(viewer.Email)
	vc.Do(http.MethodPost, f.path+"/whatsapp/account/callback-url/rotate", nil).MustStatus(t, 403)
	var key struct {
		Secret string `json:"secret"`
	}
	f.owner.Do(http.MethodPost, f.path+"/api-keys", map[string]any{"name": "Providers", "permissions": []string{"providers:view", "providers:manage"}}).MustStatus(t, 201).JSON(t, &key)
	kc := f.env.Client()
	kc.Bearer = key.Secret
	kr := kc.Do(http.MethodGet, f.path+"/whatsapp", nil).MustStatus(t, 200)
	var seen whatsapp.WhatsAppOverview
	kr.JSON(t, &seen)
	if !seen.CanManage || seen.Account == nil || seen.Account.WebhookURL != "" || seen.WebhookURL != "" || bytes.Contains(kr.Body, []byte(callback[strings.LastIndex(callback, "/")+1:])) {
		t.Fatalf("a key's view: %+v", seen.Account)
	}
	kc.Do(http.MethodPost, f.path+"/whatsapp/account/callback-url/rotate", nil).MustStatus(t, 403)
	if f.overview(t).Account.WebhookURL != a.WebhookURL {
		t.Fatal("a refused request changed the callback URL")
	}
}

// lockedBuffer is a log written by the server's goroutines.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// Neither the request log nor the WhatsApp service's own lines hold a
// callback token, whatever the request: the request log names the
// workspace and {token}.
func TestWhatsAppCallbackTokenStaysOutOfLogs(t *testing.T) {
	f := newFixture(t)
	ov := f.connect(t)
	old := "/hooks/whatsapp/" + f.ws.Slug
	token := strings.TrimPrefix(f.callback, old+"/")

	var logged lockedBuffer
	log := slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc := *f.env.App.WhatsApp()
	svc.Log = log
	r := chi.NewRouter()
	r.Use(server.LogContext, server.RequestLogger(log))
	svc.HookRoutes(r)
	srv := httptest.NewServer(r)
	call := func(method, path string, body []byte, sig string) {
		req, _ := http.NewRequestWithContext(context.Background(), method, srv.URL+path, bytes.NewReader(body))
		if sig != "" {
			req.Header.Set("X-Hub-Signature-256", sig)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
	}
	body, sig := templateDelivery()
	verify := "?hub.mode=subscribe&hub.verify_token=" + ov.Account.VerifyToken + "&hub.challenge=6"
	call(http.MethodGet, f.callback+verify, nil, "")
	call(http.MethodPost, f.callback, body, sig)
	call(http.MethodPost, f.callback, body, meta.Sign("not-the-secret", body))
	call(http.MethodPost, f.callback, body, "")
	call(http.MethodPost, old+"/"+token+"x", body, sig)
	call(http.MethodGet, old+"/"+token+"x"+verify, nil, "")
	call(http.MethodPost, old, body, sig)
	call(http.MethodGet, old+verify, nil, "")
	// Close waits for every request to finish, its log line included.
	srv.Close()

	out := logged.String()
	if strings.Contains(out, token) {
		t.Fatalf("a callback token is in the log:\n%s", out)
	}
	for _, want := range []string{old + "/{token}", "whatsapp webhook address", "whatsapp webhook signature"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the log has no %q:\n%s", want, out)
		}
	}
}
