// Package team manages the people of a workspace (A5 §3.5, §5.8): the
// members list with role, scope, sign-in methods and two-factor status,
// invitations by email or copyable link, member detail with effective
// permissions and where each comes from, role and scope changes, bulk
// actions, deactivation (keeps history, signs out everywhere, keep or revoke
// API keys) and removal. The last Owner can't be removed or demoted, and
// nobody can grant a permission they don't hold.
package team

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the team domain dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	// Mail sends invitations through the System stream; mailer.ErrNoSystemProvider
	// switches to copyable links.
	Mail mailer.Sender
	// PublicURL returns the externally reachable base URL for invite links.
	PublicURL func(context.Context) string
	Now       func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// RoleRef names a member's role.
type RoleRef struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
}

// TeamMember is one row of the members list.
type TeamMember struct {
	ID          string           `json:"id" doc:"Membership id"`
	UserID      string           `json:"userId"`
	Name        string           `json:"name"`
	Email       string           `json:"email"`
	AvatarColor string           `json:"avatarColor"`
	Role        RoleRef          `json:"role"`
	Scope       access.ScopeView `json:"scope"`
	// Notes follow the role: "instance admin", "break-glass".
	Notes        []string              `json:"notes"`
	SignIn       []access.SignInMethod `json:"signIn"`
	TwoFA        access.TwoFA          `json:"twoFA"`
	LastActiveAt *time.Time            `json:"lastActiveAt,omitempty"`
	Status       string                `json:"status" enum:"active,deactivated"`
	// Deactivated members of the instance (not only of this workspace).
	AccountDeactivated bool       `json:"accountDeactivated"`
	DeactivatedAt      *time.Time `json:"deactivatedAt,omitempty"`
	JoinedAt           time.Time  `json:"joinedAt"`
	// Source is where the membership came from: invite, sso_jit, sso_group, install, transfer, self.
	Source string `json:"source"`
	// You is true for the signed-in person.
	You bool `json:"you"`
	// LastOwner is true when this is the only active Owner.
	LastOwner bool `json:"lastOwner"`
}

// TeamCounts feeds "6 members · 1 invited".
type TeamCounts struct {
	Members     int `json:"members"`
	Active      int `json:"active"`
	Deactivated int `json:"deactivated"`
	Invited     int `json:"invited"`
}

// TeamMemberList is the members list with the caller's abilities.
type TeamMemberList struct {
	Items  []TeamMember `json:"items"`
	Counts TeamCounts   `json:"counts"`
	// Roles available in the role filter and the bulk role select.
	Roles []RoleRef `json:"roles"`
	// CanInvite / CanManage reflect the caller's team:invite and team:manage.
	CanInvite bool `json:"canInvite"`
	CanManage bool `json:"canManage"`
	// Reason explains read-only access ("Inviting needs the team:invite permission…").
	Reason string `json:"reason,omitempty"`
	// Helpers are members who hold team:manage.
	Helpers []string `json:"helpers"`
}

// ListFilter narrows the members list.
type ListFilter struct {
	Query  string
	Role   string // role id or key
	Status string // active | deactivated
	TwoFA  string // on | off | idp | required
}

type memberRow struct {
	store.TeamGetMemberRow
}

func fromListRow(r store.TeamListMembersRow) memberRow {
	return memberRow{store.TeamGetMemberRow(r)}
}

// policy returns the instance's two-factor enforcement.
func (s *Service) policy(ctx context.Context) store.InstanceSetting {
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return store.InstanceSetting{TwofaEnforcement: "sensitive", TwofaGraceDays: 7}
	}
	return st
}

func (s *Service) identities(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]access.Identity, error) {
	rows, err := s.Q.TeamIdentities(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID][]access.Identity{}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], access.Identity{Kind: r.Kind, ProviderName: r.ProviderName, ProviderPreset: r.ProviderPreset})
	}
	return out, nil
}

func (s *Service) toMember(r memberRow, ids []access.Identity, lists map[uuid.UUID]string, pol store.InstanceSetting, caller uuid.UUID, owners int64) TeamMember {
	perms := authz.Effective(r.RoleKey, r.RoleBuiltin, r.RolePermissions)
	ssoOnly := false
	if !r.HasPassword && r.PasskeyCount == 0 {
		for _, id := range ids {
			if id.Kind == "sso" {
				ssoOnly = true
			}
		}
	}
	required := pol.TwofaEnforcement == "everyone" || (pol.TwofaEnforcement == "sensitive" && perms.HasSensitive())
	m := TeamMember{
		ID: r.ID.String(), UserID: r.UserID.String(), Name: displayName(r.UserName, r.Email), Email: r.Email, AvatarColor: r.AvatarColor,
		Role:   RoleRef{ID: r.RoleID.String(), Key: r.RoleKey, Name: r.RoleName, Builtin: r.RoleBuiltin},
		Scope:  access.View(access.ParseScope(r.Scope), lists, true),
		Notes:  []string{},
		SignIn: access.SignInMethods(r.HasPassword, r.PasskeyCount, ids),
		TwoFA: access.TwoFAStatus(access.TwoFAInput{
			TOTP: r.TotpEnabledAt != nil, Passkeys: r.PasskeyCount, SSOOnly: ssoOnly,
			GraceUntil: r.TwofaGraceUntil, Required: required && r.TwofaGraceUntil != nil, Now: s.now(),
		}),
		Status: r.Status, DeactivatedAt: r.DeactivatedAt, JoinedAt: r.JoinedAt, Source: r.Source,
		AccountDeactivated: r.UserStatus != "active",
		You:                r.UserID == caller,
	}
	if m.AccountDeactivated && m.Status == "active" {
		m.Status = "deactivated"
		if m.DeactivatedAt == nil {
			m.DeactivatedAt = r.UserLastActiveAt
		}
	}
	m.LastActiveAt = r.MemberLastActiveAt
	if r.UserLastActiveAt != nil && (m.LastActiveAt == nil || r.UserLastActiveAt.After(*m.LastActiveAt)) && m.Status == "active" {
		m.LastActiveAt = r.UserLastActiveAt
	}
	if r.IsInstanceAdmin {
		m.Notes = append(m.Notes, "instance admin")
	}
	if r.BreakGlass {
		m.Notes = append(m.Notes, "break-glass")
	}
	if m.Status != "active" {
		m.SignIn = []access.SignInMethod{}
		m.TwoFA = access.TwoFA{State: "off"}
	}
	m.LastOwner = r.RoleKey == authz.RoleOwner && r.RoleBuiltin && r.Status == "active" && owners <= 1
	return m
}

func displayName(name, email string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return email
}

// List returns the members of the workspace.
func (s *Service) List(ctx context.Context, ws *store.Workspace, p *authz.Principal, f ListFilter) (TeamMemberList, error) {
	rows, err := s.Q.TeamListMembers(ctx, ws.ID)
	if err != nil {
		return TeamMemberList{}, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.UserID
	}
	idents, err := s.identities(ctx, ids)
	if err != nil {
		return TeamMemberList{}, err
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return TeamMemberList{}, err
	}
	owners, err := s.Q.TeamCountActiveOwners(ctx, ws.ID)
	if err != nil {
		return TeamMemberList{}, err
	}
	invites, err := s.Q.TeamListInvitations(ctx, ws.ID)
	if err != nil {
		return TeamMemberList{}, err
	}
	pol := s.policy(ctx)
	out := TeamMemberList{Items: []TeamMember{}, Roles: []RoleRef{}, Helpers: access.Helpers(ctx, s.Q, ws.ID, authz.TeamManage)}
	q := strings.ToLower(strings.TrimSpace(f.Query))
	for _, r := range rows {
		m := s.toMember(fromListRow(r), idents[r.UserID], lists, pol, p.UserID, owners)
		out.Counts.Members++
		if m.Status == "active" {
			out.Counts.Active++
		} else {
			out.Counts.Deactivated++
		}
		if q != "" && !strings.Contains(strings.ToLower(m.Name), q) && !strings.Contains(m.Email, q) {
			continue
		}
		if f.Role != "" && f.Role != m.Role.ID && f.Role != m.Role.Key {
			continue
		}
		if f.Status != "" && f.Status != m.Status {
			continue
		}
		if f.TwoFA != "" && f.TwoFA != m.TwoFA.State {
			continue
		}
		out.Items = append(out.Items, m)
	}
	for _, inv := range invites {
		if inv.ExpiresAt.After(s.now()) {
			out.Counts.Invited++
		}
	}
	roles, err := s.Q.ListRoles(ctx, ws.ID)
	if err != nil {
		return TeamMemberList{}, err
	}
	for _, r := range roles {
		out.Roles = append(out.Roles, RoleRef{ID: r.ID.String(), Key: r.Key, Name: r.Name, Builtin: r.Builtin})
	}
	out.CanInvite = p.Can(authz.TeamInvite)
	out.CanManage = p.Can(authz.TeamManage)
	if !out.CanInvite {
		out.Reason = "You can view the team. Inviting needs the team:invite permission and changing roles needs team:manage."
		if len(out.Helpers) > 0 {
			out.Reason += " Ask " + access.OrList(out.Helpers) + "."
		}
	} else if !out.CanManage {
		out.Reason = "You can invite people with roles you hold. Changing roles needs team:manage."
		if len(out.Helpers) > 0 {
			out.Reason += " Ask " + access.OrList(out.Helpers) + "."
		}
	}
	return out, nil
}

// EffectivePermission is one row of the member detail's permission table.
type EffectivePermission struct {
	Key       string `json:"key"`
	Group     string `json:"group"`
	Label     string `json:"label"`
	Sensitive bool   `json:"sensitive"`
	Granted   bool   `json:"granted"`
	// Source is "Role · Content Writer", "SSO group mapping · Marketing" or "not granted".
	Source string `json:"source"`
	// IncludedBy is the permission that brings this one in ("campaigns:edit").
	IncludedBy string `json:"includedBy,omitempty"`
	// Scope is "all lists", a list label, or "" when not granted.
	Scope string `json:"scope"`
}

// MemberAPIKey is a key or agent the member created.
type MemberAPIKey struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind" enum:"api_key,mcp_agent"`
	Name        string     `json:"name"`
	Masked      string     `json:"masked"`
	Permissions []string   `json:"permissions"`
	Status      string     `json:"status" enum:"active,expired,revoked"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

// MemberApproval is an approval the member gave or requested.
type MemberApproval struct {
	Campaign   string    `json:"campaign"`
	CampaignID string    `json:"campaignId,omitempty"`
	Status     string    `json:"status" doc:"approve, request_changes, reject (given) or the request status (requested)"`
	At         time.Time `json:"at"`
}

// MemberActivity is one recent audit entry by the member.
type MemberActivity struct {
	ID         string    `json:"id"`
	OccurredAt time.Time `json:"occurredAt"`
	Action     string    `json:"action"`
	Target     string    `json:"target"`
	Result     string    `json:"result" enum:"allowed,denied,error"`
	Reason     string    `json:"reason,omitempty"`
}

// TeamMemberDetail is the member page (A5 §5.8 member detail).
type TeamMemberDetail struct {
	Member      TeamMember            `json:"member"`
	Permissions []EffectivePermission `json:"permissions"`
	Groups      []string              `json:"groups"`
	APIKeys     []MemberAPIKey        `json:"apiKeys"`
	// KeysNote explains an empty list ("Creating API keys needs apikeys:manage…").
	KeysNote           string           `json:"keysNote,omitempty"`
	ApprovalsGiven     []MemberApproval `json:"approvalsGiven"`
	ApprovalsRequested []MemberApproval `json:"approvalsRequested"`
	CanApprove         bool             `json:"canApprove"`
	Sessions           int64            `json:"sessions"`
	// SSODomain is the email domain used with SSO ("acme.io").
	SSODomain string           `json:"ssoDomain,omitempty"`
	Activity  []MemberActivity `json:"activity"`
	CanManage bool             `json:"canManage"`
	// Helpers are members who hold team:manage.
	Helpers []string `json:"helpers"`
}

// Get returns the member detail.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, p *authz.Principal, memberID uuid.UUID) (TeamMemberDetail, error) {
	r, err := s.Q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: memberID, WorkspaceID: ws.ID})
	if err != nil {
		return TeamMemberDetail{}, err
	}
	idents, err := s.identities(ctx, []uuid.UUID{r.UserID})
	if err != nil {
		return TeamMemberDetail{}, err
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return TeamMemberDetail{}, err
	}
	owners, err := s.Q.TeamCountActiveOwners(ctx, ws.ID)
	if err != nil {
		return TeamMemberDetail{}, err
	}
	pol := s.policy(ctx)
	m := s.toMember(memberRow{r}, idents[r.UserID], lists, pol, p.UserID, owners)
	perms := authz.Effective(r.RoleKey, r.RoleBuiltin, r.RolePermissions)
	scope := access.ParseScope(r.Scope)
	scopeLabel := "all lists"
	if !scope.IsZero() {
		scopeLabel = strings.TrimSuffix(access.View(scope, lists, false).Label, " only")
	}
	source := "Role · " + r.RoleName
	if r.Source == "sso_group" {
		source = "SSO group mapping · " + r.RoleName
	}
	out := TeamMemberDetail{
		Member: m, Permissions: []EffectivePermission{}, Groups: access.Groups(),
		APIKeys: []MemberAPIKey{}, ApprovalsGiven: []MemberApproval{}, ApprovalsRequested: []MemberApproval{}, Activity: []MemberActivity{},
		CanApprove: perms.Has(authz.CampaignsApprove), CanManage: p.Can(authz.TeamManage),
		Helpers: access.Helpers(ctx, s.Q, ws.ID, authz.TeamManage),
	}
	for _, d := range authz.Defs {
		ep := EffectivePermission{Key: string(d.Key), Group: d.Group, Label: d.Label, Sensitive: d.Sensitive, Source: "not granted"}
		if perms.Has(d.Key) && m.Status == "active" {
			ep.Granted = true
			ep.Source = source
			ep.Scope = scopeLabel
			if by := access.ImpliedBy(perms, d.Key); by != "" {
				ep.IncludedBy = string(by)
			}
		}
		out.Permissions = append(out.Permissions, ep)
	}
	keys, err := s.Q.TeamMemberAPIKeys(ctx, store.TeamMemberAPIKeysParams{WorkspaceID: ws.ID, UserID: &r.UserID})
	if err != nil {
		return TeamMemberDetail{}, err
	}
	for _, k := range keys {
		st := "active"
		switch {
		case k.RevokedAt != nil:
			st = "revoked"
		case k.ExpiresAt != nil && !k.ExpiresAt.After(s.now()):
			st = "expired"
		}
		if st == "revoked" {
			continue
		}
		out.APIKeys = append(out.APIKeys, MemberAPIKey{ID: k.ID.String(), Kind: k.Kind, Name: k.Name, Masked: k.Prefix, Permissions: nonNil(k.Permissions), Status: st, LastUsedAt: k.LastUsedAt})
	}
	if len(out.APIKeys) == 0 {
		if !perms.Has(authz.APIKeysManage) {
			out.KeysNote = "None. Creating API keys needs apikeys:manage, which this role doesn't hold."
		} else {
			out.KeysNote = "None yet."
		}
	}
	given, err := s.Q.TeamApprovalsGiven(ctx, store.TeamApprovalsGivenParams{WorkspaceID: ws.ID, UserID: &r.UserID})
	if err != nil {
		return TeamMemberDetail{}, err
	}
	for _, g := range given {
		out.ApprovalsGiven = append(out.ApprovalsGiven, MemberApproval{Campaign: g.CampaignName, CampaignID: idString(g.CampaignID), Status: g.Decision, At: g.CreatedAt})
	}
	req, err := s.Q.TeamApprovalsRequested(ctx, store.TeamApprovalsRequestedParams{WorkspaceID: ws.ID, UserID: &r.UserID})
	if err != nil {
		return TeamMemberDetail{}, err
	}
	for _, a := range req {
		out.ApprovalsRequested = append(out.ApprovalsRequested, MemberApproval{Campaign: a.CampaignName, CampaignID: idString(a.CampaignID), Status: a.Status, At: a.RequestedAt})
	}
	if out.Sessions, err = s.Q.TeamActiveSessionCount(ctx, r.UserID); err != nil {
		return TeamMemberDetail{}, err
	}
	for _, id := range idents[r.UserID] {
		if id.Kind == "sso" {
			if i := strings.LastIndexByte(r.Email, '@'); i > 0 {
				out.SSODomain = r.Email[i+1:]
			}
		}
	}
	acts, err := s.Q.TeamMemberActivity(ctx, store.TeamMemberActivityParams{WorkspaceID: &ws.ID, UserID: &r.UserID, Lim: 8})
	if err != nil {
		return TeamMemberDetail{}, err
	}
	for _, a := range acts {
		out.Activity = append(out.Activity, MemberActivity{ID: a.ID.String(), OccurredAt: a.OccurredAt, Action: a.Action, Target: activityTarget(a), Result: a.Result, Reason: a.Reason})
	}
	return out, nil
}

// activityTarget is "November Product Update · 47,912 recipients": the
// target name plus the entry's short detail when there is one.
func activityTarget(a store.TeamMemberActivityRow) string {
	var meta struct {
		Detail string `json:"detail"`
	}
	_ = json.Unmarshal(a.Metadata, &meta)
	t := a.TargetName
	if t == "" {
		t = a.TargetType
	}
	if meta.Detail != "" {
		if t != "" {
			return t + " · " + meta.Detail
		}
		return meta.Detail
	}
	return t
}

func idString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return slices.Clone(s)
}
