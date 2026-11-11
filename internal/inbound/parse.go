package inbound

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/textproto"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	gomessage "github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset" // decode the charsets mail clients use
	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Limits of what a received message keeps.
const (
	// MaxText is the most text kept of a message's body.
	MaxText = 64 << 10
	// maxPart is the most read of one MIME part; a report attachment
	// larger than this isn't read (DMARC's own limit is 20 MiB).
	maxPart = 20 << 20
	// maxParts bounds a message's parts.
	maxParts = 64
)

// Parsed is a received message ReputeCast reads: its headers, its body as
// text and its attachments.
type Parsed struct {
	From, FromName                string
	To, References                []string
	Subject, MessageID, InReplyTo string
	Header                        textproto.MIMEHeader
	Text                          string
	HasHTML                       bool
	Attachments                   []providers.InboundAttachment
	Date                          time.Time
}

// Parse reads a raw RFC 5322 message.
func Parse(raw []byte) (Parsed, error) {
	var out Parsed
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !gomessage.IsUnknownCharset(err) {
		return out, err
	}
	h := mr.Header
	out.Header = textproto.MIMEHeader(h.Header.Map())
	readHeader(&out, h)
	var text, htmlBody string
	for range maxParts {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil && !gomessage.IsUnknownCharset(err) {
			if text != "" || htmlBody != "" {
				break // keep what was readable
			}
			return out, err
		}
		body, rerr := io.ReadAll(io.LimitReader(p.Body, maxPart+1))
		if rerr != nil {
			return out, rerr
		}
		switch ph := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := ph.ContentType()
			switch {
			case ct == "text/plain" && text == "":
				text = string(body)
			case ct == "text/html" && htmlBody == "":
				htmlBody = string(body)
			case ct == "message/rfc822" || ct == "text/rfc822-headers" || ct == "message/feedback-report" || ct == "message/delivery-status":
				out.Attachments = append(out.Attachments, attachment(ph.Header, "", ct, body))
			case strings.HasPrefix(ct, "text/"):
			default:
				out.Attachments = append(out.Attachments, attachment(ph.Header, "", ct, body))
			}
		case *mail.AttachmentHeader:
			ct, _, _ := ph.ContentType()
			name, _ := ph.Filename()
			out.Attachments = append(out.Attachments, attachment(ph.Header, name, ct, body))
		}
	}
	out.HasHTML = htmlBody != ""
	if strings.TrimSpace(text) == "" && htmlBody != "" {
		text = HTMLText(htmlBody)
	}
	out.Text = clipText(text)
	return out, nil
}

func readHeader(out *Parsed, h mail.Header) {
	if from, err := h.AddressList("From"); err == nil && len(from) > 0 {
		out.From, out.FromName = strings.ToLower(from[0].Address), from[0].Name
	} else if v := h.Get("From"); v != "" {
		out.From = strings.ToLower(strings.Trim(v, "<> "))
	}
	for _, k := range []string{"To", "Cc", "Delivered-To", "X-Original-To"} {
		if list, err := h.AddressList(k); err == nil {
			for _, a := range list {
				out.To = append(out.To, strings.ToLower(a.Address))
			}
		}
	}
	if s, err := h.Subject(); err == nil {
		out.Subject = s
	} else {
		out.Subject = h.Get("Subject")
	}
	if ids, err := h.MsgIDList("Message-Id"); err == nil && len(ids) > 0 {
		out.MessageID = ids[0]
	} else {
		out.MessageID = strings.Trim(h.Get("Message-Id"), "<> ")
	}
	if ids, err := h.MsgIDList("In-Reply-To"); err == nil && len(ids) > 0 {
		out.InReplyTo = ids[0]
	}
	if ids, err := h.MsgIDList("References"); err == nil {
		out.References = ids
	}
	if d, err := h.Date(); err == nil {
		out.Date = d
	}
}

// FromInbound reads what an adapter parsed from its provider's payload: the
// raw message when the provider passed it, else the fields it filled.
func FromInbound(m providers.InboundMessage) (Parsed, error) {
	if len(m.Raw) > 0 {
		p, err := Parse(m.Raw)
		if err != nil {
			return p, err
		}
		if len(m.Attachments) > 0 && len(p.Attachments) == 0 {
			p.Attachments = m.Attachments
		}
		return p, nil
	}
	p := Parsed{Header: textproto.MIMEHeader{}, Subject: m.Subject, MessageID: strings.Trim(m.MessageID, "<> "), InReplyTo: strings.Trim(m.InReplyTo, "<> ")}
	for k, v := range m.Headers {
		for _, x := range v {
			p.Header.Add(k, x)
		}
	}
	if a, err := mail.ParseAddress(m.From); err == nil {
		p.From, p.FromName = strings.ToLower(a.Address), a.Name
	} else {
		p.From = strings.ToLower(strings.Trim(m.From, "<> "))
	}
	for _, t := range m.To {
		if a, err := mail.ParseAddress(t); err == nil {
			p.To = append(p.To, strings.ToLower(a.Address))
		} else if t = strings.ToLower(strings.Trim(t, "<> ")); t != "" {
			p.To = append(p.To, t)
		}
	}
	for _, r := range m.References {
		if r = strings.Trim(r, "<> "); r != "" {
			p.References = append(p.References, r)
		}
	}
	text := m.Text
	if strings.TrimSpace(text) == "" && m.HTML != "" {
		text = HTMLText(m.HTML)
	}
	p.Text, p.HasHTML, p.Attachments, p.Date = clipText(text), m.HTML != "", m.Attachments, m.ReceivedAt
	return p, nil
}

func attachment(h gomessage.Header, name, ct string, body []byte) providers.InboundAttachment {
	if ct == "" {
		ct = "application/octet-stream"
	}
	if name == "" {
		name = "attachment"
		if exts, err := mime.ExtensionsByType(ct); err == nil && len(exts) > 0 {
			name += exts[0]
		}
	}
	a := providers.InboundAttachment{Name: path.Base(strings.ReplaceAll(name, "\\", "/")), ContentType: ct, Size: int64(len(body))}
	if len(body) <= maxPart {
		a.Data = body
	}
	return a
}

// clipText keeps at most MaxText bytes, whole runes, with normalized line ends.
func clipText(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	s = strings.TrimSpace(s)
	if len(s) <= MaxText {
		return s
	}
	s = s[:MaxText]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

var (
	spaceRun = regexp.MustCompile(`[ \t\f\x{00a0}]+`)
	blankRun = regexp.MustCompile(`\n{3,}`)
)

var textBlocks = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Tr: true, atom.Table: true, atom.Li: true, atom.Ul: true, atom.Ol: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true, atom.Blockquote: true,
	atom.Section: true, atom.Article: true, atom.Header: true, atom.Footer: true, atom.Pre: true, atom.Hr: true,
}

// HTMLText is a message's HTML as text: blocks on their own lines, links as
// "text (url)", scripts, styles and comments dropped.
func HTMLText(src string) string {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return ""
	}
	var b strings.Builder
	newline := func() {
		if s := b.String(); s != "" && !strings.HasSuffix(s, "\n") {
			b.WriteByte('\n')
		}
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			t := spaceRun.ReplaceAllString(strings.ReplaceAll(n.Data, "\n", " "), " ")
			if strings.HasSuffix(b.String(), "\n") || b.Len() == 0 {
				t = strings.TrimLeft(t, " ")
			}
			b.WriteString(t)
			return
		case html.CommentNode, html.DoctypeNode:
			return
		case html.ElementNode:
			switch n.DataAtom {
			case atom.Script, atom.Style, atom.Head, atom.Title:
				return
			case atom.Br:
				b.WriteByte('\n')
				return
			case atom.A:
				href := ""
				for _, a := range n.Attr {
					if a.Key == "href" {
						href = strings.TrimSpace(a.Val)
					}
				}
				var inner strings.Builder
				var collect func(*html.Node)
				collect = func(c *html.Node) {
					if c.Type == html.TextNode {
						inner.WriteString(c.Data)
					}
					for x := c.FirstChild; x != nil; x = x.NextSibling {
						collect(x)
					}
				}
				collect(n)
				label := strings.TrimSpace(spaceRun.ReplaceAllString(strings.ReplaceAll(inner.String(), "\n", " "), " "))
				switch {
				case href == "" || strings.HasPrefix(strings.ToLower(href), "javascript:"):
					b.WriteString(label)
				case label == "" || label == href || "mailto:"+label == href:
					b.WriteString(href)
				default:
					b.WriteString(label + " (" + href + ")")
				}
				return
			}
			if textBlocks[n.DataAtom] {
				newline()
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && textBlocks[n.DataAtom] {
			newline()
		}
	}
	walk(doc)
	lines := strings.Split(b.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.TrimSpace(blankRun.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}
