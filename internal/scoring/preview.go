package scoring

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// The impact preview scores an edited, unsaved set-up against today's
// scores without writing anything. Only the contacts the edit can change
// are replayed: those with events an added, removed or edited rule matches
// (everyone with events when decay changes); field conditions are
// evaluated for everyone in SQL; moving only the thresholds re-buckets
// today's scores. Very large replays are estimated from a sample.

// previewLimit is the most contacts a preview replays exactly.
const previewLimit = 250_000

// ScoringPreviewInput is an edited, unsaved set-up.
type ScoringPreviewInput struct {
	Config ScoringConfig `json:"config"`
}

// ScoringImpact is how an edit would move contacts.
type ScoringImpact struct {
	Contacts int64 `json:"contacts"`
	// Current is today's spread (live scores), Spread the spread after the edit.
	Current     []ScoringBucket   `json:"current"`
	Spread      []ScoringBucket   `json:"spread"`
	BandsBefore ScoringBandCounts `json:"bandsBefore"`
	BandsAfter  ScoringBandCounts `json:"bandsAfter"`
	// Changed is how many contacts change band; Moves says which way.
	Changed int64             `json:"changed"`
	Moves   []ScoringBandMove `json:"moves"`
	// Flows are the live "Score crosses…" automations and how many
	// contacts would cross their threshold because of the edit.
	Flows []ScoringFlowImpact `json:"automations"`
	// FieldMatches counts the contacts each field condition matches today,
	// by rule id ("1,532 contacts match today").
	FieldMatches map[string]int64 `json:"fieldMatches"`
	// Rescored is how many contacts the preview replayed.
	Rescored int64 `json:"rescored"`
	// Estimated is true when a large replay was sampled.
	Estimated  bool `json:"estimated"`
	DurationMs int  `json:"durationMs"`
}

// eventRulesChanged lists the event rules an edit adds, removes or changes,
// in both their old and new form, and whether decay changed.
func changedEventRules(prev, next ScoringConfig) (rules []ScoringRule, all bool) {
	if prev.Decay != next.Decay {
		return nil, true
	}
	old := map[string]ScoringRule{}
	for _, r := range prev.Rules {
		if r.IsEvent() {
			old[r.ID] = r
		}
	}
	seen := map[string]bool{}
	for _, r := range next.Rules {
		if !r.IsEvent() {
			continue
		}
		seen[r.ID] = true
		o, ok := old[r.ID]
		if !ok {
			if r.Enabled {
				rules = append(rules, r)
			}
			continue
		}
		if sameRule(o, r) || (!o.Enabled && !r.Enabled) {
			continue
		}
		if o.Enabled {
			rules = append(rules, o)
		}
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	for id, o := range old {
		if !seen[id] && o.Enabled {
			rules = append(rules, o)
		}
	}
	return rules, false
}

func fieldRulesChanged(prev, next ScoringConfig) bool {
	a, b := fieldRules(prev), fieldRules(next)
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if !sameRule(a[i], b[i]) {
			return true
		}
	}
	return false
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// matchSQL is the condition "a score event this rule counts".
func matchSQL(r ScoringRule, a *sqlArgs) string {
	switch r.Kind {
	case KindOpen:
		if id, err := uuid.Parse(r.ListID); err == nil {
			return "(se.kind = 'open' AND " + a.add(id) + "::uuid = ANY (se.lists))"
		}
		return "se.kind = 'open'"
	case KindClick:
		if r.Contains != "" {
			return "(se.kind = 'click' AND lower(se.ref) LIKE " + a.add("%"+escapeLike(strings.ToLower(r.Contains))+"%") + ")"
		}
		return "se.kind = 'click'"
	case KindForm:
		return "(se.kind = 'form' AND se.ref = " + a.add(strings.ToLower(r.FormID)) + ")"
	case KindAPIEvent:
		return "(se.kind = 'api_event' AND se.ref = " + a.add(r.Event) + ")"
	case KindSoftBounce:
		return "se.kind = 'soft_bounce'"
	case KindUnsubscribe:
		if r.ListID != "" {
			return "(se.kind = 'unsubscribe' AND se.ref = " + a.add(strings.ToLower(r.ListID)) + ")"
		}
		return "se.kind = 'unsubscribe'"
	}
	return "false"
}

// Preview computes the impact of an edited set-up.
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, in ScoringPreviewInput) (ScoringImpact, error) {
	start := time.Now()
	draft := in.Config.Clone()
	draft.Normalize()
	r, err := refs(ctx, s.Q, ws.ID)
	if err != nil {
		return ScoringImpact{}, err
	}
	if err := draft.Validate(r); err != nil {
		return ScoringImpact{}, problem(ctx, err)
	}
	st, err := s.load(ctx, s.Q, ws.ID, ws.Timezone)
	if err != nil {
		return ScoringImpact{}, err
	}
	now := s.now()
	since := now.AddDate(0, 0, -LookbackDays)
	base := st.applied
	firstSetup := st.row.AppliedVersion == 0
	var out ScoringImpact
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// Who to replay.
		a := &sqlArgs{}
		a.add(ws.ID)
		a.add(since)
		var cond string
		rules, all := changedEventRules(base, draft)
		switch {
		case firstSetup || all:
			cond = "true"
		case len(rules) > 0:
			parts := make([]string, 0, len(rules))
			for _, r := range rules {
				parts = append(parts, matchSQL(r, a))
			}
			cond = "(" + strings.Join(parts, " OR ") + ")"
		}
		var ids []uuid.UUID
		if cond != "" {
			rows, err := tx.Query(ctx, `SELECT DISTINCT se.contact_id FROM score_events se
				WHERE se.workspace_id = $1 AND se.occurred_at > $2 AND `+cond+` ORDER BY 1`, a.args...)
			if err != nil {
				return err
			}
			if ids, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]); err != nil {
				return err
			}
		}
		weight := 1
		replay := ids
		if len(ids) > previewLimit {
			weight = (len(ids) + previewLimit - 1) / previewLimit
			replay = make([]uuid.UUID, 0, previewLimit)
			for i := 0; i < len(ids); i += weight {
				replay = append(replay, ids[i])
			}
			out.Estimated = true
		}
		out.Rescored = int64(len(replay))
		if _, err := tx.Exec(ctx, `CREATE TEMP TABLE scoring_preview (contact_id uuid PRIMARY KEY, sampled boolean NOT NULL, score smallint) ON COMMIT DROP`); err != nil {
			return err
		}
		if len(ids) > 0 {
			sampled := make(map[uuid.UUID]int, len(replay))
			eng := NewEngine(draft, st.loc)
			for start := 0; start < len(replay); start += batchSize {
				batch := replay[start:min(start+batchSize, len(replay))]
				cs, err := compute(ctx, q, tx, eng, st.types, batch, now)
				if err != nil {
					return err
				}
				for _, c := range cs {
					sampled[c.contact] = c.score
				}
			}
			rows := make([][]any, 0, len(ids))
			for _, id := range ids {
				if sc, ok := sampled[id]; ok {
					rows = append(rows, []any{id, true, int16(sc)})
				} else {
					rows = append(rows, []any{id, false, nil})
				}
			}
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"scoring_preview"}, []string{"contact_id", "sampled", "score"}, pgx.CopyFromRows(rows)); err != nil {
				return err
			}
		}
		// New score of everyone else: today's, or today's events part with
		// the edited field conditions.
		b := &sqlArgs{}
		b.add(ws.ID)
		b.add(weight)
		other := "coalesce(cs.score, 0)"
		if firstSetup || fieldRulesChanged(base, draft) {
			other = "least(100, greatest(0, coalesce(cs.events_part, 0) + " + fieldsPartSQL(draft, st.types, b) + "))"
		}
		oldScore := "coalesce(cs.score, 0)"
		if firstSetup {
			oldScore = "-1"
		}
		rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT %s AS old, CASE WHEN p.contact_id IS NULL THEN %s ELSE coalesce(p.score, -1) END AS new,
				sum(CASE WHEN p.contact_id IS NULL THEN 1 WHEN p.sampled THEN $2::int ELSE 0 END)::bigint
			FROM contacts c
			LEFT JOIN contact_scores cs ON cs.contact_id = c.id
			LEFT JOIN scoring_preview p ON p.contact_id = c.id
			WHERE c.workspace_id = $1 AND c.erased_at IS NULL
			GROUP BY 1, 2`, oldScore, other), b.args...)
		if err != nil {
			return err
		}
		type cell struct {
			old, new int
			n        int64
		}
		var cells []cell
		for rows.Next() {
			var c cell
			if err := rows.Scan(&c.old, &c.new, &c.n); err != nil {
				rows.Close()
				return err
			}
			if c.n > 0 && c.new >= 0 {
				cells = append(cells, c)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		before, after := map[int]int64{}, map[int]int64{}
		moves := map[[2]string]int64{}
		triggers, err := s.flowTriggers(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		crossed := make([]int64, len(triggers))
		for _, c := range cells {
			after[c.new] += c.n
			if c.old < 0 {
				continue
			}
			before[c.old] += c.n
			fb, tb := base.Bands.Band(c.old), draft.Bands.Band(c.new)
			if fb != tb {
				moves[[2]string{fb, tb}] += c.n
				out.Changed += c.n
			}
			for i, t := range triggers {
				if t.trigger.Crosses(c.old, c.new) {
					crossed[i] += c.n
				}
			}
		}
		out.Current, out.BandsBefore, _ = spreadOf(before, base.Bands)
		out.Spread, out.BandsAfter, out.Contacts = spreadOf(after, draft.Bands)
		if firstSetup {
			out.Current = []ScoringBucket{}
		}
		out.Moves = []ScoringBandMove{}
		for k, n := range moves {
			out.Moves = append(out.Moves, ScoringBandMove{From: k[0], To: k[1], Count: n})
		}
		sortMoves(out.Moves)
		out.Flows = []ScoringFlowImpact{}
		for i, t := range triggers {
			out.Flows = append(out.Flows, ScoringFlowImpact{ID: t.id, Name: t.name, Direction: t.trigger.Direction, Threshold: t.trigger.Threshold, Crossed: crossed[i]})
		}
		out.FieldMatches = map[string]int64{}
		if fr := fieldRules(draft); len(fr) > 0 {
			fa := &sqlArgs{}
			fa.add(ws.ID)
			exprs := make([]string, len(fr))
			for i, r := range fr {
				exprs[i] = "count(*) FILTER (WHERE " + conditionSQL(r, st.types, fa) + ")"
			}
			dest := make([]any, len(fr))
			counts := make([]int64, len(fr))
			for i := range dest {
				dest[i] = &counts[i]
			}
			if err := tx.QueryRow(ctx, "SELECT "+strings.Join(exprs, ", ")+" FROM contacts c WHERE c.workspace_id = $1 AND c.erased_at IS NULL", fa.args...).Scan(dest...); err != nil {
				return err
			}
			for i, r := range fr {
				out.FieldMatches[r.ID] = counts[i]
			}
		}
		return nil // the temporary table goes with the transaction
	})
	out.DurationMs = int(time.Since(start).Milliseconds())
	return out, err
}

type flowTrigger struct {
	id, name string
	trigger  ScoreTrigger
}

// flowTriggers are the live "Score crosses…" automations.
func (s *Service) flowTriggers(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]flowTrigger, error) {
	rows, err := q.ScoringLiveTriggers(ctx, wsID)
	if err != nil {
		return nil, err
	}
	var out []flowTrigger
	for _, r := range rows {
		if r.Status != "live" {
			continue
		}
		if t, ok := triggerOf(r.Trigger); ok {
			out = append(out, flowTrigger{id: r.ID.String(), name: r.Name, trigger: t})
		}
	}
	return out, nil
}
