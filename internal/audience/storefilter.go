package audience

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
)

// Store customers in the contacts table (design expansion phase 7, X20):
// the "Store customer" and "Total spent" filters under More filters.

// StoreFilter narrows contacts by the connected stores: a store's
// customers, and what they spent there (orders less refunds).
type StoreFilter struct {
	// StoreCustomer is a store's id, or "any": its customers (who ordered,
	// or whom the store's sync knows).
	StoreCustomer string `json:"storeCustomer,omitempty" maxLength:"36"`
	// SpentFrom and SpentTo are a range of the total spent in the stores, in
	// the stores' currency ("100", "250.50"), both included.
	SpentFrom string `json:"spentFrom,omitempty" maxLength:"12"`
	SpentTo   string `json:"spentTo,omitempty" maxLength:"12"`
}

// StoreQuery is the same filter as query parameters.
type StoreQuery struct {
	StoreCustomer string `query:"storeCustomer" maxLength:"36" doc:"A store's id, or any"`
	SpentFrom     string `query:"spentFrom" maxLength:"12" doc:"Total spent in the stores, at least"`
	SpentTo       string `query:"spentTo" maxLength:"12" doc:"Total spent in the stores, at most"`
}

func (q StoreQuery) filter() StoreFilter { return StoreFilter(q) }

// spentArg reads an amount into minor units.
func spentArg(ctx context.Context, s, loc string) (int64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "$")), 64)
	if err != nil || v < 0 || v > 1e12 || math.IsNaN(v) {
		return 0, api.Invalid(loc, say(ctx, mAmount, nil))
	}
	return int64(math.Round(v * 100)), nil
}

// storeFilter appends the store conditions of f.
func (s *Service) storeFilter(ctx context.Context, out *filterSQL, f StoreFilter) error {
	switch v := strings.TrimSpace(f.StoreCustomer); v {
	case "":
	case "any":
		out.add("(EXISTS (SELECT 1 FROM store_customers sc WHERE sc.contact_id = c.id) OR EXISTS (SELECT 1 FROM store_orders so WHERE so.contact_id = c.id AND NOT so.test))")
	default:
		id, err := uuid.Parse(v)
		if err != nil {
			return api.Invalid("storeCustomer", say(ctx, mFilterStore, nil))
		}
		out.add("(EXISTS (SELECT 1 FROM store_customers sc WHERE sc.contact_id = c.id AND sc.store_id = ?) OR EXISTS (SELECT 1 FROM store_orders so WHERE so.contact_id = c.id AND so.store_id = ? AND NOT so.test))", id, id)
	}
	spent := "coalesce((SELECT sum(so.total_cents - so.refunded_cents) FROM store_orders so WHERE so.contact_id = c.id AND NOT so.test AND NOT so.cancelled), 0)"
	if f.SpentFrom != "" {
		n, err := spentArg(ctx, f.SpentFrom, "spentFrom")
		if err != nil {
			return err
		}
		out.add(spent+" >= ?", n)
	}
	if f.SpentTo != "" {
		n, err := spentArg(ctx, f.SpentTo, "spentTo")
		if err != nil {
			return err
		}
		out.add(spent+" <= ?", n)
	}
	return nil
}
