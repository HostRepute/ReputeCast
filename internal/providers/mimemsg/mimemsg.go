// Package mimemsg renders a providers.Message as an RFC 5322 message with
// wneessen/go-mail, so the adapters that send raw MIME (Amazon SES, Postal,
// SMTP) produce byte-for-byte the same structure: multipart/alternative
// (text, then HTML), a Message-ID derived from the ReputeCast message id, and
// every header the sending pipeline set (List-Unsubscribe, Feedback-ID…).
package mimemsg

import (
	"bytes"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	gomail "github.com/wneessen/go-mail"

	"github.com/hostrepute/reputecast/internal/providers"
)

// HeaderMessage is the header that carries the ReputeCast message id; DSNs
// and provider events echo it back.
const HeaderMessage = "X-ReputeCast-Message"

// MessageID returns the Message-ID header value used for m: "<uuid@domain>",
// with the domain of the From address.
func MessageID(m providers.Message) string {
	domain := "reputecast.local"
	if at := strings.LastIndexByte(m.From.Address, '@'); at >= 0 && at < len(m.From.Address)-1 {
		domain = strings.ToLower(m.From.Address[at+1:])
	}
	return fmt.Sprintf("<%s@%s>", m.ID, domain)
}

// Build returns the raw message and its Message-ID. extra headers (adapter
// specific, such as X-Postal-Tag) are added after the message's own.
func Build(m providers.Message, now time.Time, extra map[string]string) ([]byte, string, error) {
	msg, id := New(m, now, extra)
	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		return nil, "", fmt.Errorf("render message %s: %w", m.ID, err)
	}
	return buf.Bytes(), id, nil
}

// New returns the go-mail message for m, for adapters that send it with a
// go-mail client, and its Message-ID.
func New(m providers.Message, now time.Time, extra map[string]string) (*gomail.Msg, string) {
	msg := gomail.NewMsg(gomail.WithNoDefaultUserAgent(), gomail.WithCharset(gomail.CharsetUTF8))
	from := m.From
	msg.FromMailAddress(&from)
	to := m.To
	msg.ToMailAddress(&to)
	if m.ReplyTo != nil {
		msg.ReplyToMailAddress(m.ReplyTo)
	}
	msg.Subject(m.Subject)
	id := MessageID(m)
	msg.SetMessageIDWithValue(strings.Trim(id, "<>"))
	msg.SetDateWithValue(now)
	switch {
	case m.Text != "" && m.HTML != "":
		msg.SetBodyString(gomail.TypeTextPlain, m.Text)
		msg.AddAlternativeString(gomail.TypeTextHTML, m.HTML)
	case m.HTML != "":
		msg.SetBodyString(gomail.TypeTextHTML, m.HTML)
	default:
		msg.SetBodyString(gomail.TypeTextPlain, m.Text)
	}
	headers := map[string]string{HeaderMessage: m.ID.String()}
	for k, v := range m.Headers {
		headers[k] = v
	}
	for k, v := range extra {
		headers[k] = v
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !validHeaderName(k) || reserved[strings.ToLower(k)] {
			continue
		}
		v := strings.NewReplacer("\r", " ", "\n", " ").Replace(headers[k])
		msg.SetGenHeaderPreformatted(gomail.Header(k), v)
	}
	return msg, id
}

// NewWithExtras is New plus copies and attachments (transactional mail).
// Bcc addresses are envelope recipients only: go-mail never writes them in
// the headers.
func NewWithExtras(m providers.Message, x providers.MessageExtras, now time.Time, extra map[string]string) (*gomail.Msg, string) {
	msg, id := New(m, now, extra)
	for _, a := range x.Cc {
		addr := a
		msg.AddCcMailAddress(&addr)
	}
	for _, a := range x.Bcc {
		addr := a
		msg.AddBccMailAddress(&addr)
	}
	for _, a := range x.Attachments {
		opts := []gomail.FileOption{}
		if a.ContentType != "" {
			opts = append(opts, gomail.WithFileContentType(gomail.ContentType(a.ContentType)))
		}
		if a.ContentID != "" {
			opts = append(opts, gomail.WithFileContentID(a.ContentID))
			_ = msg.EmbedReader(a.Filename, bytes.NewReader(a.Content), opts...)
			continue
		}
		_ = msg.AttachReader(a.Filename, bytes.NewReader(a.Content), opts...)
	}
	return msg, id
}

// BuildWithExtras is Build plus copies and attachments.
func BuildWithExtras(m providers.Message, x providers.MessageExtras, now time.Time, extra map[string]string) ([]byte, string, error) {
	msg, id := NewWithExtras(m, x, now, extra)
	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		return nil, "", fmt.Errorf("render message %s: %w", m.ID, err)
	}
	return buf.Bytes(), id, nil
}

// reserved headers are set from the message fields and never overridden.
var reserved = map[string]bool{
	"from": true, "to": true, "cc": true, "bcc": true, "reply-to": true, "subject": true,
	"message-id": true, "date": true, "mime-version": true, "content-type": true, "content-transfer-encoding": true,
}

func validHeaderName(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		if r <= 32 || r >= 127 || r == ':' {
			return false
		}
	}
	return true
}

// Address formats an address for APIs that take "Name <addr>" strings.
func Address(a mail.Address) string {
	if a.Name == "" {
		return a.Address
	}
	return a.String()
}
