package offers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// TickArgs is the periodic offers job: offers past their date or at their
// limit end, and their codes are turned off.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "offers.tick" }

// InsertOpts runs on the maintenance queue, once at a time.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

// TickWorker runs TickArgs.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Timeout bounds one run.
func (w *TickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 5 * time.Minute }

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.S.Tick(ctx) }

// RegisterJobs adds the offers job, every 5 minutes.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	reg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, true)
}

// Tick ends the live offers whose date passed or whose limit is reached.
// Ending is the system's, so it's audited without a person.
func (s *Service) Tick(ctx context.Context) error {
	rows, err := s.Q.OffersLive(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	for _, o := range rows {
		ws, err := s.Q.GetWorkspace(ctx, o.WorkspaceID)
		if err != nil {
			continue
		}
		r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: o.ID, WorkspaceID: o.WorkspaceID})
		if err != nil {
			continue
		}
		reason := ""
		switch {
		case r.EndsAt != nil && !r.EndsAt.After(now):
			reason = "date"
		case r.RedeemLimit != nil && s.results(ctx, &ws, r, parseStripe(r.Stripe)).Redemptions >= int64(*r.RedeemLimit):
			reason = "limit"
		}
		if reason == "" {
			continue
		}
		at := now
		if reason == "date" {
			at = *r.EndsAt
		}
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.OfferEnd(ctx, store.OfferEndParams{ID: r.ID, At: at, Reason: reason}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Actor: &audit.Actor{Type: authz.ActorSystem, Name: "ReputeCast"}, Action: "earn.offer_ended",
				TargetType: "earn_offer", TargetID: r.ID.String(), TargetName: r.Name, Before: map[string]any{"status": "live"},
				After: map[string]any{"status": "ended", "reason": reason}})
		}); err != nil {
			return err
		}
		s.deactivate(ctx, &ws, r)
	}
	return nil
}
