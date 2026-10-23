package revenue

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type overviewInput struct {
	api.WorkspaceInput
	OverviewQuery
}

type campaignInput struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The campaign's id"`
}

type attributionInput struct {
	api.WorkspaceInput
	Body AttributionInput
}

type conversionsInput struct {
	api.WorkspaceInput
	ConversionsQuery
}

type recordInput struct {
	api.WorkspaceInput
	IdempotencyKey string `header:"Idempotency-Key" maxLength:"200" doc:"Sending the same key again answers with the first conversion instead of recording a second"`
	Body           ConversionInput
}

// Register adds the revenue and attribution operations. Reading needs
// earn:view (a campaign's Revenue section says who can show it without);
// the attribution settings need payments:manage and a person; recording a
// conversion needs conversions:create, which a conversions key holds alone.
func Register(h huma.API, s *Service) {
	tags := []string{"Earn"}
	view := api.Workspace(authz.EarnView)
	ws := "/workspaces/{workspace}"

	api.Get(h, api.Op{ID: "get-revenue-overview", Path: ws + "/earn/revenue", Summary: "Earn › Overview: MRR and its movement, paid members, trials, conversions, cancellations and attribution",
		Description: "MRR is the monthly value of every Active and Past due subscription, after discounts, before tax, live mode only; a yearly price counts as a twelfth.",
		Tags:        tags, Access: view},
		func(ctx context.Context, in *overviewInput) (*api.Body[RevenueOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.OverviewQuery)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-earn-card", Path: ws + "/earn/card", Summary: "The Earn card of the workspace Overview", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnCard], error) {
			out, err := s.Card(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign-revenue", Path: ws + "/campaigns/{id}/revenue", Summary: "A campaign report's Revenue section: upgrades and conversions credited to it",
		Description: "Without earn:view the answer is locked and names who can show it.", Tags: []string{"Reports"}, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *campaignInput) (*api.Body[CampaignRevenue], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, campaignNotFound(ctx)
			}
			out, err := s.Campaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-attribution", Path: ws + "/earn/attribution", Summary: "Earn › Settings › Attribution: the window, the click token and the conversion API",
		Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[AttributionSettings], error) {
			out, err := s.Attribution(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-attribution", Path: ws + "/earn/attribution", Summary: "Change the attribution window and the click token",
		Tags: tags, Access: api.WorkspaceHuman(authz.PaymentsManage)},
		func(ctx context.Context, in *attributionInput) (*api.Body[AttributionSettings], error) {
			out, err := s.SetAttribution(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-test-conversion", Path: ws + "/earn/attribution/test", Summary: "Record a test conversion for your own address",
		Description: "Shown in the conversions log, left out of every number.", Tags: tags, Access: api.WorkspaceHuman(authz.PaymentsManage), Status: http.StatusAccepted},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ConversionResult], error) {
			out, err := s.TestConversion(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-conversions", Path: ws + "/conversions", Summary: "The conversions log, newest first", Tags: tags, Access: view},
		func(ctx context.Context, in *conversionsInput) (*api.Body[ConversionList], error) {
			out, err := s.Conversions(ctx, api.WorkspaceFrom(ctx), in.ConversionsQuery)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-conversion", Path: ws + "/conversions", Summary: "Record a conversion: an upgrade or an order in your product, with its value",
		Description: "Credits the conversion to the last campaign or automation email the reader clicked within the attribution window: by click token " +
			"(rc_click), else by email address. Answers 202 with how it was matched. Errors: missing_identity, invalid_value, invalid_currency, " +
			"occurred_at_out_of_range (422), idempotency_key_reused (409), rate_limited (429, with Retry-After). Conversions are logged in Earn › Settings › " +
			"Attribution, not in the audit log.",
		Tags: tags, Access: api.Workspace(authz.ConversionsCreate), Status: http.StatusAccepted, MaxBodyBytes: 32 << 10},
		func(ctx context.Context, in *recordInput) (*api.Body[ConversionResult], error) {
			out, err := s.Record(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body, in.IdempotencyKey, false)
			return api.OK(out), err
		})
}
