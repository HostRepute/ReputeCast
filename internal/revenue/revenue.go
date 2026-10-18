// Package revenue is Earn's numbers and attribution (design expansion phase
// 6, M27): MRR and its movement from the ledger the database keeps of every
// change to a live subscription's value, free-to-paid conversion by how
// readers first joined, upgrades by the email they're credited to, the
// campaign report's Revenue section, and the conversion API that credits a
// valuable action in the publisher's own product to the email clicked
// within the attribution window (by click token, else by address).
//
// MRR has one definition everywhere: the monthly value of every Active and
// Past due subscription, after discounts, before tax, live mode only; a
// yearly price counts as a twelfth; trials, free periods, complimentary
// and gift access, gift sales, sponsors and conversions are left out. A
// scheduled cancellation counts as churn when access ends.
//
// Conversions are kept in their own log, not in the audit log; a changed
// attribution setting is audited ("earn.attribution_updated").
package revenue

import (
	"context"
	"embed"
	"encoding/json"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

//go:embed messages/*.json
var messagesFS embed.FS

// msgs holds the server's own text: the conversion API's problems.
var msgs = i18n.MustLoad(messagesFS, "revenue")

// ClickToken is the click token's kind (X17's linktoken range is 170–179):
// its Link is the click's time in minutes since the Unix epoch.
const ClickToken linktoken.Kind = 170

// Service is Earn's revenue and attribution.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Links *linktoken.Signer
	URLs  publicurl.Builder
	Log   *slog.Logger
	Now   func() time.Time
	// Growth names the sources readers joined through (free-to-paid).
	Growth *growth.Service
	// PerMinute is a key's conversion rate limit (600 when zero).
	PerMinute int

	mu       sync.Mutex
	buckets  map[uuid.UUID]*bucket
	settings sync.Map // workspace → cachedSettings
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

// text is a message of the catalog in the request's language.
func text(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

func userID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.Type != authz.ActorUser || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// location is the workspace's time zone.
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// startOfDay is the start of t's day in the workspace's time zone.
func startOfDay(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

// mrr is the monthly value of a yearly one (the ledger keeps twelve times
// MRR so yearly prices stay exact), rounded to the minor unit.
func mrr(arr int64) int64 {
	return int64(math.Round(float64(arr) / 12))
}

// pct is part of whole as a percentage, or nil when whole is zero.
func pct(part, whole int64) *float64 {
	if whole <= 0 {
		return nil
	}
	v := float64(part) / float64(whole) * 100
	return &v
}

// helpers names the people who can change Earn (Admins and Owners first),
// leaving out the person asking.
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID, exclude uuid.UUID, most int, perms ...authz.Permission) []string {
	out := []string{}
	rows, err := s.Q.EarnTeam(ctx, wsID)
	if err != nil {
		return out
	}
	type cand struct {
		name string
		rank int
	}
	var list []cand
	for _, r := range rows {
		if r.UserID == exclude {
			continue
		}
		eff := authz.Effective(r.RoleKey, r.Builtin, r.Permissions)
		ok := false
		for _, p := range perms {
			if eff.Has(p) {
				ok = true
			}
		}
		if !ok {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = r.Email
		}
		rank := 2
		switch r.RoleKey {
		case authz.RoleAdmin:
			rank = 0
		case authz.RoleOwner:
			rank = 1
		}
		list = append(list, cand{name, rank})
	}
	for rank := 0; rank <= 2 && len(out) < most; rank++ {
		for _, c := range list {
			if c.rank == rank && len(out) < most {
				out = append(out, c.name)
			}
		}
	}
	return out
}

// emit queues a webhook event for the workspace's endpoints that listen.
func (s *Service) emit(ctx context.Context, q *store.Queries, ws *store.Workspace, event string, data any) error {
	hooks, err := q.HooksMatching(ctx, store.HooksMatchingParams{WorkspaceID: ws.ID, Event: event})
	if err != nil || len(hooks) == 0 {
		return err
	}
	eventID := newID()
	body, err := json.Marshal(webhooks.Envelope{ID: eventID.String(), Type: event, CreatedAt: s.now().UTC(), Workspace: ws.Slug, Data: data})
	if err != nil {
		return err
	}
	now := s.now()
	for _, h := range hooks {
		state := "pending"
		if !h.Enabled {
			state = "held"
		}
		if err := q.HooksInsertDelivery(ctx, store.HooksInsertDeliveryParams{ID: newID(), WebhookID: h.ID, Event: event, EventID: &eventID, Payload: body,
			State: state, NextAttemptAt: &now, MaxAttempts: webhooks.MaxAttempts}); err != nil {
			return err
		}
	}
	return nil
}
