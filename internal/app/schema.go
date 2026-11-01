package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/version"
)

// A binary never writes to a database whose schema is behind it (review
// 2026-11 OP-3). Until `reputecast migrate up` (or another process's
// --auto-migrate) applies the missing migrations, the web app serves
// read-only, showing the maintenance banner; workers wait; /readyz answers
// 503 so an orchestrator keeps sending traffic to the previous release.

// schemaPoll is how often a process waiting for migrations looks again.
var schemaPoll = 5 * time.Second

// schemaGate is a process's view of the schema.
type schemaGate struct {
	behind atomic.Bool
	since  atomic.Int64 // unix seconds the process started waiting
}

// readOnly reports whether writes are refused because migrations are pending.
func (a *App) readOnly() bool { return a.schema.behind.Load() }

// watchSchema checks the schema once; when it is behind, the process turns
// read-only and looks again every few seconds. The returned channel closes
// once the schema is current (at once when it already is).
func (a *App) watchSchema(ctx context.Context) <-chan struct{} {
	ready := make(chan struct{})
	st, err := db.Schema(ctx, a.Pool)
	if err == nil && st.Current() {
		a.warnNewer(st)
		close(ready)
		return ready
	}
	a.schema.behind.Store(true)
	a.schema.since.Store(time.Now().Unix())
	if err != nil {
		a.Log.Error("couldn't read the database schema; serving read-only until it can", "err", err)
	} else {
		a.Log.Error("the database schema is behind this release: serving read-only and holding workers until `reputecast migrate up` applies the missing migrations",
			"pending", len(st.Pending), "applied", st.Applied, "expected", st.Expected, "river_current", st.RiverCurrent)
	}
	go func() {
		t := time.NewTicker(schemaPoll)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			st, err := db.Schema(ctx, a.Pool)
			if err != nil || !st.Current() {
				continue
			}
			a.schema.behind.Store(false)
			a.Log.Info("the database schema is current: writes and workers resume", "version", st.Applied)
			close(ready)
			return
		}
	}()
	return ready
}

// warnNewer says when a newer release has migrated the database: this
// binary keeps working (migrations are additive within a minor release),
// but it is a downgrade.
func (a *App) warnNewer(st db.SchemaState) {
	if st.Newer {
		a.Log.Warn("the database has migrations from a newer release; downgrades are supported one minor release back (docs/deploy/backup-restore.md)",
			"applied", st.Applied, "expected", st.Expected)
	}
}

// readOnlyGuard refuses writes while the schema is behind. Reads go through
// with a header the web app shows the maintenance banner from; provider
// webhooks are still stored (their normalizing waits for the workers), and
// people can still sign in and out.
func (a *App) readOnlyGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.readOnly() {
			next.ServeHTTP(w, r)
			return
		}
		since := time.Unix(a.schema.since.Load(), 0).UTC().Format(time.RFC3339)
		w.Header().Set("X-ReputeCast-Read-Only", "upgrade")
		w.Header().Set("X-ReputeCast-Upgrade-Version", version.Get().Version)
		w.Header().Set("X-ReputeCast-Upgrade-Since", since)
		switch {
		case r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions,
			strings.HasPrefix(r.URL.Path, "/hooks/"),
			strings.HasPrefix(r.URL.Path, "/api/v1/auth/"):
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Retry-After", strconv.Itoa(int(schemaPoll.Seconds())*6))
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("ReputeCast is being upgraded. Please try again in a few minutes.\n"))
			return
		}
		body, _ := json.Marshal(map[string]any{
			"status": http.StatusServiceUnavailable,
			"title":  "Service Unavailable",
			"code":   "maintenance",
			"detail": "ReputeCast is being upgraded: reading works; edits and sends resume when it finishes.",
			"meta":   map[string]any{"version": version.Get().Version, "startedAt": since},
		})
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(body)
	})
}
