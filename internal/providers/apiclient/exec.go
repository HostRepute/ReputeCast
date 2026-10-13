package apiclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Request is a call whose body the caller encodes itself (form or multipart
// data, JSON that must be signed) or whose response headers it needs
// (SendGrid's X-Message-Id, Azure's Operation-Location). Do covers plain
// JSON calls.
type Request struct {
	// Op names the call for errors and the event log ("POST /v3/mail/send").
	Op     string
	Method string
	// Path is appended to the client's base URL. An absolute URL (the
	// pagination links some APIs return) is used as it is, but only on the
	// base URL's scheme and host, so credentials never go anywhere else.
	Path        string
	ContentType string
	Body        []byte
	// Header is added to the client's headers for this call only.
	Header http.Header
	// Accept, when set, returns a non-2xx response it accepts as a Response
	// instead of an error: Mailjet answers a batch where some messages
	// failed with 400 and the result of every message.
	Accept func(status int, body []byte) bool
}

// Response is a successful (2xx) response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Exec sends r and returns the 2xx response. Other statuses and transport
// failures become providers.CallError exactly as in Do: 401 and 403 are
// rejected credentials, 429 is throttling, 408 and 5xx are outages.
func (c *Client) Exec(ctx context.Context, r Request) (*Response, error) {
	target := c.BaseURL + r.Path
	if isAbsolute(r.Path) {
		if !sameOrigin(r.Path, c.BaseURL) {
			return nil, &providers.CallError{Op: r.Op, Message: "refused to follow a link to another host: " + r.Path}
		}
		target = r.Path
	}
	var body io.Reader
	if r.Body != nil {
		body = bytes.NewReader(r.Body)
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, target, body)
	if err != nil {
		return nil, &providers.CallError{Op: r.Op, Message: err.Error(), Cause: err}
	}
	for k, vs := range c.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	for k, vs := range r.Header {
		req.Header.Del(k)
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if r.ContentType != "" {
		req.Header.Set("Content-Type", r.ContentType)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "ReputeCast")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, NetworkError(r.Op, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponse))
	if err != nil {
		return nil, NetworkError(r.Op, err)
	}
	if (resp.StatusCode < 200 || resp.StatusCode > 299) && (r.Accept == nil || !r.Accept(resp.StatusCode, raw)) {
		return nil, c.statusError(r.Op, resp, raw)
	}
	return &Response{Status: resp.StatusCode, Header: resp.Header, Body: raw}, nil
}

func isAbsolute(s string) bool {
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}

// sameOrigin reports whether link has the scheme and host of base.
func sameOrigin(link, base string) bool {
	l, err := url.Parse(link)
	if err != nil {
		return false
	}
	b, err := url.Parse(base)
	if err != nil {
		return false
	}
	return strings.EqualFold(l.Scheme, b.Scheme) && strings.EqualFold(l.Host, b.Host)
}
