package app

import (
	"context"
	"encoding/json"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Content languages (M17): workspace languages and the language versions
// of emails.

// languagesService builds the content-languages service. Readers per
// language come from Send Readiness's audience count; changes to a
// campaign's versions re-check its open approval.
func languagesService(a *App) *languages.Service {
	return &languages.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit,
		AudienceLanguages: func(ctx context.Context, ws *store.Workspace, audience json.RawMessage) (map[string]int64, int64, error) {
			f, err := readinessService(a).Audience(ctx, ws, audience)
			if err != nil {
				return nil, 0, err
			}
			return f.Languages, f.Final, nil
		},
		Changed: CampaignChangeHook(a),
	}
}

// campaignTranslations hands the campaign content endpoints' ?language= to
// the content-languages service.
func campaignTranslations(a *App) languages.Campaigns {
	return languages.Campaigns{S: &languages.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit}}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		languages.Register(h, languagesService(a))
	})
}
