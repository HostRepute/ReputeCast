package digests

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Text a member reads follows their app language (messages/<locale>.json):
// problems and what the screens show in the request's language,
// notifications in each recipient's (as placement tests do), and a
// version's name in the language of the member it is saved for. Why a
// feed failed is kept as a message beside its English (digests.feed_error,
// an issue's feed_tries) and written in each reader's language. Audit
// entries keep their English.

var (
	mScheduleFrequency   = msgs.M("schedule.frequency")
	mScheduleTimezone    = msgs.M("schedule.timezone")
	mScheduleCount       = msgs.M("schedule.count")
	mScheduleWindow      = msgs.M("schedule.window")
	mScheduleWindowOrder = msgs.M("schedule.windowOrder")
	mScheduleTime        = msgs.M("schedule.time")
	mScheduleWeekday     = msgs.M("schedule.weekday")
	mScheduleMonthDay    = msgs.M("schedule.monthDay")

	mPauseReason          = msgs.M("pause.reason")
	mIssueOwnerLost       = msgs.M("issue.ownerLost")
	mReadinessUnavailable = msgs.M("problem.readinessUnavailable")
	mReadinessRunning     = msgs.M("problem.readinessRunning")
	mTestsUnavailable     = msgs.M("problem.testsUnavailable")
	mTemplateRender       = msgs.M("problem.templateRender")
	mTryTimedOut          = msgs.M("feed.tryTimedOut")
	mTryFailed            = msgs.M("feed.tryFailed")

	mVersionStarter  = msgs.M("version.starter")
	mVersionPrepared = msgs.M("version.prepared")
	mVersionRebuilt  = msgs.M("version.rebuilt")
	mVersionNext     = msgs.M("version.next")

	mPausedTitle        = msgs.M("notify.pausedTitle")
	mPausedRole         = msgs.M("notify.pausedRole")
	mPausedLost         = msgs.M("notify.pausedLost")
	mPausedOwnerSummary = msgs.M("notify.pausedOwnerSummary")
	mPausedFeed         = msgs.M("notify.pausedFeed")
	mPausedFeedSummary  = msgs.M("notify.pausedFeedSummary")
	mPausedBy           = msgs.M("notify.pausedBy")
	mPausedByNote       = msgs.M("notify.pausedByNote")
	mPausedBySummary    = msgs.M("notify.pausedBySummary")
	mPausedAnon         = msgs.M("notify.pausedAnon")
	mPausedAnonNote     = msgs.M("notify.pausedAnonNote")
	mPausedAnonSummary  = msgs.M("notify.pausedAnonSummary")
	mHeldTitle          = msgs.M("notify.heldTitle")
	mHeldNoApprovers    = msgs.M("notify.heldNoApprovers")
	mHeldChanges        = msgs.M("notify.heldChanges")
	mHeldRejected       = msgs.M("notify.heldRejected")
	mHeldBlocked        = msgs.M("notify.heldBlocked")
	mHeldReadiness      = msgs.M("notify.heldReadiness")
	mHeldSummary        = msgs.M("notify.heldSummary")
	mDraftTitle         = msgs.M("notify.draftTitle")
	mDraftBody          = msgs.M("notify.draftBody")
	mDraftSummary       = msgs.M("notify.draftSummary")
	mFeedTitle          = msgs.M("notify.feedTitle")
	mFeedFailing        = msgs.M("notify.feedFailing")
	mFeedFailingUnknown = msgs.M("notify.feedFailingUnknown")
	mFeedSkipped        = msgs.M("notify.feedSkipped")
	mFeedFailingSummary = msgs.M("notify.feedSummary")
)

// memberLang is how a member reads what a digest writes for them.
func memberLang(ctx context.Context, q *store.Queries, id uuid.UUID) i18n.Lang {
	return applang.ForUserID(ctx, q, id)
}

// versionLang is the language of a version saved for a member, or the
// instance's when nobody saved it.
func versionLang(ctx context.Context, q *store.Queries, by *uuid.UUID) i18n.Lang {
	if by == nil {
		return applang.InstanceDefault(ctx, q)
	}
	return memberLang(ctx, q, *by)
}

// roleIn writes a role's name in a language: the built-in roles are
// translated, a custom role keeps the name it was given.
func roleIn(l i18n.Lang, name string) string {
	for _, r := range authz.BuiltinRoles {
		if r.Name == name {
			return api.RoleName(l, r.Key, name)
		}
	}
	return name
}

// whenIn is a moment as a notification writes it, in the reader's region
// and the digest's time zone: "Mon 2 Jan" and "15:04".
func whenIn(l i18n.Lang, t *time.Time, loc *time.Location) (date, clock string) {
	if t == nil {
		return "", ""
	}
	at := t.In(loc)
	return l.Region.ShortDate(at), l.Region.Time(at, "")
}

// capitalize starts a sentence with a capital letter, in any script.
func capitalize(s string) string {
	s = strings.TrimSpace(s)
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// feedText is why a read failed, written for a reader, or its English when
// the error has no message.
func feedText(l i18n.Lang, err error, address string) string {
	if t := feed.TextOf(err, address); !t.IsZero() {
		if s := t.In(l); s != "" {
			return s
		}
	}
	if t, ok := feed.GuardText(address, err); ok {
		if s := t.In(l); s != "" {
			return s
		}
	}
	return err.Error()
}

// addressProblem is why an address can't be a digest's feed, as a
// validation message in the reader's language.
func addressProblem(l i18n.Lang, err error, address string) string {
	return mFeedAddress.Text(l, i18n.Args{"detail": strings.TrimSuffix(capitalize(feedText(l, err, address)), ".")})
}

// textIn writes a kept message in a reader's language, or its English.
func textIn(t *i18n.Text, l i18n.Lang, english string) string {
	if t != nil {
		if s := t.In(l); s != "" {
			return s
		}
	}
	return english
}
