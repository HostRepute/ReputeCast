package imports

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What members read about an import follows their app language
// (i18n.LangFrom): the wizard's steps (the consent table, the field
// targets, the preview's groups and duplicates), the problems, the jobs
// tray and the live log. The source's own words stay as the source writes
// them (statuses such as subscribed and cleaned, field keys, list names),
// and so do the CSV files and the new list's description.
//
// What an import stores for people who read it later keeps, beside its
// English, the messages it came from (i18n.Text): the live log's lines
// (import_log.message_parts), the last failure talking to the source (the
// connection's errorText), and the consent history and suppressions it
// writes (their texts columns, audience.KeptTexts). Lines and failures
// stored before read in their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "imports")

var (
	mConnectFirst   = msgs.M("problem.connectFirst")
	mFileNoKey      = msgs.M("problem.fileNoKey")
	mFullAddress    = msgs.M("problem.fullAddress")
	mOnlyAddress    = msgs.M("problem.onlyAddress")
	mPrivateAddress = msgs.M("problem.privateAddress")
	mPasteKey       = msgs.M("problem.pasteKey")
	mEmptyFile      = msgs.M("problem.emptyFile")
	mNotField       = msgs.M("problem.notField")
	mFieldGone      = msgs.M("problem.fieldGone")
	mNewFieldKey    = msgs.M("problem.newFieldKey")
	mTwoNewFields   = msgs.M("problem.twoNewFields")
	mPickField      = msgs.M("problem.pickField")
	mOneEmail       = msgs.M("problem.oneEmail")
	mDiscard        = msgs.M("problem.discard")
	mRunning        = msgs.M("problem.running")
	mFinished       = msgs.M("problem.finished")
	mListOnce       = msgs.M("problem.listOnce")
	mNameList       = msgs.M("problem.nameList")
	mPickList       = msgs.M("problem.pickList")
	mNoList         = msgs.M("problem.noList")
	mNotStagedWait  = msgs.M("problem.notStagedWait")
	mStarted        = msgs.M("problem.started")
	mNotStaged      = msgs.M("problem.notStaged")
	mMapEmail       = msgs.M("problem.mapEmail")
	mNotRunning     = msgs.M("problem.notRunning")
	mChooseList     = msgs.M("problem.chooseList")
	mPickTarget     = msgs.M("problem.pickTarget")
	mUploadFirst    = msgs.M("problem.uploadFirst")
	mOnlyPaused     = msgs.M("problem.onlyPaused")

	mListmonkAddress   = msgs.M("source.listmonkAddress")
	mNotifuseAddress   = msgs.M("source.notifuseAddress")
	mMailchimpKey      = msgs.M("source.mailchimpKey")
	mMailchimpRejected = msgs.M("source.mailchimpRejected")
	mNotifuseNoWS      = msgs.M("source.notifuseNoWorkspace")
	mUnreachable       = msgs.M("source.unreachable")
	mRejectedKey       = msgs.M("source.rejectedKey")
	mRateLimited       = msgs.M("source.rateLimited")
	mAnswered          = msgs.M("source.answered")
	mRefused           = msgs.M("source.refused")
	mRedirects         = msgs.M("source.redirects")
	mUnreadable        = msgs.M("source.unreadable")
	mKeyUnreadable     = msgs.M("source.keyUnreadable")

	mFileUnreadable = msgs.M("file.unreadable")
	mRowUnreadable  = msgs.M("file.rowUnreadable")
	mTooManyRows    = msgs.M("file.tooManyRows")
	mFileEmpty      = msgs.M("file.empty")
	mBadColumns     = msgs.M("file.badColumns")

	mLogStart          = msgs.M("log.start")
	mLogMapping        = msgs.M("log.mapping")
	mLogConsent        = msgs.M("log.consent")
	mLogPulled         = msgs.M("log.pulled")
	mLogBatch          = msgs.M("log.batch")
	mLogStopped        = msgs.M("log.stopped")
	mLogPushed         = msgs.M("log.pushed")
	mLogNextSync       = msgs.M("log.nextSync")
	mLogRepermFailed   = msgs.M("log.repermissionFailed")
	mLogRepermSent     = msgs.M("log.repermissionSent")
	mLogRepermWaiting  = msgs.M("log.repermissionWaiting")
	mLogFinished       = msgs.M("log.finished")
	mLogStoppedAfter   = msgs.M("log.stoppedAfter")
	mLogAnalyzeFailed  = msgs.M("log.analyzeFailed")
	mLogFailed         = msgs.M("log.failed")
	mLogRetry          = msgs.M("log.retry")
	mLogRetry429       = msgs.M("log.retry429")
	mLogRead           = msgs.M("log.read")
	mLogPaused         = msgs.M("log.paused")
	mLogSlowed         = msgs.M("log.slowed")
	mLogTemplatesKey   = msgs.M("log.templatesKey")
	mLogTemplatesError = msgs.M("log.templatesFailed")
	mLogTemplates      = msgs.M("log.templates")
	mLogTemplatesFix   = msgs.M("log.templatesManual")

	mJobReading       = msgs.M("job.reading")
	mJobImporting     = msgs.M("job.importing")
	mJobPaused        = msgs.M("job.paused")
	mJobPausedResumes = msgs.M("job.pausedResumes")
	mJobDone          = msgs.M("job.done")
	mJobDoneDetail    = msgs.M("job.doneDetail")
	mJobFailed        = msgs.M("job.failed")
	mJobStopped       = msgs.M("job.stopped")
	mJobStoppedDetail = msgs.M("job.stoppedDetail")
	mEtaSoon          = msgs.M("job.etaSoon")
	mEtaMinutes       = msgs.M("job.etaMinutes")

	mConsentConfirmed      = msgs.M("consent.confirmed")
	mConsentPending        = msgs.M("consent.pending")
	mConsentPendingShort   = msgs.M("consent.pendingShort")
	mConsentSuppressUnsub  = msgs.M("consent.suppressUnsub")
	mConsentSuppressBounce = msgs.M("consent.suppressBounce")
	mConsentSuppressCompl  = msgs.M("consent.suppressComplaint")
	mNoteDoubleOptIn       = msgs.M("consent.noteDoubleOptIn")
	mNoteDoubleOptInSince  = msgs.M("consent.noteDoubleOptInSince")
	mNoteHardBounced       = msgs.M("consent.noteHardBounced")
	mNotePending           = msgs.M("consent.notePending")
	mLabelAbuse            = msgs.M("consent.labelAbuse")
	mEffectUnsub           = msgs.M("consent.effectUnsub")
	mEffectBlocked         = msgs.M("consent.effectBlocked")
	mEffectBlockedPushed   = msgs.M("consent.effectBlockedPushed")
	mEffectConfirmed       = msgs.M("consent.effectConfirmed")
	mEffectPending         = msgs.M("consent.effectPending")
	mEffectSkip            = msgs.M("consent.effectSkip")
	mEvidenceOptin         = msgs.M("consent.evidenceOptin")
	mHistory               = msgs.M("consent.history")
	mHistoryEvidence       = msgs.M("consent.historyEvidence")
	mHistoryNoRow          = msgs.M("consent.historyNoRow")

	mDetailList   = msgs.M("history.detailList")
	mDetailFile   = msgs.M("history.detailFile")
	mNoteCarried  = msgs.M("history.noteCarried")
	mNoteAsPend   = msgs.M("history.notePending")
	mNoteEvidence = msgs.M("history.noteEvidence")

	mSuppUnsubscribed = msgs.M("suppression.unsubscribed")
	mSuppCleaned      = msgs.M("suppression.cleaned")
	mSuppAbuse        = msgs.M("suppression.abuse")
	mSuppBlocklisted  = msgs.M("suppression.blocklisted")
	mSuppContext      = msgs.M("suppression.context")

	mFieldEmail        = msgs.M("field.emailRequired")
	mFieldName         = msgs.M("field.name")
	mFieldEvidence     = msgs.M("field.consentEvidence")
	mFieldStatus       = msgs.M("field.consentStatus")
	mFieldPhone        = msgs.M("field.phone")
	mFieldSmsConsent   = msgs.M("field.smsConsent")
	mFieldSmsDate      = msgs.M("field.smsDate")
	mFieldSmsSource    = msgs.M("field.smsSource")
	mFieldSmsText      = msgs.M("field.smsText")
	mFieldCustomExists = msgs.M("field.customExists")
	mFieldCreateNew    = msgs.M("field.createNew")
	mSkip              = msgs.M("option.skip")

	mExcludeBroadcasts = msgs.M("action.excludeBroadcasts")
	mImportNormally    = msgs.M("action.importNormally")
	mFixTypo           = msgs.M("action.fixTypo")
	mImportAsIs        = msgs.M("action.importAsIs")
	mImportPending     = msgs.M("action.pending")
	mSkipRecommended   = msgs.M("action.skipRecommended")
	mImportTagged      = msgs.M("action.tagged")
	mExampleRating     = msgs.M("example.rating")
	mExampleChanged    = msgs.M("example.ratingChanged")
	mExampleRedacted   = msgs.M("example.redacted")
	mStaleWhy          = msgs.M("risky.stale.why")

	mMergeFills     = msgs.M("merge.fills")
	mMergeTags      = msgs.M("merge.tags")
	mMergeTier      = msgs.M("merge.tier")
	mMergeSame      = msgs.M("merge.same")
	mDupStays       = msgs.M("duplicate.stays")
	mDupOnList      = msgs.M("duplicate.onList")
	mDupNoLists     = msgs.M("duplicate.noLists")
	mDupLists       = msgs.M("duplicate.lists")
	mDupEmpty       = msgs.M("duplicate.empty")
	mDupJoined      = msgs.M("duplicate.joined")
	mDupTags        = msgs.M("duplicate.tags")
	mDupTier        = msgs.M("duplicate.tier")
	mDupLastOpen    = msgs.M("duplicate.lastOpen")
	mDupFile        = msgs.M("duplicate.file")
	mInvalidNoMX    = msgs.M("invalid.noMX")
	mInvalidService = msgs.M("invalid.service")

	mNotifyTitle = msgs.M("notify.title")
	mNotifyBody  = msgs.M("notify.body")
)

// textIn writes a kept message in a reader's language. A list argument
// (provider names) is joined the way the language writes lists; "" when
// the catalog no longer has the message.
func textIn(l i18n.Lang, t i18n.Text) string {
	key, ok := strings.CutPrefix(t.ID, msgs.Namespace+".")
	if !ok {
		return t.In(l)
	}
	m, err := msgs.Lookup(key)
	if err != nil {
		return ""
	}
	return m.Text(l, listArgs(l, t.Args))
}

// listArgs joins the list arguments of a message (a []string when it's
// written, a []any once it went through JSON) for a reader.
func listArgs(l i18n.Lang, args i18n.Args) i18n.Args {
	var out i18n.Args
	for k, v := range args {
		var items []string
		switch x := v.(type) {
		case []string:
			items = x
		case []any:
			for _, it := range x {
				items = append(items, fmt.Sprint(it))
			}
		default:
			continue
		}
		if out == nil {
			out = maps.Clone(args)
		}
		out[k] = i18n.AndList(l.Locale, items)
	}
	if out == nil {
		return args
	}
	return out
}

// say is a message in the request's language, for problems and the
// wizard's labels.
func say(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	l := i18n.LangFrom(ctx)
	return m.Text(l, listArgs(l, args))
}

// english is a message as it's stored for later readers: English with
// ReputeCast's standard formats.
func english(t i18n.Text) string { return textIn(i18n.Default, t) }

// lineIn writes a live-log line's kept messages for a reader, joined with
// " · " as the stored English is; the stored English when the reader reads
// English, the line kept no messages or one is no longer known.
func lineIn(l i18n.Lang, stored string, raw json.RawMessage) string {
	if l.Locale == i18n.English || len(raw) == 0 {
		return stored
	}
	var parts []i18n.Text
	if json.Unmarshal(raw, &parts) != nil || len(parts) == 0 {
		return stored
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := textIn(l, p)
		if s == "" {
			return stored
		}
		out = append(out, s)
	}
	return strings.Join(out, " · ")
}

// keptIn writes a stored English text for a reader from the message it
// kept; the English when there is none or the reader reads English.
func keptIn(l i18n.Lang, stored string, t *i18n.Text) string {
	if t == nil || t.IsZero() || l.Locale == i18n.English {
		return stored
	}
	if s := textIn(l, *t); s != "" {
		return s
	}
	return stored
}

// explained is an error explained for people that keeps the message its
// words came from (SourceError, FileError, keptError).
type explained interface {
	error
	kept() i18n.Text
}

// keptError is a failure explained for people with its message kept.
type keptError struct {
	text i18n.Text
}

func newKeptError(m i18n.Msg, args i18n.Args) *keptError {
	return &keptError{text: m.Keep(args)}
}

func (e *keptError) Error() string   { return english(e.text) }
func (e *keptError) kept() i18n.Text { return e.text }

// keptOf is the message an error's words came from; nil for errors that
// only have their own (English) words.
func keptOf(err error) *i18n.Text {
	var ex explained
	if errors.As(err, &ex) {
		if t := ex.kept(); !t.IsZero() {
			return &t
		}
	}
	if errors.Is(err, ErrNoKey) {
		t := mPasteKey.Keep(nil)
		return &t
	}
	return nil
}

// errorIn writes an error for a reader: in their language when it kept its
// message, else in its own words.
func errorIn(l i18n.Lang, err error) string {
	return keptIn(l, err.Error(), keptOf(err))
}

// dayIn writes a day with its year: "2 Jan 2006" in English, as imports
// always wrote it, else the reader's region format ("2 thg 1, 2006").
func dayIn(l i18n.Lang, t time.Time) string {
	if l.Locale == i18n.English {
		return t.Format("2 Jan 2006")
	}
	return l.Region.Date(t)
}

// dayMonthIn writes a day without its year: "2 Jan", or the reader's
// region format.
func dayMonthIn(l i18n.Lang, t time.Time) string {
	if l.Locale == i18n.English {
		return t.Format("2 Jan")
	}
	return l.Region.DayMonth(t)
}

// clockIn writes a time of day: "15:04", or the reader's region format.
func clockIn(l i18n.Lang, t time.Time) string {
	if l.Locale == i18n.English {
		return t.Format("15:04")
	}
	return l.Region.Time(t, "")
}
