package automations

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// ProposedEmail is content for a new, flow-owned email. Key is a local graph
// reference, never an existing email ID. HTML is compiled, not caller-supplied.
type ProposedEmail struct {
	Key       string `json:"key" minLength:"1" maxLength:"40" pattern:"^[A-Za-z0-9_-]+$"`
	Name      string `json:"name" minLength:"1" maxLength:"80"`
	Subject   string `json:"subject" minLength:"1" maxLength:"250"`
	Preheader string `json:"preheader" maxLength:"500"`
	MJML      string `json:"mjml" minLength:"1" maxLength:"200000"`
}

// CheckDraftFlow restricts generated flows to email-only unpublished drafts.
// The caller supplies the human-selected trigger and checks privacy and email
// content separately. Reject extra settings rather than silently dropping them.
func CheckDraftFlow(g AutomationGraph, emails []ProposedEmail, trigger AutomationTrigger) error {
	if len(g.Nodes) > 30 || len(g.Edges) > 60 || len(emails) == 0 || len(emails) > 5 {
		return errors.New("flow size")
	}
	refs := &Refs{Lists: map[string]string{trigger.ListID: ""}, Segments: map[string]string{trigger.SegmentID: ""}, Emails: map[string]int{}}
	byKey, used := map[string]ProposedEmail{}, map[string]bool{}
	for _, e := range emails {
		if _, exists := byKey[e.Key]; exists {
			return errors.New("duplicate email")
		}
		byKey[e.Key], refs.Emails[e.Key] = e, 1
	}
	for _, n := range g.Nodes {
		allowed := AutomationNodeData{Label: n.Data.Label}
		switch n.Type {
		case NodeTrigger:
			allowed.Trigger = &trigger
		case NodeEmail:
			e := n.Data.Email
			if e == nil || e.EmailID == "" {
				return errors.New("missing email")
			}
			mail, ok := byKey[e.EmailID]
			if !ok || e.Subject != mail.Subject {
				return errors.New("unknown email")
			}
			allowed.Email = &AutomationEmailStep{EmailID: e.EmailID, EmailVersion: 1, Subject: mail.Subject}
			used[e.EmailID] = true
		case NodeDelay:
			allowed.Delay = n.Data.Delay
		case NodeBranch:
			c := n.Data.Condition
			if c == nil || (c.Kind != "opened" && c.Kind != "clicked" && c.Kind != "replied") || c.Within == nil || c.Within.Amount <= 0 || c.Within.Duration() > 366*24*time.Hour {
				return errors.New("unsupported condition")
			}
			step := g.Node(c.Step)
			if step == nil || step.Type != NodeEmail || !flowRequiresEmail(g, step.ID, n.ID) {
				return errors.New("condition must reference a preceding email")
			}
			allowed.Condition = &AutomationCondition{Kind: c.Kind, Step: c.Step, Within: c.Within, HumanOnly: true}
		case NodeSplit:
			for _, variant := range n.Data.Variants {
				if len(variant.Key) != 1 || variant.Key < "A" || variant.Key > "Z" {
					return errors.New("unsupported variant key")
				}
			}
			allowed.Variants = n.Data.Variants
		case NodeExit:
		default:
			return errors.New("unsupported step")
		}
		if !reflect.DeepEqual(n.Data, allowed) {
			return errors.New("unsupported step settings")
		}
		if n.Type != NodeExit && len(g.Out(n.ID)) == 0 {
			return errors.New("unfinished path")
		}
	}
	if len(used) != len(emails) {
		return errors.New("unused email")
	}
	for _, edge := range g.Edges {
		from := g.Node(edge.Source)
		if from == nil || ((from.Type != NodeSplit && from.Type != NodeBranch) && edge.Data.Path != "") {
			return errors.New("unsupported path")
		}
	}
	for _, issue := range Validate(g, refs) {
		if issue.Severity == "error" {
			return errors.New("invalid graph")
		}
	}
	return nil
}

// Every path to an engagement branch must pass its referenced email.
func flowRequiresEmail(g AutomationGraph, email, branch string) bool {
	start := g.Trigger()
	if start == nil {
		return false
	}
	seen := map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if id == branch {
			return true
		}
		if id == email || seen[id] {
			return false
		}
		seen[id] = true
		for _, edge := range g.Out(id) {
			if visit(edge.Target) {
				return true
			}
		}
		return false
	}
	return !visit(start.ID)
}

// CheckProposalScope applies the flow's normal audience and sender boundary
// before its content can become proposal context, including after a revoke.
func (s *Service) CheckProposalScope(ctx context.Context, q *store.Queries, wsID uuid.UUID, p *authz.Principal, g AutomationGraph) error {
	problem, err := scopeProblem(ctx, q, wsID, p, g, langOf(ctx))
	if err != nil {
		return err
	}
	if problem != "" {
		return api.Forbidden(authz.AutomationsView, nil)
	}
	for _, n := range g.Nodes {
		for _, id := range []string{listOfCondition(n.Data.Condition), listOfAction(n.Data.Action)} {
			if id == "" {
				continue
			}
			list, err := uuid.Parse(id)
			if err != nil || !p.Scope.AllowsLists([]uuid.UUID{list}) {
				return api.Forbidden(authz.AutomationsView, nil)
			}
		}
	}
	return nil
}

func listOfCondition(c *AutomationCondition) string {
	if c == nil {
		return ""
	}
	return c.ListID
}

func listOfAction(a *AutomationListAction) string {
	if a == nil {
		return ""
	}
	return a.ListID
}

// AcceptProposalIn commits a human-accepted graph and its new emails in the
// proposal decision's transaction. It cannot publish, enroll, replace an
// existing email, or change a live version. The caller refreshes membership,
// checks exact proposal ownership and validates the generated content policy.
func (s *Service) AcceptProposalIn(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, target *uuid.UUID, base int64, name string, g AutomationGraph, emails []ProposedEmail, origin audit.ProposalOrigin) (uuid.UUID, int64, error) {
	if !p.IsHuman() || p.WorkspaceID != ws.ID || !p.Can(authz.AutomationsEdit) {
		return uuid.Nil, 0, api.Forbidden(authz.AutomationsEdit, nil)
	}
	if strings.TrimSpace(name) == "" || len(g.Nodes) > maxNodes || len(g.Edges) > maxEdges || len(emails) == 0 || len(emails) > 10 {
		return uuid.Nil, 0, api.Invalid("body", say(langOf(ctx), txt(mProblemGraphSize, "steps", maxNodes, "connections", maxEdges)))
	}
	q := store.New(tx)
	id, version, revision := uuid.Must(uuid.NewV7()), int32(1), int64(1)
	if target != nil {
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: *target})
		if err != nil {
			return uuid.Nil, 0, api.MapError(err, "Automation")
		}
		if a.Status != StatusDraft || a.LiveVersion != nil || a.DraftRevision != base {
			return uuid.Nil, 0, api.Conflict(say(langOf(ctx), txt(mProblemRevision)))
		}
		id, version, revision = a.ID, a.DraftVersion, a.DraftRevision+1
	}
	refs, err := s.loadRefs(ctx, q, ws.ID)
	if err != nil {
		return uuid.Nil, 0, err
	}
	// Only the proposal's email keys may be referenced, not a guessed ID of
	// an email in another flow or workspace. Repeated references share one
	// newly created email; unused content is refused rather than orphaned.
	refs.Emails = map[string]int{}
	keys := map[string]uuid.UUID{}
	compiled := make(map[string]EmailContent, len(emails))
	for _, e := range emails {
		if _, exists := keys[e.Key]; exists || e.Key == "" || strings.TrimSpace(e.Name) == "" || strings.TrimSpace(e.Subject) == "" || strings.ContainsAny(e.Subject+e.Preheader, "\r\n\x00<>") {
			return uuid.Nil, 0, api.Invalid("body.emails", say(langOf(ctx), txt(mProblemEmailEmpty)))
		}
		html, text, issues := mjml.Checked(e.MJML, "")
		if issues != nil {
			return uuid.Nil, 0, api.Invalid("body.emails", say(langOf(ctx), txt(mProblemEmailEmpty)))
		}
		keys[e.Key], refs.Emails[e.Key] = uuid.Must(uuid.NewV7()), 1
		compiled[e.Key] = EmailContent{Subject: e.Subject, Preheader: e.Preheader, MJML: e.MJML, HTML: html, Text: text}
	}
	for _, issue := range validate(langOf(ctx), g, refs) {
		if issue.Severity == "error" {
			return uuid.Nil, 0, api.Invalid("body.graph", issue.Message)
		}
	}
	if problem, err := scopeProblem(ctx, q, ws.ID, p, g, langOf(ctx)); err != nil {
		return uuid.Nil, 0, err
	} else if problem != "" {
		return uuid.Nil, 0, api.Forbidden(authz.AutomationsEdit, nil)
	}
	used := map[string]bool{}
	for i := range g.Nodes {
		if e := g.Nodes[i].Data.Email; g.Nodes[i].Type == NodeEmail && e != nil {
			newID, ok := keys[e.EmailID]
			if !ok || e.EmailVersion != 1 || e.Subject != compiled[e.EmailID].Subject {
				return uuid.Nil, 0, api.Invalid("body.graph", say(langOf(ctx), txt(mProblemPickEmail)))
			}
			used[e.EmailID] = true
			e.EmailID = newID.String()
		}
	}
	if len(used) != len(keys) {
		return uuid.Nil, 0, api.Invalid("body.emails", say(langOf(ctx), txt(mProblemPickEmail)))
	}
	if target == nil {
		if _, err := q.CreateAutomation(ctx, store.CreateAutomationParams{ID: id, WorkspaceID: ws.ID, Name: name, Trigger: triggerOf(g), CreatedBy: &p.UserID}); err != nil {
			return uuid.Nil, 0, err
		}
	}
	meta := origin.Metadata()
	for _, e := range emails {
		eid, c := keys[e.Key], compiled[e.Key]
		if err := q.CreateAutomationEmail(ctx, store.CreateAutomationEmailParams{ID: eid, WorkspaceID: ws.ID, AutomationID: &id, Name: e.Name, CreatedBy: &p.UserID}); err != nil {
			return uuid.Nil, 0, err
		}
		if err := q.InsertAutomationEmailVersion(ctx, store.InsertAutomationEmailVersionParams{EmailID: eid, Version: 1, Subject: c.Subject, Preheader: c.Preheader, Mjml: c.MJML, Html: accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerAutomationEmail, eid, 1, c.HTML), PlainText: c.Text, CreatedBy: &p.UserID}); err != nil {
			return uuid.Nil, 0, err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "automation_email.create", TargetType: "automation_email", TargetID: eid.String(), Metadata: meta}); err != nil {
			return uuid.Nil, 0, err
		}
	}
	action := "automation.create"
	if target == nil {
		if err := q.InsertAutomationVersion(ctx, store.InsertAutomationVersionParams{AutomationID: id, Version: version, Graph: g.MustJSON(), CreatedBy: &p.UserID}); err != nil {
			return uuid.Nil, 0, err
		}
	} else {
		action = "automation.draft_update"
		rows, err := q.UpdateAutomationDraftGraph(ctx, store.UpdateAutomationDraftGraphParams{AutomationID: id, Version: version, Graph: g.MustJSON()})
		if err != nil {
			return uuid.Nil, 0, err
		}
		if rows != 1 {
			return uuid.Nil, 0, api.Conflict(say(langOf(ctx), txt(mProblemRevision)))
		}
		if err := q.SetAutomationDraft(ctx, store.SetAutomationDraftParams{ID: id, DraftVersion: version, UpdatedBy: &p.UserID, Trigger: triggerOf(g)}); err != nil {
			return uuid.Nil, 0, err
		}
	}
	meta["version"], meta["revision"] = version, revision
	err = s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "automation", TargetID: id.String(), Metadata: meta})
	return id, revision, err
}
