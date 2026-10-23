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
//
// A kept message's argument may itself be a kept message (a summary's
// "discovery failed: {error}"): it is written first, in the same language
// and without its final period, as a part of the sentence (Render).

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
	meta[key] = Render(t, i18n.Default)
	texts(meta)[key] = t
	return meta
}

// KeepReason keeps the reason of an entry that ReputeCast wrote itself (a
// group mapping's "no longer in Okta group “Marketing”") and returns its
// English, for Event.Reason; the message goes to the metadata under
// TextsKey. Reasons people type are kept as they typed them.
func KeepReason(meta map[string]any, t i18n.Text) (string, map[string]any) {
	if meta == nil {
		meta = map[string]any{}
	}
	texts(meta)["reason"] = t
	return Render(t, i18n.Default), meta
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
			english = append(english, Render(v, i18n.Default))
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
	Reason       i18n.Text         `json:"reason"`
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
	if s := Render(t, l); s != "" {
		return s
	}
	return english
}

// Render writes a kept message in a language, its kept messages among its
// arguments first (without their final period). It returns "" when a
// message isn't known, like i18n.Text.In.
func Render(t i18n.Text, l i18n.Lang) string {
	nested := false
	for _, v := range t.Args {
		if _, ok := keptText(v); ok {
			nested = true
			break
		}
	}
	if !nested {
		return t.In(l)
	}
	args := make(i18n.Args, len(t.Args))
	for k, v := range t.Args {
		if n, ok := keptText(v); ok {
			s := Render(n, l)
			if s == "" {
				return ""
			}
			v = strings.TrimSuffix(s, ".")
		}
		args[k] = v
	}
	return i18n.Text{ID: t.ID, Args: args}.In(l)
}

// keptText reads an argument that is a kept message: an i18n.Text, or one
// read back from JSON ({"id": …, "args": {…}}).
func keptText(v any) (i18n.Text, bool) {
	switch x := v.(type) {
	case i18n.Text:
		return x, x.ID != ""
	case map[string]any:
		id, _ := x["id"].(string)
		if id == "" || len(x) > 2 {
			return i18n.Text{}, false
		}
		raw, has := x["args"]
		args, _ := raw.(map[string]any)
		if has && args == nil {
			return i18n.Text{}, false
		}
		if !has && len(x) > 1 {
			return i18n.Text{}, false
		}
		return i18n.Text{ID: id, Args: args}, true
	}
	return i18n.Text{}, false
}

// SummaryIn is the entry's summary in a language: the kept message, the
// kept parts joined with " · ", or english.
func (k Kept) SummaryIn(l i18n.Lang, english string) string {
	if s := Render(k.Summary, l); s != "" {
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
		x := Render(t, l)
		if x == "" {
			return english
		}
		parts = append(parts, x)
	}
	return strings.Join(parts, " · ")
}
