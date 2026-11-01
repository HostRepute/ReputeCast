package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hostrepute/reputecast/internal/logging"
)

// Every line a request logs, the access line included, names its request,
// route, workspace and caller (review 2026-11 OP-7).
func TestRequestLinesCarryTheRequest(t *testing.T) {
	var buf bytes.Buffer
	log := logging.New(&buf, "info", "json")
	r := chi.NewRouter()
	r.Use(middleware.RequestID, LogContext, RequestLogger(log))
	r.Get("/api/v1/workspaces/{workspace}/campaigns/{id}", func(w http.ResponseWriter, req *http.Request) {
		// What the API's authentication does once it knows the caller.
		logging.SetWorkspace(req.Context(), chi.URLParam(req, "workspace"))
		logging.SetCaller(req.Context(), "key:k1")
		log.WarnContext(req.Context(), "campaign held")
		w.WriteHeader(http.StatusConflict)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/acme/campaigns/c1", nil)
	req.Header.Set("X-Request-Id", "abc")
	r.ServeHTTP(httptest.NewRecorder(), req)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines: %q", lines)
	}
	for _, l := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatal(err)
		}
		if m["request_id"] == "" || m["request_id"] == nil || m["route"] != "/api/v1/workspaces/{workspace}/campaigns/{id}" || m["workspace"] != "acme" || m["caller"] != "key:k1" {
			t.Errorf("line without its request: %s", l)
		}
	}
	if !strings.Contains(lines[1], `"msg":"http"`) || strings.Count(lines[1], "request_id") != 1 {
		t.Errorf("access line: %s", lines[1])
	}
}
