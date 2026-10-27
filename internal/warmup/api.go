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
	api.WorkspaceInput
	PlanID string `path:"planId" format:"uuid"`
	Body   WarmupSettings
}

type createInput struct {
	api.WorkspaceInput
	Body NewPlanInput
}

type holdInput struct {
	api.WorkspaceInput
	PlanID     string `path:"planId" format:"uuid"`
	CampaignID string `path:"campaignId" format:"uuid"`
	Body       WarmupHoldDecision
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
	api.Post(h, api.Op{ID: "decide-warmup-hold", Path: "/workspaces/{workspace}/warmup/{planId}/held/{campaignId}",
		Summary: "Wait for tomorrow's cap, move or cancel a share held at the warm-up cap",
		Description: "A share held at a warming provider's daily cap waits for tomorrow's cap by itself. Move sends it through another provider once it passes " +
			"the same checks as a failover target; cancel drops the held rest of this share only. Cancelling needs campaigns:pause; waiting and moving resume " +
			"the send and need campaigns:send.",
		Tags: tags, Access: api.Workspace(authz.CampaignsPause)},
		func(ctx context.Context, in *holdInput) (*api.Body[WarmupHoldDecisionResult], error) {
			out, err := s.DecideCapHold(ctx, api.WorkspaceFrom(ctx), in.PlanID, in.CampaignID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resume-warmup", Path: "/workspaces/{workspace}/warmup/{planId}/resume", Summary: "Resume a paused warm-up plan", Tags: tags, Access: manage},
		func(ctx context.Context, in *planInput) (*api.Body[WarmupPlanView], error) {
			out, err := s.SetPaused(ctx, api.WorkspaceFrom(ctx), in.PlanID, false)
			return api.OK(out), err
		})
}
