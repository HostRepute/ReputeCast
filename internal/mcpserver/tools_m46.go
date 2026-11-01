package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Read tools for what agents asked about most, and send proposals for texts,
// push and WhatsApp like email's schedule_campaign (review 2026-11 IN-3).
// Each checks the permission its REST operation needs.

// AutomationStats reads a flow's numbers as the editor's Stats panel shows
// them (internal/automations).
type AutomationStats interface {
	Stats(ctx context.Context, ws *store.Workspace, id uuid.UUID, days int) (automations.AutomationStats, error)
}

func init() {
	Tools = append(Tools,
		MCPToolInfo{"get_contact", "One contact by id or address: status, consent, tier, language, time zone, tags, lists and when they last opened.", string(authz.SubscribersView), false},
		MCPToolInfo{"list_lists", "The workspace's lists with their subscribed counts and opt-in mode.", string(authz.SubscribersView), false},
		MCPToolInfo{"list_forms", "Signup forms with their kind, status and submissions in the last 30 days.", string(authz.SubscribersView), false},
		MCPToolInfo{"get_automation", "One flow with its triggers and the Stats panel's numbers: entered, in the flow, each step and the email sequence.", string(authz.AutomationsView), false},
		MCPToolInfo{"list_approvals", "The approvals queue: open requests first, then the last 30 days' decisions.", string(authz.CampaignsView), false},
		MCPToolInfo{"list_offers", "Earn's offers: discounts, codes, limits and how often they were used.", string(authz.EarnView), false},
		MCPToolInfo{"propose_send", "Propose sending a text, push or WhatsApp campaign, now or at a time. A team member confirms it in ReputeCast.", string(authz.CampaignsSend), true},
	)
	proposeVerbs["propose_send"] = "send"
}

type contactIn struct {
	Contact string `json:"contact" jsonschema:"the contact's id or email address"`
}

type automationIn struct {
	Automation string `json:"automation" jsonschema:"the flow's id or exact name"`
	Days       int    `json:"days,omitempty" jsonschema:"the stats window in days: 7, 30 (default) or 90"`
}

type proposeSendIn struct {
	Campaign string `json:"campaign" jsonschema:"the text, push or WhatsApp campaign's id or exact name"`
	At       string `json:"at,omitempty" jsonschema:"when, as an RFC 3339 time; empty proposes sending now"`
}

func (s *Service) addM46Tools(srv *mcp.Server) {
	tool(s, srv, "get_contact", authz.SubscribersView, func(ctx context.Context, c *agentCall, in contactIn) (any, entry, error) {
		e := entry{action: "contact.read", targetType: "contact"}
		ref := strings.TrimSpace(in.Contact)
		p := store.McpContactParams{WorkspaceID: c.Workspace.ID, Email: ref}
		if id, err := uuid.Parse(ref); err == nil {
			p.ID, p.Email = &id, ""
		}
		r, err := s.Q.McpContact(ctx, p)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, e, errors.New("no contact with that id or address")
		}
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = r.ID.String(), r.Email
		return map[string]any{"id": r.ID, "email": r.Email, "name": strings.TrimSpace(r.FirstName + " " + r.LastName), "status": r.Status, "consent": r.Consent,
			"tier": r.Tier, "language": r.Language, "timezone": r.Timezone, "source": r.Source, "tags": nonNil(r.Tags), "lists": nonNil(r.Lists),
			"since": r.CreatedAt, "last_open_at": r.LastOpenAt, "last_human_open_at": r.LastHumanOpenAt}, e, nil
	}, nil)

	tool(s, srv, "list_lists", authz.SubscribersView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "list.listed", targetType: "list"}
		rows, err := s.Q.McpLists(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		for _, r := range rows {
			out = append(out, map[string]any{"id": r.ID, "name": r.Name, "description": r.Description, "subscribed": r.Subscribed, "double_opt_in": r.DoubleOptIn, "created_at": r.CreatedAt})
		}
		return map[string]any{"lists": out}, e, nil
	}, nil)

	tool(s, srv, "list_forms", authz.SubscribersView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "form.listed", targetType: "form"}
		rows, err := s.Q.McpForms(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		for _, r := range rows {
			out = append(out, map[string]any{"id": r.ID, "name": r.Name, "kind": r.Kind, "status": r.Status, "double_opt_in": r.DoubleOptIn,
				"submissions_30d": r.Submissions30d, "updated_at": r.UpdatedAt})
		}
		return map[string]any{"forms": out}, e, nil
	}, nil)

	tool(s, srv, "get_automation", authz.AutomationsView, func(ctx context.Context, c *agentCall, in automationIn) (any, entry, error) {
		e := entry{action: "automation.read", targetType: "automation", targetName: in.Automation}
		rows, err := s.Q.McpAutomations(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		ref := strings.TrimSpace(in.Automation)
		for _, r := range rows {
			if r.ID.String() != ref && !strings.EqualFold(r.Name, ref) {
				continue
			}
			e.targetID, e.targetName = r.ID.String(), r.Name
			out := map[string]any{"id": r.ID, "name": r.Name, "status": r.Status, "trigger": r.Trigger, "updated_at": r.UpdatedAt}
			if s.Automations != nil {
				days := in.Days
				if days != 7 && days != 90 {
					days = 30
				}
				st, err := s.Automations.Stats(ctx, &c.Workspace, r.ID, days)
				if err != nil {
					return nil, e, err
				}
				out["stats"] = st
			}
			return out, e, nil
		}
		return nil, e, errors.New("no automation with that id or name; list_automations names them")
	}, nil)

	tool(s, srv, "list_approvals", authz.CampaignsView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "approval.listed", targetType: "approval"}
		rows, err := s.Q.McpApprovals(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		for _, r := range rows {
			out = append(out, map[string]any{"id": r.ID, "about": r.SubjectType, "name": r.SubjectName, "status": r.Status, "requested_by": r.RequestedBy,
				"requested_at": r.RequestedAt, "decided_at": r.DecidedAt, "recipients": r.AudienceCount})
		}
		return map[string]any{"approvals": out, "note": "Agents can't decide approvals: a person with campaigns:approve does, in ReputeCast."}, e, nil
	}, nil)

	tool(s, srv, "list_offers", authz.EarnView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "offer.listed", targetType: "offer"}
		rows, err := s.Q.McpOffers(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		for _, r := range rows {
			o := map[string]any{"id": r.ID, "name": r.Name, "kind": r.Kind, "status": r.Status, "discount": r.DiscountType, "duration": r.Duration,
				"by_link": r.ByLink, "uses": r.Uses, "redeem_limit": r.RedeemLimit, "ends_at": r.EndsAt}
			switch r.DiscountType {
			case "percent":
				o["percent_off"] = r.PercentOff
			default:
				o["amount_off"], o["currency"] = r.AmountOff, r.Currency
			}
			if r.Duration == "months" {
				o["duration_months"] = r.DurationMonths
			}
			if r.ByCode {
				o["code"] = r.Code
			}
			out = append(out, o)
		}
		return map[string]any{"offers": out}, e, nil
	}, nil)

	tool(s, srv, "propose_send", authz.CampaignsSend, func(ctx context.Context, c *agentCall, in proposeSendIn) (any, entry, error) {
		e := entry{action: "campaign.send_proposed", targetType: "campaign", targetName: in.Campaign}
		ref := strings.TrimSpace(in.Campaign)
		p := store.McpChannelCampaignParams{WorkspaceID: c.Workspace.ID, Name: ref}
		if id, err := uuid.Parse(ref); err == nil {
			p.ID, p.Name = &id, ""
		}
		cp, err := s.Q.McpChannelCampaign(ctx, p)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, e, errors.New("no campaign with that id or name")
		}
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		switch {
		case cp.Channel == "email":
			return nil, e, errors.New("this is an email campaign: use schedule_campaign")
		case cp.Status != "draft" && cp.Status != "approved":
			return nil, e, fmt.Errorf("the campaign is %s: only a draft can be proposed for sending", cp.Status)
		}
		when := "now"
		if strings.TrimSpace(in.At) != "" {
			at, err := time.Parse(time.RFC3339, in.At)
			if err != nil {
				return nil, e, errors.New("at must be an RFC 3339 time, like 2026-11-04T09:00:00-05:00")
			}
			if !at.After(s.now()) {
				return nil, e, errors.New("pick a time in the future, or leave at empty to propose sending now")
			}
			when = at.Format(time.RFC3339)
			e.detailText = keep("detail.proposedFor", i18n.Args{"at": at})
		}
		s.propose(ctx, c, "propose_send", authz.CampaignsSend, entry{targetType: "campaign", targetID: cp.ID.String(), targetName: cp.Name})
		who := "a member with campaigns:send"
		if helpers := access.Helpers(ctx, s.Q, c.Workspace.ID, authz.CampaignsSend); len(helpers) > 0 {
			who = helpers[0]
		}
		return map[string]any{"proposed": true, "campaign": cp.Name, "channel": cp.Channel, "when": when,
			"message": fmt.Sprintf("Sending needs a person: this was proposed to %s, who reviews and sends it in ReputeCast. The proposal expires in %d hours.", who, c.Settings.ApprovalExpiryHours)}, e, nil
	}, nil)
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
