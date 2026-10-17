package growth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// TickArgs attributes new joins and departures in every workspace.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "growth.attribute" }

// InsertOpts runs on the events queue, once a minute.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: time.Minute}}
}

// RollupArgs adds up the days before today's sends per source.
type RollupArgs struct {
	WorkspaceID uuid.UUID `json:"workspaceId,omitempty"`
}

// Kind implements river.JobArgs.
func (RollupArgs) Kind() string { return "growth.rollup" }

// InsertOpts runs on the maintenance queue.
func (RollupArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 30 * time.Minute}}
}

type tickWorker struct {
	river.WorkerDefaults[TickArgs]
	s *Service
}

func (w *tickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.s.Tick(ctx) }

func (w *tickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 30 * time.Minute }

type rollupWorker struct {
	river.WorkerDefaults[RollupArgs]
	s *Service
}

func (w *rollupWorker) Work(ctx context.Context, job *river.Job[RollupArgs]) error {
	return w.s.RollupAll(ctx, job.Args.WorkspaceID)
}

func (w *rollupWorker) Timeout(*river.Job[RollupArgs]) time.Duration { return time.Hour }

// RegisterJobs adds the growth workers and their schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &tickWorker{s: s})
	river.AddWorker(reg.Workers, &rollupWorker{s: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, true)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return RollupArgs{}, nil }, true)
}

// Tick attributes every workspace's new joins and departures.
func (s *Service) Tick(ctx context.Context) error {
	rows, err := s.Q.GrowthWorkspaces(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, r := range rows {
		ws, err := s.Q.GetWorkspace(ctx, r.ID)
		if err != nil {
			s.log().Warn("growth: workspace not readable", "workspace", r.Slug, "error", err)
			continue
		}
		if err := s.Attribute(ctx, &ws); err != nil {
			s.log().Warn("growth: attribution failed", "workspace", r.Slug, "error", err)
			errs = append(errs, fmt.Errorf("%s: %w", r.Slug, err))
		}
	}
	return errors.Join(errs...)
}

// firstRollupDays is how far the first rollup of a workspace reads back.
const firstRollupDays = 90

// RollupAll writes the days not rolled up yet in every workspace (or one).
func (s *Service) RollupAll(ctx context.Context, only uuid.UUID) error {
	rows, err := s.Q.GrowthWorkspaces(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, r := range rows {
		if only != uuid.Nil && r.ID != only {
			continue
		}
		if err := s.Rollup(ctx, r.ID, r.Timezone, r.RolledUpThrough); err != nil {
			s.log().Warn("growth: rollup failed", "workspace", r.Slug, "error", err)
			errs = append(errs, fmt.Errorf("%s: %w", r.Slug, err))
		}
	}
	return errors.Join(errs...)
}

// rollupSQL adds up one day's sends: $1 workspace, $2 day, $3–$4 the day's
// bounds, $5 the workspace's sending providers.
const rollupSQL = `INSERT INTO growth_quality_daily (workspace_id, day, dim, key, sent, delivered, bounced, complained, human_opened)
SELECT $1, $2::date, d.dim, d.key, count(*), count(*) FILTER (WHERE x.delivered), count(*) FILTER (WHERE x.bounced),
       count(*) FILTER (WHERE x.complained), count(*) FILTER (WHERE x.opened)
FROM (
    SELECT (m.delivered_at IS NOT NULL OR m.status IN ('delivered', 'complained')) AS delivered,
           m.status = 'bounced' AS bounced, m.status = 'complained' AS complained, m.human_opened_at IS NOT NULL AS opened,
           coalesce(g.source_kind || CASE WHEN g.source_id <> '' THEN ':' || g.source_id ELSE '' END, 'unknown') AS skey,
           CASE WHEN g.contact_id IS NULL OR g.source_kind NOT IN ('form', 'subscribe_page') THEN NULL
                WHEN NOT g.captured THEN 'not_captured'
                WHEN g.utm_source = '' AND g.utm_medium = '' AND g.utm_campaign = '' THEN 'none'
                ELSE lower(g.utm_source || '/' || g.utm_medium || '/' || g.utm_campaign) END AS ukey
    FROM messages m
    LEFT JOIN contact_growth g ON g.contact_id = m.contact_id
    WHERE m.provider_id = ANY ($5::uuid[]) AND m.workspace_id = $1 AND m.stream IN ('broadcast', 'automation')
      AND m.sent_at >= $3 AND m.sent_at < $4
) x
CROSS JOIN LATERAL (VALUES ('workspace', ''), ('source', x.skey), ('utm', x.ukey)) AS d(dim, key)
WHERE d.key IS NOT NULL
GROUP BY d.dim, d.key`

// Rollup writes the days after through (or the last 90) up to yesterday.
func (s *Service) Rollup(ctx context.Context, wsID uuid.UUID, tz string, through *time.Time) error {
	loc := location(tz)
	l := s.now().In(loc)
	today := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	day := today.AddDate(0, 0, -firstRollupDays)
	if through != nil {
		t := through.In(time.UTC)
		day = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	}
	providers, err := s.Q.GrowthWorkspaceProviders(ctx, wsID)
	if err != nil {
		return err
	}
	if _, err := s.Q.GrowthStateEnsure(ctx, store.GrowthStateEnsureParams{WorkspaceID: wsID, At: s.now()}); err != nil {
		return err
	}
	for ; day.Before(today); day = day.AddDate(0, 0, 1) {
		d := day
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `DELETE FROM growth_quality_daily WHERE workspace_id = $1 AND day = $2::date`, wsID, d.Format("2006-01-02")); err != nil {
				return err
			}
			if len(providers) > 0 {
				if _, err := tx.Exec(ctx, rollupSQL, wsID, d.Format("2006-01-02"), d, d.AddDate(0, 0, 1), providers); err != nil {
					return err
				}
			}
			done := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
			return store.New(tx).GrowthStateSetRolledUp(ctx, store.GrowthStateSetRolledUpParams{WorkspaceID: wsID, RolledUpThrough: &done})
		})
		if err != nil {
			return err
		}
	}
	return nil
}
