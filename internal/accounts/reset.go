package accounts

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/passwordreuse"
	"github.com/hostrepute/reputecast/internal/store"
)

// ResetTTL is how long a password reset link works.
const ResetTTL = 15 * time.Minute

// resetResendGap stops reset emails from being sent in a loop to one inbox.
const resetResendGap = time.Minute

// PasswordResetRequest is the answer to "Email me a reset link". It never
// reveals whether an account exists, except that an email domain signs in
// with single sign-on (which the sign-in page shows anyway).
type PasswordResetRequest struct {
	// Status: sent (generic: a link is on its way if the account exists),
	// email_unavailable (no System-stream provider yet: an instance admin
	// resets it with the CLI command) or sso (the password is managed by the
	// identity provider).
	Status string `json:"status" enum:"sent,email_unavailable,sso"`
	Email  string `json:"email"`
	// Command is the server command for email_unavailable.
	Command string    `json:"command,omitempty" example:"reputecast user reset-password --email aisha@acme.io"`
	SSO     *SSOStart `json:"sso,omitempty"`
	// ExpiresInMinutes is how long the emailed link works.
	ExpiresInMinutes int `json:"expiresInMinutes"`
}

// CreateResetToken stores a one-time reset token for a user and returns it.
// Earlier unused tokens stop working.
func CreateResetToken(ctx context.Context, q *store.Queries, userID uuid.UUID, source string, now time.Time) (string, time.Time, error) {
	if err := q.InvalidatePasswordResets(ctx, userID); err != nil {
		return "", time.Time{}, err
	}
	token, hash := authn.NewToken()
	expires := now.Add(ResetTTL)
	if _, err := q.CreatePasswordResetFrom(ctx, store.CreatePasswordResetFromParams{
		ID: uuid.Must(uuid.NewV7()), UserID: userID, TokenHash: hash, ExpiresAt: expires, Source: source,
	}); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// ResetLink builds the link of a reset token.
func ResetLink(publicURL, token string) string {
	return strings.TrimRight(publicURL, "/") + "/reset-password/" + token
}

// RequestPasswordReset emails a reset link (when a System-stream provider
// exists) and explains the alternatives otherwise.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (PasswordResetRequest, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	res := PasswordResetRequest{Status: "sent", Email: email, ExpiresInMinutes: int(ResetTTL / time.Minute)}
	u, err := s.Q.GetUserByEmail(ctx, email)
	exists := err == nil && u.Status == "active"
	if err != nil && !isNoRows(err) {
		return res, err
	}
	// Single sign-on manages the password for enforced domains (except
	// break-glass Owners, who keep one) and for accounts without a password.
	p, err := s.enforcingProvider(ctx, email)
	if err != nil {
		return res, err
	}
	if p != nil && (!exists || !u.BreakGlass || u.PasswordHash == nil) {
		start := ssoStart(p, email)
		res.Status, res.SSO = "sso", &start
		return res, nil
	}
	if !s.mail().SystemAvailable(ctx, nil) {
		res.Status = "email_unavailable"
		res.Command = "reputecast user reset-password --email " + email
		return res, nil
	}
	if !exists {
		return res, nil
	}
	// An account that only signs in through single sign-on has no password
	// to reset. The answer stays the generic one, which mustn't tell that the
	// email has an account; the email says how they sign in.
	if u.PasswordHash == nil {
		if sso := s.ssoIdentityProvider(ctx, u.ID); sso != nil {
			s.ssoReminder(ctx, u, sso)
			return res, nil
		}
	}
	if last, err := s.Q.LastPasswordResetAt(ctx, u.ID); err == nil && s.now().Sub(last) < resetResendGap {
		return res, nil
	}
	token, _, err := CreateResetToken(ctx, s.Q, u.ID, "email", s.now())
	if err != nil {
		return res, err
	}
	link := ResetLink(s.publicURL(ctx), token)
	st, _ := s.policy(ctx)
	msg := mailer.Message{
		To: u.Email, ToName: u.Name, Kind: "password_reset",
		Subject: "Reset your " + st.Name + " password",
		Text: fmt.Sprintf("Someone asked to reset the password of %s on %s.\n\nSet a new password: %s\n\nThe link works once and expires in 15 minutes. If it wasn't you, ignore this email; your password stays the same.\n",
			u.Email, st.Name, link),
		HTML: fmt.Sprintf(`<p style="font:14px/1.5 sans-serif">Someone asked to reset the password of <b>%s</b> on %s.</p><p style="font:14px sans-serif"><a href="%s">Set a new password</a></p><p style="font:13px/1.5 sans-serif;color:#5B6472">The link works once and expires in 15 minutes. If it wasn't you, ignore this email; your password stays the same.</p>`,
			html.EscapeString(u.Email), html.EscapeString(st.Name), html.EscapeString(link)),
	}
	provider, err := s.mail().SendSystem(ctx, msg)
	if err != nil {
		// The answer stays "sent" so a failing provider doesn't reveal which
		// emails have accounts; the failure is in the server log.
		_ = s.Q.InvalidatePasswordResets(ctx, u.ID)
		s.Log.WarnContext(ctx, "password reset email not sent", "user", u.ID, "err", err)
		return res, nil
	}
	_ = s.Audit.Log(ctx, audit.Event{Action: "account.password_reset_request", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u),
		Metadata: map[string]any{"via": provider}})
	return res, nil
}

// ssoReminder tells someone who asked to reset a password they don't have
// how they sign in: at most once per resetResendGap, like reset emails, and
// recorded in the audit log (which is also what throttles it).
func (s *Service) ssoReminder(ctx context.Context, u store.User, p *store.SsoProvider) {
	const action = "account.sso_reminder"
	if last, err := s.Q.LastAuditAt(ctx, store.LastAuditAtParams{Action: action, TargetID: u.ID.String()}); err == nil && s.now().Sub(last) < resetResendGap {
		return
	}
	st, _ := s.policy(ctx)
	link := s.publicURL(ctx) + "/sign-in"
	msg := mailer.Message{
		To: u.Email, ToName: u.Name, Kind: "password_reset_sso",
		Subject: "How you sign in to " + st.Name,
		Text: fmt.Sprintf("Someone asked to reset the password of %s on %s. This account has no password: it signs in with %s.\n\nSign in: %s\n\nIf it wasn't you, ignore this email.\n",
			u.Email, st.Name, p.Name, link),
		HTML: fmt.Sprintf(`<p style="font:14px/1.5 sans-serif">Someone asked to reset the password of <b>%s</b> on %s. This account has no password: it signs in with %s.</p><p style="font:14px sans-serif"><a href="%s">Sign in</a></p><p style="font:13px/1.5 sans-serif;color:#5B6472">If it wasn't you, ignore this email.</p>`,
			html.EscapeString(u.Email), html.EscapeString(st.Name), html.EscapeString(p.Name), html.EscapeString(link)),
	}
	provider, err := s.mail().SendSystem(ctx, msg)
	if err != nil {
		s.Log.WarnContext(ctx, "sign-in reminder not sent", "user", u.ID, "err", err)
		return
	}
	_ = s.Audit.Log(ctx, audit.Event{Action: action, TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u),
		Metadata: map[string]any{"via": provider, "signs_in_with": p.Name}})
}

// ssoIdentityProvider returns the SSO provider of one of the user's
// identities, for accounts that only sign in through SSO.
func (s *Service) ssoIdentityProvider(ctx context.Context, userID uuid.UUID) *store.SsoProvider {
	ids, err := s.Q.ListIdentitiesWithProvider(ctx, userID)
	if err != nil {
		return nil
	}
	for _, i := range ids {
		if i.Kind == "sso" && i.SsoProviderID != nil {
			return &store.SsoProvider{ID: *i.SsoProviderID, Name: i.ProviderName, Preset: i.ProviderPreset}
		}
	}
	return nil
}

// PasswordResetInfo describes a usable reset link.
type PasswordResetInfo struct {
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expiresAt"`
	MinLength int       `json:"minLength"`
	Breached  bool      `json:"breachedCheck" doc:"New passwords are checked against known breaches"`
}

func resetExpired() *api.Problem {
	return api.Err(http.StatusGone, "reset_expired", "This reset link is older than 15 minutes or was already used. Request a new one; your current password still works until you change it.")
}

// CheckPasswordReset tells whether a reset link still works.
func (s *Service) CheckPasswordReset(ctx context.Context, token string) (PasswordResetInfo, error) {
	r, err := s.Q.GetUsablePasswordReset(ctx, authn.HashToken(token))
	if err != nil || r.UserStatus != "active" {
		return PasswordResetInfo{}, resetExpired()
	}
	st, err := s.policy(ctx)
	if err != nil {
		return PasswordResetInfo{}, err
	}
	return PasswordResetInfo{Email: r.Email, ExpiresAt: r.ExpiresAt, MinLength: int(st.PasswordMinLength), Breached: st.BreachedCheck}, nil
}

// PasswordResetDone is returned after setting a new password.
type PasswordResetDone struct {
	Email string `json:"email"`
}

// ResetPassword sets a new password with a one-time link. Every session is
// signed out and a sign-in lockout is cleared; the next sign-in still asks
// for the second factor.
func (s *Service) ResetPassword(ctx context.Context, token, password string) (PasswordResetDone, error) {
	r, err := s.Q.GetUsablePasswordReset(ctx, authn.HashToken(token))
	if err != nil || r.UserStatus != "active" {
		return PasswordResetDone{}, resetExpired()
	}
	st, err := s.policy(ctx)
	if err != nil {
		return PasswordResetDone{}, err
	}
	if p := authn.PasswordProblem(password, int(st.PasswordMinLength)); p != "" {
		return PasswordResetDone{}, api.Invalid("body.password", p)
	}
	if st.BreachedCheck {
		if hit, err := s.Breach.Breached(ctx, password); err == nil && hit {
			return PasswordResetDone{}, api.Invalid("body.password", "This password appears in known data breaches. Choose another one.")
		}
	}
	hash, err := authn.HashPassword(password)
	if err != nil {
		return PasswordResetDone{}, err
	}
	u, err := s.Q.GetUser(ctx, r.UserID)
	if err != nil {
		return PasswordResetDone{}, err
	}
	if used, err := passwordreuse.Used(ctx, s.Q, u.ID, u.PasswordHash, password, st.PasswordReuse); err != nil {
		return PasswordResetDone{}, err
	} else if used {
		return PasswordResetDone{}, api.Invalid("body.password", passwordreuse.Message(st.PasswordReuse))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.ConsumePasswordReset(ctx, authn.HashToken(token)); err != nil {
			return resetExpired()
		}
		if err := q.SetUserPassword(ctx, store.SetUserPasswordParams{ID: u.ID, PasswordHash: &hash, MustChangePassword: false}); err != nil {
			return err
		}
		if err := q.InvalidatePasswordResets(ctx, u.ID); err != nil {
			return err
		}
		if _, err := q.RevokeAllUserSessions(ctx, u.ID); err != nil {
			return err
		}
		if err := q.ClearUserLockout(ctx, u.ID); err != nil {
			return err
		}
		if err := q.ClearSignInThrottle(ctx, u.Email); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.password_reset", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Email, Actor: actorFor(u),
			Metadata: map[string]any{"link_from": r.Source}})
	})
	if err != nil {
		return PasswordResetDone{}, err
	}
	return PasswordResetDone{Email: u.Email}, nil
}
