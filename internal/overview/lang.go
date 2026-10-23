package overview

import (
	"embed"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The Overview's sentences follow the reader's app language
// (i18n.LangFrom): sender health's reasons and their links, the provider
// cards' detail lines, the engagement tiers and the sunset policy. Rates,
// numbers and times are written in the reader's region format. The words
// are in messages/<locale>.json. The verdict (Good, Fair, Poor) stays a
// value the web words itself.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "overview")

var (
	mNotChecked     = msgs.M("health.notChecked")
	mClean          = msgs.M("health.clean")
	mDomainListed   = msgs.M("health.domainListed")
	mStale          = msgs.M("health.stale")
	mAuthNotChecked = msgs.M("health.authNotChecked")
	mAuthFails      = msgs.M("health.authFails")
	mNoDMARC        = msgs.M("health.noDmarc")
	mUnaligned      = msgs.M("health.unaligned")
	mPolicyNone     = msgs.M("health.policyNone")
	mAligned        = msgs.M("health.aligned")
	mNothingSent    = msgs.M("health.nothingSent")
	mRateOver       = msgs.M("health.rateOver")
	mRateClose      = msgs.M("health.rateClose")
	mRateHalf       = msgs.M("health.rateHalf")
	mRatesOK        = msgs.M("health.ratesOk")
	mIPListed       = msgs.M("health.ipListed")
	mProviderIP     = msgs.M("health.providerIpListed")
	mWarmup         = msgs.M("health.warmup")
	mWarmupShare    = msgs.M("health.warmupShare")
	mDown           = msgs.M("health.down")
	mDownSince      = msgs.M("health.downSince")
	mDownAuth       = msgs.M("health.downAuth")
	mDegraded       = msgs.M("health.degraded")
	mSilent         = msgs.M("health.webhooksSilent")
	mSandbox        = msgs.M("health.sandbox")

	mActOpenDeliv   = msgs.M("action.openDeliverability")
	mActDelist      = msgs.M("action.requestDelisting")
	mActConnection  = msgs.M("action.checkConnection")
	mActFixRecords  = msgs.M("action.fixRecords")
	mActAddDMARC    = msgs.M("action.addDmarc")
	mActSeeDMARC    = msgs.M("action.seeDmarc")
	mActSeeProvider = msgs.M("action.seeProvider")
	mActStreams     = msgs.M("action.seeStreams")
	mActDiagnose    = msgs.M("action.diagnose")
	mActProduction  = msgs.M("action.requestProduction")

	mSelfHosted   = msgs.M("provider.selfHosted")
	mDedicated    = msgs.M("provider.dedicated")
	mStreamShare  = msgs.M("provider.streamShare")
	mStatusPage   = msgs.M("provider.statusPage")
	mTierActive   = msgs.M("tier.active")
	mTierRange    = msgs.M("tier.range")
	mTierSunset   = msgs.M("tier.sunset")
	mPolicyOff    = msgs.M("policy.off")
	mPolicyOn     = msgs.M("policy.on")
	mStreamTitles = map[string]i18n.Msg{
		"broadcast": msgs.M("stream.broadcast"), "automation": msgs.M("stream.automation"),
		"system": msgs.M("stream.system"), "transactional": msgs.M("stream.transactional"),
	}
)

// pctIn writes a percentage with as many decimals as it has: 0.1 → "0.1%",
// 5 → "5%" (a provider's limit), in the reader's region format.
func pctIn(l i18n.Lang, v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	digits := 0
	if _, frac, ok := strings.Cut(s, "."); ok {
		digits = len(frac)
	}
	return l.Region.Percent(v, digits)
}

// rateIn writes a rate with a fixed number of decimals: "0.6%", "0.04%".
func rateIn(l i18n.Lang, v float64, digits int) string { return l.Region.Percent(v, digits) }

// clockIn writes a time of day in the reader's region format: "15:04".
func clockIn(l i18n.Lang, t time.Time) string { return l.Region.Time(t, "") }
