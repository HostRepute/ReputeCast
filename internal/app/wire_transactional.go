package app

import (
	"context"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Transactional email (design expansion phase 1).

// transactionalGate decides whether publishing a template needs an
// approval. The approvals domain replaces it with the optional workspace
// rule "Publishing a transactional template needs approval"; nil means
// publishing never needs one.
var transactionalGate = func(*App) transactional.PublishGate { return nil }

// transactionalAccessibility adds the Accessibility group to a template's
// checks; the accessibility domain replaces it from its wire file.
var transactionalAccessibility = func(*App) transactional.AccessibilityChecker { return nil }

// The service keeps per-process state (each key's send rate), so each App
// has exactly one; so does the outbox (the engine's provider call logs).
var (
	transactionalServices sync.Map // *App → *transactional.Service
	transactionalOutboxes sync.Map // *App → *engine.TransactionalOutbox
)

func transactionalService(a *App) *transactional.Service {
	if v, ok := transactionalServices.Load(a); ok {
		return v.(*transactional.Service)
	}
	s := &transactional.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Links: a.Links, URLs: a.URLs, Log: a.Log,
		Jobs: func() *jobs.Client { return a.Jobs },
		Checker: &contentcheck.Checker{
			Blocklist: contentcheck.StoreBlocklist{Q: a.Q},
			Prober:    linkProber(a),
		},
		Gate: transactionalGate(a),
		Mail: func() mailer.Sender { return a.Mail },
		// Queued messages wake the sending engine's transactional outbox in
		// the transaction that queued them.
		Outbox: func(ctx context.Context, tx pgx.Tx) error { return transactionalOutbox(a).Wake(ctx, tx) },
	}
	s.Accessibility = transactionalAccessibility(a)
	v, _ := transactionalServices.LoadOrStore(a, s)
	return v.(*transactional.Service)
}

func transactionalOutbox(a *App) *engine.TransactionalOutbox {
	if v, ok := transactionalOutboxes.Load(a); ok {
		return v.(*engine.TransactionalOutbox)
	}
	s := transactionalService(a)
	o := &engine.TransactionalOutbox{E: sendingEngine(a), Open: s.OpenPayload, Hooks: transactionalHooks{s: s}}
	if a.Cfg != nil && a.Cfg.IsDevelopment() {
		o.DevSMTP = a.Cfg.Mail.DevSMTP
	}
	v, _ := transactionalOutboxes.LoadOrStore(a, o)
	return v.(*engine.TransactionalOutbox)
}

// Transactional is the transactional domain (tests set its clock).
func (a *App) Transactional() *transactional.Service { return transactionalService(a) }

// TransactionalOutbox is the sending engine's transactional outbox (tests
// drive it with Step).
func (a *App) TransactionalOutbox() *engine.TransactionalOutbox { return transactionalOutbox(a) }

// transactionalHooks tells the transactional domain what the outbox did.
type transactionalHooks struct{ s *transactional.Service }

// TransactionalOutcome implements engine.TransactionalHooks.
func (h transactionalHooks) TransactionalOutcome(ctx context.Context, tx pgx.Tx, o engine.TransactionalOutcome) error {
	failed := map[uuid.UUID]string{}
	for _, f := range o.Failed {
		failed[f.ID] = f.Reason
	}
	return h.s.Outcome(ctx, tx, o.WorkspaceID, o.Sent, failed, o.At)
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		s := transactionalService(a)
		transactional.RegisterTemplates(h, s)
		transactional.RegisterSending(h, s)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		engine.RegisterTransactional(a.JobReg, transactionalOutbox(a))
	})
	// Provider events of transactional messages fire their message.* webhooks.
	normalizerHooks = append(normalizerHooks, func(a *App) ingest.MessageHook {
		s := transactionalService(a)
		return func(ctx context.Context, tx pgx.Tx, p store.Provider, id uuid.UUID, stream string, ev providers.Event) error {
			return s.ApplyEvent(ctx, tx, p, id, stream, ev)
		}
	})
}
