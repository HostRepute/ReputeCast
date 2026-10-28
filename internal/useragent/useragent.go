// Package useragent recognizes the user agents of machines that fetch
// email images and follow links: security scanners, link checkers, HTTP
// libraries, crawlers and link previews (review AN-9). It matches whole
// product tokens rather than substrings, so a phone called CUBOT or a
// browser build with "bot" inside a word is still a person.
//
// The patterns are a list the instance keeps (tracking_user_agents, review
// AN-2): the built-in ones come with releases, and instance admins add the
// ones they meet. Words that mark any machine (a word ending in "bot",
// "crawler", "spider", "slurp", a "preview") are rules, not patterns; a
// person's device whose word a rule takes for a machine is listed as a
// person.
package useragent

import (
	"strings"
)

// Kinds of machines, as Bot names them when it isn't a product.
const (
	Headless = "a headless browser"
	Script   = "a script"
	AnyBot   = "a bot"
	Preview  = "a link preview"
)

// Kinds of patterns (tracking_user_agents.kind).
const (
	// KindScanner: a mail-security scanner or link checker, by its name.
	KindScanner = "scanner"
	// KindScript: an HTTP library or a script.
	KindScript = "script"
	// KindHeadless: a headless browser.
	KindHeadless = "headless"
	// KindPreview: a chat app's or social network's link preview.
	KindPreview = "preview"
	// KindPerson: a word of a person's device or app that a rule would
	// take for a machine's ("CUBOT" phones end in "bot").
	KindPerson = "person"
)

// Kinds are the pattern kinds, machines first.
var Kinds = []string{KindScanner, KindScript, KindHeadless, KindPreview, KindPerson}

// Pattern is one entry of a user agent list.
type Pattern struct {
	// Text is lower case. A machine's is found anywhere in the user agent;
	// a person's is a whole word of it.
	Text string
	Kind string
	// Name is the product or who runs it ("Barracuda"); a script, headless
	// browser or link preview may have none.
	Name string
}

// What is what the pattern's machine is called in a verdict: its name, or
// its kind ("a script").
func (p Pattern) What() string {
	if p.Name != "" {
		return p.Name
	}
	switch p.Kind {
	case KindHeadless:
		return Headless
	case KindScript:
		return Script
	case KindPreview:
		return Preview
	}
	return AnyBot
}

// Builtin are the patterns that come with ReputeCast.
func Builtin() []Pattern {
	return []Pattern{
		{"barracuda", KindScanner, "Barracuda"}, {"mimecast", KindScanner, "Mimecast"}, {"proofpoint", KindScanner, "Proofpoint"},
		{"safelinks", KindScanner, "Microsoft Defender"}, {"microsoft office protocol discovery", KindScanner, "Microsoft Defender"},
		{"trendmicro", KindScanner, "Trend Micro"}, {"trend micro", KindScanner, "Trend Micro"}, {"sophos", KindScanner, "Sophos"},
		{"fireeye", KindScanner, "FireEye"}, {"forcepoint", KindScanner, "Forcepoint"}, {"symantec", KindScanner, "Symantec"},
		{"ironport", KindScanner, "Cisco Secure Email"}, {"zscaler", KindScanner, "Zscaler"}, {"checkpoint", KindScanner, "Check Point"},
		{"headlesschrome", KindHeadless, ""}, {"phantomjs", KindHeadless, ""},
		{"python-requests", KindScript, ""}, {"python-urllib", KindScript, ""}, {"curl/", KindScript, ""},
		{"wget/", KindScript, ""}, {"go-http-client", KindScript, ""}, {"okhttp", KindScript, ""},
		{"java/", KindScript, ""}, {"libwww", KindScript, ""}, {"httpclient", KindScript, ""},
		{"linkexpanding", KindPreview, ""}, {"facebookexternalhit", KindPreview, ""},
		{"cubot", KindPerson, "CUBOT phones"},
	}
}

// List is a user agent list: machines found by their patterns, and the
// words of people's devices the rules would take for machines.
type List struct {
	machines []Pattern
	people   map[string]bool
}

// NewList makes a list of patterns (machines in their order).
func NewList(ps []Pattern) *List {
	l := &List{people: map[string]bool{}}
	for _, p := range ps {
		t := strings.ToLower(strings.TrimSpace(p.Text))
		if t == "" {
			continue
		}
		if p.Kind == KindPerson {
			l.people[t] = true
			continue
		}
		p.Text = t
		l.machines = append(l.machines, p)
	}
	return l
}

var builtin = NewList(Builtin())

// botWords are words a crawler's token may contain anywhere.
var botWords = []string{"crawler", "spider", "slurp"}

// Bot returns what kind of machine a user agent is by the built-in list
// ("Barracuda", "a script", "a bot", "a link preview"), or "" for a
// browser or mail client.
func Bot(ua string) string { return builtin.Bot(ua) }

// Bot returns what kind of machine a user agent is by the list, or "" for
// a browser or mail client. A nil list is the built-in one.
func (l *List) Bot(ua string) string {
	if l == nil {
		l = builtin
	}
	low := strings.ToLower(strings.TrimSpace(ua))
	if low == "" {
		return ""
	}
	for _, p := range l.machines {
		if strings.Contains(low, p.Text) {
			return p.What()
		}
	}
	for _, tok := range tokens(low) {
		if l.people[tok] {
			continue
		}
		switch {
		case strings.HasSuffix(tok, "bot"):
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

// Word reports whether a person's pattern is one word, as the rules read a
// user agent: letters and digits.
func Word(text string) bool {
	t := tokens(strings.ToLower(text))
	return len(t) == 1 && t[0] == strings.ToLower(text)
}

// people are user agents of people reading and clicking email: a machine
// pattern found in one of them would take people for machines.
var people = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
	"Mozilla/5.0 (iPad; CPU OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0",
	"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Thunderbird/128.3.0",
	"Microsoft Office/16.0 (Windows NT 10.0; Microsoft Outlook 16.0.17928; Pro)",
	"Mozilla/5.0 (Windows NT 5.1; rv:11.0) Gecko Firefox/11.0 (via ggpht.com GoogleImageProxy)",
	"YahooMailProxy; https://help.yahoo.com/kb/yahoo-mail-proxy-SLN28749.html",
	"Mozilla/5.0",
}

// TakesPeople reports whether a machine pattern is found in the user agent
// of a browser or mail client people read email with.
func TakesPeople(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	for _, ua := range people {
		if strings.Contains(strings.ToLower(ua), t) {
			return true
		}
	}
	return false
}

// tokens splits a user agent into its words: "Mozilla/5.0 (compatible;
// Googlebot/2.1)" is mozilla, 5, 0, compatible, googlebot, 2, 1.
func tokens(low string) []string {
	return strings.FieldsFunc(low, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}
