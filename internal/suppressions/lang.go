package suppressions

import (
	"embed"
	"regexp"
	"strconv"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What Sending providers › Events reads from the suppression sync card
// follows the reader's app language (messages/<locale>.json): which list
// each provider syncs with, why a provider without a suppression API needs
// none, and an import's row. Provider names, the providers' own sync
// errors and the detail ReputeCast stores on a pulled suppression ("Suppressed at
// SES", which no screen shows) stay English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "suppressions")

// text writes a message in a reader's language.
func text(l i18n.Lang, key string, args i18n.Args) string {
	return msgs.M(key).Text(l, args)
}

// SoftBounceDetail is the detail the soft-bounce rule stores with the
// suppression it adds, in English (suppressions.detail): "Soft-bounced 3
// times in 7 days (last: 4.2.2 Mailbox full)". last is the provider's code
// and diagnostic, as it wrote them.
func SoftBounceDetail(count, days int, last string) string {
	return text(i18n.Default, "detail.softBounces", i18n.Args{"count": count, "days": days, "last": last})
}

var reSoftBounces = regexp.MustCompile(`(?s)^Soft-bounced (\d+) times? in (\d+) days? \(last: (.*)\)$`)

// DetailIn is a suppression's stored detail in a reader's language: the
// soft-bounce rule's sentence is ReputeCast's; a provider's diagnostic or a
// person's note reads as stored.
func DetailIn(l i18n.Lang, stored string) string {
	m := reSoftBounces.FindStringSubmatch(stored)
	if m == nil || l.Locale == "" || l.Locale == i18n.English {
		return stored
	}
	count, _ := strconv.Atoi(m[1])
	days, _ := strconv.Atoi(m[2])
	return text(l, "detail.softBounces", i18n.Args{"count": count, "days": days, "last": m[3]})
}
