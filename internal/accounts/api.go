package accounts

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
)

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

type signInOut struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      SignInResult
}

type cookieOut struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

func signedIn(r SignInResult, cookie *http.Cookie, err error) (*signInOut, error) {
	if err != nil {
		return nil, err
	}
	out := &signInOut{Body: r}
	if cookie != nil {
		out.SetCookie = *cookie
	}
	return out, nil
}

// Access for personal operations: a signed-in person, never an API key or
// agent. pending also works before setup is finished (temporary password,
// required two-factor), so the setup screens can use it.
var (
	person  = api.Access{Kind: api.AccessUser, HumansOnly: true}
	pending = api.Access{Kind: api.AccessUser, HumansOnly: true, AllowPending: true}
)

type idPath struct {
	ID string `path:"id" format:"uuid"`
}

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return id, nil
}

type tokenPath struct {
	Token string `path:"token" minLength:"10" maxLength:"200"`
}

type inviteAcceptOut struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      InviteAccepted
}

// Register adds the accounts operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Sign-in"}
	api.Post(h, api.Op{ID: "auth-identify", Path: "/auth/identify", Summary: "Identifier-first: which sign-in step comes next", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Email string `json:"email" format:"email" maxLength:"254"`
			}
		}) (*api.Body[IdentifyResult], error) {
			r, err := s.Identify(ctx, in.Body.Email)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "auth-password", Path: "/auth/password", Summary: "Sign in with email and password", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Email    string `json:"email" format:"email" maxLength:"254"`
				Password string `json:"password" minLength:"1" maxLength:"1024"`
				Remember bool   `json:"remember,omitempty"`
			}
		}) (*signInOut, error) {
			return signedIn(s.PasswordSignIn(ctx, in.Body.Email, in.Body.Password, in.Body.Remember))
		})

	api.Post(h, api.Op{ID: "auth-two-factor", Path: "/auth/two-factor", Summary: "Finish signing in with an authenticator or recovery code", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Challenge string `json:"challenge" minLength:"10" maxLength:"200"`
				Method    string `json:"method" enum:"totp,recovery"`
				Code      string `json:"code" minLength:"6" maxLength:"32"`
			}
		}) (*signInOut, error) {
			return signedIn(s.TwoFactor(ctx, in.Body.Challenge, in.Body.Method, in.Body.Code))
		})

	api.Post(h, api.Op{ID: "auth-passkey-options", Path: "/auth/passkey/options", Summary: "Start “Sign in with a passkey” (no email needed)", Tags: tags, Access: api.Public},
		func(ctx context.Context, _ *struct{}) (*api.Body[PasskeyOptions], error) {
			r, err := s.BeginPasskeySignIn(ctx)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "auth-passkey", Path: "/auth/passkey", Summary: "Sign in with a passkey", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Challenge  string          `json:"challenge" minLength:"10" maxLength:"200"`
				Credential json.RawMessage `json:"credential" doc:"The browser's AuthenticationResponseJSON"`
				Remember   bool            `json:"remember,omitempty"`
			}
		}) (*signInOut, error) {
			return signedIn(s.PasskeySignIn(ctx, in.Body.Challenge, in.Body.Credential, in.Body.Remember))
		})

	api.Post(h, api.Op{ID: "auth-two-factor-passkey-options", Path: "/auth/two-factor/passkey/options", Summary: "Start confirming a sign-in with a passkey as the second factor", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Challenge string `json:"challenge" minLength:"10" maxLength:"200" doc:"The two-factor challenge from the password step"`
			}
		}) (*api.Body[PasskeyOptions], error) {
			r, err := s.BeginTwoFactorPasskey(ctx, in.Body.Challenge)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "auth-two-factor-passkey", Path: "/auth/two-factor/passkey", Summary: "Finish signing in with a passkey as the second factor", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Challenge        string          `json:"challenge" minLength:"10" maxLength:"200"`
				PasskeyChallenge string          `json:"passkeyChallenge" minLength:"10" maxLength:"200"`
				Credential       json.RawMessage `json:"credential" doc:"The browser's AuthenticationResponseJSON"`
			}
		}) (*signInOut, error) {
			return signedIn(s.TwoFactorPasskey(ctx, in.Body.Challenge, in.Body.PasskeyChallenge, in.Body.Credential))
		})

	api.Post(h, api.Op{ID: "auth-password-reset-request", Path: "/auth/password-reset", Summary: "Email a password reset link (or explain the alternatives)", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Email string `json:"email" format:"email" maxLength:"254"`
			}
		}) (*api.Body[PasswordResetRequest], error) {
			r, err := s.RequestPasswordReset(ctx, in.Body.Email)
			return api.OK(r), err
		})

	api.Get(h, api.Op{ID: "auth-password-reset-check", Path: "/auth/password-reset/{token}", Summary: "Check a password reset link", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *tokenPath) (*api.Body[PasswordResetInfo], error) {
			r, err := s.CheckPasswordReset(ctx, in.Token)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "auth-password-reset", Path: "/auth/password-reset/{token}", Summary: "Set a new password with a one-time reset link", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			tokenPath
			Body struct {
				Password string `json:"password" minLength:"1" maxLength:"1024"`
			}
		}) (*api.Body[PasswordResetDone], error) {
			r, err := s.ResetPassword(ctx, in.Token, in.Body.Password)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "auth-sign-out", Path: "/auth/sign-out", Summary: "Sign out of this session", Tags: tags, Access: api.Pending, Status: http.StatusNoContent},
		func(ctx context.Context, _ *struct{}) (*cookieOut, error) {
			return &cookieOut{SetCookie: s.SignOut(ctx)}, nil
		})

	inv := []string{"Invitations"}
	api.Get(h, api.Op{ID: "get-invitation", Path: "/invitations/{token}", Summary: "What an invitation link offers: workspace, inviter, role and scope", Tags: inv, Access: api.Public},
		func(ctx context.Context, in *tokenPath) (*api.Body[InvitationView], error) {
			r, err := s.Invitation(ctx, in.Token)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "accept-invitation", Path: "/invitations/{token}/accept", Summary: "Join the workspace (creates the account and signs in for new people)", Tags: inv, Access: api.Public},
		func(ctx context.Context, in *struct {
			tokenPath
			Body AcceptInviteInput
		}) (*inviteAcceptOut, error) {
			r, cookie, err := s.AcceptInvitation(ctx, in.Token, in.Body)
			if err != nil {
				return nil, err
			}
			out := &inviteAcceptOut{Body: r}
			if cookie != nil {
				out.SetCookie = *cookie
			}
			return out, nil
		})

	api.Post(h, api.Op{ID: "request-invitation", Path: "/invitations/{token}/request", Summary: "Ask the inviter for a new invitation", Tags: inv, Access: api.Public},
		func(ctx context.Context, in *tokenPath) (*api.Body[InviteRequestResult], error) {
			r, err := s.RequestNewInvite(ctx, in.Token)
			return api.OK(r), err
		})

	me := []string{"Account"}
	api.Get(h, api.Op{ID: "get-me", Path: "/me", Summary: "The signed-in person, their workspaces and the instance", Tags: me, Access: api.Pending},
		func(ctx context.Context, _ *struct{}) (*api.Body[Me], error) {
			r, err := s.Me(ctx)
			return api.OK(r), err
		})

	api.Get(h, api.Op{ID: "get-workspace-me", Path: "/workspaces/{workspace}/me", Summary: "Your role, permissions and scope in a workspace", Tags: me, Access: api.Workspace("")},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WorkspaceMe], error) {
			r, err := s.WorkspaceMe(ctx)
			return api.OK(r), err
		})

	api.Get(h, api.Op{ID: "get-account", Path: "/me/account", Summary: "Account & security: profile, sign-in methods, two-factor, workspaces", Tags: me, Access: person},
		func(ctx context.Context, _ *struct{}) (*api.Body[AccountOverview], error) {
			r, err := s.Overview(ctx)
			return api.OK(r), err
		})

	api.Patch(h, api.Op{ID: "update-profile", Path: "/me/profile", Summary: "Change your name and timezone", Tags: me, Access: person},
		func(ctx context.Context, in *struct{ Body ProfileInput }) (*api.Body[AccountProfile], error) {
			r, err := s.UpdateProfile(ctx, in.Body)
			return api.OK(r), err
		})

	api.Put(h, api.Op{ID: "set-avatar", Path: "/me/avatar", Summary: "Upload a profile photo (PNG or JPG, up to 2 MB)", Tags: me, Access: person, MaxBodyBytes: 3 << 20},
		func(ctx context.Context, in *struct{ Body AvatarInput }) (*api.Body[AccountProfile], error) {
			r, err := s.SetAvatar(ctx, in.Body.Data)
			return api.OK(r), err
		})

	api.Delete(h, api.Op{ID: "remove-avatar", Path: "/me/avatar", Summary: "Remove your profile photo", Tags: me, Access: person},
		func(ctx context.Context, _ *struct{}) (*api.Empty, error) {
			return api.NoContent, s.RemoveAvatar(ctx)
		})

	api.Get(h, api.Op{ID: "get-avatar", Path: "/users/{id}/avatar", Summary: "A person's profile photo", Tags: me, Access: api.Authenticated},
		func(ctx context.Context, in *idPath) (*AvatarImage, error) {
			id, err := parseID(in.ID, "Photo")
			if err != nil {
				return nil, err
			}
			return s.Avatar(ctx, id)
		})

	api.Post(h, api.Op{ID: "request-email-change", Path: "/me/email", Summary: "Change your email: a verification link goes to the new address", Tags: me, Access: person},
		func(ctx context.Context, in *struct{ Body EmailChangeInput }) (*api.Body[EmailChangeStarted], error) {
			r, err := s.RequestEmailChange(ctx, in.Body)
			return api.OK(r), err
		})

	api.Delete(h, api.Op{ID: "cancel-email-change", Path: "/me/email", Summary: "Cancel a pending email change", Tags: me, Access: person},
		func(ctx context.Context, _ *struct{}) (*api.Empty, error) {
			return api.NoContent, s.CancelEmailChange(ctx)
		})

	api.Post(h, api.Op{ID: "confirm-email-change", Path: "/me/email/verify", Summary: "Confirm your new email with the link from the verification email", Tags: me, Access: pending},
		func(ctx context.Context, in *struct {
			Body struct {
				Token string `json:"token" minLength:"10" maxLength:"200"`
			}
		}) (*api.Body[EmailChanged], error) {
			r, err := s.ConfirmEmailChange(ctx, in.Body.Token)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "change-password", Path: "/me/password", Summary: "Change your password (or replace the temporary one)", Tags: me, Access: api.Pending, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			Body struct {
				Current  string `json:"current,omitempty" maxLength:"1024"`
				Password string `json:"password" minLength:"1" maxLength:"1024"`
			}
		}) (*api.Empty, error) {
			return api.NoContent, s.ChangePassword(ctx, in.Body.Current, in.Body.Password)
		})

	api.Post(h, api.Op{ID: "totp-begin", Path: "/me/totp", Summary: "Start setting up an authenticator app", Tags: me, Access: api.Pending},
		func(ctx context.Context, _ *struct{}) (*api.Body[TOTPSetup], error) {
			r, err := s.BeginTOTP(ctx)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "totp-confirm", Path: "/me/totp/confirm", Summary: "Confirm the authenticator with a code; returns recovery codes once", Tags: me, Access: api.Pending},
		func(ctx context.Context, in *struct {
			Body struct {
				Challenge string `json:"challenge" minLength:"10" maxLength:"200"`
				Code      string `json:"code" minLength:"6" maxLength:"16"`
			}
		}) (*api.Body[RecoveryCodes], error) {
			r, err := s.ConfirmTOTP(ctx, in.Body.Challenge, in.Body.Code)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "recovery-codes-regenerate", Path: "/me/recovery-codes", Summary: "Replace all recovery codes; the new ones are shown once", Tags: me, Access: api.Pending},
		func(ctx context.Context, _ *struct{}) (*api.Body[RecoveryCodes], error) {
			r, err := s.RegenerateRecoveryCodes(ctx)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "recovery-codes-saved", Path: "/me/recovery-codes/saved", Summary: "Confirm that you saved your recovery codes", Tags: me, Access: pending, Status: http.StatusNoContent},
		func(ctx context.Context, _ *struct{}) (*api.Empty, error) {
			return api.NoContent, s.MarkRecoveryCodesSaved(ctx)
		})

	api.Post(h, api.Op{ID: "passkey-register-options", Path: "/me/passkeys/options", Summary: "Start adding a passkey", Tags: me, Access: pending},
		func(ctx context.Context, _ *struct{}) (*api.Body[PasskeyOptions], error) {
			r, err := s.BeginPasskeyRegistration(ctx)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "passkey-register", Path: "/me/passkeys", Summary: "Add a passkey with the browser's answer", Tags: me, Access: pending, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			Body struct {
				Challenge  string          `json:"challenge" minLength:"10" maxLength:"200"`
				Credential json.RawMessage `json:"credential" doc:"The browser's RegistrationResponseJSON"`
				Name       string          `json:"name,omitempty" maxLength:"80"`
			}
		}) (*api.Body[AccountPasskey], error) {
			r, err := s.FinishPasskeyRegistration(ctx, in.Body.Challenge, in.Body.Credential, in.Body.Name)
			return api.OK(r), err
		})

	api.Patch(h, api.Op{ID: "rename-passkey", Path: "/me/passkeys/{id}", Summary: "Rename a passkey", Tags: me, Access: person, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			idPath
			Body struct {
				Name string `json:"name" minLength:"1" maxLength:"80"`
			}
		}) (*api.Empty, error) {
			id, err := parseID(in.ID, "Passkey")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.RenamePasskey(ctx, id, in.Body.Name)
		})

	api.Delete(h, api.Op{ID: "remove-passkey", Path: "/me/passkeys/{id}", Summary: "Remove a passkey (not your last way to sign in)", Tags: me, Access: person},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			id, err := parseID(in.ID, "Passkey")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.RemovePasskey(ctx, id)
		})

	api.Delete(h, api.Op{ID: "unlink-identity", Path: "/me/identities/{id}", Summary: "Unlink a social or SSO sign-in (not your last way to sign in)", Tags: me, Access: person},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			id, err := parseID(in.ID, "Connected sign-in")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.UnlinkIdentity(ctx, id)
		})

	api.Get(h, api.Op{ID: "list-sessions", Path: "/me/sessions", Summary: "Your active sessions, this one first", Tags: me, Access: person},
		func(ctx context.Context, _ *struct{}) (*api.Body[SessionList], error) {
			r, err := s.Sessions(ctx)
			return api.OK(r), err
		})

	api.Delete(h, api.Op{ID: "revoke-session", Path: "/me/sessions/{id}", Summary: "Sign out another session", Tags: me, Access: person},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			id, err := parseID(in.ID, "Session")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.RevokeSession(ctx, id)
		})

	api.Post(h, api.Op{ID: "revoke-other-sessions", Path: "/me/sessions/revoke-others", Summary: "Sign out every other session", Tags: me, Access: person},
		func(ctx context.Context, _ *struct{}) (*api.Body[RevokedSessions], error) {
			r, err := s.RevokeOtherSessions(ctx)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "leave-workspace", Path: "/workspaces/{workspace}/leave", Summary: "Leave the workspace (the last Owner can't)", Tags: []string{"Workspaces"}, Access: api.WorkspaceHuman(""), Status: http.StatusNoContent},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Empty, error) {
			return api.NoContent, s.LeaveWorkspace(ctx)
		})

	setup := []string{"Setup"}
	adminPending := api.Access{Kind: api.AccessInstanceAdmin, AllowPending: true, HumansOnly: true}
	api.Get(h, api.Op{ID: "get-setup", Path: "/setup", Summary: "The first sign-in steps for you (and the instance, for its first Owner)", Tags: setup, Access: pending},
		func(ctx context.Context, _ *struct{}) (*api.Body[SetupState], error) {
			r, err := s.Setup(ctx)
			return api.OK(r), err
		})

	api.Put(h, api.Op{ID: "setup-instance", Path: "/setup/instance", Summary: "Save instance basics (first sign-in step 4)", Tags: setup, Access: adminPending, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct{ Body InstanceBasics }) (*api.Empty, error) {
			return api.NoContent, s.SaveInstanceBasics(ctx, in.Body)
		})

	api.Post(h, api.Op{ID: "setup-check-url", Path: "/setup/check-url", Summary: "Check the public URL's DNS, TLS and reachability", Tags: setup, Access: adminPending},
		func(ctx context.Context, in *struct {
			Body struct {
				URL string `json:"url" format:"uri" maxLength:"300"`
			}
		}) (*api.Body[URLCheck], error) {
			r, err := s.CheckPublicURL(ctx, in.Body.URL)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "create-workspace", Path: "/workspaces", Summary: "Create a workspace; you become its Owner", Tags: []string{"Workspaces"}, Access: api.Access{Kind: api.AccessUser, AllowPending: true, HumansOnly: true}, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			Body struct {
				Name     string `json:"name" minLength:"1" maxLength:"80"`
				Timezone string `json:"timezone,omitempty" maxLength:"64"`
			}
		}) (*api.Body[CreatedWorkspace], error) {
			r, err := s.CreateWorkspace(ctx, in.Body.Name, in.Body.Timezone)
			return api.OK(r), err
		})
}
