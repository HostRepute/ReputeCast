package segments

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Web push and WhatsApp conditions (design expansion phase 8, X22; brief §6
// segment builder): "Push subscribed" on a site or in a browser — linked
// subscriptions only, since anonymous ones aren't contacts — and "WhatsApp
// consent" on a list or any. They read the segment views
// push_subscriptions and whatsapp_consents, so the generated SQL also runs
// as a SQL segment.

// The push and WhatsApp fields.
const (
	FieldPushSite        = "push_site"
	FieldPushBrowser     = "push_browser"
	FieldWhatsAppConsent = "whatsapp_consent"
)

func init() {
	builtinFields[FieldPushSite] = fieldSpec{"Push subscribed · site", KindEnum, []string{"is", "is_not"}}
	builtinFields[FieldPushBrowser] = fieldSpec{"Push subscribed · browser", KindEnum, []string{"is", "is_not"}}
	builtinFields[FieldWhatsAppConsent] = fieldSpec{"WhatsApp consent", KindEnum, []string{"is", "is_not"}}
	enumValues[FieldPushBrowser] = [][2]string{{"any", "Any browser"}, {"chrome", "Chrome"}, {"chrome_android", "Chrome on Android"}, {"edge", "Edge"},
		{"firefox", "Firefox"}, {"safari", "Safari on macOS"}, {"safari_ios", "Safari on iPhone and iPad"}, {"other", "Other"}}
	builtinOrder = append(builtinOrder, FieldPushSite, FieldPushBrowser, FieldWhatsAppConsent)
	extraValidate[FieldPushSite] = validateAnyOrID("Pick a push site.")
	extraValidate[FieldWhatsAppConsent] = validateAnyOrID("Pick a list.")
	extraCompile[FieldPushSite] = compilePushSite
	extraCompile[FieldPushBrowser] = compilePushBrowser
	extraCompile[FieldWhatsAppConsent] = compileWhatsAppConsent
	RegisterFieldOptions(FieldPushSite, pushSiteOptions)
	RegisterFieldOptions(FieldWhatsAppConsent, whatsAppListOptions)
	views = append(views,
		struct{ name, body string }{"push_subscriptions", `SELECT s.contact_id, s.site_id, st.host AS site, s.browser, s.created_at
    FROM public.push_subscriptions s
    JOIN public.push_sites st ON st.id = s.site_id AND st.removed_at IS NULL
    WHERE s.workspace_id = {ws} AND s.removed_at IS NULL AND s.contact_id IS NOT NULL`},
		struct{ name, body string }{"whatsapp_consents", `SELECT cc.contact_id, cc.list_id, cc.status, cc.changed_at
    FROM public.channel_consents cc
    WHERE cc.workspace_id = {ws} AND cc.channel = 'whatsapp'`},
	)
	ViewNames = append(ViewNames, "push_subscriptions", "whatsapp_consents")
}

func validateAnyOrID(msg string) func(SegmentCondition) error {
	return func(c SegmentCondition) error {
		if c.Value == "any" {
			return nil
		}
		if _, err := uuid.Parse(c.Value); err != nil {
			return problemf("%s", msg)
		}
		return nil
	}
}

// pushSiteOptions are "Any push site" and the workspace's push sites.
func pushSiteOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]SegmentOption, error) {
	sites, err := q.PushSites(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := []SegmentOption{{Value: "any", Label: "Any push site"}}
	for _, s := range sites {
		out = append(out, SegmentOption{Value: s.ID.String(), Label: s.Host})
	}
	return out, nil
}

// whatsAppListOptions are "Any list" and the workspace's lists.
func whatsAppListOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]SegmentOption, error) {
	lists, err := q.ListWorkspaceLists(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := []SegmentOption{{Value: "any", Label: "Any list"}}
	for _, l := range lists {
		out = append(out, SegmentOption{Value: l.ID.String(), Label: l.Name})
	}
	return out, nil
}

// exists writes EXISTS (NOT EXISTS for is_not) over a view, narrowed to
// the condition's value unless it's "any"; the caller closes it.
func exists(t *sqlText, c SegmentCondition, from, col, cast string) {
	if c.Op == "is_not" {
		t.raw("NOT ")
	}
	t.raw("EXISTS (SELECT 1 FROM " + from + " x WHERE x.contact_id = c.id")
	if c.Value != "any" {
		t.raw(" AND x." + col + " = ")
		t.bind(str(c.Value))
		t.raw(cast)
	}
}

func compilePushSite(t *sqlText, c SegmentCondition) {
	exists(t, c, "push_subscriptions", "site_id", "::uuid")
	t.raw(")")
}

func compilePushBrowser(t *sqlText, c SegmentCondition) {
	exists(t, c, "push_subscriptions", "browser", "")
	t.raw(")")
}

func compileWhatsAppConsent(t *sqlText, c SegmentCondition) {
	exists(t, c, "whatsapp_consents", "list_id", "::uuid")
	t.raw(" AND x.status = 'subscribed')")
}
