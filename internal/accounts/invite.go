package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// Invitation links are /invite/<token>; the invitations table stores the
// SHA-256 of the token (authn.HashToken). The Team screens create them.

// InvitationView is what the invitee sees before joining (A5 §5.4).
type InvitationView struct {
	Status        string `json:"status" enum:"pending,expired,revoked,accepted"`
	Email         string `json:"email"`
	WorkspaceName string `json:"workspaceName"`
	WorkspaceSlug string `json:"workspaceSlug"`
	InviterName   string `json:"inviterName"`
	// InviterUnknown: the inviter's account is gone, so InviterName names an
	// admin of the workspace ("An admin of Acme Newsletter").
	InviterUnknown bool   `json:"inviterUnknown,omitempty"`
	RoleKey        string `json:"roleKey"`
	RoleName       string `json:"roleName"`
	// Scope restricts the role ("Beta Program list only"); empty for everything.
	Scope string `json:"scope,omitempty"`
	// Description says what the role can do, in one or two sentences.
	Description string     `json:"description"`
	Message     string     `json:"message,omitempty" doc:"Personal message from the inviter"`
	SentAt      time.Time  `json:"sentAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	AcceptedAt  *time.Time `json:"acceptedAt,omitempty"`
	// AccountExists: the email already has an account, so the invitee signs
	// in to accept instead of creating a password.
	AccountExists bool `json:"accountExists"`
	// SignedInAs is the email of the browser's current session, if any.
	SignedInAs string `json:"signedInAs,omitempty"`
	// Password sign-up is unavailable when the domain enforces SSO.
	PasswordAllowed bool                     `json:"passwordAllowed"`
	PasswordPolicy  InvitationPasswordPolicy `json:"passwordPolicy"`
	Social          []string                 `json:"social" doc:"Social sign-in providers to join with: google, microsoft, github"`
	SSO             *SSOStart                `json:"sso,omitempty"`
	TwoFactor       InvitationTwoFactor      `json:"twoFactor"`
	// Language is the invitation's (the instance default when it has none):
	// the page reads in it unless the visitor picks another.
	Language string `json:"language,omitempty" example:"en"`
	// Agency is set when the workspace is a client of the instance's
	// agency: the page's client variant (invite_agency.go).
	Agency *InvitationAgency `json:"agency,omitempty"`
}

// InvitationPasswordPolicy is the password policy shown while creating a password.
type InvitationPasswordPolicy struct {
	MinLength     int  `json:"minLength"`
	BreachedCheck bool `json:"breachedCheck"`
}

// InvitationTwoFactor says whether the new member must set up two-factor.
type InvitationTwoFactor struct {
	Required   bool       `json:"required"`
	Reason     string     `json:"reason,omitempty"`
	GraceDays  int        `json:"graceDays"`
	GraceUntil *time.Time `json:"graceUntil,omitempty"`
}

func inviteNotFound(ctx context.Context) *api.Problem {
	return api.Err(http.StatusNotFound, "invite_not_found", say(ctx, mInviteNotFound))
}

func (s *Service) loadInvitation(ctx context.Context, token string) (store.GetInvitationViewRow, error) {
	inv, err := s.Q.GetInvitationView(ctx, authn.HashToken(token))
	if err != nil {
		if isNoRows(err) {
			return inv, inviteNotFound(ctx)
		}
		return inv, err
	}
	return inv, nil
}

func inviteStatus(inv store.GetInvitationViewRow, now time.Time) string {
	switch {
	case inv.AcceptedAt != nil:
		return "accepted"
	case inv.RevokedAt != nil:
		return "revoked"
	case !inv.ExpiresAt.After(now):
		return "expired"
	}
	return "pending"
}

// Invitation describes an invitation for its link.
func (s *Service) Invitation(ctx context.Context, token string) (InvitationView, error) {
	inv, err := s.loadInvitation(ctx, token)
	if err != nil {
		return InvitationView{}, err
	}
	st, err := s.policy(ctx)
	if err != nil {
		return InvitationView{}, err
	}
	now := s.now()
	l := langOf(ctx)
	sent := inv.CreatedAt
	if inv.LastSentAt != nil {
		sent = *inv.LastSentAt
	}
	v := InvitationView{
		Status: inviteStatus(inv, now), Email: inv.Email, WorkspaceName: inv.WorkspaceName, WorkspaceSlug: inv.WorkspaceSlug,
		InviterName: inv.InvitedByName, RoleKey: inv.RoleKey, RoleName: api.RoleName(l, inv.RoleKey, inv.RoleName), Message: inv.Message,
		SentAt: sent, ExpiresAt: inv.ExpiresAt, RevokedAt: inv.RevokedAt, AcceptedAt: inv.AcceptedAt,
		PasswordAllowed: methodAllowed(st, "password"),
		PasswordPolicy:  InvitationPasswordPolicy{MinLength: int(st.PasswordMinLength), BreachedCheck: st.BreachedCheck},
		Social:          []string{},
		Language:        st.DefaultLanguage,
	}
	if inv.Language != nil && *inv.Language != "" {
		v.Language = *inv.Language
	}
	if v.InviterName == "" {
		v.InviterName, v.InviterUnknown = mInviterUnknown.Text(l, i18n.Args{"workspace": inv.WorkspaceName}), true
	}
	perms := authz.Effective(inv.RoleKey, inv.RoleBuiltin, inv.RolePermissions)
	scope, listNames := s.scopeOf(ctx, inv.WorkspaceID, inv.Scope)
	v.Scope = scope
	v.Description = roleDescription(l, inv.RoleKey, inv.WorkspaceName, perms, listNames, inv.WorkspaceApprovalPolicy)
	if _, err := s.Q.GetUserByEmail(ctx, inv.Email); err == nil {
		v.AccountExists = true
	}
	if u := api.UserFrom(ctx); u != nil {
		v.SignedInAs = u.Email
	}
	if p, _ := s.enforcingProvider(ctx, inv.Email); p != nil {
		v.PasswordAllowed = false
		start := ssoStart(p, inv.Email)
		start.StartURL += "&invite=" + urlEscape(token)
		v.SSO = &start
	} else if methodAllowed(st, "social") {
		var social map[string]struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.Unmarshal(st.SocialProviders, &social)
		for _, k := range []string{"google", "microsoft", "github"} {
			if social[k].Enabled {
				v.Social = append(v.Social, k)
			}
		}
	}
	v.TwoFactor = inviteTwoFactor(l, st, perms)
	v.Agency = s.invitationAgency(ctx, inv.WorkspaceID, inv.RoleKey)
	return v, nil
}

func inviteTwoFactor(l i18n.Lang, st store.InstanceSetting, perms authz.Set) InvitationTwoFactor {
	t := InvitationTwoFactor{GraceDays: int(st.TwofaGraceDays)}
	switch {
	case st.TwofaEnforcement == "everyone":
		t.Required, t.Reason = true, mTwoFactorEveryone.Text(l, nil)
	case st.TwofaEnforcement == "sensitive" && perms.HasSensitive():
		t.Required, t.Reason = true, sensitiveReason(l, perms)
	}
	return t
}

// scopeOf describes a member scope ("Beta Program list only") in the
// request's language and returns the list names.
func (s *Service) scopeOf(ctx context.Context, workspaceID uuid.UUID, raw []byte) (string, []string) {
	var sc authz.Scope
	if json.Unmarshal(raw, &sc) != nil || sc.IsZero() {
		return "", nil
	}
	var names []string
	if len(sc.Lists) > 0 {
		if rows, err := s.Q.ListNamesOfLists(ctx, store.ListNamesOfListsParams{WorkspaceID: workspaceID, Ids: sc.Lists}); err == nil {
			for _, r := range rows {
				names = append(names, r.Name)
			}
		}
	}
	return describeScope(langOf(ctx), names, sc.SenderDomains), names
}

// InvitationAcceptInput creates the account of a new member. Existing accounts
// accept while signed in and send no body.
type InvitationAcceptInput struct {
	Name     string `json:"name,omitempty" maxLength:"80"`
	Password string `json:"password,omitempty" maxLength:"1024"`
	// Passkey joins without a password: the account is created and signed
	// in, and the browser adds a passkey next (POST /me/passkeys). Until it
	// has one, a password reset link is the way back in.
	Passkey bool `json:"passkey,omitempty"`
}

// InvitationAccepted is returned after joining.
type InvitationAccepted struct {
	Workspace     string `json:"workspace" doc:"Workspace slug"`
	WorkspaceName string `json:"workspaceName"`
	RoleName      string `json:"roleName"`
	Email         string `json:"email"`
	// NewAccount is true when joining created the account (and signed in).
	NewAccount bool `json:"newAccount"`
	// HasSecondFactor tells the app whether to offer passkey and 2FA setup.
	HasSecondFactor bool                `json:"hasSecondFactor"`
	TwoFactor       InvitationTwoFactor `json:"twoFactor"`
	Next            string              `json:"next"`
}

func inviteClosed(l i18n.Lang, status string, inv store.GetInvitationViewRow) *api.Problem {
	switch status {
	case "expired":
		return api.Err(http.StatusGone, "invite_expired", mClosedExpired.Text(l, i18n.Args{"inviter": orAdmin(l, inv)}))
	case "revoked":
		return api.Err(http.StatusGone, "invite_revoked", mClosedRevoked.Text(l, nil))
	default:
		return api.Err(http.StatusGone, "invite_accepted", mClosedAccepted.Text(l, nil))
	}
}

// orAdmin names who sent an invitation in the middle of a sentence.
func orAdmin(l i18n.Lang, inv store.GetInvitationViewRow) string {
	if inv.InvitedByName != "" {
		return inv.InvitedByName
	}
	return mInviterUnknownLower.Text(l, i18n.Args{"workspace": inv.WorkspaceName})
}

// AcceptInvitation joins the workspace. Signed-in people accept with their
// account (it must be the invited email); new people create an account
// with a password, which signs them in.
func (s *Service) AcceptInvitation(ctx context.Context, token string, in InvitationAcceptInput) (InvitationAccepted, *http.Cookie, error) {
	inv, err := s.loadInvitation(ctx, token)
	if err != nil {
		return InvitationAccepted{}, nil, err
	}
	now := s.now()
	l := langOf(ctx)
	if status := inviteStatus(inv, now); status != "pending" {
		return InvitationAccepted{}, nil, inviteClosed(l, status, inv)
	}
	st, err := s.policy(ctx)
	if err != nil {
		return InvitationAccepted{}, nil, err
	}
	perms := authz.Effective(inv.RoleKey, inv.RoleBuiltin, inv.RolePermissions)
	out := InvitationAccepted{Workspace: inv.WorkspaceSlug, WorkspaceName: inv.WorkspaceName, RoleName: api.RoleName(l, inv.RoleKey, inv.RoleName), Email: inv.Email, TwoFactor: inviteTwoFactor(l, st, perms),
		Next: "/w/" + inv.WorkspaceSlug}

	var user store.User
	signedIn := api.UserFrom(ctx)
	switch {
	case signedIn != nil && signedIn.Email != inv.Email:
		return InvitationAccepted{}, nil, api.Err(http.StatusConflict, "invite_wrong_account", say(ctx, mInviteWrongAccount, "invited", inv.Email, "current", signedIn.Email)).
			WithMeta("invited", inv.Email, "signedInAs", signedIn.Email)
	case signedIn != nil:
		user = *signedIn
	default:
		if _, err := s.Q.GetUserByEmail(ctx, inv.Email); err == nil {
			return InvitationAccepted{}, nil, api.Err(http.StatusConflict, "invite_sign_in", say(ctx, mInviteSignIn, "email", inv.Email)).WithMeta("email", inv.Email)
		}
		if p, _ := s.enforcingProvider(ctx, inv.Email); p != nil {
			return InvitationAccepted{}, nil, api.Err(http.StatusForbidden, "sso_required", say(ctx, mSSORequiredJoin, "domain", emailDomain(inv.Email), "provider", p.Name))
		}
		if in.Passkey && !methodAllowed(st, "passkey") || !in.Passkey && !methodAllowed(st, "password") {
			return InvitationAccepted{}, nil, api.Err(http.StatusForbidden, "method_disabled", say(ctx, mPasswordOffJoin))
		}
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return InvitationAccepted{}, nil, api.Invalid("body.name", say(ctx, mInviteName))
		}
		var hashed *string
		if !in.Passkey {
			if p := authn.PasswordProblem(in.Password, int(st.PasswordMinLength)); p != "" {
				return InvitationAccepted{}, nil, api.Invalid("body.password", p)
			}
			if st.BreachedCheck {
				if hit, err := s.Breach.Breached(ctx, in.Password); err == nil && hit {
					return InvitationAccepted{}, nil, api.Invalid("body.password", say(ctx, mBreached))
				}
			}
			hash, err := authn.HashPassword(in.Password)
			if err != nil {
				return InvitationAccepted{}, nil, err
			}
			hashed = &hash
		}
		tz := st.DefaultTimezone
		if tz == "" {
			tz = "UTC"
		}
		user = store.User{ID: uuid.Must(uuid.NewV7()), Email: inv.Email, Name: name, Timezone: tz, PasswordHash: hashed}
		out.NewAccount = true
	}
	if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: inv.WorkspaceID, UserID: user.ID}); err == nil {
		if m.Status == "active" {
			return InvitationAccepted{}, nil, api.Conflict(say(ctx, mAlreadyMember, "workspace", inv.WorkspaceName))
		}
		return InvitationAccepted{}, nil, api.Conflict(say(ctx, mMembershipDeactivated, "workspace", inv.WorkspaceName))
	}

	var cookie *http.Cookie
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if out.NewAccount {
			var setAt *time.Time
			if user.PasswordHash != nil {
				setAt = &now
			}
			created, err := q.CreateUser(ctx, store.CreateUserParams{
				ID: user.ID, Email: user.Email, Name: user.Name, Timezone: user.Timezone, PasswordHash: user.PasswordHash, PasswordSetAt: setAt,
				AvatarColor: avatarColorFor(user.Email),
			})
			if err != nil {
				return err
			}
			user = created
		}
		if err := s.joinWorkspace(ctx, tx, inv, user, out.NewAccount, out.TwoFactor.Required); err != nil {
			return err
		}
		if out.NewAccount {
			c, _, err := s.newSession(ctx, q, user, "password", false, true)
			if err != nil {
				return err
			}
			cookie = &c
			if err := q.RecordSignInSuccess(ctx, user.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return InvitationAccepted{}, nil, err
	}
	fresh, err := s.Q.GetUser(ctx, user.ID)
	if err == nil {
		user = fresh
	}
	m := s.methodsOf(ctx, user)
	out.HasSecondFactor = m.totp || m.passkeys > 0
	if out.TwoFactor.Required && !out.HasSecondFactor {
		out.TwoFactor.GraceUntil = user.TwofaGraceUntil
	}
	return out, cookie, nil
}

// joinWorkspace accepts a pending invitation for user inside tx: the
// membership with the invited role and scope, the two-factor grace period
// when the role needs it, the audit entry and the inviter's notification.
func (s *Service) joinWorkspace(ctx context.Context, tx pgx.Tx, inv store.GetInvitationViewRow, user store.User, newAccount, twoFactorRequired bool) error {
	q := store.New(tx)
	n, err := q.AcceptPendingInvitation(ctx, store.AcceptPendingInvitationParams{ID: inv.ID, AcceptedBy: &user.ID})
	if err != nil {
		return err
	}
	if n == 0 {
		return inviteClosed(langOf(ctx), "accepted", inv)
	}
	if _, err := q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: inv.WorkspaceID, UserID: user.ID, RoleID: inv.RoleID,
		Scope: inv.Scope, Source: "invite", InvitedBy: inv.InvitedBy}); err != nil {
		return err
	}
	if twoFactorRequired {
		if err := workspaces.EnsureTwoFAGrace(ctx, tx, user.ID); err != nil {
			return err
		}
	}
	wsID := inv.WorkspaceID
	actor := actorFor(user)
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "invitation.accept", TargetType: "invitation", TargetID: inv.ID.String(), TargetName: inv.Email, Actor: actor,
		After: map[string]any{"role": inv.RoleKey, "scope": inv.Scope}, Metadata: map[string]any{"new_account": newAccount}}); err != nil {
		return err
	}
	note := notify.Notification{WorkspaceID: &wsID, Kind: notify.KindInviteAccepted, Severity: "ok", Title: "Invite accepted",
		Summary: fmt.Sprintf("%s joined %s as %s", displayName(user), inv.WorkspaceName, inv.RoleName),
		Body:    fmt.Sprintf("%s joined %s as %s.", displayName(user), inv.WorkspaceName, inv.RoleName),
		Link:    "/w/" + inv.WorkspaceSlug + "/settings/team"}
	if inv.InvitedBy != nil && inv.InvitedByStatus == "active" {
		return notify.ToUser(ctx, tx, *inv.InvitedBy, note)
	}
	return notify.ToMembersWith(ctx, tx, inv.WorkspaceID, authz.TeamManage, note)
}

// InvitationRequestResult says whether the inviter was asked.
type InvitationRequestResult struct {
	Status  string `json:"status" enum:"sent,already_sent"`
	Inviter string `json:"inviter"`
}

// RequestNewInvite asks the inviter for a new invitation: for an expired
// one, or for the signed-in email when the invitation was for another.
// One request per invitation per day.
func (s *Service) RequestNewInvite(ctx context.Context, token string) (InvitationRequestResult, error) {
	inv, err := s.loadInvitation(ctx, token)
	if err != nil {
		return InvitationRequestResult{}, err
	}
	status := inviteStatus(inv, s.now())
	signedIn := api.UserFrom(ctx)
	other := signedIn != nil && signedIn.Email != inv.Email
	if status != "expired" && (status != "pending" || !other) {
		return InvitationRequestResult{}, api.Conflict(say(ctx, mInviteNoNeed))
	}
	res := InvitationRequestResult{Status: "sent", Inviter: orAdmin(langOf(ctx), inv)}
	wsID := inv.WorkspaceID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.MarkReinviteRequested(ctx, inv.ID)
		if err != nil {
			return err
		}
		if n == 0 {
			res.Status = "already_sent"
			return nil
		}
		note := notify.Notification{WorkspaceID: &wsID, Kind: notify.KindInviteRequested, Severity: "info", Title: "New invite requested",
			Link: "/w/" + inv.WorkspaceSlug + "/settings/team"}
		requester := inv.Email
		if other {
			requester = signedIn.Email
			note.Summary = fmt.Sprintf("%s asked to be invited to %s instead of %s", signedIn.Email, inv.WorkspaceName, inv.Email)
			note.Body = fmt.Sprintf("%s opened the invitation for %s while signed in and asked for an invitation to %s for their own email.", signedIn.Email, inv.Email, inv.WorkspaceName)
		} else {
			note.Summary = fmt.Sprintf("%s asked for a new invitation to %s", inv.Email, inv.WorkspaceName)
			note.Body = fmt.Sprintf("%s asked for a new invitation to %s; the last one expired on %s.", inv.Email, inv.WorkspaceName, inv.ExpiresAt.Format("Jan 2"))
		}
		if inv.InvitedBy != nil && inv.InvitedByStatus == "active" {
			if err := notify.ToUser(ctx, tx, *inv.InvitedBy, note); err != nil {
				return err
			}
		} else if err := notify.ToMembersWith(ctx, tx, inv.WorkspaceID, authz.TeamInvite, note); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "invitation.reinvite_request", TargetType: "invitation", TargetID: inv.ID.String(), TargetName: inv.Email,
			Actor: anonActor(requester), Metadata: map[string]any{"requested_for": requester}})
	})
	return res, err
}

func displayName(u store.User) string {
	if strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	return u.Email
}

// avatarColorFor picks a stable initials colour for a new account.
func avatarColorFor(email string) string {
	colors := []string{"info", "ok", "purple", "warn", "neutral"}
	h := 0
	for _, c := range email {
		h = (h*31 + int(c)) % 1000003
	}
	return colors[h%len(colors)]
}
