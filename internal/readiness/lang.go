package readiness

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
)

// The words of Send Readiness follow the reader's app language; they are in
// messages/<locale>.json.
//
// A run is stored once and read later by other people: Review & send, the
// approval page, the overview, digests, placement tests. So every text a
// check, a group, the estimates or the progress show is kept as the
// messages it is written from (Text) beside the English the run stores, and
// readers get the run through In: in their language and region. English
// readers with the standard formats, API clients and MCP agents read the
// English as it was stored, and so does everyone for runs stored before
// their texts were kept.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "readiness")

// A are a message's arguments.
type A = i18n.Args

// say writes one of the package's messages in the request's language:
// problem details, the fixes a person applied, version labels.
func say(ctx context.Context, key string, args A) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// Text is a text of a run, kept for the people who read it later: the
// messages it is written from, each with its arguments. One message is the
// text; several are sentences, each ended with a full stop and joined with
// a space.
//
// Arguments are strings, numbers, other Texts (a phrase inside a sentence)
// and the values the Arg functions make (lists, money, percentages, times,
// language names), which are written in each reader's language and region.
type Text []i18n.Text

// tx is one of the package's messages with its arguments.
func tx(key string, args A) Text { return Text{msgs.M(key).Keep(args)} }

// RawText is text the run didn't write, such as a provider's own words or
// a DNS record: every reader reads it as it is.
func RawText(s string) Text {
	if s == "" {
		return nil
	}
	return Text{{Args: i18n.Args{"text": s}}}
}

// Kept is a message of another package's catalog as a Text (Grow's rows).
func Kept(t i18n.Text) Text {
	if t.IsZero() {
		return nil
	}
	return Text{t}
}

// sentences joins texts as sentences.
func sentences(ts ...Text) Text {
	var out Text
	for _, t := range ts {
		out = append(out, t...)
	}
	return out
}

// Empty reports whether there is no text.
func (t Text) Empty() bool { return len(t) == 0 }

// In writes the text in a reader's language and region. It returns "" when
// a message isn't known any more: the caller shows the English it stored.
func (t Text) In(l i18n.Lang) string { return t.render(l, len(t) > 1) }

// en is the English the run stores.
func (t Text) en() string { return t.In(i18n.Default) }

// sentencesIn writes every part as a sentence, even a single one (Evidence).
func (t Text) sentencesIn(l i18n.Lang) string { return t.render(l, true) }

func (t Text) render(l i18n.Lang, asSentences bool) string {
	parts := make([]string, 0, len(t))
	for _, p := range t {
		s := renderPart(p, l)
		if s == "" && p.ID != "" {
			return ""
		}
		if asSentences {
			s = sentence(s)
		}
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

func renderPart(p i18n.Text, l i18n.Lang) string {
	if p.ID == "" {
		s, _ := p.Args["text"].(string)
		return s
	}
	args := make(i18n.Args, len(p.Args))
	for k, v := range p.Args {
		args[k] = resolve(v, l)
	}
	return i18n.Text{ID: p.ID, Args: args}.In(l)
}

// MarshalJSON stores one message as an object and several as an array.
func (t Text) MarshalJSON() ([]byte, error) {
	if len(t) == 1 {
		return json.Marshal(t[0])
	}
	return json.Marshal([]i18n.Text(t))
}

// UnmarshalJSON reads what MarshalJSON stored.
func (t *Text) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	switch {
	case bytes.Equal(b, []byte("null")):
		*t = nil
		return nil
	case len(b) > 0 && b[0] == '[':
		var many []i18n.Text
		if err := json.Unmarshal(b, &many); err != nil {
			return err
		}
		*t = many
		return nil
	}
	var one i18n.Text
	if err := json.Unmarshal(b, &one); err != nil {
		return err
	}
	*t = Text{one}
	return nil
}

// Arguments written in each reader's language and region. Kept texts store
// them as small objects, so a reader in another region reads a date or an
// amount the way they write it.

// ArgAnd joins items as a list: "SES, Postal and Postmark", "SES, Postal và
// Postmark". Items are strings or Texts.
func ArgAnd[T any](items []T) map[string]any { return map[string]any{"and": anys(items)} }

// ArgOr joins alternatives: "Spanish or German".
func ArgOr[T any](items []T) map[string]any { return map[string]any{"or": anys(items)} }

// argJoin joins phrases with a separator that reads the same in every
// language (", ", "; ", " · ").
func argJoin[T any](sep string, items []T) map[string]any {
	return map[string]any{"join": sep, "items": anys(items)}
}

func anys[T any](items []T) []any {
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = v
	}
	return out
}

// ArgMoney is an amount in US dollars: "$2.84", "2,84 US$".
func ArgMoney(v float64) map[string]any { return map[string]any{"usd": v} }

// ArgPercent is a percentage with up to two decimals: "0.6%", "0,04%".
func ArgPercent(v float64) map[string]any { return map[string]any{"pct": math.Round(v*100) / 100} }

// ArgPercent1 is a percentage with one decimal: "3.0%".
func ArgPercent1(v float64) map[string]any { return map[string]any{"pct1": v} }

// ArgTime is a time of day: "09:40".
func ArgTime(t time.Time) map[string]any { return map[string]any{"time": stamp(t)} }

// ArgDayTime is a day and a time: "Mon 12 Oct, 09:40".
func ArgDayTime(t time.Time) map[string]any { return map[string]any{"dayTime": stamp(t)} }

// ArgDay is a day with its weekday: "Mon 12 Oct".
func ArgDay(t time.Time) map[string]any { return map[string]any{"day": stamp(t)} }

// ArgDayMonth is a day without its weekday: "12 Oct".
func ArgDayMonth(t time.Time) map[string]any { return map[string]any{"dayMonth": stamp(t)} }

// ArgDayMonthTime is a day and a time without the weekday: "12 Oct, 09:40".
func ArgDayMonthTime(t time.Time) map[string]any { return map[string]any{"dayMonthTime": stamp(t)} }

// ArgLanguage is a content language's name in the reader's language:
// "Spanish", "Tiếng Tây Ban Nha".
func ArgLanguage(code string) map[string]any { return map[string]any{"lang": code} }

// argSection names a section of the email the way the composer does:
// "hero", "2 columns", "article cards".
func argSection(label string) map[string]any { return map[string]any{"section": label} }

// argDecimal1 is a number with one decimal: "2.1" (a contrast ratio).
func argDecimal1(v float64) map[string]any { return map[string]any{"dec1": v} }

// argCapital starts a phrase with a capital letter: "All caps and money
// words." (v is a string, a Text or another argument).
func argCapital(v any) map[string]any { return map[string]any{"capital": v} }

// argSentence ends a phrase with a full stop when it has none.
func argSentence(v any) map[string]any { return map[string]any{"sentence": v} }

// argLower starts a phrase with a small letter inside a sentence, unless
// its first word is a name: "Accessibility: the hero image has no alt text".
func argLower(v any) map[string]any { return map[string]any{"lower": v} }

// stamp keeps a time in the zone it was given in (the workspace's).
func stamp(t time.Time) string { return t.Format(time.RFC3339) }

// clock is "09:40" today, "Mon 12 Oct, 09:40" on another day.
func clockArg(t, now time.Time, loc *time.Location) map[string]any {
	if loc == nil {
		loc = time.UTC
	}
	lt, ln := t.In(loc), now.In(loc)
	if lt.Year() == ln.Year() && lt.YearDay() == ln.YearDay() {
		return ArgTime(lt)
	}
	return ArgDayTime(lt)
}

// resolve writes an argument in a reader's language and region.
func resolve(v any, l i18n.Lang) any {
	switch x := v.(type) {
	case Text:
		return x.In(l)
	case i18n.Text:
		return Text{x}.In(l)
	case []any:
		return storedText(x).In(l)
	case map[string]any:
		return resolveMap(x, l)
	}
	return v
}

// storedText reads a Text of several parts back from storage.
func storedText(items []any) Text {
	out := make(Text, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			out = append(out, textOf(m))
		}
	}
	return out
}

func textOf(m map[string]any) i18n.Text {
	id, _ := m["id"].(string)
	args, _ := m["args"].(map[string]any)
	return i18n.Text{ID: id, Args: args}
}

func resolveMap(m map[string]any, l i18n.Lang) any {
	if _, ok := m["id"]; ok {
		return Text{textOf(m)}.In(l)
	}
	if _, ok := m["args"]; ok {
		return Text{textOf(m)}.In(l)
	}
	if items, ok := m["and"]; ok {
		return i18n.AndList(l.Locale, strs(items, l))
	}
	if items, ok := m["or"]; ok {
		return i18n.OrList(l.Locale, strs(items, l))
	}
	if sep, ok := m["join"].(string); ok {
		return strings.Join(strs(m["items"], l), sep)
	}
	r := l.Region
	if v, ok := toNumber(m["usd"]); ok {
		return r.Money(v, 2)
	}
	if v, ok := toNumber(m["pct"]); ok {
		return r.Percent(v, decimals(v))
	}
	if v, ok := toNumber(m["pct1"]); ok {
		return r.Percent(v, 1)
	}
	if v, ok := toNumber(m["dec1"]); ok {
		return r.Decimal(v, 1)
	}
	if v, ok := m["capital"]; ok {
		return capitalize(fmt.Sprint(resolve(v, l)))
	}
	if v, ok := m["sentence"]; ok {
		return sentence(fmt.Sprint(resolve(v, l)))
	}
	if v, ok := m["lower"]; ok {
		return lowerFirst(fmt.Sprint(resolve(v, l)))
	}
	for k, write := range timeFormats {
		if s, ok := m[k].(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return write(r, t)
			}
			return s
		}
	}
	if c, ok := m["lang"].(string); ok {
		return languageIn(l, c)
	}
	if s, ok := m["section"].(string); ok {
		return sectionIn(l, s)
	}
	return ""
}

var timeFormats = map[string]func(i18n.Region, time.Time) string{
	"time":         func(r i18n.Region, t time.Time) string { return r.Time(t, "") },
	"dayTime":      func(r i18n.Region, t time.Time) string { return r.ShortDate(t) + ", " + r.Time(t, "") },
	"day":          func(r i18n.Region, t time.Time) string { return r.ShortDate(t) },
	"dayMonth":     func(r i18n.Region, t time.Time) string { return r.DayMonth(t) },
	"dayMonthTime": func(r i18n.Region, t time.Time) string { return r.DayMonth(t) + ", " + r.Time(t, "") },
}

func strs(items any, l i18n.Lang) []string {
	var list []any
	switch x := items.(type) {
	case []any:
		list = x
	case []string:
		list = anys(x)
	}
	out := make([]string, 0, len(list))
	for _, it := range list {
		out = append(out, fmt.Sprint(resolve(it, l)))
	}
	return out
}

func toNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	}
	return 0, false
}

// decimals is how many decimals a percentage keeps (up to two, trailing
// zeros dropped): 0.6 → 1, 0.04 → 2, 5 → 0.
func decimals(v float64) int {
	s := strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
	if _, frac, ok := strings.Cut(s, "."); ok {
		return min(len(frac), 2)
	}
	return 0
}

// languageIn is a content language's name in the reader's language: the
// catalog's English name in English, CLDR's elsewhere ("Tiếng Tây Ban Nha").
func languageIn(l i18n.Lang, code string) string {
	switch l.Locale {
	case i18n.English, i18n.Pseudo, "":
		return languages.NameOf(code)
	}
	base := code
	if i := strings.IndexAny(base, "-_"); i > 0 {
		base = base[:i]
	}
	tag, err := language.Parse(base)
	if err != nil {
		return languages.NameOf(code)
	}
	if n := display.Tags(l.Locale.Tag()).Name(tag); n != "" {
		return n
	}
	return languages.NameOf(code)
}

var columnsLabel = regexp.MustCompile(`^(\d+) columns$`)

// sectionKeys are the words the composer and internal/mjml name sections
// with ("hero", "article cards"), by their message.
var sectionKeys = map[string]string{
	"header": "section.header", "footer": "section.footer", "hero": "section.hero", "raw HTML": "section.rawHtml", "wrapper": "section.wrapper",
	"divider": "section.divider", "image": "section.image", "button": "section.button", "text": "section.text", "article cards": "section.articleCards",
	"product cards": "section.productCards", "poll": "section.poll", "sponsor": "section.sponsor", "upgrade": "section.upgrade",
	"feed items": "section.feedItems", "store items": "section.storeItems", "section": "section.section", "column": "section.column",
	"group": "section.group", "spacer": "section.spacer", "social": "section.social", "navbar": "section.navbar", "table": "section.table",
}

// sectionIn is a section's name in the reader's language; a block's own
// name stays as it is.
func sectionIn(l i18n.Lang, label string) string {
	if m := columnsLabel.FindStringSubmatch(label); m != nil {
		n, _ := strconv.Atoi(m[1])
		return msgs.M("section.columns").Text(l, A{"count": n})
	}
	if k, ok := sectionKeys[label]; ok {
		return msgs.M(k).Text(l, nil)
	}
	return label
}
