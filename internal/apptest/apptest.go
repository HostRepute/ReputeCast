// Package apptest runs the whole HTTP stack against a fresh test database so
// API behaviour can be tested end to end: middleware, authz, handlers, SQL.
package apptest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/demo"
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

// testKey seals secrets in every test app, the same in every test binary: a
// run's binaries share the seeded demo template (testdb.Shared), and each one
// opens what the seed sealed. It only ever seals test data.
var testKey = func() string {
	sum := sha256.Sum256([]byte("reputecast test secrets"))
	return base64.StdEncoding.EncodeToString(sum[:])
}()

// sample is seeded data shared by a `go test` run's binaries
// (testdb.Shared) and cloned for each test.
type sample struct {
	mu    sync.Mutex
	day   string // sampleDay when it was seeded
	name  string
	files string // its files (digital products, a lead magnet); each clone gets a copy
	err   error
}

var demoSample, northwindSample sample

// The time zones of the samples' workspaces, where their numbers count
// whole days (Growth's last 30 days, a form's stats).
var (
	acmeZones      = zones("America/New_York")
	northwindZones = zones("America/Chicago", "America/Denver", "America/Los_Angeles", "America/New_York")
)

func zones(names ...string) []*time.Location {
	out := make([]*time.Location, len(names))
	for i, n := range names {
		loc, err := time.LoadLocation(n)
		if err != nil {
			panic(err)
		}
		out[i] = loc
	}
	return out
}

// sampleDay names the last midnight to pass in any of zones (minutes since
// the epoch): two moments share it unless a day began between them in one
// of the zones.
func sampleDay(now time.Time, zones []*time.Location) string {
	var last time.Time
	for _, z := range zones {
		l := now.In(z)
		if m := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, z); m.After(last) {
			last = m
		}
	}
	return strconv.FormatInt(last.Unix()/60, 10)
}

// template returns the sample as seeded today in zones. A sample seeded
// before a midnight there shows its days moved by one, so a new day seeds
// it again, and so does a seed that ran past midnight.
func (s *sample) template(ctx context.Context, key string, zones []*time.Location, fill func(pool *pgxpool.Pool, dir string) error) (name, files string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		day := sampleDay(time.Now(), zones)
		if day == s.day {
			return s.name, s.files, s.err
		}
		s.day = day
		s.name, s.files, s.err = testdb.Shared(ctx, key+day, fill)
	}
}

// New starts the API on a fresh database.
func New(t testing.TB) *Env {
	t.Helper()
	e := &Env{T: t}
	e.start(testdb.New(t))
	t.Cleanup(e.stop)
	return e
}

// start runs the app and an HTTP server on pool's database.
func (e *Env) start(pool *pgxpool.Pool) {
	t := e.T
	cfg := config.Defaults()
	cfg.Env = "test"
	cfg.Database.URL = testdb.URL(pool)
	cfg.Roles = []string{config.RoleWeb, config.RoleIngest}
	cfg.Secrets.Key = testKey
	cfg.Auth.CookieSecure = false
	cfg.HTTP.RateLimitAuth = 10000
	cfg.HTTP.RateLimitAPI = 100000
	// Fake provider APIs run on loopback in tests.
	cfg.Providers.AllowPrivateNetworks = true
	// Packages test in parallel against one server: keep each app's pool small.
	cfg.Database.MaxConns = 6
	// Uploaded files (digital products, lead magnets) go to the test's own folder.
	cfg.Files.Dir = t.TempDir()
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
	a.PublicURL.Set(srv.URL)
	e.App, e.Server = a, srv
}

func (e *Env) stop() {
	if e.Server != nil {
		e.Server.Close()
	}
	if e.App != nil {
		e.App.Close()
	}
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
	// Header is added to every request (Accept-Language…).
	Header http.Header
}

// requestTimeout fails a request that hangs. A Send Readiness run takes a
// few seconds here under -race, and more than 30 on a busy CI runner.
const requestTimeout = 2 * time.Minute

// Client returns a new client with an empty cookie jar.
func (e *Env) Client() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{env: e, http: &http.Client{Jar: jar, Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
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
	req, err := http.NewRequestWithContext(context.Background(), method, url, rd)
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
	for k, vs := range c.Header {
		req.Header[k] = vs
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.env.T.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
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
	ctx := context.Background()
	// The key tells the demo without transactional mail apart.
	key := "demo"
	if demo.SkipTransactional {
		key = "demonotx"
	}
	name, files, err := demoSample.template(ctx, key, acmeZones, func(pool *pgxpool.Pool, dir string) error {
		cfg := *e.App.Cfg
		cfg.Files.Dir = dir
		_, err := demo.Seed(ctx, pool, &cfg, e.App.Secrets, slog.New(slog.NewTextHandler(io.Discard, nil)))
		return err
	})
	if err != nil {
		e.T.Fatalf("seed demo: %v", err)
	}
	// Restart on a clone of the seeded database: milliseconds instead of a
	// full seed per test.
	e.stop()
	e.start(testdb.NewFrom(e.T, name))
	// Cleanups run last first: this app stops before its clone is dropped.
	e.T.Cleanup(e.stop)
	// The sample files, in this app's own folder.
	if err := os.CopyFS(e.App.Cfg.FilesDir(), os.DirFS(files)); err != nil {
		e.T.Fatalf("demo files: %v", err)
	}
}

// SeedNorthwind loads the agency sample (Northwind Studio and its seven
// clients) into the test database instead of the Acme demo.
func (e *Env) SeedNorthwind() {
	e.T.Helper()
	ctx := context.Background()
	name, files, err := northwindSample.template(ctx, "northwind", northwindZones, func(pool *pgxpool.Pool, dir string) error {
		cfg := *e.App.Cfg
		cfg.Files.Dir = dir
		// A fortieth of the contacts: the same sample, cloned in moments.
		_, err := demo.SeedNorthwindWith(ctx, pool, &cfg, e.App.Secrets, slog.New(slog.NewTextHandler(io.Discard, nil)), demo.NorthwindOptions{ContactsDivisor: 40})
		return err
	})
	if err != nil {
		e.T.Fatalf("seed northwind: %v", err)
	}
	e.stop()
	e.start(testdb.NewFrom(e.T, name))
	if err := os.CopyFS(e.App.Cfg.FilesDir(), os.DirFS(files)); err != nil {
		e.T.Fatalf("northwind files: %v", err)
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
	c.http.Jar.SetCookies(u2, []*http.Cookie{{Name: c.env.App.Cfg.Auth.CookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode}}) //nolint:gosec // test server runs on plain HTTP
}

// DoRaw sends a request with a raw body and content type (multipart
// uploads) to path under /api/v1, or to a non-API path as is ("/media/…").
func (c *Client) DoRaw(method, path, contentType string, body io.Reader) *Response {
	c.env.T.Helper()
	target := c.env.Server.URL + path
	if strings.HasPrefix(path, "/workspaces/") {
		target = c.env.Server.URL + "/api/v1" + path
	}
	req, err := http.NewRequestWithContext(context.Background(), method, target, body)
	if err != nil {
		c.env.T.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	resp, err := c.http.Do(req)
	if err != nil {
		c.env.T.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return &Response{Status: resp.StatusCode, Header: resp.Header, Body: b}
}
