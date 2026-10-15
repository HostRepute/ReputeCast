// Package transactional is ReputeCast's transactional email (design
// expansion phase 1): templates with declared variables, named test data
// and published versions; the send, batch and render API with idempotency
// keys; the message log with its timeline; categories and their opt-outs;
// the Transactional settings; delivery alerts and retention.
//
// Transactional mail is its own stream: it goes through the Transactional
// route (internal/routing) with automatic failover, honours only the
// suppressions that apply to all mail (internal/suppressions), and is
// delivered by the sending engine's transactional outbox within each
// provider's shared rate, ahead of Broadcasts.
//
// Only published versions are ever sent. Sensitive variables are never
// stored: the message log and stored content show them as ••••••, and the
// sealed delivery payload is deleted once a provider accepted the message.
package transactional

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Error codes of the transactional API (brief §3.2). Each answer says in
// its detail what to fix.
const (
	CodeTemplateNotFound     = "template_not_found"
	CodeTemplateNotPublished = "template_not_published"
	CodeMissingVariables     = "missing_variables"
	CodeInvalidRecipient     = "invalid_recipient"
	CodeRecipientSuppressed  = "recipient_suppressed"
	CodeSenderNotVerified    = "sender_not_verified"
	CodeAttachmentsTooLarge  = "attachments_too_large"
	CodeRateLimited          = "rate_limited"
	CodeNoHealthyRoute       = "no_healthy_route"
	// CodeIdempotencyReused answers a key sent again with another body.
	CodeIdempotencyReused = "idempotency_key_reused"
	// CodeTemplateNotAllowed answers a key limited to other templates.
	CodeTemplateNotAllowed = "template_not_allowed"
	// CodeRecipientOptedOut is a batch item to an address that turned the
	// template's Optional category off.
	CodeRecipientOptedOut = "recipient_opted_out"
	// Template editing.
	CodeVersionConflict   = "version_conflict"
	CodeBreakingChange    = "breaking_change"
	CodeApprovalRequired  = "approval_required"
	CodeTemplateDoesntRun = "template_does_not_render"
	CodeAgentsNeverSend   = "agents_never_send"
)

// Mask replaces a sensitive value wherever it would be stored or shown.
const Mask = "••••••"

// Problem is an RFC 9457 answer of the transactional API: the API's problem
// plus the variables a call is missing ("missing": ["reset_url"]).
type Problem struct {
	api.Problem
	Missing []string `json:"missing,omitempty" doc:"Required variables the call didn't send (missing_variables)"`
}

func problem(status int, code, detail string) *Problem {
	return &Problem{Problem: *api.Err(status, code, detail)}
}

// retryAfter attaches a Retry-After header to an error.
func retryAfter(err error, seconds int) error {
	h := http.Header{}
	h.Set("Retry-After", itoa(max(seconds, 1)))
	return huma.ErrorWithHeaders(err, h)
}

// PublishGate decides whether a person may publish (or roll back to) a
// template version now. The approvals domain implements it with the
// optional rule "Publishing a transactional template needs approval"; a
// nil gate means no approval is needed.
type PublishGate interface {
	// Check says whether the publish may go ahead. It runs inside the
	// publish transaction (q is that transaction) and again, read-only, for
	// the publish dialog.
	Check(ctx context.Context, q *store.Queries, req PublishRequest) (PublishDecision, error)
	// Published runs inside the publish transaction once the version is
	// live, so an approval it relied on can be marked as used.
	Published(ctx context.Context, q *store.Queries, req PublishRequest, version int) error
}

// PublishRequest is what a gate decides on.
type PublishRequest struct {
	WorkspaceID uuid.UUID
	TemplateID  uuid.UUID
	TemplateKey string
	// Version is the version that becomes live: the next one for a publish,
	// an older one for a rollback.
	Version int
	// Rollback is true for "Publish again" and Undo.
	Rollback bool
	// DraftHash fingerprints the draft being published (content, variables,
	// sender and tracking); an approval covers one hash.
	DraftHash string
	Principal *authz.Principal
}

// PublishDecision is a gate's answer.
type PublishDecision struct {
	Allowed bool
	// NeedsApproval is true when the rule applies and no approval covers
	// this draft yet; Approvers names who can approve.
	NeedsApproval bool
	Approvers     []string
	// Reason is shown in the publish dialog ("The workspace rule … is on").
	Reason string
	// ApprovalID is the approval request that allowed the publish, if any.
	ApprovalID string
}

// Service holds the transactional domain's dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Links *linktoken.Signer
	URLs  publicurl.Builder
	Log   *slog.Logger
	Jobs  func() *jobs.Client
	// Checker runs the composer's content checks (links are not probed in
	// the editor).
	Checker *contentcheck.Checker
	// Gate decides whether publishing needs an approval (nil: never).
	Gate PublishGate
	// Accessibility adds the Accessibility group to the checks (nil: none).
	Accessibility AccessibilityChecker
	// Mail sends system mail (alert emails go through notify instead).
	Mail func() mailer.Sender
	// Outbox wakes the sending engine's transactional outbox in the
	// transaction that queued messages.
	Outbox func(ctx context.Context, tx pgx.Tx) error
	// ProviderLimit returns the largest message a provider accepts, in
	// bytes (attachments included); 0 when it doesn't say.
	ProviderLimit func(adapter string) int64
	Now           func() time.Time

	limiter keyLimiter
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// helpers names up to three active members who hold perm (Editors and
// Admins before the Owner, as the boards list them: "Ask Sofia Rossi,
// Marcus Chen or Linh Nguyen").
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID, perm authz.Permission, limit int) []string {
	out := []string{}
	rows, err := s.Q.ListMembersWithRoles(ctx, wsID)
	if err != nil {
		return out
	}
	var owners []string
	for _, r := range rows {
		if !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			continue
		}
		if r.RoleKey == authz.RoleOwner {
			owners = append(owners, r.Name)
			continue
		}
		out = append(out, r.Name)
	}
	out = append(out, owners...)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// joinNames is "Sofia Rossi, Marcus Chen or Linh Nguyen".
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " or " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

// deniedMessage is "Publishing needs transactional:publish. Ask …".
func deniedMessage(action string, perm authz.Permission, helpers []string) string {
	msg := action + " needs " + string(perm) + "."
	if len(helpers) > 0 {
		msg += " Ask " + joinNames(helpers) + "."
	}
	return msg
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
