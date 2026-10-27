package ingest

import (
	"embed"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// What Sending providers › Events reads from the event log follows the
// reader's app language (messages/<locale>.json): its problem, the bounce
// mailbox's name and the source labels ReputeCast writes ("Double opt-in
// confirmation", "Welcome series · step 3"), which delivery_events.source
// keeps in English and eventOf writes for the reader from the message.
// Campaign and automation names, provider events (codes, diagnostics) and
// the CSV export stay as they are; so do the answers to providers'
// webhooks.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "ingest")

// systemKinds are the kinds of system mail SourceLabel names.
var systemKinds = map[string]bool{"double_opt_in": true, "preference_link": true, "invite": true, "password_reset": true, "api": true, "smtp": true, "placement": true}

// sourceIn writes an event's source label in a reader's language: the
// label of system mail, and an automation's step; a campaign's or an
// automation's name stays as stored.
func sourceIn(l i18n.Lang, r store.IngestEventLogRow) string {
	if l.Locale == "" || l.Locale == i18n.English || r.Source == "" {
		return r.Source
	}
	switch {
	case r.MessageCampaign:
		return r.Source
	case r.MessageAutomation:
		n := r.MessageNode
		if _, err := strconv.Atoi(n); err == nil {
			if name, ok := strings.CutSuffix(r.Source, " · step "+n); ok {
				return msgs.M("source.step").Text(l, i18n.Args{"automation": name, "step": n})
			}
		}
		return r.Source
	case systemKinds[r.MessageKind] && r.Source == SourceLabel("", "", nil, r.MessageKind):
		return msgs.M("source.kind").Text(l, i18n.Args{"kind": r.MessageKind})
	case r.MessageKind == "":
		// The message is gone (or never was, as in the sample data): a
		// system mail's label is still known by its English.
		for kind := range systemKinds {
			if r.Source == SourceLabel("", "", nil, kind) {
				return msgs.M("source.kind").Text(l, i18n.Args{"kind": kind})
			}
		}
	}
	return r.Source
}
