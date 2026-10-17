package accounts_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
)

func TestAccountOverviewDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var a map[string]any
	c.Do("GET", "/me/account", nil).MustStatus(t, 200).JSON(t, &a)

	if p := a["profile"].(map[string]any); p["name"] != "Linh Nguyen" || p["timezone"] != "America/New_York" || p["isInstanceAdmin"] != true {
		t.Fatalf("profile: %v", p)
	}
	if pw := a["password"].(map[string]any); pw["set"] != true || pw["ssoOnly"] != false || pw["minLength"].(float64) != 12 {
		t.Fatalf("password: %v", pw)
	}
	keys := a["passkeys"].([]any)
	if len(keys) != 3 || keys[0].(map[string]any)["name"] != "MacBook Pro · Touch ID" || keys[0].(map[string]any)["provider"] != "iCloud Keychain" ||
		keys[2].(map[string]any)["kind"] != "security_key" || keys[2].(map[string]any)["lastUsedAt"] != nil {
		t.Fatalf("passkeys: %v", keys)
	}
	tf := a["twoFactor"].(map[string]any)
	if tf["totp"] != true || tf["totpLabel"] != "1Password" || tf["required"] != true || tf["recoveryRemaining"].(float64) != 7 || tf["recoveryTotal"].(float64) != 10 || tf["recoveryLastUsedAt"] == nil {
		t.Fatalf("two-factor: %v", tf)
	}
	ids := a["identities"].([]any)
	byName := map[string]map[string]any{}
	for _, i := range ids {
		m := i.(map[string]any)
		byName[m["name"].(string)] = m
	}
	if g := byName["GitHub"]; g == nil || g["linked"] != true || g["username"] != "linh-nguyen" || g["notes"].([]any)[0] != "org acme-eng" || g["canUnlink"] != true {
		t.Fatalf("GitHub: %v", ids)
	}
	if g := byName["Google"]; g == nil || g["linked"] != false || !strings.Contains(g["linkUrl"].(string), "/auth/social/google/start") {
		t.Fatalf("Google: %v", ids)
	}
	if s := byName["Google Workspace SSO"]; s == nil || s["linked"] != true || s["email"] != "linh@acme.io" {
		t.Fatalf("SSO: %v", ids)
	}
	wss := a["workspaces"].([]any)
	if len(wss) != 2 || wss[0].(map[string]any)["canLeave"] != false || !strings.Contains(wss[0].(map[string]any)["leaveReason"].(string), "last Owner") {
		t.Fatalf("workspaces: %v", wss)
	}
	n := a["notifications"].(map[string]any)
	if n["inApp"] != "All events" || n["email"] != "Approvals, auto-pauses, provider down, HostRepute alerts, integration problems, post drafts, transactional delivery alerts, SMTP relay outages, held digest issues, paused digests, failing digest feeds, digest drafts, payment problems" || n["quietHours"] != "22:00–07:00 America/New_York, email only" {
		t.Fatalf("notifications: %v", n)
	}
	if sp := a["sessionPolicy"].(map[string]any); sp["lifetime"] != "14 days" || sp["idle"] != "12 hours" {
		t.Fatalf("session policy: %v", sp)
	}

	// Sofia signs in only through Google Workspace: no password section, her
	// one sign-in method can't be unlinked, two-factor comes from the IdP.
	s := env.Client()
	s.SignInDemo("sofia@acme.io")
	s.Do("GET", "/me/account", nil).MustStatus(t, 200).JSON(t, &a)
	if pw := a["password"].(map[string]any); pw["ssoOnly"] != true || pw["managedBy"] != "Google Workspace" {
		t.Fatalf("SSO-only password: %v", pw)
	}
	for _, i := range a["identities"].([]any) {
		m := i.(map[string]any)
		if m["kind"] == "sso" && (m["canUnlink"] != false || m["onlyMethod"] != true) {
			t.Fatalf("the only method can't be unlinked: %v", m)
		}
	}
	if a["twoFactor"].(map[string]any)["viaIdp"] != true {
		t.Fatalf("2FA via IdP: %v", a["twoFactor"])
	}
	for _, i := range a["identities"].([]any) {
		if m := i.(map[string]any); m["kind"] == "sso" {
			p := s.Do("DELETE", "/me/identities/"+m["id"].(string), nil).MustStatus(t, 409).Problem(t)
			if p["code"] != "last_method" {
				t.Fatalf("unlink last method: %v", p)
			}
		}
	}
}

func TestProfileAvatarAndSessions(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh"})
	ws := env.CreateWorkspace("Acme Newsletter", u.ID)
	c := env.Client()
	c.SignIn("linh@acme.io", "")

	c.Do("PATCH", "/me/profile", map[string]any{"name": "Linh Nguyen", "timezone": "Mars/Olympus"}).MustStatus(t, 422)
	var p map[string]any
	c.Do("PATCH", "/me/profile", map[string]any{"name": "Linh Nguyen", "timezone": "America/New_York"}).MustStatus(t, 200).JSON(t, &p)
	if p["name"] != "Linh Nguyen" || p["timezone"] != "America/New_York" {
		t.Fatalf("profile: %v", p)
	}

	// A photo is re-encoded as a 256 px PNG and served to workspace members.
	img := image.NewRGBA(image.Rect(0, 0, 600, 400))
	for x := 0; x < 600; x++ {
		for y := 0; y < 400; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	c.Do("PUT", "/me/avatar", map[string]any{"data": []byte("not an image")}).MustStatus(t, 422)
	c.Do("PUT", "/me/avatar", map[string]any{"data": buf.Bytes()}).MustStatus(t, 200).JSON(t, &p)
	url, _ := p["avatarUrl"].(string)
	if !strings.HasPrefix(url, "/api/v1/users/"+u.ID.String()+"/avatar") {
		t.Fatalf("avatar url: %v", p)
	}
	r := c.Do("GET", url, nil).MustStatus(t, 200)
	if r.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("content type %q", r.Header.Get("Content-Type"))
	}
	got, err := png.DecodeConfig(bytes.NewReader(r.Body))
	if err != nil || got.Width != 256 || got.Height != 256 {
		t.Fatalf("thumbnail %v %v", got, err)
	}
	// Someone who shares no workspace can't fetch it.
	env.CreateUser(apptest.UserOpts{Email: "kim@example.com"})
	stranger := env.Client()
	stranger.SignIn("kim@example.com", "")
	stranger.Do("GET", url, nil).MustStatus(t, 404)
	c.Do("DELETE", "/me/avatar", nil).MustStatus(t, 204)
	c.Do("GET", url, nil).MustStatus(t, 404)

	// Sessions: this one first; others can be signed out one by one or all.
	second := env.Client()
	second.SignIn("linh@acme.io", "")
	third := env.Client()
	third.SignIn("linh@acme.io", "")
	var sl map[string]any
	c.Do("GET", "/me/sessions", nil).MustStatus(t, 200).JSON(t, &sl)
	items := sl["items"].([]any)
	if len(items) != 3 || items[0].(map[string]any)["current"] != true {
		t.Fatalf("sessions: %v", items)
	}
	current := items[0].(map[string]any)["id"].(string)
	c.Do("DELETE", "/me/sessions/"+current, nil).MustStatus(t, 409)
	c.Do("DELETE", "/me/sessions/"+items[1].(map[string]any)["id"].(string), nil).MustStatus(t, 204)
	var revoked map[string]any
	c.Do("POST", "/me/sessions/revoke-others", nil).MustStatus(t, 200).JSON(t, &revoked)
	if revoked["revoked"].(float64) != 1 {
		t.Fatalf("revoke others: %v", revoked)
	}
	second.Do("GET", "/me", nil).MustStatus(t, 401)
	third.Do("GET", "/me", nil).MustStatus(t, 401)
	c.Do("GET", "/me", nil).MustStatus(t, 200)

	// API keys can't reach personal settings.
	key, hash, masked := authn.NewAPIKey(authn.APIKeyPrefix)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO api_keys (id, workspace_id, name, prefix, token_hash, permissions, created_by) VALUES ($1, $2, 'k', $3, $4, ARRAY['reports:view'], $5)`,
		uuid.Must(uuid.NewV7()), ws.ID, masked, hash, u.ID); err != nil {
		t.Fatal(err)
	}
	k := env.Client()
	k.Bearer = key
	k.Do("GET", "/me/account", nil).MustStatus(t, 403)

	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('account.profile_update','account.avatar_update','account.avatar_remove','account.session_revoke','account.sessions_revoke_others')`).Scan(&n)
	if n != 5 {
		t.Fatalf("audit entries: %d", n)
	}
}

func TestEmailChange(t *testing.T) {
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	env.CreateWorkspace("Acme Newsletter", u.ID)
	env.CreateUser(apptest.UserOpts{Email: "taken@acme.io"})

	// Without a System provider the verification email can't go out.
	c := env.Client()
	c.SignIn("linh@acme.io", "")
	p := c.Do("POST", "/me/email", map[string]any{"email": "linh.nguyen@acme.io", "password": "correct horse battery staple"}).MustStatus(t, 503).Problem(t)
	if p["code"] != "provider_unavailable" || !strings.Contains(p["detail"].(string), "linh.nguyen@acme.io couldn't be sent") {
		t.Fatalf("no provider: %v", p)
	}

	mail := &fakeMail{available: true}
	menv := withMail(t, env, mail)
	m := menv.Client()
	m.SignIn("linh@acme.io", "")
	m.Do("POST", "/me/email", map[string]any{"email": "linh.nguyen@acme.io", "password": "wrong"}).MustStatus(t, 422)
	m.Do("POST", "/me/email", map[string]any{"email": "taken@acme.io", "password": "correct horse battery staple"}).MustStatus(t, 422)
	m.Do("POST", "/me/email", map[string]any{"email": "Linh.Nguyen@acme.io", "password": "correct horse battery staple"}).MustStatus(t, 200)
	msg := mail.last(t)
	if msg.To != "linh.nguyen@acme.io" {
		t.Fatalf("verification goes to the new address: %+v", msg)
	}
	token := linkToken(t, msg.Text, "?verifyEmail=")
	var a map[string]any
	m.Do("GET", "/me/account", nil).MustStatus(t, 200).JSON(t, &a)
	if pe := a["profile"].(map[string]any)["pendingEmail"]; pe == nil || pe.(map[string]any)["email"] != "linh.nguyen@acme.io" {
		t.Fatalf("pending email: %v", a["profile"])
	}
	// Only the same account can confirm.
	env.CreateUser(apptest.UserOpts{Email: "kim@example.com"})
	other := menv.Client()
	other.SignIn("kim@example.com", "")
	other.Do("POST", "/me/email/verify", map[string]any{"token": token}).MustStatus(t, 410)
	var done map[string]any
	m.Do("POST", "/me/email/verify", map[string]any{"token": token}).MustStatus(t, 200).JSON(t, &done)
	if done["email"] != "linh.nguyen@acme.io" {
		t.Fatalf("changed: %v", done)
	}
	if last := mail.last(t); last.To != "linh@acme.io" || !strings.Contains(last.Text, "now signs in with linh.nguyen@acme.io") {
		t.Fatalf("the old address is told: %+v", last)
	}
	env.Client().SignIn("linh.nguyen@acme.io", "")
}

func TestLeaveWorkspace(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io"})
	marcus := env.CreateUser(apptest.UserOpts{Email: "marcus@acme.io"})
	ws := env.CreateWorkspace("Acme Engineering Blog", linh.ID)
	env.AddMember(ws, marcus.ID, "admin")
	key, hash, masked := authn.NewAPIKey(authn.APIKeyPrefix)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO api_keys (id, workspace_id, name, prefix, token_hash, permissions, created_by) VALUES ($1, $2, 'k', $3, $4, ARRAY['reports:view'], $5)`,
		uuid.Must(uuid.NewV7()), ws.ID, masked, hash, marcus.ID); err != nil {
		t.Fatal(err)
	}

	c := env.Client()
	c.SignIn("linh@acme.io", "")
	if p := c.Do("POST", "/workspaces/"+ws.Slug+"/leave", nil).MustStatus(t, 409).Problem(t); p["code"] != "last_owner" {
		t.Fatalf("last owner: %v", p)
	}
	m := env.Client()
	m.SignIn("marcus@acme.io", "")
	m.Do("POST", "/workspaces/"+ws.Slug+"/leave", nil).MustStatus(t, 204)
	m.Do("GET", "/workspaces/"+ws.Slug+"/me", nil).MustStatus(t, 404)
	k := env.Client()
	k.Bearer = key
	k.Do("GET", "/workspaces/"+ws.Slug+"/me", nil).MustStatus(t, 401)
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'member.leave' AND workspace_id = $1`, ws.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("audit: %d", n)
	}
}

func TestSetupFlow(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Password: "temp-pass-1234", InstanceAdmin: true, MustChange: true})
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET setup_completed_at = NULL, basics_confirmed_at = NULL`); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignIn("linh@acme.io", "temp-pass-1234")
	var st map[string]any
	c.Do("GET", "/setup", nil).MustStatus(t, 200).JSON(t, &st)
	if st["mode"] != "instance" || st["current"] != "password" || len(st["steps"].([]any)) != 6 {
		t.Fatalf("setup: %v", st)
	}
	// The first workspace waits for the earlier steps.
	if p := c.Do("POST", "/workspaces", map[string]any{"name": "Acme Newsletter"}).MustStatus(t, 409).Problem(t); p["code"] != "setup_incomplete" {
		t.Fatalf("workspace before setup: %v", p)
	}
	c.Do("POST", "/me/password", map[string]any{"current": "temp-pass-1234", "password": "a much longer passphrase"}).MustStatus(t, 204)
	c.Do("GET", "/setup", nil).MustStatus(t, 200).JSON(t, &st)
	if st["current"] != "second_factor" {
		t.Fatalf("after password: %v", st["current"])
	}
	var totpSetup map[string]any
	c.Do("POST", "/me/totp", nil).MustStatus(t, 200).JSON(t, &totpSetup)
	code := totpCode(t, totpSetup["secret"].(string))
	c.Do("POST", "/me/totp/confirm", map[string]any{"challenge": totpSetup["challenge"], "code": code}).MustStatus(t, 200)
	c.Do("GET", "/setup", nil).MustStatus(t, 200).JSON(t, &st)
	if st["current"] != "recovery_codes" {
		t.Fatalf("after authenticator: %v", st["current"])
	}
	c.Do("POST", "/me/recovery-codes/saved", nil).MustStatus(t, 204)
	c.Do("GET", "/setup", nil).MustStatus(t, 200).JSON(t, &st)
	if st["current"] != "instance" {
		t.Fatalf("after codes: %v", st["current"])
	}
	c.Do("PUT", "/setup/instance", map[string]any{"name": "Acme ReputeCast", "publicUrl": "https://reputecast.acme.io", "timezone": "America/New_York", "updateCheck": true}).MustStatus(t, 204)
	// Links built from then on use the address just saved.
	if got := env.App.URLs.Base(); got != "https://reputecast.acme.io" {
		t.Fatalf("public URL after setup: %q", got)
	}
	var ws map[string]any
	c.Do("POST", "/workspaces", map[string]any{"name": "Acme Newsletter"}).MustStatus(t, 201).JSON(t, &ws)
	c.Do("GET", "/setup", nil).MustStatus(t, 200).JSON(t, &st)
	if st["mode"] != "done" || st["next"] != "/w/"+ws["slug"].(string) {
		t.Fatalf("setup done: %v", st)
	}
	var me map[string]any
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if me["pending"] != nil || me["instance"].(map[string]any)["setupComplete"] != true {
		t.Fatalf("me after setup: %v", me)
	}
}
