package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/push"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/whatsapp"
)

// Web push and WhatsApp (design expansion phase 8, X22): agents draft push
// and WhatsApp campaigns and read their reports and costs. They never send:
// Review & send is a person's (agents' sending stays a proposal), and they
// never touch the keys, the sites, the account or the rate card.

// PushDrafts drafts push campaigns and reads their reports (internal/push).
type PushDrafts interface {
	CreateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, in push.PushCampaignCreate) (push.PushCampaign, error)
	CampaignReport(ctx context.Context, ws *store.Workspace, id uuid.UUID, canSend bool) (push.PushReport, error)
}

// WhatsAppDrafts drafts WhatsApp campaigns and reads their reports and
// costs (internal/whatsapp).
type WhatsAppDrafts interface {
	CreateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, in whatsapp.WhatsAppCampaignCreate) (whatsapp.WhatsAppCampaign, error)
	CampaignReport(ctx context.Context, ws *store.Workspace, id uuid.UUID) (whatsapp.WhatsAppReport, error)
	Templates(ctx context.Context, ws *store.Workspace) ([]whatsapp.WhatsAppTemplate, error)
}

// Design expansion (append-only): push and WhatsApp tools.
func init() {
	Tools = append(Tools,
		MCPToolInfo{"create_push_draft", "Draft a push campaign: a title, a message and the page it opens. It stays a draft: a person reviews and sends it.", string(authz.CampaignsEdit), true},
		MCPToolInfo{"get_push_report", "A push campaign's results: sent, shown, clicks, unsubscribes and expired subscriptions, by browser and site.", string(authz.ReportsView), false},
		MCPToolInfo{"create_whatsapp_draft", "Draft a WhatsApp campaign with an approved template and the lists it goes to. It stays a draft: a person reviews and sends it.", string(authz.CampaignsEdit), true},
		MCPToolInfo{"get_whatsapp_report", "A WhatsApp campaign's results and cost: sent, delivered, read, failed with Meta's codes, replies and opt-outs.", string(authz.ReportsView), false},
	)
}

type pushDraftIn struct {
	Name    string `json:"name" jsonschema:"the campaign's name"`
	Title   string `json:"title" jsonschema:"the notification's title (about 40 characters show everywhere)"`
	Message string `json:"message,omitempty" jsonschema:"the notification's text (up to 120 characters)"`
	Link    string `json:"link" jsonschema:"the page the notification opens"`
}

type whatsAppDraftIn struct {
	Name     string   `json:"name" jsonschema:"the campaign's name"`
	Template string   `json:"template" jsonschema:"the name of an approved template"`
	Lists    []string `json:"lists,omitempty" jsonschema:"the names or ids of the lists it goes to (people with WhatsApp consent)"`
}

// channelCampaign finds a campaign of a channel by name or id.
func (s *Service) channelCampaign(ctx context.Context, ws uuid.UUID, ref, channel string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var name string
	ref = strings.TrimSpace(ref)
	err := s.Pool.QueryRow(ctx, `SELECT id, name FROM campaigns WHERE workspace_id = $1 AND channel = $2 AND deleted_at IS NULL
		AND (id::text = $3 OR lower(name) = lower($3)) ORDER BY created_at DESC LIMIT 1`, ws, channel, ref).Scan(&id, &name)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("no %s campaign named %q", channel, ref)
	}
	return id, name, nil
}

// addChannelTools registers the push and WhatsApp tools.
func (s *Service) addChannelTools(srv *mcp.Server) {
	tool(s, srv, "create_push_draft", authz.CampaignsEdit, func(ctx context.Context, c *agentCall, in pushDraftIn) (any, entry, error) {
		e := entry{action: "campaign.draft_created", targetType: "campaign", targetName: in.Name}
		if s.Push == nil {
			return nil, e, errors.New("push isn't available on this server")
		}
		ws := c.Workspace
		cp, err := s.Push.CreateCampaign(ctx, &ws, c.Principal, push.PushCampaignCreate{Name: in.Name, Title: in.Title, Message: in.Message, Link: in.Link})
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName, e.detailText = cp.ID, cp.Name, keep("detail.pushDraft", nil)
		return map[string]any{"id": cp.ID, "name": cp.Name, "status": cp.Status, "audience": cp.Audience.Final, "sites": len(cp.Sites),
			"note": "A draft: a person with campaigns:send and push:send sends it from Review & send."}, e, nil
	}, nil)

	tool(s, srv, "get_push_report", authz.ReportsView, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "report.read", targetType: "campaign", targetName: in.Campaign}
		if s.Push == nil {
			return nil, e, errors.New("push isn't available on this server")
		}
		id, name, err := s.channelCampaign(ctx, c.Workspace.ID, in.Campaign, "push")
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = id.String(), name
		ws := c.Workspace
		r, err := s.Push.CampaignReport(ctx, &ws, id, false)
		if err != nil {
			return nil, e, err
		}
		return map[string]any{"name": r.Name, "status": r.Status, "sent": r.Sent, "shown": r.Shown, "shown_pct": r.ShownPct, "clicks": r.Clicks,
			"clicks_pct": r.ClicksPct, "unsubscribed_24h": r.Unsubscribed, "expired_removed": r.Expired, "by_browser": r.ByBrowser, "by_site": r.BySite,
			"note": "Shown is reported by the service worker; some browsers report it late or not at all."}, e, nil
	}, s.campaignTarget("report.read"))

	tool(s, srv, "create_whatsapp_draft", authz.CampaignsEdit, func(ctx context.Context, c *agentCall, in whatsAppDraftIn) (any, entry, error) {
		e := entry{action: "campaign.draft_created", targetType: "campaign", targetName: in.Name}
		if s.WhatsApp == nil {
			return nil, e, errors.New("WhatsApp isn't available on this server")
		}
		ws := c.Workspace
		ts, err := s.WhatsApp.Templates(ctx, &ws)
		if err != nil {
			return nil, e, err
		}
		tid := ""
		approved := []string{}
		for _, t := range ts {
			if t.Status == "approved" {
				approved = append(approved, t.Name)
				if strings.EqualFold(t.Name, strings.TrimSpace(in.Template)) {
					tid = t.ID
				}
			}
		}
		if tid == "" {
			return nil, e, fmt.Errorf("pick one of the approved templates: %s", strings.Join(approved, ", "))
		}
		lists := []string{}
		for _, l := range in.Lists {
			var id uuid.UUID
			if err := s.Pool.QueryRow(ctx, `SELECT id FROM lists WHERE workspace_id = $1 AND deleted_at IS NULL AND (id::text = $2 OR lower(name) = lower($2))
				LIMIT 1`, ws.ID, strings.TrimSpace(l)).Scan(&id); err != nil {
				return nil, e, fmt.Errorf("no list named %q", l)
			}
			lists = append(lists, id.String())
		}
		cp, err := s.WhatsApp.CreateCampaign(ctx, &ws, c.Principal, whatsapp.WhatsAppCampaignCreate{Name: in.Name, TemplateID: tid, Lists: lists})
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName, e.detailText = cp.ID, cp.Name, keep("detail.whatsappDraft", nil)
		return map[string]any{"id": cp.ID, "name": cp.Name, "status": cp.Status, "template": in.Template, "recipients": cp.Audience.Final,
			"estimated_cost": cp.Cost.Amount, "rates_as_of": cp.Cost.RatesAsOf.Format("2006-01-02"),
			"note": "A draft: map the template's variables in the composer; a person with campaigns:send and whatsapp:send sends it."}, e, nil
	}, nil)

	tool(s, srv, "get_whatsapp_report", authz.ReportsView, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "report.read", targetType: "campaign", targetName: in.Campaign}
		if s.WhatsApp == nil {
			return nil, e, errors.New("WhatsApp isn't available on this server")
		}
		id, name, err := s.channelCampaign(ctx, c.Workspace.ID, in.Campaign, "whatsapp")
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = id.String(), name
		ws := c.Workspace
		r, err := s.WhatsApp.CampaignReport(ctx, &ws, id)
		if err != nil {
			return nil, e, err
		}
		return map[string]any{"name": r.Name, "status": r.Status, "template": r.Template, "sent": r.Sent, "delivered": r.Delivered, "read": r.Read,
			"failed": r.Failed, "failures": r.Failures, "button_clicks": r.Buttons, "replies": r.Replies, "opt_outs": r.OptOuts, "cost": r.Cost,
			"note": "Read counts only people who share read receipts. Cost is what Meta charges for delivered template messages."}, e, nil
	}, s.campaignTarget("report.read"))
}
