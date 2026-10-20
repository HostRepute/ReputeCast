// Package push is ReputeCast's web push channel (design expansion phase 8,
// M32): built in, with no outside account.
//
//   - VAPID keys, generated here and rotated with a typed confirmation; the
//     private key is sealed (internal/secrets) and never returned.
//   - Sites: the publication website, whose service worker ReputeCast
//     serves, and other sites with the snippet and rc-push-sw.js, which a
//     check finds. Subscriptions from sites that aren't listed are refused.
//   - Subscriptions: anonymous until linked to a contact, each with its
//     consent record (page, exact prompt text and version, IP, time).
//   - The prompt designer, a form type: a slide-in bar, a dialog or a bell
//     button, when and where it shows, asking again after "Not now", and
//     the Home Screen steps on iPhone and iPad.
//   - Push campaigns with previews per platform, Review & send with its
//     checks, the frequency cap and quiet hours, sending over the Web Push
//     protocol (internal/push/webpush), expired subscriptions removed,
//     shown and click tracking, and the report.
//   - The Send push automation step.
//
// Push builds on X21's channel model (internal/channels): campaigns carry
// channel 'push' and each notification is a row of the message log, kept
// 90 days and never in the audit log. Everything a person changes is
// audited.
package push

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/push/webpush/webpushtest"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service is the push domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	URLs  publicurl.Builder
	Links *linktoken.Signer
	// Jobs enqueues River jobs (read at call time: the client is built
	// after the services).
	Jobs func() *jobs.Client
	Now  func() time.Time
	// HTTP is the SSRF-guarded client push services and sites are called
	// through.
	HTTP *http.Client
	// Demo is the in-process push service the demo's subscriptions are
	// sent to: they never reach FCM, Mozilla, Apple or Windows.
	Demo func() *webpushtest.Service
	// Segments resolves a segment's contacts for an audience.
	Segments SegmentResolver
	// Approvals evaluates the workspace's approval rules for push
	// campaigns and holds their requests (internal/approvals).
	Approvals Approvals
	// Changed re-checks a campaign's open approval after an edit.
	Changed func(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID)
	// Emit sends outgoing webhook events (internal/webhooks).
	Emit func(ctx context.Context, q store.DBTX, ws *store.Workspace, event string, data any) error
	// Website says whether the workspace's publication website is on and
	// its address (https://news.acme.io), "" when it has none.
	Website func(ctx context.Context, ws *store.Workspace) string
	// Member is the contact signed in on the website with a request's
	// session cookie (uuid.Nil when nobody is).
	Member func(ctx context.Context, ws *store.Workspace, r *http.Request) uuid.UUID
	// Logo is the workspace's logo address, for notifications without an
	// icon of their own ("" when it has none).
	Logo func(ctx context.Context, ws *store.Workspace) string
	// Brand is the publisher's colour and name (Forms & pages › Branding):
	// the prompt's Allow button and the Home Screen steps use them.
	Brand func(ctx context.Context, ws *store.Workspace) (accent, name string)
}

// SegmentResolver lists a segment's contacts.
type SegmentResolver interface {
	ContactIDs(ctx context.Context, workspaceID, segmentID uuid.UUID) ([]uuid.UUID, error)
}

// Approvals is the approvals seam for push campaigns.
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

// keyAD names what a sealed private key belongs to.
func keyAD(id uuid.UUID) string { return "push_key:" + id.String() + ":private" }

func notFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func ptr[T any](v T) *T { return &v }

// token is a random, URL-safe token for a notification.
func token() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// origin is the instance's own origin (https://reputecast.acme.io): the
// scripts and the endpoints of sites other than the publication website.
func (s *Service) origin() string { return strings.TrimRight(s.URLs.Base(), "/") }

// location loads a time zone, UTC when it's unknown.
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

// wsLocation is the workspace's time zone.
func wsLocation(ws *store.Workspace) *time.Location { return location(ws.Timezone) }

// clip cuts s to n runes.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func pct(n, d int64) float64 {
	if d <= 0 {
		return 0
	}
	return float64(int64(float64(n)*1000/float64(d)+0.5)) / 10
}

// errNoKeys is returned when push isn't set up.
var errNoKeys = errors.New("push: no keys")

// liveKey is the workspace's live VAPID key.
func (s *Service) liveKey(ctx context.Context, q *store.Queries, ws *store.Workspace) (store.PushKeyLiveRow, error) {
	k, err := q.PushKeyLive(ctx, ws.ID)
	if notFound(err) {
		return k, errNoKeys
	}
	return k, err
}

// Enabled reports whether the workspace can send push: it has keys and a
// site. The channel switch shows push once it does.
func (s *Service) Enabled(ctx context.Context, workspaceID uuid.UUID) (bool, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM push_keys k WHERE k.workspace_id = $1 AND k.rotated_at IS NULL
		AND EXISTS (SELECT 1 FROM push_sites s WHERE s.workspace_id = k.workspace_id AND s.removed_at IS NULL)`, workspaceID).Scan(&n)
	return n > 0, err
}

func plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func count(n int64) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
