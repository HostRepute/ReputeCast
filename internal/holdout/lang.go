package holdout

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems a member reads follow the reader's app language. They are in
// messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "holdout")

// text formats a message of the holdout catalog in the request's
// language; an unknown key shows itself, which a test catches.
func text(ctx context.Context, key string, args i18n.Args) string {
	m, err := msgs.Lookup(key)
	if err != nil {
		return key
	}
	return m.Text(i18n.LangFrom(ctx), args)
}
