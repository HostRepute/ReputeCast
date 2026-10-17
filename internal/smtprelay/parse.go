package smtprelay

import (
	"errors"
	"fmt"
	"io"
	"mime"
	netmail "net/mail"
	"path"
	"strings"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset" // decode the charsets mail clients use
	"github.com/emersion/go-message/mail"

	"github.com/hostrepute/reputecast/internal/transactional"
)

// Headers the relay reads and drops.
const (
	HeaderTags     = "X-ReputeCast-Tags"
	HeaderCategory = "X-ReputeCast-Category"
)

// parseMessage reads a message an app rendered: the From, Reply-To and
// Subject, the HTML and text bodies, attachments and inline images, and
// the X-ReputeCast-Tags (comma-separated) and X-ReputeCast-Category headers.
func parseMessage(r io.Reader) (transactional.RawMessage, error) {
	var out transactional.RawMessage
	mr, err := mail.CreateReader(r)
	if err != nil && !message.IsUnknownCharset(err) {
		return out, err
	}
	h := mr.Header
	from, err := h.AddressList("From")
	if err != nil || len(from) == 0 {
		return out, errors.New("the message has no From header")
	}
	out.From = netmail.Address{Name: from[0].Name, Address: strings.ToLower(from[0].Address)}
	if rt, err := h.AddressList("Reply-To"); err == nil && len(rt) > 0 {
		out.ReplyTo = &netmail.Address{Name: rt[0].Name, Address: strings.ToLower(rt[0].Address)}
	}
	if out.Subject, err = h.Subject(); err != nil {
		out.Subject = h.Get("Subject")
	}
	for _, t := range strings.Split(h.Get(HeaderTags), ",") {
		if t = strings.TrimSpace(t); t != "" {
			out.Tags = append(out.Tags, t)
		}
	}
	out.Category = strings.TrimSpace(h.Get(HeaderCategory))
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil && !message.IsUnknownCharset(err) {
			return out, err
		}
		body, rerr := io.ReadAll(p.Body)
		if rerr != nil {
			return out, rerr
		}
		switch ph := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := ph.ContentType()
			switch {
			case ct == "text/plain" && out.Text == "":
				out.Text = string(body)
			case ct == "text/html" && out.HTML == "":
				out.HTML = string(body)
			case strings.HasPrefix(ct, "text/"):
				// A second text part (a calendar or a footer): kept as text.
				out.Text += "\n" + string(body)
			default:
				out.Attachments = append(out.Attachments, attachmentOf(ph.Header, "", ct, body))
			}
		case *mail.AttachmentHeader:
			ct, _, _ := ph.ContentType()
			name, _ := ph.Filename()
			out.Attachments = append(out.Attachments, attachmentOf(ph.Header, name, ct, body))
		}
	}
	if out.HTML == "" && out.Text == "" {
		return out, fmt.Errorf("the message has no text or HTML body")
	}
	return out, nil
}

func attachmentOf(h message.Header, name, ct string, body []byte) transactional.RawAttachment {
	if ct == "" {
		ct = "application/octet-stream"
	}
	if name == "" {
		name = "attachment"
		if exts, err := mime.ExtensionsByType(ct); err == nil && len(exts) > 0 {
			name += exts[0]
		}
	}
	return transactional.RawAttachment{Filename: path.Base(strings.ReplaceAll(name, "\\", "/")), ContentType: ct,
		ContentID: strings.Trim(h.Get("Content-Id"), "<> "), Content: body}
}
