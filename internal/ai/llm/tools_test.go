package llm_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/ai/llm"
)

// One contract for all five providers. These are synthetic wire fixtures,
// based on their API schemas, not evidence of live model quality.
func TestToolContract(t *testing.T) {
	for _, kind := range llm.Kinds {
		t.Run(kind, func(t *testing.T) {
			response := toolWire(kind)
			var sent map[string]any
			requests := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
					t.Error(err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, response)
			}))
			defer srv.Close()
			retries := 0
			provider, err := llm.New(llm.Config{Kind: kind, BaseURL: srv.URL, Model: "fixture", Deployment: "fixture", HTTP: srv.Client(), Retries: &retries})
			if err != nil {
				t.Fatal(err)
			}
			req := llm.Request{MaxTokens: 180, Tools: []llm.Tool{{Name: "read_draft", Description: "Read a selected draft.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"handle": map[string]any{"type": "string"}}, "required": []string{"handle"}, "additionalProperties": false}}}, Messages: []llm.Message{{Role: "user", Parts: []llm.Part{{Text: "Compare the selected drafts."}}}}}
			out, err := provider.Generate(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if out.StopReason != llm.StopTools || !out.UsageKnown || out.InputTokens != 113 || out.OutputTokens != 29 || out.Text != "Reading selected drafts." {
				t.Fatalf("tool response: %+v", out)
			}
			tool := sent["tools"].([]any)[0].(map[string]any)
			var schema any
			switch kind {
			case llm.Anthropic:
				if tool["name"] != "read_draft" {
					t.Fatal(tool)
				}
				schema = tool["input_schema"]
			case llm.Google:
				declaration := tool["functionDeclarations"].([]any)[0].(map[string]any)
				if declaration["name"] != "read_draft" {
					t.Fatal(declaration)
				}
				schema = declaration["parametersJsonSchema"]
			default:
				function := tool["function"].(map[string]any)
				if tool["type"] != "function" || function["name"] != "read_draft" {
					t.Fatal(tool)
				}
				schema = function["parameters"]
			}
			wantSchema := `{"type":"object","properties":{"handle":{"type":"string"}},"required":["handle"],"additionalProperties":false}`
			assertJSON(t, schema, wantSchema)
			results := llm.Message{Role: "tool"}
			calls := []llm.ToolCall{}
			for _, part := range out.Message.Parts {
				if part.ToolCall == nil {
					continue
				}
				c := *part.ToolCall
				calls = append(calls, c)
				value := json.RawMessage(`{"subject":"First draft"}`)
				if len(calls) == 2 {
					value = json.RawMessage(`{"error":"scope_changed"}`)
				}
				results.Parts = append(results.Parts, llm.Part{ToolResult: &llm.ToolResult{ID: c.ID, Name: c.Name, Content: value, IsError: len(calls) == 2}})
			}
			if len(calls) != 2 || calls[0].Name != "read_draft" || calls[1].Name != "read_draft" || string(calls[0].Arguments) != `{"handle":"draft_a"}` || string(calls[1].Arguments) != `{"handle":"draft_b"}` {
				t.Fatalf("calls: %+v", calls)
			}
			if kind != llm.Google && (calls[0].ID != "call_a" || calls[1].ID != "call_b") {
				t.Fatal(calls)
			}
			req.Messages = append(req.Messages, out.Message, results)
			if _, err := provider.Generate(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case llm.Anthropic:
				messages := sent["messages"].([]any)
				assertJSON(t, messages[1], `{"role":"assistant","content":[{"type":"text","text":"Reading selected drafts."},{"type":"tool_use","id":"call_a","name":"read_draft","input":{"handle":"draft_a"}},{"type":"tool_use","id":"call_b","name":"read_draft","input":{"handle":"draft_b"}}]}`)
				assertJSON(t, messages[2], `{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_a","content":"{\"subject\":\"First draft\"}"},{"type":"tool_result","tool_use_id":"call_b","content":"{\"error\":\"scope_changed\"}","is_error":true}]}`)
			case llm.Google:
				messages := sent["contents"].([]any)
				assertJSON(t, messages[1], `{"role":"model","parts":[{"text":"private reasoning fixture","thought":true,"thoughtSignature":"c2lnbmF0dXJlMQ=="},{"text":"Reading selected drafts."},{"functionCall":{"id":"call_a","name":"read_draft","args":{"handle":"draft_a"}},"thoughtSignature":"c2lnbmF0dXJlMg=="},{"functionCall":{"name":"read_draft","args":{"handle":"draft_b"}}}]}`)
				assertJSON(t, messages[2], `{"role":"user","parts":[{"functionResponse":{"id":"call_a","name":"read_draft","response":{"subject":"First draft"}}},{"functionResponse":{"name":"read_draft","response":{"error":"scope_changed"}}}]}`)
			default:
				messages := sent["messages"].([]any)
				assertJSON(t, messages[1], `{"role":"assistant","content":"Reading selected drafts.","tool_calls":[{"id":"call_a","type":"function","function":{"name":"read_draft","arguments":"{\"handle\":\"draft_a\"}"}},{"id":"call_b","type":"function","function":{"name":"read_draft","arguments":"{\"handle\":\"draft_b\"}"}}]}`)
				assertJSON(t, messages[2], `{"role":"tool","tool_call_id":"call_a","content":"{\"subject\":\"First draft\"}"}`)
				assertJSON(t, messages[3], `{"role":"tool","tool_call_id":"call_b","content":"{\"error\":\"scope_changed\"}"}`)
			}

			// A damaged checkpoint must fail locally rather than send a
			// mismatched result or silently drop part of the conversation.
			for _, fault := range []string{"missing result", "wrong id", "wrong name", "extra result", "interleaved user", "plain tool text", "duplicate call", "mixed call"} {
				t.Run(fault, func(t *testing.T) {
					raw, _ := json.Marshal(req)
					var broken llm.Request
					_ = json.Unmarshal(raw, &broken)
					switch fault {
					case "missing result":
						broken.Messages = broken.Messages[:2]
					case "wrong id":
						broken.Messages[2].Parts[0].ToolResult.ID = "other"
					case "wrong name":
						broken.Messages[2].Parts[0].ToolResult.Name = "another_tool"
					case "extra result":
						broken.Messages[2].Parts = append(broken.Messages[2].Parts, broken.Messages[2].Parts[0])
					case "interleaved user":
						broken.Messages = append(broken.Messages[:2], broken.Messages[0], broken.Messages[2])
					case "plain tool text":
						broken.Messages[2].Parts[0].Text = "silently discarded"
					case "duplicate call":
						for i := range broken.Messages[1].Parts {
							if c := broken.Messages[1].Parts[i].ToolCall; c != nil {
								c.ID = "duplicate"
							}
						}
					case "mixed call":
						for i := range broken.Messages[1].Parts {
							if broken.Messages[1].Parts[i].ToolCall != nil {
								broken.Messages[1].Parts[i].Text = "cannot replay"
							}
						}
					}
					before := requests
					if _, err := provider.Generate(context.Background(), broken); err == nil || requests != before {
						t.Fatalf("broken checkpoint reached provider: %v", err)
					}
				})
			}

			// Without trustworthy usage, a cost-bounded runner must keep its reservation.
			usageKey, inputKey, outputKey := "usage", "prompt_tokens", "completion_tokens"
			if kind == llm.Anthropic {
				inputKey, outputKey = "input_tokens", "output_tokens"
			}
			if kind == llm.Google {
				usageKey, inputKey, outputKey = "usageMetadata", "promptTokenCount", "candidatesTokenCount"
			}
			for _, test := range []struct {
				name  string
				usage any
				known bool
			}{
				{"absent", nil, false},
				{"missing output", map[string]int{inputKey: 10}, false},
				{"negative", map[string]int{inputKey: -1, outputKey: 10}, false},
				{"explicit zero", map[string]int{inputKey: 0, outputKey: 0}, true},
			} {
				t.Run(test.name, func(t *testing.T) {
					var wire map[string]any
					_ = json.Unmarshal([]byte(toolWire(kind)), &wire)
					wire[usageKey] = test.usage
					raw, _ := json.Marshal(wire)
					response = string(raw)
					res, err := provider.Generate(context.Background(), req)
					if err != nil || res.UsageKnown != test.known {
						t.Fatalf("usage: %+v, %v", res, err)
					}
				})
			}
			// A truncated call must never execute, but its paid usage is not lost.
			stop, truncated := `"finish_reason":"tool_calls"`, `"finish_reason":"length"`
			if kind == llm.Anthropic {
				stop, truncated = `"stop_reason":"tool_use"`, `"stop_reason":"max_tokens"`
			}
			if kind == llm.Google {
				stop, truncated = `"finishReason":"STOP"`, `"finishReason":"MAX_TOKENS"`
			}
			response = strings.Replace(toolWire(kind), stop, truncated, 1)
			out, err = provider.Generate(context.Background(), req)
			if err == nil || !out.UsageKnown || out.OutputTokens != 29 {
				t.Fatalf("truncated: %+v %v", out, err)
			}
			response = toolWire(kind)
			req.Tools = nil
			if _, err := provider.Generate(context.Background(), req); err == nil {
				t.Fatal("unsolicited tools accepted")
			}
		})
	}
}

func assertJSON(t *testing.T, got any, want string) {
	t.Helper()
	var expected any
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("wire content: %#v\nwant: %#v", got, expected)
	}
}

func toolWire(kind string) string {
	switch kind {
	case llm.Anthropic:
		return `{"model":"fixture","stop_reason":"tool_use","content":[{"type":"text","text":"Reading selected drafts."},{"type":"tool_use","id":"call_a","name":"read_draft","input":{"handle":"draft_a"}},{"type":"tool_use","id":"call_b","name":"read_draft","input":{"handle":"draft_b"}}],"usage":{"input_tokens":83,"cache_read_input_tokens":23,"cache_creation_input_tokens":7,"output_tokens":29}}`
	case llm.Google:
		return `{"modelVersion":"fixture","candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[{"text":"private reasoning fixture","thought":true,"thoughtSignature":"c2lnbmF0dXJlMQ=="},{"text":"Reading selected drafts."},{"functionCall":{"id":"call_a","name":"read_draft","args":{"handle":"draft_a"}},"thoughtSignature":"c2lnbmF0dXJlMg=="},{"functionCall":{"name":"read_draft","args":{"handle":"draft_b"}}}]}}],"usageMetadata":{"promptTokenCount":113,"candidatesTokenCount":18,"thoughtsTokenCount":11,"cachedContentTokenCount":17,"totalTokenCount":142}}`
	default:
		return `{"model":"fixture","choices":[{"finish_reason":"tool_calls","message":{"content":"Reading selected drafts.","tool_calls":[{"id":"call_a","type":"function","function":{"name":"read_draft","arguments":"{\"handle\":\"draft_a\"}"}},{"id":"call_b","type":"function","function":{"name":"read_draft","arguments":"{\"handle\":\"draft_b\"}"}}]}}],"usage":{"prompt_tokens":113,"completion_tokens":29,"completion_tokens_details":{"reasoning_tokens":13}}}`
	}
}
