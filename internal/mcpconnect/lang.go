package mcpconnect

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "mcpconnect")

func tr(ctx context.Context, key string) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), nil)
}
