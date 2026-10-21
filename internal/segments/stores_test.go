package segments

import (
	"strings"
	"testing"
)

// The store conditions (spec §6, segment builder): a store's customers,
// their orders, the total spent, the last order, a product or a collection
// bought, and the marketing consent the store recorded.
func TestStoreConditions(t *testing.T) {
	const acme = "01a153d3-7695-7d56-bd38-921860fa2158"
	cases := []struct {
		c    SegmentCondition
		want []string
		bad  bool
	}{
		{c: SegmentCondition{Field: "store_customer", Op: "is", Value: "any"}, want: []string{"EXISTS (SELECT 1 FROM public.store_customers sc WHERE sc.contact_id = c.id)", "OR EXISTS (SELECT 1 FROM public.store_orders so"}},
		{c: SegmentCondition{Field: "store_customer", Op: "is_not", Value: acme}, want: []string{"NOT (EXISTS", "sc.store_id = '" + acme + "'"}},
		{c: SegmentCondition{Field: "store_customer", Op: "is", Value: "Acme Merch"}, bad: true},
		{c: SegmentCondition{Field: "store_marketing", Op: "is", Value: acme}, want: []string{"sc.marketing = 'subscribed'", "sc.store_id = '" + acme + "'"}},
		{c: SegmentCondition{Field: "store_orders", Op: "at_least", Value: "2"}, want: []string{"(SELECT count(*) FROM public.store_orders so WHERE so.contact_id = c.id AND NOT so.test AND NOT so.cancelled) >= 2"}},
		{c: SegmentCondition{Field: "store_orders", Op: "between", Value: "3", To: "1"}, bad: true},
		{c: SegmentCondition{Field: "store_orders", Op: "at_least", Value: "two"}, bad: true},
		// Amounts are minor units: $250.50 is 25050.
		{c: SegmentCondition{Field: "store_spent", Op: "between", Value: "100", To: "250.50"}, want: []string{"sum(so.total_cents - so.refunded_cents)", "BETWEEN 10000 AND 25050"}},
		{c: SegmentCondition{Field: "store_spent", Op: "at_most", Value: "-5"}, bad: true},
		{c: SegmentCondition{Field: "store_last_order", Op: "in_last", Amount: 30, Unit: "days"}, want: []string{"EXISTS (SELECT 1 FROM public.store_orders so", "so.placed_at >= "}},
		{c: SegmentCondition{Field: "store_last_order", Op: "never"}, want: []string{"NOT EXISTS (SELECT 1 FROM public.store_orders so"}},
		{c: SegmentCondition{Field: "store_last_order", Op: "in_last", Amount: 0, Unit: "days"}, bad: true},
		{c: SegmentCondition{Field: "store_bought", Op: "is", Value: "product:" + acme + ":8841203"}, want: []string{"it ->> 'productId' = '8841203'"}},
		{c: SegmentCondition{Field: "store_bought", Op: "is_not", Value: "collection:" + acme + ":504"}, want: []string{"NOT EXISTS", "'504' = ANY (p.collections)"}},
		{c: SegmentCondition{Field: "store_bought", Op: "is", Value: "hoodie"}, bad: true},
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
}
