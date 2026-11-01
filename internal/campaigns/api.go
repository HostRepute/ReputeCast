package campaigns

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type listInput struct {
	api.WorkspaceInput
	Status string `query:"status" enum:"all,drafts,scheduled,sending,sent,archived,deleted" default:"all"`
	Query  string `query:"q" maxLength:"200" doc:"Search names and subjects"`
	// From and To (YYYY-MM-DD, the workspace's days) bound when a campaign
	// went or goes out; Sender, List and Tag narrow the list (CA-2).
	From    string `query:"from" pattern:"^(\\d{4}-\\d{2}-\\d{2})?$"`
	To      string `query:"to" pattern:"^(\\d{4}-\\d{2}-\\d{2})?$"`
	Sender  string `query:"sender" maxLength:"40"`
	List    string `query:"list" maxLength:"40"`
	Tag     string `query:"tag" maxLength:"40"`
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
			ws := api.WorkspaceFrom(ctx)
			p := ListParams{Status: in.Status, Query: in.Query, Channel: in.Channel, PaidOnly: in.PaidOnly, Page: in.Page, PageSize: in.PageSize, List: in.List, Tag: in.Tag}
			loc := wsLocation(ws)
			if in.From != "" {
				if t, err := time.ParseInLocation(time.DateOnly, in.From, loc); err == nil {
					p.From = &t
				}
			}
			if in.To != "" {
				if t, err := time.ParseInLocation(time.DateOnly, in.To, loc); err == nil {
					t = t.AddDate(0, 0, 1)
					p.To = &t
				}
			}
			if id, err := uuid.Parse(in.Sender); err == nil {
				p.Sender = &id
			}
			out, err := s.List(ctx, ws, p)
			return api.OK(out), err
		})
	registerManage(h, s)
	api.Get(h, api.Op{ID: "campaigns-summary", Path: "/workspaces/{workspace}/campaigns/summary", Summary: "Totals across campaigns sent in the last N days", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *summaryInput) (*api.Body[CampaignSummary], error) {
			out, err := s.Summary(ctx, api.WorkspaceFrom(ctx), in.Days)
			return api.OK(out), err
		})
}
