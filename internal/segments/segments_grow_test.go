package segments

import (
	"strings"
	"testing"
)

// Grow's conditions (brief §3.7): referrals counted at least, at most or
// between; a referral milestone reached or not; joined through a referral
// or a partner, any or one.
func TestGrowConditions(t *testing.T) {
	const milestone = "01a15090-09b6-750c-bace-c1af25dcf851"
	const partner = "01a15090-0aa1-7d22-9b55-111111111111"
	cases := []struct {
		c    SegmentCondition
		want string
		bad  bool
	}{
		{c: SegmentCondition{Field: "referrals_counted", Op: "at_least", Value: "3"}, want: "c.referrals_counted >= 3"},
		{c: SegmentCondition{Field: "referrals_counted", Op: "at_most", Value: "0"}, want: "c.referrals_counted <= 0"},
		{c: SegmentCondition{Field: "referrals_counted", Op: "between", Value: "1", To: "9"}, want: "c.referrals_counted BETWEEN 1 AND 9"},
		{c: SegmentCondition{Field: "referrals_counted", Op: "between", Value: "9", To: "1"}, bad: true},
		{c: SegmentCondition{Field: "referrals_counted", Op: "at_least", Value: "many"}, bad: true},
		{c: SegmentCondition{Field: "referrals_counted", Op: "is", Value: "3"}, bad: true},
		{c: SegmentCondition{Field: "referral_milestone", Op: "reached", Value: milestone}, want: "('" + milestone + "'::uuid = ANY (c.referral_milestones))"},
		{c: SegmentCondition{Field: "referral_milestone", Op: "not_reached", Value: milestone}, want: "NOT ('" + milestone + "'::uuid = ANY (c.referral_milestones))"},
		{c: SegmentCondition{Field: "referral_milestone", Op: "reached", Value: "3"}, bad: true},
		{c: SegmentCondition{Field: "joined_through", Op: "is", Value: "referral"}, want: "coalesce(c.growth_source = 'referral' OR c.growth_source LIKE 'referral:%', false)"},
		{c: SegmentCondition{Field: "joined_through", Op: "is_not", Value: "partner"}, want: "NOT coalesce(c.growth_source = 'partner' OR c.growth_source LIKE 'partner:%', false)"},
		{c: SegmentCondition{Field: "joined_through", Op: "is", Value: "partner:" + partner}, want: "coalesce(c.growth_source = 'partner:" + partner + "', false)"},
		{c: SegmentCondition{Field: "joined_through", Op: "is", Value: "partner:ops"}, bad: true},
		{c: SegmentCondition{Field: "joined_through", Op: "is", Value: "import"}, bad: true},
	}
	for _, tc := range cases {
		d := SegmentDefinition{Groups: []SegmentGroup{{Match: "all", Conditions: []SegmentCondition{tc.c}}}}
		err := d.Validate(custom, lists)
		if tc.bad {
			if err == nil {
				t.Errorf("%+v: no error", tc.c)
			}
			continue
		}
		if err != nil {
			t.Errorf("%+v: %v", tc.c, err)
			continue
		}
		got := Compile(d, wsID, custom).Display()
		if !strings.Contains(got, tc.want) {
			t.Errorf("%+v:\n%s\nwant %s", tc.c, got, tc.want)
		}
	}
	// The contacts view carries the columns; the fields follow the growth
	// fields in the builder.
	if !strings.Contains(withViews(), "AS referrals_counted") || !strings.Contains(withViews(), "AS referral_milestones") {
		t.Fatal("the contacts view has no referral columns")
	}
	at := func(k string) int {
		for i, f := range builtinOrder {
			if f == k {
				return i
			}
		}
		return -1
	}
	if a, b := at("referring_domain"), at("joined_through"); a < 0 || b != a+1 || at("referrals_counted") != b+1 || at("referral_milestone") != b+2 {
		t.Fatalf("builder order: %v", builtinOrder)
	}
}
