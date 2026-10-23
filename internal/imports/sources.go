package imports

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
)

// Source is where contacts are imported from: an email tool's API or a file.
type Source interface {
	// Connect checks access and returns the account and its lists with
	// counts per consent status.
	Connect(ctx context.Context) (ImportConnection, error)
	// Fields returns the source's fields for the selected lists.
	Fields(ctx context.Context, lists []string) ([]SourceField, error)
	// Fetch streams every contact of the selected lists, with its status.
	Fetch(ctx context.Context, lists []string, emit func(Row) error) error
}

// ImportConnection describes the source account.
type ImportConnection struct {
	Account    string             `json:"account"`
	DataCenter string             `json:"dataCenter,omitempty"`
	BaseURL    string             `json:"baseUrl,omitempty"`
	Lists      []ImportSourceList `json:"lists"`
	CheckedAt  time.Time          `json:"checkedAt"`
	// Templates are the email templates the source can bring along
	// (Notifuse).
	Templates *ImportTemplateCounts `json:"templates,omitempty"`
}

// ImportSourceList is an audience, list or group in the source.
type ImportSourceList struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Subscribed    int64    `json:"subscribed"`
	Unsubscribed  int64    `json:"unsubscribed"`
	Cleaned       int64    `json:"cleaned" doc:"Hard bounced in the source"`
	Pending       int64    `json:"pending"`
	Transactional int64    `json:"transactional"`
	Complained    int64    `json:"complained"`
	Fields        int      `json:"fields"`
	Tags          int      `json:"tags"`
	TagExamples   []string `json:"tagExamples"`
	DoubleOptIn   bool     `json:"doubleOptIn"`
	// Since is the year the list was created ("double opt-in was on in this audience since 2024").
	Since string `json:"since,omitempty"`
}

// SourceField is a field (merge field, attribute, column) in the source.
type SourceField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Type as the source reports it: text, number, date, email, phone,
	// address, birthday, url, boolean.
	Type string `json:"type"`
}

// Source statuses, normalized.
const (
	StatusSubscribed    = "subscribed"
	StatusUnsubscribed  = "unsubscribed"
	StatusCleaned       = "cleaned"
	StatusPending       = "pending"
	StatusTransactional = "transactional"
	StatusComplained    = "complained"
)

// Row is one contact read from the source.
type Row struct {
	Email      string
	Status     string
	List       string
	Data       map[string]string
	Tags       []string
	Engagement Engagement
}

// Engagement is what the source knows about a contact's activity and consent.
type Engagement struct {
	LastOpen  *time.Time `json:"last_open,omitempty"`
	LastClick *time.Time `json:"last_click,omitempty"`
	Changed   *time.Time `json:"changed,omitempty"`
	SignupAt  *time.Time `json:"signup_at,omitempty"`
	OptinAt   *time.Time `json:"optin_at,omitempty"`
	OptinIP   string     `json:"optin_ip,omitempty"`
	// Rating is Mailchimp's member rating (1–5).
	Rating int `json:"rating,omitempty"`
}

// SourceError is a failure talking to the source, explained for people:
// Message in English, Text the message it came from for readers of other
// languages.
type SourceError struct {
	Status  int
	Code    string // auth_failed, rate_limited, unavailable, bad_response
	Message string
	Text    i18n.Text
	// Reason is the source's own short words ("API Key Invalid").
	Reason string
	// RetryAfter is set for rate limits.
	RetryAfter time.Duration
}

func (e *SourceError) Error() string   { return e.Message }
func (e *SourceError) kept() i18n.Text { return e.Text }

// newSourceError explains a source failure with a message of the catalog.
func newSourceError(code string, status int, m i18n.Msg, args i18n.Args) *SourceError {
	t := m.Keep(args)
	return &SourceError{Status: status, Code: code, Message: english(t), Text: t}
}

// Error codes of SourceError.
const (
	CodeAuthFailed  = "source_auth_failed"
	CodeRateLimited = "source_rate_limited"
	CodeUnavailable = "source_unavailable"
	CodeBadResponse = "source_bad_response"
)

// apiClient is the HTTP client shared by the API importers: JSON requests
// with retries on 429 and 5xx, honouring Retry-After.
type apiClient struct {
	http    *http.Client
	name    string // "Mailchimp"
	base    string
	auth    func(*http.Request)
	retries int
	// OnRetry is told about each back-off (the live log shows them).
	OnRetry func(attempt int, wait time.Duration, status int)
	sleep   func(context.Context, time.Duration) error
	// gap paces requests ("Slow down to 3 req/s"); zero sends the next
	// request as soon as the previous one answered.
	gap  time.Duration
	last time.Time
}

// setPace limits the client to perSecond requests a second (0: no limit).
func (c *apiClient) setPace(perSecond int) {
	c.gap = 0
	if perSecond > 0 {
		c.gap = time.Second / time.Duration(perSecond)
	}
}

// pace waits until the next request is allowed.
func (c *apiClient) pace(ctx context.Context) error {
	if c.gap <= 0 {
		return nil
	}
	if wait := time.Until(c.last.Add(c.gap)); wait > 0 {
		if err := c.sleep(ctx, wait); err != nil {
			return err
		}
	}
	c.last = time.Now()
	return nil
}

func newAPIClient(name, base string, hc *http.Client, auth func(*http.Request)) *apiClient {
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &apiClient{http: hc, name: name, base: strings.TrimRight(base, "/"), auth: auth, retries: 5, sleep: sleepCtx}
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

// get fetches path (relative to the base) into out.
func (c *apiClient) get(ctx context.Context, path string, out any) error {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if err := c.pace(ctx); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "ReputeCast importer")
		c.auth(req)
		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = newSourceError(CodeUnavailable, 0, mUnreachable, i18n.Args{"source": c.name, "reason": netguard.Describe(ctx, err)})
			if netguard.IsBlocked(err) {
				return lastErr
			}
			if err := c.backoff(ctx, attempt, 0, 0); err != nil {
				return err
			}
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			se := newSourceError(CodeAuthFailed, resp.StatusCode, mRejectedKey, i18n.Args{"source": c.name, "reason": describe(body, resp.Status)})
			se.Reason = describeFirst(body, resp.Status, "title", "detail", "message", "error")
			return se
		case resp.StatusCode == http.StatusTooManyRequests:
			wait := retryAfter(resp.Header.Get("Retry-After"))
			se := newSourceError(CodeRateLimited, 429, mRateLimited, i18n.Args{"source": c.name})
			se.RetryAfter = wait
			lastErr = se
			if c.OnRetry != nil {
				c.OnRetry(attempt+1, wait, 429)
			}
			if err := c.backoff(ctx, attempt, wait, 429); err != nil {
				return err
			}
			continue
		case resp.StatusCode >= 500:
			lastErr = newSourceError(CodeUnavailable, resp.StatusCode, mAnswered, i18n.Args{"source": c.name, "status": resp.Status})
			if c.OnRetry != nil {
				c.OnRetry(attempt+1, 0, resp.StatusCode)
			}
			if err := c.backoff(ctx, attempt, 0, resp.StatusCode); err != nil {
				return err
			}
			continue
		case resp.StatusCode >= 400:
			return newSourceError(CodeBadResponse, resp.StatusCode, mRefused, i18n.Args{"source": c.name, "reason": describe(body, resp.Status)})
		case resp.StatusCode >= 300:
			// Redirects are not followed (the guard checks each address): the
			// person enters the address the server answers at.
			return newSourceError(CodeBadResponse, resp.StatusCode, mRedirects, i18n.Args{"source": c.name, "status": resp.Status})
		}
		if err := json.Unmarshal(body, out); err != nil {
			return newSourceError(CodeBadResponse, 0, mUnreadable, i18n.Args{"source": c.name, "reason": err.Error()})
		}
		return nil
	}
	return lastErr
}

func (c *apiClient) backoff(ctx context.Context, attempt int, wait time.Duration, _ int) error {
	if attempt >= c.retries {
		return nil
	}
	if wait <= 0 {
		wait = time.Duration(1<<attempt) * time.Second
	}
	if wait > time.Minute {
		wait = time.Minute
	}
	return c.sleep(ctx, wait)
}

func retryAfter(h string) time.Duration {
	if h == "" {
		return 2 * time.Second
	}
	if n, err := strconv.Atoi(strings.TrimSpace(h)); err == nil {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		return time.Until(t)
	}
	return 2 * time.Second
}

// describe pulls a human message out of an error body.
func describe(body []byte, fallback string) string {
	return describeFirst(body, fallback, "detail", "message", "title", "error")
}

// describeFirst returns the first of keys the JSON body has as text.
func describeFirst(body []byte, fallback string, keys ...string) string {
	var m map[string]any
	if json.Unmarshal(body, &m) == nil {
		for _, k := range keys {
			if s, ok := m[k].(string); ok && s != "" {
				return strings.TrimSuffix(s, ".")
			}
		}
	}
	return fallback
}

// ErrNoKey is returned when an API importer has no credentials.
var ErrNoKey = errors.New("paste an API key first")

func parseTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
