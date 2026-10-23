package growth

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems a member reads (a custom period that can't be used) and the
// sources' names the segment builder shows follow the reader's app
// language. They are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "growth")

var (
	mPeriodPickDays   = msgs.M("period.pickDays")
	mPeriodEndsBefore = msgs.M("period.endsBefore")
	mPeriodTooLong    = msgs.M("period.tooLong")

	// A source's name where the app has no words of its own (the segment
	// builder's growth sources); the Growth screen names sources itself.
	mLabelForm             = msgs.M("label.form")
	mLabelSubscribePage    = msgs.M("label.subscribePage")
	mLabelAPIKeyNamed      = msgs.M("label.apiKeyNamed")
	mLabelAPI              = msgs.M("label.api")
	mLabelIntegration      = msgs.M("label.integration")
	mLabelPerson           = msgs.M("label.person")
	mLabelPreferenceCenter = msgs.M("label.preferenceCenter")
	mLabelImportOn         = msgs.M("label.importOn")
	mLabelImport           = msgs.M("label.import")
	mLabelFile             = msgs.M("label.file")
	mLabelOther            = msgs.M("label.other")
	mLabelUnknown          = msgs.M("label.unknown")
)

// periodError is a custom period that can't be used. Page words it in the
// reader's language, as a problem on the query parameter at location.
type periodError struct {
	location string
	msg      i18n.Msg
}

func (e periodError) Error() string { return e.msg.Text(i18n.Default, nil) }
