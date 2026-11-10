package emailaddr

import (
	"net/url"
	"strings"
)

// InboxLink is the web mail app of a consumer address, to open after a
// signup ("Open Gmail"): the provider's name (never translated) and an
// address that searches for mail from sender where the app can, else opens
// the inbox. ok is false for other domains, company ones included, and for
// AOL addresses (Yahoo filters their mail, but mail.yahoo.com doesn't sign
// them in).
func InboxLink(email, sender string) (name, link string, ok bool) {
	d := Domain(email)
	e, s := url.QueryEscape(email), url.QueryEscape(sender)
	switch d {
	case "protonmail.com", "proton.me", "pm.me":
		if sender == "" {
			return "Proton Mail", "https://mail.proton.me/u/0/inbox", true
		}
		return "Proton Mail", "https://mail.proton.me/u/0/all-mail#from=" + s, true
	}
	switch MailboxProvider(d) {
	case Gmail:
		search := "in%3Aanywhere+newer_than%3A1h"
		if sender != "" {
			search = "from%3A(" + s + ")+" + search
		}
		return "Gmail", "https://mail.google.com/mail/u/" + e + "/#search/" + search, true
	case Outlook:
		return "Outlook", "https://outlook.live.com/mail/?login_hint=" + e, true
	case Yahoo:
		if d == "aim.com" || strings.HasPrefix(d, "aol.") {
			return "", "", false
		}
		if sender == "" {
			return "Yahoo Mail", "https://mail.yahoo.com/", true
		}
		return "Yahoo Mail", "https://mail.yahoo.com/d/search/keyword=from%3A" + s, true
	case ICloud:
		return "iCloud Mail", "https://www.icloud.com/mail", true
	}
	return "", "", false
}
