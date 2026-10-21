package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/agency"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/overview"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/team"
	"github.com/hostrepute/reputecast/internal/templates"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// Agency accounts (design expansion phase 8, M32): the Agency area, agency
// mode in Instance admin, and the hourly health and usage roll-up.
func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		agency.Register(h, agencyService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		agency.RegisterJobs(a.JobReg, agencyService(a), a.Log)
	})
}

// agencyChannels read SMS, push and WhatsApp usage from the channel
// message log (X21); a channel that writes nothing there reads zero.
var agencyChannels = func(*App) []agency.ChannelUsage { return agency.MessageChannels() }

func agencyService(a *App) *agency.Service {
	return &agency.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Deps: func() agency.Deps {
		_, dl := deliverabilityServices(a)
		ov := &overview.Service{Pool: a.Pool, Q: a.Q, Deliverability: dl, Providers: a.providersService(),
			Campaigns: &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit}, Audience: overview.AudienceCards{Service: a.audience(), Q: a.Q}}
		tm := &team.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Mail: a.Mail, PublicURL: func(context.Context) string { return a.URLs.Base() }}
		ws := &workspaces.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit}
		return agency.Deps{
			Health: func(ctx context.Context, w *store.Workspace) (agency.HealthScore, error) {
				h, err := ov.SenderHealth(ctx, w)
				return agency.HealthScore{State: h.State, Score: h.Score, Tone: h.Tone}, err
			},
			Invite: func(ctx context.Context, w *store.Workspace, p *authz.Principal, in team.TeamInviteInput) (team.TeamInviteResult, error) {
				return tm.Invite(ctx, w, p, in)
			},
			DeleteWorkspace: func(ctx context.Context, w *store.Workspace, confirm string) (time.Time, error) {
				res, err := ws.Delete(ctx, w, workspaces.WorkspaceDeleteInput{Confirm: confirm})
				return res.RestoreUntil, err
			},
			Templates:     &templates.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
			Transactional: transactionalService(a),
			Channels:      agencyChannels(a),
		}
	}}
}
