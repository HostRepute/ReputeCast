// Package kickbox checks addresses with Kickbox's v2 API
// (https://docs.kickbox.com): GET /v2/verify and GET /v2/balance with the
// key as the apikey parameter. Every verify answer also carries the balance
// in X-Kickbox-Balance. Kickbox has no synchronous batch call, so bulk jobs
// check addresses one by one.
package kickbox

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

const base = "https://api.kickbox.com"

func init() {
	adapter.Register(adapter.Descriptor{
		Key: "kickbox", Name: "Kickbox", Initials: "K",
		Capabilities: adapter.Capabilities{BulkLists: true, RealTime: true, CatchAll: true, CreditBalance: true},
		Website:      "https://app.kickbox.com",
		KeyHelp:      "https://app.kickbox.com/settings/keys",
		MaxBatch:     25,
		PerSecond:    10,
		Mappings: []adapter.Mapping{
			{Codes: []string{"deliverable / accepted_email"}, Result: adapter.Valid},
			{Codes: []string{"undeliverable / rejected_email", "invalid_email", "invalid_domain", "invalid_smtp"}, Result: adapter.Invalid, Reason: adapter.ReasonInvalidAddress},
			{Codes: []string{"risky / accept_all"}, Result: adapter.Risky, Reason: adapter.ReasonCatchAll},
			{Codes: []string{"role"}, Result: adapter.Risky, Reason: adapter.ReasonRole},
			{Codes: []string{"disposable"}, Result: adapter.Risky, Reason: adapter.ReasonDisposable},
			{Codes: []string{"risky / low_quality", "low_deliverability"}, Result: adapter.Risky, Reason: "low_quality"},
			{Codes: []string{"unknown / no_connect", "timeout", "unavailable_smtp", "unexpected_error"}, Result: adapter.Unknown},
		},
		New: New,
	})
}

type client struct {
	key  string
	base string
	hc   *http.Client
}

// New opens Kickbox with an API key.
func New(cfg adapter.Config) (adapter.Adapter, error) {
	b := cfg.BaseURL
	if b == "" {
		b = base
	}
	return &client{key: cfg.APIKey, base: strings.TrimRight(b, "/"), hc: cfg.Client()}, nil
}

// noCredits: Kickbox answers 403 both for a bad key and for an empty
// balance; the message says which.
func noCredits(status int, body []byte) bool {
	m := strings.ToLower(string(body))
	return (status == http.StatusForbidden || status == http.StatusPaymentRequired) && (strings.Contains(m, "balance") || strings.Contains(m, "credit"))
}

func (c *client) Balance(ctx context.Context) (int64, error) {
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v2/balance", Query: url.Values{"apikey": {c.key}}}, noCredits)
	if err != nil {
		return 0, err
	}
	var out struct {
		Balance int64  `json:"balance"`
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := adapter.Decode(resp, &out); err != nil {
		return 0, err
	}
	if !out.Success && out.Message != "" {
		return 0, &adapter.CallError{Kind: adapter.ErrAuth, Status: resp.Status, Message: out.Message}
	}
	return out.Balance, nil
}

type answer struct {
	Result     string `json:"result"`
	Reason     string `json:"reason"`
	Role       bool   `json:"role"`
	Disposable bool   `json:"disposable"`
	AcceptAll  bool   `json:"accept_all"`
	Email      string `json:"email"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

func (c *client) Verify(ctx context.Context, email string, timeout time.Duration) (adapter.Answer, error) {
	q := url.Values{"email": {email}, "apikey": {c.key}}
	if timeout > 0 {
		q.Set("timeout", strconv.FormatInt(timeout.Milliseconds(), 10))
	}
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v2/verify", Query: q}, noCredits)
	if err != nil {
		return adapter.Answer{}, err
	}
	var a answer
	if err := adapter.Decode(resp, &a); err != nil {
		return adapter.Answer{}, err
	}
	if !a.Success && a.Message != "" {
		if noCredits(http.StatusForbidden, []byte(a.Message)) {
			return adapter.Answer{}, &adapter.CallError{Kind: adapter.ErrNoCredits, Status: resp.Status, Message: a.Message}
		}
		return adapter.Answer{}, &adapter.CallError{Kind: adapter.ErrAuth, Status: resp.Status, Message: a.Message}
	}
	return Map(email, a.Result, a.Reason, a.Role, a.Disposable, a.AcceptAll), nil
}

func (c *client) VerifyBatch(ctx context.Context, emails []string) ([]adapter.Answer, error) {
	return adapter.OneByOne(ctx, emails, 5, func(ctx context.Context, e string) (adapter.Answer, error) {
		return c.Verify(ctx, e, 6*time.Second)
	})
}

// Map turns Kickbox's result, reason and flags into an answer.
func Map(email, result, reason string, role, disposable, acceptAll bool) adapter.Answer {
	result, reason = strings.ToLower(result), strings.ToLower(reason)
	code := result
	if reason != "" {
		code = result + " / " + reason
	}
	a := adapter.Answer{Email: email, Code: code}
	switch {
	case result == "undeliverable":
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonInvalidAddress
	case result == "unknown":
		a.Result = adapter.Unknown
	case disposable:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonDisposable, code+" · disposable"
	case acceptAll:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonCatchAll, code+" · accept_all"
	case role:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonRole, code+" · role"
	case result == "deliverable":
		a.Result = adapter.Valid
	case result == "risky":
		a.Result, a.Reason = adapter.Risky, "low_quality"
		if reason == "low_deliverability" {
			a.Reason = "low_deliverability"
		}
	default:
		a.Result = adapter.Unknown
	}
	return a
}
