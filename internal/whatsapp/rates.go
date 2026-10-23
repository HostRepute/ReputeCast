package whatsapp

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// The cost (brief §3.9): Meta charges per delivered template message, by
// category and the recipient's country. ReputeCast estimates it from
// Meta's rate card, shows the card's date, and lets the workspace edit it.
// Free-form replies inside the customer service window cost nothing.

// RatesAsOf is the date of the rate card this release carries.
var RatesAsOf = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

// WhatsAppRate is one price of the rate card: per delivered message, in
// USD. Country "*" is every other country.
type WhatsAppRate struct {
	Country  string  `json:"country" pattern:"^([A-Z]{2}|\\*)$"`
	Category string  `json:"category" enum:"marketing,utility,authentication"`
	Price    float64 `json:"price" minimum:"0" maximum:"10"`
}

// DefaultRates is Meta's rate card of 1 Oct 2026 for the markets the app
// lists first, and every other country.
var DefaultRates = []WhatsAppRate{
	{"US", "marketing", 0.0250}, {"US", "utility", 0.0040}, {"US", "authentication", 0.0040},
	{"CA", "marketing", 0.0250}, {"CA", "utility", 0.0040}, {"CA", "authentication", 0.0040},
	{"GB", "marketing", 0.0529}, {"GB", "utility", 0.0220}, {"GB", "authentication", 0.0358},
	{"DE", "marketing", 0.1365}, {"DE", "utility", 0.0550}, {"DE", "authentication", 0.0768},
	{"NL", "marketing", 0.1597}, {"NL", "utility", 0.0500}, {"NL", "authentication", 0.0721},
	{"IE", "marketing", 0.0732}, {"IE", "utility", 0.0300}, {"IE", "authentication", 0.0400},
	{"FR", "marketing", 0.1432}, {"FR", "utility", 0.0300}, {"FR", "authentication", 0.0690},
	{"*", "marketing", 0.0740}, {"*", "utility", 0.0080}, {"*", "authentication", 0.0400},
}

// ensureSettings gives a workspace its WhatsApp settings and the default
// rate card, in q's transaction.
func (s *Service) ensureSettings(ctx context.Context, q *store.Queries, ws *store.Workspace, now time.Time) error {
	if err := q.WhatsAppSettingsEnsure(ctx, store.WhatsAppSettingsEnsureParams{WorkspaceID: ws.ID, RatesAsOf: RatesAsOf, At: now}); err != nil {
		return err
	}
	have, err := q.WhatsAppRates(ctx, ws.ID)
	if err != nil || len(have) > 0 {
		return err
	}
	for _, r := range DefaultRates {
		if err := q.WhatsAppRateInsert(ctx, store.WhatsAppRateInsertParams{WorkspaceID: ws.ID, Country: r.Country, Category: r.Category, Price: r.Price}); err != nil {
			return err
		}
	}
	return nil
}

// rateCard is a workspace's prices by country and category.
type rateCard struct {
	prices map[string]float64
	asOf   time.Time
}

func (s *Service) rateCard(ctx context.Context, q *store.Queries, ws *store.Workspace) rateCard {
	rc := rateCard{prices: map[string]float64{}, asOf: RatesAsOf}
	rows, err := q.WhatsAppRates(ctx, ws.ID)
	if err != nil || len(rows) == 0 {
		for _, r := range DefaultRates {
			rc.prices[r.Country+"/"+r.Category] = r.Price
		}
	} else {
		for _, r := range rows {
			rc.prices[r.Country+"/"+r.Category] = r.Price
		}
	}
	if set, err := q.WhatsAppSettings(ctx, ws.ID); err == nil {
		rc.asOf = set.RatesAsOf
	}
	return rc
}

// price is what one delivered message of a category costs in a country.
func (rc rateCard) price(country, category string) float64 {
	if p, ok := rc.prices[strings.ToUpper(country)+"/"+category]; ok {
		return p
	}
	return rc.prices["*/"+category]
}

// WhatsAppCostCountry is a country's part of an estimate.
type WhatsAppCostCountry struct {
	Country  string  `json:"country"`
	Messages int64   `json:"messages"`
	Price    float64 `json:"price"`
	Amount   float64 `json:"amount"`
}

// WhatsAppCost is an estimate.
type WhatsAppCost struct {
	Amount    float64               `json:"amount"`
	Messages  int64                 `json:"messages"`
	Category  string                `json:"category"`
	RatesAsOf time.Time             `json:"ratesAsOf"`
	ByCountry []WhatsAppCostCountry `json:"byCountry"`
}

// estimate is what sending one template of a category to these countries
// costs.
func (rc rateCard) estimate(category string, countries map[string]int64) WhatsAppCost {
	out := WhatsAppCost{Category: category, RatesAsOf: rc.asOf, ByCountry: []WhatsAppCostCountry{}}
	keys := make([]string, 0, len(countries))
	for k := range countries {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if countries[keys[i]] != countries[keys[j]] {
			return countries[keys[i]] > countries[keys[j]]
		}
		return keys[i] < keys[j]
	})
	for _, c := range keys {
		n := countries[c]
		p := rc.price(c, category)
		out.ByCountry = append(out.ByCountry, WhatsAppCostCountry{Country: c, Messages: n, Price: p, Amount: round4(p * float64(n))})
		out.Messages += n
		out.Amount += p * float64(n)
	}
	out.Amount = float64(int64(out.Amount*100+0.5)) / 100
	return out
}

// WhatsAppRates is the rate card as the Edit rates dialog shows it.
type WhatsAppRates struct {
	AsOf      time.Time      `json:"asOf"`
	Rates     []WhatsAppRate `json:"rates"`
	EditedBy  string         `json:"editedBy,omitempty"`
	EditedAt  *time.Time     `json:"editedAt,omitempty"`
	CanManage bool           `json:"canManage"`
}

// Rates is the workspace's rate card.
func (s *Service) Rates(ctx context.Context, ws *store.Workspace, p *authz.Principal) (WhatsAppRates, error) {
	out := WhatsAppRates{AsOf: RatesAsOf, Rates: []WhatsAppRate{}, CanManage: p.Can(authz.ProvidersManage)}
	rows, err := s.Q.WhatsAppRates(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	if len(rows) == 0 {
		out.Rates = append(out.Rates, DefaultRates...)
	}
	for _, r := range rows {
		out.Rates = append(out.Rates, WhatsAppRate{Country: r.Country, Category: r.Category, Price: r.Price})
	}
	if set, err := s.Q.WhatsAppSettings(ctx, ws.ID); err == nil {
		out.AsOf, out.EditedAt = set.RatesAsOf, set.RatesEditedAt
		if set.RatesEditedBy != nil {
			if u, err := s.Q.GetUser(ctx, *set.RatesEditedBy); err == nil {
				out.EditedBy = u.Name
			}
		}
	}
	return out, nil
}

// WhatsAppRatesInput replaces the rate card.
type WhatsAppRatesInput struct {
	AsOf  string         `json:"asOf" pattern:"^\\d{4}-\\d{2}-\\d{2}$" doc:"The card's date (YYYY-MM-DD)"`
	Rates []WhatsAppRate `json:"rates" minItems:"1" maxItems:"600"`
}

// SaveRates replaces the rate card (logged).
func (s *Service) SaveRates(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WhatsAppRatesInput) (WhatsAppRates, error) {
	l := langOf(ctx)
	asOf, err := time.Parse("2006-01-02", in.AsOf)
	if err != nil || asOf.After(s.now().Add(48*time.Hour)) {
		return WhatsAppRates{}, api.Invalid("body.asOf", mRatesDate.Text(l, nil))
	}
	seen := map[string]bool{}
	hasDefault := false
	for i, r := range in.Rates {
		k := r.Country + "/" + r.Category
		if seen[k] {
			return WhatsAppRates{}, api.Invalid("body.rates["+itoa(i)+"]", mRatesDuplicate.Text(l, nil))
		}
		seen[k] = true
		if r.Country == "*" {
			hasDefault = true
		}
	}
	if !hasDefault {
		return WhatsAppRates{}, api.Invalid("body.rates", mRatesDefault.Text(l, nil))
	}
	now := s.now()
	before := s.rateCard(ctx, s.Q, ws)
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := s.ensureSettings(ctx, q, ws, now); err != nil {
			return err
		}
		if err := q.WhatsAppRatesClear(ctx, ws.ID); err != nil {
			return err
		}
		for _, r := range in.Rates {
			if err := q.WhatsAppRateInsert(ctx, store.WhatsAppRateInsertParams{WorkspaceID: ws.ID, Country: r.Country, Category: r.Category, Price: round4(r.Price)}); err != nil {
				return err
			}
		}
		if err := q.WhatsAppSettingsRatesEdited(ctx, store.WhatsAppSettingsRatesEditedParams{RatesAsOf: asOf, RatesEditedBy: uuidPtr(p.UserID), At: &now,
			WorkspaceID: ws.ID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp_rates.updated", TargetType: "whatsapp_rates", TargetName: "Rate card",
			Before: map[string]any{"as_of": before.asOf.Format("2006-01-02"), "prices": len(before.prices)},
			After:  map[string]any{"as_of": in.AsOf, "prices": len(in.Rates)}})
	})
	if err != nil {
		return WhatsAppRates{}, err
	}
	return s.Rates(ctx, ws, p)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
