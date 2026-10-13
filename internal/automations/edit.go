package automations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Limits keep graphs small enough to edit and diff instantly.
const (
	maxNodes = 200
	maxEdges = 400
)

// CreateParams starts a flow from a template or a blank canvas.
type CreateParams struct {
	Name     string
	Template string
}

// Create makes a draft flow. Templates also create their emails.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, by *uuid.UUID, p CreateParams) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	name := strings.TrimSpace(p.Name)
	graph := AutomationGraph{Nodes: []AutomationNode{}, Edges: []AutomationEdge{}}
	description, summary := "", "created from a blank canvas"
	var tpl templateDef
	if p.Template != "" {
		t, ok := templateByKey(p.Template)
		if !ok {
			return uuid.Nil, api.Invalid("body.template", "There's no starter template with that name. Pick welcome-series, re-engagement or onboarding.")
		}
		tpl = t
		if name == "" {
			name = t.Name
		}
		description = t.description
		summary = fmt.Sprintf("created from the %s template", t.Name)
	}
	if name == "" {
		name = "Untitled automation"
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if tpl.Key != "" {
			lists, err := q.AutomationLists(ctx, ws.ID)
			if err != nil {
				return err
			}
			listID := ""
			if len(lists) > 0 {
				listID = lists[0].ID.String()
			}
			graph = tpl.build(listID)
		}
		if _, err := q.CreateAutomation(ctx, store.CreateAutomationParams{
			ID: id, WorkspaceID: ws.ID, Name: name, Description: description, TemplateKey: tpl.Key, Trigger: triggerOf(graph), CreatedBy: by,
		}); err != nil {
			return err
		}
		if tpl.Key != "" {
			// The template's emails, made for this flow.
			emailIDs := map[string]uuid.UUID{}
			for _, e := range tpl.emails {
				eid := uuid.Must(uuid.NewV7())
				emailIDs["tpl:"+e.key] = eid
				c := starterEmail(e)
				if err := q.CreateAutomationEmail(ctx, store.CreateAutomationEmailParams{ID: eid, WorkspaceID: ws.ID, AutomationID: &id, Name: e.name, CreatedBy: by}); err != nil {
					return err
				}
				if err := q.InsertAutomationEmailVersion(ctx, store.InsertAutomationEmailVersionParams{EmailID: eid, Version: 1, Subject: c.Subject, Preheader: c.Preheader, Mjml: c.MJML, Html: c.HTML, PlainText: c.Text, CreatedBy: by}); err != nil {
					return err
				}
			}
			for i := range graph.Nodes {
				if e := graph.Nodes[i].Data.Email; e != nil {
					if eid, ok := emailIDs[e.EmailID]; ok {
						e.EmailID = eid.String()
					}
				}
			}
			graph = Layout(graph)
		}
		if err := q.InsertAutomationVersion(ctx, store.InsertAutomationVersionParams{AutomationID: id, Version: 1, Graph: graph.MustJSON(), CreatedBy: by, Summary: summary}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.create", TargetType: "automation", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "template": tpl.Key, "steps": len(graph.Nodes)}})
	})
	return id, err
}

// UpdateMeta renames a flow or changes its description.
func (s *Service) UpdateMeta(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return api.Invalid("body.name", "Give the automation a name.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if a.Name == name && a.Description == description {
			return nil
		}
		if _, err := q.UpdateAutomationMeta(ctx, store.UpdateAutomationMetaParams{ID: id, Name: name, Description: description, UpdatedBy: by}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.update", TargetType: "automation", TargetID: id.String(), TargetName: name,
			Before: map[string]any{"name": a.Name, "description": a.Description}, After: map[string]any{"name": name, "description": description}})
	})
}

// AutomationDraftSaved is the result of a save.
type AutomationDraftSaved struct {
	DraftVersion int                    `json:"draftVersion"`
	Revision     int64                  `json:"revision"`
	Changes      int                    `json:"changes" doc:"Unpublished changes compared with the live version"`
	Issues       []AutomationGraphIssue `json:"issues"`
}

// ErrRevisionConflict means someone saved the draft since the editor loaded it.
var ErrRevisionConflict = errors.New("revision conflict")

// SaveDraft stores the canvas. The first change to a published flow starts a
// new draft version; later saves update it. Positions alone don't count as a
// change for the audit log.
func (s *Service) SaveDraft(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, g AutomationGraph, revision int64) (AutomationDraftSaved, error) {
	var out AutomationDraftSaved
	if len(g.Nodes) > maxNodes || len(g.Edges) > maxEdges {
		return out, api.Invalid("body.graph", fmt.Sprintf("A flow can have up to %d steps and %d connections.", maxNodes, maxEdges))
	}
	g = normalizeGraph(g)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if revision != 0 && a.DraftRevision != revision {
			return api.Err(http.StatusConflict, api.CodeConflict, "Someone else saved this automation since you opened it. Reload to get their changes; yours stay in this tab until you do.").
				WithMeta("revision", a.DraftRevision)
		}
		refs, err := s.loadRefs(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		prev, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: a.DraftVersion})
		if err != nil {
			return err
		}
		before := s.graphOf(prev.Graph)
		version := a.DraftVersion
		if prev.PublishedAt != nil {
			version = a.DraftVersion + 1
			if a.LiveVersion != nil && *a.LiveVersion >= version {
				version = *a.LiveVersion + 1
			}
			if err := q.InsertAutomationVersion(ctx, store.InsertAutomationVersionParams{AutomationID: id, Version: version, Graph: g.MustJSON(), CreatedBy: by}); err != nil {
				return err
			}
		} else if _, err := q.UpdateAutomationDraftGraph(ctx, store.UpdateAutomationDraftGraphParams{AutomationID: id, Version: version, Graph: g.MustJSON()}); err != nil {
			return err
		}
		if err := q.SetAutomationDraft(ctx, store.SetAutomationDraftParams{ID: id, DraftVersion: version, UpdatedBy: by, Trigger: triggerOf(g)}); err != nil {
			return err
		}
		out.DraftVersion = int(version)
		out.Revision = a.DraftRevision + 1
		out.Issues = Validate(g, refs)
		if a.LiveVersion != nil {
			lv, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: *a.LiveVersion})
			if err != nil {
				return err
			}
			out.Changes = Changes(Diff(s.graphOf(lv.Graph), g, DiffContext{FromRefs: refs, ToVersion: int(version)}))
		}
		edits := Diff(before, g, DiffContext{FromRefs: refs, ToVersion: int(version)})
		if Changes(edits) == 0 {
			return nil
		}
		var lines []string
		for _, it := range edits {
			if it.Kind != "unchanged" {
				lines = append(lines, it.Kind+": "+it.Title)
			}
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.draft_update", TargetType: "automation", TargetID: id.String(), TargetName: a.Name,
			Metadata: map[string]any{"version": version, "revision": out.Revision, "edits": lines}})
	})
	return out, err
}

// normalizeGraph drops read-only fields and fills empty slices.
func normalizeGraph(g AutomationGraph) AutomationGraph {
	if g.Nodes == nil {
		g.Nodes = []AutomationNode{}
	}
	if g.Edges == nil {
		g.Edges = []AutomationEdge{}
	}
	for i := range g.Nodes {
		if w := g.Nodes[i].Data.Webhook; w != nil {
			w.HasSecret = false
			w.URL = strings.TrimSpace(w.URL)
		}
	}
	return g
}

// DiscardDraft throws away unpublished changes.
func (s *Service) DiscardDraft(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if a.LiveVersion == nil {
			return api.Conflict("This automation was never published, so there's no live version to go back to. Delete it instead.")
		}
		if a.DraftVersion == *a.LiveVersion {
			return nil
		}
		if _, err := tx.Exec(ctx, `DELETE FROM automation_versions WHERE automation_id = $1 AND version = $2 AND published_at IS NULL`, id, a.DraftVersion); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE automations SET draft_version = live_version, draft_revision = draft_revision + 1, updated_by = $2, updated_at = now() WHERE id = $1`, id, by); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.draft_discard", TargetType: "automation", TargetID: id.String(), TargetName: a.Name,
			Metadata: map[string]any{"discarded_version": a.DraftVersion}})
	})
}

// RestoreVersion makes an earlier version the draft.
func (s *Service) RestoreVersion(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, version int) (AutomationDraftSaved, error) {
	var out AutomationDraftSaved
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return out, api.MapError(err, "Automation")
	}
	v, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: int32(version)})
	if err != nil {
		return out, api.MapError(err, "Version")
	}
	out, err = s.SaveDraft(ctx, ws, by, id, s.graphOf(v.Graph), a.DraftRevision)
	if err != nil {
		return out, err
	}
	wsID := ws.ID
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "automation.version_restore", TargetType: "automation", TargetID: id.String(), TargetName: a.Name,
		Metadata: map[string]any{"restored": version, "draft_version": out.DraftVersion}})
	return out, err
}

// Duplicate copies a flow's draft into a new draft flow.
func (s *Service) Duplicate(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID) (uuid.UUID, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return uuid.Nil, api.MapError(err, "Automation")
	}
	v, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: a.DraftVersion})
	if err != nil {
		return uuid.Nil, err
	}
	g := normalizeGraph(s.graphOf(v.Graph))
	nid := uuid.Must(uuid.NewV7())
	name := "Copy of " + a.Name
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.CreateAutomation(ctx, store.CreateAutomationParams{ID: nid, WorkspaceID: ws.ID, Name: name, Description: a.Description, TemplateKey: a.TemplateKey, Trigger: triggerOf(g), CreatedBy: by}); err != nil {
			return err
		}
		if err := q.InsertAutomationVersion(ctx, store.InsertAutomationVersionParams{AutomationID: nid, Version: 1, Graph: g.MustJSON(), CreatedBy: by, Summary: "duplicated from " + a.Name}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.duplicate", TargetType: "automation", TargetID: nid.String(), TargetName: name,
			Metadata: map[string]any{"source": id.String()}})
	})
	return nid, err
}

// Delete removes a flow; contacts in it leave right away.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, canPublish bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		if a.LiveVersion != nil && !canPublish {
			return api.Forbidden("automations:publish", nil)
		}
		exited, err := q.ExitAutomationRuns(ctx, store.ExitAutomationRunsParams{AutomationID: id, ExitReason: "deleted"})
		if err != nil {
			return err
		}
		if err := q.SoftDeleteAutomation(ctx, store.SoftDeleteAutomationParams{ID: id, UpdatedBy: by}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation.delete", TargetType: "automation", TargetID: id.String(), TargetName: a.Name,
			Before: map[string]any{"status": a.Status}, Metadata: map[string]any{"contacts_removed": exited}})
	})
}

// SetStepSecret stores (or with an empty secret, clears) a webhook step's
// signing secret. It is sealed at rest and never returned.
func (s *Service) SetStepSecret(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, node, secret string) error {
	secret = strings.TrimSpace(secret)
	if len(secret) > 0 && len(secret) < 16 {
		return api.Invalid("body.secret", "Use a signing secret of at least 16 characters.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		a, err := q.LockAutomation(ctx, store.LockAutomationParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Automation")
		}
		v, err := q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: a.DraftVersion})
		if err != nil {
			return err
		}
		n := s.graphOf(v.Graph).Node(node)
		if n == nil || n.Type != NodeWebhook {
			return api.NotFound("Webhook step")
		}
		wsID := ws.ID
		ev := audit.Event{WorkspaceID: &wsID, TargetType: "automation", TargetID: id.String(), TargetName: a.Name, Metadata: map[string]any{"step": node}}
		if secret == "" {
			if _, err := q.DeleteAutomationStepSecret(ctx, store.DeleteAutomationStepSecretParams{AutomationID: id, NodeID: node}); err != nil {
				return err
			}
			ev.Action = "automation.step_secret_clear"
			return s.Audit.LogWith(ctx, q, ev)
		}
		sealed, err := s.Box.Seal([]byte(secret), secretAD(id, node))
		if err != nil {
			return err
		}
		if err := q.UpsertAutomationStepSecret(ctx, store.UpsertAutomationStepSecretParams{AutomationID: id, NodeID: node, SecretEnc: sealed, UpdatedBy: by}); err != nil {
			return err
		}
		ev.Action = "automation.step_secret_set"
		return s.Audit.LogWith(ctx, q, ev)
	})
}

func secretAD(automation uuid.UUID, node string) string {
	return "automation:" + automation.String() + ":step:" + node + ":secret"
}

// CreateEmail makes a new email for a send step.
func (s *Service) CreateEmail(ctx context.Context, ws *store.Workspace, by *uuid.UUID, automation *uuid.UUID, name, subject string) (AutomationEmailOption, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return AutomationEmailOption{}, api.Invalid("body.name", "Name the email.")
	}
	id := uuid.Must(uuid.NewV7())
	c := BlankEmail(name, subject)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if automation != nil {
			if _, err := q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: *automation}); err != nil {
				return api.MapError(err, "Automation")
			}
		}
		if err := q.CreateAutomationEmail(ctx, store.CreateAutomationEmailParams{ID: id, WorkspaceID: ws.ID, AutomationID: automation, Name: name, CreatedBy: by}); err != nil {
			return err
		}
		if err := q.InsertAutomationEmailVersion(ctx, store.InsertAutomationEmailVersionParams{EmailID: id, Version: 1, Subject: c.Subject, Preheader: c.Preheader, Mjml: c.MJML, Html: c.HTML, PlainText: c.Text, CreatedBy: by}); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation_email.create", TargetType: "automation_email", TargetID: id.String(), TargetName: name})
	})
	return AutomationEmailOption{ID: id.String(), Name: name, LatestVersion: 1, Subject: c.Subject, UpdatedAt: s.now()}, err
}

// EmailUpdate is new content for an email; it becomes the next version.
type EmailUpdate struct {
	Subject   string
	Preheader string
	HTML      string
	Text      string
}

// UpdateEmail saves new content as the email's next version. Steps keep the
// version they point at until someone picks the new one, so a live flow
// never changes what it sends without a publish. Content edited as HTML
// drops the MJML source (the composer reopens it in HTML mode).
func (s *Service) UpdateEmail(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, u EmailUpdate) (AutomationEmailContent, error) {
	if strings.TrimSpace(u.Subject) == "" {
		return AutomationEmailContent{}, api.Invalid("body.subject", "Write a subject line.")
	}
	if strings.TrimSpace(u.HTML) == "" {
		return AutomationEmailContent{}, api.Invalid("body.html", "The email is empty. Write its HTML.")
	}
	if len(u.HTML) > 512<<10 {
		return AutomationEmailContent{}, api.Invalid("body.html", "Keep the email under 512 KB; Gmail clips anything over 102 KB anyway.")
	}
	var version int32
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var name string
		var latest int32
		if err := tx.QueryRow(ctx, `SELECT name, latest_version FROM automation_emails WHERE workspace_id = $1 AND id = $2 AND deleted_at IS NULL FOR UPDATE`, ws.ID, id).Scan(&name, &latest); err != nil {
			return api.MapError(err, "Email")
		}
		version = latest + 1
		q := store.New(tx)
		if err := q.InsertAutomationEmailVersion(ctx, store.InsertAutomationEmailVersionParams{EmailID: id, Version: version, Subject: u.Subject, Preheader: u.Preheader, Html: u.HTML, PlainText: u.Text, CreatedBy: by}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE automation_emails SET latest_version = $2, updated_at = now() WHERE id = $1`, id, version); err != nil {
			return err
		}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "automation_email.update", TargetType: "automation_email", TargetID: id.String(), TargetName: name,
			Metadata: map[string]any{"version": version}})
	})
	if err != nil {
		return AutomationEmailContent{}, err
	}
	return s.Email(ctx, ws, id, int(version))
}

// AutomationEmailContent is one version of an email, for previews.
type AutomationEmailContent struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Version       int    `json:"version"`
	LatestVersion int    `json:"latestVersion"`
	Subject       string `json:"subject"`
	Preheader     string `json:"preheader"`
	HTML          string `json:"html"`
	Text          string `json:"text"`
}

// Email returns one version of an email (the latest when version is 0).
func (s *Service) Email(ctx context.Context, ws *store.Workspace, id uuid.UUID, version int) (AutomationEmailContent, error) {
	if version <= 0 {
		emails, err := s.Q.ListAutomationEmails(ctx, ws.ID)
		if err != nil {
			return AutomationEmailContent{}, err
		}
		for _, e := range emails {
			if e.ID == id {
				version = int(e.LatestVersion)
			}
		}
	}
	e, err := s.Q.GetAutomationEmailVersion(ctx, store.GetAutomationEmailVersionParams{WorkspaceID: ws.ID, ID: id, Version: int32(version)})
	if err != nil {
		return AutomationEmailContent{}, api.MapError(err, "Email")
	}
	return AutomationEmailContent{ID: e.ID.String(), Name: e.Name, Version: int(e.Version), LatestVersion: int(e.LatestVersion), Subject: e.Subject, Preheader: e.Preheader, HTML: e.Html, Text: e.PlainText}, nil
}
