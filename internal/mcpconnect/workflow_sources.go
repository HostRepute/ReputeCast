package mcpconnect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

type MCPProposedFlow struct {
	Graph  automations.AutomationGraph `json:"graph"`
	Emails []automations.ProposedEmail `json:"emails" minItems:"1" maxItems:"5"`
}

type MCPFlowSource struct {
	Graph  automations.AutomationGraph `json:"graph"`
	Emails map[string]editorial.Source `json:"emails"`
}

func workflowSource(ctx context.Context, q *store.Queries, ws uuid.UUID, p *authz.Principal, kind string, target uuid.UUID) (sourceContext, int64, bool, error) {
	var c sourceContext
	switch kind {
	case "list":
		row, err := q.AIDraftList(ctx, store.AIDraftListParams{WorkspaceID: ws, ID: target, ScopeLists: access.ScopeLists(p)})
		if err != nil {
			return c, 0, false, err
		}
		c.Snapshot = sourceRevision(row)
		list, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws, ID: target})
		if err != nil {
			return c, 0, false, err
		}
		c.Name, c.TriggerName = list.Name, list.Name
		return c, 0, false, nil
	case "segment":
		row, err := q.MCPConnectSegment(ctx, store.MCPConnectSegmentParams{WorkspaceID: ws, ID: target})
		if err != nil || row.Mode != "builder" {
			return c, 0, false, errRequest
		}
		c.Segment = &segments.SegmentDefinition{}
		if err := json.Unmarshal(row.Definition, c.Segment); err != nil {
			return c, 0, false, err
		}
		if _, err := (&segments.Service{}).CheckProposal(ctx, q, ws, p, *c.Segment); err != nil {
			return c, 0, false, err
		}
		c.Name, c.TriggerName, c.Snapshot = row.Name, row.Name, sourceRevision(row)
		return c, 0, false, nil
	case "automation":
		row, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws, ID: target})
		if err != nil {
			return c, 0, false, err
		}
		v, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: target, Version: row.DraftVersion})
		if err != nil {
			return c, 0, false, err
		}
		graph, err := automations.ParseGraph(v.Graph)
		if err != nil || len(graph.Nodes) > 30 || len(graph.Edges) > 60 {
			return c, 0, false, errRequest
		}
		if err := (&automations.Service{}).CheckProposalScope(ctx, q, ws, p, graph); err != nil {
			return c, 0, false, err
		}
		if graph.Trigger() == nil || graph.Trigger().Data.Trigger == nil {
			return c, 0, false, errRequest
		}
		trigger := graph.Trigger().Data.Trigger
		if _, err := projectedTrigger(*trigger); err != nil {
			return c, 0, false, errRequest
		}
		if trigger.Kind == automations.TriggerJoinedList {
			id, err := uuid.Parse(trigger.ListID)
			if err != nil {
				return c, 0, false, errRequest
			}
			list, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws, ID: id})
			if err != nil {
				return c, 0, false, err
			}
			c.TriggerName = list.Name
		} else {
			id, err := uuid.Parse(trigger.SegmentID)
			if err != nil {
				return c, 0, false, errRequest
			}
			segment, err := q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws, ID: id})
			if err != nil {
				return c, 0, false, err
			}
			c.TriggerName = segment.Name
		}
		c.Flow = &MCPProposedFlow{Graph: graph, Emails: []automations.ProposedEmail{}}
		seen := map[string]bool{}
		domains := make([]string, 0, len(p.Scope.SenderDomains))
		for _, domain := range p.Scope.SenderDomains {
			domains = append(domains, strings.ToLower(domain))
		}
		for _, node := range graph.Nodes {
			if node.Type != automations.NodeEmail || node.Data.Email == nil {
				continue
			}
			e := node.Data.Email
			key := fmt.Sprintf("%s:%d", e.EmailID, e.EmailVersion)
			if seen[key] {
				continue
			}
			id, err := uuid.Parse(e.EmailID)
			if err != nil || e.EmailVersion < 1 || e.EmailVersion > 2147483647 || len(seen) >= 5 {
				return c, 0, false, errRequest
			}
			mail, err := q.AIDraftAutomationEmail(ctx, store.AIDraftAutomationEmailParams{WorkspaceID: ws, ID: id, Version: int32(e.EmailVersion), ScopeDomains: domains})
			if err != nil {
				return c, 0, false, err
			}
			seen[key] = true
			c.Flow.Emails = append(c.Flow.Emails, automations.ProposedEmail{Key: key, Name: mail.Name, Subject: mail.Subject, Preheader: mail.Preheader, MJML: mail.Mjml})
		}
		c.Name, c.Snapshot = row.Name, sourceRevision([]any{row, c.Flow})
		return c, row.DraftRevision, row.Status == automations.StatusDraft && row.LiveVersion == nil, nil
	}
	return c, 0, false, errRequest
}

func safeWorkflowLabel(text string) bool {
	_, ok := editorial.Text(text)
	return ok && !editorial.URL.MatchString(text) && !editorial.Liquid.MatchString(text) && !editorial.Reserved.MatchString(text)
}

func projectedTrigger(t automations.AutomationTrigger) (automations.AutomationTrigger, error) {
	switch t.Kind {
	case automations.TriggerJoinedList:
		t.ListID = "selected_list"
	case automations.TriggerSegmentEntered, automations.TriggerSegmentLeft:
		t.SegmentID = "selected_segment"
	default:
		return t, errors.New("unsupported trigger")
	}
	// The allowlisted trigger is compared exactly by CheckDraftFlow, so source
	// semantics outside this subset cause refusal instead of being broadened.
	want := automations.AutomationTrigger{Kind: t.Kind, Reentry: "once"}
	if t.Kind == automations.TriggerJoinedList {
		want.ListID, want.ConfirmedOnly, want.ExitOnLeave = "selected_list", true, true
	} else {
		want.SegmentID = "selected_segment"
	}
	if sourceRevision(t) != sourceRevision(want) {
		return t, errors.New("unsupported trigger settings")
	}
	return want, nil
}

func (c *sourceContext) projectFlow(ctx context.Context) (MCPFlowSource, error) {
	g, err := automations.ParseGraph(c.Flow.Graph.MustJSON())
	out := MCPFlowSource{Graph: g, Emails: map[string]editorial.Source{}}
	if err != nil || g.Trigger() == nil || g.Trigger().Data.Trigger == nil {
		return out, errRequest
	}
	trigger, err := projectedTrigger(*g.Trigger().Data.Trigger)
	if err != nil {
		return out, err
	}
	// Keep names and identifiers local, including labels on otherwise safe nodes.
	c.Emails = map[string]sourceContext{}
	keys, nodes := map[string]string{}, map[string]string{}
	emails := []automations.ProposedEmail{}
	for i, mail := range c.Flow.Emails {
		key := fmt.Sprintf("email%d", i+1)
		local := sourceContext{EmailContext: editorial.EmailContext{BindingPrefix: c.BindingPrefix + key}, Before: &editorial.Email{Subject: mail.Subject, Preheader: mail.Preheader, MJML: mail.MJML}}
		source, err := local.project(ctx)
		if err != nil {
			return out, err
		}
		keys[mail.Key], c.Emails[key], out.Emails[key] = key, local, source.Source
		emails = append(emails, automations.ProposedEmail{Key: key, Subject: source.Subject})
	}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		nodes[n.ID] = fmt.Sprintf("step%d", i+1)
		n.Data.Label = ""
		if n.Type == automations.NodeTrigger {
			n.Data.Trigger = &trigger
		}
		if n.Type == automations.NodeEmail && n.Data.Email != nil {
			e := n.Data.Email
			key := keys[fmt.Sprintf("%s:%d", e.EmailID, e.EmailVersion)]
			e.EmailID, e.EmailVersion, e.Subject = key, 1, out.Emails[key].Subject
		}
	}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		n.ID = nodes[n.ID]
		if n.Data.Condition != nil {
			n.Data.Condition.Step = nodes[n.Data.Condition.Step]
		}
	}
	for i := range g.Edges {
		e := &g.Edges[i]
		e.ID, e.Source, e.Target = fmt.Sprintf("edge%d", i+1), nodes[e.Source], nodes[e.Target]
	}
	if err := automations.CheckDraftFlow(g, emails, trigger); err != nil {
		return out, err
	}
	out.Graph = g
	return out, nil
}
