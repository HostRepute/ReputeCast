package growth

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems a member reads (a custom period that can't be used) follow the
// reader's app language. They are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "growth")

var (
	mPeriodPickDays   = msgs.M("period.pickDays")
	mPeriodEndsBefore = msgs.M("period.endsBefore")
	mPeriodTooLong    = msgs.M("period.tooLong")
)

// periodError is a custom period that can't be used. Page words it in the
// reader's language, as a problem on the query parameter at location.
type periodError struct {
	location string
	msg      i18n.Msg
}

func (e periodError) Error() string { return e.msg.Text(i18n.Default, nil) }
