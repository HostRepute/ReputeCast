package assistant

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Screen is a screen of the app the assistant knows: its route in the web
// app (web/src/routeTree.gen.ts), the permission it needs, and what it is
// for. The app only shows links to these.
type Screen struct {
	Key, Route, Title, Purpose string
	Permission                 authz.Permission
	// Param is the route's resource id parameter ("" for none).
	Param string
}

var screens = []Screen{
	{Key: "overview", Route: "/w/$workspace/", Title: "Overview", Purpose: "The workspace at a glance: recent sends, audience growth and what needs attention.", Permission: authz.AIUse},
	{Key: "campaigns", Route: "/w/$workspace/campaigns/", Title: "Campaigns", Purpose: "Every campaign by status: drafts, scheduled, sending and sent. New campaign starts one.", Permission: authz.CampaignsView},
	{Key: "campaign.edit", Route: "/w/$workspace/campaigns/$campaignId/edit", Param: "campaignId", Title: "Composer", Purpose: "Edit a campaign's subject, preheader and content; pending AI proposals for it are reviewed here.", Permission: authz.CampaignsView},
	{Key: "campaign.review", Route: "/w/$workspace/campaigns/$campaignId/review", Param: "campaignId", Title: "Review & send", Purpose: "Pick the audience and sender, run Send Readiness, then send, schedule or ask for approval.", Permission: authz.CampaignsView},
	{Key: "campaign.live", Route: "/w/$workspace/campaigns/$campaignId/live", Param: "campaignId", Title: "Live sending", Purpose: "Follow a send as it goes out, and pause or resume it.", Permission: authz.CampaignsView},
	{Key: "campaign.report", Route: "/w/$workspace/campaigns/$campaignId/report", Param: "campaignId", Title: "Campaign report", Purpose: "A sent campaign's delivered, opens, clicks, bounces, complaints and unsubscribes.", Permission: authz.ReportsView},
	{Key: "audience", Route: "/w/$workspace/audience/", Title: "Audience", Purpose: "Contacts, their lists and their activity.", Permission: authz.SubscribersView},
	{Key: "audience.segments", Route: "/w/$workspace/audience/segments/", Title: "Segments", Purpose: "Dynamic and static segments of the audience.", Permission: authz.SubscribersView},
	{Key: "audience.segment", Route: "/w/$workspace/audience/segments/$segmentId", Param: "segmentId", Title: "Segment", Purpose: "One segment's conditions and size.", Permission: authz.SubscribersView},
	{Key: "audience.suppressions", Route: "/w/$workspace/audience/suppressions", Title: "Suppressions", Purpose: "Addresses that never receive mail: bounces, complaints, unsubscribes.", Permission: authz.SubscribersView},
	{Key: "automations", Route: "/w/$workspace/automations/", Title: "Automations", Purpose: "Flows that send when something happens.", Permission: authz.AutomationsView},
	{Key: "automation", Route: "/w/$workspace/automations/$automationId", Param: "automationId", Title: "Automation", Purpose: "One flow: its trigger, emails, delays and branches.", Permission: authz.AutomationsView},
	{Key: "approvals", Route: "/w/$workspace/approvals/", Title: "Approvals", Purpose: "Campaigns waiting for review before they can be sent.", Permission: authz.CampaignsView},
	{Key: "agent.new", Route: "/w/$workspace/agent/new", Title: "New agent task", Purpose: "Brief the AI agent, review what is shared, and start a run.", Permission: authz.AIUse},
	{Key: "deliverability", Route: "/w/$workspace/deliverability/", Title: "Deliverability", Purpose: "Placement tests and the signals of how mail lands.", Permission: authz.DeliverabilityView},
	{Key: "providers", Route: "/w/$workspace/providers/", Title: "Providers", Purpose: "Sending providers, their quotas, routing and events.", Permission: authz.ProvidersView},
	{Key: "settings.senders", Route: "/w/$workspace/settings/senders/", Title: "Senders", Purpose: "Sending domains and their DNS records.", Permission: authz.ProvidersView},
	{Key: "settings.ai", Route: "/w/$workspace/settings/integrations/ai", Title: "AI models", Purpose: "Model accounts, each feature's model, the monthly cap and personal keys.", Permission: authz.IntegrationsView},
	{Key: "settings.team", Route: "/w/$workspace/settings/team/", Title: "Team", Purpose: "Members, their roles and invitations.", Permission: authz.TeamView},
}

func screenOf(key string) (Screen, bool) {
	for _, s := range screens {
		if s.Key == key {
			return s, true
		}
	}
	return Screen{}, false
}

// path is a screen's address in a workspace.
func (sc Screen) path(slug string, id uuid.UUID) string {
	p := strings.Replace(sc.Route, "$workspace", slug, 1)
	if sc.Param != "" {
		p = strings.Replace(p, "$"+sc.Param, id.String(), 1)
	}
	return p
}

func runDescribeScreen(_ context.Context, _ *Service, c *toolCall) toolResult {
	in, ok := decode[struct {
		Screen string `json:"screen"`
	}](c)
	if !ok {
		return failed("invalid_arguments")
	}
	key := in.Screen
	if key == "" {
		key = c.thread.Screen
	}
	sc, ok := screenOf(key)
	if !ok {
		return failed("unknown_screen")
	}
	return toolResult{data: map[string]any{"screen": sc.Key, "title": sc.Title, "purpose": sc.Purpose}, label: sc.Title}
}

func runSuggestLink(_ context.Context, _ *Service, c *toolCall) toolResult {
	in, ok := decode[struct {
		Screen string `json:"screen"`
		ID     string `json:"id"`
	}](c)
	sc, known := screenOf(in.Screen)
	if !ok || !known {
		return failed("unknown_screen")
	}
	if !c.owner.Can(sc.Permission) {
		return failed("forbidden")
	}
	id := uuid.Nil
	if sc.Param != "" {
		var err error
		if id, err = uuid.Parse(in.ID); err != nil {
			return failed("invalid_arguments")
		}
	}
	card := &AIThreadCard{Kind: "link", Title: sc.Title, Path: sc.path(c.ws.Slug, id)}
	return toolResult{data: map[string]any{"shown": "a link card to " + sc.Title}, label: sc.Title, card: card}
}

func runReadAttached(ctx context.Context, s *Service, c *toolCall) toolResult {
	in, _ := decode[struct {
		Handle string `json:"handle"`
	}](c)
	a, ok := c.attachment(in.Handle)
	if !ok {
		return failed("not_attached")
	}
	switch a.Kind {
	case "campaign":
		camp, err := s.AI.AssistantReadCampaign(ctx, s.Q, c.ws, c.owner, a.TargetID)
		if err != nil {
			return failed(problemCode(err))
		}
		return toolResult{data: map[string]any{"kind": "campaign", "campaign": camp}, label: camp.Name}
	default:
		att, err := s.AI.AssistantAttachment(ctx, s.Q, c.ws, c.owner, a.Kind, a.TargetID, false)
		if err != nil {
			return failed(problemCode(err))
		}
		return toolResult{data: map[string]any{"kind": a.Kind, "title": att.Title, "hint": "read_report has its numbers"}, label: att.Title}
	}
}

func runReadReport(ctx context.Context, s *Service, c *toolCall) toolResult {
	in, _ := decode[struct {
		Handle string `json:"handle"`
	}](c)
	a, ok := c.attachment(in.Handle)
	if !ok || a.Kind != "report" {
		return failed("not_attached")
	}
	// The person's access now decides; the numbers are the snapshot taken
	// when they attached it.
	att, err := s.AI.AssistantAttachment(ctx, s.Q, c.ws, c.owner, "report", a.TargetID, false)
	if err != nil {
		return failed(problemCode(err))
	}
	if len(a.SnapshotSealed) == 0 {
		return failed("report_unavailable")
	}
	raw, err := s.Box.Open(a.SnapshotSealed, contextAD(a.ID))
	if err != nil {
		return failed("unavailable")
	}
	return toolResult{data: map[string]any{"title": att.Title, "snapshot": json.RawMessage(raw)}, label: att.Title}
}

func runProposeEdit(ctx context.Context, s *Service, c *toolCall) toolResult {
	in, ok := decode[struct {
		Handle       string `json:"handle"`
		BaseRevision int64  `json:"baseRevision"`
		Subject      string `json:"subject"`
		Preheader    string `json:"preheader"`
		Source       string `json:"source"`
	}](c)
	if !ok {
		return failed("invalid_arguments")
	}
	a, attached := c.attachment(in.Handle)
	if !attached || a.Kind != "campaign" {
		return failed("not_attached")
	}
	id, err := s.AI.AssistantProposeCampaign(ctx, c.ws, c.owner, c.thread.ID, a.TargetID, in.BaseRevision,
		ai.AIDraftEmail{Subject: in.Subject, Preheader: in.Preheader, MJML: in.Source})
	if err != nil {
		return failed(problemCode(err))
	}
	sc, _ := screenOf("campaign.edit")
	card := &AIThreadCard{Kind: "proposal", Title: "Review the proposal", Path: sc.path(c.ws.Slug, a.TargetID), ProposalID: &id}
	return toolResult{data: map[string]any{"proposal": id, "shown": "a card to review the proposal in the composer"}, label: in.Subject, card: card}
}

func runDraftAgentTask(_ context.Context, _ *Service, c *toolCall) toolResult {
	in, ok := decode[struct {
		Title string `json:"title"`
		Brief string `json:"brief"`
	}](c)
	if !ok || strings.TrimSpace(in.Brief) == "" {
		return failed("invalid_arguments")
	}
	sc, _ := screenOf("agent.new")
	card := &AIThreadCard{Kind: "agent_task", Title: clip(in.Title, 120), Path: sc.path(c.ws.Slug, uuid.Nil), Brief: clip(in.Brief, 4000)}
	return toolResult{data: map[string]any{"shown": "a card to review and start the task"}, label: card.Title, card: card}
}

func runSubjectIdeas(ctx context.Context, s *Service, c *toolCall) toolResult {
	in, _ := decode[struct {
		Handle string `json:"handle"`
	}](c)
	a, ok := c.attachment(in.Handle)
	if !ok || a.Kind != "campaign" {
		return failed("not_attached")
	}
	ideas, err := s.AI.AssistantSubjectIdeas(ctx, c.ws, c.owner, a.TargetID)
	if err != nil {
		return failed(problemCode(err))
	}
	out := []map[string]any{}
	for _, i := range ideas.Ideas {
		out = append(out, map[string]any{"subject": i.Subject, "verdict": i.Verdict, "heuristics": i.Heuristics})
	}
	return toolResult{data: map[string]any{"ideas": out}, label: "Subject ideas"}
}

func runRewrite(ctx context.Context, s *Service, c *toolCall) toolResult {
	in, ok := decode[struct {
		Handle      string `json:"handle"`
		Text        string `json:"text"`
		Action      string `json:"action"`
		Instruction string `json:"instruction"`
	}](c)
	a, attached := c.attachment(in.Handle)
	if !ok || !attached || a.Kind != "campaign" {
		return failed("not_attached")
	}
	r, err := s.AI.AssistantRewrite(ctx, c.ws, c.owner, a.TargetID, in.Text, in.Action, in.Instruction)
	if err != nil {
		return failed(problemCode(err))
	}
	return toolResult{data: map[string]any{"html": r.HTML}, label: "Rewrite"}
}
