package templates

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
)

// The gallery's starter names and descriptions and the problems of saving
// a template read in the member's app language (i18n.LangFrom). A
// starter's sample content (its headlines, buttons, alt texts, the footer's
// links and placeholder address) becomes the workspace's own content when
// someone starts from it, so it is written for the email's readers in that
// member's language. The messages are in messages/<locale>.json; the
// starters' MJML marks each sample text with %%T:key%% (%%J:key%% inside a
// block's JSON settings), the message sample.<key>.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "templates")

var (
	mName      = msgs.M("problem.name")
	mMJML      = msgs.M("problem.mjml")
	mNameTaken = msgs.M("problem.nameTaken")
	mNotFound  = msgs.M("problem.notFound")
	// The gallery's preview of a template whose MJML doesn't compile.
	mPreviewCompile = msgs.M("problem.previewCompile")
	mAddress        = msgs.M("sample.address")
)

// sampleRe is a sample text of a starter: %%T:weekly.intro%%, with
// arguments after bars (%%T:grid.productLine|price=$29%%).
var sampleRe = regexp.MustCompile(`%%([TJ]):([A-Za-z0-9_.]+)((?:\|[A-Za-z]+=[^|%]*)*)%%`)

// samples writes a starter's sample texts in l.
func samples(l i18n.Lang, src string) string {
	firstName := i18n.HTML(`{{ first_name | default: "` + liquid.FirstNameFallback(l) + `" }}`)
	return sampleRe.ReplaceAllStringFunc(src, func(tok string) string {
		m := sampleRe.FindStringSubmatch(tok)
		msg, err := msgs.Lookup("sample." + m[2])
		if err != nil {
			return tok
		}
		args := i18n.Args{"firstName": firstName}
		for _, kv := range strings.Split(strings.TrimPrefix(m[3], "|"), "|") {
			if k, v, ok := strings.Cut(kv, "="); ok {
				args[k] = v
			}
		}
		if m[1] == "J" {
			return jsonText(msg.Text(l, args))
		}
		return msg.HTML(l, args, nil)
	})
}

// jsonText escapes text for a JSON string.
func jsonText(s string) string { return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) }

// DefaultAddressIn is DefaultAddress in a reader's language: the line a
// starter's footer shows until the workspace sets its postal address.
func DefaultAddressIn(l i18n.Lang) string { return mAddress.Text(l, nil) }

// starterName names a starter and says what it holds, in l.
func starterName(l i18n.Lang, st starter) (string, string) {
	return msgs.M("starter."+st.key+".name").Text(l, nil), msgs.M("starter."+st.key+".description").Text(l, nil)
}

// Every starter has its name and description (a missing one stops the
// program at once, like any message).
func init() {
	for _, st := range starters {
		starterName(i18n.Default, st)
	}
}

// notFound is the 404 of a template that isn't there.
func notFound(ctx context.Context) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, mNotFound.Text(i18n.LangFrom(ctx), nil))
}

// mapError is api.MapError with the 404 in the request's language.
func mapError(ctx context.Context, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(ctx)
	}
	return api.MapError(err, "Template")
}
