// Package sponsors is ReputeCast's sponsorships (design expansion phase 6,
// M27): a sponsor library, slots (a placement in each issue of a list),
// bookings by issue and slot with their creative, the payment status
// recorded by hand, and the sponsor reports, shared as read-only links.
//
// Sponsorships work without a payment provider: ReputeCast sends no
// invoices and moves no money. The Sponsor block of an issue fills itself
// from the issue's booking and is always labelled "Sponsored"; Send
// Readiness won't let an issue go out while the creative is missing. A
// sponsor report counts only human opens and real clicks of the
// recipients who got the block, and is final 72 hours after the send.
//
// Every change a person makes is audited (the "sponsors.*" actions); MCP
// agents can read bookings and propose one for a person to confirm.
package sponsors

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

// msgs holds the server's own text: problems, notifications and the public
// sponsor report's words.
var msgs = i18n.MustLoad(messagesFS, "sponsors")

// FinalAfter is when a sponsor report's results are final: 72 hours after
// the send, as for campaign reports.
const FinalAfter = 72 * time.Hour

// Service is the sponsorships domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	URLs  publicurl.Builder
	Log   *slog.Logger
	Now   func() time.Time
	// Images stores a creative's uploaded logo or image in the media
	// library and returns its address (nil: uploads are refused).
	Images ImageStore
}

// ImageStore keeps uploaded images (internal/media).
type ImageStore interface {
	Upload(ctx context.Context, ws *store.Workspace, p *authz.Principal, name string, data []byte) (string, error)
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

func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// text is a message of the catalog in the request's language.
func text(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(lang(ctx), args)
}

// Text is one of the shared sponsor report's words in English.
func Text(key string, args i18n.Args) string { return msgs.M(key).Text(i18n.Default, args) }

// TextIn is one of the shared sponsor report's words in the page's
// language (internal/publicpages).
func TextIn(l i18n.Lang, key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }

// EnglishText reports whether one of the catalog's messages is written in
// English for a language (it isn't translated yet): English grammar the
// code applies (a placement's name in lower case) holds only then.
func EnglishText(l i18n.Lang, key string) bool {
	if l.Locale == i18n.English || l.Locale == i18n.Pseudo || l.Locale == "" {
		return true
	}
	return slices.Contains(msgs.Missing(l.Locale), key)
}

// notFound is a 404 whose sentence is in the reader's language: what names
// a notFound.* message ("slot": "Slot not found.").
func notFound(ctx context.Context, what string) *api.Problem {
	return problem(ctx, http.StatusNotFound, api.CodeNotFound, "notFound."+what, nil)
}

// mapErr is api.MapError with its not-found sentence in the reader's
// language.
func mapErr(ctx context.Context, err error, what string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(ctx, what)
	}
	return api.MapError(err, msgs.M("notFound."+what).Source())
}

// shortDate is a day with its weekday in a language: "Tue 14 Oct" in
// ReputeCast's own format, "Th 3, 14 thg 10" in Vietnamese.
func shortDate(l i18n.Lang, t time.Time) string {
	if l.Region == "" || l.Region == i18n.Standard {
		return t.Format("Mon 2 Jan")
	}
	return l.Region.ShortDate(t)
}

// emailLang is the language a campaign's email is written in, as its
// readers read it: its source language when it has language versions,
// else the workspace's default language.
func (s *Service) emailLang(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) i18n.Lang {
	code := "en"
	_ = s.Pool.QueryRow(ctx, `SELECT coalesce(
		(SELECT language FROM content_sources WHERE owner_kind = 'campaign' AND owner_id = $1 AND owner_version = 0),
		(SELECT code FROM workspace_languages WHERE workspace_id = $2 AND is_default AND removed_at IS NULL), 'en')`, campaignID, ws.ID).Scan(&code)
	return pagelabels.Lang(code)
}

// problem is an RFC 9457 answer with its text from the catalog.
func problem(ctx context.Context, status int, code, key string, args i18n.Args) *api.Problem {
	return api.Err(status, code, text(ctx, key, args))
}

func invalid(ctx context.Context, location, key string, args i18n.Args) *api.Problem {
	return api.Invalid(location, text(ctx, key, args))
}

func userID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.Type != authz.ActorUser || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

func actorName(p *authz.Principal) string {
	if p == nil {
		return ""
	}
	return p.Name
}

// helpers names who can help with perm, Admins first ("Ask Sofia Rossi,
// Marcus Chen or Linh Nguyen"), leaving out the person asking.
func (s *Service) helpers(ctx context.Context, wsID uuid.UUID, perm authz.Permission, exclude uuid.UUID, most int) []string {
	out := []string{}
	rows, err := s.Q.EarnTeam(ctx, wsID)
	if err != nil {
		return out
	}
	// Editors hold sponsors:manage too: list the people who run sponsors
	// day to day (Editors) before Admins and Owners.
	type cand struct {
		name string
		rank int
	}
	var list []cand
	for _, r := range rows {
		if r.UserID == exclude || !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(perm) {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = r.Email
		}
		rank := 2
		switch r.RoleKey {
		case authz.RoleEditor:
			rank = 0
		case authz.RoleAdmin:
			rank = 1
		}
		if perm != authz.SponsorsManage {
			rank = 0
		}
		list = append(list, cand{name, rank})
	}
	for want := 0; want <= 2; want++ {
		for _, c := range list {
			if c.rank == want && len(out) < most {
				out = append(out, c.name)
			}
		}
	}
	return out
}

// location is the workspace's time zone.
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// today is the workspace's local date (midnight UTC of that day).
func (s *Service) today(ws *store.Workspace) time.Time {
	n := s.now().In(location(ws))
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func isUnique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
