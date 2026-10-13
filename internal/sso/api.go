package sso

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
)

// SSOStartQuery is the query of a start link.
type SSOStartQuery struct {
	LoginHint string `query:"login_hint" maxLength:"254" doc:"Email to pre-fill at the IdP"`
	Next      string `query:"next" maxLength:"1000" doc:"App path to open after signing in"`
	Invite    string `query:"invite" maxLength:"200" doc:"Invitation token: join its workspace with this identity"`
	Intent    string `query:"intent" enum:"signin,link" doc:"link adds the identity to the signed-in account"`
}

func (in SSOStartQuery) request() StartRequest {
	return StartRequest{LoginHint: strings.ToLower(strings.TrimSpace(in.LoginHint)), Next: in.Next, Invite: in.Invite, Intent: in.Intent}
}

// redirectOut is a browser navigation with cookies.
type redirectOut struct {
	Status    int
	Location  string        `header:"Location"`
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

func redirect(r Redirect) *redirectOut {
	return &redirectOut{Status: http.StatusFound, Location: r.Location, SetCookie: r.Cookies}
}

// SSOProviderPath is the {providerId} path parameter.
type SSOProviderPath struct {
	ProviderID string `path:"providerId" format:"uuid"`
}

func (p SSOProviderPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(p.ProviderID)
	if err != nil {
		return uuid.Nil, api.NotFound("SSO provider")
	}
	return id, nil
}

type testOut struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      SSOTestStart
}

// SSOSocialList is GET /admin/social.
type SSOSocialList struct {
	Items []SSOSocialProvider `json:"items"`
}

// Register adds the SSO and social sign-in operations.
func Register(h huma.API, s *Service) {
	auth := []string{"Sign-in"}
	api.Get(h, api.Op{ID: "sso-start", Path: "/auth/sso/{providerId}/start", Summary: "Start single sign-on: redirects to the identity provider", Tags: auth,
		Access: api.Public, Status: http.StatusFound},
		func(ctx context.Context, in *struct {
			SSOProviderPath
			SSOStartQuery
		}) (*redirectOut, error) {
			id, err := uuid.Parse(in.ProviderID)
			if err != nil {
				return redirect(s.errorRedirect(ctx, "Single sign-on", nil, "sso", &flowError{Code: "unknown_provider", Detail: "No such SSO provider."}, in.LoginHint)), nil
			}
			return redirect(s.StartSSO(ctx, id, in.request())), nil
		})

	api.Get(h, api.Op{ID: "social-start", Path: "/auth/social/{kind}/start", Summary: "Start Google, Microsoft or GitHub sign-in: redirects to the provider", Tags: auth,
		Access: api.Public, Status: http.StatusFound},
		func(ctx context.Context, in *struct {
			Kind string `path:"kind" enum:"google,microsoft,github"`
			SSOStartQuery
		}) (*redirectOut, error) {
			return redirect(s.StartSocial(ctx, in.Kind, in.request())), nil
		})

	tags := []string{"Instance admin · SSO"}
	api.Get(h, api.Op{ID: "list-sso-providers", Path: "/admin/sso", Summary: "SSO providers and the presets to add one", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[SSOProviderList], error) {
			out, err := s.List(ctx)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-sso-provider", Path: "/admin/sso", Summary: "Add an SSO provider from a preset (a draft until enabled)", Tags: tags, Access: api.InstanceAdmin, Status: http.StatusCreated},
		func(ctx context.Context, in *struct{ Body SSOCreateInput }) (*api.Body[SSOProviderDetail], error) {
			out, err := s.Create(ctx, in.Body)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-sso-provider", Path: "/admin/sso/{providerId}", Summary: "An SSO provider: connection, claims, access, test sign-in", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *SSOProviderPath) (*api.Body[SSOProviderDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, id)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-sso-provider", Path: "/admin/sso/{providerId}", Summary: "Save an SSO provider (the client secret is write-only)", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			SSOProviderPath
			Body SSOProviderInput
		}) (*api.Body[SSOProviderDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Update(ctx, id, in.Body)
			return api.OK(out), err
		})

	api.Delete(h, api.Op{ID: "delete-sso-provider", Path: "/admin/sso/{providerId}", Summary: "Delete a provider that isn't enabled", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *SSOProviderPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Delete(ctx, id)
		})

	api.Post(h, api.Op{ID: "test-sso-discovery", Path: "/admin/sso/{providerId}/discovery", Summary: "Fetch the issuer's discovery document and signing keys", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			SSOProviderPath
			Body SSODiscoveryInput
		}) (*api.Body[SSOProviderDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.TestDiscovery(ctx, id, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "start-sso-test", Path: "/admin/sso/{providerId}/test", Summary: "Start a test sign-in: shows the claims, signs nobody in, changes nothing", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			SSOProviderPath
			Body SSOTestInput
		}) (*testOut, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, cookie, err := s.StartTest(ctx, id, in.Body)
			if err != nil {
				return nil, err
			}
			res := &testOut{Body: out, SetCookie: []http.Cookie{}}
			if cookie != nil {
				res.SetCookie = append(res.SetCookie, *cookie)
			}
			return res, nil
		})

	api.Post(h, api.Op{ID: "enable-sso-provider", Path: "/admin/sso/{providerId}/enable", Summary: "Enable a provider: its button appears on the sign-in page", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *SSOProviderPath) (*api.Body[SSOProviderDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Enable(ctx, id)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "disable-sso-provider", Path: "/admin/sso/{providerId}/disable", Summary: "Disable a provider (its domain enforcement ends too)", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *SSOProviderPath) (*api.Body[SSOProviderDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Disable(ctx, id)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "preview-sso-enforcement", Path: "/admin/sso/{providerId}/enforcement", Summary: "Who signs in how before and after domain enforcement", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			SSOProviderPath
			Domain []string `query:"domain" maxItems:"20"`
		}) (*api.Body[SSOEnforcementPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.EnforcementPreview(ctx, id, in.Domain)
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "set-sso-enforcement", Path: "/admin/sso/{providerId}/enforcement", Summary: "Turn domain enforcement on (with break-glass Owners confirmed) or off", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			SSOProviderPath
			Body SSOEnforcementInput
		}) (*api.Body[SSOProviderDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetEnforcement(ctx, id, in.Body)
			return api.OK(out), err
		})

	social := []string{"Instance admin · Authentication"}
	api.Get(h, api.Op{ID: "list-social-providers", Path: "/admin/social", Summary: "Google, Microsoft and GitHub sign-in settings", Tags: social, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[SSOSocialList], error) {
			items, err := s.SocialProviders(ctx)
			return api.OK(SSOSocialList{Items: items}), err
		})

	api.Put(h, api.Op{ID: "update-social-provider", Path: "/admin/social/{kind}", Summary: "Configure a social sign-in provider (the secret is write-only)", Tags: social, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			Kind string `path:"kind" enum:"google,microsoft,github"`
			Body SSOSocialInput
		}) (*api.Body[SSOSocialList], error) {
			items, err := s.SaveSocial(ctx, in.Kind, in.Body)
			return api.OK(SSOSocialList{Items: items}), err
		})
}

// Routes adds the browser callbacks the identity providers redirect to.
// They sit outside /api because their URLs are registered at the IdP.
func (s *Service) Routes(r chi.Router) {
	r.Get("/auth/oidc/{slug}/callback", func(w http.ResponseWriter, req *http.Request) {
		write(w, req, s.CallbackSSO(req.Context(), chi.URLParam(req, "slug"), req.URL.Query(), req.Cookies()))
	})
	r.Get("/auth/social/{kind}/callback", func(w http.ResponseWriter, req *http.Request) {
		write(w, req, s.CallbackSocial(req.Context(), chi.URLParam(req, "kind"), req.URL.Query(), req.Cookies()))
	})
}

func write(w http.ResponseWriter, req *http.Request, r Redirect) {
	for i := range r.Cookies {
		http.SetCookie(w, &r.Cookies[i])
	}
	w.Header().Set("Cache-Control", "no-store")
	// The page that follows mustn't leak the code or state to other sites.
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, req, r.Location, http.StatusFound)
}
