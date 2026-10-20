package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Client returns the configured HTTP client, or one with a 20-second
// timeout. Production wiring passes the SSRF-guarded client.
func (c Config) Client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Call is one API request.
type Call struct {
	Op     string
	Method string
	URL    string
	// Form is sent as application/x-www-form-urlencoded; JSON as JSON.
	Form url.Values
	JSON any
	// Header is added to the request (auth).
	Header http.Header
	// BasicUser and BasicPass set HTTP basic auth.
	BasicUser, BasicPass string
}

// Do sends a call and decodes a 2xx JSON answer into out (nil to skip). Any
// other answer is a *CallError carrying the body, which adapters read for
// the provider's own error code. A transport failure is ErrUnavailable.
func Do(ctx context.Context, client *http.Client, c Call, out any) ([]byte, error) {
	var body io.Reader
	ctype := ""
	switch {
	case c.Form != nil:
		body = strings.NewReader(c.Form.Encode())
		ctype = "application/x-www-form-urlencoded"
	case c.JSON != nil:
		b, err := json.Marshal(c.JSON)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
		ctype = "application/json"
	}
	req, err := http.NewRequestWithContext(ctx, c.Method, c.URL, body)
	if err != nil {
		return nil, err
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ReputeCast")
	for k, vs := range c.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if c.BasicUser != "" {
		req.SetBasicAuth(c.BasicUser, c.BasicPass)
	}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, errors.Join(ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, errors.Join(ErrUnavailable, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return raw, &CallError{Op: c.Op, Status: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, err
		}
	}
	return raw, nil
}

// SortedParams writes form parameters sorted by key, as key=value pairs
// joined by " · " for the Events tab: MessageStatus=delivered · ErrorCode=30007.
func SortedParams(v url.Values, keys ...string) string {
	parts := []string{}
	for _, k := range keys {
		if s := v.Get(k); s != "" {
			parts = append(parts, k+"="+s)
		}
	}
	return strings.Join(parts, " · ")
}

// Concat writes form parameters the way several providers sign them: keys
// in byte order, each key followed by its values.
func Concat(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		vals := append([]string(nil), v[k]...)
		sort.Strings(vals)
		for _, x := range vals {
			b.WriteString(k)
			b.WriteString(x)
		}
	}
	return b.String()
}

// Quote writes a reply's body the way the Events tab shows it: Body="STOP".
func Quote(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) > 60 {
		s = string([]rune(s)[:57]) + "…"
	}
	return "“" + s + "”"
}
