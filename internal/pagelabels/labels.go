// Package pagelabels holds ReputeCast's built-in labels of the public
// pages, the signup forms and the emails they send subscribers (design
// expansion M17, phase 2 brief §3.6): buttons, field labels, messages and
// the confirmation email. The publisher's own text (headings, list
// descriptions, consent text) is page copy, kept by internal/forms.
//
// The labels ship in the languages ReputeCast is translated into: the app
// languages, English and Vietnamese, as an internal/i18n catalog
// (messages/<locale>.json). For any other language a workspace writes them
// itself, starting from "Copy English"; For returns the labels of one
// language either way.
package pagelabels

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
)

//go:embed messages/*.json
var messagesFS embed.FS

var catalog = i18n.MustLoad(messagesFS, "pagelabels")

var (
	keys  = catalog.Keys()
	msgs  = map[string]i18n.Msg{}
	notes = map[string]string{}
)

func init() {
	for _, k := range keys {
		msgs[k] = catalog.M(k)
	}
	raw, err := messagesFS.ReadFile("messages/en.json")
	if err != nil {
		panic(err)
	}
	var src map[string]json.RawMessage
	if err := json.Unmarshal(raw, &src); err != nil {
		panic(err)
	}
	for k, v := range src {
		var e struct {
			Note string `json:"note"`
		}
		if json.Unmarshal(v, &e) == nil {
			notes[k] = e.Note
		}
	}
}

// Label is one built-in label: its key, its English text and the note that
// says what it is for.
type Label struct {
	Key    string `json:"key" example:"form.button"`
	Source string `json:"source" example:"Subscribe"`
	Note   string `json:"note,omitempty"`
}

// All returns every built-in label, in key order.
func All() []Label {
	out := make([]Label, 0, len(keys))
	for _, k := range keys {
		out = append(out, Label{Key: k, Source: msgs[k].Source(), Note: notes[k]})
	}
	return out
}

// Count is how many built-in labels there are.
func Count() int { return len(keys) }

// Known reports whether key is a built-in label.
func Known(key string) bool {
	_, ok := msgs[key]
	return ok
}

// Source is a label's English text.
func Source(key string) string {
	if m, ok := msgs[key]; ok {
		return m.Source()
	}
	return ""
}

// Shipped reports whether ReputeCast ships its labels in a language.
func Shipped(code string) bool { return languages.BuiltinLabels[languages.Base(code)] }

// regions are the region formats of the languages that have one: dates and
// numbers on a page follow the language it's shown in.
var regions = map[string]i18n.Region{"vi": "vi-VN", "de": "de-DE", "es": "es-ES", "fr": "fr-FR", "pt": "pt-BR"}

// Lang is how a language writes: its plural rules and its dates and numbers.
// English keeps ReputeCast's standard formats.
func Lang(code string) i18n.Lang {
	base := languages.Base(code)
	switch base {
	case "", "en":
		return i18n.Lang{Locale: i18n.English, Region: i18n.Standard}
	case "vi":
		return i18n.Lang{Locale: i18n.Vietnamese, Region: regions["vi"]}
	}
	r, ok := regions[base]
	if !ok {
		r = i18n.Standard
	}
	return i18n.Lang{Locale: i18n.Locale(base), Region: r}
}

// Dir is the writing direction of a language: "rtl" or "ltr".
func Dir(code string) string {
	if l, ok := languages.Lookup(languages.Base(code)); ok && l.RTL {
		return "rtl"
	}
	return "ltr"
}

// Set is the built-in labels of one language.
type Set struct {
	code    string
	lang    i18n.Lang
	shipped bool
	written map[string]string
}

// For returns the labels of a language: ReputeCast's own when it ships
// them, else the ones the workspace wrote (written, by key). A label that
// isn't written shows in English.
func For(code string, written map[string]string) Set {
	base := languages.Base(code)
	if base == "" {
		base = "en"
	}
	s := Set{code: base, lang: Lang(base), shipped: Shipped(base)}
	if !s.shipped {
		s.written = written
	}
	return s
}

// English is the English labels.
func English() Set { return For("en", nil) }

// Code is the language's base code ("es").
func (s Set) Code() string {
	if s.code == "" {
		return "en"
	}
	return s.code
}

// Lang is how the language writes dates and numbers.
func (s Set) Lang() i18n.Lang { return s.lang }

func (s Set) custom(key string) string {
	if s.shipped || s.written == nil {
		return ""
	}
	return strings.TrimSpace(s.written[key])
}

// Text formats a label as plain text.
func (s Set) Text(key string, args i18n.Args) string {
	m, ok := msgs[key]
	if !ok {
		return key
	}
	if text := s.custom(key); text != "" {
		if out, err := i18n.Format(s.lang, text, args); err == nil {
			return out
		}
	}
	return m.Text(s.msgLang(), args)
}

// HTML formats a label as HTML: its text and string values are escaped,
// i18n.HTML values are kept, and tags go through tags (<b> works as is).
func (s Set) HTML(key string, args i18n.Args, tags i18n.Tags) string {
	m, ok := msgs[key]
	if !ok {
		return key
	}
	if text := s.custom(key); text != "" {
		if out, err := i18n.FormatHTML(s.lang, text, args, tags); err == nil {
			return out
		}
	}
	return m.HTML(s.msgLang(), args, tags)
}

// msgLang formats ReputeCast's own text: the language when it ships the
// labels, else English (a label the workspace hasn't written), in the
// language's region format.
func (s Set) msgLang() i18n.Lang {
	if s.shipped {
		return s.lang
	}
	return i18n.Lang{Locale: i18n.English, Region: s.lang.Region}
}

// Missing returns the keys the workspace still has to write: none for a
// language ReputeCast ships the labels in.
func (s Set) Missing() []string {
	if s.shipped {
		return nil
	}
	var out []string
	for _, k := range keys {
		if s.custom(k) == "" {
			out = append(out, k)
		}
	}
	return out
}

// Complete reports whether every label has text in the language.
func (s Set) Complete() bool { return s.shipped || len(s.Missing()) == 0 }

// Join joins names the way the language writes a list: "Weekly Digest,
// Product Updates and Beta Program".
func (s Set) Join(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return s.Text("list.and", i18n.Args{"items": strings.Join(items[:len(items)-1], ", "), "last": items[len(items)-1]})
}

// FullDate is an issue's date: "Tuesday 14 October 2026".
func (s Set) FullDate(t time.Time) string {
	if s.Code() == "en" {
		return t.Format("Monday 2 January 2006")
	}
	return s.lang.Region.LongDate(t)
}

// NextDate is when a list sends next: "Tuesday 14 October".
func (s Set) NextDate(t time.Time) string {
	if s.Code() == "en" {
		return t.Format("Monday 2 January")
	}
	return s.lang.Region.LongDate(t)
}

// Stamp is a moment with its time: "3 Oct 2026, 09:14".
func (s Set) Stamp(t time.Time) string {
	if s.Code() == "en" {
		return t.Format("2 Jan 2006, 15:04")
	}
	return s.lang.Region.Date(t) + ", " + s.lang.Region.Time(t, "")
}

// DayMonth is a day of the year: "14 Nov".
func (s Set) DayMonth(t time.Time) string {
	if s.Code() == "en" {
		return t.Format("2 Jan")
	}
	return s.lang.Region.DayMonth(t)
}

// Count writes a whole number: "31,988", "31.988".
func (s Set) Count(n int64) string { return s.lang.Region.Int(n) }

// ErrUnknownLabel is returned for a key that isn't a built-in label.
var ErrUnknownLabel = errors.New("pagelabels: unknown label")

// ArgError says text written for a label uses an argument or tag the
// English label doesn't have.
type ArgError struct {
	Key, Name string
	// Allowed are the English label's arguments and tags.
	Allowed []string
}

func (e *ArgError) Error() string {
	return fmt.Sprintf("pagelabels: %s uses %s, which the English label doesn't", e.Key, e.Name)
}

// Check validates text written for a label: it must be a message the
// formatter reads, using only the English label's arguments and tags.
func Check(key, text string) error {
	m, ok := msgs[key]
	if !ok {
		return ErrUnknownLabel
	}
	want, err := i18n.Arguments(m.Source())
	if err != nil {
		return err
	}
	got, err := i18n.Arguments(text)
	if err != nil {
		return err
	}
	for _, g := range got {
		if !slices.Contains(want, g) {
			return &ArgError{Key: key, Name: g, Allowed: want}
		}
	}
	return nil
}

// Arguments are the arguments and tags a label's English text uses
// ("{publisher}", "<b>"), as shown to the person writing it.
func Arguments(key string) []string {
	m, ok := msgs[key]
	if !ok {
		return nil
	}
	names, _ := i18n.Arguments(m.Source())
	out := make([]string, 0, len(names))
	for _, n := range names {
		if strings.HasPrefix(n, "<") {
			out = append(out, n)
			continue
		}
		out = append(out, "{"+n+"}")
	}
	return out
}
