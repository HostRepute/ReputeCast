package notify

import (
	"embed"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the notification center and the preferences say follows the reader's
// app language: the event types' names (event.<key>), who gets them
// (audience.<key>) and their short names in a summary (short.<key>), the
// center's tabs (category.<tab>) and the problems. An event type or a tab
// without its message in messages/<locale>.json uses its English from the
// code. A notification's own text is kept as messages by the package that
// sends it (text.go).

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "notify")

var (
	mSummaryAll        = msgs.M("summary.all")
	mSummaryOff        = msgs.M("summary.off")
	mSummaryApprovals  = msgs.M("summary.approvals")
	mSummaryQuietHours = msgs.M("summary.quietHours")

	mProblemUnknownFilter = msgs.M("problem.unknownFilter")
	mProblemBadCursor     = msgs.M("problem.badCursor")
	mProblemUnknownEvent  = msgs.M("problem.unknownEvent")
	mProblemQuietFormat   = msgs.M("problem.quietFormat")
	mProblemQuietSame     = msgs.M("problem.quietSame")
)

// named is the message key's text in a language, or the English from the
// code when the catalog doesn't have it.
func named(l i18n.Lang, key, english string) string {
	if m, err := msgs.Lookup(key); err == nil {
		return m.Text(l, nil)
	}
	return english
}

// shortName is an event type's name in a summary: "auto-pauses".
func shortName(l i18n.Lang, e EventType) string { return named(l, "short."+e.Key, e.Short) }

// eventLabel is an event type's row name in the preferences: "Send auto-paused".
func eventLabel(l i18n.Lang, e EventType) string { return named(l, "event."+e.Key, e.Label) }

// eventAudience says who gets an event type: "People who can send campaigns".
func eventAudience(l i18n.Lang, e EventType) string {
	if e.Audience == "" {
		return ""
	}
	return named(l, "audience."+e.Key, e.Audience)
}

// categoryLabelIn is a notification center tab's name in a language.
func categoryLabelIn(l i18n.Lang, c string) string { return named(l, "category."+c, categoryLabels[c]) }

// capitalize starts a summary with a capital letter, in any script.
func capitalize(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}
