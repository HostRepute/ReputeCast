package team

import (
	"context"
	"encoding/json"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

func jsonUnmarshal(b []byte, v any) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}

type listInput struct {
	api.WorkspaceInput
	Query  string `query:"q" maxLength:"200" doc:"Search name or email"`
	Role   string `query:"role" maxLength:"64" doc:"Role id or key"`
	Status string `query:"status" enum:"active,deactivated"`
	TwoFA  string `query:"twofa" enum:"on,off,idp,required"`
}

type memberInput struct {
	api.WorkspaceInput
	MemberID string `path:"memberId" format:"uuid"`
}

func (in *memberInput) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.MemberID)
	if err != nil {
		return uuid.Nil, api.NotFound("Member")
	}
	return id, nil
}

type invitationInput struct {
	api.WorkspaceInput
	InvitationID string `path:"invitationId" format:"uuid"`
}

func (in *invitationInput) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.InvitationID)
	if err != nil {
		return uuid.Nil, api.NotFound("Invitation")
	}
	return id, nil
}

// Register adds the team operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Team"}
	base := "/workspaces/{workspace}"

	api.Get(h, api.Op{ID: "list-members", Path: base + "/members", Summary: "List members with role, scope, sign-in methods and two-factor status", Tags: tags, Access: api.Workspace(authz.TeamView)},
		func(ctx context.Context, in *listInput) (*api.Body[TeamMemberList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), ListFilter{Query: in.Query, Role: in.Role, Status: in.Status, TwoFA: in.TwoFA})
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-member", Path: base + "/members/{memberId}", Summary: "Member detail: effective permissions and where each comes from, keys, approvals, activity", Tags: tags, Access: api.Workspace(authz.TeamView)},
		func(ctx context.Context, in *memberInput) (*api.Body[TeamMemberDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), api.MapError(err, "Member")
		})

	api.Patch(h, api.Op{ID: "update-member", Path: base + "/members/{memberId}", Summary: "Change a member's role or scope", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage), Status: 204},
		func(ctx context.Context, in *struct {
			memberInput
			Body ChangeInput
		}) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, api.MapError(s.ChangeRole(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body), "Member")
		})

	api.Post(h, api.Op{ID: "deactivate-member", Path: base + "/members/{memberId}/deactivate", Summary: "Deactivate a member: keeps history, signs out everywhere, keeps or revokes their API keys", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage)},
		func(ctx context.Context, in *struct {
			memberInput
			Body DeactivateInput
		}) (*api.Body[DeactivateResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Deactivate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), api.MapError(err, "Member")
		})

	api.Post(h, api.Op{ID: "reactivate-member", Path: base + "/members/{memberId}/reactivate", Summary: "Reactivate a deactivated member", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage), Status: 204},
		func(ctx context.Context, in *memberInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, api.MapError(s.Reactivate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id), "Member")
		})

	api.Delete(h, api.Op{ID: "remove-member", Path: base + "/members/{memberId}", Summary: "Remove a member (the audit log keeps their history)", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage)},
		func(ctx context.Context, in *memberInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, api.MapError(s.Remove(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id), "Member")
		})

	api.Post(h, api.Op{ID: "sign-out-member", Path: base + "/members/{memberId}/sign-out", Summary: "End every session of a member", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage)},
		func(ctx context.Context, in *memberInput) (*api.Body[SignOutResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SignOut(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), api.MapError(err, "Member")
		})

	api.Post(h, api.Op{ID: "bulk-members", Path: base + "/members/bulk", Summary: "Change the role of, require two-factor for, or deactivate several members", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body BulkInput
		}) (*api.Body[BulkResult], error) {
			out, err := s.Bulk(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), api.MapError(err, "Member")
		})

	api.Get(h, api.Op{ID: "list-invitations", Path: base + "/invitations", Summary: "List pending invitations", Tags: tags, Access: api.Workspace(authz.TeamView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[TeamInvitationList], error) {
			out, err := s.Invitations(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "invitation-options", Path: base + "/invitations/options", Summary: "Roles you can grant, scopes, SSO hints and whether email can be sent", Tags: tags, Access: api.WorkspaceHuman(authz.TeamInvite)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[InviteOptions], error) {
			out, err := s.Options(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-invitations", Path: base + "/invitations", Summary: "Invite people by email or copyable link", Tags: tags, Access: api.WorkspaceHuman(authz.TeamInvite), Status: 201},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body InviteInput
		}) (*api.Body[InviteResult], error) {
			out, err := s.Invite(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), api.MapError(err, "Invitation")
		})

	api.Post(h, api.Op{ID: "resend-invitation", Path: base + "/invitations/{invitationId}/resend", Summary: "Email a pending invitation again and restart its expiry", Tags: tags, Access: api.WorkspaceHuman(authz.TeamInvite)},
		func(ctx context.Context, in *invitationInput) (*api.Body[ResendResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Resend(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), api.MapError(err, "Invitation")
		})

	api.Post(h, api.Op{ID: "invitation-link", Path: base + "/invitations/{invitationId}/link", Summary: "Copy the link of a pending invitation", Tags: tags, Access: api.WorkspaceHuman(authz.TeamInvite)},
		func(ctx context.Context, in *invitationInput) (*api.Body[InviteLink], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Link(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	api.Delete(h, api.Op{ID: "revoke-invitation", Path: base + "/invitations/{invitationId}", Summary: "Revoke a pending invitation", Tags: tags, Access: api.WorkspaceHuman(authz.TeamInvite)},
		func(ctx context.Context, in *invitationInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, api.MapError(s.Revoke(ctx, api.WorkspaceFrom(ctx), id), "Invitation")
		})
}
