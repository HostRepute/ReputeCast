package segments

import (
	"strings"
	"testing"
)

// The SMS conditions (X21): a phone number, its country, SMS consent per
// list, and texts received or clicked.
func TestSMSConditions(t *testing.T) {
	cases := []struct {
		c    SegmentCondition
		want string
		bad  bool
	}{
		{c: SegmentCondition{Field: "phone", Op: "is", Value: "set"}, want: "c.phone <> ''"},
		{c: SegmentCondition{Field: "phone", Op: "is", Value: "not_set"}, want: "c.phone = ''"},
		{c: SegmentCondition{Field: "phone", Op: "is", Value: "maybe"}, bad: true},
		{c: SegmentCondition{Field: "phone_country", Op: "is", Value: "gb"}, want: "c.phone_country = 'GB'"},
		{c: SegmentCondition{Field: "phone_country", Op: "is_not", Value: "US"}, want: "c.phone_country <> 'US'"},
		{c: SegmentCondition{Field: "phone_country", Op: "is", Value: " "}, bad: true},
		{c: SegmentCondition{Field: "sms_consent", Op: "includes", Value: digest.String()}, want: "EXISTS (SELECT 1 FROM public.channel_consents cc WHERE cc.contact_id = c.id AND cc.list_id = '" + digest.String() + "' AND cc.channel = 'sms' AND cc.status = 'subscribed')"},
		{c: SegmentCondition{Field: "sms_consent", Op: "excludes", Value: digest.String()}, want: "NOT EXISTS (SELECT 1 FROM public.channel_consents cc WHERE cc.contact_id = c.id AND cc.list_id = '" + digest.String() + "'"},
		{c: SegmentCondition{Field: "sms_consent", Op: "includes", Value: wsID.String()}, bad: true},
		{c: SegmentCondition{Field: "texted", Op: "in_last", Amount: 30, Unit: "days"}, want: "EXISTS (SELECT 1 FROM public.channel_messages e WHERE e.contact_id = c.id AND e.channel = 'sms' AND e.sent_at >= now() - interval '30 days')"},
		{c: SegmentCondition{Field: "texted", Op: "never"}, want: "NOT EXISTS (SELECT 1 FROM public.channel_messages e WHERE e.contact_id = c.id AND e.channel = 'sms' AND e.sent_at IS NOT NULL)"},
		{c: SegmentCondition{Field: "text_clicked", Op: "not_in_last", Amount: 2, Unit: "weeks"}, want: "NOT EXISTS (SELECT 1 FROM public.sms_link_clicks e WHERE e.contact_id = c.id AND NOT e.machine AND e.occurred_at >= now() - interval '2 weeks')"},
		{c: SegmentCondition{Field: "text_clicked", Op: "never"}, bad: true},
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
	// The contacts view carries the columns, for SQL segments.
	for _, col := range []string{"c.phone, c.phone_country", "AS sms_lists", "AS last_text_at", "AS last_text_click_at"} {
		if !strings.Contains(withViews(), col) {
			t.Fatalf("the contacts view has no %s", col)
		}
	}
}
