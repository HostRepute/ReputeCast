package app

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/digests"
	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/overview"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
)

// Recurring and RSS digests (X10, M20). Each issue is a campaign: it goes
// through Send Readiness, the approval rules and the sending engine.

// digestFetcher reads feeds and their images through the same guard as
// importers (providers.allow_private_networks lets a test instance read a
// feed on its own network).
func digestFetcher(a *App) *feed.Fetcher {
	policy := netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks}
	f := &feed.Fetcher{Policy: policy}
	if a.Cfg != nil && (a.Cfg.IsDevelopment() || a.Cfg.Env == "test") {
		f.Transport = digestFeedOverlay(policy.Client(feed.Timeout).Transport)
	}
	return f
}

// digestFeedOverlay answers the demo's blog in development and tests; the
// demo replaces it. Other addresses go to next.
var digestFeedOverlay = func(next http.RoundTripper) http.RoundTripper { return next }

// digestsService builds the digests service. Send Readiness, approvals and
// the sending engine are built when first used: they build each other.
func digestsService(a *App) *digests.Service {
	return &digests.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, URLs: a.URLs,
		Fetcher:   func() *feed.Fetcher { return digestFetcher(a) },
		Checks:    func() digests.Checks { return readinessService(a) },
		Approvals: func() digests.Approvals { return newApprovals(a) },
		Start:     func() digests.Starter { return issueStarter{a: a} },
		Tests:     func() digests.Tester { return readinessService(a).Campaigns },
		Jobs: func() digests.Inserter {
			if a.Jobs == nil {
				return nil
			}
			return a.Jobs
		},
		LatestPlugin: func(ctx context.Context) string { return a.integrations().LatestPlugin(ctx) },
	}
}

// Digests returns the App's digests service (tests drive its jobs).
func (a *App) Digests() *digests.Service { return digestsService(a) }

// issueStarter starts an issue through the sending engine: on its owner's
// authority, with the send-time recheck the digest just ran, recorded as
// "System · {digest}, turned on by {owner}". A starter that isn't the
// engine (tests) gets a plain start with the recheck attached.
type issueStarter struct{ a *App }

// StartIssue implements digests.Starter.
func (s issueStarter) StartIssue(ctx context.Context, campaignID, ownerID uuid.UUID, rc sending.Recheck) error {
	ctx = sending.WithRecheck(ctx, campaignID, rc)
	if e, ok := s.a.Sending.(*engine.Engine); ok {
		owner := ownerID
		return e.StartWith(ctx, campaignID, engine.StartOptions{By: ownerID, Now: true, Scheduled: true, ScheduledBy: &owner})
	}
	return s.a.Sending.Start(ctx, campaignID, ownerID)
}

// Pause implements digests.Starter.
func (s issueStarter) Pause(ctx context.Context, campaignID, by uuid.UUID, reason string) error {
	return s.a.Sending.Pause(ctx, campaignID, by, reason)
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		digests.Register(h, digestsService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		digests.RegisterJobs(a.JobReg, digestsService(a))
	})
	// The demo's blog answers offline in development and tests.
	digestFeedOverlay = demo.DigestFeedOverlay
	// A post a connected WordPress site reports is read into its digests at once.
	integrationPostReported = func(a *App) func(ctx context.Context, siteID uuid.UUID) {
		return func(ctx context.Context, siteID uuid.UUID) { digestsService(a).PostReported(ctx, siteID) }
	}
	// The overview's "Next scheduled campaign" can be a digest's issue.
	overviewNextIssue = func(a *App) func(ctx context.Context, ws *store.Workspace) (*overview.OverviewNextIssue, error) {
		return func(ctx context.Context, ws *store.Workspace) (*overview.OverviewNextIssue, error) {
			n, err := digestsService(a).NextAcross(ctx, ws)
			if err != nil || n == nil {
				return nil, err
			}
			return &overview.OverviewNextIssue{DigestID: n.DigestID, DigestName: n.DigestName, Number: n.Number, SendAt: n.SendAt, Timezone: n.Timezone,
				Posts: n.Posts, Status: n.Status, NeedsApproval: n.NeedsApproval}, nil
		}
	}
	// The engine names the system behind an issue's start and completion.
	startHooks = append(startHooks, func(a *App) error {
		sendingEngine(a).SystemActor = digestsService(a).IssueActor
		return nil
	})
}
