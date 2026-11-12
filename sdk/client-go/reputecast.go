// Package reputecast is the ReputeCast API client for Go: every operation
// an API key may call, generated from the API's OpenAPI document
// (client.gen.go), with authentication, retries and errors as Go errors.
//
//	rc, err := reputecast.New("https://news.example.com", os.Getenv("REPUTECAST_KEY"))
//	if err != nil {
//		return err
//	}
//	res, err := rc.ListListsWithResponse(ctx, "acme-newsletter")
//	var apiErr *reputecast.Error
//	if errors.As(err, &apiErr) && apiErr.Code == "permission_denied" {
//		…
//	}
//	for _, l := range res.JSON200.Items { … }
//
// A call's error is nil exactly when the API succeeded: an error answer
// (an RFC 9457 problem) is returned as an *Error, so the success field of
// the response (JSON200, JSON201…) is set whenever err is nil.
//
// Webhook deliveries are verified with the webhooks subpackage.
package reputecast

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type options struct {
	maxRetries    int
	timeout       time.Duration
	transport     http.RoundTripper
	clientOptions []ClientOption
}

// Option configures New.
type Option func(*options)

// WithMaxRetries sets how many times a request is sent again after a 429,
// 502, 503 or 504: GET, HEAD, PUT and DELETE requests, and POSTs with an
// Idempotency-Key. 2 by default; 0 turns retries off.
func WithMaxRetries(n int) Option { return func(o *options) { o.maxRetries = n } }

// WithTimeout limits each attempt of a request, from sending it to reading
// the answer: 30 s by default, 0 for none. A context's deadline limits the
// whole call, retries included.
func WithTimeout(d time.Duration) Option { return func(o *options) { o.timeout = d } }

// WithTransport sends requests with rt (a proxy, custom TLS, tests)
// instead of http.DefaultTransport, under the retries.
func WithTransport(rt http.RoundTripper) Option { return func(o *options) { o.transport = rt } }

// WithClientOptions passes options to the generated client
// (WithRequestEditorFn adds headers to every request).
func WithClientOptions(opts ...ClientOption) Option {
	return func(o *options) { o.clientOptions = append(o.clientOptions, opts...) }
}

// New returns a client for a ReputeCast instance: baseURL is its URL
// ("https://news.example.com"), apiKey an API key (rc_live_…) or an MCP
// agent token (rc_agent_…).
func New(baseURL, apiKey string, opts ...Option) (*ClientWithResponses, error) {
	if apiKey == "" {
		return nil, errors.New("reputecast: the API key is empty; create one in Settings › Keys & MCP")
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, fmt.Errorf("reputecast: the base URL is the instance's URL, such as https://news.example.com, not %q", baseURL)
	}
	o := options{maxRetries: 2, timeout: 30 * time.Second}
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxRetries < 0 {
		return nil, errors.New("reputecast: the number of retries is 0 or more")
	}
	if o.transport == nil {
		o.transport = http.DefaultTransport
	}
	authorization := "Bearer " + apiKey
	client := &http.Client{Transport: &retryTransport{next: o.transport, maxRetries: o.maxRetries, timeout: o.timeout, sleep: sleep}}
	all := append([]ClientOption{
		WithHTTPClient(problemDoer{client}),
		WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", authorization)
			req.Header.Set("User-Agent", UserAgent)
			return nil
		}),
	}, o.clientOptions...)
	return NewClientWithResponses(strings.TrimRight(baseURL, "/"), all...)
}

// problemDoer sends requests and returns error answers as *Error.
type problemDoer struct{ client *http.Client }

// maxProblemBytes is the most of an error answer that is read.
const maxProblemBytes = 1 << 20

func (d problemDoer) Do(req *http.Request) (*http.Response, error) {
	res, err := d.client.Do(req)
	if err != nil || res.StatusCode < 400 {
		return res, err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxProblemBytes))
	if err != nil {
		return nil, fmt.Errorf("reputecast: reading the %d answer: %w", res.StatusCode, err)
	}
	return nil, NewError(res.StatusCode, res.Header, body)
}
