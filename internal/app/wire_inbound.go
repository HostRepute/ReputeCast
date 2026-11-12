package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/inbound"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
)

// Receiving mail (M58): the inbound route, its hook, the jobs that read
// mailboxes and process what arrives, and Audience › Replies. Other
// domains read how a workspace receives through the functions below.

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		inbound.Register(h, a.inbound())
	})
	ingestRegistrars = append(ingestRegistrars, func(a *App, r chi.Router) {
		inbound.NewHook(a.inbound()).Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		s := a.inbound()
		river.AddWorker(a.JobReg.Workers, &inbound.ProcessWorker{S: s})
		river.AddWorker(a.JobReg.Workers, &inbound.MailboxWorker{S: s})
		river.AddWorker(a.JobReg.Workers, &inbound.SweepWorker{S: s})
		river.AddWorker(a.JobReg.Workers, &inbound.HealthWorker{S: s})
		river.AddWorker(a.JobReg.Workers, &inbound.NoticeWorker{S: s})
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return inbound.MailboxArgs{}, nil }, false)
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return inbound.NoticeArgs{}, nil }, false)
		a.JobReg.Every(15*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return inbound.HealthArgs{}, nil }, false)
		a.JobReg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return inbound.SweepArgs{}, nil }, false)
	})
	// A contact's replies on their timeline (the timeline reads only the
	// store: a.inbound() itself needs the audience service).
	audienceReplies = func(a *App) audience.ActivitySource { return &inbound.Service{Q: a.Q} }
}

// Inbound is the receiving service (tests drive its jobs).
func (a *App) Inbound() *inbound.Service { return a.inbound() }

// inboundServices keeps one service per App: it remembers MX lookups and
// when each workspace last sent a test message.
var inboundServices sync.Map // *App → *inbound.Service

func (a *App) inbound() *inbound.Service {
	if s, ok := inboundServices.Load(a); ok {
		return s.(*inbound.Service)
	}
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations on an App without config
		d := config.Defaults()
		cfg = &d
	}
	resolver := dnscheck.NewResolver(dnscheck.Options{Servers: cfg.DNS.Servers, Timeout: cfg.DNS.Timeout})
	switch {
	case cfg.IsDevelopment():
		// The demo's receiving domain answers from the demo zone.
		resolver = demo.SendersDNSOverlay(demo.DeliverabilityDNSOverlay(resolver), a.URLs.Base)
	case cfg.Env == "test":
		resolver = demo.SendersDNSOverlay(demo.DeliverabilityDNSOverlay(dnscheck.NewZone()), a.URLs.Base)
	}
	n := a.normalizer()
	_, dl := deliverabilityServices(a)
	s := &inbound.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Signer: a.Links, Factory: a.providerFactory(), Mail: a.Mail,
		Jobs: func() *jobs.Client { return a.Jobs },
		Base: a.URLs.Base, DNS: resolver, Log: a.Log, Now: a.now,
		// Mailbox routes' servers are hosts people enter.
		Guard:    netguard.Policy{AllowPrivate: cfg.Providers.AllowPrivateNetworks},
		Audience: a.audience(),
		MessageUnsubscribed: func(ctx context.Context, q *store.Queries, id uuid.UUID, at time.Time) error {
			return (&automations.Service{Q: a.Q, Log: a.Log}).RecordMessageEvent(ctx, q, automations.MessageEvent{MessageID: id, Type: "unsubscribed", At: at})
		},
		EndRunsOnReply: func(ctx context.Context, tx pgx.Tx, ws, automation, contact uuid.UUID, at time.Time) (int, error) {
			return (&automations.Service{Q: a.Q, Log: a.Log}).EndRunsOnReply(ctx, tx, ws, automation, contact, at)
		},
		StoreDMARC: func(ctx context.Context, tx pgx.Tx, ws *store.Workspace, data []byte) (string, error) {
			rep, err := deliverability.ParseReport(data)
			if err != nil {
				return "", inbound.ErrNotAReport
			}
			if err := dl.StoreReport(ctx, tx, ws, rep, "inbound"); err != nil {
				if errors.Is(err, deliverability.ErrNotOwnedDomain) {
					return rep.Domain, inbound.ErrNotOwned
				}
				return rep.Domain, err
			}
			return rep.Domain, nil
		},
		ApplyReport:  n.ApplyReport,
		OnSuppressed: n.OnSuppressed,
	}
	actual, _ := inboundServices.LoadOrStore(a, s)
	return actual.(*inbound.Service)
}

// inboundState reads how a workspace receives mail.
func (a *App) inboundState(ctx context.Context, workspaceID uuid.UUID) (inbound.State, error) {
	return inbound.StateOf(ctx, a.Q, workspaceID)
}

// inboundReceiving: whether the workspace's route receives (reply handling
// "inbox" needs it).
func (a *App) inboundReceiving(ctx context.Context, workspaceID uuid.UUID) bool {
	st, err := a.inboundState(ctx, workspaceID)
	return err == nil && st.Receiving
}

// inboundReportAddress is dmarc@ the receiving domain while a route exists.
func (a *App) inboundReportAddress(ctx context.Context, workspaceID uuid.UUID) (string, bool) {
	st, err := a.inboundState(ctx, workspaceID)
	if err != nil || !st.Route {
		return "", false
	}
	return st.Box().Fixed(inbound.LocalDMARC), true
}

// inboundDeliverability is what the DMARC card shows about reports
// arriving by email.
func (a *App) inboundDeliverability(ctx context.Context, workspaceID uuid.UUID) (bool, string, *time.Time, error) {
	st, err := a.inboundState(ctx, workspaceID)
	if err != nil || !st.Route {
		return false, "", nil, err
	}
	return st.Receiving, st.Domain, st.LastDMARC, nil
}

// inboundReadiness is what Send Readiness checks about receiving.
func (a *App) inboundReadiness(ctx context.Context, workspaceID uuid.UUID) (bool, bool, string, string, error) {
	st, err := a.inboundState(ctx, workspaceID)
	return st.Receiving, st.Inbox, st.Domain, st.Local, err
}

// inboundBox is how a send rewrites Reply-To and mailto: (nil: no route).
func (a *App) inboundBox(ctx context.Context, workspaceID uuid.UUID) (*engine.InboundBox, error) {
	st, err := a.inboundState(ctx, workspaceID)
	if err != nil || !st.Route {
		return nil, err
	}
	return &engine.InboundBox{Domain: st.Domain, Local: st.Local, Receiving: st.Receiving, Inbox: st.Inbox}, nil
}
