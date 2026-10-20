package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/mcpserver"
)

func (a *App) mcpserver() *mcpserver.Service {
	return &mcpserver.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Sending: a.Sending, Mail: a.Mail,
		Campaigns:     &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Extras: campaignExtras(a)},
		Languages:     languagesService(a),
		Accessibility: newAccessibility(a),
		// Agents list templates, render previews and read the log; they never send.
		Transactional: transactionalService(a),
		Growth:        a.growth(),
		Digests:       digestsService(a),
		Polls:         a.polls(),
		PaidMembers:   earnService(a),
		Placement:     a.placement(),
		Website:       websiteService(a),
		// Earn's revenue and sponsor bookings (X17).
		Revenue:   revenueService(a),
		Sponsors:  sponsorService(a),
		PublicURL: func(context.Context) string { return a.URLs.Base() },
		// Grow: results, referrers, the review queue (masked) and
		// recommendations; agents only read and propose.
		Grow:            a.referrals(),
		Recommendations: a.recommendations(),
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		mcpserver.Register(h, a.mcpserver())
	})
	// The MCP endpoint is an API surface for agents, served with the web role.
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		r.Handle(mcpserver.Path, a.mcpserver().Handler(a.Log))
	})
}
