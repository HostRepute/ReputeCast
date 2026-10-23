package auditlog

import (
	"embed"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Both audit logs read in the reader's language (i18n.LangFrom): the
// actors' labels, the action filter, the What line of an entry, its target,
// the links, the sign-in summaries and the problems. Entries keep the
// English they were written with; the lines a package kept as messages
// (audit.KeepText) read in the reader's language, the others in their
// English. Exports and the stream are data: they stay in English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "auditlog")

// say writes one of the package's messages.
func say(l i18n.Lang, key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }

// typeIn names what an entry acted on: "routing rule", "API key". A type
// without a message reads as its words in English.
func typeIn(l i18n.Lang, t string) string {
	if t == "" {
		return ""
	}
	if m, err := msgs.Lookup("type." + t); err == nil {
		return m.Text(l, nil)
	}
	return TypeLabel(t, false)
}

// actorLabelIn is the line under an actor's name: "member", "API key",
// "MCP agent", "automatic", or the label the entry stored ("integration
// key"), in a language.
func actorLabelIn(l i18n.Lang, stored, actorType string) string {
	key := map[string]string{"integration key": "actor.integrationKey"}[stored]
	if stored == "" {
		key = map[string]string{"user": "actor.member", "api_key": "actor.apiKey", "mcp_agent": "actor.mcpAgent"}[actorType]
		if key == "" {
			key = "actor.automatic"
		}
	}
	if key == "" {
		return stored
	}
	return say(l, key, nil)
}

// reasonIn is an entry's stored reason in a language: "missing
// campaigns:send" and the sign-in reasons ReputeCast writes; any other as
// it was written.
func reasonIn(l i18n.Lang, reason string) string {
	if perm, ok := strings.CutPrefix(reason, "missing "); ok && !strings.Contains(perm, " ") {
		return say(l, "reason.missing", i18n.Args{"permission": perm})
	}
	if key, ok := reasons[reason]; ok {
		return say(l, key, nil)
	}
	return reason
}

// reasons are the reasons ReputeCast writes on denied entries, by their English.
var reasons = map[string]string{
	"passkey verification failed": "reason.passkeyFailed",
	"too many wrong codes":        "reason.tooManyCodes",
	"wrong admin password":        "reason.wrongAdminPassword",
}

// categoriesIn are the action filter's choices, named in a language.
func categoriesIn(l i18n.Lang) []AuditCategory {
	out := make([]AuditCategory, len(Categories))
	for i, c := range Categories {
		out[i] = c
		if m, err := msgs.Lookup("category." + c.Key); err == nil {
			out[i].Label = m.Text(l, nil)
		}
	}
	return out
}
