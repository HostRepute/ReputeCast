package i18n

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The pseudo-locale's letters: every letter that has an accented form gets
// one. web/src/i18n/pseudo.ts uses the same table.
var accents = map[rune]rune{
	'a': 'á', 'b': 'b', 'c': 'ç', 'd': 'ð', 'e': 'é', 'f': 'ƒ', 'g': 'ĝ', 'h': 'ĥ', 'i': 'î', 'j': 'ĵ', 'k': 'ķ', 'l': 'ļ', 'm': 'm',
	'n': 'ñ', 'o': 'ö', 'p': 'þ', 'q': 'q', 'r': 'ŕ', 's': 'š', 't': 'ţ', 'u': 'û', 'v': 'v', 'w': 'ŵ', 'x': 'x', 'y': 'ý', 'z': 'ž',
	'A': 'Á', 'B': 'B', 'C': 'Ç', 'D': 'Ð', 'E': 'É', 'F': 'F', 'G': 'Ĝ', 'H': 'Ĥ', 'I': 'Î', 'J': 'Ĵ', 'K': 'Ķ', 'L': 'Ļ', 'M': 'M',
	'N': 'Ñ', 'O': 'Ö', 'P': 'Þ', 'Q': 'Q', 'R': 'Ŕ', 'S': 'Š', 'T': 'Ţ', 'U': 'Û', 'V': 'V', 'W': 'Ŵ', 'X': 'X', 'Y': 'Ý', 'Z': 'Ž',
}

// accent writes text in the pseudo-locale's letters. Names and acronyms
// (a word with a capital after its first letter: ReputeCast, SES, A/B) keep
// their spelling.
func accent(s string) string {
	var b strings.Builder
	word := []rune{}
	flush := func() {
		keep := false
		for i, c := range word {
			if i > 0 && unicode.IsUpper(c) {
				keep = true
				break
			}
		}
		for _, c := range word {
			if a, ok := accents[c]; ok && !keep {
				b.WriteRune(a)
			} else {
				b.WriteRune(c)
			}
		}
		word = word[:0]
	}
	for _, c := range s {
		if unicode.IsSpace(c) {
			flush()
			b.WriteRune(c)
			continue
		}
		word = append(word, c)
	}
	flush()
	return b.String()
}

// pseudoWrap brackets a pseudo-locale string and makes it about 40% longer:
// a space and middle dots, so that ⟦Šéñţ ·⟧ shows a cut-off text by its
// missing ⟧.
func pseudoWrap(text string) (open, close string) {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return "", ""
	}
	dots := int(math.Round(float64(n)*0.4)) - 1
	if dots < 1 {
		dots = 1
	}
	return "⟦", " " + strings.Repeat("·", dots) + "⟧"
}
