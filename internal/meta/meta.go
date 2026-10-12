// Package meta serves instance information the web app needs before sign-in:
// the instance name and logo, which sign-in methods are enabled and the version.
package meta

import (
	"context"
	"encoding/json"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// SSOOption is an enabled SSO provider shown on the sign-in page.
type SSOOption struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Preset string `json:"preset" enum:"google,entra,okta,keycloak,authentik,oidc"`
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
	}
	if st.LogoSvg != nil {
		info.LogoSVG = *st.LogoSvg
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
		if social[k].Enabled {
			info.Social = append(info.Social, k)
		}
	}
	return info, nil
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
