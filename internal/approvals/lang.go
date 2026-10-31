package approvals

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
)

// The texts approvals write for people follow the reader's app language
// (i18n.LangFrom): problem details, the rules a send matches, the inbox's
// and the approval page's lines, the request reasons of Review & send, the
// publish gates of pages, posts, templates and automations. They are in
// messages/<locale>.json.
//
// What approvals store for people who read it later keeps the English it
// always stored (older rows, API clients and agents read it) and keeps the
// messages beside it, written in each reader's language when read: the
// rules a request matched (Snapshot.RuleTexts), the changes that
// invalidated it (keptChange in approval_requests.changes) and its reason
// (approval_requests.invalidated_reason_text). Rows written before have no
// kept messages and read as stored.
//
// Notifications keep their messages too (notify_text.go): each member reads
// them in their own language, and their stored English is unchanged.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "approvals")

// langOf is the language the request reads in.
func langOf(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// say writes a message in the request's language.
func say(ctx context.Context, m i18n.Msg, args i18n.Args) string { return m.Text(langOf(ctx), args) }

// english writes a message as stored and as API clients read it: in
// English, with ReputeCast's own formats.
func english(m i18n.Msg, args i18n.Args) string { return m.Text(i18n.Default, args) }

// kept is stored text for a reader: the English stored with it for English
// readers and for rows written before its message was kept, the message in
// the reader's language otherwise.
func kept(l i18n.Lang, english string, t i18n.Text) string {
	if l.Locale == i18n.English || t.IsZero() {
		return english
	}
	if s := render(l, t); s != "" {
		return s
	}
	return english
}

// render writes a kept message in a language. Some arguments are kept as
// data and written here for the reader: "moved" (how far the audience
// moved, in percent), "at" (an RFC 3339 time), "added" and "removed"
// (section labels), "fields" (the fields that changed), "lists" and
// "approvers" (names), and "money…" (US dollars).
func render(l i18n.Lang, t i18n.Text) string {
	args := i18n.Args{}
	for k, v := range t.Args {
		switch {
		case k == "moved":
			if f, ok := v.(float64); ok {
				v = signedPct(l, f)
			}
		case k == "at":
			if s, ok := v.(string); ok {
				if at, err := time.Parse(time.RFC3339, s); err == nil {
					v = monthDayTime(l, at)
				}
			}
		case k == "added" || k == "removed":
			v = sectionList(l, strs(v))
		case k == "fields":
			v = fieldList(l, strs(v))
		case k == "lists" || k == "approvers":
			if k == "lists" {
				v = i18n.AndList(l.Locale, strs(v))
			} else {
				v = i18n.OrList(l.Locale, strs(v))
			}
		case strings.HasPrefix(k, "money"):
			if f, ok := v.(float64); ok {
				v = money(l, f)
			}
		}
		args[k] = v
	}
	// A language version's change names its language and what changed in
	// the reader's language.
	if d, ok := changeDetail(t.Args["change"]); ok {
		name, _, _, what := d.Texts(l)
		args["language"], args["what"] = name, what
	}
	return i18n.Text{ID: t.ID, Args: args}.In(l)
}

// changeDetail reads a kept language change (a detail, or its JSON).
func changeDetail(v any) (languages.VersionChangeDetail, bool) {
	var d languages.VersionChangeDetail
	switch x := v.(type) {
	case nil:
		return d, false
	case *languages.VersionChangeDetail:
		if x == nil {
			return d, false
		}
		return *x, true
	case languages.VersionChangeDetail:
		return x, true
	}
	b, err := json.Marshal(v)
	if err != nil || json.Unmarshal(b, &d) != nil || d.Language == "" {
		return d, false
	}
	return d, true
}

// strs reads a kept list of strings (it comes back from JSON as []any).
func strs(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, s := range x {
			if str, ok := s.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

// count writes a number of recipients in the reader's region: 31,604 · 31.604.
func count(l i18n.Lang, n int64) string {
	if n < 0 {
		return "−" + l.Region.Int(-n)
	}
	return l.Region.Int(n)
}

// signedPct writes how far an audience moved: +14%, −9.5%.
func signedPct(l i18n.Lang, v float64) string {
	digits := 0
	if math.Abs(v) < 10 {
		digits = 1
	}
	s := l.Region.Percent(math.Abs(v), digits)
	if v >= 0 {
		return "+" + s
	}
	return "−" + s
}

// trimPct writes a percentage with at most digits decimals, without
// trailing zeros: 2%, 0.1% (Send Readiness's thresholds).
func trimPct(l i18n.Lang, v float64, digits int) string {
	s := strconv.FormatFloat(v, 'f', digits, 64)
	d := 0
	if i := strings.IndexByte(s, '.'); i >= 0 {
		d = len(strings.TrimRight(s[i+1:], "0"))
	}
	return l.Region.Percent(v, d)
}

// decimal writes a share of a route: 90, 12.5.
func decimal(l i18n.Lang, v float64) string {
	v = math.Round(v*10) / 10
	if v == math.Trunc(v) {
		return l.Region.Decimal(v, 0)
	}
	return l.Region.Decimal(v, 1)
}

// money writes US dollars: $50, $19.54, 50 US$.
func money(l i18n.Lang, v float64) string {
	if l.Region == i18n.Standard || l.Region == "" {
		return costLabel(v)
	}
	digits := 0
	if v != math.Trunc(v) {
		digits = 2
	}
	return l.Region.Money(v, digits)
}

// monthDayTime writes a moment the way the approval screens do: "Nov 4,
// 09:00", "4 thg 11, 09:00".
func monthDayTime(l i18n.Lang, t time.Time) string {
	if l.Region == i18n.Standard || l.Region == "" {
		return t.Format("Jan 2, 15:04")
	}
	return l.Region.DayMonth(t) + ", " + l.Region.Time(t, "")
}

// clock writes a time of day: 09:05, 9:05 AM.
func clock(l i18n.Lang, t time.Time) string {
	if l.Region == i18n.Standard || l.Region == "" {
		return t.Format("15:04")
	}
	return l.Region.Time(t, "")
}

var columnsRe = regexp.MustCompile(`^(\d+) columns$`)

// sectionName writes a section of an email as the layer tree names it
// ("hero", "2 columns", "footer"); a section from a block of another name
// keeps it.
func sectionName(l i18n.Lang, label string) string {
	if m := columnsRe.FindStringSubmatch(label); m != nil {
		n, _ := strconv.Atoi(m[1])
		return mSectionColumns.Text(l, i18n.Args{"count": n})
	}
	if m, err := msgs.Lookup("section." + strings.ReplaceAll(label, " ", "_")); err == nil {
		return m.Text(l, nil)
	}
	return label
}

// sectionList writes section labels the way a content change lists them:
// "hero, 2 columns".
func sectionList(l i18n.Lang, labels []string) string {
	out := make([]string, len(labels))
	for i, s := range labels {
		out[i] = sectionName(l, s)
	}
	return strings.Join(out, ", ")
}

// fieldList writes the fields that changed: "subject and a link".
func fieldList(l i18n.Lang, keys []string) string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if code, ok := strings.CutPrefix(k, "lang:"); ok {
			out = append(out, mFieldLanguage.Text(l, i18n.Args{"language": languages.NameIn(l, code)}))
			continue
		}
		if name, ok := strings.CutPrefix(k, "language:"); ok {
			out = append(out, mFieldLanguage.Text(l, i18n.Args{"language": name}))
			continue
		}
		if m, err := msgs.Lookup("field." + k); err == nil {
			out = append(out, m.Text(l, nil))
			continue
		}
		out = append(out, k)
	}
	return i18n.AndList(l.Locale, out)
}

// statusSelect is a request status as the select messages take it.
func statusSelect(status string) string {
	switch status {
	case "changes_requested", "invalidated", "cancelled", "approved", "rejected", "pending", "bypassed", "sent":
		return status
	}
	return "other"
}

var (
	// Problems.
	mNotFound               = msgs.M("problem.notFound")
	mReadinessRunning       = msgs.M("problem.readinessRunning")
	mAlreadySending         = msgs.M("problem.alreadySending")
	mPendingRequest         = msgs.M("problem.pendingRequest")
	mApprovedScheduled      = msgs.M("problem.approvedScheduled")
	mAlreadyApproved        = msgs.M("problem.alreadyApproved")
	mReadinessBlocked       = msgs.M("problem.readinessBlocked")
	mChecksBlocked          = msgs.M("problem.checksBlocked")
	mNotRequired            = msgs.M("problem.notRequired")
	mNoApprovers            = msgs.M("problem.noApprovers")
	mRequestedForCampaign   = msgs.M("problem.requestedForCampaign")
	mOwnCampaign            = msgs.M("problem.ownCampaign")
	mRequestIsAlready       = msgs.M("problem.requestIsAlready")
	mAlreadyDecided         = msgs.M("problem.alreadyDecided")
	mCantRenew              = msgs.M("problem.cantRenew")
	mAuthorGone             = msgs.M("problem.authorGone")
	mReadinessBlockedNew    = msgs.M("problem.readinessBlockedNew")
	mChecksBlockedNew       = msgs.M("problem.checksBlockedNew")
	mWriteComment           = msgs.M("problem.writeComment")
	mCancelNotAllowed       = msgs.M("problem.cancelNotAllowed")
	mRemindNotAllowed       = msgs.M("problem.remindNotAllowed")
	mRemindPendingOnly      = msgs.M("problem.remindPendingOnly")
	mRemindedRecently       = msgs.M("problem.remindedRecently")
	mRestoreInvalidatedOnly = msgs.M("problem.restoreInvalidatedOnly")
	mNewerRequest           = msgs.M("problem.newerRequest")
	mRestoreUnavailable     = msgs.M("problem.restoreUnavailable")
	mRestoreIncomplete      = msgs.M("problem.restoreIncomplete")
	mCampaignOnly           = msgs.M("problem.campaignOnly")
	mNoLongerCovers         = msgs.M("problem.noLongerCovers")
	mNotApproved            = msgs.M("problem.notApproved")
	mBypassOwnerOnly        = msgs.M("problem.bypassOwnerOnly")
	mBypassOff              = msgs.M("problem.bypassOff")
	mBypassReason           = msgs.M("problem.bypassReason")
	mWrongPassword          = msgs.M("problem.wrongPassword")
	mAlreadyScheduled       = msgs.M("problem.alreadyScheduled")
	mPickFuture             = msgs.M("problem.pickFuture")
	mReadinessChanged       = msgs.M("problem.readinessChanged")
	mReadinessStale         = msgs.M("problem.readinessStale")
	mReadinessOld           = msgs.M("problem.readinessOld")
	mSendBlocked            = msgs.M("problem.sendBlocked")
	mReadinessIncomplete    = msgs.M("problem.readinessIncomplete")
	mConfirmWarnings        = msgs.M("problem.confirmWarnings")
	mTypeCount              = msgs.M("problem.typeCount")
	mApprovalRequired       = msgs.M("problem.approvalRequired")
	mSendingUnavailable     = msgs.M("problem.sendingUnavailable")
	mBlockedAsStarted       = msgs.M("problem.blockedAsStarted")
	mBlockedAsStartedReason = msgs.M("problem.blockedAsStartedReason")
	mNotScheduled           = msgs.M("problem.notScheduled")
	mNoClientApprover       = msgs.M("problem.noClientApprover")
	mSignoffNotClient       = msgs.M("problem.signoffNotClient")
	mAudienceMin            = msgs.M("problem.audienceMin")
	mApprovalsRequired      = msgs.M("problem.approvalsRequired")
	mPagePeopleOnly         = msgs.M("problem.pagePeopleOnly")
	mPageNoNeed             = msgs.M("problem.pageNoNeed")
	mPageNoApprovers        = msgs.M("problem.pageNoApprovers")
	mPageRace               = msgs.M("problem.pageRace")
	mPostPeopleOnly         = msgs.M("problem.postPeopleOnly")
	mPostNoNeed             = msgs.M("problem.postNoNeed")
	mPostNoApprovers        = msgs.M("problem.postNoApprovers")
	mPostRace               = msgs.M("problem.postRace")
	mTemplateNoNeed         = msgs.M("problem.templateNoNeed")
	mTemplateBlocked        = msgs.M("problem.templateBlocked")
	mTemplateNoApprovers    = msgs.M("problem.templateNoApprovers")
	mTemplateRace           = msgs.M("problem.templateRace")

	// Why asking for approval isn't possible (Review & send, the publish dialog).
	mReasonNeedsEdit           = msgs.M("reason.needsEdit")
	mReasonNoRule              = msgs.M("reason.noRule")
	mReasonRunFirst            = msgs.M("reason.runFirst")
	mReasonFixBlocks           = msgs.M("reason.fixBlocks")
	mReasonRequested           = msgs.M("reason.requested")
	mReasonApproved            = msgs.M("reason.approved")
	mReasonNoOtherApprover     = msgs.M("reason.noOtherApprover")
	mReasonTemplateNoNeed      = msgs.M("reason.templateNoNeed")
	mReasonPeopleOnly          = msgs.M("reason.peopleOnly")
	mReasonTemplateNoApprovers = msgs.M("reason.templateNoApprovers")

	// Why the viewer can't decide (the approval page).
	mDecideOwn        = msgs.M("decide.own")
	mDecidePeopleOnly = msgs.M("decide.peopleOnly")
	mDecideWaiting    = msgs.M("decide.waiting")
	mDecideStatus     = msgs.M("decide.status")

	// The publish gates of pages, posts, templates and automations.
	mPublishLacksPublish       = msgs.M("publish.lacksPublish")
	mPublishPageRule           = msgs.M("publish.pageRule")
	mPublishPostRule           = msgs.M("publish.postRule")
	mPublishWaiting            = msgs.M("publish.waiting")
	mPublishPageNoApprovers    = msgs.M("publish.pageNoApprovers")
	mPublishPostNoApprovers    = msgs.M("publish.postNoApprovers")
	mPublishTemplateRule       = msgs.M("publish.templateRule")
	mPublishTemplateNoApprover = msgs.M("publish.templateNoApprovers")
	mPublishAutomationAlready  = msgs.M("publish.automationAlready")
	mPublishAutomationWaiting  = msgs.M("publish.automationWaiting")

	// Labels and lines of the screens.
	mPresenceOnline  = msgs.M("presence.online")
	mPresenceMinutes = msgs.M("presence.minutes")
	mPresenceHours   = msgs.M("presence.hours")
	mPresenceDays    = msgs.M("presence.days")
	mABLabel         = msgs.M("label.ab")
	mExcluded        = msgs.M("label.excluded")
	mAutoPause       = msgs.M("label.autoPause")
	mNoteComment     = msgs.M("note.comment")
	mNoteCancelled   = msgs.M("note.cancelled")
	mVisibility      = msgs.M("label.visibility")
	mPolicyWarming   = msgs.M("policy.warming")
	mPolicyConnected = msgs.M("policy.connected")
	mChangeKind      = msgs.M("label.automationChange")
	mSectionColumns  = msgs.M("section.columns")
	mFieldLanguage   = msgs.M("field.language")
)

// notFound is a missing approval request, in the reader's language.
func notFound(ctx context.Context) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, say(ctx, mNotFound, nil))
}

// mapErr is api.MapError for approval requests: a missing one is notFound.
func mapErr(ctx context.Context, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(ctx)
	}
	return api.MapError(err, "Approval request")
}

// Why a scheduled send was held. Approvals stores these in English (a held
// send's detail), and so does the sending engine, which lists them with a
// capital as its send-time recheck gives them ("Approval is still
// pending"); gapIn writes them in the reader's language.
var (
	mGapPending     = msgs.M("gap.pending")
	mGapInvalidated = msgs.M("gap.invalidated")
	mGapRunning     = msgs.M("gap.running")
	mGapCheckFailed = msgs.M("gap.checkFailed")
	gapMsgs         = map[string]i18n.Msg{"changes_requested": msgs.M("gap.changesRequested"), "rejected": msgs.M("gap.rejected"),
		"cancelled": msgs.M("gap.cancelled")}
	allGaps = []i18n.Msg{mGapPending, mGapInvalidated, gapMsgs["changes_requested"], gapMsgs["rejected"], gapMsgs["cancelled"], mGapRunning, mGapCheckFailed}
)

// gapIn writes why a send was held in the reader's language when it is one
// of the reasons approvals gives; anything else, such as the title of a
// blocking check, reads as it was written.
func gapIn(l i18n.Lang, stored string) string {
	if l.Locale == i18n.English || stored == "" {
		return stored
	}
	for _, m := range allGaps {
		en := english(m, nil)
		switch stored {
		case en:
			return m.Text(l, nil)
		case capitalize(en):
			return capitalize(m.Text(l, nil))
		}
	}
	return stored
}

// capitalize starts a sentence with a capital letter, in any script.
func capitalize(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// noLongerCovers is the problem of a send whose approval stopped covering it.
func (s *Service) noLongerCovers(ctx context.Context, changes []storedChange) error {
	return api.Err(http.StatusConflict, "approval_invalidated", say(ctx, mNoLongerCovers, i18n.Args{"reason": render(langOf(ctx), reasonOf(changes))}))
}

// approvalRequired is the problem of a send that needs an approval first.
func approvalRequired(ctx context.Context, rules []ApprovalRuleMatch) error {
	return api.Err(http.StatusConflict, "approval_required", say(ctx, mApprovalRequired, i18n.Args{"rules": shortsOf(localRules(langOf(ctx), rules))})).
		WithMeta("rules", ruleKeys(rules))
}
