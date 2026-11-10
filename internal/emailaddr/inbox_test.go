package emailaddr

import "testing"

func TestInboxLink(t *testing.T) {
	const sender = "digest@news.acme.io"
	cases := []struct {
		email, sender, name, link string
	}{
		{"sam@gmail.com", sender, "Gmail", "https://mail.google.com/mail/u/sam%40gmail.com/#search/from%3A(digest%40news.acme.io)+in%3Aanywhere+newer_than%3A1h"},
		{"sam+news@googlemail.com", sender, "Gmail", "https://mail.google.com/mail/u/sam%2Bnews%40googlemail.com/#search/from%3A(digest%40news.acme.io)+in%3Aanywhere+newer_than%3A1h"},
		{"sam@gmail.com", "", "Gmail", "https://mail.google.com/mail/u/sam%40gmail.com/#search/in%3Aanywhere+newer_than%3A1h"},
		{"r.walker@hotmail.co.uk", sender, "Outlook", "https://outlook.live.com/mail/?login_hint=r.walker%40hotmail.co.uk"},
		{"ana@yahoo.com", sender, "Yahoo Mail", "https://mail.yahoo.com/d/search/keyword=from%3Adigest%40news.acme.io"},
		{"ana@yahoo.com", "", "Yahoo Mail", "https://mail.yahoo.com/"},
		{"lee@icloud.com", sender, "iCloud Mail", "https://www.icloud.com/mail"},
		{"kai@proton.me", sender, "Proton Mail", "https://mail.proton.me/u/0/all-mail#from=digest%40news.acme.io"},
		{"kai@pm.me", "", "Proton Mail", "https://mail.proton.me/u/0/inbox"},
	}
	for _, c := range cases {
		name, link, ok := InboxLink(c.email, c.sender)
		if !ok || name != c.name || link != c.link {
			t.Errorf("InboxLink(%q, %q) = %q, %q, %v; want %q, %q", c.email, c.sender, name, link, ok, c.name, c.link)
		}
	}
	for _, email := range []string{"sam@acme.io", "jo@aol.com", "jo@aim.com", "not-an-address", ""} {
		if name, link, ok := InboxLink(email, sender); ok {
			t.Errorf("InboxLink(%q) = %q, %q; want none", email, name, link)
		}
	}
}
