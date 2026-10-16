// Package languages implements content languages (design expansion M17): the
// workspace's languages with its default and fallback, the language
// versions of campaigns, automation emails and transactional templates
// (statuses, block tracking, compare), which version each recipient gets,
// the Languages group of Send Readiness and results per language.
//
// Other packages build on it through a small API:
//
//   - Lookup, Normalize and Base work with language codes ("Spanish",
//     "es_MX" and "ES" normalize to es and es-MX).
//   - Service.Settings returns a workspace's languages, default and
//     fallback; Settings.Offers says whether a contact's language is one of
//     them.
//   - Service.SetPageCopy records the status of a language's page copy
//     (forms and public pages own the copy itself).
//   - Chooser picks the version a contact gets.
package languages

import (
	"sort"
	"strings"
	"unicode"
)

// Language is one language of the catalog.
type Language struct {
	// Code is the base BCP 47 code ("es").
	Code string `json:"code"`
	// Name is the English name ("Spanish").
	Name string `json:"name"`
	// Native is the language's own name ("Español").
	Native string `json:"native"`
	// RTL marks right-to-left scripts, which aren't supported yet.
	RTL bool `json:"rtl"`
	// Regions lists common regional codes, for display ("es-MX", "es-ES").
	Regions []string `json:"regions"`
}

// BuiltinLabels lists the languages ReputeCast's own page labels (buttons,
// field labels, messages, the confirmation email) ship in: the languages the
// app is translated into.
var BuiltinLabels = map[string]bool{"en": true, "vi": true}

// catalog is every language that can be added to a workspace, by code.
var catalog = []Language{
	{Code: "af", Name: "Afrikaans", Native: "Afrikaans"},
	{Code: "am", Name: "Amharic", Native: "አማርኛ"},
	{Code: "ar", Name: "Arabic", Native: "العربية", RTL: true, Regions: []string{"ar-SA", "ar-EG", "ar-AE"}},
	{Code: "az", Name: "Azerbaijani", Native: "Azərbaycan"},
	{Code: "be", Name: "Belarusian", Native: "Беларуская"},
	{Code: "bg", Name: "Bulgarian", Native: "Български"},
	{Code: "bn", Name: "Bengali", Native: "বাংলা", Regions: []string{"bn-BD", "bn-IN"}},
	{Code: "bs", Name: "Bosnian", Native: "Bosanski"},
	{Code: "ca", Name: "Catalan", Native: "Català"},
	{Code: "cs", Name: "Czech", Native: "Čeština"},
	{Code: "cy", Name: "Welsh", Native: "Cymraeg"},
	{Code: "da", Name: "Danish", Native: "Dansk"},
	{Code: "de", Name: "German", Native: "Deutsch", Regions: []string{"de-DE", "de-AT", "de-CH"}},
	{Code: "el", Name: "Greek", Native: "Ελληνικά"},
	{Code: "en", Name: "English", Native: "English", Regions: []string{"en-US", "en-GB", "en-AU"}},
	{Code: "es", Name: "Spanish", Native: "Español", Regions: []string{"es-MX", "es-ES", "es-AR"}},
	{Code: "et", Name: "Estonian", Native: "Eesti"},
	{Code: "eu", Name: "Basque", Native: "Euskara"},
	{Code: "fa", Name: "Persian", Native: "فارسی", RTL: true},
	{Code: "fi", Name: "Finnish", Native: "Suomi"},
	{Code: "fil", Name: "Filipino", Native: "Filipino"},
	{Code: "fr", Name: "French", Native: "Français", Regions: []string{"fr-FR", "fr-CA", "fr-BE"}},
	{Code: "ga", Name: "Irish", Native: "Gaeilge"},
	{Code: "gl", Name: "Galician", Native: "Galego"},
	{Code: "gu", Name: "Gujarati", Native: "ગુજરાતી"},
	{Code: "he", Name: "Hebrew", Native: "עברית", RTL: true},
	{Code: "hi", Name: "Hindi", Native: "हिन्दी"},
	{Code: "hr", Name: "Croatian", Native: "Hrvatski"},
	{Code: "hu", Name: "Hungarian", Native: "Magyar"},
	{Code: "hy", Name: "Armenian", Native: "Հայերեն"},
	{Code: "id", Name: "Indonesian", Native: "Bahasa Indonesia"},
	{Code: "is", Name: "Icelandic", Native: "Íslenska"},
	{Code: "it", Name: "Italian", Native: "Italiano", Regions: []string{"it-IT", "it-CH"}},
	{Code: "ja", Name: "Japanese", Native: "日本語"},
	{Code: "ka", Name: "Georgian", Native: "ქართული"},
	{Code: "kk", Name: "Kazakh", Native: "Қазақ тілі"},
	{Code: "km", Name: "Khmer", Native: "ខ្មែរ"},
	{Code: "kn", Name: "Kannada", Native: "ಕನ್ನಡ"},
	{Code: "ko", Name: "Korean", Native: "한국어"},
	{Code: "ku", Name: "Kurdish", Native: "Kurdî"},
	{Code: "lo", Name: "Lao", Native: "ລາວ"},
	{Code: "lt", Name: "Lithuanian", Native: "Lietuvių"},
	{Code: "lv", Name: "Latvian", Native: "Latviešu"},
	{Code: "mk", Name: "Macedonian", Native: "Македонски"},
	{Code: "ml", Name: "Malayalam", Native: "മലയാളം"},
	{Code: "mn", Name: "Mongolian", Native: "Монгол"},
	{Code: "mr", Name: "Marathi", Native: "मराठी"},
	{Code: "ms", Name: "Malay", Native: "Bahasa Melayu"},
	{Code: "mt", Name: "Maltese", Native: "Malti"},
	{Code: "my", Name: "Burmese", Native: "မြန်မာ"},
	{Code: "nb", Name: "Norwegian", Native: "Norsk bokmål"},
	{Code: "ne", Name: "Nepali", Native: "नेपाली"},
	{Code: "nl", Name: "Dutch", Native: "Nederlands", Regions: []string{"nl-NL", "nl-BE"}},
	{Code: "pa", Name: "Punjabi", Native: "ਪੰਜਾਬੀ"},
	{Code: "pl", Name: "Polish", Native: "Polski"},
	{Code: "ps", Name: "Pashto", Native: "پښتو", RTL: true},
	{Code: "pt", Name: "Portuguese", Native: "Português", Regions: []string{"pt-BR", "pt-PT"}},
	{Code: "ro", Name: "Romanian", Native: "Română"},
	{Code: "ru", Name: "Russian", Native: "Русский"},
	{Code: "si", Name: "Sinhala", Native: "සිංහල"},
	{Code: "sk", Name: "Slovak", Native: "Slovenčina"},
	{Code: "sl", Name: "Slovenian", Native: "Slovenščina"},
	{Code: "sq", Name: "Albanian", Native: "Shqip"},
	{Code: "sr", Name: "Serbian", Native: "Српски"},
	{Code: "sv", Name: "Swedish", Native: "Svenska"},
	{Code: "sw", Name: "Swahili", Native: "Kiswahili"},
	{Code: "ta", Name: "Tamil", Native: "தமிழ்"},
	{Code: "te", Name: "Telugu", Native: "తెలుగు"},
	{Code: "th", Name: "Thai", Native: "ไทย"},
	{Code: "tr", Name: "Turkish", Native: "Türkçe"},
	{Code: "uk", Name: "Ukrainian", Native: "Українська"},
	{Code: "ur", Name: "Urdu", Native: "اردو", RTL: true},
	{Code: "uz", Name: "Uzbek", Native: "Oʻzbek"},
	{Code: "vi", Name: "Vietnamese", Native: "Tiếng Việt"},
	{Code: "yi", Name: "Yiddish", Native: "ייִדיש", RTL: true},
	{Code: "zh", Name: "Chinese", Native: "中文", Regions: []string{"zh-CN", "zh-TW", "zh-HK"}},
	{Code: "zu", Name: "Zulu", Native: "isiZulu"},
}

var byCode = func() map[string]Language {
	m := make(map[string]Language, len(catalog))
	for _, l := range catalog {
		m[l.Code] = l
	}
	return m
}()

// aliases maps names, native names and three-letter codes to base codes.
var aliases = func() map[string]string {
	m := map[string]string{}
	for _, l := range catalog {
		m[fold(l.Name)] = l.Code
		m[fold(l.Native)] = l.Code
	}
	for k, v := range map[string]string{
		"eng": "en", "spa": "es", "deu": "de", "ger": "de", "fra": "fr", "fre": "fr", "ita": "it", "por": "pt", "vie": "vi",
		"nld": "nl", "dut": "nl", "pol": "pl", "rus": "ru", "jpn": "ja", "kor": "ko", "zho": "zh", "chi": "zh", "ara": "ar",
		"heb": "he", "tur": "tr", "swe": "sv", "dan": "da", "fin": "fi", "nor": "nb", "no": "nb", "nn": "nb", "ces": "cs",
		"cze": "cs", "ell": "el", "gre": "el", "hun": "hu", "ron": "ro", "rum": "ro", "ukr": "uk", "hin": "hi", "ind": "id",
		"tha": "th", "fas": "fa", "per": "fa", "urd": "ur", "iw": "he", "in": "id", "tl": "fil",
		"castellano": "es", "brazilian portuguese": "pt", "mandarin": "zh", "farsi": "fa", "tagalog": "fil",
	} {
		m[k] = v
	}
	return m
}()

// fold lowercases and strips diacritics, for matching names.
func fold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch r {
		case 'á', 'à', 'â', 'ä', 'ã', 'å':
			r = 'a'
		case 'é', 'è', 'ê', 'ë':
			r = 'e'
		case 'í', 'ì', 'î', 'ï':
			r = 'i'
		case 'ó', 'ò', 'ô', 'ö', 'õ':
			r = 'o'
		case 'ú', 'ù', 'û', 'ü':
			r = 'u'
		case 'ç':
			r = 'c'
		case 'ñ':
			r = 'n'
		case 'ệ', 'ế', 'ề', 'ể', 'ễ':
			r = 'e'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Lookup returns the catalog entry of a code; regional codes ("es-MX") give
// their base language.
func Lookup(code string) (Language, bool) {
	l, ok := byCode[Base(code)]
	return l, ok
}

// Catalog returns every language, sorted by English name.
func Catalog() []Language {
	out := append([]Language(nil), catalog...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Base returns the base language of a code: "es-MX" and "es_mx" give "es".
// It returns "" for an empty code.
func Base(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if i := strings.IndexAny(code, "-_"); i >= 0 {
		code = code[:i]
	}
	return code
}

// Normalize turns what a form, an import or the API sends into a BCP 47 code:
// "Spanish", "español", "es_MX", "ES" and "es-mx" become "es" or "es-MX".
// Regional and script subtags are kept ("zh-Hant-TW"). It reports false
// for values it can't read as a language; an empty value is valid and
// stays empty (the contact has no language).
func Normalize(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", true
	}
	if code, ok := aliases[fold(s)]; ok {
		return code, true
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 {
		return "", false
	}
	base := strings.ToLower(parts[0])
	if code, ok := aliases[base]; ok && len(base) == 3 {
		base = code
	}
	if _, ok := byCode[base]; !ok {
		return "", false
	}
	out := []string{base}
	for _, p := range parts[1:] {
		switch {
		case len(p) == 4 && isLetters(p):
			out = append(out, strings.ToUpper(p[:1])+strings.ToLower(p[1:])) // script: Hant
		case (len(p) == 2 && isLetters(p)) || (len(p) == 3 && isDigits(p)):
			out = append(out, strings.ToUpper(p)) // region: MX, 419
		default:
			return "", false
		}
	}
	return strings.Join(out, "-"), true
}

// Suggest returns the closest catalog language for a value Normalize
// refused ("Spanosh" → "es"), or "".
func Suggest(raw string) string {
	f := fold(raw)
	if f == "" {
		return ""
	}
	best, bestD := "", 3
	for _, l := range catalog {
		for _, cand := range []string{fold(l.Name), fold(l.Native)} {
			if d := distance(f, cand); d < bestD {
				best, bestD = l.Code, d
			}
		}
	}
	return best
}

func isLetters(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) || r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// distance is the Levenshtein distance between a and b.
func distance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		cur := make([]int, len(br)+1)
		cur[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(br)]
}

// NameOf returns a code's English name, or the code itself when unknown.
func NameOf(code string) string {
	if l, ok := Lookup(code); ok {
		return l.Name
	}
	return code
}
