// Package app wires configuration, the database, background jobs, the HTTP
// API and every domain package into one runnable process.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
	"github.com/hostrepute/reputecast/web"
)

// App holds the shared dependencies every domain uses.
type App struct {
	Cfg     *config.Config
	Log     *slog.Logger
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Secrets *secrets.Box
	Audit   *audit.Logger
	Jobs    *jobs.Client
	JobReg  *jobs.Registry
	Auth    *api.Authenticator
	Metrics *prometheus.Registry
	// Mail sends system mail through the System-stream provider.
	Mail mailer.Sender

	httpMetrics *server.HTTPMetrics

	policyMu    sync.Mutex
	policy      api.Policy
	policyUntil time.Time
}

// New connects to the database and builds every dependency. It does not
// start listeners or workers; see Run.
func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*App, error) {
	pool, err := db.Open(ctx, cfg.Database.URL, cfg.Database.MaxConns)
	if err != nil {
		return nil, err
	}
	box, err := SecretBox(cfg, log)
	if err != nil {
		pool.Close()
		return nil, err
	}
	q := store.New(pool)
	a := &App{
		Cfg:     cfg,
		Log:     log,
		Pool:    pool,
		Q:       q,
		Secrets: box,
		Audit:   audit.New(q, log),
		JobReg:  jobs.NewRegistry(),
		Metrics: prometheus.NewRegistry(),
		Mail:    mailer.Unavailable{},
	}
	a.Metrics.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	a.Auth = &api.Authenticator{
		Q:          q,
		Audit:      a.Audit,
		Log:        log,
		CookieName: cfg.Auth.CookieName,
		Policy:     a.authPolicy,
	}
	registerJobs(a)
	for _, hook := range startHooks {
		if err := hook(a); err != nil {
			pool.Close()
			return nil, err
		}
	}
	client, err := jobs.NewClient(pool, a.JobReg, jobs.Options{
		Run:         cfg.HasRole(config.RoleWorker),
		Concurrency: cfg.Worker.Concurrency,
		Logger:      log,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("jobs: %w", err)
	}
	a.Jobs = client
	return a, nil
}

// SecretBox loads the encryption key (from config or the key file) and
// returns the box that seals secrets at rest.
func SecretBox(cfg *config.Config, log *slog.Logger) (*secrets.Box, error) {
	keyStr := cfg.Secrets.Key
	if keyStr == "" {
		k, created, err := secrets.LoadOrCreateKeyFile(cfg.Secrets.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("secret key file %s: %w", cfg.Secrets.KeyFile, err)
		}
		if created {
			log.Warn("generated a new secret key; back it up together with the database", "file", cfg.Secrets.KeyFile)
		}
		keyStr = k
	}
	cur, err := config.DecodeKey(keyStr)
	if err != nil {
		return nil, fmt.Errorf("secret key: %w", err)
	}
	var prev [][]byte
	for _, p := range cfg.Secrets.PreviousKeys {
		k, err := config.DecodeKey(p)
		if err != nil {
			return nil, fmt.Errorf("previous secret key: %w", err)
		}
		prev = append(prev, k)
	}
	return secrets.New(cur, prev...)
}

// authPolicy returns the instance authentication policy, cached for 30 s.
func (a *App) authPolicy(ctx context.Context) (api.Policy, error) {
	a.policyMu.Lock()
	defer a.policyMu.Unlock()
	if time.Now().Before(a.policyUntil) {
		return a.policy, nil
	}
	s, err := a.Q.GetInstanceSettings(ctx)
	if err != nil {
		return api.Policy{SessionIdle: a.Cfg.Auth.SessionIdle, TwoFAEnforcement: "sensitive"}, err
	}
	a.policy = api.Policy{SessionIdle: s.SessionIdle, TwoFAEnforcement: s.TwofaEnforcement}
	a.policyUntil = time.Now().Add(30 * time.Second)
	return a.policy, nil
}

// InvalidatePolicy drops the cached authentication policy after a change.
func (a *App) InvalidatePolicy() {
	a.policyMu.Lock()
	a.policyUntil = time.Time{}
	a.policyMu.Unlock()
}

// Close releases the database pool.
func (a *App) Close() { a.Pool.Close() }

// NewHumaConfig returns the OpenAPI configuration shared by the server and
// the `reputecast openapi` command.
func NewHumaConfig() huma.Config {
	cfg := huma.DefaultConfig("ReputeCast API", version.Get().Version)
	cfg.Info.Description = "The ReputeCast HTTP API. Every operation declares the permission it needs " +
		"(x-permission); users, API keys and MCP agents go through the same check. Errors use RFC 9457 problem details."
	cfg.Info.License = &huma.License{Name: "AGPL-3.0", URL: "https://www.gnu.org/licenses/agpl-3.0.html"}
	cfg.OpenAPIPath = "/api/openapi"
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	cfg.CreateHooks = nil // no $schema links in bodies
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {Type: "apiKey", In: "cookie", Name: "rc_session", Description: "Browser session cookie"},
		"bearer":  {Type: "http", Scheme: "bearer", Description: "API key (rc_live_…) or MCP agent token (rc_agent_…)"},
	}
	return cfg
}

// Handler builds the full HTTP handler for the configured roles.
func (a *App) Handler() http.Handler {
	r := chi.NewRouter()
	if a.httpMetrics == nil {
		a.httpMetrics = server.NewHTTPMetrics(a.Metrics)
	}
	r.Use(a.httpMetrics.Middleware)
	r.Use(middleware.RequestID)
	r.Use(server.RealIP(trustedProxies(a.Cfg.HTTP.TrustedProxies, a.Log)))
	r.Use(server.RequestLogger(a.Log))
	r.Use(middleware.Recoverer)
	r.Use(server.AuditContext)
	r.Use(server.SecurityHeaders(strings.HasPrefix(a.Cfg.HTTP.PublicURL, "https://")))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := a.Pool.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ready"))
	})
	if a.Cfg.Metrics.Enabled && a.Cfg.Metrics.Addr == "" {
		r.Handle("/metrics", promhttp.HandlerFor(a.Metrics, promhttp.HandlerOpts{}))
	}

	// Cross-origin protection (Go 1.25+): rejects cross-site browser requests
	// that change state. Provider webhooks and public form posts come from
	// other origins by design and are verified differently.
	cop := http.NewCrossOriginProtection()
	cop.AddInsecureBypassPattern("POST /hooks/")
	cop.AddInsecureBypassPattern("POST /p/")
	if u, err := url.Parse(a.Cfg.HTTP.PublicURL); err == nil {
		_ = cop.AddTrustedOrigin(u.Scheme + "://" + u.Host)
	}

	r.Group(func(r chi.Router) {
		r.Use(cop.Handler)
		if a.Cfg.HTTP.RateLimitAPI > 0 {
			r.Use(server.RateLimits{AuthPerMinute: a.Cfg.HTTP.RateLimitAuth, APIPerMinute: a.Cfg.HTTP.RateLimitAPI}.Middleware())
		}
		r.Use(middleware.Compress(5, "application/json", "application/problem+json", "text/html", "text/css", "text/plain", "application/javascript", "image/svg+xml"))
		if a.Cfg.HasRole(config.RoleWeb) {
			hapi := humachi.New(r, NewHumaConfig())
			hapi.UseMiddleware(a.Auth.Middleware(hapi))
			registerAPI(a, hapi)
			r.Handle("/api/*", http.HandlerFunc(apiNotFound))
			registerPublic(a, r)
		}
		if a.Cfg.HasRole(config.RoleIngest) {
			registerIngest(a, r)
		}
	})

	if a.Cfg.HasRole(config.RoleWeb) {
		r.Handle("/*", server.NewSPA(web.Dist()))
	}
	return r
}

// apiNotFound answers unknown /api paths with a problem document instead of
// falling through to the web app.
func apiNotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"status":404,"title":"Not Found","detail":"No such API endpoint.","code":"not_found"}`))
}

func trustedProxies(cidrs []string, log *slog.Logger) []netip.Prefix {
	var out []netip.Prefix
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			if ip, err := netip.ParseAddr(c); err == nil {
				out = append(out, netip.PrefixFrom(ip, ip.BitLen()))
				continue
			}
		}
		p, err := netip.ParsePrefix(c)
		if err != nil {
			log.Warn("ignoring invalid trusted proxy", "value", c)
			continue
		}
		out = append(out, p)
	}
	return out
}

// Run serves HTTP (web/ingest roles) and runs workers (worker role) until ctx ends.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 3)
	var srv *http.Server
	if a.Cfg.HasRole(config.RoleWeb) || a.Cfg.HasRole(config.RoleIngest) {
		srv = &http.Server{
			Addr:              a.Cfg.HTTP.Addr,
			Handler:           a.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       a.Cfg.HTTP.ReadTimeout,
			WriteTimeout:      a.Cfg.HTTP.WriteTimeout,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
		}
		go func() {
			a.Log.Info("http listening", "addr", a.Cfg.HTTP.Addr, "public_url", a.Cfg.HTTP.PublicURL, "roles", a.Cfg.Roles)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("http: %w", err)
			}
		}()
	}
	var metricsSrv *http.Server
	if a.Cfg.Metrics.Enabled && a.Cfg.Metrics.Addr != "" {
		metricsSrv = &http.Server{Addr: a.Cfg.Metrics.Addr, Handler: promhttp.HandlerFor(a.Metrics, promhttp.HandlerOpts{}), ReadHeaderTimeout: 5 * time.Second}
		go func() {
			if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("metrics: %w", err)
			}
		}()
	}
	if a.Cfg.HasRole(config.RoleWorker) {
		if err := a.Jobs.Start(ctx); err != nil {
			return fmt.Errorf("start workers: %w", err)
		}
		a.Log.Info("workers started", "concurrency", a.Cfg.Worker.Concurrency)
	}

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errCh:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.Cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if srv != nil {
		_ = srv.Shutdown(shutdownCtx)
	}
	if metricsSrv != nil {
		_ = metricsSrv.Shutdown(shutdownCtx)
	}
	if a.Cfg.HasRole(config.RoleWorker) {
		if err := a.Jobs.Stop(shutdownCtx); err != nil {
			a.Log.Warn("workers did not stop cleanly", "err", err)
		}
	}
	a.Log.Info("stopped")
	return runErr
}
