package app

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hostrepute/reputecast/internal/config"
)

// mountMetrics answers /metrics on the main listener. The route is always
// mounted so the request never falls through to the web app (which answers
// every path); it serves the metrics only to a request that carries the
// configured bearer token, and 404 to everyone else.
func mountMetrics(r chi.Router, cfg config.MetricsConfig, metrics http.Handler) {
	want := sha256.Sum256([]byte(cfg.Token))
	r.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !cfg.Enabled || cfg.Token == "" || !bearerMatches(req, want) {
			http.NotFound(w, req)
			return
		}
		metrics.ServeHTTP(w, req)
	}))
}

// bearerMatches compares the request's bearer token with want (a SHA-256)
// in constant time.
func bearerMatches(req *http.Request, want [sha256.Size]byte) bool {
	h := req.Header.Get("Authorization")
	tok, ok := strings.CutPrefix(h, "Bearer ")
	if !ok || tok == "" {
		return false
	}
	got := sha256.Sum256([]byte(tok))
	return subtle.ConstantTimeCompare(got[:], want[:]) == 1
}

// serveMetrics starts the metrics listener when one is configured. A
// listener that can't bind (another ReputeCast process on the host already
// serves the port) is logged and skipped: metrics never stop the process.
func (a *App) serveMetrics() *http.Server {
	m := a.Cfg.Metrics
	if !m.Enabled {
		return nil
	}
	if m.Token != "" {
		a.Log.Info("metrics on the main listener", "auth", "bearer")
	}
	if m.Addr == "" {
		if m.Token == "" {
			a.Log.Warn("metrics are enabled but not served: set metrics.addr or metrics.token")
		}
		return nil
	}
	ln, err := net.Listen("tcp", m.Addr)
	if err != nil {
		a.Log.Error("metrics listener not started", "addr", m.Addr, "err", err)
		return nil
	}
	srv := &http.Server{Handler: promhttp.HandlerFor(a.Metrics, promhttp.HandlerOpts{}), ReadHeaderTimeout: 5 * time.Second}
	a.Log.Info("metrics listening", "addr", ln.Addr().String())
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Log.Error("metrics listener stopped", "err", err)
		}
	}()
	return srv
}

// shutdownMetrics stops the metrics listener, if one runs.
func shutdownMetrics(ctx context.Context, srv *http.Server) {
	if srv != nil {
		_ = srv.Shutdown(ctx)
	}
}
