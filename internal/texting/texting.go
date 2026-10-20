// Package texting is ReputeCast's SMS channel (design expansion phase 8,
// M31): SMS providers and their senders with the carriers' registrations,
// routing by country, the SMS rules (keywords and replies, quiet hours,
// the brand prefix and opt-out text, the budget and auto-pause thresholds,
// the short-link domain), phone numbers and SMS consent per list, the SMS
// suppression list, SMS campaigns from the composer to the report, the Send
// SMS automation step, provider webhooks and the text message terms page.
//
// Providers are reached through internal/sms only; the channel model it
// shares with push and WhatsApp is internal/channels. Texts, receipts,
// replies and clicks stay in the message log for 90 days, never in the
// audit log; everything a person or key changes is audited.
package texting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service is the SMS domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	URLs  publicurl.Builder
	// Jobs enqueues River jobs (read at call time: the client is built after
	// the services).
	Jobs func() *jobs.Client
	Now  func() time.Time
	// HTTP is the SSRF-guarded client providers are called through.
	HTTP *http.Client
	// DNS checks the short-link domain's CNAME.
	DNS dnscheck.Resolver
	// Demo answers a demo provider's calls in process: its client and base
	// URL. The demo's sample providers never reach the real provider.
	Demo func(adapter string) (*http.Client, string)
	// Segments resolves a segment's contacts for an audience.
	Segments SegmentResolver
	// Approvals evaluates the workspace's approval rules for SMS campaigns
	// and holds their requests (internal/approvals, through the app).
	Approvals Approvals
	// PageBase is the workspace's own site address (https://news.acme.io)
	// when it has a verified custom domain, for the terms page's address.
	PageBase func(ctx context.Context, ws *store.Workspace) string
	// Emit sends outgoing webhook events (internal/webhooks).
	Emit func(ctx context.Context, q store.DBTX, ws *store.Workspace, event string, data any) error
	// Changed re-checks a campaign's open approval after an edit.
	Changed func(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID)
	// DemoReceipt builds the delivery receipt a demo provider's fake would
	// send for a text, signed as the provider signs it; the demo plays it
	// back through the webhook path a few seconds after the send. False
	// when the adapter has no fake.
	DemoReceipt func(adapter string, cfg sms.Config, hookURL, providerMessageID, to, from string, at time.Time) (*http.Request, []byte, bool)
}

// demoReceipt plays a demo provider's delivery receipt back through the
// webhook path, as the real provider would send it, a few seconds after
// the text went.
func (s *Service) demoReceipt(p store.SmsProvider, providerMessageID, to, from string) {
	if s.DemoReceipt == nil || providerMessageID == "" {
		return
	}
	cfg, err := s.config(p)
	if err != nil {
		return
	}
	delay := 2*time.Second + time.Duration(len(providerMessageID)%3)*500*time.Millisecond
	time.AfterFunc(delay, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r, body, ok := s.DemoReceipt(p.Adapter, cfg, s.hookURL(p), providerMessageID, to, from, s.now())
		if !ok {
			return
		}
		if code := s.Ingest(ctx, p.Adapter, p.WebhookToken, r, body); code != http.StatusOK {
			s.log().WarnContext(ctx, "sms demo receipt", "provider", p.ID, "status", code)
		}
	})
}

// SegmentResolver lists a segment's contacts.
type SegmentResolver interface {
	ContactIDs(ctx context.Context, workspaceID, segmentID uuid.UUID) ([]uuid.UUID, error)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) jobs() *jobs.Client {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// tx runs fn in a transaction with the store bound to it.
func (s *Service) tx(ctx context.Context, fn func(q *store.Queries, tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(store.New(tx), tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// credentialsAD and secretAD name what a sealed value belongs to.
func credentialsAD(id uuid.UUID) string { return "sms_provider:" + id.String() + ":credentials" }
func secretAD(id uuid.UUID) string      { return "sms_provider:" + id.String() + ":webhook_secret" }

// settingsOf reads a provider's non-secret settings.
func settingsOf(p store.SmsProvider) map[string]string {
	m := map[string]string{}
	var raw map[string]any
	if json.Unmarshal(p.Config, &raw) == nil {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				m[k] = s
			}
		}
	}
	return m
}

// config builds the adapter's configuration for a stored provider.
func (s *Service) config(p store.SmsProvider) (sms.Config, error) {
	cfg := sms.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Settings: settingsOf(p), Credentials: map[string]string{}, HTTP: s.HTTP, Now: s.now}
	if len(p.CredentialsEnc) > 0 {
		b, err := s.Box.Open(p.CredentialsEnc, credentialsAD(p.ID))
		if err != nil {
			return cfg, fmt.Errorf("open %s credentials: %w", p.Name, err)
		}
		if err := json.Unmarshal(b, &cfg.Credentials); err != nil {
			return cfg, err
		}
	}
	if len(p.WebhookSecretEnc) > 0 {
		b, err := s.Box.Open(p.WebhookSecretEnc, secretAD(p.ID))
		if err != nil {
			return cfg, fmt.Errorf("open %s webhook secret: %w", p.Name, err)
		}
		cfg.WebhookSecret = string(b)
	}
	if p.Demo && s.Demo != nil {
		cfg.HTTP, cfg.BaseURL = s.Demo(p.Adapter)
	}
	return cfg, nil
}

// open opens a stored provider's adapter.
func (s *Service) open(p store.SmsProvider) (sms.Provider, error) {
	d, ok := sms.Lookup(p.Adapter)
	if !ok {
		return nil, fmt.Errorf("sms adapter %q isn't in this build", p.Adapter)
	}
	cfg, err := s.config(p)
	if err != nil {
		return nil, err
	}
	return d.New(cfg)
}

// hookURL is where a provider sends its webhooks.
func (s *Service) hookURL(p store.SmsProvider) string {
	return strings.TrimRight(s.URLs.Base(), "/") + "/hooks/sms/" + p.Adapter + "/" + p.WebhookToken
}

// ErrNotFound reports a missing row.
var ErrNotFound = errors.New("texting: not found")

func notFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func ptr[T any](v T) *T { return &v }

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
