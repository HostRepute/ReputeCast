package segments

import (
	"strings"
	"testing"
)

// "Signed in to the website" (WebsiteStates): since a date or a number of
// days ago, not since, ever and never.
func TestSignedInCondition(t *testing.T) {
	cases := []struct {
		c    SegmentCondition
		want string
		bad  bool
	}{
		{c: SegmentCondition{Field: "signed_in", Op: "since", Value: "30d"}, want: "c.signed_in_at >= now() - interval '30 days'"},
		{c: SegmentCondition{Field: "signed_in", Op: "since", Value: "2026-09-29"}, want: "c.signed_in_at >= '2026-09-29'::date"},
		{c: SegmentCondition{Field: "signed_in", Op: "not_since", Value: "7d"}, want: "(c.signed_in_at IS NULL OR c.signed_in_at < now() - interval '7 days')"},
		{c: SegmentCondition{Field: "signed_in", Op: "ever"}, want: "c.signed_in_at IS NOT NULL"},
		{c: SegmentCondition{Field: "signed_in", Op: "never"}, want: "c.signed_in_at IS NULL"},
		{c: SegmentCondition{Field: "signed_in", Op: "since", Value: "last week"}, bad: true},
		{c: SegmentCondition{Field: "signed_in", Op: "since", Value: "0d"}, bad: true},
		{c: SegmentCondition{Field: "signed_in", Op: "in_last", Value: "7d"}, bad: true},
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
	// The contacts view carries the column.
	if !strings.Contains(withViews(), "wm.last_signed_in AS signed_in_at") || !strings.Contains(withViews(), "public.website_members wm") {
		t.Fatal("the contacts view has no signed_in_at")
	}
}
