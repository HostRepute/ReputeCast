package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/recommendations"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

// Grow › Recommendations (M25): recommended newsletters, the partner
// profile and partner links with other ReputeCast instances (/partner/v1).

var (
	recommendationsMu       sync.Mutex
	recommendationsServices = map[*App]*recommendations.Service{}
)

func (a *App) recommendations() *recommendations.Service {
	recommendationsMu.Lock()
	defer recommendationsMu.Unlock()
	if s, ok := recommendationsServices[a]; ok {
		return s
	}
	s := &recommendations.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, Links: a.Links,
		Pages:    a.growPages,
		Instance: a.URLs.Base,
		// Partner instances and link pages go through the SSRF guard;
		// private addresses only where providers may use them.
		AllowPrivate: func() bool { return a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks },
		Audience:     a.audience(),
		Forms:        a.formsService(),
		Mail:         func() mailer.Sender { return a.Mail },
		Media:        &media.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		Jobs: func() recommendations.Inserter {
			if a.Jobs == nil {
				return nil
			}
			return a.Jobs
		},
		Referrals: a.referrals,
	}
	recommendationsServices[a] = s
	return s
}

// partnerCodePath is a partner code opened in a browser.
const partnerCodePath = "/p/{code:[2-9A-HJKMNP-Za-hjkmnp-z]{4}-[2-9A-HJKMNP-Za-hjkmnp-z]{4}-[2-9A-HJKMNP-Za-hjkmnp-z]{4}}"

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		recommendations.Register(h, a.recommendations())
		recommendations.RegisterOverview(h, a.recommendations())
	})
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		a.recommendations().Routes(r)
		r.Get(partnerCodePath, a.recommendations().ServeCodePage)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		recommendations.RegisterJobs(a.JobReg, a.recommendations())
	})
	signupHookRegistrars = append(signupHookRegistrars, func(a *App) forms.SignupHook {
		return recommendations.Hook{S: a.recommendations}
	})
	recipientDataRegistrars = append(recipientDataRegistrars, func(a *App) liquid.RecipientData { return a.recommendations() })
	// Delivered emails count their recommendations as shown.
	normalizerHooks = append(normalizerHooks, func(a *App) ingest.MessageHook {
		s := a.recommendations()
		return func(ctx context.Context, tx pgx.Tx, p store.Provider, id uuid.UUID, stream string, ev providers.Event) error {
			return s.MessageEvent(ctx, tx, p, id, stream, ev)
		}
	})
	referralPartnerEvents = func(a *App) func() referrals.PartnerEvents {
		return func() referrals.PartnerEvents { return a.recommendations() }
	}
}
