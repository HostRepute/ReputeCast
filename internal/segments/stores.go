package segments

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Store conditions (design expansion phase 7, X20): a connected store's
// customers, their orders (how many, the total spent, the last order's
// date, a product or a collection bought) and whether they accepted
// marketing in the store. They read the store tables the sync fills; test
// and cancelled orders don't count.

func init() {
	opLabels["at_least"] = "is at least"
	opLabels["at_most"] = "is at most"
	opLabels["between"] = "is between"
	builtinFields["store_customer"] = fieldSpec{"Store customer", KindEnum, []string{"is", "is_not"}}
	builtinFields["store_orders"] = fieldSpec{"Store orders", KindNumber, scoreOps}
	builtinFields["store_spent"] = fieldSpec{"Total spent", KindNumber, scoreOps}
	builtinFields["store_last_order"] = fieldSpec{"Last store order", KindPeriod, []string{"in_last", "not_in_last", "never"}}
	builtinFields["store_bought"] = fieldSpec{"Bought from a store", KindEnum, []string{"is", "is_not"}}
	builtinFields["store_marketing"] = fieldSpec{"Accepted marketing in a store", KindEnum, []string{"is", "is_not"}}
	builtinOrder = append(builtinOrder, "store_customer", "store_orders", "store_spent", "store_last_order", "store_bought", "store_marketing")
	extraValidate["store_customer"] = validateStoreRef
	extraValidate["store_marketing"] = validateStoreRef
	extraValidate["store_orders"] = validateCount
	extraValidate["store_spent"] = validateAmount
	extraValidate["store_last_order"] = validatePeriod
	extraValidate["store_bought"] = validateStoreBought
	extraCompile["store_customer"] = compileStoreCustomer
	extraCompile["store_marketing"] = compileStoreMarketing
	extraCompile["store_orders"] = compileStoreOrders
	extraCompile["store_spent"] = compileStoreSpent
	extraCompile["store_last_order"] = compileLastOrder
	extraCompile["store_bought"] = compileStoreBought
	extraOptions["store_customer"] = storeOptions
	extraOptions["store_marketing"] = storeOptions
	extraOptions["store_bought"] = storeBoughtOptions
}

// ordersOf is the contact's counted orders.
const ordersOf = "public.store_orders so WHERE so.contact_id = c.id AND NOT so.test AND NOT so.cancelled"

func validateStoreRef(c SegmentCondition) error {
	if c.Value == "any" {
		return nil
	}
	if _, err := uuid.Parse(c.Value); err != nil {
		return problemf("Pick a store.")
	}
	return nil
}

func validateCount(c SegmentCondition) error {
	return validateRange(c, func(s string) bool {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		return err == nil && n >= 0 && n <= 1000000
	}, "The number of orders needs a whole number.")
}

func validateAmount(c SegmentCondition) error {
	return validateRange(c, func(s string) bool {
		v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return err == nil && v >= 0 && v <= 1e12
	}, "The total spent needs an amount, like 100 or 250.50.")
}

func validateRange(c SegmentCondition, ok func(string) bool, msg string) error {
	if !ok(c.Value) {
		return problemf("%s", msg)
	}
	if c.Op == "between" {
		if !ok(c.To) {
			return problemf("Enter both ends of the range.")
		}
		a, _ := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
		b, _ := strconv.ParseFloat(strings.TrimSpace(c.To), 64)
		if b < a {
			return problemf("The range ends below where it starts. Swap the two numbers.")
		}
	}
	return nil
}

func validatePeriod(c SegmentCondition) error {
	if c.Op == "never" {
		return nil
	}
	if c.Amount < 1 || c.Amount > 3650 {
		return problemf("Last store order needs a number of days or weeks between 1 and 3650.")
	}
	if c.Unit != "days" && c.Unit != "weeks" {
		return problemf("Last store order needs days or weeks.")
	}
	return nil
}

// boughtRef reads "product:<store>:<id>" or "collection:<store>:<id>".
func boughtRef(v string) (kind string, storeID uuid.UUID, id string, ok bool) {
	kind, rest, found := strings.Cut(v, ":")
	if !found || (kind != "product" && kind != "collection") {
		return "", uuid.Nil, "", false
	}
	sid, id, found := strings.Cut(rest, ":")
	u, err := uuid.Parse(sid)
	if !found || err != nil || id == "" || len(id) > 64 {
		return "", uuid.Nil, "", false
	}
	return kind, u, id, true
}

func validateStoreBought(c SegmentCondition) error {
	if _, _, _, ok := boughtRef(c.Value); !ok {
		return problemf("Pick a product or a collection.")
	}
	return nil
}

func compileStoreCustomer(t *sqlText, c SegmentCondition) {
	if c.Op == "is_not" {
		t.raw("NOT ")
	}
	if c.Value == "any" {
		t.raw("(EXISTS (SELECT 1 FROM public.store_customers sc WHERE sc.contact_id = c.id)" + indent +
			"OR EXISTS (SELECT 1 FROM public.store_orders so WHERE so.contact_id = c.id AND NOT so.test))")
		return
	}
	id, _ := uuid.Parse(c.Value)
	t.raw("(EXISTS (SELECT 1 FROM public.store_customers sc WHERE sc.contact_id = c.id AND sc.store_id = ")
	t.bind(uid(id))
	t.raw(")" + indent + "OR EXISTS (SELECT 1 FROM public.store_orders so WHERE so.contact_id = c.id AND NOT so.test AND so.store_id = ")
	t.bind(uid(id))
	t.raw("))")
}

func compileStoreMarketing(t *sqlText, c SegmentCondition) {
	if c.Op == "is_not" {
		t.raw("NOT ")
	}
	t.raw("EXISTS (SELECT 1 FROM public.store_customers sc WHERE sc.contact_id = c.id AND sc.marketing = 'subscribed'")
	if c.Value != "any" {
		id, _ := uuid.Parse(c.Value)
		t.raw(" AND sc.store_id = ")
		t.bind(uid(id))
	}
	t.raw(")")
}

// compareRange renders "expr >= a", "expr <= a" or "expr BETWEEN a AND b".
func compareRange(t *sqlText, expr string, c SegmentCondition, scale float64) {
	a, _ := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
	switch c.Op {
	case "at_most":
		t.raw(expr + " <= ")
		t.bind(num(a * scale))
	case "between":
		b, _ := strconv.ParseFloat(strings.TrimSpace(c.To), 64)
		t.raw(expr + " BETWEEN ")
		t.bind(num(a * scale))
		t.raw(" AND ")
		t.bind(num(b * scale))
	default:
		t.raw(expr + " >= ")
		t.bind(num(a * scale))
	}
}

func compileStoreOrders(t *sqlText, c SegmentCondition) {
	compareRange(t, "(SELECT count(*) FROM "+ordersOf+")", c, 1)
}

func compileStoreSpent(t *sqlText, c SegmentCondition) {
	// Amounts are stored in minor units.
	compareRange(t, "coalesce((SELECT sum(so.total_cents - so.refunded_cents) FROM "+ordersOf+"), 0)", c, 100)
}

func compileLastOrder(t *sqlText, c SegmentCondition) {
	switch c.Op {
	case "never":
		t.raw("NOT EXISTS (SELECT 1 FROM " + ordersOf + ")")
	case "not_in_last":
		t.raw("NOT EXISTS (SELECT 1 FROM " + ordersOf + " AND so.placed_at >= " + interval(c) + ")")
	default:
		t.raw("EXISTS (SELECT 1 FROM " + ordersOf + " AND so.placed_at >= " + interval(c) + ")")
	}
}

func compileStoreBought(t *sqlText, c SegmentCondition) {
	kind, storeID, id, _ := boughtRef(c.Value)
	if c.Op == "is_not" {
		t.raw("NOT ")
	}
	t.raw("EXISTS (SELECT 1 FROM " + ordersOf + " AND so.store_id = ")
	t.bind(uid(storeID))
	t.raw(indent + "AND EXISTS (SELECT 1 FROM jsonb_array_elements(so.items) it WHERE ")
	if kind == "product" {
		t.raw("it ->> 'productId' = ")
		t.bind(str(id))
		t.raw("))")
		return
	}
	t.raw("it ->> 'productId' IN (SELECT p.external_id FROM public.store_products p WHERE p.store_id = ")
	t.bind(uid(storeID))
	t.raw(" AND ")
	t.bind(str(id))
	t.raw(" = ANY (p.collections))))")
}

// storeOptions are the stores, and "Any store".
func storeOptions(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]SegmentOption, error) {
	out := []SegmentOption{{Value: "any", Label: "Any store"}}
	rows, err := q.CommerceStores(ctx, ws)
	if err != nil {
		return nil, err
	}
	for _, st := range rows {
		if st.Status == "connecting" {
			continue
		}
		out = append(out, SegmentOption{Value: st.ID.String(), Label: st.Name})
	}
	return out, nil
}

// storeBoughtOptions are each store's collections, then its products.
func storeBoughtOptions(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]SegmentOption, error) {
	out := []SegmentOption{}
	rows, err := q.CommerceStores(ctx, ws)
	if err != nil {
		return nil, err
	}
	for _, st := range rows {
		if st.Status == "connecting" {
			continue
		}
		cols, err := q.CommerceCollections(ctx, st.ID)
		if err != nil {
			return nil, err
		}
		for _, col := range cols {
			out = append(out, SegmentOption{Value: "collection:" + st.ID.String() + ":" + col.ExternalID, Label: st.Name + " · " + col.Title + " (collection)"})
		}
		products, err := q.CommerceProducts(ctx, store.CommerceProductsParams{StoreID: st.ID, Lim: 500})
		if err != nil {
			return nil, err
		}
		for _, p := range products {
			out = append(out, SegmentOption{Value: "product:" + st.ID.String() + ":" + p.ExternalID, Label: st.Name + " · " + p.Title})
		}
	}
	return out, nil
}
