package offers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Referral rewards (Grow › Referrals): a milestone's coupon reward issues
// the referrer a single-use promotion code of a live "Referral reward"
// offer. The code works on the upgrade page like an offer link
// (upgrade?offer=INS-OH7K2Q), once and until it expires; the subscription
// that uses it marks it redeemed.

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

// ReferralCode is an issued single-use code and the link that applies it.
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
		code := prefix + "-" + randomCode(6)
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
		return ReferralCode{Code: code, ClaimURL: s.Earn.PageURL(ctx, ws, "upgrade?offer="+code), ExpiresAt: expires}, nil
	}
	return ReferralCode{}, errors.New("offers: no free referral code after 5 tries")
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

// randomCode is n characters of codeAlphabet from crypto/rand.
func randomCode(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}

// singleUse is a referral reward's code by the code a reader brings, with
// its offer.
func (s *Service) singleUse(ctx context.Context, ws *store.Workspace, code string) (store.EarnOfferCode, store.OfferGetRow, bool) {
	c, err := s.Q.OfferCodeByCode(ctx, store.OfferCodeByCodeParams{WorkspaceID: ws.ID, Code: code})
	if err != nil {
		return store.EarnOfferCode{}, store.OfferGetRow{}, false
	}
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: c.OfferID, WorkspaceID: ws.ID})
	return c, r, err == nil
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
