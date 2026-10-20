package publicpages

import (
	"context"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Web push on the public pages (design expansion phase 8, X22): the
// publication website's pages load the push prompt when the site is a push
// site, and the preference center offers "Turn off notifications in this
// browser". Both work only on the site's own address: a browser's
// subscription belongs to that origin.

// PushPages is what internal/push gives the pages.
type PushPages interface {
	// SiteScripts are the scripts a page on host loads (none when the site
	// isn't a push site).
	SiteScripts(ctx context.Context, ws *store.Workspace, host string) []string
	// TurnOff is the preference center's button, in the page's language
	// (nil when the site isn't a push site).
	TurnOff(ctx context.Context, ws *store.Workspace, host string, l i18n.Lang) *PushTurnOff
}

// PushTurnOff is the preference center's "Turn off notifications in this
// browser": its label, and what it says once done or when this browser gets
// nothing.
type PushTurnOff struct {
	Label string
	Done  string
	None  string
}

// pushHost is the site's own host when the page is on it, "" otherwise.
func pushHost(s *site) string {
	if s.preview || s.set.CustomDomain() == "" || s.links.Origin() != s.canon.Origin() {
		return ""
	}
	return strings.ToLower(s.set.CustomDomain())
}

// pushScripts are the push scripts of a page.
func (h *Handler) pushScripts(ctx context.Context, s *site, host string) []string {
	if h.Push == nil || host == "" {
		return nil
	}
	return h.Push.SiteScripts(ctx, s.ws, host)
}
