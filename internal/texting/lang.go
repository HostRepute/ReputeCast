package texting

import (
	"embed"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The words of the public text message terms page (headings, the closing
// box, the footer) are in messages/<locale>.json; the terms themselves are
// the workspace's own text.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "texting")

// The evidence ReputeCast records for a consent change made by text ("Opted
// out by text", "Text message") is stored in English, as every consent
// record is, and read in each reader's language (evidenceIn); evidence
// people or forms gave stays as it was given.
var (
	evOptedOut     = msgs.M("evidence.optedOut")
	evOptedIn      = msgs.M("evidence.optedIn")
	evConfirmed    = msgs.M("evidence.confirmed")
	evTextMessage  = msgs.M("evidence.textMessage")
	evConfirmation = msgs.M("evidence.confirmation")
)

// evidenceWords are those messages by their English.
var evidenceWords = map[string]i18n.Msg{}

func init() {
	for _, k := range msgs.Keys() {
		if strings.HasPrefix(k, "evidence.") {
			m := msgs.M(k)
			evidenceWords[m.Source()] = m
		}
	}
}

// evidenceIn writes a consent record's page or note in a language when
// ReputeCast wrote it; any other stays as it was given.
func evidenceIn(l i18n.Lang, s string) string {
	if m, ok := evidenceWords[s]; ok {
		return m.Text(l, nil)
	}
	return s
}
