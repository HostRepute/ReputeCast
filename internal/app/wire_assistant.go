package app

import (
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/jobs"
)

// The assistant's threads keep per-process state (the hub that wakes this
// process's streams), so each App has exactly one service.
var assistants sync.Map // *App → *assistant.Service

// Assistant is the AI assistant's threads (B3), on the AI service's
// models, budget and projections.
func (a *App) Assistant() *assistant.Service {
	if v, ok := assistants.Load(a); ok {
		return v.(*assistant.Service)
	}
	s := &assistant.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, AI: a.ai(), MCP: a.MCPClient(), Log: a.Log,
		Jobs: func() *jobs.Client { return a.Jobs },
		Hub:  &assistant.Hub{Pool: a.Pool, Log: a.Log}}
	v, _ := assistants.LoadOrStore(a, s)
	return v.(*assistant.Service)
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		assistant.Register(h, a.Assistant())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &assistant.TurnWorker{S: a.Assistant()})
		river.AddWorker(a.JobReg.Workers, &assistant.SweepWorker{S: a.Assistant()})
		river.AddWorker(a.JobReg.Workers, &assistant.RetentionWorker{S: a.Assistant()})
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return assistant.SweepArgs{}, nil }, true)
		a.JobReg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return assistant.RetentionArgs{}, nil }, true)
	})
	closeHooks = append(closeHooks, func(a *App) {
		if v, ok := assistants.LoadAndDelete(a); ok {
			v.(*assistant.Service).Hub.Close()
		}
	})
}
