// Package bouncer checks addresses with Bouncer's v1.1 API
// (https://docs.usebouncer.com): GET /v1.1/credits and
// GET /v1.1/email/verify, authenticated with the x-api-key header. Bouncer
// answers 402 when the account has no credits left.
package bouncer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

const base = "https://api.usebouncer.com"

func init() {
	adapter.Register(adapter.Descriptor{
		Key: "bouncer", Name: "Bouncer", Initials: "B",
		Capabilities: adapter.Capabilities{BulkLists: true, RealTime: true, CatchAll: true, CreditBalance: true},
		Website:      "https://app.usebouncer.com",
		KeyHelp:      "https://app.usebouncer.com/api",
		MaxBatch:     25,
		PerSecond:    10,
		Mappings: []adapter.Mapping{
			{Codes: []string{"deliverable / accepted_email"}, Result: adapter.Valid},
			{Codes: []string{"undeliverable / rejected_email", "invalid_email", "invalid_domain"}, Result: adapter.Invalid, Reason: adapter.ReasonInvalidAddress},
			{Codes: []string{"risky · acceptAll"}, Result: adapter.Risky, Reason: adapter.ReasonCatchAll},
			{Codes: []string{"role"}, Result: adapter.Risky, Reason: adapter.ReasonRole},
			{Codes: []string{"disposable"}, Result: adapter.Risky, Reason: adapter.ReasonDisposable},
			{Codes: []string{"risky / low_quality", "low_deliverability"}, Result: adapter.Risky, Reason: "low_quality"},
			{Codes: []string{"unknown / unavailable_smtp", "timeout", "dns_error"}, Result: adapter.Unknown},
		},
		New: New,
	})
}

type client struct {
	key  string
	base string
	hc   *http.Client
}

// New opens Bouncer with an API key.
func New(cfg adapter.Config) (adapter.Adapter, error) {
	b := cfg.BaseURL
	if b == "" {
		b = base
	}
	return &client{key: cfg.APIKey, base: strings.TrimRight(b, "/"), hc: cfg.Client()}, nil
}

func (c *client) header() http.Header { return http.Header{"X-Api-Key": {c.key}} }

func (c *client) Balance(ctx context.Context) (int64, error) {
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v1.1/credits", Header: c.header()}, nil)
	if err != nil {
		return 0, err
	}
	var out struct {
		Credits int64 `json:"credits"`
	}
	if err := adapter.Decode(resp, &out); err != nil {
		return 0, err
	}
	return out.Credits, nil
}

type answer struct {
	Email  string `json:"email"`
	Status string `json:"status"`
	Reason string `json:"reason"`
	Domain struct {
		AcceptAll  string `json:"acceptAll"`
		Disposable string `json:"disposable"`
	} `json:"domain"`
	Account struct {
		Role string `json:"role"`
	} `json:"account"`
	Toxic string `json:"toxic"`
}

func (c *client) Verify(ctx context.Context, email string, timeout time.Duration) (adapter.Answer, error) {
	q := url.Values{"email": {email}}
	if timeout > 0 {
		q.Set("timeout", strconv.Itoa(max(int(timeout/time.Second), 1)))
	}
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v1.1/email/verify", Query: q, Header: c.header()}, nil)
	if err != nil {
		return adapter.Answer{}, err
	}
	var a answer
	if err := adapter.Decode(resp, &a); err != nil {
		return adapter.Answer{}, err
	}
	return Map(email, a.Status, a.Reason, a.Domain.AcceptAll == "yes", a.Domain.Disposable == "yes", a.Account.Role == "yes", a.Toxic == "yes"), nil
}

func (c *client) VerifyBatch(ctx context.Context, emails []string) ([]adapter.Answer, error) {
	return adapter.OneByOne(ctx, emails, 5, func(ctx context.Context, e string) (adapter.Answer, error) {
		return c.Verify(ctx, e, 10*time.Second)
	})
}

// Map turns Bouncer's status, reason and flags into an answer.
func Map(email, status, reason string, acceptAll, disposable, role, toxic bool) adapter.Answer {
	status, reason = strings.ToLower(status), strings.ToLower(reason)
	code := status
	if reason != "" {
		code = status + " / " + reason
	}
	a := adapter.Answer{Email: email, Code: code}
	switch {
	case status == "undeliverable":
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonInvalidAddress
	case status == "unknown":
		a.Result = adapter.Unknown
	case disposable:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonDisposable, code+" · disposable"
	case acceptAll:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonCatchAll, code+" · acceptAll"
	case role:
		a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonRole, code+" · role"
	case toxic:
		a.Result, a.Reason, a.Code = adapter.Risky, "toxic", code+" · toxic"
	case status == "deliverable":
		a.Result = adapter.Valid
	case status == "risky":
		a.Result, a.Reason = adapter.Risky, "low_quality"
		if reason == "low_deliverability" {
			a.Reason = "low_deliverability"
		}
	default:
		a.Result = adapter.Unknown
	}
	return a
}
