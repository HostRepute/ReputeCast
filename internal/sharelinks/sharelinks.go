// Package sharelinks builds plain share links: an address a reader's
// browser opens, with no third-party scripts and no trackers. Posts on the
// publication website and readers' referral links share through it.
package sharelinks

import (
	"net/url"
	"strings"
)

// Channels a share link opens.
const (
	Email    = "email"
	X        = "x"
	LinkedIn = "linkedin"
	Facebook = "facebook"
	Bluesky  = "bluesky"
	WhatsApp = "whatsapp"
)

// Channels are every channel, in the order a share row shows them.
var Channels = []string{Email, X, LinkedIn, Facebook, Bluesky, WhatsApp}

// URL is channel's share link of link with text (the message) and, for
// email, subject. Channels that take one message put text and link in it
// with a space between; an empty link leaves it to text (a caller whose
// message already holds the link), an empty text leaves only the link.
// An unknown channel is "".
func URL(channel, link, text, subject string) string {
	switch channel {
	case Email:
		return "mailto:?subject=" + mailEscape(subject) + "&body=" + mailEscape(joined(text, link))
	case X:
		q := []string{}
		if text != "" {
			q = append(q, "text="+url.QueryEscape(text))
		}
		if link != "" {
			q = append(q, "url="+url.QueryEscape(link))
		}
		return "https://x.com/intent/post?" + strings.Join(q, "&")
	case LinkedIn:
		return "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(link)
	case Facebook:
		return "https://www.facebook.com/sharer/sharer.php?u=" + url.QueryEscape(link)
	case Bluesky:
		return "https://bsky.app/intent/compose?text=" + url.QueryEscape(joined(text, link))
	case WhatsApp:
		return "https://wa.me/?text=" + url.QueryEscape(joined(text, link))
	}
	return ""
}

// joined is the one message of text and link.
func joined(text, link string) string {
	switch {
	case text == "":
		return link
	case link == "":
		return text
	}
	return text + " " + link
}

// mailEscape escapes a mailto: header value (RFC 6068): every reserved
// character, & = + ? # included, is percent-encoded, so a title such as
// "x&bcc=a@evil.io" stays in the subject and never adds a header; spaces
// are %20 (mail apps don't read "+" as a space).
func mailEscape(v string) string {
	return strings.ReplaceAll(url.QueryEscape(v), "+", "%20")
}
