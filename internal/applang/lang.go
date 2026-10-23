package applang

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Saving a language reads in the request's language: its problems.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "applang")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// languageNames are the app languages, each named in itself: "English or
// Tiếng Việt", in a language's list words.
func languageNames(l i18n.Lang) string {
	names := []string{}
	for _, loc := range i18n.Locales() {
		if loc != i18n.Pseudo {
			names = append(names, loc.Name())
		}
	}
	return i18n.OrList(l.Locale, names)
}
