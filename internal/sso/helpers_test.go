package sso_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/sso"
	"github.com/hostrepute/reputecast/internal/sso/ssotest"
	"github.com/hostrepute/reputecast/internal/store"
)

// setup starts the API with the Acme demo data and an instance admin (Linh).
func setup(t *testing.T) (*apptest.Env, *apptest.Client) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	admin := env.Client()
	admin.SignInDemo("linh@acme.io")
	return env, admin
}

func workspaceID(t *testing.T, env *apptest.Env, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM workspaces WHERE slug = $1`, slug).Scan(&id); err != nil {
		t.Fatalf("workspace %s: %v", slug, err)
	}
	return id
}

// detail is the part of SSOProviderDetail the tests read.
type detail struct {
	ID        string   `json:"id"`
	Slug      string   `json:"slug"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Issuer    string   `json:"issuer"`
	Enforced  []string `json:"enforced"`
	ClientID  string   `json:"clientId"`
	Discovery *struct {
		OK        bool     `json:"ok"`
		URL       string   `json:"url"`
		Status    int      `json:"status"`
		Error     string   `json:"error"`
		Keys      int      `json:"keys"`
		Endpoints []string `json:"endpoints"`
	} `json:"discovery"`
	Secret struct {
		Set bool `json:"set"`
	} `json:"secret"`
	Test *struct {
		Error         string `json:"error"`
		Claims        string `json:"claims"`
		GroupsMissing bool   `json:"groupsMissing"`
		Outcome       []struct {
			Tone, Label, Detail string
		} `json:"outcome"`
		Samples []struct {
			Field, Claim, Value, Missing, Note string
			Found                              bool
		} `json:"samples"`
	} `json:"test"`
	EnableBlockers []string `json:"enableBlockers"`
	Rejections     []struct {
		Email, Reason, Reference string
	} `json:"rejections"`
	BreakGlassList []struct {
		Name, Email string
	} `json:"breakGlassList"`
	Workspaces []struct {
		Slug  string `json:"slug"`
		Roles []struct {
			Key       string `json:"key"`
			Grantable bool   `json:"grantable"`
			Reason    string `json:"reason"`
		} `json:"roles"`
	} `json:"workspaces"`
}

func (d detail) outcome(label string) string {
	var out []string
	for _, o := range d.Test.Outcome {
		if o.Label == label {
			out = append(out, o.Tone+": "+o.Detail)
		}
	}
	return strings.Join(out, " | ")
}

// providerBody is a complete PUT /admin/sso/{id} body for a fake issuer:
// example.com people, JIT into Acme Newsletter as Viewer, amr two-factor.
func providerBody(env *apptest.Env, t *testing.T, idp *ssotest.IdP) map[string]any {
	return map[string]any{
		"name": "Test IdP", "issuerUrl": idp.Issuer(), "clientId": idp.ClientID, "clientSecret": idp.ClientSecret,
		"scopes":         []string{"openid", "profile", "email", "groups"},
		"claims":         map[string]any{"email": "email", "emailVerified": "email_verified", "name": "name", "groups": "groups"},
		"allowedDomains": []string{"example.com"},
		"jit":            map[string]any{"enabled": true, "workspaceId": workspaceID(t, env, "acme-newsletter").String(), "roleKey": "viewer"},
		"groupMappings":  []any{},
		"twoFactorMode":  "amr",
	}
}

// addProvider creates a generic OIDC provider for the fake issuer, runs
// discovery and enables it. mutate adjusts the PUT body.
func addProvider(t *testing.T, env *apptest.Env, admin *apptest.Client, idp *ssotest.IdP, mutate func(map[string]any)) detail {
	t.Helper()
	var d detail
	admin.Do("POST", "/admin/sso", map[string]any{"preset": "oidc"}).MustStatus(t, 201).JSON(t, &d)
	if d.Status != "draft" || len(d.EnableBlockers) != 3 {
		t.Fatalf("new provider: %+v", d)
	}
	body := providerBody(env, t, idp)
	if mutate != nil {
		mutate(body)
	}
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{}).MustStatus(t, 200).JSON(t, &d)
	if d.Discovery == nil || !d.Discovery.OK {
		t.Fatalf("discovery: %+v", d.Discovery)
	}
	admin.Do("POST", "/admin/sso/"+d.ID+"/enable", nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "enabled" {
		t.Fatalf("enable: %+v", d)
	}
	return d
}

// browser is a person's browser: a cookie jar, and redirects followed by hand.
type browser struct {
	t    *testing.T
	env  *apptest.Env
	http *http.Client
	// Visited are the URLs the browser was sent to.
	Visited []string
}

func newBrowser(t *testing.T, env *apptest.Env) *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{t: t, env: env, http: &http.Client{Jar: jar, Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (b *browser) abs(u string) string {
	if strings.HasPrefix(u, "/") {
		return b.env.Server.URL + u
	}
	return u
}

// visit is what loading a URL returned.
type visit struct {
	Status   int
	Location string
}

// get loads a URL without following redirects.
func (b *browser) get(u string) visit {
	b.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, b.abs(u), nil)
	if err != nil {
		b.t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	resp, err := b.http.Do(req)
	if err != nil {
		b.t.Fatalf("GET %s: %v", u, err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return visit{Status: resp.StatusCode, Location: resp.Header.Get("Location")}
}

// do sends an API request with a JSON body and decodes the answer into out.
func (b *browser) do(method, path string, body, out any) int {
	b.t.Helper()
	var rd io.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		rd = bytes.NewReader(buf)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, b.abs(path), rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	resp, err := b.http.Do(req)
	if err != nil {
		b.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			b.t.Fatalf("decode %s: %v", raw, err)
		}
	}
	return resp.StatusCode
}

// follow starts at a URL and follows the redirects through the identity
// provider and the callback until the app sends the browser to one of its
// pages. It returns that page's path and query.
func (b *browser) follow(start string) string {
	b.t.Helper()
	loc := start
	for range 8 {
		v := b.get(loc)
		if v.Status != http.StatusFound {
			b.t.Fatalf("GET %s: status %d, want a redirect", loc, v.Status)
		}
		next := v.Location
		b.Visited = append(b.Visited, next)
		if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "/api/") && !strings.HasPrefix(next, "/auth/") {
			return next
		}
		loc = next
	}
	b.t.Fatalf("too many redirects from %s: %v", start, b.Visited)
	return ""
}

// me returns the signed-in person's email, or "" when signed out.
func (b *browser) me() string {
	b.t.Helper()
	var m struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if b.do("GET", "/api/v1/me", nil, &m) != 200 {
		return ""
	}
	return m.User.Email
}

// signInAs gives the browser a session for an existing person.
func (b *browser) signInAs(email string) {
	b.t.Helper()
	ctx := context.Background()
	u, err := b.env.App.Q.GetUserByEmail(ctx, email)
	if err != nil {
		b.t.Fatalf("user %s: %v", email, err)
	}
	token, hash := authn.NewToken()
	if _, err := b.env.App.Q.CreateSession(ctx, store.CreateSessionParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, TokenHash: hash, AuthMethod: "test", MfaVerified: true, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		b.t.Fatal(err)
	}
	su, _ := url.Parse(b.env.Server.URL)
	b.http.Jar.SetCookies(su, []*http.Cookie{{Name: b.env.App.Cfg.Auth.CookieName, Value: token, Path: "/", HttpOnly: true}}) //nolint:gosec // test server runs on plain HTTP
}

// setCookies stores the cookies of a response (the test sign-in's binding cookie).
func (b *browser) setCookies(h http.Header) {
	u, _ := url.Parse(b.env.Server.URL)
	b.http.Jar.SetCookies(u, (&http.Response{Header: h}).Cookies())
}

func query(t *testing.T, loc string) url.Values {
	t.Helper()
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query()
}

func count(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// socialEndpoints points the social providers at fake servers for one test.
func socialEndpoints(t *testing.T, e sso.SocialEndpoints) {
	old := sso.DefaultSocialEndpoints
	sso.DefaultSocialEndpoints = e
	t.Cleanup(func() { sso.DefaultSocialEndpoints = old })
}

// invite stores a pending invitation and returns its link token.
func invite(t *testing.T, env *apptest.Env, slug, email, role string) string {
	t.Helper()
	ctx := context.Background()
	ws := workspaceID(t, env, slug)
	r, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws, Key: role})
	if err != nil {
		t.Fatal(err)
	}
	linh, err := env.App.Q.GetUserByEmail(ctx, "linh@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	token, hash := authn.NewToken()
	now := time.Now()
	if _, err := env.App.Q.CreateInvitation(ctx, store.CreateInvitationParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, Email: email, RoleID: r.ID, Scope: json.RawMessage(`{}`), TokenHash: hash,
		InvitedBy: &linh.ID, LastSentAt: &now, ExpiresAt: now.Add(72 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	return token
}
