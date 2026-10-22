package httpgeneric

import (
	"embed"
	"errors"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// What the adapter tells people (its connection fields, default terms, the
// problems of a mapping) reads in each reader's language: the messages are
// in messages/<locale>.json (internal/providers, text.go). What a mapping's
// manifest writes (its name, terms, pricing, fields) is the author's
// English and reads as it is.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.httpgeneric")

var (
	mEventsNone     = msgs.M("eventsNone")
	mTermsNote      = msgs.M("terms.0.note")
	mBodyNotJSON    = msgs.M("mapping.bodyNotJson")
	mMethod         = msgs.M("mapping.method")
	mNoBody         = msgs.M("mapping.noBody")
	mMessageIDTwice = msgs.M("mapping.messageIdTwice")
	mHeaderName     = msgs.M("mapping.headerName")
	mHeaderReserved = msgs.M("mapping.headerReserved")
	mURL            = msgs.M("mapping.url")
	mURLHost        = msgs.M("mapping.urlHost")
	mURLPlainHTTP   = msgs.M("mapping.urlPlainHttp")
	mURLFragment    = msgs.M("mapping.urlFragment")
	mWebhookMapping = msgs.M("mapping.webhookMapping")
	mEncoding       = msgs.M("mapping.encoding")
	mPayloadBody    = msgs.M("mapping.payloadBody")
	mPayloadTime    = msgs.M("mapping.payloadTimestamp")
	mTokenHeader    = msgs.M("mapping.tokenHeader")
	mAuth           = msgs.M("mapping.auth")
	mTypePath       = msgs.M("mapping.typePath")
	mTypeMapsTo     = msgs.M("mapping.typeMapsTo")
)

// sentence is an error whose text is shown to the person as it is, with
// the message it comes from.
type sentence struct {
	msg  string
	text i18n.Text
}

func (s sentence) Error() string { return s.msg }

// say makes a sentence from a message.
func say(m i18n.Msg, args i18n.Args) sentence {
	return sentence{msg: m.Text(i18n.Default, args), text: m.Keep(args)}
}

// fieldError is a FieldError about field, saying m.
func fieldError(field string, m i18n.Msg, args i18n.Args) *FieldError {
	s := say(m, args)
	return &FieldError{Field: field, Message: s.msg, Text: s.text}
}

// fieldErr is a FieldError about field from err: a sentence keeps its
// message, any other error reads as it is.
func fieldErr(field string, err error) *FieldError {
	var s sentence
	if errors.As(err, &s) {
		return &FieldError{Field: field, Message: s.msg, Text: s.text}
	}
	return &FieldError{Field: field, Message: err.Error()}
}

// MessageIn writes the problem in the reader's language (manage reads it
// without importing the adapter).
func (e *FieldError) MessageIn(l i18n.Lang) string { return providers.TextIn(e.Message, e.Text, l) }
