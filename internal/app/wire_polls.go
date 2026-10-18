package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/polls"
)

// Polls and surveys (design expansion, X11): the report's Poll section, the
// survey builder and results, the answer and survey pages (publicpages),
// and the contact's answers on their profile.

func (a *App) polls() *polls.Service {
	return &polls.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Links: a.Links, URLs: a.URLs, Forms: a.formsService(), Log: a.Log}
}

func init() {
	// The timeline reads answers only: no page links, so no forms service
	// (which is built with the audience service).
	audienceAnswers = func(a *App) audience.ActivitySource { return &polls.Service{Pool: a.Pool, Q: a.Q, Log: a.Log} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		polls.Register(h, a.polls())
	})
}
