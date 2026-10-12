// Package apptest runs the whole HTTP stack against a fresh test database so
// API behaviour can be tested end to end: middleware, authz, handlers, SQL.
package apptest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// Env is a running test instance.
type Env struct {
	T      testing.TB
	App    *app.App
	Server *httptest.Server
}

// New starts the API on a fresh database.
func New(t testing.TB) *Env {
	t.Helper()
	pool := testdb.New(t)
	cfg := config.Defaults()
	cfg.Env = "test"
	cfg.Database.URL = testdb.URL(pool)
	cfg.Roles = []string{config.RoleWeb, config.RoleIngest}
	cfg.Secrets.Key = secrets.GenerateKey()
	cfg.Auth.CookieSecure = false
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if testing.Verbose() {
		log = slog.New(slog.NewTextHandler(&testWriter{t}, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	a, err := app.New(context.Background(), &cfg, log)
	if err != nil {
		t.Fatalf("app: %v", err)
	}
	srv := httptest.NewServer(a.Handler())
	a.Cfg.HTTP.PublicURL = srv.URL
	_, _ = a.Pool.Exec(context.Background(), `UPDATE instance_settings SET public_url = $1`, srv.URL)
	t.Cleanup(func() {
		srv.Close()
		a.Close()
	})
	return &Env{T: t, App: a, Server: srv}
}

type testWriter struct{ t testing.TB }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// Client is a browser-like client with its own cookie jar.
type Client struct {
	env  *Env
	http *http.Client
	// Bearer is sent as Authorization when set (API keys, agents).
	Bearer string
}

// Client returns a new client with an empty cookie jar.
func (e *Env) Client() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{env: e, http: &http.Client{Jar: jar, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Response is a decoded API response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// JSON decodes the body into v.
func (r *Response) JSON(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

// Problem decodes an error body.
func (r *Response) Problem(t testing.TB) map[string]any {
	t.Helper()
	var m map[string]any
	r.JSON(t, &m)
	return m
}

// Do sends a request with a JSON body (nil for none) to path under /api/v1.
func (c *Client) Do(method, path string, body any) *Response {
	c.env.T.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.env.T.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	url := c.env.Server.URL + path
	if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/p/") && !strings.HasPrefix(path, "/hooks/") {
		url = c.env.Server.URL + "/api/v1" + path
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		c.env.T.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	// Browsers send these on same-origin fetches; cross-origin protection reads them.
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if c.Bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.Bearer)
		req.Header.Del("Sec-Fetch-Site")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.env.T.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return &Response{Status: resp.StatusCode, Header: resp.Header, Body: b}
}

// MustStatus fails the test unless the response has the given status.
func (r *Response) MustStatus(t testing.TB, want int) *Response {
	t.Helper()
	if r.Status != want {
		t.Fatalf("status %d, want %d: %s", r.Status, want, r.Body)
	}
	return r
}

// UserOpts describes a test user.
type UserOpts struct {
	Email, Name, Password string
	InstanceAdmin         bool
	MustChange            bool
}

// CreateUser inserts a user directly.
func (e *Env) CreateUser(o UserOpts) store.User {
	e.T.Helper()
	if o.Password == "" {
		o.Password = "correct horse battery staple"
	}
	h, err := authn.HashPassword(o.Password)
	if err != nil {
		e.T.Fatal(err)
	}
	now := time.Now()
	u, err := e.App.Q.CreateUser(context.Background(), store.CreateUserParams{
		ID: uuid.Must(uuid.NewV7()), Email: strings.ToLower(o.Email), Name: o.Name, Timezone: "UTC",
		PasswordHash: &h, PasswordSetAt: &now, MustChangePassword: o.MustChange, IsInstanceAdmin: o.InstanceAdmin, AvatarColor: "info",
	})
	if err != nil {
		e.T.Fatal(err)
	}
	return u
}

// CreateWorkspace creates a workspace owned by owner.
func (e *Env) CreateWorkspace(name string, owner uuid.UUID) store.Workspace {
	e.T.Helper()
	var ws store.Workspace
	err := db.Tx(context.Background(), e.App.Pool, func(tx pgx.Tx) error {
		var err error
		ws, err = workspaces.Create(context.Background(), tx, workspaces.CreateParams{Name: name, Owner: owner})
		return err
	})
	if err != nil {
		e.T.Fatal(err)
	}
	return ws
}

// AddMember adds user to ws with a built-in role key.
func (e *Env) AddMember(ws store.Workspace, user uuid.UUID, roleKey string) {
	e.T.Helper()
	ctx := context.Background()
	role, err := e.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws.ID, Key: roleKey})
	if err != nil {
		e.T.Fatal(err)
	}
	if _, err := e.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, UserID: user, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		e.T.Fatal(err)
	}
}

// SignIn signs a client in with a password (no second factor).
func (c *Client) SignIn(email, password string) {
	c.env.T.Helper()
	if password == "" {
		password = "correct horse battery staple"
	}
	r := c.Do("POST", "/auth/password", map[string]any{"email": email, "password": password})
	r.MustStatus(c.env.T, 200)
}

// SeedDemo loads the Acme demo data into the test database.
func (e *Env) SeedDemo() {
	e.T.Helper()
	if _, err := demo.Seed(context.Background(), e.App.Pool, e.App.Cfg, e.App.Secrets, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		e.T.Fatalf("seed demo: %v", err)
	}
}

// SignInDemo signs in as a demo user. Demo users with an authenticator get
// a session directly (tests don't type TOTP codes).
func (c *Client) SignInDemo(email string) {
	c.env.T.Helper()
	ctx := context.Background()
	u, err := c.env.App.Q.GetUserByEmail(ctx, email)
	if err != nil {
		c.env.T.Fatalf("demo user %s: %v", email, err)
	}
	token, hash := authn.NewToken()
	if _, err := c.env.App.Q.CreateSession(ctx, store.CreateSessionParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, TokenHash: hash, AuthMethod: "test", MfaVerified: true, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		c.env.T.Fatal(err)
	}
	u2, _ := url.Parse(c.env.Server.URL)
	c.http.Jar.SetCookies(u2, []*http.Cookie{{Name: c.env.App.Cfg.Auth.CookieName, Value: token, Path: "/"}})
}
