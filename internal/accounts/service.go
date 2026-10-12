// Package accounts implements sign-in (identifier first, password, two-factor,
// recovery codes), sessions, the signed-in person's profile and security
// settings, password resets and the first sign-in setup of an instance.
package accounts

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the dependencies of the accounts domain.
type Service struct {
	Pool           *pgxpool.Pool
	Q              *store.Queries
	Audit          *audit.Logger
	Box            *secrets.Box
	Cfg            *config.Config
	Mail           mailer.Sender
	Log            *slog.Logger
	Breach         *authn.BreachChecker
	Now            func() time.Time
	OnPolicyChange func()
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// policy loads the instance authentication policy.
func (s *Service) policy(ctx context.Context) (store.InstanceSetting, error) {
	return s.Q.GetInstanceSettings(ctx)
}

// totpAD is the associated data binding a user's TOTP secret to the user.
func totpAD(userID uuid.UUID) string { return "user:" + userID.String() + ":totp" }

// cookie builds the session cookie. maxAge 0 makes a browser-session cookie.
func (s *Service) cookie(token string, expires time.Time, persistent bool) http.Cookie {
	c := http.Cookie{
		Name:     s.Cfg.Auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies(),
		SameSite: http.SameSiteLaxMode,
	}
	if persistent {
		c.Expires = expires
		c.MaxAge = int(time.Until(expires).Seconds())
	}
	return c
}

func (s *Service) clearCookie() http.Cookie {
	return http.Cookie{Name: s.Cfg.Auth.CookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.secureCookies(), SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(0, 0)}
}

func (s *Service) secureCookies() bool {
	return strings.HasPrefix(s.Cfg.HTTP.PublicURL, "https://") || (s.Cfg.Auth.CookieSecure && !s.Cfg.IsDevelopment())
}

// newSession stores a session and returns its cookie.
func (s *Service) newSession(ctx context.Context, q *store.Queries, u store.User, method string, mfa, remember bool) (http.Cookie, error) {
	pol, err := s.policy(ctx)
	if err != nil {
		return http.Cookie{}, err
	}
	lifetime := pol.SessionLifetime
	if !remember && pol.SessionIdle < lifetime {
		lifetime = pol.SessionIdle
	}
	token, hash := authn.NewToken()
	req := audit.RequestFrom(ctx)
	expires := s.now().Add(lifetime)
	if _, err := q.CreateSession(ctx, store.CreateSessionParams{
		ID:          uuid.Must(uuid.NewV7()),
		UserID:      u.ID,
		TokenHash:   hash,
		AuthMethod:  method,
		MfaVerified: mfa,
		Ip:          req.IP,
		UserAgent:   req.UserAgent,
		ExpiresAt:   expires,
	}); err != nil {
		return http.Cookie{}, err
	}
	return s.cookie(token, expires, remember), nil
}

// actorFor builds an audit actor for a user who is not yet signed in.
func actorFor(u store.User) *audit.Actor {
	id := u.ID
	name := u.Name
	if name == "" {
		name = u.Email
	}
	return &audit.Actor{Type: authz.ActorUser, ID: &id, Name: name}
}

func hashRecovery(code string) []byte {
	sum := sha256.Sum256([]byte(authn.NormalizeRecoveryCode(code)))
	return sum[:]
}

// emailDomain returns the part after @.
func emailDomain(email string) string {
	_, d, _ := strings.Cut(email, "@")
	return strings.ToLower(d)
}

// enforcingProvider returns the enabled SSO provider that enforces the
// email's domain, if any.
func (s *Service) enforcingProvider(ctx context.Context, email string) (*store.SsoProvider, error) {
	domain := emailDomain(email)
	if domain == "" {
		return nil, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, name, preset FROM sso_providers WHERE status = 'enabled' AND $1 = ANY(enforce_domains) LIMIT 1`, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var p store.SsoProvider
		if err := rows.Scan(&p.ID, &p.Name, &p.Preset); err != nil {
			return nil, err
		}
		return &p, nil
	}
	return nil, rows.Err()
}

// nextPath tells the web app where to go after signing in.
func (s *Service) nextPath(ctx context.Context, u store.User) string {
	if u.MustChangePassword {
		return "/setup"
	}
	if u.IsInstanceAdmin {
		if st, err := s.policy(ctx); err == nil && st.SetupCompletedAt == nil {
			return "/setup"
		}
	}
	wss, err := s.Q.ListUserWorkspaces(ctx, u.ID)
	if err == nil && len(wss) > 0 {
		return "/w/" + wss[0].Slug
	}
	if u.IsInstanceAdmin {
		return "/setup"
	}
	return "/account"
}

var errNotFound = errors.New("not found")

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// lockMessage formats the lockout explanation.
func lockMessage(until time.Time, failures int32) *api.Problem {
	return api.Err(http.StatusTooManyRequests, api.CodeLocked, "Sign-in is locked after too many attempts. Try again after "+until.Format("15:04")+", reset your password, or use a passkey.").
		WithMeta("lockedUntil", until, "failures", failures)
}
