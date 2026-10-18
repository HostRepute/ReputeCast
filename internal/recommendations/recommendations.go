// Package recommendations implements Grow › Recommendations (design
// expansion phase 5, M25; brief reputecast-design-addon-grow.md §3.8–3.9):
// the newsletters a workspace recommends on its opt-in confirmation page,
// in the Recommendations block and on the hosted recommendations page, its
// partner profile, and partner links with newsletters on other ReputeCast
// instances.
//
// A recommendation is a link to any subscribe page (readers subscribe
// there; clicks are counted) or a partner on another instance: readers
// press Subscribe on our pages and our server sends the partner a signed
// subscribe request; the partner runs its own double opt-in and tells us,
// signed, when the reader confirmed. We keep only the counts, never which
// reader confirmed. Partner links are set up with a single-use partner code
// and sign every message with a secret agreed over HTTPS (partnerproto;
// docs/grow/partners.md).
//
// Changes people make are audited. Subscribe requests are automatic and
// high volume: the partner request logs (90 days) are their record.
package recommendations

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "recommendations")

// Kinds of recommendation.
const (
	KindLink    = "link"
	KindPartner = "partner"
)

// Partner link directions: out, we recommend them (we send subscribe
// requests); in, they recommend us (we receive them).
const (
	DirOut = "out"
	DirIn  = "in"
)

// Partner link statuses.
const (
	LinkConnected = "connected"
	LinkPaused    = "paused"
	LinkInvalid   = "invalid"
	LinkEnded     = "ended"
)

// Timing.
const (
	// ConfirmPageMax recommendations show under "While you're here".
	ConfirmPageMax = 3
	// RequestRetryFor: a subscribe request the partner didn't answer is
	// retried with back-off for 24 hours, then marked failed.
	RequestRetryFor = 24 * time.Hour
	// OptInRetryFor: a partner signup's confirmation email waits for a
	// System provider for 3 days, like the forms' confirmations.
	OptInRetryFor = 72 * time.Hour
	// LogKept: the partner request logs keep 90 days.
	LogKept = 90 * 24 * time.Hour
	// PageValid: the personal link to the recommendations page.
	PageValid = 30 * 24 * time.Hour
	// UnreachableAfter: members with grow:connect hear about a partner whose
	// requests have failed this long.
	UnreachableAfter = 30 * time.Minute
)

// Inserter enqueues River jobs (the App's client).
type Inserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Service is the recommendations domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	Now   func() time.Time
	// Links signs the personal links of the recommendations page.
	Links *linktoken.Signer
	// Pages are a workspace's public page addresses (its custom domain when
	// verified).
	Pages func(ctx context.Context, ws *store.Workspace) (forms.PageLinks, error)
	// Instance is this instance's public base URL: partners call it back.
	Instance func() string
	// AllowPrivate lets partner instances and link pages resolve to private
	// and loopback addresses (development and tests; providers.allow_private_networks).
	AllowPrivate func() bool
	// HTTP overrides the client of outbound calls (tests).
	HTTP func() *http.Client
	// Audience adds partner signups (double opt-in) on the partner side.
	Audience *audience.Service
	// Forms builds the confirmation link and the publisher's branding.
	Forms *forms.Service
	// Mail sends the partner signups' confirmation emails on the System
	// stream (App.Mail, read at call time).
	Mail func() mailer.Sender
	// Media keeps the logos fetched from pages and partners.
	Media *media.Service
	// Jobs enqueues River jobs (a request to send now).
	Jobs func() Inserter
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

func (s *Service) allowPrivate() bool { return s.AllowPrivate != nil && s.AllowPrivate() }

func (s *Service) guard() netguard.Policy { return netguard.Policy{AllowPrivate: s.allowPrivate()} }

// client is the outbound client: through the SSRF guard, never following
// redirects (partners and pages answer directly).
func (s *Service) client(timeout time.Duration) *http.Client {
	if s.HTTP != nil {
		if c := s.HTTP(); c != nil {
			return c
		}
	}
	key := clientKey{timeout: timeout, private: s.allowPrivate()}
	if c, ok := clients.Load(key); ok {
		return c.(*http.Client)
	}
	c, _ := clients.LoadOrStore(key, s.guard().Client(timeout))
	return c.(*http.Client)
}

type clientKey struct {
	timeout time.Duration
	private bool
}

var clients sync.Map

func (s *Service) instance() string {
	if s.Instance == nil {
		return ""
	}
	return strings.TrimRight(s.Instance(), "/")
}

func (s *Service) pages(ctx context.Context, ws *store.Workspace) (forms.PageLinks, error) {
	if s.Pages == nil {
		return forms.LinksOn("http://localhost", ws.Slug, false), nil
	}
	return s.Pages(ctx, ws)
}

func (s *Service) text(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	return m.Text(i18n.LangFrom(ctx), args)
}

// hostOf is the host a member recognizes: "reputecast.opsweekly.example",
// "fabrikam.example/newsletter".
func hostOf(raw string, withPath bool) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	}
	h := strings.TrimPrefix(u.Host, "www.")
	if withPath {
		if p := strings.TrimRight(u.EscapedPath(), "/"); p != "" {
			h += p
		}
	}
	return h
}

// initials are a newsletter's avatar letters: "OW" for Ops Weekly.
func initials(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		r := []rune(w)
		if len(r) == 0 || strings.EqualFold(w, "the") && len(strings.Fields(name)) > 1 {
			continue
		}
		out = append(out, []rune(strings.ToUpper(string(r[0])))...)
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 0 {
		return "?"
	}
	return string(out)
}

// nextMidnightUTC is when a partner's day resets.
func nextMidnightUTC(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
}

func midnightUTC(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func dayOf(t time.Time, ws *store.Workspace) time.Time {
	loc := time.UTC
	if ws != nil && ws.Timezone != "" {
		if l, err := time.LoadLocation(ws.Timezone); err == nil {
			loc = l
		}
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

func ptr[T any](v T) *T { return &v }

func uuidOrNil(raw string) *uuid.UUID {
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}
