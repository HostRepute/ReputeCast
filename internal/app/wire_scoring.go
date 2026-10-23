package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		scoring.Register(h, a.scoring(), a.segmentHasContact)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		scoring.RegisterJobs(a.JobReg, a.scoring())
	})
}

var scoringServices sync.Map // *App → *scoring.Service

// scoring returns the app's scoring service. The River client is created
// after the job registrars run, so the service reads it lazily.
func (a *App) scoring() *scoring.Service {
	if v, ok := scoringServices.Load(a); ok {
		return v.(*scoring.Service)
	}
	s := &scoring.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Jobs: func() *jobs.Client { return a.Jobs }, Log: a.Log}
	v, _ := scoringServices.LoadOrStore(a, s)
	return v.(*scoring.Service)
}

// Scoring is the scoring domain (tests set its clock).
func (a *App) Scoring() *scoring.Service { return a.scoring() }

// segmentHasContact answers the profile's "Sam is in it".
func (a *App) segmentHasContact(ctx context.Context, ws *store.Workspace, segment, contact uuid.UUID) (bool, error) {
	return a.segments().HasContact(ctx, ws.ID, segment, contact)
}
