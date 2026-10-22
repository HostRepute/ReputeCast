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
		key: "abandoned-cart", msg: "cart", steps: 6, preview: []string{NodeTrigger, NodeEmail, NodeDelay, NodeGoal, NodeEmail},
		store:  true,
		emails: []starterEmailDef{{"c1", 1}, {"c2", 1}},
		build: func(storeID string, say func(string) string) AutomationGraph {
			skip := 7
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerAbandonedCheckout, Store: storeID, AfterHours: 1, SkipDays: &skip, Reentry: "always"}}),
					emailNode("n2", "c1", say("c1.subject")),
					fork("n3", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 23, Unit: "hours"}}),
					fork("n4", NodeGoal, AutomationNodeData{Label: say("goal"), Condition: &AutomationCondition{Kind: ConditionPlacedOrder, Store: storeID}}),
					emailNode("n5", "c2", say("c2.subject")),
					fork("n6", NodeExit, AutomationNodeData{Label: say("exit")}),
				},
				Edges: []AutomationEdge{link("n1", "n2", ""), link("n2", "n3", ""), link("n3", "n4", ""), link("n4", "n5", ""), link("n5", "n6", "")},
			}
		},
	})
}

var (
	mProblemStoreNotInWorkspace = msg("problem.storeNotInWorkspace")
	mProblemConnectStore        = msg("problem.connectStore")
)

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
		return "", api.Invalid("body.store", say(langOf(ctx), txt(mProblemStoreNotInWorkspace)))
	}
	if first == "" {
		return "", api.Invalid("body.store", say(langOf(ctx), txt(mProblemConnectStore)))
	}
	return first, nil
}

// hasStore reports whether a store is connected (the store starters).
func (s *Service) hasStore(ctx context.Context, wsID uuid.UUID) bool {
	_, err := templateStore(ctx, s.Q, wsID, "")
	return err == nil
}
