package digests

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "digests")

// Checks runs Send Readiness on issues and counts audiences (readiness.Service).
type Checks interface {
	Run(ctx context.Context, ws *store.Workspace, id uuid.UUID, o readiness.RunOptions) (readiness.ReadinessRun, error)
	Latest(ctx context.Context, campaignID uuid.UUID) (*readiness.ReadinessRun, error)
	Audience(ctx context.Context, ws *store.Workspace, raw json.RawMessage) (readiness.AudienceFacts, error)
}

// Approvals asks for approval of issues in their owner's name and checks,
// as an issue starts, that its approval still covers it (approvals.Service).
type Approvals interface {
	RequestForIssue(ctx context.Context, ws *store.Workspace, ownerID, campaignID uuid.UUID, in approvals.IssueRequest) (uuid.UUID, error)
	SendRecheck(ctx context.Context, campaignID uuid.UUID) (approvals.SendRecheck, error)
	CampaignChanged(ctx context.Context, ws *store.Workspace, id uuid.UUID)
}

// Starter starts an issue's send through the sending engine, on its
// owner's authority, with the send-time recheck that was just run.
type Starter interface {
	StartIssue(ctx context.Context, campaignID, ownerID uuid.UUID, rc sending.Recheck) error
	Pause(ctx context.Context, campaignID, by uuid.UUID, reason string) error
}

// Tester sends test emails of a campaign (campaigns.Service.SendTest).
type Tester interface {
	SendTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in campaigns.ComposerTestSend) (campaigns.ComposerTestSendResult, error)
}

// Inserter enqueues River jobs (the job client).
type Inserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Service runs recurring digests.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	// Fetcher reads feeds and images through the SSRF guard.
	Fetcher func() *feed.Fetcher
	// Checks, Approvals and Start are Send Readiness, the approvals domain
	// and the sending engine, read at call time (they are built lazily).
	Checks    func() Checks
	Approvals func() Approvals
	Start     func() Starter
	Tests     func() Tester
	URLs      publicurl.Builder
	Jobs      func() Inserter
	// LatestPlugin is the newest WordPress plugin version (the integrations hub).
	LatestPlugin func(ctx context.Context) string
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

func (s *Service) fetcher() *feed.Fetcher {
	if s.Fetcher != nil {
		if f := s.Fetcher(); f != nil {
			return f
		}
	}
	return &feed.Fetcher{}
}

func (s *Service) checks() Checks {
	if s.Checks == nil {
		return nil
	}
	return s.Checks()
}

func (s *Service) approvals() Approvals {
	if s.Approvals == nil {
		return nil
	}
	return s.Approvals()
}

func (s *Service) starter() Starter {
	if s.Start == nil {
		return nil
	}
	return s.Start()
}

// enqueue adds a job when a job client exists (tests drive jobs directly).
func (s *Service) enqueue(ctx context.Context, args river.JobArgs) {
	if s.Jobs == nil {
		return
	}
	c := s.Jobs()
	if c == nil {
		return
	}
	if _, err := c.Insert(ctx, args, nil); err != nil {
		s.log().WarnContext(ctx, "enqueue digest job", "kind", args.Kind(), "err", err)
	}
}

// member is a workspace member with their permissions.
type member struct {
	ID       uuid.UUID
	Name     string
	Color    string
	RoleName string
	Active   bool
	Perms    authz.Set
}

func (s *Service) member(ctx context.Context, q *store.Queries, wsID, userID uuid.UUID) (*member, error) {
	r, err := q.DigestMember(ctx, store.DigestMemberParams{WorkspaceID: wsID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &member{ID: r.ID, Name: r.Name, Color: r.AvatarColor, RoleName: r.RoleName, Active: r.UserStatus == "active" && r.MemberStatus == "active",
		Perms: authz.Effective(r.RoleKey, r.Builtin, r.Permissions)}, nil
}

// members are the workspace's active members, Owners and Admins first.
func (s *Service) members(ctx context.Context, wsID uuid.UUID) ([]member, error) {
	rows, err := s.Q.ApprovalMembers(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]member, 0, len(rows))
	for _, r := range rows {
		out = append(out, member{ID: r.ID, Name: r.Name, Color: r.AvatarColor, RoleName: r.RoleName, Active: true,
			Perms: authz.Effective(r.RoleKey, r.Builtin, r.Permissions)})
	}
	return out, nil
}

// helpers names up to three members holding perm, first one (the digest's
// owner) when they hold it.
func helpers(ms []member, perm authz.Permission, first *uuid.UUID) []string {
	out := []string{}
	if first != nil {
		for _, m := range ms {
			if m.ID == *first && m.Perms.Has(perm) {
				out = append(out, m.Name)
			}
		}
	}
	for _, m := range ms {
		if len(out) == 3 {
			break
		}
		if m.Perms.Has(perm) && (first == nil || m.ID != *first) {
			out = append(out, m.Name)
		}
	}
	return out
}

func person(id uuid.UUID, name, color string) *DigestPerson {
	return &DigestPerson{ID: id.String(), Name: name, Initials: initials(name), Color: color}
}

// initials of a name: "Sofia Rossi" → "SR".
func initials(name string) string {
	out := ""
	for _, p := range strings.Fields(name) {
		r := []rune(p)
		if len(r) > 0 {
			out += strings.ToUpper(string(r[0]))
		}
		if len([]rune(out)) == 2 {
			break
		}
	}
	return out
}

func (s *Service) workspace(ctx context.Context, id uuid.UUID) (*store.Workspace, error) {
	ws, err := s.Q.GetWorkspace(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// actorID is the person behind a request, nil for keys and agents.
func actorID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.Type != authz.ActorUser || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// shortURL is an address without its scheme ("blog.acme.io/feed/").
func shortURL(u string) string {
	return strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
}

func hostOf(u string) string {
	s := shortURL(u)
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return strings.ToLower(s)
}
