package warmup

import (
	"context"
	"embed"
	"encoding/json"
	"math"
	"strconv"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the Warm-up tab reads from the server follows the reader's app
// language (messages/<locale>.json): the problems of its operations, and
// what a plan stores for later readers, its log and its status detail.
// Those keep their English, as before, and the message beside it
// (warmup_events.texts, warmup_plans.texts, migration 02300); an English
// reader, an API client and a row written before read the stored English.
// The audit log, provider names and the providers' status_detail column
// (a word the Sending providers page writes from the status) stay English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "warmup")

// say writes a message in the request's language.
func say(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// kept is one stored text: its English, as the column keeps it, and the
// message it came from.
type kept struct {
	English string
	Text    i18n.Text
}

// keep writes a message's English and keeps the message for later readers.
func keep(key string, args i18n.Args) kept {
	m := msgs.M(key)
	return kept{English: m.Text(i18n.Default, args), Text: m.Keep(args)}
}

// textsJSON is a texts column: the messages by the name of the column whose
// English each one wrote ({} when there is none).
func textsJSON(texts map[string]i18n.Text) []byte {
	out := map[string]i18n.Text{}
	for k, t := range texts {
		if !t.IsZero() {
			out[k] = t
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

// textIn writes a stored English text for a reader: the message kept for
// that column in their language, or the English for English readers, rows
// without a message and messages a catalog no longer has.
func textIn(l i18n.Lang, texts json.RawMessage, column, english string) string {
	if english == "" || len(texts) == 0 || l.Locale == "" || l.Locale == i18n.English {
		return english
	}
	var kept map[string]i18n.Text
	if json.Unmarshal(texts, &kept) != nil {
		return english
	}
	if s := kept[column].In(l); s != "" {
		return s
	}
	return english
}

// actor is who acts, for the log's sentences: their name, or "someone".
func actor(ctx context.Context) i18n.Args {
	if p := api.PrincipalFrom(ctx); p != nil && p.Name != "" {
		return i18n.Args{"named": "yes", "name": p.Name}
	}
	return i18n.Args{"named": "no", "name": ""}
}

// with adds arguments to a copy of args.
func with(args i18n.Args, kv ...any) i18n.Args {
	out := i18n.Args{}
	for k, v := range args {
		out[k] = v
	}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i].(string)] = kv[i+1]
	}
	return out
}

// digits is how many decimals a guard needs (2, 0.5, 0.08): the messages
// write it as a percentage with that many.
func digits(v float64) string {
	for d := 0; d < 2; d++ {
		p := math.Pow(10, float64(d))
		if math.Abs(v*p-math.Round(v*p)) < 1e-9 {
			return strconv.Itoa(d)
		}
	}
	return "2"
}

// mailboxArg names the mailbox provider most bounces came from: a name, or
// "rest" for the other mailbox providers ("none" when there is none).
func mailboxArg(key, name string) string {
	switch {
	case key == "":
		return "none"
	case key == "other":
		return "rest"
	}
	return name
}

// ShiftedEvent is the log entry the sending engine adds when a provider's
// share moved away during an outage and its plan repeats a day: the title
// and body in English, as warmup_events stores them, and the texts column
// that keeps their messages.
func ShiftedEvent(provider string) (title, body string, texts []byte) {
	t, b := keep("log.shiftedTitle", nil), keep("log.shifted", i18n.Args{"provider": provider})
	return t.English, b.English, textsJSON(map[string]i18n.Text{"title": t.Text, "body": b.Text})
}
