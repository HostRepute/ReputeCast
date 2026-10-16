package segments

import (
	"errors"
	"strings"
	"testing"
)

// The scoring sample's segment: Hot on paid plans.
func TestCompileScoreBandAndOneOf(t *testing.T) {
	d := SegmentDefinition{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{
		{Field: "score_band", Op: "is", Value: "hot"},
		{Field: "custom:plan", Op: "is_one_of", Value: "team, Business,team"},
		{Field: "score", Op: "between", Value: "50", To: "100"},
		{Field: "utm_campaign", Op: "is", Value: "Flags-GA"},
		{Field: "referring_domain", Op: "ends_with", Value: "acme.io"},
	}}}}
	if err := d.Validate(custom, lists); err != nil {
		t.Fatal(err)
	}
	disp := Compile(d, wsID, custom).Display()
	for _, want := range []string{
		"c.score_band = 'hot'",
		"lower(coalesce(c.fields->>'plan', '')) IN ('team', 'business')",
		"c.score BETWEEN 50 AND 100",
		"lower(coalesce(c.utm_campaign, '')) = 'flags-ga'",
		"lower(coalesce(c.referring_domain, '')) LIKE '%acme.io'",
	} {
		if !strings.Contains(disp, want) {
			t.Errorf("display SQL lacks %q:\n%s", want, disp)
		}
	}
	q, args := Compile(d, wsID, custom).Query(1)
	if strings.Contains(q, "'team'") || strings.Contains(q, "'hot'") {
		t.Fatalf("values must be bound:\n%s", q)
	}
	if len(args) != 8 || args[2] != "team" || args[3] != "business" {
		t.Fatalf("args: %#v", args)
	}
}

func TestValidateScoringConditions(t *testing.T) {
	bad := []SegmentCondition{
		{Field: "score", Op: "at_least", Value: "101"},
		{Field: "score", Op: "between", Value: "60", To: "40"},
		{Field: "score", Op: "between", Value: "40"},
		{Field: "score", Op: "is", Value: "40"},
		{Field: "score_band", Op: "is", Value: "lukewarm"},
		{Field: "growth_source", Op: "is", Value: ""},
		{Field: "custom:plan", Op: "is_one_of", Value: "  "},
		{Field: "custom:mrr", Op: "is_one_of", Value: "9,29"},
	}
	for i, c := range bad {
		d := SegmentDefinition{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{c}}}}
		var ve *ValidationError
		if err := d.Validate(custom, lists); !errors.As(err, &ve) {
			t.Errorf("case %d (%+v): want a validation error, got %v", i, c, err)
		}
	}
	ok := SegmentDefinition{Groups: []SegmentGroup{{Match: "any", Conditions: []SegmentCondition{
		{Field: "score", Op: "at_most", Value: "19"},
		{Field: "growth_source", Op: "is_not", Value: "import:0199c0de-0000-7000-8000-000000000009"},
	}}}}
	if err := ok.Validate(custom, lists); err != nil {
		t.Fatal(err)
	}
}

func TestOneOfWithoutValuesMatchesNobody(t *testing.T) {
	var st sqlText
	oneOfCondition(&st, "c.fields->>'plan'", " , ,")
	if got, _ := st.render(true, 1); got != "false" {
		t.Fatalf("got %q", got)
	}
}
