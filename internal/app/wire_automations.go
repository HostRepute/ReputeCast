package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/automations"
)

// automationMailer builds the mailer the automations engine uses for "Send
// email" steps. The default records each message as queued on a provider of
// the Automations route; the sending domain replaces it from its wire file.
var automationMailer = func(*App) automations.Mailer { return automations.QueueOnlyMailer{} }

// automationGate decides whether publishing an automation needs approval
// (add-on §3.8). The approvals domain replaces it from its own wire file the
// same way; the default publishes right away.
var automationGate = func(*App) automations.PublishGate { return automations.NoApproval{} }

func newAutomations(a *App) *automations.Service {
	return &automations.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
		Jobs: func() automations.Inserter {
			if a.Jobs == nil {
				return nil
			}
			return a.Jobs
		},
		Mailer: automationMailer(a),
		Gate:   automationGate(a),
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		automations.Register(h, newAutomations(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		automations.RegisterJobs(a.JobReg, newAutomations(a))
	})
}
