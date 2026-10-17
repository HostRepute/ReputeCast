package scoring

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// batchSize is how many contacts one transaction scores.
const batchSize = 2000

const upsertScoresSQL = `INSERT INTO contact_scores (contact_id, workspace_id, score, events_part, fields_part, active_at, changed_at, next_check_at, version, computed_at)
SELECT u.contact_id, $1, u.score, u.events_part, u.fields_part, u.active_at, u.changed_at, u.next_check_at, $2, $3
FROM unnest($4::uuid[], $5::smallint[], $6::smallint[], $7::smallint[], $8::timestamptz[], $9::timestamptz[], $10::timestamptz[])
     AS u(contact_id, score, events_part, fields_part, active_at, changed_at, next_check_at)
JOIN contacts c ON c.id = u.contact_id AND c.erased_at IS NULL
ON CONFLICT (contact_id) DO UPDATE
SET score = excluded.score, events_part = excluded.events_part, fields_part = excluded.fields_part,
    active_at = excluded.active_at, changed_at = excluded.changed_at, next_check_at = excluded.next_check_at,
    version = excluded.version, computed_at = excluded.computed_at`

const stageScoresSQL = `INSERT INTO scoring_recalc_scores (recalc_id, contact_id, score, events_part, fields_part, active_at, changed_at, next_check_at)
SELECT $1, u.contact_id, u.score, u.events_part, u.fields_part, u.active_at, u.changed_at, u.next_check_at
FROM unnest($2::uuid[], $3::smallint[], $4::smallint[], $5::smallint[], $6::timestamptz[], $7::timestamptz[], $8::timestamptz[])
     AS u(contact_id, score, events_part, fields_part, active_at, changed_at, next_check_at)
ON CONFLICT (recalc_id, contact_id) DO UPDATE
SET score = excluded.score, events_part = excluded.events_part, fields_part = excluded.fields_part,
    active_at = excluded.active_at, changed_at = excluded.changed_at, next_check_at = excluded.next_check_at`

func writeScores(ctx context.Context, tx store.DBTX, wsID uuid.UUID, version int32, at time.Time, cs []computed) error {
	if len(cs) == 0 {
		return nil
	}
	c := toColumns(cs)
	_, err := tx.Exec(ctx, upsertScoresSQL, wsID, version, at, c.ids, c.scores, c.events, c.fields, c.active, c.changed, c.nextTime)
	return err
}

func stageScores(ctx context.Context, tx store.DBTX, recalc uuid.UUID, cs []computed) error {
	if len(cs) == 0 {
		return nil
	}
	c := toColumns(cs)
	_, err := tx.Exec(ctx, stageScoresSQL, recalc, c.ids, c.scores, c.events, c.fields, c.active, c.changed, c.nextTime)
	return err
}

// Trigger causes, kept in the trigger event's payload.
const (
	CauseActivity   = "activity"    // an open, a click, a form, an API event, a bounce or an unsubscribe
	CauseDecay      = "decay"       // weeks without a human open or click, or events leaving the look-back
	CauseField      = "field"       // a field condition turned true or false (nightly)
	CauseRuleChange = "rule_change" // a recalculation the save dialog let start automations
)

// crossing is a contact whose score moved past a trigger's threshold.
type crossing struct {
	contact  uuid.UUID
	from, to int
}

// liveTriggers returns the thresholds of the workspace's live "Score
// crosses…" automations.
func liveTriggers(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]ScoreTrigger, error) {
	rows, err := q.ScoringLiveTriggers(ctx, wsID)
	if err != nil {
		return nil, err
	}
	var out []ScoreTrigger
	for _, r := range rows {
		if r.Status != "live" {
			continue
		}
		if t, ok := triggerOf(r.Trigger); ok {
			out = append(out, t)
		}
	}
	return out, nil
}

// ScoreTrigger is the part of an automation's "Score crosses…" trigger
// the scorer needs (internal/automations owns the trigger).
type ScoreTrigger struct {
	Direction string `json:"direction"` // rises, falls
	Threshold int    `json:"threshold"`
}

// Crosses reports whether a change from one score to another crosses t.
func (t ScoreTrigger) Crosses(from, to int) bool {
	if t.Direction == "falls" {
		return from >= t.Threshold && to < t.Threshold
	}
	return from < t.Threshold && to >= t.Threshold
}

// anyCrosses reports whether a change crosses any of the triggers.
func anyCrosses(ts []ScoreTrigger, from, to int) bool {
	for _, t := range ts {
		if t.Crosses(from, to) {
			return true
		}
	}
	return false
}

// emitCrossings queues "Score crosses…" trigger events. The automations
// dispatcher matches each against its flows' direction and threshold.
func emitCrossings(ctx context.Context, q *store.Queries, wsID uuid.UUID, cause string, at time.Time, cs []crossing) error {
	if len(cs) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(cs))
	from := make([]int32, len(cs))
	to := make([]int32, len(cs))
	for i, c := range cs {
		ids[i], from[i], to[i] = c.contact, int32(c.from), int32(c.to)
	}
	return q.ScoringInsertTriggerEvents(ctx, store.ScoringInsertTriggerEventsParams{
		WorkspaceID: wsID, Cause: cause, At: at, ContactIds: ids, FromScores: from, ToScores: to})
}
