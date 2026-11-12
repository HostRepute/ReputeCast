package automations

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The site tracker's store triggers (M62): their settings are checked, they
// describe themselves, and an event starts only the flow it names, on its
// store.
func TestStoreFlowTriggers(t *testing.T) {
	storeID := uuid.Must(uuid.NewV7()).String()
	refs := &Refs{Stores: map[string]string{storeID: "Acme Merch"}}
	codes := func(tr AutomationTrigger) []string {
		g := AutomationGraph{Nodes: []AutomationNode{{ID: "t", Type: NodeTrigger, Data: AutomationNodeData{Trigger: &tr}}}}
		var out []string
		for _, p := range checkConfig(g.Nodes[0], refs) {
			out = append(out, p.code)
		}
		return out
	}
	good := []AutomationTrigger{
		{Kind: TriggerViewedProduct, Browse: &AutomationBrowseTrigger{StoreID: storeID, AfterHours: 4, MinViews: 2, CollectionIDs: []string{"501"}}},
		{Kind: TriggerAddedToCart, Browse: &AutomationBrowseTrigger{StoreID: storeID, AfterHours: 1}},
		{Kind: TriggerBackInStock, Stock: &AutomationStockTrigger{StoreID: storeID, IncludeViewers: true}},
		{Kind: TriggerPriceDrop, Stock: &AutomationStockTrigger{StoreID: storeID, Days: 30, MinDropPercent: 15}},
	}
	for _, tr := range good {
		if c := codes(tr); len(c) != 0 {
			t.Errorf("%s: %v", tr.Kind, c)
		}
	}
	bad := map[string]AutomationTrigger{
		"missing_ref":         {Kind: TriggerViewedProduct},
		"invalid_delay":       {Kind: TriggerViewedProduct, Browse: &AutomationBrowseTrigger{StoreID: storeID, AfterHours: 25}},
		"invalid_views":       {Kind: TriggerAddedToCart, Browse: &AutomationBrowseTrigger{StoreID: storeID, AfterHours: 2, MinViews: 6}},
		"invalid_collections": {Kind: TriggerViewedProduct, Browse: &AutomationBrowseTrigger{StoreID: storeID, AfterHours: 2, CollectionIDs: make([]string, 21)}},
		"invalid_days":        {Kind: TriggerPriceDrop, Stock: &AutomationStockTrigger{StoreID: storeID, Days: 61}},
		"invalid_drop":        {Kind: TriggerPriceDrop, Stock: &AutomationStockTrigger{StoreID: storeID, MinDropPercent: 95}},
	}
	for code, tr := range bad {
		if c := codes(tr); !slices.Contains(c, code) {
			t.Errorf("%s: %v, want %s", tr.Kind, c, code)
		}
	}
	gone := AutomationTrigger{Kind: TriggerBackInStock, Stock: &AutomationStockTrigger{StoreID: uuid.Must(uuid.NewV7()).String()}}
	if c := codes(gone); !slices.Contains(c, "missing_ref") {
		t.Errorf("a store that isn't connected: %v", c)
	}
	// Not one of the "also" triggers.
	if slices.Contains(alsoKinds, TriggerViewedProduct) || !IsStoreTrigger(TriggerPriceDrop) {
		t.Error("store flows are first triggers and store triggers")
	}

	if got := TriggerLabel(&good[0], refs); got != "Viewed a product and didn't buy · Acme Merch · after 4 hours" {
		t.Errorf("label %q", got)
	}
	if got := TriggerRef(&good[3], refs); got != "Acme Merch · 15% or more" {
		t.Errorf("price-drop ref %q", got)
	}
	if got := TriggerRef(&good[2], refs); got != "Acme Merch" {
		t.Errorf("back-in-stock ref %q", got)
	}

	flow, other := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	payload, _ := json.Marshal(map[string]any{"automation": flow.String(), "store": storeID})
	ev := store.ClaimAutomationEventsRow{Kind: TriggerViewedProduct, Payload: payload}
	if !triggerMatches(good[0], ev) || !trackerTriggerAllows(flow, ev) || trackerTriggerAllows(other, ev) {
		t.Error("an event starts the flow it names, on its store")
	}
	elsewhere := good[0]
	elsewhere.Browse = &AutomationBrowseTrigger{StoreID: uuid.Must(uuid.NewV7()).String(), AfterHours: 4}
	if triggerMatches(elsewhere, ev) {
		t.Error("another store's event matched")
	}
	if (&AutomationStockTrigger{}).LookbackDays() != DefaultStockDays || (&AutomationStockTrigger{}).DropPercent() != DefaultMinDropPercent || (*AutomationBrowseTrigger)(nil).Views() != 1 {
		t.Error("defaults")
	}
}
