// Package dsn parses bounce and complaint reports that arrive by email:
// delivery status notifications (RFC 3464, multipart/report;
// report-type=delivery-status) and abuse reports (ARF, RFC 5965,
// report-type=feedback-report). Common non-standard bounces (Exim's
// X-Failed-Recipients, plain-text bounces quoting the original headers) are
// handled on a best-effort basis. Only the standard library is used.
package dsn

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
	"time"
)

// Kinds of report.
const (
	KindDSN       = "dsn"
	KindARF       = "arf"
	KindHeuristic = "heuristic"
)

// Recipient is one per-recipient block of a DSN.
type Recipient struct {
	FinalRecipient    string // address only
	OriginalRecipient string
	Action            string // failed, delayed, delivered, relayed, expanded
	Status            string // enhanced status code, e.g. 5.1.1
	DiagnosticCode    string // "smtp; 550 5.1.1 <a@b>: User unknown"
	RemoteMTA         string
}

// Report is a parsed bounce or complaint.
type Report struct {
	Kind         string
	ReportingMTA string
	ArrivalDate  time.Time
	Recipients   []Recipient
	// Original holds the headers of the message the report is about, when
	// the report includes them (message/rfc822 or text/rfc822-headers).
	Original textproto.MIMEHeader
	// FeedbackType, UserAgent and ARFRecipient come from ARF reports.
	FeedbackType string
	UserAgent    string
	ARFRecipient string
	// Mailbox is the address the report was delivered to (the bounce mailbox).
	Mailbox string
	// texts are a multipart message's text parts, read as a bounce only
	// when no delivery-status or feedback part says what happened.
	texts [][]byte
}

// ErrNotAReport means the message is neither a DSN, an ARF report nor a
// recognisable bounce (an auto-reply, a human answer…).
var ErrNotAReport = errors.New("not a bounce or complaint report")

var (
	addrRe     = regexp.MustCompile(`[A-Za-z0-9.!#$%&'*+/=?^_{|}~-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	enhancedRe = regexp.MustCompile(`\b[245]\.\d{1,3}\.\d{1,3}\b`)
	headerRe   = regexp.MustCompile(`(?mi)^(X-ReputeCast-Message|Message-ID):[ \t]*(.+)$`)
)

// Parse reads a raw RFC 5322 message.
func Parse(raw []byte) (*Report, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	rep := &Report{Original: textproto.MIMEHeader{}}
	if to, err := mail.ParseAddress(msg.Header.Get("To")); err == nil {
		rep.Mailbox = strings.ToLower(to.Address)
	}
	mediaType, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if strings.HasPrefix(mediaType, "multipart/") {
		if err := rep.walk(msg.Body, params["boundary"], 0); err != nil {
			return nil, err
		}
		// A report's human-readable part restates its delivery-status part
		// (a success notice, a bounce): only a message without one is read
		// as a bounce by its text.
		for _, t := range rep.texts {
			if rep.Kind != "" {
				break
			}
			rep.heuristic(t)
		}
		rep.texts = nil
	} else {
		body, _ := io.ReadAll(io.LimitReader(msg.Body, 4<<20))
		rep.heuristic(body)
	}
	if rep.Kind == "" {
		if xf := msg.Header.Get("X-Failed-Recipients"); xf != "" {
			rep.Kind = KindHeuristic
			for _, a := range strings.Split(xf, ",") {
				if addr := strings.TrimSpace(a); addr != "" {
					rep.Recipients = append(rep.Recipients, Recipient{FinalRecipient: strings.ToLower(addr), Action: "failed"})
				}
			}
		}
	}
	if rep.Kind == "" || rep.Kind != KindARF && len(rep.Recipients) == 0 {
		return nil, ErrNotAReport
	}
	return rep, nil
}

func (r *Report) walk(body io.Reader, boundary string, depth int) error {
	if boundary == "" || depth > 3 {
		return nil
	}
	mr := multipart.NewReader(body, boundary)
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		ct, params, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		data, _ := io.ReadAll(io.LimitReader(part, 4<<20))
		switch ct {
		case "message/delivery-status", "message/global-delivery-status":
			r.Kind = KindDSN
			r.parseDeliveryStatus(data)
		case "message/feedback-report":
			r.Kind = KindARF
			r.parseFeedback(data)
		case "message/rfc822", "text/rfc822-headers", "message/global", "message/global-headers":
			if h, err := textproto.NewReader(bufio.NewReader(bytes.NewReader(data))).ReadMIMEHeader(); err == nil || len(h) > 0 {
				for k, v := range h {
					r.Original[k] = v
				}
			}
		default:
			if strings.HasPrefix(ct, "multipart/") {
				if err := r.walk(bytes.NewReader(data), params["boundary"], depth+1); err != nil {
					return err
				}
			} else if strings.HasPrefix(ct, "text/") {
				r.texts = append(r.texts, data)
			}
		}
	}
}

// parseDeliveryStatus reads the per-message block and the per-recipient
// blocks of a message/delivery-status part.
func (r *Report) parseDeliveryStatus(data []byte) {
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(normalizeBlocks(data))))
	first := true
	for {
		h, err := tp.ReadMIMEHeader()
		if len(h) > 0 {
			if first {
				r.ReportingMTA = typed(h.Get("Reporting-MTA"))
				if t, err := mail.ParseDate(h.Get("Arrival-Date")); err == nil {
					r.ArrivalDate = t
				}
				first = false
				if h.Get("Final-Recipient") == "" && h.Get("Original-Recipient") == "" {
					continue
				}
			}
			rcpt := Recipient{
				FinalRecipient:    strings.ToLower(typed(h.Get("Final-Recipient"))),
				OriginalRecipient: strings.ToLower(typed(h.Get("Original-Recipient"))),
				Action:            strings.ToLower(strings.TrimSpace(h.Get("Action"))),
				Status:            strings.TrimSpace(firstField(h.Get("Status"))),
				DiagnosticCode:    strings.TrimSpace(h.Get("Diagnostic-Code")),
				RemoteMTA:         typed(h.Get("Remote-MTA")),
			}
			if rcpt.FinalRecipient == "" {
				rcpt.FinalRecipient = rcpt.OriginalRecipient
			}
			if rcpt.FinalRecipient != "" {
				r.Recipients = append(r.Recipients, rcpt)
			}
		}
		if err != nil {
			return
		}
	}
}

// normalizeBlocks makes sure blank lines between header blocks are bare
// CRLFs, which textproto needs to end each block.
func normalizeBlocks(data []byte) []byte {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var b strings.Builder
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			b.WriteString("\r\n")
			continue
		}
		b.WriteString(l)
		b.WriteString("\r\n")
	}
	b.WriteString("\r\n")
	return []byte(b.String())
}

func (r *Report) parseFeedback(data []byte) {
	h, _ := textproto.NewReader(bufio.NewReader(bytes.NewReader(normalizeBlocks(data)))).ReadMIMEHeader()
	r.FeedbackType = strings.ToLower(strings.TrimSpace(h.Get("Feedback-Type")))
	r.UserAgent = strings.TrimSpace(h.Get("User-Agent"))
	r.ARFRecipient = strings.ToLower(strings.Trim(strings.TrimSpace(h.Get("Original-Rcpt-To")), "<>"))
	if t, err := mail.ParseDate(h.Get("Arrival-Date")); err == nil {
		r.ArrivalDate = t
	}
}

// heuristic handles bounces that aren't multipart/report: it looks for an
// enhanced status code, a failed address and quoted original headers.
func (r *Report) heuristic(body []byte) {
	text := string(body)
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "delivery") && !strings.Contains(lower, "undeliver") && !strings.Contains(lower, "failed") && !strings.Contains(lower, "returned") {
		return
	}
	status := enhancedRe.FindString(text)
	if status == "" || status[0] == '2' {
		// No status, or a success: not a bounce.
		return
	}
	for _, m := range headerRe.FindAllStringSubmatch(text, -1) {
		r.Original.Set(m[1], strings.TrimSpace(m[2]))
	}
	addr := ""
	for _, a := range addrRe.FindAllString(text, 20) {
		a = strings.ToLower(a)
		if strings.HasPrefix(a, "mailer-daemon@") || strings.HasPrefix(a, "postmaster@") || a == r.Mailbox {
			continue
		}
		addr = a
		break
	}
	if addr == "" {
		return
	}
	r.Kind = KindHeuristic
	diag := ""
	if i := strings.Index(text, status); i >= 0 {
		line := text[max(0, strings.LastIndex(text[:i], "\n")+1):]
		if j := strings.IndexByte(line, '\n'); j >= 0 {
			line = line[:j]
		}
		diag = strings.TrimSpace(line)
	}
	action := "failed"
	if status[0] == '4' && (strings.Contains(lower, "delayed") || strings.Contains(lower, "will retry") || strings.Contains(lower, "still trying") ||
		strings.Contains(lower, "not yet been delivered") || strings.Contains(lower, "attempts will continue")) {
		action = "delayed"
	}
	r.Recipients = append(r.Recipients, Recipient{FinalRecipient: addr, Action: action, Status: status, DiagnosticCode: diag})
}

// typed strips the address-type prefix of DSN fields ("rfc822; a@b").
func typed(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.IndexByte(v, ';'); i >= 0 {
		v = v[i+1:]
	}
	return strings.Trim(strings.TrimSpace(v), "<>")
}

func firstField(v string) string {
	if f := strings.Fields(v); len(f) > 0 {
		return f[0]
	}
	return ""
}
