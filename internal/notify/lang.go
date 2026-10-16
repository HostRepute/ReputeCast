package notify

import (
	"embed"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The summary of a member's notification preferences (Account & security)
// reads in their app language. The event types' short names are
// messages/<locale>.json's short.<event key>; an event type without one
// there uses its English Short.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "notify")

var (
	mSummaryAll        = msgs.M("summary.all")
	mSummaryOff        = msgs.M("summary.off")
	mSummaryApprovals  = msgs.M("summary.approvals")
	mSummaryQuietHours = msgs.M("summary.quietHours")
)

// shortName is an event type's name in a summary: "auto-pauses".
func shortName(l i18n.Lang, e EventType) string {
	if m, err := msgs.Lookup("short." + e.Key); err == nil {
		return m.Text(l, nil)
	}
	return e.Short
}

// capitalize starts a summary with a capital letter, in any script.
func capitalize(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}
