package team

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// TeamMemberChangeInput changes a member's role and scope.
type TeamMemberChangeInput struct {
	RoleID string               `json:"roleId" format:"uuid"`
	Scope  *access.ScopeRequest `json:"scope,omitempty"`
}

// lastOwner is the 409 for removing, demoting or deactivating the last Owner.
func lastOwner(ctx context.Context, q *store.Queries, ws *store.Workspace, name string) *api.Problem {
	suggest := ""
	if rows, err := q.TeamListMembers(ctx, ws.ID); err == nil {
		for _, r := range rows {
			if r.Status == "active" && r.UserStatus == "active" && r.RoleKey == authz.RoleAdmin {
				suggest = displayName(r.UserName, r.Email)
				break
			}
		}
	}
	p := api.Err(http.StatusConflict, api.CodeLastOwner, mLastOwner.Text(i18n.LangFrom(ctx), i18n.Args{"name": name}))
	if suggest != "" {
		p = p.WithMeta("suggest", suggest)
	}
	return p
}

// outranks reports whether the actor holds every permission of the member's
// role: nobody can change, deactivate or remove someone with more access.
func outranks(p *authz.Principal, roleKey string, builtin bool, perms []string) (bool, []authz.Permission) {
	missing := p.Permissions.Missing(authz.Effective(roleKey, builtin, perms))
	return len(missing) == 0, missing
}

func outrankError(ctx context.Context, name string, missing []authz.Permission) *api.Problem {
	keys := make([]string, len(missing))
	for i, m := range missing {
		keys[i] = string(m)
	}
	pr := api.Err(http.StatusForbidden, api.CodeEscalation, mOutrank.Text(i18n.LangFrom(ctx), i18n.Args{"name": name, "permissions": strings.Join(keys, ", ")}))
	if len(missing) > 0 {
		pr.Permission = string(missing[0])
	}
	return pr
}

// ChangeRole gives a member another role and scope.
func (s *Service) ChangeRole(ctx context.Context, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID, in TeamMemberChangeInput) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.changeRole(ctx, store.New(tx), tx, ws, p, memberID, in)
	})
}

func (s *Service) changeRole(ctx context.Context, q *store.Queries, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID, in TeamMemberChangeInput) error {
	m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: memberID, WorkspaceID: ws.ID})
	if err != nil {
		return api.MapError(err, "Member")
	}
	name := displayName(m.UserName, m.Email)
	roleID, err := uuid.Parse(in.RoleID)
	if err != nil {
		return api.Invalid("body.roleId", mRolePick.Text(i18n.LangFrom(ctx), nil))
	}
	role, err := q.GetRole(ctx, roleID)
	if err != nil || role.WorkspaceID != ws.ID {
		return api.Invalid("body.roleId", mRoleUnknown.Text(i18n.LangFrom(ctx), nil))
	}
	newPerms := authz.Effective(role.Key, role.Builtin, role.Permissions)
	wasOwner := m.RoleBuiltin && m.RoleKey == authz.RoleOwner
	toOwner := role.Builtin && role.Key == authz.RoleOwner
	if wasOwner && !toOwner {
		if !p.Can(authz.WorkspaceTransfer) {
			return api.Forbidden(authz.WorkspaceTransfer, access.Helpers(ctx, q, ws.ID, authz.WorkspaceTransfer))
		}
		owners, err := q.TeamCountActiveOwners(ctx, ws.ID)
		if err != nil {
			return err
		}
		if owners <= 1 && m.Status == "active" {
			return lastOwner(ctx, q, ws, name)
		}
	}
	if toOwner && !wasOwner {
		if !p.Can(authz.WorkspaceTransfer) {
			return api.Err(http.StatusForbidden, api.CodeEscalation, mOwnerOnly.Text(i18n.LangFrom(ctx), nil))
		}
		if m.TotpEnabledAt == nil && m.PasskeyCount == 0 {
			if ids, _ := s.identities(ctx, []uuid.UUID{m.UserID}); !hasSSO(ids[m.UserID]) {
				return api.Conflict(mOwnerTwoFactor.Text(i18n.LangFrom(ctx), i18n.Args{"name": name}))
			}
		}
	}
	if !toOwner {
		if ok, reason := canGrantRole(i18n.LangFrom(ctx), p.Permissions, role.Key, role.Builtin, newPerms); !ok {
			pr := api.Err(http.StatusForbidden, api.CodeEscalation, reason)
			if mm := p.Permissions.Missing(newPerms); len(mm) > 0 {
				pr.Permission = string(mm[0])
			}
			return pr
		}
	}
	if ok, missing := outranks(p, m.RoleKey, m.RoleBuiltin, m.RolePermissions); !ok {
		return outrankError(ctx, name, missing)
	}
	lists, err := access.ListNames(ctx, q, ws.ID)
	if err != nil {
		return err
	}
	domains, err := q.TeamWorkspaceSenderDomains(ctx, ws.ID)
	if err != nil {
		return err
	}
	scope := access.ParseScope(m.Scope)
	if in.Scope != nil {
		if scope, err = in.Scope.Resolve(lists, domains); err != nil {
			return api.Invalid("body.scope", capitalize(err.Error())+".")
		}
	}
	if !scope.Within(p.Scope) {
		return api.Err(http.StatusForbidden, api.CodeEscalation, mScopeWithin.Text(i18n.LangFrom(ctx), nil))
	}
	before := access.View(access.ParseScope(m.Scope), lists, true)
	after := access.View(scope, lists, true)
	if m.RoleID == role.ID && before.Label == after.Label {
		return nil
	}
	if err := q.TeamSetMemberRole(ctx, store.TeamSetMemberRoleParams{ID: m.ID, WorkspaceID: ws.ID, RoleID: role.ID, Scope: access.ScopeJSON(scope)}); err != nil {
		return err
	}
	if newPerms.HasSensitive() {
		if err := workspaces.EnsureTwoFAGrace(ctx, tx, m.UserID); err != nil {
			return err
		}
	}
	summary := m.RoleName + " → " + role.Name
	if before.Label != after.Label {
		summary += " · scope " + orAll(before.Label) + " → " + orAll(after.Label)
	}
	return s.Audit.LogWith(ctx, q, audit.Event{
		WorkspaceID: &ws.ID, Action: "member.role_changed", TargetType: "member", TargetID: m.ID.String(), TargetName: name,
		Before:   map[string]any{"role": m.RoleName, "scope": orAll(before.Label)},
		After:    map[string]any{"role": role.Name, "scope": orAll(after.Label)},
		Metadata: map[string]any{"summary": summary, "permission": string(authz.TeamManage), "user_id": m.UserID.String()},
	})
}

func orAll(label string) string {
	if label == "" {
		return "all lists"
	}
	return label
}

func hasSSO(ids []access.Identity) bool {
	for _, id := range ids {
		if id.Kind == "sso" {
			return true
		}
	}
	return false
}

// TeamDeactivateInput deactivates a member.
type TeamDeactivateInput struct {
	// RevokeKeys revokes the API keys and agents the member created; false keeps them working.
	RevokeKeys bool   `json:"revokeKeys,omitempty"`
	Reason     string `json:"reason,omitempty" maxLength:"500" doc:"Recorded in the audit log"`
}

// TeamDeactivateResult says what happened.
type TeamDeactivateResult struct {
	SessionsRevoked int64    `json:"sessionsRevoked"`
	KeysRevoked     []string `json:"keysRevoked"`
}

// Deactivate keeps the member's history, signs them out everywhere and
// keeps or revokes the API keys they created.
func (s *Service) Deactivate(ctx context.Context, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID, in TeamDeactivateInput) (TeamDeactivateResult, error) {
	var res TeamDeactivateResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		res, err = s.deactivate(ctx, store.New(tx), ws, p, memberID, in)
		return err
	})
	return res, err
}

func (s *Service) deactivate(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID, in TeamDeactivateInput) (TeamDeactivateResult, error) {
	res := TeamDeactivateResult{KeysRevoked: []string{}}
	m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: memberID, WorkspaceID: ws.ID})
	if err != nil {
		return res, api.MapError(err, "Member")
	}
	name := displayName(m.UserName, m.Email)
	if m.Status == "deactivated" {
		return res, api.Conflict(name + " is already deactivated.")
	}
	if m.UserID == p.UserID {
		return res, api.Conflict("You can't deactivate yourself. Ask another member with team:manage, or leave the workspace from Account & security.")
	}
	if m.RoleBuiltin && m.RoleKey == authz.RoleOwner {
		owners, err := q.TeamCountActiveOwners(ctx, ws.ID)
		if err != nil {
			return res, err
		}
		if owners <= 1 {
			return res, lastOwner(ctx, q, ws, name)
		}
	}
	if ok, missing := outranks(p, m.RoleKey, m.RoleBuiltin, m.RolePermissions); !ok {
		return res, outrankError(ctx, name, missing)
	}
	if err := q.TeamSetMemberStatus(ctx, store.TeamSetMemberStatusParams{ID: m.ID, WorkspaceID: ws.ID, Status: "deactivated"}); err != nil {
		return res, err
	}
	if res.SessionsRevoked, err = q.TeamRevokeUserSessions(ctx, m.UserID); err != nil {
		return res, err
	}
	kept := []string{}
	if in.RevokeKeys {
		revoked, err := q.TeamRevokeKeysCreatedBy(ctx, store.TeamRevokeKeysCreatedByParams{WorkspaceID: ws.ID, UserID: &m.UserID})
		if err != nil {
			return res, err
		}
		for _, k := range revoked {
			res.KeysRevoked = append(res.KeysRevoked, k.Name)
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: keyAction(k.Kind) + ".revoked", TargetType: k.Kind, TargetID: k.ID.String(), TargetName: k.Name,
				Reason: "creator " + name + " deactivated"}); err != nil {
				return res, err
			}
		}
	} else {
		keys, err := q.TeamMemberAPIKeys(ctx, store.TeamMemberAPIKeysParams{WorkspaceID: ws.ID, UserID: &m.UserID})
		if err != nil {
			return res, err
		}
		for _, k := range keys {
			if k.RevokedAt == nil {
				kept = append(kept, k.Name)
			}
		}
	}
	summary := fmt.Sprintf("signed out of %d %s", res.SessionsRevoked, access.Plural(int(res.SessionsRevoked), "session", "sessions"))
	if len(res.KeysRevoked) > 0 {
		summary += " · revoked " + access.AndList(quoteAll(res.KeysRevoked))
	} else if len(kept) > 0 {
		summary += " · kept " + access.AndList(quoteAll(kept))
	}
	return res, s.Audit.LogWith(ctx, q, audit.Event{
		WorkspaceID: &ws.ID, Action: "member.deactivated", TargetType: "member", TargetID: m.ID.String(), TargetName: name, Reason: strings.TrimSpace(in.Reason),
		Before:   map[string]any{"status": "active", "role": m.RoleName},
		After:    map[string]any{"status": "deactivated", "keys_revoked": res.KeysRevoked, "keys_kept": kept},
		Metadata: map[string]any{"summary": summary, "permission": string(authz.TeamManage), "user_id": m.UserID.String()},
	})
}

func keyAction(kind string) string {
	if kind == "mcp_agent" {
		return "agent"
	}
	return "apikey"
}

func quoteAll(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "“" + n + "”"
	}
	return out
}

// Reactivate restores a deactivated member. A deactivated account (instance
// level) needs an instance admin.
func (s *Service) Reactivate(ctx context.Context, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: memberID, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "Member")
		}
		name := displayName(m.UserName, m.Email)
		if m.Status == "active" && m.UserStatus == "active" {
			return api.Conflict(name + " is already active.")
		}
		if ok, missing := outranks(p, m.RoleKey, m.RoleBuiltin, m.RolePermissions); !ok {
			return outrankError(ctx, name, missing)
		}
		if m.UserStatus != "active" {
			if !p.IsInstanceAdmin {
				return api.Conflict(fmt.Sprintf("%s's account is deactivated on this instance. Ask an instance admin to reactivate the account first.", name))
			}
			if err := q.SetUserStatus(ctx, store.SetUserStatusParams{ID: m.UserID, Status: "active"}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "user.reactivated", TargetType: "user", TargetID: m.UserID.String(), TargetName: m.Email}); err != nil {
				return err
			}
		}
		if err := q.TeamSetMemberStatus(ctx, store.TeamSetMemberStatusParams{ID: m.ID, WorkspaceID: ws.ID, Status: "active"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "member.reactivated", TargetType: "member", TargetID: m.ID.String(), TargetName: name,
			Before: map[string]any{"status": "deactivated"}, After: map[string]any{"status": "active", "role": m.RoleName},
			Metadata: map[string]any{"permission": string(authz.TeamManage), "user_id": m.UserID.String()}})
	})
}

// Remove deletes the membership. The audit log keeps naming the person; the
// API keys they created in this workspace are revoked.
func (s *Service) Remove(ctx context.Context, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: memberID, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "Member")
		}
		name := displayName(m.UserName, m.Email)
		if m.UserID == p.UserID {
			return api.Conflict("You can't remove yourself. Leave the workspace from Account & security instead.")
		}
		if m.RoleBuiltin && m.RoleKey == authz.RoleOwner && m.Status == "active" {
			owners, err := q.TeamCountActiveOwners(ctx, ws.ID)
			if err != nil {
				return err
			}
			if owners <= 1 {
				return lastOwner(ctx, q, ws, name)
			}
		}
		if ok, missing := outranks(p, m.RoleKey, m.RoleBuiltin, m.RolePermissions); !ok {
			return outrankError(ctx, name, missing)
		}
		revoked, err := q.TeamRevokeKeysCreatedBy(ctx, store.TeamRevokeKeysCreatedByParams{WorkspaceID: ws.ID, UserID: &m.UserID})
		if err != nil {
			return err
		}
		names := []string{}
		for _, k := range revoked {
			names = append(names, k.Name)
		}
		if _, err := q.TeamDeleteMember(ctx, store.TeamDeleteMemberParams{ID: m.ID, WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "member.removed", TargetType: "member", TargetID: m.ID.String(), TargetName: name,
			Before:   map[string]any{"email": m.Email, "role": m.RoleName, "status": m.Status},
			After:    map[string]any{"keys_revoked": names},
			Metadata: map[string]any{"permission": string(authz.TeamManage), "user_id": m.UserID.String()}})
	})
}

// TeamSignOutResult reports a forced sign-out.
type TeamSignOutResult struct {
	SessionsRevoked int64 `json:"sessionsRevoked"`
}

// SignOut ends every session of the member.
func (s *Service) SignOut(ctx context.Context, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID) (TeamSignOutResult, error) {
	var res TeamSignOutResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: memberID, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "Member")
		}
		name := displayName(m.UserName, m.Email)
		if ok, missing := outranks(p, m.RoleKey, m.RoleBuiltin, m.RolePermissions); !ok && m.UserID != p.UserID {
			return outrankError(ctx, name, missing)
		}
		if res.SessionsRevoked, err = q.TeamRevokeUserSessions(ctx, m.UserID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "member.signed_out", TargetType: "member", TargetID: m.ID.String(), TargetName: name,
			Metadata: map[string]any{"summary": fmt.Sprintf("%d %s ended", res.SessionsRevoked, access.Plural(int(res.SessionsRevoked), "session", "sessions")), "permission": string(authz.TeamManage)}})
	})
	return res, err
}

// TeamBulkInput applies one action to several members.
type TeamBulkInput struct {
	MemberIDs  []string `json:"memberIds" minItems:"1" maxItems:"200"`
	Action     string   `json:"action" enum:"change_role,require_2fa,deactivate"`
	RoleID     string   `json:"roleId,omitempty" doc:"For change_role"`
	RevokeKeys bool     `json:"revokeKeys,omitempty" doc:"For deactivate"`
	Reason     string   `json:"reason,omitempty" maxLength:"500"`
}

// TeamBulkResult reports how many members changed.
type TeamBulkResult struct {
	Changed int `json:"changed"`
	// Skipped names members already in the requested state.
	Skipped []string `json:"skipped"`
}

// Bulk changes roles, requires two-factor now or deactivates several members
// in one transaction: all of them or none.
func (s *Service) Bulk(ctx context.Context, ws *store.Workspace, p *authz.Principal, in TeamBulkInput) (TeamBulkResult, error) {
	res := TeamBulkResult{Skipped: []string{}}
	ids := make([]uuid.UUID, 0, len(in.MemberIDs))
	for _, raw := range in.MemberIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return res, api.Invalid("body.memberIds", "Unknown member.")
		}
		ids = append(ids, id)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		switch in.Action {
		case "change_role":
			if in.RoleID == "" {
				return api.Invalid("body.roleId", "Pick the role to give.")
			}
			for _, id := range ids {
				m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: id, WorkspaceID: ws.ID})
				if err != nil {
					return api.MapError(err, "Member")
				}
				if m.RoleID.String() == in.RoleID {
					res.Skipped = append(res.Skipped, displayName(m.UserName, m.Email))
					continue
				}
				if err := s.changeRole(ctx, q, tx, ws, p, id, TeamMemberChangeInput{RoleID: in.RoleID}); err != nil {
					return err
				}
				res.Changed++
			}
		case "require_2fa":
			var users []uuid.UUID
			for _, id := range ids {
				m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: id, WorkspaceID: ws.ID})
				if err != nil {
					return api.MapError(err, "Member")
				}
				name := displayName(m.UserName, m.Email)
				if m.TotpEnabledAt != nil || m.PasskeyCount > 0 || m.Status != "active" {
					res.Skipped = append(res.Skipped, name)
					continue
				}
				if ids, _ := s.identities(ctx, []uuid.UUID{m.UserID}); hasSSO(ids[m.UserID]) && !m.HasPassword {
					res.Skipped = append(res.Skipped, name)
					continue
				}
				users = append(users, m.UserID)
				if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "member.twofa_required", TargetType: "member", TargetID: m.ID.String(), TargetName: name,
					Before: map[string]any{"grace_until": m.TwofaGraceUntil}, After: map[string]any{"grace_until": "now"},
					Metadata: map[string]any{"summary": "two-factor required at next sign-in", "permission": string(authz.TeamManage)}}); err != nil {
					return err
				}
			}
			if len(users) > 0 {
				n, err := q.TeamEndTwoFAGrace(ctx, users)
				if err != nil {
					return err
				}
				res.Changed = int(n)
			}
		case "deactivate":
			for _, id := range ids {
				m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: id, WorkspaceID: ws.ID})
				if err != nil {
					return api.MapError(err, "Member")
				}
				if m.Status == "deactivated" {
					res.Skipped = append(res.Skipped, displayName(m.UserName, m.Email))
					continue
				}
				if _, err := s.deactivate(ctx, q, ws, p, id, TeamDeactivateInput{RevokeKeys: in.RevokeKeys, Reason: in.Reason}); err != nil {
					return err
				}
				res.Changed++
			}
		default:
			return api.Invalid("body.action", "Unknown action.")
		}
		return nil
	})
	return res, err
}
