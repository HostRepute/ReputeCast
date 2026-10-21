package segments

import (
	"strings"
	"testing"
)

// "Push subscribed" on a site or in a browser, and "WhatsApp consent" on a
// list or any (X22): EXISTS over the push_subscriptions and
// whatsapp_consents views, which SQL segments can read too.
func TestPushWhatsAppConditions(t *testing.T) {
	site := "0199c0de-7f1a-7c3e-9b2a-000000000001"
	cases := []struct {
		c    SegmentCondition
		want string
		bad  bool
	}{
		{c: SegmentCondition{Field: FieldPushSite, Op: "is", Value: "any"}, want: "EXISTS (SELECT 1 FROM push_subscriptions x WHERE x.contact_id = c.id)"},
		{c: SegmentCondition{Field: FieldPushSite, Op: "is", Value: site}, want: "x.site_id = '" + site + "'::uuid)"},
		{c: SegmentCondition{Field: FieldPushSite, Op: "is_not", Value: "any"}, want: "NOT EXISTS (SELECT 1 FROM push_subscriptions x"},
		{c: SegmentCondition{Field: FieldPushBrowser, Op: "is", Value: "safari_ios"}, want: "x.browser = 'safari_ios')"},
		{c: SegmentCondition{Field: FieldWhatsAppConsent, Op: "is", Value: "any"}, want: "EXISTS (SELECT 1 FROM whatsapp_consents x WHERE x.contact_id = c.id AND x.status = 'subscribed')"},
		{c: SegmentCondition{Field: FieldWhatsAppConsent, Op: "is", Value: digest.String()}, want: "x.list_id = '" + digest.String() + "'::uuid AND x.status = 'subscribed')"},
		{c: SegmentCondition{Field: FieldPushSite, Op: "is", Value: "news.acme.io"}, bad: true},
		{c: SegmentCondition{Field: FieldPushBrowser, Op: "is", Value: "netscape"}, bad: true},
		{c: SegmentCondition{Field: FieldWhatsAppConsent, Op: "is", Value: ""}, bad: true},
		{c: SegmentCondition{Field: FieldWhatsAppConsent, Op: "since", Value: "any"}, bad: true},
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
		if got := Compile(d, wsID, custom).Display(); !strings.Contains(got, tc.want) {
			t.Errorf("%+v:\n%s\nwant %s", tc.c, got, tc.want)
		}
	}
	for _, v := range []string{"push_subscriptions AS NOT MATERIALIZED", "whatsapp_consents AS NOT MATERIALIZED", "s.contact_id IS NOT NULL"} {
		if !strings.Contains(withViews(), v) {
			t.Fatalf("the views have no %q", v)
		}
	}
}
