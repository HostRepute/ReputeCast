package roles

import (
	"embed"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// The roles screens read in the reader's language (i18n.LangFrom): the
// problems, the impact preview's sentences and the built-in roles' names
// and descriptions. The texts are in messages/<locale>.json; the audit log
// keeps the summaries' messages beside their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "roles")

// builtinDescription is a role's description in a language: a built-in
// role's own, unless someone changed it, and a custom role's as written.
func builtinDescription(l i18n.Lang, key string, builtin bool, stored string) string {
	if !builtin {
		return stored
	}
	b, ok := authz.Builtin(key)
	if !ok || stored != b.Description {
		return stored
	}
	if d := authz.RoleDescriptionIn(l, key); d != "" {
		return d
	}
	return stored
}

// roleName is a role's name in a language (the built-in ones are translated).
func roleName(l i18n.Lang, key, name string) string { return api.RoleName(l, key, name) }

// lowerFirst starts a label with a small letter inside a sentence ("view
// campaigns"), leaving acronyms ("API keys") as they are.
func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 || !unicode.IsUpper(r) {
		return s
	}
	if next, _ := utf8.DecodeRuneInString(s[n:]); unicode.IsUpper(next) {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}

// joinKeys joins permission keys, which are never translated:
// "campaigns:send, segments:sql".
func joinKeys(ps []string) string { return strings.Join(ps, ", ") }
