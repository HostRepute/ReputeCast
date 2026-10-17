package automations

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The trigger "Answered a poll or survey" (design expansion, X11): an
// answer recorded on a poll's answer page, or a survey submitted. Its
// events (internal/polls) carry the poll or survey as ref ("poll:<id>",
// "survey:<id>") and the answer and whether it changed as payload. Test
// answers and answers from people who aren't contacts never start it.

// TriggerPollAnswered is the trigger's kind.
const TriggerPollAnswered = "poll_answered"

// Answer filters of the trigger.
const (
	AnswerAny   = "any"
	AnswerIs    = "is"
	AnswerIsNot = "is_not"
)

type pollPayload struct {
	Answer  *int `json:"answer"`
	Changed bool `json:"changed"`
}

func pollEvent(ev store.ClaimAutomationEventsRow) pollPayload {
	var p pollPayload
	_ = json.Unmarshal(ev.Payload, &p)
	return p
}

// pollTriggerMatches decides whether an answer starts a flow: its poll or
// survey, its answer, and a changed answer only when the trigger starts
// again on changes.
func pollTriggerMatches(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	if t.Poll == "" || t.Poll != ev.Ref {
		return false
	}
	p := pollEvent(ev)
	if p.Changed && !t.OnChange {
		return false
	}
	if len(t.Answers) == 0 || t.AnswerOp == "" || t.AnswerOp == AnswerAny || p.Answer == nil || strings.HasPrefix(ev.Ref, "survey:") {
		return true
	}
	in := slices.Contains(t.Answers, *p.Answer)
	if t.AnswerOp == AnswerIsNot {
		return !in
	}
	return in
}

// pollReentry lets a changed answer start the flow again when the trigger
// says so, whatever its re-entry setting.
func pollReentry(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	return t.Kind == TriggerPollAnswered && t.OnChange && pollEvent(ev).Changed
}

// loadPollRefs names the polls and surveys a trigger can point at.
func loadPollRefs(ctx context.Context, q *store.Queries, wsID uuid.UUID, r *Refs) error {
	rows, err := q.AutomationPollRefs(ctx, wsID)
	if err != nil {
		return err
	}
	for _, p := range rows {
		r.Polls[p.Ref] = p.Name
	}
	return nil
}
