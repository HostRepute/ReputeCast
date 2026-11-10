package referrals

import (
	"net/url"
	"testing"
)

// Each share link carries its ?via=, and a list name can't add a header to
// the reader's email.
func TestShareLinks(t *testing.T) {
	const link = "https://news.acme.io/join/K7Q2MX"
	got := map[string]string{}
	for _, sl := range ShareLinks(link, "Read this: {link}", "Digest&bcc=a@evil.io", ShareButtons) {
		got[sl.Channel] = sl.URL
	}
	if len(got) != 4 {
		t.Fatalf("links %v", got)
	}
	if want := "https://x.com/intent/post?text=" + url.QueryEscape("Read this: "+link+"?via=x"); got[ViaX] != want {
		t.Errorf("x %s", got[ViaX])
	}
	if want := "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(link+"?via=linkedin"); got[ViaLinkedIn] != want {
		t.Errorf("linkedin %s", got[ViaLinkedIn])
	}
	if want := "https://wa.me/?text=" + url.QueryEscape("Read this: "+link+"?via=whatsapp"); got[ViaWhatsApp] != want {
		t.Errorf("whatsapp %s", got[ViaWhatsApp])
	}
	u, err := url.Parse(got[ViaEmail])
	if err != nil {
		t.Fatal(err)
	}
	if q := u.Query(); len(q) != 2 || q.Get("subject") != "Digest&bcc=a@evil.io" || q.Get("body") != "Read this: "+link+"?via=email" {
		t.Errorf("email %s: %v", got[ViaEmail], q)
	}
}
