package accessibility

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// vaguePhrases are link texts that don't say where the link goes, by base
// language. Other languages get the bare-URL and no-text checks only.
var vaguePhrases = map[string][]string{
	"en": {"click here", "click", "here", "click here to read more", "read more", "more", "learn more", "find out more", "more info",
		"more information", "see more", "this link", "link", "click this link", "tap here", "go here", "continue reading", "read on"},
	"es": {"haz clic aquí", "haga clic aquí", "hacer clic aquí", "clic aquí", "click aquí", "pulsa aquí", "pulse aquí", "aquí", "leer más",
		"más", "más información", "ver más", "saber más", "este enlace", "enlace"},
	"de": {"hier klicken", "klicken sie hier", "klick hier", "hier", "mehr", "weiterlesen", "mehr lesen", "mehr erfahren",
		"mehr informationen", "mehr infos", "dieser link", "link", "hier lesen", "zum artikel"},
	"fr": {"cliquez ici", "cliquer ici", "clique ici", "ici", "lire la suite", "en savoir plus", "plus", "plus d'informations",
		"plus d’informations", "voir plus", "ce lien", "lien"},
	"pt": {"clique aqui", "clica aqui", "carregue aqui", "aqui", "leia mais", "ler mais", "saiba mais", "mais", "ver mais",
		"mais informações", "este link", "link"},
	"vi": {"bấm vào đây", "nhấp vào đây", "nhấn vào đây", "click vào đây", "bấm vào", "tại đây", "ở đây", "đây", "xem thêm",
		"đọc thêm", "tìm hiểu thêm", "thêm", "liên kết này", "liên kết"},
}

var phraseIndex = func() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for lang, list := range vaguePhrases {
		m := map[string]bool{}
		for _, p := range list {
			m[normalizeText(p)] = true
		}
		out[lang] = m
	}
	return out
}()

// BaseLanguage is the base of a BCP 47 code: "es-MX" → "es".
func BaseLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if i := strings.IndexAny(code, "-_"); i >= 0 {
		code = code[:i]
	}
	return code
}

// HasPhrases reports whether the language has a list of phrases that say
// nothing.
func HasPhrases(lang string) bool { return phraseIndex[BaseLanguage(lang)] != nil }

// normalizeText lower-cases, composes accents (NFC) and drops the
// punctuation, arrows and spaces around a link's text.
func normalizeText(s string) string {
	s = norm.NFC.String(strings.ToLower(s))
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r)
	})
}

// SaysNothing reports whether a link's text is a phrase that doesn't say
// where the link goes in the given language.
func SaysNothing(text, lang string) bool {
	m := phraseIndex[BaseLanguage(lang)]
	if m == nil {
		return false
	}
	return m[normalizeText(text)]
}

var bareHost = regexp.MustCompile(`(?i)^(www\.)?([a-z0-9-]+\.)+[a-z]{2,}(:\d+)?(/\S*)?$`)

// BareURL reports whether a link's text is an address rather than words:
// "https://docs.acme.io/migrate-v2", "www.acme.io", "docs.acme.io/flags".
func BareURL(text, href string) bool {
	t := strings.TrimSpace(text)
	if t == "" || strings.ContainsAny(t, " \t\n") {
		return false
	}
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	if !bareHost.MatchString(t) {
		return false
	}
	// "acme.io" alone is a name people say; it's a bare URL when it is the
	// link's own address.
	if u, err := url.Parse(href); err == nil && u.Host != "" {
		host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
		if strings.TrimPrefix(strings.TrimRight(lower, "/"), "www.") == host {
			return true
		}
	}
	return strings.Contains(t, "/") || strings.HasPrefix(lower, "www.")
}
