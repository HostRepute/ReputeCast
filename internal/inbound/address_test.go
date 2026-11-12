package inbound

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

func signer(t *testing.T, key string, previous ...string) *linktoken.Signer {
	t.Helper()
	prev := make([][]byte, 0, len(previous))
	for _, p := range previous {
		prev = append(prev, []byte(strings.Repeat(p, 32)))
	}
	s, err := linktoken.NewSigner([]byte(strings.Repeat(key, 32)), prev...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// A tag names its letter and message and verifies only for its workspace,
// in any case, under the current key or a previous one; a changed byte
// doesn't verify.
func TestEncodeDecode(t *testing.T) {
	s := signer(t, "a")
	ws, other, msg := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	for _, letter := range []byte{LetterReply, LetterUnsubscribe, LetterTest} {
		tag := Encode(s, letter, ws, msg)
		if len(tag) != 41 || tag[0] != letter || tag[1] != '-' || strings.ToLower(tag) != tag {
			t.Fatalf("tag %q", tag)
		}
		got, id, ok := Decode(s, ws, tag)
		if !ok || got != letter || id != msg {
			t.Fatalf("round trip %q: %c %v %v", tag, got, id, ok)
		}
		if _, id, ok := Decode(s, ws, strings.ToUpper(tag)); !ok || id != msg {
			t.Fatalf("upper case %q", tag)
		}
		if _, _, ok := Decode(s, other, tag); ok {
			t.Fatalf("another workspace's tag verified: %q", tag)
		}
	}
	tag := Encode(s, LetterReply, ws, msg)
	// Another letter with the same MAC, a changed character, a short tag.
	for _, bad := range []string{"u" + tag[1:], tag[:len(tag)-1] + flip(tag[len(tag)-1]), tag[:30], "x-" + tag[2:], "", "r-"} {
		if _, _, ok := Decode(s, ws, bad); ok {
			t.Fatalf("forged tag verified: %q", bad)
		}
	}
	// After a key rotation, addresses handed out before still verify.
	rotated := signer(t, "b", "a")
	if _, id, ok := Decode(rotated, ws, tag); !ok || id != msg {
		t.Fatal("a tag made under the previous key doesn't verify")
	}
	if _, _, ok := Decode(signer(t, "c"), ws, tag); ok {
		t.Fatal("a tag verified under a key that never made it")
	}
}

func flip(c byte) string {
	if c == 'a' {
		return "b"
	}
	return "a"
}

// Box addresses: a provider route's tag is the local part, a mailbox
// route's goes after a plus; tagOf finds them again.
func TestBoxAddresses(t *testing.T) {
	s := signer(t, "a")
	ws, msg := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	p := Box{Domain: "reply.news.acme.io"}
	addr := p.Address(s, LetterReply, ws, msg)
	if !strings.HasPrefix(addr, "r-") || !strings.HasSuffix(addr, "@reply.news.acme.io") || len(addr) > 64+1+len(p.Domain) {
		t.Fatalf("provider address %q", addr)
	}
	if tag, ok := p.tagOf(strings.ToUpper(addr)); !ok || tag != strings.Split(addr, "@")[0] {
		t.Fatalf("tagOf %q: %q %v", addr, tag, ok)
	}
	if p.Fixed(LocalDMARC) != "dmarc@reply.news.acme.io" {
		t.Fatal(p.Fixed(LocalDMARC))
	}
	if _, ok := p.tagOf("r-x@elsewhere.io"); ok {
		t.Fatal("an address on another domain is the box's")
	}
	m := Box{Domain: "acme.io", Local: "replies"}
	addr = m.Address(s, LetterUnsubscribe, ws, msg)
	local, _, _ := strings.Cut(addr, "@")
	if !strings.HasPrefix(addr, "replies+u-") || len(local) > 64 {
		t.Fatalf("mailbox address %q", addr)
	}
	if tag, ok := m.tagOf(addr); !ok || !strings.HasPrefix(tag, "u-") {
		t.Fatalf("mailbox tagOf: %q %v", tag, ok)
	}
	if tag, ok := m.tagOf("replies@acme.io"); !ok || tag != "" {
		t.Fatalf("the mailbox's own address: %q %v", tag, ok)
	}
	if _, ok := m.tagOf("other+u-x@acme.io"); ok {
		t.Fatal("another mailbox on the domain is the box's")
	}
	// The longest local part a mailbox may have keeps tagged addresses in RFC 5321's 64.
	long := Box{Domain: "acme.io", Local: strings.Repeat("x", MaxMailboxLocal)}
	if l, _, _ := strings.Cut(long.Address(s, LetterReply, ws, msg), "@"); len(l) > 64 {
		t.Fatalf("local part of %d characters", len(l))
	}
}
