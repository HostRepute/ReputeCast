package automations

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Abandoned cart starter (design expansion phase 7, Shopify and
// WooCommerce): a store's abandoned checkout after an hour, an email with
// the checkout's items, a day's wait, the goal "Placed an order" (placing
// it exits the flow), then a second email with the newest products. It's
// offered once a store is connected; internal/commerce writes its emails
// in the store's voice (Service.StoreEmail).
func init() {
	templates = append(templates, templateDef{
		AutomationStarterTemplate: AutomationStarterTemplate{
			Key: "abandoned-cart", Name: "Abandoned cart",
			Summary: "Trigger an hour after a store checkout is abandoned, 2 emails with the cart and new products, goal on the order.",
			Outline: "Abandoned checkout → email with the cart → wait 23 h → goal: placed an order → second email → exit",
			Steps:   6, Preview: []string{NodeTrigger, NodeEmail, NodeDelay, NodeGoal, NodeEmail},
		},
		description: "2 emails over a day",
		store:       true,
		emails: []templateEmail{
			{key: "c1", name: `Abandoned cart · email 1 "You left something behind"`, subject: "You left something behind", preheader: "Your cart is saved.",
				heading: "You left something behind", cta: "Return to checkout", body: []string{"Your cart is saved. Pick up where you left off."}},
			{key: "c2", name: `Email 2 "Still deciding?"`, subject: "Still deciding?", preheader: "Your cart is still saved.",
				heading: "Still deciding?", cta: "Return to checkout", body: []string{"Your cart is saved for a few more days."}},
		},
		build: func(storeID string) AutomationGraph {
			skip := 7
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerAbandonedCheckout, Store: storeID, AfterHours: 1, SkipDays: &skip, Reentry: "always"}}),
					emailNode("n2", "c1", "You left something behind"),
					fork("n3", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 23, Unit: "hours"}}),
					fork("n4", NodeGoal, AutomationNodeData{Label: "Placed an order", Condition: &AutomationCondition{Kind: ConditionPlacedOrder, Store: storeID}}),
					emailNode("n5", "c2", "Still deciding?"),
					fork("n6", NodeExit, AutomationNodeData{Label: "Leave the flow"}),
				},
				Edges: []AutomationEdge{link("n1", "n2", ""), link("n2", "n3", ""), link("n3", "n4", ""), link("n4", "n5", ""), link("n5", "n6", "")},
			}
		},
	})
}

// templateStore is the store a store-triggered starter starts on: the one
// asked for, else the workspace's first connected store.
func templateStore(ctx context.Context, q *store.Queries, wsID uuid.UUID, want string) (string, error) {
	rows, err := q.AutomationStoreRefs(ctx, wsID)
	if err != nil {
		return "", err
	}
	first := ""
	for _, r := range rows {
		if r.Status != "connected" && r.Status != "importing" {
			continue
		}
		if want == "" && first == "" {
			first = r.ID.String()
		}
		if want != "" && r.ID.String() == want {
			return want, nil
		}
	}
	if want != "" {
		return "", api.Invalid("body.store", "That store isn't connected to this workspace.")
	}
	if first == "" {
		return "", api.Invalid("body.store", "Connect a store first: the Abandoned cart starter starts on a store's abandoned checkouts.")
	}
	return first, nil
}

// hasStore reports whether a store is connected (the store starters).
func (s *Service) hasStore(ctx context.Context, wsID uuid.UUID) bool {
	_, err := templateStore(ctx, s.Q, wsID, "")
	return err == nil
}
