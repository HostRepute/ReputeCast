package segments

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Texts (design expansion phase 8, X21): whether a contact has a phone
// number and its country, SMS consent per list, and texts received or
// clicked. The contacts view reads them from the contact and the SMS
// tables, so SQL segments see them too.

func init() {
	builtinFields["phone"] = fieldSpec{"Phone number", KindEnum, []string{"is"}}
	builtinFields["phone_country"] = fieldSpec{"Phone country", KindText, []string{"is", "is_not"}}
	builtinFields["sms_consent"] = fieldSpec{"SMS consent", KindList, []string{"includes", "excludes"}}
	builtinFields["texted"] = fieldSpec{"Got a text", KindPeriod, []string{"in_last", "not_in_last", "never"}}
	builtinFields["text_clicked"] = fieldSpec{"Clicked a text's link", KindPeriod, []string{"in_last", "not_in_last"}}
	enumValues["phone"] = [][2]string{{"set", "Has a number"}, {"not_set", "Has no number"}}
	builtinOrder = insertAfter(builtinOrder, "language", "phone", "phone_country", "sms_consent", "texted", "text_clicked")
	extraCompile["phone"] = compilePhone
	extraCompile["phone_country"] = compilePhoneCountry
	extraCompile["sms_consent"] = compileSMSConsent
	extraCompile["texted"] = func(t *sqlText, c SegmentCondition) {
		compileLast(t, "public.channel_messages e WHERE e.contact_id = c.id AND e.channel = 'sms'", "e.sent_at", c)
	}
	extraCompile["text_clicked"] = func(t *sqlText, c SegmentCondition) {
		compileLast(t, "public.sms_link_clicks e WHERE e.contact_id = c.id AND NOT e.machine", "e.occurred_at", c)
	}
	extraOptions["sms_consent"] = listOptions
}

// channelColumns add the phone, the lists with SMS consent and the last
// text sent and clicked to the contacts view, for SQL segments; they cost
// nothing when a query doesn't read them. Builder conditions read the SMS
// tables themselves (semi-joins, one pass over the workspace's rows).
const channelColumns = `,
           c.phone, c.phone_country,
           (SELECT array_agg(cc.list_id) FROM public.channel_consents cc
             WHERE cc.contact_id = c.id AND cc.channel = 'sms' AND cc.status = 'subscribed') AS sms_lists,
           (SELECT max(m.sent_at) FROM public.channel_messages m WHERE m.contact_id = c.id AND m.channel = 'sms') AS last_text_at,
           (SELECT max(k.occurred_at) FROM public.sms_link_clicks k WHERE k.contact_id = c.id AND NOT k.machine) AS last_text_click_at`

func compilePhone(t *sqlText, c SegmentCondition) {
	if c.Value == "not_set" {
		t.raw("c.phone = ''")
		return
	}
	t.raw("c.phone <> ''")
}

func compilePhoneCountry(t *sqlText, c SegmentCondition) {
	op := " = "
	if c.Op == "is_not" {
		op = " <> "
	}
	t.raw("c.phone_country" + op)
	t.bind(str(strings.ToUpper(strings.TrimSpace(c.Value))))
}

func compileSMSConsent(t *sqlText, c SegmentCondition) {
	id, _ := uuid.Parse(c.Value)
	if c.Op == "excludes" {
		t.raw("NOT ")
	}
	t.raw("EXISTS (SELECT 1 FROM public.channel_consents cc WHERE cc.contact_id = c.id AND cc.list_id = ")
	t.bind(uid(id))
	t.raw(" AND cc.channel = 'sms' AND cc.status = 'subscribed')")
}

// compileLast is "in the last", "not in the last" and "never" on the time
// col of the rows from (a FROM … WHERE … over e).
func compileLast(t *sqlText, from, col string, c SegmentCondition) {
	switch c.Op {
	case "never":
		t.raw("NOT EXISTS (SELECT 1 FROM " + from + " AND " + col + " IS NOT NULL)")
	case "not_in_last":
		t.raw("NOT EXISTS (SELECT 1 FROM " + from + " AND " + col + " >= " + interval(c) + ")")
	default:
		t.raw("EXISTS (SELECT 1 FROM " + from + " AND " + col + " >= " + interval(c) + ")")
	}
}

// listOptions are the workspace's lists, for SMS consent.
func listOptions(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]SegmentOption, error) {
	lists, err := q.ListWorkspaceLists(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make([]SegmentOption, 0, len(lists))
	for _, l := range lists {
		out = append(out, SegmentOption{Value: l.ID.String(), Label: l.Name})
	}
	return out, nil
}
