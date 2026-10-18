// Package contracttest is the shared suite every verification service
// adapter passes, like the sending adapters' suite: registration and
// capabilities, the balance (which sends no address), each of the four
// results with the service's own code, batches in order, and the errors
// jobs act on (a rejected key, no credits, rate limits, an outage), against
// a fake of the service's API that serves recorded answers.
//
//	func TestContract(t *testing.T) {
//		contracttest.Run(t, contracttest.Suite{Service: "zerobounce", API: fake})
//	}
//
// The fake reads Env.Mode and the address in the request to choose a
// recorded answer: the addresses below stand for each result.
package contracttest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// Modes the fake API simulates.
const (
	ModeOK          = ""
	ModeAuth        = "auth"         // the key is rejected
	ModeNoCredits   = "no_credits"   // the account is empty
	ModeRateLimited = "rate_limited" // 429 or the service's throttle answer
	ModeOutage      = "outage"       // 5xx
)

// Addresses the fake answers for, one per expected answer.
const (
	AddrValid      = "valid@contract.test"
	AddrInvalid    = "invalid@contract.test"
	AddrCatchAll   = "catchall@contract.test"
	AddrRole       = "role@contract.test"
	AddrDisposable = "disposable@contract.test"
	AddrUnknown    = "unknown@contract.test"
	// AddrSpamTrap is answered as a spam trap by services that report them.
	AddrSpamTrap = "trap@contract.test"
)

// Key is the API key the suite connects with.
const Key = "contract-test-key-7f3a91c2"

// Balance is what the fake's balance answer holds.
const Balance = 36656

// Env is shared between the suite and the fake API.
type Env struct {
	mu       sync.Mutex
	mode     string
	requests []Recorded
}

// Recorded is one request the fake received.
type Recorded struct {
	Method, Path, Query string
	Header              http.Header
	Body                string
}

// Mode is the simulated failure.
func (e *Env) Mode() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.mode
}

func (e *Env) setMode(m string) {
	e.mu.Lock()
	e.mode = m
	e.mu.Unlock()
}

// Requests returns what the fake received so far.
func (e *Env) Requests() []Recorded {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Recorded(nil), e.requests...)
}

func (e *Env) reset() {
	e.mu.Lock()
	e.requests = nil
	e.mu.Unlock()
}

// Address returns the address a request asks about: the email query
// parameter, or the first contract address in the body.
func Address(r *http.Request, body string) string {
	if v := r.URL.Query().Get("email"); v != "" {
		return v
	}
	for _, a := range []string{AddrValid, AddrInvalid, AddrCatchAll, AddrRole, AddrDisposable, AddrUnknown, AddrSpamTrap} {
		if strings.Contains(body, a) {
			return a
		}
	}
	return ""
}

// Suite configures a run.
type Suite struct {
	// Service is the registry key.
	Service string
	// API is the fake service: it gets the Env to read the mode.
	API func(env *Env) http.Handler
	// SpamTrap is true when the service reports spam traps.
	SpamTrap bool
}

type want struct {
	email  string
	result adapter.Result
	reason string
}

// Run runs the suite.
func Run(t *testing.T, s Suite) {
	t.Helper()
	d, ok := adapter.Lookup(s.Service)
	if !ok {
		t.Fatalf("service %q is not registered", s.Service)
	}
	env := &Env{}
	api := s.API(env)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		env.mu.Lock()
		env.requests = append(env.requests, Recorded{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Header: r.Header.Clone(), Body: string(b)})
		env.mu.Unlock()
		r.Body = io.NopCloser(strings.NewReader(string(b)))
		api.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	a, err := d.New(adapter.Config{APIKey: Key, BaseURL: srv.URL, HTTPClient: &http.Client{Timeout: 5 * time.Second}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	t.Run("descriptor", func(t *testing.T) {
		if d.Name == "" || d.Initials == "" || d.Website == "" {
			t.Errorf("descriptor needs a name, initials and website: %+v", d)
		}
		if !d.Capabilities.BulkLists || !d.Capabilities.RealTime || !d.Capabilities.CreditBalance {
			t.Errorf("every catalog service checks lists and signups and reads its balance: %+v", d.Capabilities)
		}
		if d.MaxBatch < 1 || d.PerSecond <= 0 {
			t.Errorf("MaxBatch and PerSecond pace jobs: %d, %v", d.MaxBatch, d.PerSecond)
		}
		results := map[adapter.Result]bool{}
		for _, m := range d.Mappings {
			if len(m.Codes) == 0 {
				t.Errorf("a mapping row needs the service's codes: %+v", m)
			}
			results[m.Result] = true
		}
		for _, r := range []adapter.Result{adapter.Valid, adapter.Invalid, adapter.Risky, adapter.Unknown} {
			if !results[r] {
				t.Errorf("the mapping table has no row that becomes %s", r)
			}
		}
	})

	t.Run("balance sends no address", func(t *testing.T) {
		env.setMode(ModeOK)
		env.reset()
		n, err := a.Balance(ctx)
		if err != nil {
			t.Fatalf("Balance: %v", err)
		}
		if n != Balance {
			t.Errorf("balance = %d, want %d", n, Balance)
		}
		for _, r := range env.Requests() {
			if strings.Contains(r.Query, "%40") || strings.Contains(r.Query, "@") || strings.Contains(r.Body, "@") {
				t.Errorf("the balance call sent an address: %s %s?%s %s", r.Method, r.Path, r.Query, r.Body)
			}
			if !sendsKey(r) {
				t.Errorf("the balance call didn't send the key: %+v", r)
			}
		}
	})

	cases := []want{
		{AddrValid, adapter.Valid, ""},
		{AddrInvalid, adapter.Invalid, adapter.ReasonInvalidAddress},
		{AddrCatchAll, adapter.Risky, adapter.ReasonCatchAll},
		{AddrRole, adapter.Risky, adapter.ReasonRole},
		{AddrDisposable, adapter.Risky, adapter.ReasonDisposable},
		{AddrUnknown, adapter.Unknown, ""},
	}
	if s.SpamTrap {
		cases = append(cases, want{AddrSpamTrap, adapter.Invalid, adapter.ReasonSpamTrap})
	}

	t.Run("verify maps every result", func(t *testing.T) {
		env.setMode(ModeOK)
		for _, c := range cases {
			got, err := a.Verify(ctx, c.email, 3*time.Second)
			if err != nil {
				t.Fatalf("Verify(%s): %v", c.email, err)
			}
			if got.Result != c.result || got.Reason != c.reason {
				t.Errorf("Verify(%s) = %s/%s, want %s/%s", c.email, got.Result, got.Reason, c.result, c.reason)
			}
			if got.Code == "" {
				t.Errorf("Verify(%s) kept no code from the service", c.email)
			}
			if !strings.EqualFold(got.Email, c.email) {
				t.Errorf("Verify(%s) answered for %q", c.email, got.Email)
			}
			if c.result == adapter.Risky && !strings.HasPrefix(got.Tag(), "verify:") {
				t.Errorf("a risky answer is tagged verify:…, got %q", got.Tag())
			}
		}
	})

	t.Run("batch keeps the order", func(t *testing.T) {
		env.setMode(ModeOK)
		emails := make([]string, 0, len(cases))
		for i := len(cases) - 1; i >= 0; i-- {
			emails = append(emails, cases[i].email)
		}
		got, err := a.VerifyBatch(ctx, emails)
		if err != nil {
			t.Fatalf("VerifyBatch: %v", err)
		}
		if len(got) != len(emails) {
			t.Fatalf("VerifyBatch returned %d answers for %d addresses", len(got), len(emails))
		}
		for i, e := range emails {
			if !strings.EqualFold(got[i].Email, e) {
				t.Errorf("answer %d is for %s, want %s", i, got[i].Email, e)
			}
		}
	})

	failures := []struct {
		mode string
		err  error
	}{
		{ModeAuth, adapter.ErrAuth},
		{ModeNoCredits, adapter.ErrNoCredits},
		{ModeRateLimited, adapter.ErrRateLimited},
		{ModeOutage, adapter.ErrUnavailable},
	}
	for _, f := range failures {
		t.Run("verify "+f.mode, func(t *testing.T) {
			env.setMode(f.mode)
			defer env.setMode(ModeOK)
			_, err := a.Verify(ctx, AddrValid, 3*time.Second)
			if !errors.Is(err, f.err) {
				t.Fatalf("Verify in %s mode: %v, want %v", f.mode, err, f.err)
			}
			if adapter.Temporary(err) != (f.mode == ModeRateLimited || f.mode == ModeOutage) {
				t.Errorf("Temporary(%v) = %v", err, adapter.Temporary(err))
			}
			if strings.Contains(err.Error(), Key) {
				t.Errorf("the error shows the key: %v", err)
			}
			if _, err := a.VerifyBatch(ctx, []string{AddrValid, AddrInvalid}); !errors.Is(err, f.err) {
				t.Errorf("VerifyBatch in %s mode: %v, want %v", f.mode, err, f.err)
			}
		})
	}

	t.Run("balance with a rejected key", func(t *testing.T) {
		env.setMode(ModeAuth)
		defer env.setMode(ModeOK)
		if _, err := a.Balance(ctx); !errors.Is(err, adapter.ErrAuth) {
			t.Fatalf("Balance with a rejected key: %v, want ErrAuth", err)
		}
	})

	t.Run("balance in an outage", func(t *testing.T) {
		env.setMode(ModeOutage)
		defer env.setMode(ModeOK)
		if _, err := a.Balance(ctx); !errors.Is(err, adapter.ErrUnavailable) {
			t.Fatalf("Balance in an outage: %v, want ErrUnavailable", err)
		}
	})
}

// sendsKey reports whether a request carries the suite's key, in the query,
// a header or the body.
func sendsKey(r Recorded) bool {
	if strings.Contains(r.Query, Key) || strings.Contains(r.Body, Key) {
		return true
	}
	for _, vs := range r.Header {
		for _, v := range vs {
			if strings.Contains(v, Key) {
				return true
			}
		}
	}
	return false
}
