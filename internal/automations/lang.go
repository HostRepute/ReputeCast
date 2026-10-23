package automations

import (
	"context"
	"embed"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
)

// What people read from this package (step names, conditions, the
// version diff, publish checks, alerts, problems, the starter templates)
// is in messages/<locale>.json, written in the reader's app language
// (i18n.LangFrom).
//
// Most of it is built from parts the way the canvas builds it
// (web/src/features/automations/graph.ts): a condition inside a branch's
// title, a step's name inside a diff line. Such text is a tree of
// i18n.Text: a message whose arguments may be texts themselves. say writes
// a tree in a language. Text stored for later readers (a version's
// summary) keeps the tree beside its English and is written when it's
// read, in the reader's language.
//
// A few nodes have no message (their ID is empty): a name, which reads the
// same in every language (workspace content: a list's or an email's
// name); lists joined the way the language joins them; and lower- or
// upper-casing the first letter of what they hold, as running text needs.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "automations")

// msg returns one of the package's messages; an unknown key panics when
// the package loads (the vars below) or in the tests.
func msg(key string) i18n.Msg { return msgs.M(key) }

// txt is a message with its arguments, as name/value pairs.
func txt(m i18n.Msg, kv ...any) i18n.Text {
	if len(kv) == 0 {
		return m.Keep(nil)
	}
	args := make(i18n.Args, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		args[kv[i].(string)] = kv[i+1]
	}
	return m.Keep(args)
}

// name is text that reads the same in every language: a name, a URL.
func name(s string) i18n.Text { return i18n.Text{Args: i18n.Args{"name": s}} }

// andList joins texts as the language joins a list: "A, B and C".
func andList(items []i18n.Text) i18n.Text { return i18n.Text{Args: i18n.Args{"and": items}} }

// orList joins alternatives: "team or business".
func orList(items []i18n.Text) i18n.Text { return i18n.Text{Args: i18n.Args{"or": items}} }

// joined joins texts with a separator that doesn't depend on the language
// ("; " between a summary's changes, " / " between a route's providers).
func joined(sep string, items []i18n.Text) i18n.Text {
	return i18n.Text{Args: i18n.Args{"join": items, "sep": sep}}
}

// lower lower-cases the first letter of an ordinary word ("Step 2" → "step
// 2", "Gửi email" → "gửi email"; "A/B split" and "SES" stay).
func lower(t i18n.Text) i18n.Text { return i18n.Text{Args: i18n.Args{"lower": t}} }

// lowerType lower-cases a step type's first letter, whatever follows it
// ("Branch" → "branch"; in English "A/B split" → "a/B split", as the
// product always wrote it).
func lowerType(t i18n.Text) i18n.Text { return i18n.Text{Args: i18n.Args{"lowerType": t}} }

// lowercase lower-cases a whole name in English ("A/B split" → "a/b
// split"); other languages lower-case its first word only.
func lowercase(t i18n.Text) i18n.Text { return i18n.Text{Args: i18n.Args{"lowercase": t}} }

// upper starts the text with a capital letter.
func upper(t i18n.Text) i18n.Text { return i18n.Text{Args: i18n.Args{"upper": t}} }

// emptyText reports whether t holds nothing (rows written before their
// text was kept).
func emptyText(t i18n.Text) bool { return t.ID == "" && len(t.Args) == 0 }

// say writes a text in a language (English when it's unset).
func say(l i18n.Lang, t i18n.Text) string {
	if l.Locale == "" {
		l.Locale = i18n.English
	}
	if !l.Region.Valid() {
		l.Region = l.Locale.DefaultRegion()
	}
	if t.ID == "" {
		return sayNode(l, t.Args)
	}
	if len(t.Args) == 0 {
		return t.In(l)
	}
	args := make(i18n.Args, len(t.Args))
	for k, v := range t.Args {
		args[k] = sayArg(l, v)
	}
	return i18n.Text{ID: t.ID, Args: args}.In(l)
}

// sayNode writes a node without a message.
func sayNode(l i18n.Lang, args map[string]any) string {
	switch {
	case args["and"] != nil:
		return i18n.AndList(l.Locale, sayAll(l, args["and"]))
	case args["or"] != nil:
		return i18n.OrList(l.Locale, sayAll(l, args["or"]))
	case args["join"] != nil:
		sep, _ := args["sep"].(string)
		return strings.Join(sayAll(l, args["join"]), sep)
	case args["lower"] != nil:
		return lowerFirstIn(l, sayAny(l, args["lower"]))
	case args["lowerType"] != nil:
		s := sayAny(l, args["lowerType"])
		if l.Locale == i18n.English && s != "" {
			return strings.ToLower(s[:1]) + s[1:]
		}
		return lowerFirstIn(l, s)
	case args["lowercase"] != nil:
		s := sayAny(l, args["lowercase"])
		if l.Locale == i18n.English {
			return strings.ToLower(s)
		}
		return lowerFirstIn(l, s)
	case args["upper"] != nil:
		return capFirstIn(l, sayAny(l, args["upper"]))
	case args["language"] != nil:
		code, _ := args["language"].(string)
		return sayLanguage(l, code)
	case args["dayMonth"] != nil:
		if t, ok := timeArg(args["dayMonth"]); ok {
			return l.Region.DayMonth(t)
		}
	case args["clock"] != nil:
		if t, ok := timeArg(args["clock"]); ok {
			return l.Region.Time(t, "")
		}
	}
	s, _ := args["name"].(string)
	return s
}

// sayArg writes an argument that is a text; other values stay as they are.
func sayArg(l i18n.Lang, v any) any {
	if t, ok := asText(v); ok {
		return say(l, t)
	}
	return v
}

func sayAny(l i18n.Lang, v any) string {
	if t, ok := asText(v); ok {
		return say(l, t)
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func sayAll(l i18n.Lang, v any) []string {
	var out []string
	switch xs := v.(type) {
	case []i18n.Text:
		for _, x := range xs {
			out = append(out, say(l, x))
		}
	case []any:
		for _, x := range xs {
			out = append(out, sayAny(l, x))
		}
	}
	return out
}

// timeArg reads a time: as built, or as it comes back from JSON (RFC 3339,
// with the zone's offset).
func timeArg(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case string:
		t, err := time.Parse(time.RFC3339Nano, x)
		return t, err == nil
	}
	return time.Time{}, false
}

// asText reads a text from a value: as built, or as it comes back from
// JSON (a stored summary).
func asText(v any) (i18n.Text, bool) {
	switch x := v.(type) {
	case i18n.Text:
		return x, true
	case *i18n.Text:
		if x != nil {
			return *x, true
		}
	case map[string]any:
		id, okID := x["id"].(string)
		args, okArgs := x["args"].(map[string]any)
		if !okID && !okArgs {
			return i18n.Text{}, false
		}
		return i18n.Text{ID: id, Args: args}, true
	}
	return i18n.Text{}, false
}

// langOf is the language a request reads in.
func langOf(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// english is how text for agents, API logs and the audit log reads.
var english = i18n.Default

// lowerFirstIn lower-cases a first letter followed by a lower-case one. In
// English it keeps the rule the product has always used (ASCII letters
// only); in other languages any letter counts, as on the canvas.
func lowerFirstIn(l i18n.Lang, s string) string {
	if l.Locale == i18n.English {
		return lowerFirst(s)
	}
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 || !unicode.IsUpper(r) {
		return s
	}
	next, _ := utf8.DecodeRuneInString(s[n:])
	if !unicode.IsLower(next) {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}

// capFirstIn starts s with a capital letter: in English the way the
// product always has (a–z only), in other languages any letter.
func capFirstIn(l i18n.Lang, s string) string {
	if l.Locale == i18n.English {
		return capFirst(s)
	}
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// langName names a content language: the English name ReputeCast has
// always used for English readers, the name in the reader's language
// otherwise (CLDR), lower-cased where running text needs it.
func langName(code string) i18n.Text {
	return i18n.Text{Args: i18n.Args{"language": code}}
}

// sayLanguage writes a content language's name for a reader.
func sayLanguage(l i18n.Lang, code string) string {
	if l.Locale == i18n.English || l.Locale == i18n.Pseudo {
		return languages.NameOf(code)
	}
	// The names every screen uses ("tiếng Đức" inside a sentence).
	if n := languages.NameInSentence(l, code); n != "" && n != code {
		return n
	}
	return languages.NameOf(code)
}
