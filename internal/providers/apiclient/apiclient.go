// Package apiclient is the small JSON-over-HTTP client the API adapters
// (Postmark, Postal…) share: timeouts, a response size limit, and mapping
// of HTTP failures to providers.CallError so health checks and the sending
// pipeline can tell an outage from throttling from rejected credentials.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

// MaxResponse caps how much of a provider response is read.
const MaxResponse = 8 << 20

// DefaultTimeout applies when the HTTP client has none.
const DefaultTimeout = 30 * time.Second

// Client calls one provider API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Header  http.Header
	// Decode extracts the provider's error code and message from an error
	// body; nil uses the raw body as the message.
	Decode func(status int, body []byte) (code, message string)
}

// New returns a client for base with a default HTTP client when hc is nil.
func New(base string, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: DefaultTimeout}
	}
	return &Client{BaseURL: strings.TrimRight(base, "/"), HTTP: hc, Header: http.Header{}}
}

// Do sends method path with in as JSON (nil for none) and decodes a 2xx
// response into out (nil to ignore). op names the call for errors.
func (c *Client) Do(ctx context.Context, op, method, path string, in, out any) error {
	_, err := c.DoRaw(ctx, op, method, path, in, out)
	return err
}

// DoRaw is Do that also returns the raw response body.
func (c *Client) DoRaw(ctx context.Context, op, method, path string, in, out any) ([]byte, error) {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, &providers.CallError{Op: op, Message: err.Error(), Cause: err}
	}
	for k, vs := range c.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "ReputeCast")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, NetworkError(op, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponse))
	if err != nil {
		return nil, NetworkError(op, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return raw, c.statusError(op, resp, raw)
	}
	if out != nil && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, &providers.CallError{Op: op, Status: resp.StatusCode, Message: "unexpected response: " + err.Error(), Temporary: true, Cause: err}
		}
	}
	return raw, nil
}

func (c *Client) statusError(op string, resp *http.Response, raw []byte) error {
	code, msg := "", strings.TrimSpace(string(raw))
	if c.Decode != nil {
		code, msg = c.Decode(resp.StatusCode, raw)
	}
	if len(msg) > 300 {
		msg = msg[:300] + "…"
	}
	e := &providers.CallError{Op: op, Status: resp.StatusCode, Code: code, Message: msg}
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		e.Auth = true
	case resp.StatusCode == http.StatusTooManyRequests:
		e.Temporary, e.Throttled = true, true
	case resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout:
		e.Temporary = true
	}
	return e
}

// NetworkError wraps a transport failure (DNS, connection refused, timeout)
// as a temporary outage.
func NetworkError(op string, err error) error {
	msg := err.Error()
	var ue *url.Error
	if errors.As(err, &ue) {
		msg = ue.Err.Error()
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		msg = "timed out"
	}
	return &providers.CallError{Op: op, Message: msg, Temporary: true, Cause: err}
}

// Path joins path segments with escaping.
func Path(format string, args ...string) string {
	esc := make([]any, len(args))
	for i, a := range args {
		esc[i] = url.PathEscape(a)
	}
	return fmt.Sprintf(format, esc...)
}
