package editorial

import (
	"regexp"

	"golang.org/x/text/unicode/norm"
)

// What a person writes to the assistant is chat, not a brief: greetings
// with a name, dates, ids and encoded text are ordinary there, so Text
// would refuse too much. Chat refuses only what is plainly a contact's
// record or a credential: an email address, a phone number in
// international form or shaped as one beyond doubt (an ISO date can't
// match), an API key, a private key, or a password, secret or token given
// with its value.
var (
	chatEmail = regexp.MustCompile(`[\pL\pN.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[\pL\pN.-]+\.[\pL]{2,}`)
	chatPhone = regexp.MustCompile(`\+\d[\d ().-]{6,}\d|\(\d{2,4}\)[ .-]?\d{3,4}[ .-]?\d{3,4}\b|\b\d{3}[ .-]\d{3}[ .-]\d{4}\b`)
	chatKey   = regexp.MustCompile(`(?i)\b(?:sk-|AKIA|ASIA|ghp_|github_pat_)[a-z0-9_-]{12,}|-----BEGIN [^-]*PRIVATE KEY-----|\b(?:api[_ -]?key|password|secret|bearer|access[_ -]?token|token)\s*[:=]\s*\S{6,}`)
)

// Chat names the private record a chat message holds: "email", "phone" or
// "credential", or "" for none.
func Chat(text string) string {
	check := norm.NFKC.String(text)
	switch {
	case chatEmail.MatchString(check):
		return "email"
	case chatKey.MatchString(check):
		return "credential"
	}
	for _, m := range chatPhone.FindAllString(check, -1) {
		// At least eight digits: a short "+1 (2) 3" isn't a number.
		digits := 0
		for _, r := range m {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		if digits >= 8 {
			return "phone"
		}
	}
	return ""
}
