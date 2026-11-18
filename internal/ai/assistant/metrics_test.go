package assistant_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/hostrepute/reputecast/internal/mcpclient/mcptest"
	"github.com/hostrepute/reputecast/internal/opsmetrics"
)

// exposition is the app's /metrics as Prometheus scrapes it.
func (f *fixture) exposition() string {
	f.t.Helper()
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(f.env.App.Metrics, promhttp.HandlerOpts{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rec.Body.String()
}

// samples is how many observations (histograms) or how much (counters,
// gauges) a family has, over its series whose labels hold every pair given
// ("name", "value", …).
func (f *fixture) samples(name string, labels ...string) float64 {
	f.t.Helper()
	total, text := 0.0, f.exposition()
	histogram := strings.Contains(text, "# TYPE "+name+" histogram")
	for _, line := range strings.Split(text, "\n") {
		series, value, ok := strings.Cut(line, " ")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		metric, set, _ := strings.Cut(series, "{")
		if metric != name && metric != name+"_count" {
			continue
		}
		if metric == name && histogram {
			continue
		}
		match := true
		for i := 0; i+1 < len(labels); i += 2 {
			match = match && strings.Contains(set, labels[i]+`="`+labels[i+1]+`"`)
		}
		if v, err := strconv.ParseFloat(value, 64); err == nil && match {
			total += v
		}
	}
	return total
}

// labelsOf are every label value of the families with a prefix.
func (f *fixture) labelsOf(prefixes ...string) map[string][]string {
	f.t.Helper()
	out := map[string][]string{}
	for _, line := range strings.Split(f.exposition(), "\n") {
		series, _, ok := strings.Cut(line, " ")
		if !ok || strings.HasPrefix(line, "#") || !slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(series, p) }) {
			continue
		}
		metric, set, _ := strings.Cut(series, "{")
		for _, pair := range strings.Split(strings.TrimSuffix(set, "}"), ",") {
			if k, v, ok := strings.Cut(pair, "="); ok && k != "le" {
				out[metric] = append(out[metric], strings.Trim(v, `"`))
			}
		}
	}
	return out
}

// The assistant's metrics are on the app's registry and move with a turn:
// the time to first token the server adds, the stream's duration, its
// tokens, tool calls and MCP calls by outcome, the open streams and how
// late they write events. Their labels are bounded: no id, name or content.
func TestAssistantMetrics(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	fake := mcptest.NewServer(t, "", "")
	fake.Tool("search", "Search the wiki", true, mcptest.Text("hit"))
	fake.Tool("broken", "Fail", true, func(map[string]any) *mcp.CallToolResult {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "down"}}}
	})
	srv := f.remote(map[string]any{"name": "Wiki", "url": fake.Endpoint(), "auth": "none"}, map[string]string{"search": "read_only", "broken": "read_only"})
	th := f.thread(f.mc, map[string]any{})
	f.useServers(f.mc, th.ID, srv.ID)
	names := []string{"reputecast_ai_assistant_ttft_overhead_seconds", "reputecast_ai_assistant_stream_seconds", "reputecast_ai_assistant_tokens_total",
		"reputecast_ai_assistant_tool_calls_total", "reputecast_mcp_call_seconds", "reputecast_mcp_call_errors_total", "reputecast_ai_assistant_streams",
		"reputecast_ai_assistant_event_lag_seconds"}
	before := map[string]float64{}
	for _, n := range names {
		before[n] = f.samples(n)
	}
	inputs, outputs := testutil.ToFloat64(opsmetrics.AssistantTokens.WithLabelValues("input")), testutil.ToFloat64(opsmetrics.AssistantTokens.WithLabelValues("output"))
	s := f.openStream(f.mc, th.ID, "")
	if ev := s.next(t, 10*time.Second); ev.Event != "snapshot" {
		t.Fatalf("first event %s", ev.Event)
	}
	if n := f.samples("reputecast_ai_assistant_streams"); n < 1 {
		t.Fatalf("%v open streams while one is open", n)
	}
	// A plain answer, then one that calls a ReputeCast tool, an MCP tool
	// and one whose server fails.
	f.streams.then(says("Hello there."), calls("search_help", `{"query":"dkim"}`, "mcp__wiki__search", `{"query":"x"}`, "mcp__wiki__broken", `{}`), says("Done."))
	for _, text := range []string{"Hello", "Look it up"} {
		turn := f.send(f.mc, th.ID, text).Turn.ID
		if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
			t.Fatal(err)
		}
		s.until(t, "done")
	}
	for _, n := range names {
		if n != "reputecast_ai_assistant_streams" && f.samples(n) <= before[n] {
			t.Fatalf("%s didn't move", n)
		}
	}
	if testutil.ToFloat64(opsmetrics.AssistantTokens.WithLabelValues("input")) <= inputs || testutil.ToFloat64(opsmetrics.AssistantTokens.WithLabelValues("output")) <= outputs {
		t.Fatal("the tokens didn't move")
	}
	for _, c := range []struct {
		name   string
		labels []string
		want   float64
	}{
		{"reputecast_ai_assistant_tool_calls_total", []string{"kind", "reputecast", "outcome", "ok"}, 1},
		{"reputecast_ai_assistant_tool_calls_total", []string{"kind", "mcp_read", "outcome", "ok"}, 1},
		{"reputecast_ai_assistant_tool_calls_total", []string{"kind", "mcp_read", "outcome", "error"}, 1},
		{"reputecast_mcp_call_errors_total", []string{"code", "tool_error"}, 1},
		{"reputecast_mcp_call_seconds", []string{"result", "ok"}, 1},
		{"reputecast_ai_assistant_stream_seconds", []string{"kind", "answer", "outcome", "answered"}, 3},
		{"reputecast_ai_assistant_event_lag_seconds", []string{"kind", "delta"}, 1},
	} {
		if got := f.samples(c.name, c.labels...); got < c.want {
			t.Fatalf("%s%v: %v, want at least %v", c.name, c.labels, got, c.want)
		}
	}
	s.close()
	waitFor(t, "the stream to close", func() bool { return f.samples("reputecast_ai_assistant_streams") < 1 })
	// Bounded labels only.
	allowed := []string{"answer", "summary", "answered", "refused", "error", "cancelled", "input", "output", "reputecast", "mcp_read", "mcp_write", "ok",
		"not_available", "forbidden", "not_found", "tool_limit", "denied", "outcome_unknown", "timeout", "mcp_unreachable", "mcp_sign_in", "mcp_timeout",
		"mcp_too_large", "mcp_tool_changed", "mcp_tool_blocked", "tool_error", "turn", "delta", "message", "usage", "tool", "confirmation"}
	for metric, values := range f.labelsOf("reputecast_ai_assistant_", "reputecast_mcp_") {
		for _, v := range values {
			if _, err := uuid.Parse(v); err == nil || !slices.Contains(allowed, v) {
				t.Fatalf("%s has the label value %q", metric, v)
			}
		}
	}
}
