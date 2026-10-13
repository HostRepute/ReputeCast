package instance_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/instance"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/sso"
)

func setup(t *testing.T) (*apptest.Env, *apptest.Client) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	admin := env.Client()
	admin.SignInDemo("linh@acme.io")
	return env, admin
}

// fakeMail records system mail.
type fakeMail struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *fakeMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *fakeMail) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

// serve runs the instance admin operations with a tweaked service (a fake
// mailer, a release server) on env's database and authenticator.
func serve(t *testing.T, env *apptest.Env, tweak func(*instance.Service)) *apptest.Env {
	t.Helper()
	r := chi.NewRouter()
	r.Use(server.AuditContext)
	h := humachi.New(r, app.NewHumaConfig())
	h.UseMiddleware(env.App.Auth.Middleware(h))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	acc := &accounts.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Cfg: env.App.Cfg, Mail: mailer.Unavailable{}, Log: log, Breach: authn.NewBreachChecker()}
	svc := &instance.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Cfg: env.App.Cfg, Log: log, Accounts: acc,
		SSO: &sso.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Cfg: env.App.Cfg, Log: log, Accounts: acc}, OnPolicyChange: env.App.InvalidatePolicy}
	tweak(svc)
	instance.Register(h, svc)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &apptest.Env{T: t, App: env.App, Server: srv}
}

func count(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func userID(t *testing.T, env *apptest.Env, email string) string {
	t.Helper()
	u, err := env.App.Q.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("user %s: %v", email, err)
	}
	return u.ID.String()
}

type userRow struct {
	ID, Email, Name, Status string
	InstanceAdmin           bool `json:"instanceAdmin"`
	BreakGlass              bool `json:"breakGlass"`
	Workspaces              []struct {
		Slug, Role, Status string
	} `json:"workspaces"`
	Methods     []access.MemberSignInMethod `json:"methods"`
	TwoFactor   access.MemberTwoFA          `json:"twoFactor"`
	SoleOwnerOf []string                    `json:"soleOwnerOf"`
	Self        bool                        `json:"self"`
}

type userList struct {
	Items      []userRow `json:"items"`
	Total      int       `json:"total"`
	AdminNames []string  `json:"adminNames"`
}

func (l userList) find(t *testing.T, email string) userRow {
	t.Helper()
	for _, u := range l.Items {
		if u.Email == email {
			return u
		}
	}
	t.Fatalf("no %s in the list", email)
	return userRow{}
}

func labels(ms []access.MemberSignInMethod) string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Label
	}
	return strings.Join(out, " · ")
}

func TestUsersAcrossWorkspaces(t *testing.T) {
	env, admin := setup(t)
	var l userList
	admin.Do("GET", "/admin/users", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Total != len(l.Items) || l.Total < 7 || strings.Join(l.AdminNames, ",") != "Linh Nguyen" {
		t.Fatalf("list: total %d, %d items, admins %v", l.Total, len(l.Items), l.AdminNames)
	}
	linh := l.find(t, "linh@acme.io")
	if !linh.InstanceAdmin || !linh.BreakGlass || !linh.Self || len(linh.Workspaces) != 2 || linh.TwoFactor.State != "on" || labels(linh.Methods) != "Password · passkey" {
		t.Fatalf("linh: %+v", linh)
	}
	kenji := l.find(t, "kenji@acme.io")
	if strings.Join(kenji.SoleOwnerOf, ",") != "Acme Engineering Blog" || labels(kenji.Methods) != "Google SSO · passkey" || kenji.TwoFactor.Method != "passkey" {
		t.Fatalf("kenji: %+v", kenji)
	}
	if tom := l.find(t, "tom@acme.io"); tom.Status != "deactivated" {
		t.Fatalf("tom: %+v", tom)
	}
	for _, f := range []struct {
		query string
		want  []string
	}{
		{"q=AISHA", []string{"aisha@acme.io"}},
		{"status=deactivated", []string{"tom@acme.io"}},
		{"workspace=acme-engineering-blog", []string{"kenji@acme.io", "linh@acme.io", "marcus@acme.io"}},
	} {
		var fl userList
		admin.Do("GET", "/admin/users?"+f.query, nil).MustStatus(t, 200).JSON(t, &fl)
		got := []string{}
		for _, u := range fl.Items {
			got = append(got, u.Email)
		}
		for _, w := range f.want {
			if !strings.Contains(strings.Join(got, ","), w) {
				t.Fatalf("%s: %v lacks %s", f.query, got, w)
			}
		}
		if len(got) != len(f.want) {
			t.Fatalf("%s: %v", f.query, got)
		}
	}
	// Instance admins only.
	for _, email := range []string{"aisha@acme.io", "marcus@acme.io"} {
		c := env.Client()
		c.SignInDemo(email)
		c.Do("GET", "/admin/users", nil).MustStatus(t, 403)
		c.Do("POST", "/admin/users/"+linh.ID+"/sign-out", map[string]any{}).MustStatus(t, 403)
		c.Do("GET", "/admin/workspaces", nil).MustStatus(t, 403)
		c.Do("PUT", "/admin/policy", map[string]any{}).MustStatus(t, 403)
	}
}

func TestDeactivateSignOutReactivate(t *testing.T) {
	env, admin := setup(t)
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("GET", "/me", nil).MustStatus(t, 200)
	id := userID(t, env, "aisha@acme.io")

	var so struct{ Revoked int }
	admin.Do("POST", "/admin/users/"+id+"/sign-out", map[string]any{}).MustStatus(t, 200).JSON(t, &so)
	if so.Revoked < 1 {
		t.Fatalf("sign-out: %+v", so)
	}
	aisha.Do("GET", "/me", nil).MustStatus(t, 401)

	aisha.SignInDemo("aisha@acme.io")
	admin.Do("POST", "/admin/users/"+id+"/deactivate", map[string]any{"reason": ""}).MustStatus(t, 422)
	var res struct{ Sessions, Keys, Memberships int }
	admin.Do("POST", "/admin/users/"+id+"/deactivate", map[string]any{"reason": "left the company", "revokeKeys": true}).MustStatus(t, 200).JSON(t, &res)
	if res.Sessions < 1 || res.Memberships != 1 {
		t.Fatalf("deactivate: %+v", res)
	}
	aisha.Do("GET", "/me", nil).MustStatus(t, 401)
	env.Client().Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": demo.Password}).MustStatus(t, 403)
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'user.deactivated' AND target_id = $1 AND reason = 'left the company'`, id); n != 1 {
		t.Fatal("deactivation not audited with its reason")
	}
	admin.Do("POST", "/admin/users/"+id+"/deactivate", map[string]any{"reason": "again"}).MustStatus(t, 409)

	admin.Do("POST", "/admin/users/"+id+"/reactivate", nil).MustStatus(t, 204)
	if n := count(t, env, `SELECT count(*) FROM workspace_members WHERE user_id = $1 AND status = 'active'`, id); n != 1 {
		t.Fatal("membership not back")
	}
	env.Client().Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": demo.Password}).MustStatus(t, 200)

	// Never yourself, never a workspace's only Owner.
	admin.Do("POST", "/admin/users/"+userID(t, env, "linh@acme.io")+"/deactivate", map[string]any{"reason": "testing"}).MustStatus(t, 409)
	if p := admin.Do("POST", "/admin/users/"+userID(t, env, "kenji@acme.io")+"/deactivate", map[string]any{"reason": "testing"}).MustStatus(t, 409).Problem(t); p["code"] != "last_owner" {
		t.Fatalf("sole owner: %v", p)
	}
}

func TestResetTwoFactor(t *testing.T) {
	env, admin := setup(t)
	id := userID(t, env, "aisha@acme.io")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE users SET totp_enabled_at = now(), totp_secret_enc = '\x00' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	admin.Do("POST", "/admin/users/"+id+"/two-factor-reset", map[string]any{"reason": "", "password": demo.Password}).MustStatus(t, 422)
	admin.Do("POST", "/admin/users/"+id+"/two-factor-reset", map[string]any{"reason": "lost her phone", "password": "not my password"}).MustStatus(t, 422)
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'user.twofa_reset' AND result = 'denied'`); n != 1 {
		t.Fatal("wrong admin password not audited")
	}
	admin.Do("POST", "/admin/users/"+id+"/two-factor-reset", map[string]any{"reason": "lost her phone", "password": demo.Password}).MustStatus(t, 204)
	if n := count(t, env, `SELECT count(*) FROM users WHERE id = $1 AND totp_enabled_at IS NULL AND twofa_grace_until > now()`, id); n != 1 {
		t.Fatal("authenticator not removed or no grace period")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'user.twofa_reset' AND target_id = $1 AND reason = 'lost her phone' AND result = 'allowed'`, id); n != 1 {
		t.Fatal("reset not audited with its reason")
	}
	if n := count(t, env, `SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'twofa_reset'`, id); n != 1 {
		t.Fatal("no notification")
	}
	admin.Do("POST", "/admin/users/"+id+"/two-factor-reset", map[string]any{"reason": "again", "password": demo.Password}).MustStatus(t, 409)
}

func TestInstanceAdminGrantAndLastAdmin(t *testing.T) {
	env, admin := setup(t)
	linh, aisha := userID(t, env, "linh@acme.io"), userID(t, env, "aisha@acme.io")
	if p := admin.Do("PUT", "/admin/users/"+linh+"/instance-admin", map[string]any{"admin": false}).MustStatus(t, 409).Problem(t); p["code"] != "last_admin" {
		t.Fatalf("last admin: %v", p)
	}
	admin.Do("PUT", "/admin/users/"+aisha+"/instance-admin", map[string]any{"admin": true}).MustStatus(t, 204)
	c := env.Client()
	c.SignInDemo("aisha@acme.io")
	c.Do("GET", "/admin/users", nil).MustStatus(t, 200)
	admin.Do("PUT", "/admin/users/"+aisha+"/instance-admin", map[string]any{"admin": false}).MustStatus(t, 204)
	c.Do("GET", "/admin/users", nil).MustStatus(t, 403)
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action IN ('instance_admin.granted', 'instance_admin.revoked') AND target_id = $1`, aisha); n != 2 {
		t.Fatalf("%d admin changes audited", n)
	}
}

func TestPasswordReset(t *testing.T) {
	env, admin := setup(t)
	id := userID(t, env, "aisha@acme.io")
	var info struct {
		Command      string `json:"command"`
		MinutesValid int    `json:"minutesValid"`
		SSOOnly      bool   `json:"ssoOnly"`
	}
	admin.Do("GET", "/admin/users/"+id+"/password-reset", nil).MustStatus(t, 200).JSON(t, &info)
	if info.Command != "reputecast user reset-password --email aisha@acme.io" || info.MinutesValid != 15 || info.SSOOnly {
		t.Fatalf("info: %+v", info)
	}
	admin.Do("GET", "/admin/users/"+userID(t, env, "marcus@acme.io")+"/password-reset", nil).MustStatus(t, 200).JSON(t, &info)
	if !info.SSOOnly {
		t.Fatalf("marcus signs in with SSO only: %+v", info)
	}

	mail := &fakeMail{}
	e2 := serve(t, env, func(s *instance.Service) { s.Mail = func() mailer.Sender { return mail } })
	a2 := e2.Client()
	a2.SignInDemo("linh@acme.io")
	var sent struct{ Via string }
	a2.Do("POST", "/admin/users/"+id+"/password-reset", nil).MustStatus(t, 200).JSON(t, &sent)
	if sent.Via != "Postmark" || len(mail.sent) != 1 || mail.sent[0].To != "aisha@acme.io" || !strings.Contains(mail.sent[0].Text, "/reset-password/") {
		t.Fatalf("sent: %+v %+v", sent, mail.sent)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'user.password_reset_sent' AND target_id = $1`, id); n != 1 {
		t.Fatal("reset email not audited")
	}
}

func TestWorkspacesCreateDeleteRestorePurge(t *testing.T) {
	env, admin := setup(t)
	var ws struct{ ID, Slug, Name string }
	admin.Do("POST", "/admin/workspaces", map[string]any{"name": "Acme Labs Digest", "ownerId": userID(t, env, "kenji@acme.io")}).MustStatus(t, 201).JSON(t, &ws)
	if ws.Slug != "acme-labs-digest" {
		t.Fatalf("created: %+v", ws)
	}
	if n := count(t, env, `SELECT count(*) FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id WHERE m.workspace_id = $1 AND u.email = 'kenji@acme.io' AND r.key = 'owner'`, ws.ID); n != 1 {
		t.Fatal("kenji isn't the Owner")
	}
	admin.Do("POST", "/admin/workspaces/"+ws.ID+"/delete", map[string]any{"confirm": "acme labs digest"}).MustStatus(t, 422)
	var del struct {
		PurgeAt time.Time `json:"purgeAt"`
	}
	admin.Do("POST", "/admin/workspaces/"+ws.ID+"/delete", map[string]any{"confirm": "Acme Labs Digest"}).MustStatus(t, 200).JSON(t, &del)
	if d := time.Until(del.PurgeAt); d < 6*24*time.Hour || d > 8*24*time.Hour {
		t.Fatalf("purge at %v", del.PurgeAt)
	}
	kenji := env.Client()
	kenji.SignInDemo("kenji@acme.io")
	kenji.Do("GET", "/workspaces/acme-labs-digest/me", nil).MustStatus(t, 404)

	var list struct {
		Items []struct {
			ID        string     `json:"id"`
			DeletedAt *time.Time `json:"deletedAt"`
		} `json:"items"`
	}
	admin.Do("GET", "/admin/workspaces", nil).MustStatus(t, 200).JSON(t, &list)
	found := false
	for _, w := range list.Items {
		if w.ID == ws.ID && w.DeletedAt != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("deleted workspace not listed in its hold")
	}
	admin.Do("POST", "/admin/workspaces/"+ws.ID+"/restore", nil).MustStatus(t, 200).JSON(t, &ws)
	if ws.Slug != "acme-labs-digest" {
		t.Fatalf("restored as %s", ws.Slug)
	}
	kenji.Do("GET", "/workspaces/acme-labs-digest/me", nil).MustStatus(t, 200)

	// The purge job waits for the hold, then deletes everything, even the
	// full demo workspace.
	newsletter := ""
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT id::text FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&newsletter)
	admin.Do("POST", "/admin/workspaces/"+newsletter+"/delete", map[string]any{"confirm": "Acme Newsletter"}).MustStatus(t, 200)
	w := &instance.WorkspacePurgeWorker{Pool: env.App.Pool, Audit: env.App.Audit, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	job := &river.Job[instance.WorkspacePurgeArgs]{Args: instance.WorkspacePurgeArgs{WorkspaceID: uuid.MustParse(newsletter)}}
	if err := w.Work(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if n := count(t, env, `SELECT count(*) FROM workspaces WHERE id = $1`, newsletter); n != 1 {
		t.Fatal("purged during the hold")
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE workspaces SET deleted_at = now() - interval '8 days' WHERE id = $1`, newsletter); err != nil {
		t.Fatal(err)
	}
	if err := w.Work(context.Background(), job); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if n := count(t, env, `SELECT count(*) FROM workspaces WHERE id = $1`, newsletter); n != 0 {
		t.Fatal("not purged")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'workspace.purged' AND target_id = $1`, newsletter); n != 1 {
		t.Fatal("purge not audited")
	}
	admin.Do("POST", "/admin/workspaces/"+newsletter+"/restore", nil).MustStatus(t, 404)
}

func TestPolicy(t *testing.T) {
	env, admin := setup(t)
	var p map[string]any
	admin.Do("GET", "/admin/policy", nil).MustStatus(t, 200).JSON(t, &p)
	body := map[string]any{
		"methods": map[string]any{"password": true, "passkeys": true, "sso": true}, "social": map[string]any{"google": true, "microsoft": false, "github": true},
		"passwordMinLength": 12, "passwordReuse": 5, "breachedCheck": true, "twoFactor": "sensitive", "graceDays": 14,
		"sessionLifetimeHours": 336, "sessionIdleMinutes": 720, "lockoutAttempts": 3, "lockoutMinutes": 15, "jitDomains": []string{"acme.io", "acme-labs.io"},
	}
	admin.Do("PUT", "/admin/policy", body).MustStatus(t, 200).JSON(t, &p)
	if p["graceDays"].(float64) != 14 || p["lockoutAttempts"].(float64) != 3 {
		t.Fatalf("saved: %v", p)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'policy.updated' AND metadata->>'summary' LIKE '%grace 14 days%'`); n != 1 {
		t.Fatal("policy change not audited with a summary")
	}
	// The new lockout applies at once (the cached policy was dropped).
	r := env.Client().Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "wrong password here"}).MustStatus(t, 401).Problem(t)
	if !strings.Contains(r["detail"].(string), "2 attempts left") {
		t.Fatalf("lockout: %v", r)
	}
	// Changes that would lock everyone out are refused.
	for name, mutate := range map[string]func(map[string]any){
		"idle longer than lifetime": func(b map[string]any) { b["sessionLifetimeHours"], b["sessionIdleMinutes"] = 1, 120 },
		"no way to sign in": func(b map[string]any) {
			b["methods"] = map[string]any{"password": false, "passkeys": false, "sso": false}
			b["social"] = map[string]any{}
		},
		"microsoft without a client": func(b map[string]any) { b["social"] = map[string]any{"microsoft": true} },
		"bad JIT domain":             func(b map[string]any) { b["jitDomains"] = []string{"not a domain"} },
	} {
		b := map[string]any{}
		for k, v := range body {
			b[k] = v
		}
		mutate(b)
		if resp := admin.Do("PUT", "/admin/policy", b); resp.Status != 422 {
			t.Fatalf("%s: %d %s", name, resp.Status, resp.Body)
		}
	}
}

func TestInstanceSettingsAndLogo(t *testing.T) {
	env, admin := setup(t)
	admin.Do("PUT", "/admin/instance", map[string]any{"name": "Acme ReputeCast", "publicUrl": "ftp://x", "defaultTimezone": "America/New_York"}).MustStatus(t, 422)
	admin.Do("PUT", "/admin/instance", map[string]any{"name": "Acme ReputeCast", "publicUrl": "https://reputecast.acme.io", "defaultTimezone": "Mars/Olympus"}).MustStatus(t, 422)
	var s struct {
		Name, PublicURL, DefaultTimezone, LogoSVG string
	}
	admin.Do("PUT", "/admin/instance", map[string]any{"name": "Acme Mail", "publicUrl": "https://mail.acme.io/", "defaultTimezone": "Europe/Berlin"}).MustStatus(t, 200).JSON(t, &s)
	if s.Name != "Acme Mail" || s.PublicURL != "https://mail.acme.io" || s.DefaultTimezone != "Europe/Berlin" {
		t.Fatalf("saved: %+v", s)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'instance.settings_updated' AND metadata->>'summary' LIKE '%public URL https://mail.acme.io%'`); n != 1 {
		t.Fatal("settings change not audited")
	}
	// The saved URL replaces http.public_url for every link the app builds
	// (the demo's providers have no real webhook to move).
	if got := env.App.URLs.Report("tok"); got != "https://mail.acme.io/r/tok" {
		t.Fatalf("links after the change: %q", got)
	}
	var moved struct {
		Webhooks *struct {
			Reregistering int
			Manual        []struct{ Provider string }
		}
	}
	admin.Do("PUT", "/admin/instance", map[string]any{"name": "Acme Mail", "publicUrl": "https://news.acme.io", "defaultTimezone": "Europe/Berlin"}).MustStatus(t, 200).JSON(t, &moved)
	if moved.Webhooks == nil || moved.Webhooks.Reregistering != 0 || len(moved.Webhooks.Manual) != 0 || env.App.URLs.Base() != "https://news.acme.io" {
		t.Fatalf("second change: %+v, %q", moved.Webhooks, env.App.URLs.Base())
	}
	// Browser requests from the new address pass the cross-origin check;
	// the address the instance moved away from is no longer trusted.
	for origin, want := range map[string]int{"https://news.acme.io": http.StatusOK, "https://mail.acme.io": http.StatusForbidden} {
		req, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/auth/identify", strings.NewReader(`{"email":"x@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("request from %s: %d, want %d", origin, res.StatusCode, want)
		}
	}
	// SVG logos are checked: no scripts, no external references.
	for _, bad := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><script>alert(1)</script></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" onload="alert(1)"><rect width="10" height="10"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><image href="https://evil.example/x.png"/></svg>`,
	} {
		admin.Do("PUT", "/admin/instance/logo", map[string]any{"data": bad}).MustStatus(t, 422)
	}
	good := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="#2563EB"/></svg>`
	admin.Do("PUT", "/admin/instance/logo", map[string]any{"data": good}).MustStatus(t, 200).JSON(t, &s)
	if !strings.Contains(s.LogoSVG, "#2563EB") {
		t.Fatalf("logo: %q", s.LogoSVG)
	}
	s.LogoSVG = ""
	admin.Do("DELETE", "/admin/instance/logo", nil).MustStatus(t, 200).JSON(t, &s)
	if s.LogoSVG != "" || count(t, env, `SELECT count(*) FROM instance_settings WHERE logo_svg IS NULL`) != 1 {
		t.Fatal("logo not removed")
	}
}

func TestUpdateCheck(t *testing.T) {
	env, _ := setup(t)
	var mu sync.Mutex
	var seen []*http.Request
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Clone(context.Background()))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"tag_name":"v9.1.0","name":"ReputeCast 9.1.0","html_url":"https://github.com/hostrepute/reputecast/releases/tag/v9.1.0",
			"published_at":"2026-10-10T10:00:00Z","body":"## Changes\n- **Faster** imports\n- Fix: [webhooks](https://x) retry\n\nNo downtime."}`)
	}))
	defer gh.Close()
	e2 := serve(t, env, func(s *instance.Service) { s.ReleasesURL = gh.URL + "/repos/hostrepute/reputecast/releases/latest" })
	admin := e2.Client()
	admin.SignInDemo("linh@acme.io")

	var v struct {
		Installed    string `json:"installed"`
		CheckEnabled bool   `json:"checkEnabled"`
		Latest       *struct {
			Tag   string   `json:"tag"`
			Notes []string `json:"notes"`
			Note  string   `json:"note"`
			Demo  bool     `json:"demo"`
		} `json:"latest"`
		UpgradeCommand string `json:"upgradeCommand"`
	}
	admin.Do("GET", "/admin/updates", nil).MustStatus(t, 200).JSON(t, &v)
	if v.Installed == "" || !v.CheckEnabled || v.Latest == nil || !v.Latest.Demo || v.UpgradeCommand == "" {
		t.Fatalf("version: %+v", v)
	}
	// Off: no request at all.
	admin.Do("PUT", "/admin/updates", map[string]any{"enabled": false}).MustStatus(t, 200)
	admin.Do("POST", "/admin/updates/check", nil).MustStatus(t, 409)
	if len(seen) != 0 {
		t.Fatal("a request was made with the check off")
	}
	// On: one plain GET that names nothing but the product.
	admin.Do("PUT", "/admin/updates", map[string]any{"enabled": true}).MustStatus(t, 200)
	admin.Do("POST", "/admin/updates/check", nil).MustStatus(t, 200).JSON(t, &v)
	if v.Latest == nil || v.Latest.Tag != "v9.1.0" || v.Latest.Demo || strings.Join(v.Latest.Notes, "|") != "Faster imports|Fix: webhooks retry" || v.Latest.Note != "No downtime." {
		t.Fatalf("release: %+v", v.Latest)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 {
		t.Fatalf("%d requests", len(seen))
	}
	r := seen[0]
	if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.ContentLength > 0 || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("User-Agent") != "ReputeCast-update-check" {
		t.Fatalf("request: %s %s %v", r.Method, r.URL, r.Header)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'instance.update_check_changed'`); n != 2 {
		t.Fatalf("%d update check changes audited", n)
	}
}
