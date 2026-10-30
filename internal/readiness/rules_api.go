package readiness

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type ruleCreateInput struct {
	api.WorkspaceInput
	Body SendingRuleInput
}

type rulePathInput struct {
	api.WorkspaceInput
	RuleID string `path:"ruleId" format:"uuid"`
}

// The update's input names the path parameter itself: Huma doesn't fill
// it through a second level of embedding.
type ruleUpdateInput struct {
	api.WorkspaceInput
	RuleID string `path:"ruleId" format:"uuid"`
	Body   SendingRuleInput
}

func ruleID(ctx context.Context, raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ruleGone(ctx)
	}
	return id, nil
}

type sendDefaultsInput struct {
	api.WorkspaceInput
	Body SendDefaults
}

// registerRules adds Settings › Sending rules (SR-10): anyone who can see
// campaigns reads the rules; workspace:manage changes them (audited).
func registerRules(h huma.API, s *Service) {
	tags := []string{"Sending rules"}
	api.Get(h, api.Op{ID: "list-sending-rules", Path: "/workspaces/{workspace}/sending-rules", Summary: "The workspace's sending rules and audience defaults",
		Description: "Send Readiness checks every rule that is on for a send's stream, next to its built-in checks.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SendingRulesView], error) {
			out, err := s.SendingRules(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-sending-rule", Path: "/workspaces/{workspace}/sending-rules", Summary: "Add a sending rule",
		Tags: tags, Access: api.WorkspaceHuman(authz.WorkspaceManage), Status: 201},
		func(ctx context.Context, in *ruleCreateInput) (*api.Body[SendingRule], error) {
			out, err := s.CreateSendingRule(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-sending-rule", Path: "/workspaces/{workspace}/sending-rules/{ruleId}", Summary: "Change a sending rule, its severity, scope or status",
		Description: "Changes apply from the next check; a scheduled send picks them up at its send-time recheck.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.WorkspaceManage)},
		func(ctx context.Context, in *ruleUpdateInput) (*api.Body[SendingRule], error) {
			id, err := ruleID(ctx, in.RuleID)
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateSendingRule(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-sending-rule", Path: "/workspaces/{workspace}/sending-rules/{ruleId}", Summary: "Remove a sending rule",
		Tags: tags, Access: api.WorkspaceHuman(authz.WorkspaceManage), Status: 204},
		func(ctx context.Context, in *rulePathInput) (*api.Empty, error) {
			id, err := ruleID(ctx, in.RuleID)
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.DeleteSendingRule(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Put(h, api.Op{ID: "update-send-defaults", Path: "/workspaces/{workspace}/send-defaults", Summary: "Change how every send's audience is counted and confirmed",
		Description: "Whether sends leave out the unengaged contacts of the sunset policy, and when a sender types the recipient count.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.WorkspaceManage)},
		func(ctx context.Context, in *sendDefaultsInput) (*api.Body[SendingRulesView], error) {
			out, err := s.SaveSendDefaults(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
