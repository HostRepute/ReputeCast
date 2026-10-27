package campaigns

import (
	"embed"
	"regexp"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The campaign list's own words (the audience "All subscribers", the A/B
// and auto-pause notes), the composer's problems, its preview's sample
// contacts and the default names of new drafts follow the reader's app
// language (i18n.LangFrom). They are in messages/<locale>.json, with the
// labels campaigns writes on versions (versionLabel).

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "campaigns")

var (
	mAudienceAll      = msgs.M("audience.all")
	mAudienceSegments = msgs.M("audience.segments")
	mAudiencePaid     = msgs.M("audience.paid")
	mNoteAB           = msgs.M("note.ab")
	mNoteABWinner     = msgs.M("note.abWinner")
	mNoteAutoPaused   = msgs.M("note.autoPaused")
	mNoteAutoPausedOn = msgs.M("note.autoPausedOn")
	mMailboxOther     = msgs.M("mailbox.other")

	mProblemReadOnlySending   = msgs.M("problem.readOnlySending")
	mProblemReadOnlySent      = msgs.M("problem.readOnlySent")
	mProblemReadOnlySentOn    = msgs.M("problem.readOnlySentOn")
	mProblemNoTemplates       = msgs.M("problem.noTemplates")
	mProblemSavedSection      = msgs.M("problem.savedSection")
	mProblemName              = msgs.M("problem.name")
	mProblemPickSender        = msgs.M("problem.pickSender")
	mProblemUnknownSender     = msgs.M("problem.unknownSender")
	mProblemSenderScope       = msgs.M("problem.senderScope")
	mProblemListGone          = msgs.M("problem.listGone")
	mProblemListScope         = msgs.M("problem.listScope")
	mProblemSegmentGone       = msgs.M("problem.segmentGone")
	mProblemTierGone          = msgs.M("problem.tierGone")
	mProblemNotEmail          = msgs.M("problem.notEmail")
	mProblemTestOwnAddress    = msgs.M("problem.testOwnAddress")
	mProblemNoSystemRoute     = msgs.M("problem.noSystemRoute")
	mProblemTestRefused       = msgs.M("problem.testRefused")
	mProblemVersionConflict   = msgs.M("problem.versionConflict")
	mProblemVersionConflictBy = msgs.M("problem.versionConflictSomeone")
	mProblemSubjectOneLine    = msgs.M("problem.subjectOneLine")
	mProblemMJMLCompile       = msgs.M("problem.mjmlCompile")
	mProblemVariantLetter     = msgs.M("problem.variantLetter")
	mProblemVariantsAB        = msgs.M("problem.variantsAB")
	mProblemVariantSubject    = msgs.M("problem.variantSubject")
	mProblemApprovalCancel    = msgs.M("problem.approvalWouldCancel")

	mLabelUntitled        = msgs.M("label.untitled")
	mLabelCopyOf          = msgs.M("label.copyOf")
	mLabelSampleNoFirst   = msgs.M("label.sampleNoFirstName")
	mLabelSample          = msgs.M("label.sample")
	mLabelSampleNoContact = msgs.M("label.sampleNoContacts")
	mLabelFirstNameSet    = msgs.M("label.firstNameSet")
	mLabelNoFirstFallback = msgs.M("label.noFirstNameFallback")
	mLabelNoFirstName     = msgs.M("label.noFirstName")
	mLabelPerRecipient    = msgs.M("label.perRecipient")

	mVersionFromTemplate = msgs.M("version.fromTemplate")
	mVersionFromWebPost  = msgs.M("version.fromWebPost")
	mVersionRestored     = msgs.M("version.restored")
	mVersionCopiedFrom   = msgs.M("version.copiedFrom")
	mVersionFromPost     = msgs.M("version.fromPost")
	mNoteDraftFrom       = msgs.M("note.draftFrom")
	mVersionKeptYours    = msgs.M("version.keptYours")
	mVersionLoadedTheirs = msgs.M("version.loadedTheirs")

	// Collaboration (CO-10, CO-11).
	mVersionMerged          = msgs.M("version.merged")
	mVersionSynced          = msgs.M("version.synced")
	mVersionSharedForReview = msgs.M("version.sharedForReview")
	mCommentEmpty           = msgs.M("problem.commentEmpty")
	mCommentAnchor          = msgs.M("problem.commentAnchor")
	mCommentMention         = msgs.M("problem.commentMention")
	mNotifyMentionTitle     = msgs.M("notify.mentionTitle")
	mNotifyMentionSummary   = msgs.M("notify.mentionSummary")
	mNotifyMentionBody      = msgs.M("notify.mentionBody")
	mReviewLimited          = msgs.M("problem.reviewLimited")
	mReviewGone             = msgs.M("problem.reviewGone")
	mReviewName             = msgs.M("problem.reviewName")
	mDismissNote            = msgs.M("problem.dismissNote")
)

// PostDraftLabel is the first version's label of a draft made from a
// WordPress post, in English as it's stored: "created from the post on
// blog.acme.io". The composer's history reads it in each reader's language.
func PostDraftLabel(site string) string {
	return mVersionFromPost.Text(i18n.Default, i18n.Args{"site": site})
}

// PostDraftNote is the campaign list's note of a draft made from a
// WordPress post, in English as it's stored: "draft from blog.acme.io".
func PostDraftNote(site string) string {
	return mNoteDraftFrom.Text(i18n.Default, i18n.Args{"site": site})
}

var reNoteDraftFrom = regexp.MustCompile(`^draft from ([^\s]+)$`)

// campaignNote is a campaign's stored note in a reader's language when
// ReputeCast wrote it; a note a person wrote stays as it is.
func campaignNote(l i18n.Lang, note string) string {
	if l.Locale == i18n.English {
		return note
	}
	if m := reNoteDraftFrom.FindStringSubmatch(note); m != nil {
		return mNoteDraftFrom.Text(l, i18n.Args{"site": m[1]})
	}
	return note
}

// The labels campaigns writes on versions are stored in English
// (campaign_versions keeps no message beside them): versionLabel writes
// them in the reader's language, rows from before included. Labels other
// packages and people wrote stay as they were written.
var (
	reVersionFromTemplate = regexp.MustCompile(`^created from template “(.*)”$`)
	reVersionRestored     = regexp.MustCompile(`^restored v([0-9]+)$`)
	reVersionCopiedFrom   = regexp.MustCompile(`^copied from “(.*)”$`)
	reVersionFromPost     = regexp.MustCompile(`^created from the post on ([^\s]+)$`)
	reVersionKeptYours    = regexp.MustCompile(`^your edits, kept when (.*)'s v([0-9]+) was loaded$`)
	reVersionLoadedTheirs = regexp.MustCompile(`^loaded (.*)'s v([0-9]+)$`)
)

// versionLabel is a version's label in a reader's language.
func versionLabel(l i18n.Lang, label string) string {
	if l.Locale == i18n.English {
		return label
	}
	if label == mVersionFromWebPost.Source() {
		return mVersionFromWebPost.Text(l, nil)
	}
	if m := reVersionFromTemplate.FindStringSubmatch(label); m != nil {
		return mVersionFromTemplate.Text(l, i18n.Args{"name": m[1]})
	}
	if m := reVersionRestored.FindStringSubmatch(label); m != nil {
		return mVersionRestored.Text(l, i18n.Args{"version": m[1]})
	}
	if m := reVersionCopiedFrom.FindStringSubmatch(label); m != nil {
		return mVersionCopiedFrom.Text(l, i18n.Args{"name": m[1]})
	}
	if m := reVersionFromPost.FindStringSubmatch(label); m != nil {
		return mVersionFromPost.Text(l, i18n.Args{"site": m[1]})
	}
	if m := reVersionKeptYours.FindStringSubmatch(label); m != nil {
		return mVersionKeptYours.Text(l, i18n.Args{"name": m[1], "version": m[2]})
	}
	if m := reVersionLoadedTheirs.FindStringSubmatch(label); m != nil {
		return mVersionLoadedTheirs.Text(l, i18n.Args{"name": m[1], "version": m[2]})
	}
	if m := reVersionMerged.FindStringSubmatch(label); m != nil {
		return mVersionMerged.Text(l, i18n.Args{"name": m[1], "version": m[2]})
	}
	if m := reVersionSynced.FindStringSubmatch(label); m != nil {
		return mVersionSynced.Text(l, i18n.Args{"kind": strings.ToLower(m[1])})
	}
	return label
}

var (
	reVersionMerged = regexp.MustCompile(`^merged with (.+)'s v(\d+)$`)
	reVersionSynced = regexp.MustCompile(`^(Header|Footer) updated in Brand kit$`)
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
