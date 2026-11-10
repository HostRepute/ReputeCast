// Package clicktoken is the click token: an opaque, signed token the click
// redirect adds to links to a publisher's own sites (stores, as `rc`; and
// the domains phase 6's conversion API lists, as `rc_click`), so an order
// or a conversion arriving later can be credited to the email that brought
// the click. It names the message (so the contact and the campaign or the
// automation step) and when the click happened; it carries nothing in
// clear and can't be forged (internal/linktoken's HMAC).
package clicktoken

import (
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Param is the query parameter that carries the token on links to a store
// (phase 7).
const Param = "rc"

// ConversionParam carries it on links to the conversion API's domains
// (phase 6).
const ConversionParam = "rc_click"

// Kind is the token's linktoken kind (X20's range, 200–209).
const Kind linktoken.Kind = 201

// legacyKind is the kind click tokens had before they got their own: the
// hidden link's (linktoken.Honeypot), so anyone who saw a store link's
// rc= could open it as the email's hidden link and make the reader's real
// click settle as a bot's. Tokens already in store links still credit
// orders: a click token's Link is the minute of its click, never 0 as
// the hidden link's is, so the two can't be taken for each other. They
// matter only within a store's attribution window (30 days at most)
// after the upgrade; then this can go.
const legacyKind = linktoken.Honeypot

// epoch is where the click minute counts from (fits 8,000 years in 32 bits).
var epoch = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// Click is what a token says.
type Click struct {
	WorkspaceID uuid.UUID
	MessageID   uuid.UUID
	ContactID   uuid.UUID
	// At is the click's time, to the minute.
	At time.Time
}

// ErrInvalid is a token that wasn't signed by this instance.
var ErrInvalid = errors.New("clicktoken: invalid token")

// Sign returns the token for a click.
func Sign(s *linktoken.Signer, c Click) string {
	minutes := c.At.UTC().Sub(epoch) / time.Minute
	if minutes < 0 {
		minutes = 0
	}
	return s.Sign(linktoken.Claims{Kind: Kind, WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: c.ContactID, Link: uint32(minutes)})
}

// Verify reads a token.
func Verify(s *linktoken.Signer, token string) (Click, error) {
	if s == nil || strings.TrimSpace(token) == "" {
		return Click{}, ErrInvalid
	}
	token = strings.TrimSpace(token)
	c, err := s.Verify(token, Kind)
	if err != nil {
		if c, err = s.Verify(token, legacyKind); err != nil || c.Link == 0 {
			return Click{}, ErrInvalid
		}
	}
	if c.MessageID == uuid.Nil {
		return Click{}, ErrInvalid
	}
	return Click{WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: c.ContactID, At: epoch.Add(time.Duration(c.Link) * time.Minute)}, nil
}

// UTM are the parameters a tagged link gets.
type UTM struct {
	Source   string `json:"source"`
	Medium   string `json:"medium"`
	Campaign string `json:"campaign"`
}

// ForEmail is the UTM of an email's links (Settings › Senders' "Add UTM
// tags to links"): the workspace's slug, email, and the campaign's or the
// automation's name as a slug, fallback when the name has nothing a slug
// keeps.
func ForEmail(workspace, name, fallback string) UTM {
	c := Slug(name)
	if c == "" {
		c = fallback
	}
	return UTM{Source: workspace, Medium: "email", Campaign: c}
}

// Key is the query AddUTM gives a link without parameters
// ("utm_source=acme-newsletter&utm_medium=email&utm_campaign=welcome"):
// what tells apart the link sets of one email sent with different tags.
func (u UTM) Key() string {
	parts := make([]string, 0, 3)
	for _, p := range u.params() {
		if p[1] != "" {
			parts = append(parts, p[0]+"="+url.QueryEscape(p[1]))
		}
	}
	return strings.Join(parts, "&")
}

// params are the parameters in the order AddUTM adds them.
func (u UTM) params() [3][2]string {
	return [3][2]string{{"utm_source", u.Source}, {"utm_medium", u.Medium}, {"utm_campaign", u.Campaign}}
}

// Taggable reports whether AddUTM tags a link: a web link without Liquid
// (a personalized address is left as written).
func Taggable(href string) bool {
	lower := strings.ToLower(href)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	return !strings.Contains(href, "{{") && !strings.Contains(href, "{%")
}

// AddUTM adds to a link the UTM parameters it doesn't set (the author's
// choice wins), in the order utm_source, utm_medium, utm_campaign, after
// its query and before its fragment. Unlike Tag it never re-encodes or
// reorders what the link already has, so a tagged link still reads as
// written. Links that aren't Taggable come back unchanged.
func AddUTM(href string, u UTM) string {
	if !Taggable(href) {
		return href
	}
	base, fragment, hasFragment := strings.Cut(href, "#")
	_, query, hasQuery := strings.Cut(base, "?")
	have, _ := url.ParseQuery(query)
	sep := "?"
	if hasQuery {
		sep = "&"
		if query == "" || strings.HasSuffix(query, "&") {
			sep = ""
		}
	}
	var b strings.Builder
	b.WriteString(base)
	added := false
	for _, p := range u.params() {
		if p[1] == "" || have.Has(p[0]) {
			continue
		}
		b.WriteString(sep)
		b.WriteString(p[0])
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p[1]))
		sep, added = "&", true
	}
	if !added {
		return href
	}
	if hasFragment {
		b.WriteByte('#')
		b.WriteString(fragment)
	}
	return b.String()
}

// Tag adds the UTM parameters (where the link doesn't set them already:
// the author's choice wins) and the click token to a link.
func Tag(link string, utm UTM, token string) string {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return link
	}
	q := u.Query()
	set := func(k, v string) {
		if v != "" && q.Get(k) == "" {
			q.Set(k, v)
		}
	}
	set("utm_source", utm.Source)
	set("utm_medium", utm.Medium)
	set("utm_campaign", utm.Campaign)
	if token != "" {
		q.Set(Param, token)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Slug turns a campaign's or an automation's name into a utm_campaign
// value ("October Product Update" → "october-product-update"). Letters lose
// their marks ("Bản tin tháng 10" → "ban-tin-thang-10"); letters outside
// ASCII once their marks are gone (中, ж) are dropped.
func Slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(asciiFold(strings.TrimSpace(name))) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Other scripts' letters (中, ж) are dropped.
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// asciiFold writes a name's letters without their marks: "Bản tin" →
// "Ban tin", "Straße" → "Strasse".
func asciiFold(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r == 'đ', r == 'Đ':
			b.WriteByte('d')
		case r == 'ß':
			b.WriteString("ss")
		case r == 'æ', r == 'Æ':
			b.WriteString("ae")
		case r == 'œ', r == 'Œ':
			b.WriteString("oe")
		case r == 'ø', r == 'Ø':
			b.WriteByte('o')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
