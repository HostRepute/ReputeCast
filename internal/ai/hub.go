package ai

import (
	"context"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/store"
)

// CatalogEntries are the "AI models" group of Settings › Integrations: the
// five providers, each connected one with this month's use. A provider
// without a connected account names the connected one, which Connect waits
// for (it changes the first account).
func (s *Service) CatalogEntries(ctx context.Context, ws *store.Workspace) ([]integrations.IntegrationEntry, error) {
	m, err := s.models(ctx, s.Q, ws.ID)
	if err != nil {
		return nil, err
	}
	// The first connected account, the first account when it is.
	var first *connection
	for _, c := range m.list {
		if c.connected() {
			first = c
			break
		}
	}
	out := make([]integrations.IntegrationEntry, 0, len(llm.Providers))
	for _, p := range llm.Providers {
		e := integrations.IntegrationEntry{Key: p.Kind, Status: "not_connected", Reasons: []integrations.IntegrationReason{}, Facts: integrations.IntegrationFacts{}}
		if first != nil {
			facts := &integrations.IntegrationAIFacts{}
			var c *connection
			for _, a := range m.list {
				if a.Provider == p.Kind && a.connected() {
					c = a
					break
				}
			}
			if c != nil {
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
				facts.Connected = first.providerName()
			}
			e.Facts.AI = facts
		}
		out = append(out, e)
	}
	return out, nil
}
