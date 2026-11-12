package engine

import (
	"context"
	"net/mail"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/inbound"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// InboundBox is how a workspace receives mail (M58), as a send captures
// it: the inbound route's domain and a mailbox route's local part, whether
// the route is receiving, and whether replies stay in ReputeCast's inbox
// (Settings › Senders › Reply handling).
type InboundBox struct {
	Domain    string `json:"domain"`
	Local     string `json:"local,omitempty"`
	Receiving bool   `json:"receiving,omitempty"`
	Inbox     bool   `json:"inbox,omitempty"`
}

func (b *InboundBox) box() inbound.Box { return inbound.Box{Domain: b.Domain, Local: b.Local} }

// ReplyTo is where replies to a message go while replies stay in the
// inbox: the route's reply address for the message, with name (the
// sender's reply-to name, else the sender's). nil keeps the sender's own
// reply-to.
func (b *InboundBox) ReplyTo(s *linktoken.Signer, workspaceID, messageID uuid.UUID, name string) *mail.Address {
	if b == nil || !b.Inbox || b.Domain == "" || s == nil {
		return nil
	}
	return &mail.Address{Name: name, Address: b.box().Address(s, inbound.LetterReply, workspaceID, messageID)}
}

// unsubscribe is the mailto: unsubscribe address of a message while the
// route receives ("" otherwise): ReputeCast applies what arrives there.
func (b *InboundBox) unsubscribe(s *linktoken.Signer, workspaceID, messageID uuid.UUID) string {
	if b == nil || !b.Receiving || b.Domain == "" || s == nil {
		return ""
	}
	return b.box().Address(s, inbound.LetterUnsubscribe, workspaceID, messageID)
}

// inboundBox loads how the workspace receives mail now; nil when it
// doesn't (or Engine.Inbound isn't set).
func (e *Engine) inboundBox(ctx context.Context, workspaceID uuid.UUID) *InboundBox {
	if e.Inbound == nil {
		return nil
	}
	b, err := e.Inbound(ctx, workspaceID)
	if err != nil {
		e.Log.WarnContext(ctx, "inbound route", "workspace", workspaceID, "err", err)
		return nil
	}
	return b
}

// ListUnsubscribeMailto is the List-Unsubscribe mailto: of a message and its subject:
// the route's unsubscribe address while it receives, else the reply-to or
// From with the signed token in the subject; "" when the workspace leaves
// mailto: out.
func ListUnsubscribeMailto(on bool, b *InboundBox, s *linktoken.Signer, workspaceID, messageID uuid.UUID, replyTo, from, token string) (string, string) {
	if !on {
		return "", ""
	}
	if a := b.unsubscribe(s, workspaceID, messageID); a != "" {
		return a, "unsubscribe"
	}
	return unsubscribeMailto(on, replyTo, from), "unsubscribe " + token
}

// ReplyName is the name a rewritten Reply-To carries: the reply-to's own
// name, else the sender's.
func ReplyName(replyTo, fromName string) string {
	if replyTo != "" {
		if a, err := mail.ParseAddress(replyTo); err == nil && a.Name != "" {
			return a.Name
		}
	}
	return fromName
}
