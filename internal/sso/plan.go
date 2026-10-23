package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// identity is what an IdP said about a person, after claim mapping.
type identity struct {
	Kind         string // sso, google, microsoft, github
	ProviderID   *uuid.UUID
	ProviderName string // "Google Workspace", "GitHub"
	Subject      string
	Email        string
	// EmailVerified is nil when the IdP didn't say.
	EmailVerified *bool
	Name          string
	Username      string
	Groups        []string
	// GroupsClaim: the groups claim was present (even if empty).
	GroupsClaim bool
	// GroupsOverage: Entra sent an overage link instead of the groups.
	GroupsOverage bool
	AMR           []string
	Details       map[string]any
	Claims        map[string]any
}

// mfaMethods are amr values (RFC 8176) that mean a second factor was used.
var mfaMethods = []string{"mfa", "otp", "hwk", "swk", "sc", "fpt", "face", "iris", "retina", "sms", "tel", "mca"}

func (id identity) amrMFA() bool {
	for _, a := range id.AMR {
		if slices.Contains(mfaMethods, strings.ToLower(a)) {
			return true
		}
	}
	return false
}

// extract maps raw claims through the provider's claim names.
func extract(p store.SsoProvider, claims map[string]any) identity {
	cm := claimsOf(p)
	id := identity{Kind: "sso", ProviderID: &p.ID, ProviderName: p.Name, Claims: claims, Details: map[string]any{}}
	id.Subject = claimString(claims, "sub")
	id.Email = strings.ToLower(strings.TrimSpace(claimString(claims, cm.Email)))
	if id.Email == "" {
		// Entra ID without the optional email claim: the UPN is an email
		// at a verified tenant domain.
		if pu := claimString(claims, "preferred_username"); strings.Contains(pu, "@") {
			id.Email = strings.ToLower(strings.TrimSpace(pu))
		}
	}
	if v, ok := claimBool(claims, cm.EmailVerified); ok {
		id.EmailVerified = &v
	}
	id.Name = strings.TrimSpace(claimString(claims, cm.Name))
	if id.Name == "" {
		id.Name = strings.TrimSpace(claimString(claims, "given_name") + " " + claimString(claims, "family_name"))
	}
	id.Username = claimString(claims, "preferred_username")
	if g, ok := claims[cm.Groups]; ok {
		id.GroupsClaim = true
		id.Groups = claimStrings(g)
	} else if names, ok := claims["_claim_names"].(map[string]any); ok {
		if _, over := names[cm.Groups]; over {
			id.GroupsOverage = true
		}
	}
	id.AMR = claimStrings(claims["amr"])
	return id
}

func claimString(c map[string]any, name string) string {
	if name == "" {
		return ""
	}
	switch v := c[name].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case json.Number:
		return v.String()
	}
	return ""
}

func claimBool(c map[string]any, name string) (bool, bool) {
	if name == "" {
		return false, false
	}
	switch v := c[name].(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(v) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func claimStrings(v any) []string {
	out := []string{}
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			switch s := x.(type) {
			case string:
				out = append(out, s)
			case float64:
				out = append(out, fmt.Sprintf("%.0f", s))
			}
		}
	case []string:
		out = append(out, t...)
	case string:
		out = append(out, strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ' ' })...)
	}
	return out
}

// memberChange is one membership a sign-in creates or changes.
type memberChange struct {
	WorkspaceID   uuid.UUID
	WorkspaceName string
	FromKey       string // "" for a new membership
	FromName      string
	FromSource    string
	To            store.SSOWorkspaceRolesRow
	Scope         authz.Scope
	Source        string // sso_jit or sso_group
	Group         string // the mapping's label ("Marketing")
	// Kept explains why nothing changes (Owner, deactivated, same role).
	Kept string
	// MemberID of the existing membership.
	MemberID *uuid.UUID
	// LeftGroup is set when the person is no longer in the group whose
	// mapping granted the membership ("Marketing"): the membership ends
	// (Remove), or, in the JIT workspace, falls back to the default role.
	LeftGroup string
	Remove    bool
}

// grant is the role a sign-in gives in a workspace: the best matching
// group mapping (the role with most permissions) or the JIT default.
type grant struct {
	role  store.SSOWorkspaceRolesRow
	scope authz.Scope
	group string
}

// signInPlan is what a sign-in would do; the test sign-in shows it and a
// real sign-in applies it.
type signInPlan struct {
	User     *store.User
	Identity *store.UserIdentity
	// Link: an existing account with this verified email gets the identity.
	Link bool
	// Create: just-in-time account.
	Create  bool
	Reject  *flowError
	Changes []memberChange
	MFA     bool
	// MFANote says how two-factor is satisfied; MFAAmr: it is about the
	// amr claim (shown in code type).
	MFANote i18n.Text
	MFAAmr  bool
	// GroupsMissing: the provider has mappings but the IdP sent no groups.
	GroupsMissing bool
	Mappings      int
}

// plan resolves the account an SSO identity signs in to and the roles its
// groups map to. It writes nothing.
func (s *Service) plan(ctx context.Context, q *store.Queries, p store.SsoProvider, id identity) (signInPlan, error) {
	pl := signInPlan{}
	maps := mappingsOf(p)
	pl.Mappings = len(maps)
	mode := twoFAMode(p)
	switch {
	case mode == "trusted":
		pl.MFA, pl.MFANote = true, trustedTwoFA(p.Preset)
	case id.amrMFA():
		pl.MFA, pl.MFANote, pl.MFAAmr = true, keep("mfa.amr", nil), true
	case len(id.AMR) > 0:
		pl.MFANote, pl.MFAAmr = keep("mfa.amrWithout", i18n.Args{"amr": strings.Join(id.AMR, "+")}), true
	default:
		pl.MFANote = keep("mfa.noAmr", nil)
	}
	if id.Subject == "" {
		pl.Reject = failure("no_subject", "flow.noSubject", nil)
		return pl, nil
	}
	if id.Email == "" {
		pl.Reject = failure("no_email", "flow.noEmail", i18n.Args{"name": p.Name, "claim": claimsOf(p).Email})
		return pl, nil
	}
	domain := emailDomain(id.Email)
	if len(p.AllowedDomains) > 0 && !slices.Contains(p.AllowedDomains, domain) {
		pl.Reject = failure("domain_not_allowed", "flow.domain", i18n.Args{"name": p.Name}).noAccount(id.Email)
		return pl, nil
	}
	verified := false
	if id.EmailVerified != nil {
		verified = *id.EmailVerified
		if !verified {
			pl.Reject = failure("email_not_verified", "flow.notVerified", i18n.Args{"name": p.Name, "email": id.Email}).about(id.Email)
			return pl, nil
		}
	} else {
		// Missing email_verified: trusted only for the provider's own domains.
		verified = len(p.AllowedDomains) > 0 && slices.Contains(p.AllowedDomains, domain)
	}

	ident, err := q.GetIdentity(ctx, store.GetIdentityParams{Kind: "sso", SsoProviderID: &p.ID, Subject: id.Subject})
	switch {
	case err == nil:
		pl.Identity = &ident
		u, err := q.GetUser(ctx, ident.UserID)
		if err != nil {
			return pl, err
		}
		pl.User = &u
	case isNoRows(err):
		if verified {
			if u, err := q.GetUserByEmail(ctx, id.Email); err == nil {
				pl.User, pl.Link = &u, true
			} else if !isNoRows(err) {
				return pl, err
			}
		}
	default:
		return pl, err
	}
	if pl.User != nil && pl.User.Status != "active" {
		pl.Reject = failure("account_deactivated", "flow.deactivated", i18n.Args{"email": id.Email}).about(id.Email)
		return pl, nil
	}
	if pl.User == nil {
		st, err := q.GetInstanceSettings(ctx)
		if err != nil {
			return pl, err
		}
		switch {
		case !p.JitEnabled:
			pl.Reject = failure("no_account", "flow.noAccount", i18n.Args{"email": id.Email}).noAccount(id.Email)
		case !verified:
			pl.Reject = failure("email_not_verified", "flow.unconfirmed", i18n.Args{"name": p.Name, "email": id.Email}).noAccount(id.Email)
		case len(st.JitDomainAllowlist) > 0 && !slices.Contains(st.JitDomainAllowlist, domain):
			pl.Reject = failure("domain_not_allowed", "flow.allowlist", nil).noAccount(id.Email)
		default:
			pl.Create = true
		}
		if pl.Reject != nil {
			return pl, nil //nolint:nilerr // a rejection is part of the plan, not a failure
		}
	}

	roles, err := q.SSOWorkspaceRoles(ctx)
	if err != nil {
		return pl, err
	}
	role := func(ws uuid.UUID, key string) (store.SSOWorkspaceRolesRow, bool) {
		for _, r := range roles {
			if r.WorkspaceID == ws && r.RoleKey == key {
				return r, true
			}
		}
		return store.SSOWorkspaceRolesRow{}, false
	}
	// The best matching mapping per workspace: the role with most permissions.
	targets := map[uuid.UUID]grant{}
	order := []uuid.UUID{}
	if len(maps) > 0 && !id.GroupsClaim {
		pl.GroupsMissing = true
	}
	if id.GroupsClaim {
		for _, m := range maps {
			if !slices.Contains(id.Groups, m.Group) {
				continue
			}
			r, ok := role(m.WorkspaceID, m.RoleKey)
			if !ok || r.RoleKey == authz.RoleOwner {
				continue
			}
			label := m.Label
			if label == "" {
				label = m.Group
			}
			cur, seen := targets[m.WorkspaceID]
			if !seen {
				order = append(order, m.WorkspaceID)
			}
			if !seen || len(authz.Effective(r.RoleKey, r.Builtin, r.Permissions)) > len(authz.Effective(cur.role.RoleKey, cur.role.Builtin, cur.role.Permissions)) {
				targets[m.WorkspaceID] = grant{role: r, scope: m.Scope, group: label}
			}
		}
	}
	if pl.Create && p.JitEnabled && p.JitWorkspaceID != nil && p.JitRoleKey != nil {
		if _, mapped := targets[*p.JitWorkspaceID]; !mapped {
			if r, ok := role(*p.JitWorkspaceID, *p.JitRoleKey); ok && r.RoleKey != authz.RoleOwner {
				targets[*p.JitWorkspaceID] = grant{role: r}
				order = append([]uuid.UUID{*p.JitWorkspaceID}, order...)
			}
		}
	}
	for _, wsID := range order {
		t := targets[wsID]
		c := memberChange{WorkspaceID: wsID, WorkspaceName: t.role.WorkspaceName, To: t.role, Scope: t.scope, Source: "sso_group", Group: t.group}
		if t.group == "" {
			c.Source = "sso_jit"
		}
		if pl.User != nil {
			m, err := q.SSOMembership(ctx, store.SSOMembershipParams{WorkspaceID: wsID, UserID: pl.User.ID})
			switch {
			case err == nil:
				id := m.ID
				c.MemberID, c.FromKey, c.FromName, c.FromSource = &id, m.RoleKey, m.RoleName, m.Source
				switch {
				case m.Status != "active":
					c.Kept = "deactivated"
				case m.RoleKey == authz.RoleOwner && m.RoleBuiltin:
					c.Kept = "owner"
				case m.RoleID == t.role.RoleID:
					c.Kept = "same"
				}
			case !isNoRows(err):
				return pl, err
			}
		}
		pl.Changes = append(pl.Changes, c)
	}
	exits, err := s.groupExits(ctx, q, p, maps, id, pl.User, targets, role)
	if err != nil {
		return pl, err
	}
	pl.Changes = append(pl.Changes, exits...)
	return pl, nil
}

// groupExits re-applies the mappings to the memberships they granted
// earlier: when the groups claim no longer contains a group that maps into
// a workspace, that role ends (the JIT workspace falls back to the default
// role; elsewhere the membership is removed). Only memberships a mapping of
// this provider granted (source sso_group, a role the provider maps to
// there) change; hand-set roles, Owners and workspaces the provider doesn't
// map into stay. Without a groups claim the person's groups aren't known,
// so nothing changes.
func (s *Service) groupExits(ctx context.Context, q *store.Queries, p store.SsoProvider, maps []mapping, id identity, u *store.User,
	targets map[uuid.UUID]grant, role func(uuid.UUID, string) (store.SSOWorkspaceRolesRow, bool)) ([]memberChange, error) {
	if !id.GroupsClaim || u == nil || len(maps) == 0 {
		return nil, nil
	}
	rows, err := q.SSOGroupMemberships(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	out := []memberChange{}
	for _, m := range rows {
		if _, mapped := targets[m.WorkspaceID]; mapped || (m.RoleKey == authz.RoleOwner && m.RoleBuiltin) {
			continue
		}
		left := ""
		for _, mp := range maps {
			if mp.WorkspaceID == m.WorkspaceID && mp.RoleKey == m.RoleKey {
				left = firstNonEmpty(mp.Label, mp.Group)
				break
			}
		}
		if left == "" {
			continue
		}
		memberID := m.ID
		c := memberChange{WorkspaceID: m.WorkspaceID, WorkspaceName: m.WorkspaceName, FromKey: m.RoleKey, FromName: m.RoleName, FromSource: m.Source,
			MemberID: &memberID, LeftGroup: left, Source: "sso_jit"}
		if p.JitEnabled && p.JitWorkspaceID != nil && *p.JitWorkspaceID == m.WorkspaceID && p.JitRoleKey != nil {
			if r, ok := role(m.WorkspaceID, *p.JitRoleKey); ok && r.RoleKey != authz.RoleOwner {
				if r.RoleID == m.RoleID {
					continue // the default role is the one the group gave
				}
				c.To = r
				out = append(out, c)
				continue
			}
		}
		c.Remove = true
		out = append(out, c)
	}
	return out, nil
}

// SSOTestOutcome is one row of "What this sign-in would do".
type SSOTestOutcome struct {
	Tone   string `json:"tone" enum:"ok,warn,bad,info,neutral"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
	// Strong is a part of Detail shown in bold ("Admin → Editor").
	Strong string `json:"strong,omitempty"`
	// Code is a part of Detail shown in mono ("amr").
	Code string `json:"code,omitempty"`
}

// describe turns a plan into the lines of the test sign-in view, in a
// language. Built-in roles read in it too.
func describe(l i18n.Lang, id identity, pl signInPlan) []SSOTestOutcome {
	text := func(key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }
	out := []SSOTestOutcome{}
	if pl.Reject != nil {
		out = append(out, SSOTestOutcome{Tone: "bad", Label: text("outcome.rejected", nil), Detail: rejectSentence(l, id, pl.Reject)})
		return out
	}
	switch {
	case pl.Create:
		out = append(out, SSOTestOutcome{Tone: "info", Label: text("outcome.newAccount", nil), Detail: text("detail.newAccount", i18n.Args{"email": id.Email})})
	default:
		out = append(out, SSOTestOutcome{Tone: "ok", Label: text("outcome.existingAccount", nil),
			Detail: text("detail.existingAccount", i18n.Args{"email": id.Email, "name": displayName(*pl.User)})})
	}
	for _, c := range pl.Changes {
		from, to := api.RoleName(l, c.FromKey, c.FromName), api.RoleName(l, c.To.RoleKey, c.To.RoleName)
		change := from + " → " + to
		args := i18n.Args{"workspace": c.WorkspaceName, "group": c.Group, "source": via(c.Group), "role": to, "change": change}
		switch {
		case c.Remove:
			out = append(out, SSOTestOutcome{Tone: "warn", Label: text("outcome.removed", nil), Strong: from,
				Detail: text("detail.removed", i18n.Args{"workspace": c.WorkspaceName, "group": c.LeftGroup, "role": from})})
		case c.LeftGroup != "":
			out = append(out, SSOTestOutcome{Tone: "warn", Label: text("outcome.roleChange", nil), Strong: change,
				Detail: text("detail.leftGroup", i18n.Args{"workspace": c.WorkspaceName, "change": change, "group": c.LeftGroup})})
		case c.Kept == "owner":
			out = append(out, SSOTestOutcome{Tone: "neutral", Label: text("outcome.role", nil), Detail: text("detail.staysOwner", args)})
		case c.Kept == "deactivated":
			out = append(out, SSOTestOutcome{Tone: "neutral", Label: text("outcome.role", nil), Detail: text("detail.deactivated", args)})
		case c.Kept == "same":
			out = append(out, SSOTestOutcome{Tone: "ok", Label: text("outcome.role", nil), Detail: text("detail.stays", args)})
		case c.FromKey == "":
			out = append(out, SSOTestOutcome{Tone: "ok", Label: text("outcome.joins", nil), Detail: text("detail.joins", args), Strong: to})
		default:
			// A demotion is worth a second look: the mapping wins over the
			// role someone was given by hand.
			if roleRank(c.To.RoleKey) > roleRank(c.FromKey) {
				out = append(out, SSOTestOutcome{Tone: "warn", Label: text("outcome.roleChange", nil), Strong: change, Detail: text("detail.demoted", args)})
			} else {
				out = append(out, SSOTestOutcome{Tone: "ok", Label: text("outcome.role", nil), Strong: change, Detail: text("detail.changed", args)})
			}
		}
	}
	if pl.GroupsMissing {
		out = append(out, SSOTestOutcome{Tone: "warn", Label: text("outcome.mappingsInactive", nil), Detail: text("detail.mappingsInactive", i18n.Args{"count": pl.Mappings})})
	}
	mfa := SSOTestOutcome{Tone: "ok", Label: text("outcome.twoFactor", nil), Detail: pl.MFANote.In(l)}
	if !pl.MFA {
		mfa.Tone = "warn"
	}
	if pl.MFAAmr {
		mfa.Code = "amr"
	}
	out = append(out, mfa)
	return out
}

// rejectSentence is why the sign-in would be turned away, in a language,
// led by the email when the reason doesn't start with it.
func rejectSentence(l i18n.Lang, id identity, r *flowError) string {
	why := r.in(l)
	if strings.HasPrefix(why, id.Email) || id.Email == "" {
		return why
	}
	return id.Email + ": " + why
}

// roleRank orders built-in roles from most to least powerful.
func roleRank(key string) int {
	switch key {
	case authz.RoleOwner:
		return 0
	case authz.RoleAdmin:
		return 1
	case authz.RoleEditor:
		return 2
	case authz.RoleViewer:
		return 4
	}
	return 3
}

// apply carries out a sign-in plan inside tx and returns the account.
// Every membership change is audited in its workspace.
func (s *Service) apply(ctx context.Context, tx pgx.Tx, p store.SsoProvider, id identity, pl signInPlan) (store.User, error) {
	q := store.New(tx)
	var u store.User
	details := mustJSON(id.Details)
	switch {
	case pl.Create:
		st, err := q.GetInstanceSettings(ctx)
		if err != nil {
			return u, err
		}
		tz := st.DefaultTimezone
		if tz == "" {
			tz = "UTC"
		}
		name := id.Name
		if name == "" {
			name = strings.SplitN(id.Email, "@", 2)[0]
		}
		u, err = q.SSOCreateUser(ctx, store.SSOCreateUserParams{ID: uuid.Must(uuid.NewV7()), Email: id.Email, Name: name, Timezone: tz, AvatarColor: avatarColor(id.Email)})
		if err != nil {
			return u, err
		}
		if _, err := q.SSOCreateIdentity(ctx, store.SSOCreateIdentityParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, Kind: "sso", SsoProviderID: &p.ID,
			Subject: id.Subject, Email: id.Email, Username: id.Username, Details: details}); err != nil {
			return u, err
		}
		reason, meta := audit.KeepReason(map[string]any{"provider": p.Name, "provider_id": p.ID.String(), "source": "sso_jit"}, keep("reason.jit", i18n.Args{"name": p.Name}))
		if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "user.created", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: reason,
			After: map[string]any{"email": u.Email, "name": u.Name}, Metadata: meta}); err != nil {
			return u, err
		}
	case pl.Link:
		u = *pl.User
		if _, err := q.SSOCreateIdentity(ctx, store.SSOCreateIdentityParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, Kind: "sso", SsoProviderID: &p.ID,
			Subject: id.Subject, Email: id.Email, Username: id.Username, Details: details}); err != nil {
			return u, err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "account.identity_link", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
			Actor: actorOf(u), Metadata: map[string]any{"kind": "sso", "provider": p.Name, "provider_id": p.ID.String(), "by": "verified email"}}); err != nil {
			return u, err
		}
	default:
		u = *pl.User
		if err := q.SSOTouchIdentity(ctx, store.SSOTouchIdentityParams{ID: pl.Identity.ID, Email: id.Email, Username: id.Username, Details: details}); err != nil {
			return u, err
		}
	}
	for _, c := range pl.Changes {
		if c.Kept != "" {
			continue
		}
		ws := c.WorkspaceID
		if c.Remove {
			if err := s.removeMember(ctx, q, p, u, c); err != nil {
				return u, err
			}
			continue
		}
		scope := mustJSON(c.Scope)
		if c.MemberID == nil {
			if err := q.SSOCreateMember(ctx, store.SSOCreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, UserID: u.ID, RoleID: c.To.RoleID, Scope: scope, Source: c.Source}); err != nil {
				return u, err
			}
		} else if err := q.SSOSetMemberRole(ctx, store.SSOSetMemberRoleParams{ID: *c.MemberID, RoleID: c.To.RoleID, Scope: scope, Source: c.Source}); err != nil {
			return u, err
		}
		if authz.Effective(c.To.RoleKey, c.To.Builtin, c.To.Permissions).HasSensitive() {
			if err := workspaces.EnsureTwoFAGrace(ctx, tx, u.ID); err != nil {
				return u, err
			}
		}
		action, before := "member.joined", any(nil)
		if c.MemberID != nil {
			action, before = "member.role_changed", map[string]any{"role": c.FromKey}
		}
		meta := map[string]any{"source": c.Source, "group": c.Group, "provider": p.Name, "provider_id": p.ID.String()}
		if c.LeftGroup != "" {
			meta["left_group"] = c.LeftGroup
		}
		reason, meta := audit.KeepReason(meta, sourceReason(p, c))
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws, Action: action, TargetType: "member", TargetID: u.ID.String(), TargetName: displayName(u),
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: reason,
			Before: before, After: map[string]any{"role": c.To.RoleKey, "scope": c.Scope}, Metadata: meta}); err != nil {
			return u, err
		}
	}
	return u, nil
}

// sourceReason is the audit reason of a membership a sign-in set, kept for
// later readers (a built-in role reads in each reader's language).
func sourceReason(p store.SsoProvider, c memberChange) i18n.Text {
	switch {
	case c.LeftGroup != "":
		return keep("reason.leftGroupDefault", i18n.Args{"name": p.Name, "group": c.LeftGroup})
	case c.Group != "":
		return keep("reason.groupMaps", i18n.Args{"name": p.Name, "group": c.Group, "role": api.RoleArg(c.To.RoleKey, c.To.RoleName)})
	}
	return keep("reason.jitDefault", i18n.Args{"name": p.Name})
}

// removeMember ends a membership whose group mapping no longer matches,
// like removing a member in Team: the API keys and agents the person
// created in the workspace stop working too.
func (s *Service) removeMember(ctx context.Context, q *store.Queries, p store.SsoProvider, u store.User, c memberChange) error {
	ws := c.WorkspaceID
	revoked, err := q.TeamRevokeKeysCreatedBy(ctx, store.TeamRevokeKeysCreatedByParams{WorkspaceID: ws, UserID: &u.ID})
	if err != nil {
		return err
	}
	names := []string{}
	for _, k := range revoked {
		names = append(names, k.Name)
	}
	if _, err := q.DeleteMember(ctx, store.DeleteMemberParams{ID: *c.MemberID, WorkspaceID: ws}); err != nil {
		return err
	}
	reason, meta := audit.KeepReason(map[string]any{"source": "sso_group", "group": c.LeftGroup, "provider": p.Name, "provider_id": p.ID.String(), "user_id": u.ID.String()},
		keep("reason.leftGroup", i18n.Args{"name": p.Name, "group": c.LeftGroup, "role": api.RoleArg(c.FromKey, c.FromName)}))
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws, Action: "member.removed", TargetType: "member", TargetID: c.MemberID.String(), TargetName: displayName(u),
		Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: reason,
		Before:   map[string]any{"email": u.Email, "role": c.FromName, "status": "active"},
		After:    map[string]any{"keys_revoked": names},
		Metadata: meta})
}

func actorOf(u store.User) *audit.Actor {
	id := u.ID
	return &audit.Actor{Type: authz.ActorUser, ID: &id, Name: displayName(u)}
}

// avatarColor picks a stable initials colour for a new account.
func avatarColor(email string) string {
	colors := []string{"info", "ok", "purple", "warn", "neutral"}
	h := 0
	for _, c := range email {
		h = (h*31 + int(c)) % 1000003
	}
	return colors[h%len(colors)]
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
