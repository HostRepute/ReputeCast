package llm_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/ai/llm/contracttest"
)

const testKey = "sk-test-0123456789"

func TestAnthropicContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Kind: llm.Anthropic,
		Dir:  "testdata/anthropic",
		Config: func(url string) llm.Config {
			return llm.Config{BaseURL: url, APIKey: testKey, Model: "claude-sonnet-5-5"}
		},
		CheckRequest: func(t *testing.T, scenario string, r contracttest.Recorded) {
			if r.Header.Get("x-api-key") != testKey || r.Header.Get("anthropic-version") != "2023-06-01" || r.Header.Get("Authorization") != "" {
				t.Fatalf("headers: %v", r.Header)
			}
			if scenario == contracttest.Models {
				if r.Path != "/v1/models" {
					t.Fatalf("path: %s", r.Path)
				}
				return
			}
			if r.Path != "/v1/messages" {
				t.Fatalf("path: %s", r.Path)
			}
			b := r.JSONBody(t)
			if b["model"] != "claude-sonnet-5-5" || b["max_tokens"] != float64(64) || !strings.HasPrefix(b["system"].(string), contracttest.SystemMarker) {
				t.Fatalf("body: %v", b)
			}
			msgs := b["messages"].([]any)
			content := msgs[0].(map[string]any)["content"].([]any)
			if scenario == contracttest.Image {
				img := content[0].(map[string]any)
				src := img["source"].(map[string]any)
				if img["type"] != "image" || src["type"] != "base64" || src["media_type"] != "image/png" {
					t.Fatalf("image block: %v", img)
				}
			}
			if scenario == contracttest.JSON && !strings.Contains(b["system"].(string), "JSON") {
				t.Fatalf("a JSON answer is asked for in the system prompt: %v", b["system"])
			}
		},
		Want: contracttest.Want{
			Text: "Ready.", InputTokens: 14, OutputTokens: 3, JSONKey: "subject", JSONValue: "A year of on-call, in numbers",
			RefusalText: "I can't write a subject that pretends to be a reply to an email the reader never sent.",
			AuthStatus:  401, AuthMessage: "authentication_error: invalid x-api-key", RetryAfter: 20 * time.Second, OverloadedStatus: 529,
			Models: []string{"claude-sonnet-5-5", "claude-haiku-4-5"},
		},
	})
}

func checkChat(t *testing.T, scenario string, r contracttest.Recorded, model string, maxKey string) {
	t.Helper()
	b := r.JSONBody(t)
	if model == "" {
		if _, ok := b["model"]; ok {
			t.Fatalf("an Azure deployment names the model in its URL: %v", b)
		}
	} else if b["model"] != model {
		t.Fatalf("model: %v", b["model"])
	}
	if b[maxKey] != float64(64) {
		t.Fatalf("%s: %v", maxKey, b)
	}
	msgs := b["messages"].([]any)
	sys := msgs[0].(map[string]any)
	if sys["role"] != "system" || sys["content"] != contracttest.SystemMarker {
		t.Fatalf("system message: %v", sys)
	}
	user := msgs[1].(map[string]any)
	switch scenario {
	case contracttest.Image:
		parts := user["content"].([]any)
		img := parts[0].(map[string]any)
		if img["type"] != "image_url" || !strings.HasPrefix(img["image_url"].(map[string]any)["url"].(string), "data:image/png;base64,") {
			t.Fatalf("image part: %v", img)
		}
	default:
		if user["content"] != contracttest.UserMarker {
			t.Fatalf("a text turn is a plain string: %v", user["content"])
		}
	}
	if scenario == contracttest.JSON {
		rf, _ := b["response_format"].(map[string]any)
		if rf == nil || (rf["type"] != "json_schema" && rf["type"] != "json_object") {
			t.Fatalf("response_format: %v", b["response_format"])
		}
	}
}

func TestOpenAIContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Kind: llm.OpenAI,
		Dir:  "testdata/openai",
		Config: func(url string) llm.Config {
			return llm.Config{BaseURL: url + "/v1", APIKey: testKey, Model: "gpt-5"}
		},
		CheckRequest: func(t *testing.T, scenario string, r contracttest.Recorded) {
			if r.Header.Get("Authorization") != "Bearer "+testKey {
				t.Fatalf("authorization: %q", r.Header.Get("Authorization"))
			}
			if scenario == contracttest.Models {
				if r.Path != "/v1/models" {
					t.Fatalf("path: %s", r.Path)
				}
				return
			}
			if r.Path != "/v1/chat/completions" {
				t.Fatalf("path: %s", r.Path)
			}
			checkChat(t, scenario, r, "gpt-5", "max_completion_tokens")
			if scenario == contracttest.JSON {
				if rf := r.JSONBody(t)["response_format"].(map[string]any); rf["type"] != "json_schema" {
					t.Fatalf("OpenAI gets the schema: %v", rf)
				}
			}
		},
		Want: contracttest.Want{
			Text: "Ready.", InputTokens: 19, OutputTokens: 2, JSONKey: "subject", JSONValue: "A year of on-call, in numbers",
			RefusalText: "I can't write a subject that pretends to be a reply to an email the reader never sent.",
			AuthStatus:  401, AuthMessage: "Incorrect API key provided", RetryAfter: 20 * time.Second, OverloadedStatus: 503,
			Models: []string{"gpt-5", "gpt-5-mini"},
		},
	})
}

func TestAzureOpenAIContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Kind: llm.Azure,
		Dir:  "testdata/azure",
		Config: func(url string) llm.Config {
			return llm.Config{BaseURL: url, APIKey: testKey, Model: "gpt-4.1", Deployment: "acme-gpt41", APIVersion: "2024-10-21"}
		},
		CheckRequest: func(t *testing.T, scenario string, r contracttest.Recorded) {
			if r.Header.Get("api-key") != testKey || r.Header.Get("Authorization") != "" {
				t.Fatalf("headers: %v", r.Header)
			}
			if r.Path != "/openai/deployments/acme-gpt41/chat/completions" || r.Query != "api-version=2024-10-21" {
				t.Fatalf("url: %s?%s", r.Path, r.Query)
			}
			checkChat(t, scenario, r, "", "max_tokens")
		},
		Want: contracttest.Want{
			Text: "Ready.", InputTokens: 19, OutputTokens: 2, JSONKey: "subject", JSONValue: "A year of on-call, in numbers",
			AuthStatus: 401, AuthMessage: "invalid subscription key", RetryAfter: 20 * time.Second, OverloadedStatus: 503,
			Models: []string{"acme-gpt41"},
		},
	})
}

func TestCompatibleContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Kind: llm.Compatible,
		Dir:  "testdata/compatible",
		Config: func(url string) llm.Config {
			// Local servers usually take no key: none is sent.
			return llm.Config{BaseURL: url + "/v1", Model: "llama3.3:70b"}
		},
		CheckRequest: func(t *testing.T, scenario string, r contracttest.Recorded) {
			if r.Header.Get("Authorization") != "" {
				t.Fatalf("no key, no Authorization: %q", r.Header.Get("Authorization"))
			}
			if scenario == contracttest.Models {
				if r.Path != "/v1/models" {
					t.Fatalf("path: %s", r.Path)
				}
				return
			}
			if r.Path != "/v1/chat/completions" {
				t.Fatalf("path: %s", r.Path)
			}
			checkChat(t, scenario, r, "llama3.3:70b", "max_tokens")
			if scenario == contracttest.JSON {
				if rf := r.JSONBody(t)["response_format"].(map[string]any); rf["type"] != "json_object" {
					t.Fatalf("compatible servers get json_object: %v", rf)
				}
			}
		},
		Want: contracttest.Want{
			Text: "Ready.", InputTokens: 26, OutputTokens: 3, JSONKey: "subject", JSONValue: "A year of on-call, in numbers",
			AuthStatus: 401, AuthMessage: "Unauthorized", OverloadedStatus: 503,
			Models: []string{"llama3.3:70b", "gemma3:27b"},
		},
	})
}

func TestGeminiContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Kind: llm.Google,
		Dir:  "testdata/google",
		Config: func(url string) llm.Config {
			return llm.Config{BaseURL: url, APIKey: testKey, Model: "gemini-2.5-flash"}
		},
		CheckRequest: func(t *testing.T, scenario string, r contracttest.Recorded) {
			if r.Header.Get("x-goog-api-key") != testKey || strings.Contains(r.Query, "key=") {
				t.Fatalf("the key goes in x-goog-api-key, never the URL: %v %s", r.Header, r.Query)
			}
			if scenario == contracttest.Models {
				if r.Path != "/v1beta/models" {
					t.Fatalf("path: %s", r.Path)
				}
				return
			}
			if r.Path != "/v1beta/models/gemini-2.5-flash:generateContent" {
				t.Fatalf("path: %s", r.Path)
			}
			b := r.JSONBody(t)
			sys := b["systemInstruction"].(map[string]any)["parts"].([]any)[0].(map[string]any)
			if sys["text"] != contracttest.SystemMarker {
				t.Fatalf("system: %v", sys)
			}
			gc := b["generationConfig"].(map[string]any)
			if gc["maxOutputTokens"] != float64(64) {
				t.Fatalf("generationConfig: %v", gc)
			}
			parts := b["contents"].([]any)[0].(map[string]any)["parts"].([]any)
			if scenario == contracttest.Image {
				if parts[0].(map[string]any)["inline_data"].(map[string]any)["mime_type"] != "image/png" {
					t.Fatalf("image part: %v", parts[0])
				}
			}
			if scenario == contracttest.JSON && gc["responseMimeType"] != "application/json" {
				t.Fatalf("JSON answer: %v", gc)
			}
		},
		Want: contracttest.Want{
			Text: "Ready.", InputTokens: 12, OutputTokens: 2, JSONKey: "subject", JSONValue: "A year of on-call, in numbers",
			AuthStatus: 400, AuthMessage: "API key not valid", RetryAfter: 20 * time.Second, OverloadedStatus: 503,
			Models: []string{"gemini-2.5-flash", "gemini-2.5-pro"},
		},
	})
}

func TestFirstJSON(t *testing.T) {
	for in, want := range map[string]string{
		`{"a":1}`:                            `{"a":1}`,
		"```json\n{\"a\": \"x}\"}\n```":      `{"a": "x}"}`,
		`Here you go: {"b":{"c":[1,2]}} ok.`: `{"b":{"c":[1,2]}}`,
		`{broken {"d":true}`:                 `{"d":true}`,
	} {
		got, ok := llm.FirstJSON(in)
		if !ok || string(got) != want {
			t.Errorf("FirstJSON(%q) = %s, %v", in, got, ok)
		}
	}
	if _, ok := llm.FirstJSON("no json here"); ok {
		t.Error("found JSON in plain text")
	}
}

func TestCatalog(t *testing.T) {
	if k, ok := llm.Known(llm.Anthropic, "claude-haiku-4-5-20251001"); !ok || k.ID != "claude-haiku-4-5" {
		t.Fatalf("dated snapshot: %+v %v", k, ok)
	}
	if k, ok := llm.Known(llm.Anthropic, "claude-sonnet-5-5"); !ok || k.Input != 2 || k.Output != 10 || k.Name != "Claude Sonnet 5.5" {
		t.Fatalf("the sample's model: %+v", k)
	}
	if !llm.ImageInput(llm.Anthropic, "claude-sonnet-5-5") || llm.ImageInput(llm.Compatible, "llama3.3:70b") || !llm.ImageInput(llm.Compatible, "gemma3:27b") {
		t.Fatal("image input")
	}
	if _, err := llm.New(llm.Config{Kind: llm.Azure, APIKey: "k"}); err == nil {
		t.Fatal("Azure needs its endpoint and deployment")
	}
	if _, err := llm.New(llm.Config{Kind: "palm"}); err == nil {
		t.Fatal("unknown provider")
	}
}
