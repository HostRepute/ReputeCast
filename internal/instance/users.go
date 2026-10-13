package instance

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// InstanceUserWorkspace is a workspace a person belongs to, with their role.
type InstanceUserWorkspace struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status" enum:"active,deactivated"`
}

// InstanceUser is one row of Instance admin › Users.
type InstanceUser struct {
	ID            string                  `json:"id"`
	Email         string                  `json:"email"`
	Name          string                  `json:"name"`
	AvatarColor   string                  `json:"avatarColor"`
	AvatarURL     string                  `json:"avatarUrl,omitempty"`
	InstanceAdmin bool                    `json:"instanceAdmin"`
	BreakGlass    bool                    `json:"breakGlass"`
	Workspaces    []InstanceUserWorkspace `json:"workspaces"`
	// Methods: their own credentials first ("Password · passkey"); people
	// without a password list their SSO or social sign-in first
	// ("Google SSO · passkey"). AllMethods has every one.
	Methods    []access.MemberSignInMethod `json:"methods"`
	AllMethods []access.MemberSignInMethod `json:"allMethods"`
	// TwoFactor is the same status as on the team list (On · TOTP, Via
	// IdP, Required · off with the grace period, Off).
	TwoFactor    access.MemberTwoFA `json:"twoFactor"`
	LastSignInAt *time.Time         `json:"lastSignInAt,omitempty"`
	Status       string             `json:"status" enum:"active,deactivated"`
	// DeactivatedAt is set for deactivated accounts.
	DeactivatedAt *time.Time `json:"deactivatedAt,omitempty"`
	LockedUntil   *time.Time `json:"lockedUntil,omitempty"`
	Sessions      int        `json:"sessions"`
	APIKeys       int        `json:"apiKeys" doc:"Active API keys and agents the person created"`
	// SoleOwnerOf names the workspaces where they are the only Owner.
	SoleOwnerOf []string `json:"soleOwnerOf"`
	HasPassword bool     `json:"hasPassword"`
	// SSOProvider is the SSO they sign in with, for "managed by" texts.
	SSOProvider string `json:"ssoProvider,omitempty"`
	Self        bool   `json:"self"`
}

// InstanceWorkspaceRef names a workspace for the filter.
type InstanceWorkspaceRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// InstanceUserList is GET /admin/users.
type InstanceUserList struct {
	Items []InstanceUser `json:"items"`
	// Total is every account, before filters.
	Total      int                    `json:"total"`
	AdminNames []string               `json:"adminNames"`
	Workspaces []InstanceWorkspaceRef `json:"workspaces"`
}

// UserFilter narrows the users list.
type UserFilter struct {
	Search    string
	Workspace string
	Status    string
	TwoFactor string // on, off, idp
}

// Users lists every account on the instance.
func (s *Service) Users(ctx context.Context, f UserFilter) (InstanceUserList, error) {
	me, err := admin(ctx)
	if err != nil {
		return InstanceUserList{}, err
	}
	all, err := s.loadUsers(ctx, me)
	if err != nil {
		return InstanceUserList{}, err
	}
	out := InstanceUserList{Items: []InstanceUser{}, Total: len(all), AdminNames: []string{}, Workspaces: []InstanceWorkspaceRef{}}
	wss, err := s.Q.ListWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	for _, w := range wss {
		out.Workspaces = append(out.Workspaces, InstanceWorkspaceRef{Slug: w.Slug, Name: w.Name})
	}
	q := strings.ToLower(strings.TrimSpace(f.Search))
	for _, u := range all {
		if u.InstanceAdmin && u.Status == "active" {
			out.AdminNames = append(out.AdminNames, u.Name)
		}
		if q != "" && !strings.Contains(strings.ToLower(u.Name), q) && !strings.Contains(u.Email, q) {
			continue
		}
		if f.Workspace != "" && !slices.ContainsFunc(u.Workspaces, func(w InstanceUserWorkspace) bool { return w.Slug == f.Workspace }) {
			continue
		}
		if f.Status != "" && u.Status != f.Status {
			continue
		}
		switch f.TwoFactor {
		case "on":
			if u.TwoFactor.State != "on" {
				continue
			}
		case "off":
			if u.TwoFactor.State != "off" && u.TwoFactor.State != "required" {
				continue
			}
		case "idp":
			if u.TwoFactor.State != "idp" {
				continue
			}
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

// roleRank orders roles: Owner and Admin first, then Editor, custom roles, Viewer.
func roleRank(key string, builtin bool) int {
	if !builtin {
		return 3
	}
	switch key {
	case authz.RoleOwner, authz.RoleAdmin:
		return 1
	case authz.RoleEditor:
		return 2
	case authz.RoleViewer:
		return 4
	}
	return 3
}

func (s *Service) loadUsers(ctx context.Context, me *store.User) ([]InstanceUser, error) {
	rows, err := s.Q.InstanceAdminUsers(ctx)
	if err != nil {
		return nil, err
	}
	members, err := s.Q.InstanceAdminMemberships(ctx)
	if err != nil {
		return nil, err
	}
	idents, err := s.Q.InstanceAdminIdentities(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return nil, err
	}
	byUser := map[uuid.UUID][]store.InstanceAdminMembershipsRow{}
	for _, m := range members {
		byUser[m.UserID] = append(byUser[m.UserID], m)
	}
	idsByUser := map[uuid.UUID][]store.InstanceAdminIdentitiesRow{}
	for _, i := range idents {
		idsByUser[i.UserID] = append(idsByUser[i.UserID], i)
	}
	type sorted struct {
		u    InstanceUser
		rank int
		at   time.Time
	}
	list := make([]sorted, 0, len(rows))
	for _, r := range rows {
		u := s.userView(ctx, r, byUser[r.ID], idsByUser[r.ID], st, me)
		rank := 5
		for _, m := range byUser[r.ID] {
			if m.Status == "active" {
				rank = min(rank, roleRank(m.RoleKey, m.RoleBuiltin))
			}
		}
		if u.InstanceAdmin {
			rank = 0
		}
		if u.Status != "active" {
			rank = 9
		}
		list = append(list, sorted{u, rank, r.CreatedAt})
	}
	slices.SortStableFunc(list, func(a, b sorted) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return a.at.Compare(b.at)
	})
	out := make([]InstanceUser, len(list))
	for i, x := range list {
		out[i] = x.u
	}
	return out, nil
}

func (s *Service) userView(ctx context.Context, r store.InstanceAdminUsersRow, ms []store.InstanceAdminMembershipsRow, ids []store.InstanceAdminIdentitiesRow, st store.InstanceSetting, me *store.User) InstanceUser {
	u := InstanceUser{ID: r.ID.String(), Email: r.Email, Name: displayName(r.Name, r.Email), AvatarColor: r.AvatarColor, InstanceAdmin: r.IsInstanceAdmin,
		BreakGlass: r.BreakGlass, Workspaces: []InstanceUserWorkspace{}, LastSignInAt: r.LastSignInAt,
		Status: r.Status, DeactivatedAt: r.DeactivatedAt, Sessions: int(r.Sessions), APIKeys: int(r.ApiKeys), SoleOwnerOf: []string{},
		HasPassword: r.HasPassword, Self: me != nil && me.ID == r.ID}
	if r.AvatarUpdatedAt != nil {
		u.AvatarURL = fmt.Sprintf("/api/v1/users/%s/avatar?v=%d", r.ID, r.AvatarUpdatedAt.Unix())
	}
	if r.LockedUntil != nil && r.LockedUntil.After(s.now()) {
		u.LockedUntil = r.LockedUntil
	}
	for _, m := range ms {
		u.Workspaces = append(u.Workspaces, InstanceUserWorkspace{Slug: m.Slug, Name: m.WorkspaceName, Role: m.RoleName, Status: m.Status})
	}
	// Active memberships first, in workspace order.
	slices.SortStableFunc(u.Workspaces, func(a, b InstanceUserWorkspace) int {
		if (a.Status == "active") != (b.Status == "active") {
			if a.Status == "active" {
				return -1
			}
			return 1
		}
		return 0
	})
	ssoOnly := !r.HasPassword
	var linked []access.Identity
	hasSSO := false
	for _, i := range ids {
		linked = append(linked, access.Identity{Kind: i.Kind, ProviderName: i.ProviderName, ProviderPreset: i.ProviderPreset})
		if i.Kind == "sso" {
			hasSSO = true
			if u.SSOProvider == "" {
				u.SSOProvider = i.ProviderName
			}
		}
	}
	u.AllMethods = access.SignInMethods(r.HasPassword, r.Passkeys, linked)
	u.Methods = primaryMethods(u.AllMethods, r.HasPassword)
	u.TwoFactor = access.TwoFAStatus(access.TwoFAInput{TOTP: r.TotpEnabledAt != nil, Passkeys: r.Passkeys, SSOOnly: ssoOnly && hasSSO,
		GraceUntil: r.TwofaGraceUntil, Required: requiresTwoFactor(st, ms), Now: s.now()})
	if r.Status != "active" {
		u.Methods = []access.MemberSignInMethod{}
	}
	if sole, err := s.Q.InstanceAdminSoleOwnerships(ctx, r.ID); err == nil {
		for _, w := range sole {
			u.SoleOwnerOf = append(u.SoleOwnerOf, w.Name)
		}
	}
	return u
}

// requiresTwoFactor reports whether the policy asks the person for two-factor.
func requiresTwoFactor(st store.InstanceSetting, ms []store.InstanceAdminMembershipsRow) bool {
	switch st.TwofaEnforcement {
	case "everyone":
		return true
	case "off":
		return false
	}
	for _, m := range ms {
		if m.Status == "active" && authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions).HasSensitive() {
			return true
		}
	}
	return false
}

// primaryMethods is the sign-in column of the users list: a person with a
// password shows their own credentials ("Password · passkey"); without
// one, their SSO or social sign-in comes first ("Google SSO · passkey").
// The Google Workspace preset is shortened to "Google SSO" as on the board.
func primaryMethods(all []access.MemberSignInMethod, hasPassword bool) []access.MemberSignInMethod {
	out := []access.MemberSignInMethod{}
	if hasPassword {
		return access.PrimarySignIn(all)
	}
	var keys []access.MemberSignInMethod
	for _, m := range all {
		switch m.Kind {
		case "passkey":
			keys = append(keys, access.MemberSignInMethod{Kind: m.Kind, Label: "passkey"})
		case "sso":
			if m.Preset == "google" {
				m.Label = "Google SSO"
			}
			out = append(out, m)
		default:
			out = append(out, m)
		}
	}
	if len(out) == 0 && len(keys) > 0 {
		keys[0].Label = "Passkey"
	}
	return append(out, keys...)
}

// user loads one account for an action, or a 404.
func (s *Service) user(ctx context.Context, id uuid.UUID) (store.InstanceAdminUserRow, error) {
	r, err := s.Q.InstanceAdminUser(ctx, id)
	if err != nil {
		return r, api.MapError(err, "User")
	}
	return r, nil
}

// InstanceUserReason carries the reason an action is logged with.
type InstanceUserReason struct {
	Reason string `json:"reason,omitempty" maxLength:"500"`
}

// InstanceSignOutResult says how many sessions ended.
type InstanceSignOutResult struct {
	Revoked int64 `json:"revoked"`
}

// SignOut ends every session of a person.
func (s *Service) SignOut(ctx context.Context, id uuid.UUID, in InstanceUserReason) (InstanceSignOutResult, error) {
	u, err := s.user(ctx, id)
	if err != nil {
		return InstanceSignOutResult{}, err
	}
	var n int64
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if n, err = q.RevokeAllUserSessions(ctx, id); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "user.sign_out_everywhere", TargetType: "user", TargetID: id.String(), TargetName: u.Email,
			Reason: strings.TrimSpace(in.Reason), Metadata: map[string]any{"sessions": n, "summary": fmt.Sprintf("%d %s ended", n, plural(int(n), "session", "sessions"))}})
	})
	return InstanceSignOutResult{Revoked: n}, err
}

// InstanceDeactivateInput deactivates an account.
type InstanceDeactivateInput struct {
	Reason string `json:"reason" minLength:"3" maxLength:"500"`
	// RevokeKeys revokes the API keys and agents the person created.
	RevokeKeys bool `json:"revokeKeys"`
}

// InstanceDeactivateResult says what the deactivation ended.
type InstanceDeactivateResult struct {
	Sessions    int64 `json:"sessions"`
	Keys        int64 `json:"keys"`
	Memberships int64 `json:"memberships"`
}

// Deactivate signs a person out everywhere and blocks sign-in; history,
// memberships and content stay. The last instance admin and the only
// Owner of a workspace can't be deactivated.
func (s *Service) Deactivate(ctx context.Context, id uuid.UUID, in InstanceDeactivateInput) (InstanceDeactivateResult, error) {
	me, err := admin(ctx)
	if err != nil {
		return InstanceDeactivateResult{}, err
	}
	u, err := s.user(ctx, id)
	if err != nil {
		return InstanceDeactivateResult{}, err
	}
	if me.ID == id {
		return InstanceDeactivateResult{}, api.Conflict("You can't deactivate your own account. Ask another instance admin.")
	}
	if u.Status != "active" {
		return InstanceDeactivateResult{}, api.Conflict(displayName(u.Name, u.Email) + " is already deactivated.")
	}
	if u.IsInstanceAdmin {
		if n, err := s.Q.InstanceAdminCountAdmins(ctx); err == nil && n <= 1 {
			return InstanceDeactivateResult{}, api.Err(http.StatusConflict, "last_admin", displayName(u.Name, u.Email)+" is the last instance admin. Make someone else an instance admin first.")
		}
	}
	if sole, err := s.Q.InstanceAdminSoleOwnerships(ctx, id); err == nil && len(sole) > 0 {
		names := make([]string, len(sole))
		for i, w := range sole {
			names[i] = w.Name
		}
		return InstanceDeactivateResult{}, api.Err(http.StatusConflict, api.CodeLastOwner, fmt.Sprintf("%s is the only Owner of %s. Transfer ownership first, then deactivate.", displayName(u.Name, u.Email), strings.Join(names, " and ")))
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return InstanceDeactivateResult{}, api.Invalid("body.reason", "Say why; the reason is recorded in the audit log.")
	}
	var res InstanceDeactivateResult
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		at := s.now()
		if n, err := q.InstanceAdminDeactivateUser(ctx, store.InstanceAdminDeactivateUserParams{ID: id, At: &at}); err != nil {
			return err
		} else if n == 0 {
			return api.Conflict(displayName(u.Name, u.Email) + " is already deactivated.")
		}
		var err error
		if res.Memberships, err = q.InstanceAdminDeactivateMemberships(ctx, store.InstanceAdminDeactivateMembershipsParams{UserID: id, At: &at}); err != nil {
			return err
		}
		if res.Sessions, err = q.RevokeAllUserSessions(ctx, id); err != nil {
			return err
		}
		if in.RevokeKeys {
			if res.Keys, err = q.InstanceAdminRevokeKeysCreatedBy(ctx, &id); err != nil {
				return err
			}
		}
		parts := []string{"reason: " + reason}
		if res.Keys > 0 {
			parts = append(parts, fmt.Sprintf("%d API %s revoked", res.Keys, plural(int(res.Keys), "key", "keys")))
		}
		parts = append(parts, fmt.Sprintf("%d %s ended", res.Sessions, plural(int(res.Sessions), "session", "sessions")))
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "user.deactivate", TargetType: "user", TargetID: id.String(), TargetName: displayName(u.Name, u.Email) + " · " + u.Email,
			Reason: reason, Before: map[string]any{"status": "active"}, After: map[string]any{"status": "deactivated"},
			Metadata: map[string]any{"summary": strings.Join(parts, " · "), "sessions": res.Sessions, "keys_revoked": res.Keys, "memberships": res.Memberships}})
	})
	return res, err
}

// Reactivate lets a deactivated person sign in again; the memberships that
// were deactivated with the account come back.
func (s *Service) Reactivate(ctx context.Context, id uuid.UUID) error {
	u, err := s.user(ctx, id)
	if err != nil {
		return err
	}
	if u.Status == "active" {
		return api.Conflict(displayName(u.Name, u.Email) + " is already active.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.InstanceAdminReactivateUser(ctx, id); err != nil {
			return err
		}
		n := int64(0)
		if u.DeactivatedAt != nil {
			var err error
			if n, err = q.InstanceAdminReactivateMemberships(ctx, store.InstanceAdminReactivateMembershipsParams{UserID: id, At: u.DeactivatedAt}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "user.reactivate", TargetType: "user", TargetID: id.String(), TargetName: displayName(u.Name, u.Email) + " · " + u.Email,
			Before: map[string]any{"status": "deactivated"}, After: map[string]any{"status": "active"}, Metadata: map[string]any{"memberships": n}})
	})
}

// Unlock clears a sign-in lockout.
func (s *Service) Unlock(ctx context.Context, id uuid.UUID) error {
	u, err := s.user(ctx, id)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InstanceAdminUnlockUser(ctx, id); err != nil {
			return err
		}
		if err := q.ClearSignInThrottle(ctx, u.Email); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "user.unlock", TargetType: "user", TargetID: id.String(), TargetName: u.Email})
	})
}

// InstanceTwoFactorResetInput resets a person's authenticator.
type InstanceTwoFactorResetInput struct {
	Reason string `json:"reason" minLength:"3" maxLength:"500"`
	// Password re-confirms the admin (required when they have one).
	Password string `json:"password,omitempty" maxLength:"1024"`
}

// ResetTwoFactor removes a person's authenticator and recovery codes
// (passkeys stay), restarts the grace period, signs them out and notifies
// them. The reason is required and logged; the admin confirms with their
// own password.
func (s *Service) ResetTwoFactor(ctx context.Context, id uuid.UUID, in InstanceTwoFactorResetInput) error {
	me, err := admin(ctx)
	if err != nil {
		return err
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return api.Invalid("body.reason", "A reason is required; it's recorded in the audit log.")
	}
	if err := s.reauth(ctx, me, in.Password); err != nil {
		return err
	}
	u, err := s.user(ctx, id)
	if err != nil {
		return err
	}
	if u.TotpEnabledAt == nil {
		return api.Conflict(displayName(u.Name, u.Email) + " has no authenticator app to reset.")
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return err
	}
	grace := s.now().Add(time.Duration(st.TwofaGraceDays) * 24 * time.Hour)
	var sessions int64
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InstanceAdminResetTwoFactor(ctx, store.InstanceAdminResetTwoFactorParams{ID: id, GraceUntil: &grace}); err != nil {
			return err
		}
		if err := q.DeleteRecoveryCodes(ctx, id); err != nil {
			return err
		}
		var err error
		if sessions, err = q.RevokeAllUserSessions(ctx, id); err != nil {
			return err
		}
		if err := notify.ToUser(ctx, tx, id, notify.Notification{Kind: "twofa_reset", Severity: "warn", Title: "Two-factor was reset",
			Summary: displayName(me.Name, me.Email) + " reset your two-factor",
			Body:    fmt.Sprintf("%s reset two-factor on your account (reason: %s). Set up an authenticator app again before %s.", displayName(me.Name, me.Email), reason, grace.Format("Jan 2")),
			Link:    accountLink(ctx, s, id)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "user.twofa_reset", TargetType: "user", TargetID: id.String(), TargetName: displayName(u.Name, u.Email) + " · " + u.Email,
			Reason: reason, Before: map[string]any{"totp": true}, After: map[string]any{"totp": false, "grace_until": grace},
			Metadata: map[string]any{"summary": "reason: " + reason + fmt.Sprintf(" · grace %d days", st.TwofaGraceDays), "sessions": sessions}})
	})
	if err != nil {
		return err
	}
	// An email too, when one can be sent: the person may not sign in soon.
	if s.mail().SystemAvailable(ctx, nil) {
		msg := mailer.Message{To: u.Email, ToName: u.Name, Kind: "twofa_reset", Subject: "Two-factor was reset on your " + st.Name + " account",
			Text: fmt.Sprintf("%s, an instance admin, reset two-factor on your account %s.\n\nReason: %s\n\nSign in with your password and set up an authenticator app again before %s. If you didn't ask for this, tell your admin right away.\n",
				displayName(me.Name, me.Email), u.Email, reason, grace.Format("January 2")),
			HTML: fmt.Sprintf(`<p style="font:14px/1.5 sans-serif">%s, an instance admin, reset two-factor on your account <b>%s</b>.</p><p style="font:14px/1.5 sans-serif">Reason: %s</p><p style="font:14px/1.5 sans-serif">Sign in with your password and set up an authenticator app again before %s. If you didn't ask for this, tell your admin right away.</p>`,
				html.EscapeString(displayName(me.Name, me.Email)), html.EscapeString(u.Email), html.EscapeString(reason), grace.Format("January 2"))}
		if _, err := s.mail().SendSystem(ctx, msg); err != nil {
			s.Log.WarnContext(ctx, "two-factor reset email not sent", "user", id, "err", err)
		}
	}
	return nil
}

// accountLink is the person's Account & security page in their first workspace.
func accountLink(ctx context.Context, s *Service, userID uuid.UUID) string {
	wss, err := s.Q.ListUserWorkspaces(ctx, userID)
	if err != nil || len(wss) == 0 {
		return ""
	}
	return "/w/" + wss[0].Slug + "/account"
}

// reauth confirms a sensitive admin action with the admin's password, or,
// for admins without one (SSO only), a session confirmed by a second factor.
func (s *Service) reauth(ctx context.Context, me *store.User, password string) error {
	if me.PasswordHash == nil {
		if sess := api.SessionFrom(ctx); sess != nil && sess.MfaVerified {
			return nil
		}
		return api.Err(http.StatusForbidden, "reauth_required", "Sign in again with two-factor to reset someone's two-factor.")
	}
	if password == "" {
		return api.Invalid("body.password", "Confirm with your password.")
	}
	ok, err := authn.VerifyPassword(password, *me.PasswordHash)
	if err != nil || !ok {
		_ = s.Audit.Log(ctx, audit.Event{Action: "user.twofa_reset", Result: audit.Denied, Reason: "wrong admin password", TargetType: "user", TargetName: me.Email})
		return api.Invalid("body.password", "That isn't your password.")
	}
	return nil
}

// InstancePasswordResetInfo is what the "Send a password reset" dialog shows.
type InstancePasswordResetInfo struct {
	Email string `json:"email"`
	// System: a System-stream provider can send the email now.
	System bool `json:"system"`
	// Provider is the System-stream provider ("Postmark") and its status.
	Provider       string `json:"provider,omitempty"`
	ProviderStatus string `json:"providerStatus,omitempty"`
	Command        string `json:"command"`
	// SSOOnly: the person has no password; their IdP manages sign-in.
	SSOOnly           bool   `json:"ssoOnly"`
	SSOProvider       string `json:"ssoProvider,omitempty"`
	MinutesValid      int    `json:"minutesValid"`
	LastSentAt        string `json:"lastSentAt,omitempty"`
	DeactivatedNotice string `json:"deactivatedNotice,omitempty"`
}

// PasswordResetInfo returns the dialog's state for a person.
func (s *Service) PasswordResetInfo(ctx context.Context, id uuid.UUID) (InstancePasswordResetInfo, error) {
	u, err := s.user(ctx, id)
	if err != nil {
		return InstancePasswordResetInfo{}, err
	}
	out := InstancePasswordResetInfo{Email: u.Email, Command: "reputecast user reset-password --email " + u.Email, MinutesValid: int(accounts.ResetTTL / time.Minute)}
	out.System = s.mail().SystemAvailable(ctx, nil)
	if p, err := s.Q.InstanceAdminSystemProvider(ctx); err == nil {
		out.Provider, out.ProviderStatus = p.Name, p.Status
	}
	if !u.HasPassword {
		ids, _ := s.Q.InstanceAdminIdentities(ctx)
		for _, i := range ids {
			if i.UserID == id && i.Kind == "sso" {
				out.SSOOnly, out.SSOProvider = true, i.ProviderName
				break
			}
		}
	}
	if u.Status != "active" {
		out.DeactivatedNotice = displayName(u.Name, u.Email) + " is deactivated. Reactivate the account before sending a reset link."
	}
	return out, nil
}

// InstancePasswordResetSent says the reset email went out.
type InstancePasswordResetSent struct {
	Via       string    `json:"via"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SendPasswordReset emails a one-time reset link (15 minutes) on the System stream.
func (s *Service) SendPasswordReset(ctx context.Context, id uuid.UUID) (InstancePasswordResetSent, error) {
	me, err := admin(ctx)
	if err != nil {
		return InstancePasswordResetSent{}, err
	}
	u, err := s.user(ctx, id)
	if err != nil {
		return InstancePasswordResetSent{}, err
	}
	if u.Status != "active" {
		return InstancePasswordResetSent{}, api.Conflict(displayName(u.Name, u.Email) + " is deactivated. Reactivate the account first.")
	}
	if !s.mail().SystemAvailable(ctx, nil) {
		return InstancePasswordResetSent{}, api.Err(http.StatusConflict, api.CodeProviderUnavailable,
			"No System-stream provider can send email yet. Run reputecast user reset-password --email "+u.Email+" on the server instead; it prints the same link.")
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return InstancePasswordResetSent{}, err
	}
	token, expires, err := accounts.CreateResetToken(ctx, s.Q, id, "admin", s.now())
	if err != nil {
		return InstancePasswordResetSent{}, err
	}
	link := accounts.ResetLink(s.publicURL(st), token)
	admin := displayName(me.Name, me.Email)
	msg := mailer.Message{To: u.Email, ToName: u.Name, Kind: "password_reset", Subject: "Reset your " + st.Name + " password",
		Text: fmt.Sprintf("%s, an instance admin, sent you a link to set a new password for %s on %s.\n\nSet a new password: %s\n\nThe link works once and expires in 15 minutes. Your current password keeps working until you use it.\n",
			admin, u.Email, st.Name, link),
		HTML: fmt.Sprintf(`<p style="font:14px/1.5 sans-serif">%s, an instance admin, sent you a link to set a new password for <b>%s</b> on %s.</p><p style="font:14px sans-serif"><a href="%s">Set a new password</a></p><p style="font:13px/1.5 sans-serif;color:#5B6472">The link works once and expires in 15 minutes. Your current password keeps working until you use it.</p>`,
			html.EscapeString(admin), html.EscapeString(u.Email), html.EscapeString(st.Name), html.EscapeString(link))}
	via, err := s.mail().SendSystem(ctx, msg)
	if err != nil {
		_ = s.Q.InvalidatePasswordResets(ctx, id)
		return InstancePasswordResetSent{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, "The reset email couldn't be sent ("+err.Error()+"). Nothing changed; use the CLI command instead.")
	}
	err = s.Audit.Log(ctx, audit.Event{Action: "user.password_reset_send", TargetType: "user", TargetID: id.String(), TargetName: displayName(u.Name, u.Email) + " · " + u.Email,
		Metadata: map[string]any{"via": via, "summary": "one-time link sent via " + via + " · expires in 15 minutes"}})
	return InstancePasswordResetSent{Via: via, ExpiresAt: expires}, err
}

// InstanceAdminInput grants or removes instance admin.
type InstanceAdminInput struct {
	Admin bool `json:"admin"`
}

// SetInstanceAdmin makes a person an instance admin or removes it; the last
// one can't be removed.
func (s *Service) SetInstanceAdmin(ctx context.Context, id uuid.UUID, in InstanceAdminInput) error {
	u, err := s.user(ctx, id)
	if err != nil {
		return err
	}
	if u.IsInstanceAdmin == in.Admin {
		return nil
	}
	if in.Admin && u.Status != "active" {
		return api.Conflict(displayName(u.Name, u.Email) + " is deactivated. Reactivate the account first.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if !in.Admin {
			// Counted inside the transaction so two admins removing each
			// other can't both succeed.
			if _, err := tx.Exec(ctx, `LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE`); err != nil {
				return err
			}
			n, err := q.InstanceAdminCountAdmins(ctx)
			if err != nil {
				return err
			}
			if n <= 1 {
				return api.Err(http.StatusConflict, "last_admin", displayName(u.Name, u.Email)+" is the last instance admin. Make someone else an instance admin first.")
			}
		}
		if err := q.SetUserInstanceAdmin(ctx, store.SetUserInstanceAdminParams{ID: id, IsInstanceAdmin: in.Admin}); err != nil {
			return err
		}
		action := "user.instance_admin_grant"
		if !in.Admin {
			action = "user.instance_admin_revoke"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: action, TargetType: "user", TargetID: id.String(), TargetName: displayName(u.Name, u.Email),
			Before: map[string]any{"instance_admin": u.IsInstanceAdmin}, After: map[string]any{"instance_admin": in.Admin}})
	})
}
