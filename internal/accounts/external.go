package accounts

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sign-ins completed by an identity provider (SSO or social sign-in, see
// internal/sso) end here, so they get the same session, cookie, audit entry
// and two-factor rules as a password sign-in.

// ExternalSignIn is a sign-in an identity provider confirmed.
type ExternalSignIn struct {
	User store.User
	// Method is "sso", "google", "microsoft" or "github".
	Method string
	// MFA: the IdP asserted a second factor (amr) or is trusted to enforce it.
	MFA      bool
	Remember bool
	// Metadata is added to the signin.succeeded audit entry (provider, amr…).
	Metadata map[string]any
}

// ExternalResult tells the callback where to send the browser.
type ExternalResult struct {
	// Cookie is the session cookie (nil while a second factor is pending).
	Cookie *http.Cookie
	// Next is where to go after signing in.
	Next string
	// Challenge is set when the person still has to confirm their own
	// second factor because the IdP didn't assert one.
	Challenge   string
	Methods     []string
	PasskeyHint string
}

// CompleteExternalSignIn signs a person in after an identity provider
// confirmed them. When the IdP asserted no second factor and the person
// has one (authenticator or passkey), it asks for it like a password
// sign-in does; otherwise the session is created.
func (s *Service) CompleteExternalSignIn(ctx context.Context, in ExternalSignIn) (ExternalResult, error) {
	u := in.User
	if u.Status != "active" {
		return ExternalResult{}, api.Err(http.StatusForbidden, "account_deactivated", say(ctx, mDeactivated))
	}
	// The email's sign-in failures clear once the session exists
	// (completeSignInWith), not while the person's own second factor is
	// still to come: wrong codes there count toward the lockout.
	if !in.MFA {
		methods, hint := s.secondFactors(ctx, u)
		if len(methods) > 0 {
			token, hash := authn.NewToken()
			uid := u.ID
			if _, err := s.Q.CreateChallenge(ctx, store.CreateChallengeParams{
				ID: uuid.Must(uuid.NewV7()), TokenHash: hash, Kind: "two_factor", UserID: &uid,
				Data:      mustJSON(twoFactorChallenge{Remember: in.Remember, Method: in.Method, Extra: in.Metadata}),
				ExpiresAt: s.now().Add(10 * time.Minute),
			}); err != nil {
				return ExternalResult{}, err
			}
			return ExternalResult{Challenge: token, Methods: methods, PasskeyHint: hint}, nil
		}
	}
	cookie, err := s.completeSignInWith(ctx, u, in.Method, in.MFA, in.Remember, nil, in.Metadata)
	if err != nil {
		return ExternalResult{}, err
	}
	return ExternalResult{Cookie: &cookie, Next: s.nextPath(ctx, u)}, nil
}

// NextPath is where a person goes after signing in.
func (s *Service) NextPath(ctx context.Context, u store.User) string { return s.nextPath(ctx, u) }

// ExternalIdentity is an identity an IdP verified for an invitee.
type ExternalIdentity struct {
	Kind       string // google, microsoft, github, sso
	ProviderID *uuid.UUID
	// ProviderName is "Google", "GitHub" or the SSO provider's name.
	ProviderName string
	Subject      string
	Email        string
	// EmailVerified: the IdP vouches for Email the way its sign-in takes it
	// (internal/sso decides for each provider). Only then may joining create
	// an account for the email.
	EmailVerified bool
	Name          string
	Username      string
	Details       json.RawMessage
	MFA           bool
	Metadata      map[string]any
}

// AcceptInvitationExternal joins the workspace of a pending invitation with
// an identity an IdP verified, whose email must be the invited one. An
// invitation link is easy to come by (anyone can create a workspace and
// invite any address), so the IdP's word counts only as far as sign-in
// takes it: a new account needs an email the IdP vouches for, and an
// account that already uses the email accepts only with an identity it
// already signs in with. Otherwise the person signs in first and accepts
// while signed in, as with a password. Domains under single sign-on join
// through their provider, as they sign in. New people get an account
// without a password and the identity linked.
func (s *Service) AcceptInvitationExternal(ctx context.Context, token string, id ExternalIdentity) (ExternalResult, error) {
	inv, err := s.loadInvitation(ctx, token)
	if err != nil {
		return ExternalResult{}, err
	}
	if status, err := s.invitationStatus(ctx, s.Q, inv); err != nil {
		return ExternalResult{}, err
	} else if status != "pending" {
		return ExternalResult{}, inviteClosed(langOf(ctx), status, inv)
	}
	if !strings.EqualFold(strings.TrimSpace(id.Email), inv.Email) {
		return ExternalResult{}, api.Err(http.StatusConflict, "invite_wrong_account",
			say(ctx, mInviteWrongIdentity, "invited", inv.Email, "provider", id.ProviderName, "current", id.Email)).
			WithMeta("invited", inv.Email, "signedInAs", id.Email)
	}
	st, err := s.policy(ctx)
	if err != nil {
		return ExternalResult{}, err
	}
	perms := authz.Effective(inv.RoleKey, inv.RoleBuiltin, inv.RolePermissions)
	user, err := s.Q.GetUserByEmail(ctx, inv.Email)
	newAccount := isNoRows(err)
	if err != nil && !newAccount {
		return ExternalResult{}, err
	}
	linked, err := s.Q.GetIdentity(ctx, store.GetIdentityParams{Kind: id.Kind, SsoProviderID: id.ProviderID, Subject: id.Subject})
	known := err == nil
	if err != nil && !isNoRows(err) {
		return ExternalResult{}, err
	}
	// Only the enforcing provider joins people of its domains; break-glass
	// Owners keep their other sign-ins, as on the sign-in page.
	if p, err := s.enforcingProvider(ctx, inv.Email); err != nil {
		return ExternalResult{}, err
	} else if p != nil && (id.Kind != "sso" || id.ProviderID == nil || *id.ProviderID != p.ID) && (newAccount || !user.BreakGlass) {
		return ExternalResult{}, api.Err(http.StatusForbidden, "sso_required", say(ctx, mSSORequiredWith, "domain", emailDomain(inv.Email), "provider", p.Name)).
			WithMeta("providerId", p.ID.String(), "provider", p.Name, "preset", p.Preset)
	}
	if newAccount {
		if known {
			return ExternalResult{}, api.Err(http.StatusConflict, "identity_in_use", say(ctx, mIdentityInUse, "provider", id.ProviderName))
		}
		if !id.EmailVerified {
			return ExternalResult{}, api.Err(http.StatusForbidden, "email_not_verified", say(ctx, mInviteUnverified, "provider", id.ProviderName, "email", inv.Email))
		}
	} else {
		// Whoever controls this IdP account hasn't shown they own the
		// account yet: they sign in to it first. Deactivation is said only
		// to them, after that.
		if !known || linked.UserID != user.ID {
			return ExternalResult{}, api.Err(http.StatusConflict, "invite_sign_in", say(ctx, mInviteSignIn, "email", inv.Email)).WithMeta("email", inv.Email)
		}
		if user.Status != "active" {
			return ExternalResult{}, api.Err(http.StatusForbidden, "account_deactivated", say(ctx, mDeactivated))
		}
		if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: inv.WorkspaceID, UserID: user.ID}); err == nil {
			if m.Status == "active" {
				return ExternalResult{}, api.Conflict(say(ctx, mAlreadyMember, "workspace", inv.WorkspaceName))
			}
			return ExternalResult{}, api.Conflict(say(ctx, mMembershipDeactivated, "workspace", inv.WorkspaceName))
		}
	}
	details := id.Details
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if newAccount {
			tz := st.DefaultTimezone
			if tz == "" {
				tz = "UTC"
			}
			name := strings.TrimSpace(id.Name)
			if name == "" {
				name = strings.SplitN(inv.Email, "@", 2)[0]
			}
			created, err := q.CreateUser(ctx, store.CreateUserParams{ID: uuid.Must(uuid.NewV7()), Email: inv.Email, Name: name, Timezone: tz, AvatarColor: avatarColorFor(inv.Email)})
			if err != nil {
				return err
			}
			user = created
			if _, err := q.SSOCreateIdentity(ctx, store.SSOCreateIdentityParams{ID: uuid.Must(uuid.NewV7()), UserID: user.ID, Kind: id.Kind, SsoProviderID: id.ProviderID,
				Subject: id.Subject, Email: strings.ToLower(id.Email), Username: id.Username, Details: details}); err != nil {
				return err
			}
		}
		return s.joinWorkspace(ctx, tx, inv, user, newAccount, inviteTwoFactor(langOf(ctx), st, perms).Required)
	})
	if err != nil {
		return ExternalResult{}, err
	}
	meta := map[string]any{"invitation": inv.ID.String(), "workspace": inv.WorkspaceName}
	for k, v := range id.Metadata {
		meta[k] = v
	}
	res, err := s.CompleteExternalSignIn(ctx, ExternalSignIn{User: user, Method: id.Kind, MFA: id.MFA, Remember: true, Metadata: meta})
	if err != nil {
		return ExternalResult{}, err
	}
	if res.Cookie != nil {
		res.Next = "/w/" + inv.WorkspaceSlug
	}
	return res, nil
}
