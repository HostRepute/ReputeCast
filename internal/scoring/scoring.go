// Package scoring implements contact scoring (audience-insights brief §3.1,
// M23): rules that add or take points for human opens, clicks, form
// submissions, API events, soft bounces, list unsubscribes and field
// conditions, with caps, decay after weeks without activity and the Cold,
// Warm and Hot bands; the score and its breakdown on the contact profile;
// the impact preview of an edited rule; recalculation in the background;
// and the "Score crosses…" automation trigger.
//
// The score measures interest. It never stops or starts mail by itself:
// engagement tiers protect deliverability, and only automations the team
// builds on the score act on it.
package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service is the scoring domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	// Jobs returns the River client (created after the services are wired);
	// nil runs recalculations in the request, as tests do.
	Jobs func() *jobs.Client
	Log  *slog.Logger
	Now  func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// setup is a workspace's scoring as stored.
type setup struct {
	row     store.ScoringConfig
	saved   ScoringConfig // the latest saved version
	applied ScoringConfig // the version live scores use
	types   map[string]string
	loc     *time.Location
}

// configured reports whether the workspace set up scoring.
func (st *setup) configured() bool { return st != nil && st.row.Version > 0 }

func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// load reads a workspace's scoring. A workspace without scoring returns a
// setup with version 0 and the defaults.
func (s *Service) load(ctx context.Context, q *store.Queries, wsID uuid.UUID, tz string) (*setup, error) {
	st := &setup{loc: location(tz), saved: emptyConfig(), applied: emptyConfig()}
	row, err := q.ScoringConfigGet(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		row = store.ScoringConfig{WorkspaceID: wsID}
	} else if err != nil {
		return nil, err
	}
	st.row = row
	if row.Version > 0 {
		v, err := q.ScoringVersionGet(ctx, store.ScoringVersionGetParams{WorkspaceID: wsID, Version: row.Version})
		if err != nil {
			return nil, err
		}
		if st.saved, err = ParseConfig(v.Config); err != nil {
			return nil, err
		}
	}
	switch {
	case row.AppliedVersion == row.Version:
		st.applied = st.saved
	case row.AppliedVersion > 0:
		v, err := q.ScoringVersionGet(ctx, store.ScoringVersionGetParams{WorkspaceID: wsID, Version: row.AppliedVersion})
		if err != nil {
			return nil, err
		}
		if st.applied, err = ParseConfig(v.Config); err != nil {
			return nil, err
		}
	}
	fields, err := q.ScoringFieldsOf(ctx, wsID)
	if err != nil {
		return nil, err
	}
	st.types = map[string]string{}
	for _, f := range fields {
		st.types[f.Key] = f.Type
	}
	return st, nil
}

func emptyConfig() ScoringConfig {
	return ScoringConfig{Rules: []ScoringRule{}, Decay: DefaultDecay, Bands: DefaultBands}
}

// computed is one contact's score from a replay.
type computed struct {
	contact    uuid.UUID
	score      int
	eventsPart int
	fieldsPart int
	activeAt   *time.Time
	changedAt  *time.Time
	nextCheck  *time.Time
}

// eventsFor loads the score events of a batch of contacts, by contact.
func eventsFor(ctx context.Context, q *store.Queries, ids []uuid.UUID, since time.Time) (map[uuid.UUID][]Event, error) {
	rows, err := q.ScoringEventsOf(ctx, store.ScoringEventsOfParams{ContactIds: ids, Since: since})
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID][]Event, len(ids))
	for _, r := range rows {
		ev := Event{At: r.OccurredAt, Kind: r.Kind, Ref: r.Ref, Lists: r.Lists, Counted: r.Counted}
		if r.MessageID != nil {
			ev.Message = *r.MessageID
		}
		out[r.ContactID] = append(out[r.ContactID], ev)
	}
	return out, nil
}

// fieldsFor evaluates the field conditions for a batch of contacts.
func fieldsFor(ctx context.Context, db store.DBTX, cfg ScoringConfig, types map[string]string, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(ids))
	if len(fieldRules(cfg)) == 0 {
		return out, nil
	}
	a := &sqlArgs{}
	idsArg := a.add(ids)
	expr := fieldsPartSQL(cfg, types, a)
	rows, err := db.Query(ctx, "SELECT c.id, "+expr+" FROM contacts c WHERE c.id = ANY ("+idsArg+"::uuid[])", a.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// compute replays a batch of contacts under cfg at now.
func compute(ctx context.Context, q *store.Queries, db store.DBTX, eng *Engine, types map[string]string, ids []uuid.UUID, now time.Time) ([]computed, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	evs, err := eventsFor(ctx, q, ids, now.AddDate(0, 0, -LookbackDays))
	if err != nil {
		return nil, err
	}
	fields, err := fieldsFor(ctx, db, eng.Config(), types, ids)
	if err != nil {
		return nil, err
	}
	out := make([]computed, 0, len(ids))
	for _, id := range ids {
		r := eng.Replay(evs[id], now, fields[id])
		out = append(out, computed{contact: id, score: r.Score, eventsPart: r.EventsPart, fieldsPart: r.FieldsPart,
			activeAt: r.ActiveAt, changedAt: r.ChangedAt(), nextCheck: r.NextCheck})
	}
	return out, nil
}

// columns splits computed scores into the arrays the batch queries take.
type columns struct {
	ids                       []uuid.UUID
	scores, events, fields    []int16
	active, changed, nextTime []*time.Time
}

func toColumns(cs []computed) columns {
	c := columns{}
	for _, x := range cs {
		c.ids = append(c.ids, x.contact)
		c.scores = append(c.scores, int16(x.score))
		c.events = append(c.events, int16(x.eventsPart))
		c.fields = append(c.fields, int16(x.fieldsPart))
		c.active = append(c.active, x.activeAt)
		c.changed = append(c.changed, x.changedAt)
		c.nextTime = append(c.nextTime, x.nextCheck)
	}
	return c
}

// marshal encodes v as JSON for a jsonb column.
func marshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
