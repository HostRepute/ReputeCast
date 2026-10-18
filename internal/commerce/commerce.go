// Package commerce is Shopify and WooCommerce in ReputeCast (design
// expansion phase 7, M30): connecting a merchant's own store through the
// integrations hub, syncing its products, the customers who accepted
// marketing, its orders and abandoned checkouts (a first import of 3, 6 or
// 12 months, then webhooks and a nightly reconciliation), consent (only
// customers who accepted marketing ever join a list; unsubscribes travel
// both ways), store triggers and blocks in emails, and attributing orders
// to the email whose link the buyer clicked last.
//
// The store platforms are reached through internal/stores only. Webhooks
// are verified, stored raw and answered at once; a River job applies them.
// Never stored: addresses, phone numbers, payment details and order notes.
// Orders and store webhooks aren't audited (ROADMAP default): the orders,
// the store's activity log and its raw events are their record; everything
// a person changes is.
package commerce

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// Platforms.
const (
	Shopify     = "shopify"
	WooCommerce = "woocommerce"
)

// Statuses and steps of a store (stores.status, stores.step).
const (
	StatusConnecting   = "connecting"
	StatusImporting    = "importing"
	StatusConnected    = "connected"
	StatusDisconnected = "disconnected"

	StepAddress  = "address"
	StepApp      = "app"
	StepKeys     = "keys"
	StepInstall  = "install"
	StepSync     = "sync"
	StepKey      = "key"
	StepWebhooks = "webhooks"
	StepPlugin   = "plugin"
	StepDone     = "done"
)

// Timings.
const (
	// HealthEvery is how often webhooks are checked, and a missing one's
	// data polled until it's registered again.
	HealthEvery = 15 * time.Minute
	// Kept is how long raw events and the activity log are kept.
	Kept = 30 * 24 * time.Hour
	// stateTTL is how long Shopify's approval page may take.
	stateTTL = 30 * time.Minute
	// NightlyHour is the reconciliation's hour, in the workspace's timezone.
	NightlyHour = 3
)

// Service is the commerce domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Links *linktoken.Signer
	URLs  publicurl.Builder
	Log   *slog.Logger
	Jobs  func() *jobs.Client
	Now   func() time.Time
	// BaseURL and HTTP send every store call to one address (tests: a
	// fake's server).
	BaseURL string
	HTTP    *http.Client
	// Demo answers the demo's sample stores from in-process fakes, and
	// DemoApprove approves their app at once (the redirect Shopify would
	// send).
	Demo        func(platform string) *http.Client
	DemoApprove func(redirect, state string) string
	// AllowPrivate lets store addresses resolve to private networks
	// (development with a local store).
	AllowPrivate func() bool
	// Kick wakes the automations engine after store events were queued.
	Kick func(ctx context.Context)
	// Sites finds the WordPress site connected for a host (the
	// WooCommerce plugin check).
	Sites func(ctx context.Context, wsID uuid.UUID, host string) (PluginSite, bool)
	// PublicCheck reports whether the instance's public URL is reachable
	// from the internet, and when it was checked (the connect dialog).
	PublicCheck func(ctx context.Context) (bool, time.Time)

	domains sync.Map // workspace → domainSet (the link decorator's cache)
	names   sync.Map // campaign or automation id → utm_campaign slug
}

// PluginSite is a WordPress site running the ReputeCast plugin.
type PluginSite struct {
	ID            uuid.UUID
	Host          string
	PluginVersion string
	WooCommerce   string
	Consent       bool
	Checkouts     bool
	LastReportAt  *time.Time
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

func (s *Service) jobs() *jobs.Client {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) base() string { return strings.TrimRight(s.URLs.Base(), "/") }

// AppURL and CallbackURL are what the merchant pastes into their Shopify
// app; WebhookURL is where a store's webhooks post.
func (s *Service) AppURL() string      { return s.base() + "/integrations/shopify/app" }
func (s *Service) CallbackURL() string { return s.base() + "/integrations/shopify/callback" }

// WebhookURL is a store's webhook address.
func (s *Service) WebhookURL(st store.Store) string {
	tok := ""
	if st.WebhookToken != nil {
		tok = *st.WebhookToken
	}
	return s.base() + "/hooks/" + st.Platform + "/" + tok
}

// credentials are a store's secrets, sealed together (write-only).
type credentials struct {
	ClientSecret string     `json:"clientSecret,omitempty"`
	AccessToken  string     `json:"accessToken,omitempty"`
	RefreshToken string     `json:"refreshToken,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	Key          string     `json:"key,omitempty"`
	Secret       string     `json:"secret,omitempty"`
}

func credentialsAD(id uuid.UUID) string { return "store:" + id.String() + ":credentials" }
func webhookAD(id uuid.UUID) string     { return "store:" + id.String() + ":webhook_secret" }

func (s *Service) credentials(st store.Store) (credentials, error) {
	var c credentials
	if len(st.CredentialsSealed) == 0 {
		return c, errNoCredentials
	}
	raw, err := s.Box.Open(st.CredentialsSealed, credentialsAD(st.ID))
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(raw, &c)
	return c, err
}

func (s *Service) seal(id uuid.UUID, c credentials) ([]byte, error) {
	raw, err := json.Marshal(c) //nolint:gosec // G117: sealed on the next line, never stored or returned in clear
	if err != nil {
		return nil, err
	}
	return s.Box.Seal(raw, credentialsAD(id))
}

var errNoCredentials = errors.New("commerce: the store has no credentials")

// webhookSecret is what a store's deliveries are signed with: Shopify's
// app client secret, or the secret WooCommerce's webhooks were registered
// with.
func (s *Service) webhookSecret(st store.Store) (string, error) {
	if st.Platform == Shopify {
		c, err := s.credentials(st)
		if err != nil {
			return "", err
		}
		return c.ClientSecret, nil
	}
	if len(st.WebhookSecretSealed) == 0 {
		return "", errNoCredentials
	}
	return s.Box.OpenString(st.WebhookSecretSealed, webhookAD(st.ID))
}

// config is how to reach a store's platform: the demo's fake, the tests'
// fake, or the store itself through the SSRF guard.
func (s *Service) config(platform, domain string, demo bool) stores.Config {
	cfg := stores.Config{Domain: domain, Now: s.Now}
	switch {
	case demo && s.Demo != nil:
		cfg.HTTP, cfg.BaseURL = s.Demo(platform), "https://"+domain
	case s.BaseURL != "":
		cfg.BaseURL, cfg.HTTP = s.BaseURL, s.HTTP
	default:
		allow := s.AllowPrivate != nil && s.AllowPrivate()
		cfg.HTTP = netguard.Policy{AllowPrivate: allow}.Client(30 * time.Second)
	}
	return cfg
}

// client opens a store's platform client with its credentials.
func (s *Service) client(st store.Store) (stores.Client, error) {
	d, ok := stores.Lookup(st.Platform)
	if !ok {
		return nil, errors.New("commerce: unknown platform " + st.Platform)
	}
	c, err := s.credentials(st)
	if err != nil {
		return nil, err
	}
	cfg := s.config(st.Platform, st.Domain, st.Demo)
	cfg.Token, cfg.Key, cfg.Secret = c.AccessToken, c.Key, c.Secret
	if st.Platform == Shopify && c.RefreshToken != "" && c.ExpiresAt != nil && s.now().After(c.ExpiresAt.Add(-5*time.Minute)) {
		if t, err := shopify.Refresh(context.Background(), cfg, st.ClientID, c.ClientSecret, c.RefreshToken); err == nil {
			c.AccessToken, c.RefreshToken, c.ExpiresAt = t.AccessToken, t.RefreshToken, t.ExpiresAt
			if sealed, err := s.seal(st.ID, c); err == nil {
				_ = s.Q.CommerceStoreSetCredentials(context.Background(), store.CommerceStoreSetCredentialsParams{ID: st.ID, CredentialsSealed: sealed, At: s.now()})
			}
			cfg.Token = c.AccessToken
		}
	}
	return d.Open(cfg)
}

// descriptor is a platform's registry entry.
func descriptor(platform string) stores.Descriptor {
	d, _ := stores.Lookup(platform)
	return d
}

// platformName is how people call a platform.
func platformName(platform string) string {
	if platform == WooCommerce {
		return "WooCommerce"
	}
	return "Shopify"
}

// helpers names who can help with perm, Admins first ("Ask Marcus Chen or
// Linh Nguyen").
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID, perm authz.Permission) []string {
	out := []string{}
	rows, err := s.Q.CommerceMembers(ctx, wsID)
	if err != nil {
		return out
	}
	for _, r := range rows {
		if !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			continue
		}
		if name := strings.TrimSpace(r.Name); name != "" {
			out = append(out, name)
		}
		if len(out) == 2 {
			break
		}
	}
	return out
}

// userName is a member's name, "" when unknown.
func (s *Service) userName(ctx context.Context, id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	n, err := s.Q.CommerceUserName(ctx, *id)
	if err != nil {
		return ""
	}
	return n
}

// userID is the person behind an action (a key's creator for keys).
func userID(p *authz.Principal) *uuid.UUID {
	if p == nil {
		return nil
	}
	id := p.UserID
	if id == uuid.Nil && p.Type == authz.ActorUser {
		id = p.ID
	}
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func ptr[T any](v T) *T { return &v }

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
