// Package clicktoken is the click token of phase 6 (`rc_click`): an opaque,
// signed token the click redirect adds to links to a publisher's own sites
// (stores, and the domains the conversion API lists), so an order or a
// conversion arriving later can be credited to the email that brought the
// click. It names the message (so the contact and the campaign or the
// automation step) and when the click happened; it carries nothing in
// clear and can't be forged (internal/linktoken's HMAC).
package clicktoken

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Param is the query parameter that carries the token.
const Param = "rc_click"

// Kind is the token's linktoken kind (X20's range, 200–209).
const Kind linktoken.Kind = 200

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
	c, err := s.Verify(strings.TrimSpace(token), Kind)
	if err != nil || c.MessageID == uuid.Nil {
		return Click{}, ErrInvalid
	}
	return Click{WorkspaceID: c.WorkspaceID, MessageID: c.MessageID, ContactID: c.ContactID, At: epoch.Add(time.Duration(c.Link) * time.Minute)}, nil
}

// UTM are the parameters a tagged link gets.
type UTM struct {
	Source, Medium, Campaign string
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
// value ("October Product Update" → "october-product-update").
func Slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
