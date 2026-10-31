package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hostrepute/reputecast/internal/config"
)

// On the main listener /metrics answers only a request with the configured
// bearer token; everyone else gets 404, never the web app behind it.
func TestMetricsOnTheMainListener(t *testing.T) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	const token = "0123456789abcdef-token"
	cases := []struct {
		name   string
		cfg    config.MetricsConfig
		header string
		want   int
	}{
		{"no token configured", config.MetricsConfig{Enabled: true, Addr: ":9464"}, "", http.StatusNotFound},
		{"no token configured, one sent", config.MetricsConfig{Enabled: true}, "Bearer " + token, http.StatusNotFound},
		{"missing", config.MetricsConfig{Enabled: true, Token: token}, "", http.StatusNotFound},
		{"wrong", config.MetricsConfig{Enabled: true, Token: token}, "Bearer " + token + "x", http.StatusNotFound},
		{"basic auth", config.MetricsConfig{Enabled: true, Token: token}, "Basic " + token, http.StatusNotFound},
		{"disabled", config.MetricsConfig{Enabled: false, Token: token}, "Bearer " + token, http.StatusNotFound},
		{"right", config.MetricsConfig{Enabled: true, Token: token}, "Bearer " + token, http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := chi.NewRouter()
			mountMetrics(r, c.cfg, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
			// The web app answers every other path with 200.
			r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<!doctype html>")) }))
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("status %d, want %d", rec.Code, c.want)
			}
			if c.want == http.StatusOK && !strings.Contains(rec.Body.String(), "go_goroutines") {
				t.Fatalf("body has no metrics: %.200s", rec.Body.String())
			}
		})
	}
}
