package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/hostrepute/reputecast/internal/logging"
)

func TestContextHandlerAddsRequestFields(t *testing.T) {
	var buf bytes.Buffer
	log := logging.New(&buf, "debug", "json")
	route := "/api/v1/workspaces/{workspace}/campaigns"
	ctx := logging.WithRequest(context.Background(), "req-1", func() string { return route })
	logging.SetWorkspace(ctx, "acme-newsletter")
	logging.SetCaller(ctx, "user:0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44")

	// A service's line, through a logger with attributes of its own.
	log.With("component", "campaigns").WarnContext(ctx, "send held", "campaign", "c1")
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"request_id": "req-1", "route": route, "workspace": "acme-newsletter",
		"caller": "user:0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44", "component": "campaigns", "campaign": "c1"}
	for k, v := range want {
		if line[k] != v {
			t.Errorf("%s = %v, want %v (line %s)", k, line[k], v, buf.String())
		}
	}

	// Lines without a request context carry nothing extra; a job's say so.
	buf.Reset()
	log.Info("started")
	if bytes.Contains(buf.Bytes(), []byte("request_id")) || bytes.Contains(buf.Bytes(), []byte("workspace")) {
		t.Fatalf("a line outside a request: %s", buf.String())
	}
	buf.Reset()
	log.InfoContext(logging.WithJob(context.Background(), "ingest.normalize:42"), "normalized")
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil || line["job"] != "ingest.normalize:42" {
		t.Fatalf("a job's line: %s", buf.String())
	}
}

func TestEmptyFieldsAreLeftOut(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(logging.ContextHandler{Handler: slog.NewTextHandler(&buf, nil)})
	ctx := logging.WithRequest(context.Background(), "req-2", nil)
	log.InfoContext(ctx, "public page")
	if got := buf.String(); !bytes.Contains([]byte(got), []byte("request_id=req-2")) || bytes.Contains([]byte(got), []byte("caller=")) || bytes.Contains([]byte(got), []byte("route=")) {
		t.Fatalf("line: %s", got)
	}
}
