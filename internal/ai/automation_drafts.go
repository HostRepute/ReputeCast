package ai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// AIAutomationBriefInput keeps enrollment semantics outside model output.
// This one-shot recipe supports consented, future list joins only. More
// complex audience conditions require the full agent's reviewed plan.
type AIAutomationBriefInput struct {
	SharingReviewed bool       `json:"sharingReviewed" doc:"Confirm review of the brief, selected content, residual personal-data risk and provider data policy before sharing"`
	Brief           string     `json:"brief" minLength:"1" maxLength:"4000"`
	Name            string     `json:"name" minLength:"1" maxLength:"200"`
	Language        string     `json:"language" enum:"en,vi,de,es,fr"`
	ListID          uuid.UUID  `json:"listId"`
	AutomationID    *uuid.UUID `json:"automationId,omitempty"`
	BaseRevision    int64      `json:"baseRevision,omitempty" minimum:"0"`
	TemplateID      string     `json:"templateId,omitempty" maxLength:"100"`
	Links           []string   `json:"links,omitempty" maxItems:"10"`
	VoiceRevision   *int32     `json:"voiceRevision,omitempty" minimum:"0"`
}

type AIAutomationContent struct {
	Graph  automations.AutomationGraph `json:"graph"`
	Emails []automations.ProposedEmail `json:"emails" minItems:"1" maxItems:"5"`
}

type AIAutomationProposal struct {
	ID           uuid.UUID                    `json:"id"`
	Name         string                       `json:"name"`
	AutomationID *uuid.UUID                   `json:"automationId,omitempty"`
	BaseRevision int64                        `json:"baseRevision"`
	ListID       uuid.UUID                    `json:"listId"`
	Before       *automations.AutomationGraph `json:"before,omitempty"`
	BeforeEmails []AIAutomationEmailBefore    `json:"beforeEmails"`
	Content      AIAutomationContent          `json:"content"`
	Previews     []string                     `json:"previews" doc:"Compiled email previews, in content.emails order; no personalization rendering"`
	Sent         AISent                       `json:"sent"`
	ExpiresAt    time.Time                    `json:"expiresAt"`
	Decision     string                       `json:"decision" enum:",accepted,rejected"`
}

type AIAutomationEmailBefore struct {
	Key     string       `json:"key"`
	ID      uuid.UUID    `json:"id"`
	Version int32        `json:"version"`
	Name    string       `json:"name"`
	Content AIDraftEmail `json:"content"`
	HTML    string       `json:"html"`
}

type automationDraftContext struct {
	Name     string
	ListID   uuid.UUID
	Before   *automations.AutomationGraph
	Sources  []AIAutomationEmailBefore
	Starter  draftContext
	Sent     AISent
	Policies map[string]mjml.GeneratedPolicy
}

type automationDraftTask struct {
	Brief, Language string
	Graph           *automations.AutomationGraph
	Emails          map[string]campaignDraftTask
	Starter         campaignDraftTask
}

func automationDraftPrompt(t automationDraftTask) llm.Request {
	data, _ := json.Marshal(t)
	req := userRequest(systemPrompt, textPart(
		"Draft an unpublished list-join email sequence from this brief. The person has chosen future confirmed list joins, once per reader, excluding imports, with exit on leaving the list. If the brief requires different audience semantics or unsupported steps, return a refusal instead of changing or omitting the requirement.",
		`Return graph and emails. Graph has exactly one trigger with data.trigger {"kind":"joined_list","listId":"selected_list","confirmedOnly":true,"includeImports":false,"reentry":"once","exitOnLeave":true}. Other node types allowed: email, delay, branch, split, exit. Branches may test opened, clicked or replied for an earlier email step only, with a positive time window. No fields, URLs, contacts, lists, webhooks, other channels or routes. Every email node references an email's local key with emailVersion 1 and its exact subject. Every path ends at an exit. At most 30 nodes, 60 edges and five new emails.`,
		`Graph: {"nodes":[{"id":"step1","type":"trigger","position":{"x":0,"y":0},"data":{"trigger":{...}}}],"edges":[{"id":"edge1","source":"step1","target":"step2","data":{}}]}. Node data contains only its type's settings: email {"emailId":"email1","emailVersion":1,"subject":"Exact subject"}; delay {"amount":2,"unit":"days"}; condition {"kind":"clicked","step":"precedingEmailStep","within":{"amount":3,"unit":"days"},"humanOnly":true}; variants [{"key":"A","weight":70},{"key":"B","weight":30}]; exit {}. Branch edges have data.path yes or no; split edges have the variant key. Delay/window units: minutes, hours, days, weeks. Email array items: {"key":"email1","name":"Welcome","subject":"Exact subject","preheader":"Preview text","mjml":"<mjml><mj-body>...</mj-body></mjml>"}.`,
		"Emails have key, name, subject, preheader and mjml. Revise the supplied graph and emails when present, retaining every source email key. New emails use Starter. Each email preserves all its source's opaque Links, Images and Tokens: links in href, images in src, text tokens in their original subject, preheader or body. Never copy bindings between emails or invent URLs or Liquid. Use ordinary mj-section, mj-column, mj-text, mj-button, mj-image, mj-divider and mj-spacer; no comments, raw blocks, includes, scripts, stylesheets or footer. The server restores the local header and footer. Preserve facts; do not invent offers, dates or recipients. Position nodes for a top-down editor.",
		tag("brief", string(data)),
	))
	req.SchemaName = "automation_draft"
	req.Schema = map[string]any{"type": "object", "properties": map[string]any{"graph": map[string]any{"type": "object"}, "emails": map[string]any{"type": "array", "items": map[string]any{"type": "object"}}, "refusal": map[string]any{"type": "string"}}}
	req.MaxTokens, req.Task = 8000, t
	return req
}

// Only fields assigned here survive validation. Reject unused fields instead
// of silently stripping a condition or subscription action. Matching the
// trigger exactly prevents a model from broadening audience semantics.
func checkAutomationDraft(content AIAutomationContent, list string) error {
	return checkDraftFlow(content, automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: list, ConfirmedOnly: true, Reentry: "once", ExitOnLeave: true})
}

func checkDraftFlow(content AIAutomationContent, trigger automations.AutomationTrigger) error {
	for _, n := range content.Graph.Nodes {
		if _, safe := voiceText(n.Data.Label); !safe || voiceURL.MatchString(n.Data.Label) || voiceLiquid.MatchString(n.Data.Label) || draftReserved.MatchString(n.Data.Label) {
			return errors.New("unsafe label")
		}
	}
	return automations.CheckDraftFlow(content.Graph, content.Emails, trigger)
}

func (s *Service) automationDraftAccess(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, list uuid.UUID, target *uuid.UUID) (*store.Automation, *automations.AutomationGraph, error) {
	if _, err := q.AIDraftList(ctx, store.AIDraftListParams{ID: list, WorkspaceID: ws.ID, ScopeLists: access.ScopeLists(p)}); err != nil {
		return nil, nil, api.MapError(err, "List")
	}
	if target == nil {
		return nil, nil, nil
	}
	if !p.Can(authz.AutomationsView) {
		return nil, nil, api.Forbidden(authz.AutomationsView, nil)
	}
	a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: *target})
	if err != nil {
		return nil, nil, api.MapError(err, "Automation")
	}
	v, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: a.ID, Version: a.DraftVersion})
	if err != nil {
		return nil, nil, err
	}
	g, err := automations.ParseGraph(v.Graph)
	if err != nil {
		return nil, nil, err
	}
	if err := s.Automations.CheckProposalScope(ctx, q, ws.ID, p, g); err != nil {
		return nil, nil, err
	}
	return &a, &g, nil
}

func automationSource(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, id uuid.UUID, version int32) (store.AIDraftAutomationEmailRow, error) {
	domains := make([]string, 0, len(p.Scope.SenderDomains))
	for _, domain := range p.Scope.SenderDomains {
		domains = append(domains, strings.ToLower(domain))
	}
	e, err := q.AIDraftAutomationEmail(ctx, store.AIDraftAutomationEmailParams{WorkspaceID: ws.ID, ID: id, Version: version, ScopeDomains: domains})
	return e, api.MapError(err, "Email")
}

// Use a validated graph, with fresh handles for nodes and references. Never
// serialize a stored graph directly: IDs, labels and unused node settings
// can contain contact data even when the graph has no contact condition.
func (c *automationDraftContext) projectFlow(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, before automations.AutomationGraph, t *automationDraftTask, contexts map[string]draftContext, links []string, trigger automations.AutomationTrigger, prefix string) error {
	g, err := automations.ParseGraph(before.MustJSON())
	if err != nil || len(g.Nodes) > 30 || len(g.Edges) > 60 {
		return draftProblem(ctx, 422, "source")
	}
	nodes := map[string]string{}
	content := AIAutomationContent{}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		nodes[n.ID] = fmt.Sprintf("step%d", i+1)
		if n.Type == automations.NodeTrigger && n.Data.Trigger != nil {
			if n.Data.Trigger.ListID != "" {
				n.Data.Trigger.ListID = "selected_list"
			}
			if n.Data.Trigger.SegmentID != "" {
				n.Data.Trigger.SegmentID = "selected_segment"
			}
		}
		if n.Type != automations.NodeEmail || n.Data.Email == nil {
			continue
		}
		e := n.Data.Email
		id, err := uuid.Parse(e.EmailID)
		if err != nil || e.EmailVersion <= 0 || e.EmailVersion > 2147483647 {
			return draftProblem(ctx, 422, "source")
		}
		key := ""
		for _, source := range c.Sources {
			if source.ID == id && source.Version == int32(e.EmailVersion) {
				key = source.Key
				break
			}
		}
		if key == "" {
			if len(c.Sources) >= 5 {
				return draftProblem(ctx, 422, "source")
			}
			row, err := automationSource(ctx, q, ws, p, id, int32(e.EmailVersion))
			if err != nil {
				return err
			}
			key = fmt.Sprintf("source%d", len(c.Sources)+1)
			mail := AIDraftEmail{Subject: row.Subject, Preheader: row.Preheader, MJML: row.Mjml}
			local := draftContext{Before: &mail, EmailContext: editorial.EmailContext{BindingPrefix: prefix + key}}
			projected, err := local.projectSource(ctx, row.Mjml, links)
			if err != nil {
				return err
			}
			c.Sources = append(c.Sources, AIAutomationEmailBefore{Key: key, ID: id, Version: row.Version, Name: row.Name, Content: mail})
			t.Emails[key], contexts[key] = projected, local
			content.Emails = append(content.Emails, automations.ProposedEmail{Key: key, Subject: projected.Subject})
		}
		e.EmailID, e.EmailVersion, e.Subject = key, 1, t.Emails[key].Subject
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
	content.Graph = g
	if checkDraftFlow(content, trigger) != nil {
		return draftProblem(ctx, 422, "source")
	}
	t.Graph = &g
	return nil
}

func (c automationDraftContext) checkSources(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal) error {
	if err := c.Starter.checkTemplate(ctx, q, ws, p, true); err != nil {
		return err
	}
	for _, source := range c.Sources {
		row, err := automationSource(ctx, q, ws, p, source.ID, source.Version)
		if err != nil {
			return err
		}
		if row.Subject != source.Content.Subject || row.Preheader != source.Content.Preheader || row.Mjml != source.Content.MJML {
			return draftProblem(ctx, 409, "changed")
		}
	}
	return nil
}

func (s *Service) GenerateAutomation(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAutomationBriefInput) (AIAutomationProposal, error) {
	var out AIAutomationProposal
	if !in.SharingReviewed {
		return out, draftProblem(ctx, 422, "sharing_review")
	}
	if err := validateDraftJSON(in); err != nil || (in.AutomationID == nil) != (in.BaseRevision == 0) || (in.AutomationID != nil && in.TemplateID != "") || strings.TrimSpace(in.Name) == "" || in.ListID == uuid.Nil || voiceURL.MatchString(in.Brief) || voiceLiquid.MatchString(in.Brief) || draftReserved.MatchString(in.Brief) {
		return out, draftProblem(ctx, 422, "input")
	}
	brief, safe := voiceText(in.Brief)
	if !safe || strings.TrimSpace(brief) == "" {
		return out, draftProblem(ctx, 422, "privacy")
	}
	c := automationDraftContext{Name: strings.TrimSpace(in.Name), ListID: in.ListID, Policies: map[string]mjml.GeneratedPolicy{}}
	t := automationDraftTask{Brief: brief, Language: languageName(in.Language), Emails: map[string]campaignDraftTask{}}
	emailContexts := map[string]draftContext{}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, true, authz.AutomationsEdit)
		if err != nil {
			return err
		}
		a, before, err := s.automationDraftAccess(ctx, q, ws, fresh, c.ListID, in.AutomationID)
		if err != nil {
			return err
		}
		if a != nil && (a.Status != automations.StatusDraft || a.LiveVersion != nil || a.DraftRevision != in.BaseRevision) {
			return draftProblem(ctx, 409, "changed")
		}
		if before != nil {
			want := &automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: c.ListID.String(), ConfirmedOnly: true, Reentry: "once", ExitOnLeave: true}
			if before.Trigger() == nil || !reflect.DeepEqual(before.Trigger().Data.Trigger, want) {
				return draftProblem(ctx, 422, "source")
			}
			c.Name = a.Name
			projectedTrigger := *want
			projectedTrigger.ListID = "selected_list"
			if err := c.projectFlow(ctx, q, ws, fresh, *before, &t, emailContexts, in.Links, projectedTrigger, ""); err != nil {
				return err
			}
		}
		c.Before = before
		return nil
	})
	if err != nil {
		return out, err
	}
	source, _ := templates.StarterMJML("blank", "")
	if in.TemplateID != "" {
		if !p.Can(authz.TemplatesView) {
			return out, api.Forbidden(authz.TemplatesView, nil)
		}
		tpl, err := s.Campaigns.Templates.Get(ctx, ws, in.TemplateID, "")
		if err != nil {
			return out, err
		}
		source = tpl.MJML
	}
	c.Starter.TemplateID = in.TemplateID
	c.Starter.BindingPrefix = "starter"
	c.Starter.TemplateHash = sha256.Sum256([]byte(source))
	t.Starter, err = c.Starter.projectSource(ctx, source, in.Links)
	if err != nil {
		return out, err
	}
	contextBytes, _ := json.Marshal(t)
	if len(contextBytes) > 80_000 {
		return out, draftProblem(ctx, 422, "source")
	}
	checkSource := func(ctx context.Context, q *store.Queries, fresh *authz.Principal) error {
		if !fresh.Can(authz.AutomationsEdit) {
			return api.Forbidden(authz.AutomationsEdit, nil)
		}
		a, _, err := s.automationDraftAccess(ctx, q, ws, fresh, c.ListID, in.AutomationID)
		if err != nil {
			return err
		}
		if a != nil && (a.Status != automations.StatusDraft || a.LiveVersion != nil || a.DraftRevision != in.BaseRevision) {
			return draftProblem(ctx, 409, "changed")
		}
		return c.checkSources(ctx, q, ws, fresh)
	}
	var content AIAutomationContent
	r, err := s.runDraft(ctx, ws, p, task{feature: FeatureDraft, targetKind: "automation", targetID: in.AutomationID, req: automationDraftPrompt(t), selectedVoice: in.VoiceRevision, checkSource: checkSource}, func(r llm.Response) (string, error) {
		content = AIAutomationContent{}
		c.Policies = map[string]mjml.GeneratedPolicy{}
		if decodeDraft([]byte(r.Text), &content) != nil {
			return "schema", nil //nolint:nilerr // A safe validation code requests bounded repair; an error aborts generation.
		}
		if checkAutomationDraft(content, "selected_list") != nil {
			return "graph", nil //nolint:nilerr // Repair invalid output without sending the validator's raw error.
		}
		for key := range emailContexts {
			if !slices.ContainsFunc(content.Emails, func(e automations.ProposedEmail) bool { return e.Key == key }) {
				return "source_emails", nil
			}
		}
		for i := range content.Emails {
			e := &content.Emails[i]
			emailContext, existing := emailContexts[e.Key]
			if !existing {
				emailContext = c.Starter
			}
			mail := AIDraftEmail{Subject: e.Subject, Preheader: e.Preheader, MJML: e.MJML}
			if _, safe := voiceText(e.Name); !safe || emailContext.restoreEmail(&mail) != nil {
				return "email_bindings", nil //nolint:nilerr // Bound values stay local; the model receives only a repair code.
			}
			if !existing {
				mail.MJML = s.BrandKit.Starter(ctx, ws, mail.MJML)
			}
			policy := savedDraftPolicy(mail.MJML)
			c.Policies[e.Key] = policy
			if err := (draftContext{EmailContext: editorial.EmailContext{FinalPolicy: policy}}).validateSaved(ctx, s.Q, ws, mail); err != nil {
				if errors.Is(err, errDraftContent) {
					return "email_content", nil
				}
				return "", err
			}
			if _, _, issues := mjml.Checked(mail.MJML, ""); issues != nil {
				return "mjml", nil
			}
			e.Subject, e.Preheader, e.MJML = mail.Subject, mail.Preheader, mail.MJML
			for j := range content.Graph.Nodes {
				if node := content.Graph.Nodes[j].Data.Email; node != nil && node.EmailID == e.Key {
					node.Subject = mail.Subject
				}
			}
		}
		return "", nil
	})
	if err != nil {
		return out, err
	}
	content.Graph.Trigger().Data.Trigger.ListID = c.ListID.String()
	content.Graph = automations.Layout(content.Graph)
	projected, _ := json.Marshal(t)
	c.Sent = r.sent(chars(string(projected)))
	contextJSON, _ := json.Marshal(c)
	contentJSON, _ := json.Marshal(content)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, true, authz.AutomationsEdit)
		if err != nil {
			return err
		}
		if err := checkSource(ctx, q, fresh); err != nil {
			return err
		}
		conn, err := s.current(ctx, q, ws.ID, r.conn)
		if err != nil {
			return err
		}
		if !conn.connected() || !conn.featureOn(FeatureDraft) {
			return draftProblem(ctx, 409, "unavailable")
		}
		if err := q.AIDraftInsert(ctx, store.AIDraftInsertParams{ID: r.RequestID, WorkspaceID: ws.ID, UserID: p.UserID, Kind: "automation", TargetID: in.AutomationID, BaseRevision: in.BaseRevision, Context: contextJSON, Content: contentJSON, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.draft.propose", TargetType: "ai_request", TargetID: r.RequestID.String(), Metadata: map[string]any{"kind": "automation", "base_revision": in.BaseRevision, "sharing_policy": editorial.SharingPolicy}})
	})
	if err != nil {
		s.finish(ctx, ws, p, r.RequestID, OutcomeDiscarded, "access changed")
		return out, err
	}
	return s.AutomationProposal(ctx, ws, p, r.RequestID)
}

func (s *Service) AutomationProposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AIAutomationProposal, error) {
	out, err := s.automationProposal(ctx, ws, p, id)
	if err != nil {
		return out, err
	}
	out.Previews = make([]string, 0, len(out.Content.Emails))
	for _, email := range out.Content.Emails {
		out.Previews = append(out.Previews, mjml.Compile(email.MJML).HTML)
	}
	for i := range out.BeforeEmails {
		out.BeforeEmails[i].HTML = mjml.Compile(out.BeforeEmails[i].Content.MJML).HTML
	}
	return out, nil
}

func (s *Service) automationProposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AIAutomationProposal, error) {
	var out AIAutomationProposal
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, false, authz.AutomationsView)
		if err != nil {
			return err
		}
		r, err := q.AIDraftGet(ctx, store.AIDraftGetParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, Now: s.now()})
		if err != nil {
			return api.MapError(err, "Proposal")
		}
		if r.Kind != "automation" {
			return api.NotFound("Proposal")
		}
		var c automationDraftContext
		if err := json.Unmarshal(r.Context, &c); err != nil {
			return err
		}
		target := r.TargetID
		if target == nil {
			target = r.ResultID
		}
		if _, _, err := s.automationDraftAccess(ctx, q, ws, fresh, c.ListID, target); err != nil {
			return err
		}
		if c.Before != nil {
			if err := s.Automations.CheckProposalScope(ctx, q, ws.ID, fresh, *c.Before); err != nil {
				return err
			}
		}
		if err := c.checkSources(ctx, q, ws, fresh); err != nil {
			return err
		}
		if err := json.Unmarshal(r.Content, &out.Content); err != nil {
			return err
		}
		out.ID, out.Name, out.AutomationID, out.BaseRevision, out.Before = r.ID, c.Name, r.TargetID, r.BaseRevision, c.Before
		out.ListID, out.Sent, out.ExpiresAt, out.Decision = c.ListID, c.Sent, r.ExpiresAt, r.Decision
		out.BeforeEmails = append([]AIAutomationEmailBefore{}, c.Sources...)
		return nil
	})
	return out, err
}

func (s *Service) DecideAutomation(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIDraftDecision) (AIDraftDecided, error) {
	var out AIDraftDecided
	if in.Decision != "accepted" && in.Decision != "rejected" {
		return out, draftProblem(ctx, 422, "input")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, false, authz.AutomationsEdit)
		if err != nil {
			return err
		}
		r, err := q.AIDraftLock(ctx, store.AIDraftLockParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, Now: s.now()})
		if err != nil {
			return api.MapError(err, "Proposal")
		}
		if r.Kind != "automation" {
			return api.NotFound("Proposal")
		}
		var c automationDraftContext
		if err := json.Unmarshal(r.Context, &c); err != nil {
			return err
		}
		target := r.TargetID
		if target == nil {
			target = r.ResultID
		}
		if _, _, err := s.automationDraftAccess(ctx, q, ws, fresh, c.ListID, target); err != nil {
			return err
		}
		if c.Before != nil {
			if err := s.Automations.CheckProposalScope(ctx, q, ws.ID, fresh, *c.Before); err != nil {
				return err
			}
		}
		if r.Decision != "" {
			if r.Decision != in.Decision {
				return draftProblem(ctx, 409, "decided")
			}
			out = AIDraftDecided{Decision: r.Decision, ID: r.ResultID, Revision: r.ResultRevision}
			return nil
		}
		out.Decision = in.Decision
		if in.Decision == "accepted" {
			if err := c.checkSources(ctx, q, ws, fresh); err != nil {
				return err
			}
			var content AIAutomationContent
			if err := decodeDraft(r.Content, &content); err != nil || checkAutomationDraft(content, c.ListID.String()) != nil {
				return draftProblem(ctx, 422, "invalid")
			}
			for _, e := range content.Emails {
				if err := (draftContext{EmailContext: editorial.EmailContext{FinalPolicy: c.Policies[e.Key]}}).validateSaved(ctx, q, ws, AIDraftEmail{Subject: e.Subject, Preheader: e.Preheader, MJML: e.MJML}); err != nil {
					return draftProblem(ctx, 422, "invalid")
				}
			}
			result, revision, err := s.Automations.AcceptProposalIn(ctx, tx, ws, fresh, r.TargetID, r.BaseRevision, c.Name, content.Graph, content.Emails, audit.ProposalOrigin{ProposalID: id})
			if err != nil {
				return err
			}
			out.ID, out.Revision = &result, &revision
		} else if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.draft.reject", TargetType: "ai_request", TargetID: id.String()}); err != nil {
			return err
		}
		if err := q.AIDraftDecide(ctx, store.AIDraftDecideParams{ID: id, Decision: in.Decision, ResultID: out.ID, ResultRevision: out.Revision}); err != nil {
			return err
		}
		_, err = q.AIRequestSetOutcome(ctx, store.AIRequestSetOutcomeParams{ID: id, WorkspaceID: ws.ID, UserID: &p.UserID, Outcome: in.Decision, DecidedAt: ptr(s.now())})
		return err
	})
	return out, err
}

func registerAutomationDrafts(h huma.API, s *Service) {
	base := "/workspaces/{workspace}/ai/drafts/automations"
	tags := []string{"AI assistant"}
	api.Get(h, api.Op{ID: "list-automation-draft-proposals", Path: base, Tags: tags, Summary: "Recover your pending proposals for this draft or new automations", Access: api.WorkspaceHuman(authz.AutomationsView)},
		func(ctx context.Context, in *pendingDraftInput) (*api.Body[AIPendingDrafts], error) {
			out, err := s.pendingDrafts(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), "automation", in)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "generate-automation-draft", Path: base, Tags: tags, Summary: "Propose an unpublished email flow for future confirmed list joins", Access: api.WorkspaceHuman(authz.AIUse), WriteTimeout: draftWriteTimeout},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body AIAutomationBriefInput
		}) (*api.Body[AIAutomationProposal], error) {
			out, err := s.GenerateAutomation(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-automation-draft-proposal", Path: base + "/{proposal}", Tags: tags, Summary: "Read your unexpired automation proposal", Access: api.WorkspaceHuman(authz.AutomationsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Proposal uuid.UUID `path:"proposal"`
		}) (*api.Body[AIAutomationProposal], error) {
			out, err := s.AutomationProposal(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Proposal)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "decide-automation-draft-proposal", Path: base + "/{proposal}/decision", Tags: tags, Summary: "Accept or reject an automation proposal without publishing or enrolling", Access: api.WorkspaceHuman(authz.AutomationsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Proposal uuid.UUID `path:"proposal"`
			Body     AIDraftDecision
		}) (*api.Body[AIDraftDecided], error) {
			out, err := s.DecideAutomation(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Proposal, in.Body)
			return api.OK(out), err
		})
}
