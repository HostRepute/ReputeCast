package scoring

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems a member reads (a rule that can't be saved, a conflict) follow
// the reader's app language. They are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "scoring")

// text formats a message of the scoring catalog; an unknown key shows
// itself, which a test catches.
func text(l i18n.Lang, key string, args map[string]any) string {
	m, err := msgs.Lookup(key)
	if err != nil {
		return key
	}
	return m.Text(l, i18n.Args(args))
}
