package campaigns

import (
	"embed"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The campaign list's own words (the audience "All subscribers", the A/B
// and auto-pause notes) follow the reader's app language (i18n.LangFrom).
// They are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "campaigns")

var (
	mAudienceAll      = msgs.M("audience.all")
	mAudienceSegments = msgs.M("audience.segments")
	mNoteAB           = msgs.M("note.ab")
	mNoteABWinner     = msgs.M("note.abWinner")
	mNoteAutoPaused   = msgs.M("note.autoPaused")
	mNoteAutoPausedOn = msgs.M("note.autoPausedOn")
	mMailboxOther     = msgs.M("mailbox.other")
)

// abNote describes an A/B test that ran: "A/B subject, winner B".
func abNote(l i18n.Lang, kind, winner string) string {
	args := i18n.Args{"kind": kind, "kindName": strings.ReplaceAll(kind, "_", " "), "winner": winner}
	if winner == "" {
		return mNoteAB.Text(l, args)
	}
	return mNoteABWinner.Text(l, args)
}

// autoPausedNote describes the guard's pauses: "auto-paused once, Yahoo deferrals".
func autoPausedNote(l i18n.Lang, n int64, mailbox, metric string) string {
	if mailbox == "" {
		return mNoteAutoPaused.Text(l, i18n.Args{"count": n})
	}
	name := mailboxName(mailbox)
	if name == "" {
		name = mMailboxOther.Text(l, nil)
	}
	if metric == "hard_bounce" {
		metric = "bounce"
	}
	return mNoteAutoPausedOn.Text(l, i18n.Args{"count": n, "mailbox": name, "metric": metric, "metricName": metric})
}
