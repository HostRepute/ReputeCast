package server

import (
	"embed"
	"net/http"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The rate limits' problems read in the request's language: the
// Accept-Language header, as the app sends it (the limits run before the
// request is signed in).

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "server")

// requestLang is the language a request asks for, or English.
func requestLang(r *http.Request) i18n.Lang {
	if l, ok := i18n.Negotiate(r.Header.Get("Accept-Language")); ok {
		return i18n.For(l)
	}
	return i18n.Default
}
