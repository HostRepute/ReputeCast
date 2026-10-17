package scoring

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

// NightlyHour is when the nightly run happens in each workspace's
// timezone: with the tier job, at 03:00.
const NightlyHour = 3

// tickEvery is how often new activity is scored: within a minute of an
// open, a click, a form submission or an API event.
const tickEvery = 15 * time.Second

// tickBudget bounds one tick's work per workspace; the rest waits for the
// next tick.
const tickBudget = 10 * time.Second

// TickArgs scores new activity in every workspace with scoring.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "scoring.tick" }

// InsertOpts runs on the events queue, once per period. Ticks that
// overlap are harmless: the event copy is idempotent and each takes its
// own dirty contacts.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: tickEvery}}
}

// NightlyArgs applies decay, field conditions and the look-back in every
// workspace whose 03:00 run is due.
type NightlyArgs struct {
	// WorkspaceID forces a run for one workspace (tests).
	WorkspaceID uuid.UUID `json:"workspaceId,omitempty"`
}

// Kind implements river.JobArgs.
func (NightlyArgs) Kind() string { return "scoring.nightly" }

// InsertOpts runs on the maintenance queue, once per half hour at most.
func (NightlyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 30 * time.Minute}}
}

// RecalcArgs recalculates every contact of a workspace after a rule change.
type RecalcArgs struct {
	RecalcID uuid.UUID `json:"recalcId"`
}

// Kind implements river.JobArgs.
func (RecalcArgs) Kind() string { return "scoring.recalculate" }

// InsertOpts runs on the default queue; a failed attempt is retried twice.
func (RecalcArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type tickWorker struct {
	river.WorkerDefaults[TickArgs]
	s *Service
}

func (w *tickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.s.Tick(ctx) }

type nightlyWorker struct {
	river.WorkerDefaults[NightlyArgs]
	s *Service
}

func (w *nightlyWorker) Work(ctx context.Context, job *river.Job[NightlyArgs]) error {
	return w.s.RunNightly(ctx, job.Args.WorkspaceID)
}

func (w *nightlyWorker) Timeout(*river.Job[NightlyArgs]) time.Duration { return time.Hour }

type recalcWorker struct {
	river.WorkerDefaults[RecalcArgs]
	s *Service
}

func (w *recalcWorker) Work(ctx context.Context, job *river.Job[RecalcArgs]) error {
	err := w.s.Recalculate(ctx, job.Args.RecalcID)
	if err != nil && job.Attempt >= job.MaxAttempts {
		w.s.failRecalc(ctx, job.Args.RecalcID, err)
		return nil
	}
	return err
}

func (w *recalcWorker) Timeout(*river.Job[RecalcArgs]) time.Duration { return 6 * time.Hour }

// RegisterJobs adds the scoring workers and their schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &tickWorker{s: s})
	river.AddWorker(reg.Workers, &nightlyWorker{s: s})
	river.AddWorker(reg.Workers, &recalcWorker{s: s})
	reg.Every(tickEvery, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, true)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return NightlyArgs{}, nil }, true)
}

// Tick scores what happened since the last tick in every workspace.
func (s *Service) Tick(ctx context.Context) error {
	rows, err := s.Q.ScoringWorkspaces(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, w := range rows {
		if err := s.tickWorkspace(ctx, w); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", w.Slug, err))
		}
	}
	return errors.Join(errs...)
}

// TickWorkspace runs one tick for one workspace (tests and the demo).
func (s *Service) TickWorkspace(ctx context.Context, wsID uuid.UUID) error {
	rows, err := s.Q.ScoringWorkspaces(ctx)
	if err != nil {
		return err
	}
	for _, w := range rows {
		if w.WorkspaceID == wsID {
			return s.tickWorkspace(ctx, w)
		}
	}
	return nil
}

// maxWindow bounds what one tick reads, so a worker that was down catches
// up in steps.
const maxWindow = time.Hour

func (s *Service) tickWorkspace(ctx context.Context, w store.ScoringWorkspacesRow) error {
	if w.IngestedThrough == nil {
		return nil // the first recalculation backfills
	}
	deadline := time.Now().Add(tickBudget)
	to := s.now().Add(-2 * time.Second)
	from := w.IngestedThrough.Add(-ingestOverlap)
	if to.Sub(from) > maxWindow {
		to = from.Add(maxWindow)
	}
	if to.After(*w.IngestedThrough) {
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			ids, err := ingest(ctx, tx, w.WorkspaceID, from, to)
			if err != nil {
				return err
			}
			if len(ids) > 0 {
				ws := make([]uuid.UUID, len(ids))
				for i := range ws {
					ws[i] = w.WorkspaceID
				}
				if _, err := tx.Exec(ctx, `INSERT INTO scoring_dirty (contact_id, workspace_id)
					SELECT unnest($1::uuid[]), unnest($2::uuid[]) ON CONFLICT (contact_id) DO NOTHING`, ids, ws); err != nil {
					return err
				}
			}
			return store.New(tx).ScoringConfigSetIngested(ctx, store.ScoringConfigSetIngestedParams{WorkspaceID: w.WorkspaceID, IngestedThrough: &to})
		})
		if err != nil {
			return err
		}
	}
	st, err := s.load(ctx, s.Q, w.WorkspaceID, w.Timezone)
	if err != nil {
		return err
	}
	for time.Now().Before(deadline) {
		n, err := s.rescoreDirty(ctx, w.WorkspaceID, st)
		if err != nil || n < batchSize {
			return err
		}
	}
	return nil
}

// rescoreDirty takes a batch of dirty contacts and scores them in one
// transaction: a failure leaves them dirty for the next tick.
func (s *Service) rescoreDirty(ctx context.Context, wsID uuid.UUID, st *setup) (int, error) {
	eng := NewEngine(st.applied, st.loc)
	n := 0
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		ids, err := q.ScoringDirtyTake(ctx, store.ScoringDirtyTakeParams{WorkspaceID: wsID, Lim: batchSize})
		if err != nil || len(ids) == 0 {
			return err
		}
		n = len(ids)
		_, err = s.rescoreTx(ctx, tx, q, wsID, st, eng, ids, CauseActivity)
		return err
	})
	return n, err
}

// rescoreTx scores a batch inside tx and queues trigger events for the
// contacts that crossed a live threshold. It returns how many changed.
func (s *Service) rescoreTx(ctx context.Context, tx pgx.Tx, q *store.Queries, wsID uuid.UUID, st *setup, eng *Engine, ids []uuid.UUID, cause string) (int, error) {
	now := s.now()
	cs, err := compute(ctx, q, tx, eng, st.types, ids, now)
	if err != nil {
		return 0, err
	}
	old, err := q.ScoringScoresOf(ctx, ids)
	if err != nil {
		return 0, err
	}
	before := make(map[uuid.UUID]int, len(old))
	for _, o := range old {
		before[o.ContactID] = int(o.Score)
	}
	triggers, err := liveTriggers(ctx, q, wsID)
	if err != nil {
		return 0, err
	}
	changed := 0
	var crossings []crossing
	for _, c := range cs {
		from := before[c.contact]
		if from != c.score {
			changed++
		}
		if anyCrosses(triggers, from, c.score) {
			crossings = append(crossings, crossing{c.contact, from, c.score})
		}
	}
	if err := writeScores(ctx, tx, wsID, st.row.AppliedVersion, now, cs); err != nil {
		return 0, err
	}
	return changed, emitCrossings(ctx, q, wsID, cause, now, crossings)
}

// nightlyDue reports whether today's 03:00 run hasn't happened yet.
func nightlyDue(now time.Time, loc *time.Location, last *time.Time) bool {
	n := now.In(loc)
	runAt := time.Date(n.Year(), n.Month(), n.Day(), NightlyHour, 0, 0, 0, loc)
	if n.Before(runAt) {
		runAt = runAt.AddDate(0, 0, -1)
	}
	return last == nil || last.Before(runAt)
}

// RunNightly runs the nightly step in every workspace where it is due (or
// in one workspace, forced).
func (s *Service) RunNightly(ctx context.Context, only uuid.UUID) error {
	rows, err := s.Q.ScoringWorkspaces(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, w := range rows {
		if only != uuid.Nil && w.WorkspaceID != only {
			continue
		}
		if only == uuid.Nil && !nightlyDue(s.now(), location(w.Timezone), w.NightlyAt) {
			continue
		}
		if err := s.nightly(ctx, w); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", w.Slug, err))
		}
	}
	return errors.Join(errs...)
}

// NightlyResult says what a nightly run changed.
type NightlyResult struct {
	Rescored      int
	FieldsChanged int
	Pruned        int64
}

func (s *Service) nightly(ctx context.Context, w store.ScoringWorkspacesRow) error {
	_, err := s.NightlyWorkspace(ctx, w.WorkspaceID, w.Timezone)
	return err
}

// NightlyWorkspace applies decay steps and events leaving the look-back,
// re-evaluates field conditions for every contact, and prunes old events.
func (s *Service) NightlyWorkspace(ctx context.Context, wsID uuid.UUID, tz string) (NightlyResult, error) {
	var out NightlyResult
	st, err := s.load(ctx, s.Q, wsID, tz)
	if err != nil {
		return out, err
	}
	if st.row.AppliedVersion == 0 {
		return out, nil
	}
	eng := NewEngine(st.applied, st.loc)
	now := s.now()
	// Decay steps and events leaving the window: contacts whose next check is due.
	for {
		var n int
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			ids, err := q.ScoringDueChecks(ctx, store.ScoringDueChecksParams{WorkspaceID: wsID, At: now, Lim: batchSize})
			if err != nil || len(ids) == 0 {
				return err
			}
			n = len(ids)
			_, err = s.rescoreTx(ctx, tx, q, wsID, st, eng, ids, CauseDecay)
			return err
		})
		if err != nil {
			return out, err
		}
		out.Rescored += n
		if n < batchSize {
			break
		}
	}
	// Field conditions, for everyone at once.
	changed, err := s.refreshFields(ctx, wsID, st, now)
	if err != nil {
		return out, err
	}
	out.FieldsChanged = changed
	q := s.Q
	if out.Pruned, err = q.ScoringPruneEvents(ctx, store.ScoringPruneEventsParams{WorkspaceID: wsID, Before: now.AddDate(0, 0, -KeepDays)}); err != nil {
		return out, err
	}
	// Erased contacts keep no score or events.
	if _, err := s.Pool.Exec(ctx, `DELETE FROM contact_scores cs USING contacts c WHERE c.id = cs.contact_id AND c.workspace_id = $1 AND c.erased_at IS NOT NULL`, wsID); err != nil {
		return out, err
	}
	if _, err := s.Pool.Exec(ctx, `DELETE FROM score_events se USING contacts c WHERE c.id = se.contact_id AND c.workspace_id = $1 AND c.erased_at IS NOT NULL`, wsID); err != nil {
		return out, err
	}
	ran := now
	return out, q.ScoringConfigSetNightly(ctx, store.ScoringConfigSetNightlyParams{WorkspaceID: wsID, NightlyAt: &ran})
}

// refreshFields re-evaluates the field conditions of every contact in one
// statement and writes the scores that change.
func (s *Service) refreshFields(ctx context.Context, wsID uuid.UUID, st *setup, now time.Time) (int, error) {
	a := &sqlArgs{}
	a.add(wsID)
	a.add(st.row.AppliedVersion)
	a.add(now)
	expr := fieldsPartSQL(st.applied, st.types, a)
	sql := `WITH f AS (
		SELECT c.id, ` + expr + ` AS fp FROM contacts c WHERE c.workspace_id = $1 AND c.erased_at IS NULL
	), chg AS (
		SELECT f.id, f.fp, coalesce(cs.events_part, 0) AS ep, coalesce(cs.score, 0) AS old
		FROM f LEFT JOIN contact_scores cs ON cs.contact_id = f.id
		WHERE coalesce(cs.fields_part, 0) <> f.fp
	), up AS (
		INSERT INTO contact_scores (contact_id, workspace_id, score, events_part, fields_part, version, computed_at, changed_at)
		SELECT id, $1, least(100, greatest(0, ep + fp)), ep, fp, $2, $3, $3 FROM chg
		ON CONFLICT (contact_id) DO UPDATE
		SET fields_part = excluded.fields_part, score = excluded.score, computed_at = excluded.computed_at,
		    changed_at = CASE WHEN contact_scores.score <> excluded.score THEN excluded.changed_at ELSE contact_scores.changed_at END
		RETURNING contact_id, score
	)
	SELECT up.contact_id, chg.old, up.score FROM up JOIN chg ON chg.id = up.contact_id`
	changed := 0
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sql, a.args...)
		if err != nil {
			return err
		}
		type row struct {
			id       uuid.UUID
			old, new int
		}
		var rs []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.old, &r.new); err != nil {
				rows.Close()
				return err
			}
			rs = append(rs, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		q := store.New(tx)
		triggers, err := liveTriggers(ctx, q, wsID)
		if err != nil {
			return err
		}
		var crossings []crossing
		for _, r := range rs {
			if r.old != r.new {
				changed++
			}
			if anyCrosses(triggers, r.old, r.new) {
				crossings = append(crossings, crossing{r.id, r.old, r.new})
			}
		}
		return emitCrossings(ctx, q, wsID, CauseField, now, crossings)
	})
	return changed, err
}
