package linktoken

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoundTripAndTamper(t *testing.T) {
	s, err := NewSigner(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	c := Claims{Kind: Click, WorkspaceID: uuid.New(), MessageID: uuid.New(), ContactID: uuid.New(), Link: 7}
	tok := s.Sign(c)
	got, err := s.Verify(tok, Click)
	if err != nil || got != c {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	if _, err := s.Verify(tok, Unsubscribe); err == nil {
		t.Fatal("a click token must not work as an unsubscribe token")
	}
	b := []byte(tok)
	b[10] ^= 1
	if _, err := s.Verify(string(b), Click); err == nil {
		t.Fatal("tampered token accepted")
	}
	other, _ := NewSigner(append(make([]byte, 31), 1))
	if _, err := other.Verify(tok, Click); err == nil {
		t.Fatal("token from another key accepted")
	}
	if len(tok) > 100 {
		t.Fatalf("token too long: %d", len(tok))
	}
}

// A well-formed token signed with another key fails as a signature
// mismatch (an earlier instance key); a malformed one doesn't.
func TestSignatureMismatch(t *testing.T) {
	a, err := NewSigner([]byte(strings.Repeat("a", 32)))
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSigner([]byte(strings.Repeat("b", 32)))
	if err != nil {
		t.Fatal(err)
	}
	c := Claims{Kind: Click, WorkspaceID: uuid.New(), MessageID: uuid.New(), ContactID: uuid.New(), Link: 3}
	if _, err := a.Verify(b.Sign(c), Click); !errors.Is(err, ErrSignature) || !errors.Is(err, ErrInvalid) {
		t.Errorf("another key: %v", err)
	}
	for _, tok := range []string{"", "not-a-token", a.Sign(c) + "x"} {
		if _, err := a.Verify(tok, Click); !errors.Is(err, ErrInvalid) || errors.Is(err, ErrSignature) {
			t.Errorf("malformed %q: %v", tok, err)
		}
	}
	if _, err := a.Verify(a.Sign(c), Open); errors.Is(err, ErrSignature) {
		t.Error("another kind is malformed, not a signature mismatch")
	}
}

// DP-3: a confirmation link works for 30 days after the day it was issued.
func TestExpired(t *testing.T) {
	issued := time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC)
	c := Claims{Kind: Confirm, Link: IssueDay(issued)}
	if Expired(c, ConfirmValid, issued.Add(30*24*time.Hour)) {
		t.Fatal("expired on day 30")
	}
	if !Expired(c, ConfirmValid, issued.Add(32*24*time.Hour)) {
		t.Fatal("still valid on day 32")
	}
}
