package smtprelay

import (
	"context"
	"embed"
	"strconv"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What Instance admin › SMTP relay reads from the server follows the
// reader's app language (messages/<locale>.json): the problems of saving
// its settings, and the status checks' details, which the check job stores
// for later readers. A check keeps its English, as before, and its message
// beside it in the stored status; English readers, API clients and checks
// stored before read the English. The relay's SMTP replies go to SMTP
// clients and stay English, and so do certificate errors from Let's
// Encrypt, network errors and the audit log.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "smtprelay")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// check is a status check with its detail kept as a message: the English
// detail is written from it (or given, when the English keeps an older
// wording, such as its dates).
func check(key string, ok bool, port int, msg string, args i18n.Args) storedCheck {
	m := msgs.M(msg)
	c := storedCheck{SMTPRelayCheck: SMTPRelayCheck{Key: key, OK: ok, Port: port, Detail: m.Text(i18n.Default, args)}, Text: m.Keep(args)}
	return c
}

// portArg writes a port as it is ("587", never grouped).
func portArg(port int) string { return strconv.Itoa(port) }

// dateArg is a day for a message: the day the reader's region writes.
func dateArg(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// detailIn is a stored check's detail for a reader: its message in their
// language, or the English for English readers and checks stored before.
func detailIn(l i18n.Lang, c storedCheck) string {
	if c.Text.IsZero() || l.Locale == "" || l.Locale == i18n.English {
		return c.Detail
	}
	if s := c.Text.In(l); s != "" {
		return s
	}
	return c.Detail
}
