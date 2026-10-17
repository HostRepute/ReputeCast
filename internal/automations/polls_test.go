package automations

import (
	"encoding/json"
	"testing"

	"github.com/hostrepute/reputecast/internal/store"
)

// The trigger "Answered a poll or survey": its poll or survey, the answer
// filter, and changed answers only when it starts again on changes.
func TestPollTriggerMatches(t *testing.T) {
	ev := func(ref string, answer int, changed bool) store.ClaimAutomationEventsRow {
		b, _ := json.Marshal(map[string]any{"answer": answer, "changed": changed})
		return store.ClaimAutomationEventsRow{Kind: TriggerPollAnswered, Ref: ref, Payload: b}
	}
	nps := "poll:01a146c1-534d-7d2f-bd27-6065beff9b30"
	survey := "survey:01a147b3-20e7-7c8e-9efb-c21cee536466"
	detractors := AutomationTrigger{Kind: TriggerPollAnswered, Poll: nps, AnswerOp: AnswerIs, Answers: []int{0, 1, 2, 3, 4, 5, 6}}
	cases := []struct {
		name string
		t    AutomationTrigger
		ev   store.ClaimAutomationEventsRow
		want bool
	}{
		{"a detractor", detractors, ev(nps, 4, false), true},
		{"a promoter", detractors, ev(nps, 9, false), false},
		{"another poll", detractors, ev("poll:01a146c1-5349-7d2f-8f1a-7ef9fbe2f2de", 4, false), false},
		{"no poll picked", AutomationTrigger{Kind: TriggerPollAnswered}, ev(nps, 4, false), false},
		{"a changed answer, not starting again", detractors, ev(nps, 3, true), false},
		{"a changed answer, starting again", AutomationTrigger{Kind: TriggerPollAnswered, Poll: nps, AnswerOp: AnswerIs, Answers: []int{3}, OnChange: true}, ev(nps, 3, true), true},
		{"is not", AutomationTrigger{Kind: TriggerPollAnswered, Poll: nps, AnswerOp: AnswerIsNot, Answers: []int{9, 10}}, ev(nps, 7, false), true},
		{"is not, excluded", AutomationTrigger{Kind: TriggerPollAnswered, Poll: nps, AnswerOp: AnswerIsNot, Answers: []int{9, 10}}, ev(nps, 10, false), false},
		{"any answer", AutomationTrigger{Kind: TriggerPollAnswered, Poll: nps, AnswerOp: AnswerAny}, ev(nps, 0, false), true},
		{"a survey ignores the answer filter", AutomationTrigger{Kind: TriggerPollAnswered, Poll: survey, AnswerOp: AnswerIs, Answers: []int{1}}, ev(survey, 0, false), true},
	}
	for _, c := range cases {
		if got := pollTriggerMatches(c.t, c.ev); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	// A changed answer starts the flow again only when the trigger says so.
	if pollReentry(detractors, ev(nps, 4, true)) {
		t.Fatal("re-entry without OnChange")
	}
	again := detractors
	again.OnChange = true
	if !pollReentry(again, ev(nps, 4, true)) || pollReentry(again, ev(nps, 4, false)) {
		t.Fatal("re-entry on a changed answer")
	}
}
