// Package bounce maps SMTP replies and enhanced status codes (RFC 3463) to
// ReputeCast's shared bounce taxonomy, so every adapter classifies the same
// diagnostic the same way: mailbox_unknown, mailbox_full, domain_invalid,
// policy_block, spam_block, reputation, temporary, other.
package bounce

import (
	"regexp"
	"strings"
)

// Classes of the shared taxonomy.
const (
	MailboxUnknown = "mailbox_unknown"
	MailboxFull    = "mailbox_full"
	DomainInvalid  = "domain_invalid"
	PolicyBlock    = "policy_block"
	SpamBlock      = "spam_block"
	Reputation     = "reputation"
	Temporary      = "temporary"
	Other          = "other"
)

var (
	enhancedRe     = regexp.MustCompile(`\b([245])\.(\d{1,3})\.(\d{1,3})\b`)
	enhancedOnlyRe = regexp.MustCompile(`^[245]\.\d{1,3}\.\d{1,3}$`)
	replyOnlyRe    = regexp.MustCompile(`^[245]\d\d$`)
	replyRe        = regexp.MustCompile(`\b([245]\d\d)\b`)
)

// Split separates an SMTP reply such as "smtp; 550 5.1.1 <a@b>: User unknown"
// into its code ("550 5.1.1") and the rest ("<a@b>: User unknown"). Replies
// without a leading code return ("", reply).
func Split(reply string) (code, rest string) {
	s := strings.TrimSpace(reply)
	if len(s) >= 5 && strings.EqualFold(s[:5], "smtp;") {
		s = strings.TrimSpace(s[5:])
	}
	var codes []string
	pos := 0
	for len(codes) < 2 {
		tok, next := nextToken(s, pos)
		if tok == "" {
			break
		}
		t := strings.TrimRight(tok, "-")
		if replyOnlyRe.MatchString(t) && len(codes) == 0 || enhancedOnlyRe.MatchString(t) {
			codes = append(codes, t)
			pos = next
			continue
		}
		// Multi-line replies glue the code to the next part: "550-5.7.1".
		if len(codes) == 0 && len(t) > 4 && t[3] == '-' && replyOnlyRe.MatchString(t[:3]) {
			codes = append(codes, t[:3])
			if enhancedOnlyRe.MatchString(t[4:]) {
				codes = append(codes, t[4:])
				pos = next
				continue
			}
			pos = next - len(tok) + 4
			continue
		}
		break
	}
	if len(codes) == 0 {
		return "", s
	}
	return strings.Join(codes, " "), strings.TrimSpace(s[pos:])
}

// nextToken returns the whitespace-delimited token at or after pos and the
// index just past it.
func nextToken(s string, pos int) (string, int) {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t' || s[pos] == '\r' || s[pos] == '\n') {
		pos++
	}
	start := pos
	for pos < len(s) && s[pos] != ' ' && s[pos] != '\t' && s[pos] != '\r' && s[pos] != '\n' {
		pos++
	}
	return s[start:pos], pos
}

// Enhanced returns the first enhanced status code ("5.1.1") in s, or "".
func Enhanced(s string) string {
	return enhancedRe.FindString(s)
}

// Permanent reports whether a status or reply is permanent (5xx / 5.x.x).
// Unknown codes are treated as temporary so nothing is suppressed by guess.
func Permanent(s string) bool {
	if e := Enhanced(s); e != "" {
		return e[0] == '5'
	}
	if m := replyRe.FindString(s); m != "" {
		return m[0] == '5'
	}
	return false
}

// Classify returns the taxonomy class for a status code and diagnostic text.
func Classify(status, diagnostic string) string {
	text := strings.ToLower(status + " " + diagnostic)
	e := Enhanced(status)
	if e == "" {
		e = Enhanced(diagnostic)
	}
	switch {
	case containsAny(text, "reputation", "tss04", "sender score", "ip rate limited", "rate limited due to"):
		return Reputation
	case containsAny(text, "spamhaus", "blocklist", "blacklist", "listed on", "rbl", "spam", "content rejected", "message content", "virus", "malware", "phish"):
		return SpamBlock
	}
	switch e {
	case "5.1.1", "5.1.0", "5.1.6", "5.2.1":
		return MailboxUnknown
	case "4.2.2", "5.2.2":
		return MailboxFull
	case "5.1.2", "5.1.10", "4.1.2", "5.4.4", "4.4.4":
		return DomainInvalid
	case "5.7.1", "5.7.0", "5.7.23", "5.7.24", "5.7.25", "5.7.26", "5.7.27", "5.4.1", "5.7.509", "5.7.606":
		return PolicyBlock
	}
	switch {
	case containsAny(text, "user unknown", "no such user", "does not exist", "mailbox not found", "unknown recipient", "invalid recipient", "no mailbox"):
		return MailboxUnknown
	case containsAny(text, "mailbox full", "over quota", "quota exceeded", "insufficient storage"):
		return MailboxFull
	case containsAny(text, "domain not found", "host not found", "no mx", "nxdomain", "unrouteable", "unroutable"):
		return DomainInvalid
	case containsAny(text, "dmarc", "spf", "dkim", "not authorized", "access denied", "relay", "policy"):
		return PolicyBlock
	}
	if e != "" && e[0] == '4' {
		return Temporary
	}
	if m := replyRe.FindString(status); m != "" && m[0] == '4' {
		return Temporary
	}
	return Other
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}
