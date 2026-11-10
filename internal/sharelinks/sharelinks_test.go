package sharelinks

import (
	"net/url"
	"strings"
	"testing"
)

func TestURL(t *testing.T) {
	const link, text = "https://news.acme.io/posts/the-case-for-boring-infrastructure", "The case for boring infrastructure"
	esc := url.QueryEscape(link)
	for _, c := range []struct{ channel, want string }{
		{Email, "mailto:?subject=Acme%20newsletter&body=The%20case%20for%20boring%20infrastructure%20" + esc},
		{X, "https://x.com/intent/post?text=The+case+for+boring+infrastructure&url=" + esc},
		{LinkedIn, "https://www.linkedin.com/sharing/share-offsite/?url=" + esc},
		{Facebook, "https://www.facebook.com/sharer/sharer.php?u=" + esc},
		{Bluesky, "https://bsky.app/intent/compose?text=The+case+for+boring+infrastructure+" + esc},
		{WhatsApp, "https://wa.me/?text=The+case+for+boring+infrastructure+" + esc},
		{"myspace", ""},
	} {
		if got := URL(c.channel, link, text, "Acme newsletter"); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.channel, got, c.want)
		}
	}
	// A message that already holds the link (a referral's).
	if got := URL(X, "", "Join me "+link, ""); got != "https://x.com/intent/post?text="+url.QueryEscape("Join me "+link) {
		t.Errorf("x without a link: %s", got)
	}
	if got := URL(WhatsApp, link, "", ""); got != "https://wa.me/?text="+esc {
		t.Errorf("whatsapp without text: %s", got)
	}
	if len(Channels) != 6 {
		t.Errorf("channels %v", Channels)
	}
}

// A title can't add a header to the reader's email (RFC 6068): &, =, +,
// ? and # stay inside the subject and the body.
func TestEmailEscapesHeaders(t *testing.T) {
	const title = "x&bcc=a@evil.io?cc=b@evil.io#1 + 2"
	got := URL(Email, "https://news.acme.io/posts/x?ref=1&via=email", title, title)
	if strings.Contains(got, "&bcc") || strings.Contains(got, "?cc") || strings.Contains(got, "+") {
		t.Fatalf("unescaped: %s", got)
	}
	u, err := url.Parse(got)
	if err != nil || u.Scheme != "mailto" {
		t.Fatalf("parse %s: %v", got, err)
	}
	q := u.Query()
	if len(q) != 2 || q.Get("subject") != title || q.Get("body") != title+" https://news.acme.io/posts/x?ref=1&via=email" {
		t.Fatalf("headers %v", q)
	}
}
