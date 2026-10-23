package publicpages

import (
	"context"
	"net/http"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// The app's previews of the public pages answer a member, so their problems
// are in the member's app language (the request's context), while the pages
// they draw read in the page's language. The sentences are in the package's
// catalog (messages/<locale>.json, under problem.*).

// appText is one of the app's sentences in the member's language.
func appText(ctx context.Context, key string) string {
	return growMsgs.M(key).Text(i18n.LangFrom(ctx), nil)
}

// appNotFound is a 404 for a named thing: what names a problem.notFound.*
// message ("form": "Form not found.").
func appNotFound(ctx context.Context, what string) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, appText(ctx, "problem.notFound."+what))
}

// appInvalid is a 422 about one field, its sentence a problem.* message.
func appInvalid(ctx context.Context, location, key string) *api.Problem {
	return api.Invalid(location, appText(ctx, "problem."+key))
}

// appUnavailable is a 503: a part of the server isn't wired on this
// instance.
func appUnavailable(ctx context.Context, key string) *api.Problem {
	return api.Err(http.StatusServiceUnavailable, "unavailable", appText(ctx, "problem."+key))
}
