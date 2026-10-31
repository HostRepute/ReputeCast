package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// CampaignRef names a campaign by name or id.
type CampaignRef struct {
	Campaign string `json:"campaign" jsonschema:"the campaign's name (exact) or id"`
}

type listCampaignsIn struct {
	Status string `json:"status,omitempty" jsonschema:"all, drafts, scheduled, sending or sent (default all)"`
	Query  string `json:"query,omitempty" jsonschema:"search by name"`
	Limit  int    `json:"limit,omitempty" jsonschema:"at most this many campaigns (default 25, max 100)"`
}

type draftIn struct {
	Name      string `json:"name" jsonschema:"campaign name"`
	Subject   string `json:"subject,omitempty"`
	Preheader string `json:"preheader,omitempty"`
	MJML      string `json:"mjml,omitempty" jsonschema:"MJML 4 source"`
}

type updateContentIn struct {
	Campaign  string  `json:"campaign" jsonschema:"the draft's name or id"`
	Subject   *string `json:"subject,omitempty"`
	Preheader *string `json:"preheader,omitempty"`
	MJML      *string `json:"mjml,omitempty" jsonschema:"MJML 4 source"`
	Language  string  `json:"language,omitempty" jsonschema:"change this language version (its code, such as de) instead of the source; get_campaign_languages lists them"`
}

type scheduleIn struct {
	Campaign string `json:"campaign" jsonschema:"the campaign's name or id"`
	At       string `json:"at" jsonschema:"send time, RFC 3339 (2026-11-04T09:00:00-05:00)"`
}

type pauseIn struct {
	Campaign string `json:"campaign" jsonschema:"the campaign's name or id"`
	Reason   string `json:"reason,omitempty" jsonschema:"why, recorded in the audit log"`
}

type searchIn struct {
	Query string `json:"query" jsonschema:"part of an email address or name"`
	Limit int    `json:"limit,omitempty" jsonschema:"at most this many contacts (default 20, max 100)"`
}

type segmentIn struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Definition  map[string]any `json:"definition" jsonschema:"condition-builder JSON: {\"groups\":[{\"match\":\"all\",\"conditions\":[{\"field\":…,\"op\":…,\"value\":…}]}]}"`
}

type suppressIn struct {
	Email  string `json:"email"`
	Reason string `json:"reason,omitempty" jsonschema:"why, kept with the suppression"`
}

type routingIn struct {
	Stream  string         `json:"stream" jsonschema:"broadcast, automation or system"`
	Weights map[string]int `json:"weights" jsonschema:"sending provider name → percent; the weights add up to 100"`
}

type testIn struct {
	Campaign string `json:"campaign" jsonschema:"the campaign's name or id"`
	To       string `json:"to" jsonschema:"email of a member of this workspace"`
}

type noIn struct{}

func (s *Service) findCampaign(ctx context.Context, ws uuid.UUID, r string) (store.McpFindCampaignRow, error) {
	c, err := s.Q.McpFindCampaign(ctx, store.McpFindCampaignParams{WorkspaceID: ws, Ref: ref(r)})
	if errors.Is(err, pgx.ErrNoRows) {
		return c, fmt.Errorf("no campaign named or with id %q in this workspace", r)
	}
	return c, err
}

// tool wraps a handler with authentication, the permission check and the audit entry.
func tool[In any](s *Service, srv *mcp.Server, name string, perm authz.Permission, run func(ctx context.Context, c *agentCall, in In) (any, entry, error), target func(ctx context.Context, c *agentCall, in In) entry) {
	var desc string
	for _, t := range Tools {
		if t.Name == name {
			desc = t.Description + " Needs " + t.Permission + "."
		}
	}
	mcp.AddTool(srv, &mcp.Tool{Name: name, Description: desc}, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		ctx, c, err := call(ctx, req.Extra)
		if err != nil {
			return nil, nil, err
		}
		if !c.Principal.Can(perm) {
			var e entry
			if target != nil {
				e = target(ctx, c, in)
			}
			if e.action == "" {
				e.action = "mcp." + name
			}
			return nil, nil, s.deny(ctx, c, name, perm, e)
		}
		out, e, err := run(ctx, c, in)
		if e.action == "" {
			e.action = "mcp." + name
		}
		if err != nil {
			s.record(ctx, c, name, perm, e, audit.Failed, err.Error(), i18n.Text{})
			return nil, nil, err
		}
		s.record(ctx, c, name, perm, e, audit.Allowed, "", i18n.Text{})
		return nil, out, nil
	})
}

// campaignTarget names the campaign in denied entries when the agent may see it.
func (s *Service) campaignTarget(action string) func(ctx context.Context, c *agentCall, in CampaignRef) entry {
	return func(ctx context.Context, c *agentCall, in CampaignRef) entry {
		e := entry{action: action, targetType: "campaign", targetName: in.Campaign}
		if c.Principal.Can(authz.CampaignsView) {
			if cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign); err == nil {
				e.targetID, e.targetName = cp.ID.String(), cp.Name
			}
		}
		return e
	}
}

// campaigns is the campaigns service the tools list, create and save
// through, as the composer does.
func (s *Service) campaigns() *campaigns.Service {
	if s.Campaigns != nil {
		return s.Campaigns
	}
	return &campaigns.Service{Pool: s.Pool, Q: s.Q, Audit: s.Audit}
}

func (s *Service) addTools(srv *mcp.Server) {
	tool(s, srv, "list_campaigns", authz.CampaignsView, func(ctx context.Context, c *agentCall, in listCampaignsIn) (any, entry, error) {
		e := entry{action: "campaign.listed", targetType: "campaign", targetText: keep("target.campaigns", nil)}
		status := in.Status
		if status == "" {
			status = "all"
		}
		if !strings.Contains("all drafts scheduled sending sent", status) {
			return nil, e, fmt.Errorf("status must be all, drafts, scheduled, sending or sent")
		}
		limit := in.Limit
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		list, err := s.campaigns().List(ctx, &c.Workspace, campaigns.ListParams{Status: status, Query: in.Query, PageSize: limit, Page: 1})
		if err != nil {
			return nil, e, err
		}
		type row struct {
			ID          string     `json:"id"`
			Name        string     `json:"name"`
			Status      string     `json:"status"`
			Audience    string     `json:"audience,omitempty"`
			Recipients  *int64     `json:"recipients,omitempty"`
			ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
			StartedAt   *time.Time `json:"started_at,omitempty"`
			Delivered   *float64   `json:"delivered_pct,omitempty"`
			HumanOpens  *float64   `json:"human_opens_pct,omitempty"`
			Clicks      *float64   `json:"clicks_pct,omitempty"`
		}
		rows := []row{}
		for _, it := range list.Items {
			rows = append(rows, row{it.ID, it.Name, it.Status, it.Audience, it.Recipients, it.ScheduledAt, it.StartedAt, it.Delivered, it.HumanOpens, it.Clicks})
		}
		e.detailText = keep("detail.count", i18n.Args{"n": len(rows), "total": list.Total})
		return map[string]any{"campaigns": rows, "total": list.Total}, e, nil
	}, nil)

	tool(s, srv, "get_send_readiness", authz.CampaignsView, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "readiness.read", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		r, err := s.Q.McpLatestReadiness(ctx, cp.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]any{"campaign": cp.Name, "verdict": "not_checked", "message": "Send Readiness hasn't run for this campaign yet."}, e, nil
		} else if err != nil {
			return nil, e, err
		}
		e.detail = r.Verdict
		return map[string]any{"campaign": cp.Name, "verdict": r.Verdict, "checks": readiness.WithoutTexts(r.Checks), "estimates": readiness.WithoutTexts(r.Estimates),
			"ran_at": r.RanAt}, e, nil
	}, nil)

	tool(s, srv, "create_campaign_draft", authz.CampaignsEdit, func(ctx context.Context, c *agentCall, in draftIn) (any, entry, error) {
		name := strings.TrimSpace(in.Name)
		e := entry{action: "campaign.created", targetType: "campaign", targetName: name}
		if name == "" {
			return nil, e, errors.New("give the draft a name")
		}
		// Stored as the composer stores a draft: the server compiles and
		// checks the MJML, and the draft has its first version (CO-3).
		ws := c.Workspace
		row, err := s.campaigns().CreateForAgent(ctx, &ws, c.Principal, name, campaigns.AgentContent{Subject: &in.Subject, Preheader: &in.Preheader, MJML: &in.MJML})
		if err != nil {
			return nil, e, err
		}
		e.targetID = row.ID
		e.after = map[string]any{"name": name, "subject": in.Subject, "version": row.Version}
		e.detailText = keep("detail.draft", nil)
		return map[string]any{"id": row.ID, "name": row.Name, "status": row.Status, "version": row.Version,
			"message": "Draft created. ReputeCast compiled the MJML; a person reviews and sends it from ReputeCast."}, e, nil
	}, nil)

	tool(s, srv, "update_campaign_content", authz.CampaignsEdit, func(ctx context.Context, c *agentCall, in updateContentIn) (any, entry, error) {
		e := entry{action: "campaign.updated", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		if cp.Status != "draft" {
			return nil, e, fmt.Errorf("only drafts can be changed by an agent; %q is %s", cp.Name, cp.Status)
		}
		if in.Language != "" && s.Languages != nil {
			return s.updateLanguage(ctx, c, cp, in, e)
		}
		// Saved as the composer saves (CO-3): the server compiles and checks
		// the MJML (the HTML sent is that compile), the change is a new
		// version, and approvals and Send Readiness see it.
		ws := c.Workspace
		saved, err := s.campaigns().SaveForAgent(ctx, &ws, c.Principal, cp.ID, campaigns.AgentContent{Subject: in.Subject, Preheader: in.Preheader, MJML: in.MJML})
		if err != nil {
			return nil, e, err
		}
		e.detailText = keep("detail.changed", i18n.Args{"fields": changedFields(in)})
		e.before = map[string]any{"subject": saved.BeforeSubject, "preheader": saved.BeforePreheader, "version": saved.BeforeVersion}
		e.after = map[string]any{"subject": saved.Subject, "preheader": saved.Preheader, "version": saved.Version, "sizeBytes": saved.SizeBytes, "contentHash": saved.ContentHash}
		return map[string]any{"id": cp.ID, "name": cp.Name, "version": saved.Version, "subject": saved.Subject, "size_bytes": saved.SizeBytes, "unchanged": saved.Unchanged}, e, nil
	}, nil)

	tool(s, srv, "schedule_campaign", authz.CampaignsSend, func(ctx context.Context, c *agentCall, in scheduleIn) (any, entry, error) {
		e := entry{action: "campaign.schedule_proposed", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		at, err := time.Parse(time.RFC3339, in.At)
		if err != nil {
			return nil, e, errors.New("at must be an RFC 3339 time, like 2026-11-04T09:00:00-05:00")
		}
		if !at.After(s.now()) {
			return nil, e, errors.New("pick a time in the future")
		}
		e.detailText = keep("detail.proposedFor", i18n.Args{"at": at})
		helpers := access.Helpers(ctx, s.Q, c.Workspace.ID, authz.CampaignsSend)
		s.propose(ctx, c, "schedule_campaign", authz.CampaignsSend, entry{targetType: "campaign", targetID: cp.ID.String(), targetName: cp.Name})
		who := "a member with campaigns:send"
		if len(helpers) > 0 {
			who = helpers[0]
		}
		return map[string]any{"proposed": true, "campaign": cp.Name, "at": at,
			"message": fmt.Sprintf("Sending needs a person: the schedule was proposed to %s, who confirms it in ReputeCast (Send Readiness and approvals still apply). The proposal expires in %d hours.", who, c.Settings.ApprovalExpiryHours)}, e, nil
	}, func(ctx context.Context, c *agentCall, in scheduleIn) entry {
		return s.campaignTarget("campaign.schedule_proposed")(ctx, c, CampaignRef{Campaign: in.Campaign})
	})

	tool(s, srv, "pause_campaign", authz.CampaignsPause, func(ctx context.Context, c *agentCall, in pauseIn) (any, entry, error) {
		e := entry{action: "campaign.paused", targetType: "campaign", targetName: in.Campaign, detail: strings.TrimSpace(in.Reason)}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		if cp.Status != "sending" {
			return nil, e, fmt.Errorf("%q isn't sending (it is %s)", cp.Name, cp.Status)
		}
		if err := s.Sending.Pause(ctx, cp.ID, c.Principal.ID, "MCP agent "+c.Principal.Name+": "+in.Reason); err != nil {
			if errors.Is(err, sending.ErrUnavailable) {
				return nil, e, errors.New("the sending engine isn't running in this process, so the send can't be paused from here; pause it in ReputeCast")
			}
			return nil, e, err
		}
		e.before, e.after = map[string]any{"status": "sending"}, map[string]any{"status": "paused"}
		return map[string]any{"campaign": cp.Name, "status": "paused", "message": "Paused. Resuming needs campaigns:send."}, e, nil
	}, func(ctx context.Context, c *agentCall, in pauseIn) entry {
		return s.campaignTarget("campaign.paused")(ctx, c, CampaignRef{Campaign: in.Campaign})
	})

	tool(s, srv, "resume_campaign", authz.CampaignsSend, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "campaign.resumed", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		if cp.Status != "paused" {
			return nil, e, fmt.Errorf("%q isn't paused (it is %s)", cp.Name, cp.Status)
		}
		if err := s.Sending.Resume(ctx, cp.ID, c.Principal.ID); err != nil {
			if errors.Is(err, sending.ErrUnavailable) {
				return nil, e, errors.New("the sending engine isn't running in this process; resume the send in ReputeCast")
			}
			return nil, e, err
		}
		e.before, e.after = map[string]any{"status": "paused"}, map[string]any{"status": "sending"}
		return map[string]any{"campaign": cp.Name, "status": "sending"}, e, nil
	}, s.campaignTarget("campaign.resumed"))

	tool(s, srv, "get_campaign_live_stats", authz.CampaignsView, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "campaign.live_read", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		t, err := s.Q.McpCampaignTotals(ctx, cp.ID)
		if err != nil {
			return nil, e, err
		}
		routes, err := s.Q.McpCampaignRoutes(ctx, cp.ID)
		if err != nil {
			return nil, e, err
		}
		rs := []map[string]any{}
		for _, r := range routes {
			rs = append(rs, map[string]any{"sending_provider": r.Provider, "status": r.Status, "detail": r.StatusDetail, "planned": r.Planned})
		}
		return map[string]any{"campaign": cp.Name, "status": cp.Status, "started_at": cp.StartedAt, "audience": cp.AudienceCount, "sent": t.Sent, "delivered": t.Delivered,
			"deferred": t.Deferred, "bounced": t.Bounced, "complained": t.Complained, "routes": rs}, e, nil
	}, nil)

	tool(s, srv, "search_contacts", authz.SubscribersView, func(ctx context.Context, c *agentCall, in searchIn) (any, entry, error) {
		q := strings.TrimSpace(in.Query)
		e := entry{action: "contacts.searched", targetType: "contacts", targetText: keep("target.contacts", nil), detailText: keep("detail.query", i18n.Args{"query": q})}
		if len(q) < 2 {
			return nil, e, errors.New("search with at least 2 characters")
		}
		limit := in.Limit
		if limit <= 0 || limit > 100 {
			limit = 20
		}
		rows, err := s.Q.McpSearchContacts(ctx, store.McpSearchContactsParams{WorkspaceID: c.Workspace.ID, Q: q, Lists: c.Principal.Scope.Lists, Lim: int32(limit)})
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		ids := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			out = append(out, map[string]any{"id": r.ID, "email": r.Email, "name": strings.TrimSpace(r.FirstName + " " + r.LastName), "status": r.Status, "tier": r.Tier, "since": r.CreatedAt})
			ids = append(ids, r.ID)
		}
		s.scoresFor(ctx, c.Workspace.ID, out, ids)
		e.detail += fmt.Sprintf(" · %d found", len(out))
		return map[string]any{"contacts": out}, e, nil
	}, nil)

	tool(s, srv, "create_segment", authz.SegmentsEdit, func(ctx context.Context, c *agentCall, in segmentIn) (any, entry, error) {
		name := strings.TrimSpace(in.Name)
		e := entry{action: "segment.created", targetType: "segment", targetName: name}
		if name == "" {
			return nil, e, errors.New("give the segment a name")
		}
		// Validate as the builder does: the compiler inlines keys and units.
		raw, _ := json.Marshal(in.Definition)
		if len(in.Definition) == 0 {
			raw = []byte(`{}`)
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var d segments.SegmentDefinition
		if err := dec.Decode(&d); err != nil {
			return nil, e, fmt.Errorf("the definition isn't condition-builder JSON: %w", err)
		}
		if _, err := (&segments.Service{Pool: s.Pool, Q: s.Q}).CheckDefinition(ctx, c.Workspace.ID, &d); err != nil {
			var ve *segments.ValidationError
			if errors.As(err, &ve) {
				return nil, e, fmt.Errorf("%s: %s", ve.Location, ve.Message)
			}
			return nil, e, err
		}
		def, _ := json.Marshal(d)
		creator := c.Principal.UserID
		var by *uuid.UUID
		if creator != uuid.Nil {
			by = &creator
		}
		row, err := s.Q.McpCreateSegment(ctx, store.McpCreateSegmentParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: c.Workspace.ID, Name: name, Description: in.Description, Definition: def, CreatedBy: by})
		if err != nil {
			return nil, e, err
		}
		e.targetID = row.ID.String()
		e.after = map[string]any{"name": name, "definition": json.RawMessage(def)}
		return map[string]any{"id": row.ID, "name": row.Name}, e, nil
	}, nil)

	tool(s, srv, "add_to_suppression", authz.SubscribersEdit, func(ctx context.Context, c *agentCall, in suppressIn) (any, entry, error) {
		email := strings.ToLower(strings.TrimSpace(in.Email))
		e := entry{action: "suppression.added", targetType: "suppression", targetName: email, detail: strings.TrimSpace(in.Reason)}
		if a, err := mail.ParseAddress(email); err != nil || a.Address != email {
			return nil, e, fmt.Errorf("%q isn't an email address", in.Email)
		}
		creator := c.Principal.UserID
		var by *uuid.UUID
		if creator != uuid.Nil {
			by = &creator
		}
		id, err := s.Q.McpAddSuppression(ctx, store.McpAddSuppressionParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: c.Workspace.ID, Email: email, Detail: "MCP agent " + c.Principal.Name + ": " + in.Reason, CreatedBy: by})
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]any{"email": email, "suppressed": true, "message": "Already suppressed."}, e, nil
		} else if err != nil {
			return nil, e, err
		}
		e.targetID = id.String()
		return map[string]any{"email": email, "suppressed": true}, e, nil
	}, nil)

	tool(s, srv, "get_provider_health", authz.ProvidersView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "provider.health_read", targetType: "provider", targetText: keep("target.providers", nil)}
		rows, err := s.Q.McpProviders(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		var healthy []string
		for _, r := range rows {
			out = append(out, map[string]any{"name": r.Name, "adapter": r.Adapter, "label": r.Label, "status": r.Status, "detail": r.StatusDetail, "health": r.Health, "ips": r.Ips, "last_event_at": r.LastEventAt})
			if r.Status == "healthy" {
				healthy = append(healthy, r.Name)
			} else {
				e.summaryParts = append(e.summaryParts, keep("outcome.status", i18n.Args{"name": r.Name, "status": r.Status}))
			}
		}
		if len(healthy) > 0 {
			e.summaryParts = append(e.summaryParts, keep("outcome.healthy", i18n.Args{"names": healthy}))
		}
		return map[string]any{"providers": out, "note": "Credentials are never shown to agents."}, e, nil
	}, nil)

	tool(s, srv, "get_routing", authz.ProvidersView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "routing.read", targetType: "routing_rule", targetText: keep("target.streams", nil)}
		names, err := s.providerNames(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		rows, err := s.Q.McpRouting(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		for _, r := range rows {
			out = append(out, map[string]any{"id": r.ID, "stream": r.Stream, "priority": r.Priority, "name": r.Name, "conditions": r.Conditions,
				"weights": weights(r.Splits, names), "fallback": nameOf(r.FallbackProviderID, names), "enabled": r.Enabled})
		}
		return map[string]any{"rules": out}, e, nil
	}, nil)

	tool(s, srv, "update_routing", authz.ProvidersManage, func(ctx context.Context, c *agentCall, in routingIn) (any, entry, error) {
		e := entry{action: "routing.updated", targetType: "routing_rule", targetName: streamName(in.Stream), targetText: keep("target.stream", i18n.Args{"stream": in.Stream})}
		if in.Stream != "broadcast" && in.Stream != "automation" && in.Stream != "system" {
			return nil, e, errors.New("stream must be broadcast, automation or system")
		}
		rule, err := s.Q.McpDefaultRoutingRule(ctx, store.McpDefaultRoutingRuleParams{WorkspaceID: c.Workspace.ID, Stream: in.Stream})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, e, fmt.Errorf("the %s stream has no routing rule yet; set one up in ReputeCast", in.Stream)
		} else if err != nil {
			return nil, e, err
		}
		e.targetID = rule.ID.String()
		provs, err := s.Q.McpProviders(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		byName := map[string]uuid.UUID{}
		names := map[uuid.UUID]string{}
		for _, p := range provs {
			byName[strings.ToLower(p.Name)] = p.ID
			names[p.ID] = p.Name
		}
		total := 0
		type split struct {
			ProviderID string `json:"provider_id"`
			Weight     int    `json:"weight"`
		}
		var splits []split
		keys := make([]string, 0, len(in.Weights))
		for k := range in.Weights {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			w := in.Weights[k]
			id, ok := byName[strings.ToLower(strings.TrimSpace(k))]
			if !ok {
				return nil, e, fmt.Errorf("no connected sending provider named %q", k)
			}
			if w < 0 || w > 100 {
				return nil, e, errors.New("weights are percentages from 0 to 100")
			}
			if w > 0 {
				splits = append(splits, split{id.String(), w})
			}
			total += w
		}
		if total != 100 {
			return nil, e, fmt.Errorf("weights add up to %d; they must add up to 100", total)
		}
		raw, _ := json.Marshal(splits)
		if err := s.Q.McpSetRoutingSplits(ctx, store.McpSetRoutingSplitsParams{ID: rule.ID, WorkspaceID: c.Workspace.ID, Splits: raw}); err != nil {
			return nil, e, err
		}
		before, after := weights(rule.Splits, names), weights(raw, names)
		e.before = map[string]any{"stream": in.Stream, "weights": before}
		e.after = map[string]any{"stream": in.Stream, "weights": after}
		e.summary = weightLine(before) + " → " + weightLine(after)
		return map[string]any{"stream": in.Stream, "weights": after}, e, nil
	}, nil)

	tool(s, srv, "list_automations", authz.AutomationsView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "automation.listed", targetType: "automation", targetText: keep("target.automations", nil)}
		rows, err := s.Q.McpAutomations(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		out := []map[string]any{}
		for _, r := range rows {
			out = append(out, map[string]any{"id": r.ID, "name": r.Name, "status": r.Status, "trigger": r.Trigger, "updated_at": r.UpdatedAt})
		}
		return map[string]any{"automations": out}, e, nil
	}, nil)

	tool(s, srv, "get_deliverability", authz.DeliverabilityView, func(ctx context.Context, c *agentCall, _ noIn) (any, entry, error) {
		e := entry{action: "deliverability.read", targetType: "sending_domain"}
		doms, err := s.Q.McpSendingDomains(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		bl, err := s.Q.McpBlocklists(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		pm, err := s.Q.McpPostmaster(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		ds := []map[string]any{}
		var names []string
		for _, d := range doms {
			ds = append(ds, map[string]any{"domain": d.Domain, "status": d.Status, "authentication": d.AuthSummary, "checked_at": d.LastCheckedAt})
			names = append(names, d.Domain)
		}
		e.targetName = strings.Join(names, ", ")
		lists := []map[string]any{}
		for _, b := range bl {
			lists = append(lists, map[string]any{"asset": b.Asset, "list": b.Source, "impact": b.Impact, "since": b.ListedSince})
		}
		post := []map[string]any{}
		for _, p := range pm {
			post = append(post, map[string]any{"domain": p.Domain, "day": p.Day.Format("2006-01-02"), "domain_reputation": p.DomainReputation, "ip_reputation": p.IpReputation, "spam_rate": p.SpamRate})
		}
		if len(lists) == 0 {
			e.summaryText = keep("outcome.clear", nil)
		} else {
			e.summaryText = keep("outcome.listings", i18n.Args{"count": len(lists)})
		}
		return map[string]any{"domains": ds, "blocklists": lists, "postmaster": post}, e, nil
	}, nil)

	tool(s, srv, "send_test_email", authz.CampaignsEdit, func(ctx context.Context, c *agentCall, in testIn) (any, entry, error) {
		e := entry{action: "campaign.test_sent", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		to := strings.ToLower(strings.TrimSpace(in.To))
		var member bool
		_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE m.workspace_id = $1 AND m.status = 'active' AND u.email = $2)`, c.Workspace.ID, to).Scan(&member)
		if !member {
			return nil, e, errors.New("tests go only to members of this workspace")
		}
		e.detailText = keep("detail.to", i18n.Args{"email": to})
		var html, text string
		_ = s.Pool.QueryRow(ctx, `SELECT html, plain_text FROM campaigns WHERE id = $1`, cp.ID).Scan(&html, &text)
		if s.Mail == nil {
			return nil, e, errors.New("test emails need a sending provider on the System stream")
		}
		wsID := c.Workspace.ID
		provider, err := s.Mail.SendSystem(ctx, mailer.Message{WorkspaceID: &wsID, To: to, Subject: "[Test] " + cp.Subject, HTML: html, Text: text, Kind: "campaign_test"})
		if errors.Is(err, mailer.ErrNoSystemProvider) {
			return nil, e, errors.New("test emails need a sending provider on the System stream; connect one in ReputeCast")
		} else if err != nil {
			return nil, e, err
		}
		return map[string]any{"sent_to": to, "via": provider}, e, nil
	}, nil)

	s.addLanguageTools(srv)
	s.addTransactionalTools(srv)
}

func streamName(s string) string {
	switch s {
	case "broadcast":
		return "Broadcasts"
	case "automation":
		return "Automations"
	case "system":
		return "System"
	}
	return s
}

func (s *Service) providerNames(ctx context.Context, ws uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := s.Q.McpProviders(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]string{}
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out, nil
}

func nameOf(id *uuid.UUID, names map[uuid.UUID]string) string {
	if id == nil {
		return ""
	}
	return names[*id]
}

type weightEntry struct {
	Provider string `json:"provider"`
	Weight   int    `json:"weight"`
}

func weights(raw json.RawMessage, names map[uuid.UUID]string) []weightEntry {
	var splits []struct {
		ProviderID uuid.UUID `json:"provider_id"`
		Weight     int       `json:"weight"`
	}
	_ = json.Unmarshal(raw, &splits)
	out := []weightEntry{}
	for _, sp := range splits {
		n := names[sp.ProviderID]
		if n == "" {
			n = sp.ProviderID.String()
		}
		out = append(out, weightEntry{n, sp.Weight})
	}
	return out
}

func weightLine(ws []weightEntry) string {
	parts := make([]string, len(ws))
	for i, w := range ws {
		parts[i] = fmt.Sprintf("%s %d%%", w.Provider, w.Weight)
	}
	return strings.Join(parts, " / ")
}

// propose records an action the agent can't take for a person to review: a
// notification to the first member who holds the permission.
func (s *Service) propose(ctx context.Context, c *agentCall, tool string, perm authz.Permission, e entry) {
	helpers := access.Helpers(ctx, s.Q, c.Workspace.ID, perm)
	if len(helpers) == 0 {
		return
	}
	var userID uuid.UUID
	members, err := s.Q.TeamListMembers(ctx, c.Workspace.ID)
	if err != nil {
		return
	}
	for _, m := range members {
		if (m.UserName == helpers[0] || m.Email == helpers[0]) && m.Status == "active" {
			userID = m.UserID
			break
		}
	}
	if userID == uuid.Nil {
		return
	}
	verb := strings.ReplaceAll(strings.TrimSuffix(tool, "_campaign"), "_", " ")
	if v, ok := proposeVerbs[tool]; ok {
		verb = v
	}
	link := "/w/" + c.Workspace.Slug + "/settings/mcp"
	if e.targetType == "campaign" && e.targetID != "" {
		link = "/w/" + c.Workspace.Slug + "/campaigns/" + e.targetID + "/live"
	}
	if e.targetType == "website_post" && e.targetID != "" {
		link = "/w/" + c.Workspace.Slug + "/forms/website/posts/" + e.targetID
	}
	expires := s.now().Add(time.Duration(c.Settings.ApprovalExpiryHours) * time.Hour)
	// Read later by the member, in their own language; the time in the
	// workspace's timezone.
	_ = notify.ToUser(ctx, s.Pool, userID, notify.Notification{
		WorkspaceID: &c.Workspace.ID, Kind: "agent_proposal", Severity: "warn",
		TitleText: keep("notify.proposalTitle", i18n.Args{"agent": c.Principal.Name, "tool": tool, "verb": verb, "target": e.targetName, "hasTarget": yesNo(e.targetName != "")}),
		BodyText:  keep("notify.proposalBody", i18n.Args{"permission": string(perm), "at": expires.In(zone(c.Workspace.Timezone))}),
		Link:      link, Data: map[string]any{"tool": tool, "agent_id": c.Principal.ID.String(), "expires_at": expires},
	})
}
