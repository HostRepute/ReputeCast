package texting

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The words of the public text message terms page (headings, the closing
// box, the footer) are in messages/<locale>.json; the terms themselves are
// the workspace's own text.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "texting")
