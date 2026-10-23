package automations

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Store triggers and the "Placed an order" goal (design expansion phase 7,
// Shopify and WooCommerce; internal/commerce writes the events).
//
//   - Order placed: a contact's order, filtered by product, collection, a
//     minimum total and "first order only".
//   - First order: a contact's first order in the store.
//   - Abandoned checkout: a checkout with no order after 1 to 24 hours.
//     Only contacts who accepted marketing in the store enter (a filter
//     that can't be removed: internal/commerce only sends their events),
//     and the skip rule keeps out anyone who got an abandoned-cart email
//     in the last 7 days. Placing the order exits the flow through the
//     "Placed an order" goal.
const (
	TriggerOrderPlaced       = "order_placed"
	TriggerFirstOrder        = "first_order"
	TriggerAbandonedCheckout = "abandoned_checkout"

	// ConditionPlacedOrder is the goal "Placed an order".
	ConditionPlacedOrder = "placed_order"
)

// AutomationRevenue is the store orders credited to a flow's emails.
type AutomationRevenue struct {
	Orders   int64  `json:"orders"`
	Revenue  int64  `json:"revenue" doc:"In minor units of the currency"`
	Currency string `json:"currency"`
}

// Kick wakes the engine soon (a store event was queued).
func (s *Service) Kick(ctx context.Context) { s.kick(ctx, nil) }

func isStoreTrigger(kind string) bool {
	return kind == TriggerOrderPlaced || kind == TriggerFirstOrder || kind == TriggerAbandonedCheckout
}

// IsStoreTrigger reports whether a trigger kind is a store's.
func IsStoreTrigger(kind string) bool { return isStoreTrigger(kind) }

var (
	mProblemPickStore    = msg("problem.pickStore")
	mProblemStoreGone    = msg("problem.storeGone")
	mProblemCheckoutWait = msg("problem.checkoutWait")
)

// checkStoreTrigger checks a store trigger's settings.
func checkStoreTrigger(t *AutomationTrigger, refs *Refs) []configProblem {
	var out []configProblem
	if t.Store == "" {
		return append(out, configProblem{"missing_ref", txt(mProblemPickStore)})
	}
	if refs != nil && refs.Stores != nil {
		if _, ok := refs.Stores[t.Store]; !ok {
			out = append(out, configProblem{"missing_ref", txt(mProblemStoreGone)})
		}
	}
	if t.Kind == TriggerAbandonedCheckout && t.AfterHours != 0 && (t.AfterHours < 1 || t.AfterHours > 24) {
		out = append(out, configProblem{"invalid_delay", txt(mProblemCheckoutWait)})
	}
	return out
}

// storeEvent is what a store's trigger event carries.
type storeEvent struct {
	Automation  string   `json:"automation"`
	Store       string   `json:"store"`
	Total       int64    `json:"total"`
	Products    []string `json:"products"`
	Collections []string `json:"collections"`
	First       bool     `json:"first"`
}

// storeTriggerMatches decides whether a store event starts a flow.
func storeTriggerMatches(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	var p storeEvent
	if json.Unmarshal(ev.Payload, &p) != nil || p.Store == "" || p.Store != t.Store {
		return false
	}
	if t.Kind != TriggerOrderPlaced {
		return true
	}
	if len(t.Products) > 0 && !slices.ContainsFunc(p.Products, func(id string) bool { return slices.Contains(t.Products, id) }) {
		return false
	}
	if len(t.Collections) > 0 && !slices.ContainsFunc(p.Collections, func(id string) bool { return slices.Contains(t.Collections, id) }) {
		return false
	}
	if t.MinTotal > 0 && p.Total < t.MinTotal {
		return false
	}
	return !t.FirstOnly || p.First
}

// storeTriggerAllows: an abandoned checkout's event names the automation it
// was found for (each has its own delay and skip rule).
func storeTriggerAllows(t AutomationTrigger, flow uuid.UUID, ev store.ClaimAutomationEventsRow) bool {
	if t.Kind != TriggerAbandonedCheckout {
		return true
	}
	var p storeEvent
	_ = json.Unmarshal(ev.Payload, &p)
	return p.Automation == flow.String()
}

// eventContext reads what an event passes to the flow's emails.
func eventContext(raw json.RawMessage) map[string]string {
	var p struct {
		Ctx map[string]string `json:"ctx"`
	}
	if json.Unmarshal(raw, &p) != nil || len(p.Ctx) == 0 {
		return nil
	}
	return p.Ctx
}

func storeOf(c *AutomationCondition) *uuid.UUID {
	if c == nil || c.Store == "" {
		return nil
	}
	id, err := uuid.Parse(c.Store)
	if err != nil {
		return nil
	}
	return &id
}

// placedOrder checks the goal now: the contact ordered since entering.
func placedOrder(ctx context.Context, q *store.Queries, ws, contact uuid.UUID, c *AutomationCondition, since time.Time) (bool, error) {
	return q.AutomationContactOrdered(ctx, store.AutomationContactOrderedParams{WorkspaceID: ws, ContactID: &contact, Since: since, StoreID: storeOf(c)})
}

// goalRunsOrdered finds the runs whose contact ordered since entering:
// placing the order exits the flow wherever the contact is.
func goalRunsOrdered(ctx context.Context, q *store.Queries, automation uuid.UUID, version int32, c *AutomationCondition) ([]uuid.UUID, error) {
	rows, err := q.AutomationGoalRunsOrdered(ctx, store.AutomationGoalRunsOrderedParams{AutomationID: automation, Version: version, StoreID: storeOf(c)})
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out, nil
}

// loadStoreRefs adds the workspace's connected stores to refs.
func loadStoreRefs(ctx context.Context, q *store.Queries, ws uuid.UUID, refs *Refs) {
	refs.Stores = map[string]string{}
	rows, err := q.AutomationStoreRefs(ctx, ws)
	if err != nil {
		return
	}
	for _, r := range rows {
		if r.Status == "connected" || r.Status == "importing" {
			refs.Stores[r.ID.String()] = r.Name
		}
	}
}
