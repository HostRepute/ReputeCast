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

var _ earn.OfferHook = (*Service)(nil)

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
// applies to, when it applies to one.
func (s *Service) PeriodFor(ctx context.Context, ws *store.Workspace, code string) string {
	r, ok := s.byCode(ctx, ws, code)
	if !ok {
		return ""
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

// CheckoutCode implements earn.OfferHook.
func (s *Service) CheckoutCode(ctx context.Context, ws *store.Workspace, mode, code string, tier uuid.UUID, period string) string {
	r, ok := s.byCode(ctx, ws, code)
	if !ok || !s.live(ctx, ws, r, mode) || !applies(r, tier.String(), period) {
		return ""
	}
	return parseStripe(r.Stripe)[mode].Promo
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
func dateOf(t time.Time, ws *store.Workspace) string { return t.In(location(ws)).Format("2 Jan 2006") }

// UpgradeOffer implements earn.OfferHook.
func (s *Service) UpgradeOffer(ctx context.Context, ws *store.Workspace, mode, code string, v *earn.UpgradeView) error {
	r, ok := s.byCode(ctx, ws, code)
	if !ok {
		return nil
	}
	loc := location(ws)
	if !s.live(ctx, ws, r, mode) {
		args := i18n.Args{"offer": r.Name}
		key := "page.endedText"
		if r.EndsAt != nil && (r.EndedReason == "date" || (r.EndedAt == nil && !r.EndsAt.After(s.now()))) {
			args["date"] = r.EndsAt.In(loc).Format("2 Jan")
			key = "page.endedTextOn"
		}
		v.Notice = &earn.PageNotice{Tone: "info", Title: page("page.endedTitle", nil), Text: page(key, args)}
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
		c.Unit = page("page.unit."+key, a)
		c.Sub = page("page.sub."+key, a)
		c.Trial = ""
		c.Button = page("page.button", a)
		// When the full price starts: a year or a month on, or after the
		// offer's months.
		renews := now.AddDate(1, 0, 0)
		if c.Period == "month" {
			renews = now.AddDate(0, 1, 0)
		}
		if r.DurationMonths != nil && key == "months" {
			renews = now.AddDate(0, int(*r.DurationMonths), 0)
		}
		a["date"] = dateOf(renews, ws)
		c.Note = page("page.note."+key, a)
		if v.Offer == nil {
			who := "page.forNew"
			if !r.NewOnly {
				who = "page.forAll"
			}
			if r.EndsAt != nil {
				a["until"] = r.EndsAt.In(loc).Format("2 Jan")
				who += "Until"
			}
			v.Offer = &earn.OfferBanner{Label: page("page.line."+key, a), Text: page(who, a)}
		}
	}
	if v.Offer != nil {
		v.OfferCode = r.Code
	}
	return nil
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
