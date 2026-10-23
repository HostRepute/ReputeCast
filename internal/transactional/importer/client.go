package importer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
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

// SourceError is a provider failure explained for people: in English in
// Message, Text its message for other languages (errorText).
type SourceError struct {
	Status  int
	Code    string
	Message string
	Text    i18n.Text
}

// sourceErr is a SourceError worded by a message.
func sourceErr(status int, code, key string, args i18n.Args) *SourceError {
	t := say(key, args)
	return &SourceError{Status: status, Code: code, Message: t.In(i18n.Default), Text: t}
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
	authFailed func(status int, body []byte) i18n.Text
	retries    int
	sleep      func(context.Context, time.Duration) error
}

func newClient(name, base string, hc *http.Client, auth func(*http.Request), authFailed func(int, []byte) i18n.Text) *client {
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
			last = sourceErr(0, CodeUnavailable, "source.unreachable", i18n.Args{"name": c.name, "reason": netguard.Describe(ctx, err)})
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
			t := c.authFailed(resp.StatusCode, body)
			return &SourceError{Status: resp.StatusCode, Code: CodeAuthFailed, Message: t.In(i18n.Default), Text: t}
		case resp.StatusCode == http.StatusNotFound:
			return sourceErr(404, CodeNotFound, "source.notFound", i18n.Args{"name": c.name, "detail": describe(body, "not found")})
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			wait := time.Duration(0)
			if resp.StatusCode == http.StatusTooManyRequests {
				wait = retryAfter(resp.Header.Get("Retry-After"))
				last = sourceErr(429, CodeRateLimited, "source.rateLimited", i18n.Args{"name": c.name})
			} else {
				last = sourceErr(resp.StatusCode, CodeUnavailable, "source.unavailable", i18n.Args{"name": c.name, "status": resp.Status})
			}
			if err := c.backoff(ctx, attempt, wait); err != nil {
				return err
			}
			continue
		case resp.StatusCode >= 400:
			return sourceErr(resp.StatusCode, CodeBadResponse, "source.refused", i18n.Args{"name": c.name, "code": strconv.Itoa(resp.StatusCode), "detail": describe(body, resp.Status)})
		case resp.StatusCode >= 300:
			return sourceErr(resp.StatusCode, CodeBadResponse, "source.redirect", i18n.Args{"name": c.name, "status": resp.Status})
		}
		if err := json.Unmarshal(body, out); err != nil {
			return sourceErr(0, CodeBadResponse, "source.unreadable", i18n.Args{"name": c.name, "detail": err.Error()})
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
