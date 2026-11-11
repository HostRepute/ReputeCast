// Package leads is Facebook and Instagram Lead Ads in ReputeCast (M66,
// G-41): a workspace connects its own Meta app with a system user's token,
// chooses the Pages whose lead forms it collects, and maps each form to
// contact fields, lists, tags and a consent mode. Meta posts each new lead's
// ids to /hooks/meta-leads/{token}; the delivery is verified, stored as it
// came and answered at once, and a River job fetches the lead with the
// Page's token and signs the person up, once per lead, with the form's
// consent as evidence. When the webhook is quiet (or the instance can't be
// reached from the internet), active forms are polled every 15 minutes; a
// refused token turns the connection to Needs attention and tells the
// people who can manage integrations.
//
// Meta is reached through internal/leads/meta only. Never stored: a lead's
// answers (they go to the contact, or are dropped) and the credentials in
// clear. Leads aren't audited (their consent records are the evidence);
// everything a person changes is.
package leads

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/leads/meta"
	"github.com/hostrepute/reputecast/internal/leads/meta/metatest"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Platform is the only lead ads platform (lead_connections.platform).
const Platform = "meta"

// Statuses of a connection.
const (
	StatusConnected    = "connected"
	StatusAttention    = "attention"
	StatusDisconnected = "disconnected"
)

// Problems of a connection that needs attention (status_detail).
const (
	ProblemTokenRejected     = "token_rejected"
	ProblemTokenExpired      = "token_expired"
	ProblemAppRejected       = "app_rejected"
	ProblemPermissionMissing = "permission_missing"
)

// Why leads are polled (polling_reason).
const (
	PollingNotSubscribed = "not_subscribed"
	PollingQuiet         = "quiet"
	PollingUnreachable   = "unreachable"
)

// Outcomes of a lead (lead_entries.outcome).
const (
	OutcomeCreated  = "created"
	OutcomeUpdated  = "updated"
	OutcomePending  = "pending"
	OutcomeBlocked  = "blocked"
	OutcomeInvalid  = "invalid"
	OutcomeDeclined = "declined"
	OutcomeTest     = "test"
)

// Where a lead came from (lead_entries.source).
const (
	SourceWebhook = "webhook"
	SourcePoll    = "poll"
	SourceImport  = "import"
)

// ConsentSourcePrefix starts a lead's consent source: "lead_ad:<form id>".
const ConsentSourcePrefix = "lead_ad:"

// ConsentSource is a lead form's consent source.
func ConsentSource(formID uuid.UUID) string { return ConsentSourcePrefix + formID.String() }

// Timings.
const (
	// HealthEvery is how often connections are checked, their forms
	// refreshed and, when the webhook is quiet, polled.
	HealthEvery = 15 * time.Minute
	// QuietAfter is how long without a delivery, while forms are active,
	// before leads are polled.
	QuietAfter = 24 * time.Hour
	// Kept is how long Meta's deliveries are kept.
	Kept = 30 * 24 * time.Hour
	// ImportWindow is how far back Import past leads goes.
	ImportWindow = 90 * 24 * time.Hour
	// RateLimitWait is how long Meta's rate limits pause a connection's calls.
	RateLimitWait = 15 * time.Minute
)

// MaxBody is the largest delivery accepted; MaxRejected how much of a
// refused one is kept.
const (
	MaxBody     = 1 << 20
	MaxRejected = 2048
)

// Service is the lead ads domain.
type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Audit    *audit.Logger
	Box      *secrets.Box
	Log      *slog.Logger
	URLs     publicurl.Builder
	Audience *audience.Service
	Jobs     func() *jobs.Client
	// HTTP reaches Meta's Graph API through the SSRF guard (the in-process
	// Graph API in tests).
	HTTP *http.Client
	// Demo is the in-process Graph API the demo's connection answers from.
	Demo func() *metatest.Fake
	// Kick wakes the automations engine after a lead queued an event.
	Kick func(ctx context.Context)
	// Reachable says whether Meta can reach the instance's public URL,
	// and the URL.
	Reachable func(ctx context.Context) (url string, ok bool)
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

func (s *Service) jobs() *jobs.Client {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

func (s *Service) kick(ctx context.Context) {
	if s.Kick != nil {
		s.Kick(ctx)
	}
}

func (s *Service) reachable(ctx context.Context) (string, bool) {
	if s.Reachable == nil {
		return s.URLs.Base(), true
	}
	return s.Reachable(ctx)
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// The associated data of each sealed value names its row and field.
func credAD(id uuid.UUID) string     { return "lead_connection:" + id.String() + ":credentials" }
func callbackAD(id uuid.UUID) string { return "lead_connection:" + id.String() + ":callback_token" }
func verifyAD(id uuid.UUID) string   { return "lead_connection:" + id.String() + ":verify_token" }
func pageAD(id uuid.UUID) string     { return "lead_page:" + id.String() + ":token" }

// credentials opens a connection's sealed credentials.
func (s *Service) credentials(c store.LeadConnection) (meta.Credentials, error) {
	var out meta.Credentials
	if len(c.CredentialsSealed) == 0 {
		return out, errNoCredentials
	}
	raw, err := s.Box.Open(c.CredentialsSealed, credAD(c.ID))
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(raw, &out)
}

// client is a connection's Graph API client: the in-process one for the
// demo's.
func (s *Service) client(c store.LeadConnection) (*meta.Client, error) {
	creds, err := s.credentials(c)
	if err != nil {
		return nil, err
	}
	return s.clientFor(creds, c.Demo), nil
}

func (s *Service) clientFor(creds meta.Credentials, demo bool) *meta.Client {
	cl := &meta.Client{Creds: creds, HTTP: s.HTTP}
	if demo && s.Demo != nil {
		cl.HTTP = s.Demo().Client()
	}
	if cl.HTTP == nil {
		cl.HTTP = &http.Client{Timeout: 20 * time.Second}
	}
	return cl
}

// pageOf is a stored Page with its opened token.
func (s *Service) pageOf(id uuid.UUID, metaID string, sealed []byte) (meta.Page, error) {
	if len(sealed) == 0 {
		return meta.Page{}, errNoCredentials
	}
	token, err := s.Box.OpenString(sealed, pageAD(id))
	return meta.Page{ID: metaID, AccessToken: token}, err
}

// callbackURL is the address Meta posts a connection's leads to. It carries
// the connection's token: only people who can manage integrations see it,
// and it's never logged.
func (s *Service) callbackURL(c store.LeadConnection) (string, error) {
	token, err := s.Box.OpenString(c.CallbackTokenSealed, callbackAD(c.ID))
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s.URLs.Base(), "/") + "/hooks/meta-leads/" + token, nil
}

// randomToken is a verify token for Meta's handshake: "rc-leads-" and 24
// letters and digits.
func randomToken() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return "rc-leads-" + string(b)
}

// consentVersion names a consent text by its hash ("meta:3f9a1c0b2e4d").
func consentVersion(text string) string {
	if text == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(text))
	return "meta:" + hex.EncodeToString(sum[:6])
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8Start(s[n]) {
		n--
	}
	return s[:n]
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "leads")
