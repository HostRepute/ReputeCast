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
	"math"
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
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// Features, each with its own switch.
const (
	FeatureSubjectIdeas   = "subject_ideas"
	FeatureRewrite        = "rewrite"
	FeatureTranslate      = "translate"
	FeatureAltText        = "alt_text"
	FeatureAccessibility  = "accessibility_fix"
	FeatureSegment        = "segment_draft"
	FeatureSummary        = "report_summary"
	FeatureBrandVoice     = "brand_voice"
	FeatureDraft          = "draft"
	FeatureAgent          = "agent"
	FeatureConnectionTest = "connection_test"
	// FeatureAssistant is the assistant's threads (B3); it has a default
	// model already.
	FeatureAssistant = "assistant"
)

// Features in the settings' order.
var Features = []string{FeatureSubjectIdeas, FeatureRewrite, FeatureTranslate, FeatureAltText, FeatureAccessibility, FeatureSegment, FeatureSummary, FeatureBrandVoice, FeatureDraft}

func validFeature(f string) bool { return slices.Contains(Features, f) }

// defaultFeatures are the features with a default model.
var defaultFeatures = append(slices.Clone(Features), FeatureAgent, FeatureAssistant)

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
	Jobs  func() *jobs.Client
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
	Segments    *segments.Service
	Reports     *reports.Service
	Languages   *languages.Service
	BrandKit    *brandkit.Service
	Campaigns   *campaigns.Service
	Automations *automations.Service
	Website     *website.Service
	Now         func() time.Time
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

// keyAD is the associated data an account's key is sealed with. The account
// migration 03630 made of a workspace's one connection kept the workspace's
// id, so the key sealed then with "ai:<workspace>:key" opens unchanged.
func keyAD(account uuid.UUID) string { return "ai:" + account.String() + ":key" }

// connection is a model with its account and the workspace's AI settings:
// what a request is bound to (assistant brief D3).
type connection struct {
	store.AIBindingRow
}

// connected reports whether a model is connected (and not disconnected).
func (c *connection) connected() bool { return c != nil && c.DisconnectedAt == nil }

func (c *connection) featureOn(f string) bool {
	if f == FeatureAltText && !c.ImageInput {
		return false
	}
	return !slices.Contains(c.FeaturesOff, f)
}

// providerName is how people call the provider ("Anthropic"): a hosted
// OpenAI-compatible service by its own name ("OpenRouter").
func (c *connection) providerName() string {
	if p, ok := c.preset(); ok {
		return p.Name
	}
	return llm.Info(c.Provider).Name
}

// preset is the hosted service a Compatible connection's address belongs to.
func (c *connection) preset() (llm.Preset, bool) {
	if c.Provider != llm.Compatible {
		return llm.Preset{}, false
	}
	return llm.PresetOf(c.BaseUrl)
}

// modelName is how people call the model ("Claude Sonnet 5.5").
func (c *connection) modelName() string {
	if c.ModelName != "" {
		return c.ModelName
	}
	return llm.ModelName(c.Provider, c.Model)
}

// load returns the workspace's first account's first model, which the
// single-model settings and operations act on; nil when there has never
// been one.
func (s *Service) load(ctx context.Context, q *store.Queries, ws uuid.UUID) (*connection, error) {
	id, err := q.AIPrimaryModel(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.binding(ctx, q, ws, &id)
}

// loadFor returns the model a feature uses (its default), nil when it has
// none.
func (s *Service) loadFor(ctx context.Context, q *store.Queries, ws uuid.UUID, feature string) (*connection, error) {
	id, err := q.AIFeatureModel(ctx, store.AIFeatureModelParams{WorkspaceID: ws, Feature: feature})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.binding(ctx, q, ws, &id)
}

// binding returns a model of the workspace with its account and settings,
// nil when there is no such model.
func (s *Service) binding(ctx context.Context, q *store.Queries, ws uuid.UUID, model *uuid.UUID) (*connection, error) {
	if model == nil {
		return nil, nil
	}
	row, err := q.AIBinding(ctx, store.AIBindingParams{WorkspaceID: ws, ModelID: *model})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &connection{row}, nil
}

// lockSettings takes the AI configuration lock: the workspace's settings
// row, made with the default cap when there is none yet. Every change to
// the settings, accounts, models and defaults takes it first, and so does
// every reservation (before the budget lock).
func lockSettings(ctx context.Context, q *store.Queries, ws uuid.UUID) (store.AiWorkspaceSetting, error) {
	row, err := q.AIWorkspaceSettingsLock(ctx, ws)
	if !errors.Is(err, pgx.ErrNoRows) {
		return row, err
	}
	if err := q.AIWorkspaceSettingsEnsure(ctx, ws); err != nil {
		return row, err
	}
	return q.AIWorkspaceSettingsLock(ctx, ws)
}

// current re-reads a request's model under the configuration lock, for a
// result's commit: nil when the model is gone.
func (s *Service) current(ctx context.Context, q *store.Queries, ws uuid.UUID, c *connection) (*connection, error) {
	if _, err := lockSettings(ctx, q, ws); err != nil {
		return nil, err
	}
	return s.binding(ctx, q, ws, &c.ModelID)
}

// policy is the SSRF guard for model servers.
func (s *Service) policy() netguard.Policy {
	return netguard.Policy{AllowPrivate: s.AllowPrivate != nil && s.AllowPrivate()}
}

// client returns the HTTP client that reaches providers. Its own limit is the
// longest feature's; each request's context sets its feature's timeout.
func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return s.policy().Client(longestTimeout)
}

// longestTimeout bounds a draft or an agent step: up to 8,000 output tokens
// of MJML take minutes at typical provider speeds.
const longestTimeout = 3 * time.Minute

// timeouts per feature.
func timeoutOf(feature string) time.Duration {
	switch feature {
	case FeatureAgent, FeatureDraft:
		return longestTimeout
	case FeatureTranslate:
		return 120 * time.Second
	case FeatureSummary, FeatureSegment, FeatureSubjectIdeas:
		return 60 * time.Second
	}
	return 45 * time.Second
}

// configOf is how to reach a connection.
func (s *Service) configOf(c *connection, key, feature string) llm.Config {
	// Paid retries must have their own reservation and checkpoint. The
	// transport cannot infer that a missing response was never billed.
	retries := 0
	return llm.Config{
		Kind: c.Provider, BaseURL: c.BaseUrl, APIKey: key, Model: c.Model, Deployment: c.Deployment, APIVersion: c.ApiVersion,
		HTTP: s.client(), Timeout: timeoutOf(feature), Retries: &retries, RetryWait: s.RetryWait,
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
		k, err := s.Box.OpenString(c.KeySealed, keyAD(c.AccountID))
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
	return int64(math.Ceil(float64(in)*c.InputPrice + float64(out)*c.OutputPrice))
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
