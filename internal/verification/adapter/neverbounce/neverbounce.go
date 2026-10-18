// Package neverbounce checks addresses with NeverBounce's v4.2 API
// (https://developers.neverbounce.com): GET /v4.2/account/info and
// GET /v4.2/single/check with the key as the key parameter. NeverBounce
// answers most errors with HTTP 200 and a status word (auth_failure,
// throttle_triggered, temp_unavail, general_failure).
package neverbounce

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

const base = "https://api.neverbounce.com"

func init() {
	adapter.Register(adapter.Descriptor{
		Key: "neverbounce", Name: "NeverBounce", Initials: "NB",
		Capabilities: adapter.Capabilities{BulkLists: true, RealTime: true, CatchAll: true, CreditBalance: true},
		Website:      "https://app.neverbounce.com",
		KeyHelp:      "https://app.neverbounce.com/apps",
		MaxBatch:     25,
		PerSecond:    10,
		Mappings: []adapter.Mapping{
			{Codes: []string{"valid"}, Result: adapter.Valid},
			{Codes: []string{"invalid"}, Result: adapter.Invalid, Reason: adapter.ReasonInvalidAddress},
			{Codes: []string{"invalid · spamtrap_network"}, Result: adapter.Invalid, Reason: adapter.ReasonSpamTrap},
			{Codes: []string{"catchall"}, Result: adapter.Risky, Reason: adapter.ReasonCatchAll},
			{Codes: []string{"valid · role_account"}, Result: adapter.Risky, Reason: adapter.ReasonRole},
			{Codes: []string{"disposable"}, Result: adapter.Risky, Reason: adapter.ReasonDisposable},
			{Codes: []string{"unknown"}, Result: adapter.Unknown},
		},
		New: New,
	})
}

type client struct {
	key  string
	base string
	hc   *http.Client
}

// New opens NeverBounce with an API key.
func New(cfg adapter.Config) (adapter.Adapter, error) {
	b := cfg.BaseURL
	if b == "" {
		b = base
	}
	return &client{key: cfg.APIKey, base: strings.TrimRight(b, "/"), hc: cfg.Client()}, nil
}

// status maps NeverBounce's status word to an error.
func status(word, msg string, code int) error {
	switch word {
	case "", "success":
		return nil
	case "auth_failure", "bad_referrer":
		return &adapter.CallError{Kind: adapter.ErrAuth, Status: code, Message: msg}
	case "throttle_triggered":
		return &adapter.CallError{Kind: adapter.ErrRateLimited, Status: code, Message: msg}
	case "temp_unavail":
		return &adapter.CallError{Kind: adapter.ErrUnavailable, Status: code, Message: msg}
	default:
		if m := strings.ToLower(msg); strings.Contains(m, "credit") {
			return &adapter.CallError{Kind: adapter.ErrNoCredits, Status: code, Message: msg}
		}
		return &adapter.CallError{Kind: adapter.ErrRefused, Status: code, Message: msg}
	}
}

func (c *client) Balance(ctx context.Context) (int64, error) {
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v4.2/account/info", Query: url.Values{"key": {c.key}}}, nil)
	if err != nil {
		return 0, err
	}
	var out struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Credits struct {
			Paid int64 `json:"paid_credits_remaining"`
			Free int64 `json:"free_credits_remaining"`
		} `json:"credits_info"`
	}
	if err := adapter.Decode(resp, &out); err != nil {
		return 0, err
	}
	if err := status(out.Status, out.Message, resp.Status); err != nil {
		return 0, err
	}
	return out.Credits.Paid + out.Credits.Free, nil
}

func (c *client) Verify(ctx context.Context, email string, timeout time.Duration) (adapter.Answer, error) {
	q := url.Values{"key": {c.key}, "email": {email}}
	if timeout > 0 {
		q.Set("timeout", strconv.Itoa(max(int(timeout/time.Second), 1)))
	}
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v4.2/single/check", Query: q}, nil)
	if err != nil {
		return adapter.Answer{}, err
	}
	var out struct {
		Status  string   `json:"status"`
		Message string   `json:"message"`
		Result  string   `json:"result"`
		Flags   []string `json:"flags"`
	}
	if err := adapter.Decode(resp, &out); err != nil {
		return adapter.Answer{}, err
	}
	if err := status(out.Status, out.Message, resp.Status); err != nil {
		return adapter.Answer{}, err
	}
	return Map(email, out.Result, out.Flags), nil
}

func (c *client) VerifyBatch(ctx context.Context, emails []string) ([]adapter.Answer, error) {
	return adapter.OneByOne(ctx, emails, 5, func(ctx context.Context, e string) (adapter.Answer, error) {
		return c.Verify(ctx, e, 10*time.Second)
	})
}

// Map turns NeverBounce's result and flags into an answer.
func Map(email, result string, flags []string) adapter.Answer {
	result = strings.ToLower(result)
	a := adapter.Answer{Email: email, Code: result}
	has := func(f string) bool { return slices.Contains(flags, f) }
	switch result {
	case "invalid":
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonInvalidAddress
		if has("spamtrap_network") || has("squatter_host") {
			a.Reason, a.Code = adapter.ReasonSpamTrap, result+" · spamtrap_network"
		}
	case "disposable":
		a.Result, a.Reason = adapter.Risky, adapter.ReasonDisposable
	case "catchall":
		a.Result, a.Reason = adapter.Risky, adapter.ReasonCatchAll
	case "valid":
		a.Result = adapter.Valid
		if has("role_account") {
			a.Result, a.Reason, a.Code = adapter.Risky, adapter.ReasonRole, result+" · role_account"
		}
	default:
		a.Result = adapter.Unknown
	}
	return a
}
