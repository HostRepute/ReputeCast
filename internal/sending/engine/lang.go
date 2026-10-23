package engine

import (
	"context"
	"embed"
	"strconv"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/routing"
)

// What the live monitor reads from the server follows the reader's app
// language (messages/<locale>.json): the problems of its actions, the feed's
// details, how events arrive, the audience and route lines, the names of
// streams and of the "Other" mailbox provider, why a send failed, and the
// incident panel's causes, recommendations and options. The analysis is
// stored with the incident in English, with the messages its sentences come
// from beside it ("i18n" in campaign_incidents.detail, live_lang.go), so
// each reader reads it in their own language; incidents analysed before
// read in their English. Notifications and the incident timeline's titles
// keep their messages too (notify, incidents.texts).
//
// Campaign, list, segment, provider, flow and mailbox provider names, the
// providers' own replies (SMTP codes, API errors, the outage log), a
// reason someone typed, webhook payloads, the audit log and logs stay as
// they are.

//go:embed messages/*.json
var messagesFS embed.FS

var catalog = i18n.MustLoad(messagesFS, "sending.engine")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return catalog.M(key).Text(i18n.LangFrom(ctx), args)
}

// text writes a message in a language.
func text(l i18n.Lang, key string, args i18n.Args) string {
	return catalog.M(key).Text(l, args)
}

// english writes a message in English, as the engine stores it.
func english(key string, args i18n.Args) string {
	return catalog.M(key).Text(i18n.Default, args)
}

// keep is a message kept for later readers.
func keep(key string, args i18n.Args) i18n.Text {
	return catalog.M(key).Keep(args)
}

// isEnglish reports whether a reader reads the English as stored.
func isEnglish(l i18n.Lang) bool { return l.Locale == "" || l.Locale == i18n.English }

// textIn is stored English in a reader's language when its message was kept.
func textIn(english string, t i18n.Text, l i18n.Lang) string {
	return providers.TextIn(english, t, l)
}

// quoted is a name as %q writes it, without the quotes: the messages put
// it between quotes ("October Product Update").
func quoted(name string) string {
	q := strconv.Quote(name)
	return q[1 : len(q)-1]
}

// mailboxIn is a mailbox provider's name in a reader's language ("Other"
// is a word).
func mailboxIn(l i18n.Lang, key string) string { return routing.MailboxNameIn(l, key) }

// streamArg is a stream for the messages' select.
func streamArg(s string) string {
	switch s {
	case "automation", "system", "transactional":
		return s
	}
	return "broadcast"
}

// streamIn names a stream in a reader's language ("Automations").
func streamIn(l i18n.Lang, s string) string {
	return text(l, "stream", i18n.Args{"stream": streamArg(s)})
}

// systemIn is the actor "System" in a reader's language; names stay.
func systemIn(l i18n.Lang, name string) string {
	if name == "System" {
		return text(l, "actor.system", nil)
	}
	return name
}

// StateError explains why an action isn't possible now. Msg is the English;
// In writes it in a reader's language.
type StateError struct {
	Msg  string
	text i18n.Text
}

func (e *StateError) Error() string { return e.Msg }
func (e *StateError) Unwrap() error { return ErrState }

// In is why the action isn't possible, in a reader's language.
func (e *StateError) In(l i18n.Lang) string { return textIn(e.Msg, e.text, l) }

// stateErr is a StateError from a message.
func stateErr(key string, args i18n.Args) error {
	t := keep(key, args)
	return &StateError{Msg: t.In(i18n.Default), text: t}
}

// campaignArgs are a campaign's name, quoted as the problems write it, and
// its status.
func campaignArgs(name, status string) i18n.Args {
	return i18n.Args{"name": quoted(name), "status": status}
}
