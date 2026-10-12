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
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// SSOStart describes the identity provider a domain signs in with.
type SSOStart struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name" example:"Google Workspace"`
	Preset     string `json:"preset"`
	StartURL   string `json:"startUrl" doc:"Navigate here to start the OIDC flow"`
}

// IdentifyResult tells the sign-in page what to show after the email step.
type IdentifyResult struct {
	Email string `json:"email"`
	// Next is "password" for the password step or "sso" when the email's
	// domain enforces single sign-on.
	Next string    `json:"next" enum:"password,sso"`
	SSO  *SSOStart `json:"sso,omitempty"`
	// BreakGlass says Owners of this domain may still use a password. It is
	// shown for every enforced domain, so it reveals nothing about the account.
	BreakGlass bool `json:"breakGlass"`
}

// Identify is the first step of identifier-first sign-in. It never reveals
// whether an account exists.
func (s *Service) Identify(ctx context.Context, email string) (IdentifyResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	res := IdentifyResult{Email: email, Next: "password"}
	p, err := s.enforcingProvider(ctx, email)
	if err != nil {
		return res, err
	}
	if p != nil {
		res.Next = "sso"
		res.BreakGlass = true
		res.SSO = &SSOStart{ProviderID: p.ID.String(), Name: p.Name, Preset: p.Preset, StartURL: "/api/v1/auth/sso/" + p.ID.String() + "/start?login_hint=" + urlEscape(email)}
	}
	return res, nil
}

// SignInResult is returned by every sign-in step.
type SignInResult struct {
	Status string `json:"status" enum:"signed_in,two_factor"`
	// Challenge continues a two-factor sign-in (opaque, single use, 10 minutes).
	Challenge string   `json:"challenge,omitempty"`
	Methods   []string `json:"methods,omitempty" doc:"Second factors available: totp, passkey, recovery"`
	Email     string   `json:"email"`
	// Next is where the web app should go after signing in.
	Next string `json:"next,omitempty"`
	// RecoveryCodesLeft is set when a recovery code was used.
	RecoveryCodesLeft *int `json:"recoveryCodesLeft,omitempty"`
}

type twoFactorChallenge struct {
	Remember bool   `json:"remember"`
	Method   string `json:"method"`
}

// PasswordSignIn checks a password and either signs in or asks for a second factor.
func (s *Service) PasswordSignIn(ctx context.Context, email, password string, remember bool) (SignInResult, *http.Cookie, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	pol, err := s.policy(ctx)
	if err != nil {
		return SignInResult{}, nil, err
	}
	// Locked emails are refused before the password is even checked.
	if th, err := s.Q.GetSignInThrottle(ctx, email); err == nil && th.LockedUntil != nil && th.LockedUntil.After(s.now()) {
		return SignInResult{}, nil, lockMessage(*th.LockedUntil, th.Failures)
	}
	u, err := s.Q.GetUserByEmail(ctx, email)
	exists := err == nil
	if err != nil && !isNoRows(err) {
		return SignInResult{}, nil, err
	}
	if exists && u.Status == "deactivated" {
		return SignInResult{}, nil, api.Err(http.StatusForbidden, "account_deactivated", "This account was deactivated. Ask your instance admin to reactivate it; nothing was deleted.").
			WithMeta("deactivatedAt", u.DeactivatedAt)
	}
	if exists {
		if p, _ := s.enforcingProvider(ctx, email); p != nil && !u.BreakGlass {
			return SignInResult{}, nil, api.Err(http.StatusForbidden, "sso_required", email[strings.Index(email, "@")+1:]+" signs in with "+p.Name+". Password sign-in is turned off for this domain.").
				WithMeta("providerId", p.ID.String(), "provider", p.Name)
		}
	}
	ok := false
	if exists && u.PasswordHash != nil {
		ok, err = authn.VerifyPassword(password, *u.PasswordHash)
		if err != nil {
			s.Log.ErrorContext(ctx, "password hash unreadable", "user", u.ID, "err", err)
			ok = false
		}
	} else {
		authn.VerifyDummy(password)
	}
	if !ok {
		th, err := s.Q.RecordSignInThrottleFailure(ctx, store.RecordSignInThrottleFailureParams{Email: email, MaxAttempts: pol.LockoutAttempts, LockMinutes: pol.LockoutMinutes})
		if err != nil {
			return SignInResult{}, nil, err
		}
		if exists {
			_, _ = s.Q.RecordSignInFailure(ctx, store.RecordSignInFailureParams{ID: u.ID, MaxAttempts: pol.LockoutAttempts, LockMinutes: pol.LockoutMinutes})
		}
		_ = s.Audit.Log(ctx, audit.Event{Action: "auth.sign_in", Result: audit.Denied, Reason: "wrong password or unknown email", TargetType: "user", TargetName: email, Actor: anonActor(email),
			Metadata: map[string]any{"method": "password", "failures": th.Failures}})
		if th.LockedUntil != nil && th.LockedUntil.After(s.now()) {
			return SignInResult{}, nil, lockMessage(*th.LockedUntil, th.Failures)
		}
		left := pol.LockoutAttempts - th.Failures
		return SignInResult{}, nil, api.Err(http.StatusUnauthorized, api.CodeInvalidCredentials,
			fmt.Sprintf("The email or password is incorrect. %d %s left before a %d-minute lock.", left, plural(int(left), "attempt", "attempts"), pol.LockoutMinutes)).
			WithMeta("attemptsLeft", left, "lockMinutes", pol.LockoutMinutes)
	}
	_ = s.Q.ClearSignInThrottle(ctx, email)
	if u.PasswordHash != nil && authn.NeedsRehash(*u.PasswordHash) {
		if h, err := authn.HashPassword(password); err == nil {
			_ = s.Q.SetUserPassword(ctx, store.SetUserPasswordParams{ID: u.ID, PasswordHash: &h, MustChangePassword: u.MustChangePassword})
		}
	}
	methods := s.secondFactors(ctx, u)
	if len(methods) > 0 {
		token, hash := authn.NewToken()
		uid := u.ID
		if _, err := s.Q.CreateChallenge(ctx, store.CreateChallengeParams{
			ID:        uuid.Must(uuid.NewV7()),
			TokenHash: hash,
			Kind:      "two_factor",
			UserID:    &uid,
			Data:      mustJSON(twoFactorChallenge{Remember: remember, Method: "password"}),
			ExpiresAt: s.now().Add(10 * time.Minute),
		}); err != nil {
			return SignInResult{}, nil, err
		}
		return SignInResult{Status: "two_factor", Challenge: token, Methods: methods, Email: u.Email}, nil, nil
	}
	cookie, err := s.completeSignIn(ctx, u, "password", false, remember)
	if err != nil {
		return SignInResult{}, nil, err
	}
	return SignInResult{Status: "signed_in", Email: u.Email, Next: s.nextPath(ctx, u)}, &cookie, nil
}

func (s *Service) secondFactors(ctx context.Context, u store.User) []string {
	var m []string
	if u.TotpEnabledAt != nil {
		m = append(m, "totp")
	}
	if pk, err := s.Q.ListPasskeys(ctx, u.ID); err == nil && len(pk) > 0 {
		m = append(m, "passkey")
	}
	if len(m) > 0 {
		if n, err := s.Q.CountUnusedRecoveryCodes(ctx, u.ID); err == nil && n > 0 {
			m = append(m, "recovery")
		}
	}
	return m
}

func (s *Service) completeSignIn(ctx context.Context, u store.User, method string, mfa, remember bool) (http.Cookie, error) {
	var cookie http.Cookie
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.newSession(ctx, q, u, method, mfa, remember)
		if err != nil {
			return err
		}
		cookie = c
		if err := q.RecordSignInSuccess(ctx, u.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "auth.sign_in", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u),
			Metadata: map[string]any{"method": method, "mfa": mfa}})
	})
	return cookie, err
}

// TwoFactor completes a sign-in with an authenticator code or a recovery code.
func (s *Service) TwoFactor(ctx context.Context, challenge, method, code string) (SignInResult, *http.Cookie, error) {
	ch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(challenge), Kind: "two_factor"})
	if err != nil {
		return SignInResult{}, nil, api.Err(http.StatusUnauthorized, "challenge_expired", "This sign-in expired. Start again with your email.")
	}
	if ch.UserID == nil {
		return SignInResult{}, nil, api.Err(http.StatusUnauthorized, "challenge_expired", "This sign-in expired. Start again with your email.")
	}
	u, err := s.Q.GetUser(ctx, *ch.UserID)
	if err != nil || u.Status != "active" {
		return SignInResult{}, nil, api.Err(http.StatusUnauthorized, "challenge_expired", "This sign-in expired. Start again with your email.")
	}
	var data twoFactorChallenge
	_ = json.Unmarshal(ch.Data, &data)
	const maxAttempts = 5
	fail := func(detail string) (SignInResult, *http.Cookie, error) {
		n, _ := s.Q.BumpChallengeAttempts(ctx, ch.ID)
		left := maxAttempts - int(n)
		if left <= 0 {
			_ = s.Q.DeleteChallenge(ctx, ch.ID)
			_ = s.Audit.Log(ctx, audit.Event{Action: "auth.two_factor", Result: audit.Denied, Reason: "too many wrong codes", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u)})
			return SignInResult{}, nil, api.Err(http.StatusUnauthorized, "challenge_expired", "Too many wrong codes. Start again with your email.")
		}
		return SignInResult{}, nil, api.Err(http.StatusUnauthorized, "invalid_code", fmt.Sprintf("%s %d %s left.", detail, left, plural(left, "attempt", "attempts"))).WithMeta("attemptsLeft", left)
	}
	var left *int
	switch method {
	case "totp":
		if u.TotpSecretEnc == nil {
			return fail("That code didn't work.")
		}
		secret, err := s.Box.OpenString(u.TotpSecretEnc, totpAD(u.ID))
		if err != nil {
			return SignInResult{}, nil, err
		}
		if !authn.VerifyTOTP(secret, code, s.now()) {
			return fail("That code didn't work. Codes change every 30 seconds; check that your phone's clock is set automatically.")
		}
	case "recovery":
		if _, err := s.Q.UseRecoveryCode(ctx, store.UseRecoveryCodeParams{UserID: u.ID, CodeHash: hashRecovery(code)}); err != nil {
			return fail("That recovery code isn't valid or was already used.")
		}
		n, _ := s.Q.CountUnusedRecoveryCodes(ctx, u.ID)
		v := int(n)
		left = &v
		_ = s.Audit.Log(ctx, audit.Event{Action: "auth.recovery_code_used", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u), Metadata: map[string]any{"remaining": v}})
	default:
		return SignInResult{}, nil, api.BadRequest("Unknown method. Use totp or recovery; passkeys use /auth/passkey.")
	}
	_ = s.Q.DeleteChallenge(ctx, ch.ID)
	cookie, err := s.completeSignIn(ctx, u, data.Method+"+"+method, true, data.Remember)
	if err != nil {
		return SignInResult{}, nil, err
	}
	return SignInResult{Status: "signed_in", Email: u.Email, Next: s.nextPath(ctx, u), RecoveryCodesLeft: left}, &cookie, nil
}

// SignOut revokes the current session.
func (s *Service) SignOut(ctx context.Context) http.Cookie {
	if sess := api.SessionFrom(ctx); sess != nil {
		_, _ = s.Q.RevokeSession(ctx, store.RevokeSessionParams{ID: sess.ID, UserID: sess.UserID})
		_ = s.Audit.Log(ctx, audit.Event{Action: "auth.sign_out", TargetType: "session", TargetID: sess.ID.String()})
	}
	return s.clearCookie()
}

func anonActor(email string) *audit.Actor {
	return &audit.Actor{Type: "user", Name: email}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func urlEscape(s string) string {
	r := strings.NewReplacer("%", "%25", "+", "%2B", "@", "%40", "&", "%26", "=", "%3D", "#", "%23", " ", "%20", "?", "%3F")
	return r.Replace(s)
}
