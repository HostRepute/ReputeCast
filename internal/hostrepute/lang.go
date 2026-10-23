package hostrepute

import (
	"context"
	"embed"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What people read from the HostRepute connection follows their app
// language (messages/<locale>.json): what each scope is used for, the
// problems of connecting and of adding to monitoring, why HostRepute
// couldn't be reached and the notification that asks to reconnect. The
// sync's error is stored in English, as before
// (hostrepute_connections.last_sync_error); SyncErrorIn writes it in a
// reader's language, rows from before included. HostRepute's alerts are
// HostRepute's own words, and the audit log, its reasons and the API calls
// stay English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "hostrepute")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// scopeKeys are the scopes' messages.
var scopeKeys = map[string]string{ScopeMonitorRead: "scope.monitorRead", ScopeAlertsRead: "scope.alertsRead", ScopeAssetsWrite: "scope.assetsWrite"}

// scopeUse is what a scope is used for, in a reader's language (nothing
// for a scope ReputeCast doesn't ask for).
func scopeUse(l i18n.Lang, scope string) string {
	if key, ok := scopeKeys[scope]; ok {
		return msgs.M(key).Text(l, nil)
	}
	return ""
}

var reTimeout = regexp.MustCompile(`^timeout after (\d+) (s|ms)$`)

// reasonArgs are a reason's arguments for the "unreachable" message.
func reasonArgs(reason string) i18n.Args {
	switch {
	case reason == "timeout":
		return i18n.Args{"reason": "timeout"}
	case reason == "connection refused":
		return i18n.Args{"reason": "refused"}
	case reason == "DNS lookup failed":
		return i18n.Args{"reason": "dns"}
	case reason == "token refresh failed":
		return i18n.Args{"reason": "token"}
	case strings.HasPrefix(reason, "HTTP "):
		return i18n.Args{"reason": "http", "status": strings.TrimPrefix(reason, "HTTP ")}
	}
	if m := reTimeout.FindStringSubmatch(reason); m != nil {
		return i18n.Args{"reason": "timeout_" + m[2], "n": m[1]}
	}
	return i18n.Args{"reason": "network"}
}

// In writes why HostRepute couldn't be reached in a language.
func (e *UnavailableError) In(l i18n.Lang) string {
	args := reasonArgs(e.Reason)
	args["host"], args["attempts"] = e.Host, e.Attempts
	return msgs.M("unreachable").Text(l, args)
}

var reUnreachable = regexp.MustCompile(`^Couldn't reach (.+) \((.+), (\d+) attempts?\)$`)

// SyncErrorIn writes a stored sync error ("Couldn't reach api.hostrepute.com
// (timeout after 30 s, 3 attempts)") in a reader's language; an English
// reader reads it as stored.
func SyncErrorIn(l i18n.Lang, english string) string {
	if english == "" || l.Locale == "" || l.Locale == i18n.English {
		return english
	}
	if english == "Couldn't reach HostRepute" {
		return msgs.M("unreachableHostRepute").Text(l, nil)
	}
	if m := reUnreachable.FindStringSubmatch(english); m != nil {
		n, _ := strconv.Atoi(m[3])
		return (&UnavailableError{Host: m[1], Reason: m[2], Attempts: n}).In(l)
	}
	if rest, ok := strings.CutPrefix(english, "HostRepute sync failed: "); ok {
		return msgs.M("syncFailed").Text(l, i18n.Args{"error": rest})
	}
	return english
}

// monthDay is a day without its year ("Oct 2" in the ReputeCast standard
// format, as these sentences always wrote it).
func monthDay(l i18n.Lang, t time.Time) string {
	if l.Region == i18n.Standard || l.Region == "" {
		return t.Format("Jan 2")
	}
	return l.Region.DayMonth(t)
}
