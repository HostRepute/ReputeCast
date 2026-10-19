// Package demo loads the sample data from the design ("Acme Newsletter") so
// the app can be explored, screenshotted and compared with the prototype.
//
// Seeding is split per domain: each domain fills its own seedX function in
// its own file (seed_<domain>.go). Seeders run in the order of the list in
// Seed and share IDs through *Seeder, so later seeders can reference users,
// lists and providers created earlier. Times are relative to now so "2 min
// ago" in the design reads the same in the demo.
package demo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Password is shared by every demo account that signs in with a password.
const Password = "reputecast-demo" //nolint:gosec // public demo password, documented in the README

// TOTPSecret is Linh Nguyen's authenticator secret in the demo, so the
// two-factor screens can be tried. Generate codes with any TOTP app.
const TOTPSecret = "JBSWY3DPEHPK3PXP" //nolint:gosec // public demo secret, documented in the README

// Result describes what was seeded.
type Result struct {
	OwnerEmail string
	Password   string
	TOTPSecret string
}

// Seeder carries shared state between domain seeders.
type Seeder struct {
	Tx  pgx.Tx
	Q   *store.Queries
	Box *secrets.Box
	Cfg *config.Config
	Now time.Time
	Log *slog.Logger

	// Instance and people
	Users map[string]store.User // by email
	// Workspaces by slug ("acme-newsletter", "acme-engineering-blog")
	Workspaces map[string]store.Workspace
	// Roles by workspace slug + key ("acme-newsletter/content-writer")
	Roles map[string]store.Role
	// Lists by name ("Weekly Digest")
	Lists map[string]uuid.UUID
	// Providers by short name ("ses", "postmark", "postal")
	Providers map[string]uuid.UUID
	// Campaigns by name
	Campaigns map[string]uuid.UUID
	// Senders by email
	Senders map[string]uuid.UUID
	// Domains by name
	Domains map[string]uuid.UUID
	// Extra is free-form storage for domain seeders.
	Extra map[string]any

	after []func(context.Context, *Seeder) error
}

// After runs fn once every seeder has run, for rows that need IDs from a
// seeder that comes later in the order (for example per-provider domain
// verification, seeded with the senders but needing the providers).
func (s *Seeder) After(fn func(context.Context, *Seeder) error) { s.after = append(s.after, fn) }

// seedClock is the demo's "now"; tests seed other days with it.
var seedClock = time.Now

// ID returns a fresh UUIDv7.
func ID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Ago returns now minus d.
func (s *Seeder) Ago(d time.Duration) time.Time { return s.Now.Add(-d) }

// WS returns the main demo workspace (Acme Newsletter).
func (s *Seeder) WS() store.Workspace { return s.Workspaces["acme-newsletter"] }

// Seed loads the demo data in one transaction. It is idempotent: when the
// demo owner already exists nothing happens.
func Seed(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, box *secrets.Box, log *slog.Logger) (*Result, error) {
	res := &Result{OwnerEmail: "linh@acme.io", Password: Password, TOTPSecret: TOTPSecret}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = 'linh@acme.io')`).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		log.Info("demo data already present")
		return res, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	s := &Seeder{
		Tx: tx, Q: store.New(tx), Box: box, Cfg: cfg, Now: seedClock(), Log: log,
		Users: map[string]store.User{}, Workspaces: map[string]store.Workspace{}, Roles: map[string]store.Role{},
		Lists: map[string]uuid.UUID{}, Providers: map[string]uuid.UUID{}, Campaigns: map[string]uuid.UUID{},
		Senders: map[string]uuid.UUID{}, Domains: map[string]uuid.UUID{}, Extra: map[string]any{},
	}
	// The order matters: later seeders use IDs from earlier ones.
	steps := []struct {
		name string
		fn   func(context.Context, *Seeder) error
	}{
		{"core", seedCore},
		// Sending domains and senders come first: team settings (default
		// sender), the composer and deliverability all point at them.
		{"senders", seedSenders},
		{"team", seedTeam},
		{"account", seedAccount},
		{"instance", seedInstance},
		{"audience", seedAudience},
		{"providers", seedProviders},
		{"deliverability", seedDeliverability},
		{"campaigns", seedCampaigns},
		{"composer", seedComposer},
		{"readiness", seedReadiness},
		{"sending", seedSending},
		{"reports", seedReports},
		{"automations", seedAutomations},
		{"forms", seedForms},
		{"integrations", seedIntegrations},
		{"notifications", seedNotifications},
		{"audit", seedAudit},
		{"transactional", seedTransactional},
		{"integrations-hub", seedIntegrationsHub},
		{"transactional templates", seedTransactionalTemplates},
		{"accessibility", seedAccessibility},
		{"languages", seedLanguages},
		{"app_languages", seedAppLanguages},
		{"page languages", seedPageLanguages},
		{"scoring and growth", seedScoringGrowth},
		{"digests", seedDigests},
		{"polls", seedPolls},
		{"earn", seedEarn},
		{"verification_placement", seedVerificationPlacement},
		{"ai", seedAI},
		{"website", seedWebsite},
		{"stores", seedStores},
	}
	for _, st := range steps {
		if err := st.fn(ctx, s); err != nil {
			return nil, fmt.Errorf("seed %s: %w", st.name, err)
		}
	}
	for i, fn := range s.after {
		if err := fn(ctx, s); err != nil {
			return nil, fmt.Errorf("seed deferred step %d: %w", i+1, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}
