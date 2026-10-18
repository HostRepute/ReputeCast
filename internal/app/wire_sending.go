package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// Sending engine, live monitor and tracking.

// sendingReadiness is the send-time recheck (SendRecheck).
var sendingReadiness = func(a *App) sending.Readiness {
	return &readinessRecheck{build: func() recheckFunc { return SendRecheck(a) }}
}

type recheckFunc = func(ctx context.Context, campaignID uuid.UUID) (approvals.SendRecheck, error)

// readinessRecheck adapts SendRecheck to sending.Readiness. The approvals
// service is built at the first recheck, not with the engine: its
// automations service sends through this engine, so building it while the
// engine is being built would never end.
type readinessRecheck struct {
	build func() recheckFunc
	once  sync.Once
	run   recheckFunc
}

// Recheck implements sending.Readiness.
func (r *readinessRecheck) Recheck(ctx context.Context, campaignID uuid.UUID) (sending.Recheck, error) {
	r.once.Do(func() { r.run = r.build() })
	rc, err := r.run(ctx, campaignID)
	if err != nil {
		return sending.Recheck{}, err
	}
	return sending.Recheck{Verdict: rc.Verdict, Blocking: rc.Blocking, Throttles: rc.Throttles, Audience: rc.Audience, RunID: rc.RunID}, nil
}

// sendingConditional decides conditional content and fills poll links;
// set from wire_visibility.go. Nil sends every block.
var sendingConditional = func(*App) engine.Conditional { return nil }

// sendingSegments resolves segment audiences: the audience domain's
// segments service (dynamic segments are evaluated when the send starts,
// static ones use their snapshot).
var sendingSegments = func(a *App) sending.Segments { return a.segments() }

// The engine keeps per-process state (provider call logs, live hubs,
// parsed templates), so each App has exactly one.
var (
	sendingEngines sync.Map // *App → *engine.Engine
	trackers       sync.Map // *App → *tracking.Service
)

func sendingTracker(a *App) *tracking.Service {
	if v, ok := trackers.Load(a); ok {
		return v.(*tracking.Service)
	}
	t := &tracking.Service{Pool: a.Pool, Q: a.Q, Links: a.Links, Log: a.Log,
		Automations:   &automations.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log},
		Transactional: transactionalService(a),
		Placement:     a.placement()}
	v, _ := trackers.LoadOrStore(a, t)
	return v.(*tracking.Service)
}

// SendingEngine returns the App's sending engine (the demo command drives it).
func (a *App) SendingEngine() *engine.Engine { return sendingEngine(a) }

func sendingEngine(a *App) *engine.Engine {
	if v, ok := sendingEngines.Load(a); ok {
		return v.(*engine.Engine)
	}
	e := &engine.Engine{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Factory: a.providerFactory(), Links: a.Links, URLs: a.URLs, Log: a.Log,
		Jobs:        func() *jobs.Client { return a.Jobs },
		Readiness:   sendingReadiness(a),
		Segments:    sendingSegments(a),
		Normalizer:  a.normalizer(),
		Recorder:    sendingTracker(a),
		Automations: &automations.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log},
		// Each reader gets the language version for them.
		Languages: &languages.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		// Each reader gets their version of conditional content, and their
		// own poll and survey links.
		Conditional: sendingConditional(a),
	}
	v, _ := sendingEngines.LoadOrStore(a, e)
	return v.(*engine.Engine)
}

func init() {
	startHooks = append(startHooks, func(a *App) error {
		a.Sending = sendingEngine(a)
		return nil
	})
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		engine.Register(h, sendingEngine(a))
	})
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		sendingTracker(a).Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		engine.RegisterJobs(a.JobReg, sendingEngine(a))
	})
	// The automations engine's "Send email" step goes through the sending
	// engine (routing, failover, rendering, tracking, delivery).
	automationMailer = func(a *App) automations.Mailer { return engine.AutomationMailer{E: sendingEngine(a)} }
}
