package audience

// A contact's language (design expansion M17, brief §3.4): the existing
// field, stored as a normalized BCP 47 code, with where it came from. A
// language the workspace doesn't offer is kept (pt-BR), so adding it later
// works at once; until then the contact gets the fallback.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Where a contact's language came from (contacts.language_source).
const (
	LanguageFromImport      = "import"
	LanguageFromForm        = "form"
	LanguageFromPreferences = "preference_center"
	LanguageFromAPI         = "api"
	LanguageFromTeam        = "team"
)

// Values of the Contacts filter besides a language's code.
const (
	LanguageFilterNone       = "none"
	LanguageFilterNotOffered = "not_offered"
)

// NormalizeLanguage turns what an import, the API or a person sends into a
// BCP 47 code: "Spanish", "es_MX" and "ES" become es and es-MX. An empty
// value clears the language. A value that isn't a language answers 422 at
// loc, with the closest match when there is one (meta "closest").
func NormalizeLanguage(raw, loc string) (string, error) {
	return NormalizeLanguageIn(i18n.Default, raw, loc)
}

// NormalizeLanguageIn is NormalizeLanguage with its problem in a reader's
// language.
func NormalizeLanguageIn(rl i18n.Lang, raw, loc string) (string, error) {
	if code, ok := languages.Normalize(raw); ok {
		return code, nil
	}
	v := strings.TrimSpace(raw)
	if code := ClosestLanguage(v); code != "" {
		l, _ := languages.Lookup(code)
		return "", api.Invalid(loc, mNotLanguageNear.Text(rl, i18n.Args{"value": strconv.Quote(v), "name": languageNameIn(rl, l.Code), "code": l.Code})).
			WithMeta("closest", l.Code, "closestName", l.Name)
	}
	return "", api.Invalid(loc, mNotLanguage.Text(rl, i18n.Args{"value": strconv.Quote(v)}))
}

// ClosestLanguage is the catalog language a value that isn't one most
// likely means ("Spanosh", "Spanglish!" and "Span" → es), or "".
func ClosestLanguage(v string) string {
	letters := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || r == ' ' {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(v))
	// Two or three letters are too short to guess from ("xx" is no closer
	// to Chinese than to anything else).
	if len([]rune(letters)) < 4 {
		return ""
	}
	if code := languages.Suggest(v); code != "" {
		return code
	}
	if code := languages.Suggest(letters); code != "" {
		return code
	}
	// A name cut short, or a name with something after it.
	for _, l := range languages.Catalog() {
		name := strings.ToLower(l.Name)
		if strings.HasPrefix(name, letters) || strings.HasPrefix(letters, name) {
			return l.Code
		}
	}
	return ""
}

// BrowserLanguage is the first language of an Accept-Language header,
// normalized ("es-MX,es;q=0.9,en;q=0.8" → "es-MX"), or "" when it names
// none ReputeCast knows.
func BrowserLanguage(header string) string {
	langs := BrowserLanguages(header)
	if len(langs) == 0 {
		return ""
	}
	return langs[0]
}

// BrowserLanguages are the languages of an Accept-Language header,
// normalized, most preferred first; tags ReputeCast doesn't know are left
// out.
func BrowserLanguages(header string) []string {
	type pref struct {
		code string
		q    float64
	}
	var prefs []pref
	for _, part := range strings.Split(header, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if tag == "" || tag == "*" {
			continue
		}
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if _, err := fmt.Sscan(v, &q); err != nil {
				q = 0
			}
		}
		code, ok := languages.Normalize(tag)
		if !ok || code == "" || q <= 0 {
			continue
		}
		prefs = append(prefs, pref{code, q})
		if len(prefs) == 20 {
			break
		}
	}
	// A stable sort keeps the browser's order between equal weights.
	for i := 1; i < len(prefs); i++ {
		for j := i; j > 0 && prefs[j].q > prefs[j-1].q; j-- {
			prefs[j], prefs[j-1] = prefs[j-1], prefs[j]
		}
	}
	out := make([]string, 0, len(prefs))
	for _, p := range prefs {
		out = append(out, p.code)
	}
	return out
}

// ContactLanguageOption is a choice of the profile's language select.
type ContactLanguageOption struct {
	// Code is what choosing it stores: the workspace language, or the
	// contact's own regional code when it belongs to that language.
	Code string `json:"code"`
	Name string `json:"name"`
	// Offered is false for the contact's own language when the workspace
	// doesn't offer it.
	Offered bool `json:"offered"`
}

// ContactLanguage is the profile's language field: the code, where it came
// from, and whether the workspace offers it.
type ContactLanguage struct {
	// Code is the stored code ("es-MX"), "" when the contact has none.
	Code string `json:"code"`
	// Name is the language's English name ("Spanish"), "" when unknown.
	Name string `json:"name"`
	// Offered reports whether the workspace has the language; Gets is the
	// language the contact gets instead when it hasn't (the fallback).
	Offered bool   `json:"offered"`
	Gets    string `json:"gets"`
	// Source is where it came from; SourceName names the import, form,
	// integration or team member ("Brevo", "Newsletter signup", "Zapier").
	Source     string     `json:"source" enum:"import,form,preference_center,api,team,"`
	SourceName string     `json:"sourceName"`
	Browser    bool       `json:"browser" doc:"A signup form read it from the browser"`
	SetAt      *time.Time `json:"setAt,omitempty"`
	// Options are the select's choices: the contact's own language first
	// when the workspace doesn't offer it, then the workspace's languages.
	Options []ContactLanguageOption `json:"options"`
}

// workspaceLanguages returns the workspace's languages (X05's settings).
func (s *Service) workspaceLanguages(ctx context.Context, q *store.Queries, wsID uuid.UUID) (languages.Settings, error) {
	if q == nil {
		q = s.Q
	}
	return (&languages.Service{Q: q}).Settings(ctx, q, wsID)
}

// contactLanguage builds the profile's language field.
func (s *Service) contactLanguage(ctx context.Context, c store.Contact) (ContactLanguage, error) {
	st, err := s.workspaceLanguages(ctx, nil, c.WorkspaceID)
	if err != nil {
		return ContactLanguage{}, err
	}
	code, _ := languages.Normalize(c.Language)
	if code == "" && strings.TrimSpace(c.Language) != "" {
		code = strings.TrimSpace(c.Language) // kept as stored when it can't be read
	}
	rl := i18n.LangFrom(ctx)
	out := ContactLanguage{Code: code, Name: languageNameIn(rl, code), Offered: code == "" || st.Offers(code), Gets: st.Fallback,
		Source: c.LanguageSource, SourceName: c.LanguageSourceName, Browser: c.LanguageBrowser, SetAt: c.LanguageSetAt, Options: []ContactLanguageOption{}}
	if code != "" && st.Offers(code) {
		out.Gets = languages.Base(code)
	}
	if c.LanguageSource == LanguageFromTeam && c.LanguageSetBy != nil {
		if name, err := s.Q.ContactLanguageSetBy(ctx, *c.LanguageSetBy); err == nil {
			out.SourceName = name
		}
	}
	if code != "" && !st.Offers(code) {
		out.Options = append(out.Options, ContactLanguageOption{Code: code, Name: languageNameIn(rl, code)})
	}
	for _, l := range st.Languages {
		name := languageNameIn(rl, l)
		if name == "" {
			name = languages.NameOf(l)
		}
		opt := ContactLanguageOption{Code: l, Name: name, Offered: true}
		if code != "" && languages.Base(code) == l {
			opt.Code = code
		}
		out.Options = append(out.Options, opt)
	}
	return out, nil
}

func languageName(code string) string {
	if code == "" {
		return ""
	}
	if l, ok := languages.Lookup(code); ok {
		return l.Name
	}
	return ""
}

// languageSourceOf is who sets a language through the app or the API: a
// person on the profile, or an API key (named after its integration).
func languageSourceOf(ctx context.Context) (source, name string, by *uuid.UUID) {
	if p := api.PrincipalFrom(ctx); p != nil && !p.IsHuman() {
		return LanguageFromAPI, p.Name, nil
	}
	return LanguageFromTeam, "", actorID(ctx)
}

// FillLanguage sets a signup's language from the browser when the contact
// has none, inside the caller's transaction. form names the form ("Newsletter
// signup", or "" for the hosted subscribe page). It never overwrites a
// language the contact already has.
func FillLanguage(ctx context.Context, q *store.Queries, contactID uuid.UUID, browser, form string, at time.Time) error {
	code, ok := languages.Normalize(browser)
	if !ok || code == "" {
		return nil
	}
	_, err := q.ContactFillLanguage(ctx, store.ContactFillLanguageParams{ID: contactID, Language: code, SourceName: form, At: at})
	return err
}

// SetLanguageFrom records a language set by an import, the API, an
// integration or the preference center, inside the caller's transaction.
func SetLanguageFrom(ctx context.Context, q *store.Queries, contactID uuid.UUID, code, source, name string, at time.Time) error {
	return q.ContactSetLanguage(ctx, store.ContactSetLanguageParams{ID: contactID, Language: code, Source: source, SourceName: name, At: at})
}

// RecordLanguageSource notes where a contact's language came from, after a
// query that sets several fields wrote it.
func RecordLanguageSource(ctx context.Context, q *store.Queries, contactID uuid.UUID, source, name string, at time.Time, by *uuid.UUID) error {
	return q.ContactSetLanguageSource(ctx, store.ContactSetLanguageSourceParams{ID: contactID, Source: source, SourceName: name, At: at, SetBy: by})
}

// languageFilter adds the Contacts filter's language condition: a
// workspace language (its regional codes included), contacts with no
// language, or contacts with a language the workspace doesn't offer.
func (s *Service) languageFilter(ctx context.Context, wsID uuid.UUID, out *filterSQL, value string) error {
	const base = "lower(split_part(replace(c.language, '_', '-'), '-', 1))"
	switch value {
	case "":
		return nil
	case LanguageFilterNone:
		out.add("c.language = ''")
		return nil
	case LanguageFilterNotOffered:
		st, err := s.workspaceLanguages(ctx, nil, wsID)
		if err != nil {
			return err
		}
		out.add("c.language <> '' AND NOT ("+base+" = ANY (?))", st.Languages)
		return nil
	}
	code := languages.Base(value)
	if _, ok := languages.Lookup(code); !ok {
		return api.Invalid("language", say(ctx, mLanguageFilter, nil))
	}
	out.add(base+" = ?", code)
	return nil
}
