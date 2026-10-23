package team

import (
	"embed"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The Team screens' words from the server (the notes after a role, why the
// list is read-only, the member page's sources, the problems) follow the
// reader's app language (i18n.LangFrom). They are in
// messages/<locale>.json. Audit entries keep their English and, beside it,
// their summaries' messages for readers of other languages.

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

	// The invite dialog: how invitees sign in, what a role can do, and why
	// a role can't be granted (also the problem inviting it answers).
	mInviteSignIn       = msgs.M("invite.signIn")
	mInviteSignInSocial = msgs.M("invite.signInSocial")
	mGrantOwner         = msgs.M("grant.owner")
	mGrantMissing       = msgs.M("grant.missing")
	mRoleHint           = msgs.M("hint.role")

	mInviteOwnScope       = msgs.M("invite.ownScope")
	mInviteAlreadyMember  = msgs.M("invite.alreadyMember")
	mInviteAlreadyPending = msgs.M("invite.alreadyPending")
	mInviteNotEmail       = msgs.M("invite.notEmail")
	mInviteNoEmails       = msgs.M("invite.noEmails")
	mRolePick             = msgs.M("role.pick")
	mRoleUnknown          = msgs.M("role.unknown")

	// Changing, deactivating and removing members.
	mLastOwner      = msgs.M("problem.lastOwner")
	mOutrank        = msgs.M("problem.outrank")
	mOwnerOnly      = msgs.M("problem.ownerOnly")
	mOwnerTwoFactor = msgs.M("problem.ownerTwoFactor")
	mScopeWithin    = msgs.M("problem.scopeWithin")
)

// listArgs are a kept message's arguments for names it lists ("revoked
// “A”, “B” and “C”"): every name but the last joined with commas, the last
// one and their count, so each language joins them in its own words.
func listArgs(names []string) i18n.Args {
	if len(names) == 0 {
		return i18n.Args{"names": "", "last": "", "count": 0}
	}
	return i18n.Args{"names": strings.Join(names[:len(names)-1], ", "), "last": names[len(names)-1], "count": len(names)}
}

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
