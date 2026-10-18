package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/media"
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

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		sponsors.Register(h, sponsorService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		sponsors.RegisterJobs(a.JobReg, sponsorService(a))
	})
}
