package liquid

import (
	"embed"
	"strconv"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The composer's token menu names the contact fields and the system links
// in the member's app language, and the first-name fallback it suggests is
// written for the email's readers in that language too ("there", "bạn").
// The problems Validate reports read in the reader's language as well,
// except the template parser's own explanation, which is English. They are
// in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "liquid")

var (
	mProblemLine = msgs.M("problem.line")
	mProblemLoop = msgs.M("problem.loopLimit")
	mFallback    = msgs.M("fallback.first_name")
)

// CatalogIn is Catalog in a reader's language: the contact fields' and the
// system links' labels and the first-name fallback read in l. Custom fields
// are the workspace's own and keep their labels; tokens other packages
// register keep theirs.
func CatalogIn(l i18n.Lang, custom []CustomField) []PersonalizationToken {
	out := Catalog(custom)
	for i := range out {
		t := &out[i]
		if t.Kind != "contact" && t.Kind != "system" {
			continue
		}
		if m, err := msgs.Lookup("token." + t.Key); err == nil {
			t.Label = m.Text(l, nil)
		}
		if t.Fallback != "" {
			if m, err := msgs.Lookup("fallback." + t.Key); err == nil {
				t.Fallback = m.Text(l, nil)
			}
		}
	}
	return out
}

// FirstNameFallback is the word a missing first name falls back to in an
// email written in l: "there" ("Hi there"), "bạn".
func FirstNameFallback(l i18n.Lang) string { return mFallback.Text(l, nil) }

// ValidateIn is Validate with its problems in a reader's language.
func ValidateIn(l i18n.Lang, src string) []LiquidProblem {
	out := []LiquidProblem{}
	p := Prepare(src)
	if r, ok := longRange(p); ok {
		out = append(out, LiquidProblem{Message: mProblemLoop.Text(l, i18n.Args{"max": strconv.Itoa(MaxRange), "from": r.from, "to": r.to, "count": strconv.Itoa(r.n)})})
	}
	if _, err := engine.ParseString(p); err != nil {
		out = append(out, problem(l, err, p))
	}
	return out
}
