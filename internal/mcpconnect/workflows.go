package mcpconnect

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// One proposal is one dependency bundle. No independently applicable child
// proposal can leave a flow referring to an unkept segment or email.
type MCPWorkflowInput struct {
	Key            string                      `json:"key" minLength:"1" maxLength:"80"`
	Name           string                      `json:"name" minLength:"1" maxLength:"120"`
	Source         string                      `json:"source" minLength:"1" maxLength:"100"`
	TemplateSource string                      `json:"templateSource,omitempty" maxLength:"100"`
	Change         bool                        `json:"change"`
	Trigger        string                      `json:"trigger,omitempty" enum:"joined_list,segment_entered,segment_left"`
	Segment        *segments.SegmentDefinition `json:"segment,omitempty"`
	SegmentName    string                      `json:"segmentName,omitempty" maxLength:"120"`
	Flow           *MCPProposedFlow            `json:"flow,omitempty"`
}

type MCPWorkflowContent struct {
	Segment     *segments.SegmentDefinition `json:"segment,omitempty"`
	SegmentName string                      `json:"segmentName,omitempty"`
	Flow        *MCPProposedFlow            `json:"flow,omitempty"`
	Previews    []string                    `json:"previews"`
}

// Use the same tagged contracts as the HTTP API. The SDK's inference cannot
// handle the recursive domain trigger type; Huma emits resolvable definitions.
func workflowSchema() map[string]any {
	r := huma.NewMapRegistry("#/$defs/", huma.DefaultSchemaNamer)
	schema := huma.SchemaFromType(r, reflect.TypeFor[MCPWorkflowInput]())
	raw, _ := json.Marshal(schema)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	out["$defs"] = r.Map()
	return out
}

func workflowShape(in any) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return errRequest
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return errRequest
	}
	r := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	schema := huma.SchemaFromType(r, reflect.TypeOf(in))
	result := &huma.ValidateResult{}
	huma.Validate(r, schema, huma.NewPathBuffer(nil, 0), huma.ModeWriteToServer, value, result)
	if len(result.Errors) != 0 {
		return errRequest
	}
	return nil
}

func (s *Service) workflowSelection(ctx context.Context, q *store.Queries, c *connectionCall, handle string) (store.McpConnectSource, sourceContext, error) {
	var local sourceContext
	row, err := q.MCPConnectSource(ctx, store.MCPConnectSourceParams{GrantID: c.Grant.ID, Handle: handle, Now: s.now()})
	if err != nil || checkSource(ctx, q, c.Principal, sourceRef(row), true) != nil {
		return row, local, errors.New("source_changed")
	}
	err = json.Unmarshal(row.Context, &local)
	return row, local, err
}

func (s *Service) proposeWorkflow(ctx context.Context, q *store.Queries, c *connectionCall, in MCPWorkflowInput) (MCPProposalReceipt, error) {
	var out MCPProposalReceipt
	kind := "segment"
	if in.Flow != nil {
		kind = "automation"
	}
	if workflowShape(in) != nil || strings.TrimSpace(in.Name) == "" || !safeWorkflowLabel(in.Name) || !safeWorkflowLabel(in.SegmentName) ||
		(in.Flow == nil && in.Segment == nil) || (in.Change && (in.Flow == nil || in.Segment != nil)) ||
		(in.Flow == nil && (in.Trigger != "" || in.TemplateSource != "")) ||
		(in.Segment != nil && in.Flow != nil && strings.TrimSpace(in.SegmentName) == "") {
		return out, errRequest
	}
	if kindAccess(c.Principal, kind, true) != nil || (in.Segment != nil && kindAccess(c.Principal, "segment", true) != nil) || !containsScope(c.Grant.Scopes, "editorial:read") {
		return out, errors.New("insufficient_scope")
	}
	raw, _ := json.Marshal(in)
	digest := sha256.Sum256(raw)
	prior, err := s.proposalRetry(ctx, q, c, in.Key, digest)
	if err != nil {
		return out, err
	}
	if prior != nil {
		return *prior, nil
	}
	row, source, err := s.workflowSelection(ctx, q, c, in.Source)
	if err != nil {
		return out, err
	}
	ref := sourceRef(row)
	ref.CreatesSegment = in.Segment != nil
	local := proposalContext{Source: &ref, Draft: source}
	local.Draft.Name = in.Name
	content := MCPWorkflowContent{SegmentName: in.SegmentName, Flow: in.Flow, Previews: []string{}}
	if in.Segment != nil {
		if row.Kind != "segment" {
			return out, errRequest
		}
		definition, err := segments.RestoreDefinition(*in.Segment, source.Literals)
		if err != nil {
			return out, errors.New("invalid_content")
		}
		if _, err := s.Segments.CheckProposal(ctx, q, c.Grant.WorkspaceID, c.Principal, definition); err != nil {
			return out, errors.New("invalid_content")
		}
		content.Segment = &definition
	}
	var target *uuid.UUID
	var revision int64
	if in.Flow != nil {
		var trigger automations.AutomationTrigger
		switch row.Kind {
		case "automation":
			if in.Trigger != "" || in.TemplateSource != "" || source.Flow == nil || source.Flow.Graph.Trigger() == nil {
				return out, errRequest
			}
			trigger = *source.Flow.Graph.Trigger().Data.Trigger
			if in.Change {
				_, _, editable, err := sourceRead(ctx, q, c.Grant.WorkspaceID, c.Principal, "automation", row.TargetID)
				if err != nil || !editable {
					return out, errors.New("source_changed")
				}
				target, revision = &row.TargetID, row.BaseRevision
				local.Draft.Name = source.Name
			}
		case "list":
			if in.Change || in.Trigger != automations.TriggerJoinedList {
				return out, errRequest
			}
			trigger = automations.AutomationTrigger{Kind: in.Trigger, ListID: row.TargetID.String(), ConfirmedOnly: true, ExitOnLeave: true, Reentry: "once"}
		case "segment":
			if in.Change || (in.Trigger != automations.TriggerSegmentEntered && in.Trigger != automations.TriggerSegmentLeft) {
				return out, errRequest
			}
			trigger = automations.AutomationTrigger{Kind: in.Trigger, SegmentID: row.TargetID.String(), Reentry: "once"}
		default:
			return out, errRequest
		}
		projected, err := projectedTrigger(trigger)
		if err != nil || automations.CheckDraftFlow(in.Flow.Graph, in.Flow.Emails, projected) != nil {
			return out, errors.New("invalid_content")
		}
		starter := sourceContext{}
		if in.TemplateSource != "" {
			template, context, err := s.workflowSelection(ctx, q, c, in.TemplateSource)
			if err != nil || template.Kind != "template" {
				return out, errors.New("source_changed")
			}
			local.Source.Related = append(local.Source.Related, sourceRef(template))
			starter = context
		} else {
			blank, _ := templates.StarterMJML("blank", "")
			if _, err := starter.Project(blank, nil, nil, 20_000); err != nil {
				return out, err
			}
		}
		local.Draft.Emails = map[string]sourceContext{}
		for i := range content.Flow.Emails {
			e := &content.Flow.Emails[i]
			if !safeWorkflowLabel(e.Name) {
				return out, errors.New("invalid_content")
			}
			context := starter
			if source.Flow != nil {
				var ok bool
				context, ok = source.Emails[e.Key]
				if !ok {
					return out, errors.New("invalid_content")
				}
			}
			email := editorial.Email{Subject: e.Subject, Preheader: e.Preheader, MJML: e.MJML}
			if err := context.Restore(&email, context.Before); err != nil {
				return out, errors.New("invalid_content")
			}
			e.Subject, e.Preheader, e.MJML = email.Subject, email.Preheader, email.MJML
			context.FinalPolicy = editorial.SavedPolicy(e.MJML)
			local.Draft.Emails[e.Key] = context
			for j := range content.Flow.Graph.Nodes {
				n := &content.Flow.Graph.Nodes[j]
				if n.Type == automations.NodeEmail && n.Data.Email != nil && n.Data.Email.EmailID == e.Key {
					n.Data.Email.Subject = e.Subject
				}
			}
		}
		if source.Flow != nil && len(local.Draft.Emails) != len(source.Emails) {
			return out, errors.New("invalid_content")
		}
		content.Flow.Graph.Trigger().Data.Trigger = &trigger
		if err := s.checkWorkflow(ctx, q, c.Grant.WorkspaceID, c.Principal, local, content); err != nil {
			return out, err
		}
		for _, email := range content.Flow.Emails {
			html, _, issues := mjml.Checked(email.MJML, "")
			if issues != nil {
				return out, errors.New("invalid_content")
			}
			content.Previews = append(content.Previews, html)
		}
	}
	contentJSON, _ := json.Marshal(content)
	bindings, _ := json.Marshal(local)
	return s.saveProposal(ctx, q, c, store.MCPConnectProposalInsertParams{RequestKey: in.Key, RequestHash: digest[:], Kind: kind,
		TargetID: target, BaseRevision: revision, Context: bindings, Content: contentJSON, Preview: ""})
}

func (s *Service) checkWorkflow(ctx context.Context, q *store.Queries, ws uuid.UUID, p *authz.Principal, local proposalContext, content MCPWorkflowContent) error {
	if content.Segment != nil {
		if kindAccess(p, "segment", true) != nil {
			return errors.New("insufficient_scope")
		}
		if _, err := s.Segments.CheckProposal(ctx, q, ws, p, *content.Segment); err != nil {
			return errors.New("invalid_content")
		}
	}
	if content.Flow == nil {
		return nil
	}
	f := content.Flow
	if f.Graph.Trigger() == nil || f.Graph.Trigger().Data.Trigger == nil || automations.CheckDraftFlow(f.Graph, f.Emails, *f.Graph.Trigger().Data.Trigger) != nil {
		return errors.New("invalid_content")
	}
	for _, n := range f.Graph.Nodes {
		if !safeWorkflowLabel(n.Data.Label) {
			return errors.New("invalid_content")
		}
	}
	if err := s.Automations.CheckProposalScope(ctx, q, ws, p, f.Graph); err != nil {
		return errors.New("insufficient_scope")
	}
	for _, e := range f.Emails {
		context, ok := local.Draft.Emails[e.Key]
		if !ok || checkSavedEmail(ctx, q, ws, context, "campaign", MCPProposedEmail{Subject: e.Subject, Preheader: e.Preheader, MJML: e.MJML}) != nil {
			return errors.New("invalid_content")
		}
	}
	return nil
}

func (s *Service) keepWorkflow(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, row store.McpConnectProposal, local proposalContext, origin audit.ProposalOrigin) (uuid.UUID, int64, error) {
	var content MCPWorkflowContent
	if err := json.Unmarshal(row.Content, &content); err != nil {
		return uuid.Nil, 0, err
	}
	if err := s.checkWorkflow(ctx, store.New(tx), ws.ID, p, local, content); err != nil {
		return uuid.Nil, 0, err
	}
	var segment uuid.UUID
	if content.Segment != nil {
		name := local.Draft.Name
		if content.Flow != nil {
			name = content.SegmentName
		}
		var err error
		segment, err = s.Segments.KeepProposalIn(ctx, tx, ws, p, name, *content.Segment, origin)
		if err != nil {
			return uuid.Nil, 0, err
		}
	}
	if content.Flow == nil {
		return segment, 1, nil
	}
	if segment != uuid.Nil {
		content.Flow.Graph.Trigger().Data.Trigger.SegmentID = segment.String()
	}
	return s.Automations.AcceptProposalIn(ctx, tx, ws, p, row.TargetID, row.BaseRevision, local.Draft.Name, content.Flow.Graph, content.Flow.Emails, origin)
}
