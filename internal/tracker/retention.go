package tracker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Retention, hourly: the coming months' partitions are made; events go
// after 90 days (whole months are dropped, the rest is deleted per site),
// or after 30 where the workspace keeps event detail 30 days; visitors
// never linked to a contact go 30 days after they were last seen, with
// their events and products; a contact's product interest goes 180 days
// after the last view or add-to-cart; refused origins after 7 days.

// Events is tracker_events as a monthly table. Rows its default partition
// holds for a month (the job didn't run for months) move into the month
// when it is made, so a month's rows never stay there.
var Events = db.MonthlyTable{Name: "tracker_events", Key: "occurred_at", Ahead: 2, Adopt: true}

// RetentionResult says what a run did.
type RetentionResult struct {
	Made, Dropped                         []string
	Events, Visitors, Interests, Refusals int64
}

// staleBatch is how many stale visitors go at a time.
const staleBatch = 1000

// Retain runs the retention once.
func (s *Service) Retain(ctx context.Context) (RetentionResult, error) {
	var out RetentionResult
	now := s.now()
	var err error
	if out.Made, err = db.EnsureMonthlyPartitions(ctx, s.Pool, Events, now, time.Time{}); err != nil {
		return out, err
	}
	cut := now.AddDate(0, 0, -EventDays)
	if out.Dropped, err = db.DropMonthlyPartitions(ctx, s.Pool, Events, cut); err != nil {
		return out, err
	}
	sites, err := s.Q.TrackerRetentionSites(ctx)
	if err != nil {
		return out, err
	}
	for _, st := range sites {
		n, err := s.Q.TrackerDeleteSiteEvents(ctx, store.TrackerDeleteSiteEventsParams{SiteID: st.ID, Before: now.AddDate(0, 0, -int(st.Days))})
		if err != nil {
			return out, err
		}
		out.Events += n
	}
	// The default partition should stay empty: what it holds past the 90
	// days goes, whichever site it was.
	n, err := s.Q.TrackerDeleteDefaultEvents(ctx, cut)
	if err != nil {
		return out, err
	}
	out.Events += n
	for {
		ids, err := s.Q.TrackerStaleVisitors(ctx, store.TrackerStaleVisitorsParams{Before: now.AddDate(0, 0, -AnonymousDays), Lim: staleBatch})
		if err != nil || len(ids) == 0 {
			if err != nil {
				return out, err
			}
			break
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			n, err := q.TrackerDeleteVisitorEvents(ctx, ids)
			if err != nil {
				return err
			}
			out.Events += n
			n, err = q.TrackerDeleteVisitors(ctx, ids)
			out.Visitors += n
			return err
		})
		if err != nil {
			return out, err
		}
		if len(ids) < staleBatch {
			break
		}
	}
	interest := now.AddDate(0, 0, -InterestDays)
	if out.Interests, err = s.Q.TrackerPurgeProductViews(ctx, &interest); err != nil {
		return out, err
	}
	out.Refusals, err = s.Q.TrackerPurgeRefusals(ctx, now.AddDate(0, 0, -RefusalDays))
	return out, err
}

// RetentionArgs runs the retention (hourly).
type RetentionArgs struct{}

// Kind implements river.JobArgs.
func (RetentionArgs) Kind() string { return "tracker.retention" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (RetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// RetentionWorker runs RetentionArgs.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) error {
	res, err := w.S.Retain(ctx)
	if err != nil {
		return err
	}
	if len(res.Made)+len(res.Dropped) > 0 || res.Events+res.Visitors+res.Interests+res.Refusals > 0 {
		w.S.log().InfoContext(ctx, "tracker retention", "made", res.Made, "dropped", res.Dropped, "events", res.Events, "visitors", res.Visitors,
			"interests", res.Interests, "refusals", res.Refusals)
	}
	return nil
}

// RegisterJobs adds the retention job (hourly, and at start so the coming
// months exist).
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &RetentionWorker{S: s})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return RetentionArgs{}, nil }, true)
}
