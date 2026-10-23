package audience

import (
	"encoding/json"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Consent events and suppressions keep, beside the English they store for
// people who read it later (a timeline event's detail and note, a
// suppression's context and original code), the message that English came
// from (i18n.Text) in their texts column, keyed by the English column's
// name. A reader in another language sees the message in their language;
// English readers, API clients and rows written before a message was kept
// see the stored English.

// KeptTexts is a texts column holding these messages, by the name of the
// column whose English each one wrote: "detail", "note" (consent events),
// "context", "original_code" (suppressions). Messages without an ID are
// left out; nil when none is left, which stores {}.
func KeptTexts(texts map[string]i18n.Text) json.RawMessage {
	out := map[string]i18n.Text{}
	for k, t := range texts {
		if !t.IsZero() {
			out[k] = t
		}
	}
	if len(out) == 0 {
		return nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil
	}
	return b
}

// KeptIn writes a stored English text for a reader: the message kept for
// that column in their language, or the English when the reader reads
// English, the row kept no message or its catalog no longer has it.
func KeptIn(l i18n.Lang, texts json.RawMessage, column, english string) string {
	if english == "" || len(texts) == 0 || l.Locale == "" || l.Locale == i18n.English {
		return english
	}
	var kept map[string]i18n.Text
	if json.Unmarshal(texts, &kept) != nil {
		return english
	}
	if s := kept[column].In(l); s != "" {
		return s
	}
	return english
}
