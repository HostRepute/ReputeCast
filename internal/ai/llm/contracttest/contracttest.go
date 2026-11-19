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
//
// Streamed answers (llm.Streamer) have their own recordings, whose
// "stream" is the raw text/event-stream body in the pieces the server
// wrote: text, two tool calls with arguments in fragments, a refusal, the
// token limit, an error event after the first text and, for a server that
// may omit it, no usage. From the text stream the suite makes a cut
// connection, silence (before and during the answer), a slow stream, one
// over the size limit in small lines, one with a line over the line limit,
// and errors that quote the key.
package contracttest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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

// Recorded streams.
const (
	StreamText    = "stream_text"
	StreamTools   = "stream_tools"
	StreamRefusal = "stream_refusal"
	StreamLength  = "stream_length"
	StreamError   = "stream_error"
	StreamNoUsage = "stream_no_usage"
)

// Scenarios the suite makes up itself.
const (
	TooLarge = "too_large"
	Timeout  = "timeout"
	Invalid  = "invalid"

	streamCut      = "stream_cut"       // stream_text's first half, then the end
	streamHold     = "stream_hold"      // its first half, then silence
	streamSilent   = "stream_silent"    // no answer at all
	streamSlow     = "stream_slow"      // a pause between pieces
	streamPadded   = "stream_padded"    // comment lines between its events
	streamLongLine = "stream_long_line" // a line over llm.MaxStreamLineBytes
	streamKey      = "stream_key"       // stream_error quoting the key
	authKey        = "auth_key"         // auth quoting the key
)

// Streaming reports whether a scenario's request asks for a stream.
func Streaming(scenario string) bool { return strings.HasPrefix(scenario, "stream_") }

// suiteKey is the key of the scenarios where the provider quotes it.
const suiteKey = "sk-contract-6d1e-QUOTED-KEY"

// Markers the suite puts in every request, to find them in what was sent.
const (
	SystemMarker = "SYSTEM-7f3a: follow the house rules."
	UserMarker   = "<content>USER-91c2 This week: on-call numbers {{ first_name | default: \"there\" }}</content>"
)

// TestImage is the image of the image scenario (a 1×1 PNG).
var TestImage = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0x1f, 0x15, 0xc4, 0x89}

// Fixture is a recorded answer: a JSON body, or a stream of server-sent
// events (text/event-stream) in the pieces the server wrote; the fake API
// flushes after each.
type Fixture struct {
	Status int               `json:"status"`
	Header map[string]string `json:"header"`
	Body   json.RawMessage   `json:"body"`
	Stream []string          `json:"stream"`
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
	// StreamErrorMessage is part of the provider's message in stream_error.
	StreamErrorMessage string
	// ToolArgsWhole is set when the provider sends each tool call's
	// arguments in one piece (Gemini); others split them across chunks.
	ToolArgsWhole bool
	// Signatures are the thought signatures a stream's Message keeps, by
	// scenario.
	Signatures map[string][]string
}

type fake struct {
	mu       sync.Mutex
	scenario string
	// seq, when set, is the scenario of each request in turn (the last
	// repeats).
	seq      []string
	fixtures map[string]Fixture
	requests []Recorded
	delay    time.Duration
	// gap is the pause between a stream's pieces.
	gap time.Duration
	// gone is closed when a held stream's client goes away.
	gone chan struct{}
}

func (f *fake) set(s string) {
	f.mu.Lock()
	f.scenario, f.seq, f.requests, f.delay, f.gap, f.gone = s, nil, nil, 0, 0, make(chan struct{})
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
	s, delay, gap, gone := f.scenario, f.delay, f.gap, f.gone
	if len(f.seq) > 0 {
		s = f.seq[min(len(f.requests), len(f.seq))-1]
	}
	fx, ok := f.fixtures[s]
	text := f.fixtures[StreamText].Stream
	f.mu.Unlock()
	if delay > 0 {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(delay):
		}
	}
	half := len(text) / 2
	hold := false
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
	case streamSilent:
		<-r.Context().Done()
		return
	case streamCut:
		// The connection ends cleanly halfway through an event: nothing
		// but the provider's own last event says the answer is whole.
		next := text[half]
		fx, ok = Fixture{Stream: append(append([]string(nil), text[:half]...), next[:len(next)/2])}, true
	case streamHold:
		fx, ok, hold = Fixture{Stream: text[:half]}, true, true
	case streamSlow:
		fx, ok = Fixture{Stream: text}, true
	case streamPadded:
		// Many small lines: each is far under the line limit, together
		// they are over the stream's.
		pieces := []string{text[0]}
		for range 128 {
			pieces = append(pieces, ": "+strings.Repeat("x", 97)+"\n")
		}
		fx, ok = Fixture{Stream: append(pieces, text[1:]...)}, true
	case streamLongLine:
		fx, ok = Fixture{Stream: append([]string{text[0], ": " + strings.Repeat("x", llm.MaxStreamLineBytes) + "\n"}, text[1:]...)}, true
	case streamKey:
		fx, ok = f.fixtures[StreamError]
		pieces := make([]string, len(fx.Stream))
		for i, p := range fx.Stream {
			pieces[i] = quoteKey(p)
		}
		fx.Stream = pieces
	case authKey:
		fx, ok = f.fixtures[Auth]
		fx.Body = json.RawMessage(quoteKey(string(fx.Body)))
	}
	if !ok {
		http.Error(w, "no recorded answer for "+s, http.StatusNotImplemented)
		return
	}
	for k, v := range fx.Header {
		w.Header().Set(k, v)
	}
	status := fx.Status
	if status == 0 {
		status = http.StatusOK
	}
	if fx.Stream == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(fx.Body)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(status)
	flusher, _ := w.(http.Flusher)
	for i, p := range fx.Stream {
		if i > 0 && gap > 0 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(gap):
			}
		}
		if _, err := io.WriteString(w, p); err != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	if hold {
		select {
		case <-r.Context().Done():
			close(gone)
		case <-time.After(10 * time.Second):
		}
	}
}

// quoteKey adds the key to every "message" of a JSON answer, or of each
// JSON event of a piece of a stream.
func quoteKey(piece string) string {
	quote := func(raw string, indent string) (string, bool) {
		var v any
		if json.Unmarshal([]byte(raw), &v) != nil {
			return raw, false
		}
		var walk func(any)
		walk = func(v any) {
			switch x := v.(type) {
			case map[string]any:
				for k, e := range x {
					if m, ok := e.(string); ok && k == "message" {
						x[k] = m + " (the key was " + suiteKey + ")"
						continue
					}
					walk(e)
				}
			case []any:
				for _, e := range x {
					walk(e)
				}
			}
		}
		walk(v)
		b, _ := json.MarshalIndent(v, "", indent)
		return string(b), true
	}
	if out, ok := quote(strings.TrimSpace(piece), "  "); ok {
		return out + "\n"
	}
	lines := strings.SplitAfter(piece, "\n")
	for i, l := range lines {
		data, ok := strings.CutPrefix(l, "data: ")
		if !ok {
			continue
		}
		value := strings.TrimRight(data, "\r\n")
		if out, ok := quote(value, ""); ok {
			lines[i] = "data: " + strings.ReplaceAll(out, "\n", "") + data[len(value):]
		}
	}
	return strings.Join(lines, "")
}

// required are the scenarios every provider has a recorded answer for. A
// server that speaks the OpenAI API may not flag refusals or limit rates:
// those scenarios run when their answer is recorded.
var required = map[string]bool{Text: true, Image: true, JSON: true, Auth: true, Overloaded: true, BadRequest: true, Models: true, StreamText: true, StreamTools: true, StreamLength: true, StreamError: true}

func load(t *testing.T, dir string) map[string]Fixture {
	t.Helper()
	out := map[string]Fixture{}
	for _, s := range []string{Text, Image, JSON, Refusal, Length, Auth, RateLimit, Overloaded, BadRequest, Models, StreamText, StreamTools, StreamRefusal, StreamLength, StreamError, StreamNoUsage} {
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
		if Streaming(s) && len(fx.Stream) < 4 {
			t.Fatalf("recorded answer %s: a stream needs at least four pieces", s)
		}
		out[s] = fx
	}
	return out
}

func textRequest() llm.Request {
	return llm.Request{System: SystemMarker, MaxTokens: 64, Messages: []llm.Message{{Role: "user", Parts: []llm.Part{{Text: UserMarker}}}}}
}

// toolsRequest offers read_draft; stream_tools calls it for draft_a, then
// draft_b.
func toolsRequest() llm.Request {
	req := textRequest()
	req.Tools = []llm.Tool{{Name: "read_draft", Description: "Read a selected draft.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"handle": map[string]any{"type": "string"}}, "required": []string{"handle"}, "additionalProperties": false}}}
	return req
}

// deltas collects what a stream emitted.
type deltas struct {
	n     int
	texts int
	text  strings.Builder
	// starts are the tool calls begun, args and frags their arguments.
	starts []llm.ToolCall
	args   []string
	frags  []int
	orphan bool // arguments before any call began
}

func (d *deltas) emit(x llm.Delta) error {
	d.n++
	if x.Text != "" {
		d.texts++
		d.text.WriteString(x.Text)
	}
	if x.ToolCallStart != nil {
		d.starts = append(d.starts, *x.ToolCallStart)
		d.args = append(d.args, "")
		d.frags = append(d.frags, 0)
	}
	if x.ToolArgs != "" {
		if len(d.starts) == 0 {
			d.orphan = true
			return nil
		}
		d.args[len(d.args)-1] += x.ToolArgs
		d.frags[len(d.frags)-1]++
	}
	return nil
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

	runStreams(t, s, f, newProvider, check, errorOf)
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

// runStreams is the suite of llm.Streamer.
func runStreams(t *testing.T, s Suite, f *fake, newProvider func(*testing.T, func(*llm.Config)) llm.Provider, check func(*testing.T, string), errorOf func(*testing.T, error) *llm.Error) {
	ctx := context.Background()
	streamer := func(t *testing.T, mutate func(*llm.Config)) llm.Streamer {
		t.Helper()
		p, ok := newProvider(t, mutate).(llm.Streamer)
		if !ok {
			t.Fatalf("%s doesn't stream", s.Kind)
		}
		return p
	}
	stream := func(t *testing.T, scenario string, req llm.Request, mutate func(*llm.Config)) (llm.Response, *deltas, error) {
		t.Helper()
		// A server that speaks the OpenAI API may not flag refusals, and
		// only such a server may omit the usage.
		if _, ok := f.fixtures[scenario]; !ok && (scenario == StreamRefusal || scenario == StreamNoUsage) {
			t.Skipf("no recorded %s answer for %s", scenario, s.Kind)
		}
		f.set(scenario)
		d := &deltas{}
		res, err := streamer(t, mutate).Stream(ctx, req, d.emit)
		return res, d, err
	}
	requests := func(t *testing.T, want int) {
		t.Helper()
		if n := len(f.recorded()); n != want {
			t.Fatalf("%d requests, want %d", n, want)
		}
	}
	// emitted checks the invariant of every stream: what was shown is the
	// answer's text, thoughts are never shown, and each tool call's
	// fragments are its arguments.
	emitted := func(t *testing.T, scenario string, res llm.Response, d *deltas) {
		t.Helper()
		if d.text.String() != res.Text {
			t.Fatalf("the deltas say %q, the answer %q", d.text.String(), res.Text)
		}
		if d.orphan {
			t.Fatal("tool arguments arrived before their call")
		}
		var calls []llm.ToolCall
		for _, p := range res.Message.Parts {
			if p.Thought && p.Text != "" && strings.Contains(d.text.String(), p.Text) {
				t.Fatalf("a thought was shown: %q", p.Text)
			}
			if p.ToolCall != nil {
				calls = append(calls, *p.ToolCall)
			}
		}
		if len(calls) > len(d.starts) {
			t.Fatalf("%d calls, %d begun in the stream", len(calls), len(d.starts))
		}
		for i, c := range calls {
			if d.starts[i].ID != c.ID || d.starts[i].Name != c.Name || d.starts[i].Arguments != nil {
				t.Fatalf("call %d began as %+v, ended as %+v", i, d.starts[i], c)
			}
			if d.args[i] != string(c.Arguments) {
				t.Fatalf("call %d: fragments %q, arguments %s", i, d.args[i], c.Arguments)
			}
		}
		for _, sig := range s.Want.Signatures[scenario] {
			kept := false
			for _, p := range res.Message.Parts {
				kept = kept || p.ThoughtSignature == sig
			}
			if !kept {
				t.Fatalf("the thought signature %s is lost: %+v", sig, res.Message.Parts)
			}
		}
	}
	// billable: an error once the answer began may have been charged; it
	// never reads as a request the provider refused before running it.
	billable := func(t *testing.T, e *llm.Error, res llm.Response) {
		t.Helper()
		if e.Kind == llm.ErrAuth || e.Kind == llm.ErrRateLimit || e.Kind == llm.ErrBadRequest || e.Attempts != 1 {
			t.Fatalf("an error after the answer began: %+v", e)
		}
		if res.UsageKnown {
			t.Fatalf("usage the provider never reported: %+v", res)
		}
	}
	prefix := func(t *testing.T, res llm.Response) {
		t.Helper()
		if !strings.HasPrefix(s.Want.Text, res.Text) {
			t.Fatalf("the answer so far %q isn't the start of %q", res.Text, s.Want.Text)
		}
	}

	t.Run("stream_text", func(t *testing.T) {
		f.set(Text)
		want, err := newProvider(t, nil).Generate(ctx, textRequest())
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		res, d, err := stream(t, StreamText, textRequest(), nil)
		if err != nil {
			t.Fatalf("stream: %v", err)
		}
		check(t, StreamText)
		emitted(t, StreamText, res, d)
		if d.texts < 2 {
			t.Fatalf("the text came in %d deltas", d.texts)
		}
		// The same answer as Generate's, usage and replayable turn included.
		if !reflect.DeepEqual(res, want) {
			t.Fatalf("stream:   %+v\ngenerate: %+v", res, want)
		}
		if res.Text != s.Want.Text || !res.UsageKnown || res.InputTokens != s.Want.InputTokens || res.OutputTokens != s.Want.OutputTokens || res.StopReason != llm.StopEnd {
			t.Fatalf("answer: %+v", res)
		}
	})

	t.Run("stream_tools", func(t *testing.T) {
		res, d, err := stream(t, StreamTools, toolsRequest(), nil)
		if err != nil {
			t.Fatalf("stream: %v", err)
		}
		check(t, StreamTools)
		emitted(t, StreamTools, res, d)
		if res.StopReason != llm.StopTools || !res.UsageKnown || res.Refused {
			t.Fatalf("answer: %+v", res)
		}
		results := llm.Message{Role: "tool"}
		var calls []llm.ToolCall
		for _, p := range res.Message.Parts {
			if p.ToolCall != nil {
				calls = append(calls, *p.ToolCall)
				results.Parts = append(results.Parts, llm.Part{ToolResult: &llm.ToolResult{ID: p.ToolCall.ID, Name: p.ToolCall.Name, Content: json.RawMessage(`{"subject":"First draft"}`)}})
			}
		}
		if len(calls) != 2 {
			t.Fatalf("calls: %+v", calls)
		}
		split := false
		for i, want := range []string{`{"handle":"draft_a"}`, `{"handle":"draft_b"}`} {
			var got, exp any
			if json.Unmarshal(calls[i].Arguments, &got) != nil || json.Unmarshal([]byte(want), &exp) != nil || !reflect.DeepEqual(got, exp) || calls[i].Name != "read_draft" {
				t.Fatalf("call %d: %s %s", i, calls[i].Name, calls[i].Arguments)
			}
			split = split || d.frags[i] > 1
		}
		if !split && !s.Want.ToolArgsWhole {
			t.Fatalf("no call's arguments came in fragments: %v", d.frags)
		}
		// The turn replays: the next request carries it and the results.
		req := toolsRequest()
		req.Messages = append(req.Messages, res.Message, results)
		if _, _, err := stream(t, StreamText, req, nil); err != nil {
			t.Fatalf("replaying the turn: %v", err)
		}
		requests(t, 1)
		// Calls nobody offered are refused, their usage kept.
		res, _, err = stream(t, StreamTools, textRequest(), nil)
		if err == nil || !res.UsageKnown {
			t.Fatalf("unsolicited tools: %+v %v", res, err)
		}
	})

	t.Run("stream_refusal", func(t *testing.T) {
		res, d, err := stream(t, StreamRefusal, textRequest(), nil)
		if err != nil {
			t.Fatalf("stream: %v", err)
		}
		emitted(t, StreamRefusal, res, d)
		if !res.Refused || res.StopReason != llm.StopRefusal || !res.UsageKnown {
			t.Fatalf("not a refusal: %+v", res)
		}
	})

	t.Run("stream_length", func(t *testing.T) {
		res, d, err := stream(t, StreamLength, textRequest(), nil)
		if err != nil {
			t.Fatalf("stream: %v", err)
		}
		emitted(t, StreamLength, res, d)
		if res.StopReason != llm.StopLength || res.Text == "" || !res.UsageKnown {
			t.Fatalf("stop reason: %+v", res)
		}
	})

	t.Run("stream_no_usage", func(t *testing.T) {
		res, d, err := stream(t, StreamNoUsage, textRequest(), nil)
		if err != nil {
			t.Fatalf("stream: %v", err)
		}
		emitted(t, StreamNoUsage, res, d)
		if res.Text != s.Want.Text || res.UsageKnown || res.InputTokens != 0 || res.OutputTokens != 0 {
			t.Fatalf("no usage reported must read as unknown, never as zero: %+v", res)
		}
	})

	t.Run("stream_error", func(t *testing.T) {
		res, d, err := stream(t, StreamError, textRequest(), nil)
		e := errorOf(t, err)
		emitted(t, StreamError, res, d)
		billable(t, e, res)
		if res.Text == "" || !strings.Contains(e.Error(), s.Want.StreamErrorMessage) {
			t.Fatalf("error event: %+v (%s), answer so far %q", e, e.Error(), res.Text)
		}
		// Not asked again, even when the provider says it is overloaded.
		requests(t, 1)
	})

	t.Run("stream_cut", func(t *testing.T) {
		res, d, err := stream(t, streamCut, textRequest(), nil)
		e := errorOf(t, err)
		emitted(t, streamCut, res, d)
		billable(t, e, res)
		prefix(t, res)
		if e.Kind != llm.ErrNetwork || d.texts == 0 {
			t.Fatalf("cut stream: %+v, answer so far %q", e, res.Text)
		}
		requests(t, 1)
	})

	t.Run("stream_idle", func(t *testing.T) {
		start := time.Now()
		res, d, err := stream(t, streamHold, textRequest(), func(c *llm.Config) { c.IdleTimeout, c.Timeout = 100*time.Millisecond, 10*time.Second })
		e := errorOf(t, err)
		if e.Kind != llm.ErrTimeout || time.Since(start) > 2*time.Second {
			t.Fatalf("silence: %+v after %s", e, time.Since(start))
		}
		emitted(t, streamHold, res, d)
		billable(t, e, res)
		prefix(t, res)
		requests(t, 1)
	})

	t.Run("stream_idle_before_answer", func(t *testing.T) {
		start := time.Now()
		_, d, err := stream(t, streamSilent, textRequest(), func(c *llm.Config) { c.IdleTimeout, c.Timeout = 100*time.Millisecond, 10*time.Second })
		if e := errorOf(t, err); e.Kind != llm.ErrTimeout || time.Since(start) > 2*time.Second || d.n != 0 {
			t.Fatalf("silence: %+v after %s", e, time.Since(start))
		}
		requests(t, 1)
	})

	t.Run("stream_slow", func(t *testing.T) {
		// Pauses longer than none, shorter than the idle timeout: the
		// timeout counts silence, not the whole answer.
		f.set(streamSlow)
		f.mu.Lock()
		f.gap = 100 * time.Millisecond
		f.mu.Unlock()
		d := &deltas{}
		start := time.Now()
		res, err := streamer(t, func(c *llm.Config) { c.IdleTimeout = 250 * time.Millisecond }).Stream(ctx, textRequest(), d.emit)
		if err != nil || res.Text != s.Want.Text || time.Since(start) < 250*time.Millisecond {
			t.Fatalf("slow stream: %+v %v after %s", res, err, time.Since(start))
		}
	})

	t.Run("stream_timeout", func(t *testing.T) {
		// The request's timeout bounds the whole stream.
		f.set(streamSlow)
		f.mu.Lock()
		f.gap = 80 * time.Millisecond
		f.mu.Unlock()
		d := &deltas{}
		res, err := streamer(t, func(c *llm.Config) { c.Timeout = 150 * time.Millisecond }).Stream(ctx, textRequest(), d.emit)
		e := errorOf(t, err)
		if e.Kind != llm.ErrTimeout {
			t.Fatalf("timeout: %+v", e)
		}
		emitted(t, streamSlow, res, d)
		billable(t, e, res)
	})

	t.Run("stream_too_large", func(t *testing.T) {
		limit := func(c *llm.Config) { c.MaxResponseBytes = 8 << 10 }
		if _, _, err := stream(t, StreamText, textRequest(), limit); err != nil {
			t.Fatalf("the text stream is under the limit: %v", err)
		}
		res, _, err := stream(t, streamPadded, textRequest(), limit)
		if e := errorOf(t, err); e.Kind != llm.ErrTooLarge || res.UsageKnown {
			t.Fatalf("the limit is the whole stream's: %+v %+v", e, res)
		}
		res, _, err = stream(t, streamLongLine, textRequest(), nil)
		if e := errorOf(t, err); e.Kind != llm.ErrTooLarge || res.UsageKnown {
			t.Fatalf("a line over the line limit: %+v %+v", e, res)
		}
	})

	t.Run("stream_emit_error", func(t *testing.T) {
		// The stream goes silent after its first half; the person leaves at
		// the first text, long before any timeout.
		f.set(streamHold)
		stop := errors.New("the person left")
		stopped := false
		start := time.Now()
		_, err := streamer(t, func(c *llm.Config) { c.IdleTimeout, c.Timeout = 10*time.Second, 10*time.Second }).Stream(ctx, textRequest(), func(d llm.Delta) error {
			if stopped {
				t.Error("emit was called after it failed")
			}
			if d.Text != "" {
				stopped = true
				return stop
			}
			return nil
		})
		if !errors.Is(err, stop) || !stopped || time.Since(start) > 2*time.Second {
			t.Fatalf("emit's error: %v after %s", err, time.Since(start))
		}
		f.mu.Lock()
		gone := f.gone
		f.mu.Unlock()
		select {
		case <-gone:
		case <-time.After(2 * time.Second):
			t.Fatal("the connection stayed open")
		}
	})

	t.Run("stream_overloaded", func(t *testing.T) {
		// Before the stream begins, an overloaded provider is asked again
		// (Config.Retries is 1), then reported.
		f.set(Overloaded)
		d := &deltas{}
		_, err := streamer(t, nil).Stream(ctx, textRequest(), d.emit)
		if e := errorOf(t, err); e.Kind != llm.ErrOverloaded || e.Status != s.Want.OverloadedStatus || e.Attempts != 2 || d.n != 0 {
			t.Fatalf("overloaded: %+v", e)
		}
		requests(t, 2)
		f.set(Overloaded)
		f.mu.Lock()
		f.seq = []string{Overloaded, StreamText}
		f.mu.Unlock()
		d = &deltas{}
		res, err := streamer(t, nil).Stream(ctx, textRequest(), d.emit)
		if err != nil || d.text.String() != s.Want.Text || res.Text != s.Want.Text {
			t.Fatalf("after one overloaded answer: %+v %v, shown %q", res, err, d.text.String())
		}
		requests(t, 2)
	})

	t.Run("stream_key", func(t *testing.T) {
		key := func(c *llm.Config) { c.APIKey = suiteKey }
		for _, scenario := range []string{authKey, streamKey} {
			f.set(scenario)
			d := &deltas{}
			_, err := streamer(t, key).Stream(ctx, textRequest(), d.emit)
			e := errorOf(t, err)
			if strings.Contains(err.Error(), suiteKey) || strings.Contains(e.Message, suiteKey) || !strings.Contains(e.Message, "[key]") {
				t.Fatalf("%s: the error quotes the key: %q", scenario, err.Error())
			}
		}
	})

	t.Run("stream_invalid", func(t *testing.T) {
		f.set(Invalid)
		d := &deltas{}
		_, err := streamer(t, nil).Stream(ctx, textRequest(), d.emit)
		if e := errorOf(t, err); e.Kind != llm.ErrInvalid {
			t.Fatalf("a page instead of a stream: %+v", e)
		}
	})
}
