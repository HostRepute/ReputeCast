package state

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The health check's words in providers.status_detail (providers/manage
// writes them, and rows keep the English).
const (
	DetailDown           = "Down"
	DetailDownAuth       = "Down · auth failed"
	DetailWebhooksSilent = "Webhooks silent"
	DetailRejecting      = "Rejecting messages"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providerstate")

var details = map[string]i18n.Msg{
	DetailDown:           msgs.M("detail.down"),
	DetailDownAuth:       msgs.M("detail.downAuth"),
	DetailWebhooksSilent: msgs.M("detail.webhooksSilent"),
	DetailRejecting:      msgs.M("detail.rejecting"),
}

// DetailIn is a provider's status detail for a reader: the health check's
// words in the reader's language, anything else (the account's standing,
// in the provider's words) as stored. English readers get the stored text.
func DetailIn(detail string, l i18n.Lang) string {
	if m, ok := details[detail]; ok && l.Locale != "" && l.Locale != i18n.English {
		return m.Text(l, nil)
	}
	return detail
}
