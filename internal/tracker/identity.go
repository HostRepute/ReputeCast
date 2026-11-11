package tracker

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Who a visitor is, never from an address a page states:
//
//   - a click from an email: the click token (clicktoken, kind 201) the
//     click redirect adds as rc= to links to a tracked site;
//   - a signup on the same page: the form's answer carries a token (kind
//     LinkKind, ten minutes) that form.js hands to reputecast('link', …);
//   - a signed identify call: the site's server signs the lowercased
//     address with the site's identify secret (hex HMAC-SHA256).
//
// Only an existing contact of the workspace who hasn't been erased is
// linked; their last 30 days on the site then join their profile. A
// visitor linked to one contact and then identified as another (a shared
// browser, someone signing in as someone else) is linked to the second
// from then on: later events are theirs, earlier ones stay the first's.

// claim is who a call says its visitor is, verified (the contact's
// existence is checked when it is written).
type claim struct {
	contact uuid.UUID
	email   string
	by      string
}

// vidHash is how a visitor id is stored: its SHA-256 with the workspace,
// never the id itself.
func vidHash(ws uuid.UUID, vid string) []byte {
	h := sha256.New()
	h.Write(ws[:])
	h.Write([]byte{0})
	h.Write([]byte(vid))
	return h.Sum(nil)
}

// FormLink is the token a form signup's answer carries for the tracker on
// the same page: the contact, for ten minutes.
func FormLink(signer *linktoken.Signer, workspace, contact uuid.UUID, now time.Time) string {
	if signer == nil || contact == uuid.Nil {
		return ""
	}
	return signer.Sign(linktoken.Claims{Kind: LinkKind, WorkspaceID: workspace, ContactID: contact, Link: linktoken.UntilMinute(now.Add(LinkValid))})
}

// Signature is what a site's server sends with reputecast('identify'):
// hex(HMAC-SHA256(secret, lower(email))).
func Signature(secret, email string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(m.Sum(nil))
}

// claimOf verifies the identity a call carries: the form's link token
// first (the freshest), then a signed identify call, then the click token.
func (s *Service) claimOf(ctx context.Context, c collectSite, b *collectBody, now time.Time) *claim {
	if b.Link != "" && len(b.Link) <= 200 && s.Links != nil {
		if cl, err := s.Links.Verify(strings.TrimSpace(b.Link), LinkKind); err == nil && cl.WorkspaceID == c.Workspace && cl.ContactID != uuid.Nil &&
			!now.After(linktoken.Until(cl)) {
			return &claim{contact: cl.ContactID, by: ByForm}
		}
	}
	if b.Identify != nil && len(c.sealed) > 0 {
		if email, ok := s.identified(ctx, c, b.Identify); ok {
			return &claim{email: email, by: ByIdentify}
		}
	}
	if b.RC != "" && len(b.RC) <= 200 {
		if cl, err := clicktoken.Verify(s.Links, b.RC); err == nil && cl.WorkspaceID == c.Workspace && cl.ContactID != uuid.Nil {
			return &claim{contact: cl.ContactID, by: ByClick}
		}
	}
	return nil
}

// identified checks a signed identify call against the site's secret
// (compared in constant time) and returns the address it names.
func (s *Service) identified(ctx context.Context, c collectSite, in *identifyBody) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || len(email) > 254 {
		return "", false
	}
	if a, err := mail.ParseAddress(email); err != nil || a.Address != email {
		return "", false
	}
	want, err := hex.DecodeString(strings.TrimSpace(in.Signature))
	if err != nil || len(want) != sha256.Size {
		return "", false
	}
	secret, err := s.Box.Open(c.sealed, secretAD(c.ID))
	if err != nil {
		s.log().WarnContext(ctx, "tracker identify secret unreadable", "site", c.ID, "err", err)
		return "", false
	}
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(email))
	return email, hmac.Equal(m.Sum(nil), want)
}
