package engine

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Stuck sending jobs are rescued within minutes (review GS-1). A sending job
// whose worker died (crashed, killed after its grace period) stays
// "running" in River until River's own rescuer, an hour later, makes it
// available; its unique key keeps the campaign's next run out meanwhile.
// The watchdog gives such a job back to River once it has been running
// longer than its kind's timeout plus a minute (a live worker's job is
// cancelled at its timeout): about three minutes after a run job started.
// A send loop that still holds its campaign's lock (another process works
// on it) is left alone.

// jobTimeouts are the sending workers' Timeout values.
var jobTimeouts = map[string]time.Duration{
	RunArgs{}.Kind():                 2 * time.Minute,
	PrepareArgs{}.Kind():             20 * time.Minute,
	SimArgs{}.Kind():                 2 * time.Minute,
	OutboxArgs{}.Kind():              2 * time.Minute,
	TransactionalOutboxArgs{}.Kind(): 2 * time.Minute,
}

// rescueSlack is how long past its timeout a job is given before it counts
// as stuck.
const rescueSlack = time.Minute

const rescueReason = "Stuck send job rescued by the sending watchdog: its worker stopped without finishing it"

// rescueStuck gives River back the sending jobs whose worker is gone. It
// returns the ids it rescued. Times are the database's, as River's.
func (e *Engine) rescueStuck(ctx context.Context) ([]int64, error) {
	kinds := make([]string, 0, len(jobTimeouts))
	least := time.Duration(1<<62 - 1)
	for k, d := range jobTimeouts {
		kinds = append(kinds, k)
		least = min(least, d)
	}
	rows, err := e.Pool.Query(ctx, `SELECT id, kind, args, extract(epoch FROM now() - attempted_at)::float8
		FROM river_job WHERE kind = ANY($1) AND state = 'running' AND attempted_at < now() - make_interval(secs => $2)
		ORDER BY id LIMIT 50`, kinds, (least + rescueSlack).Seconds())
	if err != nil {
		return nil, err
	}
	type stuck struct {
		id      int64
		kind    string
		args    []byte
		running time.Duration
	}
	var found []stuck
	for rows.Next() {
		var s stuck
		var secs float64
		if err := rows.Scan(&s.id, &s.kind, &s.args, &secs); err != nil {
			rows.Close()
			return nil, err
		}
		s.running = time.Duration(secs * float64(time.Second))
		found = append(found, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []int64
	for _, s := range found {
		if s.running < jobTimeouts[s.kind]+rescueSlack {
			continue
		}
		if s.kind == (RunArgs{}).Kind() || s.kind == (PrepareArgs{}).Kind() {
			var a struct {
				CampaignID uuid.UUID `json:"campaign_id"`
			}
			if json.Unmarshal(s.args, &a) == nil && a.CampaignID != uuid.Nil {
				unlock, err := e.lockSend(ctx, a.CampaignID)
				if errors.Is(err, errBusy) {
					continue // a process still runs this send
				}
				if err != nil {
					return out, err
				}
				unlock()
			}
		}
		tag, err := e.Pool.Exec(ctx, `UPDATE river_job
			SET state = 'retryable', scheduled_at = now(),
			    errors = array_append(errors, jsonb_build_object('at', now(), 'attempt', attempt, 'error', $2::text, 'trace', ''))
			WHERE id = $1 AND state = 'running'`, s.id, rescueReason)
		if err != nil {
			return out, err
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		out = append(out, s.id)
		e.log().WarnContext(ctx, "rescued a stuck send job", "job", s.id, "kind", s.kind, "running", s.running.Round(time.Second))
		if c := e.jobs(); c != nil {
			if _, err := c.JobRetry(ctx, s.id); err != nil {
				e.log().WarnContext(ctx, "make a rescued send job available", "job", s.id, "err", err)
			}
		}
	}
	return out, nil
}

// syncPausedInDoubt resolves the in-doubt incident of a send that waits
// only for it once events settled every message in doubt, so the watchdog
// sends it on.
func (e *Engine) syncPausedInDoubt(ctx context.Context, id uuid.UUID, now time.Time) error {
	cc, err := e.Q.SendingCampaignContext(ctx, id)
	if err != nil {
		return err
	}
	return e.syncInDoubt(ctx, cc, now)
}
