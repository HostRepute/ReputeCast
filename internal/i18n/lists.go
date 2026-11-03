package i18n

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// listWords are the words that join the last two items of a list.
var listWords = map[Locale][2]string{
	English:    {"and", "or"},
	German:     {"und", "oder"},
	Spanish:    {"y", "o"},
	French:     {"et", "ou"},
	Vietnamese: {"và", "hoặc"},
}

func join(l Locale, items []string, or bool) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	words, ok := listWords[l]
	if !ok {
		words = listWords[English]
	}
	w := words[0]
	if or {
		w = words[1]
	}
	last := items[len(items)-1]
	if l == Spanish {
		w = spanishJoiner(w, last)
	}
	if l == Pseudo {
		w = accent(w)
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + w + " " + last
}

// spanishJoiner is "y" or "o" as Spanish writes it before the next word:
// "e" before an /i/ sound ("español e inglés"), "u" before an /o/ sound
// ("siete u ocho"), as CLDR's list patterns and Intl.ListFormat do.
func spanishJoiner(w, next string) string {
	s := strings.ToLower(strings.TrimSpace(next))
	switch w {
	case "y":
		// "i…", "hi…" but not "hie…"/"hia…" (a diphthong: "agua y hielo").
		if strings.HasPrefix(s, "i") || strings.HasPrefix(s, "í") {
			return "e"
		}
		if strings.HasPrefix(s, "hi") || strings.HasPrefix(s, "hí") {
			r, _ := utf8.DecodeRuneInString(strings.TrimPrefix(strings.TrimPrefix(s, "hi"), "hí"))
			if !strings.ContainsRune("aeiouáéó", r) {
				return "e"
			}
		}
	case "o":
		if strings.HasPrefix(s, "o") || strings.HasPrefix(s, "ó") || strings.HasPrefix(s, "ho") || strings.HasPrefix(s, "hó") || strings.HasPrefix(s, "8") {
			return "u"
		}
		if strings.HasPrefix(s, "11") {
			r, _ := utf8.DecodeRuneInString(s[2:])
			if !unicode.IsDigit(r) {
				return "u"
			}
		}
	}
	return w
}

// AndList joins items the way the language writes a list: "A, B and C"
// (English keeps the house style, without the serial comma), "A, B y C",
// "A, B và C".
func AndList(l Locale, items []string) string { return join(l, items, false) }

// OrList joins alternatives: "Marcus Chen or Linh Nguyen", "… hoặc …".
func OrList(l Locale, items []string) string { return join(l, items, true) }
