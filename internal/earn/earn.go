// Package earn is ReputeCast's payments and paid subscriptions (design
// expansion phase 6, M26): a workspace connects its own payment account
// (Stripe, with a restricted key), defines tiers with monthly and yearly
// prices, sells them on a hosted upgrade page through the provider's
// checkout, and keeps every contact's paid status in step with the
// provider's events. Readers pay the publisher; ReputeCast takes no fee and
// gates nothing.
//
// The provider holds the money and ReputeCast mirrors it: webhook events
// are verified, stored raw and applied by a River job (the payments event
// log), an hourly check reads any event the webhook missed, and nothing
// about cards is ever stored. Paid members can be imported from Substack,
// Ghost, beehiiv and Kit exports, given complimentary access, and emailed
// through eight member emails, which are transactional templates in an
// Essential "Memberships" category.
//
// Every change a person makes is audited; payment events and readers' own
// changes stay in the payments event log and the paid members' timelines,
// as the ROADMAP's defaults say.
package earn

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

// msgs holds the server's own text: problem details and the public
// pages' built-in words. English only until the pages are translated.
var msgs = i18n.MustLoad(messagesFS, "earn")

// Link token kinds of the Earn pages (numbered after the workstream so
// parallel kinds never collide): a reader's personal upgrade link, the
// link that opens their billing portal, and their cancel page.
const (
	TokenUpgrade linktoken.Kind = 160
	TokenBilling linktoken.Kind = 161
	TokenCancel  linktoken.Kind = 162
)

// Mailer sends the member emails through transactional email and keeps
// their templates (internal/transactional, wired in internal/app).
type Mailer interface {
	// Category returns the Essential "Memberships" category, creating it.
	Category(ctx context.Context, ws *store.Workspace) (uuid.UUID, error)
	// CreateDraft creates a template draft from a member email's starter.
	CreateDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, d TemplateDraft) (uuid.UUID, error)
	// Templates reads the templates' keys and versions.
	Templates(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) (map[uuid.UUID]TemplateState, error)
	// Send sends a template's live version; ErrNotPublished when it has none.
	Send(ctx context.Context, ws *store.Workspace, m MemberMessage) (Sent, error)
	// Route names the Transactional route ("" when there is none).
	Route(ctx context.Context, ws *store.Workspace) (Route, error)
}

// ErrNotPublished means a member email's template has no published version.
var ErrNotPublished = errors.New("earn: the template has no published version")

// TemplateDraft is a member email's starter.
type TemplateDraft struct {
	Key, Name, Description string
	Category               uuid.UUID
	Subject, Preheader     string
	MJML                   string
	Variables              []TemplateVariable
	TestData               map[string]any
}

// TemplateVariable is a member email's variable.
type TemplateVariable struct {
	Name, Type, Sample, Default string
	Required, Sensitive         bool
}

// TemplateState is a template's key and versions.
type TemplateState struct {
	Key  string
	Name string
	// LiveVersion is 0 when nothing was ever published.
	LiveVersion int
	// DraftVersion is the version publishing the draft would make, when
	// the draft has unpublished changes.
	DraftVersion int
	Deleted      bool
	FromName     string
	FromEmail    string
	ReplyTo      string
	CategoryName string
	CategoryKind string
}

// MemberMessage is one member email to send.
type MemberMessage struct {
	Template string
	To       string
	ToName   string
	Data     map[string]any
	// Key makes a retry send nothing twice.
	Key string
}

// Sent is what sending a member email did.
type Sent struct {
	MessageID uuid.UUID
	Status    string
	Provider  string
}

// Route is the Transactional route member emails go through.
type Route struct {
	Primary  string
	Fallback string
}

// Confirmations sends the double opt-in email of a contact whose lists are
// pending (complimentary access for a new address), through forms.
type Confirmations interface {
	Send(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) error
}

// Signup adds a contact to lists with consent (internal/audience).
type Signup interface {
	// Join subscribes email to lists with the evidence; double keeps them
	// pending until the contact confirms. It returns the contact.
	Join(ctx context.Context, tx pgx.Tx, ws *store.Workspace, j Join) (uuid.UUID, bool, error)
}

// Join is a subscription request from a paid flow.
type Join struct {
	Email, FirstName, LastName string
	Lists                      []uuid.UUID
	Double                     bool
	// Source and Note are the consent evidence ("paid_subscription").
	Source, Note string
	At           time.Time
}

// Service is the Earn domain.
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
	// Mailer sends member emails (nil: none are sent; they are logged as
	// not sent).
	Mailer Mailer
	// Signup joins lists with consent; Confirmations sends double opt-in.
	Signup        Signup
	Confirmations Confirmations
	// PublicPages names the upgrade page's address on the workspace's own
	// domain (news.acme.io/upgrade), when it has one.
	PageBase func(ctx context.Context, ws *store.Workspace) string
	// StripeBase and StripeHTTP point Stripe at a fake (tests).
	StripeBase string
	StripeHTTP *http.Client
	// Demo is the in-process Stripe the demo's sample connections use.
	Demo func() *http.Client
	// AllowPrivate lets webhooks use a private public URL (development,
	// tests): providers.allow_private_networks.
	AllowPrivate bool
	// Reach overrides the public URL's reachability check (tests).
	Reach func(ctx context.Context, url, token string) Reach
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

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// open connects a provider client for a connection with its sealed key.
func (s *Service) open(c store.PaymentConnection) (payments.Client, error) {
	if len(c.KeySealed) == 0 {
		return nil, errNoKey
	}
	key, err := s.Box.Open(c.KeySealed, keyAD(c.ID))
	if err != nil {
		return nil, err
	}
	return s.openKey(c.Provider, string(key), c.Demo)
}

var errNoKey = errors.New("earn: the connection has no key")

// openKey connects with a key in hand (the wizard's check).
func (s *Service) openKey(provider, key string, demo bool) (payments.Client, error) {
	d, ok := payments.Lookup(provider)
	if !ok || d.Open == nil {
		return nil, errors.New("earn: payment provider not available")
	}
	cfg := payments.Config{Key: key, BaseURL: s.StripeBase, HTTP: s.StripeHTTP, Now: s.Now}
	if demo && s.Demo != nil {
		cfg.BaseURL, cfg.HTTP = "https://api.stripe.com", s.Demo()
	}
	return d.Open(cfg)
}

func keyAD(id uuid.UUID) string    { return "payment_connection:" + id.String() + ":key" }
func secretAD(id uuid.UUID) string { return "payment_connection:" + id.String() + ":webhook_secret" }

// helpers names who can help with perm, Admins first ("Ask Marcus Chen or
// Linh Nguyen", as the read-only lines say).
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID, perm authz.Permission, exclude uuid.UUID) []string {
	out := []string{}
	rows, err := s.Q.EarnTeam(ctx, wsID)
	if err != nil {
		return out
	}
	for _, r := range rows {
		if r.UserID == exclude || !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = r.Email
		}
		out = append(out, name)
		if len(out) == 2 {
			break
		}
	}
	return out
}

// actorName is the signed-in person's name, for timelines.
func actorName(p *authz.Principal) string {
	if p == nil {
		return ""
	}
	return p.Name
}

func userID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.Type != authz.ActorUser || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// lang is the reader's language for server text.
func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// problem is an RFC 9457 answer with its text from the catalog.
func problem(ctx context.Context, status int, code, key string, args i18n.Args) *api.Problem {
	return api.Err(status, code, msgs.M(key).Text(lang(ctx), args))
}

// invalid is a 422 about one field.
func invalid(ctx context.Context, location, key string, args i18n.Args) *api.Problem {
	return api.Invalid(location, msgs.M(key).Text(lang(ctx), args))
}
