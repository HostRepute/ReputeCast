package appearance

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Saving a theme reads in the request's language: its problem.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "appearance")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), nil)
}
