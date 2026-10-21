package commerce

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/store"
)

// Revenue from email outside the store's page: the campaign report's store
// rows, Earn › Overview's card, and the automations list. Each order is
// credited to one email (a campaign, or an automation's email step), so
// these never count an order twice.

// CampaignRevenue is a campaign's credited store orders, one row per store
// (the Revenue section's store rows).
func (s *Service) CampaignRevenue(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (CommerceCampaignRevenue, error) {
	out := CommerceCampaignRevenue{Rows: []CommerceCampaignStoreRow{}}
	rows, err := s.Q.CommerceCampaignRevenue(ctx, store.CommerceCampaignRevenueParams{CampaignID: &campaignID, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Rows = append(out.Rows, CommerceCampaignStoreRow{StoreID: r.StoreID.String(), Name: r.Name, Platform: r.Platform, Orders: r.Orders, Revenue: r.Revenue,
			Currency: r.Currency, Days: int(r.AttributionDays), ByToken: r.ByToken, ByEmail: r.ByEmail})
	}
	return out, nil
}

// RevenueSummary is Earn › Overview's "Store revenue from email": every
// store's orders in the last 30 days and the part credited to email.
func (s *Service) RevenueSummary(ctx context.Context, ws *store.Workspace) (CommerceRevenueSummary, error) {
	out := CommerceRevenueSummary{Days: PeriodDays}
	rows, err := s.Q.CommerceStores(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, st := range rows {
		if st.Status != StatusConnecting {
			out.Stores++
			if out.Currency == "" {
				out.Currency = st.Currency
			}
		}
	}
	if out.Stores == 0 {
		return out, nil
	}
	now := s.now()
	r, err := s.Q.CommerceWorkspaceRevenue(ctx, store.CommerceWorkspaceRevenueParams{WorkspaceID: ws.ID, Since: now.Add(-PeriodDays * 24 * time.Hour), Until: now})
	if err != nil {
		return out, err
	}
	out.Orders, out.Revenue, out.Credited, out.CreditedRevenue = r.Orders, r.Revenue, r.Credited, r.CreditedRevenue
	if r.Currency != "" {
		out.Currency = r.Currency
	}
	out.Share = share(r.CreditedRevenue, r.Revenue)
	return out, nil
}

// AutomationRevenue is the credited store orders of automations since a
// time (the automations list's 30-day columns). Nil when the workspace has
// no store.
func (s *Service) AutomationRevenue(ctx context.Context, wsID uuid.UUID, ids []uuid.UUID, since time.Time) map[uuid.UUID]automations.AutomationRevenue {
	rows, err := s.Q.CommerceStores(ctx, wsID)
	if err != nil || len(rows) == 0 {
		return nil
	}
	currency := rows[0].Currency
	out := map[uuid.UUID]automations.AutomationRevenue{}
	for _, id := range ids {
		out[id] = automations.AutomationRevenue{Currency: currency}
	}
	got, err := s.Q.CommerceAutomationRevenue(ctx, store.CommerceAutomationRevenueParams{WorkspaceID: wsID, Ids: ids, Since: since})
	if err != nil {
		s.log().WarnContext(ctx, "automations revenue not read", "err", err)
		return nil
	}
	for _, r := range got {
		if r.AutomationID == nil {
			continue
		}
		out[*r.AutomationID] = automations.AutomationRevenue{Orders: r.Orders, Revenue: r.Revenue, Currency: r.Currency}
	}
	return out
}
