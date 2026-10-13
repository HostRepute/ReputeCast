package approvals

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/readiness"
)

type campaignInput = readiness.ReadinessCampaignPath

type sendInput struct {
	readiness.ReadinessCampaignPath
	Body ReviewSendInput
}

type bypassInput struct {
	readiness.ReadinessCampaignPath
	Body ApprovalBypassInput
}

type requestInput struct {
	readiness.ReadinessCampaignPath
	Body ApprovalRequestInput
}

// ApprovalPath is the {workspace}/{approvalId} of approval operations.
type ApprovalPath struct {
	api.WorkspaceInput
	ApprovalID string `path:"approvalId" format:"uuid"`
}

func (in ApprovalPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.ApprovalID)
	if err != nil {
		return uuid.Nil, api.NotFound("Approval request")
	}
	return id, nil
}

type listInput struct {
	api.WorkspaceInput
	Status   string `query:"status" enum:"pending,approved,changes_requested,rejected,invalidated,all" default:"pending"`
	Author   string `query:"author" doc:"Requester user id"`
	Approver string `query:"approver" doc:"Approver user id"`
	Days     int    `query:"days" minimum:"0" maximum:"3650" default:"30" doc:"Requested in the last N days (0: all)"`
}

type decisionInput struct {
	ApprovalPath
	Body ApprovalDecisionInput
}

type commentInput struct {
	ApprovalPath
	Body ApprovalCommentInput
}

type scheduleInput struct {
	ApprovalPath
	Body ReviewSendInput
}

type policyInput struct {
	api.WorkspaceInput
	Body ApprovalPolicy
}

// ApprovalRequested is the request that was opened.
type ApprovalRequested struct {
	ID string `json:"id"`
}

// Register adds the Review & send and approval operations.
func Register(h huma.API, s *Service) {
	review := []string{"Review & send"}
	tags := []string{"Approvals"}
	api.Get(h, api.Op{ID: "get-campaign-review", Path: "/workspaces/{workspace}/campaigns/{campaignId}/review",
		Summary: "Review & send: the latest Send Readiness run, estimates, schedule, thresholds, test sends and approval state", Tags: review, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *campaignInput) (*api.Body[CampaignReview], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			out, err := s.Review(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/send", Summary: "Schedule or start a campaign after Send Readiness",
		Description: "Needs a fresh, unblocked Send Readiness run, every warning confirmed, the recipient count typed for large sends, and an approval when the policy asks for one.",
		Tags:        review, Access: api.WorkspaceHuman(authz.CampaignsSend)},
		func(ctx context.Context, in *sendInput) (*api.Body[ReviewSendResult], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			out, err := s.Send(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unschedule-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/unschedule", Summary: "Take a scheduled campaign back",
		Tags: review, Access: api.WorkspaceHuman(authz.CampaignsEdit), Status: 204},
		func(ctx context.Context, in *campaignInput) (*api.Empty, error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Unschedule(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "bypass-approval", Path: "/workspaces/{workspace}/campaigns/{campaignId}/bypass", Summary: "Send without approval (Owners, with a reason)",
		Description: "Logged as campaign.approval_bypassed with the reason and the snapshot; every approver is notified. Send Readiness still runs and a Blocked result still stops the send.",
		Tags:        review, Access: api.WorkspaceHuman(authz.CampaignsSend)},
		func(ctx context.Context, in *bypassInput) (*api.Body[ReviewSendResult], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			out, err := s.Bypass(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "request-approval", Path: "/workspaces/{workspace}/campaigns/{campaignId}/approval", Summary: "Ask the approvers to approve this send",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsEdit), Status: 201},
		func(ctx context.Context, in *requestInput) (*api.Body[ApprovalRequested], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			rid, err := s.RequestApproval(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(ApprovalRequested{ID: rid.String()}), err
		})
	api.Get(h, api.Op{ID: "list-approvals", Path: "/workspaces/{workspace}/approvals", Summary: "The approval inbox: approvers see every request, authors their own",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *listInput) (*api.Body[ApprovalList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), ListParams{Status: in.Status, Author: in.Author, Approver: in.Approver, Days: in.Days})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-approval", Path: "/workspaces/{workspace}/approvals/{approvalId}", Summary: "An approval request with its snapshot, readiness, diff and discussion",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *ApprovalPath) (*api.Body[ApprovalDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "preview-approval", Path: "/workspaces/{workspace}/approvals/{approvalId}/preview", Summary: "The email an approval covers",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *ApprovalPath) (*api.Body[ApprovalPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Preview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "decide-approval", Path: "/workspaces/{workspace}/approvals/{approvalId}/decision", Summary: "Approve, request changes or reject",
		Description: "Authors can't decide on their own request. Deciding on an invalidated request decides on the campaign as it stands (a new request with the diff).",
		Tags:        tags, Access: api.WorkspaceHuman(authz.CampaignsApprove)},
		func(ctx context.Context, in *decisionInput) (*api.Body[ApprovalDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			rid, err := s.Decide(ctx, ws, p, id, in.Body)
			if err != nil {
				return nil, err
			}
			out, err := s.Detail(ctx, ws, p, rid)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "comment-approval", Path: "/workspaces/{workspace}/approvals/{approvalId}/comments", Summary: "Add to the discussion (author and approvers)",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsView), Status: 204},
		func(ctx context.Context, in *commentInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Comment(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
		})
	api.Post(h, api.Op{ID: "cancel-approval", Path: "/workspaces/{workspace}/approvals/{approvalId}/cancel", Summary: "Withdraw a request (its author or an Owner)",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsEdit), Status: 204},
		func(ctx context.Context, in *ApprovalPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Cancel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "remind-approvers", Path: "/workspaces/{workspace}/approvals/{approvalId}/remind", Summary: "Remind the approvers (once an hour)",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsEdit), Status: 204},
		func(ctx context.Context, in *ApprovalPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Remind(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "restore-approval", Path: "/workspaces/{workspace}/approvals/{approvalId}/restore", Summary: "Put the campaign back as approved (keeps the approval)",
		Tags: tags, Access: api.WorkspaceHuman(authz.CampaignsEdit), Status: 204},
		func(ctx context.Context, in *ApprovalPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Restore(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "schedule-approved", Path: "/workspaces/{workspace}/approvals/{approvalId}/schedule", Summary: "Schedule an approved campaign as approved",
		Description: "The author can schedule an approved campaign without campaigns:send: the approval is the permission for this exact snapshot.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *scheduleInput) (*api.Body[ReviewSendResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.ScheduleApproved(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-approval-policy", Path: "/workspaces/{workspace}/approval-policy", Summary: "The approval policy and what it matches today",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[ApprovalPolicyView], error) {
			out, err := s.PolicyView(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-approval-policy", Path: "/workspaces/{workspace}/approval-policy", Summary: "Change the approval policy",
		Tags: tags, Access: api.WorkspaceHuman(authz.WorkspaceManage)},
		func(ctx context.Context, in *policyInput) (*api.Body[ApprovalPolicyView], error) {
			out, err := s.SavePolicy(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
