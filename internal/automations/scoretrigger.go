package automations

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// "Score crosses…" (audience-insights brief §3.1): a flow starts when a
// contact's score rises to a threshold or more, or falls below it. The
// scorer (internal/scoring) queues a score_crosses event when a score
// crosses a live flow's threshold through activity, decay or a field
// condition; it never does for contacts who cross only because a rule
// changed, unless the person saving the rules chose to start the flows
// for them. A contact enters a flow at most once in OnceInDays days.

var (
	mProblemScoreDirection = msg("problem.scoreDirection")
	mProblemScoreThreshold = msg("problem.scoreThreshold")
	mProblemScoreOnce      = msg("problem.scoreOnce")
	mRefScoreRises         = msg("ref.scoreRises")
	mRefScoreFalls         = msg("ref.scoreFalls")
)

// checkScoreTrigger checks a "Score crosses…" trigger's settings.
func checkScoreTrigger(t *AutomationTrigger) []configProblem {
	var out []configProblem
	if t.Direction != "" && t.Direction != "rises" && t.Direction != "falls" {
		out = append(out, configProblem{"invalid_direction", txt(mProblemScoreDirection)})
	}
	if t.Threshold < 1 || t.Threshold > 100 {
		out = append(out, configProblem{"invalid_threshold", txt(mProblemScoreThreshold)})
	}
	if t.OnceInDays < 0 || t.OnceInDays > 3650 {
		out = append(out, configProblem{"invalid_once", txt(mProblemScoreOnce, "max", 3650)})
	}
	return out
}

// scoreTriggerMatches reports whether a score change crosses the trigger.
func scoreTriggerMatches(t AutomationTrigger, payload json.RawMessage) bool {
	var p struct {
		From int `json:"from"`
		To   int `json:"to"`
	}
	if json.Unmarshal(payload, &p) != nil || t.Threshold <= 0 {
		return false
	}
	if t.Direction == "falls" {
		return p.From >= t.Threshold && p.To < t.Threshold
	}
	return p.From < t.Threshold && p.To >= t.Threshold
}

// scoreTriggerAllows applies "at most once per contact in N days": a
// contact still in the flow, or who entered it in the last N days, doesn't
// enter again.
func scoreTriggerAllows(ctx context.Context, q *store.Queries, t AutomationTrigger, automation, contact uuid.UUID, now time.Time) (bool, error) {
	since := now.AddDate(0, 0, -t.OnceInDays)
	if t.OnceInDays <= 0 {
		since = now.Add(time.Hour) // only a run in progress blocks
	}
	busy, err := q.AutomationScoreRunSince(ctx, store.AutomationScoreRunSinceParams{AutomationID: automation, ContactID: contact, Since: since})
	return !busy, err
}

// scoreTriggerRef reads "Rises to 50 or more", "Falls below 20".
func scoreTriggerRef(t *AutomationTrigger) i18n.Text {
	if t.Direction == "falls" {
		return txt(mRefScoreFalls, "threshold", t.Threshold)
	}
	return txt(mRefScoreRises, "threshold", t.Threshold)
}

// oneOf reports whether v is one of the comma-separated values ("team,
// business"), ignoring case: the "is one of" field condition, used by the
// "Sales-ready handoff" filter ("plan is team or business").
func oneOf(v, values string) bool {
	for _, x := range strings.Split(values, ",") {
		if x = strings.TrimSpace(x); x != "" && strings.EqualFold(v, x) {
			return true
		}
	}
	return false
}

// orValues reads comma-separated values as "team or business".
func orValues(values string) string { return say(english, orValuesText(values)) }

func orValuesText(values string) i18n.Text {
	vs := []i18n.Text{}
	for _, x := range strings.Split(values, ",") {
		if x = strings.TrimSpace(x); x != "" {
			vs = append(vs, name(x))
		}
	}
	return orList(vs)
}
