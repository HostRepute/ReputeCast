package website

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// PaywallTier is a tier on the paywall card: "Weekly Digest Insider · $8
// a month or $80 a year · 14 days free".
type PaywallTier struct {
	ID        uuid.UUID
	Name      string
	Label     string
	Monthly   string
	Yearly    string
	TrialDays int
	Benefits  []string
	// Includes is the label of the lower tier it includes ("Insider").
	Includes string
}

// Payments is the payment provider the site sells through ("" when none
// is connected).
type Payments struct {
	Provider string
	Mode     string
}

// PaywallTiers are the tiers a reader can buy to read a post, lowest
// first, with their current prices; nil when nothing can be bought.
func (s *Service) PaywallTiers(ctx context.Context, ws *store.Workspace, readers []gate.Tier) ([]PaywallTier, Payments, error) {
	pay, err := s.Q.WebsitePayments(ctx, ws.ID)
	if err != nil {
		return nil, Payments{}, err
	}
	if pay.Mode == "" || len(readers) == 0 {
		return nil, Payments{}, nil
	}
	rows, err := s.Q.WebsiteTierPrices(ctx, ws.ID)
	if err != nil {
		return nil, Payments{}, err
	}
	byID := map[uuid.UUID]*PaywallTier{}
	order := []uuid.UUID{}
	for _, r := range rows {
		if !slices.ContainsFunc(readers, func(t gate.Tier) bool { return t.ID == r.ID }) || r.Visibility == "archived" {
			continue
		}
		t, ok := byID[r.ID]
		if !ok {
			t = &PaywallTier{ID: r.ID, Name: r.Name, Label: TierLabel(r.Name), TrialDays: int(r.TrialDays), Benefits: r.Benefits}
			byID[r.ID] = t
			order = append(order, r.ID)
		}
		switch r.Period {
		case "month":
			t.Monthly = Money(r.AmountCents, r.Currency)
		case "year":
			t.Yearly = Money(r.AmountCents, r.Currency)
		}
	}
	out := make([]PaywallTier, 0, len(order))
	for i, id := range order {
		t := *byID[id]
		if i > 0 {
			t.Includes = out[0].Label
		}
		out = append(out, t)
	}
	return out, Payments{Provider: "Stripe", Mode: pay.Mode}, nil
}

var moneySymbols = map[string]string{"usd": "$", "eur": "€", "gbp": "£", "cad": "CA$", "aud": "A$", "jpy": "¥", "inr": "₹", "brl": "R$"}

var zeroDecimal = map[string]bool{"jpy": true, "krw": true, "vnd": true, "clp": true, "isk": true, "ugx": true, "xof": true, "xaf": true}

// Money writes a price in minor units the design's way: "$8", "$8.50",
// "¥1,200".
func Money(cents int64, currency string) string {
	cur := strings.ToLower(currency)
	sym, ok := moneySymbols[cur]
	if !ok {
		sym = strings.ToUpper(cur) + " "
	}
	if zeroDecimal[cur] {
		return sym + thousands(cents)
	}
	whole, frac := cents/100, cents%100
	if frac == 0 {
		return sym + thousands(whole)
	}
	return sym + thousands(whole) + fmt.Sprintf(".%02d", frac)
}

func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
