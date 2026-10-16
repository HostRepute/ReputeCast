// Package meta serves instance information the web app needs before sign-in:
// the instance name and logo, which sign-in methods are enabled and the version.
package meta

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// SSOOption is an enabled SSO provider shown on the sign-in page.
type SSOOption struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Preset string `json:"preset" enum:"google,entra,okta,keycloak,authentik,oidc"`
	// Domains are the email domains the provider signs in (allowed or enforced).
	Domains []string `json:"domains"`
	// StartURL starts the OIDC flow.
	StartURL string `json:"startUrl"`
}

// InstanceInfo is the public instance description.
type InstanceInfo struct {
	InstanceName  string      `json:"instanceName" example:"Acme ReputeCast"`
	LogoSVG       string      `json:"logoSvg,omitempty"`
	PublicURL     string      `json:"publicUrl" example:"https://reputecast.acme.io"`
	Version       string      `json:"version" example:"v1.4.2"`
	SetupComplete bool        `json:"setupComplete"`
	Password      bool        `json:"password"`
	Passkeys      bool        `json:"passkeys"`
	Social        []string    `json:"social" doc:"Enabled social sign-in providers" enum:"google,microsoft,github"`
	SSO           []SSOOption `json:"sso"`
	// The sign-in page reads in the instance default unless the browser
	// prefers a language ReputeCast has, or the visitor picks one.
	DefaultLanguage     string                   `json:"defaultLanguage,omitempty" example:"en"`
	DefaultRegionFormat string                   `json:"defaultRegionFormat,omitempty" example:"standard"`
	Languages           []applang.LanguageOption `json:"languages,omitempty"`
}

// Service reads the instance settings.
type Service struct {
	Q *store.Queries
}

// Get returns the public instance description.
func (s *Service) Get(ctx context.Context) (InstanceInfo, error) {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstanceInfo{}, err
	}
	info := InstanceInfo{
		InstanceName:  st.Name,
		PublicURL:     st.PublicUrl,
		Version:       version.Get().Version,
		SetupComplete: st.SetupCompletedAt != nil,
		Social:        []string{},
		SSO:           []SSOOption{},

		DefaultLanguage: st.DefaultLanguage, DefaultRegionFormat: st.DefaultRegionFormat, Languages: applang.Languages(),
	}
	if st.LogoSvg != nil {
		info.LogoSVG = *st.LogoSvg
	}
	if len(st.AllowedMethods) == 0 {
		info.Password, info.Passkeys = true, true
	}
	for _, m := range st.AllowedMethods {
		switch m {
		case "password":
			info.Password = true
		case "passkey":
			info.Passkeys = true
		}
	}
	var social map[string]struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.Unmarshal(st.SocialProviders, &social)
	for _, k := range []string{"google", "microsoft", "github"} {
		if social[k].Enabled && allowed(st.AllowedMethods, "social") {
			info.Social = append(info.Social, k)
		}
	}
	if allowed(st.AllowedMethods, "sso") {
		ssos, err := s.Q.EnabledSSOProviders(ctx)
		if err != nil {
			return InstanceInfo{}, err
		}
		for _, p := range ssos {
			domains := append([]string{}, p.AllowedDomains...)
			for _, d := range p.EnforceDomains {
				if !slices.Contains(domains, d) {
					domains = append(domains, d)
				}
			}
			info.SSO = append(info.SSO, SSOOption{ID: p.ID.String(), Name: p.Name, Preset: p.Preset, Domains: domains, StartURL: "/api/v1/auth/sso/" + p.ID.String() + "/start"})
		}
	}
	return info, nil
}

func allowed(methods []string, m string) bool {
	return len(methods) == 0 || slices.Contains(methods, m)
}

// Register adds GET /api/v1/meta.
func Register(h huma.API, s *Service) {
	api.Get(h, api.Op{
		ID:      "get-meta",
		Path:    "/meta",
		Summary: "Instance information for the sign-in page",
		Tags:    []string{"Meta"},
		Access:  api.Public,
	}, func(ctx context.Context, _ *struct{}) (*api.Body[InstanceInfo], error) {
		info, err := s.Get(ctx)
		if err != nil {
			return nil, err
		}
		return api.OK(info), nil
	})
}
