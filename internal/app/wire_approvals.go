package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// W11 · Review & send (Send Readiness) and send approvals.

// readinessService builds Send Readiness: the composer's content and link
// checks (links are requested; the demo's answer offline in development),
// deliverability's blocklists and authentication, routing and the audience.
func readinessService(a *App) *readiness.Service {
	var prober contentcheck.Prober = contentcheck.NewHTTPProber()
	if a.Cfg != nil && (a.Cfg.IsDevelopment() || a.Cfg.Env == "test") {
		// The sample emails' links answer offline (and tests never hit the network).
		prober = demo.ReadinessProberOverlay(prober)
	}
	cs := &campaigns.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit,
		Templates: &templates.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		Mail:      func() mailer.Sender { return a.Mail },
		URLs:      a.URLs,
		Checker:   &contentcheck.Checker{Blocklist: contentcheck.StoreBlocklist{Q: a.Q}, Prober: prober},
	}
	_, dl := deliverabilityServices(a)
	return &readiness.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, Campaigns: cs, Deliverability: dl, URLs: a.URLs, Segments: a.segments()}
}

// newApprovals builds the approvals service with its own Send Readiness.
func newApprovals(a *App) *approvals.Service {
	rs := readinessService(a)
	s := &approvals.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, Readiness: rs, Campaigns: rs.Campaigns,
		Sending: func() sending.Starter { return a.Sending },
		// Publishing an approved automation: the gate already passed.
		Automations: &automations.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
			Jobs: func() automations.Inserter {
				if a.Jobs == nil {
					return nil
				}
				return a.Jobs
			},
			Mailer: automationMailer(a),
			Gate:   automations.NoApproval{},
		},
	}
	// Fixes and send settings re-check open approvals; so do composer saves.
	rs.AfterChange = s.CampaignChanged
	rs.Campaigns.AfterChange = s.CampaignChanged
	return s
}

// CampaignChangeHook re-checks a campaign's open approval after the composer
// saved it (the composer's campaigns.Service.AfterChange).
func CampaignChangeHook(a *App) campaigns.ChangeHook {
	return func(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
		newApprovals(a).CampaignChanged(ctx, ws, id)
	}
}

// SendRecheck is the recheck the sending engine runs as a send starts
// (its sending.Readiness): Send Readiness again, and the approval must
// still cover the campaign. It records nothing: the engine holds the send
// and tells the author and approvers when the verdict is Blocked, and
// Review & send shows that hold with the run's blocking checks. The
// engine's wiring adapts it in one step:
//
//	sendingReadiness = func(a *App) sending.Readiness { return readinessRecheck{SendRecheck(a)} }
//
// where readinessRecheck.Recheck copies Verdict, Blocking and Throttles
// into a sending.Recheck.
func SendRecheck(a *App) func(ctx context.Context, campaignID uuid.UUID) (approvals.SendRecheck, error) {
	return newApprovals(a).SendRecheck
}

// HeldSendCheck is the send-time check the minute job runs for scheduled
// and held sends: it holds a send that turns Blocked or lost its approval
// (and tells people), and returns an error wrapping sending.ErrNotReady.
func HeldSendCheck(a *App) func(ctx context.Context, workspaceID, campaignID uuid.UUID) error {
	return newApprovals(a).CheckSendTime
}

func init() {
	automationGate = func(a *App) automations.PublishGate { return approvals.AutomationGate{S: newApprovals(a)} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		s := newApprovals(a)
		readiness.Register(h, s.Readiness)
		approvals.Register(h, s)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &approvals.TickWorker{S: newApprovals(a)})
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return approvals.TickArgs{}, nil }, false)
	})
}
