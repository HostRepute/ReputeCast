package media

import (
	"context"
	"embed"
	"net/http"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Why an upload was refused reads in the uploader's app language
// (i18n.LangFrom). The messages are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "media")

var (
	mChooseImage = msgs.M("problem.chooseImage")
	mInterrupted = msgs.M("problem.interrupted")
	mEmpty       = msgs.M("problem.empty")
	mTooLarge    = msgs.M("problem.tooLarge")
	// An upload is compressed before it's stored, so it may be larger.
	mUploadTooLarge = msgs.M("problem.uploadTooLarge")
	mGIFTooLarge    = msgs.M("problem.gifTooLarge")
	mStillTooLarge  = msgs.M("problem.stillTooLarge")
	mWebP           = msgs.M("problem.webp")
	mSVG            = msgs.M("problem.svg")
	mNotImage       = msgs.M("problem.notImage")
	mDamaged        = msgs.M("problem.damaged")
	mDimensions     = msgs.M("problem.dimensions")
	mNotFound       = msgs.M("problem.notFound")
)

// say writes a message in the request's language.
func say(ctx context.Context, m i18n.Msg) string { return m.Text(i18n.LangFrom(ctx), nil) }

// notFound is the 404 of an image that isn't in the library.
func notFound(ctx context.Context) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, say(ctx, mNotFound))
}
