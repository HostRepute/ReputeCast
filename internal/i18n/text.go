package i18n

import "strings"

// Text is a message kept for readers who come later: a catalog message's ID
// and its arguments, written in each reader's language when it's read. The
// server stores it beside the English for text other people read later (a
// Send Readiness run's checks, a notification, a plan's log).
//
// Arguments go through JSON: numbers come back as float64 and times as
// RFC 3339 strings, which {at, date} and {at, time} read again. Keep a time
// in the zone the reader should see it in (the workspace's).
type Text struct {
	ID   string `json:"id"`
	Args Args   `json:"args,omitempty"`
}

// Keep returns the message with its arguments, to store for later readers.
func (m Msg) Keep(args Args) Text { return Text{ID: m.ID(), Args: args} }

// IsZero reports whether there is no message, as in rows written before
// their text was kept.
func (t Text) IsZero() bool { return t.ID == "" }

// In writes the text in a reader's language. It returns "" when the message
// isn't known (no ID, or a catalog that no longer has it): the caller shows
// the English it stored instead.
func (t Text) In(l Lang) string {
	m, ok := lookupID(t.ID)
	if !ok {
		return ""
	}
	return m.Text(l, t.Args)
}

// lookupID finds a message by its full ID, namespace.key. Keys and
// namespaces may contain dots, so it tries each split, longest namespace
// first.
func lookupID(id string) (Msg, bool) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for i := strings.LastIndexByte(id, '.'); i > 0; i = strings.LastIndexByte(id[:i], '.') {
		if c := registry[id[:i]]; c != nil {
			if _, ok := c.sources[id[i+1:]]; ok {
				return Msg{c: c, key: id[i+1:]}, true
			}
		}
	}
	return Msg{}, false
}
