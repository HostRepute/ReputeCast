package segments

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Digital products (design expansion phase 7, X19): "Bought a product",
// any product or one of them, or didn't. The purchases view lists live
// orders (test orders never count); a purchase that was fully refunded
// doesn't count as bought.

func init() {
	builtinFields["bought_product"] = fieldSpec{KindEnum, []string{"bought", "not_bought"}}
	builtinOrder = insertAfter(builtinOrder, "joined", "bought_product")
	extraValidate["bought_product"] = validateBought
	extraCompile["bought_product"] = compileBought
	extraOptions["bought_product"] = boughtOptions
}

// purchasesView is what SQL segments read about purchases: one row per
// live order, with what was paid after refunds.
const purchasesView = `SELECT o.contact_id, o.product_id, coalesce(p.name, o.product_name) AS product, o.status,
           (o.amount_cents - o.refunded_cents) / 100.0 AS paid, o.currency, o.paid_at
    FROM public.product_orders o
    LEFT JOIN public.products p ON p.id = o.product_id
    WHERE o.workspace_id = {ws} AND o.mode = 'live' AND o.contact_id IS NOT NULL`

// AnyProduct is the value for any product.
const AnyProduct = "any"

func validateBought(c SegmentCondition) error {
	if c.Value == AnyProduct {
		return nil
	}
	if _, err := uuid.Parse(c.Value); err != nil {
		return says(mPickProduct)
	}
	return nil
}

func compileBought(t *sqlText, c SegmentCondition) {
	if c.Op == "not_bought" {
		t.raw("NOT ")
	}
	t.raw("EXISTS (SELECT 1 FROM purchases pu WHERE pu.contact_id = c.id AND pu.status <> 'refunded'")
	if c.Value != AnyProduct {
		id, _ := uuid.Parse(c.Value)
		t.raw(" AND pu.product_id = ")
		t.bind(uid(id))
	}
	t.raw(")")
}

// boughtOptions are any product and the workspace's products, published
// first.
func boughtOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]SegmentOption, error) {
	rows, err := q.SegmentProductOptions(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := []SegmentOption{{Value: AnyProduct, Label: mAnyProduct.Text(i18n.LangFrom(ctx), nil)}}
	for _, r := range rows {
		out = append(out, SegmentOption{Value: r.ID.String(), Label: r.Name})
	}
	return out, nil
}
