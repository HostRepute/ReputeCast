package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/smtp"
	"github.com/hostrepute/reputecast/internal/providers/smtp/bouncebox"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// RetentionDays is how long raw payloads and the event log are kept.
const RetentionDays = 90

// SweepArgs re-queues payloads whose normalization never finished and
// purges raw payloads past retention.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "ingest.sweep" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

// SweepWorker runs SweepArgs.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	N    *Normalizer
	Jobs func() *jobs.Client
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	now := w.N.now()
	ids, err := w.N.Q.IngestStaleRaw(ctx, store.IngestStaleRawParams{Before: now.Add(-2 * time.Minute), Lim: 500})
	if err != nil {
		return err
	}
	if c := w.Jobs(); c != nil && len(ids) > 0 {
		params := make([]river.InsertManyParams, 0, len(ids))
		for _, id := range ids {
			params = append(params, river.InsertManyParams{Args: NormalizeArgs{RawEventID: id}})
		}
		if _, err := c.InsertMany(ctx, params); err != nil {
			return err
		}
	}
	_, err = w.N.Q.IngestPurgeRaw(ctx, now.AddDate(0, 0, -RetentionDays))
	return err
}

// MailboxArgs polls the bounce mailboxes of SMTP providers.
type MailboxArgs struct{}

// Kind implements river.JobArgs.
func (MailboxArgs) Kind() string { return "ingest.bounce_mailbox" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (MailboxArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// MailboxWorker reads bounce reports from every SMTP provider's mailbox.
type MailboxWorker struct {
	river.WorkerDefaults[MailboxArgs]
	N    *Normalizer
	Jobs func() *jobs.Client
	// Open overrides bouncebox.Open (tests).
	Open func(ctx context.Context, c bouncebox.Config) (bouncebox.Box, error)
	// OnPoll is told how each read of a bounce mailbox went, so a mailbox
	// that can't be read reaches the provider's health (PR-10).
	OnPoll func(ctx context.Context, p store.Provider, err error)
}

// Timeout bounds one poll.
func (w *MailboxWorker) Timeout(*river.Job[MailboxArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *MailboxWorker) Work(ctx context.Context, _ *river.Job[MailboxArgs]) error {
	provs, err := w.N.Q.ProvListAllActive(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range provs {
		if p.Adapter != smtp.Adapter || manage.IsDemo(p) {
			continue
		}
		if _, err := w.Poll(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
		}
	}
	return errors.Join(errs...)
}

// Poll reads up to 200 new reports from a provider's bounce mailbox, stores
// each as a raw event and queues its normalization.
func (w *MailboxWorker) Poll(ctx context.Context, p store.Provider) (n int, err error) {
	prov, err := w.N.Factory.Open(ctx, p)
	if err != nil {
		return 0, err
	}
	sp, ok := prov.(*smtp.Provider)
	if !ok || sp.BounceMailbox() == nil {
		return 0, nil
	}
	if w.OnPoll != nil {
		defer func() { w.OnPoll(ctx, p, err) }()
	}
	open := w.Open
	if open == nil {
		open = bouncebox.Open
	}
	box, err := open(ctx, *sp.BounceMailbox())
	if err != nil {
		return 0, err
	}
	defer func() { _ = box.Close() }()
	msgs, err := box.Fetch(ctx, 200)
	if err != nil {
		return 0, err
	}
	acked := make([]string, 0, len(msgs))
	err = db.Tx(ctx, w.N.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		hdr, _ := json.Marshal(map[string]string{"X-ReputeCast-Mailbox": sp.BounceMailbox().Address, "X-ReputeCast-Protocol": sp.BounceMailbox().Label()})
		for _, m := range msgs {
			id := newID()
			if err := q.IngestInsertRaw(ctx, store.IngestInsertRawParams{ID: id, WorkspaceID: p.WorkspaceID, ProviderID: p.ID, ReceivedAt: w.N.now(),
				ContentType: "message/rfc822", Body: m.Raw, Headers: hdr, SignatureOk: true, Source: "bounce_mailbox"}); err != nil {
				return err
			}
			if c := w.Jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, NormalizeArgs{RawEventID: id}, nil); err != nil {
					return err
				}
			}
			acked = append(acked, m.ID)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(acked), box.Ack(ctx, acked)
}

// BackfillWorker pulls message status from a provider's API for messages
// that got no webhook event (manage.BackfillArgs).
type BackfillWorker struct {
	river.WorkerDefaults[manage.BackfillArgs]
	N *Normalizer
}

// Timeout bounds one backfill.
func (w *BackfillWorker) Timeout(*river.Job[manage.BackfillArgs]) time.Duration {
	return 10 * time.Minute
}

// Work implements river.Worker.
func (w *BackfillWorker) Work(ctx context.Context, job *river.Job[manage.BackfillArgs]) error {
	_, err := w.N.Backfill(ctx, job.Args.ProviderID)
	return err
}

// Backfill looks up messages accepted since the webhooks went silent (or in
// the last week) and records what the provider reports about them.
func (n *Normalizer) Backfill(ctx context.Context, providerID uuid.UUID) (int, error) {
	p, err := n.Q.ProvGetAny(ctx, providerID)
	if err != nil {
		return 0, err
	}
	prov, err := n.Factory.Open(ctx, p)
	if err != nil {
		return 0, err
	}
	bf, ok := prov.(providers.EventBackfiller)
	if !ok {
		return 0, nil
	}
	var cfg struct {
		Events string `json:"events"`
	}
	_ = json.Unmarshal(p.Config, &cfg)
	polled := cfg.Events == "polling"
	since := n.now().AddDate(0, 0, -7)
	if h := state.HealthOf(p); h.Webhooks != nil && h.Webhooks.SilentSince != nil {
		since = *h.Webhooks.SilentSince
	}
	total := 0
	for round := 0; round < 20; round++ {
		ids, err := n.Q.ProvAwaitingProviderIDs(ctx, store.ProvAwaitingProviderIDsParams{ProviderID: &p.ID, Since: &since, Lim: 100})
		if err != nil {
			return total, err
		}
		if len(ids) == 0 {
			break
		}
		events, err := bf.Backfill(ctx, ids)
		if err != nil {
			return total, err
		}
		if len(events) == 0 {
			break
		}
		suppressed := 0
		err = db.Tx(ctx, n.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			now := n.now()
			// A provider whose events come by polling ("Use polling for
			// now") gets its freshness from the poll (PR-6); "Check now" on a
			// silent webhook doesn't make its webhooks fresh.
			if polled {
				if err := q.ProvTouchLastEvent(ctx, store.ProvTouchLastEventParams{ID: p.ID, At: now}); err != nil {
					return err
				}
			}
			for _, ev := range events {
				rawID := newID()
				body := []byte(ev.Raw)
				if len(body) == 0 {
					body = []byte(`{}`)
				}
				if err := q.IngestInsertRaw(ctx, store.IngestInsertRawParams{ID: rawID, WorkspaceID: p.WorkspaceID, ProviderID: p.ID, ReceivedAt: now,
					ContentType: "application/json", Body: body, Headers: json.RawMessage(`{}`), SignatureOk: true, Source: "backfill"}); err != nil {
					return err
				}
				added, err := n.Apply(ctx, tx, p, &rawID, "backfill", now, ev)
				if err != nil {
					return err
				}
				suppressed += added
				if err := q.IngestMarkNormalized(ctx, store.IngestMarkNormalizedParams{ID: rawID, NormalizedAt: &now, Events: 1}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return total, err
		}
		total += len(events)
		if suppressed > 0 && n.OnSuppressed != nil {
			n.OnSuppressed(ctx, p.WorkspaceID)
		}
		if len(ids) < 100 {
			break
		}
	}
	return total, nil
}
