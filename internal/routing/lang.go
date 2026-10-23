package routing

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// What Sending providers › Routing reads from the server follows the
// reader's app language (messages/<locale>.json): the routing's issues and
// the problems of saving it and of the route tester, and the labels of a
// condition's values that are gone ("deleted list"). Provider, list and
// category names, mailbox provider keys and the audit log stay as they
// are; MailboxName stays English for the code that writes names into its
// own messages, MailboxNameIn writes "Other" in a reader's language.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "routing")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// text writes a message in a language.
func text(l i18n.Lang, key string, args i18n.Args) string {
	return msgs.M(key).Text(l, args)
}

// streamArg is a stream for the messages' select: broadcast, automation,
// transactional or system.
func streamArg(s providers.Stream) string {
	switch s {
	case providers.StreamBroadcast, providers.StreamAutomation, StreamTransactional:
		return string(s)
	}
	return "system"
}

// MailboxNameIn is a mailbox provider's name in a reader's language: names
// stay as they are, "Other" is a word.
func MailboxNameIn(l i18n.Lang, k string) string {
	switch k {
	case Gmail, Outlook, Yahoo, ICloud:
		return MailboxName(k)
	}
	return text(l, "mailbox.other", nil)
}
