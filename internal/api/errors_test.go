package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

// An unexpected error answers 500 with a reference; its text, which can name
// tables or hosts, is only in the log, under that reference.
func TestUnexpectedErrorsStayInTheLog(t *testing.T) {
	var logged bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logged, nil))
	_, h := humatest.New(t)
	h.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(hctx, context.WithValue(hctx.Context(), logKey, log)))
	})
	huma.Get(h, "/boom", func(context.Context, *struct{}) (*struct{}, error) {
		return nil, errors.New(`ERROR: relation "secret_table" does not exist (SQLSTATE 42P01)`)
	})
	huma.Get(h, "/busy", func(context.Context, *struct{}) (*struct{}, error) {
		return nil, Err(http.StatusServiceUnavailable, CodeUnavailable, "Try again in a minute.")
	})

	resp := h.Get("/boom")
	if resp.Code != http.StatusInternalServerError || strings.Contains(resp.Body.String(), "secret_table") {
		t.Fatalf("response: %d %s", resp.Code, resp.Body.String())
	}
	var p Problem
	if err := json.Unmarshal(resp.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	ref, _ := p.Meta["reference"].(string)
	if p.Code != CodeInternal || !strings.HasPrefix(ref, "err_") || len(p.Errors) != 0 {
		t.Fatalf("problem: %+v", p)
	}
	if line := logged.String(); !strings.Contains(line, ref) || !strings.Contains(line, "secret_table") {
		t.Fatalf("log: %s", line)
	}

	// A problem a handler chose keeps its own words.
	resp = h.Get("/busy")
	var busy Problem
	if err := json.Unmarshal(resp.Body.Bytes(), &busy); err != nil {
		t.Fatal(err)
	}
	if resp.Code != http.StatusServiceUnavailable || busy.Code != CodeUnavailable || busy.Detail != "Try again in a minute." || busy.Meta["reference"] != nil {
		t.Fatalf("chosen problem: %d %+v", resp.Code, busy)
	}
}
