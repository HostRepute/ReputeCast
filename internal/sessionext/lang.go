package sessionext

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The session's problems read in the request's language.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "sessionext")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}
