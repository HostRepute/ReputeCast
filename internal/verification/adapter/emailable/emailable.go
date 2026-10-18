// Package emailable checks addresses with Emailable's v1 API
// (https://emailable.com/docs/api): GET /v1/account and GET /v1/verify with
// the key as api_key. Emailable answers 402 without credits and 249 when a
// check takes longer than asked ("try again"), which counts as no answer.
package emailable

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

const base = "https://api.emailable.com"

// statusTryAgain is Emailable's "still checking, try again" answer.
const statusTryAgain = 249

func init() {
	adapter.Register(adapter.Descriptor{
		Key: "emailable", Name: "Emailable", Initials: "E",
		Capabilities: adapter.Capabilities{BulkLists: true, RealTime: true, CatchAll: true, CreditBalance: true},
		Website:      "https://app.emailable.com",
		KeyHelp:      "https://app.emailable.com/api",
		MaxBatch:     25,
		PerSecond:    10,
		Mappings: []adapter.Mapping{
			{Codes: []string{"deliverable / accepted_email"}, Result: adapter.Valid},
			{Codes: []string{"undeliverable / rejected_email", "invalid_email", "invalid_domain", "invalid_smtp"}, Result: adapter.Invalid, Reason: adapter.ReasonInvalidAddress},
			{Codes: []string{"risky · accept_all"}, Result: adapter.Risky, Reason: adapter.ReasonCatchAll},
			{Codes: []string{"role"}, Result: adapter.Risky, Reason: adapter.ReasonRole},
			{Codes: []string{"disposable"}, Result: adapter.Risky, Reason: adapter.ReasonDisposable},
			{Codes: []string{"risky / low_quality", "low_deliverability"}, Result: adapter.Risky, Reason: "low_quality"},
			{Codes: []string{"unknown / no_connect", "timeout", "unavailable_smtp"}, Result: adapter.Unknown},
		},
		New: New,
	})
}

type client struct {
	key  string
	base string
	hc   *http.Client
}

// New opens Emailable with an API key.
func New(cfg adapter.Config) (adapter.Adapter, error) {
	b := cfg.BaseURL
	if b == "" {
		b = base
	}
	return &client{key: cfg.APIKey, base: strings.TrimRight(b, "/"), hc: cfg.Client()}, nil
}

func (c *client) Balance(ctx context.Context) (int64, error) {
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v1/account", Query: url.Values{"api_key": {c.key}}}, nil)
	if err != nil {
		return 0, err
	}
	var out struct {
		Credits int64 `json:"available_credits"`
	}
	if err := adapter.Decode(resp, &out); err != nil {
		return 0, err
	}
	return out.Credits, nil
}

type answer struct {
	Email      string `json:"email"`
	State      string `json:"state"`
	Reason     string `json:"reason"`
	Role       bool   `json:"role"`
	Disposable bool   `json:"disposable"`
	AcceptAll  bool   `json:"accept_all"`
}

func (c *client) Verify(ctx context.Context, email string, timeout time.Duration) (adapter.Answer, error) {
	q := url.Values{"email": {email}, "api_key": {c.key}}
	if timeout > 0 {
		q.Set("timeout", strconv.Itoa(max(int(timeout/time.Second), 2)))
	}
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v1/verify", Query: q}, nil)
	if err != nil {
		return adapter.Answer{}, err
	}
	if resp.Status == statusTryAgain {
		return adapter.Answer{}, &adapter.CallError{Kind: adapter.ErrUnavailable, Status: resp.Status, Message: "Emailable is still checking this address"}
	}
	var a answer
	if err := adapter.Decode(resp, &a); err != nil {
		return adapter.Answer{}, err
	}
	return Map(email, a.State, a.Reason, a.Role, a.Disposable, a.AcceptAll), nil
}

func (c *client) VerifyBatch(ctx context.Context, emails []string) ([]adapter.Answer, error) {
	return adapter.OneByOne(ctx, emails, 5, func(ctx context.Context, e string) (adapter.Answer, error) {
		return c.Verify(ctx, e, 10*time.Second)
	})
}

// Map turns Emailable's state, reason and flags into an answer.
func Map(email, state, reason string, role, disposable, acceptAll bool) adapter.Answer {
	state, reason = strings.ToLower(state), strings.ToLower(reason)
	code := state
	if reason != "" {
		code = state + " / " + reason
	}
	a := adapter.Answer{Email: email, Code: code}
	switch {
	case state == "undeliverable":
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonInvalidAddress
	case state == "unknown":
		a.Result = adapter.Unknown
	case disposable:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonDisposable, code+" · disposable"
	case acceptAll:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonCatchAll, code+" · accept_all"
	case role:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonRole, code+" · role"
	case state == "deliverable":
		a.Result = adapter.Valid
	case state == "risky":
		a.Result, a.Reason = adapter.Risky, "low_quality"
		if reason == "low_deliverability" {
			a.Reason = "low_deliverability"
		}
	default:
		a.Result = adapter.Unknown
	}
	return a
}
