package i18n

import "strings"

// listWords are the words that join the last two items of a list.
var listWords = map[Locale][2]string{
	English:    {"and", "or"},
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
	if l == Pseudo {
		w = accent(w)
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + w + " " + items[len(items)-1]
}

// AndList joins items the way the language writes a list: "A, B and C"
// (English keeps the house style, without the serial comma), "A, B và C".
func AndList(l Locale, items []string) string { return join(l, items, false) }

// OrList joins alternatives: "Marcus Chen or Linh Nguyen", "… hoặc …".
func OrList(l Locale, items []string) string { return join(l, items, true) }
