package transactional

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// What a member reads in Transactional's own screens (a viewer's read-only
// note, Settings and its categories, a template's details, the editor's
// checks, previews, test renders and publish dialog, the message log's
// filters and a message's content) follows their language
// (messages/<locale>.json), and so do the delivery alerts' notifications,
// kept as messages for each member. The send API's and the SMTP relay's
// answers to apps, a published version's change summary (stored with the
// version) and the audit log keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var texts = i18n.MustLoad(messagesFS, "transactional")

// say formats a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return texts.M(key).Text(i18n.LangFrom(ctx), args)
}

// lang is the request's language.
func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// notFound is a 404 whose sentence is in the reader's language: what names
// a notFound.* message ("template": "Template not found.").
func notFound(ctx context.Context, what string) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, say(ctx, "notFound."+what, nil))
}

// mapErr is api.MapError with its not-found sentence in the reader's
// language.
func mapErr(ctx context.Context, err error, what string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(ctx, what)
	}
	return api.MapError(err, texts.M("notFound."+what).Source())
}

// mapTemplateErr is mapErr for a template.
func mapTemplateErr(ctx context.Context, err error) error { return mapErr(ctx, err, "template") }

// andList joins names as the reader's language writes a list: "a, b and c".
func andList(l i18n.Lang, items []string) string { return i18n.AndList(l.Locale, items) }

// valueError is a sample or default that isn't a value of its variable's
// type ("\"abc\" is not a number"): its English for the API, its message
// for the editor.
type valueError struct {
	key   string // value.number, value.boolean, value.list, value.object
	value string
}

func (e *valueError) Error() string {
	return texts.M(e.key).Text(i18n.Default, i18n.Args{"value": e.value})
}

// mjmlError is MJML that doesn't compile, with the compiler's own finding
// (in English, as gomjml writes it).
type mjmlError struct{ detail string }

func (e *mjmlError) Error() string { return "the MJML doesn't compile: " + e.detail }

// problemText is an error from rendering or checking a template in the
// reader's language: ReputeCast's own sentences translated, the Liquid
// engine's and the MJML compiler's findings as they wrote them.
func problemText(l i18n.Lang, err error) string {
	var re *renderError
	if errors.As(err, &re) {
		msg := re.Message
		if re.key != "" {
			msg = texts.M(re.key).Text(l, re.args)
		}
		return texts.M("render.part").Text(l, i18n.Args{"part": strings.ToLower(re.Part), "message": msg})
	}
	var me *mjmlError
	if errors.As(err, &me) {
		return texts.M("render.mjml").Text(l, i18n.Args{"detail": strings.TrimSuffix(strings.TrimSpace(me.detail), ".")})
	}
	var ve *valueError
	if errors.As(err, &ve) {
		return texts.M(ve.key).Text(l, i18n.Args{"value": ve.value})
	}
	return err.Error()
}

// workspaceLang is the workspace's default language, for what ReputeCast
// writes once into the workspace's own content (a default category).
func workspaceLang(ctx context.Context, q *store.Queries, wsID uuid.UUID) i18n.Lang {
	if rows, err := q.LanguagesList(ctx, wsID); err == nil {
		for _, r := range rows {
			if r.IsDefault && r.RemovedAt == nil {
				if loc, ok := i18n.ParseLocale(r.Code); ok {
					return i18n.For(loc)
				}
			}
		}
	}
	return i18n.Default
}

// askNames joins the members who can help as the language writes
// alternatives: "Marcus Chen or Linh Nguyen".
func askNames(ctx context.Context, names []string) string {
	return i18n.OrList(i18n.LangFrom(ctx).Locale, names)
}

// deniedMessage is "Publishing needs transactional:publish. Ask …": what a
// role can't do here, in the request's language. action is one of
// categories, settings, resend, edit and publish.
func deniedMessage(ctx context.Context, action string, perm authz.Permission, helpers []string) string {
	if len(helpers) > 0 {
		return say(ctx, "denied.ask", i18n.Args{"action": action, "permission": string(perm), "names": askNames(ctx, helpers)})
	}
	return say(ctx, "denied", i18n.Args{"action": action, "permission": string(perm)})
}

// dayIn writes a day as the reader's region format does: "2 Oct 2026" in
// ReputeCast's standard formats, "2 thg 10, 2026" in Vietnamese.
func dayIn(ctx context.Context, t time.Time) string {
	l := i18n.LangFrom(ctx)
	if l.Region == i18n.Standard {
		return t.Format("2 Jan 2006")
	}
	return l.Region.Date(t)
}

// clockIn writes a time of day as the reader's region format does: "09:41",
// or "9:41 AM" on a 12-hour clock.
func clockIn(ctx context.Context, t time.Time) string {
	l := i18n.LangFrom(ctx)
	if l.Region == i18n.Standard {
		return t.Format("15:04")
	}
	return l.Region.Time(t, "")
}
