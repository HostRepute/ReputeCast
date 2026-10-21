package agency

import (
	"context"
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
)

// AgencyModeView is Instance admin › Instance & updates › Agency.
type AgencyModeView struct {
	Enabled bool `json:"enabled"`
	// Name is the agency's name; the instance name until one is set.
	Name          string     `json:"name"`
	InstanceName  string     `json:"instanceName"`
	EnabledByName string     `json:"enabledByName,omitempty"`
	EnabledAt     *time.Time `json:"enabledAt,omitempty"`
	// AgencyWorkspaceID is the agency's own workspace.
	AgencyWorkspaceID string `json:"agencyWorkspaceId,omitempty"`
	// DefaultTemplateID is the template new clients start from.
	DefaultTemplateID string `json:"defaultTemplateId,omitempty"`
	// Workspaces are every workspace of the instance, marked when it's a
	// client or the agency's own.
	Workspaces []AgencyModeWorkspace `json:"workspaces"`
	Templates  []AgencyTemplateRef   `json:"templates"`
	Clients    int                   `json:"clients"`
}

// AgencyModeWorkspace is a workspace in the agency section.
type AgencyModeWorkspace struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	Color    string `json:"color"`
	Client   bool   `json:"client"`
	Agency   bool   `json:"agency"`
}

// AgencyTemplateRef names a workspace template.
type AgencyTemplateRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AgencyModeInput saves the agency section.
type AgencyModeInput struct {
	Enabled           bool     `json:"enabled"`
	Name              string   `json:"name,omitempty" maxLength:"80"`
	AgencyWorkspaceID string   `json:"agencyWorkspaceId,omitempty"`
	DefaultTemplateID string   `json:"defaultTemplateId,omitempty"`
	ClientIDs         []string `json:"clientIds,omitempty" maxItems:"1000"`
}

// Mode returns the agency section of Instance admin.
func (s *Service) Mode(ctx context.Context) (AgencyModeView, error) {
	st, err := s.settings(ctx, s.Q)
	if err != nil {
		return AgencyModeView{}, err
	}
	out := AgencyModeView{Enabled: st.Enabled, Name: st.Name, InstanceName: st.InstanceName, EnabledByName: st.EnabledByName, EnabledAt: st.EnabledAt,
		Workspaces: []AgencyModeWorkspace{}, Templates: []AgencyTemplateRef{}}
	if st.DefaultTemplateID != nil {
		out.DefaultTemplateID = st.DefaultTemplateID.String()
	}
	kinds, err := s.Q.AgencyWorkspaceKinds(ctx)
	if err != nil {
		return out, err
	}
	rows, err := s.Q.AgencyWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	colors := map[uuid.UUID]string{}
	for _, r := range rows {
		colors[r.ID] = tileColor(r.Kind, r.Color)
	}
	for _, k := range kinds {
		w := AgencyModeWorkspace{ID: k.ID.String(), Slug: k.Slug, Name: k.Name, Initials: Initials(k.Name), Color: colors[k.ID], Client: k.Kind == "client", Agency: k.Kind == "agency"}
		if w.Color == "" {
			w.Color = tileColors[len(tileColors)-1]
		}
		if w.Agency {
			out.AgencyWorkspaceID = w.ID
		}
		if w.Client {
			out.Clients++
		}
		out.Workspaces = append(out.Workspaces, w)
	}
	tpls, err := s.Q.AgencyTemplates(ctx)
	if err != nil {
		return out, err
	}
	for _, t := range tpls {
		out.Templates = append(out.Templates, AgencyTemplateRef{ID: t.ID.String(), Name: t.Name})
	}
	return out, nil
}

// SaveMode turns agency mode on or off, names the agency, and marks which
// workspaces are clients and which is the agency's own. Turning it off
// keeps every workspace and membership; only the Agency area and the
// agency roles go, and they come back when it's turned on again.
func (s *Service) SaveMode(ctx context.Context, in AgencyModeInput) (AgencyModeView, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return AgencyModeView{}, api.NotFound("Agency")
	}
	l := i18n.LangFrom(ctx)
	name := strings.TrimSpace(in.Name)
	agencyWS, err := optionalID(in.AgencyWorkspaceID, "body.agencyWorkspaceId", mPickWorkspace.Text(l, nil))
	if err != nil {
		return AgencyModeView{}, err
	}
	tplID, err := optionalID(in.DefaultTemplateID, "body.defaultTemplateId", mPickTemplate.Text(l, nil))
	if err != nil {
		return AgencyModeView{}, err
	}
	clients := map[uuid.UUID]bool{}
	for _, c := range in.ClientIDs {
		id, err := uuid.Parse(c)
		if err != nil {
			return AgencyModeView{}, api.Invalid("body.clientIds", mPickWorkspace.Text(l, nil))
		}
		clients[id] = true
	}
	if agencyWS != nil && clients[*agencyWS] {
		return AgencyModeView{}, api.Invalid("body.agencyWorkspaceId", mAgencyNotClient.Text(l, nil))
	}
	me := &Staff{UserID: u.ID, Name: displayName(u.Name, u.Email), Email: u.Email, Level: authz.AgencyManagerLevel, InstanceAdmin: true,
		Permissions: authz.AgencyRoles[0].Permissions}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		prev, err := q.AgencySettingsForUpdate(ctx)
		if err != nil {
			return err
		}
		kinds, err := q.AgencyWorkspaceKinds(ctx)
		if err != nil {
			return err
		}
		live := map[uuid.UUID]store.AgencyWorkspaceKindsRow{}
		for _, k := range kinds {
			live[k.ID] = k
		}
		for id := range clients {
			if _, ok := live[id]; !ok {
				return api.Invalid("body.clientIds", mPickWorkspace.Text(l, nil))
			}
		}
		if agencyWS != nil {
			if _, ok := live[*agencyWS]; !ok {
				return api.Invalid("body.agencyWorkspaceId", mPickWorkspace.Text(l, nil))
			}
		}
		if tplID != nil {
			if _, err := q.AgencyTemplate(ctx, *tplID); err != nil {
				return api.Invalid("body.defaultTemplateId", mPickTemplate.Text(l, nil))
			}
		}
		enabledBy, enabledAt := prev.EnabledBy, prev.EnabledAt
		if in.Enabled && !prev.Enabled {
			now := s.now()
			enabledBy, enabledAt = &u.ID, &now
		}
		if err := q.AgencySaveSettings(ctx, store.AgencySaveSettingsParams{Enabled: in.Enabled, Name: name, DefaultTemplateID: tplID,
			EnabledBy: enabledBy, EnabledAt: enabledAt, UpdatedBy: &u.ID}); err != nil {
			return err
		}
		if in.Enabled != prev.Enabled {
			action := "agency.mode_enabled"
			if !in.Enabled {
				action = "agency.mode_disabled"
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{Action: action, TargetType: "agency", TargetName: firstNonEmpty(name, "agency"),
				Before: map[string]any{"enabled": prev.Enabled}, After: map[string]any{"enabled": in.Enabled}}); err != nil {
				return err
			}
		}
		if name != prev.Name || !samePtr(tplID, prev.DefaultTemplateID) {
			if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.settings_changed", TargetType: "agency", TargetName: firstNonEmpty(name, "agency"),
				Before: map[string]any{"name": prev.Name, "default_template": idText(prev.DefaultTemplateID)},
				After:  map[string]any{"name": name, "default_template": idText(tplID)}}); err != nil {
				return err
			}
		}
		existing, err := q.AgencyWorkspaces(ctx)
		if err != nil {
			return err
		}
		have := map[uuid.UUID]store.AgencyWorkspacesRow{}
		for _, r := range existing {
			have[r.ID] = r
		}
		today := dateOnly(s.now())
		add := func(id uuid.UUID, kind string) error {
			color := ""
			if kind == "client" {
				color = nextTileColor(existing)
			}
			if err := q.AgencyWorkspaceInsert(ctx, store.AgencyWorkspaceInsertParams{WorkspaceID: id, Kind: kind, ClientSince: today, SetupSteps: []string{},
				Color: color, AddedBy: &u.ID}); err != nil {
				return err
			}
			existing = append(existing, store.AgencyWorkspacesRow{ID: id, Kind: kind, Color: color})
			if kind == "client" {
				if err := s.ensureClientRoles(ctx, q, id); err != nil {
					return err
				}
			}
			wsID := id
			action := "agency.client_added"
			if kind == "agency" {
				action = "agency.workspace_set"
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: action, TargetType: "workspace", TargetID: id.String(), TargetName: live[id].Name,
				After: map[string]any{"kind": kind}, Metadata: map[string]any{"by": "instance admin"}})
		}
		remove := func(r store.AgencyWorkspacesRow) error {
			if _, err := q.AgencyWorkspaceRemove(ctx, r.ID); err != nil {
				return err
			}
			wsID := r.ID
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.client_removed", TargetType: "workspace", TargetID: r.ID.String(), TargetName: r.Name,
				Before: map[string]any{"kind": r.Kind}, Metadata: map[string]any{"by": "instance admin", "summary": "memberships kept"}})
		}
		for _, r := range existing {
			wantClient := clients[r.ID]
			wantAgency := agencyWS != nil && *agencyWS == r.ID
			if (r.Kind == "client" && !wantClient) || (r.Kind == "agency" && !wantAgency) {
				if err := remove(r); err != nil {
					return err
				}
				delete(have, r.ID)
			}
		}
		ordered := make([]uuid.UUID, 0, len(clients))
		for id := range clients {
			ordered = append(ordered, id)
		}
		slices.SortFunc(ordered, func(a, b uuid.UUID) int { return live[a].CreatedAt.Compare(live[b].CreatedAt) })
		for _, id := range ordered {
			if r, ok := have[id]; ok && r.Kind == "client" {
				continue
			}
			if err := add(id, "client"); err != nil {
				return err
			}
		}
		if agencyWS != nil {
			if r, ok := have[*agencyWS]; !ok || r.Kind != "agency" {
				if err := add(*agencyWS, "agency"); err != nil {
					return err
				}
			}
		}
		if !in.Enabled {
			return nil
		}
		// Agency managers work in every client.
		all, err := q.AgencyWorkspaces(ctx)
		if err != nil {
			return err
		}
		people, err := s.peopleTx(ctx, q)
		if err != nil {
			return err
		}
		for _, p := range people {
			if p.Level != authz.AgencyManagerLevel || p.UserID == nil || !p.Active {
				continue
			}
			pu, err := q.GetUser(ctx, *p.UserID)
			if err != nil {
				return err
			}
			if err := s.ensureManagerAccess(ctx, q, tx, all, pu, me); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return AgencyModeView{}, err
	}
	return s.Mode(ctx)
}

// peopleTx is people through a transaction's queries.
func (s *Service) peopleTx(ctx context.Context, q *store.Queries) ([]person, error) {
	return peopleWith(ctx, q)
}

// ensureClientRoles gives a client workspace the two client roles.
func (s *Service) ensureClientRoles(ctx context.Context, q *store.Queries, wsID uuid.UUID) error {
	added := []string{}
	for _, r := range authz.ClientRoles {
		if _, err := q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: wsID, Key: r.Key}); err == nil {
			continue
		} else if !isNoRows(err) {
			return err
		}
		if _, err := q.CreateRole(ctx, store.CreateRoleParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, Key: r.Key, Name: r.Name, Description: r.Description,
			Builtin: true, Permissions: []string{}}); err != nil {
			return err
		}
		added = append(added, r.Name)
	}
	if len(added) == 0 {
		return nil
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.client_roles_added", TargetType: "role", TargetName: strings.Join(added, ", "),
		After: map[string]any{"roles": added}})
}

func optionalID(v, loc, msg string) (*uuid.UUID, error) {
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return nil, api.Invalid(loc, msg)
	}
	return &id, nil
}

func samePtr(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func idText(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// dateOnly is the UTC calendar day of t.
func dateOnly(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
