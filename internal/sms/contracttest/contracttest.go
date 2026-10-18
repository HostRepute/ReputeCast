// Package contracttest is the shared contract suite every SMS adapter
// passes (brief §3.1): sending, delivery receipts and inbound messages,
// signature checks, and the optional capabilities its manifest declares:
// the account's numbers, webhook registration, opt-out sync, registration
// status and pricing.
//
// It runs against the adapter's fake (recorded payloads, no network): the
// fake serves the provider's API in each failure mode and signs webhooks
// the way the provider would.
package contracttest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sms"
)

// Modes the fake API answers in.
const (
	ModeOK = ""
	// ModeOutage answers 5xx to every call.
	ModeOutage = "outage"
	// ModeAuth answers 401: the credentials were revoked.
	ModeAuth = "auth"
)

// InvalidNumber is a recipient the fake refuses as not a valid number.
const InvalidNumber = "+15005550001"

// LandlineNumber is a recipient the fake refuses as a landline (or reports
// undelivered as one, for providers that only learn it later).
const LandlineNumber = "+15005550009"

// Env is a running fake.
type Env struct {
	URL  string
	mu   sync.Mutex
	mode string
}

// Mode is the fake's current failure mode.
func (e *Env) Mode() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.mode
}

// SetMode switches the fake's failure mode.
func (e *Env) SetMode(m string) {
	e.mu.Lock()
	e.mode = m
	e.mu.Unlock()
}

// Webhook is a recorded webhook and what it must parse into.
type Webhook struct {
	Name string
	// Request builds the webhook as the provider sends it to url, signed.
	Request func(t *testing.T, env *Env, url string) (*http.Request, []byte)
	Want    []sms.Event
}

// Suite describes an adapter under test.
type Suite struct {
	Adapter string
	// API serves the provider's API from recorded payloads; it reads
	// env.Mode() for failure modes.
	API func(env *Env) http.Handler
	// New opens the adapter on the fake.
	New func(t *testing.T, env *Env) sms.Provider
	// From is the sender the send tests use.
	From string
	// Webhooks are recorded receipts and inbound texts (at least one of
	// each).
	Webhooks []Webhook
	// Tamper changes a signed webhook so its signature no longer matches.
	Tamper func(r *http.Request, body []byte) (*http.Request, []byte)
	// SkipOptOutPush leaves out pushing numbers for adapters whose fake
	// keeps a list it can't be told about (none so far).
	SkipOptOutPush bool
}

// HookURL is where the suite's webhooks are addressed.
const HookURL = "https://reputecast.contract.test/hooks/sms/contract/0123456789abcdef"

// Run runs the suite.
func Run(t *testing.T, s Suite) {
	t.Helper()
	env := &Env{}
	srv := httptest.NewServer(s.API(env))
	t.Cleanup(srv.Close)
	env.URL = srv.URL
	ctx := context.Background()

	d, ok := sms.Lookup(s.Adapter)
	if !ok {
		t.Fatalf("adapter %q is not registered", s.Adapter)
	}
	t.Run("Manifest", func(t *testing.T) { checkManifest(t, d) })

	p := s.New(t, env)
	t.Run("Capabilities", func(t *testing.T) { checkCapabilities(t, d, p) })

	t.Run("Check", func(t *testing.T) {
		env.SetMode(ModeOK)
		a, err := p.Check(ctx)
		if err != nil {
			t.Fatalf("check: %v", err)
		}
		if a.Hint == "" || a.Name == "" {
			t.Fatalf("the account needs a name and a hint: %+v", a)
		}
	})
	t.Run("CheckRejected", func(t *testing.T) {
		env.SetMode(ModeAuth)
		defer env.SetMode(ModeOK)
		if _, err := p.Check(ctx); !errors.Is(err, sms.ErrCredentialsRejected) {
			t.Fatalf("revoked credentials: %v, want ErrCredentialsRejected", err)
		}
	})
	t.Run("Send", func(t *testing.T) {
		env.SetMode(ModeOK)
		for i, body := range []string{"Acme: This is a test from ReputeCast. Reply STOP to opt out", "Acme: Doors open at 8:30. You" + string(rune(0x2019)) + "ll find badges at the north entrance."} {
			c := sms.CountText(body)
			r, err := p.Send(ctx, sms.Message{ID: uuid.Must(uuid.NewV7()), To: []string{"+13125550161", "+447700900456"}[i], From: s.From, Body: body,
				Encoding: c.Encoding, StatusURL: HookURL})
			if err != nil {
				t.Fatalf("send %d: %v", i, err)
			}
			if r.ProviderMessageID == "" {
				t.Fatalf("send %d: no provider message id", i)
			}
		}
	})
	t.Run("SendInvalid", func(t *testing.T) {
		env.SetMode(ModeOK)
		_, err := p.Send(ctx, sms.Message{ID: uuid.Must(uuid.NewV7()), To: InvalidNumber, From: s.From, Body: "Acme: test", Encoding: sms.GSM7, StatusURL: HookURL})
		var se *sms.SendError
		if !errors.As(err, &se) {
			t.Fatalf("an invalid number: %v, want a *SendError", err)
		}
		if se.Temporary || se.Reason != sms.ReasonInvalid || se.Code == "" {
			t.Fatalf("an invalid number is permanent, reason invalid, with the provider's code: %+v", se)
		}
	})
	t.Run("SendOutage", func(t *testing.T) {
		env.SetMode(ModeOutage)
		defer env.SetMode(ModeOK)
		_, err := p.Send(ctx, sms.Message{ID: uuid.Must(uuid.NewV7()), To: "+13125550161", From: s.From, Body: "Acme: test", Encoding: sms.GSM7, StatusURL: HookURL})
		var se *sms.SendError
		if !errors.Is(err, sms.ErrUnavailable) && (!errors.As(err, &se) || !se.Temporary) {
			t.Fatalf("an outage: %v, want ErrUnavailable or a temporary error", err)
		}
	})
	t.Run("SendAuth", func(t *testing.T) {
		env.SetMode(ModeAuth)
		defer env.SetMode(ModeOK)
		_, err := p.Send(ctx, sms.Message{ID: uuid.Must(uuid.NewV7()), To: "+13125550161", From: s.From, Body: "Acme: test", Encoding: sms.GSM7, StatusURL: HookURL})
		if !errors.Is(err, sms.ErrCredentialsRejected) {
			t.Fatalf("revoked credentials: %v, want ErrCredentialsRejected", err)
		}
	})

	receipts, inbound := 0, 0
	for _, w := range s.Webhooks {
		for _, e := range w.Want {
			if e.Kind == sms.EventInbound {
				inbound++
			} else {
				receipts++
			}
		}
		t.Run("Webhook/"+w.Name, func(t *testing.T) {
			env.SetMode(ModeOK)
			r, body := w.Request(t, env, HookURL)
			if err := p.VerifyWebhook(ctx, r, body); err != nil {
				t.Fatalf("verify: %v", err)
			}
			got, err := p.ParseWebhook(ctx, r.Header.Get("Content-Type"), body)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			compareEvents(t, got, w.Want)
			if s.Tamper != nil {
				r2, b2 := w.Request(t, env, HookURL)
				r2, b2 = s.Tamper(r2, b2)
				if err := p.VerifyWebhook(ctx, r2, b2); !errors.Is(err, sms.ErrBadSignature) {
					t.Fatalf("a tampered webhook: %v, want ErrBadSignature", err)
				}
			}
			// A webhook for another URL (a replay at another address) fails
			// for providers that sign the URL; at worst it still verifies.
		})
	}
	t.Run("WebhookCoverage", func(t *testing.T) {
		if receipts == 0 || inbound == 0 {
			t.Fatalf("the suite needs recorded receipts and inbound texts (%d receipts, %d inbound)", receipts, inbound)
		}
		if s.Tamper == nil {
			t.Fatal("every adapter checks signatures: the suite needs Tamper")
		}
	})

	if l, ok := p.(sms.SenderLister); ok {
		t.Run("Senders", func(t *testing.T) {
			ss, err := l.Senders(ctx)
			if err != nil || len(ss) == 0 {
				t.Fatalf("senders: %v (%d)", err, len(ss))
			}
			for _, x := range ss {
				if !slices.Contains([]string{"toll_free", "10dlc", "local", "mobile", "sender_id"}, x.Kind) || x.Value == "" {
					t.Fatalf("sender %+v", x)
				}
			}
		})
	}
	if reg, ok := p.(sms.WebhookRegistrar); ok {
		t.Run("RegisterWebhooks", func(t *testing.T) {
			if _, err := reg.RegisterWebhooks(ctx, sms.Webhooks{StatusURL: HookURL, InboundURL: HookURL}, []sms.Sender{{Kind: "toll_free", Value: s.From}}); err != nil {
				t.Fatalf("register webhooks: %v", err)
			}
		})
	}
	if r, ok := p.(sms.OptOutReader); ok {
		t.Run("OptOuts", func(t *testing.T) {
			list, err := r.OptOuts(ctx)
			if err != nil {
				t.Fatalf("opt-outs: %v", err)
			}
			if syncer, ok := p.(sms.OptOutSyncer); ok && !s.SkipOptOutPush {
				if err := syncer.AddOptOuts(ctx, []string{"+13125550199"}); err != nil {
					t.Fatalf("add opt-outs: %v", err)
				}
				after, err := r.OptOuts(ctx)
				if err != nil || !hasPhone(after, "+13125550199") {
					t.Fatalf("an added number is on the list: %v %+v", err, after)
				}
				if err := syncer.RemoveOptOut(ctx, "+13125550199"); err != nil {
					t.Fatalf("remove opt-out: %v", err)
				}
				if again, _ := r.OptOuts(ctx); hasPhone(again, "+13125550199") {
					t.Fatal("a removed number stays on the list")
				}
			}
			_ = list
		})
	}
	if rr, ok := p.(sms.RegistrationReader); ok {
		t.Run("Registrations", func(t *testing.T) {
			regs, err := rr.Registrations(ctx, []sms.Sender{{Kind: "toll_free", Value: s.From}})
			if err != nil {
				t.Fatalf("registrations: %v", err)
			}
			for _, g := range regs {
				if !slices.Contains([]string{"not_registered", "pending", "verified", "approved", "rejected", "allowed", "not_supported"}, g.Status) {
					t.Fatalf("registration status %q isn't one of ReputeCast's", g.Status)
				}
			}
		})
	}
	if pr, ok := p.(sms.PricingReader); ok {
		t.Run("Prices", func(t *testing.T) {
			prices, err := pr.Prices(ctx, []string{"US", "GB"})
			if err != nil {
				t.Fatalf("prices: %v", err)
			}
			if prices["US"] <= 0 && prices["GB"] <= 0 {
				t.Fatalf("prices %+v", prices)
			}
		})
	}
}

func hasPhone(list []sms.OptOut, phone string) bool {
	return slices.ContainsFunc(list, func(o sms.OptOut) bool { return o.Phone == phone })
}

var secretish = []string{"secret", "token", "password", "private"}

func checkManifest(t *testing.T, d sms.Descriptor) {
	if d.Name == "" || d.Summary == "" || d.New == nil || d.Maintainer == "" || d.Version == "" {
		t.Fatalf("the manifest needs a name, a summary, a version, a maintainer and New: %+v", d.Manifest)
	}
	if d.Webhooks != "registered" && d.Webhooks != "manual" {
		t.Fatalf("webhooks %q: registered or manual", d.Webhooks)
	}
	keys := map[string]bool{}
	for _, f := range d.Fields {
		if keys[f.Key] {
			t.Fatalf("field %q twice", f.Key)
		}
		keys[f.Key] = true
		for _, s := range secretish {
			if strings.Contains(f.Key, s) && !f.Secret && !strings.HasSuffix(f.Key, "_sid") && !strings.HasSuffix(f.Key, "_id") {
				t.Fatalf("field %q looks secret and must be write-only", f.Key)
			}
		}
	}
	seen := map[sms.Capability]bool{}
	for _, b := range d.Badges {
		if seen[b.Capability] {
			t.Fatalf("badge %q twice", b.Capability)
		}
		seen[b.Capability] = true
		if b.Support != sms.Yes && b.Support != sms.Partial && b.Support != sms.No {
			t.Fatalf("badge %q support %q", b.Capability, b.Support)
		}
		if b.Support == sms.Partial && b.Note == "" {
			t.Fatalf("a partial badge says how (%q)", b.Capability)
		}
	}
	for _, c := range sms.Capabilities {
		if !seen[c] {
			t.Fatalf("the manifest declares every capability; %q is missing", c)
		}
	}
	// Receipts and inbound texts are required of every adapter.
	if d.Has(sms.CapReceipts) != sms.Yes || d.Has(sms.CapInbound) != sms.Yes {
		t.Fatal("every SMS adapter handles delivery receipts and inbound texts")
	}
	if d.Console == nil || d.Console("registration", "") == "" {
		t.Fatal("the manifest links to the provider's console")
	}
}

// checkCapabilities holds the manifest to what the adapter implements.
func checkCapabilities(t *testing.T, d sms.Descriptor, p sms.Provider) {
	_, reads := p.(sms.OptOutReader)
	_, syncs := p.(sms.OptOutSyncer)
	switch d.Has(sms.CapOptOutSync) {
	case sms.Yes:
		if !syncs {
			t.Fatal("opt-out sync: yes means two-way (OptOutSyncer)")
		}
	case sms.No:
		if reads {
			t.Fatal("opt-out sync: no, but the adapter reads the opt-out list")
		}
	}
	_, regs := p.(sms.RegistrationReader)
	if (d.Has(sms.CapRegistration) == sms.Yes) != regs {
		t.Fatalf("registration status via API: manifest %s, RegistrationReader %v", d.Has(sms.CapRegistration), regs)
	}
	_, prices := p.(sms.PricingReader)
	if (d.Has(sms.CapPricing) == sms.Yes) != prices {
		t.Fatalf("pricing via API: manifest %s, PricingReader %v", d.Has(sms.CapPricing), prices)
	}
	_, registers := p.(sms.WebhookRegistrar)
	if (d.Webhooks == "registered") != registers {
		t.Fatalf("webhooks %s, WebhookRegistrar %v", d.Webhooks, registers)
	}
	if p.Adapter() != d.Adapter {
		t.Fatalf("Adapter() %q, registered as %q", p.Adapter(), d.Adapter)
	}
}

func compareEvents(t *testing.T, got, want []sms.Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d events, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Kind != w.Kind || g.ProviderMessageID != w.ProviderMessageID || g.Code != w.Code || g.Reason != w.Reason ||
			(w.To != "" && g.To != w.To) || (w.From != "" && g.From != w.From) || (w.Body != "" && g.Body != w.Body) || g.OptOut != w.OptOut {
			t.Fatalf("event %d = %+v, want %+v", i, g, w)
		}
		if g.OccurredAt.IsZero() {
			t.Fatalf("event %d has no time", i)
		}
		if g.Summary == "" {
			t.Fatalf("event %d needs the raw summary the Events tab shows", i)
		}
	}
}
