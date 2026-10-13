// Package instance is the Instance admin area (add-on spec 3.3, 3.13,
// 5.10): every account across workspaces with deactivate, force sign-out,
// two-factor reset, instance admin and password resets; workspaces with
// create, delete (a 7-day restore hold, then a River job) and restore; the
// authentication policy; instance settings with the public URL check; and
// version & updates with the opt-out release check. Everything here needs
// an instance admin and every change is audited.
package instance

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/sso"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the instance admin domain's dependencies.
type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Audit    *audit.Logger
	Box      *secrets.Box
	Cfg      *config.Config
	Log      *slog.Logger
	Accounts *accounts.Service
	SSO      *sso.Service
	// Mail returns the system mailer (set by a start hook after wiring).
	Mail func() mailer.Sender
	// Jobs returns the River client (it exists only after wiring).
	Jobs func() *jobs.Client
	// OnPolicyChange drops the cached authentication policy (App.InvalidatePolicy).
	OnPolicyChange func()
	// OnPublicURL applies a new saved public URL (before and after; empty
	// means http.public_url) and moves provider webhooks to it.
	OnPublicURL func(ctx context.Context, before, after string) (*PublicURLWebhooks, error)
	// HTTP fetches releases and checks the public URL. Nil uses a client
	// with a 10 s timeout.
	HTTP *http.Client
	// ReleasesURL overrides the GitHub API endpoint of the latest release.
	ReleasesURL string
	Now         func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) mail() mailer.Sender {
	if s.Mail == nil {
		return mailer.Unavailable{}
	}
	if m := s.Mail(); m != nil {
		return m
	}
	return mailer.Unavailable{}
}

func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s *Service) policyChanged() {
	if s.OnPolicyChange != nil {
		s.OnPolicyChange()
	}
}

// admin is the instance admin making the request.
func admin(ctx context.Context) (*store.User, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return nil, api.Err(http.StatusUnauthorized, api.CodeUnauthenticated, "Sign in to continue.")
	}
	return u, nil
}

// publicURL is where people open ReputeCast, without a trailing slash.
func (s *Service) publicURL(st store.InstanceSetting) string {
	if st.PublicUrl != "" {
		return strings.TrimRight(st.PublicUrl, "/")
	}
	if s.Cfg != nil {
		return strings.TrimRight(s.Cfg.HTTP.PublicURL, "/")
	}
	return ""
}

func displayName(name, email string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return email
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
