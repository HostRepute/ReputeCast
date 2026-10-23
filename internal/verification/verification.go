// Package verification checks contacts' addresses with the workspace's own
// verification service (ZeroBounce, NeverBounce, Kickbox, Bouncer,
// Emailable): connecting a service with a write-only API key and the plan's
// price per check, the uses (an import's new addresses, signup forms in real
// time, lists, segments, tiers, selections, one contact), and what each
// result does: invalid addresses are suppressed for all mail, risky and
// unknown ones are tagged, and every contact keeps its latest result.
//
// Addresses leave the instance only on purpose: every use is preceded by a
// notice that names the service and is audited, and only the addresses are
// sent. Real-time checks on forms are counted by day, not audited one by
// one (the roadmap's default for automatic, high-volume events).
package verification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification/adapter"

	// The five services of the catalog.
	_ "github.com/hostrepute/reputecast/internal/verification/adapter/bouncer"
	_ "github.com/hostrepute/reputecast/internal/verification/adapter/emailable"
	_ "github.com/hostrepute/reputecast/internal/verification/adapter/kickbox"
	_ "github.com/hostrepute/reputecast/internal/verification/adapter/neverbounce"
	_ "github.com/hostrepute/reputecast/internal/verification/adapter/zerobounce"
)

// Selection is what a bulk job checks: a list, a segment, one or more
// tiers, chosen contacts, or every contact matching a Contacts filter (a
// saved view).
type Selection struct {
	ListID     *uuid.UUID
	SegmentID  *uuid.UUID
	Tiers      []string
	ContactIDs []uuid.UUID
	// Filter is a Contacts filter (audience.ContactFilter as JSON).
	Filter json.RawMessage
}

// Audience resolves the contacts a job checks (internal/audience and
// internal/segments, wired in internal/app).
type Audience interface {
	// SelectionIDs returns the contacts of a selection that the caller can
	// see (list scope included).
	SelectionIDs(ctx context.Context, ws *store.Workspace, sel Selection) ([]uuid.UUID, error)
}

// Imports gives the new addresses of an import (its preview's).
type Imports interface {
	NewAddresses(ctx context.Context, ws *store.Workspace, importID uuid.UUID) (label string, emails []string, err error)
}

// Notifier tells members about finished jobs and low credits.
type Notifier interface {
	JobEnded(ctx context.Context, ws *store.Workspace, j store.VerificationJob)
	CreditsLow(ctx context.Context, ws *store.Workspace, svc store.VerificationService, needed int64, job *store.VerificationJob)
}

// Service is the verification domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	// Audience and Imports resolve what a job checks.
	Audience Audience
	Imports  Imports
	Notify   Notifier
	// Jobs is River's client, read lazily; nil (or Inline) runs jobs in
	// the request (tests).
	Jobs   func() *jobs.Client
	Inline bool
	// HTTP calls the services; BaseURLs points services at fakes (tests).
	HTTP     *http.Client
	BaseURLs map[string]string
	Now      func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// RecheckAfter is how long an unknown result waits before it's checked again.
const RecheckAfter = 72 * time.Hour

// SkipWithin is "Skip addresses checked in the last 90 days".
const SkipWithin = 90 * 24 * time.Hour

// LowCredits is the balance under which members with integrations:manage
// are told.
const LowCredits = 1000

// credentials is the sealed connection secret.
type credentials struct {
	APIKey string `json:"apiKey"`
	// Demo marks the demo's sample accounts: they answer from a recording
	// and never reach the service.
	Demo bool `json:"demo,omitempty"`
}

func credentialsAD(workspaceID uuid.UUID, service string) string {
	return "verification:" + workspaceID.String() + ":" + service + ":credentials"
}

// SealKey seals an API key for a workspace's service (the demo seeds with it).
func SealKey(box *secrets.Box, workspaceID uuid.UUID, service, key string, demo bool) ([]byte, error) {
	b, err := json.Marshal(credentials{APIKey: key, Demo: demo}) //nolint:gosec // G117: sealed on the next line, never stored or returned in clear
	if err != nil {
		return nil, err
	}
	return box.Seal(b, credentialsAD(workspaceID, service))
}

func (s *Service) credentials(v store.VerificationService) (credentials, error) {
	var c credentials
	if s.Box == nil {
		return c, errors.New("verification: no secrets box")
	}
	b, err := s.Box.Open(v.CredentialsEnc, credentialsAD(v.WorkspaceID, v.Service))
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

// open builds the adapter of a connected service.
func (s *Service) open(v store.VerificationService) (adapter.Adapter, adapter.Descriptor, error) {
	d, ok := adapter.Lookup(v.Service)
	if !ok {
		return nil, d, fmt.Errorf("verification: unknown service %q", v.Service)
	}
	c, err := s.credentials(v)
	if err != nil {
		return nil, d, err
	}
	if c.Demo {
		return &demoAdapter{service: v.Service, balance: ptr(v.Balance)}, d, nil
	}
	a, err := d.New(s.config(v.Service, c.APIKey, v.Region))
	return a, d, err
}

func (s *Service) config(service, key, region string) adapter.Config {
	return adapter.Config{APIKey: key, Region: region, BaseURL: s.BaseURLs[service], HTTPClient: s.HTTP}
}

func ptr(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// Money is a price in millionths of a dollar.
type Money int64

// Dollars converts to dollars.
func (m Money) Dollars() float64 { return float64(m) / 1e6 }

// FromDollars converts dollars, rounding to a millionth.
func FromDollars(d float64) Money { return Money(math.Round(d * 1e6)) }

// cost is n checks at price, in dollars rounded to the cent.
func cost(n int64, priceMicros int64) float64 {
	return math.Round(float64(n)*float64(priceMicros)/1e4) / 100
}

func actor(ctx context.Context) *uuid.UUID {
	if u := api.UserFrom(ctx); u != nil {
		id := u.ID
		return &id
	}
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		id := p.UserID
		return &id
	}
	return nil
}

// ServiceName is a service's display name ("ZeroBounce").
func ServiceName(key string) string {
	if d, ok := adapter.Lookup(key); ok {
		return d.Name
	}
	return key
}

// RegionLabel is where a service processes the addresses ("United States").
func RegionLabel(service, region string) string {
	d, ok := adapter.Lookup(service)
	if !ok {
		return ""
	}
	for _, r := range d.Regions {
		if r.Key == region {
			return r.Label
		}
	}
	return d.DefaultRegion
}

// SuppressionSource is how a service is recorded as the source of a
// suppression ("verify:zerobounce").
func SuppressionSource(service string) string { return "verify:" + service }

// SourceService returns the service of a suppression source, or "".
func SourceService(source string) string {
	if strings.HasPrefix(source, "verify:") {
		return strings.TrimPrefix(source, "verify:")
	}
	return ""
}

// demoAdapter answers for the demo's sample accounts without a network
// call: deterministic results by address, in the proportions of the
// sample (most valid, a few invalid, risky and unknown), the board's
// addresses answered as the boards show them.
type demoAdapter struct {
	service string
	balance int64
}

// demoAnswers are the sample's named addresses.
var demoAnswers = map[string]adapter.Answer{
	"g.okafor@outlook.com":        {Result: adapter.Valid, Code: "valid"},
	"m.keller@kellerdesign.ch":    {Result: adapter.Risky, Reason: adapter.ReasonCatchAll, Code: "catch-all"},
	"contact@lindqvist-studio.se": {Result: adapter.Risky, Reason: adapter.ReasonRole, Code: "do_not_mail / role_based"},
	"sam.rivera@outlook.com":      {Result: adapter.Valid, Code: "valid"},
	"lena.fischer@posteo.de":      {Result: adapter.Valid, Code: "valid"},
}

func (d *demoAdapter) Balance(context.Context) (int64, error) { return d.balance, nil }

func (d *demoAdapter) answer(email string) adapter.Answer {
	e := strings.ToLower(email)
	if a, ok := demoAnswers[e]; ok {
		a.Email = email
		return demoCode(d.service, a)
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(e))
	n := h.Sum32() % 1000
	a := adapter.Answer{Email: email}
	switch {
	case n < 803:
		a.Result = adapter.Valid
	case n < 880:
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonInvalidAddress
	case n < 881:
		a.Result, a.Reason = adapter.Invalid, adapter.ReasonSpamTrap
	case n < 950:
		a.Result, a.Reason = adapter.Risky, adapter.ReasonCatchAll
	case n < 965:
		a.Result, a.Reason = adapter.Risky, adapter.ReasonDisposable
	case n < 971:
		a.Result, a.Reason = adapter.Risky, adapter.ReasonRole
	default:
		a.Result = adapter.Unknown
	}
	return demoCode(d.service, a)
}

// demoCode writes the service's own code for a demo answer.
func demoCode(service string, a adapter.Answer) adapter.Answer {
	key := string(a.Result)
	if a.Reason != "" {
		key += "/" + a.Reason
	}
	switch service {
	case "zerobounce":
		if a.Code == "" {
			a.Code = demoZeroBounce[key]
		}
	case "kickbox":
		a.Code = demoKickbox[key]
	}
	if a.Code == "" {
		a.Code = string(a.Result)
	}
	return a
}

var demoZeroBounce = map[string]string{
	"valid": "valid", "invalid/invalid_address": "invalid / mailbox_not_found", "invalid/spam_trap": "spamtrap",
	"risky/catch_all": "catch-all", "risky/role": "do_not_mail / role_based", "risky/disposable": "do_not_mail / disposable",
	"unknown": "unknown / mail_server_did_not_respond",
}

var demoKickbox = map[string]string{
	"valid": "deliverable / accepted_email", "invalid/invalid_address": "undeliverable / rejected_email",
	"invalid/spam_trap": "undeliverable / rejected_email", "risky/catch_all": "risky / low_deliverability · accept_all",
	"risky/role": "deliverable / accepted_email · role", "risky/disposable": "risky / low_quality · disposable",
	"unknown": "unknown / no_connect",
}

// Verify is a single check: a signup form's, or a contact's Check now. In
// the demo it lets an address through unless it is one of the sample's or
// obviously made up (a .invalid domain), so trying a demo form never fails
// at random; jobs get the sample's spread from VerifyBatch.
func (d *demoAdapter) Verify(_ context.Context, email string, _ time.Duration) (adapter.Answer, error) {
	if d.balance <= 0 {
		return adapter.Answer{}, &adapter.CallError{Kind: adapter.ErrNoCredits, Message: "the demo account has no credits left"}
	}
	d.balance--
	e := strings.ToLower(email)
	if a, ok := demoAnswers[e]; ok {
		a.Email = email
		return demoCode(d.service, a), nil
	}
	if strings.HasSuffix(e, ".invalid") {
		return demoCode(d.service, adapter.Answer{Email: email, Result: adapter.Invalid, Reason: adapter.ReasonInvalidAddress}), nil
	}
	return demoCode(d.service, adapter.Answer{Email: email, Result: adapter.Valid}), nil
}

func (d *demoAdapter) VerifyBatch(_ context.Context, emails []string) ([]adapter.Answer, error) {
	out := make([]adapter.Answer, 0, len(emails))
	for _, e := range emails {
		if d.balance <= 0 {
			return out, &adapter.CallError{Kind: adapter.ErrNoCredits, Message: "the demo account has no credits left"}
		}
		d.balance--
		out = append(out, d.answer(e))
	}
	return out, nil
}
