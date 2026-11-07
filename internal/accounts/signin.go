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
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
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
			return SignInResult{}, nil, lockMessage(ctx, *th.LockedUntil, th.Failures)
		}
		failedBefore = th.Failures
	}
	u, err := s.Q.GetUserByEmail(ctx, email)
	exists := err == nil
	if err != nil && !isNoRows(err) {
		return SignInResult{}, nil, err
	}
	// Domain enforcement and a turned-off method answer the same whether or
	// not the account exists, so neither reveals who has one; only break-glass
	// Owners get past them. extra marks their sign-ins in the audit log.
	breakGlass := exists && u.BreakGlass
	var extra map[string]any
	if p, _ := s.enforcingProvider(ctx, email); p != nil {
		if !breakGlass {
			return SignInResult{}, nil, api.Err(http.StatusForbidden, "sso_required", say(ctx, mSSORequired, "domain", email[strings.Index(email, "@")+1:], "provider", p.Name)).
				WithMeta("providerId", p.ID.String(), "provider", p.Name, "preset", p.Preset)
		}
		extra = map[string]any{"break_glass": true, "enforced_by": p.Name}
	}
	if !methodAllowed(pol, "password") && !breakGlass {
		return SignInResult{}, nil, api.Err(http.StatusForbidden, "method_disabled", say(ctx, mPasswordOff))
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
			return SignInResult{}, nil, lockMessage(ctx, *th.LockedUntil, th.Failures)
		}
		left := pol.LockoutAttempts - th.Failures
		return SignInResult{}, nil, api.Err(http.StatusUnauthorized, api.CodeInvalidCredentials, say(ctx, mWrongPassword, "left", left, "minutes", pol.LockoutMinutes)).
			WithMeta("attemptsLeft", left, "lockMinutes", pol.LockoutMinutes)
	}
	// A deactivated account says so only to someone who knows its password.
	if u.Status == "deactivated" {
		return SignInResult{}, nil, api.Err(http.StatusForbidden, "account_deactivated", say(ctx, mDeactivated)).
			WithMeta("deactivatedAt", u.DeactivatedAt)
	}
	// The email's failures stay until the whole sign-in succeeds
	// (completeSignInWith): wrong second-factor codes count with them.
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
		// Only a whole sign-in clears the email's failures: a right password
		// followed by wrong codes still counts toward the lockout.
		if err := q.ClearSignInThrottle(ctx, u.Email); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "signin.succeeded", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u),
			Metadata: meta})
	})
	return cookie, err
}

// loadTwoFactor finds a pending two-factor sign-in. While the email's
// sign-in is locked, no challenge goes on, whenever it was made: the
// lockout counts wrong codes as well as wrong passwords.
func (s *Service) loadTwoFactor(ctx context.Context, challenge string) (store.AuthChallenge, store.User, twoFactorChallenge, error) {
	expired := api.Err(http.StatusUnauthorized, "challenge_expired", say(ctx, mSignInExpired))
	ch, err := s.Q.GetChallenge(ctx, store.GetChallengeParams{TokenHash: authn.HashToken(challenge), Kind: "two_factor"})
	if err != nil || ch.UserID == nil {
		return store.AuthChallenge{}, store.User{}, twoFactorChallenge{}, expired
	}
	u, err := s.Q.GetUser(ctx, *ch.UserID)
	if err != nil || u.Status != "active" {
		return store.AuthChallenge{}, store.User{}, twoFactorChallenge{}, expired
	}
	if th, err := s.Q.GetSignInThrottle(ctx, u.Email); err == nil && th.LockedUntil != nil && th.LockedUntil.After(s.now()) {
		_ = s.Q.DeleteChallenge(ctx, ch.ID)
		return store.AuthChallenge{}, store.User{}, twoFactorChallenge{}, lockMessage(ctx, *th.LockedUntil, th.Failures)
	}
	var data twoFactorChallenge
	_ = json.Unmarshal(ch.Data, &data)
	return ch, u, data, nil
}

const maxTwoFactorAttempts = 5

// countAttempt counts an attempt at a challenge before its answer is
// checked, so parallel guesses can't go over maxTwoFactorAttempts, and
// returns the attempt's number. Once none are left the challenge ends.
func (s *Service) countAttempt(ctx context.Context, ch store.AuthChallenge) (int32, error) {
	n, err := s.Q.AccountsCountChallengeAttempt(ctx, store.AccountsCountChallengeAttemptParams{ID: ch.ID, MaxAttempts: maxTwoFactorAttempts})
	if isNoRows(err) {
		_ = s.Q.DeleteChallenge(ctx, ch.ID)
		return 0, api.Err(http.StatusUnauthorized, "challenge_expired", say(ctx, mTooManyCodes))
	}
	return n, err
}

// failTwoFactor answers a wrong second factor, attempt number attempt of
// the challenge, which ends after maxTwoFactorAttempts. A wrong code (not
// a passkey, which can't be guessed) also counts like a wrong password
// toward the email's sign-in lockout, which outlasts every challenge: a
// known password doesn't buy fresh guesses with each new sign-in.
func (s *Service) failTwoFactor(ctx context.Context, ch store.AuthChallenge, attempt int32, u store.User, method string, code bool, wrong i18n.Msg) error {
	left := maxTwoFactorAttempts - int(attempt)
	if code {
		pol, err := s.policy(ctx)
		if err != nil {
			return err
		}
		th, err := s.Q.RecordSignInThrottleFailure(ctx, store.RecordSignInThrottleFailureParams{Email: u.Email, MaxAttempts: pol.LockoutAttempts, LockMinutes: pol.LockoutMinutes})
		if err != nil {
			return err
		}
		_, _ = s.Q.RecordSignInFailure(ctx, store.RecordSignInFailureParams{ID: u.ID, MaxAttempts: pol.LockoutAttempts, LockMinutes: pol.LockoutMinutes})
		_ = s.Audit.Log(ctx, audit.Event{Action: "signin.failed", Result: audit.Denied, Reason: "wrong second-factor code", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u),
			Metadata: map[string]any{"method": method, "failures": th.Failures, "max_attempts": pol.LockoutAttempts}})
		if th.LockedUntil != nil && th.LockedUntil.After(s.now()) {
			_ = s.Q.DeleteChallenge(ctx, ch.ID)
			_ = s.Audit.Log(ctx, audit.Event{Action: "auth.two_factor", Result: audit.Denied, Reason: "too many wrong codes, sign-in locked", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u)})
			return lockMessage(ctx, *th.LockedUntil, th.Failures)
		}
		left = min(left, int(pol.LockoutAttempts-th.Failures))
	}
	if int(attempt) >= maxTwoFactorAttempts {
		_ = s.Q.DeleteChallenge(ctx, ch.ID)
		_ = s.Audit.Log(ctx, audit.Event{Action: "auth.two_factor", Result: audit.Denied, Reason: "too many wrong codes", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u)})
		return api.Err(http.StatusUnauthorized, "challenge_expired", say(ctx, mTooManyCodes))
	}
	return api.Err(http.StatusUnauthorized, "invalid_code", say(ctx, wrong, "left", left)).WithMeta("attemptsLeft", left)
}

// takeChallenge uses up a challenge after a right answer: of two requests
// with one, only the first goes on to a session.
func (s *Service) takeChallenge(ctx context.Context, ch store.AuthChallenge) error {
	n, err := s.Q.AccountsTakeChallenge(ctx, ch.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		return api.Err(http.StatusUnauthorized, "challenge_expired", say(ctx, mSignInExpired))
	}
	return nil
}

// TwoFactor completes a sign-in with an authenticator code or a recovery code.
func (s *Service) TwoFactor(ctx context.Context, challenge, method, code string) (SignInResult, *http.Cookie, error) {
	if method != "totp" && method != "recovery" {
		return SignInResult{}, nil, api.BadRequest(say(ctx, msgs.M("problem.unknownMethod")))
	}
	ch, u, data, err := s.loadTwoFactor(ctx, challenge)
	if err != nil {
		return SignInResult{}, nil, err
	}
	attempt, err := s.countAttempt(ctx, ch)
	if err != nil {
		return SignInResult{}, nil, err
	}
	wrong := func(m i18n.Msg) (SignInResult, *http.Cookie, error) {
		return SignInResult{}, nil, s.failTwoFactor(ctx, ch, attempt, u, data.Method+"+"+method, true, m)
	}
	var left, total *int
	switch method {
	case "totp":
		if u.TotpSecretEnc == nil {
			return wrong(mWrongCode)
		}
		secret, err := s.Box.OpenString(u.TotpSecretEnc, totpAD(u.ID))
		if err != nil {
			return SignInResult{}, nil, err
		}
		step, ok := authn.MatchTOTP(secret, code, s.now())
		if !ok {
			return wrong(mWrongTOTP)
		}
		// A code works once: its time step must come after the last one
		// accepted, so a code someone saw can't be replayed in its 30 s.
		if n, err := s.Q.AccountsUseTOTPStep(ctx, store.AccountsUseTOTPStepParams{ID: u.ID, Step: step}); err != nil {
			return SignInResult{}, nil, err
		} else if n == 0 {
			return wrong(mTOTPReused)
		}
	case "recovery":
		if _, err := s.Q.UseRecoveryCode(ctx, store.UseRecoveryCodeParams{UserID: u.ID, CodeHash: hashRecovery(code)}); err != nil {
			return wrong(mWrongRecovery)
		}
		rc, _ := s.Q.RecoveryCodeStats(ctx, u.ID)
		v, tot := int(rc.Remaining), int(rc.Total)
		left, total = &v, &tot
		_ = s.Audit.Log(ctx, audit.Event{Action: "auth.recovery_code_used", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u), Metadata: map[string]any{"remaining": v}})
	}
	if err := s.takeChallenge(ctx, ch); err != nil {
		return SignInResult{}, nil, err
	}
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

func urlEscape(s string) string {
	r := strings.NewReplacer("%", "%25", "+", "%2B", "@", "%40", "&", "%26", "=", "%3D", "#", "%23", " ", "%20", "?", "%3F")
	return r.Replace(s)
}
