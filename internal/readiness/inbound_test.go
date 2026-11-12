package readiness_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// M58: while ReputeCast receives the workspace's mail, the mailto: address
// is an unsubscribe address on the receiving domain and the row passes;
// without receiving it warns as before.
func TestUnsubscribeMailtoWhenReceiving(t *testing.T) {
	f := digest()
	f.Compliance.Mailto = "hello@acme.io"
	f.Compliance.Receiving, f.Compliance.InboundDomain = true, "reply.news.acme.io"
	res := readiness.Assemble(f)
	c := mustCheck(t, res, "compliance.unsubscribe_mailto")
	if c.Status != readiness.Pass || c.Title != "Unsubscribe emails are processed by ReputeCast" || c.Ack != "" || len(c.Fixes) != 0 ||
		c.Detail != "Receiving mail is on, so the mailto: address is on reply.news.acme.io and ReputeCast applies those unsubscribes within a minute. No one has to remove readers by hand." {
		t.Fatalf("mailto while receiving: %+v", c)
	}
	if h := mustCheck(t, res, "compliance.list_unsubscribe"); !strings.Contains(strings.Join(h.Raw, "\n"), "<mailto:u-{token}@reply.news.acme.io?subject=unsubscribe>") {
		t.Fatalf("header while receiving: %v", h.Raw)
	}
	if res.Counts.Warnings != 2 {
		t.Fatalf("warnings %v", statuses(res, readiness.Warn))
	}
	noCheck(t, res, "compliance.reply_inbox")

	// A mailbox route: the address is the mailbox's, with the tag after "+".
	f.Compliance.InboundDomain, f.Compliance.InboundLocal = "acme.io", "replies"
	if h := mustCheck(t, readiness.Assemble(f), "compliance.list_unsubscribe"); !strings.Contains(strings.Join(h.Raw, "\n"), "<mailto:replies+u-{token}@acme.io?subject=unsubscribe>") {
		t.Fatalf("header with a mailbox: %v", h.Raw)
	}

	// The setting off: no mailto at all, receiving or not.
	f.Compliance.Mailto = ""
	res = readiness.Assemble(f)
	noCheck(t, res, "compliance.unsubscribe_mailto")
	if h := mustCheck(t, res, "compliance.list_unsubscribe"); strings.Contains(strings.Join(h.Raw, "\n"), "mailto:") {
		t.Fatalf("header without the setting: %v", h.Raw)
	}

	// Not receiving: today's warning and header.
	f = digest()
	f.Compliance.Mailto = "hello@acme.io"
	f.Compliance.InboundDomain = "reply.news.acme.io"
	res = readiness.Assemble(f)
	if c := mustCheck(t, res, "compliance.unsubscribe_mailto"); c.Status != readiness.Warn || c.Ack == "" {
		t.Fatalf("mailto without receiving: %+v", c)
	}
	if h := mustCheck(t, res, "compliance.list_unsubscribe"); !strings.Contains(strings.Join(h.Raw, "\n"), "<mailto:hello@acme.io?subject=unsubscribe%20{token}>") {
		t.Fatalf("header without receiving: %v", h.Raw)
	}
}

// M58: replies kept in ReputeCast's inbox while receiving fails warn, with
// what Receiving mail says and Settings › Senders to fix it; receiving,
// nothing to say.
func TestReplyInboxWhileReceivingFails(t *testing.T) {
	f := digest()
	f.Campaign.ReplyTo = "hello@acme.io"
	f.Compliance.Inbox, f.Compliance.InboundFailing = true, true
	f.Compliance.InboundDetail = readiness.RawText("reply.news.acme.io's MX record no longer points at inbound.postmarkapp.com. Replies aren't reaching ReputeCast.")
	res := readiness.Assemble(f)
	c := mustCheck(t, res, "compliance.reply_inbox")
	want := `Reply handling is "Keep replies in ReputeCast inbox", and Receiving mail says Failing: reply.news.acme.io's MX record no longer points at ` +
		`inbound.postmarkapp.com. Replies aren't reaching ReputeCast. Replies to this send won't reach Audience › Replies until it's fixed. ` +
		`Fix it, or forward replies to ` + f.Campaign.ReplyTo + ` instead.`
	if c.Status != readiness.Warn || c.Group != "compliance" || c.Title != "Replies go to ReputeCast, but receiving is failing" ||
		c.Detail != want || c.Ack != "" ||
		len(c.Fixes) != 1 || c.Fixes[0].Label != "Open Settings" || c.Fixes[0].Href != "/w/acme-newsletter/settings/senders" {
		t.Fatalf("reply inbox: %+v", c)
	}
	if res.Counts.Warnings != 3 {
		t.Fatalf("warnings %v", statuses(res, readiness.Warn))
	}
	// Not received its first message yet: what to try.
	f.Compliance.InboundFailing, f.Compliance.InboundDetail = false, nil
	if c := mustCheck(t, readiness.Assemble(f), "compliance.reply_inbox"); !strings.Contains(c.Detail, "hasn't received its first message yet") || !strings.Contains(c.Detail, f.Campaign.ReplyTo) {
		t.Fatalf("reply inbox, pending: %q", c.Detail)
	}
	f.Compliance.Receiving, f.Compliance.InboundDomain = true, "reply.news.acme.io"
	noCheck(t, readiness.Assemble(f), "compliance.reply_inbox")
}
