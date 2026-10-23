package mcpserver

import (
	"embed"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What people read of the MCP server is in their language: Settings ›
// MCP's capability matrix and resources (CellsIn, CapabilitiesIn), and the
// agents' activity: each call's audit entry keeps its summary, target,
// detail, outcome and note as messages beside the English (audit.KeepText,
// audit.KeepParts), and a proposal's notification its title and body.
// What agents read stays in English: the tools' and parameters'
// descriptions, the results and the errors they get back, the prompts.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "mcpserver")

// keep is one of the package's messages kept for later readers.
func keep(key string, args i18n.Args) i18n.Text { return msgs.M(key).Keep(args) }

// text writes one of the package's messages in a language.
func text(l i18n.Lang, key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }

// yesNo is the select value of a condition.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// zone is a workspace's timezone, where its members read its times.
func zone(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// changedFields are the parts of a campaign an edit changed, as kept
// messages for a list: subject, preheader, content.
func changedFields(in updateContentIn) []any {
	var out []any
	if in.Subject != nil {
		out = append(out, keep("field.subject", nil))
	}
	if in.Preheader != nil {
		out = append(out, keep("field.preheader", nil))
	}
	if in.MJML != nil {
		out = append(out, keep("field.content", nil))
	}
	return out
}
