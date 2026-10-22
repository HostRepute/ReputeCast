package transactional

import (
	"context"
	"embed"
	"time"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// What a member reads in Transactional's own screens (a viewer's read-only
// note, Settings and its categories, a template's details, the message log's
// filters and a message's content) follows their language
// (messages/<locale>.json). The send API's answers to apps, the template
// checks and the audit log keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var texts = i18n.MustLoad(messagesFS, "transactional")

// say formats a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return texts.M(key).Text(i18n.LangFrom(ctx), args)
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
