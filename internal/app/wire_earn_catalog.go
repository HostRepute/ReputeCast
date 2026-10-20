package app

import (
	"context"

	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/store"
)

// paymentsCatalog is the Payments group of Settings › Integrations (Earn,
// brief §6): Stripe, connected or not, which opens Earn › Settings ›
// Payments, and the merchants of record that aren't available yet.
func paymentsCatalog(a *App) integrations.GroupSource {
	return func(ctx context.Context, ws *store.Workspace) ([]integrations.IntegrationEntry, error) {
		st, err := earnService(a).Status(ctx, ws, nil)
		if err != nil {
			return nil, err
		}
		stripe := integrations.IntegrationEntry{Key: "stripe", Status: integrations.StatusNotConnected, Reasons: []integrations.IntegrationReason{},
			Facts: integrations.IntegrationFacts{Payment: &integrations.IntegrationPaymentFacts{Available: true}}}
		if st.Provider == "stripe" {
			stripe.Status = integrations.StatusConnected
			if st.Health != "" && st.Health != "connected" {
				stripe.Status = integrations.StatusAttention
			}
			stripe.Facts.Payment.Account, stripe.Facts.Payment.Mode = st.Account, st.Mode
		}
		out := []integrations.IntegrationEntry{stripe}
		for _, key := range []string{"paddle", "lemonsqueezy"} {
			out = append(out, integrations.IntegrationEntry{Key: key, Status: integrations.StatusNotConnected, Reasons: []integrations.IntegrationReason{},
				Facts: integrations.IntegrationFacts{Payment: &integrations.IntegrationPaymentFacts{}}})
		}
		return out, nil
	}
}
