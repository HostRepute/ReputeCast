package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/demoworkspace"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/providers/eventqueue"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/suppressions"
	"github.com/hostrepute/reputecast/internal/warmup"

	// Built-in sending adapters register themselves.
	_ "github.com/hostrepute/reputecast/internal/providers/postal"
	_ "github.com/hostrepute/reputecast/internal/providers/postmark"
	_ "github.com/hostrepute/reputecast/internal/providers/ses"
	_ "github.com/hostrepute/reputecast/internal/providers/smtp"
)

// Sending providers & routing.
func (a *App) providerFactory() *manage.Factory {
	// Provider URLs come from workspace admins: every adapter call goes through
	// the SSRF guard (metadata and link-local addresses are always refused).
	// (The OpenAPI command registers operations on an App without config.)
	guard := netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks}
	return &manage.Factory{Box: a.Secrets, HTTP: guard.Client(30 * time.Second), Dialer: guard.Dialer(15 * time.Second)}
}

func (a *App) providersService() *manage.Service {
	return &manage.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Factory: a.providerFactory(), URLs: a.URLs, Log: a.Log,
		Jobs:    func() *jobs.Client { return a.Jobs },
		Sending: func() sending.Starter { return a.Sending },
		// Installed community adapters and the generic HTTP form (wire_remote.go).
		Community:    a.remoteRuntime(),
		HTTPMappings: httpMappings{},
	}
}

func (a *App) suppressionSync() *suppressions.Service {
	return &suppressions.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Factory: a.providerFactory(), Log: a.Log, Jobs: func() *jobs.Client { return a.Jobs }}
}

// normalizerHooks are told about the events applied to messages; domains
// add theirs from their own wire file (transactional mail's webhooks).
var normalizerHooks []func(*App) ingest.MessageHook

func (a *App) normalizer() *ingest.Normalizer {
	sync := a.suppressionSync()
	n := &ingest.Normalizer{Pool: a.Pool, Q: a.Q, Factory: a.providerFactory(), Log: a.Log,
		OnSuppressed: func(ctx context.Context, ws uuid.UUID) { sync.Enqueue(ctx, ws) },
		// Provider-reported opens and clicks count (wire_counters.go).
		Engagement: trackerEngagement{a}}
	for _, h := range normalizerHooks {
		n.Hooks = append(n.Hooks, h(a))
	}
	return n
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		manage.Register(h, a.providersService())
		routing.Register(h, &routing.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Mailbox: a.mailboxClassifier(),
			// Warm-up imports routing: its allowance comes in as functions.
			WarmupToday: func(ctx context.Context, id uuid.UUID) *state.WarmupToday {
				return warmup.Today(ctx, a.Q, id, time.Now(), nil)
			},
			WarmCap: func(ctx context.Context, id uuid.UUID, at time.Time) (routing.WarmCap, bool) {
				al, ok, err := warmup.AllowanceFor(ctx, a.Q, id, at, nil)
				if err != nil || !ok {
					return routing.WarmCap{}, false
				}
				return routing.WarmCap{Cap: al.Cap, Day: al.Day, Days: al.Days, Overflow: al.Overflow}, true
			}})
		demoworkspace.Register(h, &demoworkspace.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets})
		warmup.Register(h, &warmup.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
		suppressions.Register(h, a.suppressionSync())
		ingest.RegisterLog(h, &ingest.EventLog{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
	ingestRegistrars = append(ingestRegistrars, func(a *App, r chi.Router) {
		h := &ingest.Handler{Pool: a.Pool, Q: a.Q, Factory: a.providerFactory(), Log: a.Log, Jobs: func() *jobs.Client { return a.Jobs }}
		h.Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		svc := a.providersService()
		n := a.normalizer()
		jobsFn := func() *jobs.Client { return a.Jobs }
		river.AddWorker(a.JobReg.Workers, &manage.HealthWorker{S: svc})
		river.AddWorker(a.JobReg.Workers, &manage.TrackingWorker{S: svc})
		river.AddWorker(a.JobReg.Workers, &manage.MoveWebhooksWorker{S: svc, PublicURL: a.currentPublicURL})
		river.AddWorker(a.JobReg.Workers, &ingest.NormalizeWorker{N: n})
		river.AddWorker(a.JobReg.Workers, &ingest.SweepWorker{N: n, Jobs: jobsFn})
		river.AddWorker(a.JobReg.Workers, &ingest.MailboxWorker{N: n, Jobs: jobsFn, OnPoll: svc.RecordMailboxPoll})
		// SES events through an SQS queue ReputeCast polls (PR-4).
		river.AddWorker(a.JobReg.Workers, &eventqueue.Worker{Pool: a.Pool, Q: a.Q, Factory: a.providerFactory(), Jobs: jobsFn, Log: a.Log, OnPoll: svc.RecordQueuePoll})
		river.AddWorker(a.JobReg.Workers, &ingest.BackfillWorker{N: n})
		river.AddWorker(a.JobReg.Workers, &suppressions.SyncWorker{S: a.suppressionSync()})
		river.AddWorker(a.JobReg.Workers, &warmup.DailyWorker{S: &warmup.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit}, Log: a.Log,
			Skip: func(p store.Provider) bool { return manage.IsDemo(p) }})
		river.AddWorker(a.JobReg.Workers, &warmup.ResumeWorker{S: &warmup.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit}})
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return manage.HealthArgs{}, nil }, true)
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return ingest.MailboxArgs{}, nil }, false)
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return eventqueue.PollArgs{}, nil }, false)
		a.JobReg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return ingest.SweepArgs{}, nil }, false)
		a.JobReg.Every(15*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return suppressions.SyncArgs{}, nil }, false)
		a.JobReg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return warmup.DailyArgs{}, nil }, true)
		a.JobReg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return warmup.ResumeArgs{}, nil }, false)
		a.JobReg.Every(24*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return manage.TrackingArgs{}, nil }, true)
	})
}
