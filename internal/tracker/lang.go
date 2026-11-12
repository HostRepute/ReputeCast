package tracker

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The tracker's words follow their reader's language, from
// messages/<locale>.json: problems and the snippets' instructions in the
// request's language (i18n.LangFrom), the contact timeline's lines and the
// hub card's facts in the member's. Audit entries and logs keep English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "tracker")

var (
	mProblemName        = msgs.M("problem.name")
	mProblemOrigin      = msgs.M("problem.origin")
	mProblemOriginLocal = msgs.M("problem.originLocal")
	mProblemOriginOwn   = msgs.M("problem.originOwn")
	mProblemOriginTaken = msgs.M("problem.originTaken")
	mProblemStore       = msgs.M("problem.store")
	mProblemSiteGone    = msgs.M("problem.siteGone")

	mSnippetWordPress = msgs.M("snippet.wordpress")
	mPixelConsent     = msgs.M("snippet.pixelConsent")
	mPixelOpen        = msgs.M("snippet.pixelOpen")

	mActivityViewed  = msgs.M("activity.viewed")
	mActivityVisited = msgs.M("activity.visited")
	mActivityCarts   = msgs.M("activity.carts")
	mActivitySource  = msgs.M("activity.source")

	mHubFacts = msgs.M("hub.facts")
)
