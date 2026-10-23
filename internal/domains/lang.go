package domains

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What Settings › Senders and a domain's records page read from the server
// follows the reader's app language (messages/<locale>.json): the cells of
// the domains table ("Verified", "Pending CNAME 2 of 3", "3 of 3
// providers"), the records' purposes, notes and statuses, a sender's
// verification badges, the DMARC tip, the provider notes and the problems.
// Everything is written per request; nothing the package stores is text
// for people. DNS words stay as DNS writes them ("pass", "fail", "-all",
// "p=quarantine", record types and names), and so do provider and DNS host
// names and the audit log.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "domains")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// text writes a message in a language.
func text(l i18n.Lang, key string, args i18n.Args) string {
	return msgs.M(key).Text(l, args)
}

// yesNo is a select value.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
