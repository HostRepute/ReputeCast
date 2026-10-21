package pages

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var fieldKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// fieldKey is a contact field's key (an event's date field, dmarc_live_at).
func fieldKey(v string) (string, bool) {
	v = strings.TrimSpace(v)
	return v, v == "" || fieldKeyRe.MatchString(v)
}

// EventInfo is a page's event block: its title, date, start and end
// ("HH:MM"), time zone, joining link, the date field registrations write
// and when the reminders go.
type EventInfo struct {
	Title, Date, Start, End, TZ, Link, Field, Reminders string
}

// FirstEvent is a page's first event block (ok false: none).
func FirstEvent(src string) (EventInfo, bool) {
	var out EventInfo
	found := false
	walk(parse(src), func(n *html.Node) bool {
		if found {
			return false
		}
		if n.Type != html.ElementNode || n.Data != ElEvent {
			return true
		}
		found = true
		out = EventInfo{Title: attr(n, "title"), Date: attr(n, "date"), Start: attr(n, "start"), End: attr(n, "end"), TZ: attr(n, "tz"), Link: attr(n, "link"),
			Field: attr(n, "field"), Reminders: attr(n, "reminders")}
		return false
	})
	return out, found
}
