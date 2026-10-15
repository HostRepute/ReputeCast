package accessibility

import (
	"context"

	"github.com/hostrepute/reputecast/internal/store"
)

// ContentLanguage is the language a workspace writes its emails in: the
// source language of every email, and the lang the compiled HTML gets.
// Content languages (M17) replace it from their wire file; until then every
// workspace writes in English.
var ContentLanguage = func(context.Context, *store.Workspace) string { return "en" }

// FinalizeStored adds lang, dir and the layout tables' role to HTML that is
// about to be stored for a workspace, in its source language. Every save of
// compiled HTML (campaigns, automation emails, transactional templates)
// goes through it.
func FinalizeStored(ctx context.Context, ws *store.Workspace, html string) string {
	if html == "" {
		return html
	}
	out, _ := Finalize(html, ContentLanguage(ctx, ws))
	return out
}
