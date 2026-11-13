// Package inbound receives mail for a workspace (M58): a connected
// provider's receiving or a mailbox ReputeCast polls. Replies land in
// Audience › Replies, mailto: unsubscribes are applied, flows can stop when
// someone replies, DMARC aggregate and feedback-loop reports are collected,
// and developers get email.replied and email.received webhooks.
package inbound

import (
	"encoding/base32"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Address letters: what a tagged address is for.
const (
	LetterReply       byte = 'r'
	LetterUnsubscribe byte = 'u'
	LetterTest        byte = 't'
)

// Fixed local parts on a route's domain.
const (
	LocalDMARC = "dmarc"
	LocalFBL   = "fbl"
)

// MaxMailboxLocal is the longest local part a mailbox route may have: its
// tagged addresses (<local>+<tag>) stay within the 64 characters RFC 5321
// allows a local part.
const MaxMailboxLocal = 22

// tagEncoding is RFC 4648 base32, lower case, without padding: it survives
// mail systems that lower-case addresses.
var tagEncoding = base32.NewEncoding(tagAlphabet).WithPadding(base32.NoPadding)

const tagAlphabet = "abcdefghijklmnopqrstuvwxyz234567"

// macLen is how much of the digest a tag carries: 64 bits, so an address
// for a message can't be guessed from its id.
const macLen = 8

// Encode makes the tag of an address for one message of a workspace:
// "<letter>-" and the base32 of the message id and a keyed digest of the
// letter, the workspace and the message (41 characters).
func Encode(s *linktoken.Signer, letter byte, workspaceID, messageID uuid.UUID) string {
	sum := s.Digest(digestParts(letter, workspaceID, messageID)...)
	b := make([]byte, 0, 16+macLen)
	b = append(b, messageID[:]...)
	b = append(b, sum[:macLen]...)
	return string(letter) + "-" + tagEncoding.EncodeToString(b)
}

// Decode reads a tag made by Encode for the workspace, in any case: the
// letter and the message id when its digest holds under the instance key or
// a previous one.
func Decode(s *linktoken.Signer, workspaceID uuid.UUID, tag string) (letter byte, messageID uuid.UUID, ok bool) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if len(tag) < 3 || tag[1] != '-' {
		return 0, uuid.Nil, false
	}
	letter = tag[0]
	switch letter {
	case LetterReply, LetterUnsubscribe, LetterTest:
	default:
		return 0, uuid.Nil, false
	}
	b, err := tagEncoding.DecodeString(tag[2:])
	// Only the spelling Encode makes: the last character's spare bits are
	// zero, so a tag has one form and a changed last character never
	// decodes to the same bytes.
	if err != nil || len(b) != 16+macLen || tagEncoding.EncodeToString(b) != tag[2:] {
		return 0, uuid.Nil, false
	}
	copy(messageID[:], b[:16])
	if !s.VerifyDigest(b[16:], digestParts(letter, workspaceID, messageID)...) {
		return 0, uuid.Nil, false
	}
	return letter, messageID, true
}

func digestParts(letter byte, workspaceID, messageID uuid.UUID) [][]byte {
	return [][]byte{[]byte("inbound"), {letter}, workspaceID[:], messageID[:]}
}

// Box is where a route receives: its domain and, for a mailbox route, the
// mailbox's local part (tags go after a plus).
type Box struct {
	Domain string
	Local  string
}

// Address is the tagged address of a message on the box.
func (b Box) Address(s *linktoken.Signer, letter byte, workspaceID, messageID uuid.UUID) string {
	return b.local(Encode(s, letter, workspaceID, messageID)) + "@" + b.Domain
}

// Fixed is a fixed address on the box (dmarc@, fbl@).
func (b Box) Fixed(local string) string { return b.local(local) + "@" + b.Domain }

func (b Box) local(part string) string {
	if b.Local == "" {
		return part
	}
	return b.Local + "+" + part
}

// tagOf finds the part of a recipient address that addresses the box: the
// local part of a provider route's address, the part after the plus of a
// mailbox route's. ok is false for an address elsewhere.
func (b Box) tagOf(addr string) (string, bool) {
	addr = strings.ToLower(strings.TrimSpace(addr))
	local, domain, found := strings.Cut(addr, "@")
	if !found || domain != strings.ToLower(b.Domain) {
		return "", false
	}
	if b.Local == "" {
		return local, true
	}
	base, tag, plus := strings.Cut(local, "+")
	if base != strings.ToLower(b.Local) {
		return "", false
	}
	if !plus {
		return "", true
	}
	return tag, true
}
