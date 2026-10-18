// Package website is the publication website (design expansion phase 7,
// M28; brief docs/spec/reputecast-design-addon-publish-sell.md): the
// publisher's site at their custom domain, grown out of the public archive.
//
//   - Turning it on makes every archived campaign a public post (its old
//     archive address redirects), each archived list a tag, the root of the
//     domain the home page and members able to sign in. Turning it off brings
//     the archive back and keeps everything for next time.
//   - Posts come from campaigns ("Publish to web": with the send, at a set
//     time, or now for a sent campaign) or are written for the web only in
//     the composer's post mode. A post's web version is drawn from its MJML
//     by internal/website/render, in the site's theme.
//   - Visibility, the free preview marker and the meter gate posts on the
//     server (internal/website/gate): text a reader may not see never reaches
//     their browser, the feed, the sitemap or the search index.
//   - Members sign in with a link and a 6-digit code (System stream, the
//     contact's language); sessions are a first-party cookie on the site.
//   - Views are counted on the server without cookies: one per visitor, page
//     and day, from a hash with a salt that changes daily.
//
// The public pages themselves are rendered by internal/publicpages with the
// site's frame (templates/site); this package gives them their data.
package website

import (
	"context"
	"embed"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "website")

// Inserter enqueues River jobs (the job client).
type Inserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Approvals is the approval rule for publishing posts (internal/approvals):
// whether a person may publish a post as it is, and asking for approval.
type Approvals interface {
	PostDecision(ctx context.Context, ws *store.Workspace, p *authz.Principal, req PostApprovalRequest) (PostApprovalDecision, error)
	RequestPost(ctx context.Context, ws *store.Workspace, p *authz.Principal, req PostApprovalRequest, note string) (uuid.UUID, error)
	// PostPublished settles the approval a publish relied on.
	PostPublished(ctx context.Context, ws *store.Workspace, postID uuid.UUID, hash string) error
}

// CampaignDrafts makes a campaign draft from a web-only post ("Send as a
// campaign"), returning its id.
type CampaignDrafts interface {
	DraftFromPost(ctx context.Context, ws *store.Workspace, p *authz.Principal, name, subject, preheader, mjml string) (uuid.UUID, error)
}

// Service is the website domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	URLs  publicurl.Builder
	// Forms holds the branding, the custom domain and the archive.
	Forms *forms.Service
	// Audience signs readers up and changes their newsletters.
	Audience *audience.Service
	// Earn gives the paid tiers, their prices and the billing links (nil
	// without payments).
	Earn func() *earn.Service
	// Mail sends the sign-in emails on the System stream.
	Mail      func() mailer.Sender
	Jobs      func() Inserter
	Approvals func() Approvals
	Campaigns func() CampaignDrafts
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

func (s *Service) earn() *earn.Service {
	if s.Earn == nil {
		return nil
	}
	return s.Earn()
}

// location is the workspace's time zone (dates on the site, the meter's months).
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// day is a date in the workspace's time zone, as a UTC midnight.
func day(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}
