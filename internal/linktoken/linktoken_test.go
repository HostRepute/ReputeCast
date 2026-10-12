package linktoken

import (
	"testing"

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
