package offers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Referral rewards (Grow › Referrals): a milestone's coupon reward issues
// the referrer a single-use promotion code of a live "Referral reward"
// offer. The code works on the upgrade page like an offer link, through
// the referrer's own personal link (upgrade?r=…&offer=INS-…), once and
// until it expires; the subscription that uses it marks it redeemed. For
// anyone else the code is unknown: it can't be guessed (12 random
// characters), tried (an address that keeps trying finds none for a
// while) or used by whoever gets hold of it.

// ReferralOffer is a live referral offer a milestone can reward with.
type ReferralOffer struct {
	ID   uuid.UUID
	Name string
	// Discount and AppliesTo are in the reader's language: "100% off for 3
	// months", "Insider · monthly".
	Discount  string
	AppliesTo string
	Issued    int64
	Redeemed  int64
}

// ReferralCode is an issued single-use code and the link that applies it:
// the referrer's own upgrade link, the only one the code works with.
type ReferralCode struct {
	Code      string
	ClaimURL  string
	ExpiresAt time.Time
}

// errNotIssuable: the offer can't issue codes now (ended, not a referral
// reward, or its coupon isn't saved in the provider yet); the reward waits
// and tries again.
var errNotIssuable = errors.New("offers: this referral offer can't issue a code now")

// codeAlphabet leaves out characters people confuse (0/O, 1/I/L).
const codeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// codeLength is how many random characters follow a code's prefix: 12 of
// codeAlphabet's 31 are about 59 bits, so a code can't be found by trying
// codes, on the upgrade page or in the provider's checkout.
const codeLength = 12

// ReferralOffers lists the workspace's live referral offers.
func (s *Service) ReferralOffers(ctx context.Context, ws *store.Workspace) ([]ReferralOffer, error) {
	rows, err := s.Q.OfferReferralOffers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]ReferralOffer, 0, len(rows))
	for _, r := range rows {
		a := i18n.Args{"type": r.DiscountType, "duration": r.Duration, "percent": 0, "amount": "", "months": 0}
		if r.PercentOff != nil {
			a["percent"] = int(*r.PercentOff)
		}
		if r.AmountOff != nil {
			a["amount"] = moneyOf(*r.AmountOff, r.Currency)
		}
		if r.DurationMonths != nil {
			a["months"] = int(*r.DurationMonths)
		}
		applies := []string{}
		for _, ap := range parseApplies(r.Applies) {
			tid, err := uuid.Parse(ap.Tier)
			if err != nil {
				continue
			}
			applies = append(applies, text(ctx, "referral.applies", i18n.Args{"tier": s.Earn.TierShortName(ctx, ws.ID, tid), "period": ap.Period}))
		}
		out = append(out, ReferralOffer{ID: r.ID, Name: r.Name, Discount: text(ctx, "referral.discount", a), AppliesTo: strings.Join(applies, ", "),
			Issued: r.Issued, Redeemed: r.Redeemed})
	}
	return out, nil
}

// IssueReferralCode creates a single-use promotion code of a referral offer
// in the payment provider for a contact and records it; validDays of 0 uses
// the offer's own validity.
func (s *Service) IssueReferralCode(ctx context.Context, ws *store.Workspace, offerID, contactID uuid.UUID, validDays int) (ReferralCode, error) {
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: offerID, WorkspaceID: ws.ID})
	if err != nil {
		return ReferralCode{}, err
	}
	mode := s.Earn.ActiveMode(ctx, ws.ID)
	sm := parseStripe(r.Stripe)[mode]
	if r.Kind != KindReferral || r.Status != "live" || mode == "" || sm == nil || sm.Coupon == "" || sm.MissingAt != "" {
		return ReferralCode{}, errNotIssuable
	}
	client, _, err := s.Earn.ClientFor(ctx, ws.ID, mode)
	if err != nil {
		return ReferralCode{}, err
	}
	off, ok := client.(payments.Offers)
	if !ok {
		return ReferralCode{}, errNotIssuable
	}
	if validDays <= 0 {
		validDays = int(r.CodeDays)
	}
	now := s.now()
	expires := now.AddDate(0, 0, validDays)
	issuedTo := ""
	if err := s.Pool.QueryRow(ctx, `SELECT coalesce(nullif(trim(first_name || ' ' || last_name), ''), email) FROM contacts WHERE id = $1 AND workspace_id = $2`,
		contactID, ws.ID).Scan(&issuedTo); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ReferralCode{}, err
	}
	meta := map[string]string{"reputecast_offer": r.ID.String(), "reputecast_workspace": ws.ID.String(), "reputecast_contact": contactID.String(),
		"reputecast_source": "referral"}
	prefix := s.codePrefix(ctx, ws, r)
	// A code someone already holds (here or in the provider) is drawn again.
	for range 5 {
		code := prefix + "-" + randomCode(codeLength)
		if _, err := s.Q.OfferCodeByCode(ctx, store.OfferCodeByCodeParams{WorkspaceID: ws.ID, Code: code}); err == nil {
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return ReferralCode{}, err
		}
		pc, err := off.CreatePromotionCode(ctx, payments.PromotionCode{Code: code, CouponID: sm.Coupon, MaxRedemptions: 1, ExpiresAt: &expires,
			FirstTimeOnly: r.NewOnly, Metadata: meta})
		var ae *payments.APIError
		if errors.As(err, &ae) && ae.Code == "resource_already_exists" {
			continue
		}
		if err != nil {
			return ReferralCode{}, err
		}
		stripe, _ := json.Marshal(stripeState{mode: {Promo: pc.ID}})
		var contact *uuid.UUID
		if contactID != uuid.Nil {
			contact = &contactID
		}
		if err := s.Q.OfferCodeInsert(ctx, store.OfferCodeInsertParams{ID: newID(), WorkspaceID: ws.ID, OfferID: r.ID, Code: code, ContactID: contact,
			IssuedTo: issuedTo, IssuedAt: now, ExpiresAt: expires, Stripe: stripe, Source: "referral"}); err != nil {
			// Unused, the code mustn't stay redeemable in the provider.
			_ = off.SetPromotionCodeActive(context.WithoutCancel(ctx), pc.ID, false)
			return ReferralCode{}, err
		}
		return ReferralCode{Code: code, ClaimURL: s.ReferralClaimURL(ctx, ws, contactID, code), ExpiresAt: expires}, nil
	}
	return ReferralCode{}, errors.New("offers: no free referral code after 5 tries")
}

// ReferralClaimURL is the link that applies a referral reward's code: its
// referrer's personal upgrade link, the only one the code works with.
func (s *Service) ReferralClaimURL(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, code string) string {
	return s.Earn.UpgradeURL(ctx, ws, contactID, uuid.Nil) + "&offer=" + code
}

// codePrefix starts a code with the tier it unlocks ("INS" for Insider), or
// "REF" when the offer applies to several or none.
func (s *Service) codePrefix(ctx context.Context, ws *store.Workspace, r store.OfferGetRow) string {
	tiers := map[string]bool{}
	for _, a := range parseApplies(r.Applies) {
		tiers[a.Tier] = true
	}
	if len(tiers) == 1 {
		for t := range tiers {
			if tid, err := uuid.Parse(t); err == nil {
				var b strings.Builder
				for _, c := range strings.ToUpper(s.Earn.TierShortName(ctx, ws.ID, tid)) {
					if c >= 'A' && c <= 'Z' && b.Len() < 3 {
						b.WriteRune(c)
					}
				}
				if b.Len() == 3 {
					return b.String()
				}
			}
		}
	}
	return "REF"
}

// randomCode is n characters of codeAlphabet from crypto/rand, each as
// likely as the others.
func randomCode(n int) string {
	// 256 isn't a multiple of 31: bytes from 248 up are drawn again rather
	// than make the first characters likelier.
	const unbiased = 256 - 256%len(codeAlphabet)
	out := make([]byte, 0, n)
	buf := make([]byte, n)
	for len(out) < n {
		_, _ = rand.Read(buf)
		for _, b := range buf {
			if int(b) < unbiased && len(out) < n {
				out = append(out, codeAlphabet[int(b)%len(codeAlphabet)])
			}
		}
	}
	return string(out)
}

// singleUse is a referral reward's code by the code a reader brings, with
// its offer, when it was issued to reader (the contact of the page's
// personal link). A code issued to someone else, or to no one, is as
// unknown as a code that doesn't exist; so is every code for an address
// that tried too many (codeMisses).
func (s *Service) singleUse(ctx context.Context, ws *store.Workspace, code string, reader uuid.UUID) (store.EarnOfferCode, store.OfferGetRow, bool) {
	if reader == uuid.Nil || !s.codeMisses.allow(clientOf(ctx), s.now()) {
		return store.EarnOfferCode{}, store.OfferGetRow{}, false
	}
	c, err := s.Q.OfferCodeByCode(ctx, store.OfferCodeByCodeParams{WorkspaceID: ws.ID, Code: code})
	if err != nil || c.ContactID == nil || *c.ContactID != reader {
		return store.EarnOfferCode{}, store.OfferGetRow{}, false
	}
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: c.OfferID, WorkspaceID: ws.ID})
	return c, r, err == nil
}

// missed counts a code that was neither an offer's nor one of the reader's
// single-use codes against the request's address.
func (s *Service) missed(ctx context.Context) {
	s.codeMisses.miss(clientOf(ctx), s.now())
}

// clientOf is the address a request came from (the public pages'), for
// the miss limit.
func clientOf(ctx context.Context) string {
	if ip := audit.RequestFrom(ctx).IP; ip != nil {
		return ip.String()
	}
	return "unknown"
}

// Failed lookups of single-use codes per address: after missLimit in
// missWindow the address finds none until the window is over, the limit
// of the read-only report links' unknown links (reports.Limiter).
const (
	missLimit  = 20
	missWindow = 10 * time.Minute
)

// misses counts each address's failed single-use code lookups in a fixed
// window. The zero value is ready to use.
type misses struct {
	mu    sync.Mutex
	by    map[string]*missCount
	sweep time.Time
}

type missCount struct {
	start time.Time
	n     int
}

// allow reports whether an address may still look single-use codes up.
func (m *misses) allow(addr string, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	w := m.by[addr]
	return w == nil || now.Sub(w.start) >= missWindow || w.n < missLimit
}

// miss counts a failed lookup from an address.
func (m *misses) miss(addr string, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.by == nil {
		m.by = map[string]*missCount{}
	}
	if now.Sub(m.sweep) >= missWindow {
		// Windows that are over are forgotten, so the map stays small.
		m.sweep = now
		for k, w := range m.by {
			if now.Sub(w.start) >= missWindow {
				delete(m.by, k)
			}
		}
	}
	w := m.by[addr]
	if w == nil || now.Sub(w.start) >= missWindow {
		w = &missCount{start: now}
		m.by[addr] = w
	}
	w.n++
}

// usable reports whether a single-use code can still be applied.
func (s *Service) usable(c store.EarnOfferCode) bool {
	return c.RedeemedAt == nil && c.ExpiresAt.After(s.now())
}

// CodeRedeemed implements earn.OfferCodeRedeemer: a subscription took a
// single-use code (a code that isn't one changes nothing).
func (s *Service) CodeRedeemed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, code string, at time.Time) error {
	_, err := store.New(tx).OfferCodeRedeem(ctx, store.OfferCodeRedeemParams{WorkspaceID: ws.ID, Code: code, At: at})
	return err
}
