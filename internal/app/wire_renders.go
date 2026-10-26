package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/renders"
	"github.com/hostrepute/reputecast/internal/store"
)

// Real renders from the workspace's own Litmus or Mailgun Inspect account
// (review 2026-10, CO-13).

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		renders.Register(h, a.renders())
	})
}

func (a *App) renders() *renders.Service {
	guard := netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks}
	return &renders.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets,
		Content: renderContent{a: a},
		// The services are fixed public hosts; calls still go through the
		// address guard like every call that leaves the instance.
		HTTP: guard.Client(30 * time.Second),
	}
}

// renderContent gives renders the saved campaign as the sample contact
// gets it (internal/campaigns).
type renderContent struct{ a *App }

func (r renderContent) SampleRender(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, variant string) (renders.Email, error) {
	s := r.a.campaigns()
	pv, version, err := s.SampleRender(ctx, ws, p, id, variant)
	if err != nil {
		return renders.Email{}, err
	}
	name, err := s.CampaignName(ctx, ws, id)
	return renders.Email{Subject: pv.Subject, HTML: pv.HTML, Version: version, Name: name}, err
}

func (r renderContent) CampaignName(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, error) {
	return r.a.campaigns().CampaignName(ctx, ws, id)
}
