// Package shopify is the Shopify store adapter: a custom app the merchant
// creates in their own Shopify account (Dev Dashboard), installed through
// Shopify's own approval page (OAuth authorization code grant), then
// Shopify's GraphQL Admin API over plain HTTPS with the app's access token
// (no SDK), and webhooks verified with the app's client secret (HMAC).
//
// The API version is pinned, and webhooks are registered with it, so the
// payloads keep the shape the parsers read. Addresses, phone numbers,
// payment details and order notes are never read: the queries don't ask
// for them and the webhook structs have no field for them.
package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
)

// Key is the adapter's key in the stores registry.
const Key = "shopify"

// APIVersion is the Admin API version ReputeCast speaks (brief §4).
const APIVersion = "2026-10"

// Scopes the app needs, and the optional one that passes unsubscribes back.
var (
	Scopes        = []string{"read_customers", "read_orders", "read_products"}
	OptionalScope = "write_customers"
)

// Topics are the 11 webhooks ReputeCast registers, in the order of the
// store page's table (brief §4).
var Topics = []string{
	"orders/create",
	"orders/paid",
	"orders/updated",
	"refunds/create",
	"customers/create",
	"customers/update",
	"checkouts/create",
	"checkouts/update",
	"products/create",
	"products/update",
	"products/delete",
}

// LifecycleTopic tells ReputeCast the app was uninstalled. It is
// registered with the others but isn't one of the store's data webhooks.
const LifecycleTopic = "app/uninstalled"

func init() {
	stores.Register(stores.Descriptor{
		Key:  Key,
		Name: "Shopify",
		Capabilities: []stores.Capability{stores.CapOrders, stores.CapProducts, stores.CapCustomers, stores.CapCheckouts,
			stores.CapWriteBack, stores.CapAttribution},
		Topics:   Topics,
		Open:     func(cfg stores.Config) (stores.Client, error) { return New(cfg) },
		Webhooks: Webhooks{},
	})
}

// Client is a store reached with the app's access token.
type Client struct {
	domain    string
	base      string
	token     string
	version   string
	hc        *http.Client
	now       func() time.Time
	retryWait time.Duration
}

// New opens a client on a store.
func New(cfg stores.Config) (*Client, error) {
	domain := stores.NormalizeHost(cfg.Domain)
	if domain == "" {
		return nil, fmt.Errorf("shopify: %w: no store domain", stores.ErrNotAStore)
	}
	c := &Client{domain: domain, base: "https://" + domain, token: strings.TrimSpace(cfg.Token), version: APIVersion,
		hc: cfg.HTTP, now: cfg.Now, retryWait: 500 * time.Millisecond}
	if cfg.BaseURL != "" {
		c.base = strings.TrimRight(cfg.BaseURL, "/")
		c.retryWait = 5 * time.Millisecond
	}
	if cfg.RetryWait > 0 {
		c.retryWait = cfg.RetryWait
	}
	if cfg.APIVersion != "" {
		c.version = cfg.APIVersion
	}
	if c.hc == nil {
		c.hc = &http.Client{Timeout: 30 * time.Second}
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c, nil
}

// gqlRequest is one GraphQL call.
type gqlRequest struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
}

type gqlError struct {
	Message    string `json:"message"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

type gqlResponse struct {
	Data       json.RawMessage `json:"data"`
	Errors     []gqlError      `json:"errors"`
	Extensions struct {
		Cost struct {
			RequestedQueryCost float64 `json:"requestedQueryCost"`
			ThrottleStatus     struct {
				MaximumAvailable   float64 `json:"maximumAvailable"`
				CurrentlyAvailable float64 `json:"currentlyAvailable"`
				RestoreRate        float64 `json:"restoreRate"`
			} `json:"throttleStatus"`
		} `json:"cost"`
	} `json:"extensions"`
}

// gql runs one operation. A 5xx answer or a network error is retried
// twice; a throttle is returned as *stores.RateLimited with the time
// Shopify's bucket needs to refill, for the caller to wait and say so.
func (c *Client) gql(ctx context.Context, op, query string, vars map[string]any, out any) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryWait * time.Duration(attempt)):
			}
		}
		err := c.once(ctx, op, query, vars, out)
		if err == nil {
			return nil
		}
		last = err
		if !errors.Is(err, stores.ErrUnavailable) {
			return err
		}
	}
	return last
}

func (c *Client) once(ctx context.Context, op, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(gqlRequest{Query: query, OperationName: op, Variables: vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/admin/api/"+c.version+"/graphql.json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Shopify-Access-Token", c.token)
	req.Header.Set("User-Agent", "ReputeCast")
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", stores.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("%w: %w", stores.ErrUnavailable, err)
	}
	if err := statusError(resp, raw); err != nil {
		return err
	}
	var r gqlResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("shopify: unreadable answer to %s: %w", op, err)
	}
	for _, e := range r.Errors {
		switch e.Extensions.Code {
		case "THROTTLED":
			ts := r.Extensions.Cost.ThrottleStatus
			wait := 2 * time.Second
			if ts.RestoreRate > 0 {
				need := r.Extensions.Cost.RequestedQueryCost - ts.CurrentlyAvailable
				if need > 0 {
					wait = time.Duration(need/ts.RestoreRate*float64(time.Second)) + 100*time.Millisecond
				}
			}
			return &stores.RateLimited{RetryAfter: wait}
		case "ACCESS_DENIED":
			return fmt.Errorf("%w: %s", stores.ErrPermission, e.Message)
		case "INTERNAL_SERVER_ERROR":
			return fmt.Errorf("%w: %s", stores.ErrUnavailable, e.Message)
		}
	}
	if len(r.Errors) > 0 && len(r.Data) == 0 || len(r.Errors) > 0 && string(r.Data) == "null" {
		return fmt.Errorf("shopify: %s: %s", op, r.Errors[0].Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(r.Data, out); err != nil {
		return fmt.Errorf("shopify: unreadable data of %s: %w", op, err)
	}
	return nil
}

// statusError maps an HTTP status to the contract's errors.
func statusError(resp *http.Response, raw []byte) error {
	switch {
	case resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", stores.ErrUnauthorized, snippet(raw))
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: %s", stores.ErrPermission, snippet(raw))
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s", stores.ErrNotFound, snippet(raw))
	case resp.StatusCode == http.StatusTooManyRequests:
		wait := 2 * time.Second
		if s := resp.Header.Get("Retry-After"); s != "" {
			if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 {
				wait = time.Duration(f * float64(time.Second))
			}
		}
		return &stores.RateLimited{RetryAfter: wait}
	case resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusLocked:
		// A frozen or locked shop answers again once its owner fixes it.
		return fmt.Errorf("%w: %d %s", stores.ErrUnavailable, resp.StatusCode, snippet(raw))
	case resp.StatusCode >= 500:
		return fmt.Errorf("%w: %d", stores.ErrUnavailable, resp.StatusCode)
	}
	return fmt.Errorf("shopify: %d %s", resp.StatusCode, snippet(raw))
}

func snippet(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

// gid builds a GraphQL id from a numeric one.
func gid(kind, id string) string {
	if strings.HasPrefix(id, "gid://") {
		return id
	}
	return "gid://shopify/" + kind + "/" + id
}

// numericID keeps a GraphQL id's number ("gid://shopify/Product/123" → "123").
func numericID(id string) string {
	if i := strings.LastIndexByte(id, '/'); i >= 0 && strings.HasPrefix(id, "gid://") {
		id = id[i+1:]
	}
	if i := strings.IndexByte(id, '?'); i >= 0 {
		id = id[:i]
	}
	return id
}
