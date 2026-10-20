// Package mcpserver is ReputeCast's Model Context Protocol server (main spec
// 5.13, A5 §5.13). AI agents connect over streamable HTTP at /mcp with their
// own token (rc_agent_…). Every tool checks the agent's permissions through
// internal/authz, exactly like a member or API key, and every call lands in
// the audit log with the agent as actor. Anything that would send mail needs
// a person to confirm it in ReputeCast.
package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/apikeys"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

type Service struct {
	Pool      *pgxpool.Pool
	Q         *store.Queries
	Audit     *audit.Logger
	Sending   sending.Starter
	Mail      mailer.Sender
	Campaigns *campaigns.Service
	// Languages handles campaigns' language versions; nil leaves them out.
	Languages *languages.Service
	// Accessibility reads campaigns' accessibility issues (nil: unavailable).
	Accessibility AccessibilityReader
	// PaidMembers counts paid members (internal/earn; nil: unavailable).
	PaidMembers PaidMembersReader
	// Transactional lists templates, renders previews and reads the log
	// for agents (they never send).
	Transactional *transactional.Service
	// Growth reads Audience › Growth (nil: unavailable).
	Growth GrowthReader
	// Digests reads recurring digests and their issues (nil: unavailable).
	Digests DigestReader
	// Placement reads placement tests (nil: unavailable).
	Placement PlacementReader
	PublicURL func(context.Context) string
	Now       func() time.Time
	// Polls reads poll and survey results (nil: unavailable).
	Polls PollReader
	// Website lists, reads and drafts posts (nil: unavailable).
	Website WebsitePosts
	// Revenue reads Earn › Overview and Sponsors takes proposed bookings
	// (X17; nil: unavailable).
	Revenue  RevenueReader
	Sponsors SponsorProposer

	limiters sync.Map // agent id → *limiter
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// MCPToolInfo describes a tool for the settings screen and the docs.
type MCPToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Permission  string `json:"permission"`
	// Writes is true for tools that change something.
	Writes bool `json:"writes"`
}

// Tools is the catalogue, in the order shown on the MCP settings screen.
var Tools = []MCPToolInfo{
	{"list_campaigns", "List campaigns with status, audience and headline results.", string(authz.CampaignsView), false},
	{"get_campaign_report", "Results of a campaign by sending provider and mailbox provider.", string(authz.ReportsView), false},
	{"get_send_readiness", "The latest Send Readiness verdict and checks of a campaign.", string(authz.CampaignsView), false},
	{"create_campaign_draft", "Create a draft campaign from MJML.", string(authz.CampaignsEdit), true},
	{"update_campaign_content", "Change the subject, preheader or MJML of a draft, or of one of its language versions.", string(authz.CampaignsEdit), true},
	{"schedule_campaign", "Propose a send time. A team member confirms it in ReputeCast.", string(authz.CampaignsSend), true},
	{"pause_campaign", "Pause a running send.", string(authz.CampaignsPause), true},
	{"resume_campaign", "Resume a paused send.", string(authz.CampaignsSend), true},
	{"get_campaign_live_stats", "Live counters and routes of a running send.", string(authz.CampaignsView), false},
	{"search_contacts", "Find contacts by email or name.", string(authz.SubscribersView), false},
	{"create_segment", "Create a segment from condition-builder JSON.", string(authz.SegmentsEdit), true},
	{"add_to_suppression", "Suppress an address so nothing is sent to it again.", string(authz.SubscribersEdit), true},
	{"get_provider_health", "Status, bounce and complaint rates and quota of each sending provider.", string(authz.ProvidersView), false},
	{"get_routing", "Routing rules per stream with weights and fallbacks.", string(authz.ProvidersView), false},
	{"update_routing", "Change the weights of a stream's default route.", string(authz.ProvidersManage), true},
	{"list_automations", "Automations with their status.", string(authz.AutomationsView), false},
	{"get_deliverability", "Domain authentication, blocklists and Postmaster reputation.", string(authz.DeliverabilityView), false},
	{"send_test_email", "Send a test of a campaign to a member of the workspace.", string(authz.CampaignsEdit), true},
	{"get_campaign_languages", "A campaign's language versions: each one's status (up to date, outdated, draft, missing) and who gets it.", string(authz.CampaignsView), false},
	{"get_accessibility_issues", "Accessibility issues of a campaign in every language: what is serious, where it is and how to fix it.", string(authz.CampaignsView), false},
	{"list_transactional_templates", "Transactional templates with their category, live version and numbers.", string(authz.TransactionalView), false},
	{"render_transactional_template", "Render a transactional template with test data or given data. Nothing is sent or stored.", string(authz.TransactionalView), false},
	{"search_transactional_messages", "Search the transactional message log by status, template or recipient, without message content.", string(authz.TransactionalView), false},
	{"get_growth_sources", "Joins, departures, quality and cohort retention per growth source over a period (totals only, no addresses).", string(authz.SubscribersView), false},
	{"list_digests", "Recurring digests with their status, schedule, next issue and owner. Agents can't turn a digest on.", string(authz.CampaignsView), false},
	{"get_digest_issues", "A digest's issues: sent, skipped, held and waiting, each with its posts and results. Agents can't release or skip an issue.", string(authz.CampaignsView), false},
	{"get_poll_results", "Results of a campaign's polls or of a survey: answers by answer, NPS, by tier and language. Aggregate only.", string(authz.ReportsView), false},
	{"list_placement_tests", "Placement tests people ran: the email, the providers and where the copies landed.", string(authz.DeliverabilityView), false},
	{"get_placement_test", "One placement test: each copy's folder, time to arrive and SPF, DKIM and DMARC, with the guidance.", string(authz.DeliverabilityView), false},
}

// Settings are the per-workspace MCP options.
type Settings struct {
	Enabled             bool `json:"enabled"`
	RateLimitPerMinute  int  `json:"rate_limit_per_minute"`
	ApprovalExpiryHours int  `json:"approval_expiry_hours"`
}

// ParseSettings reads workspaces.mcp_settings with defaults.
func ParseSettings(raw json.RawMessage) Settings {
	st := Settings{Enabled: true, RateLimitPerMinute: 120, ApprovalExpiryHours: 2}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &st)
	}
	if st.RateLimitPerMinute <= 0 {
		st.RateLimitPerMinute = 120
	}
	if st.ApprovalExpiryHours <= 0 {
		st.ApprovalExpiryHours = 2
	}
	return st
}

// Capabilities are the columns of the agents matrix.
var Capabilities = []string{"Read reports and contacts", "Draft campaigns", "Edit audience", "Schedule or send", "Change routing"}

// MCPCapabilityCell is one cell of the matrix.
type MCPCapabilityCell struct {
	Capability string `json:"capability"`
	Level      string `json:"level" enum:"allow,approve,partial,deny"`
	Label      string `json:"label" example:"Pause only · never start"`
}

// Cells derives what an agent can do from its permission keys.
func Cells(perms authz.Set) []MCPCapabilityCell {
	cell := func(i int, level, label string) MCPCapabilityCell {
		return MCPCapabilityCell{Capability: Capabilities[i], Level: level, Label: label}
	}
	out := make([]MCPCapabilityCell, 0, len(Capabilities))
	switch {
	case perms.Has(authz.ReportsView) || perms.Has(authz.SubscribersView):
		out = append(out, cell(0, "allow", "Allow"))
	case perms.Has(authz.ProvidersView):
		out = append(out, cell(0, "allow", "Allow · provider health"))
	case perms.Has(authz.DeliverabilityView):
		out = append(out, cell(0, "allow", "Allow · deliverability"))
	default:
		out = append(out, cell(0, "deny", "Deny"))
	}
	if perms.Has(authz.CampaignsEdit) {
		out = append(out, cell(1, "allow", "Allow"))
	} else {
		out = append(out, cell(1, "deny", "Deny"))
	}
	if perms.Has(authz.SubscribersEdit) || perms.Has(authz.SegmentsEdit) || perms.Has(authz.ListsManage) {
		out = append(out, cell(2, "allow", "Allow"))
	} else {
		out = append(out, cell(2, "deny", "Deny"))
	}
	switch {
	case perms.Has(authz.CampaignsSend):
		out = append(out, cell(3, "approve", "Approve"))
	case perms.Has(authz.CampaignsPause):
		out = append(out, cell(3, "partial", "Pause only · never start"))
	default:
		out = append(out, cell(3, "deny", "Deny"))
	}
	if perms.Has(authz.ProvidersManage) {
		out = append(out, cell(4, "allow", "Allow"))
	} else {
		out = append(out, cell(4, "deny", "Deny"))
	}
	return out
}

// MCPAgentRow is one agent in the matrix.
type MCPAgentRow struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Masked      string              `json:"masked"`
	CreatedBy   string              `json:"createdBy"`
	Permissions []string            `json:"permissions"`
	Cells       []MCPCapabilityCell `json:"cells"`
	Status      string              `json:"status" enum:"active,expired"`
}

// MCPOverview is the MCP server settings screen.
type MCPOverview struct {
	Enabled             bool          `json:"enabled"`
	Endpoint            string        `json:"endpoint"`
	Transport           string        `json:"transport" enum:"streamable_http"`
	RateLimitPerMinute  int           `json:"rateLimitPerMinute"`
	ApprovalExpiryHours int           `json:"approvalExpiryHours"`
	Capabilities        []string      `json:"capabilities"`
	Agents              []MCPAgentRow `json:"agents"`
	Tools               []MCPToolInfo `json:"tools"`
	Resources           []string      `json:"resources"`
	Prompts             []string      `json:"prompts"`
	CanManage           bool          `json:"canManage"`
}

func (s *Service) base(ctx context.Context) string {
	if s.PublicURL != nil {
		return strings.TrimRight(s.PublicURL(ctx), "/")
	}
	return ""
}

// Overview returns the MCP settings of the workspace.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal) (MCPOverview, error) {
	st := ParseSettings(ws.McpSettings)
	out := MCPOverview{
		Enabled: st.Enabled, Endpoint: s.base(ctx) + "/mcp", Transport: "streamable_http",
		RateLimitPerMinute: st.RateLimitPerMinute, ApprovalExpiryHours: st.ApprovalExpiryHours,
		Capabilities: Capabilities, Agents: []MCPAgentRow{}, Tools: Tools,
		Resources: []string{"campaign MJML", "reports as CSV", "the audit log"},
		Prompts:   []string{"draft the weekly digest", "explain this auto-pause"},
		CanManage: p.Can(authz.MCPManage),
	}
	rows, err := s.Q.KeysList(ctx, store.KeysListParams{WorkspaceID: ws.ID, Kind: "mcp_agent"})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		perms := authz.ParseSet(r.Permissions)
		row := MCPAgentRow{ID: r.ID.String(), Name: r.Name, Masked: r.Prefix, CreatedBy: r.CreatedByName, Permissions: apikeys.StoredOrder(r.Permissions), Cells: Cells(perms), Status: "active"}
		if r.ExpiresAt != nil && !r.ExpiresAt.After(s.now()) {
			row.Status = "expired"
		}
		out.Agents = append(out.Agents, row)
	}
	return out, nil
}

// MCPSettingsInput changes the MCP options.
type MCPSettingsInput struct {
	Enabled             bool `json:"enabled"`
	RateLimitPerMinute  int  `json:"rateLimitPerMinute" minimum:"1" maximum:"6000"`
	ApprovalExpiryHours int  `json:"approvalExpiryHours" minimum:"1" maximum:"168"`
}

// UpdateSettings saves the MCP options.
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, in MCPSettingsInput) (MCPOverview, error) {
	before := ParseSettings(ws.McpSettings)
	after := Settings(in)
	raw, _ := json.Marshal(after)
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return MCPOverview{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	q := store.New(tx)
	if err := q.WsSetMCPSettings(ctx, store.WsSetMCPSettingsParams{ID: ws.ID, McpSettings: raw}); err != nil {
		return MCPOverview{}, err
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp.settings_updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
		Before: before, After: after, Metadata: map[string]any{"permission": string(authz.MCPManage)}}); err != nil {
		return MCPOverview{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MCPOverview{}, err
	}
	s.limiters.Range(func(k, _ any) bool { s.limiters.Delete(k); return true })
	ws.McpSettings = raw
	return s.Overview(ctx, ws, api.PrincipalFrom(ctx))
}
