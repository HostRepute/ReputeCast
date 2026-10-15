package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// The optional workspace rule "Publishing a transactional template needs
// approval" (add-on transactional §3.6, ApprovalPolicy). It is off by
// default. When it is on, a publisher asks for approval from the publish
// dialog; the request covers one draft (its hash). Once an approver other
// than the publisher approves, publishing that draft goes ahead; a changed
// draft needs a new request. Rolling back to an earlier version (Publish
// again, Undo) republishes content that was live before and needs none.

// TransactionalTemplates is the part of the transactional service the
// approval request needs.
type TransactionalTemplates interface {
	Template(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (transactional.TransactionalTemplate, error)
	PublishPreview(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in transactional.TransactionalPublishPreviewInput) (transactional.TransactionalPublishPreview, error)
}

// ApprovalTransactionalSnapshot is what a template publish request covers,
// as the approval review shows it.
type ApprovalTransactionalSnapshot struct {
	TemplateID    string `json:"templateId"`
	Key           string `json:"key"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	Version       int    `json:"version"`
	LiveVersion   *int   `json:"liveVersion,omitempty"`
	DraftRevision int    `json:"draftRevision"`
	// Subject is the subject's change ("" when unchanged).
	SubjectBefore string `json:"subjectBefore,omitempty"`
	SubjectAfter  string `json:"subjectAfter,omitempty"`
	// Variables are the variable changes ("+ trial_end_date · date · required").
	Variables      []string `json:"variables"`
	ContentSummary string   `json:"contentSummary"`
	// Breaking is the breaking-change sentence, empty when nothing breaks.
	Breaking    string                            `json:"breaking,omitempty"`
	TestRenders []ApprovalTransactionalTestRender `json:"testRenders"`
	Passed      int                               `json:"passed"`
	Warnings    int                               `json:"warnings"`
	// PublishedVersion is set once the approved draft was published.
	PublishedVersion int `json:"published_version,omitempty"`
}

// ApprovalTransactionalTestRender is one test render at request time.
type ApprovalTransactionalTestRender struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
}

// TransactionalGate implements transactional.PublishGate with the rule.
type TransactionalGate struct{ S *Service }

// approverNamesFor are the approvers who may decide on p's change.
func (g TransactionalGate) approverNamesFor(ctx context.Context, q *store.Queries, wsID uuid.UUID, p *authz.Principal) ([]string, error) {
	ms, err := g.S.members(ctx, q, wsID)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, a := range approversOf(ms) {
		if p == nil || a.ID != p.UserID {
			names = append(names, a.Name)
		}
	}
	return names, nil
}

// Check implements transactional.PublishGate.
func (g TransactionalGate) Check(ctx context.Context, q *store.Queries, req transactional.PublishRequest) (transactional.PublishDecision, error) {
	ws, err := q.GetWorkspace(ctx, req.WorkspaceID)
	if err != nil {
		return transactional.PublishDecision{}, err
	}
	if !ParsePolicy(ws.ApprovalPolicy).TransactionalPublish || req.Rollback {
		return transactional.PublishDecision{Allowed: true}, nil
	}
	approvers, err := g.approverNamesFor(ctx, q, ws.ID, req.Principal)
	if err != nil {
		return transactional.PublishDecision{}, err
	}
	if id, err := q.TxnApprovalForHash(ctx, store.TxnApprovalForHashParams{TemplateID: &req.TemplateID, DraftHash: req.DraftHash}); err == nil {
		return transactional.PublishDecision{Allowed: true, ApprovalID: id.String(), Approvers: approvers}, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return transactional.PublishDecision{}, err
	}
	who := orNames(approvers)
	reason := fmt.Sprintf("The workspace rule “Publishing a transactional template needs approval” is on. %s can approve; you can't approve your own change.", who)
	if pending, err := q.TxnApprovalPending(ctx, &req.TemplateID); err == nil && pending.DraftHash == req.DraftHash {
		reason = fmt.Sprintf("Approval is requested. Waiting for %s.", who)
	}
	if len(approvers) == 0 {
		reason = "The workspace rule “Publishing a transactional template needs approval” is on, and nobody else holds campaigns:approve. Ask an Owner to change Settings › Approvals."
	}
	return transactional.PublishDecision{NeedsApproval: true, Approvers: approvers, Reason: reason}, nil
}

// Published implements transactional.PublishGate: the approval a publish
// relied on remembers the version it made live.
func (g TransactionalGate) Published(ctx context.Context, q *store.Queries, req transactional.PublishRequest, version int) error {
	if req.Rollback {
		return nil
	}
	id, err := q.TxnApprovalForHash(ctx, store.TxnApprovalForHashParams{TemplateID: &req.TemplateID, DraftHash: req.DraftHash})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return q.TxnApprovalMarkPublished(ctx, store.TxnApprovalMarkPublishedParams{ID: id, Version: int32(version)})
}

// orNames is "Marcus Chen or Linh Nguyen".
func orNames(names []string) string {
	switch len(names) {
	case 0:
		return "Nobody"
	case 1:
		return names[0]
	case 2:
		return names[0] + " or " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

// TransactionalApprovalState is the approval side of a template's publish
// dialog.
type TransactionalApprovalState struct {
	// Required is true when the workspace's rule is on.
	Required  bool     `json:"required"`
	Approvers []string `json:"approvers"`
	// Status of the latest request: none, pending, approved,
	// changes_requested or rejected.
	Status string `json:"status" enum:"none,pending,approved,changes_requested,rejected"`
	// CoversDraft is true when that request is about the draft as it is.
	CoversDraft bool       `json:"coversDraft"`
	RequestID   string     `json:"requestId,omitempty"`
	RequestedBy string     `json:"requestedBy,omitempty"`
	RequestedAt *time.Time `json:"requestedAt,omitempty"`
	DecidedBy   string     `json:"decidedBy,omitempty"`
	DecidedAt   *time.Time `json:"decidedAt,omitempty"`
	Note        string     `json:"note,omitempty"`
	// Comment is the approver's last comment (changes requested, rejected).
	Comment string `json:"comment,omitempty"`
	// CanRequest is true for a person who can publish while the rule is on.
	CanRequest    bool   `json:"canRequest"`
	RequestReason string `json:"requestReason,omitempty"`
}

// TemplateApproval returns the approval state of a template's draft.
func (s *Service) TemplateApproval(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (TransactionalApprovalState, error) {
	if s.TransactionalTemplates == nil {
		return TransactionalApprovalState{}, errors.New("transactional templates aren't wired")
	}
	t, err := s.TransactionalTemplates.Template(ctx, ws, p, ref)
	if err != nil {
		return TransactionalApprovalState{}, err
	}
	return s.templateApproval(ctx, s.Q, ws, p, t)
}

func (s *Service) templateApproval(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, t transactional.TransactionalTemplate) (TransactionalApprovalState, error) {
	out := TransactionalApprovalState{Required: ParsePolicy(ws.ApprovalPolicy).TransactionalPublish, Approvers: []string{}, Status: "none"}
	g := TransactionalGate{S: s}
	names, err := g.approverNamesFor(ctx, q, ws.ID, p)
	if err != nil {
		return out, err
	}
	out.Approvers = names
	id, _ := uuid.Parse(t.ID)
	r, err := q.TxnApprovalLatest(ctx, &id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return out, err
	default:
		out.Status = r.Status
		if out.Status != "pending" && out.Status != "approved" && out.Status != "changes_requested" && out.Status != "rejected" {
			out.Status = "none"
		}
		if out.Status != "none" {
			at, decided := r.RequestedAt, r.DecidedAt
			out.RequestID, out.RequestedBy, out.RequestedAt, out.DecidedBy, out.DecidedAt = r.ID.String(), r.RequestedByName, &at, r.DecidedByName, decided
			out.Note, out.Comment, out.CoversDraft = r.Note, r.LastComment, r.DraftHash == t.DraftHash
		}
	}
	switch {
	case !out.Required:
		out.RequestReason = "Publishing doesn't need approval in this workspace."
	case !p.Can(authz.TransactionalPublish):
		out.RequestReason = t.Access.PublishReason
	case !p.IsHuman():
		out.RequestReason = "Only people can ask for approval."
	case len(names) == 0:
		out.RequestReason = "Nobody else holds campaigns:approve. Ask an Owner to change Settings › Approvals."
	default:
		out.CanRequest = true
	}
	return out, nil
}

// TransactionalApprovalInput asks for approval to publish a template's draft.
type TransactionalApprovalInput struct {
	Note string `json:"note,omitempty" maxLength:"1000" doc:"Shown to the approvers"`
}

// RequestTemplateApproval asks the approvers to approve publishing the
// draft as it is now. A pending request for an older draft is replaced.
func (s *Service) RequestTemplateApproval(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TransactionalApprovalInput) (TransactionalApprovalState, error) {
	if s.TransactionalTemplates == nil {
		return TransactionalApprovalState{}, errors.New("transactional templates aren't wired")
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	if !policy.TransactionalPublish {
		return TransactionalApprovalState{}, api.Conflict("Publishing doesn't need approval in this workspace: publish the draft directly.")
	}
	t, err := s.TransactionalTemplates.Template(ctx, ws, p, ref)
	if err != nil {
		return TransactionalApprovalState{}, err
	}
	pv, err := s.TransactionalTemplates.PublishPreview(ctx, ws, p, ref, transactional.TransactionalPublishPreviewInput{})
	if err != nil {
		return TransactionalApprovalState{}, err
	}
	if len(pv.Blocking) > 0 {
		return TransactionalApprovalState{}, api.Err(http.StatusConflict, transactional.CodeTemplateDoesntRun,
			"This draft can't be published yet: "+strings.TrimSuffix(pv.Blocking[0], ".")+". Fix it before asking for approval.")
	}
	templateID, _ := uuid.Parse(t.ID)
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return TransactionalApprovalState{}, err
	}
	approvers := []member{}
	for _, a := range approversOf(ms) {
		if a.ID != p.UserID {
			approvers = append(approvers, a)
		}
	}
	if len(approvers) == 0 {
		return TransactionalApprovalState{}, api.Conflict("Nobody else holds campaigns:approve, so nobody can approve this. Ask an Owner to change Settings › Approvals.")
	}
	snap := Snapshot{Version: pv.NextVersion, DraftHash: t.DraftHash, Subject: t.Subject, Preheader: t.Preheader,
		Rules:         []ApprovalRuleMatch{{Key: RuleTransactionalPublish, Label: "Publishing a transactional template needs approval", Short: "template publish", Badge: "Template publish"}},
		Transactional: transactionalSnapshot(t, pv)}
	if m := memberByID(ms, &p.UserID); m != nil {
		snap.AuthorRole = m.RoleName
	}
	note := strings.TrimSpace(in.Note)
	id := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		if err := q.TxnApprovalCancelPending(ctx, store.TxnApprovalCancelPendingParams{TemplateID: &templateID, Now: &now}); err != nil {
			return err
		}
		if err := q.ApprovalInsertRequest(ctx, store.ApprovalInsertRequestParams{ID: id, WorkspaceID: ws.ID, SubjectType: "transactional_template", TransactionalTemplateID: &templateID,
			RequestedBy: &p.UserID, RequestedAt: now, Note: note, Status: "pending", MatchedRules: []string{RuleTransactionalPublish},
			ApprovalsRequired: int32(policy.ApprovalsRequired), Snapshot: snapJSON(snap)}); err != nil {
			return err
		}
		name := fmt.Sprintf("%s v%d", t.Key, pv.NextVersion)
		if err := emitRequested(ctx, tx, ws, id, "transactional_template", templateID, t.Name, p.Name, []string{RuleTransactionalPublish}, 0); err != nil {
			return err
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalRequested, Severity: "warn", Title: "Approval requested",
			Summary: fmt.Sprintf("%s asked you to approve publishing %s", p.Name, name),
			Body:    fmt.Sprintf("%s asked to publish %s (“%s”)%s", p.Name, name, t.Name, noteSuffix(note)),
			Link:    "/w/" + ws.Slug + "/approvals/" + id.String(), Data: map[string]any{"request_id": id.String(), "transactional_template_id": templateID.String()}}
		for _, a := range approvers {
			if err := notify.ToUser(ctx, tx, a.ID, n); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.publish_approval_requested", TargetType: "transactional_template", TargetID: templateID.String(),
			TargetName: name, After: map[string]any{"status": "pending", "version": pv.NextVersion, "note": note},
			Metadata: map[string]any{"request_id": id.String(), "approvers": orNames(names(approvers)), "rule": RuleTransactionalPublish}})
	})
	if err != nil {
		if isUniqueViolation(err) {
			return TransactionalApprovalState{}, api.Conflict("Someone asked for approval of this template a moment ago. Reload the dialog.")
		}
		return TransactionalApprovalState{}, err
	}
	return s.templateApproval(ctx, s.Q, ws, p, t)
}

func snapJSON(s Snapshot) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

func names(ms []member) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return out
}

func noteSuffix(note string) string {
	if note == "" {
		return ""
	}
	return ": “" + note + "”"
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505")
}

// transactionalSnapshot keeps what the approvers review.
func transactionalSnapshot(t transactional.TransactionalTemplate, pv transactional.TransactionalPublishPreview) *ApprovalTransactionalSnapshot {
	ts := &ApprovalTransactionalSnapshot{TemplateID: t.ID, Key: t.Key, Name: t.Name, Version: pv.NextVersion, LiveVersion: pv.LiveVersion, DraftRevision: t.DraftRevision,
		ContentSummary: pv.Diff.ContentSummary, Breaking: pv.Breaking.Summary, Variables: []string{}, TestRenders: []ApprovalTransactionalTestRender{},
		Passed: pv.Checks.Passed, Warnings: pv.Checks.Warnings}
	if t.Category != nil {
		ts.Category = t.Category.Name
	}
	if pv.Diff.Subject != nil {
		ts.SubjectBefore, ts.SubjectAfter = pv.Diff.Subject.Before, pv.Diff.Subject.After
	}
	for _, v := range pv.Diff.Variables {
		sign := map[string]string{"added": "+", "removed": "−", "changed": "~"}[v.Change]
		ts.Variables = append(ts.Variables, fmt.Sprintf("%s %s · %s", sign, v.Name, v.Summary))
	}
	for _, r := range pv.TestRenders {
		ts.TestRenders = append(ts.TestRenders, ApprovalTransactionalTestRender{Name: r.Name, OK: r.OK})
	}
	return ts
}

// transactionalDecisionAudit names a decision on a template publish request.
func transactionalDecisionAudit(templateID uuid.UUID, decision string) (targetType, targetID, action string) {
	action = map[string]string{DecisionApprove: "transactional.publish_approved", DecisionRequestChanges: "transactional.publish_changes_requested",
		DecisionReject: "transactional.publish_rejected"}[decision]
	return "transactional_template", templateID.String(), action
}

// transactionalDecisionNote points the requester at the editor, where
// publishing the approved draft goes ahead.
func transactionalDecisionNote(ws *store.Workspace, templateID uuid.UUID, snap Snapshot, status, summary, body string) (link, newSummary, newBody string) {
	link = "/w/" + ws.Slug + "/transactional/templates/" + templateID.String()
	if status == "approved" {
		link += "?publish=1"
		version := ""
		if snap.Transactional != nil {
			version = fmt.Sprintf(" v%d", snap.Transactional.Version)
		}
		summary += " · publish" + version + " now"
		body += ". The draft can be published now; a change to it needs a new approval."
	}
	return link, summary, body
}

type TemplateApprovalPath struct {
	api.WorkspaceInput
	Template string `path:"template" maxLength:"64" doc:"The template's ID or UUID"`
}

type templateApprovalInput struct {
	TemplateApprovalPath
	Body TransactionalApprovalInput
}

// RegisterTransactional adds the approval operations of the template
// publish dialog.
func RegisterTransactional(h huma.API, s *Service) {
	tags := []string{"Approvals"}
	api.Get(h, api.Op{ID: "get-transactional-template-approval", Path: "/workspaces/{workspace}/transactional/templates/{template}/approval",
		Summary: "Whether publishing the draft needs approval, and its latest request", Tags: tags, Access: api.Workspace(authz.TransactionalView)},
		func(ctx context.Context, in *TemplateApprovalPath) (*api.Body[TransactionalApprovalState], error) {
			out, err := s.TemplateApproval(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "request-transactional-template-approval", Path: "/workspaces/{workspace}/transactional/templates/{template}/approval",
		Summary:     "Ask for approval to publish the draft",
		Description: "When the workspace's rule is on. The request covers the draft as it is; a changed draft needs a new request. Approvers are notified.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.TransactionalPublish), Status: http.StatusCreated},
		func(ctx context.Context, in *templateApprovalInput) (*api.Body[TransactionalApprovalState], error) {
			out, err := s.RequestTemplateApproval(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
}
