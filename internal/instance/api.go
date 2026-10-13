package instance

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
)

type userPath struct {
	UserID string `path:"userId" format:"uuid"`
}

func (p userPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(p.UserID)
	if err != nil {
		return uuid.Nil, api.NotFound("User")
	}
	return id, nil
}

type workspacePath struct {
	WorkspaceID string `path:"workspaceId" format:"uuid"`
}

func (p workspacePath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(p.WorkspaceID)
	if err != nil {
		return uuid.Nil, api.NotFound("Workspace")
	}
	return id, nil
}

// Register adds the Instance admin operations. Every one needs an instance
// admin (a person, never an API key).
func Register(h huma.API, s *Service) {
	users := []string{"Instance admin · Users"}
	admin := api.InstanceAdmin

	api.Get(h, api.Op{ID: "list-instance-users", Path: "/admin/users", Summary: "Every account on the instance, with workspaces, sign-in methods and two-factor", Tags: users, Access: admin},
		func(ctx context.Context, in *struct {
			Q         string `query:"q" maxLength:"200"`
			Workspace string `query:"workspace" maxLength:"64" doc:"Workspace slug"`
			Status    string `query:"status" enum:"active,deactivated"`
			TwoFactor string `query:"twofa" enum:"on,off,idp"`
		}) (*api.Body[InstanceUserList], error) {
			out, err := s.Users(ctx, UserFilter{Search: in.Q, Workspace: in.Workspace, Status: in.Status, TwoFactor: in.TwoFactor})
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "sign-out-instance-user", Path: "/admin/users/{userId}/sign-out", Summary: "Sign a person out of every session", Tags: users, Access: admin},
		func(ctx context.Context, in *struct {
			userPath
			Body InstanceUserReason
		}) (*api.Body[InstanceSignOutResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SignOut(ctx, id, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "deactivate-instance-user", Path: "/admin/users/{userId}/deactivate", Summary: "Deactivate an account (reason required): signed out everywhere, history kept", Tags: users, Access: admin},
		func(ctx context.Context, in *struct {
			userPath
			Body InstanceDeactivateInput
		}) (*api.Body[InstanceDeactivateResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Deactivate(ctx, id, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "reactivate-instance-user", Path: "/admin/users/{userId}/reactivate", Summary: "Let a deactivated person sign in again", Tags: users, Access: admin, Status: http.StatusNoContent},
		func(ctx context.Context, in *userPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Reactivate(ctx, id)
		})

	api.Post(h, api.Op{ID: "unlock-instance-user", Path: "/admin/users/{userId}/unlock", Summary: "Clear a sign-in lockout", Tags: users, Access: admin, Status: http.StatusNoContent},
		func(ctx context.Context, in *userPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Unlock(ctx, id)
		})

	api.Post(h, api.Op{ID: "reset-instance-user-two-factor", Path: "/admin/users/{userId}/two-factor-reset", Summary: "Reset someone's authenticator (reason required and logged; confirm with your password)", Tags: users, Access: admin, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			userPath
			Body InstanceTwoFactorResetInput
		}) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.ResetTwoFactor(ctx, id, in.Body)
		})

	api.Get(h, api.Op{ID: "get-instance-user-password-reset", Path: "/admin/users/{userId}/password-reset", Summary: "Whether a reset email can go out now, and the CLI command", Tags: users, Access: admin},
		func(ctx context.Context, in *userPath) (*api.Body[InstancePasswordResetInfo], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.PasswordResetInfo(ctx, id)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "send-instance-user-password-reset", Path: "/admin/users/{userId}/password-reset", Summary: "Email a one-time password reset link on the System stream", Tags: users, Access: admin},
		func(ctx context.Context, in *userPath) (*api.Body[InstancePasswordResetSent], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SendPasswordReset(ctx, id)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "set-instance-admin", Path: "/admin/users/{userId}/instance-admin", Summary: "Make someone an instance admin, or remove it (never the last one)", Tags: users, Access: admin, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			userPath
			Body InstanceAdminInput
		}) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.SetInstanceAdmin(ctx, id, in.Body)
		})

	api.Get(h, api.Op{ID: "list-instance-sign-ins", Path: "/admin/sign-ins", Summary: "The latest sign-in attempts", Tags: users, Access: admin},
		func(ctx context.Context, in *struct {
			Limit int `query:"limit" minimum:"1" maximum:"50" default:"8"`
		}) (*api.Body[InstanceSignInList], error) {
			out, err := s.SignIns(ctx, in.Limit)
			return api.OK(out), err
		})

	wsTags := []string{"Instance admin · Workspaces"}
	api.Get(h, api.Op{ID: "list-instance-workspaces", Path: "/admin/workspaces", Summary: "Every workspace with members, subscribers and providers", Tags: wsTags, Access: admin},
		func(ctx context.Context, _ *struct{}) (*api.Body[InstanceWorkspaceList], error) {
			out, err := s.Workspaces(ctx)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-instance-workspace", Path: "/admin/workspaces", Summary: "Create a workspace with an Owner", Tags: wsTags, Access: admin, Status: http.StatusCreated},
		func(ctx context.Context, in *struct{ Body InstanceWorkspaceInput }) (*api.Body[InstanceWorkspace], error) {
			out, err := s.CreateWorkspace(ctx, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "delete-instance-workspace", Path: "/admin/workspaces/{workspaceId}/delete", Summary: "Delete a workspace (typed name; restorable for 7 days, then purged by a job)", Tags: wsTags, Access: admin},
		func(ctx context.Context, in *struct {
			workspacePath
			Body InstanceWorkspaceDeleteInput
		}) (*api.Body[InstanceWorkspaceDeleted], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.DeleteWorkspace(ctx, id, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "restore-instance-workspace", Path: "/admin/workspaces/{workspaceId}/restore", Summary: "Restore a deleted workspace during its 7-day hold", Tags: wsTags, Access: admin},
		func(ctx context.Context, in *workspacePath) (*api.Body[InstanceWorkspace], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.RestoreWorkspace(ctx, id)
			return api.OK(out), err
		})

	pol := []string{"Instance admin · Authentication"}
	api.Get(h, api.Op{ID: "get-instance-policy", Path: "/admin/policy", Summary: "Sign-in methods, password policy, two-factor, sessions, lockout, JIT allowlist", Tags: pol, Access: admin},
		func(ctx context.Context, _ *struct{}) (*api.Body[InstancePolicy], error) {
			out, err := s.Policy(ctx)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-instance-policy", Path: "/admin/policy", Summary: "Save the authentication policy", Tags: pol, Access: admin},
		func(ctx context.Context, in *struct{ Body InstancePolicyInput }) (*api.Body[InstancePolicy], error) {
			out, err := s.SavePolicy(ctx, in.Body)
			return api.OK(out), err
		})

	inst := []string{"Instance admin · Instance & updates"}
	api.Get(h, api.Op{ID: "get-instance-settings", Path: "/admin/instance", Summary: "Instance name, logo, public URL and default timezone", Tags: inst, Access: admin},
		func(ctx context.Context, _ *struct{}) (*api.Body[InstanceSettings], error) {
			out, err := s.Settings(ctx)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-instance-settings", Path: "/admin/instance", Summary: "Save the instance name, public URL and default timezone", Tags: inst, Access: admin},
		func(ctx context.Context, in *struct{ Body InstanceSettingsInput }) (*api.Body[InstanceSettings], error) {
			out, err := s.SaveSettings(ctx, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "check-instance-url", Path: "/admin/instance/check-url", Summary: "Check the public URL's DNS, TLS and reachability", Tags: inst, Access: admin},
		func(ctx context.Context, in *struct{ Body InstanceURLCheckInput }) (*api.Body[InstanceSettings], error) {
			out, err := s.CheckURL(ctx, in.Body)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "set-instance-logo", Path: "/admin/instance/logo", Summary: "Upload the instance logo (SVG or PNG, square)", Tags: inst, Access: admin, MaxBodyBytes: 2 << 20},
		func(ctx context.Context, in *struct{ Body InstanceLogoInput }) (*api.Body[InstanceSettings], error) {
			out, err := s.SetLogo(ctx, in.Body)
			return api.OK(out), err
		})

	api.Delete(h, api.Op{ID: "remove-instance-logo", Path: "/admin/instance/logo", Summary: "Use the ReputeCast mark instead of a logo", Tags: inst, Access: admin, Status: http.StatusOK},
		func(ctx context.Context, _ *struct{}) (*api.Body[InstanceSettings], error) {
			out, err := s.RemoveLogo(ctx)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-instance-version", Path: "/admin/updates", Summary: "Installed version, latest release, changelog and how to upgrade", Tags: inst, Access: admin},
		func(ctx context.Context, _ *struct{}) (*api.Body[InstanceVersion], error) {
			out, err := s.Version(ctx)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "set-instance-update-check", Path: "/admin/updates", Summary: "Turn the daily GitHub release check on or off", Tags: inst, Access: admin},
		func(ctx context.Context, in *struct{ Body InstanceUpdateCheckInput }) (*api.Body[InstanceVersion], error) {
			out, err := s.SetUpdateCheck(ctx, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "check-instance-updates", Path: "/admin/updates/check", Summary: "Fetch the latest release now (when the check is on)", Tags: inst, Access: admin},
		func(ctx context.Context, _ *struct{}) (*api.Body[InstanceVersion], error) {
			out, err := s.CheckNow(ctx)
			return api.OK(out), err
		})
}
