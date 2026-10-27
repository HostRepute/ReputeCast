package tracking

import (
	"embed"
	"regexp"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What people read from tracking follows their language
// (messages/<locale>.json): why an open or a click was a machine's, and
// the mail client of a person's, as the live monitor's feed and the event
// views show them, and the page of a link that no longer works. A verdict
// is stored in English in delivery_events.diagnostic, as before;
// DiagnosticIn writes it in a reader's language when it reads, rows from
// before included. Mail client, browser, system and scanner product names
// stay as they are ("Chrome on macOS" is "Chrome trên macOS").

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "tracking")

// reasons are the machine verdicts by their English.
var reasons = map[string]string{
	reasonHead:      "reason.head",
	reasonNoAgent:   "reason.noAgent",
	reasonApple:     "reason.apple",
	reasonGoogle:    "reason.google",
	reasonYahoo:     "reason.yahoo",
	reasonFastOpen:  "reason.fastOpen",
	reasonFastClick: "reason.fastClick",
	// Before the windows were measured from delivery.
	"Opened within 2 s of sending, likely a scanner":        "reason.fastOpen",
	"Clicked within 10 s of sending, likely a link scanner": "reason.fastClick",
	reasonBurst:    "reason.burst",
	reasonHoneypot: "reason.honeypot",
	clientUnknown:  "client.unknown",
}

// The verdicts' English, as diagnostic stores it.
const (
	reasonHead      = "HEAD request from a link scanner"
	reasonNoAgent   = "No user agent, likely a scanner"
	reasonApple     = "Apple Mail Privacy Protection prefetch"
	reasonGoogle    = "Google image proxy prefetch"
	reasonYahoo     = "Yahoo Mail image proxy prefetch"
	reasonFastOpen  = "Opened within 2 s of arriving, likely a scanner"
	reasonFastClick = "Clicked within 10 s of arriving, likely a link scanner"
	reasonBurst     = "Every link clicked within seconds, likely a link scanner"
	reasonHoneypot  = "Followed the email's hidden link, a link scanner"
	clientUnknown   = "Unknown mail client"
)

// scannerKinds are the scanners named by what they are, not a product.
var scannerKinds = map[string]string{
	"a headless browser": "headless",
	"a script":           "script",
	"a bot":              "bot",
	"a link preview":     "preview",
}

var (
	reProxy    = regexp.MustCompile(`^Privacy proxy prefetch \((.+)\)$`)
	reScanner  = regexp.MustCompile(`^(Security|Link) scanner \((.+)\)$`)
	reAnyOn    = regexp.MustCompile(`^A mail client on (iOS|Android|Windows|macOS|ChromeOS|Linux)$`)
	reClientOn = regexp.MustCompile(`^(Outlook|Thunderbird|Edge|Firefox|Chrome|Safari|Apple Mail) on (iOS|Android|Windows|macOS|ChromeOS|Linux)$`)
)

// DiagnosticIn writes a stored verdict ("Apple Mail Privacy Protection
// prefetch", "Chrome on macOS") in a reader's language. Text tracking
// didn't write, such as a provider's bounce message, is returned as it is.
func DiagnosticIn(l i18n.Lang, s string) string {
	if s == "" || l.Locale == "" || l.Locale == i18n.English {
		return s
	}
	if key, ok := reasons[s]; ok {
		return msgs.M(key).Text(l, nil)
	}
	if m := reProxy.FindStringSubmatch(s); m != nil {
		return msgs.M("reason.privacyProxy").Text(l, i18n.Args{"name": m[1]})
	}
	if m := reScanner.FindStringSubmatch(s); m != nil {
		args := i18n.Args{"kind": "product", "name": m[2]}
		if k, ok := scannerKinds[m[2]]; ok {
			args["kind"] = k
		}
		if m[1] == "Link" {
			return msgs.M("reason.linkScanner").Text(l, args)
		}
		return msgs.M("reason.securityScanner").Text(l, args)
	}
	if m := reAnyOn.FindStringSubmatch(s); m != nil {
		return msgs.M("client.anyOn").Text(l, i18n.Args{"os": m[1]})
	}
	if m := reClientOn.FindStringSubmatch(s); m != nil {
		return msgs.M("client.on").Text(l, i18n.Args{"app": m[1], "os": m[2]})
	}
	return s
}
