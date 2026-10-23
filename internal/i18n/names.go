package i18n

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Vietnamese names a language with the common noun "tiếng": "Tiếng Anh" on
// its own and at the start of a sentence, "tiếng Anh" inside one ("Thêm
// tiếng Anh"). Translators write {language} wherever the sentence needs it:
// a Vietnamese translation notes, when it loads, which plain arguments sit
// inside a sentence (markInSentence), and a value that is a language's name,
// or a list of them, takes the case its place needs (viNames). Other values
// are kept as they are.

// markInSentence flags the plain arguments that follow other words of their
// sentence. An argument at the start of the message, after the end of a
// sentence, a colon or a "·" separator starts a sentence (a label's value,
// "Ngôn ngữ: Tiếng Việt").
func markInSentence(ns []node) []node {
	out, _ := markFrom(ns, true)
	return out
}

func markFrom(ns []node, start bool) ([]node, bool) {
	out := make([]node, len(ns))
	for i, n := range ns {
		switch v := n.(type) {
		case textNode:
			start = startsSentence(string(v), start)
			out[i] = v
		case argNode:
			v.inSentence = v.kind == "" && !start
			start = false
			out[i] = v
		case pluralNode:
			v.options = markOptions(v.options, start)
			start = false
			out[i] = v
		case selectNode:
			v.options = markOptions(v.options, start)
			start = false
			out[i] = v
		case tagNode:
			v.children, start = markFrom(v.children, start)
			out[i] = v
		default:
			start = false
			out[i] = n
		}
	}
	return out, start
}

func markOptions(opts map[string][]node, start bool) map[string][]node {
	out := make(map[string][]node, len(opts))
	for k, o := range opts {
		out[k], _ = markFrom(o, start)
	}
	return out
}

// startsSentence reports whether the word after text t starts a sentence;
// start is whether the word before t did.
func startsSentence(t string, start bool) bool {
	s := strings.TrimRightFunc(t, unicode.IsSpace)
	if strings.Contains(t[len(s):], "\n") {
		return true
	}
	if s == "" {
		return start
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return strings.ContainsRune(".!?…·:", r)
}

var (
	// viLanguageName is a Vietnamese language name: "tiếng" and up to five
	// capitalised words, maybe with a region in brackets ("Tiếng Anh (Mỹ)").
	viLanguageName = regexp.MustCompile(`^[Tt]iếng(?: \p{Lu}[\p{L}\p{M}'’-]*){1,5}(?: \([^()]+\))?$`)
	viListSep      = regexp.MustCompile(`, | và | hoặc `)
)

// viNames writes a language's name, or a list of them, with "tiếng" in the
// case its place needs: capitalised at the start of a sentence, for the
// first name only. Any other text comes back as it is.
func viNames(s string, inSentence bool) string {
	if !strings.Contains(s, "iếng ") {
		return s
	}
	seps := viListSep.FindAllStringIndex(s, -1)
	items := make([]string, 0, len(seps)+1)
	at := 0
	for _, p := range seps {
		items = append(items, s[at:p[0]])
		at = p[1]
	}
	items = append(items, s[at:])
	for _, it := range items {
		if !viLanguageName.MatchString(it) {
			return s
		}
	}
	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteString(s[seps[i-1][0]:seps[i-1][1]])
		}
		if i == 0 && !inSentence {
			b.WriteString("T" + it[1:])
		} else {
			b.WriteString("t" + it[1:])
		}
	}
	return b.String()
}
