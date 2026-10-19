package ai

import (
	"context"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/store"
)

// CatalogEntries are the "AI models" group of Settings › Integrations: the
// five providers, the connected one with this month's use.
func (s *Service) CatalogEntries(ctx context.Context, ws *store.Workspace) ([]integrations.IntegrationEntry, error) {
	c, err := s.load(ctx, s.Q, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]integrations.IntegrationEntry, 0, len(llm.Providers))
	for _, p := range llm.Providers {
		e := integrations.IntegrationEntry{Key: p.Kind, Status: "not_connected", Reasons: []integrations.IntegrationReason{}, Facts: integrations.IntegrationFacts{}}
		if c.connected() {
			facts := &integrations.IntegrationAIFacts{}
			if c.Provider == p.Kind {
				e.Status = "connected"
				if c.Status == "key_rejected" {
					e.Status = "attention"
					facts.KeyRejected = true
				}
				u, err := s.usage(ctx, ws, c)
				if err != nil {
					return nil, err
				}
				facts.ModelName, facts.CostMicros, facts.Tokens, facts.CapKind, facts.CapUSD, facts.CapTokens, facts.Share =
					c.modelName(), u.costMicros, u.tokens, c.CapKind, c.CapUsd, c.CapTokens, u.share
			} else {
				facts.Connected = c.providerName()
			}
			e.Facts.AI = facts
		}
		out = append(out, e)
	}
	return out, nil
}
