// Package roles implements Roles & permissions (A5 §3.6, §5.9): the four
// read-only built-in roles, custom roles and the impact preview shown before
// saving. Nobody can add a permission they don't hold.
package roles

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
}

// RoleMemberRef is a member holding a role.
type RoleMemberRef struct {
	ID          string `json:"id" doc:"Membership id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarColor string `json:"avatarColor"`
	Status      string `json:"status" enum:"active,deactivated"`
}

// RoleSummary is one role in the roles list.
type RoleSummary struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Builtin     bool            `json:"builtin"`
	Permissions []string        `json:"permissions"`
	Sensitive   bool            `json:"sensitive"`
	Members     []RoleMemberRef `json:"members"`
	Active      int             `json:"active"`
	Deactivated int             `json:"deactivated"`
	Invited     int             `json:"invited"`
	CreatedBy   string          `json:"createdBy,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// RoleList is the roles screen.
type RoleList struct {
	Roles       []RoleSummary           `json:"roles"`
	Permissions []access.PermissionInfo `json:"permissions"`
	Groups      []string                `json:"groups"`
	// Held lists the caller's permissions: only these can be granted.
	Held      []string `json:"held"`
	CanManage bool     `json:"canManage"`
	// Helpers hold team:manage (for read-only explanations).
	Helpers []string `json:"helpers"`
}

func (s *Service) members(ctx context.Context, wsID uuid.UUID) (map[uuid.UUID][]store.RolesMembersRow, error) {
	rows, err := s.Q.RolesMembers(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID][]store.RolesMembersRow{}
	for _, r := range rows {
		out[r.RoleID] = append(out[r.RoleID], r)
	}
	return out, nil
}

// summary is a role for the roles screens; a built-in role's name and
// description read in the reader's language.
func summary(l i18n.Lang, r store.RolesListRow, members []store.RolesMembersRow, invited int64) RoleSummary {
	perms := authz.Effective(r.Key, r.Builtin, r.Permissions)
	title, about := r.Name, r.Description
	if r.Builtin {
		title, about = roleName(l, r.Key, r.Name), builtinDescription(l, r.Key, true, r.Description)
	}
	rs := RoleSummary{
		ID: r.ID.String(), Key: r.Key, Name: title, Description: about, Builtin: r.Builtin, Permissions: perms.Strings(),
		Sensitive: perms.HasSensitive(), Members: []RoleMemberRef{}, Invited: int(invited), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if !r.Builtin {
		rs.CreatedBy = r.CreatedByName
	}
	for _, m := range members {
		rs.Members = append(rs.Members, RoleMemberRef{ID: m.MemberID.String(), Name: name(m.Name, m.Email), Email: m.Email, AvatarColor: m.AvatarColor, Status: m.Status})
		if m.Status == "active" {
			rs.Active++
		} else {
			rs.Deactivated++
		}
	}
	return rs
}

func name(n, email string) string {
	if strings.TrimSpace(n) != "" {
		return n
	}
	return email
}

// List returns every role with its members and the permission catalogue.
func (s *Service) List(ctx context.Context, ws *store.Workspace, p *authz.Principal) (RoleList, error) {
	rows, err := s.Q.RolesList(ctx, ws.ID)
	if err != nil {
		return RoleList{}, err
	}
	members, err := s.members(ctx, ws.ID)
	if err != nil {
		return RoleList{}, err
	}
	inv, err := s.Q.RolesPendingInvites(ctx, ws.ID)
	if err != nil {
		return RoleList{}, err
	}
	invited := map[uuid.UUID]int64{}
	for _, i := range inv {
		invited[i.RoleID] = i.N
	}
	l := i18n.LangFrom(ctx)
	out := RoleList{Roles: []RoleSummary{}, Permissions: access.Catalog(l), Groups: access.Groups(l), Held: p.Permissions.Strings(), CanManage: p.Can(authz.TeamManage),
		Helpers: access.Helpers(ctx, s.Q, ws.ID, authz.TeamManage)}
	for _, r := range rows {
		out.Roles = append(out.Roles, summary(l, r, members[r.ID], invited[r.ID]))
	}
	return out, nil
}

// RoleDetail is one role for the editor.
type RoleDetail struct {
	Role        RoleSummary             `json:"role"`
	Permissions []access.PermissionInfo `json:"permissions"`
	Groups      []string                `json:"groups"`
	Held        []string                `json:"held"`
	CanManage   bool                    `json:"canManage"`
	Helpers     []string                `json:"helpers"`
}

// Get returns one role.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (RoleDetail, error) {
	r, err := s.Q.RolesGet(ctx, store.RolesGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return RoleDetail{}, err
	}
	members, err := s.members(ctx, ws.ID)
	if err != nil {
		return RoleDetail{}, err
	}
	inv, err := s.Q.RolesPendingInvites(ctx, ws.ID)
	if err != nil {
		return RoleDetail{}, err
	}
	var invited int64
	for _, i := range inv {
		if i.RoleID == r.ID {
			invited = i.N
		}
	}
	l := i18n.LangFrom(ctx)
	return RoleDetail{
		Role: summary(l, store.RolesListRow(r), members[r.ID], invited), Permissions: access.Catalog(l), Groups: access.Groups(l),
		Held: p.Permissions.Strings(), CanManage: p.Can(authz.TeamManage), Helpers: access.Helpers(ctx, s.Q, ws.ID, authz.TeamManage),
	}, nil
}

// RoleInput creates or updates a custom role.
type RoleInput struct {
	Name        string   `json:"name" minLength:"1" maxLength:"60"`
	Description string   `json:"description,omitempty" maxLength:"300"`
	Permissions []string `json:"permissions" maxItems:"100"`
}

// RoleCreated is returned after creating a role.
type RoleCreated struct {
	ID string `json:"id"`
}

var keyRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(name string) string {
	s := strings.Trim(keyRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 50 {
		s = strings.Trim(s[:50], "-")
	}
	if s == "" {
		s = "role"
	}
	return s
}

// validate checks the name and permission keys. It returns the expanded set.
func (s *Service) validate(ctx context.Context, q *store.Queries, ws *store.Workspace, in RoleInput, exclude uuid.UUID) (authz.Set, string, error) {
	l := i18n.LangFrom(ctx)
	n := strings.TrimSpace(in.Name)
	if n == "" {
		return nil, "", api.Invalid("body.name", msgs.M("problem.name").Text(l, nil))
	}
	for _, b := range authz.BuiltinRoles {
		if strings.EqualFold(b.Name, n) {
			return nil, "", api.Invalid("body.name", msgs.M("problem.builtinName").Text(l, i18n.Args{"name": b.Name}))
		}
	}
	taken, err := q.RolesNameTaken(ctx, store.RolesNameTakenParams{WorkspaceID: ws.ID, Name: n, ExcludeID: exclude})
	if err != nil {
		return nil, "", err
	}
	if taken {
		return nil, "", api.Invalid("body.name", msgs.M("problem.nameTaken").Text(l, i18n.Args{"name": n}))
	}
	perms := make([]authz.Permission, 0, len(in.Permissions))
	for _, k := range in.Permissions {
		pk := authz.Permission(k)
		if !authz.Valid(pk) {
			return nil, "", api.Invalid("body.permissions", msgs.M("problem.unknownPermission").Text(l, i18n.Args{"permission": k}))
		}
		if !authz.Grantable(pk) {
			return nil, "", api.Invalid("body.permissions", msgs.M("problem.ownerOnly").Text(l, i18n.Args{"permission": k}))
		}
		perms = append(perms, pk)
	}
	return authz.NewSet(perms...), n, nil
}

// escalation rejects permissions the caller adds without holding them. The
// permissions the caller asked for are named before the ones they imply.
func escalation(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, requested []string, added []authz.Permission) error {
	var missing []authz.Permission
	isAdded := func(k authz.Permission) bool { return slices.Contains(added, k) }
	for _, r := range requested {
		k := authz.Permission(r)
		if isAdded(k) && !p.Permissions.Has(k) && !slices.Contains(missing, k) {
			missing = append(missing, k)
		}
	}
	for _, a := range added {
		if !p.Permissions.Has(a) && !slices.Contains(missing, a) {
			missing = append(missing, a)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	pr := api.Err(http.StatusForbidden, api.CodeEscalation, access.CantGrant(i18n.LangFrom(ctx), missing[0], access.Helpers(ctx, q, ws.ID, missing[0])))
	pr.Permission = string(missing[0])
	return pr.WithMeta("missing", strs(missing))
}

// Create adds a custom role.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in RoleInput) (RoleCreated, error) {
	var out RoleCreated
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		set, n, err := s.validate(ctx, q, ws, in, uuid.Nil)
		if err != nil {
			return err
		}
		if err := escalation(ctx, q, ws, p, in.Permissions, set.Keys()); err != nil {
			return err
		}
		key := slug(n)
		for i := 2; ; i++ {
			taken, err := q.RolesKeyTaken(ctx, store.RolesKeyTakenParams{WorkspaceID: ws.ID, Key: key})
			if err != nil {
				return err
			}
			if !taken && authz.RoleOwner != key && authz.RoleAdmin != key && authz.RoleEditor != key && authz.RoleViewer != key {
				break
			}
			key = fmt.Sprintf("%s-%d", slug(n), i)
		}
		creator := p.UserID
		id, err := q.RolesCreate(ctx, store.RolesCreateParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Key: key, Name: n,
			Description: strings.TrimSpace(in.Description), Permissions: set.Strings(), CreatedBy: &creator})
		if err != nil {
			return err
		}
		out.ID = id.String()
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "role.created", TargetType: "role", TargetID: id.String(), TargetName: n,
			After:    map[string]any{"name": n, "description": strings.TrimSpace(in.Description), "permissions": set.Strings()},
			Metadata: audit.KeepText(map[string]any{"permission": string(authz.TeamManage)}, "summary", msgs.M("audit.permissions").Keep(i18n.Args{"count": len(set)}))})
	})
	return out, err
}

// Update changes a custom role. Built-in roles are read-only.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in RoleInput) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.RolesGet(ctx, store.RolesGetParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "Role")
		}
		if r.Builtin {
			l := i18n.LangFrom(ctx)
			return api.Conflict(msgs.M("problem.builtinEdit").Text(l, i18n.Args{"role": roleName(l, r.Key, r.Name)}))
		}
		set, n, err := s.validate(ctx, q, ws, in, r.ID)
		if err != nil {
			return err
		}
		before := authz.ParseSet(r.Permissions)
		added := before.Missing(set)
		removed := set.Missing(before)
		if err := escalation(ctx, q, ws, p, in.Permissions, added); err != nil {
			return err
		}
		if _, err := q.RolesUpdate(ctx, store.RolesUpdateParams{ID: r.ID, WorkspaceID: ws.ID, Name: n, Description: strings.TrimSpace(in.Description), Permissions: set.Strings()}); err != nil {
			return err
		}
		if set.HasSensitive() && len(added) > 0 {
			ms, err := q.RolesMembers(ctx, ws.ID)
			if err != nil {
				return err
			}
			for _, m := range ms {
				if m.RoleID == r.ID && m.Status == "active" {
					if err := workspaces.EnsureTwoFAGrace(ctx, tx, m.UserID); err != nil {
						return err
					}
				}
			}
		}
		// "renamed from “Writers” · +campaigns:send · −segments:sql": the
		// rename is words, the permission changes are keys.
		var parts []any
		if r.Name != n {
			parts = append(parts, msgs.M("audit.renamed").Keep(i18n.Args{"from": r.Name}))
		}
		for _, a := range added {
			parts = append(parts, "+"+string(a))
		}
		for _, rm := range removed {
			parts = append(parts, "−"+string(rm))
		}
		meta := audit.KeepSummary(map[string]any{"permission": string(authz.TeamManage)}, parts...)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "role.updated", TargetType: "role", TargetID: r.ID.String(), TargetName: n,
			Before:   map[string]any{"name": r.Name, "description": r.Description, "permissions": before.Strings()},
			After:    map[string]any{"name": n, "description": strings.TrimSpace(in.Description), "permissions": set.Strings()},
			Metadata: meta})
	})
}

// Delete removes a custom role that nobody holds.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.RolesGet(ctx, store.RolesGetParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "Role")
		}
		l := i18n.LangFrom(ctx)
		if r.Builtin {
			return api.Conflict(msgs.M("problem.builtinDelete").Text(l, i18n.Args{"role": roleName(l, r.Key, r.Name)}))
		}
		ms, err := q.RolesMembers(ctx, ws.ID)
		if err != nil {
			return err
		}
		var holders []string
		for _, m := range ms {
			if m.RoleID == r.ID {
				holders = append(holders, name(m.Name, m.Email))
			}
		}
		if len(holders) > 0 {
			return api.Conflict(msgs.M("problem.assigned").Text(l, i18n.Args{"role": r.Name, "count": len(holders), "names": i18n.AndList(l.Locale, holders)}))
		}
		n, err := q.RolesDelete(ctx, store.RolesDeleteParams{ID: r.ID, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.Conflict(msgs.M("problem.pendingInvite").Text(l, i18n.Args{"role": r.Name}))
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "role.deleted", TargetType: "role", TargetID: r.ID.String(), TargetName: r.Name,
			Before: map[string]any{"name": r.Name, "permissions": authz.ParseSet(r.Permissions).Strings()}, Metadata: map[string]any{"permission": string(authz.TeamManage)}})
	})
}

// RoleImpactInput is a proposed permission set, for a new role (no roleId) or an existing one.
type RoleImpactInput struct {
	RoleID      string   `json:"roleId,omitempty"`
	Permissions []string `json:"permissions" maxItems:"100"`
}

// RoleImpactMember is a member whose access changes.
type RoleImpactMember struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	AvatarColor string   `json:"avatarColor"`
	Gains       []string `json:"gains"`
	Loses       []string `json:"loses"`
	Effect      string   `json:"effect"`
	// TwoFactor tells that the change makes two-factor required for them:
	// with a grace period, or already enforced by their IdP.
	TwoFactor string `json:"twoFactor,omitempty" enum:"required,idp"`
}

// RoleImpactKey is an API key or agent whose access changes.
type RoleImpactKey struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Kind   string   `json:"kind" enum:"api_key,mcp_agent"`
	Loses  []string `json:"loses"`
	Effect string   `json:"effect"`
}

// RoleLockedPermission is one the caller can't grant.
type RoleLockedPermission struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

// RoleImpact is the preview shown before saving.
type RoleImpact struct {
	Added            []string `json:"added"`
	Removed          []string `json:"removed"`
	AddedSensitive   []string `json:"addedSensitive"`
	RemovedSensitive []string `json:"removedSensitive"`
	// Members holding the role (all of them, with what changes for each).
	Members []RoleImpactMember `json:"members"`
	// MembersAffected counts members whose permissions change.
	MembersAffected int             `json:"membersAffected"`
	Keys            []RoleImpactKey `json:"keys"`
	APIKeys         int             `json:"apiKeys" doc:"API keys that lose permissions"`
	Agents          int             `json:"agents" doc:"MCP agents that lose permissions"`
	// Sensitive is "None selected · two-factor not required" or the flagged keys.
	Sensitive string `json:"sensitive"`
	// Approvals explains what holders can do with sends.
	Approvals string `json:"approvals"`
	// PendingApprovals counts approval requests by holders that stay pending.
	PendingApprovals int `json:"pendingApprovals"`
	// PendingRequests are those requests, with the rules that still match.
	PendingRequests []RoleImpactPending `json:"pendingRequests"`
	// Summary is "4 members and 1 API key will lose campaigns:send." (empty without losses).
	Summary string                 `json:"summary"`
	Locked  []RoleLockedPermission `json:"locked"`
	// GraceDays is the two-factor grace period for new holders of ⚑ permissions.
	GraceDays int `json:"graceDays"`
}

// Impact computes who gains or loses what if the role had these permissions.
// Its sentences read in the request's language.
func (s *Service) Impact(ctx context.Context, ws *store.Workspace, p *authz.Principal, in RoleImpactInput) (RoleImpact, error) {
	l := i18n.LangFrom(ctx)
	perms := make([]authz.Permission, 0, len(in.Permissions))
	for _, k := range in.Permissions {
		if pk := authz.Permission(k); authz.Valid(pk) && authz.Grantable(pk) {
			perms = append(perms, pk)
		}
	}
	next := authz.NewSet(perms...)
	out := RoleImpact{Added: []string{}, Removed: []string{}, AddedSensitive: []string{}, RemovedSensitive: []string{}, Members: []RoleImpactMember{}, Keys: []RoleImpactKey{}, Locked: []RoleLockedPermission{}, PendingRequests: []RoleImpactPending{}}
	if st, err := s.Q.GetInstanceSettings(ctx); err == nil {
		out.GraceDays = int(st.TwofaGraceDays)
	}
	var prev authz.Set
	var roleID uuid.UUID
	if in.RoleID != "" {
		id, err := uuid.Parse(in.RoleID)
		if err != nil {
			return out, api.NotFound("Role")
		}
		r, err := s.Q.RolesGet(ctx, store.RolesGetParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return out, api.MapError(err, "Role")
		}
		roleID = r.ID
		prev = authz.Effective(r.Key, r.Builtin, r.Permissions)
	} else {
		prev = authz.Set{}
	}
	added := prev.Missing(next)
	removed := next.Missing(prev)
	for _, a := range added {
		out.Added = append(out.Added, string(a))
		if authz.IsSensitive(a) {
			out.AddedSensitive = append(out.AddedSensitive, string(a))
		}
	}
	for _, r := range removed {
		out.Removed = append(out.Removed, string(r))
		if authz.IsSensitive(r) {
			out.RemovedSensitive = append(out.RemovedSensitive, string(r))
		}
	}
	if roleID != uuid.Nil {
		ms, err := s.Q.RolesMembers(ctx, ws.ID)
		if err != nil {
			return out, err
		}
		var holders []uuid.UUID
		for _, m := range ms {
			if m.RoleID != roleID || m.Status != "active" {
				continue
			}
			holders = append(holders, m.UserID)
			im := RoleImpactMember{ID: m.MemberID.String(), Name: name(m.Name, m.Email), Email: m.Email, AvatarColor: m.AvatarColor, Gains: strs(added), Loses: strs(removed)}
			im.Effect, im.TwoFactor = memberEffect(l, next, added, removed, m.TotpEnabledAt != nil || m.PasskeyCount > 0, m.SsoIdentities > 0, out.GraceDays)
			if len(added)+len(removed) > 0 {
				out.MembersAffected++
			}
			out.Members = append(out.Members, im)
		}
		if len(holders) > 0 && len(removed) > 0 {
			ks, err := s.Q.RolesKeysOfMembers(ctx, store.RolesKeysOfMembersParams{WorkspaceID: ws.ID, UserIds: holders})
			if err != nil {
				return out, err
			}
			for _, k := range ks {
				has := authz.ParseSet(k.Permissions)
				var loses []authz.Permission
				for _, r := range removed {
					if has.Has(r) {
						loses = append(loses, r)
					}
				}
				if len(loses) == 0 {
					continue
				}
				out.Keys = append(out.Keys, RoleImpactKey{ID: k.ID.String(), Name: k.Name, Kind: k.Kind, Loses: strs(loses), Effect: keyEffect(l, loses)})
				if k.Kind == "mcp_agent" {
					out.Agents++
				} else {
					out.APIKeys++
				}
			}
		}
		if len(holders) > 0 {
			pending, err := s.pendingRequests(ctx, ws.ID, holders, next)
			if err != nil {
				return out, err
			}
			out.PendingRequests, out.PendingApprovals = pending, len(pending)
		}
	}
	var flagged []string
	for _, k := range next.Keys() {
		if authz.IsSensitive(k) {
			flagged = append(flagged, string(k))
		}
	}
	if len(flagged) == 0 {
		out.Sensitive = msgs.M("impact.sensitiveNone").Text(l, nil)
	} else {
		out.Sensitive = msgs.M("impact.sensitive").Text(l, i18n.Args{"permissions": joinKeys(flagged)})
	}
	out.Approvals = approvalsText(l, next)
	if len(removed) > 0 && (len(out.Members) > 0 || len(out.Keys) > 0) {
		var who []string
		if n := len(out.Members); n > 0 {
			who = append(who, msgs.M("count.members").Text(l, i18n.Args{"count": n}))
		}
		if out.APIKeys > 0 {
			who = append(who, msgs.M("count.apiKeys").Text(l, i18n.Args{"count": out.APIKeys}))
		}
		if out.Agents > 0 {
			who = append(who, msgs.M("count.agents").Text(l, i18n.Args{"count": out.Agents}))
		}
		out.Summary = msgs.M("impact.loses").Text(l, i18n.Args{"who": i18n.AndList(l.Locale, who), "permissions": joinKeys(out.Removed)})
	}
	for _, d := range authz.Defs {
		if d.OwnerOnly || p.Permissions.Has(d.Key) {
			continue
		}
		out.Locked = append(out.Locked, RoleLockedPermission{Key: string(d.Key), Reason: access.CantGrant(l, d.Key, access.Helpers(ctx, s.Q, ws.ID, d.Key))})
	}
	return out, nil
}

func strs(ps []authz.Permission) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = string(p)
	}
	return out
}

// memberEffect describes what changes for one holder, and whether
// two-factor becomes required for them ("required", or "idp" when their IdP
// already enforces it). Each part is a clause of its own; they are listed
// with semicolons.
func memberEffect(l i18n.Lang, next authz.Set, added, removed []authz.Permission, has2FA, sso bool, grace int) (string, string) {
	var parts []string
	twoFactor := ""
	if slices.Contains(removed, authz.CampaignsSend) && next.Has(authz.CampaignsEdit) {
		parts = append(parts, msgs.M("effect.stillEdit").Text(l, nil))
	} else if len(removed) > 0 {
		var labels []string
		for _, r := range removed {
			if _, ok := authz.Lookup(r); ok {
				labels = append(labels, lowerFirst(authz.LabelIn(l, r)))
			}
		}
		parts = append(parts, msgs.M("effect.noLonger").Text(l, i18n.Args{"actions": strings.Join(labels, ", ")}))
	}
	sensitiveAdded := false
	for _, a := range added {
		if authz.IsSensitive(a) {
			sensitiveAdded = true
		}
	}
	if len(added) > 0 {
		if sensitiveAdded && !has2FA {
			if sso {
				twoFactor = "idp"
				parts = append(parts, msgs.M("effect.twoFactorIdp").Text(l, nil))
			} else {
				twoFactor = "required"
				parts = append(parts, msgs.M("effect.twoFactorGrace").Text(l, i18n.Args{"days": grace}))
			}
		} else {
			parts = append(parts, msgs.M("effect.gains").Text(l, i18n.Args{"permissions": joinKeys(strs(added))}))
		}
	}
	if len(parts) == 0 {
		return msgs.M("effect.same").Text(l, nil), ""
	}
	return strings.Join(parts, "; "), twoFactor
}

func keyEffect(l i18n.Lang, loses []authz.Permission) string {
	if slices.Contains(loses, authz.CampaignsSend) {
		return msgs.M("effect.keyAutomations").Text(l, nil)
	}
	return msgs.M("effect.keyCalls").Text(l, i18n.Args{"permissions": joinKeys(strs(loses))})
}

func approvalsText(l i18n.Lang, p authz.Set) string {
	key := "approvals.none"
	switch {
	case p.Has(authz.CampaignsApprove) && p.Has(authz.CampaignsSend):
		key = "approvals.sendApprove"
	case p.Has(authz.CampaignsApprove):
		key = "approvals.approve"
	case p.Has(authz.CampaignsSend):
		key = "approvals.send"
	case p.Has(authz.CampaignsEdit):
		key = "approvals.request"
	}
	return msgs.M(key).Text(l, nil)
}
