package warmup

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type wsInput struct {
	api.WorkspaceInput
}

type planInput struct {
	api.WorkspaceInput
	PlanID string `path:"planId" format:"uuid"`
}

type settingsInput struct {
	planInput
	Body WarmupSettings
}

type createInput struct {
	api.WorkspaceInput
	Body NewPlanInput
}

// Register adds the warm-up operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Warm-up"}
	manage := api.Workspace(authz.ProvidersManage)
	api.Get(h, api.Op{ID: "get-warmup", Path: "/workspaces/{workspace}/warmup", Summary: "Warm-up plans with their ramp, progress and log",
		Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, _ *wsInput) (*api.Body[WarmupOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-warmup", Path: "/workspaces/{workspace}/warmup", Summary: "Start a warm-up plan for a provider or dedicated IP",
		Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *createInput) (*api.Body[WarmupPlanView], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-warmup", Path: "/workspaces/{workspace}/warmup/{planId}", Summary: "Change a plan's length, ordering and guards", Tags: tags, Access: manage},
		func(ctx context.Context, in *settingsInput) (*api.Body[WarmupPlanView], error) {
			out, err := s.UpdateSettings(ctx, api.WorkspaceFrom(ctx), in.PlanID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "pause-warmup", Path: "/workspaces/{workspace}/warmup/{planId}/pause", Summary: "Pause a warm-up plan", Tags: tags, Access: manage},
		func(ctx context.Context, in *planInput) (*api.Body[WarmupPlanView], error) {
			out, err := s.SetPaused(ctx, api.WorkspaceFrom(ctx), in.PlanID, true)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resume-warmup", Path: "/workspaces/{workspace}/warmup/{planId}/resume", Summary: "Resume a paused warm-up plan", Tags: tags, Access: manage},
		func(ctx context.Context, in *planInput) (*api.Body[WarmupPlanView], error) {
			out, err := s.SetPaused(ctx, api.WorkspaceFrom(ctx), in.PlanID, false)
			return api.OK(out), err
		})
}
