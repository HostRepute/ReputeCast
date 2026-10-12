package audience

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// TierRunHour is the local hour after which the nightly tier run happens.
const TierRunHour = 3

// TiersArgs is the hourly job that recomputes engagement tiers for every
// workspace whose nightly run (03:00 in its timezone) is due.
type TiersArgs struct {
	// WorkspaceID forces a run for one workspace (tests, "recompute now").
	WorkspaceID uuid.UUID `json:"workspaceId,omitempty"`
}

// Kind implements river.JobArgs.
func (TiersArgs) Kind() string { return "audience.tiers" }

// InsertOpts runs on the maintenance queue, one at a time.
func (TiersArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 30 * time.Minute}}
}

// TiersWorker runs TiersArgs.
type TiersWorker struct {
	river.WorkerDefaults[TiersArgs]
	Svc *Service
	Log *slog.Logger
}

// Work implements river.Worker.
func (w *TiersWorker) Work(ctx context.Context, job *river.Job[TiersArgs]) error {
	rows, err := w.Svc.Q.TierRunWorkspaces(ctx)
	if err != nil {
		return err
	}
	now := w.Svc.now()
	for _, r := range rows {
		forced := job.Args.WorkspaceID != uuid.Nil && job.Args.WorkspaceID == r.ID
		if job.Args.WorkspaceID != uuid.Nil && !forced {
			continue
		}
		if !forced && !tierRunDue(now, r.Timezone, r.TiersComputedAt) {
			continue
		}
		res, err := w.Svc.RecomputeTiers(ctx, r.ID)
		if err != nil {
			return fmt.Errorf("tiers for %s: %w", r.Slug, err)
		}
		if w.Log != nil {
			w.Log.InfoContext(ctx, "tiers recomputed", "workspace", r.Slug, "changed", res.Changed, "entered_sunset", res.EnteredSunset)
		}
	}
	return nil
}

// tierRunDue reports whether today's 03:00 run hasn't happened yet.
func tierRunDue(now time.Time, tz string, last *time.Time) bool {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	n := now.In(loc)
	runAt := time.Date(n.Year(), n.Month(), n.Day(), TierRunHour, 0, 0, 0, loc)
	if n.Before(runAt) {
		runAt = runAt.AddDate(0, 0, -1)
	}
	return last == nil || last.Before(runAt)
}

// RegisterJobs adds the tier worker and its hourly schedule.
func RegisterJobs(reg *jobs.Registry, s *Service, log *slog.Logger) {
	river.AddWorker(reg.Workers, &TiersWorker{Svc: s, Log: log})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return TiersArgs{}, nil }, true)
}

// TierRun reports what a run changed.
type TierRun struct {
	Changed           int64 `json:"changed"`
	EnteredSunset     int64 `json:"enteredSunset"`
	LeftSunset        int64 `json:"leftSunset"`
	StoppedBroadcasts int64 `json:"stoppedBroadcasts"`
	Erased            int64 `json:"erased"`
}

// RecomputeTiers assigns tiers from engagement under the workspace's policy,
// applies the sunset policy (tag for the re-engagement flow, stop
// Broadcasts, optional erasure), writes the daily rollups behind List
// health and refreshes segment counts.
func (s *Service) RecomputeTiers(ctx context.Context, wsID uuid.UUID) (TierRun, error) {
	var out TierRun
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return out, err
	}
	p := ParsePolicy(ws.SunsetPolicy)
	now := s.now()
	loc := workspaceLocation(&ws)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		reengage := p.Enabled && p.Reengage
		err := tx.QueryRow(ctx, fmt.Sprintf(`WITH calc AS (
				SELECT c.id, c.tier AS old, %s AS new FROM contacts c WHERE c.workspace_id = $1 AND c.erased_at IS NULL
			), upd AS (
				UPDATE contacts c SET tier = calc.new, tier_changed_at = $2,
				       sunset_at = CASE WHEN calc.new = 'sunset' THEN $2 END,
				       broadcasts_stopped_at = CASE WHEN calc.new = 'sunset' THEN c.broadcasts_stopped_at END,
				       tags = CASE WHEN calc.new = 'sunset' AND $3 AND NOT ('sunset' = ANY (c.tags)) THEN array_append(c.tags, 'sunset')
				                   WHEN calc.new <> 'sunset' THEN array_remove(c.tags, 'sunset') ELSE c.tags END
				FROM calc WHERE c.id = calc.id AND calc.old <> calc.new
				RETURNING calc.old, calc.new
			)
			SELECT count(*), count(*) FILTER (WHERE new = 'sunset'), count(*) FILTER (WHERE old = 'sunset') FROM upd`, tierExpr(p, "$2::timestamptz")),
			wsID, now, reengage).Scan(&out.Changed, &out.EnteredSunset, &out.LeftSunset)
		if err != nil {
			return err
		}
		if p.Enabled && p.StopBroadcasts {
			tag, err := tx.Exec(ctx, `UPDATE contacts SET broadcasts_stopped_at = $2 WHERE workspace_id = $1 AND erased_at IS NULL AND tier = 'sunset'
				AND broadcasts_stopped_at IS NULL AND sunset_at <= $2::timestamptz - make_interval(days => $3)`, wsID, now, p.StopAfterDays)
			if err != nil {
				return err
			}
			out.StoppedBroadcasts = tag.RowsAffected()
		}
		if p.Enabled && p.DeleteAfter {
			rows, err := tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL AND tier = 'sunset'
				AND sunset_at <= $2::timestamptz - make_interval(months => $3)`, wsID, now, p.DeleteAfterMonths)
			if err != nil {
				return err
			}
			ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
			if err != nil {
				return err
			}
			if err := eraseContacts(ctx, tx, ids); err != nil {
				return err
			}
			out.Erased = int64(len(ids))
		}
		if err := s.writeRollups(ctx, tx, &ws, now, loc); err != nil {
			return err
		}
		st, err := q.GetAudienceState(ctx, wsID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if reengage {
			st.Reengaged += out.EnteredSunset
			st.WonBack += out.LeftSunset
		}
		if err := q.UpsertAudienceState(ctx, store.UpsertAudienceStateParams{WorkspaceID: wsID, TiersComputedAt: &now, Reengaged: st.Reengaged, WonBack: st.WonBack}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "audience.tiers_recompute", TargetType: "workspace", TargetID: wsID.String(),
			TargetName: ws.Name, Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, After: out})
	})
	if err != nil {
		return out, err
	}
	if s.Segments != nil {
		if err := s.Segments.RecountAll(ctx, &ws); err != nil {
			return out, err
		}
	}
	return out, nil
}

// writeRollups records today's tier mix per scope and the last three days
// of joins, departures and Broadcast engagement (late opens and bounces
// keep arriving for a few days).
func (s *Service) writeRollups(ctx context.Context, tx pgx.Tx, ws *store.Workspace, now time.Time, loc *time.Location) error {
	n := now.In(loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	from := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -3)
	tz := loc.String()
	// Tier snapshots: workspace, lists, followed views.
	if _, err := tx.Exec(ctx, `INSERT INTO audience_daily (workspace_id, scope, scope_id, day, contacts, active, cooling, at_risk, sunset)
		SELECT $1, 'workspace', $1, $2, count(*), count(*) FILTER (WHERE tier = 'active'), count(*) FILTER (WHERE tier = 'cooling'),
		       count(*) FILTER (WHERE tier = 'at_risk'), count(*) FILTER (WHERE tier = 'sunset')
		FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL
		ON CONFLICT (workspace_id, scope, scope_id, day) DO UPDATE SET contacts = excluded.contacts, active = excluded.active,
		    cooling = excluded.cooling, at_risk = excluded.at_risk, sunset = excluded.sunset`, ws.ID, today); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audience_daily (workspace_id, scope, scope_id, day, contacts, active, cooling, at_risk, sunset)
		SELECT $1, 'list', l.id, $2, count(c.id), count(c.id) FILTER (WHERE c.tier = 'active'), count(c.id) FILTER (WHERE c.tier = 'cooling'),
		       count(c.id) FILTER (WHERE c.tier = 'at_risk'), count(c.id) FILTER (WHERE c.tier = 'sunset')
		FROM lists l
		LEFT JOIN list_memberships lm ON lm.list_id = l.id AND lm.status = 'subscribed'
		LEFT JOIN contacts c ON c.id = lm.contact_id AND c.erased_at IS NULL
		WHERE l.workspace_id = $1 AND l.deleted_at IS NULL
		GROUP BY l.id
		ON CONFLICT (workspace_id, scope, scope_id, day) DO UPDATE SET contacts = excluded.contacts, active = excluded.active,
		    cooling = excluded.cooling, at_risk = excluded.at_risk, sunset = excluded.sunset`, ws.ID, today); err != nil {
		return err
	}
	// Joins and departures per list and day.
	if _, err := tx.Exec(ctx, `INSERT INTO audience_daily (workspace_id, scope, scope_id, day, joined, departed)
		SELECT $1, 'list', x.list_id, x.day, sum(x.joined), sum(x.departed) FROM (
			SELECT lm.list_id, (lm.subscribed_at AT TIME ZONE $3)::date AS day, 1 AS joined, 0 AS departed
			FROM list_memberships lm JOIN lists l ON l.id = lm.list_id AND l.workspace_id = $1 WHERE lm.subscribed_at >= $2
			UNION ALL
			SELECT lm.list_id, (lm.unsubscribed_at AT TIME ZONE $3)::date, 0, 1
			FROM list_memberships lm JOIN lists l ON l.id = lm.list_id AND l.workspace_id = $1 WHERE lm.unsubscribed_at >= $2
		) x GROUP BY x.list_id, x.day
		ON CONFLICT (workspace_id, scope, scope_id, day) DO UPDATE SET joined = excluded.joined, departed = excluded.departed`, ws.ID, from, tz); err != nil {
		return err
	}
	// Broadcast engagement of each list's members, by sending day.
	engagement := `count(*), count(*) FILTER (WHERE m.delivered_at IS NOT NULL),
		count(*) FILTER (WHERE EXISTS (SELECT 1 FROM delivery_events de WHERE de.message_id = m.id AND de.type = 'open' AND NOT de.machine)),
		count(*) FILTER (WHERE EXISTS (SELECT 1 FROM delivery_events de WHERE de.message_id = m.id AND de.type = 'unsubscribe')),
		count(*) FILTER (WHERE EXISTS (SELECT 1 FROM delivery_events de WHERE de.message_id = m.id AND de.type = 'complaint')),
		count(*) FILTER (WHERE m.status = 'bounced' OR EXISTS (SELECT 1 FROM delivery_events de WHERE de.message_id = m.id AND de.type = 'hard_bounce'))`
	upsert := ` ON CONFLICT (workspace_id, scope, scope_id, day) DO UPDATE SET sent = excluded.sent, delivered = excluded.delivered,
		opened_human = excluded.opened_human, unsubscribed = excluded.unsubscribed, complained = excluded.complained, bounced = excluded.bounced`
	if _, err := tx.Exec(ctx, `INSERT INTO audience_daily (workspace_id, scope, scope_id, day, sent, delivered, opened_human, unsubscribed, complained, bounced)
		SELECT $1, 'list', lm.list_id, (m.queued_at AT TIME ZONE $3)::date, `+engagement+`
		FROM messages m
		JOIN list_memberships lm ON lm.contact_id = m.contact_id
		JOIN lists l ON l.id = lm.list_id AND l.workspace_id = $1 AND l.deleted_at IS NULL
		WHERE m.workspace_id = $1 AND m.stream = 'broadcast' AND m.queued_at >= $2
		GROUP BY lm.list_id, 4`+upsert, ws.ID, from, tz); err != nil {
		return err
	}
	// Followed views: tier snapshot and engagement of the matching contacts.
	views, err := store.New(tx).HealthViews(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, v := range views {
		var f ContactFilter
		_ = json.Unmarshal(v.Filters, &f)
		fs := filterSQL{where: "c.workspace_id = $1 AND c.erased_at IS NULL", args: []any{ws.ID}}
		if err := s.applyFilter(ctx, ws.ID, &fs, f); err != nil {
			continue
		}
		k := len(fs.args)
		args := append(fs.args, v.ID, today)
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO audience_daily (workspace_id, scope, scope_id, day, contacts, active, cooling, at_risk, sunset)
			SELECT $1, 'view', $%d, $%d, count(*), count(*) FILTER (WHERE c.tier = 'active'), count(*) FILTER (WHERE c.tier = 'cooling'),
			       count(*) FILTER (WHERE c.tier = 'at_risk'), count(*) FILTER (WHERE c.tier = 'sunset')
			FROM contacts c WHERE %s
			ON CONFLICT (workspace_id, scope, scope_id, day) DO UPDATE SET contacts = excluded.contacts, active = excluded.active,
			    cooling = excluded.cooling, at_risk = excluded.at_risk, sunset = excluded.sunset`, k+1, k+2, fs.where), args...); err != nil {
			return err
		}
		args = append(fs.args, v.ID, from, tz)
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO audience_daily (workspace_id, scope, scope_id, day, sent, delivered, opened_human, unsubscribed, complained, bounced)
			SELECT $1, 'view', $%d, (m.queued_at AT TIME ZONE $%d)::date, `+engagement+`
			FROM messages m JOIN contacts c ON c.id = m.contact_id
			WHERE m.workspace_id = $1 AND m.stream = 'broadcast' AND m.queued_at >= $%d AND %s
			GROUP BY 4`+upsert, k+1, k+3, k+2, fs.where), args...); err != nil {
			return err
		}
	}
	return nil
}
