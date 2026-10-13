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
		start := ssoStart(p, email)
		res.SSO = &start
	}
	return res, nil
}

// SignInResult is returned by every sign-in step.
type SignInResult struct {
	Status string `json:"status" enum:"signed_in,two_factor"`
	// Challenge continues a two-factor sign-in (opaque, single use, 10 minutes).
	Challenge string   `json:"challenge,omitempty"`
	Methods   []string `json:"methods,omitempty" doc:"Second factors available: totp, passkey, recovery"`
	// PasskeyHint names the person's most recently used passkey for the
	// "Confirm with your passkey" step ("MacBook Pro · Touch ID").
	PasskeyHint string `json:"passkeyHint,omitempty"`
	Email       string `json:"email"`
	// Next is where the web app should go after signing in.
	Next string `json:"next,omitempty"`
	// RecoveryCodesLeft is set when a recovery code was used, and with the
	// two-factor step ("Each code works once. 7 of 10 remain.").
	RecoveryCodesLeft *int `json:"recoveryCodesLeft,omitempty"`
	// RecoveryCodesTotal is how many codes the last set had.
	RecoveryCodesTotal *int `json:"recoveryCodesTotal,omitempty"`
}

type twoFactorChallenge struct {
	Remember bool   `json:"remember"`
	Method   string `json:"method"`
	// Extra is added to the sign-in's audit entry (an SSO sign-in that
	// still needs the person's own second factor keeps its provider).
	Extra map[string]any `json:"extra,omitempty"`
}

// PasswordSignIn checks a password and either signs in or asks for a second factor.
func (s *Service) PasswordSignIn(ctx context.Context, email, password string, remember bool) (SignInResult, *http.Cookie, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	pol, err := s.policy(ctx)
	if err != nil {
		return SignInResult{}, nil, err
	}
	// Locked emails are refused before the password is even checked.
	failedBefore := int32(0)
	if th, err := s.Q.GetSignInThrottle(ctx, email); err == nil {
		if th.LockedUntil != nil && th.LockedUntil.After(s.now()) {
			return SignInResult{}, nil, lockMessage(*th.LockedUntil, th.Failures)
		}
		failedBefore = th.Failures
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
	// extra marks break-glass sign-ins under domain enforcement in the audit log.
	var extra map[string]any
	if exists {
		if p, _ := s.enforcingProvider(ctx, email); p != nil {
			if !u.BreakGlass {
				return SignInResult{}, nil, api.Err(http.StatusForbidden, "sso_required", email[strings.Index(email, "@")+1:]+" signs in with "+p.Name+". Password sign-in is turned off for this domain.").
					WithMeta("providerId", p.ID.String(), "provider", p.Name, "preset", p.Preset)
			}
			extra = map[string]any{"break_glass": true, "enforced_by": p.Name}
		}
		if !methodAllowed(pol, "password") && !u.BreakGlass {
			return SignInResult{}, nil, api.Err(http.StatusForbidden, "method_disabled", "Password sign-in is turned off on this instance. Use a passkey or single sign-on.")
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
		_ = s.Audit.Log(ctx, audit.Event{Action: "signin.failed", Result: audit.Denied, Reason: "wrong password or unknown email", TargetType: "user", TargetName: email, Actor: anonActor(email),
			Metadata: map[string]any{"method": "password", "failures": th.Failures, "max_attempts": pol.LockoutAttempts}})
		if th.LockedUntil != nil && th.LockedUntil.After(s.now()) {
			return SignInResult{}, nil, lockMessage(*th.LockedUntil, th.Failures)
		}
		left := pol.LockoutAttempts - th.Failures
		return SignInResult{}, nil, api.Err(http.StatusUnauthorized, api.CodeInvalidCredentials,
			fmt.Sprintf("The email or password is incorrect. %d %s left before a %d-minute lock.", left, plural(int(left), "attempt", "attempts"), pol.LockoutMinutes)).
			WithMeta("attemptsLeft", left, "lockMinutes", pol.LockoutMinutes)
	}
	_ = s.Q.ClearSignInThrottle(ctx, email)
	// The audit log shows "Password · after 3 failed attempts".
	if failedBefore > 0 {
		if extra == nil {
			extra = map[string]any{}
		}
		extra["failed_before"] = failedBefore
	}
	if u.PasswordHash != nil && authn.NeedsRehash(*u.PasswordHash) {
		if h, err := authn.HashPassword(password); err == nil {
			_ = s.Q.SetUserPassword(ctx, store.SetUserPasswordParams{ID: u.ID, PasswordHash: &h, MustChangePassword: u.MustChangePassword})
		}
	}
	methods, hint := s.secondFactors(ctx, u)
	if len(methods) > 0 {
		var left, total *int
		if rc, err := s.Q.RecoveryCodeStats(ctx, u.ID); err == nil && rc.Total > 0 {
			l, t := int(rc.Remaining), int(rc.Total)
			left, total = &l, &t
		}
		token, hash := authn.NewToken()
		uid := u.ID
		if _, err := s.Q.CreateChallenge(ctx, store.CreateChallengeParams{
			ID:        uuid.Must(uuid.NewV7()),
			TokenHash: hash,
			Kind:      "two_factor",
			UserID:    &uid,
			Data:      mustJSON(twoFactorChallenge{Remember: remember, Method: "password", Extra: extra}),
			ExpiresAt: s.now().Add(10 * time.Minute),
		}); err != nil {
			return SignInResult{}, nil, err
		}
		return SignInResult{Status: "two_factor", Challenge: token, Methods: methods, PasskeyHint: hint, Email: u.Email, RecoveryCodesLeft: left, RecoveryCodesTotal: total}, nil, nil
	}
	cookie, err := s.completeSignInWith(ctx, u, "password", false, remember, nil, extra)
	if err != nil {
		return SignInResult{}, nil, err
	}
	return SignInResult{Status: "signed_in", Email: u.Email, Next: s.nextPath(ctx, u)}, &cookie, nil
}

// secondFactors lists the second factors a person can use, and the name of
// their most recently used passkey.
func (s *Service) secondFactors(ctx context.Context, u store.User) ([]string, string) {
	var m []string
	hint := ""
	if u.TotpEnabledAt != nil {
		m = append(m, "totp")
	}
	if pk, err := s.Q.ListPasskeys(ctx, u.ID); err == nil && len(pk) > 0 {
		m = append(m, "passkey")
		last := pk[0]
		for _, p := range pk[1:] {
			if p.LastUsedAt != nil && (last.LastUsedAt == nil || p.LastUsedAt.After(*last.LastUsedAt)) {
				last = p
			}
		}
		hint = last.Name
	}
	if len(m) > 0 {
		if n, err := s.Q.CountUnusedRecoveryCodes(ctx, u.ID); err == nil && n > 0 {
			m = append(m, "recovery")
		}
	}
	return m, hint
}

// completeSignIn creates the session and records the sign-in. passkeyID
// names the passkey the person used, if any.
func (s *Service) completeSignIn(ctx context.Context, u store.User, method string, mfa, remember bool, passkeyID *uuid.UUID) (http.Cookie, error) {
	return s.completeSignInWith(ctx, u, method, mfa, remember, passkeyID, nil)
}

// completeSignInWith is completeSignIn with extra audit metadata.
func (s *Service) completeSignInWith(ctx context.Context, u store.User, method string, mfa, remember bool, passkeyID *uuid.UUID, extra map[string]any) (http.Cookie, error) {
	meta := map[string]any{"method": method, "mfa": mfa}
	for k, v := range extra {
		if _, taken := meta[k]; !taken {
			meta[k] = v
		}
	}
	var cookie http.Cookie
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, sid, err := s.newSession(ctx, q, u, method, mfa, remember)
		if err != nil {
			return err
		}
		cookie = c
		if passkeyID != nil {
			if err := q.SetSessionPasskey(ctx, store.SetSessionPasskeyParams{ID: sid, PasskeyID: passkeyID}); err != nil {
				return err
			}
		}
		if err := q.RecordSignInSuccess(ctx, u.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "signin.succeeded", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u),
			Metadata: meta})
	})
	return cookie, err
}

// loadTwoFactor finds a pending two-factor sign-in.
func (s *Service) loadTwoFactor(ctx context.Context, challenge string) (store.AuthChallenge, store.User, twoFactorChallenge, error) {
	expired := api.Err(http.StatusUnauthorized, "challenge_expired", "This sign-in expired. Start again with your email.")
	ch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(challenge), Kind: "two_factor"})
	if err != nil || ch.UserID == nil {
		return store.AuthChallenge{}, store.User{}, twoFactorChallenge{}, expired
	}
	u, err := s.Q.GetUser(ctx, *ch.UserID)
	if err != nil || u.Status != "active" {
		return store.AuthChallenge{}, store.User{}, twoFactorChallenge{}, expired
	}
	var data twoFactorChallenge
	_ = json.Unmarshal(ch.Data, &data)
	return ch, u, data, nil
}

const maxTwoFactorAttempts = 5

// failTwoFactor counts a wrong second factor; after five the sign-in ends.
func (s *Service) failTwoFactor(ctx context.Context, ch store.AuthChallenge, u store.User, detail string) error {
	n, _ := s.Q.BumpChallengeAttempts(ctx, ch.ID)
	left := maxTwoFactorAttempts - int(n)
	if left <= 0 {
		_ = s.Q.DeleteChallenge(ctx, ch.ID)
		_ = s.Audit.Log(ctx, audit.Event{Action: "auth.two_factor", Result: audit.Denied, Reason: "too many wrong codes", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u)})
		return api.Err(http.StatusUnauthorized, "challenge_expired", "Too many wrong codes. Start again with your email.")
	}
	return api.Err(http.StatusUnauthorized, "invalid_code", fmt.Sprintf("%s %d %s left.", detail, left, plural(left, "attempt", "attempts"))).WithMeta("attemptsLeft", left)
}

// TwoFactor completes a sign-in with an authenticator code or a recovery code.
func (s *Service) TwoFactor(ctx context.Context, challenge, method, code string) (SignInResult, *http.Cookie, error) {
	ch, u, data, err := s.loadTwoFactor(ctx, challenge)
	if err != nil {
		return SignInResult{}, nil, err
	}
	var left, total *int
	switch method {
	case "totp":
		if u.TotpSecretEnc == nil {
			return SignInResult{}, nil, s.failTwoFactor(ctx, ch, u, "That code didn't work.")
		}
		secret, err := s.Box.OpenString(u.TotpSecretEnc, totpAD(u.ID))
		if err != nil {
			return SignInResult{}, nil, err
		}
		if !authn.VerifyTOTP(secret, code, s.now()) {
			return SignInResult{}, nil, s.failTwoFactor(ctx, ch, u, "That code didn't work. Codes change every 30 seconds; check that your phone's clock is set automatically.")
		}
	case "recovery":
		if _, err := s.Q.UseRecoveryCode(ctx, store.UseRecoveryCodeParams{UserID: u.ID, CodeHash: hashRecovery(code)}); err != nil {
			return SignInResult{}, nil, s.failTwoFactor(ctx, ch, u, "That recovery code isn't valid or was already used.")
		}
		rc, _ := s.Q.RecoveryCodeStats(ctx, u.ID)
		v, tot := int(rc.Remaining), int(rc.Total)
		left, total = &v, &tot
		_ = s.Audit.Log(ctx, audit.Event{Action: "auth.recovery_code_used", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u), Metadata: map[string]any{"remaining": v}})
	default:
		return SignInResult{}, nil, api.BadRequest("Unknown method. Use totp or recovery; passkeys use /auth/two-factor/passkey.")
	}
	_ = s.Q.DeleteChallenge(ctx, ch.ID)
	cookie, err := s.completeSignInWith(ctx, u, data.Method+"+"+method, true, data.Remember, nil, data.Extra)
	if err != nil {
		return SignInResult{}, nil, err
	}
	return SignInResult{Status: "signed_in", Email: u.Email, Next: s.nextPath(ctx, u), RecoveryCodesLeft: left, RecoveryCodesTotal: total}, &cookie, nil
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
