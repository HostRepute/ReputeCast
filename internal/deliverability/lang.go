package deliverability

import (
	"context"
	"embed"
	"encoding/json"
	"math"
	"regexp"
	"time"

	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// What the Deliverability screen reads from the server follows the reader's
// app language (messages/<locale>.json): the providers' account health
// (status words, rate labels, thresholds and notes), the incident
// timeline's links and the delisting state added to a listing, the
// problems of uploading a DMARC report, requesting a delisting and
// describing a source, and the built-in check's listing incidents and
// notifications, which keep their messages for later readers (incidents
// in incidents.texts, migration 02250; notifications through notify).
// Incidents other areas recorded read in the reader's language when they
// kept their messages too, and in English otherwise. Blocklist, provider
// and asset names, HostRepute's own alerts, DNS results, the CSV export
// and the audit log stay as they are; the reputation reason and the
// "HostRepute unreachable" message stay English for Send Readiness, which
// stores them, and read in the reader's language through the API.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "deliverability")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// text writes a message in a language.
func text(l i18n.Lang, key string, args i18n.Args) string {
	return msgs.M(key).Text(l, args)
}

// pct writes a threshold the way the screen always has, with only the
// decimals it needs ("5%", "0.1%"), in a reader's region format.
func pct(l i18n.Lang, v float64) string {
	d := 0
	for d < 3 && math.Abs(v*math.Pow(10, float64(d))-math.Round(v*math.Pow(10, float64(d)))) > 1e-9 {
		d++
	}
	return l.Region.Percent(v, d)
}

// keptTexts is an incidents.texts value: the messages of its title and
// detail, by the column whose English each one wrote.
func keptTexts(title, detail i18n.Text) []byte {
	out := map[string]i18n.Text{}
	if !title.IsZero() {
		out["title"] = title
	}
	if !detail.IsZero() {
		out["detail"] = detail
	}
	b, err := json.Marshal(out)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

// keptIn is an incident's stored English in a reader's language.
func keptIn(l i18n.Lang, texts []byte, column, english string) string {
	if len(texts) == 0 {
		return english
	}
	var kept map[string]i18n.Text
	if json.Unmarshal(texts, &kept) != nil {
		return english
	}
	return providers.TextIn(english, kept[column], l)
}

var reUnanswered = regexp.MustCompile(`^(\d+) of (\d+) lookups didn't answer; they show as unknown\.$`)

// runErrorIn is why a check run failed or fell short, stored in English,
// in a reader's language: the built-in check's and HostRepute's own
// sentences; anything else (an error from elsewhere) stays as stored.
func runErrorIn(l i18n.Lang, english string) string {
	if english == "" || l.Locale == "" || l.Locale == i18n.English {
		return english
	}
	if english == text(i18n.Default, "run.stopped", nil) {
		return text(l, "run.stopped", nil)
	}
	if m := reUnanswered.FindStringSubmatch(english); m != nil {
		return text(l, "run.unanswered", i18n.Args{"failed": m[1], "total": m[2]})
	}
	return hostrepute.SyncErrorIn(l, english)
}

// dayMonth writes a day without its year in a reader's region ("8 Oct").
func dayMonth(l i18n.Lang, t time.Time) string { return l.Region.DayMonth(t) }
