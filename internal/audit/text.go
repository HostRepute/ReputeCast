package audit

import (
	"encoding/json"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// An entry's lines that people read in the audit log (its summary, detail,
// note…) are written once, in English, when the action happens. So that
// every reader sees them in their own language, a package also keeps them
// as catalog messages (msgs.M(key).Keep(args)) in the entry's metadata,
// under "i18n", beside the English: KeepText and KeepSummary do both.
// internal/auditlog writes kept lines in the reader's language; entries
// written before, and lines that weren't kept, read in English.

// TextsKey is the metadata member that holds an entry's kept lines.
const TextsKey = "i18n"

// SummaryPartsKey holds a summary made of facts joined with " · " (kept
// messages and plain values), under TextsKey.
const SummaryPartsKey = "summary_parts"

// KeepText sets a line of an entry's metadata ("summary", "detail"…): its
// English under key, and the message under TextsKey.
func KeepText(meta map[string]any, key string, t i18n.Text) map[string]any {
	if meta == nil {
		meta = map[string]any{}
	}
	meta[key] = t.In(i18n.Default)
	texts(meta)[key] = t
	return meta
}

// KeepSummary sets an entry's summary made of facts joined with " · ":
// each part is a kept message (i18n.Text) or a plain value (string) that
// reads the same in every language, such as a name or a permission key.
func KeepSummary(meta map[string]any, parts ...any) map[string]any {
	if meta == nil {
		meta = map[string]any{}
	}
	english := make([]string, 0, len(parts))
	kept := make([]any, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case i18n.Text:
			english = append(english, v.In(i18n.Default))
			kept = append(kept, v)
		case string:
			english = append(english, v)
			kept = append(kept, v)
		}
	}
	meta["summary"] = strings.Join(english, " · ")
	texts(meta)[SummaryPartsKey] = kept
	return meta
}

func texts(meta map[string]any) map[string]any {
	if t, ok := meta[TextsKey].(map[string]any); ok {
		return t
	}
	t := map[string]any{}
	meta[TextsKey] = t
	return t
}

// Kept are an entry's kept lines, read from its metadata.
type Kept struct {
	Summary      i18n.Text         `json:"summary"`
	SummaryParts []json.RawMessage `json:"summary_parts"`
	Detail       i18n.Text         `json:"detail"`
	What         i18n.Text         `json:"what"`
	Target       i18n.Text         `json:"target"`
	Note         i18n.Text         `json:"note"`
	Outcome      i18n.Text         `json:"outcome"`
}

// KeptIn reads the kept lines of an entry's metadata.
func KeptIn(metadata []byte) Kept {
	var m struct {
		Kept Kept `json:"i18n"`
	}
	if len(metadata) == 0 || json.Unmarshal(metadata, &m) != nil {
		return Kept{}
	}
	return m.Kept
}

// In writes a kept line in a language, or returns english when it wasn't
// kept (or its message is gone).
func In(t i18n.Text, l i18n.Lang, english string) string {
	if s := t.In(l); s != "" {
		return s
	}
	return english
}

// SummaryIn is the entry's summary in a language: the kept message, the
// kept parts joined with " · ", or english.
func (k Kept) SummaryIn(l i18n.Lang, english string) string {
	if s := k.Summary.In(l); s != "" {
		return s
	}
	if len(k.SummaryParts) == 0 {
		return english
	}
	parts := make([]string, 0, len(k.SummaryParts))
	for _, raw := range k.SummaryParts {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			parts = append(parts, s)
			continue
		}
		var t i18n.Text
		if json.Unmarshal(raw, &t) != nil {
			return english
		}
		x := t.In(l)
		if x == "" {
			return english
		}
		parts = append(parts, x)
	}
	return strings.Join(parts, " · ")
}
