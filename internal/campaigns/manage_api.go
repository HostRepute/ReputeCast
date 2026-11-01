package campaigns

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

type campaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" maxLength:"40"`
}

// followUpInput is a follow-up to make (huma reads path parameters of
// exported fields only, so the path isn't embedded).
type followUpInput struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" maxLength:"40"`
	Body       CampaignFollowUpInput
}

func (f *followUpInput) campaignPath() campaignPath {
	return campaignPath{WorkspaceInput: f.WorkspaceInput, CampaignID: f.CampaignID}
}

func (c campaignPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(c.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

// wsLocation is the workspace's time zone (UTC when unknown).
func wsLocation(ws *store.Workspace) *time.Location {
	if loc, err := time.LoadLocation(ws.Timezone); err == nil && ws.Timezone != "" {
		return loc
	}
	return time.UTC
}

// registerManage adds deleting, restoring, archiving, bulk actions, tags,
// the calendar and follow-ups (review 2026-11 CA-1–CA-4).
func registerManage(h huma.API, s *Service) {
	tags := []string{"Campaigns"}
	edit := api.Workspace(authz.CampaignsEdit)
	api.Delete(h, api.Op{ID: "delete-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}", Summary: "Delete a draft or a campaign that sent nothing (restorable for 30 days)",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *campaignPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), userRef(api.PrincipalFrom(ctx)), id)
		})
	api.Post(h, api.Op{ID: "restore-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/restore", Summary: "Bring back a campaign deleted in the last 30 days",
		Tags: tags, Access: edit, Status: http.StatusNoContent},
		func(ctx context.Context, in *campaignPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.RestoreDeleted(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "archive-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/archive", Summary: "Archive a sent campaign (its report and web post stay)",
		Tags: tags, Access: edit, Status: http.StatusNoContent},
		func(ctx context.Context, in *campaignPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Archive(ctx, api.WorkspaceFrom(ctx), userRef(api.PrincipalFrom(ctx)), id, true)
		})
	api.Post(h, api.Op{ID: "unarchive-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/unarchive", Summary: "Put an archived campaign back on the list",
		Tags: tags, Access: edit, Status: http.StatusNoContent},
		func(ctx context.Context, in *campaignPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Archive(ctx, api.WorkspaceFrom(ctx), userRef(api.PrincipalFrom(ctx)), id, false)
		})
	api.Post(h, api.Op{ID: "bulk-campaigns", Path: "/workspaces/{workspace}/campaigns/bulk", Summary: "Archive, unarchive, delete or tag campaigns together",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body CampaignBulkInput
		}) (*api.Body[CampaignBulkResult], error) {
			out, err := s.Bulk(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-campaign-tags", Path: "/workspaces/{workspace}/campaigns/tags", Summary: "The tags campaigns carry",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[CampaignTags], error) {
			out, err := s.Tags(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "campaign-calendar", Path: "/workspaces/{workspace}/campaigns/calendar", Summary: "Scheduled and sent campaigns and digest issues between two days",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			From string `query:"from" required:"true" pattern:"^\\d{4}-\\d{2}-\\d{2}$" doc:"First day (the workspace's time zone)"`
			To   string `query:"to" required:"true" pattern:"^\\d{4}-\\d{2}-\\d{2}$" doc:"Day after the last"`
		}) (*api.Body[CampaignCalendar], error) {
			ws := api.WorkspaceFrom(ctx)
			loc := wsLocation(ws)
			from, err := time.ParseInLocation(time.DateOnly, in.From, loc)
			if err != nil {
				return nil, api.Invalid("from", err.Error())
			}
			to, err := time.ParseInLocation(time.DateOnly, in.To, loc)
			if err != nil {
				return nil, api.Invalid("to", err.Error())
			}
			out, err := s.Calendar(ctx, ws, from, to)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "campaign-follow-up-counts", Path: "/workspaces/{workspace}/campaigns/{campaignId}/follow-up", Summary: "Who a follow-up would reach: non-openers and clickers",
		Tags: tags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *campaignPath) (*api.Body[CampaignFollowUpCounts], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.FollowUpCounts(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-campaign-follow-up", Path: "/workspaces/{workspace}/campaigns/{campaignId}/follow-up", Summary: "Make a follow-up draft for non-openers or clickers",
		Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *followUpInput) (*api.Body[ComposerCampaign], error) {
			id, err := in.campaignPath().id()
			if err != nil {
				return nil, err
			}
			out, err := s.FollowUp(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
}
