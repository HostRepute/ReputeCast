// Package publicurl builds the absolute URLs that appear in emails and on
// public pages, so the code that writes them (sending, tracking, forms,
// reports) and the handlers that serve them agree on one layout:
//
//	/t/o/{token}                          open pixel (linktoken.Open)
//	/t/c/{token}                          click redirect (linktoken.Click)
//	/p/{workspace}/unsubscribe/{token}    unsubscribe page; POST is RFC 8058 one-click
//	/p/{workspace}/preferences/{token}    preference center
//	/p/{workspace}/confirm/{token}        double opt-in confirmation
//	/p/{workspace}/view/{token}           view an email in the browser
//	/p/{workspace}/subscribe[/{form}]     hosted subscribe page
//	/p/{workspace}/archive[/{issue}]      public newsletter archive
//	/p/{workspace}/archive.rss            archive feed
//	/r/{token}                            read-only shared campaign report
//
// {workspace} is the workspace slug. Handlers must still check that a token's
// workspace matches the slug in the path.
package publicurl

import (
	"net/url"
	"strings"
)

// Path prefixes, for routers.
const (
	TrackingPrefix = "/t"
	PagesPrefix    = "/p"
	ReportPrefix   = "/r"
)

// Builder joins paths onto the instance's public URL.
type Builder struct{ base string }

// New returns a Builder for the configured public URL (http.public_url).
func New(publicURL string) Builder {
	return Builder{base: strings.TrimRight(publicURL, "/")}
}

// Base returns the public URL without a trailing slash.
func (b Builder) Base() string { return b.base }

func (b Builder) join(parts ...string) string {
	var sb strings.Builder
	sb.WriteString(b.base)
	for _, p := range parts {
		sb.WriteByte('/')
		sb.WriteString(url.PathEscape(p))
	}
	return sb.String()
}

// Open is the tracking pixel URL.
func (b Builder) Open(token string) string { return b.join("t", "o", token) }

// Click is the click redirect URL.
func (b Builder) Click(token string) string { return b.join("t", "c", token) }

// Unsubscribe is the unsubscribe page; mail clients POST to the same URL for
// one-click unsubscribe (List-Unsubscribe-Post: List-Unsubscribe=One-Click).
func (b Builder) Unsubscribe(workspace, token string) string {
	return b.join("p", workspace, "unsubscribe", token)
}

// Preferences is the preference center.
func (b Builder) Preferences(workspace, token string) string {
	return b.join("p", workspace, "preferences", token)
}

// Confirm is the double opt-in confirmation link.
func (b Builder) Confirm(workspace, token string) string {
	return b.join("p", workspace, "confirm", token)
}

// ViewOnline shows an email in the browser.
func (b Builder) ViewOnline(workspace, token string) string {
	return b.join("p", workspace, "view", token)
}

// Subscribe is the hosted subscribe page; form is optional (the default form
// when empty).
func (b Builder) Subscribe(workspace, form string) string {
	if form == "" {
		return b.join("p", workspace, "subscribe")
	}
	return b.join("p", workspace, "subscribe", form)
}

// Archive is the public newsletter archive.
func (b Builder) Archive(workspace string) string { return b.join("p", workspace, "archive") }

// ArchiveIssue is one issue in the public archive.
func (b Builder) ArchiveIssue(workspace, issue string) string {
	return b.join("p", workspace, "archive", issue)
}

// ArchiveFeed is the archive's RSS feed.
func (b Builder) ArchiveFeed(workspace string) string {
	return b.join("p", workspace, "archive.rss")
}

// Report is a read-only shared campaign report.
func (b Builder) Report(token string) string { return b.join("r", token) }
