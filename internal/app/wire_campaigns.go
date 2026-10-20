package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/demo"
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
				Prober:    linkProber(a),
			},
			// Saves re-check the campaign's open approval.
			AfterChange: CampaignChangeHook(a),
			// ?language= reaches the campaign's language versions.
			Translations: campaignTranslations(a),
			// Test emails get their poll links and the contact's version.
			Extras: campaignExtras(a),
			// Other domains' checks (the Sponsor block's creative).
			More: campaignMore(a),
		}
		campaigns.Register(h, s)
		campaigns.RegisterComposer(h, s)
	})
}

// campaignExtras plugs polls and conditional content into previews and
// test sends; set from wire_visibility.go.
var campaignExtras = func(*App) campaigns.Extras { return nil }

// campaignMore adds other domains' findings to the composer's checks; set
// from wire_sponsors.go. Nil leaves them out.
var campaignMore = func(*App) campaigns.MoreChecks { return nil }

// linkProber requests the links of an email. In development and tests the
// demo's links answer offline, as in Send Readiness: their hosts are real
// domains the demo has no business calling.
func linkProber(a *App) contentcheck.Prober {
	var p contentcheck.Prober = contentcheck.NewHTTPProber()
	if a.Cfg != nil && (a.Cfg.IsDevelopment() || a.Cfg.Env == "test") {
		p = demo.ReadinessProberOverlay(p)
	}
	return p
}
