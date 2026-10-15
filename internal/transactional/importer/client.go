package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/netguard"
)

// Account is the account or server a key reads from.
type Account struct {
	// Name is how the provider names it ("Acme Cloud production").
	Name string `json:"name"`
	// Kind says what Name is: server, account, workspace or domains.
	Kind string `json:"kind"`
}

// Ref is a template in the provider's list, before it is read.
type Ref struct {
	Ref      string `json:"ref"`
	Alias    string `json:"alias"`
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
	// Suggested templates are picked by default (Notifuse's transactional
	// ones; every template elsewhere).
	Suggested bool `json:"suggested"`
}

// TemplateSource reads a provider's templates. Nothing is ever written to
// the provider.
type TemplateSource interface {
	Connect(ctx context.Context) (Account, error)
	List(ctx context.Context) ([]Ref, error)
	Fetch(ctx context.Context, ref string) (Source, error)
	Dialect() Dialect
}

// SourceError is a provider failure explained for people.
type SourceError struct {
	Status  int
	Code    string
	Message string
}

func (e *SourceError) Error() string { return e.Message }

// Error codes.
const (
	CodeAuthFailed  = "source_auth_failed"
	CodeRateLimited = "source_rate_limited"
	CodeUnavailable = "source_unavailable"
	CodeBadResponse = "source_bad_response"
	CodeNotFound    = "source_not_found"
)

// client is the JSON client the template sources share: retries on 429 and
// 5xx (honouring Retry-After, at most a minute), redirects never followed.
type client struct {
	http *http.Client
	name string
	base string
	auth func(*http.Request)
	// authFailed explains a 401 or 403 in the provider's terms.
	authFailed func(status int, body []byte) string
	retries    int
	sleep      func(context.Context, time.Duration) error
}

func newClient(name, base string, hc *http.Client, auth func(*http.Request), authFailed func(int, []byte) string) *client {
	if hc == nil {
		hc = (netguard.Policy{}).Client(30 * time.Second)
	}
	return &client{http: hc, name: name, base: strings.TrimRight(base, "/"), auth: auth, authFailed: authFailed, retries: 3, sleep: sleepCtx}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// get reads path (relative to the base) into out.
func (c *client) get(ctx context.Context, path string, out any) error {
	var last error
	for attempt := 0; attempt <= c.retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "ReputeCast template importer")
		c.auth(req)
		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			last = &SourceError{Code: CodeUnavailable, Message: fmt.Sprintf("Couldn't reach %s: %s.", c.name, netguard.Describe(ctx, err))}
			if netguard.IsBlocked(err) {
				return last
			}
			if err := c.backoff(ctx, attempt, 0); err != nil {
				return err
			}
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return &SourceError{Status: resp.StatusCode, Code: CodeAuthFailed, Message: c.authFailed(resp.StatusCode, body)}
		case resp.StatusCode == http.StatusNotFound:
			return &SourceError{Status: 404, Code: CodeNotFound, Message: fmt.Sprintf("%s answered 404: %s.", c.name, describe(body, "not found"))}
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			wait := time.Duration(0)
			if resp.StatusCode == http.StatusTooManyRequests {
				wait = retryAfter(resp.Header.Get("Retry-After"))
				last = &SourceError{Status: 429, Code: CodeRateLimited, Message: fmt.Sprintf("%s is rate-limiting requests (429). Try again in a minute.", c.name)}
			} else {
				last = &SourceError{Status: resp.StatusCode, Code: CodeUnavailable, Message: fmt.Sprintf("%s answered %s. Try again in a few minutes.", c.name, resp.Status)}
			}
			if err := c.backoff(ctx, attempt, wait); err != nil {
				return err
			}
			continue
		case resp.StatusCode >= 400:
			return &SourceError{Status: resp.StatusCode, Code: CodeBadResponse, Message: fmt.Sprintf("%s refused the request (%d): %s.", c.name, resp.StatusCode, describe(body, resp.Status))}
		case resp.StatusCode >= 300:
			return &SourceError{Status: resp.StatusCode, Code: CodeBadResponse, Message: fmt.Sprintf("%s redirects to another address (%s). Enter the address it answers at.", c.name, resp.Status)}
		}
		if err := json.Unmarshal(body, out); err != nil {
			return &SourceError{Code: CodeBadResponse, Message: fmt.Sprintf("%s sent an answer ReputeCast can't read: %v.", c.name, err)}
		}
		return nil
	}
	return last
}

func (c *client) backoff(ctx context.Context, attempt int, wait time.Duration) error {
	if attempt >= c.retries {
		return nil
	}
	if wait <= 0 {
		wait = time.Duration(1<<attempt) * time.Second
	}
	return c.sleep(ctx, min(wait, time.Minute))
}

func retryAfter(h string) time.Duration {
	if n, err := strconv.Atoi(strings.TrimSpace(h)); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		return time.Until(t)
	}
	return 2 * time.Second
}

// describe pulls a human message out of an error body.
func describe(body []byte, fallback string) string {
	var m map[string]any
	if json.Unmarshal(body, &m) == nil {
		for _, k := range []string{"Message", "message", "error", "detail", "title"} {
			if s, ok := m[k].(string); ok && s != "" {
				return strings.TrimSuffix(s, ".")
			}
		}
		if errs, ok := m["errors"].([]any); ok && len(errs) > 0 {
			if e, ok := errs[0].(map[string]any); ok {
				if s, ok := e["message"].(string); ok && s != "" {
					return strings.TrimSuffix(s, ".")
				}
			}
		}
	}
	return fallback
}
