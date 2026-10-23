package scoring

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// A recalculation scores every contact of a workspace under a newly saved
// configuration, from the events of the last 180 days, without touching
// the scores in use: results go to scoring_recalc_scores, and the last step
// applies them in one transaction, flips the applied version and says what
// moved. Until then contacts keep their previous score, and segments and
// triggers use it. A workspace's first recalculation also copies the source
// events of the last 270 days (the backfill).

var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// ScoringRecalcSummary says what a finished recalculation changed.
type ScoringRecalcSummary struct {
	// Changed is how many contacts changed band.
	Changed int64               `json:"changed"`
	Moves   []ScoringBandMove   `json:"moves"`
	Bands   ScoringBandCounts   `json:"bands"`
	Flows   []ScoringFlowImpact `json:"automations"`
}

// ScoringBandMove is how many contacts went from one band to another.
type ScoringBandMove struct {
	From  string `json:"from" enum:"cold,warm,hot"`
	To    string `json:"to" enum:"cold,warm,hot"`
	Count int64  `json:"count"`
}

// ScoringBandCounts are contacts per band.
type ScoringBandCounts struct {
	Cold int64 `json:"cold"`
	Warm int64 `json:"warm"`
	Hot  int64 `json:"hot"`
}

// ScoringFlowImpact is a "Score crosses…" automation and the contacts who
// cross its threshold because of a change.
type ScoringFlowImpact struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Direction string `json:"direction" enum:"rises,falls"`
	Threshold int    `json:"threshold"`
	// Crossed is how many contacts cross it only because of the change.
	Crossed int64 `json:"crossed"`
	// Started is how many of them the automation started for (0 unless the
	// save dialog said to start it).
	Started int64 `json:"started"`
}

// Recalculate runs (or resumes) a recalculation.
func (s *Service) Recalculate(ctx context.Context, id uuid.UUID) error {
	rc, err := s.Q.ScoringRecalcGet(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if rc.Status != "running" {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, rc.WorkspaceID)
	if err != nil {
		return err
	}
	st, err := s.load(ctx, s.Q, ws.ID, ws.Timezone)
	if err != nil {
		return err
	}
	v, err := s.Q.ScoringVersionGet(ctx, store.ScoringVersionGetParams{WorkspaceID: ws.ID, Version: rc.Version})
	if err != nil {
		return err
	}
	cfg, err := ParseConfig(v.Config)
	if err != nil {
		return err
	}
	if st.row.IngestedThrough == nil {
		if err := s.backfill(ctx, ws.ID, s.now()); err != nil {
			return fmt.Errorf("backfill: %w", err)
		}
	}
	eng := NewEngine(cfg, st.loc)
	cursor := uuid.Nil
	if rc.Cursor != nil {
		cursor = *rc.Cursor
	}
	done := rc.Done
	for {
		status, err := s.Q.ScoringRecalcStatus(ctx, rc.ID)
		if err != nil {
			return err
		}
		if status != "running" {
			return nil // superseded by a newer save
		}
		var n int
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			ids, err := q.ScoringContactsAfter(ctx, store.ScoringContactsAfterParams{WorkspaceID: ws.ID, After: cursor, Lim: batchSize})
			if err != nil || len(ids) == 0 {
				return err
			}
			n = len(ids)
			cs, err := compute(ctx, q, tx, eng, st.types, ids, s.now())
			if err != nil {
				return err
			}
			if err := stageScores(ctx, tx, rc.ID, cs); err != nil {
				return err
			}
			cursor = ids[len(ids)-1]
			done += int64(n)
			last := cursor
			return q.ScoringRecalcProgress(ctx, store.ScoringRecalcProgressParams{ID: rc.ID, Done: done, Cursor: &last})
		})
		if err != nil {
			return err
		}
		if n < batchSize {
			break
		}
	}
	return s.applyRecalc(ctx, &ws, st, rc, cfg)
}

// applyRecalc applies a recalculation's scores in one transaction.
func (s *Service) applyRecalc(ctx context.Context, ws *store.Workspace, st *setup, rc store.ScoringRecalc, cfg ScoringConfig) error {
	now := s.now()
	var sum ScoringRecalcSummary
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		status, err := q.ScoringRecalcStatus(ctx, rc.ID)
		if err != nil || status != "running" {
			return err
		}
		oldBands := st.applied.Bands
		if st.row.AppliedVersion == 0 {
			oldBands = cfg.Bands
		}
		sum, err = summarize(ctx, tx, rc.ID, oldBands, cfg.Bands)
		if err != nil {
			return err
		}
		flows, err := q.ScoringLiveTriggers(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, f := range flows {
			if f.Status != "live" {
				continue
			}
			t, ok := triggerOf(f.Trigger)
			if !ok {
				continue
			}
			fi := ScoringFlowImpact{ID: f.ID.String(), Name: f.Name, Direction: t.Direction, Threshold: t.Threshold}
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM scoring_recalc_scores st LEFT JOIN contact_scores cs ON cs.contact_id = st.contact_id
				WHERE st.recalc_id = $1 AND `+crossSQL(t, "coalesce(cs.score, 0)", "st.score"), rc.ID).Scan(&fi.Crossed); err != nil {
				return err
			}
			if rc.StartAutomations && fi.Crossed > 0 {
				tag, err := tx.Exec(ctx, `INSERT INTO automation_events (workspace_id, kind, contact_id, ref, payload, occurred_at)
					SELECT $2, 'score_crosses', st.contact_id, '', jsonb_build_object('from', coalesce(cs.score, 0), 'to', st.score, 'cause', 'rule_change'), $3
					FROM scoring_recalc_scores st JOIN contacts c ON c.id = st.contact_id AND c.erased_at IS NULL
					LEFT JOIN contact_scores cs ON cs.contact_id = st.contact_id
					WHERE st.recalc_id = $1 AND `+crossSQL(t, "coalesce(cs.score, 0)", "st.score"), rc.ID, ws.ID, now)
				if err != nil {
					return err
				}
				fi.Started = tag.RowsAffected()
			}
			sum.Flows = append(sum.Flows, fi)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO contact_scores (contact_id, workspace_id, score, events_part, fields_part, active_at, changed_at, next_check_at, version, computed_at)
			SELECT st.contact_id, $2, st.score, st.events_part, st.fields_part, st.active_at, st.changed_at, st.next_check_at, $3, $4
			FROM scoring_recalc_scores st JOIN contacts c ON c.id = st.contact_id AND c.erased_at IS NULL
			WHERE st.recalc_id = $1
			ON CONFLICT (contact_id) DO UPDATE
			SET score = excluded.score, events_part = excluded.events_part, fields_part = excluded.fields_part,
			    active_at = excluded.active_at, changed_at = excluded.changed_at, next_check_at = excluded.next_check_at,
			    version = excluded.version, computed_at = excluded.computed_at`, rc.ID, ws.ID, rc.Version, now); err != nil {
			return err
		}
		if err := q.ScoringConfigSetApplied(ctx, store.ScoringConfigSetAppliedParams{WorkspaceID: ws.ID, AppliedVersion: rc.Version, At: now}); err != nil {
			return err
		}
		// Activity during the run is scored again under the new rules.
		if err := q.ScoringDirtyMarkSince(ctx, store.ScoringDirtyMarkSinceParams{WorkspaceID: ws.ID, Since: rc.StartedAt.Add(-ingestOverlap)}); err != nil {
			return err
		}
		if err := q.ScoringRecalcFinish(ctx, store.ScoringRecalcFinishParams{ID: rc.ID, At: &now, Summary: marshal(sum)}); err != nil {
			return err
		}
		if err := q.ScoringRecalcClearStage(ctx, rc.ID); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "scoring.recalculation_finished", TargetType: "scoring", TargetID: ws.ID.String(),
			TargetName: "Scoring", Actor: systemActor, After: sum,
			Metadata: map[string]any{"summary": recalcSummaryLine(rc, sum), "version": rc.Version, "contacts": rc.Total}}); err != nil {
			return err
		}
		if rc.StartedBy != nil {
			// Kept messages: the person reads it in their language. A body
			// naming more than three automations stays in its English.
			return notify.ToUser(ctx, tx, *rc.StartedBy, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindScoreRecalculated, Severity: "ok",
				TitleText: msgs.M("notify.recalcTitle").Keep(nil), Body: recalcSummaryLine(rc, sum), BodyText: recalcSummaryText(rc, sum),
				Link: "/w/" + ws.Slug + "/audience/scoring", Data: map[string]any{"recalc": rc.ID.String(), "changed": sum.Changed}})
		}
		return nil
	})
	return err
}

// recalcSummaryLine is the notification's sentence: "1,312 contacts
// changed band · Sales-ready handoff not started for 1,108".
func recalcSummaryLine(rc store.ScoringRecalc, sum ScoringRecalcSummary) string {
	parts := []string{fmt.Sprintf("%s contacts changed band", thousands(sum.Changed))}
	if sum.Changed == 1 {
		parts[0] = "1 contact changed band"
	}
	for _, f := range sum.Flows {
		if f.Crossed == 0 {
			continue
		}
		if rc.StartAutomations {
			parts = append(parts, fmt.Sprintf("%s started for %s", f.Name, thousands(f.Started)))
		} else {
			parts = append(parts, fmt.Sprintf("%s not started for %s", f.Name, thousands(f.Crossed)))
		}
	}
	return strings.Join(parts, " · ")
}

// recalcSummaryText is recalcSummaryLine as a kept message, for up to three
// automations; zero for more, which keep the English line.
func recalcSummaryText(rc store.ScoringRecalc, sum ScoringRecalcSummary) i18n.Text {
	args := i18n.Args{"changed": sum.Changed, "start": "no"}
	if rc.StartAutomations {
		args["start"] = "yes"
	}
	n := 0
	for _, f := range sum.Flows {
		if f.Crossed == 0 {
			continue
		}
		n++
		if n > 3 {
			return i18n.Text{}
		}
		count := f.Crossed
		if rc.StartAutomations {
			count = f.Started
		}
		args[fmt.Sprintf("flow%d", n)], args[fmt.Sprintf("n%d", n)] = f.Name, count
	}
	args["flows"] = []string{"none", "one", "two", "three"}[n]
	return msgs.M("notify.recalcBody").Keep(args)
}

func thousands(n int64) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// crossSQL is the condition "a change from old to new crosses t".
func crossSQL(t ScoreTrigger, old, new string) string {
	n := fmt.Sprint(t.Threshold)
	if t.Direction == "falls" {
		return "(" + old + " >= " + n + " AND " + new + " < " + n + ")"
	}
	return "(" + old + " < " + n + " AND " + new + " >= " + n + ")"
}

// bandSQL names the band of a score expression under thresholds.
func bandSQL(expr string, b ScoringBands) string {
	return fmt.Sprintf("CASE WHEN %s >= %d THEN 'hot' WHEN %s >= %d THEN 'warm' ELSE 'cold' END", expr, b.HotFrom, expr, b.WarmFrom)
}

// summarize counts the band moves a staged recalculation makes.
func summarize(ctx context.Context, tx pgx.Tx, recalc uuid.UUID, before, after ScoringBands) (ScoringRecalcSummary, error) {
	out := ScoringRecalcSummary{Moves: []ScoringBandMove{}, Flows: []ScoringFlowImpact{}}
	rows, err := tx.Query(ctx, `SELECT `+bandSQL("coalesce(cs.score, 0)", before)+` AS f, `+bandSQL("st.score", after)+` AS t, count(*)
		FROM scoring_recalc_scores st LEFT JOIN contact_scores cs ON cs.contact_id = st.contact_id
		WHERE st.recalc_id = $1 GROUP BY 1, 2`, recalc)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var m ScoringBandMove
		if err := rows.Scan(&m.From, &m.To, &m.Count); err != nil {
			return out, err
		}
		switch m.To {
		case BandHot:
			out.Bands.Hot += m.Count
		case BandWarm:
			out.Bands.Warm += m.Count
		default:
			out.Bands.Cold += m.Count
		}
		if m.From != m.To {
			out.Changed += m.Count
			out.Moves = append(out.Moves, m)
		}
	}
	sortMoves(out.Moves)
	return out, rows.Err()
}

// sortMoves orders moves biggest first.
func sortMoves(ms []ScoringBandMove) {
	for i := 1; i < len(ms); i++ {
		for j := i; j > 0 && ms[j].Count > ms[j-1].Count; j-- {
			ms[j], ms[j-1] = ms[j-1], ms[j]
		}
	}
}

// failRecalc marks a recalculation as stopped after its last attempt.
func (s *Service) failRecalc(ctx context.Context, id uuid.UUID, cause error) {
	now := s.now()
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rc, err := q.ScoringRecalcGet(ctx, id)
		if err != nil {
			return err
		}
		if err := q.ScoringRecalcFail(ctx, store.ScoringRecalcFailParams{ID: id, At: &now, Error: clip(cause.Error(), 500)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &rc.WorkspaceID, Action: "scoring.recalculation_failed", TargetType: "scoring", TargetID: rc.WorkspaceID.String(),
			TargetName: "Scoring", Actor: systemActor, Result: "error", Reason: clip(cause.Error(), 500),
			Metadata: map[string]any{"summary": fmt.Sprintf("stopped at %s of %s; scores are unchanged", thousands(rc.Done), thousands(rc.Total)), "version": rc.Version}})
	})
	if err != nil {
		s.log().WarnContext(ctx, "mark scoring recalculation failed", "recalc", id, "err", err)
	}
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// startRecalc queues a recalculation, or runs it in the request when
// there is no job client (tests).
func (s *Service) startRecalc(ctx context.Context, id uuid.UUID) error {
	if s.Jobs != nil {
		if c := s.Jobs(); c != nil {
			_, err := c.Insert(ctx, RecalcArgs{RecalcID: id}, nil)
			return err
		}
	}
	err := s.Recalculate(ctx, id)
	if err != nil {
		s.failRecalc(ctx, id, err)
	}
	return nil
}

// triggerOf reads an automation's trigger as a score threshold.
func triggerOf(raw []byte) (ScoreTrigger, bool) {
	var t struct {
		Kind      string `json:"kind"`
		Direction string `json:"direction"`
		Threshold int    `json:"threshold"`
	}
	if err := unmarshal(raw, &t); err != nil || t.Kind != "score_crosses" || t.Threshold <= 0 {
		return ScoreTrigger{}, false
	}
	if t.Direction != "falls" {
		t.Direction = "rises"
	}
	return ScoreTrigger{Direction: t.Direction, Threshold: t.Threshold}, true
}
