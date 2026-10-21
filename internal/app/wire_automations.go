package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

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

// automationAccessibility checks the emails of a version before publishing;
// the accessibility domain replaces it from its wire file. Nil skips it.
var automationAccessibility = func(*App) automations.AccessibilityGate { return nil }

// automationLanguages gives emails their language versions; the content
// languages domain replaces it from its wire file.
var automationLanguages = func(*App) automations.EmailLanguages { return nil }

// automationStoreRevenue is the store orders credited to flows (the list's
// revenue column); the commerce domain sets it from its wire file.
var automationStoreRevenue = func(*App) func(context.Context, uuid.UUID, []uuid.UUID, time.Time) map[uuid.UUID]automations.AutomationRevenue {
	return nil
}

// automationStoreEmail writes a store starter's emails in the store's
// voice (the Abandoned cart starter); the commerce domain sets it.
var automationStoreEmail = func(*App) func(context.Context, uuid.UUID, uuid.UUID, string) (automations.EmailContent, bool) {
	return nil
}

func newAutomations(a *App) *automations.Service {
	return &automations.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
		Jobs: func() automations.Inserter {
			if a.Jobs == nil {
				return nil
			}
			return a.Jobs
		},
		Mailer:        automationMailer(a),
		Gate:          automationGate(a),
		Accessibility: automationAccessibility(a),
		Languages:     automationLanguages(a),
		StoreRevenue:  automationStoreRevenue(a),
		StoreEmail:    automationStoreEmail(a),
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
