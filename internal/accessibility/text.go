package accessibility

import (
	"strconv"
	"strings"
	"unicode"
)

// quote puts text in curly quotes, as the product writes it: “Click here”.
func quote(s string) string { return "“" + s + "”" }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

func capitalize(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// trimFloat writes 11 as "11" and 11.5 as "11.5".
func trimFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// count writes a number with thousands separators (48,213).
func count(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// plural is "1 issue" or "6 issues".
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return count(n) + " " + many
}

// joinAnd is "a", "a and b", "a, b and c".
func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// languageNames are the English names of common content languages.
var languageNames = map[string]string{
	"ar": "Arabic", "bg": "Bulgarian", "ca": "Catalan", "cs": "Czech", "da": "Danish", "de": "German", "el": "Greek", "en": "English",
	"es": "Spanish", "et": "Estonian", "fa": "Persian", "fi": "Finnish", "fr": "French", "he": "Hebrew", "hi": "Hindi", "hr": "Croatian",
	"hu": "Hungarian", "id": "Indonesian", "it": "Italian", "ja": "Japanese", "ko": "Korean", "lt": "Lithuanian", "lv": "Latvian",
	"ms": "Malay", "nb": "Norwegian Bokmål", "nl": "Dutch", "no": "Norwegian", "pl": "Polish", "pt": "Portuguese", "ro": "Romanian",
	"ru": "Russian", "sk": "Slovak", "sl": "Slovenian", "sr": "Serbian", "sv": "Swedish", "th": "Thai", "tr": "Turkish", "uk": "Ukrainian",
	"ur": "Urdu", "vi": "Vietnamese", "zh": "Chinese",
}

// LanguageName is the English name of a language code ("es-MX" → "Spanish").
func LanguageName(code string) string {
	if n, ok := languageNames[BaseLanguage(code)]; ok {
		return n
	}
	return code
}

// LanguageList is "English, Spanish and German" for the given codes.
func LanguageList(codes []string) string {
	names := make([]string, 0, len(codes))
	for _, c := range codes {
		names = append(names, LanguageName(c))
	}
	return joinAnd(names)
}
