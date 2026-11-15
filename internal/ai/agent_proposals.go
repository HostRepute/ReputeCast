package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// The original graph and email bindings remain local. Only Task, projected
// with run-local node/email handles, crosses the model boundary.
type agentFlowSource struct {
	Context automationDraftContext
	Emails  map[string]draftContext
	Task    automationDraftTask
	Trigger automations.AutomationTrigger
}

func flowProposalRevision(row store.Automation, graph automations.AutomationGraph) string {
	raw, _ := json.Marshal([]any{row, graph})
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func (s *Service) prepareAgentProposal(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, kind string, target uuid.UUID, handle string, literals privateLiterals) (agentProposalSource, any, error) {
	out := agentProposalSource{ID: target, Kind: kind}
	if err := agentKindAccess(p, kind, true); err != nil {
		return out, nil, err
	}
	draft := draftContext{EmailContext: editorial.EmailContext{BindingPrefix: handle}}
	switch kind {
	case "campaign":
		row, err := draftCampaign(ctx, q, ws, p, target)
		if err != nil {
			return out, nil, err
		}
		if row.Status != "draft" {
			return out, nil, draftProblem(ctx, 409, "changed")
		}
		out.Revision = int64(row.Version)
		draft.Name, draft.Before = row.Name, &AIDraftEmail{Subject: row.Subject, Preheader: row.Preheader, MJML: row.Mjml}
	case "post":
		row, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: target})
		if err != nil {
			return out, nil, api.MapError(err, "Post")
		}
		if row.Kind != website.KindWeb || row.Status != website.PostDraft || row.FirstPublishedAt != nil {
			return out, nil, draftProblem(ctx, 409, "changed")
		}
		out.Revision, out.Snapshot = int64(row.Version), website.ProposalRevision(row)
		draft.Name, draft.Before = row.Title, &AIDraftEmail{Subject: row.Title, Preheader: row.Excerpt, MJML: row.Mjml}
	case "automation":
		row, graph, err := s.agentFlow(ctx, q, ws, p, target)
		if err != nil {
			return out, nil, err
		}
		if row.Status != automations.StatusDraft || row.LiveVersion != nil {
			return out, nil, draftProblem(ctx, 409, "changed")
		}
		start := graph.Trigger()
		if start == nil || start.Data.Trigger == nil {
			return out, nil, draftProblem(ctx, 422, "source")
		}
		trigger := *start.Data.Trigger
		if trigger.Kind != automations.TriggerJoinedList && trigger.Kind != automations.TriggerSegmentEntered && trigger.Kind != automations.TriggerSegmentLeft {
			return out, nil, draftProblem(ctx, 422, "source")
		}
		// Check against the safe subset after masking reference IDs. Do not
		// silently broaden or replace enrollment semantics from the source.
		projectedTrigger := automations.AutomationTrigger{Kind: trigger.Kind, Reentry: "once"}
		if trigger.Kind == automations.TriggerJoinedList {
			projectedTrigger.ListID, projectedTrigger.ConfirmedOnly, projectedTrigger.ExitOnLeave = "selected_list", true, true
		} else {
			projectedTrigger.SegmentID = "selected_segment"
		}
		flow := &agentFlowSource{Context: automationDraftContext{Name: row.Name, Before: &graph}, Emails: map[string]draftContext{}, Task: automationDraftTask{Emails: map[string]campaignDraftTask{}}, Trigger: trigger}
		if err := flow.Context.projectFlow(ctx, q, ws, p, graph, &flow.Task, flow.Emails, nil, projectedTrigger, handle); err != nil {
			return out, nil, err
		}
		for key, task := range flow.Task.Emails {
			literals.source(&task)
			flow.Task.Emails[key] = task
		}
		out.Revision, out.Snapshot, out.Flow = row.DraftRevision, flowProposalRevision(row, graph), flow
		return out, flow.Task, nil
	default:
		return out, nil, agentProblem(ctx, "input")
	}
	task, err := draft.projectSource(ctx, draft.Before.MJML, nil)
	if err != nil {
		return out, nil, err
	}
	literals.source(&task)
	out.Draft = draft
	return out, task, nil
}

func (s *Service) agentFlow(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (store.Automation, automations.AutomationGraph, error) {
	row, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return row, automations.AutomationGraph{}, api.MapError(err, "Automation")
	}
	version, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: row.DraftVersion})
	if err != nil {
		return row, automations.AutomationGraph{}, err
	}
	graph, err := automations.ParseGraph(version.Graph)
	if err != nil {
		return row, graph, err
	}
	err = s.Automations.CheckProposalScope(ctx, q, ws.ID, p, graph)
	return row, graph, err
}

func (s *Service) agentProposalAccess(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, source agentProposalSource, strict bool) error {
	if err := agentKindAccess(p, source.kind(), strict); err != nil {
		return err
	}
	switch source.kind() {
	case "post":
		row, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: source.ID})
		if err != nil {
			return api.MapError(err, "Post")
		}
		if strict && (row.Status != website.PostDraft || row.FirstPublishedAt != nil || int64(row.Version) != source.Revision || website.ProposalRevision(row) != source.Snapshot) {
			return draftProblem(ctx, 409, "changed")
		}
	case "automation":
		row, graph, err := s.agentFlow(ctx, q, ws, p, source.ID)
		if err != nil {
			return err
		}
		if strict && (row.Status != automations.StatusDraft || row.LiveVersion != nil || row.DraftRevision != source.Revision || flowProposalRevision(row, graph) != source.Snapshot) {
			return draftProblem(ctx, 409, "changed")
		}
		if err := s.Automations.CheckProposalScope(ctx, q, ws.ID, p, *source.Flow.Context.Before); err != nil {
			return err
		}
		for _, email := range source.Flow.Context.Sources {
			row, err := automationSource(ctx, q, ws, p, email.ID, email.Version)
			if err != nil {
				return err
			}
			if strict && (row.Subject != email.Content.Subject || row.Preheader != email.Content.Preheader || row.Mjml != email.Content.MJML) {
				return draftProblem(ctx, 409, "changed")
			}
		}
	}
	return nil
}

func (c agentContext) proposalFor(a store.AiAgentArtifact) (agentProposalSource, bool) {
	i := slices.IndexFunc(c.Plan.Items, func(item AIAgentPlanItem) bool { return item.Handle == a.Handle })
	if i < 0 {
		return agentProposalSource{}, false
	}
	source, ok := c.Proposals[c.Plan.Items[i].Target]
	return source, ok
}
