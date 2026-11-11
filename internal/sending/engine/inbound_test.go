package engine

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/inbound"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// While the route receives, List-Unsubscribe's mailto: is the message's
// u- address with subject "unsubscribe"; replies kept in the inbox go to
// its r- address under the sender's name. Without a route, today's
// reply-to and token subject stay.
func TestInboundAddresses(t *testing.T) {
	s, err := linktoken.NewSigner([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	ws, msg := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	box := &InboundBox{Domain: "reply.news.acme.io", Receiving: true, Inbox: true}

	to, subject := ListUnsubscribeMailto(true, box, s, ws, msg, "hello@acme.io", "news@acme.io", "TOKEN")
	if !strings.HasPrefix(to, "u-") || !strings.HasSuffix(to, "@reply.news.acme.io") || subject != "unsubscribe" {
		t.Fatalf("receiving: %q %q", to, subject)
	}
	local, _, _ := strings.Cut(to, "@")
	if letter, id, ok := inbound.Decode(s, ws, local); !ok || letter != inbound.LetterUnsubscribe || id != msg {
		t.Fatalf("decode %q: %c %v %v", local, letter, id, ok)
	}
	h := bulkHeaders("https://x/u/T", to, subject, "f")
	if !strings.Contains(h["List-Unsubscribe"], "<mailto:"+to+"?subject=unsubscribe>") {
		t.Fatalf("List-Unsubscribe %q", h["List-Unsubscribe"])
	}

	a := box.ReplyTo(s, ws, msg, ReplyName("Acme Team <hello@acme.io>", "Acme"))
	if a == nil || a.Name != "Acme Team" || !strings.HasPrefix(a.Address, "r-") || !strings.HasSuffix(a.Address, "@reply.news.acme.io") {
		t.Fatalf("reply-to %+v", a)
	}
	if got := ReplyName("", "Acme"); got != "Acme" {
		t.Fatalf("reply name %q", got)
	}

	// A mailbox route tags after a plus.
	mb := &InboundBox{Domain: "acme.io", Local: "replies", Receiving: true}
	if to, _ := ListUnsubscribeMailto(true, mb, s, ws, msg, "", "news@acme.io", "T"); !strings.HasPrefix(to, "replies+u-") || !strings.HasSuffix(to, "@acme.io") {
		t.Fatalf("mailbox mailto %q", to)
	}
	if mb.ReplyTo(s, ws, msg, "Acme") != nil {
		t.Fatal("replies go to the inbox only when reply handling is inbox")
	}

	// Not receiving: the reply-to and the token subject, as before.
	off := &InboundBox{Domain: "reply.news.acme.io", Inbox: true}
	if to, subject := ListUnsubscribeMailto(true, off, s, ws, msg, "hello@acme.io", "news@acme.io", "TOKEN"); to != "hello@acme.io" || subject != "unsubscribe TOKEN" {
		t.Fatalf("not receiving: %q %q", to, subject)
	}
	if to, _ := ListUnsubscribeMailto(false, box, s, ws, msg, "hello@acme.io", "news@acme.io", "T"); to != "" {
		t.Fatalf("mailto off: %q", to)
	}
	var none *InboundBox
	if none.ReplyTo(s, ws, msg, "x") != nil || none.unsubscribe(s, ws, msg) != "" {
		t.Fatal("no route, no rewrite")
	}
}
