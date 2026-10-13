package demo

import (
	"context"
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

func init() { seedAudit = seedAuditImpl }

const (
	uaChromeMac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 26_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	uaSafariMac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 26_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15"
	uaClaude    = "Claude-Desktop/1.4 (MCP streamable-http)"
)

type auditSeed struct {
	at                                   time.Time
	actorType                            string
	actor                                string // user email, key name or "" for the system
	action, targetType, targetID, target string
	result, reason                       string
	ip, ua, request                      string
	before, after                        any
	meta                                 map[string]any
}

// seedAuditImpl writes the audit history of the prototype: the sample
// events of A5 §4 at the times of the SettingsAudit board, the agents'
// recent actions, Daniel's activity, the routing change with before/after,
// and a year of background history so the totals look lived in.
func seedAuditImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	m := func(kv ...any) map[string]any {
		out := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}
	camp := func(name string) string {
		if id, ok := s.Campaigns[name]; ok {
			return id.String()
		}
		return ""
	}
	routeBefore := map[string]any{"stream": "broadcasts", "weights": map[string]any{"ses_us_east_1": 95, "postal_mta": 5}, "failover": "pause_and_ask", "conditions": []any{}}
	routeAfter := map[string]any{"stream": "broadcasts", "weights": map[string]any{"ses_us_east_1": 90, "postal_mta": 10}, "failover": "pause_and_ask", "conditions": []any{}}
	events := []auditSeed{
		// Sun 12 Oct (today in the prototype), newest first.
		{at: s.Ago(2*time.Minute + 29*time.Second), actorType: "user", actor: "daniel@acme.io", action: "approval.requested", targetType: "campaign", targetID: camp("November Product Update"), target: "November Product Update",
			ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "47,912 recipients · rules: audience ≥ 10,000, warming-up provider, author lacks send", "detail", "47,912 recipients", "permission", "campaigns:edit")},
		{at: s.Ago(5*time.Minute + 58*time.Second), actorType: "user", actor: "linh@acme.io", action: "apikey.created", targetType: "api_key", target: "Website signup form",
			ip: "203.0.113.9", ua: uaChromeMac, after: map[string]any{"name": "Website signup form", "permissions": []string{"subscribers:create"}, "scope": "Weekly Digest only", "expires_at": nil},
			meta: m("summary", "subscribers:create · Weekly Digest only · no expiry", "target", "API key “Website signup form”", "permission", "apikeys:manage")},
		{at: s.Ago(10*time.Minute + 12*time.Second), actorType: "mcp_agent", actor: "Claude — analytics", action: "report.read", targetType: "campaign", targetID: camp("October Product Update"), target: "October Product Update",
			ip: "203.0.113.9", ua: uaClaude, meta: m("summary", "MCP agent “Claude — analytics” · tool get_campaign_report", "tool", "get_campaign_report", "source", "mcp", "via", "rc_agent_7a…c2",
				"outcome", "October Product Update · full report read", "permission", "reports:view")},
		{at: s.Ago(23*time.Minute + 30*time.Second), actorType: "mcp_agent", actor: "Ops agent", action: "campaign.resumed", targetType: "campaign", targetID: camp("October Product Update"), target: "October Product Update",
			result: "denied", reason: "missing campaigns:send", ip: "10.0.7.3", ua: "ops-runbook/2.3 (mcp-go)",
			meta: m("summary", "MCP agent “Ops agent” · tool resume_campaign", "tool", "resume_campaign", "source", "mcp", "via", "rc_agent_e0…44",
				"note", "needs campaigns:send · proposal recorded for Linh Nguyen", "outcome", "Denied: resuming needs campaigns:send; proposal recorded for Linh Nguyen", "permission", "campaigns:send")},
		{at: s.Ago(25*time.Minute + 46*time.Second), actorType: "system", action: "campaign.auto_paused", targetType: "campaign", targetID: camp("October Product Update"), target: "October Product Update",
			meta: m("summary", "complaint rate 0.18% over limit 0.10% · 7 of 3,980 · 2,522 recipients held", "target", "October Product Update · Outlook route", "source", "guard", "via", "threshold")},
		{at: s.Ago(25*time.Minute + 50*time.Second), actorType: "mcp_agent", actor: "Ops agent", action: "campaign.paused", targetType: "campaign", targetID: camp("October Product Update"), target: "October Product Update",
			ip: "10.0.7.3", ua: "ops-runbook/2.3 (mcp-go)", before: map[string]any{"status": "sending"}, after: map[string]any{"status": "paused"},
			meta: m("summary", "MCP agent “Ops agent” · tool pause_campaign", "tool", "pause_campaign", "source", "mcp", "via", "rc_agent_e0…44", "detail", "Outlook route",
				"note", "campaigns:pause", "outcome", "Paused October Product Update on the Outlook route · complaint rate 0.18%", "permission", "campaigns:pause")},
		{at: s.Ago(26*time.Minute + 30*time.Second), actorType: "mcp_agent", actor: "Ops agent", action: "provider.health_read", targetType: "provider", target: "All providers",
			ip: "10.0.7.3", ua: "ops-runbook/2.3 (mcp-go)",
			meta: m("summary", "MCP agent “Ops agent” · tool get_provider_health", "tool", "get_provider_health", "source", "mcp", "via", "rc_agent_e0…44",
				"outcome", "All providers · Postal degraded, SES and Postmark healthy", "permission", "providers:view")},
		{at: s.Ago(54*time.Minute + 50*time.Second), actorType: "user", actor: "aisha@acme.io", action: "subscribers.export", targetType: "contacts", target: "Contacts",
			result: "denied", reason: "missing subscribers:export", ip: "198.51.100.77", ua: uaSafariMac,
			meta: m("target", "Contacts · 48,213 rows", "permission", "subscribers:export")},
		{at: s.Ago(66*time.Minute + 5*time.Second), actorType: "user", actor: "marcus@acme.io", action: "routing.updated", targetType: "routing_rule", targetID: "route_broadcasts", target: "Broadcasts",
			ip: "198.51.100.40", ua: uaChromeMac, request: "req_8c1f2a9d", before: routeBefore, after: routeAfter,
			meta: m("summary", "SES 95% / Postal 5% → SES 90% / Postal 10%", "target", "Routing rule · Broadcasts", "what", "Changed the Broadcasts routing rule",
				"related", "warm-up plan step 9 raised the Postal cap to 4,800/day at 08:00", "permission", "providers:manage")},
		{at: s.designDay(0, 6, 40), actorType: "user", actor: "daniel@acme.io", action: "campaign.updated", targetType: "campaign", targetID: camp("November Product Update"), target: "November Product Update",
			ip: "203.0.113.9", ua: uaChromeMac, before: map[string]any{"subject": "Product update: November"}, after: map[string]any{"subject": "What's new in November: audit exports and faster sends"},
			meta: m("detail", "subject, hero section", "summary", "subject, hero section · version 7", "permission", "campaigns:edit")},
		// Sat 11 Oct.
		{at: s.designDay(1, 17, 20), actorType: "user", actor: "marcus@acme.io", action: "routing.updated", targetType: "routing_rule", targetID: "route_broadcasts", target: "Broadcasts",
			ip: "198.51.100.40", ua: uaChromeMac, before: map[string]any{"stream": "broadcasts", "fallback": nil}, after: map[string]any{"stream": "broadcasts", "fallback": "postmark"},
			meta: m("summary", "fallback set to Postmark · warning shown: terms forbid bulk · reverted 17:22", "target", "Routing rule · Broadcasts", "permission", "providers:manage")},
		{at: s.designDay(1, 15, 2), actorType: "user", actor: "daniel@acme.io", action: "template.created", targetType: "template", target: "“Product update v2”",
			ip: "203.0.113.9", ua: uaChromeMac, meta: m("permission", "templates:edit")},
		// Fri 10 Oct.
		{at: s.designDay(2, 16, 20), actorType: "user", actor: "marcus@acme.io", action: "campaign.approved", targetType: "campaign", targetID: camp("Weekly Digest #142"), target: "Weekly Digest #142",
			ip: "198.51.100.40", ua: uaChromeMac, meta: m("summary", "snapshot v6 · 31,604 recipients", "permission", "campaigns:approve")},
		{at: s.designDay(2, 11, 41), actorType: "user", actor: "linh@acme.io", action: "team.invited", targetType: "invitation", target: "jo@contractor.dev",
			ip: "203.0.113.9", ua: uaChromeMac, after: map[string]any{"email": "jo@contractor.dev", "role": "Editor", "scope": "Beta Program list only"},
			meta: m("summary", "role Editor · scope Beta Program · expires Oct 17 · sent via Postmark", "permission", "team:invite")},
		// Thu 9 Oct.
		{at: s.designDay(3, 10, 51), actorType: "system", action: "campaign.completed", targetType: "campaign", targetID: camp("October Product Update"), target: "October Product Update",
			meta: m("summary", "47,980 sent · SES 43,182 · Postal 4,798 · 99.1% delivered · $4.32", "source", "send", "via", "worker")},
		// Wed 8 Oct: Daniel tried to schedule without campaigns:send.
		{at: s.designDay(4, 14, 12), actorType: "user", actor: "daniel@acme.io", action: "campaign.schedule", targetType: "campaign", targetID: camp("November Product Update"), target: "November Product Update",
			result: "denied", reason: "missing campaigns:send", ip: "203.0.113.9", ua: uaChromeMac, meta: m("permission", "campaigns:send")},
		// Mon 6 Oct.
		{at: s.designDay(6, 8, 2), actorType: "mcp_agent", actor: "Claude — analytics", action: "deliverability.read", targetType: "sending_domain", target: "news.acme.io",
			ip: "203.0.113.9", ua: uaClaude, meta: m("summary", "MCP agent “Claude — analytics” · tool get_deliverability", "tool", "get_deliverability", "source", "mcp", "via", "rc_agent_7a…c2",
				"outcome", "news.acme.io · clean on 84 blocklists · Postmaster high", "permission", "deliverability:view")},
	}
	for _, e := range events {
		if err := s.insertAudit(ctx, ws.ID, e); err != nil {
			return err
		}
	}
	return s.backgroundAudit(ctx)
}

func (s *Seeder) actor(e auditSeed) (*uuid.UUID, string) {
	switch e.actorType {
	case "user":
		u := s.Users[e.actor]
		return &u.ID, u.Name
	case "api_key", "mcp_agent":
		if id, ok := s.Extra["key:"+e.actor].(uuid.UUID); ok {
			return &id, e.actor
		}
		return nil, e.actor
	}
	return nil, "System"
}

func (s *Seeder) insertAudit(ctx context.Context, wsID uuid.UUID, e auditSeed) error {
	actorID, name := s.actor(e)
	var ip *netip.Addr
	if e.ip != "" {
		a := netip.MustParseAddr(e.ip)
		ip = &a
	}
	result := e.result
	if result == "" {
		result = "allowed"
	}
	j := func(v any) json.RawMessage {
		if v == nil {
			return nil
		}
		b, _ := json.Marshal(v)
		return b
	}
	meta := e.meta
	if meta == nil {
		meta = map[string]any{}
	}
	req := e.request
	if req == "" && e.actorType != "system" {
		req = "req_" + ID().String()[24:32]
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
		result, reason, ip, user_agent, request_id, before, after, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		ID(), wsID, e.at, e.actorType, actorID, name, e.action, e.targetType, e.targetID, e.target, result, e.reason, ip, e.ua, req, j(e.before), j(e.after), j(meta))
	return err
}

// backgroundAudit adds a year of everyday history (older than the last
// seven days) so the log has the size of a lived-in workspace.
func (s *Seeder) backgroundAudit(ctx context.Context) error {
	ws := s.WS()
	var users []uuid.UUID
	var names []string
	for _, email := range []string{"linh@acme.io", "marcus@acme.io", "sofia@acme.io", "daniel@acme.io"} {
		users = append(users, s.Users[email].ID)
		names = append(names, s.Users[email].Name)
	}
	var existing int
	if err := s.Tx.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id = $1`, ws.ID).Scan(&existing); err != nil {
		return err
	}
	n := 41208 - existing
	if n <= 0 {
		return nil
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
			result, reason, ip, user_agent, request_id, metadata)
		SELECT gen_random_uuid(), $1, $2::timestamptz - (g * (350.0 * 86400 / $3)) * interval '1 second',
		       CASE WHEN g % 9 = 0 THEN 'system' ELSE 'user' END,
		       CASE WHEN g % 9 = 0 THEN NULL ELSE ($4::uuid[])[1 + g % 4] END,
		       CASE WHEN g % 9 = 0 THEN 'System' ELSE ($5::text[])[1 + g % 4] END,
		       (ARRAY['campaign.updated','contact.updated','campaign.created','template.updated','segment.updated','auth.signed_in','campaign.completed','list.updated','form.updated'])[1 + g % 9],
		       (ARRAY['campaign','contact','campaign','template','segment','user','campaign','list','form'])[1 + g % 9],
		       '', (ARRAY['Weekly Digest #' || (140 - g % 90)::text, 'contact', 'Draft', 'Weekly Digest', 'Engaged last 90 days', 'Sign-in', 'Weekly Digest #' || (140 - g % 90)::text, 'Product Updates', 'Signup form'])[1 + g % 9],
		       'allowed', '', CASE WHEN g % 9 = 0 THEN NULL ELSE '203.0.113.9'::inet END, '', '', '{}'::jsonb
		FROM generate_series(1, $3::int) g`,
		ws.ID, s.Ago(8*24*time.Hour), n, users, names)
	return err
}
