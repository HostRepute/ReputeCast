package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/audience"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		audience.Register(h, a.audience(), a.Secrets)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		audience.RegisterJobs(a.JobReg, a.audience(), a.Log)
	})
}

func (a *App) audience() *audience.Service {
	return &audience.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Segments: a.segments(), Mail: a.Mail, Links: a.Links, URLs: a.URLs,
		Answers: audienceAnswers(a), SourceNames: audienceSourceNames(a)}
}

// audienceSourceNames names Grow's sources in the contacts list and the
// profile; Grow sets it from its wire file.
var audienceSourceNames = func(*App) audience.SourceNamer { return nil }

// audienceAnswers adds poll answers and survey responses to the contact
// timeline; the polls domain sets it from its wire file.
var audienceAnswers = func(*App) audience.ActivitySource { return nil }
