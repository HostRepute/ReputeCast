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

// AddUTM adds only the absent tags, after the query as written and before
// the fragment; personalized and non-web links stay as they are.
func TestAddUTM(t *testing.T) {
	u := UTM{Source: "acme-newsletter", Medium: "email", Campaign: "weekly-digest-142"}
	const tags = "utm_source=acme-newsletter&utm_medium=email&utm_campaign=weekly-digest-142"
	for _, c := range []struct{ in, want string }{
		{"https://acme.io/blog", "https://acme.io/blog?" + tags},
		{"https://acme.io", "https://acme.io?" + tags},
		{"HTTPS://ACME.IO/Blog", "HTTPS://ACME.IO/Blog?" + tags},
		// The query as written: not re-encoded, not reordered.
		{"https://acme.io/p?b=2&a=1", "https://acme.io/p?b=2&a=1&" + tags},
		{"https://acme.io/s?q=a+b&x=%7e&flag", "https://acme.io/s?q=a+b&x=%7e&flag&" + tags},
		{"https://acme.io/?", "https://acme.io/?" + tags},
		{"https://acme.io/?a=1&", "https://acme.io/?a=1&" + tags},
		// The author's tags win; the others are added.
		{"https://acme.io/?utm_source=x", "https://acme.io/?utm_source=x&utm_medium=email&utm_campaign=weekly-digest-142"},
		{"https://acme.io/?utm_campaign=launch&utm_source=x&utm_medium=social", "https://acme.io/?utm_campaign=launch&utm_source=x&utm_medium=social"},
		// The fragment stays last, even one that looks like a query.
		{"https://docs.acme.io/flags#install", "https://docs.acme.io/flags?" + tags + "#install"},
		{"https://docs.acme.io/flags?v=2#install", "https://docs.acme.io/flags?v=2&" + tags + "#install"},
		{"https://app.acme.io/#/billing?tab=plan", "https://app.acme.io/?" + tags + "#/billing?tab=plan"},
		// Personalized and non-web links.
		{"{{ upgrade_url }}", "{{ upgrade_url }}"},
		{"https://acme.io/u/{{ contact.id }}", "https://acme.io/u/{{ contact.id }}"},
		{"https://acme.io/{% if vip %}vip{% endif %}", "https://acme.io/{% if vip %}vip{% endif %}"},
		{"mailto:hello@acme.io", "mailto:hello@acme.io"},
		{"#top", "#top"},
	} {
		if got := AddUTM(c.in, u); got != c.want {
			t.Errorf("AddUTM(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := AddUTM("https://acme.io/", UTM{Source: "a b&c", Medium: "email"}); got != "https://acme.io/?utm_source=a+b%26c&utm_medium=email" {
		t.Errorf("values are escaped and empty ones skipped: %q", got)
	}
	if u.Key() != tags {
		t.Errorf("Key = %q", u.Key())
	}
}

// ForEmail names the email: a non-ASCII name keeps its letters without
// their marks, a name with nothing to keep gets the fallback.
func TestForEmail(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"Weekly Digest #142", "weekly-digest-142"},
		{"Bản tin tháng 10 · Đặc biệt", "ban-tin-thang-10-dac-biet"},
		{"Straße & Café", "strasse-cafe"},
		{"新闻 Weekly", "weekly"},
		{"新闻", "campaign"},
		{"", "campaign"},
	} {
		if got := ForEmail("acme-newsletter", c.name, "campaign"); got != (UTM{Source: "acme-newsletter", Medium: "email", Campaign: c.want}) {
			t.Errorf("ForEmail(%q) = %+v, want campaign %q", c.name, got, c.want)
		}
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
