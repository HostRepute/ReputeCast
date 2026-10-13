package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/templates"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		s := &campaigns.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit,
			Templates: &templates.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
			Mail:      func() mailer.Sender { return a.Mail },
			URLs:      a.URLs,
			Checker: &contentcheck.Checker{
				Blocklist: contentcheck.StoreBlocklist{Q: a.Q},
				Prober:    contentcheck.NewHTTPProber(),
			},
			// Saves re-check the campaign's open approval (W11).
			AfterChange: CampaignChangeHook(a),
		}
		campaigns.Register(h, s)
		campaigns.RegisterComposer(h, s)
	})
}
