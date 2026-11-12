// Package received holds what the adapters whose provider receives mail
// (providers.InboundReceiver) share: the hook URL with the route's basic
// authentication, its check, and the readers that turn a provider's parsed
// payload into a providers.InboundMessage the same way everywhere
// (verdicts, addresses, Message-ID lists, base64 content, spam scores).
package received

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// HookURL returns the inbound hook URL with the route's basic
// authentication in it (https://reputecast:<secret>@host/hooks/inbound/…),
// for providers that authenticate a URL's user information.
func HookURL(raw, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("the inbound route has no secret")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", fmt.Errorf("inbound hook URL %q is not an HTTP URL", raw)
	}
	u.User = url.UserPassword(providers.InboundUser, secret)
	return u.String(), nil
}

// WithoutUser is a URL without its user information, as providers that
// keep credentials apart list it; an unparsable URL is returned as it is.
func WithoutUser(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.User = nil
	return u.String()
}

// SameURL compares two URLs without their user information and a trailing
// slash.
func SameURL(a, b string) bool {
	return strings.TrimRight(WithoutUser(a), "/") == strings.TrimRight(WithoutUser(b), "/")
}

// CheckBasic checks the hook's basic authentication (providers.InboundUser
// and the route's secret) in constant time.
func CheckBasic(r *http.Request, route providers.InboundRoute) error {
	if err := webhookauth.Basic(r, providers.InboundUser, route.Secret); err != nil {
		return fmt.Errorf("%w: the request doesn't carry the route's credentials", providers.ErrBadSignature)
	}
	return nil
}

// CheckToken compares a secret sent as it is (a header) with the route's
// secret in constant time.
func CheckToken(got string, route providers.InboundRoute) error {
	if !webhookauth.Equal(got, route.Secret) {
		return fmt.Errorf("%w: the request doesn't carry the route's token", providers.ErrBadSignature)
	}
	return nil
}

// Verdict maps a provider's SPF, DKIM or DMARC result to "pass", "fail",
// "none" or "" (unknown, neutral, a soft failure, or a check that
// couldn't run, as Amazon SES's GRAY and PROCESSING_FAILED).
func Verdict(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pass", "passed":
		return "pass"
	case "fail", "failed", "hardfail":
		return "fail"
	case "none":
		return "none"
	}
	return ""
}

// SpamScore turns a spam score into a verdict: "spam" at or above
// threshold, "pass" below it, "" when there is no score.
func SpamScore(raw string, threshold float64) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return ""
	}
	if v >= threshold {
		return "spam"
	}
	return "pass"
}

// SpamAssassin is the score at which SpamAssassin calls a message spam by
// default (required_score).
const SpamAssassin = 5.0

// Address writes a mailbox as a header does: "Name <addr>", or the address
// alone without a name.
func Address(name, addr string) string {
	addr = strings.TrimSpace(addr)
	name = strings.TrimSpace(name)
	if addr == "" {
		return ""
	}
	if name == "" {
		return addr
	}
	return (&mail.Address{Name: name, Address: addr}).String()
}

// Addresses splits an address list header ("A <a@x>, b@y") into its
// addresses, each as Address writes it; a list that doesn't parse is kept
// as one entry.
func Addresses(header string) []string {
	header = strings.TrimSpace(header)
	if header == "" {
		return []string{}
	}
	list, err := mail.ParseAddressList(header)
	if err != nil {
		return []string{header}
	}
	out := make([]string, 0, len(list))
	for _, a := range list {
		out = append(out, Address(a.Name, a.Address))
	}
	return out
}

// Bare is the address of "Name <addr>" (or addr itself), lower-cased.
func Bare(s string) string {
	s = strings.TrimSpace(s)
	if a, err := mail.ParseAddress(s); err == nil {
		return strings.ToLower(a.Address)
	}
	return strings.ToLower(strings.Trim(s, "<>"))
}

// MessageIDs reads the Message-IDs of a References or In-Reply-To header,
// each with its angle brackets.
func MessageIDs(header string) []string {
	out := []string{}
	for _, f := range strings.Fields(header) {
		if strings.HasPrefix(f, "<") && strings.HasSuffix(f, ">") {
			out = append(out, f)
		}
	}
	return out
}

// Base64 decodes content a provider sends base64-encoded, with or without
// padding and line breaks.
func Base64(s string) ([]byte, error) {
	clean := strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', ' ', '\t':
			return -1
		}
		return r
	}, s)
	if b, err := base64.StdEncoding.DecodeString(clean); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(strings.TrimRight(clean, "="))
}

// Header returns the first value of a header in a list of name and value
// pairs, matching the name without regard to case.
func Header(h map[string][]string, name string) string {
	for k, vs := range h {
		if strings.EqualFold(k, name) && len(vs) > 0 {
			return strings.TrimSpace(vs[0])
		}
	}
	return ""
}

// RawHeaders reads the header block of a raw message (what precedes the
// first blank line), for an adapter that needs a header or two from a
// message it hands over whole. A message without a parsable header block
// has none.
func RawHeaders(raw []byte) map[string][]string {
	end := bytes.Index(raw, []byte("\r\n\r\n"))
	if end < 0 {
		end = bytes.Index(raw, []byte("\n\n"))
	}
	if end < 0 {
		end = len(raw)
	}
	m, err := mail.ReadMessage(bytes.NewReader(append(append([]byte(nil), raw[:end]...), "\r\n\r\n"...)))
	if err != nil {
		return map[string][]string{}
	}
	return m.Header
}
