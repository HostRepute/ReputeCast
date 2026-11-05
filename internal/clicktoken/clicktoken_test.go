package clicktoken

import (
	"bytes"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

func TestRoundTrip(t *testing.T) {
	s, err := linktoken.NewSigner(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	c := Click{WorkspaceID: uuid.New(), MessageID: uuid.New(), ContactID: uuid.New(), At: time.Date(2026, 10, 9, 14, 3, 41, 0, time.UTC)}
	tok := Sign(s, c)
	got, err := Verify(s, tok)
	if err != nil {
		t.Fatal(err)
	}
	if got.MessageID != c.MessageID || got.ContactID != c.ContactID || got.WorkspaceID != c.WorkspaceID || !got.At.Equal(c.At.Truncate(time.Minute)) {
		t.Fatalf("%+v", got)
	}
	if _, err := Verify(s, tok[:len(tok)-2]+"AA"); err == nil {
		t.Fatal("a forged token verified")
	}
	other, _ := linktoken.NewSigner(bytes.Repeat([]byte{8}, 32))
	if _, err := Verify(other, tok); err == nil {
		t.Fatal("another instance's token verified")
	}
	// A click link's token never works as a click token.
	click := s.Sign(linktoken.Claims{Kind: linktoken.Click, WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: c.ContactID})
	if _, err := Verify(s, click); err == nil {
		t.Fatal("a click link's token verified as a click token")
	}
}

func TestTag(t *testing.T) {
	got := Tag("https://shop.acme.io/products/hoodie?utm_source=mine", UTM{Source: "acme-newsletter", Medium: "email", Campaign: "october-product-update"}, "tok")
	u, _ := url.Parse(got)
	q := u.Query()
	if q.Get("utm_source") != "mine" || q.Get("utm_medium") != "email" || q.Get("utm_campaign") != "october-product-update" || q.Get(Param) != "tok" {
		t.Fatalf("%s", got)
	}
	if Tag("mailto:a@b.c", UTM{Source: "x"}, "t") != "mailto:a@b.c" {
		t.Fatal("non-web links are left alone")
	}
	if Slug("Weekly Digest #142") != "weekly-digest-142" || Slug("  October Product Update ") != "october-product-update" || Slug("Abandoned cart") != "abandoned-cart" {
		t.Fatal("Slug")
	}
}

// A click token isn't a hidden link's token, nor the other way round:
// they used to share a kind, so a store link's rc= worked at /t/h/.
// Tokens of that kind already in store links still credit orders.
func TestNotTheHiddenLink(t *testing.T) {
	s, err := linktoken.NewSigner(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	c := Click{WorkspaceID: uuid.New(), MessageID: uuid.New(), ContactID: uuid.New(), At: time.Date(2026, 11, 3, 9, 41, 0, 0, time.UTC)}
	if Kind == linktoken.Honeypot {
		t.Fatal("the click token has the hidden link's kind")
	}
	if _, err := s.Verify(Sign(s, c), linktoken.Honeypot); err == nil {
		t.Fatal("a click token verified as the hidden link's")
	}
	legacy := s.Sign(linktoken.Claims{Kind: linktoken.Honeypot, WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: c.ContactID,
		Link: uint32(c.At.Sub(epoch) / time.Minute)})
	got, err := Verify(s, legacy)
	if err != nil || got.MessageID != c.MessageID || !got.At.Equal(c.At) {
		t.Fatalf("a click token from before: %+v %v", got, err)
	}
	hidden := s.Sign(linktoken.Claims{Kind: linktoken.Honeypot, WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: c.ContactID})
	if _, err := Verify(s, hidden); err == nil {
		t.Fatal("a hidden link's token verified as a click token")
	}
}
