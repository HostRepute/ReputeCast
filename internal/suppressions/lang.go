package suppressions

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What Sending providers › Events reads from the suppression sync card
// follows the reader's app language (messages/<locale>.json): which list
// each provider syncs with, why a provider without a suppression API needs
// none, and an import's row. Provider names, the providers' own sync
// errors and the detail ReputeCast stores on a pulled suppression ("Suppressed at
// SES", which no screen shows) stay English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "suppressions")

// text writes a message in a reader's language.
func text(l i18n.Lang, key string, args i18n.Args) string {
	return msgs.M(key).Text(l, args)
}
