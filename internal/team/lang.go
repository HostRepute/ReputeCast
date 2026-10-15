package team

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The Team list's own words (the notes after a role, why the list is
// read-only) follow the reader's app language (i18n.LangFrom). They are in
// messages/<locale>.json. Audit entries keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "team")

var (
	mNoteInstanceAdmin = msgs.M("note.instanceAdmin")
	mNoteBreakGlass    = msgs.M("note.breakGlass")
	mReasonViewOnly    = msgs.M("reason.viewOnly")
	mReasonViewOnlyAsk = msgs.M("reason.viewOnlyAsk")
	mReasonInvite      = msgs.M("reason.invite")
	mReasonInviteAsk   = msgs.M("reason.inviteAsk")
)

// readOnlyReason explains what a caller without team:invite or
// team:manage can do, and who can help.
func readOnlyReason(l i18n.Lang, canInvite bool, helpers []string) string {
	plain, ask := mReasonViewOnly, mReasonViewOnlyAsk
	if canInvite {
		plain, ask = mReasonInvite, mReasonInviteAsk
	}
	if len(helpers) == 0 {
		return plain.Text(l, nil)
	}
	return ask.Text(l, i18n.Args{"helpers": i18n.OrList(l.Locale, helpers)})
}
