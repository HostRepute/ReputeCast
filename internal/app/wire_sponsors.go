package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sponsors"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sponsorships (design expansion phase 6, X17): the sponsor library, slots,
// bookings with their creative and payment, sponsor reports and their
// read-only links, and the composer's Sponsor block.

var sponsorServices sync.Map // *App → *sponsors.Service

func sponsorService(a *App) *sponsors.Service {
	if v, ok := sponsorServices.Load(a); ok {
		return v.(*sponsors.Service)
	}
	s := &sponsors.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, URLs: a.URLs, Log: a.Log,
		Images: sponsorImages{&media.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit}}}
	v, _ := sponsorServices.LoadOrStore(a, s)
	return v.(*sponsors.Service)
}

// Sponsors is the sponsorships domain (tests set its clock).
func (a *App) Sponsors() *sponsors.Service { return sponsorService(a) }

// sponsorImages keeps a creative's logo or image in the media library.
type sponsorImages struct{ m *media.Service }

func (i sponsorImages) Upload(ctx context.Context, ws *store.Workspace, p *authz.Principal, name string, data []byte) (string, error) {
	var actor *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		id := p.UserID
		actor = &id
	}
	asset, err := i.m.Upload(ctx, ws, actor, name, data)
	if err != nil {
		return "", err
	}
	return asset.URL, nil
}

// readinessSponsors is the Sponsor block in Send Readiness.
type readinessSponsors struct{ s *sponsors.Service }

func (r readinessSponsors) SponsorFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src string) readiness.SponsorFacts {
	f := r.s.ReadinessFacts(ctx, ws, campaignID, src)
	return readiness.SponsorFacts{HasBlock: f.HasBlock, State: f.State, Sponsor: f.Sponsor, Issue: f.Issue, BookingID: f.BookingID, CreativeDue: f.CreativeDue,
		Stale: f.Stale, Labelled: f.Labelled}
}

func (r readinessSponsors) FixSponsor(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, src, action string) (string, string, error) {
	return r.s.FixMJML(ctx, ws, campaignID, src, action)
}

func init() {
	// The Sponsor block: the composer's checks and Send Readiness.
	campaignMore = func(a *App) campaigns.MoreChecks { return sponsorService(a).ComposerIssues }
	readinessSponsor = func(a *App) readiness.SponsorChecker { return readinessSponsors{sponsorService(a)} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		sponsors.Register(h, sponsorService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		sponsors.RegisterJobs(a.JobReg, sponsorService(a))
	})
}
