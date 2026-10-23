package apikeys

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Settings › Keys & MCP reads in the reader's language (i18n.LangFrom): the
// problems, the permission catalogue and the locked permissions' reasons,
// scopes and the agents' recent actions. The texts are in
// messages/<locale>.json; audit entries keep their summaries' messages
// beside the English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "apikeys")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}
