package inbound

import (
	"net/textproto"
	"strings"
	"testing"
)

const sampleReply = "From: Sam Rivera <Sam.Rivera@outlook.com>\r\n" +
	"To: r-abc@reply.news.acme.io\r\n" +
	"Subject: =?utf-8?q?Re:_October_Product_Update?=\r\n" +
	"Message-ID: <CAF123@mail.outlook.com>\r\n" +
	"In-Reply-To: <0199@news.acme.io>\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/mixed; boundary=b1\r\n\r\n" +
	"--b1\r\nContent-Type: multipart/alternative; boundary=b2\r\n\r\n" +
	"--b2\r\nContent-Type: text/html; charset=utf-8\r\n\r\n" +
	"<p>Loved the <b>new</b> routing rules.</p><p>See <a href=\"https://acme.io/api\">the API</a></p><script>x()</script>\r\n" +
	"--b2--\r\n" +
	"--b1\r\nContent-Type: image/png\r\nContent-Disposition: attachment; filename=\"screenshot.png\"\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
	"iVBORw0KGgo=\r\n" +
	"--b1--\r\n"

// A reply's headers, its HTML as text (links kept as "text (url)") and
// its attachments' names, types and sizes.
func TestParse(t *testing.T) {
	p, err := Parse([]byte(sampleReply))
	if err != nil {
		t.Fatal(err)
	}
	if p.From != "sam.rivera@outlook.com" || p.FromName != "Sam Rivera" || p.Subject != "Re: October Product Update" ||
		p.MessageID != "CAF123@mail.outlook.com" || p.InReplyTo != "0199@news.acme.io" || len(p.To) != 1 {
		t.Fatalf("headers: %+v", p)
	}
	if !p.HasHTML || p.Text != "Loved the new routing rules.\nSee the API (https://acme.io/api)" {
		t.Fatalf("text %q", p.Text)
	}
	if len(p.Attachments) != 1 || p.Attachments[0].Name != "screenshot.png" || p.Attachments[0].ContentType != "image/png" || p.Attachments[0].Size != 8 {
		t.Fatalf("attachments %+v", p.Attachments)
	}
	long := strings.Repeat("é", MaxText)
	if c := clipText(long); len(c) > MaxText || !strings.HasPrefix(long, c) {
		t.Fatalf("clipped to %d bytes", len(c))
	}
}

func TestClassify(t *testing.T) {
	for _, c := range []struct {
		subject, text string
		want          bool
	}{
		{"Re: Weekly Digest #141", "Please take me off this list, I don't use Acme anymore.", true},
		{"Re: October Product Update", "Thanks, all set now.", false},
		{"Re: Weekly Digest #141", "Unsubscribe", true},
		{"unsubscribe", "", true},
		{"AW: Newsletter", "> quoted\nBitte abmelden.", true},
		{"Re: hi", "Stop by our booth next week", false},
		{"Re: hi", "STOP", true},
		{"Re: hi", "Hủy đăng ký", true},
		{"Re: hi", "Me désinscrire svp", true},
	} {
		if got := asksUnsubscribe(c.subject, c.text); got != c.want {
			t.Errorf("asksUnsubscribe(%q, %q) = %v", c.subject, c.text, got)
		}
	}
	h := func(kv ...string) textproto.MIMEHeader {
		m := textproto.MIMEHeader{}
		for i := 0; i < len(kv); i += 2 {
			m.Add(kv[i], kv[i+1])
		}
		return m
	}
	for _, c := range []struct {
		h    textproto.MIMEHeader
		want bool
	}{
		{h("Auto-Submitted", "auto-replied"), true},
		{h("Auto-Submitted", "no"), false},
		{h("X-Autoreply", "yes"), true},
		{h("Precedence", "bulk"), true},
		{h("Return-Path", "<>"), true},
		{h("Return-Path", "<sam@outlook.com>"), false},
		{h(), false},
	} {
		if got := isAutoReply(c.h); got != c.want {
			t.Errorf("isAutoReply(%v) = %v", c.h, got)
		}
	}
	if !spamVerdict("spam", nil) || spamVerdict("pass", h("X-Spam-Flag", "YES")) || !spamVerdict("", h("X-Spam-Status", "Yes, score=7")) {
		t.Fatal("spam verdicts")
	}
	if got := preview("Thanks!\n\n> On Tue, Acme wrote:\n> quoted"); got != "Thanks!" {
		t.Fatalf("preview %q", got)
	}
}
