package campaigns

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type listInput struct {
	api.WorkspaceInput
	Status  string `query:"status" enum:"all,drafts,scheduled,sending,sent" default:"all"`
	Query   string `query:"q" maxLength:"200" doc:"Search by name"`
	Channel string `query:"channel" enum:"email,sms,push,whatsapp" doc:"Only this channel's campaigns"`
	// PaidOnly lists the campaigns for paid members only (Earn).
	PaidOnly bool `query:"paidOnly" doc:"Only campaigns sent to paid members only"`
	Page     int  `query:"page" minimum:"1" default:"1"`
	PageSize int  `query:"pageSize" minimum:"1" maximum:"100" default:"25"`
}

type summaryInput struct {
	api.WorkspaceInput
	Days int `query:"days" minimum:"1" maximum:"365" default:"30"`
}

// Register adds the campaigns operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Campaigns"}
	api.Get(h, api.Op{ID: "list-campaigns", Path: "/workspaces/{workspace}/campaigns", Summary: "List campaigns with status, audience, route and results", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *listInput) (*api.Body[CampaignList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), ListParams{Status: in.Status, Query: in.Query, Channel: in.Channel, PaidOnly: in.PaidOnly, Page: in.Page,
				PageSize: in.PageSize})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "campaigns-summary", Path: "/workspaces/{workspace}/campaigns/summary", Summary: "Totals across campaigns sent in the last N days", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *summaryInput) (*api.Body[CampaignSummary], error) {
			out, err := s.Summary(ctx, api.WorkspaceFrom(ctx), in.Days)
			return api.OK(out), err
		})
}
