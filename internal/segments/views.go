package segments

import (
	"strings"

	"github.com/google/uuid"
)

// The segment views are what builder SQL and user-written SQL read: one
// workspace's contacts (custom fields as `fields`, the suppression date as
// `suppressed_at`), list memberships, delivery and engagement events, and
// lists. Builder segments get them as CTEs with the workspace bound as $1;
// SQL segments get them as temporary views created for one query.
var views = []struct{ name, body string }{
	{"contacts", `SELECT c.id, c.workspace_id, c.email, c.first_name, c.last_name, c.company, c.country, c.timezone,
           c.language, c.mailbox_provider, c.tier, c.consent, c.source, c.status, c.tags, c.attributes AS fields,
           c.role_address, c.created_at, c.last_open_at, c.last_human_open_at, c.last_click_at,
           s.created_at AS suppressed_at,` + scoringColumns + `
    FROM public.contacts c
    LEFT JOIN public.suppressions s ON s.workspace_id = c.workspace_id AND s.email = c.email` + scoringJoins + `
    WHERE c.workspace_id = {ws} AND c.erased_at IS NULL`},
	{"list_members", `SELECT lm.contact_id, lm.list_id, l.name AS list, lm.status, lm.subscribed_at, lm.unsubscribed_at
    FROM public.list_memberships lm
    JOIN public.lists l ON l.id = lm.list_id
    WHERE l.workspace_id = {ws} AND l.deleted_at IS NULL`},
	{"events", `SELECT de.contact_id, de.type, NOT de.machine AS human, de.occurred_at, de.campaign_id, de.url,
           de.mailbox_provider
    FROM public.delivery_events de
    WHERE de.workspace_id = {ws}`},
	{"lists", `SELECT l.id, l.name FROM public.lists l WHERE l.workspace_id = {ws} AND l.deleted_at IS NULL`},
}

// ViewNames are the relations a SQL segment may read.
var ViewNames = []string{"contacts", "list_members", "events", "lists"}

// withViews is the CTE prelude for builder queries; $1 is the workspace.
func withViews() string {
	var sb strings.Builder
	sb.WriteString("WITH ")
	for i, v := range views {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(v.name + " AS NOT MATERIALIZED (\n    " + strings.ReplaceAll(v.body, "{ws}", "$1") + "\n)")
	}
	sb.WriteString("\n")
	return sb.String()
}

// tempViews returns the statements that create the views for a SQL segment.
func tempViews(ws uuid.UUID) []string {
	lit := "'" + ws.String() + "'::uuid"
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, "CREATE OR REPLACE TEMP VIEW "+v.name+" AS "+strings.ReplaceAll(v.body, "{ws}", lit))
	}
	return out
}

// tierCounts is the select list that splits a count by engagement tier.
const tierCounts = `count(*)::bigint AS n,
       count(*) FILTER (WHERE x.tier = 'active')::bigint AS active,
       count(*) FILTER (WHERE x.tier = 'cooling')::bigint AS cooling,
       count(*) FILTER (WHERE x.tier = 'at_risk')::bigint AS at_risk,
       count(*) FILTER (WHERE x.tier = 'sunset')::bigint AS sunset`
