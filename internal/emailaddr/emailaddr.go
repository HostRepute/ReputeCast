// Package emailaddr checks and classifies email addresses for the audience
// and the importer: syntax, role addresses, disposable domains, likely
// typos, spam-trap-like patterns, mailbox providers and masking.
package emailaddr

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Problems returned by Check. They are short phrases shown next to the
// raw value in import previews and reports.
const (
	ProblemNoDomain     = "no domain"
	ProblemDoubleDot    = "double dot"
	ProblemNotAnAddress = "not an address"
	ProblemTooLong      = "too long"
)

// Normalize trims spaces and angle brackets and lower-cases the address.
func Normalize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "mailto:")
	s = strings.Trim(s, "<> \t")
	return strings.ToLower(s)
}

// Check normalizes s and reports why it isn't a deliverable address, or ""
// when the syntax is fine. It does not look up DNS.
func Check(s string) (string, string) {
	e := Normalize(s)
	if e == "" {
		return e, ProblemNotAnAddress
	}
	at := strings.LastIndexByte(e, '@')
	if at < 0 || strings.Count(e, "@") > 1 || strings.ContainsAny(e, " \t\"(),:;<>[]\\") {
		return e, ProblemNotAnAddress
	}
	local, domain := e[:at], e[at+1:]
	if local == "" {
		return e, ProblemNotAnAddress
	}
	if domain == "" {
		return e, ProblemNoDomain
	}
	if len(e) > 254 || len(local) > 64 {
		return e, ProblemTooLong
	}
	if strings.Contains(local, "..") || strings.Contains(domain, "..") {
		return e, ProblemDoubleDot
	}
	if local[0] == '.' || local[len(local)-1] == '.' {
		return e, ProblemNotAnAddress
	}
	for _, r := range local {
		if !localRune(r) {
			return e, ProblemNotAnAddress
		}
	}
	if !strings.Contains(domain, ".") {
		return e, ProblemNoDomain
	}
	labels := strings.Split(domain, ".")
	for _, l := range labels {
		if l == "" || l[0] == '-' || l[len(l)-1] == '-' || len(l) > 63 {
			return e, ProblemNotAnAddress
		}
		for _, r := range l {
			if r != '-' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return e, ProblemNotAnAddress
			}
		}
	}
	if tld := labels[len(labels)-1]; utf8.RuneCountInString(tld) < 2 || strings.IndexFunc(tld, unicode.IsDigit) >= 0 {
		return e, ProblemNotAnAddress
	}
	return e, ""
}

// Valid reports whether s is a syntactically valid address.
func Valid(s string) bool {
	_, p := Check(s)
	return p == ""
}

func localRune(r rune) bool {
	if r > unicode.MaxASCII {
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	return strings.ContainsRune(".!#$%&'*+/=?^_`{|}~-", r)
}

// Local returns the part before the @.
func Local(email string) string {
	if i := strings.LastIndexByte(email, '@'); i >= 0 {
		return email[:i]
	}
	return email
}

// Domain returns the part after the @, lower-cased.
func Domain(email string) string {
	if i := strings.LastIndexByte(email, '@'); i >= 0 {
		return strings.ToLower(email[i+1:])
	}
	return ""
}

// MailboxKey folds the addresses that reach the same mailbox: it
// normalizes the address, drops a plus tag and the dots of a Gmail
// address, and reads googlemail.com as gmail.com. Sam.Rivera+news@gmail.com
// and samrivera@googlemail.com give the same key.
func MailboxKey(email string) string {
	e := Normalize(email)
	at := strings.LastIndexByte(e, '@')
	if at < 0 {
		return e
	}
	local, domain := e[:at], e[at+1:]
	if i := strings.IndexByte(local, '+'); i > 0 {
		local = local[:i]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	return local + "@" + domain
}

// roleLocals are shared inboxes: high complaint rates, never personal consent.
var roleLocals = map[string]bool{
	"abuse": true, "accounts": true, "admin": true, "administrator": true, "billing": true, "careers": true,
	"contact": true, "customerservice": true, "devnull": true, "dns": true, "enquiries": true, "ftp": true,
	"hello": true, "help": true, "helpdesk": true, "hostmaster": true, "hr": true, "info": true, "inquiries": true,
	"it": true, "jobs": true, "mail": true, "mailer-daemon": true, "marketing": true, "media": true, "news": true,
	"no-reply": true, "noc": true, "noreply": true, "office": true, "orders": true, "postmaster": true, "press": true,
	"privacy": true, "root": true, "sales": true, "security": true, "service": true, "support": true, "team": true,
	"webmaster": true, "www": true,
}

// IsRole reports whether the address is a role address (info@, sales@…).
func IsRole(email string) bool {
	l := strings.ToLower(Local(email))
	if i := strings.IndexByte(l, '+'); i > 0 {
		l = l[:i]
	}
	return roleLocals[l]
}

// Mailbox providers, in the order the charts use.
const (
	Gmail   = "gmail"
	Outlook = "outlook"
	Yahoo   = "yahoo"
	ICloud  = "icloud"
	Other   = "other"
)

// MailboxProviders lists the keys in chart order.
var MailboxProviders = []string{Gmail, Outlook, Yahoo, ICloud, Other}

// MailboxName is the display name of a mailbox provider key.
func MailboxName(key string) string {
	switch key {
	case Gmail:
		return "Gmail"
	case Outlook:
		return "Outlook"
	case Yahoo:
		return "Yahoo"
	case ICloud:
		return "iCloud"
	}
	return "Other"
}

// MailboxProvider classifies a domain into the mailbox provider that
// filters its mail. Hosted domains (Google Workspace, Microsoft 365) stay
// "other"; their hosting is found from MX records (see HostingFromMX).
func MailboxProvider(domain string) string {
	d := strings.ToLower(strings.TrimSpace(domain))
	switch d {
	case "gmail.com", "googlemail.com":
		return Gmail
	case "outlook.com", "hotmail.com", "live.com", "msn.com", "windowslive.com", "passport.com":
		return Outlook
	case "ymail.com", "rocketmail.com", "aol.com", "aim.com", "aol.co.uk":
		return Yahoo
	case "icloud.com", "me.com", "mac.com":
		return ICloud
	}
	// Country variants: hotmail.co.uk, outlook.fr, yahoo.co.jp, live.com.au.
	if i := strings.IndexByte(d, '.'); i > 0 && countryTLD(d[i+1:]) {
		switch d[:i] {
		case "hotmail", "outlook", "live":
			return Outlook
		case "yahoo":
			return Yahoo
		}
	}
	return Other
}

// countryTLD reports suffixes such as "fr", "co.uk" and "com.au".
func countryTLD(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		if len(p) < 2 || len(p) > 3 {
			return false
		}
	}
	return true
}

// HostingFromMX names the hosting of a domain from one of its MX hosts:
// "Google Workspace", "Microsoft 365", "Fastmail", "Proton", "Zoho Mail".
func HostingFromMX(host string) string {
	h := strings.TrimSuffix(strings.ToLower(host), ".")
	switch {
	case strings.HasSuffix(h, "google.com"), strings.HasSuffix(h, "googlemail.com"):
		return "Google Workspace"
	case strings.HasSuffix(h, "mail.protection.outlook.com"), strings.HasSuffix(h, "outlook.com"):
		return "Microsoft 365"
	case strings.HasSuffix(h, "messagingengine.com"), strings.HasSuffix(h, "fastmail.com"):
		return "Fastmail"
	case strings.HasSuffix(h, "protonmail.ch"), strings.HasSuffix(h, "proton.me"):
		return "Proton"
	case strings.HasSuffix(h, "zoho.com"), strings.HasSuffix(h, "zoho.eu"):
		return "Zoho Mail"
	case strings.HasSuffix(h, "mx.icloud.com"):
		return "iCloud Mail"
	}
	return ""
}

// Mask hides most of the local part: "r.walker@outlook.com" → "r.w···@outlook.com".
func Mask(email string) string {
	local, domain := Local(email), Domain(email)
	if domain == "" {
		return email
	}
	keep := 3
	if utf8.RuneCountInString(local) <= 3 {
		keep = 1
	}
	r := []rune(local)
	if keep > len(r) {
		keep = len(r)
	}
	return string(r[:keep]) + "···@" + domain
}

// trapLocals are local parts that only exist to catch list buyers and scrapers.
var trapLocals = []string{"spamtrap", "spam-trap", "spam.trap", "honeypot", "honey-pot", "trapaddress", "blackhole"}

// TrapLike reports addresses that look like spam traps: honeypot local
// parts and catch-all scraper patterns. HostRepute knows real trap
// networks; this is the offline heuristic.
func TrapLike(email string) bool {
	l := strings.ToLower(Local(email))
	for _, t := range trapLocals {
		if strings.Contains(l, t) {
			return true
		}
	}
	return false
}

// typoTargets are the domains people mistype most. Short domains (me.com,
// aol.com, live.com) are left out because one edit away are real domains.
var typoTargets = []string{
	"gmail.com", "googlemail.com", "outlook.com", "hotmail.com", "hotmail.co.uk",
	"yahoo.com", "yahoo.co.uk", "icloud.com", "protonmail.com",
}

// realNeighbours are real mail domains one edit away from a typo target.
var realNeighbours = map[string]bool{
	"mail.com": true, "email.com": true, "gmx.com": true, "gmx.net": true, "gmx.de": true, "web.de": true,
	"zmail.com": true, "xmail.com": true, "amail.com": true,
}

// TypoFix returns the intended domain when domain is one edit (insertion,
// deletion, substitution or swap of neighbours) away from a major provider.
func TypoFix(domain string) (string, bool) {
	d := strings.ToLower(domain)
	if MailboxProvider(d) != Other || realNeighbours[d] || IsDisposable(d) {
		return "", false
	}
	for _, t := range typoTargets {
		if editDistanceAtMostOne(d, t) {
			return t, true
		}
	}
	return "", false
}

// editDistanceAtMostOne implements the optimal string alignment distance
// check for a threshold of one.
func editDistanceAtMostOne(a, b string) bool {
	if a == b {
		return true
	}
	la, lb := len(a), len(b)
	switch {
	case la == lb:
		var diffs []int
		for i := 0; i < la; i++ {
			if a[i] != b[i] {
				diffs = append(diffs, i)
				if len(diffs) > 2 {
					return false
				}
			}
		}
		if len(diffs) == 1 {
			return true
		}
		// One swap of neighbours ("gmial" → "gmail").
		return len(diffs) == 2 && diffs[1] == diffs[0]+1 && a[diffs[0]] == b[diffs[1]] && a[diffs[1]] == b[diffs[0]]
	case la == lb+1:
		return oneDeletion(a, b)
	case lb == la+1:
		return oneDeletion(b, a)
	}
	return false
}

// oneDeletion reports whether removing one byte from long gives short.
func oneDeletion(long, short string) bool {
	i := 0
	for i < len(short) && long[i] == short[i] {
		i++
	}
	return long[i+1:] == short[i:]
}
