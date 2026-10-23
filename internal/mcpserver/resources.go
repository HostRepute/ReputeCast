package mcpserver

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

const (
	mjmlTemplate   = "reputecast://campaigns/{id}.mjml"
	reportTemplate = "reputecast://campaigns/{id}/report.csv"
	auditResource  = "reputecast://audit/recent.json"
)

// resourceGuard authenticates a resource read and checks a permission.
func (s *Service) resourceGuard(ctx context.Context, req *mcp.ReadResourceRequest, perm authz.Permission, kind string) (context.Context, *agentCall, error) {
	ctx, c, err := call(ctx, req.Extra)
	if err != nil {
		return ctx, nil, err
	}
	if !c.Principal.Can(perm) {
		return ctx, c, s.deny(ctx, c, "read "+kind, perm, entry{action: "mcp.resource_read", targetType: kind, targetName: req.Params.URI})
	}
	return ctx, c, nil
}

func campaignID(uri, suffix string) (uuid.UUID, error) {
	rest := strings.TrimPrefix(uri, "reputecast://campaigns/")
	rest = strings.TrimSuffix(rest, suffix)
	return uuid.Parse(rest)
}

func (s *Service) addResources(srv *mcp.Server) {
	srv.AddResourceTemplate(&mcp.ResourceTemplate{Name: "campaign-mjml", Title: "Campaign MJML", URITemplate: mjmlTemplate, MIMEType: "text/mjml",
		Description: "The MJML source of a campaign. Needs campaigns:view."},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			ctx, c, err := s.resourceGuard(ctx, req, authz.CampaignsView, "campaign")
			if err != nil {
				return nil, err
			}
			id, err := campaignID(req.Params.URI, ".mjml")
			if err != nil {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			cp, err := s.Q.McpFindCampaign(ctx, store.McpFindCampaignParams{WorkspaceID: c.Workspace.ID, Ref: id.String()})
			if err != nil {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			s.record(ctx, c, "resource campaign MJML", authz.CampaignsView, entry{action: "campaign.content_read", targetType: "campaign", targetID: cp.ID.String(), targetName: cp.Name}, audit.Allowed, "", i18n.Text{})
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "text/mjml", Text: cp.Mjml}}}, nil
		})

	srv.AddResourceTemplate(&mcp.ResourceTemplate{Name: "campaign-report", Title: "Campaign report (CSV)", URITemplate: reportTemplate, MIMEType: "text/csv",
		Description: "Results by sending and mailbox provider. Needs reports:view."},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			ctx, c, err := s.resourceGuard(ctx, req, authz.ReportsView, "campaign")
			if err != nil {
				return nil, err
			}
			id, err := campaignID(req.Params.URI, "/report.csv")
			if err != nil {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			cp, err := s.Q.McpFindCampaign(ctx, store.McpFindCampaignParams{WorkspaceID: c.Workspace.ID, Ref: id.String()})
			if err != nil {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			rows, err := s.Pool.Query(ctx, `SELECT coalesce(p.name, ''), k.mailbox_provider, k.sent, k.delivered, k.opened_human, k.opened_machine, k.clicked, k.hard_bounced + k.soft_bounced, k.complained, k.unsubscribed
				FROM campaign_counters k LEFT JOIN providers p ON p.id = k.provider_id WHERE k.campaign_id = $1 ORDER BY 1, 2`, cp.ID)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			_ = w.Write([]string{"sending_provider", "mailbox_provider", "sent", "delivered", "human_opens", "machine_opens", "clicks", "bounces", "complaints", "unsubscribes"})
			for rows.Next() {
				var prov, box string
				var n [8]int64
				if err := rows.Scan(&prov, &box, &n[0], &n[1], &n[2], &n[3], &n[4], &n[5], &n[6], &n[7]); err != nil {
					return nil, err
				}
				rec := []string{prov, box}
				for _, v := range n {
					rec = append(rec, fmt.Sprint(v))
				}
				_ = w.Write(rec)
			}
			w.Flush()
			s.record(ctx, c, "resource report CSV", authz.ReportsView, entry{action: "report.read", targetType: "campaign", targetID: cp.ID.String(), targetName: cp.Name}, audit.Allowed, "", i18n.Text{})
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "text/csv", Text: buf.String()}}}, nil
		})

	srv.AddResource(&mcp.Resource{Name: "audit-log", Title: "Audit log (latest 100 entries)", URI: auditResource, MIMEType: "application/json",
		Description: "The workspace audit log, newest first. Needs audit:view."},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			ctx, c, err := s.resourceGuard(ctx, req, authz.AuditView, "audit_log")
			if err != nil {
				return nil, err
			}
			rows, err := s.Q.AuditLogList(ctx, store.AuditLogListParams{WorkspaceID: &c.Workspace.ID, Lim: 100})
			if err != nil {
				return nil, err
			}
			type item struct {
				OccurredAt string `json:"occurred_at"`
				Actor      string `json:"actor"`
				ActorType  string `json:"actor_type"`
				Action     string `json:"action"`
				Target     string `json:"target"`
				Result     string `json:"result"`
				Reason     string `json:"reason,omitempty"`
			}
			out := make([]item, 0, len(rows))
			for _, r := range rows {
				out = append(out, item{r.OccurredAt.UTC().Format("2006-01-02T15:04:05Z"), r.ActorName, r.ActorType, r.Action, r.TargetName, r.Result, r.Reason})
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			s.record(ctx, c, "resource audit log", authz.AuditView, entry{action: "audit.read", targetType: "audit_log", targetText: keep("target.auditLog", nil)}, audit.Allowed, "", i18n.Text{})
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(b)}}}, nil
		})
}

func (s *Service) addPrompts(srv *mcp.Server) {
	srv.AddPrompt(&mcp.Prompt{Name: "draft_weekly_digest", Title: "Draft the weekly digest",
		Description: "Draft the next Weekly Digest from what performed well recently.",
		Arguments:   []*mcp.PromptArgument{{Name: "topics", Description: "Topics or links to include, comma-separated"}}},
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			topics := req.Params.Arguments["topics"]
			text := "Draft the next Weekly Digest for this workspace.\n" +
				"1. Call list_campaigns with status sent and query \"Weekly Digest\" to find the latest issues.\n" +
				"2. Call get_campaign_report on the last two issues and note which links and topics got the most human clicks (ignore machine opens).\n" +
				"3. Write MJML 4 in the same structure as the last issue, with a subject under 60 characters and a preheader under 90.\n" +
				"4. Call create_campaign_draft. Don't schedule it: a person reviews Send Readiness and sends it."
			if topics != "" {
				text += "\nInclude these topics: " + topics + "."
			}
			return &mcp.GetPromptResult{Description: "Draft the weekly digest", Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}}}, nil
		})
	srv.AddPrompt(&mcp.Prompt{Name: "explain_auto_pause", Title: "Explain this auto-pause",
		Description: "Explain why a send paused automatically and what the options are.",
		Arguments:   []*mcp.PromptArgument{{Name: "campaign", Description: "The paused campaign's name or id", Required: true}}},
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			c := req.Params.Arguments["campaign"]
			text := fmt.Sprintf("Explain why the send of %q paused.\n"+
				"1. Call get_campaign_live_stats for the counters and the state of each sending provider's route.\n"+
				"2. Call get_provider_health and compare bounce and complaint rates with each provider's limits.\n"+
				"3. Say which threshold tripped, how many recipients are held, and the choices: resume, adjust the audience, move the held recipients to another provider, or cancel. "+
				"Resuming needs campaigns:send; if you don't hold it, say so instead of trying.", c)
			return &mcp.GetPromptResult{Description: "Explain this auto-pause", Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}}}, nil
		})
}
