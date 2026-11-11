package app

import (
	"context"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/tracker"
)

// Store flows from the site tracker (M62, G-29–G-31): tracker.tick finds
// who enters the browse and cart flows every minute, and the store blocks
// whose products depend on the reader (recently viewed, bought together)
// read each page of readers' products at once. Back in stock and price
// drops are commerce's (commerce.product_changed, wire_commerce.go).

func trackerFlows(a *App) *tracker.Flows {
	s := commerceService(a)
	return &tracker.Flows{Pool: a.Pool, Q: a.Q, Log: a.Log, Now: s.Now,
		Kick: func(ctx context.Context) { newAutomations(a).Kick(ctx) }}
}

func init() {
	jobRegistrars = append(jobRegistrars, func(a *App) {
		tracker.RegisterFlowJobs(a.JobReg, trackerFlows(a))
	})
	recipientDataRegistrars = append(recipientDataRegistrars, func(a *App) liquid.RecipientData { return commerceService(a).Recipients() })
}
