package awseum

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the manifest tells people (the catalog card's line, the badges'
// notes, the connection fields' labels and hints) reads in each reader's
// language: the messages are in messages/<locale>.json (internal/sms,
// text.go). AWS's own names (Access key ID, External ID, region codes) stay as AWS writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "sms.awseum")
