package mcpserver

import (
	"context"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// SmsAgent is what agents can do with texts (X21, internal/texting): read
// the SMS providers' overview and a text campaign's report, and draft a
// text campaign. Sending stays with people: it needs sms:send in Review &
// send.
type SmsAgent interface {
	AgentOverview(ctx context.Context, ws *store.Workspace, p *authz.Principal) (map[string]any, error)
	AgentReport(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (map[string]any, string, string, error)
	AgentDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, name, body string, lists []string) (map[string]any, error)
}

// SmsCampaignRef names a text campaign by id or name.
type SmsCampaignRef struct {
	Campaign string `json:"campaign" jsonschema:"The text campaign's id, or its name (Summit: 8 days to go)"`
}

type smsDraftIn struct {
	Name  string   `json:"name" jsonschema:"The campaign's name"`
	Body  string   `json:"body" jsonschema:"The text, without the brand prefix and opt-out text the rules add. Personalization tokens like {{ first_name | default: \"there\" }} work."`
	Lists []string `json:"lists,omitempty" jsonschema:"Lists by name or id; texts go to their members with SMS consent"`
}

// Design expansion (append-only): the SMS tools.
func init() {
	Tools = append(Tools,
		MCPToolInfo{"get_sms_overview", "SMS providers, senders and their registrations, routes by country and the month's spend against the budget.", string(authz.ProvidersView), false},
		MCPToolInfo{"get_sms_campaign_report", "A text campaign's results by country: delivery, failures with the providers' codes, clicks, opt-outs, replies and cost. Counts only.", string(authz.ReportsView), false},
		MCPToolInfo{"create_sms_draft", "Draft a text campaign with its lists. A person with sms:send reviews and sends it.", string(authz.CampaignsEdit), true},
	)
}

// addSmsTools registers the SMS tools.
func (s *Service) addSmsTools(srv *mcp.Server) {
	unavailable := errors.New("texting isn't available on this server")
	tool(s, srv, "get_sms_overview", authz.ProvidersView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "sms.overview_read", targetType: "workspace", targetID: c.Workspace.ID.String(), targetName: c.Workspace.Name}
		if s.Sms == nil {
			return nil, e, unavailable
		}
		ws := c.Workspace
		out, err := s.Sms.AgentOverview(ctx, &ws, c.Principal)
		return out, e, err
	}, nil)
	tool(s, srv, "get_sms_campaign_report", authz.ReportsView, func(ctx context.Context, c *agentCall, in SmsCampaignRef) (any, entry, error) {
		e := entry{action: "campaign.report_read", targetType: "campaign", targetName: in.Campaign}
		if s.Sms == nil {
			return nil, e, unavailable
		}
		ws := c.Workspace
		out, id, name, err := s.Sms.AgentReport(ctx, &ws, c.Principal, in.Campaign)
		if err == nil {
			e.targetID, e.targetName = id, name
		}
		return out, e, err
	}, nil)
	tool(s, srv, "create_sms_draft", authz.CampaignsEdit, func(ctx context.Context, c *agentCall, in smsDraftIn) (any, entry, error) {
		name := strings.TrimSpace(in.Name)
		e := entry{action: "campaign.created", targetType: "campaign", targetName: name, detail: "text draft"}
		if s.Sms == nil {
			return nil, e, unavailable
		}
		ws := c.Workspace
		out, err := s.Sms.AgentDraft(ctx, &ws, c.Principal, name, in.Body, in.Lists)
		if err != nil {
			return nil, e, err
		}
		if id, ok := out["id"].(string); ok {
			e.targetID = id
		}
		e.after = map[string]any{"name": name, "channel": "sms", "lists": in.Lists}
		return out, e, nil
	}, nil)
}
