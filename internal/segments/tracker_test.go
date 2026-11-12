package segments

import (
	"strings"
	"testing"
)

// The site tracker conditions (M62): viewed a product in the last N days,
// any product or one of a store's collections, and last seen on a tracked
// site.
func TestTrackerConditions(t *testing.T) {
	const acme = "01a153d3-7695-7d56-bd38-921860fa2158"
	cases := []struct {
		c    SegmentCondition
		want []string
		bad  bool
	}{
		{c: SegmentCondition{Field: "site_viewed_product", Op: "in_last", Amount: 7, Unit: "days"}, want: []string{"coalesce(c.site_viewed_at >= now() - interval '7 days', false)"}},
		{c: SegmentCondition{Field: "site_viewed_product", Op: "in_last", Amount: 7, Unit: "days", Value: "any"}, want: []string{"c.site_viewed_at >= "}},
		{c: SegmentCondition{Field: "site_viewed_product", Op: "not_in_last", Amount: 2, Unit: "weeks", Value: "collection:" + acme + ":504"},
			want: []string{"NOT EXISTS (SELECT 1 FROM public.tracker_product_views pv JOIN public.store_products p", "pv.last_viewed_at >= now() - interval '2 weeks'",
				"p.store_id = '" + acme + "'", "'504' = ANY (p.collections)"}},
		{c: SegmentCondition{Field: "site_viewed_product", Op: "in_last", Amount: 7, Unit: "days", Value: "hoodies"}, bad: true},
		{c: SegmentCondition{Field: "site_viewed_product", Op: "in_last", Amount: 0, Unit: "days"}, bad: true},
		{c: SegmentCondition{Field: "site_viewed_product", Op: "never"}, bad: true},
		{c: SegmentCondition{Field: "site_last_seen", Op: "in_last", Amount: 30, Unit: "days"}, want: []string{"c.site_last_seen_at >= now() - interval '30 days'"}},
		{c: SegmentCondition{Field: "site_last_seen", Op: "not_in_last", Amount: 30, Unit: "days"}, want: []string{"(c.site_last_seen_at IS NULL OR c.site_last_seen_at < now() - interval '30 days')"}},
		{c: SegmentCondition{Field: "site_last_seen", Op: "never"}, want: []string{"c.site_last_seen_at IS NULL"}},
		{c: SegmentCondition{Field: "site_last_seen", Op: "in_last", Amount: 3, Unit: "months"}, bad: true},
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
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("%+v:\n%s\nwant %s", tc.c, got, w)
			}
		}
	}
	// The contacts view reads both moments.
	if !strings.Contains(withViews(), "AS site_viewed_at") || !strings.Contains(withViews(), "AS site_last_seen_at") {
		t.Error("the contacts view lacks the site tracker's columns")
	}
}
