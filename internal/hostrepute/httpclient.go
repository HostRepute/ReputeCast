package hostrepute

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
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/hostrepute/reputecast/internal/version"
)

// HTTPClient implements Client against HostRepute's REST API
// (docs/integrations/hostrepute.md). Requests carry the OAuth access token
// or the API key as a bearer token; network errors, 429 and 5xx answers
// are retried before the call fails with ErrUnavailable.
type HTTPClient struct {
	// BaseURL is the API root, e.g. https://api.hostrepute.com/v1.
	BaseURL string
	Tokens  oauth2.TokenSource
	HTTP    *http.Client
	// Attempts per call (default 3) and the wait before each retry.
	Attempts int
	Wait     func(attempt int) time.Duration
	// RevokeURL and ClientID are used by Revoke (RFC 7009).
	RevokeURL string
	ClientID  string
}

// DefaultTimeout is the per-request timeout of HostRepute calls.
const DefaultTimeout = 30 * time.Second

// NewHTTPClient returns a client with ReputeCast's defaults: 30 s per
// request, 3 attempts.
func NewHTTPClient(baseURL string, tokens oauth2.TokenSource, httpClient *http.Client) *HTTPClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	return &HTTPClient{BaseURL: strings.TrimRight(baseURL, "/"), Tokens: tokens, HTTP: httpClient, Attempts: 3}
}

// UnavailableError explains why HostRepute couldn't be reached. It wraps
// ErrUnavailable.
type UnavailableError struct {
	Host     string
	Attempts int
	Reason   string // "timeout after 30 s", "connection refused", "HTTP 503"
}

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("Couldn't reach %s (%s, %d attempts)", e.Host, e.Reason, e.Attempts)
}

// Unwrap makes errors.Is(err, ErrUnavailable) true.
func (e *UnavailableError) Unwrap() error { return ErrUnavailable }

func (c *HTTPClient) host() string {
	if u, err := url.Parse(c.BaseURL); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return "HostRepute"
}

func (c *HTTPClient) wait(attempt int) time.Duration {
	if c.Wait != nil {
		return c.Wait(attempt)
	}
	return time.Duration(attempt*attempt) * time.Second // 1 s, 4 s
}

func reasonFor(err error, timeout time.Duration) string {
	var ne net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		switch {
		case timeout >= time.Second:
			return fmt.Sprintf("timeout after %d s", int(timeout.Seconds()))
		case timeout > 0:
			return fmt.Sprintf("timeout after %d ms", timeout.Milliseconds())
		}
		return "timeout"
	case strings.Contains(err.Error(), "connection refused"):
		return "connection refused"
	case strings.Contains(err.Error(), "no such host"):
		return "DNS lookup failed"
	}
	return "network error"
}

// do sends one API call with retries and decodes a JSON answer into out.
func (c *HTTPClient) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	attempts := c.Attempts
	if attempts <= 0 {
		attempts = 3
	}
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	reason := ""
	for attempt := 1; ; attempt++ {
		tok, err := c.Tokens.Token()
		if err != nil {
			var re *oauth2.RetrieveError
			if errors.As(err, &re) && re.Response != nil && re.Response.StatusCode < 500 {
				return fmt.Errorf("%w: refreshing the token failed: %s", ErrUnauthorized, re.ErrorCode)
			}
			reason = "token refresh failed"
		} else {
			var rd io.Reader
			if payload != nil {
				rd = bytes.NewReader(payload)
			}
			req, err := http.NewRequestWithContext(ctx, method, u, rd)
			if err != nil {
				return err
			}
			tok.SetAuthHeader(req)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("User-Agent", "ReputeCast/"+version.Get().Version)
			if payload != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := c.HTTP.Do(req)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				reason = reasonFor(err, c.HTTP.Timeout)
			} else {
				retry, err := c.handle(resp, out)
				if !retry {
					return err
				}
				reason = fmt.Sprintf("HTTP %d", resp.StatusCode)
				if ra := resp.Header.Get("Retry-After"); ra != "" && attempt < attempts {
					if s, perr := strconv.Atoi(ra); perr == nil && s > 0 && s <= 10 {
						if !sleep(ctx, time.Duration(s)*time.Second) {
							return ctx.Err()
						}
						continue
					}
				}
			}
		}
		if attempt >= attempts {
			return &UnavailableError{Host: c.host(), Attempts: attempts, Reason: reason}
		}
		if !sleep(ctx, c.wait(attempt)) {
			return ctx.Err()
		}
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

type problem struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Limit  int    `json:"limit"`
	Used   int    `json:"used"`
	Team   string `json:"team"`
}

// handle maps an answer to a result; retry is true for 429 and 5xx.
func (c *HTTPClient) handle(resp *http.Response, out any) (retry bool, err error) {
	defer resp.Body.Close() //nolint:errcheck // a failed close of a read body changes nothing
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil || len(bytes.TrimSpace(body)) == 0 {
			return false, nil
		}
		if err := json.Unmarshal(body, out); err != nil {
			return false, fmt.Errorf("hostrepute: unexpected answer: %w", err)
		}
		return false, nil
	}
	var p problem
	_ = json.Unmarshal(body, &p)
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return true, nil
	case resp.StatusCode == http.StatusUnauthorized:
		return false, ErrUnauthorized
	case p.Code == "monitor_limit_reached":
		return false, &LimitError{Limit: p.Limit, Used: p.Used, Team: p.Team}
	case resp.StatusCode == http.StatusForbidden:
		return false, fmt.Errorf("%w: %s", ErrForbidden, p.Detail)
	case resp.StatusCode == http.StatusNotFound:
		return false, ErrNotFound
	}
	return false, &APIError{Status: resp.StatusCode, Code: p.Code, Detail: p.Detail}
}

func dayRange(from, to time.Time) url.Values {
	return url.Values{"from": {from.Format("2006-01-02")}, "to": {to.Format("2006-01-02")}}
}

// Account implements Client.
func (c *HTTPClient) Account(ctx context.Context) (Account, error) {
	var out Account
	err := c.do(ctx, http.MethodGet, "/me", nil, nil, &out)
	return out, err
}

// Assets implements Client.
func (c *HTTPClient) Assets(ctx context.Context) ([]Asset, error) {
	var out struct {
		Assets []Asset `json:"assets"`
	}
	err := c.do(ctx, http.MethodGet, "/assets", nil, nil, &out)
	return out.Assets, err
}

// AddAsset implements Client.
func (c *HTTPClient) AddAsset(ctx context.Context, asset, typ, label string) (Asset, error) {
	var out Asset
	err := c.do(ctx, http.MethodPost, "/assets", nil, map[string]string{"asset": asset, "type": typ, "label": label}, &out)
	return out, err
}

// RemoveAsset implements Client.
func (c *HTTPClient) RemoveAsset(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/assets/"+url.PathEscape(id), nil, nil, nil)
}

// Blocklists implements Client.
func (c *HTTPClient) Blocklists(ctx context.Context) (Blocklists, error) {
	var out Blocklists
	err := c.do(ctx, http.MethodGet, "/blocklists", nil, nil, &out)
	return out, err
}

// Postmaster implements Client.
func (c *HTTPClient) Postmaster(ctx context.Context, domain string, from, to time.Time) ([]PostmasterDay, error) {
	var out struct {
		Days []PostmasterDay `json:"days"`
	}
	err := c.do(ctx, http.MethodGet, "/postmaster/"+url.PathEscape(domain), dayRange(from, to), nil, &out)
	return out.Days, err
}

// SNDS implements Client.
func (c *HTTPClient) SNDS(ctx context.Context, ip string, from, to time.Time) ([]SNDSDay, error) {
	var out struct {
		Days []SNDSDay `json:"days"`
	}
	err := c.do(ctx, http.MethodGet, "/snds/"+url.PathEscape(ip), dayRange(from, to), nil, &out)
	return out.Days, err
}

// DMARCReports implements Client.
func (c *HTTPClient) DMARCReports(ctx context.Context, domain string, from, to time.Time) ([]DMARCReport, error) {
	var out struct {
		Reports []DMARCReport `json:"reports"`
	}
	err := c.do(ctx, http.MethodGet, "/dmarc/"+url.PathEscape(domain)+"/reports", dayRange(from, to), nil, &out)
	for i := range out.Reports {
		if out.Reports[i].Domain == "" {
			out.Reports[i].Domain = domain
		}
	}
	return out.Reports, err
}

// Alerts implements Client.
func (c *HTTPClient) Alerts(ctx context.Context, since time.Time) ([]Alert, error) {
	var out struct {
		Alerts []Alert `json:"alerts"`
	}
	err := c.do(ctx, http.MethodGet, "/alerts", url.Values{"since": {since.UTC().Format(time.RFC3339)}}, nil, &out)
	return out.Alerts, err
}

// StartCheck implements Client.
func (c *HTTPClient) StartCheck(ctx context.Context, assets []string) (Check, error) {
	var out Check
	err := c.do(ctx, http.MethodPost, "/checks", nil, map[string]any{"assets": assets}, &out)
	return out, err
}

// Check implements Client.
func (c *HTTPClient) Check(ctx context.Context, id string) (Check, error) {
	var out Check
	err := c.do(ctx, http.MethodGet, "/checks/"+url.PathEscape(id), nil, nil, &out)
	return out, err
}

// RequestDelisting implements Client.
func (c *HTTPClient) RequestDelisting(ctx context.Context, asset, source string) (Delisting, error) {
	var out Delisting
	err := c.do(ctx, http.MethodPost, "/delistings", nil, map[string]string{"asset": asset, "source": source}, &out)
	return out, err
}

// Revoke implements Client: RFC 7009 token revocation. API keys can't be
// revoked from ReputeCast; they are only forgotten.
func (c *HTTPClient) Revoke(ctx context.Context) error {
	if c.RevokeURL == "" {
		return nil
	}
	tok, err := c.Tokens.Token()
	if err != nil {
		return err
	}
	form := url.Values{"token": {tok.AccessToken}, "token_type_hint": {"access_token"}}
	if c.ClientID != "" {
		form.Set("client_id", c.ClientID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.RevokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return &UnavailableError{Host: c.host(), Attempts: 1, Reason: reasonFor(err, c.HTTP.Timeout)}
	}
	defer resp.Body.Close() //nolint:errcheck // a failed close of a read body changes nothing
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Code: "revoke_failed"}
	}
	return nil
}
