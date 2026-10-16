// Package placement runs placement tests: one copy of an email to each seed
// mailbox the workspace owns, through the route as configured, one provider
// or several side by side, then finds each copy by its test token and says
// where it landed (Inbox, a tab, Spam, Not arrived), how long it took, and
// what the mailbox's Authentication-Results say.
//
// Tests are one-off checks people start (Review & send, the composer, the
// warm-up planner, Placement tests): no schedules and no alerts, which are
// HostRepute's side of the line. A test never changes or blocks a send.
// Copies are real sends with the same headers and link tracking; their
// opens and clicks are never counted and they never appear in reports or
// contact timelines.
package placement

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/systemmail"
)

// Waiting: a copy missing 15 minutes after the last hand-off is Not
// arrived; one that turns up within 60 minutes updates the result.
const (
	WaitFor    = 15 * time.Minute
	LateWithin = 60 * time.Minute
	// CompareGap separates the two providers' copies to a seed.
	CompareGap = 2 * time.Minute
)

// Kind is the messages.kind of a copy.
const Kind = "placement"

// Sender hands one copy to a provider (systemmail.Deliverer).
type Sender interface {
	Send(ctx context.Context, p store.Provider, msg providers.Message, kind string) (systemmail.Delivery, error)
}

// Transactional renders a transactional template's live version with test data.
type Transactional interface {
	RenderLive(ctx context.Context, ws *store.Workspace, key string, data map[string]any, to string) (subject, html, text string, version int, err error)
}

// MailApp is the instance's Google or Microsoft sign-in app (Instance admin
// › Authentication) as seed mailboxes use it for mail access.
type MailApp struct {
	Kind         string
	ClientID     string
	ClientSecret string
	// Tenant is Microsoft's (common, organizations, consumers or one tenant).
	Tenant string
}

// MailApps reads the instance's sign-in apps.
type MailApps interface {
	MailApp(ctx context.Context, kind string) (MailApp, bool, error)
}

// Readiness is the latest Send Readiness run of a campaign, for the guidance.
type Readiness interface {
	Latest(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (ReadinessSummary, bool)
}

// ReadinessSummary is a run's counts and blocking findings: their labels
// and one sentence of evidence each.
type ReadinessSummary struct {
	Passed, Warnings, Blocks int
	Blocking, Evidence       []string
}

// Notifier tells the starter a test finished and managers a seed failed.
type Notifier interface {
	TestFinished(ctx context.Context, ws *store.Workspace, t store.PlacementTest, summary string)
	SeedFailed(ctx context.Context, ws *store.Workspace, seed store.PlacementSeed)
}

// Service is the placement domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	// Deliver hands copies to providers through the normal sending path.
	Deliver       Sender
	Links         *linktoken.Signer
	URLs          publicurl.Builder
	Transactional Transactional
	Apps          MailApps
	Readiness     Readiness
	Notify        Notifier
	// LangOf is a member's language, for what ReputeCast writes to them.
	LangOf func(ctx context.Context, userID uuid.UUID) i18n.Lang
	// Jobs is River's client (read lazily); nil or Inline runs work in
	// the request (tests).
	Jobs   func() *jobs.Client
	Inline bool
	// Guard dials seed mailboxes' IMAP servers (user-supplied hosts).
	Guard netguard.Policy
	// HTTP calls Microsoft Graph and the OAuth token endpoints.
	HTTP *http.Client
	// OAuthEndpoints replaces Google's and Microsoft's (tests).
	OAuthEndpoints *Endpoints
	// GraphURL replaces Microsoft Graph's root (tests).
	GraphURL string
	// InsecureIMAP dials test IMAP servers without TLS; IMAPHost and
	// IMAPPort replace the fixed servers of OAuth seeds (tests).
	InsecureIMAP bool
	IMAPHost     string
	IMAPPort     int
	Now          func() time.Time
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

func (s *Service) client() bool { return !s.Inline && s.Jobs != nil && s.Jobs() != nil }

func (s *Service) httpClient() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func actor(ctx context.Context) *uuid.UUID {
	if u := api.UserFrom(ctx); u != nil {
		id := u.ID
		return &id
	}
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		id := p.UserID
		return &id
	}
	return nil
}

// newToken is a copy's test token ("pt-7f3a91c2").
func newToken() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "pt-" + hex.EncodeToString(b)
}

// Helpers names up to three active members who hold perm, narrowest role
// first and the Owner last ("Ask Sofia Rossi, Marcus Chen or Linh Nguyen").
func Helpers(ctx context.Context, q *store.Queries, wsID uuid.UUID, perm authz.Permission) []string {
	out := []string{}
	rows, err := q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return out
	}
	rank := func(key string) int {
		switch key {
		case authz.RoleOwner:
			return 2
		case authz.RoleAdmin:
			return 1
		}
		return 0
	}
	slices.SortStableFunc(rows, func(a, b store.ListMembersWithRolesRow) int { return rank(a.RoleKey) - rank(b.RoleKey) })
	for _, r := range rows {
		if authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			out = append(out, r.Name)
			if len(out) == 3 {
				break
			}
		}
	}
	return out
}

func ptrTime(t time.Time) *time.Time { return &t }
