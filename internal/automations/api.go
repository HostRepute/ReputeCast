package automations

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

type listInput struct {
	api.WorkspaceInput
	Status string `query:"status" enum:"all,live,paused,drafts" default:"all"`
}

type summaryInput struct {
	api.WorkspaceInput
	Days int `query:"days" minimum:"1" maximum:"90" default:"7"`
}

type AutomationIDParam struct {
	api.WorkspaceInput
	ID string `path:"id" doc:"Automation id" maxLength:"40"`
}

type statsInput struct {
	AutomationIDParam
	Days int `query:"days" minimum:"1" maximum:"365" default:"30"`
}

type versionInput struct {
	AutomationIDParam
	Version int `path:"version" minimum:"1"`
}

type automationCreateBody struct {
	Name     string `json:"name,omitempty" maxLength:"120"`
	Template string `json:"template,omitempty" enum:"welcome-series,re-engagement,onboarding" doc:"Starter template; omit for a blank canvas"`
}

type createInput struct {
	api.WorkspaceInput
	Body automationCreateBody
}

type automationMetaBody struct {
	Name        string `json:"name" minLength:"1" maxLength:"120"`
	Description string `json:"description" maxLength:"200"`
}

type metaInput struct {
	AutomationIDParam
	Body automationMetaBody
}

type automationDraftBody struct {
	Graph AutomationGraph `json:"graph"`
	// Revision is the draft revision the editor loaded; 0 skips the check.
	Revision int64 `json:"revision" minimum:"0"`
}

type draftInput struct {
	AutomationIDParam
	Body automationDraftBody
}

type diffInput struct {
	AutomationIDParam
	From int `query:"from" minimum:"0" doc:"Defaults to the live version"`
	To   int `query:"to" minimum:"0" doc:"Defaults to the draft"`
}

type automationPublishBody struct {
	Version   int    `json:"version" minimum:"1" doc:"The draft version reviewed in the publish dialog"`
	Migration string `json:"migration" enum:"finish,move" doc:"finish: contacts in the flow finish on the old version; move: they move to the new one at their current step"`
}

type publishInput struct {
	AutomationIDParam
	Body automationPublishBody
}

type automationPauseBody struct {
	Reason string `json:"reason,omitempty" maxLength:"200"`
}

type pauseInput struct {
	AutomationIDParam
	Body automationPauseBody
}

type automationSecretBody struct {
	// Secret signs webhook calls (HMAC-SHA256). Empty clears it. Never returned.
	Secret string `json:"secret" maxLength:"256"`
}

type secretInput struct {
	AutomationIDParam
	Node string `path:"node" maxLength:"40"`
	Body automationSecretBody
}

type callsInput struct {
	AutomationIDParam
	Node  string `path:"node" maxLength:"40"`
	Limit int    `query:"limit" minimum:"1" maximum:"200" default:"50"`
}

type automationEmailCreateBody struct {
	Name         string `json:"name" minLength:"1" maxLength:"160"`
	Subject      string `json:"subject,omitempty" maxLength:"250"`
	AutomationID string `json:"automationId,omitempty" maxLength:"40"`
}

type emailCreateInput struct {
	api.WorkspaceInput
	Body automationEmailCreateBody
}

type emailInput struct {
	api.WorkspaceInput
	ID      string `path:"id" maxLength:"40"`
	Version int    `query:"version" minimum:"0" doc:"Defaults to the latest version"`
}

type automationEmailUpdateBody struct {
	Subject   string `json:"subject" minLength:"1" maxLength:"250"`
	Preheader string `json:"preheader" maxLength:"250"`
	HTML      string `json:"html" minLength:"1"`
	Text      string `json:"text,omitempty"`
}

type emailUpdateInput struct {
	api.WorkspaceInput
	ID   string `path:"id" maxLength:"40"`
	Body automationEmailUpdateBody
}

type automationEventBody struct {
	Event      string         `json:"event" minLength:"1" maxLength:"100" example:"workspace.created"`
	Email      string         `json:"email,omitempty" maxLength:"320" format:"email"`
	ContactID  string         `json:"contactId,omitempty" maxLength:"40"`
	Properties map[string]any `json:"properties,omitempty"`
}

type eventInput struct {
	api.WorkspaceInput
	Body automationEventBody
}

// AutomationCreated is the result of a create or duplicate.
type AutomationCreated struct {
	ID string `json:"id"`
}

// AutomationTemplates lists the starter templates.
type AutomationTemplates struct {
	Templates []AutomationStarterTemplate `json:"templates"`
}

type AutomationDiff struct {
	From    int                  `json:"from"`
	To      int                  `json:"to"`
	Items   []AutomationDiffItem `json:"items"`
	Changes int                  `json:"changes"`
}

// AutomationWebhookCalls lists a webhook step's recent calls.
type AutomationWebhookCalls struct {
	Calls []AutomationWebhookCall `json:"calls"`
}

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return id, nil
}

// actor is the user an action is attributed to in created_by/updated_by
// columns: the signed-in person, or the creator of the API key.
func actor(ctx context.Context) *uuid.UUID {
	if u := api.UserFrom(ctx); u != nil {
		id := u.ID
		return &id
	}
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		id := p.UserID
		return &id
	}
	return nil
}

func actorID(ctx context.Context) uuid.UUID {
	if a := actor(ctx); a != nil {
		return *a
	}
	return uuid.Nil
}

// Register adds the automations operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Automations"}
	view := api.Workspace(authz.AutomationsView)
	edit := api.Workspace(authz.AutomationsEdit)
	publish := api.Workspace(authz.AutomationsPublish)

	api.Get(h, api.Op{ID: "list-automations", Path: "/workspaces/{workspace}/automations", Summary: "List automations with their 30-day performance", Tags: tags, Access: view},
		func(ctx context.Context, in *listInput) (*api.Body[AutomationList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), in.Status)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "automations-summary", Path: "/workspaces/{workspace}/automations/summary", Summary: "Contacts in flows now and the last N days of automation email", Tags: tags, Access: view},
		func(ctx context.Context, in *summaryInput) (*api.Body[AutomationSummary], error) {
			out, err := s.Summary(ctx, api.WorkspaceFrom(ctx), in.Days)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-automation-templates", Path: "/workspaces/{workspace}/automations/templates", Summary: "List starter templates", Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[AutomationTemplates], error) {
			return api.OK(AutomationTemplates{Templates: Templates()}), nil
		})
	api.Get(h, api.Op{ID: "automation-options", Path: "/workspaces/{workspace}/automations/options", Summary: "Lists, forms, fields, emails, providers, tags and events steps can use", Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[AutomationOptions], error) {
			out, err := s.Options(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-automation", Path: "/workspaces/{workspace}/automations", Summary: "Create a draft automation from a starter template or a blank canvas", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *createInput) (*api.Body[AutomationCreated], error) {
			id, err := s.Create(ctx, api.WorkspaceFrom(ctx), actor(ctx), CreateParams{Name: in.Body.Name, Template: in.Body.Template})
			if err != nil {
				return nil, err
			}
			return api.OK(AutomationCreated{ID: id.String()}), nil
		})
	api.Post(h, api.Op{ID: "send-automation-event", Path: "/workspaces/{workspace}/automations/events", Summary: "Send an API event for a contact (triggers, goals and branches listen to it)", Tags: tags, Access: api.Workspace(authz.SubscribersEdit), Status: http.StatusAccepted},
		func(ctx context.Context, in *eventInput) (*api.Body[AutomationEventResult], error) {
			out, err := s.SendEvent(ctx, api.WorkspaceFrom(ctx), APIEvent{Event: in.Body.Event, Email: in.Body.Email, ContactID: in.Body.ContactID, Properties: in.Body.Properties})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-automation", Path: "/workspaces/{workspace}/automations/{id}", Summary: "Get an automation with its draft graph", Tags: tags, Access: view},
		func(ctx context.Context, in *AutomationIDParam) (*api.Body[AutomationDetail], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-automation", Path: "/workspaces/{workspace}/automations/{id}", Summary: "Rename an automation or change its description", Tags: tags, Access: edit},
		func(ctx context.Context, in *metaInput) (*api.Body[AutomationDetail], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx)
			if err := s.UpdateMeta(ctx, ws, actor(ctx), id, in.Body.Name, in.Body.Description); err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, ws, id)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-automation", Path: "/workspaces/{workspace}/automations/{id}", Summary: "Delete an automation; contacts in it leave (published flows need automations:publish)", Tags: tags, Access: edit},
		func(ctx context.Context, in *AutomationIDParam) (*api.Empty, error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			p := api.PrincipalFrom(ctx)
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), actor(ctx), id, p != nil && p.Can(authz.AutomationsPublish))
		})
	api.Post(h, api.Op{ID: "duplicate-automation", Path: "/workspaces/{workspace}/automations/{id}/duplicate", Summary: "Copy an automation's draft into a new draft", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *AutomationIDParam) (*api.Body[AutomationCreated], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			nid, err := s.Duplicate(ctx, api.WorkspaceFrom(ctx), actor(ctx), id)
			if err != nil {
				return nil, err
			}
			return api.OK(AutomationCreated{ID: nid.String()}), nil
		})
	api.Put(h, api.Op{ID: "save-automation-draft", Path: "/workspaces/{workspace}/automations/{id}/draft", Summary: "Save the canvas as the draft", Tags: tags, Access: edit, MaxBodyBytes: 2 << 20},
		func(ctx context.Context, in *draftInput) (*api.Body[AutomationDraftSaved], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.SaveDraft(ctx, api.WorkspaceFrom(ctx), actor(ctx), id, in.Body.Graph, in.Body.Revision)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "discard-automation-draft", Path: "/workspaces/{workspace}/automations/{id}/draft", Summary: "Discard unpublished changes", Tags: tags, Access: edit},
		func(ctx context.Context, in *AutomationIDParam) (*api.Empty, error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.DiscardDraft(ctx, api.WorkspaceFrom(ctx), actor(ctx), id)
		})
	api.Get(h, api.Op{ID: "automation-stats", Path: "/workspaces/{workspace}/automations/{id}/stats", Summary: "Live stats per step", Tags: tags, Access: view},
		func(ctx context.Context, in *statsInput) (*api.Body[AutomationStats], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.Stats(ctx, api.WorkspaceFrom(ctx), id, in.Days)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-automation-versions", Path: "/workspaces/{workspace}/automations/{id}/versions", Summary: "Version history", Tags: tags, Access: view},
		func(ctx context.Context, in *AutomationIDParam) (*api.Body[AutomationVersions], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.Versions(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-automation-version", Path: "/workspaces/{workspace}/automations/{id}/versions/{version}", Summary: "Get one version's graph", Tags: tags, Access: view},
		func(ctx context.Context, in *versionInput) (*api.Body[AutomationVersionGraph], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.Version(ctx, api.WorkspaceFrom(ctx), id, in.Version)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "restore-automation-version", Path: "/workspaces/{workspace}/automations/{id}/versions/{version}/restore", Summary: "Make an earlier version the draft", Tags: tags, Access: edit},
		func(ctx context.Context, in *versionInput) (*api.Body[AutomationDraftSaved], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.RestoreVersion(ctx, api.WorkspaceFrom(ctx), actor(ctx), id, in.Version)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "automation-diff", Path: "/workspaces/{workspace}/automations/{id}/diff", Summary: "Compare two versions (live → draft by default)", Tags: tags, Access: view},
		func(ctx context.Context, in *diffInput) (*api.Body[AutomationDiff], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.CompareVersions(ctx, api.WorkspaceFrom(ctx), id, in.From, in.To)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "automation-publish-preview", Path: "/workspaces/{workspace}/automations/{id}/publish-preview", Summary: "What publishing the draft would change, for whom, and the pre-publish checks", Tags: tags, Access: view},
		func(ctx context.Context, in *AutomationIDParam) (*api.Body[AutomationPublishPreview], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.PublishPreview(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "publish-automation", Path: "/workspaces/{workspace}/automations/{id}/publish", Summary: "Publish the draft as the live version (may need approval)", Tags: tags, Access: publish},
		func(ctx context.Context, in *publishInput) (*api.Body[AutomationPublishResult], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.Publish(ctx, api.WorkspaceFrom(ctx), actorID(ctx), id, in.Body.Version, in.Body.Migration)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "pause-automation", Path: "/workspaces/{workspace}/automations/{id}/pause", Summary: "Pause a live automation", Tags: tags, Access: publish},
		func(ctx context.Context, in *pauseInput) (*api.Body[AutomationDetail], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx)
			if err := s.Pause(ctx, ws, actorID(ctx), id, in.Body.Reason); err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, ws, id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resume-automation", Path: "/workspaces/{workspace}/automations/{id}/resume", Summary: "Resume a paused automation", Tags: tags, Access: publish},
		func(ctx context.Context, in *AutomationIDParam) (*api.Body[AutomationDetail], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx)
			if err := s.Resume(ctx, ws, actorID(ctx), id); err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, ws, id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-automation-step-secret", Path: "/workspaces/{workspace}/automations/{id}/steps/{node}/secret", Summary: "Set or clear a webhook step's signing secret (write-only)", Tags: tags, Access: edit, Status: http.StatusNoContent},
		func(ctx context.Context, in *secretInput) (*api.Empty, error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.SetStepSecret(ctx, api.WorkspaceFrom(ctx), actor(ctx), id, in.Node, in.Body.Secret)
		})
	api.Get(h, api.Op{ID: "list-automation-webhook-calls", Path: "/workspaces/{workspace}/automations/{id}/steps/{node}/calls", Summary: "Delivery log of a webhook step", Tags: tags, Access: view},
		func(ctx context.Context, in *callsInput) (*api.Body[AutomationWebhookCalls], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			calls, err := s.WebhookCalls(ctx, api.WorkspaceFrom(ctx), id, in.Node, in.Limit)
			return api.OK(AutomationWebhookCalls{Calls: calls}), err
		})
	api.Post(h, api.Op{ID: "create-automation-email", Path: "/workspaces/{workspace}/automation-emails", Summary: "Create an email for a send step", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *emailCreateInput) (*api.Body[AutomationEmailOption], error) {
			var automation *uuid.UUID
			if in.Body.AutomationID != "" {
				id, err := parseID(in.Body.AutomationID, "Automation")
				if err != nil {
					return nil, err
				}
				automation = &id
			}
			out, err := s.CreateEmail(ctx, api.WorkspaceFrom(ctx), actor(ctx), automation, in.Body.Name, in.Body.Subject)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-automation-email", Path: "/workspaces/{workspace}/automation-emails/{id}", Summary: "Save new content as the email's next version", Tags: tags, Access: edit, MaxBodyBytes: 1 << 20},
		func(ctx context.Context, in *emailUpdateInput) (*api.Body[AutomationEmailContent], error) {
			id, err := parseID(in.ID, "Email")
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateEmail(ctx, api.WorkspaceFrom(ctx), actor(ctx), id, EmailUpdate{Subject: in.Body.Subject, Preheader: in.Body.Preheader, HTML: in.Body.HTML, Text: in.Body.Text})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-automation-email", Path: "/workspaces/{workspace}/automation-emails/{id}", Summary: "Get an email version (for previews)", Tags: tags, Access: view},
		func(ctx context.Context, in *emailInput) (*api.Body[AutomationEmailContent], error) {
			id, err := parseID(in.ID, "Email")
			if err != nil {
				return nil, err
			}
			out, err := s.Email(ctx, api.WorkspaceFrom(ctx), id, in.Version)
			return api.OK(out), err
		})
}

// CompareVersions diffs two versions; zeros mean live → draft.
func (s *Service) CompareVersions(ctx context.Context, ws *store.Workspace, id uuid.UUID, from, to int) (AutomationDiff, error) {
	a, err := s.Q.GetAutomation(ctx, store.GetAutomationParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return AutomationDiff{}, api.MapError(err, "Automation")
	}
	if to == 0 {
		to = int(a.DraftVersion)
	}
	if from == 0 && a.LiveVersion != nil {
		from = int(*a.LiveVersion)
	}
	if from == 0 || from == to {
		if to > 1 {
			from = to - 1
		} else {
			from = to
		}
	}
	fv, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: int32(from)})
	if err != nil {
		return AutomationDiff{}, api.MapError(err, "Version")
	}
	tv, err := s.Q.GetAutomationVersion(ctx, store.GetAutomationVersionParams{AutomationID: id, Version: int32(to)})
	if err != nil {
		return AutomationDiff{}, api.MapError(err, "Version")
	}
	refs, err := s.loadRefs(ctx, s.Q, ws.ID)
	if err != nil {
		return AutomationDiff{}, err
	}
	failures := map[string]WebhookFailure{}
	if rows, err := s.Q.AutomationStepFailures(ctx, id); err == nil {
		for _, f := range rows {
			failures[f.NodeID] = WebhookFailure{Since: f.FirstAt, Failures: f.Failures}
		}
	}
	items := Diff(s.graphOf(fv.Graph), s.graphOf(tv.Graph), DiffContext{FromRefs: refs, ToVersion: to, Loc: workspaceLoc(ws), Failures: failures})
	return AutomationDiff{From: from, To: to, Items: items, Changes: Changes(items)}, nil
}
