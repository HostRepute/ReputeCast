package tracking

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Only the hidden link's own token counts as a machine's click: a store
// link's click token, whether signed now or with the hidden link's kind
// as click tokens used to be, doesn't.
func TestHoneypotClaims(t *testing.T) {
	s, err := linktoken.NewSigner(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	ws, msg, contact := uuid.New(), uuid.New(), uuid.New()
	hidden := s.Sign(linktoken.Claims{Kind: linktoken.Honeypot, WorkspaceID: ws, MessageID: msg, ContactID: contact})
	if c, ok := honeypotClaims(s, hidden); !ok || c.MessageID != msg {
		t.Fatalf("the hidden link: %+v %v", c, ok)
	}
	click := clicktoken.Sign(s, clicktoken.Click{WorkspaceID: ws, MessageID: msg, ContactID: contact, At: time.Now()})
	before := s.Sign(linktoken.Claims{Kind: linktoken.Honeypot, WorkspaceID: ws, MessageID: msg, ContactID: contact, Link: 3_617_000})
	if _, err := clicktoken.Verify(s, before); err != nil {
		t.Fatalf("the click token from before is a click token: %v", err)
	}
	for name, tok := range map[string]string{"a click token": click, "a click token from before": before, "no message": s.Sign(linktoken.Claims{Kind: linktoken.Honeypot, WorkspaceID: ws})} {
		if _, ok := honeypotClaims(s, tok); ok {
			t.Errorf("%s counted as the hidden link", name)
		}
	}
}
