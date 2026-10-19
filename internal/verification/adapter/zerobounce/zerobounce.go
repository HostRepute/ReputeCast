// Package zerobounce checks addresses with ZeroBounce's v2 API
// (https://www.zerobounce.net/docs/email-validation-api-quickstart/):
// GET /v2/getcredits, GET /v2/validate and POST /v2/validatebatch, on the
// default, United States or EU endpoint. A bad key and an empty balance
// answer the same error; the balance call tells them apart (-1 is a bad key).
package zerobounce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// Endpoints per data region.
var endpoints = map[string]string{
	"":   "https://api.zerobounce.net",
	"us": "https://api-us.zerobounce.net",
	"eu": "https://api-eu.zerobounce.net",
}

func init() {
	adapter.Register(adapter.Descriptor{
		Key: "zerobounce", Name: "ZeroBounce", Initials: "ZB",
		Capabilities:  adapter.Capabilities{BulkLists: true, RealTime: true, CatchAll: true, CreditBalance: true, DataRegion: true},
		Regions:       []adapter.Region{{Key: "us", Label: "United States"}, {Key: "eu", Label: "European Union"}},
		DefaultRegion: "United States",
		Website:       "https://www.zerobounce.net/members/",
		KeyHelp:       "https://www.zerobounce.net/members/API",
		// The batch endpoint takes up to 200 addresses and can take 70 s.
		MaxBatch:  100,
		PerSecond: 20,
		Mappings: []adapter.Mapping{
			{Codes: []string{"valid"}, Result: adapter.Valid},
			{Codes: []string{"invalid / mailbox_not_found", "no_dns_entries", "failed_syntax_check"}, Result: adapter.Invalid, Reason: adapter.ReasonInvalidAddress},
			{Codes: []string{"spamtrap"}, Result: adapter.Invalid, Reason: adapter.ReasonSpamTrap},
			{Codes: []string{"catch-all"}, Result: adapter.Risky, Reason: adapter.ReasonCatchAll},
			{Codes: []string{"do_not_mail / role_based"}, Result: adapter.Risky, Reason: adapter.ReasonRole},
			{Codes: []string{"do_not_mail / disposable"}, Result: adapter.Risky, Reason: adapter.ReasonDisposable},
			{Codes: []string{"abuse"}, Result: adapter.Risky, Reason: "abuse"},
			{Codes: []string{"unknown"}, Result: adapter.Unknown},
		},
		New: New,
	})
}

type client struct {
	cfg  adapter.Config
	base string
	hc   *http.Client
}

// New opens ZeroBounce with an API key.
func New(cfg adapter.Config) (adapter.Adapter, error) {
	base := cfg.BaseURL
	if base == "" {
		base = endpoints[cfg.Region]
		if base == "" {
			base = endpoints[""]
		}
	}
	return &client{cfg: cfg, base: strings.TrimRight(base, "/"), hc: cfg.Client()}, nil
}

// keyOrCredits is ZeroBounce's shared answer for a bad key and no credits.
func keyOrCredits(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "invalid api key") || strings.Contains(m, "ran out of credits")
}

func (c *client) Balance(ctx context.Context) (int64, error) {
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v2/getcredits", Query: url.Values{"api_key": {c.cfg.APIKey}}}, nil)
	if err != nil {
		return 0, err
	}
	var out struct {
		Credits json.Number `json:"Credits"`
	}
	if err := adapter.Decode(resp, &out); err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(strings.Split(out.Credits.String(), ".")[0], 10, 64)
	if err != nil {
		return 0, &adapter.CallError{Kind: adapter.ErrUnavailable, Status: resp.Status, Message: "the balance wasn't a number"}
	}
	if n < 0 {
		return 0, &adapter.CallError{Kind: adapter.ErrAuth, Status: resp.Status, Message: "ZeroBounce answered -1 credits: the API key is invalid"}
	}
	return n, nil
}

// answer is one validation result.
type answer struct {
	Address   string `json:"address"`
	Status    string `json:"status"`
	SubStatus string `json:"sub_status"`
	Error     string `json:"error"`
}

func (c *client) Verify(ctx context.Context, email string, timeout time.Duration) (adapter.Answer, error) {
	q := url.Values{"api_key": {c.cfg.APIKey}, "email": {email}, "ip_address": {""}}
	if timeout > 0 {
		// ZeroBounce takes 3 to 60 seconds.
		s := min(max(int(timeout/time.Second), 3), 60)
		q.Set("timeout", strconv.Itoa(s))
	}
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{URL: c.base + "/v2/validate", Query: q}, nil)
	if err != nil {
		return adapter.Answer{}, err
	}
	var a answer
	if err := adapter.Decode(resp, &a); err != nil {
		return adapter.Answer{}, err
	}
	if a.Error != "" {
		return adapter.Answer{}, c.keyError(ctx, resp.Status, a.Error)
	}
	if a.Address == "" {
		a.Address = email
	}
	return Map(a.Address, a.Status, a.SubStatus), nil
}

// keyError tells a bad key from an empty balance by reading the balance.
func (c *client) keyError(ctx context.Context, status int, msg string) error {
	if !keyOrCredits(msg) {
		return &adapter.CallError{Kind: adapter.ErrRefused, Status: status, Message: msg}
	}
	n, err := c.Balance(ctx)
	switch {
	case err != nil:
		return err
	case n <= 0:
		return &adapter.CallError{Kind: adapter.ErrNoCredits, Status: status, Message: msg}
	default:
		return &adapter.CallError{Kind: adapter.ErrAuth, Status: status, Message: msg}
	}
}

func (c *client) VerifyBatch(ctx context.Context, emails []string) ([]adapter.Answer, error) {
	type item struct {
		Email string  `json:"email_address"`
		IP    *string `json:"ip_address"`
	}
	batch := make([]item, len(emails))
	for i, e := range emails {
		batch[i] = item{Email: e}
	}
	resp, err := adapter.Do(ctx, c.hc, adapter.Request{Method: http.MethodPost, URL: c.base + "/v2/validatebatch", Body: map[string]any{"api_key": c.cfg.APIKey, "email_batch": batch}}, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		EmailBatch []answer `json:"email_batch"`
		Errors     []struct {
			Error string `json:"error"`
			Email string `json:"email_address"`
		} `json:"errors"`
	}
	if err := adapter.Decode(resp, &out); err != nil {
		return nil, err
	}
	if len(out.EmailBatch) == 0 && len(out.Errors) > 0 {
		return nil, c.keyError(ctx, resp.Status, out.Errors[0].Error)
	}
	by := make(map[string]answer, len(out.EmailBatch))
	for _, a := range out.EmailBatch {
		by[strings.ToLower(a.Address)] = a
	}
	answers := make([]adapter.Answer, 0, len(emails))
	for _, e := range emails {
		a, ok := by[strings.ToLower(e)]
		if !ok {
			// Addresses the batch left out are checked again later.
			return answers, &adapter.CallError{Kind: adapter.ErrUnavailable, Status: resp.Status, Message: "ZeroBounce left addresses out of the batch"}
		}
		answers = append(answers, Map(e, a.Status, a.SubStatus))
	}
	return answers, nil
}

// Map turns ZeroBounce's status and sub-status into an answer.
func Map(email, status, sub string) adapter.Answer {
	status, sub = strings.ToLower(strings.TrimSpace(status)), strings.ToLower(strings.TrimSpace(sub))
	code := status
	if sub != "" {
		code = status + " / " + sub
	}
	a := adapter.Answer{Email: email, Code: code}
	switch status {
	case "valid":
		a.Result = adapter.Valid
	case "invalid":
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonInvalidAddress
	case "spamtrap":
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonSpamTrap
	case "catch-all":
		a.Result, a.Reason = adapter.Risky, adapter.ReasonCatchAll
	case "abuse":
		a.Result, a.Reason = adapter.Risky, "abuse"
	case "do_not_mail":
		a.Result = adapter.Risky
		switch sub {
		case "role_based", "role_based_catch_all":
			a.Reason = adapter.ReasonRole
		case "disposable":
			a.Reason = adapter.ReasonDisposable
		case "possible_trap":
			// "A possible spam trap answer uses the reason Spam trap."
			a.Result, a.Reason = adapter.Invalid, adapter.ReasonSpamTrap
		case "":
			a.Reason = "do_not_mail"
		default:
			a.Reason = sub
		}
	default:
		a.Result = adapter.Unknown
	}
	return a
}
