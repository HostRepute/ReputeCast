package app

import (
	"context"
	"sort"

	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/store"
)

// verificationCatalog is the Verification group of Settings ›
// Integrations: one card per service, with its status and balance.
func verificationCatalog(a *App) integrations.GroupSource {
	return func(ctx context.Context, ws *store.Workspace) ([]integrations.IntegrationEntry, error) {
		o, err := a.verification().Overview(ctx, ws)
		if err != nil {
			return nil, err
		}
		out := []integrations.IntegrationEntry{}
		for _, v := range o.Services {
			e := integrations.IntegrationEntry{Key: v.Key, Status: integrations.StatusNotConnected, Reasons: []integrations.IntegrationReason{}}
			switch {
			case !v.Connected:
			case v.Status == "low_credits" || v.Status == "error":
				e.Status, e.Facts.Balance, e.Facts.Price = integrations.StatusAttention, v.Balance, v.Price
			default:
				e.Status, e.Facts.Balance, e.Facts.Price = integrations.StatusConnected, v.Balance, v.Price
			}
			out = append(out, e)
		}
		// The catalog's order (Integrations · catalog board).
		rank := map[string]int{"zerobounce": 0, "neverbounce": 1, "kickbox": 2, "bouncer": 3, "emailable": 4}
		sort.SliceStable(out, func(i, j int) bool { return rank[out[i].Key] < rank[out[j].Key] })
		return out, nil
	}
}
