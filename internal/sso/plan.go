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

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
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
	MFANote string
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
		pl.MFA, pl.MFANote = true, PresetFor(p.Preset).TrustedTwoFA
	case id.amrMFA():
		pl.MFA, pl.MFANote = true, "amr includes mfa · counts as two-factor via IdP"
	case len(id.AMR) > 0:
		pl.MFANote = "amr is " + strings.Join(id.AMR, "+") + " · no second factor, so the person's own is asked for"
	default:
		pl.MFANote = "No amr claim · the person's own second factor is asked for"
	}
	if id.Subject == "" {
		pl.Reject = &flowError{Code: "no_subject", Detail: "The ID token has no sub claim."}
		return pl, nil
	}
	if id.Email == "" {
		pl.Reject = &flowError{Code: "no_email", Detail: fmt.Sprintf("%s sent no email claim (%q). Check the claim mapping.", p.Name, claimsOf(p).Email)}
		return pl, nil
	}
	domain := emailDomain(id.Email)
	if len(p.AllowedDomains) > 0 && !slices.Contains(p.AllowedDomains, domain) {
		pl.Reject = &flowError{Code: "domain_not_allowed", Detail: "email domain not allowed for " + p.Name + " SSO", NoAccount: true, Email: id.Email}
		return pl, nil
	}
	verified := false
	if id.EmailVerified != nil {
		verified = *id.EmailVerified
		if !verified {
			pl.Reject = &flowError{Code: "email_not_verified", Detail: p.Name + " says " + id.Email + " isn't verified.", Email: id.Email}
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
		pl.Reject = &flowError{Code: "account_deactivated", Detail: id.Email + " is deactivated.", Email: id.Email}
		return pl, nil
	}
	if pl.User == nil {
		st, err := q.GetInstanceSettings(ctx)
		if err != nil {
			return pl, err
		}
		switch {
		case !p.JitEnabled:
			pl.Reject = &flowError{Code: "no_account", Detail: "no account for " + id.Email + " and automatic accounts (JIT) are off", NoAccount: true, Email: id.Email}
		case !verified:
			pl.Reject = &flowError{Code: "email_not_verified", Detail: p.Name + " didn't confirm that " + id.Email + " is verified, so no account is created.", NoAccount: true, Email: id.Email}
		case len(st.JitDomainAllowlist) > 0 && !slices.Contains(st.JitDomainAllowlist, domain):
			pl.Reject = &flowError{Code: "domain_not_allowed", Detail: "email domain not on the allowlist for automatic accounts", NoAccount: true, Email: id.Email}
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
	type target struct {
		role  store.SSOWorkspaceRolesRow
		scope authz.Scope
		group string
	}
	targets := map[uuid.UUID]target{}
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
				targets[m.WorkspaceID] = target{role: r, scope: m.Scope, group: label}
			}
		}
	}
	if pl.Create && p.JitEnabled && p.JitWorkspaceID != nil && p.JitRoleKey != nil {
		if _, mapped := targets[*p.JitWorkspaceID]; !mapped {
			if r, ok := role(*p.JitWorkspaceID, *p.JitRoleKey); ok && r.RoleKey != authz.RoleOwner {
				targets[*p.JitWorkspaceID] = target{role: r}
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
	return pl, nil
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

// describe turns a plan into the lines of the test sign-in view.
func describe(p store.SsoProvider, id identity, pl signInPlan) []SSOTestOutcome {
	out := []SSOTestOutcome{}
	if pl.Reject != nil {
		out = append(out, SSOTestOutcome{Tone: "bad", Label: "Rejected", Detail: rejectSentence(id, pl.Reject)})
		return out
	}
	switch {
	case pl.Create:
		out = append(out, SSOTestOutcome{Tone: "info", Label: "New account", Detail: "No account for " + id.Email + " yet · a just-in-time account is created at the first sign-in"})
	default:
		out = append(out, SSOTestOutcome{Tone: "ok", Label: "Existing account", Detail: id.Email + " matches " + displayName(*pl.User) + " · no account created"})
	}
	for _, c := range pl.Changes {
		from := ""
		if c.Group != "" {
			from = " from group “" + c.Group + "”"
		} else {
			from = " (JIT default)"
		}
		switch {
		case c.Kept == "owner":
			out = append(out, SSOTestOutcome{Tone: "neutral", Label: "Role", Detail: c.WorkspaceName + ": stays Owner · mappings never change an Owner"})
		case c.Kept == "deactivated":
			out = append(out, SSOTestOutcome{Tone: "neutral", Label: "Role", Detail: c.WorkspaceName + ": membership is deactivated · the mapping doesn't reactivate it"})
		case c.Kept == "same":
			out = append(out, SSOTestOutcome{Tone: "ok", Label: "Role", Detail: c.WorkspaceName + ": stays " + c.To.RoleName + from})
		case c.FromKey == "":
			out = append(out, SSOTestOutcome{Tone: "ok", Label: "Joins", Detail: c.WorkspaceName + ": joins as " + c.To.RoleName + from, Strong: c.To.RoleName})
		default:
			change := c.FromName + " → " + c.To.RoleName
			// A demotion is worth a second look: the mapping wins over the
			// role someone was given by hand.
			if roleRank(c.To.RoleKey) > roleRank(c.FromKey) {
				out = append(out, SSOTestOutcome{Tone: "warn", Label: "Role change", Strong: change,
					Detail: c.WorkspaceName + ": " + change + from + ". The direct assignment is replaced at their next sign-in; review the mapping if that's not intended."})
			} else {
				out = append(out, SSOTestOutcome{Tone: "ok", Label: "Role", Strong: change, Detail: c.WorkspaceName + ": " + change + from})
			}
		}
	}
	if pl.GroupsMissing {
		n := pl.Mappings
		out = append(out, SSOTestOutcome{Tone: "warn", Label: "Mappings inactive", Detail: "No groups claim · " + plural(n, "the group mapping doesn't apply", fmt.Sprintf("the %d group mappings don't apply", n))})
	}
	mfa := SSOTestOutcome{Tone: "ok", Label: "Two-factor", Detail: pl.MFANote}
	if !pl.MFA {
		mfa.Tone = "warn"
	}
	if strings.HasPrefix(pl.MFANote, "amr") {
		mfa.Code = "amr"
	}
	out = append(out, mfa)
	return out
}

func rejectSentence(id identity, r *flowError) string {
	if strings.HasPrefix(r.Detail, id.Email) || id.Email == "" {
		return r.Detail
	}
	return id.Email + ": " + r.Detail
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

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
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
		if err := s.Audit.LogWith(ctx, q, audit.Event{Action: "user.created", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: "just-in-time account from " + p.Name + " SSO",
			After: map[string]any{"email": u.Email, "name": u.Name}, Metadata: map[string]any{"provider": p.Name, "provider_id": p.ID.String(), "source": "sso_jit"}}); err != nil {
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
		scope := mustJSON(c.Scope)
		ws := c.WorkspaceID
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
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws, Action: action, TargetType: "member", TargetID: u.ID.String(), TargetName: displayName(u),
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: sourceReason(p, c),
			Before: before, After: map[string]any{"role": c.To.RoleKey, "scope": c.Scope},
			Metadata: map[string]any{"source": c.Source, "group": c.Group, "provider": p.Name, "provider_id": p.ID.String()}}); err != nil {
			return u, err
		}
	}
	return u, nil
}

func sourceReason(p store.SsoProvider, c memberChange) string {
	if c.Group != "" {
		return p.Name + " group “" + c.Group + "” maps to " + c.To.RoleName
	}
	return "just-in-time default role of " + p.Name + " SSO"
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
