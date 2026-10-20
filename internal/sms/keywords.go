package sms

import (
	"errors"
	"slices"
	"strings"
	"unicode"
)

// Keyword kinds: what a reply asks for.
const (
	KeywordOptOut = "opt_out"
	KeywordOptIn  = "opt_in"
	KeywordHelp   = "help"
	// KeywordYes confirms consent when the workspace asks people to reply
	// YES first.
	KeywordYes = "yes"
	// KeywordReply is any other reply: it goes on the contact's timeline
	// and into the campaign report.
	KeywordReply = "reply"
)

// Keywords are the replies ReputeCast acts on.
type Keywords struct {
	OptOut []string `json:"optOut"`
	OptIn  []string `json:"optIn"`
	Help   []string `json:"help"`
}

// DefaultKeywords are the carriers' standard words.
var DefaultKeywords = Keywords{
	OptOut: []string{"STOP", "STOPALL", "UNSUBSCRIBE", "CANCEL", "END", "QUIT", "OPTOUT", "REVOKE"},
	OptIn:  []string{"START", "UNSTOP"},
	Help:   []string{"HELP", "INFO"},
}

// Normalize writes a keyword the way it's stored: trimmed, upper case.
func NormalizeKeyword(w string) string { return strings.ToUpper(strings.TrimSpace(w)) }

// Errors from Validate.
var (
	ErrKeywordEmpty     = errors.New("sms: a keyword can't be empty")
	ErrKeywordFormat    = errors.New("sms: a keyword is one word of letters and digits, up to 20")
	ErrKeywordDuplicate = errors.New("sms: a keyword can only mean one thing")
	ErrKeywordRequired  = errors.New("sms: STOP and HELP can't be removed")
)

func validWord(w string) bool {
	if w == "" || len([]rune(w)) > 20 {
		return false
	}
	for _, r := range w {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// Validate checks the lists: one word each, each word in one list only,
// and the words carriers require (STOP and HELP) kept.
func (k Keywords) Validate() error {
	seen := map[string]bool{}
	for _, list := range [][]string{k.OptOut, k.OptIn, k.Help} {
		for _, w := range list {
			w = NormalizeKeyword(w)
			if w == "" {
				return ErrKeywordEmpty
			}
			if !validWord(w) {
				return ErrKeywordFormat
			}
			if seen[w] {
				return ErrKeywordDuplicate
			}
			seen[w] = true
		}
	}
	if !slices.Contains(normalized(k.OptOut), "STOP") || !slices.Contains(normalized(k.Help), "HELP") {
		return ErrKeywordRequired
	}
	return nil
}

func normalized(ws []string) []string {
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, NormalizeKeyword(w))
	}
	return out
}

// Normalized returns the lists in their stored form.
func (k Keywords) Normalized() Keywords {
	return Keywords{OptOut: normalized(k.OptOut), OptIn: normalized(k.OptIn), Help: normalized(k.Help)}
}

// Match reads a reply. A keyword matches when it's the whole reply, in any
// case; anything else is a reply. yes reports whether YES confirms consent
// (the workspace asks people to reply YES first).
func (k Keywords) Match(body string, yes bool) (kind, word string) {
	w := strings.TrimSpace(body)
	if w == "" {
		return KeywordReply, ""
	}
	for _, c := range []struct {
		kind  string
		words []string
	}{{KeywordOptOut, k.OptOut}, {KeywordOptIn, k.OptIn}, {KeywordHelp, k.Help}} {
		for _, kw := range c.words {
			if strings.EqualFold(w, strings.TrimSpace(kw)) {
				return c.kind, NormalizeKeyword(kw)
			}
		}
	}
	if yes && strings.EqualFold(w, "YES") {
		return KeywordYes, "YES"
	}
	return KeywordReply, ""
}
