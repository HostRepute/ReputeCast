package i18n

import (
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// A language's name takes the case its place needs in some languages.
// Vietnamese names a language with the common noun "tiếng": "Tiếng Anh" on
// its own and at the start of a sentence, "tiếng Anh" inside one ("Thêm
// tiếng Anh"). Spanish and French write language names in lower case
// inside a sentence and capitalise them on their own: "Inglés" in a list of
// languages, "Añadir inglés"; "Anglais", "Version en anglais". English and
// German write a name the same everywhere.
//
// Translators write {language} wherever the sentence needs it: such a
// translation notes, when it loads, which plain arguments sit inside a
// sentence (markInSentence), and a value that is a language's name, or a
// list of them, takes the case its place needs (caseNames). Other values
// are kept as they are. web/src/i18n/names.ts is the app's side of the
// same rule.

// LowersLanguageNames reports whether l writes a language's name in lower
// case inside a sentence ("tiếng Anh", "inglés", "anglais").
func LowersLanguageNames(l Locale) bool {
	return l == Vietnamese || l == Spanish || l == French
}

// caseNames writes a language's name, or a list of them, in the case its
// place needs in l. Any other text comes back as it is.
func caseNames(l Locale, s string, inSentence bool) string {
	switch l {
	case Vietnamese:
		return viNames(s, inSentence)
	case Spanish, French:
		return romanceNames(l, s, inSentence)
	}
	return s
}

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

// listSeparators split a list of names in Spanish and French: "Inglés,
// español y alemán", "Anglais, espagnol et allemand".
var listSeparators = map[Locale]*regexp.Regexp{
	Spanish: regexp.MustCompile(`, | y | e | o | u `),
	French:  regexp.MustCompile(`, | et | ou `),
}

// romanceNames writes a Spanish or French language name, or a list of
// them, capitalised at the start of a sentence (the first name only) and in
// lower case elsewhere: "Inglés, español y alemán", "Añadir inglés".
func romanceNames(l Locale, s string, inSentence bool) string {
	sep := listSeparators[l]
	if s == "" || sep == nil {
		return s
	}
	seps := sep.FindAllStringIndex(s, -1)
	items := make([]string, 0, len(seps)+1)
	at := 0
	for _, p := range seps {
		items = append(items, s[at:p[0]])
		at = p[1]
	}
	items = append(items, s[at:])
	for _, it := range items {
		if !isLanguageName(l, it) {
			return s
		}
	}
	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteString(s[seps[i-1][0]:seps[i-1][1]])
		}
		if i == 0 && !inSentence {
			b.WriteString(upperFirst(it))
		} else {
			b.WriteString(lowerFirst(it))
		}
	}
	return b.String()
}

// iso639 are the ISO 639-1 language codes: the names CLDR gives them are the
// language names a message can be given.
const iso639 = "aa ab ae af ak am an ar as av ay az ba be bg bi bm bn bo br bs ca ce ch co cr cs cu cv cy da de dv dz ee el en eo es et eu fa ff fi fj fo fr fy ga gd gl gn gu gv ha he hi ho hr ht hu hy hz ia id ie ig ii ik io is it iu ja jv ka kg ki kj kk kl km kn ko kr ks ku kv kw ky la lb lg li ln lo lt lu lv mg mh mi mk ml mn mr ms mt my na nb nd ne ng nl nn no nr nv ny oc oj om or os pa pi pl ps pt qu rm rn ro ru rw sa sc sd se sg si sk sl sm sn so sq sr ss st su sv sw ta te tg th ti tk tl tn to tr ts tt tw ty ug uk ur uz ve vi vo wa wo xh yi yo za zh zu fil haw"

var (
	namesMu sync.Mutex
	// knownNames are the language names of each language, in lower case.
	knownNames = map[Locale]map[string]bool{}
)

func namesOf(l Locale) map[string]bool {
	namesMu.Lock()
	defer namesMu.Unlock()
	known, ok := knownNames[l]
	if ok {
		return known
	}
	known = map[string]bool{}
	if n := display.Languages(l.Tag()); n != nil {
		for _, code := range strings.Fields(iso639) {
			if name := n.Name(language.Make(code)); name != "" {
				known[strings.ToLower(name)] = true
			}
		}
	}
	knownNames[l] = known
	return known
}

// isLanguageName reports whether s is a language's name in l, maybe with a
// region in brackets ("inglés (Estados Unidos)").
func isLanguageName(l Locale, s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	known := namesOf(l)
	namesMu.Lock()
	defer namesMu.Unlock()
	key := strings.ToLower(s)
	if known[key] {
		return true
	}
	if i := strings.Index(key, " ("); i > 0 && strings.HasSuffix(key, ")") {
		return known[key[:i]]
	}
	return false
}

// LanguageName names a language (a code such as "es" or "pt-BR") in an app
// language as CLDR does, as the name stands on its own: "Inglés",
// "Anglais", "Englisch", "Tiếng Anh". It is "" for English (callers keep
// their own English names) and for a code CLDR doesn't name in l.
func LanguageName(l Locale, code string) string {
	if l == "" || l == English || l == Pseudo || !isAppLocale(l) {
		return ""
	}
	tag, err := language.Parse(code)
	if err != nil {
		return ""
	}
	base, _ := tag.Base()
	n := display.Languages(l.Tag())
	if n == nil {
		return ""
	}
	name := n.Name(language.Make(base.String()))
	if name == "" {
		return ""
	}
	return StandaloneLanguageName(l, name)
}

// StandaloneLanguageName is a language's name as it stands on its own in l:
// Spanish and French capitalise it ("inglés" → "Inglés"). A name a catalog
// gives (a translation of "Spanish") is noted, so the name takes the case
// its place needs inside a sentence too.
func StandaloneLanguageName(l Locale, name string) string {
	if l != Spanish && l != French || name == "" {
		return name
	}
	known := namesOf(l)
	namesMu.Lock()
	known[strings.ToLower(name)] = true
	namesMu.Unlock()
	return upperFirst(name)
}

func upperFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsUpper(r) {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || unicode.IsLower(r) {
		return s
	}
	// An acronym or a name written in capitals keeps them ("EE. UU.").
	if r2, _ := utf8.DecodeRuneInString(s[n:]); unicode.IsUpper(r2) {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}

// KeepsCapitals reports whether l writes a phrase's first letter as it is
// inside a sentence: German capitalises every noun, so a label put inside a
// sentence ("Kampagnen senden") keeps its capital, where English and the
// others start it with a small letter ("send campaigns").
func KeepsCapitals(l Locale) bool { return l == German }

// Quote puts a name between the quotation marks of l: „Weekly Digest“ in
// German, « Weekly Digest » in French (with no-break spaces), “Weekly
// Digest” in English and the others.
func Quote(l Locale, s string) string {
	switch l {
	case German:
		return "„" + s + "“"
	case French:
		return "« " + s + " »"
	}
	return "“" + s + "”"
}
