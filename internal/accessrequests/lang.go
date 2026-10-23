package accessrequests

import (
	"context"
	"embed"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Access requests read in the member's language (i18n.LangFrom): the
// problems and what the permission allows (authz's words). The request's
// notification keeps its messages beside the English (notify's
// TitleText…), so each manager reads it in their own language; the note is
// the member's own words.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "accessrequests")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// keep is one of the package's messages kept for later readers.
func keep(key string, args i18n.Args) i18n.Text { return msgs.M(key).Keep(args) }

// capitalize upper-cases the first letter, whatever its script: the
// action leads the 403 screen's sentence.
func capitalize(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}
