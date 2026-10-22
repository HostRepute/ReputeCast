package automations

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The trigger and goal "Bought a product" (digital products, design
// expansion phase 7, X19): a contact's paid order. Its events
// (internal/products) carry the product as ref ("product:<id>") and the
// order as payload. Only contacts enter: a buyer who isn't a contact has
// no event. The payload's "liquid" object becomes Liquid data of the
// flow's emails ({{ order.product_name }}, {{ order.amount }}), and a
// refund makes the contact leave the flow (ExitRefunded).

// TriggerBoughtProduct is the trigger's kind, and the goal's.
const TriggerBoughtProduct = "bought_product"

// productTriggerMatches decides whether a purchase starts a flow: any
// product, or the one chosen.
func productTriggerMatches(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	return t.ProductID == "" || ev.Ref == "product:"+t.ProductID
}

// eventVars is the Liquid data an event gives the run it starts: its
// payload's "liquid" object.
func eventVars(ev store.ClaimAutomationEventsRow) map[string]any {
	var p struct {
		Liquid map[string]any `json:"liquid"`
	}
	if json.Unmarshal(ev.Payload, &p) != nil || len(p.Liquid) == 0 {
		return nil
	}
	return p.Liquid
}

// keepVars stores the run's Liquid data with an email it queued.
func (e *executor) keepVars(ctx context.Context, msg uuid.UUID) error {
	if len(e.st.Vars) == 0 || msg == uuid.Nil {
		return nil
	}
	b, err := json.Marshal(e.st.Vars)
	if err != nil {
		return err
	}
	return e.q.AutomationMessageVarsInsert(ctx, store.AutomationMessageVarsInsertParams{MessageID: msg, Vars: b})
}

// MessageVars is the Liquid data an automation email was queued with
// (nil: none). The sending engine adds it to the contact's.
func MessageVars(ctx context.Context, q *store.Queries, msg uuid.UUID) map[string]any {
	raw, err := q.AutomationMessageVars(ctx, msg)
	if err != nil {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

// boughtSince checks the condition "Bought a product" for a run: a paid
// order of the product (any product when none is chosen) since the
// contact entered the flow, not refunded in full.
func (e *executor) boughtSince(ctx context.Context, c *AutomationCondition) (bool, error) {
	contact := e.run.ContactID
	return e.q.AutomationContactBought(ctx, store.AutomationContactBoughtParams{WorkspaceID: e.run.WorkspaceID, ContactID: &contact,
		Since: e.run.EnteredAt, ProductID: c.ProductID})
}

// goalRunsBought is the runs of a version whose contact met a "Bought a
// product" goal.
func goalRunsBought(ctx context.Context, q *store.Queries, automation uuid.UUID, version int32, c *AutomationCondition) ([]uuid.UUID, error) {
	return q.AutomationGoalRunsBought(ctx, store.AutomationGoalRunsBoughtParams{AutomationID: automation, Version: version, ProductID: c.ProductID})
}

// ExitRefunded makes a refunded buyer leave the "Bought a product" flows
// of that product, or of any product, they're in. It runs in the
// transaction that records the refund.
func ExitRefunded(ctx context.Context, q *store.Queries, wsID, contactID, productID uuid.UUID, now time.Time) (int, error) {
	rows, err := q.AutomationRunsExitBought(ctx, store.AutomationRunsExitBoughtParams{At: now, WorkspaceID: wsID, ContactID: contactID, ProductID: productID.String()})
	if err != nil {
		return 0, err
	}
	stats := statsBuf{}
	for _, r := range rows {
		stats.at(r.AutomationID, r.Version, r.NodeID, now).Exited++
	}
	return len(rows), stats.flush(ctx, q)
}

// loadProductRefs names the products a trigger or goal can point at.
func loadProductRefs(ctx context.Context, q *store.Queries, wsID uuid.UUID, r *Refs) error {
	rows, err := q.AutomationProductRefs(ctx, wsID)
	if err != nil {
		return err
	}
	if r.Products == nil {
		r.Products = map[string]string{}
	}
	for _, p := range rows {
		r.Products[p.ID.String()] = p.Name
	}
	return nil
}

var mProblemProductDeleted = msg("problem.productDeleted")

// checkProduct validates the product a trigger or goal names (empty: any).
func checkProduct(id string, refs *Refs, bad func(code string, t i18n.Text)) {
	if id == "" || refs == nil || refs.Products == nil {
		return
	}
	if _, ok := refs.Products[id]; !ok {
		bad("missing_ref", txt(mProblemProductDeleted))
	}
}
