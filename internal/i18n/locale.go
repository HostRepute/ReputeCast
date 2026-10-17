package i18n

import (
	"context"
	"strings"
	"sync/atomic"

	"golang.org/x/text/language"
)

// Locale is an app language: the language of ReputeCast's own interface and
// of the emails it sends to members.
type Locale string

const (
	English    Locale = "en"
	Vietnamese Locale = "vi"
	// Pseudo is the pseudo-locale "Pseudo (+40%)": every letter accented,
	// about 40% longer and wrapped in ⟦ ⟧. It checks layouts, and text that
	// skipped the catalogs shows without the brackets. Available only in
	// development or when the server runs with i18n.pseudo_locale on.
	Pseudo Locale = "en-XA"
)

// appLocales are the languages ReputeCast is translated into, in picker order.
var appLocales = []Locale{English, Vietnamese}

var pseudoOn atomic.Bool

// SetPseudo turns the pseudo-locale on or off for this process.
func SetPseudo(on bool) { pseudoOn.Store(on) }

// PseudoEnabled reports whether the pseudo-locale can be picked.
func PseudoEnabled() bool { return pseudoOn.Load() }

// Locales returns the app languages that can be picked, in picker order:
// English, Vietnamese, then the pseudo-locale when it's on.
func Locales() []Locale {
	out := append([]Locale{}, appLocales...)
	if PseudoEnabled() {
		out = append(out, Pseudo)
	}
	return out
}

func isAppLocale(l Locale) bool {
	for _, x := range appLocales {
		if x == l {
			return true
		}
	}
	return false
}

// Valid reports whether l can be picked right now.
func (l Locale) Valid() bool {
	for _, x := range Locales() {
		if x == l {
			return true
		}
	}
	return false
}

// Name is the language's name in itself: "English", "Tiếng Việt".
func (l Locale) Name() string {
	switch l {
	case English:
		return "English"
	case Vietnamese:
		return "Tiếng Việt"
	case Pseudo:
		return "Pseudo (+40%)"
	}
	return string(l)
}

// Tag is the BCP 47 tag of the language, for plural rules and matching.
func (l Locale) Tag() language.Tag {
	switch l {
	case Vietnamese:
		return language.Vietnamese
	case Pseudo:
		return language.MustParse("en-XA")
	}
	return language.English
}

// pluralTag is the language whose plural rules apply (the pseudo-locale is
// English). Text a workspace wrote in another language (Format with
// Locale("es")) follows that language's rules.
func (l Locale) pluralTag() language.Tag {
	switch l {
	case Vietnamese:
		return language.Vietnamese
	case English, Pseudo, "":
		return language.English
	}
	if t, err := language.Parse(string(l)); err == nil {
		return t
	}
	return language.English
}

// ParseLocale reads a stored or requested app language ("en", "vi",
// "en-XA"), ignoring case. Regional variants of a supported language match
// it ("vi-VN" is Vietnamese), except the pseudo-locale, which must be exact.
func ParseLocale(s string) (Locale, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if strings.EqualFold(s, string(Pseudo)) {
		return Pseudo, PseudoEnabled()
	}
	base, _, _ := strings.Cut(strings.ReplaceAll(s, "_", "-"), "-")
	for _, l := range appLocales {
		if strings.EqualFold(base, string(l)) {
			return l, true
		}
	}
	return "", false
}

// Negotiate picks the app language for an Accept-Language header. ok is
// false when the header is empty or names nothing ReputeCast is translated
// into; the language is English then.
func Negotiate(acceptLanguage string) (l Locale, ok bool) {
	acceptLanguage = strings.TrimSpace(acceptLanguage)
	if acceptLanguage == "" {
		return English, false
	}
	tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
	if err != nil || len(tags) == 0 {
		return English, false
	}
	// The pseudo-locale is only ever asked for by name.
	for _, t := range tags {
		if strings.EqualFold(t.String(), string(Pseudo)) {
			if PseudoEnabled() {
				return Pseudo, true
			}
			break
		}
	}
	supported := make([]language.Tag, len(appLocales))
	for i, x := range appLocales {
		supported[i] = x.Tag()
	}
	_, idx, conf := language.NewMatcher(supported).Match(tags...)
	if conf == language.No || idx < 0 || idx >= len(appLocales) {
		return English, false
	}
	return appLocales[idx], true
}

// Lang is how one person reads: the app language and the region format that
// writes dates and numbers.
type Lang struct {
	Locale Locale
	Region Region
}

// Default is English with the ReputeCast standard formats: what API clients
// get unless they send Accept-Language.
var Default = Lang{Locale: English, Region: Standard}

// For returns the language a locale reads in with its own region format.
func For(l Locale) Lang {
	if l == "" {
		l = English
	}
	return Lang{Locale: l, Region: l.DefaultRegion()}
}

func (l Lang) norm() Lang {
	if l.Locale == "" {
		l.Locale = English
	}
	if !l.Region.known() {
		l.Region = l.Locale.DefaultRegion()
	}
	return l
}

// Resolve picks how a member reads: their own choices when they made them,
// else the instance defaults. A region the member didn't choose follows
// their language, or the instance's default region when the language is the
// instance default too.
func Resolve(memberLocale, memberRegion, defaultLocale, defaultRegion string) Lang {
	def, ok := ParseLocale(defaultLocale)
	if !ok {
		def = English
	}
	defRegion := Region(defaultRegion)
	if !defRegion.known() {
		defRegion = def.DefaultRegion()
	}
	l, chose := ParseLocale(memberLocale)
	if !chose {
		l = def
	}
	if r := Region(memberRegion); r.known() {
		return Lang{Locale: l, Region: r}
	}
	if !chose {
		return Lang{Locale: l, Region: defRegion}
	}
	return Lang{Locale: l, Region: l.DefaultRegion()}
}

type ctxKey struct{}

// WithLang returns a context that formats messages for l.
func WithLang(ctx context.Context, l Lang) context.Context {
	return context.WithValue(ctx, ctxKey{}, l.norm())
}

// LangFrom returns the language stored by WithLang, or Default.
func LangFrom(ctx context.Context) Lang {
	l, _ := LangFromContext(ctx)
	return l
}

// LangFromContext is LangFrom, saying whether a language was stored.
func LangFromContext(ctx context.Context) (Lang, bool) {
	if l, ok := ctx.Value(ctxKey{}).(Lang); ok {
		return l, true
	}
	return Default, false
}
