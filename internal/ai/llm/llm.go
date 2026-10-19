// Package llm talks to the AI model a workspace connects (design expansion,
// phase 3): Anthropic, OpenAI, Google Gemini, Azure OpenAI and servers that
// speak the OpenAI API (Ollama, vLLM, LM Studio). It is plain HTTP with the
// standard library: each provider is a small adapter that turns a Request
// into the provider's JSON and its answer (or error) back into a Response.
//
// Every adapter passes the shared contract suite in llm/contracttest against
// recorded answers; tests never reach the network.
//
// Requests have a timeout and a response size limit, and the client the
// caller passes decides which addresses may be dialed (internal/netguard).
// What goes into a Request is the caller's business: internal/ai builds
// prompts from templates with the person's content delimited, and never
// puts contact data in them.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Provider kinds.
const (
	Anthropic  = "anthropic"
	OpenAI     = "openai"
	Google     = "google"
	Azure      = "azure"
	Compatible = "compatible"
)

// Kinds lists the providers in the catalog's order.
var Kinds = []string{Anthropic, OpenAI, Google, Azure, Compatible}

// Valid reports whether kind is a provider this package can talk to.
func Valid(kind string) bool {
	for _, k := range Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// Defaults for requests.
const (
	DefaultTimeout          = 60 * time.Second
	DefaultMaxResponseBytes = 4 << 20
	DefaultMaxTokens        = 2048
	// DefaultAzureAPIVersion is Azure OpenAI's generally available API
	// version for chat completions with images and JSON output.
	DefaultAzureAPIVersion = "2024-10-21"
)

// Config says how to reach a model.
type Config struct {
	Kind string
	// BaseURL overrides the provider's API root: the Azure resource endpoint
	// (https://acme.openai.azure.com) or a compatible server's address with
	// its /v1 (http://ollama.acme.internal:11434/v1). Empty is the
	// provider's public API.
	BaseURL string
	// APIKey is optional for compatible servers.
	APIKey string
	Model  string
	// Deployment and APIVersion are Azure OpenAI's.
	Deployment string
	APIVersion string
	// HTTP is the client to use (the SSRF guard's in production).
	HTTP *http.Client
	// Timeout bounds one request, retries included (DefaultTimeout).
	Timeout time.Duration
	// MaxResponseBytes caps the answer read (DefaultMaxResponseBytes).
	MaxResponseBytes int64
	// Retries is how many times an overloaded or unavailable provider is
	// asked again (1 by default: "returned an error twice"); RetryWait is
	// the pause before each retry (1 s).
	Retries   *int
	RetryWait time.Duration
}

// Image is an image the model should look at.
type Image struct {
	MediaType string // image/png, image/jpeg, image/gif
	Data      []byte
}

// Part is a piece of a message: text or an image.
type Part struct {
	Text  string
	Image *Image
}

// Message is a turn of the conversation.
type Message struct {
	Role  string // user or assistant
	Parts []Part
}

// Request is one call to the model.
type Request struct {
	// System is the instruction the model follows.
	System   string
	Messages []Message
	// MaxTokens bounds the answer (DefaultMaxTokens).
	MaxTokens int
	// JSON asks for one JSON object; providers with structured output get
	// Schema (a JSON Schema) named SchemaName.
	JSON       bool
	Schema     map[string]any
	SchemaName string
	// Task is the request as data, for in-process models (the demo's
	// recorded model and tests). Adapters never send it.
	Task any
}

// Stop reasons, normalized.
const (
	StopEnd     = "end"
	StopLength  = "length"
	StopRefusal = "refusal"
	StopOther   = "other"
)

// Response is the model's answer.
type Response struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
	StopReason   string
	// Refused is true when the provider says the model declined (Anthropic's
	// "refusal" stop reason, OpenAI's refusal field, Gemini's safety block);
	// Text is then the model's own words, when it gave any.
	Refused bool
}

// Model is one model a provider offers.
type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Provider is a connected model.
type Provider interface {
	// Generate sends one request.
	Generate(ctx context.Context, req Request) (Response, error)
	// Models lists the models the key can use.
	Models(ctx context.Context) ([]Model, error)
}

// ErrorKind classifies what went wrong.
type ErrorKind string

const (
	// ErrAuth: the provider rejected the key (401, 403, Gemini's
	// API_KEY_INVALID).
	ErrAuth ErrorKind = "auth"
	// ErrRateLimit: too many requests; RetryAfter says when to try again.
	ErrRateLimit ErrorKind = "rate_limit"
	// ErrOverloaded: the provider is overloaded (Anthropic 529, 503).
	ErrOverloaded ErrorKind = "overloaded"
	// ErrUnavailable: a server error (5xx).
	ErrUnavailable ErrorKind = "unavailable"
	// ErrBadRequest: the provider refused the request itself (400, 404, 422):
	// an unknown model, a request it can't take.
	ErrBadRequest ErrorKind = "bad_request"
	// ErrTimeout: no answer within the timeout.
	ErrTimeout ErrorKind = "timeout"
	// ErrNetwork: the provider couldn't be reached (DNS, refused connection,
	// an address the SSRF guard blocks).
	ErrNetwork ErrorKind = "network"
	// ErrTooLarge: the answer was over the size limit.
	ErrTooLarge ErrorKind = "too_large"
	// ErrInvalid: the answer couldn't be read.
	ErrInvalid ErrorKind = "invalid"
)

// Error is a failed call, with what the provider said.
type Error struct {
	Kind   ErrorKind
	Status int
	// Code is the provider's error type ("authentication_error",
	// "rate_limit_exceeded", "RESOURCE_EXHAUSTED").
	Code string
	// Message is the provider's message ("invalid x-api-key").
	Message    string
	RetryAfter time.Duration
	// Attempts is how many times the request was sent.
	Attempts int
	// Blocked is set when the SSRF guard refused the address.
	Blocked bool
	Err     error
}

func (e *Error) Error() string {
	var b strings.Builder
	if e.Status > 0 {
		b.WriteString(strconv.Itoa(e.Status))
	}
	if e.Code != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(e.Code)
	}
	if e.Message != "" {
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString(e.Message)
	}
	if b.Len() == 0 {
		if e.Err != nil {
			return e.Err.Error()
		}
		return string(e.Kind)
	}
	return b.String()
}

func (e *Error) Unwrap() error { return e.Err }

// AsError returns err as an *Error.
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// New returns the adapter of cfg.Kind.
func New(cfg Config) (Provider, error) {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: DefaultTimeout}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = DefaultMaxResponseBytes
	}
	if cfg.RetryWait <= 0 {
		cfg.RetryWait = time.Second
	}
	switch cfg.Kind {
	case Anthropic:
		return &anthropic{cfg: cfg}, nil
	case OpenAI, Azure, Compatible:
		if cfg.Kind == Azure && (strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Deployment) == "") {
			return nil, errors.New("azure OpenAI needs the resource endpoint and the deployment")
		}
		if cfg.Kind == Compatible && strings.TrimSpace(cfg.BaseURL) == "" {
			return nil, errors.New("an OpenAI-compatible server needs its address")
		}
		return &openAI{cfg: cfg}, nil
	case Google:
		return &gemini{cfg: cfg}, nil
	}
	return nil, fmt.Errorf("unknown AI provider %q", cfg.Kind)
}

// retries returns how many times to retry an overloaded provider.
func (c Config) retries() int {
	if c.Retries != nil {
		return *c.Retries
	}
	return 1
}

// base returns the configured base URL or def, without a trailing slash.
func (c Config) base(def string) string {
	b := strings.TrimSpace(c.BaseURL)
	if b == "" {
		b = def
	}
	return strings.TrimRight(b, "/")
}

// maxTokens returns req.MaxTokens or the default.
func maxTokens(req Request) int {
	if req.MaxTokens > 0 {
		return req.MaxTokens
	}
	return DefaultMaxTokens
}

// call is one HTTP exchange: build sends the request, classify reads a
// non-2xx answer. Overloaded and unavailable answers are retried.
type call struct {
	cfg      Config
	method   string
	url      string
	header   http.Header
	body     any
	classify func(status int, h http.Header, body []byte) *Error
}

// do runs the call within the configured timeout and returns the body of a
// 2xx answer.
func (c call) do(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	var payload []byte
	if c.body != nil {
		b, err := json.Marshal(c.body)
		if err != nil {
			return nil, &Error{Kind: ErrInvalid, Err: err}
		}
		payload = b
	}
	attempts := 0
	for {
		attempts++
		body, err := c.once(ctx, payload)
		if err == nil {
			return body, nil
		}
		e, _ := AsError(err)
		if e != nil {
			e.Attempts = attempts
		}
		retry := e != nil && (e.Kind == ErrOverloaded || e.Kind == ErrUnavailable) && attempts <= c.cfg.retries()
		if !retry {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(c.cfg.RetryWait):
		}
	}
}

func (c call) once(ctx context.Context, payload []byte) ([]byte, error) {
	var rd io.Reader
	if payload != nil {
		rd = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, c.method, c.url, rd)
	if err != nil {
		return nil, &Error{Kind: ErrNetwork, Err: err, Message: "the address isn't valid"}
	}
	for k, vs := range c.header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return nil, transportError(ctx, err)
	}
	defer res.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(io.LimitReader(res.Body, c.cfg.MaxResponseBytes+1))
	if err != nil {
		return nil, transportError(ctx, err)
	}
	if int64(len(body)) > c.cfg.MaxResponseBytes {
		return nil, &Error{Kind: ErrTooLarge, Status: res.StatusCode, Message: fmt.Sprintf("the answer is over %d MB", c.cfg.MaxResponseBytes>>20)}
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return body, nil
	}
	e := c.classify(res.StatusCode, res.Header, body)
	if e.RetryAfter == 0 {
		e.RetryAfter = retryAfter(res.Header)
	}
	return nil, e
}

// transportError classifies a failure before an answer arrived.
func transportError(ctx context.Context, err error) *Error {
	var ne net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()):
		return &Error{Kind: ErrTimeout, Err: err, Message: "no answer in time"}
	case errors.Is(err, context.Canceled):
		return &Error{Kind: ErrNetwork, Err: err, Message: "cancelled"}
	}
	msg := err.Error()
	blocked := strings.Contains(msg, "address not allowed")
	if i := strings.Index(msg, "address not allowed: "); i >= 0 {
		msg = msg[i+len("address not allowed: "):]
	}
	return &Error{Kind: ErrNetwork, Err: err, Message: msg, Blocked: blocked}
}

// retryAfter reads Retry-After (seconds or an HTTP date) and Anthropic's
// and OpenAI's reset headers.
func retryAfter(h http.Header) time.Duration {
	if v := strings.TrimSpace(h.Get("Retry-After")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 0 {
			return time.Duration(n * float64(time.Second))
		}
		if t, err := http.ParseTime(v); err == nil {
			if d := time.Until(t); d > 0 {
				return d.Round(time.Second)
			}
		}
	}
	if v := strings.TrimSpace(h.Get("x-ratelimit-reset-requests")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 0
}

// kindOf maps an HTTP status to an error kind.
func kindOf(status int) ErrorKind {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrAuth
	case status == http.StatusTooManyRequests:
		return ErrRateLimit
	case status == 529 || status == http.StatusServiceUnavailable:
		return ErrOverloaded
	case status >= 500:
		return ErrUnavailable
	}
	return ErrBadRequest
}

// imageDataURL is an image as a data: URL (OpenAI's image_url).
func imageDataURL(img *Image) string {
	return "data:" + img.MediaType + ";base64," + base64Std(img.Data)
}

// FirstJSON returns the first JSON object in s: models asked for JSON
// sometimes wrap it in a code fence or a sentence.
func FirstJSON(s string) (json.RawMessage, bool) {
	s = strings.TrimSpace(s)
	if json.Valid([]byte(s)) && strings.HasPrefix(s, "{") {
		return json.RawMessage(s), true
	}
	start := strings.Index(s, "{")
	for start >= 0 {
		depth, inStr, esc := 0, false, false
		for i := start; i < len(s); i++ {
			ch := s[i]
			if inStr {
				switch {
				case esc:
					esc = false
				case ch == '\\':
					esc = true
				case ch == '"':
					inStr = false
				}
				continue
			}
			switch ch {
			case '"':
				inStr = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					cand := s[start : i+1]
					if json.Valid([]byte(cand)) {
						return json.RawMessage(cand), true
					}
					i = len(s)
				}
			}
		}
		next := strings.Index(s[start+1:], "{")
		if next < 0 {
			break
		}
		start += next + 1
	}
	return nil, false
}
