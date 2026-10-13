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
	// Metadata is added to the auth.sign_in audit entry (provider, amr…).
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
		return ExternalResult{}, api.Err(http.StatusForbidden, "account_deactivated", "This account was deactivated. Ask your instance admin to reactivate it; nothing was deleted.")
	}
	_ = s.Q.ClearSignInThrottle(ctx, u.Email)
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
	Name         string
	Username     string
	Details      json.RawMessage
	MFA          bool
	Metadata     map[string]any
}

// AcceptInvitationExternal joins the workspace of a pending invitation with
// an identity an IdP verified: the invitation link proved the invitee owns
// the invited email, and the identity's email must be the same. New people
// get an account without a password; the identity is linked either way.
func (s *Service) AcceptInvitationExternal(ctx context.Context, token string, id ExternalIdentity) (ExternalResult, error) {
	inv, err := s.loadInvitation(ctx, token)
	if err != nil {
		return ExternalResult{}, err
	}
	now := s.now()
	if status := inviteStatus(inv, now); status != "pending" {
		return ExternalResult{}, inviteClosed(status, inv)
	}
	if !strings.EqualFold(strings.TrimSpace(id.Email), inv.Email) {
		return ExternalResult{}, api.Err(http.StatusConflict, "invite_wrong_account",
			fmt.Sprintf("This invite is for %s, but your %s account is %s. Continue with the invited account, or ask for an invite for %s.", inv.Email, id.ProviderName, id.Email, id.Email)).
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
	if !newAccount {
		if user.Status != "active" {
			return ExternalResult{}, api.Err(http.StatusForbidden, "account_deactivated", "This account was deactivated. Ask your instance admin to reactivate it; nothing was deleted.")
		}
		if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: inv.WorkspaceID, UserID: user.ID}); err == nil {
			if m.Status == "active" {
				return ExternalResult{}, api.Conflict("You're already a member of " + inv.WorkspaceName + ".")
			}
			return ExternalResult{}, api.Conflict("Your membership in " + inv.WorkspaceName + " was deactivated. Ask an admin to reactivate it.")
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
		}
		if _, err := q.GetIdentity(ctx, store.GetIdentityParams{Kind: id.Kind, SsoProviderID: id.ProviderID, Subject: id.Subject}); isNoRows(err) {
			ident, err := q.CreateIdentity(ctx, store.CreateIdentityParams{ID: uuid.Must(uuid.NewV7()), UserID: user.ID, Kind: id.Kind, SsoProviderID: id.ProviderID, Subject: id.Subject, Email: strings.ToLower(id.Email)})
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE user_identities SET username = $2, details = $3, last_used_at = now() WHERE id = $1`, ident.ID, id.Username, details); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return s.joinWorkspace(ctx, tx, inv, user, newAccount, inviteTwoFactor(st, perms).Required)
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
