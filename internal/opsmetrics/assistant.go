package opsmetrics

import "github.com/prometheus/client_golang/prometheus"

// The AI assistant's metrics (B6). Their labels are bounded sets of
// constants: never content, ids or names (assistant brief §3.7).
var (
	// AssistantTTFTOverhead is what the server adds before a turn's first
	// answer token reaches the database, the provider's own time taken out.
	AssistantTTFTOverhead = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "reputecast_ai_assistant_ttft_overhead_seconds",
		Help: "Server time before an assistant turn's first answer token is committed, without the provider: from the person's message committed to the first delta committed, " +
			"minus the provider's time from the request sent to its first token. Counts turns whose first model request streams text; includes the job's wait for a worker and a slot, " +
			"the context assembly, the budget reservation and writing the first delta.",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
	})
	// AssistantStream is how long a turn's model request streamed.
	AssistantStream = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "reputecast_ai_assistant_stream_seconds",
		Help: "Duration of an assistant model request, from sending it to the provider's last event or the stream's end, by kind (answer, summary) and outcome " +
			"(answered, refused, error, cancelled).",
		Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	}, []string{"kind", "outcome"})
	// AssistantTokens counts the tokens the provider reported.
	AssistantTokens = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_ai_assistant_tokens_total",
		Help: "Tokens of assistant model requests as providers reported them, by direction (input, output). Requests whose usage wasn't reported aren't counted.",
	}, []string{"direction"})
	// AssistantToolCalls counts the tool calls models asked for.
	AssistantToolCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_ai_assistant_tool_calls_total",
		Help: "Tool calls assistant models asked for, by kind (reputecast, mcp_read, mcp_write) and outcome (ok, not_available, forbidden, not_found, tool_limit, denied, " +
			"outcome_unknown, timeout, error).",
	}, []string{"kind", "outcome"})
	// MCPCall is how long calls of MCP servers' tools took.
	MCPCall = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "reputecast_mcp_call_seconds",
		Help:    "Duration of calls of remote MCP servers' tools, connection included, by result (ok, error).",
		Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10, 30},
	}, []string{"result"})
	// MCPCallErrors counts failed calls by code.
	MCPCallErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_mcp_call_errors_total",
		Help: "Failed calls of remote MCP servers' tools, by code (mcp_unreachable, mcp_sign_in, mcp_timeout, mcp_too_large, mcp_tool_changed, mcp_tool_blocked, tool_error, " +
			"forbidden, error).",
	}, []string{"code"})
	// AssistantStreams counts the thread streams this process serves.
	AssistantStreams = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "reputecast_ai_assistant_streams",
		Help: "Assistant thread event streams (SSE) this process serves now.",
	})
	// AssistantEventLag is how late a stream writes an event.
	AssistantEventLag = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "reputecast_ai_assistant_event_lag_seconds",
		Help: "Time from an assistant thread event's insert in its committing transaction to an open stream writing it to its client, by kind (turn, delta, message, usage, tool, " +
			"confirmation). Events a stream catches up on when it (re)connects aren't counted.",
		Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
	}, []string{"kind"})
)

func init() {
	counters = append(counters, AssistantTTFTOverhead, AssistantStream, AssistantTokens, AssistantToolCalls, MCPCall, MCPCallErrors, AssistantStreams, AssistantEventLag)
}

// ToolOutcome is a tool call's outcome label for its result code ("" is
// ok): the codes a person or operator acts on, anything else error.
func ToolOutcome(code string) string {
	switch code {
	case "":
		return "ok"
	case "not_available", "not_attached", "confirmation_required":
		return "not_available"
	case "forbidden", "not_found", "tool_limit", "denied", "outcome_unknown":
		return code
	case "mcp_timeout", "timeout":
		return "timeout"
	}
	return "error"
}

// MCPErrorCode is a failed MCP call's code label.
func MCPErrorCode(outcome string) string {
	switch outcome {
	case "mcp_unreachable", "mcp_sign_in", "mcp_timeout", "mcp_too_large", "mcp_tool_changed", "mcp_tool_blocked", "tool_error", "forbidden":
		return outcome
	case "timeout":
		return "mcp_timeout"
	case "too_large":
		return "mcp_too_large"
	}
	return "error"
}
