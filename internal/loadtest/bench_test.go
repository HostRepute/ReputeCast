package loadtest_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/loadtest"
	"github.com/hostrepute/reputecast/internal/store"
)

// The hot queries on the load test data set (review 2026-11 OP-9, AU-17).
// They run only against a database seeded with
//
//	reputecast seed && reputecast seed --sample load
//
// named by REPUTECAST_LOAD_DATABASE_URL; `make loadtest` and the manual
// "Load test" workflow do it. Numbers are in docs/deploy/performance.md.

type harness struct {
	app      *app.App
	srv      *httptest.Server
	cookie   string
	ws       store.Workspace
	campaign string
	page400  string
}

var (
	once  sync.Once
	h     *harness
	hErr  error
	lgr   = slog.New(slog.NewTextHandler(io.Discard, nil))
	dbURL = os.Getenv("REPUTECAST_LOAD_DATABASE_URL")
)

func setup(tb testing.TB) *harness {
	tb.Helper()
	if dbURL == "" {
		tb.Skip("set REPUTECAST_LOAD_DATABASE_URL to a database seeded with `reputecast seed --sample load`")
	}
	once.Do(func() { h, hErr = build() })
	if hErr != nil {
		tb.Fatal(hErr)
	}
	return h
}

func build() (*harness, error) {
	ctx := context.Background()
	cfg, err := config.Load("", nil)
	if err != nil {
		return nil, err
	}
	cfg.Database.URL = dbURL
	cfg.Database.MaxConns = 20
	cfg.HTTP.RateLimitAPI = 0 // measure the queries, not the limiter
	cfg.Roles = []string{config.RoleWeb}
	if cfg.Secrets.Key == "" {
		k := make([]byte, 32)
		_, _ = rand.Read(k)
		cfg.Secrets.Key = base64.StdEncoding.EncodeToString(k)
	}
	a, err := app.New(ctx, cfg, lgr)
	if err != nil {
		return nil, err
	}
	out := &harness{app: a, srv: httptest.NewServer(a.Handler())}
	if err := a.Pool.QueryRow(ctx, `SELECT id, slug FROM workspaces WHERE slug = $1`, loadtest.Slug).Scan(&out.ws.ID, &out.ws.Slug); err != nil {
		return nil, fmt.Errorf("no %s workspace: %w", loadtest.Slug, err)
	}
	// A session for the workspace's owner, as if they had signed in.
	var owner uuid.UUID
	if err := a.Pool.QueryRow(ctx, `SELECT m.user_id FROM workspace_members m JOIN roles r ON r.id = m.role_id WHERE m.workspace_id = $1 AND r.key = 'owner' LIMIT 1`, out.ws.ID).Scan(&owner); err != nil {
		return nil, err
	}
	token := uuid.NewString()
	if _, err := a.Pool.Exec(ctx, `INSERT INTO sessions (id, user_id, token_hash, auth_method, mfa_verified, created_at, last_seen_at, expires_at)
		VALUES ($1, $2, $3, 'password', true, now(), now(), now() + interval '1 day')`, uuid.Must(uuid.NewV7()), owner, authn.HashToken(token)); err != nil {
		return nil, err
	}
	out.cookie = cfg.Auth.CookieName + "=" + token
	if err := a.Pool.QueryRow(ctx, `SELECT id::text FROM campaigns WHERE workspace_id = $1 ORDER BY started_at DESC LIMIT 1`, out.ws.ID).Scan(&out.campaign); err != nil {
		return nil, err
	}
	// The cursor of page 400 (25 a page), walked once.
	cursor := ""
	for page := 1; page < 400; page++ {
		var list struct{ NextCursor string }
		if err := out.get("/contacts?pageSize=25&cursor="+url.QueryEscape(cursor), &list); err != nil {
			return nil, err
		}
		cursor = list.NextCursor
	}
	out.page400 = cursor
	return out, nil
}

func (h *harness) do(method, path string, body any, into any) error {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = strings.NewReader(string(b))
	}
	req, _ := http.NewRequest(method, h.srv.URL+"/api/v1/workspaces/"+h.ws.Slug+path, r)
	req.Header.Set("Cookie", h.cookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", h.srv.URL)
	resp, err := h.srv.Client().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, b)
	}
	if into != nil {
		return json.Unmarshal(b, into)
	}
	return nil
}

func (h *harness) get(path string, into any) error { return h.do(http.MethodGet, path, nil, into) }

// The calls measured, each a request through the whole stack.
func calls(h *harness) map[string]func() error {
	segment := map[string]any{"mode": "builder", "definition": map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{
		{"field": "email_domain", "op": "is", "value": "gmail.com"},
		{"field": "tier", "op": "is", "value": "active"},
	}}}}}
	return map[string]func() error{
		"contacts page 1":        func() error { return h.get("/contacts?pageSize=25", nil) },
		"contacts page 400":      func() error { return h.get("/contacts?pageSize=25&cursor="+url.QueryEscape(h.page400), nil) },
		"contacts search":        func() error { return h.get("/contacts?pageSize=25&q=load4242", nil) },
		"contacts exact email":   func() error { return h.get("/contacts?pageSize=25&q="+url.QueryEscape("load424242@gmail.com"), nil) },
		"contacts by list":       func() error { return h.get("/contacts?pageSize=25&mailbox=yahoo", nil) },
		"segment count":          func() error { return h.do(http.MethodPost, "/segments/preview", segment, nil) },
		"audience summary":       func() error { return h.get("/audience", nil) },
		"campaign report":        func() error { return h.get("/campaigns/"+h.campaign+"/report", nil) },
		"campaigns list":         func() error { return h.get("/campaigns", nil) },
		"overview sender health": func() error { return h.get("/overview/sender-health", nil) },
	}
}

// TestAudienceAtOneMillion holds AU-1 and AU-2 to their targets: the
// contacts list's first page and page 400 under 200 ms, a search under
// 150 ms (the median of 15 runs, after one warm-up).
func TestAudienceAtOneMillion(t *testing.T) {
	h := setup(t)
	c := calls(h)
	targets := map[string]time.Duration{
		"contacts page 1": 200 * time.Millisecond, "contacts page 400": 200 * time.Millisecond,
		"contacts search": 150 * time.Millisecond, "contacts exact email": 150 * time.Millisecond,
	}
	for name, limit := range targets {
		median, err := measure(c[name], 15)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%-22s median %v (target %v)", name, median.Round(time.Millisecond), limit)
		if median > limit {
			t.Errorf("%s: median %v, over %v", name, median, limit)
		}
	}
}

// TestReportTimes logs every call's median, for docs/deploy/performance.md.
func TestReportTimes(t *testing.T) {
	h := setup(t)
	c := calls(h)
	names := make([]string, 0, len(c))
	for n := range c {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		median, err := measure(c[n], 9)
		if err != nil {
			t.Errorf("%s: %v", n, err)
			continue
		}
		t.Logf("| %s | %v |", n, median.Round(time.Millisecond))
	}
}

func measure(fn func() error, runs int) (time.Duration, error) {
	if err := fn(); err != nil { // warm-up
		return 0, err
	}
	took := make([]time.Duration, 0, runs)
	for range runs {
		start := time.Now()
		if err := fn(); err != nil {
			return 0, err
		}
		took = append(took, time.Since(start))
	}
	slices.Sort(took)
	return took[len(took)/2], nil
}

func BenchmarkCalls(b *testing.B) {
	h := setup(b)
	for name, fn := range calls(h) {
		b.Run(strings.ReplaceAll(name, " ", "_"), func(b *testing.B) {
			for b.Loop() {
				if err := fn(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkIngestInsert stores provider events, each with its own dedupe
// key, as the webhook normalizer does.
func BenchmarkIngestInsert(b *testing.B) {
	h := setup(b)
	ctx := context.Background()
	var provider uuid.UUID
	if err := h.app.Pool.QueryRow(ctx, `SELECT id FROM providers ORDER BY id LIMIT 1`).Scan(&provider); err != nil {
		b.Skip("no provider to store events for")
	}
	q := store.New(h.app.Pool)
	i := 0
	for b.Loop() {
		i++
		now := time.Now()
		if _, err := q.IngestInsertEvent(ctx, store.IngestInsertEventParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: h.ws.ID, ProviderID: &provider,
			Type: "delivered", OccurredAt: now, MailboxProvider: "gmail", ReceivedAt: now, DedupeKey: fmt.Sprintf("bench-%d-%d", now.UnixNano(), i)}); err != nil {
			b.Fatal(err)
		}
	}
}

// TestConcurrentBrowse runs ten people browsing and searching the contacts
// at once for 15 seconds, and logs the throughput and the 95th percentile,
// as k6's contacts.js does against a running server.
func TestConcurrentBrowse(t *testing.T) {
	h := setup(t)
	c := calls(h)
	mix := []string{"contacts page 1", "contacts page 400", "contacts search", "contacts exact email", "contacts by list"}
	const workers = 10
	deadline := time.Now().Add(15 * time.Second)
	var mu sync.Mutex
	var took []time.Duration
	var failed int
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := w; time.Now().Before(deadline); i++ {
				start := time.Now()
				err := c[mix[i%len(mix)]]()
				d := time.Since(start)
				mu.Lock()
				if err != nil {
					failed++
				} else {
					took = append(took, d)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	slices.Sort(took)
	if len(took) == 0 {
		t.Fatal("no request went through")
	}
	p := func(q float64) time.Duration { return took[int(float64(len(took)-1)*q)].Round(time.Millisecond) }
	t.Logf("%d workers: %d requests in 15 s (%.0f/s), p50 %v, p95 %v, p99 %v, %d failed",
		workers, len(took), float64(len(took))/15, p(0.5), p(0.95), p(0.99), failed)
	if failed > 0 || p(0.95) > 200*time.Millisecond {
		t.Errorf("p95 %v with %d failures", p(0.95), failed)
	}
}
