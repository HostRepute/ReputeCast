package forms

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// M60 G-23–G-25: the pure parts of steps, targeting and variant assignment.

func testFields() []fieldDoc {
	return []fieldDoc{{Key: FieldEmail, Required: true}, {Key: FieldFirstName}, {Key: "topics", Type: FieldChoice, Multiple: true,
		Options: []optionDoc{{Label: "Product news", Tag: "product"}, {Label: "Engineering", Tag: "engineering"}}}}
}

func TestParseSteps(t *testing.T) {
	ctx := context.Background()
	fields := testFields()
	ok := []SignupFormStep{{ID: "s1", Fields: []string{"email"}, Button: "Next"}, {ID: "s2", Fields: []string{"topics"}}, {ID: "s3", Fields: []string{"first_name"}}}
	steps, err := parseSteps(ctx, ok, fields, "body.content.steps")
	if err != nil || len(steps) != 3 || steps[0].Button != "Next" || steps[2].Fields[0] != "first_name" {
		t.Fatalf("three steps: %+v %v", steps, err)
	}
	if steps, err := parseSteps(ctx, ok[:1], fields, "x"); err != nil || steps != nil {
		t.Fatalf("one step is no steps: %+v %v", steps, err)
	}
	cases := map[string][]SignupFormStep{
		"a field in two steps": {{ID: "s1", Fields: []string{"email", "topics"}}, {ID: "s2", Fields: []string{"topics", "first_name"}}},
		"a field in none":      {{ID: "s1", Fields: []string{"email"}}, {ID: "s2", Fields: []string{"topics"}}},
		"no email":             {{ID: "s1", Fields: []string{"first_name"}}, {ID: "s2", Fields: []string{"topics"}}},
		"an unknown field":     {{ID: "s1", Fields: []string{"email", "company"}}, {ID: "s2", Fields: []string{"topics", "first_name"}}},
		"an empty step":        {{ID: "s1", Fields: []string{"email", "topics", "first_name"}}, {ID: "s2", Fields: []string{}}},
		"the same id twice":    {{ID: "s1", Fields: []string{"email"}}, {ID: "s1", Fields: []string{"topics", "first_name"}}},
		"five steps":           {{ID: "s1"}, {ID: "s2"}, {ID: "s3"}, {ID: "s4"}, {ID: "s4"}},
	}
	for name, in := range cases {
		if _, err := parseSteps(ctx, in, fields, "body.content.steps"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// No email anywhere says so.
	_, err = parseSteps(ctx, []SignupFormStep{{ID: "s1", Fields: []string{"first_name"}}, {ID: "s2", Fields: []string{"topics"}}}, fields[1:], "x")
	if err == nil || !strings.Contains(err.Error(), "email") {
		t.Errorf("no email: %v", err)
	}
}

func TestFitSteps(t *testing.T) {
	steps := []stepDoc{{ID: "s1", Fields: []string{"email", "company"}}, {ID: "s2", Fields: []string{"first_name"}}}
	got := fitSteps(steps, testFields())
	if len(got) != 2 || strings.Join(got[0].Fields, ",") != "email" || strings.Join(got[1].Fields, ",") != "first_name,topics" {
		t.Fatalf("fit: %+v", got)
	}
	if got := fitSteps([]stepDoc{{ID: "s1", Fields: []string{"email"}}, {ID: "s2", Fields: []string{"company"}}}, testFields()[:1]); got != nil {
		t.Fatalf("one step left is none: %+v", got)
	}
}

func TestChoiceTags(t *testing.T) {
	f := testFields()[2]
	if got := choiceTags(f, []string{"1", "0", "1", "9", "x"}); strings.Join(got, ",") != "engineering,product" {
		t.Fatalf("several: %v", got)
	}
	f.Multiple = false
	if got := choiceTags(f, []string{"1", "0"}); strings.Join(got, ",") != "engineering" {
		t.Fatalf("one: %v", got)
	}
	ctx := context.Background()
	if _, err := parseChoice(ctx, SignupFormField{Key: "q1", Label: "Topics", Type: FieldChoice, Options: []SignupFormChoiceOption{{Label: "A", Tag: "x"}, {Label: "B", Tag: " X "}}}, "f"); err == nil {
		t.Error("the same tag twice: accepted")
	}
	if _, err := parseChoice(ctx, SignupFormField{Key: "q1", Label: "Topics", Type: FieldChoice, Options: []SignupFormChoiceOption{{Label: "A", Tag: "x"}}}, "f"); err == nil {
		t.Error("one answer: accepted")
	}
	d, err := parseChoice(ctx, SignupFormField{Key: "q1", Label: " Topics ", Type: FieldChoice, Options: []SignupFormChoiceOption{{Label: "A", Tag: "Alpha"}, {Label: "B", Tag: "beta"}}}, "f")
	if err != nil || d.Label != "Topics" || d.Options[0].Tag != "alpha" {
		t.Fatalf("choice: %+v %v", d, err)
	}
}

func TestTargeting(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC)
	in := &SignupFormTargeting{
		Pages:     SignupFormPageRules{Mode: "only", Rules: []SignupFormPageRule{{Op: "starts_with", Value: " /blog "}}},
		Sources:   SignupFormSources{UTMSource: []string{"newsletter", "newsletter", ""}, Referrer: "any"},
		Devices:   []string{"mobile"},
		Countries: SignupFormCountries{Mode: "except", Codes: []string{"de", "FR"}},
		Visitors:  "all",
		Schedule:  SignupFormSchedule{StartsAt: &start, EndsAt: &end},
	}
	doc, err := parseTargeting(ctx, in, KindPopup)
	if err != nil || doc.Pages.Rules[0].Value != "/blog" || len(doc.Sources.UTMSource) != 1 || doc.Countries.Codes[1] != "FR" || doc.Visitors != "" {
		t.Fatalf("parse: %+v %v", doc, err)
	}
	c := Config{Kind: KindPopup, Content: contentDoc{Targeting: doc}}
	if c.Open(start.Add(-time.Minute)) || !c.Open(start) || c.Open(end) {
		t.Error("schedule")
	}
	if c.CountryShown("DE") || !c.CountryShown("US") || !c.CountryShown("") {
		t.Error("except countries")
	}
	c.Content.Targeting.Countries.Mode = "only"
	if !c.CountryShown("FR") || c.CountryShown("US") || c.CountryShown("") {
		t.Error("only countries: an unknown country hides the form")
	}
	if !c.serverRules() {
		t.Error("schedule and countries are the session's")
	}
	// Hosted forms keep only the schedule.
	hosted, err := parseTargeting(ctx, in, KindHosted)
	if err != nil || hosted.Pages.Mode != "" || hosted.Countries.Mode != "" || hosted.Schedule.EndsAt == nil {
		t.Fatalf("hosted: %+v %v", hosted, err)
	}
	// Nothing set is no targeting; every device is no device rule.
	empty, err := parseTargeting(ctx, &SignupFormTargeting{Pages: SignupFormPageRules{Mode: "all"}, Sources: SignupFormSources{Referrer: "any"}, Devices: []string{"desktop", "tablet", "mobile"}, Countries: SignupFormCountries{Mode: "all"}, Visitors: "all"}, KindEmbedded)
	if err != nil || empty != nil {
		t.Fatalf("empty: %+v %v", empty, err)
	}
	bad := map[string]*SignupFormTargeting{
		"ends before it starts": {Schedule: SignupFormSchedule{StartsAt: &end, EndsAt: &start}},
		"only, no rule":         {Pages: SignupFormPageRules{Mode: "only"}},
		"a bad country":         {Countries: SignupFormCountries{Mode: "only", Codes: []string{"USA"}}},
		"a bad utm value":       {Sources: SignupFormSources{UTMSource: []string{"<script>"}}},
	}
	for name, b := range bad {
		if _, err := parseTargeting(ctx, b, KindEmbedded); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if (Config{}).Open(start) != true || !(Config{}).CountryShown("") {
		t.Error("no targeting shows everywhere, always")
	}
}

// The same browser always gets the same variant, and the split holds
// within 2% over 10,000 browsers.
func TestPickVariant(t *testing.T) {
	form := uuid.MustParse("019a1b2c-3d4e-7f00-8a9b-0c1d2e3f4a5b")
	for _, split := range [][]int32{{50, 50}, {20, 30, 50}} {
		vs := []store.FormVariant{}
		for i, w := range split {
			vs = append(vs, store.FormVariant{Key: variantKeys[i], Weight: w})
		}
		got := map[string]int{}
		const n = 10000
		for i := range n {
			vid := fmt.Sprintf("visitor-%08d-%04x", i, i*7919)
			k := PickVariant(form, vid, vs)
			if PickVariant(form, vid, vs) != k {
				t.Fatalf("%s: not stable", vid)
			}
			got[k]++
		}
		for i, w := range split {
			share := float64(got[variantKeys[i]]) / n * 100
			if math.Abs(share-float64(w)) > 2 {
				t.Errorf("split %v: %s got %.1f%%", split, variantKeys[i], share)
			}
		}
	}
	if PickVariant(form, "x", nil) != "" {
		t.Error("no variants: no variant")
	}
}

func TestVariantApply(t *testing.T) {
	base := contentDoc{Heading: "A", Button: "Subscribe", Footnote: "keep", PopupDelay: 8, Steps: []stepDoc{{ID: "s1"}, {ID: "s2"}}}
	v := variantDoc{Heading: "B", Button: "Join", PopupScroll: 40}
	got := v.apply(base)
	if got.Heading != "B" || got.Button != "Join" || got.Footnote != "keep" || got.PopupDelay != 0 || got.PopupScroll != 40 || got.Steps != nil {
		t.Fatalf("apply: %+v", got)
	}
	if !sameVariant(variantOfContent(base), variantOfContent(base)) || sameVariant(variantOfContent(base), v) {
		t.Error("sameVariant")
	}
}
