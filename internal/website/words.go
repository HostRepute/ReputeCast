package website

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
)

// Words are the site's built-in words (header, footer, paywall, sign-in,
// account) in one language: this package's catalog, English until a
// language ships them. The publisher's own text (titles, the paywall's
// heading) is content and isn't translated.
type Words struct{ L i18n.Lang }

// WordsIn returns the site's words in a language.
func WordsIn(l i18n.Lang) Words { return Words{L: l} }

// notFound is a 404 whose sentence is in the reader's language: what names
// a notFound.* message ("post": "Post not found.").
func notFound(ctx context.Context, what string) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, tr(ctx, "notFound."+what, nil))
}

// defaultLanguage is the workspace's default content language ("en" when
// it never set one): what its readers read by default.
func (s *Service) defaultLanguage(ctx context.Context, wsID uuid.UUID) i18n.Lang {
	code := "en"
	if rows, err := s.Q.LanguagesList(ctx, wsID); err == nil {
		for _, r := range rows {
			if r.IsDefault && r.RemovedAt == nil {
				code = r.Code
			}
		}
	}
	return pagelabels.Lang(code)
}

// Text is a word or sentence as plain text.
func (w Words) Text(key string, args i18n.Args) string { return msgs.M(key).Text(w.L, args) }

// HTML is a sentence with markup: values are escaped, tags go through tags.
func (w Words) HTML(key string, args i18n.Args, tags i18n.Tags) string {
	return msgs.M(key).HTML(w.L, args, tags)
}

// Has reports whether the catalog has a key.
func Has(key string) bool {
	_, err := msgs.Lookup(key)
	return err == nil
}
