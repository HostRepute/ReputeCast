// Package whatsapp is ReputeCast's WhatsApp channel (design expansion phase
// 8, M32, optional): Meta's WhatsApp Cloud API, the only WhatsApp provider
// in this release, reached over plain HTTP (internal/whatsapp/meta).
//
//   - The account: a WhatsApp Business Account, its phone number and a
//     system user's access token, with the app secret that signs Meta's
//     webhooks. Credentials are sealed (internal/secrets) and never
//     returned. The display name, quality rating and messaging limit are
//     shown as Meta reports them.
//   - Templates, synced from Meta every hour and on demand. They're created
//     and edited in Meta's WhatsApp Manager; ReputeCast reads them.
//   - WhatsApp consent per list, like SMS consent (channel_consents with
//     channel 'whatsapp'); the SMS opt-out keywords work on WhatsApp too.
//   - The 24-hour customer service window: free-form messages only go to
//     people who wrote in the last 24 hours.
//   - WhatsApp campaigns with an approved template, its variables mapped to
//     contact fields with fallbacks, the pre-send checks (consent, the
//     template, the quality rating, the messaging limit and the cost from
//     Meta's rate card), and the report.
//   - The Send WhatsApp automation step.
//   - Meta's webhooks: verified with the app secret, stored raw, answered
//     200 at once and normalized by a River job.
//
// WhatsApp builds on X21's channel model: campaigns carry channel
// 'whatsapp', each message is a row of the message log (90 days, never the
// audit log), suppressions are channel_suppressions. Everything a person
// changes is audited.
package whatsapp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

// Service is the WhatsApp domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	URLs  publicurl.Builder
	// Jobs enqueues River jobs (read at call time: the client is built
	// after the services).
	Jobs func() *jobs.Client
	Now  func() time.Time
	// HTTP is the SSRF-guarded client Meta's Graph API is called through
	// (the in-process Graph API in tests).
	HTTP *http.Client
	// Demo is the in-process Graph API the demo's account answers from:
	// it never reaches Meta.
	Demo func() *metatest.Fake
	// Segments resolves a segment's contacts for an audience.
	Segments SegmentResolver
	// Approvals evaluates the workspace's approval rules for WhatsApp
	// campaigns and holds their requests (internal/approvals).
	Approvals Approvals
	// Changed re-checks a campaign's open approval after an edit.
	Changed func(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID)
}

// SegmentResolver lists a segment's contacts.
type SegmentResolver interface {
	ContactIDs(ctx context.Context, workspaceID, segmentID uuid.UUID) ([]uuid.UUID, error)
}

// Approvals is the approvals seam for WhatsApp campaigns.
type Approvals interface {
	ChannelGate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (*store.ApprovalRequest, error)
	ChannelReview(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (approvals.ReviewApproval, error)
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

func (s *Service) helpers(ctx context.Context, ws *store.Workspace, perm authz.Permission) []string {
	return access.Helpers(ctx, s.Q, ws.ID, perm)
}

var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

func notFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func ptr[T any](v T) *T { return &v }

func jsonOf(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// credAD names what sealed credentials belong to.
func credAD(id uuid.UUID) string { return "whatsapp_account:" + id.String() + ":credentials" }

// errNoAccount is a workspace without a WhatsApp account.
var errNoAccount = errors.New("whatsapp: not connected")

// account is the workspace's connected account.
func (s *Service) account(ctx context.Context, q *store.Queries, ws *store.Workspace) (store.WhatsAppAccountRow, error) {
	a, err := q.WhatsAppAccount(ctx, ws.ID)
	if notFound(err) {
		return a, errNoAccount
	}
	return a, err
}

// credentials opens an account's sealed credentials.
func (s *Service) credentials(id uuid.UUID, sealed []byte) (meta.Credentials, error) {
	var c meta.Credentials
	raw, err := s.Box.Open(sealed, credAD(id))
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(raw, &c)
}

// client is the Graph API client of an account: the in-process one for
// the demo's.
func (s *Service) client(id uuid.UUID, sealed []byte, demo bool) (*meta.Client, error) {
	creds, err := s.credentials(id, sealed)
	if err != nil {
		return nil, err
	}
	return s.clientFor(creds, demo), nil
}

func (s *Service) clientFor(creds meta.Credentials, demo bool) *meta.Client {
	c := &meta.Client{Creds: creds, HTTP: s.HTTP}
	if demo && s.Demo != nil {
		c.HTTP = s.Demo().Client()
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 20 * time.Second}
	}
	return c
}

// randomToken is a verify token for Meta's handshake: "rc-wa-" and ten
// letters and digits.
func randomToken() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return "rc-wa-" + string(b)
}

// hint is the safe end of an id: "…4410".
func hint(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 4 {
		return id
	}
	return "…" + id[len(id)-4:]
}

// webhookURL is the address Meta posts to.
func (s *Service) webhookURL(ws *store.Workspace) string {
	return strings.TrimRight(s.URLs.Base(), "/") + "/hooks/whatsapp/" + ws.Slug
}

func location(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	l, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return l
}

func wsLocation(ws *store.Workspace) *time.Location { return location(ws.Timezone) }

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func round4(v float64) float64 { return float64(int64(v*10000+0.5)) / 10000 }

// Connected reports whether the workspace can send WhatsApp messages: an
// account Meta verified. The channel switch shows WhatsApp once it can.
func (s *Service) Connected(ctx context.Context, workspaceID uuid.UUID) (bool, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM whatsapp_accounts WHERE workspace_id = $1 AND removed_at IS NULL`, workspaceID).Scan(&n)
	return n > 0, err
}
