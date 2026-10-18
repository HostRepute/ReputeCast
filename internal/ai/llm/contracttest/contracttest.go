// Package contracttest is the shared test suite every AI provider adapter
// passes (design expansion, plan §8: each new provider kind gets one, like
// the sending adapters' internal/providers/contracttest).
//
// It runs the adapter against a fake of the provider's API that answers
// with recorded answers (testdata/<provider>/<scenario>.json: the status,
// the headers and the body the provider sent), and records every request,
// so the adapter's own test can check how it talks to the provider. No test
// reaches the network.
//
//	func TestContract(t *testing.T) {
//		contracttest.Run(t, contracttest.Suite{Kind: llm.Anthropic, Dir: "testdata/anthropic", Config: …, CheckRequest: …})
//	}
//
// Scenarios: a text answer, an image in the request, a JSON answer, a
// refusal, an answer cut at the token limit, a rejected key, a rate limit
// with Retry-After, an overloaded provider (retried once, then reported), a
// bad request, the model list, an answer over the size limit, a timeout and
// an unreadable answer.
package contracttest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/ai/llm"
)

// Scenarios with a recorded answer in Dir.
const (
	Text       = "text"
	Image      = "image"
	JSON       = "json"
	Refusal    = "refusal"
	Length     = "length"
	Auth       = "auth"
	RateLimit  = "rate_limit"
	Overloaded = "overloaded"
	BadRequest = "bad_request"
	Models     = "models"
)

// Scenarios the suite makes up itself.
const (
	TooLarge = "too_large"
	Timeout  = "timeout"
	Invalid  = "invalid"
)

// Markers the suite puts in every request, to find them in what was sent.
const (
	SystemMarker = "SYSTEM-7f3a: follow the house rules."
	UserMarker   = "<content>USER-91c2 This week: on-call numbers {{ first_name | default: \"there\" }}</content>"
)

// TestImage is the image of the image scenario (a 1×1 PNG).
var TestImage = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0x1f, 0x15, 0xc4, 0x89}

// Fixture is a recorded answer.
type Fixture struct {
	Status int               `json:"status"`
	Header map[string]string `json:"header"`
	Body   json.RawMessage   `json:"body"`
}

// Recorded is a request the fake API received.
type Recorded struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

// JSONBody decodes the request body.
func (r Recorded) JSONBody(t testing.TB) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("request body isn't JSON: %v\n%s", err, r.Body)
	}
	return m
}

// Suite describes an adapter under test.
type Suite struct {
	Kind string
	// Dir holds the recorded answers: <scenario>.json.
	Dir string
	// Config returns the adapter's configuration for the fake API at url.
	Config func(url string) llm.Config
	// CheckRequest asserts how the adapter talks to its provider (path,
	// authentication, body fields) for one scenario's request.
	CheckRequest func(t *testing.T, scenario string, r Recorded)
	// Want is what the recorded answers say.
	Want Want
}

// Want is what the recorded answers hold.
type Want struct {
	// Text of the text and image answers.
	Text string
	// InputTokens and OutputTokens of the text answer.
	InputTokens, OutputTokens int
	// JSONKey and JSONValue are a field of the JSON answer.
	JSONKey, JSONValue string
	// RefusalText is the model's words in the refusal answer ("" when the
	// provider gives none).
	RefusalText string
	// AuthStatus is the status of a rejected key (401; Gemini says 400).
	AuthStatus int
	// AuthMessage is part of the provider's message about the key.
	AuthMessage string
	// RetryAfter of the rate-limit answer.
	RetryAfter time.Duration
	// OverloadedStatus is the status of the overloaded answer (529, 503).
	OverloadedStatus int
	// Models are ids the model list holds.
	Models []string
}

type fake struct {
	mu       sync.Mutex
	scenario string
	fixtures map[string]Fixture
	requests []Recorded
	delay    time.Duration
}

func (f *fake) set(s string) {
	f.mu.Lock()
	f.scenario, f.requests, f.delay = s, nil, 0
	f.mu.Unlock()
}

func (f *fake) recorded() []Recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Recorded(nil), f.requests...)
}

func (f *fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.requests = append(f.requests, Recorded{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Header: r.Header.Clone(), Body: body})
	s, delay := f.scenario, f.delay
	fx, ok := f.fixtures[s]
	f.mu.Unlock()
	if delay > 0 {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(delay):
		}
	}
	switch s {
	case TooLarge:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pad":"` + strings.Repeat("x", 4096) + `"}`))
		return
	case Invalid:
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!DOCTYPE html><title>Gateway</title>"))
		return
	case Timeout:
		fx, ok = f.fixtures[Text]
	}
	if !ok {
		http.Error(w, "no recorded answer for "+s, http.StatusNotImplemented)
		return
	}
	for k, v := range fx.Header {
		w.Header().Set(k, v)
	}
	w.Header().Set("Content-Type", "application/json")
	status := fx.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write(fx.Body)
}

// required are the scenarios every provider has a recorded answer for. A
// server that speaks the OpenAI API may not flag refusals or limit rates:
// those scenarios run when their answer is recorded.
var required = map[string]bool{Text: true, Image: true, JSON: true, Auth: true, Overloaded: true, BadRequest: true, Models: true}

func load(t *testing.T, dir string) map[string]Fixture {
	t.Helper()
	out := map[string]Fixture{}
	for _, s := range []string{Text, Image, JSON, Refusal, Length, Auth, RateLimit, Overloaded, BadRequest, Models} {
		b, err := os.ReadFile(filepath.Join(dir, s+".json")) //nolint:gosec // the suite's own recorded answers
		if os.IsNotExist(err) && !required[s] {
			continue
		}
		if err != nil {
			t.Fatalf("recorded answer %s: %v", s, err)
		}
		var fx Fixture
		if err := json.Unmarshal(b, &fx); err != nil {
			t.Fatalf("recorded answer %s: %v", s, err)
		}
		out[s] = fx
	}
	return out
}

func textRequest() llm.Request {
	return llm.Request{System: SystemMarker, MaxTokens: 64, Messages: []llm.Message{{Role: "user", Parts: []llm.Part{{Text: UserMarker}}}}}
}

// Run runs the suite.
func Run(t *testing.T, s Suite) {
	f := &fake{fixtures: load(t, s.Dir)}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	retries := 1
	newProvider := func(t *testing.T, mutate func(*llm.Config)) llm.Provider {
		t.Helper()
		cfg := s.Config(srv.URL)
		cfg.Kind = s.Kind
		cfg.HTTP = srv.Client()
		cfg.Retries = &retries
		cfg.RetryWait = time.Millisecond
		if mutate != nil {
			mutate(&cfg)
		}
		p, err := llm.New(cfg)
		if err != nil {
			t.Fatalf("new %s: %v", s.Kind, err)
		}
		return p
	}
	ctx := context.Background()
	check := func(t *testing.T, scenario string) {
		t.Helper()
		reqs := f.recorded()
		if len(reqs) == 0 {
			t.Fatalf("%s: the fake API received nothing", scenario)
		}
		if s.CheckRequest != nil {
			s.CheckRequest(t, scenario, reqs[0])
		}
	}
	sent := func(t *testing.T) Recorded {
		t.Helper()
		reqs := f.recorded()
		if len(reqs) == 0 {
			t.Fatal("nothing was sent")
		}
		return reqs[0]
	}

	t.Run("text", func(t *testing.T) {
		f.set(Text)
		res, err := newProvider(t, nil).Generate(ctx, textRequest())
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if res.Text != s.Want.Text || res.StopReason != llm.StopEnd || res.Refused {
			t.Fatalf("answer: %+v", res)
		}
		if res.InputTokens != s.Want.InputTokens || res.OutputTokens != s.Want.OutputTokens {
			t.Fatalf("tokens: %d in, %d out", res.InputTokens, res.OutputTokens)
		}
		r := sent(t)
		if r.Method != http.MethodPost || !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			t.Fatalf("request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		// The system prompt and the person's content go as given, tokens
		// as written; nothing else is added to them.
		if !bytes.Contains(r.Body, []byte("SYSTEM-7f3a")) || !bytes.Contains(r.Body, []byte("USER-91c2")) {
			t.Fatalf("the prompt isn't in the request: %s", r.Body)
		}
		var decoded any
		_ = json.Unmarshal(r.Body, &decoded)
		if !strings.Contains(flatten(decoded), `{{ first_name | default: "there" }}`) {
			t.Fatalf("the Liquid token changed on the way: %s", r.Body)
		}
		check(t, Text)
	})

	t.Run("image", func(t *testing.T) {
		f.set(Image)
		req := llm.Request{System: SystemMarker, MaxTokens: 64, Messages: []llm.Message{{Role: "user", Parts: []llm.Part{
			{Image: &llm.Image{MediaType: "image/png", Data: TestImage}}, {Text: UserMarker},
		}}}}
		res, err := newProvider(t, nil).Generate(ctx, req)
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if res.Text == "" {
			t.Fatalf("answer: %+v", res)
		}
		if !bytes.Contains(sent(t).Body, []byte(base64.StdEncoding.EncodeToString(TestImage))) {
			t.Fatalf("the image isn't in the request: %s", sent(t).Body)
		}
		check(t, Image)
	})

	t.Run("json", func(t *testing.T) {
		f.set(JSON)
		req := textRequest()
		req.JSON = true
		req.SchemaName = "subjects"
		req.Schema = map[string]any{"type": "object", "properties": map[string]any{s.Want.JSONKey: map[string]any{"type": "string"}}, "required": []string{s.Want.JSONKey}}
		res, err := newProvider(t, nil).Generate(ctx, req)
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		raw, ok := llm.FirstJSON(res.Text)
		if !ok {
			t.Fatalf("not JSON: %q", res.Text)
		}
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		if m[s.Want.JSONKey] != s.Want.JSONValue {
			t.Fatalf("JSON answer: %s", raw)
		}
		check(t, JSON)
	})

	recorded := func(t *testing.T, scenario string) {
		t.Helper()
		if _, ok := f.fixtures[scenario]; !ok {
			t.Skipf("no recorded %s answer for %s", scenario, s.Kind)
		}
	}

	t.Run("refusal", func(t *testing.T) {
		recorded(t, Refusal)
		f.set(Refusal)
		res, err := newProvider(t, nil).Generate(ctx, textRequest())
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if !res.Refused || res.StopReason != llm.StopRefusal {
			t.Fatalf("not a refusal: %+v", res)
		}
		if s.Want.RefusalText != "" && res.Text != s.Want.RefusalText {
			t.Fatalf("the model's words: %q", res.Text)
		}
	})

	t.Run("length", func(t *testing.T) {
		recorded(t, Length)
		f.set(Length)
		res, err := newProvider(t, nil).Generate(ctx, textRequest())
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if res.StopReason != llm.StopLength {
			t.Fatalf("stop reason: %+v", res)
		}
	})

	errorOf := func(t *testing.T, err error) *llm.Error {
		t.Helper()
		e, ok := llm.AsError(err)
		if !ok {
			t.Fatalf("want an *llm.Error, got %v", err)
		}
		return e
	}

	t.Run("auth", func(t *testing.T) {
		f.set(Auth)
		_, err := newProvider(t, nil).Generate(ctx, textRequest())
		e := errorOf(t, err)
		if e.Kind != llm.ErrAuth || e.Status != s.Want.AuthStatus || !strings.Contains(e.Error(), s.Want.AuthMessage) {
			t.Fatalf("rejected key: %+v (%s)", e, e.Error())
		}
		if n := len(f.recorded()); n != 1 {
			t.Fatalf("a rejected key is never retried: %d requests", n)
		}
	})

	t.Run("rate_limit", func(t *testing.T) {
		recorded(t, RateLimit)
		f.set(RateLimit)
		_, err := newProvider(t, nil).Generate(ctx, textRequest())
		e := errorOf(t, err)
		if e.Kind != llm.ErrRateLimit || e.Status != http.StatusTooManyRequests || e.RetryAfter != s.Want.RetryAfter {
			t.Fatalf("rate limit: %+v", e)
		}
		if n := len(f.recorded()); n != 1 {
			t.Fatalf("a rate limit is reported, not retried: %d requests", n)
		}
	})

	t.Run("overloaded", func(t *testing.T) {
		f.set(Overloaded)
		_, err := newProvider(t, nil).Generate(ctx, textRequest())
		e := errorOf(t, err)
		if e.Kind != llm.ErrOverloaded || e.Status != s.Want.OverloadedStatus || e.Attempts != 2 {
			t.Fatalf("overloaded: %+v", e)
		}
		if n := len(f.recorded()); n != 2 {
			t.Fatalf("an overloaded provider is asked twice: %d requests", n)
		}
	})

	t.Run("bad_request", func(t *testing.T) {
		f.set(BadRequest)
		_, err := newProvider(t, nil).Generate(ctx, textRequest())
		if e := errorOf(t, err); e.Kind != llm.ErrBadRequest || e.Message == "" {
			t.Fatalf("bad request: %+v", e)
		}
	})

	t.Run("models", func(t *testing.T) {
		f.set(Models)
		models, err := newProvider(t, nil).Models(ctx)
		if err != nil {
			t.Fatalf("models: %v", err)
		}
		ids := map[string]bool{}
		for _, m := range models {
			ids[m.ID] = true
		}
		for _, want := range s.Want.Models {
			if !ids[want] {
				t.Fatalf("model list %v lacks %s", models, want)
			}
		}
		if reqs := f.recorded(); len(reqs) > 0 {
			if reqs[0].Method != http.MethodGet {
				t.Fatalf("the model list is a GET: %s", reqs[0].Method)
			}
			check(t, Models)
		}
	})

	t.Run("too_large", func(t *testing.T) {
		f.set(TooLarge)
		_, err := newProvider(t, func(c *llm.Config) { c.MaxResponseBytes = 1024 }).Generate(ctx, textRequest())
		if e := errorOf(t, err); e.Kind != llm.ErrTooLarge {
			t.Fatalf("too large: %+v", e)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		f.set(Timeout)
		f.mu.Lock()
		f.delay = 2 * time.Second
		f.mu.Unlock()
		start := time.Now()
		_, err := newProvider(t, func(c *llm.Config) { c.Timeout = 80 * time.Millisecond }).Generate(ctx, textRequest())
		if e := errorOf(t, err); e.Kind != llm.ErrTimeout {
			t.Fatalf("timeout: %+v", e)
		}
		if time.Since(start) > time.Second {
			t.Fatal("the timeout didn't stop the request")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		f.set(Invalid)
		_, err := newProvider(t, nil).Generate(ctx, textRequest())
		if e := errorOf(t, err); e.Kind != llm.ErrInvalid {
			t.Fatalf("unreadable answer: %+v", e)
		}
	})
}

// flatten joins every string of a decoded JSON value.
func flatten(v any) string {
	var b strings.Builder
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			b.WriteString(x)
			b.WriteString("\n")
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(v)
	return b.String()
}
