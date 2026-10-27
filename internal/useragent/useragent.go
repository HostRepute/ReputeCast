// Package useragent recognizes the user agents of machines that fetch
// email images and follow links: security scanners, link checkers, HTTP
// libraries, crawlers and link previews (review AN-9). It matches whole
// product tokens rather than substrings, so a phone called CUBOT or a
// browser build with "bot" inside a word is still a person.
package useragent

import "strings"

// Kinds of machines, as Bot names them when it isn't a product.
const (
	Headless = "a headless browser"
	Script   = "a script"
	AnyBot   = "a bot"
	Preview  = "a link preview"
)

// products are scanners and checkers known by a word in their user agent.
var products = []struct{ needle, name string }{
	{"barracuda", "Barracuda"}, {"mimecast", "Mimecast"}, {"proofpoint", "Proofpoint"},
	{"safelinks", "Microsoft Defender"}, {"microsoft office protocol discovery", "Microsoft Defender"},
	{"trendmicro", "Trend Micro"}, {"trend micro", "Trend Micro"}, {"sophos", "Sophos"},
	{"fireeye", "FireEye"}, {"forcepoint", "Forcepoint"}, {"symantec", "Symantec"},
	{"ironport", "Cisco Secure Email"}, {"zscaler", "Zscaler"}, {"checkpoint", "Check Point"},
	{"headlesschrome", Headless}, {"phantomjs", Headless},
	{"python-requests", Script}, {"python-urllib", Script}, {"curl/", Script},
	{"wget/", Script}, {"go-http-client", Script}, {"okhttp", Script},
	{"java/", Script}, {"libwww", Script}, {"httpclient", Script},
	{"linkexpanding", Preview}, {"facebookexternalhit", Preview},
}

// notBots are product tokens that end in "bot" but are people's devices.
var notBots = map[string]bool{"cubot": true}

// botWords are words a crawler's token may contain anywhere.
var botWords = []string{"crawler", "spider", "slurp"}

// Bot returns what kind of machine a user agent is ("Barracuda", "a
// script", "a bot", "a link preview"), or "" for a browser or mail client.
func Bot(ua string) string {
	low := strings.ToLower(strings.TrimSpace(ua))
	if low == "" {
		return ""
	}
	for _, p := range products {
		if strings.Contains(low, p.needle) {
			return p.name
		}
	}
	for _, tok := range tokens(low) {
		switch {
		case strings.HasSuffix(tok, "bot") && !notBots[tok]:
			return AnyBot
		case strings.Contains(tok, "preview"):
			return Preview
		}
		for _, w := range botWords {
			if strings.Contains(tok, w) {
				return AnyBot
			}
		}
	}
	return ""
}

// tokens splits a user agent into its words: "Mozilla/5.0 (compatible;
// Googlebot/2.1)" is mozilla, 5, 0, compatible, googlebot, 2, 1.
func tokens(low string) []string {
	return strings.FieldsFunc(low, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}
