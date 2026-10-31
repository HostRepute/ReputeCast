package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		audience.Register(h, a.audience(), a.Secrets)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		audience.RegisterJobs(a.JobReg, a.audience(), a.Log)
	})
}

// Audience is the audience service (tests and tools reach it here).
func (a *App) Audience() *audience.Service { return a.audience() }

func (a *App) audience() *audience.Service {
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations without config
		d := config.Defaults()
		cfg = &d
	}
	files := cfg.FilesDir()
	return &audience.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Segments: a.segments(), Mail: a.Mail, Links: a.Links, URLs: a.URLs,
		Jobs: func() *jobs.Client { return a.Jobs }, FilesDir: files,
		// The team's opt-in emails are the forms' branded ones (review 2026-11 AU-15).
		ConfirmationEmail: func(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, language, link string) (mailer.Message, error) {
			return a.formsService().AppConfirmation(ctx, ws, contactID, language, link)
		},
		// Poll answers and survey responses, what the contact bought, and
		// their store orders and abandoned checkouts (X20).
		Answers: audience.ActivitySources{audienceAnswers(a), audiencePurchases(a), audienceStores(a)}, SourceNames: audienceSourceNames(a),
		Erasure: erasureRegistry()}
}

// audiencePurchases adds what a contact bought to the timeline; the
// products domain sets it from its wire file.
var audiencePurchases = func(*App) audience.ActivitySource { return nil }

// audienceSourceNames names Grow's sources in the contacts list and the
// profile; Grow sets it from its wire file.
var audienceSourceNames = func(*App) audience.SourceNamer { return nil }

// audienceAnswers adds poll answers and survey responses to the contact
// timeline; the polls domain sets it from its wire file.
var audienceAnswers = func(*App) audience.ActivitySource { return nil }

// audienceStores adds store orders and abandoned checkouts; the commerce
// domain sets it from its wire file.
var audienceStores = func(*App) audience.ActivitySource { return nil }
