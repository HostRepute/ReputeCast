package brandkit

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems and notes read in the caller's app language (i18n.LangFrom).
// The messages are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "brandkit")

var (
	mBadColor         = msgs.M("problem.badColor")
	mBadFallback      = msgs.M("problem.badFallback")
	mBadFontURL       = msgs.M("problem.badFontUrl")
	mBadLogo          = msgs.M("problem.badLogo")
	mOneSection       = msgs.M("problem.oneSection")
	mCompile          = msgs.M("problem.compile")
	mNeedsUnsubscribe = msgs.M("problem.needsUnsubscribe")
	mNotSaved         = msgs.M("problem.notSaved")
	mNameRequired     = msgs.M("problem.syncedName")
	mAlreadySynced    = msgs.M("problem.alreadySynced")
	mHoldsSynced      = msgs.M("problem.holdsSynced")
	mKeepHeaderFooter = msgs.M("problem.keepHeaderFooter")
	mBadWebsite       = msgs.M("problem.badWebsite")
	mUnreachable      = msgs.M("problem.unreachable")
	mLinkContrast     = msgs.M("import.linkContrast")
	mSVGLogo          = msgs.M("import.svgLogo")
	mLogoNotAdded     = msgs.M("import.logoNotAdded")
)

func say(ctx context.Context, m i18n.Msg) string { return m.Text(i18n.LangFrom(ctx), nil) }
