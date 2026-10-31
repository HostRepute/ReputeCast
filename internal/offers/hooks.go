package offers

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The upgrade page and checkout (earn.OfferHook): an offer link shows the
// offer's price on the tier it applies to and checkout applies its code;
// an ended offer says so and the page shows the normal prices.

var (
	_ earn.OfferHook         = (*Service)(nil)
	_ earn.ComposerOfferHook = (*Service)(nil)
	_ earn.OfferCodeRedeemer = (*Service)(nil)
)

func moneyOf(cents int64, currency string) string { return earn.Money(cents, currency) }

// discounted is a price after the offer's discount.
func discounted(price int64, r store.OfferGetRow) int64 {
	switch {
	case r.PercentOff != nil:
		return int64(math.Round(float64(price) * float64(100-*r.PercentOff) / 100))
	case r.AmountOff != nil:
		return max(price-*r.AmountOff, 0)
	}
	return price
}

// live reports whether an offer works now: not ended, before its date,
// under its limit, and saved in the provider of mode.
func (s *Service) live(ctx context.Context, ws *store.Workspace, r store.OfferGetRow, mode string) bool {
	if r.Status != "live" || (r.EndsAt != nil && !r.EndsAt.After(s.now())) {
		return false
	}
	st := parseStripe(r.Stripe)
	sm := st[mode]
	if sm == nil || sm.Coupon == "" || sm.MissingAt != "" {
		return false
	}
	if r.Kind == KindSignup && sm.Promo == "" {
		return false
	}
	if r.RedeemLimit != nil && s.results(ctx, ws, r, st).Redemptions >= int64(*r.RedeemLimit) {
		return false
	}
	return true
}

func (s *Service) byCode(ctx context.Context, ws *store.Workspace, code string) (store.OfferGetRow, bool) {
	o, err := s.Q.OfferByCode(ctx, store.OfferByCodeParams{WorkspaceID: ws.ID, Code: code})
	if err != nil {
		return store.OfferGetRow{}, false
	}
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: o.ID, WorkspaceID: ws.ID})
	return r, err == nil
}

// PeriodFor is the period an offer link opens the page on: the one it
// applies to, when it applies to one. A code that isn't one counts against
// the address in UpgradeOffer, once per page, rather than here.
func (s *Service) PeriodFor(ctx context.Context, ws *store.Workspace, code string, reader uuid.UUID) string {
	r, ok := s.byCode(ctx, ws, code)
	if !ok {
		if _, r, ok = s.singleUse(ctx, ws, code, reader); !ok {
			return ""
		}
	}
	periods := []string{}
	for _, a := range parseApplies(r.Applies) {
		if !slices.Contains(periods, a.Period) {
			periods = append(periods, a.Period)
		}
	}
	if len(periods) == 1 {
		return periods[0]
	}
	return ""
}

// CheckoutCode implements earn.OfferHook: a signup offer's code, or a
// referral reward's single-use code while it's unused and unexpired, for
// the reader it was issued to.
func (s *Service) CheckoutCode(ctx context.Context, ws *store.Workspace, mode, code string, reader, tier uuid.UUID, period string) string {
	if r, ok := s.byCode(ctx, ws, code); ok {
		if !s.live(ctx, ws, r, mode) || !applies(r, tier.String(), period) {
			return ""
		}
		return parseStripe(r.Stripe)[mode].Promo
	}
	c, r, ok := s.singleUse(ctx, ws, code, reader)
	if !ok {
		s.missed(ctx)
		return ""
	}
	if !s.usable(c) || !s.live(ctx, ws, r, mode) || !applies(r, tier.String(), period) {
		return ""
	}
	if sm := parseStripe(c.Stripe)[mode]; sm != nil {
		return sm.Promo
	}
	return ""
}

func applies(r store.OfferGetRow, tier, period string) bool {
	for _, a := range parseApplies(r.Applies) {
		if a.Tier == tier && a.Period == period {
			return true
		}
	}
	return false
}

// dateOf is a day as the pages write it: "15 Oct 2027".
func dateOf(l i18n.Lang, t time.Time, ws *store.Workspace) string {
	return earn.DateIn(l, t.In(location(ws)))
}

// UpgradeOffer implements earn.OfferHook: the offer's lines in the page's
// language (the view's).
func (s *Service) UpgradeOffer(ctx context.Context, ws *store.Workspace, mode, code string, reader uuid.UUID, v *earn.UpgradeView) error {
	l := v.Lang
	pg := func(key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }
	// offerCode is what checkout applies: the offer's code, or a referral
	// reward's single-use code.
	r, ok := s.byCode(ctx, ws, code)
	offerCode := r.Code
	if !ok {
		// A single-use code shows only on its reader's own page. To anyone
		// else a live, used, expired or unknown code looks the same: the
		// normal prices, with nothing to tell them apart.
		c, rr, found := s.singleUse(ctx, ws, code, reader)
		if !found {
			s.missed(ctx)
			return nil
		}
		if !s.usable(c) {
			// Its reader learns why it no longer applies.
			key := "page.codeExpired"
			if c.RedeemedAt != nil {
				key = "page.codeUsed"
			}
			v.Notice = &earn.PageNotice{Tone: "info", Title: pg(key+"Title", nil), Text: pg(key+"Text", i18n.Args{"offer": rr.Name})}
			return nil
		}
		r, offerCode = rr, c.Code
	}
	loc := location(ws)
	if !s.live(ctx, ws, r, mode) {
		args := i18n.Args{"offer": r.Name}
		key := "page.endedText"
		if r.EndsAt != nil && (r.EndedReason == "date" || (r.EndedAt == nil && !r.EndsAt.After(s.now()))) {
			args["date"] = earn.DayMonthIn(l, r.EndsAt.In(loc))
			key = "page.endedTextOn"
		}
		v.Notice = &earn.PageNotice{Tone: "info", Title: pg("page.endedTitle", nil), Text: pg(key, args)}
		return nil
	}
	now := s.now()
	for i := range v.Cards {
		c := &v.Cards[i]
		if !applies(r, c.ID, c.Period) {
			continue
		}
		tid, err := uuid.Parse(c.ID)
		if err != nil {
			continue
		}
		full, cur, ok := s.priceOf(ctx, ws, tid, c.Period)
		if !ok {
			continue
		}
		price := discounted(full, r)
		short := s.Earn.TierShortName(ctx, ws.ID, tid)
		a := i18n.Args{"price": moneyOf(price, cur), "full": moneyOf(full, cur), "period": c.Period, "tier": short, "offer": r.Name}
		if r.DurationMonths != nil {
			a["months"] = int(*r.DurationMonths)
		}
		key := r.Duration
		if r.Duration == "months" && c.Period == "year" {
			key = "once"
		}
		c.Amount = moneyOf(price, cur)
		c.Unit = pg("page.unit."+key, a)
		c.Sub = pg("page.sub."+key, a)
		c.Trial = ""
		c.Button = pg("page.button", a)
		// When the full price starts: a year or a month on, or after the
		// offer's months.
		renews := now.AddDate(1, 0, 0)
		if c.Period == "month" {
			renews = now.AddDate(0, 1, 0)
		}
		if r.DurationMonths != nil && key == "months" {
			renews = now.AddDate(0, int(*r.DurationMonths), 0)
		}
		a["date"] = dateOf(l, renews, ws)
		c.Note = pg("page.note."+key, a)
		if v.Offer == nil {
			who := "page.forNew"
			if !r.NewOnly {
				who = "page.forAll"
			}
			if r.EndsAt != nil {
				a["until"] = earn.DayMonthIn(l, r.EndsAt.In(loc))
				who += "Until"
			}
			v.Offer = &earn.OfferBanner{Label: pg("page.line."+key, a), Text: pg(who, a)}
		}
	}
	if v.Offer != nil {
		v.OfferCode = offerCode
	}
	return nil
}

// ComposerOffers implements earn.ComposerOfferHook: the live signup
// offers on a tier, each with the period it applies to (the year when it
// applies to both), as the Upgrade block's settings and card say them.
func (s *Service) ComposerOffers(ctx context.Context, ws *store.Workspace, tier uuid.UUID) []earn.EarnComposerOffer {
	out := []earn.EarnComposerOffer{}
	mode, _, _ := s.account(ctx, ws)
	if mode == "" {
		return out
	}
	rows, err := s.Q.OfferList(ctx, ws.ID)
	if err != nil {
		return out
	}
	short := s.Earn.TierShortName(ctx, ws.ID, tier)
	// The line and the button go into the email: in the workspace's default
	// language, the language its emails are written in. The option is the
	// settings' own, in the member's.
	el := earn.WorkspaceLang(ctx, s.Q, ws.ID)
	for _, row := range rows {
		r := store.OfferGetRow(row)
		if r.Kind != KindSignup || !s.live(ctx, ws, r, mode) {
			continue
		}
		for _, period := range []string{"year", "month"} {
			if !applies(r, tier.String(), period) {
				continue
			}
			full, cur, ok := s.priceOf(ctx, ws, tier, period)
			if !ok {
				continue
			}
			a := i18n.Args{"price": moneyOf(discounted(full, r), cur), "full": moneyOf(full, cur), "period": period, "tier": short, "offer": r.Name}
			key := r.Duration
			if r.DurationMonths != nil {
				a["months"] = int(*r.DurationMonths)
			}
			if r.Duration == "months" && period == "year" {
				key = "once"
			}
			out = append(out, earn.EarnComposerOffer{Code: r.Code, Name: r.Name, Period: period, Option: text(ctx, "composer.option."+key, a),
				Line: msgs.M("composer.line."+key).Text(el, a), Button: msgs.M("composer.button").Text(el, a), EndsAt: r.EndsAt})
			break
		}
	}
	return out
}

// priceOf is a tier's current price for a period.
func (s *Service) priceOf(ctx context.Context, ws *store.Workspace, tier uuid.UUID, period string) (int64, string, bool) {
	rows, err := s.Q.OfferTierPrices(ctx, ws.ID)
	if err != nil {
		return 0, "", false
	}
	for _, r := range rows {
		if r.TierID == tier && r.Period == period {
			return r.AmountCents, r.Currency, true
		}
	}
	return 0, "", false
}

// Retention is the retention offer a paid member gets on the cancel page.
type Retention struct {
	ID       uuid.UUID
	Name     string
	Coupon   string
	Currency string
	Period   string
	// PriceCents is what they'd pay with it, FullCents without; Months is
	// how long (0: once, -1: forever).
	PriceCents int64
	FullCents  int64
	Months     int
}

// RetentionFor is the retention offer a paid member is eligible for, or
// nil: the offer applies to their tier and period, they've paid for at
// least its months, they're not past due or on a discount, and they took
// no retention offer in the last 12 months.
func (s *Service) RetentionFor(ctx context.Context, ws *store.Workspace, mode string, sub store.PaidSubscription) (*Retention, error) {
	o, err := s.Q.OfferRetentionOf(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: o.ID, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	if sub.TierID == nil || sub.Kind != "stripe" || sub.Status != "active" || len(sub.Discount) > 0 || !applies(r, sub.TierID.String(), sub.Period) {
		return nil, nil
	}
	if !s.live(ctx, ws, r, mode) {
		return nil, nil
	}
	now := s.now()
	if r.MinPaidMonths > 0 && sub.StartedAt.AddDate(0, int(r.MinPaidMonths), 0).After(now) {
		return nil, nil
	}
	used, err := s.Q.OfferRetentionUsedSince(ctx, store.OfferRetentionUsedSinceParams{ContactID: sub.ContactID, Since: now.Add(-RetentionGap)})
	if err != nil || used {
		return nil, err
	}
	ret := &Retention{ID: r.ID, Name: r.Name, Coupon: parseStripe(r.Stripe)[mode].Coupon, Currency: sub.Currency, Period: sub.Period, FullCents: sub.AmountCents,
		PriceCents: discounted(sub.AmountCents, r)}
	switch r.Duration {
	case "months":
		if r.DurationMonths != nil {
			ret.Months = int(*r.DurationMonths)
		}
	case "forever":
		ret.Months = -1
	}
	return ret, nil
}
