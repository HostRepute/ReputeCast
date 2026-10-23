package agency

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/team"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// AgencyClientNew creates a client workspace.
type AgencyClientNew struct {
	Name     string `json:"name" minLength:"1" maxLength:"80"`
	Slug     string `json:"slug,omitempty" maxLength:"63" doc:"Defaults to the name in lowercase with hyphens"`
	Timezone string `json:"timezone" minLength:"1" maxLength:"64"`
	// TemplateID is the workspace template to start from (optional).
	TemplateID string `json:"templateId,omitempty" format:"uuid"`
	// OwnerID is the client workspace's Owner: an agency manager.
	OwnerID string `json:"ownerId" format:"uuid"`
	// AccountManagerIDs are assigned as Admin; the first is the client's contact.
	AccountManagerIDs []string `json:"accountManagerIds,omitempty" maxItems:"20"`
	// FirstPerson is the client's first person, invited as Client approver
	// or Client viewer.
	FirstPerson *AgencyFirstPerson `json:"firstPerson,omitempty"`
}

// AgencyFirstPerson is the client's first person.
type AgencyFirstPerson struct {
	Email string `json:"email" maxLength:"254"`
	Role  string `json:"role" enum:"client_approver,client_viewer"`
}

// AgencyClientCreated says what creating the client did.
type AgencyClientCreated struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	// Plan is what the template added, when one was used.
	Plan       *AgencyTemplatePlan         `json:"plan,omitempty"`
	Setup      *AgencySetup                `json:"setup,omitempty"`
	Invitation *team.TeamCreatedInvitation `json:"invitation,omitempty"`
	CreatedAt  time.Time                   `json:"createdAt"`
	CreatedBy  string                      `json:"createdBy"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

// CreateClient creates a client workspace: its Owner (an agency manager),
// Admin for every agency manager and the account managers, the client
// roles, the template's setup, and an invitation for the client's first
// person.
func (s *Service) CreateClient(ctx context.Context, in AgencyClientNew) (AgencyClientCreated, error) {
	me, st, err := s.require(ctx, authz.AgencyManage)
	if err != nil {
		return AgencyClientCreated{}, err
	}
	l := i18n.LangFrom(ctx)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return AgencyClientCreated{}, api.Invalid("body.name", mClientName.Text(l, nil))
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = workspaces.Slugify(name)
	}
	if !slugPattern.MatchString(slug) {
		return AgencyClientCreated{}, api.Invalid("body.slug", mSlugInvalid.Text(l, nil))
	}
	if taken, err := s.Q.SlugTaken(ctx, slug); err != nil {
		return AgencyClientCreated{}, err
	} else if taken {
		return AgencyClientCreated{}, api.Invalid("body.slug", mSlugTaken.Text(l, i18n.Args{"slug": slug}))
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return AgencyClientCreated{}, api.Invalid("body.timezone", mTimezone.Text(l, nil))
	}
	people, err := s.people(ctx)
	if err != nil {
		return AgencyClientCreated{}, err
	}
	staffByID := map[uuid.UUID]person{}
	for _, p := range people {
		if p.UserID != nil && p.Active {
			staffByID[*p.UserID] = p
		}
	}
	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil || staffByID[ownerID].Level != authz.AgencyManagerLevel {
		return AgencyClientCreated{}, api.Invalid("body.ownerId", mOwnerManager.Text(l, nil))
	}
	managers := []uuid.UUID{}
	for _, v := range in.AccountManagerIDs {
		id, err := uuid.Parse(v)
		if _, ok := staffByID[id]; err != nil || !ok {
			return AgencyClientCreated{}, api.Invalid("body.accountManagerIds", mPickStaff.Text(l, nil))
		}
		if !slices.Contains(managers, id) {
			managers = append(managers, id)
		}
	}
	var tpl *store.AgencyTemplateRow
	if in.TemplateID != "" {
		id, err := uuid.Parse(in.TemplateID)
		if err != nil {
			return AgencyClientCreated{}, api.Invalid("body.templateId", mPickTemplate.Text(l, nil))
		}
		t, err := s.Q.AgencyTemplate(ctx, id)
		if err != nil {
			return AgencyClientCreated{}, api.Invalid("body.templateId", mPickTemplate.Text(l, nil))
		}
		tpl = &t
	}
	var first *AgencyFirstPerson
	if in.FirstPerson != nil && strings.TrimSpace(in.FirstPerson.Email) != "" {
		e := strings.ToLower(strings.TrimSpace(in.FirstPerson.Email))
		if !looksLikeEmail(e) {
			return AgencyClientCreated{}, api.Invalid("body.firstPerson.email", mEmailInvalid.Text(l, i18n.Args{"value": in.FirstPerson.Email}))
		}
		if !authz.IsClientRole(in.FirstPerson.Role) {
			return AgencyClientCreated{}, api.Invalid("body.firstPerson.role", mFirstPersonRole.Text(l, nil))
		}
		first = &AgencyFirstPerson{Email: e, Role: in.FirstPerson.Role}
	}
	now := s.now()
	out := AgencyClientCreated{CreatedAt: now, CreatedBy: me.Name}
	var ws store.Workspace
	var row store.AgencyWorkspacesRow
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		ws, err = workspaces.Create(ctx, tx, workspaces.CreateParams{Name: name, Timezone: in.Timezone, Owner: ownerID, Source: "agency"})
		if err != nil {
			return err
		}
		if ws.Slug != slug {
			if _, err := tx.Exec(ctx, `UPDATE workspaces SET slug = $2 WHERE id = $1`, ws.ID, slug); err != nil {
				return err
			}
			ws.Slug = slug
		}
		all, err := q.AgencyWorkspaces(ctx)
		if err != nil {
			return err
		}
		steps := []string{"provider", "domain", "invite"}
		var tplID *uuid.UUID
		if tpl != nil {
			tplID = &tpl.ID
			if c := parseContent(tpl.Content); c.NextSteps != nil {
				steps = c.NextSteps
			}
		}
		var manager *uuid.UUID
		if len(managers) > 0 {
			manager = &managers[0]
		}
		color := nextTileColor(all)
		if err := q.AgencyWorkspaceInsert(ctx, store.AgencyWorkspaceInsertParams{WorkspaceID: ws.ID, Kind: "client", AccountManagerID: manager, ClientSince: dateOnly(now),
			TemplateID: tplID, SetupSteps: steps, Color: color, AddedBy: &me.UserID}); err != nil {
			return err
		}
		if err := s.ensureClientRoles(ctx, q, ws.ID); err != nil {
			return err
		}
		row = store.AgencyWorkspacesRow{ID: ws.ID, Slug: ws.Slug, Name: ws.Name, Timezone: ws.Timezone, CreatedAt: ws.CreatedAt, Kind: "client", AccountManagerID: manager,
			ClientSince: dateOnly(now), TemplateID: tplID, SetupSteps: steps, Color: color, AddedAt: now}
		// Agency managers work in every client; the account managers too.
		for _, p := range people {
			if p.UserID == nil || !p.Active || *p.UserID == ownerID {
				continue
			}
			if p.Level != authz.AgencyManagerLevel && !slices.Contains(managers, *p.UserID) {
				continue
			}
			u, err := q.GetUser(ctx, *p.UserID)
			if err != nil {
				return err
			}
			admin, err := q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws.ID, Key: authz.RoleAdmin})
			if err != nil {
				return err
			}
			if err := s.assignTx(ctx, q, tx, row, u, store.AgencyWorkspaceRolesRow{ID: admin.ID, Key: admin.Key, Name: admin.Name, Builtin: true}, me); err != nil {
				return err
			}
		}
		wsID := ws.ID
		after := map[string]any{"name": name, "slug": slug, "timezone": in.Timezone, "owner": staffByID[ownerID].Name, "template": ""}
		if tpl != nil {
			after["template"] = tpl.Name
			// The template's setup, as the Owner would apply it.
			plan, err := s.plan(ctx, q, tx, *tpl, row, authz.All(), &me.UserID, true)
			if err != nil {
				return err
			}
			plan.AppliedAt, plan.AppliedBy = &now, me.Name
			out.Plan = &plan
			result, _ := json.Marshal(plan)
			if err := q.AgencyTemplateApplicationInsert(ctx, store.AgencyTemplateApplicationInsertParams{ID: uuid.Must(uuid.NewV7()), TemplateID: tpl.ID, WorkspaceID: ws.ID,
				AppliedBy: &me.UserID, AppliedAt: now, Result: result}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.template_applied", TargetType: "agency_template", TargetID: tpl.ID.String(), TargetName: tpl.Name,
				After: planSummary(plan), Metadata: map[string]any{"permission": string(authz.AgencyManage)}}); err != nil {
				return err
			}
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.client_created", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: name,
			After: after, Metadata: map[string]any{"permission": string(authz.AgencyManage), "agency": st.Name}}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.client_created", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: name,
			After: after, Metadata: map[string]any{"permission": string(authz.AgencyManage), "agency": st.Name}})
	})
	if err != nil {
		return out, err
	}
	out.Workspace = wsRef(row)
	if first != nil {
		set, err := s.workspacesOf(ctx, s.Q, me)
		if err != nil {
			return out, err
		}
		role, err := s.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws.ID, Key: first.Role})
		if err != nil {
			return out, err
		}
		res, err := s.invite(ctx, me, set, row, first.Email, role.ID)
		if err != nil {
			return out, err
		}
		if len(res.Invitations) > 0 {
			inv := res.Invitations[0]
			out.Invitation = &inv
		}
	}
	setup, err := s.setupOf(ctx, []store.AgencyWorkspacesRow{row})
	if err != nil {
		return out, err
	}
	out.Setup = setup[ws.ID]
	return out, nil
}

// planSummary is what the audit log keeps of an applied template.
func planSummary(p AgencyTemplatePlan) map[string]any {
	n := func(part AgencyPlanPart) int { return len(part.Added) + len(part.Renamed) }
	renamed := []string{}
	for _, part := range []AgencyPlanPart{p.Lists, p.Automations, p.Sections, p.Templates, p.Forms, p.Roles} {
		for _, r := range part.Renamed {
			renamed = append(renamed, r.From+" → "+r.To)
		}
	}
	return map[string]any{"lists": n(p.Lists), "automations": n(p.Automations), "sections": n(p.Sections), "templates": n(p.Templates),
		"forms": n(p.Forms), "roles": n(p.Roles), "client_signoff": p.ClientSignoff, "renamed": renamed, "skipped": len(p.Skipped)}
}

// TemplatePlan previews applying a template to one of your clients.
func (s *Service) TemplatePlan(ctx context.Context, templateID, wsID string) (AgencyTemplatePlan, error) {
	me, _, err := s.require(ctx, authz.AgencyLibrary)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	tpl, w, set, err := s.templateTarget(ctx, me, templateID, wsID)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	return s.plan(ctx, s.Q, nil, tpl, w, set.permsIn(w.ID), &me.UserID, false)
}

// ApplyTemplate applies a template to an existing client: it adds what's
// missing, renames clashes, and never overwrites or deletes anything.
func (s *Service) ApplyTemplate(ctx context.Context, templateID, wsID string) (AgencyTemplatePlan, error) {
	me, _, err := s.require(ctx, authz.AgencyLibrary)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	tpl, w, set, err := s.templateTarget(ctx, me, templateID, wsID)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	var out AgencyTemplatePlan
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		plan, err := s.plan(ctx, q, tx, tpl, w, set.permsIn(w.ID), &me.UserID, true)
		if err != nil {
			return err
		}
		plan.AppliedAt, plan.AppliedBy = &now, me.Name
		out = plan
		result, _ := json.Marshal(plan)
		if err := q.AgencyTemplateApplicationInsert(ctx, store.AgencyTemplateApplicationInsertParams{ID: uuid.Must(uuid.NewV7()), TemplateID: tpl.ID, WorkspaceID: w.ID,
			AppliedBy: &me.UserID, AppliedAt: now, Result: result}); err != nil {
			return err
		}
		if w.TemplateID == nil {
			if err := q.AgencyWorkspaceSetTemplate(ctx, store.AgencyWorkspaceSetTemplateParams{WorkspaceID: w.ID, TemplateID: &tpl.ID, SetupSteps: []string{}}); err != nil {
				return err
			}
		}
		wsIDp := w.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsIDp, Action: "agency.template_applied", TargetType: "agency_template", TargetID: tpl.ID.String(), TargetName: tpl.Name,
			After: planSummary(plan), Metadata: map[string]any{"permission": string(authz.AgencyLibrary)}})
	})
	return out, err
}

func (s *Service) templateTarget(ctx context.Context, me *Staff, templateID, wsID string) (store.AgencyTemplateRow, store.AgencyWorkspacesRow, workspaceSet, error) {
	tid, err := uuid.Parse(templateID)
	if err != nil {
		return store.AgencyTemplateRow{}, store.AgencyWorkspacesRow{}, workspaceSet{}, api.NotFound("Template")
	}
	tpl, err := s.Q.AgencyTemplate(ctx, tid)
	if err != nil {
		return tpl, store.AgencyWorkspacesRow{}, workspaceSet{}, api.MapError(err, "Template")
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return tpl, store.AgencyWorkspacesRow{}, set, err
	}
	id, err := uuid.Parse(wsID)
	if err != nil {
		return tpl, store.AgencyWorkspacesRow{}, set, api.NotFound("Client")
	}
	w, ok := set.find(id)
	if !ok {
		return tpl, w, set, api.NotFound("Client")
	}
	return tpl, w, set, nil
}

// AgencyClientDelete confirms deleting a client.
type AgencyClientDelete struct {
	Confirm string `json:"confirm" doc:"The client's name, typed"`
}

// AgencyClientDeleted says until when it can be restored.
type AgencyClientDeleted struct {
	RestoreUntil time.Time `json:"restoreUntil"`
}

// DeleteClient deletes a client workspace through the existing workspace
// delete: the typed name, sends stop at once, restorable for 7 days.
func (s *Service) DeleteClient(ctx context.Context, wsID string, in AgencyClientDelete) (AgencyClientDeleted, error) {
	if _, _, err := s.require(ctx, authz.AgencyManage); err != nil {
		return AgencyClientDeleted{}, err
	}
	id, err := uuid.Parse(wsID)
	if err != nil {
		return AgencyClientDeleted{}, api.NotFound("Client")
	}
	w, err := s.Q.AgencyWorkspace(ctx, id)
	if err != nil {
		return AgencyClientDeleted{}, api.MapError(err, "Client")
	}
	if w.Kind != "client" {
		return AgencyClientDeleted{}, api.Conflict(mAgencyWorkspaceDelete.Text(i18n.LangFrom(ctx), nil))
	}
	d := s.deps()
	if d.DeleteWorkspace == nil {
		return AgencyClientDeleted{}, api.Conflict(mDeleteUnavailable.Text(i18n.LangFrom(ctx), nil))
	}
	ws, err := s.Q.GetWorkspace(ctx, id)
	if err != nil {
		return AgencyClientDeleted{}, err
	}
	until, err := d.DeleteWorkspace(ctx, &ws, in.Confirm)
	if err != nil {
		return AgencyClientDeleted{}, err
	}
	if err := s.Audit.Log(ctx, audit.Event{Action: "agency.client_deleted", TargetType: "workspace", TargetID: id.String(), TargetName: w.Name,
		After: map[string]any{"restore_until": until}, Metadata: map[string]any{"permission": string(authz.AgencyManage)}}); err != nil {
		return AgencyClientDeleted{}, err
	}
	return AgencyClientDeleted{RestoreUntil: until}, nil
}

// LatestApplication is the last template applied to a workspace (the
// "Template applied" result).
func (s *Service) LatestApplication(ctx context.Context, wsID string) (AgencyTemplatePlan, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	id, err := uuid.Parse(wsID)
	if err != nil {
		return AgencyTemplatePlan{}, api.NotFound("Client")
	}
	if _, ok := set.find(id); !ok {
		return AgencyTemplatePlan{}, api.NotFound("Client")
	}
	a, err := s.Q.AgencyLatestApplication(ctx, id)
	if err != nil {
		return AgencyTemplatePlan{}, api.MapError(err, "Template application")
	}
	var out AgencyTemplatePlan
	_ = json.Unmarshal(a.Result, &out)
	at := a.AppliedAt
	out.AppliedAt, out.AppliedBy = &at, a.AppliedByName
	return out, nil
}
