package ai

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// Audience semantics are selected by a person, not inferred from model text.
// FromAudience binds to a segment in the same confirmed dependency bundle.
type AIAgentAutomationSelection struct {
	Trigger      string     `json:"trigger" enum:"joined_list,segment_entered,segment_left"`
	ListID       *uuid.UUID `json:"listId,omitempty"`
	SegmentID    *uuid.UUID `json:"segmentId,omitempty"`
	FromAudience bool       `json:"fromAudience"`
}

type agentStageAutomation struct {
	Handle  string              `json:"handle" pattern:"^[a-z][a-z0-9_]{0,39}$"`
	Content AIAutomationContent `json:"content"`
}

type agentAutomationContext struct {
	Name     string
	Policies map[string]mjml.GeneratedPolicy
	Previews []string
	Source   *agentFlowSource
}

func (s *Service) agentAutomationAccess(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, c agentContext, execute bool) error {
	if !p.Can(authz.AutomationsView) || (execute && !p.Can(authz.AutomationsEdit)) {
		return agentProblem(ctx, "permission")
	}
	a := c.Brief.Automation
	if a.Trigger == automations.TriggerJoinedList {
		if a.ListID == nil || a.SegmentID != nil || a.FromAudience {
			return agentProblem(ctx, "input")
		}
		_, err := q.AIDraftList(ctx, store.AIDraftListParams{WorkspaceID: ws.ID, ID: *a.ListID, ScopeLists: access.ScopeLists(p)})
		return api.MapError(err, "List")
	}
	if (a.Trigger != automations.TriggerSegmentEntered && a.Trigger != automations.TriggerSegmentLeft) || a.ListID != nil || (a.SegmentID != nil) == a.FromAudience || (a.FromAudience && c.Brief.Audience == nil) {
		return agentProblem(ctx, "input")
	}
	if !p.Can(authz.SubscribersView) || len(p.Scope.Lists) != 0 {
		return api.Forbidden(authz.SubscribersView, nil)
	}
	if a.SegmentID != nil {
		_, err := q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws.ID, ID: *a.SegmentID})
		return api.MapError(err, "Segment")
	}
	return nil
}

// Project only the selected trigger semantics, never the resource ID/name.
// Segment handles are supplied by the confirmed plan's dependency edge.
func (c agentContext) automationTrigger(item AIAgentPlanItem) (automations.AutomationTrigger, error) {
	if item.Target != "" {
		source, ok := c.Proposals[item.Target]
		if !ok || source.kind() != "automation" || source.Flow == nil {
			return automations.AutomationTrigger{}, errors.New("unselected automation")
		}
		return *source.Flow.Task.Graph.Trigger().Data.Trigger, nil
	}
	a := c.Brief.Automation
	if a == nil {
		return automations.AutomationTrigger{}, errors.New("unselected automation")
	}
	t := automations.AutomationTrigger{Kind: a.Trigger, Reentry: "once"}
	if a.Trigger == automations.TriggerJoinedList {
		t.ListID, t.ConfirmedOnly, t.ExitOnLeave = "selected_list", true, true
	} else {
		t.SegmentID = "selected_segment"
	}
	if a.FromAudience {
		if len(item.DependsOn) != 1 || !slices.ContainsFunc(c.Plan.Items, func(dep AIAgentPlanItem) bool {
			return dep.Handle == item.DependsOn[0] && dep.Kind == "segment" && dep.Bundle == item.Bundle && dep.Target == ""
		}) {
			return t, errors.New("missing segment dependency")
		}
		t.SegmentID = item.DependsOn[0]
	}
	return t, nil
}

func (s *Service) stageAgentAutomation(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, run *store.AiAgentRun, c *agentContext, call llm.ToolCall) (json.RawMessage, bool, error) {
	denied := json.RawMessage(`{"ok":false,"code":"tool_not_allowed"}`)
	var in agentStageAutomation
	if decodeDraft(call.Arguments, &in) != nil {
		return json.RawMessage(`{"ok":false,"code":"invalid_flow_schema"}`), true, nil //nolint:nilerr // Record the repairable tool failure, not a transaction failure or raw error.
	}
	i := slices.IndexFunc(c.Plan.Items, func(item AIAgentPlanItem) bool { return item.Handle == in.Handle && item.Kind == "automation" })
	if i < 0 || !p.Can(authz.AutomationsEdit) || (c.Refine != "" && c.Refine != in.Handle) {
		return denied, true, nil
	}
	item := c.Plan.Items[i]
	trigger, err := c.automationTrigger(item)
	if err != nil {
		return denied, true, nil //nolint:nilerr // Invalid tool arguments must commit a denied result without exposing the error.
	}
	if c.Discarded[item.Bundle] {
		return json.RawMessage(`{"ok":false,"code":"bundle_discarded"}`), false, nil
	}
	all, err := q.AIAgentArtifacts(ctx, run.ID)
	if err != nil {
		return nil, false, err
	}
	if slices.ContainsFunc(all, func(a store.AiAgentArtifact) bool { return a.Handle == in.Handle && a.Decision != "" }) {
		return json.RawMessage(`{"ok":false,"code":"already_decided"}`), true, nil
	}
	for _, dep := range item.DependsOn {
		if !slices.ContainsFunc(all, func(a store.AiAgentArtifact) bool {
			return a.Handle == dep && a.Bundle == item.Bundle && a.Decision == ""
		}) {
			return json.RawMessage(`{"ok":false,"code":"stage_dependency_first"}`), true, nil
		}
	}
	if checkDraftFlow(in.Content, trigger) != nil {
		return json.RawMessage(`{"ok":false,"code":"invalid_flow_graph"}`), true, nil //nolint:nilerr // Record a safe repair code without rolling back the attempted call.
	}
	local := agentAutomationContext{Name: item.Name, Policies: map[string]mjml.GeneratedPolicy{}, Previews: []string{}}
	var target *uuid.UUID
	var base int64
	if item.Target != "" {
		source := c.Proposals[item.Target]
		local.Source, target, base = source.Flow, &source.ID, source.Revision
		for key := range source.Flow.Emails {
			if !slices.ContainsFunc(in.Content.Emails, func(e automations.ProposedEmail) bool { return e.Key == key }) {
				return json.RawMessage(`{"ok":false,"code":"preserve_source_emails"}`), true, nil
			}
		}
	}
	for i := range in.Content.Emails {
		e := &in.Content.Emails[i]
		draft := c.Draft
		existing := false
		if local.Source != nil {
			if saved, ok := local.Source.Emails[e.Key]; ok {
				draft, existing = saved, true
			}
		}
		// A selected campaign is a source, not an automation email target.
		mail := AIDraftEmail{Subject: e.Subject, Preheader: e.Preheader, MJML: e.MJML}
		if _, safe := voiceText(e.Name); !safe || draft.restoreEmail(&mail) != nil {
			return json.RawMessage(`{"ok":false,"code":"invalid_email_bindings_or_markup"}`), true, nil //nolint:nilerr // The tool result carries the failure without exposing bound values.
		}
		if !existing {
			mail.MJML = s.BrandKit.Starter(ctx, ws, mail.MJML)
		}
		draft.FinalPolicy = savedDraftPolicy(mail.MJML)
		if err := draft.validateSaved(ctx, q, ws, mail); err != nil {
			if errors.Is(err, errDraftContent) {
				return json.RawMessage(`{"ok":false,"code":"invalid_email_content"}`), true, nil
			}
			return nil, false, err
		}
		html, _, issues := mjml.Checked(mail.MJML, "")
		if issues != nil {
			return json.RawMessage(`{"ok":false,"code":"invalid_email_compile"}`), true, nil
		}
		local.Policies[e.Key], local.Previews = draft.FinalPolicy, append(local.Previews, html)
		e.Subject, e.Preheader, e.MJML = mail.Subject, mail.Preheader, mail.MJML
		for j := range in.Content.Graph.Nodes {
			if node := in.Content.Graph.Nodes[j].Data.Email; node != nil && node.EmailID == e.Key {
				node.Subject = mail.Subject
			}
		}
	}
	in.Content.Graph = automations.Layout(in.Content.Graph)
	content, err := json.Marshal(in.Content)
	if err != nil {
		return nil, false, err
	}
	bindings, err := json.Marshal(local)
	if err != nil {
		return nil, false, err
	}
	if err := q.AIAgentArtifactSave(ctx, store.AIAgentArtifactSaveParams{ID: uuid.Must(uuid.NewV7()), RunID: run.ID, Handle: in.Handle, Kind: "automation", Bundle: item.Bundle,
		TargetID: target, BaseRevision: base, Content: content, Context: bindings, UpdatedAt: s.now()}); err != nil {
		return nil, false, err
	}
	if err := s.agentAudit(ctx, q, p, *run, "draft", map[string]any{"handle": in.Handle}); err != nil {
		return nil, false, err
	}
	result, _ := json.Marshal(map[string]any{"ok": true, "handle": in.Handle, "staged": true})
	return result, false, nil
}

// Resolve references only after preceding bundle members were kept in the
// same transaction. A later validator failure rolls back the entire bundle.
func (s *Service) keepAgentAutomation(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, c agentContext, a store.AiAgentArtifact, all []store.AiAgentArtifact) (uuid.UUID, error) {
	var content AIAutomationContent
	var local agentAutomationContext
	if err := json.Unmarshal(a.Context, &local); err != nil {
		return uuid.Nil, err
	}
	i := slices.IndexFunc(c.Plan.Items, func(item AIAgentPlanItem) bool { return item.Handle == a.Handle })
	if i < 0 {
		return uuid.Nil, agentProblem(ctx, "changed")
	}
	trigger, err := c.automationTrigger(c.Plan.Items[i])
	if err != nil || decodeDraft(a.Content, &content) != nil || checkDraftFlow(content, trigger) != nil {
		return uuid.Nil, draftProblem(ctx, 422, "invalid")
	}
	q := store.New(tx)
	for _, email := range content.Emails {
		if err := (draftContext{EmailContext: editorial.EmailContext{FinalPolicy: local.Policies[email.Key]}}).validateSaved(ctx, q, ws, AIDraftEmail{Subject: email.Subject, Preheader: email.Preheader, MJML: email.MJML}); err != nil {
			return uuid.Nil, draftProblem(ctx, 422, "invalid")
		}
	}
	if a.TargetID != nil {
		source, ok := c.proposalFor(a)
		if !ok || source.Flow == nil {
			return uuid.Nil, agentProblem(ctx, "changed")
		}
		if err := s.agentProposalAccess(ctx, q, ws, p, source, true); err != nil {
			return uuid.Nil, err
		}
		content.Graph.Trigger().Data.Trigger = &source.Flow.Trigger
		id, _, err := s.Automations.AcceptProposalIn(ctx, tx, ws, p, a.TargetID, a.BaseRevision, local.Name, content.Graph, content.Emails, audit.ProposalOrigin{ProposalID: a.ID, RunID: &a.RunID})
		return id, err
	}
	selection := c.Brief.Automation
	switch {
	case selection.ListID != nil:
		trigger.ListID = selection.ListID.String()
	case selection.SegmentID != nil:
		trigger.SegmentID = selection.SegmentID.String()
	default:
		dep := slices.IndexFunc(all, func(dep store.AiAgentArtifact) bool {
			return dep.Handle == trigger.SegmentID && dep.Bundle == a.Bundle && dep.Kind == "segment" && dep.Decision == "kept" && dep.ResultID != nil
		})
		if dep < 0 {
			return uuid.Nil, agentProblem(ctx, "bundle")
		}
		trigger.SegmentID = all[dep].ResultID.String()
	}
	content.Graph.Trigger().Data.Trigger = &trigger
	id, _, err := s.Automations.AcceptProposalIn(ctx, tx, ws, p, nil, 0, local.Name, content.Graph, content.Emails, audit.ProposalOrigin{ProposalID: a.ID, RunID: &a.RunID})
	return id, err
}
