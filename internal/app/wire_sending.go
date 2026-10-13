package app

import (
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// W12 · Sending engine, live monitor and tracking.

// sendingReadiness is the send-time recheck. The readiness domain replaces
// it from its own wire file:
//
//	func init() {
//		sendingReadiness = func(a *App) sending.Readiness { return &readiness.Recheck{…} }
//	}
var sendingReadiness = func(*App) sending.Readiness { return sending.ReadyAlways{} }

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
		Automations: &automations.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log}}
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
