package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
)

// The brand kit and its synced blocks (review 2026-10, CO-11).

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		brandkit.Register(h, a.brandKit())
	})
}

func (a *App) brandKit() *brandkit.Service {
	return &brandkit.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit,
		Campaigns: brandCampaigns{a: a},
		Media:     &media.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		// A website import fetches pages outside the instance.
		Guard: netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks},
	}
}

// brandCampaigns reaches the campaign domain when a synced block changes
// (built on use: the campaign domain's templates use the brand kit too).
type brandCampaigns struct{ a *App }

func (b brandCampaigns) CountSynced(ctx context.Context, ws *store.Workspace, kind string) (campaigns.SyncedUses, error) {
	return b.a.campaigns().CountSynced(ctx, ws, kind)
}

func (b brandCampaigns) ApplySynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind, block, label string) (campaigns.SyncedUses, []uuid.UUID, error) {
	return b.a.campaigns().ApplySynced(ctx, ws, p, kind, block, label)
}

func (b brandCampaigns) UnlinkSynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind, label string) (campaigns.SyncedUses, []uuid.UUID, error) {
	return b.a.campaigns().UnlinkSynced(ctx, ws, p, kind, label)
}
