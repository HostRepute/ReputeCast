package countdown

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The timer's labels and alt text are in the block's language; problems
// and checks in the reader's (i18n.LangFrom). The messages are in
// messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "countdown")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// keep is one of the package's messages kept for later readers.
func keep(key string, args i18n.Args) i18n.Text { return msgs.M(key).Keep(args) }

// labelKeys are the units' labels, in the order the timer shows them.
var labelKeys = [4]string{"days", "hours", "minutes", "seconds"}

// langOf is how a block's language reads: the app language with its own
// region format (dates as "Fri 28 Nov", times as "23:59" in English).
func langOf(code string) i18n.Lang {
	l, ok := i18n.ParseLocale(code)
	if !ok {
		l = i18n.English
	}
	return i18n.For(l)
}
