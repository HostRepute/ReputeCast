package texting

import (
	"crypto/sha256"
	"encoding/binary"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
)

// linkRe finds links in a text: an address with a scheme, or a host with
// a path ("acme.io/summit/agenda"). An e-mail address isn't a link.
var linkRe = regexp.MustCompile(`(?i)(?:https?://[^\s]+|\b(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}/[^\s]*)`)

// publicShorteners are filtered by carriers: texts never carry them.
var publicShorteners = []string{"bit.ly", "tinyurl.com", "t.co", "goo.gl", "ow.ly", "is.gd", "buff.ly", "rebrand.ly", "cutt.ly", "shorturl.at", "rb.gy", "tiny.cc", "s.id", "t.ly", "lnkd.in"}

// CodeLength is a short link's code: go.acme.io/k3P9xq.
const CodeLength = 6

// Link is a link in a text.
type Link struct {
	// URL as written ("acme.io/summit/agenda").
	URL string
	// Host is its host, lower case.
	Host string
	// Shortener is set when the host is a public shortener (blocked).
	Shortener bool
	// Start and End are byte offsets in the body.
	Start, End int
}

// findLinks lists the links of a body.
func findLinks(body string) []Link {
	out := []Link{}
	for _, m := range linkRe.FindAllStringIndex(body, -1) {
		u := strings.TrimRight(body[m[0]:m[1]], ".,;:!?)")
		end := m[0] + len(u)
		if m[0] > 0 && body[m[0]-1] == '@' {
			continue
		}
		raw := u
		if !strings.Contains(raw, "://") {
			raw = "https://" + raw
		}
		pu, err := url.Parse(raw)
		if err != nil || pu.Host == "" {
			continue
		}
		host := strings.ToLower(strings.TrimPrefix(pu.Hostname(), "www."))
		out = append(out, Link{URL: u, Host: host, Shortener: slices.Contains(publicShorteners, host), Start: m[0], End: end})
	}
	return out
}

// Text is a text as it will be sent: the locked parts and the body.
type Text struct {
	Prefix string
	Body   string
	OptOut string
}

// Full writes the text: "Acme: … Reply STOP to opt out".
func (t Text) Full() string {
	parts := []string{}
	for _, p := range []string{t.Prefix, strings.TrimSpace(t.Body), t.OptOut} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, strings.TrimSpace(p))
		}
	}
	return strings.Join(parts, " ")
}

// shortLen is a short link's length on a domain: go.acme.io/k3P9xq is 17.
func shortLen(domain string) int { return len(domain) + 1 + CodeLength }

// withShortLinks replaces each link (not public shorteners) by a short
// link: code(i) gives the code of the i-th link. Without a short-link
// domain, links stay as written.
func withShortLinks(body, domain string, code func(i int, l Link) string) string {
	if domain == "" {
		return body
	}
	links := findLinks(body)
	var b strings.Builder
	last := 0
	for i, l := range links {
		if l.Shortener {
			continue
		}
		b.WriteString(body[last:l.Start])
		b.WriteString(domain + "/" + code(i, l))
		last = l.End
	}
	b.WriteString(body[last:])
	return b.String()
}

// placeholder is a code for counting: the length of a real one.
func placeholder(int, Link) string { return strings.Repeat("x", CodeLength) }

// codeAlphabet is base62 without look-alikes kept: codes are read aloud
// rarely, typed never.
const codeAlphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// LinkCode is the code of a recipient's i-th link in a message: derived
// from the message, so a preview shows the code the person gets.
func LinkCode(seed uuid.UUID, contact uuid.UUID, i int) string {
	h := sha256.New()
	h.Write(seed[:])
	h.Write(contact[:])
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(i))
	h.Write(b[:])
	sum := h.Sum(nil)
	n := binary.BigEndian.Uint64(sum[:8])
	out := make([]byte, CodeLength)
	for j := range out {
		out[j] = codeAlphabet[n%uint64(len(codeAlphabet))]
		n /= uint64(len(codeAlphabet))
	}
	return string(out)
}

// render personalizes a body for a recipient (fallbacks for missing
// fields). A Liquid error leaves the body as written.
func render(body string, r Recipient) string {
	if !strings.Contains(body, "{{") && !strings.Contains(body, "{%") {
		return body
	}
	out, err := liquid.Render(body, liquid.Vars(r.Email, r.FirstName, r.LastName, r.Attributes, nil))
	if err != nil {
		return body
	}
	return out
}

// tokensOf lists the fields a body prints.
func tokensOf(body string) []liquid.Use { return liquid.Uses(body) }

// stripTokens is a body without its tokens, for "151 characters + first
// name".
var tokenRe = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}`)

func stripTokens(body string) string { return tokenRe.ReplaceAllString(body, "") }

// Measured is how a text counts for the composer.
type Measured struct {
	// Count is the text with every token at its fallback and links at a
	// short link's length.
	Count sms.Count
	// Base is the length without the personalized parts.
	Base int
	// Fields are the fields that change the length.
	Fields []liquid.Use
	Links  []Link
}

// measure counts a text.
func measure(t Text, domain string) Measured {
	body := withShortLinks(t.Body, domain, placeholder)
	full := Text{Prefix: t.Prefix, Body: render(body, Recipient{}), OptOut: t.OptOut}.Full()
	m := Measured{Count: sms.CountText(full), Links: findLinks(t.Body), Fields: []liquid.Use{}}
	m.Fields = append(m.Fields, tokensOf(t.Body)...)
	base := Text{Prefix: t.Prefix, Body: stripTokens(body), OptOut: t.OptOut}.Full()
	m.Base = sms.CountText(base).Characters
	return m
}

// Spread is how many recipients get each number of segments.
type Spread struct {
	Segments   int
	Recipients int64
}

// spread counts the segments of a personalized text per recipient: the
// shares, and the longest example.
func spread(t Text, domain string, rs []Recipient) (out []Spread, longest *Recipient, longestCount sms.Count) {
	bySeg := map[int]int64{}
	for i := range rs {
		c := sms.CountText(Text{Prefix: t.Prefix, Body: render(withShortLinks(t.Body, domain, placeholder), rs[i]), OptOut: t.OptOut}.Full())
		bySeg[c.Segments]++
		if longest == nil || c.Characters > longestCount.Characters {
			longest, longestCount = &rs[i], c
		}
	}
	keys := make([]int, 0, len(bySeg))
	for k := range bySeg {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		out = append(out, Spread{Segments: k, Recipients: bySeg[k]})
	}
	return out, longest, longestCount
}

// utmCampaign is a campaign's UTM name: its first three words, as a slug
// ("Summit: 8 days to go" → "summit-8-days").
func utmCampaign(name string) string {
	words := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	if len(words) > 3 {
		words = words[:3]
	}
	return strings.Join(words, "-")
}

// target is where a short link leads: the address with UTM parameters.
func target(link, campaign string) string {
	raw := link
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Set("utm_source", "sms")
	q.Set("utm_medium", "sms")
	if campaign != "" {
		q.Set("utm_campaign", campaign)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// displayPhone writes a number for people: +1 312 555 0123.
func displayPhone(e164 string) string { return phone.Display(e164) }

// maskPhone hides its middle: +1 773 ••• 0148.
func maskPhone(e164 string) string { return phone.Mask(e164) }

// countOf counts a text's segments.
func countOf(text string) sms.Count { return sms.CountText(text) }
