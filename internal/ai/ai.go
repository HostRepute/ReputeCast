// Package ai is the AI assistant with the workspace's own model (design
// expansion, phase 3, M22): the connection in Settings › Integrations' "AI
// models" group (one model per workspace, a write-only key, the features
// switched on, a monthly cap and the prices it is estimated with), the
// suggestions (subject ideas, rewrites, translations of a language version,
// alt text, accessibility fixes, segment drafts and report summaries), the
// AI activity log and the usage meter.
//
// What leaves the instance is the content a person is working on, when they
// ask, and only for the features switched on. Prompts are templates in this
// package with the person's content delimited; contact data is never put in
// them: personalization tokens go as written, a preview "as a contact" is
// never sent, segment drafts get the condition schema without any contact's
// values, and report summaries get the report's aggregate numbers.
//
// Every suggestion is an answer to accept or reject: nothing here changes an
// email, a segment or a report by itself. Accepted text goes through the
// composer's own saves (the version history); a kept summary and an
// accepted translation are written by the operations that say so. Each
// request is in the AI activity log (90 days, never the content); the
// configuration's changes are audited.
package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Features, each with its own switch.
const (
	FeatureSubjectIdeas  = "subject_ideas"
	FeatureRewrite       = "rewrite"
	FeatureTranslate     = "translate"
	FeatureAltText       = "alt_text"
	FeatureAccessibility = "accessibility_fix"
	FeatureSegment       = "segment_draft"
	FeatureSummary       = "report_summary"
)

// Features in the settings' order.
var Features = []string{FeatureSubjectIdeas, FeatureRewrite, FeatureTranslate, FeatureAltText, FeatureAccessibility, FeatureSegment, FeatureSummary}

func validFeature(f string) bool { return slices.Contains(Features, f) }

// Outcomes of a request in the activity log.
const (
	OutcomePending        = "pending"
	OutcomeAccepted       = "accepted"
	OutcomeAcceptedEdits  = "accepted_edits"
	OutcomeKept           = "kept"
	OutcomeRejected       = "rejected"
	OutcomeDiscarded      = "discarded"
	OutcomeRefused        = "refused"
	OutcomeError          = "error"
	OutcomeNothingDrafted = "nothing_drafted"
	OutcomeCancelled      = "cancelled"
)

// Retention of the activity log.
const Retention = 90 * 24 * time.Hour

// NoticeShare is when the usage notice goes out (80% of the cap).
const NoticeShare = 0.8

// Service is the AI assistant.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	// AllowPrivate reports whether model servers may resolve to private
	// addresses (providers.allow_private_networks: the setting SMTP
	// servers and community adapters use).
	AllowPrivate func() bool
	// Development lets the demo's sample connection answer from its
	// recordings; elsewhere it refuses.
	Development bool
	// HTTP overrides the client that calls providers (tests).
	HTTP *http.Client
	// Provider overrides how a connection is reached (tests).
	Provider func(cfg llm.Config) (llm.Provider, error)
	// Retries and RetryWait override how an overloaded provider is retried.
	Retries   *int
	RetryWait time.Duration
	// Segments, Reports and Languages are the domains the drafts, the
	// summaries and the translations read and write (wired in internal/app).
	Segments  *segments.Service
	Reports   *reports.Service
	Languages *languages.Service
	Now       func() time.Time
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

// keyAD is the associated data the key is sealed with.
func keyAD(ws uuid.UUID) string { return "ai:" + ws.String() + ":key" }

// connection is the workspace's model, when one is connected.
type connection struct {
	store.AiConnection
}

// connected reports whether a model is connected (and not disconnected).
func (c *connection) connected() bool { return c != nil && c.DisconnectedAt == nil }

func (c *connection) featureOn(f string) bool {
	if f == FeatureAltText && !c.ImageInput {
		return false
	}
	return !slices.Contains(c.FeaturesOff, f)
}

// providerName is how people call the provider ("Anthropic").
func (c *connection) providerName() string { return llm.Info(c.Provider).Name }

// modelName is how people call the model ("Claude Sonnet 5.5").
func (c *connection) modelName() string {
	if c.ModelName != "" {
		return c.ModelName
	}
	return llm.ModelName(c.Provider, c.Model)
}

// load returns the workspace's connection row, nil when there has never
// been one.
func (s *Service) load(ctx context.Context, q *store.Queries, ws uuid.UUID) (*connection, error) {
	row, err := q.AIConnection(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &connection{row}, nil
}

// policy is the SSRF guard for model servers.
func (s *Service) policy() netguard.Policy {
	return netguard.Policy{AllowPrivate: s.AllowPrivate != nil && s.AllowPrivate()}
}

// client returns the HTTP client that reaches providers.
func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return s.policy().Client(llm.DefaultTimeout)
}

// timeouts per feature: a translation of a whole email takes longest.
func timeoutOf(feature string) time.Duration {
	switch feature {
	case FeatureTranslate:
		return 120 * time.Second
	case FeatureSummary, FeatureSegment, FeatureSubjectIdeas:
		return 60 * time.Second
	}
	return 45 * time.Second
}

// configOf is how to reach a connection.
func (s *Service) configOf(c *connection, key, feature string) llm.Config {
	return llm.Config{
		Kind: c.Provider, BaseURL: c.BaseUrl, APIKey: key, Model: c.Model, Deployment: c.Deployment, APIVersion: c.ApiVersion,
		HTTP: s.client(), Timeout: timeoutOf(feature), Retries: s.Retries, RetryWait: s.RetryWait,
	}
}

// provider builds the client of a connection: the demo's recordings for the
// sample connection, the provider's API otherwise.
func (s *Service) provider(c *connection, feature string) (llm.Provider, error) {
	if c.Demo {
		if !s.Development {
			return nil, errDemoElsewhere
		}
		return demoModel{}, nil
	}
	key := ""
	if len(c.KeySealed) > 0 {
		k, err := s.Box.OpenString(c.KeySealed, keyAD(c.WorkspaceID))
		if err != nil {
			return nil, fmt.Errorf("open the AI key: %w", err)
		}
		key = k
	}
	cfg := s.configOf(c, key, feature)
	if s.Provider != nil {
		return s.Provider(cfg)
	}
	return llm.New(cfg)
}

// helpers are the members who can connect or change the model.
func (s *Service) helpers(ctx context.Context, ws uuid.UUID) []string {
	return access.Helpers(ctx, s.Q, ws, authz.IntegrationsManage)
}

// month returns the start of the current month and of the next one, in
// the workspace's timezone: the cap resets on the 1st.
func (s *Service) month(ws *store.Workspace) (time.Time, time.Time) {
	loc := time.UTC
	if ws != nil && ws.Timezone != "" {
		if l, err := time.LoadLocation(ws.Timezone); err == nil {
			loc = l
		}
	}
	now := s.now().In(loc)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 1, 0)
}

// monthKey names the current month ("2026-10").
func monthKey(start time.Time) string { return start.Format("2006-01") }

// costMicros is the estimated cost of a request in millionths of a
// dollar: prices are dollars per million tokens.
func costMicros(c *connection, in, out int) int64 {
	return int64(float64(in)*c.InputPrice + float64(out)*c.OutputPrice + 0.5)
}

// actorID returns the person's id, nil for others.
func actorID(p *authz.Principal) *uuid.UUID {
	if p == nil || !p.IsHuman() {
		return nil
	}
	id := p.ID
	return &id
}

// clip shortens s to n runes.
func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n-1]) + "…"
}
