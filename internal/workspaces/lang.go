package workspaces

import (
	"context"
	"embed"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Settings › General reads in the reader's language (i18n.LangFrom): its
// problems, the approval rules, how members sign in and the transfer's
// notes. The notifications it sends and the audit entries' summaries keep
// their messages beside the English (notify's TitleText…,
// audit.KeepSummary). The export's README stays in English: it's a file
// for whoever opens the archive.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "workspaces")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// keep is one of the package's messages kept for later readers.
func keep(key string, args i18n.Args) i18n.Text { return msgs.M(key).Keep(args) }

// location is a workspace's timezone, where its members read its dates.
func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// capitalize upper-cases the first letter, whatever its script.
func capitalize(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// yesNo is the select value of a condition.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
