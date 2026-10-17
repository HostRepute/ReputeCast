package integrations

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// URLChecker runs the public URL check of Instance admin (DNS, TLS, reachable).
type URLChecker interface {
	CheckPublicURL(ctx context.Context, raw string) (accounts.URLCheck, error)
}

// Confirmations sends double opt-in emails (internal/audience).
type Confirmations interface {
	SendConfirmations(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) (int, error)
}

// Service is the integrations hub.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	Cfg   *config.Config
	// PublicURL is the instance's public URL (the saved setting, else http.public_url).
	PublicURL func() string
	URLCheck  URLChecker
	// Confirm sends double opt-in emails for actions that ask for them.
	Confirm Confirmations
	// AllowPrivate reports whether webhooks may reach private addresses
	// (webhooks.allow_private_networks).
	AllowPrivate func() bool
	// Languages turns a post's translations into its draft's language
	// versions (WordPress plugin 1.3.0).
	Languages DraftLanguages
	// Sources fill catalog groups from other domains (AI models from
	// internal/ai), keyed by group.
	Sources map[string]GroupSource
	// HTTP overrides the client of "Check now" and the release check (tests).
	HTTP *http.Client
	// CheckTimeout overrides how long "Check now" waits (10 s).
	CheckTimeout time.Duration
	Now          func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) base() string {
	if s.PublicURL == nil {
		return ""
	}
	return strings.TrimRight(s.PublicURL(), "/")
}

func (s *Service) allowPrivate() bool { return s.AllowPrivate != nil && s.AllowPrivate() }

func (s *Service) policy() netguard.Policy { return netguard.Policy{AllowPrivate: s.allowPrivate()} }

func (s *Service) client(timeout time.Duration) *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return s.policy().Client(timeout)
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Key implements api.Integrations: what an integration key may call.
func (s *Service) Key(ctx context.Context, id uuid.UUID) (*api.IntegrationKey, error) {
	r, err := s.Q.IntegrationKeyLimit(ctx, id)
	if err != nil {
		return nil, err
	}
	return &api.IntegrationKey{
		ID: r.ID, WorkspaceID: r.WorkspaceID, Kind: r.Kind, Name: Name(r.Kind),
		Operations: Operations(r.Kind, r.Items), Disconnected: r.DisconnectedAt != nil,
	}, nil
}

// Workflow is what X-ReputeCast-Workflow says about the calling workflow.
type Workflow struct {
	ID   string
	Name string
}

// Key identifies the workflow in the list: its id, else its name.
func (w Workflow) Key() string {
	if w.ID != "" {
		return w.ID
	}
	return w.Name
}

// String is the header as a person reads it, the name decoded:
// "Unsubscribes → Acme Cloud CRM; id=wf_2Lm8".
func (w Workflow) String() string {
	switch {
	case w.Name != "" && w.ID != "":
		return w.Name + "; id=" + w.ID
	case w.ID != "":
		return "id=" + w.ID
	}
	return w.Name
}

// ParseWorkflow reads "<name>; id=<id>" (the name may be percent-encoded, or
// given as name=…). Either part may be missing.
func ParseWorkflow(h string) Workflow {
	var w Workflow
	for i, part := range strings.Split(h, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		k = strings.ToLower(strings.TrimSpace(k))
		switch {
		case ok && k == "id":
			w.ID = clip(unquote(v), 120)
		case ok && (k == "name" || k == "name*"):
			w.Name = clip(decodeName(unquote(v)), 200)
		case i == 0:
			w.Name = clip(decodeName(part), 200)
		}
	}
	return w
}

func unquote(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
	}
	return v
}

// decodeName undoes percent-encoding (and RFC 8187's UTF-8” prefix).
func decodeName(v string) string {
	v = strings.TrimPrefix(strings.TrimPrefix(v, "UTF-8''"), "utf-8''")
	if strings.Contains(v, "%") {
		if d, err := url.PathUnescape(v); err == nil {
			return d
		}
	}
	return v
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && (s[cut]&0xC0) == 0x80 {
		cut--
	}
	return s[:cut]
}
