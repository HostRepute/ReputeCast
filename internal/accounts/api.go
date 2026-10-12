package accounts

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

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
			r, cookie, err := s.PasswordSignIn(ctx, in.Body.Email, in.Body.Password, in.Body.Remember)
			if err != nil {
				return nil, err
			}
			out := &signInOut{Body: r}
			if cookie != nil {
				out.SetCookie = *cookie
			}
			return out, nil
		})

	api.Post(h, api.Op{ID: "auth-two-factor", Path: "/auth/two-factor", Summary: "Finish signing in with an authenticator or recovery code", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *struct {
			Body struct {
				Challenge string `json:"challenge" minLength:"10" maxLength:"200"`
				Method    string `json:"method" enum:"totp,recovery"`
				Code      string `json:"code" minLength:"6" maxLength:"32"`
			}
		}) (*signInOut, error) {
			r, cookie, err := s.TwoFactor(ctx, in.Body.Challenge, in.Body.Method, in.Body.Code)
			if err != nil {
				return nil, err
			}
			out := &signInOut{Body: r}
			if cookie != nil {
				out.SetCookie = *cookie
			}
			return out, nil
		})

	api.Post(h, api.Op{ID: "auth-sign-out", Path: "/auth/sign-out", Summary: "Sign out of this session", Tags: tags, Access: api.Pending, Status: http.StatusNoContent},
		func(ctx context.Context, _ *struct{}) (*cookieOut, error) {
			return &cookieOut{SetCookie: s.SignOut(ctx)}, nil
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

	setup := []string{"Setup"}
	adminPending := api.Access{Kind: api.AccessInstanceAdmin, AllowPending: true, HumansOnly: true}
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
