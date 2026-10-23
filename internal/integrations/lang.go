package integrations

import (
	"context"
	"embed"
	"encoding/json"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// What a member reads from the integrations hub follows their app language
// (messages/<locale>.json): the problems of the app's endpoints and of the
// integration API (API clients that send no Accept-Language, such as
// Zapier, Make, n8n and the WordPress plugin, keep getting English), the
// public URL check's lines, and the notifications, kept as messages so
// every member reads them in their own language. A problem an integration
// got is kept the same way for the call log (noteDetail). Audit entries,
// webhook payloads, sample data and client names stay English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "integrations")

// lang is the language of the request.
func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// text writes a message in the request's language.
func text(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(lang(ctx), args)
}

// noteDetail is the call note that keeps a problem's message for the call log.
const noteDetail = "detail"

// problem writes a problem's detail in the request's language and, on an
// integration's call, keeps its message for the call log, which reads it
// in each member's language.
func problem(ctx context.Context, key string, args i18n.Args) string {
	m := msgs.M(key)
	if b, err := json.Marshal(m.Keep(args)); err == nil {
		api.NoteCall(ctx, noteDetail, string(b))
	}
	return m.Text(lang(ctx), args)
}

// itemName names a trigger or an action in a language: "Contact
// unsubscribed" (item.<key>), or its key when the catalog doesn't have it.
func itemName(l i18n.Lang, key string) string {
	if m, err := msgs.Lookup("item." + key); err == nil {
		return m.Text(l, nil)
	}
	return key
}
