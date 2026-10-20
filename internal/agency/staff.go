package agency

import (
	"context"
	"encoding/json"
	"net/http"
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

// person is one member of the agency's staff: a staff row, an instance
// admin (an agency manager without a row), or someone invited by email who
// has no account yet.
type person struct {
	StaffID       *uuid.UUID
	UserID        *uuid.UUID
	Name          string
	Email         string
	Color         string
	Level         string
	Title         string
	InstanceAdmin bool
	Active        bool
	LastActiveAt  *time.Time
	AddedAt       time.Time
	// Permissions narrow the level's when set.
	Permissions []string
}

func (p person) key() string {
	if p.StaffID != nil {
		return p.StaffID.String()
	}
	return "admin:" + p.UserID.String()
}

// people lists the staff, in the order they joined; instance admins
// without a staff row come first.
func (s *Service) people(ctx context.Context) ([]person, error) { return peopleWith(ctx, s.Q) }

func peopleWith(ctx context.Context, q *store.Queries) ([]person, error) {
	rows, err := q.AgencyStaffRows(ctx)
	if err != nil {
		return nil, err
	}
	admins, err := q.AgencyInstanceAdmins(ctx)
	if err != nil {
		return nil, err
	}
	out := []person{}
	seen := map[uuid.UUID]bool{}
	for _, r := range rows {
		if r.UserID != nil {
			seen[*r.UserID] = true
		}
	}
	for _, a := range admins {
		if seen[a.ID] {
			continue
		}
		id := a.ID
		out = append(out, person{UserID: &id, Name: displayName(a.Name, a.Email), Email: a.Email, Color: a.AvatarColor, Level: authz.AgencyManagerLevel,
			InstanceAdmin: true, Active: a.Status == "active", LastActiveAt: a.LastActiveAt})
	}
	for _, r := range rows {
		id := r.ID
		p := person{StaffID: &id, UserID: r.UserID, Name: displayName(r.Name, r.Email), Email: r.Email, Color: r.AvatarColor, Level: r.Level, Title: r.Title,
			InstanceAdmin: r.IsInstanceAdmin, Active: r.UserStatus == "active", LastActiveAt: r.LastActiveAt, AddedAt: r.AddedAt, Permissions: r.Permissions}
		if p.InstanceAdmin {
			p.Level = authz.AgencyManagerLevel
		}
		out = append(out, p)
	}
	return out, nil
}

// AgencyStaffMember is one row of Agency › Staff.
type AgencyStaffMember struct {
	ID          string `json:"id" doc:"Staff id: the staff row, or admin:<user id> for an instance admin"`
	UserID      string `json:"userId,omitempty" doc:"Empty while an invited person has no account yet"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarColor string `json:"avatarColor"`
	Level       string `json:"level" enum:"agency_manager,account_manager"`
	// Title is what they do at the agency ("designer").
	Title         string `json:"title,omitempty"`
	InstanceAdmin bool   `json:"instanceAdmin"`
	// Invited: no account yet; they join through the invitations sent to them.
	Invited bool `json:"invited"`
	// EveryClient is true when they work in every client workspace.
	EveryClient bool `json:"everyClient"`
	// Clients names the clients they're assigned to, when not every one.
	Clients []string `json:"clients"`
	// Roles are the roles they hold in their clients, most common first.
	Roles        []AgencyRoleRef `json:"roles"`
	LastActiveAt *time.Time      `json:"lastActiveAt,omitempty"`
	You          bool            `json:"you"`
	// Permissions are the agency permissions they hold: their level's, or
	// fewer when narrowed.
	Permissions []string `json:"permissions"`
}

// AgencyRoleRef names a role.
type AgencyRoleRef struct {
	ID      string `json:"id,omitempty"`
	Key     string `json:"key"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
}

// AgencyAssignmentRow is one workspace of the assignment matrix.
type AgencyAssignmentRow struct {
	Workspace AgencyWorkspaceRef     `json:"workspace"`
	Cells     []AgencyAssignmentCell `json:"cells"`
	// Roles are the workspace's roles, each with whether you can assign it.
	Roles []AgencyAssignableRole `json:"roles"`
	// CanAssign: you can change assignments here (agency:manage, or an
	// assignment of your own in this client).
	CanAssign bool `json:"canAssign"`
	// YourRole is your own role here, for "Your role here".
	YourRole string `json:"yourRole,omitempty"`
}

// AgencyWorkspaceRef names one of the agency's workspaces.
type AgencyWorkspaceRef struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	Color    string `json:"color"`
	Kind     string `json:"kind" enum:"client,agency"`
}

// AgencyAssignmentCell is one staff member in one workspace.
type AgencyAssignmentCell struct {
	StaffID string `json:"staffId"`
	// Role is empty when they aren't assigned there.
	Role   *AgencyRoleRef `json:"role,omitempty"`
	Source string         `json:"source,omitempty"`
}

// AgencyAssignableRole is a role in the assignment dropdown.
type AgencyAssignableRole struct {
	AgencyRoleRef
	Description string `json:"description"`
	// Grantable is false for roles whose permissions you don't hold there
	// ("Not yours to give") and for Owner (only a transfer makes an Owner).
	Grantable bool   `json:"grantable"`
	Reason    string `json:"reason,omitempty"`
}

// AgencyStaffView is Agency › Staff.
type AgencyStaffView struct {
	Staff       []AgencyStaffMember   `json:"staff"`
	Assignments []AgencyAssignmentRow `json:"assignments"`
	Clients     int                   `json:"clients"`
	// HasAgencyWorkspace: the matrix ends with the agency's own workspace.
	HasAgencyWorkspace bool `json:"hasAgencyWorkspace"`
	// CanManage: you hold agency:manage (add and remove staff).
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
	Reason    string   `json:"reason,omitempty"`
}

func roleRef(id uuid.UUID, key, name string, builtin bool) AgencyRoleRef {
	return AgencyRoleRef{ID: id.String(), Key: key, Name: name, Builtin: builtin}
}

func wsRef(r store.AgencyWorkspacesRow) AgencyWorkspaceRef {
	return AgencyWorkspaceRef{ID: r.ID.String(), Slug: r.Slug, Name: r.Name, Initials: Initials(r.Name), Color: tileColor(r.Kind, r.Color), Kind: r.Kind}
}

// StaffView returns the staff and the assignment matrix.
func (s *Service) StaffView(ctx context.Context) (AgencyStaffView, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyStaffView{}, err
	}
	l := i18n.LangFrom(ctx)
	out := AgencyStaffView{Staff: []AgencyStaffMember{}, Assignments: []AgencyAssignmentRow{}, CanManage: me.Can(authz.AgencyManage), Helpers: s.managerNames(ctx, 2)}
	if !out.CanManage {
		out.Reason = s.denied(ctx, authz.AgencyManage).Detail
	}
	people, err := s.people(ctx)
	if err != nil {
		return out, err
	}
	all, err := s.Q.AgencyWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, 0, len(people))
	for _, p := range people {
		if p.UserID != nil {
			ids = append(ids, *p.UserID)
		}
	}
	ms, err := s.Q.AgencyMemberships(ctx, ids)
	if err != nil {
		return out, err
	}
	byUser := map[uuid.UUID]map[uuid.UUID]store.AgencyMembershipsRow{}
	for _, m := range ms {
		if byUser[m.UserID] == nil {
			byUser[m.UserID] = map[uuid.UUID]store.AgencyMembershipsRow{}
		}
		byUser[m.UserID][m.WorkspaceID] = m
	}
	clients := clientIDs(all)
	out.Clients = len(clients)
	for _, p := range people {
		row := AgencyStaffMember{ID: p.key(), Name: p.Name, Email: p.Email, AvatarColor: p.Color, Level: p.Level, Title: p.Title, InstanceAdmin: p.InstanceAdmin,
			Invited: p.UserID == nil, Clients: []string{}, Roles: []AgencyRoleRef{}, LastActiveAt: p.LastActiveAt, Permissions: p.Permissions}
		if level, ok := authz.Agency(p.Level); ok && (p.Permissions == nil || p.InstanceAdmin) {
			row.Permissions = level.Permissions.Strings()
		}
		if p.UserID != nil {
			row.UserID = p.UserID.String()
			row.You = *p.UserID == me.UserID
			mine := byUser[*p.UserID]
			counts := map[string]int{}
			refs := map[string]AgencyRoleRef{}
			n := 0
			for _, w := range all {
				m, ok := mine[w.ID]
				if !ok || w.Kind != "client" {
					continue
				}
				n++
				row.Clients = append(row.Clients, w.Name)
				counts[m.RoleKey]++
				refs[m.RoleKey] = AgencyRoleRef{Key: m.RoleKey, Name: api.RoleName(l, m.RoleKey, m.RoleName), Builtin: m.RoleBuiltin}
			}
			row.EveryClient = n > 0 && n == len(clients)
			if row.EveryClient {
				row.Clients = []string{}
			}
			keys := make([]string, 0, len(counts))
			for k := range counts {
				keys = append(keys, k)
			}
			slices.SortStableFunc(keys, func(a, b string) int {
				if counts[a] != counts[b] {
					return counts[b] - counts[a]
				}
				return roleRank(a) - roleRank(b)
			})
			for _, k := range keys {
				row.Roles = append(row.Roles, refs[k])
			}
		}
		out.Staff = append(out.Staff, row)
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return out, err
	}
	ordered := slices.Clone(all)
	slices.SortStableFunc(ordered, func(a, b store.AgencyWorkspacesRow) int {
		if a.Kind != b.Kind {
			if a.Kind == "agency" {
				return 1
			}
			return -1
		}
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	for _, w := range ordered {
		if w.Kind == "agency" {
			out.HasAgencyWorkspace = true
		}
		roles, err := s.Q.AgencyWorkspaceRoles(ctx, w.ID)
		if err != nil {
			return out, err
		}
		mine := set.permsIn(w.ID)
		row := AgencyAssignmentRow{Workspace: wsRef(w), Cells: []AgencyAssignmentCell{}, Roles: []AgencyAssignableRole{}}
		_, assigned := set.mine[w.ID]
		row.CanAssign = me.Can(authz.AgencyManage) || assigned
		if m, ok := set.mine[w.ID]; ok {
			row.YourRole = m.RoleKey
		}
		for _, r := range roles {
			perms := authz.Effective(r.Key, r.Builtin, r.Permissions)
			ar := AgencyAssignableRole{AgencyRoleRef: roleRef(r.ID, r.Key, api.RoleName(l, r.Key, r.Name), r.Builtin), Description: roleDescription(r.Key, r.Builtin, r.Name)}
			ar.Grantable, ar.Reason = canAssign(l, mine, r.Key, r.Builtin, perms)
			row.Roles = append(row.Roles, ar)
		}
		slices.SortStableFunc(row.Roles, func(a, b AgencyAssignableRole) int { return roleRank(a.Key) - roleRank(b.Key) })
		for _, p := range people {
			cell := AgencyAssignmentCell{StaffID: p.key()}
			if p.UserID != nil {
				if m, ok := byUser[*p.UserID][w.ID]; ok {
					ref := roleRef(m.RoleID, m.RoleKey, api.RoleName(l, m.RoleKey, m.RoleName), m.RoleBuiltin)
					cell.Role, cell.Source = &ref, m.Source
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		out.Assignments = append(out.Assignments, row)
	}
	return out, nil
}

// roleRank orders roles as the boards do: Owner, Admin, Editor, Viewer,
// the client roles, then custom roles.
func roleRank(key string) int {
	switch key {
	case authz.RoleOwner:
		return 0
	case authz.RoleAdmin:
		return 1
	case authz.RoleEditor:
		return 2
	case authz.RoleViewer:
		return 3
	case authz.RoleClientApprover:
		return 4
	case authz.RoleClientViewer:
		return 5
	}
	return 6
}

// roleDescription is a role's one line in the assignment dropdown.
func roleDescription(key string, builtin bool, name string) string {
	if builtin {
		if r, ok := authz.Builtin(key); ok {
			switch key {
			case authz.RoleOwner:
				return "Transfers ownership and deletes the workspace"
			case authz.RoleAdmin:
				return "Everything except ownership"
			case authz.RoleEditor:
				return "Edits, sends and publishes"
			case authz.RoleViewer:
				return "Read-only, everything except secrets"
			}
			return r.Description
		}
	}
	return name
}

// canAssign is the escalation guard: you can assign only roles whose
// permissions you hold in that workspace, and nobody becomes Owner this way.
func canAssign(l i18n.Lang, granter authz.Set, key string, builtin bool, perms authz.Set) (bool, string) {
	if builtin && key == authz.RoleOwner {
		return false, mNotYoursOwner.Text(l, nil)
	}
	if missing := granter.Missing(perms); len(missing) > 0 {
		return false, mNotYours.Text(l, nil)
	}
	return true, ""
}

// AgencyStaffCandidate is an account that can join the staff.
type AgencyStaffCandidate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarColor string `json:"avatarColor"`
}

// Candidates are the instance's accounts that can join the staff.
func (s *Service) Candidates(ctx context.Context) ([]AgencyStaffCandidate, error) {
	if _, _, err := s.require(ctx, authz.AgencyManage); err != nil {
		return nil, err
	}
	rows, err := s.Q.AgencyUserCandidates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AgencyStaffCandidate, 0, len(rows))
	for _, r := range rows {
		out = append(out, AgencyStaffCandidate{ID: r.ID.String(), Name: displayName(r.Name, r.Email), Email: r.Email, AvatarColor: r.AvatarColor})
	}
	return out, nil
}

// AgencyStaffInput adds someone to the staff: an account of the instance,
// or an email to invite.
type AgencyStaffInput struct {
	UserID string `json:"userId,omitempty" format:"uuid" doc:"An account of this instance"`
	Email  string `json:"email,omitempty" maxLength:"254" doc:"Invite by email instead"`
	Level  string `json:"level" enum:"agency_manager,account_manager"`
	Title  string `json:"title,omitempty" maxLength:"60"`
	// Assignments are an account manager's clients and their role in each.
	// Agency managers get Admin in every client.
	Assignments []AgencyAssignmentSpec `json:"assignments,omitempty" maxItems:"500"`
}

// AgencyAssignmentSpec is a client and a role.
type AgencyAssignmentSpec struct {
	WorkspaceID string `json:"workspaceId" format:"uuid"`
	RoleID      string `json:"roleId" format:"uuid"`
}

// AgencyStaffAdded says what adding someone did.
type AgencyStaffAdded struct {
	ID       string `json:"id"`
	Assigned int    `json:"assigned"`
	// Invitations are the links created when inviting by email.
	Invitations []team.TeamCreatedInvitation `json:"invitations"`
	Emailed     bool                         `json:"emailed"`
}

// AddStaff adds an account of the instance, or invites someone by email.
func (s *Service) AddStaff(ctx context.Context, in AgencyStaffInput) (AgencyStaffAdded, error) {
	me, _, err := s.require(ctx, authz.AgencyManage)
	if err != nil {
		return AgencyStaffAdded{}, err
	}
	l := i18n.LangFrom(ctx)
	out := AgencyStaffAdded{Invitations: []team.TeamCreatedInvitation{}}
	if _, ok := authz.Agency(in.Level); !ok {
		return out, api.Invalid("body.level", mLevelPick.Text(l, nil))
	}
	var target *store.User
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if in.UserID != "" {
		id, err := uuid.Parse(in.UserID)
		if err != nil {
			return out, api.Invalid("body.userId", mPickAccount.Text(l, nil))
		}
		u, err := s.Q.GetUser(ctx, id)
		if err != nil || u.Status != "active" {
			return out, api.Invalid("body.userId", mPickAccount.Text(l, nil))
		}
		target, email = &u, u.Email
	} else {
		if !strings.Contains(email, "@") || strings.ContainsAny(email, " ,;") {
			return out, api.Invalid("body.email", mEmailInvalid.Text(l, i18n.Args{"value": in.Email}))
		}
		if u, err := s.Q.AgencyUserByEmail(ctx, email); err == nil {
			target = &u
		}
	}
	if target != nil {
		if held, err := s.Q.AgencyHoldsClientRole(ctx, target.ID); err != nil {
			return out, err
		} else if held {
			return out, api.Conflict(mClientPerson.Text(l, i18n.Args{"name": displayName(target.Name, target.Email)}))
		}
		if target.IsInstanceAdmin {
			return out, api.Conflict(mAlreadyAdmin.Text(l, i18n.Args{"name": displayName(target.Name, target.Email)}))
		}
	}
	if _, err := s.Q.AgencyStaffOf(ctx, store.AgencyStaffOfParams{UserID: userIDOf(target), Email: email}); err == nil {
		return out, api.Conflict(mAlreadyStaff.Text(l, i18n.Args{"name": email}))
	} else if !isNoRows(err) {
		return out, err
	}
	all, err := s.Q.AgencyWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	specs, err := s.resolveSpecs(ctx, all, in.Assignments, in.Level)
	if err != nil {
		return out, err
	}
	id := uuid.Must(uuid.NewV7())
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return out, err
	}
	// The escalation guard applies to every assignment in the request.
	for _, sp := range specs {
		if ok, reason := canAssign(l, set.permsIn(sp.ws.ID), sp.role.Key, sp.role.Builtin, authz.Effective(sp.role.Key, sp.role.Builtin, sp.role.Permissions)); !ok {
			return out, s.escalation(ctx, sp.ws, me, sp.role, reason)
		}
	}
	var invites []spec
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.AgencyStaffInsert(ctx, store.AgencyStaffInsertParams{ID: id, UserID: userIDOf(target), Email: email, Level: in.Level, Title: strings.TrimSpace(in.Title), AddedBy: &me.UserID}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.staff_added", TargetType: "agency_staff", TargetID: id.String(), TargetName: email,
			After:    map[string]any{"level": in.Level, "title": strings.TrimSpace(in.Title), "invited": target == nil},
			Metadata: map[string]any{"permission": string(authz.AgencyManage), "summary": levelName(in.Level) + clientCount(len(specs))}}); err != nil {
			return err
		}
		if target == nil {
			invites = specs
			return nil
		}
		for _, sp := range specs {
			if err := s.assignTx(ctx, q, tx, sp.ws, *target, sp.role, me); err != nil {
				return err
			}
			out.Assigned++
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	out.ID = id.String()
	// Someone without an account joins through invitations, one per client.
	if len(invites) > 0 {
		out.Emailed = true
		for _, sp := range invites {
			res, err := s.invite(ctx, me, set, sp.ws, email, sp.role.ID)
			if err != nil {
				return out, err
			}
			out.Invitations = append(out.Invitations, res.Invitations...)
			out.Emailed = out.Emailed && res.Emailed
		}
	}
	return out, nil
}

func userIDOf(u *store.User) *uuid.UUID {
	if u == nil {
		return nil
	}
	id := u.ID
	return &id
}

func levelName(level string) string {
	if r, ok := authz.Agency(level); ok {
		return r.Name
	}
	return level
}

func clientCount(n int) string {
	switch n {
	case 0:
		return ""
	case 1:
		return " · 1 workspace"
	}
	return " · " + itoa(n) + " workspaces"
}

type spec struct {
	ws   store.AgencyWorkspacesRow
	role store.AgencyWorkspaceRolesRow
}

// resolveSpecs checks the requested assignments. Agency managers get Admin
// in every workspace of the agency, whatever was asked.
func (s *Service) resolveSpecs(ctx context.Context, all []store.AgencyWorkspacesRow, in []AgencyAssignmentSpec, level string) ([]spec, error) {
	l := i18n.LangFrom(ctx)
	out := []spec{}
	if level == authz.AgencyManagerLevel {
		for _, w := range all {
			roles, err := s.Q.AgencyWorkspaceRoles(ctx, w.ID)
			if err != nil {
				return nil, err
			}
			for _, r := range roles {
				if r.Builtin && r.Key == authz.RoleAdmin {
					out = append(out, spec{ws: w, role: r})
				}
			}
		}
		return out, nil
	}
	seen := map[uuid.UUID]bool{}
	for i, a := range in {
		wsID, err1 := uuid.Parse(a.WorkspaceID)
		roleID, err2 := uuid.Parse(a.RoleID)
		if err1 != nil || err2 != nil {
			return nil, api.Invalid("body.assignments", mAssignmentInvalid.Text(l, nil))
		}
		if seen[wsID] {
			continue
		}
		seen[wsID] = true
		var ws *store.AgencyWorkspacesRow
		for j := range all {
			if all[j].ID == wsID {
				ws = &all[j]
			}
		}
		if ws == nil {
			return nil, api.Invalid("body.assignments", mAssignmentInvalid.Text(l, nil))
		}
		roles, err := s.Q.AgencyWorkspaceRoles(ctx, wsID)
		if err != nil {
			return nil, err
		}
		found := false
		for _, r := range roles {
			if r.ID == roleID {
				if authz.IsClientRole(r.Key) {
					return nil, api.Invalid("body.assignments", mStaffClientRole.Text(l, nil))
				}
				out = append(out, spec{ws: *ws, role: r})
				found = true
			}
		}
		if !found {
			return nil, api.Invalid("body.assignments", mAssignmentInvalid.Text(l, nil))
		}
		_ = i
	}
	return out, nil
}

// invite sends one workspace invitation to a person who has no account
// yet, as the staff member in that workspace.
func (s *Service) invite(ctx context.Context, me *Staff, set workspaceSet, w store.AgencyWorkspacesRow, email string, roleID uuid.UUID) (team.TeamInviteResult, error) {
	d := s.deps()
	if d.Invite == nil {
		return team.TeamInviteResult{Invitations: []team.TeamCreatedInvitation{}}, nil
	}
	ws, err := s.Q.GetWorkspace(ctx, w.ID)
	if err != nil {
		return team.TeamInviteResult{}, err
	}
	p := s.principalIn(me, set, w.ID)
	return d.Invite(ctx, &ws, p, team.TeamInviteInput{Emails: []string{email}, RoleID: roleID.String()})
}

// principalIn is the staff member acting as a member of one workspace,
// with the role they hold there.
func (s *Service) principalIn(me *Staff, set workspaceSet, wsID uuid.UUID) *authz.Principal {
	p := &authz.Principal{Type: authz.ActorUser, ID: me.UserID, UserID: me.UserID, Name: me.Name, Email: me.Email, WorkspaceID: wsID,
		IsInstanceAdmin: me.InstanceAdmin, Permissions: set.permsIn(wsID), MFAVerified: true}
	if m, ok := set.mine[wsID]; ok {
		p.RoleKey, p.RoleName = m.RoleKey, m.RoleName
		_ = json.Unmarshal([]byte("{}"), &p.Scope)
	}
	return p
}

// AgencyStaffUpdate changes someone's level or title.
type AgencyStaffUpdate struct {
	Level string `json:"level" enum:"agency_manager,account_manager"`
	Title string `json:"title,omitempty" maxLength:"60"`
	// Permissions narrows the level's agency permissions (an account
	// manager without agency:usage); omitted, they hold all of the level's.
	// agency:view is always kept.
	Permissions []string `json:"permissions,omitempty" maxItems:"4" enum:"agency:view,agency:usage,agency:library,agency:manage"`
}

// UpdateStaff changes a staff member's level or title. A new agency
// manager gets Admin in every client.
func (s *Service) UpdateStaff(ctx context.Context, staffID string, in AgencyStaffUpdate) error {
	me, _, err := s.require(ctx, authz.AgencyManage)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	id, err := uuid.Parse(staffID)
	if err != nil {
		return api.Conflict(mAdminLevel.Text(l, nil))
	}
	people, err := s.people(ctx)
	if err != nil {
		return err
	}
	var p *person
	for i := range people {
		if people[i].StaffID != nil && *people[i].StaffID == id {
			p = &people[i]
		}
	}
	if p == nil {
		return api.NotFound("Staff member")
	}
	if p.InstanceAdmin && in.Level != authz.AgencyManagerLevel {
		return api.Conflict(mAdminLevel.Text(l, nil))
	}
	all, err := s.Q.AgencyWorkspaces(ctx)
	if err != nil {
		return err
	}
	// Narrowing keeps to the level's permissions, and always to the overview.
	var perms []string
	if in.Permissions != nil {
		level, _ := authz.Agency(in.Level)
		chosen := []authz.Permission{authz.AgencyView}
		for _, v := range in.Permissions {
			pm := authz.Permission(v)
			if !level.Permissions.Has(pm) {
				return api.Invalid("body.permissions", mPermissionLevel.Text(l, i18n.Args{"permission": v}))
			}
			if !slices.Contains(chosen, pm) {
				chosen = append(chosen, pm)
			}
		}
		if len(chosen) < len(level.Permissions) {
			perms = authz.AgencySet(chosen...).Strings()
		}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AgencyStaffUpdate(ctx, store.AgencyStaffUpdateParams{ID: id, Level: in.Level, Title: strings.TrimSpace(in.Title), Permissions: perms}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.staff_changed", TargetType: "agency_staff", TargetID: id.String(), TargetName: p.Email,
			Before:   map[string]any{"level": p.Level, "title": p.Title, "permissions": p.Permissions},
			After:    map[string]any{"level": in.Level, "title": strings.TrimSpace(in.Title), "permissions": perms},
			Metadata: map[string]any{"permission": string(authz.AgencyManage)}}); err != nil {
			return err
		}
		if in.Level == authz.AgencyManagerLevel && p.Level != authz.AgencyManagerLevel && p.UserID != nil {
			u, err := q.GetUser(ctx, *p.UserID)
			if err != nil {
				return err
			}
			return s.ensureManagerAccess(ctx, q, tx, all, u, me)
		}
		return nil
	})
}

// AgencyStaffRemoveInput removes someone from the staff.
type AgencyStaffRemoveInput struct {
	// RemoveAssignments also removes their memberships in the agency's
	// workspaces; false keeps them as ordinary members.
	RemoveAssignments bool `json:"removeAssignments,omitempty"`
	// RevokeKeys revokes the API keys they created in those workspaces.
	RevokeKeys bool `json:"revokeKeys,omitempty"`
}

// RemoveStaff takes someone off the staff.
func (s *Service) RemoveStaff(ctx context.Context, staffID string, in AgencyStaffRemoveInput) error {
	me, _, err := s.require(ctx, authz.AgencyManage)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	id, err := uuid.Parse(staffID)
	if err != nil {
		return api.Conflict(mAdminRemove.Text(l, nil))
	}
	people, err := s.people(ctx)
	if err != nil {
		return err
	}
	var p *person
	for i := range people {
		if people[i].StaffID != nil && *people[i].StaffID == id {
			p = &people[i]
		}
	}
	if p == nil {
		return api.NotFound("Staff member")
	}
	if p.InstanceAdmin {
		return api.Conflict(mAdminRemove.Text(l, nil))
	}
	if p.UserID != nil && *p.UserID == me.UserID {
		return api.Conflict(mRemoveSelf.Text(l, nil))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AgencyStaffDelete(ctx, id); err != nil {
			return err
		}
		removed := []string{}
		if in.RemoveAssignments && p.UserID != nil {
			ms, err := q.AgencyMemberships(ctx, []uuid.UUID{*p.UserID})
			if err != nil {
				return err
			}
			for _, m := range ms {
				ws, err := q.AgencyWorkspace(ctx, m.WorkspaceID)
				if err != nil {
					return err
				}
				if err := s.unassignTx(ctx, q, ws.ID, ws.Name, m, p.Name, in.RevokeKeys); err != nil {
					return err
				}
				removed = append(removed, ws.Name)
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.staff_removed", TargetType: "agency_staff", TargetID: id.String(), TargetName: p.Email,
			Before:   map[string]any{"level": p.Level, "title": p.Title},
			After:    map[string]any{"assignments_removed": removed, "keys_revoked": in.RevokeKeys && len(removed) > 0},
			Metadata: map[string]any{"permission": string(authz.AgencyManage)}})
	})
}

// AgencyAssignInput assigns a staff member to a workspace with a role.
type AgencyAssignInput struct {
	StaffID     string `json:"staffId"`
	WorkspaceID string `json:"workspaceId" format:"uuid"`
	RoleID      string `json:"roleId" format:"uuid"`
}

// Assign gives a staff member a role in one of the agency's workspaces: a
// new membership, or another role for an existing one. You can assign only
// roles whose permissions you hold there.
func (s *Service) Assign(ctx context.Context, in AgencyAssignInput) error {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	w, target, set, err := s.assignmentTarget(ctx, me, in.StaffID, in.WorkspaceID)
	if err != nil {
		return err
	}
	roleID, err := uuid.Parse(in.RoleID)
	if err != nil {
		return api.Invalid("body.roleId", mAssignmentInvalid.Text(l, nil))
	}
	roles, err := s.Q.AgencyWorkspaceRoles(ctx, w.ID)
	if err != nil {
		return err
	}
	var role *store.AgencyWorkspaceRolesRow
	for i := range roles {
		if roles[i].ID == roleID {
			role = &roles[i]
		}
	}
	if role == nil {
		return api.Invalid("body.roleId", mAssignmentInvalid.Text(l, nil))
	}
	if authz.IsClientRole(role.Key) {
		return api.Invalid("body.roleId", mStaffClientRole.Text(l, nil))
	}
	perms := authz.Effective(role.Key, role.Builtin, role.Permissions)
	if ok, reason := canAssign(l, set.permsIn(w.ID), role.Key, role.Builtin, perms); !ok {
		return s.escalation(ctx, w, me, *role, reason)
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		return s.assignTx(ctx, q, tx, w, target, *role, me)
	})
}

// escalation records the refused assignment as denied and explains it:
// "You're Editor in Contoso Coffee, so Admin and Owner aren't available."
func (s *Service) escalation(ctx context.Context, w store.AgencyWorkspacesRow, me *Staff, role store.AgencyWorkspaceRolesRow, reason string) error {
	l := i18n.LangFrom(ctx)
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return err
	}
	yours := ""
	if m, ok := set.mine[w.ID]; ok {
		yours = api.RoleName(l, m.RoleKey, m.RoleName)
	}
	wsID := w.ID
	_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "agency.assignment_changed", Result: audit.Denied, TargetType: "membership", TargetName: role.Name,
		Reason: "role holds permissions the assigner doesn't hold", Metadata: map[string]any{"role": role.Name, "assigner_role": yours, "summary": reason}})
	msg := mEscalationNoRole.Text(l, i18n.Args{"workspace": w.Name})
	if yours != "" {
		msg = mEscalation.Text(l, i18n.Args{"role": yours, "workspace": w.Name, "wanted": api.RoleName(l, role.Key, role.Name)})
	}
	if helpers := s.managerNames(ctx, 2); len(helpers) > 0 {
		msg += " " + mAsk.Text(l, i18n.Args{"helpers": i18n.OrList(l.Locale, helpers)})
	}
	p := api.Err(http.StatusForbidden, api.CodeEscalation, msg)
	if missing := set.permsIn(w.ID).Missing(authz.Effective(role.Key, role.Builtin, role.Permissions)); len(missing) > 0 {
		p.Permission = string(missing[0])
	}
	return p
}

// assignmentTarget resolves the workspace and the staff member of an
// assignment, and checks the caller may change assignments there.
func (s *Service) assignmentTarget(ctx context.Context, me *Staff, staffID, wsID string) (store.AgencyWorkspacesRow, store.User, workspaceSet, error) {
	l := i18n.LangFrom(ctx)
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return store.AgencyWorkspacesRow{}, store.User{}, set, err
	}
	id, err := uuid.Parse(wsID)
	if err != nil {
		return store.AgencyWorkspacesRow{}, store.User{}, set, api.NotFound("Workspace")
	}
	var w *store.AgencyWorkspacesRow
	for i := range set.all {
		if set.all[i].ID == id {
			w = &set.all[i]
		}
	}
	if w == nil {
		return store.AgencyWorkspacesRow{}, store.User{}, set, api.NotFound("Workspace")
	}
	if _, assigned := set.mine[w.ID]; !me.Can(authz.AgencyManage) && !assigned {
		return *w, store.User{}, set, s.denied(ctx, authz.AgencyManage)
	}
	people, err := s.people(ctx)
	if err != nil {
		return *w, store.User{}, set, err
	}
	for _, p := range people {
		if p.key() == staffID {
			if p.UserID == nil {
				return *w, store.User{}, set, api.Conflict(mNoAccountYet.Text(l, i18n.Args{"email": p.Email}))
			}
			u, err := s.Q.GetUser(ctx, *p.UserID)
			if err != nil {
				return *w, store.User{}, set, err
			}
			return *w, u, set, nil
		}
	}
	return *w, store.User{}, set, api.NotFound("Staff member")
}

// assignTx creates or changes a staff member's membership in a workspace.
func (s *Service) assignTx(ctx context.Context, q *store.Queries, tx pgx.Tx, w store.AgencyWorkspacesRow, u store.User, role store.AgencyWorkspaceRolesRow, me *Staff) error {
	l := i18n.LangFrom(ctx)
	wsID := w.ID
	name := displayName(u.Name, u.Email)
	m, err := q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: w.ID, UserID: u.ID})
	switch {
	case isNoRows(err):
		mid := uuid.Must(uuid.NewV7())
		if _, err := q.CreateMember(ctx, store.CreateMemberParams{ID: mid, WorkspaceID: w.ID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: "agency", InvitedBy: &me.UserID}); err != nil {
			return err
		}
		if authz.Effective(role.Key, role.Builtin, role.Permissions).HasSensitive() {
			if err := workspaces.EnsureTwoFAGrace(ctx, tx, u.ID); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.assigned", TargetType: "member", TargetID: mid.String(), TargetName: name,
			After:    map[string]any{"role": role.Name, "source": "agency"},
			Metadata: map[string]any{"user_id": u.ID.String(), "summary": role.Name + " via the agency", "permission": string(authz.AgencyManage)}})
	case err != nil:
		return err
	}
	if m.RoleID == role.ID && m.Status == "active" {
		return nil
	}
	if m.RoleBuiltin && m.RoleKey == authz.RoleOwner {
		owners, err := q.TeamCountActiveOwners(ctx, w.ID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return api.Err(http.StatusConflict, api.CodeLastOwner, mLastOwner.Text(l, i18n.Args{"name": name, "workspace": w.Name}))
		}
	}
	// You can't change the role of someone who holds more than you there.
	set, err := s.workspacesOf(ctx, q, me)
	if err != nil {
		return err
	}
	if missing := set.permsIn(w.ID).Missing(authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions)); len(missing) > 0 && u.ID != me.UserID {
		return api.Err(http.StatusForbidden, api.CodeEscalation, mOutranks.Text(l, i18n.Args{"name": name, "workspace": w.Name}))
	}
	if _, err := q.UpdateMemberRole(ctx, store.UpdateMemberRoleParams{ID: m.ID, WorkspaceID: w.ID, RoleID: role.ID, Scope: m.Scope}); err != nil {
		return err
	}
	if m.Status != "active" {
		if err := q.TeamSetMemberStatus(ctx, store.TeamSetMemberStatusParams{ID: m.ID, WorkspaceID: w.ID, Status: "active"}); err != nil {
			return err
		}
	}
	if authz.Effective(role.Key, role.Builtin, role.Permissions).HasSensitive() {
		if err := workspaces.EnsureTwoFAGrace(ctx, tx, u.ID); err != nil {
			return err
		}
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.assignment_changed", TargetType: "member", TargetID: m.ID.String(), TargetName: name,
		Before:   map[string]any{"role": m.RoleName},
		After:    map[string]any{"role": role.Name},
		Metadata: map[string]any{"user_id": u.ID.String(), "summary": m.RoleName + " → " + role.Name, "permission": string(authz.AgencyManage)}})
}

// ensureManagerAccess gives an agency manager Admin in every workspace of
// the agency where they have no membership yet ("Agency managers get Admin
// in every client workspace this way").
func (s *Service) ensureManagerAccess(ctx context.Context, q *store.Queries, tx pgx.Tx, all []store.AgencyWorkspacesRow, u store.User, me *Staff) error {
	for _, w := range all {
		if _, err := q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: w.ID, UserID: u.ID}); err == nil {
			continue
		} else if !isNoRows(err) {
			return err
		}
		admin, err := q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: w.ID, Key: authz.RoleAdmin})
		if err != nil {
			return err
		}
		role := store.AgencyWorkspaceRolesRow{ID: admin.ID, Key: admin.Key, Name: admin.Name, Builtin: admin.Builtin, Permissions: admin.Permissions}
		if err := s.assignTx(ctx, q, tx, w, u, role, me); err != nil {
			return err
		}
	}
	return nil
}

// AgencyUnassignInput removes a staff member from a workspace.
type AgencyUnassignInput struct {
	StaffID     string `json:"staffId"`
	WorkspaceID string `json:"workspaceId" format:"uuid"`
	// RevokeKeys revokes the API keys and agents they created there; false
	// keeps them (they stop working until the person is assigned again).
	RevokeKeys bool `json:"revokeKeys,omitempty"`
}

// AgencyAssignmentKeys are the keys a staff member created in a workspace,
// for the question removing the assignment asks.
type AgencyAssignmentKeys struct {
	Keys []AgencyKeyRef `json:"keys"`
}

// AgencyKeyRef is an API key or agent.
type AgencyKeyRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind" enum:"api_key,mcp_agent"`
	Prefix string `json:"prefix"`
}

// AssignmentKeys lists the active keys a staff member created in a workspace.
func (s *Service) AssignmentKeys(ctx context.Context, staffID, wsID string) (AgencyAssignmentKeys, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyAssignmentKeys{}, err
	}
	w, u, _, err := s.assignmentTarget(ctx, me, staffID, wsID)
	if err != nil {
		return AgencyAssignmentKeys{}, err
	}
	keys, err := s.Q.TeamMemberAPIKeys(ctx, store.TeamMemberAPIKeysParams{WorkspaceID: w.ID, UserID: &u.ID})
	if err != nil {
		return AgencyAssignmentKeys{}, err
	}
	out := AgencyAssignmentKeys{Keys: []AgencyKeyRef{}}
	for _, k := range keys {
		if k.RevokedAt == nil {
			out.Keys = append(out.Keys, AgencyKeyRef{ID: k.ID.String(), Name: k.Name, Kind: k.Kind, Prefix: k.Prefix})
		}
	}
	return out, nil
}

// Unassign removes a staff member's membership in a workspace and keeps or
// revokes the keys they created there, as deactivating does.
func (s *Service) Unassign(ctx context.Context, in AgencyUnassignInput) error {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	w, u, set, err := s.assignmentTarget(ctx, me, in.StaffID, in.WorkspaceID)
	if err != nil {
		return err
	}
	if u.ID == me.UserID {
		return api.Conflict(mUnassignSelf.Text(l, nil))
	}
	m, err := s.Q.AgencyMemberships(ctx, []uuid.UUID{u.ID})
	if err != nil {
		return err
	}
	var mem *store.AgencyMembershipsRow
	for i := range m {
		if m[i].WorkspaceID == w.ID {
			mem = &m[i]
		}
	}
	if mem == nil {
		return api.NotFound("Assignment")
	}
	if missing := set.permsIn(w.ID).Missing(authz.Effective(mem.RoleKey, mem.RoleBuiltin, mem.RolePermissions)); len(missing) > 0 {
		return api.Err(http.StatusForbidden, api.CodeEscalation, mOutranks.Text(l, i18n.Args{"name": displayName(u.Name, u.Email), "workspace": w.Name}))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.unassignTx(ctx, store.New(tx), w.ID, w.Name, *mem, displayName(u.Name, u.Email), in.RevokeKeys)
	})
}

func (s *Service) unassignTx(ctx context.Context, q *store.Queries, wsID uuid.UUID, wsName string, m store.AgencyMembershipsRow, name string, revokeKeys bool) error {
	l := i18n.LangFrom(ctx)
	if m.RoleBuiltin && m.RoleKey == authz.RoleOwner {
		owners, err := q.TeamCountActiveOwners(ctx, wsID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return api.Err(http.StatusConflict, api.CodeLastOwner, mLastOwner.Text(l, i18n.Args{"name": name, "workspace": wsName}))
		}
	}
	revoked, kept := []string{}, []string{}
	if revokeKeys {
		rows, err := q.TeamRevokeKeysCreatedBy(ctx, store.TeamRevokeKeysCreatedByParams{WorkspaceID: wsID, UserID: &m.UserID})
		if err != nil {
			return err
		}
		for _, k := range rows {
			revoked = append(revoked, k.Name)
			kind := "apikey"
			if k.Kind == "mcp_agent" {
				kind = "agent"
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: kind + ".revoked", TargetType: k.Kind, TargetID: k.ID.String(), TargetName: k.Name,
				Reason: "creator " + name + " unassigned by the agency"}); err != nil {
				return err
			}
		}
	} else {
		keys, err := q.TeamMemberAPIKeys(ctx, store.TeamMemberAPIKeysParams{WorkspaceID: wsID, UserID: &m.UserID})
		if err != nil {
			return err
		}
		for _, k := range keys {
			if k.RevokedAt == nil {
				kept = append(kept, k.Name)
			}
		}
	}
	if _, err := q.TeamDeleteMember(ctx, store.TeamDeleteMemberParams{ID: m.ID, WorkspaceID: wsID}); err != nil {
		return err
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.unassigned", TargetType: "member", TargetID: m.ID.String(), TargetName: name,
		Before:   map[string]any{"role": m.RoleName, "source": m.Source},
		After:    map[string]any{"keys_revoked": revoked, "keys_kept": kept},
		Metadata: map[string]any{"user_id": m.UserID.String(), "permission": string(authz.AgencyManage)}})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
