package verification

import (
	"embed"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Problems and notifications a member reads follow their language
// (messages/<locale>.json). Audit entries keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "verification")

var (
	mUnknownService   = msgs.M("problem.unknownService")
	mAlreadyConnected = msgs.M("problem.alreadyConnected")
	mNotConnected     = msgs.M("problem.notConnected")
	mKeyRejected      = msgs.M("problem.keyRejected")
	mNoCreditsKey     = msgs.M("problem.noCreditsKey")
	mUnreachable      = msgs.M("problem.unreachable")
	mNoService        = msgs.M("problem.noService")
	mNoServiceAsk     = msgs.M("problem.noServiceAsk")
	mNothingToCheck   = msgs.M("problem.nothingToCheck")
	mJobFinished      = msgs.M("problem.jobFinished")
	mJobNotStopped    = msgs.M("problem.jobNotStopped")
	mFormsEdit        = msgs.M("problem.formsEdit")
	mFormsEditPlain   = msgs.M("problem.formsEditPlain")
	mNoFormsService   = msgs.M("problem.noFormsService")
	mContactErased    = msgs.M("problem.contactErased")

	mTrayRunning       = msgs.M("tray.running")
	mTrayDone          = msgs.M("tray.done")
	mTrayStopped       = msgs.M("tray.stopped")
	mTrayCancelled     = msgs.M("tray.cancelled")
	mTrayDoneDetail    = msgs.M("tray.doneDetail")
	mTrayStoppedDetail = msgs.M("tray.stoppedDetail")
	mTrayPaused        = msgs.M("tray.paused")
	mTrayEta           = msgs.M("tray.eta")
	mTrayEtaSoon       = msgs.M("tray.etaSoon")

	// MJobFinishedTitle and the rest word the notifications (notify.go in
	// internal/app formats them per recipient).
	MJobFinishedTitle   = msgs.M("notify.jobFinishedTitle")
	MJobFinishedSummary = msgs.M("notify.jobFinishedSummary")
	MJobStoppedTitle    = msgs.M("notify.jobStoppedTitle")
	MJobStoppedCredits  = msgs.M("notify.jobStoppedCredits")
	MJobStoppedDown     = msgs.M("notify.jobStoppedDown")
	MJobStoppedKey      = msgs.M("notify.jobStoppedKey")
	MCreditsLowTitle    = msgs.M("notify.creditsLowTitle")
	MCreditsLowSummary  = msgs.M("notify.creditsLowSummary")
	MCreditsLowNeeded   = msgs.M("notify.creditsLowNeeded")
)

var (
	mInvalidID      = msgs.M("problem.invalidId")
	mPickTier       = msgs.M("problem.pickTier")
	mTiers          = msgs.M("problem.tiers")
	mPickContacts   = msgs.M("problem.pickContacts")
	mPickTarget     = msgs.M("problem.pickTarget")
	mLabelTiers     = msgs.M("label.tiers")
	mLabelSelection = msgs.M("label.selection")
	mWhatImport     = msgs.M("what.import")
	mWhatContact    = msgs.M("what.contact")
	mWhatForm       = msgs.M("what.form")
	mWhatJob        = msgs.M("what.job")
)

// tierMsgs name the engagement tiers, by key.
var tierMsgs = map[string]i18n.Msg{
	"active":  msgs.M("tier.active"),
	"cooling": msgs.M("tier.cooling"),
	"at_risk": msgs.M("tier.atRisk"),
	"sunset":  msgs.M("tier.sunset"),
}

// regionMsgs name where a service processes addresses, by the English
// label its descriptor gives (adapter.Region.Label).
var regionMsgs = map[string]i18n.Msg{
	"United States":  msgs.M("region.unitedStates"),
	"European Union": msgs.M("region.europeanUnion"),
}

// regionIn is a service's data region in a reader's language.
func regionIn(l i18n.Lang, service, region string) string {
	return regionName(l, RegionLabel(service, region))
}

// regionName is a region's English label in a reader's language.
func regionName(l i18n.Lang, label string) string {
	if m, ok := regionMsgs[label]; ok {
		return m.Text(l, nil)
	}
	return label
}

// tierList names tiers in a language: "At risk and Sunset".
func tierList(l i18n.Lang, keys []string) string {
	names := make([]string, 0, len(keys))
	for _, k := range keys {
		if m, ok := tierMsgs[k]; ok {
			names = append(names, m.Text(l, nil))
		}
	}
	return i18n.AndList(l.Locale, names)
}

// tierKeys reads back the tier names a job's English label joins
// ("At risk and Sunset").
func tierKeys(label string) ([]string, bool) {
	keys := []string{}
	for _, name := range strings.Split(label, " and ") {
		key := ""
		for k, n := range tierNames {
			if n == name {
				key = k
			}
		}
		if key == "" {
			return nil, false
		}
		keys = append(keys, key)
	}
	return keys, len(keys) > 0
}

// selectedCount reads back a selection's English label ("12 selected contacts").
func selectedCount(label string) (int, bool) {
	n, ok := strings.CutSuffix(label, " selected contacts")
	if !ok {
		return 0, false
	}
	v, err := strconv.Atoi(n)
	return v, err == nil
}

// labelIn is what a job checks, in a reader's language: a list's,
// segment's or import's name, or an address, stay as they are; tiers and a
// selection of contacts are named in the reader's words. tiers are the
// job's tier keys when it knows them.
func labelIn(l i18n.Lang, kind, label string, tiers []string) string {
	switch kind {
	case "tier":
		if len(tiers) == 0 {
			tiers, _ = tierKeys(label)
		}
		if len(tiers) > 0 {
			return tierList(l, tiers)
		}
	case "selection":
		if n, ok := selectedCount(label); ok {
			return mLabelSelection.Text(l, i18n.Args{"count": n})
		}
	}
	return label
}

// jobTiers are a job's tier keys, from its target.
func jobTiers(j store.VerificationJob) []string {
	var t jobTarget
	_ = json.Unmarshal(j.Target, &t)
	return t.Tiers
}

// JobLabel names a job the way a notification does, in a reader's
// language: "Sunset tier", "Product Updates".
func JobLabel(l i18n.Lang, j store.VerificationJob) string {
	if j.Kind == "tier" {
		tiers := jobTiers(j)
		if len(tiers) == 0 {
			tiers, _ = tierKeys(j.Label)
		}
		if len(tiers) > 0 {
			return mLabelTiers.Text(l, i18n.Args{"tiers": tierList(l, tiers), "count": len(tiers)})
		}
		return j.Label + " tier"
	}
	return labelIn(l, j.Kind, j.Label, nil)
}

// whatIn is a service's last use ("Sunset tier job", "import preview") in
// a reader's language. It reads back what useWhat and the real-time checks
// wrote; anything else, such as a list's name, stays as it is.
func whatIn(l i18n.Lang, what string) string {
	if w, ok := strings.CutSuffix(what, " job"); ok && w != "" {
		return mWhatJob.Text(l, i18n.Args{"what": whatIn(l, w)})
	}
	switch what {
	case "import preview":
		return mWhatImport.Text(l, nil)
	case "one contact":
		return mWhatContact.Text(l, nil)
	case "signup form":
		return mWhatForm.Text(l, nil)
	}
	if w, ok := strings.CutSuffix(what, " tier"); ok {
		if keys, ok := tierKeys(w); ok {
			return mLabelTiers.Text(l, i18n.Args{"tiers": tierList(l, keys), "count": len(keys)})
		}
	}
	return labelIn(l, "selection", what, nil)
}
