package products

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Products in other screens (design expansion phase 7, X19): the campaign
// report's Revenue rows for product sales. An order is credited to the
// email its buyer last clicked within the attribution window before
// paying (consumer.go's attribute).

// CampaignProductSale is one product's sales credited to a campaign.
type CampaignProductSale struct {
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Orders    int64  `json:"orders"`
	// RevenueCents is after refunds.
	RevenueCents int64 `json:"revenueCents"`
}

// CampaignProductSales are the Revenue section's product sales rows.
type CampaignProductSales struct {
	Orders       int64                 `json:"orders"`
	RevenueCents int64                 `json:"revenueCents"`
	Currency     string                `json:"currency"`
	Products     []CampaignProductSale `json:"products"`
	// Window is the attribution window in days: an order counts for the
	// email its buyer last clicked within it.
	Window int `json:"window"`
}

// CampaignPath is a campaign of the workspace.
type CampaignPath struct {
	api.WorkspaceInput
	Campaign string `path:"campaign" format:"uuid"`
}

// CampaignSales are the product sales credited to a campaign (live orders).
func (s *Service) CampaignSales(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (CampaignProductSales, error) {
	out := CampaignProductSales{Products: []CampaignProductSale{}, Currency: "usd", Window: 7}
	if st, err := s.Q.EarnSettings(ctx, ws.ID); err == nil && st.AttributionDays > 0 {
		out.Window = int(st.AttributionDays)
	}
	var one int
	if err := s.Pool.QueryRow(ctx, `SELECT 1 FROM campaigns WHERE id = $1 AND workspace_id = $2`, campaignID, ws.ID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, api.NotFound("Campaign")
		}
		return out, err
	}
	cid := campaignID
	tot, err := s.Q.ProductOrdersByCampaign(ctx, store.ProductOrdersByCampaignParams{WorkspaceID: ws.ID, CampaignID: &cid})
	if err != nil {
		return out, err
	}
	out.Orders, out.RevenueCents = tot.Orders, tot.Revenue
	if tot.Currency != "" {
		out.Currency = tot.Currency
	}
	rows, err := s.Q.ProductOrdersByCampaignProducts(ctx, store.ProductOrdersByCampaignProductsParams{WorkspaceID: ws.ID, CampaignID: &cid})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Products = append(out.Products, CampaignProductSale{ProductID: r.ProductID.String(), Name: r.Name, Orders: r.Orders, RevenueCents: r.Revenue})
	}
	return out, nil
}

func registerElsewhere(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "get-campaign-product-sales", Path: "/workspaces/{workspace}/campaigns/{campaign}/product-sales",
		Summary: "Product sales credited to a campaign (its report's Revenue rows)", Tags: []string{"Products"}, Access: api.Workspace(authz.ProductsView)},
		func(ctx context.Context, in *CampaignPath) (*api.Body[CampaignProductSales], error) {
			out, err := s.CampaignSales(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Campaign))
			return api.OK(out), err
		})
}
