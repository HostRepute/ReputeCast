package automations

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Store flows from the site tracker (M62, G-29 and G-30; spec publish-sell
// §3.6a). Each event names the automation it was found for, as an
// abandoned checkout's does, because each flow has its own delay, views,
// collections or price threshold:
//
//   - Viewed a product and didn't buy, Added to cart and didn't check out:
//     internal/tracker's tracker.tick finds contacts whose last view (or
//     add-to-cart) of the store's products was AfterHours ago, with no
//     order or checkout since. The event's "liquid" object becomes
//     {{ browse.products }} (title, url, image, price) and its context the
//     products the store block "Viewed products" shows.
//   - Back in stock, Price drop: internal/commerce's commerce.product_changed
//     job, when a product comes back or gets cheaper: people who asked to
//     hear about it (and, when IncludeViewers is on, people who viewed it
//     in the last Days days), or for a price drop people who viewed it or
//     added it to their cart in the last Days days, when the price fell by
//     MinDropPercent or more. {{ stock.product.title }}, url, image, price,
//     and for a price drop {{ stock.previous_price }} and
//     {{ stock.drop_percent }}.
//
// Contacts who aren't active never get an event, and a flow takes the same
// contact again only after 7 days (tracker_trigger_entries): the finders
// decide who enters, so the flow's own re-entry setting doesn't apply.
// Only a flow's first trigger can be one of these.
const (
	TriggerViewedProduct = "viewed_product"
	TriggerAddedToCart   = "added_to_cart"
	TriggerBackInStock   = "back_in_stock"
	TriggerPriceDrop     = "price_drop"
)

// Defaults of the settings a trigger may leave out.
const (
	DefaultBrowseViews    = 1
	DefaultStockDays      = 14
	DefaultMinDropPercent = 10
	// ReentryDays is how long a store flow waits before it takes the same
	// contact again.
	ReentryDays = 7
)

// AutomationBrowseTrigger is what "Viewed a product and didn't buy" and
// "Added to cart and didn't check out" watch.
type AutomationBrowseTrigger struct {
	StoreID string `json:"storeId" maxLength:"40" doc:"The connected store (uuid) whose products the tracked site's events name"`
	// AfterHours: how long after the last view (or add-to-cart) the flow
	// starts, with no order or checkout meanwhile.
	AfterHours int `json:"afterHours" minimum:"1" maximum:"24"`
	// CollectionIDs keeps only products in these store collections (every
	// product when empty).
	CollectionIDs []string `json:"collectionIds,omitempty" maxItems:"20"`
	// MinViews: how many times the contact viewed (or added to their
	// cart) one of the products, 1 to 5.
	MinViews int `json:"minViews" minimum:"1" maximum:"5"`
}

// AutomationStockTrigger is what "Back in stock" and "Price drop" watch.
type AutomationStockTrigger struct {
	StoreID string `json:"storeId" maxLength:"40" doc:"The connected store (uuid) whose products it watches"`
	// Days: how far back a view or an add-to-cart counts.
	Days int `json:"days,omitempty" minimum:"1" maximum:"60" default:"14"`
	// MinDropPercent: a price drop starts the flow when the price fell by
	// at least this share.
	MinDropPercent int `json:"minDropPercent,omitempty" minimum:"1" maximum:"90" default:"10"`
	// IncludeViewers: "Also people who viewed it in the last N days" (Back
	// in stock; off by default).
	IncludeViewers bool `json:"includeViewers,omitempty"`
}

// Hours is the delay, 1 to 24.
func (b *AutomationBrowseTrigger) Hours() int {
	if b == nil {
		return 1
	}
	return min(max(b.AfterHours, 1), 24)
}

// Views is the minimum views, 1 to 5.
func (b *AutomationBrowseTrigger) Views() int {
	if b == nil || b.MinViews < 1 {
		return DefaultBrowseViews
	}
	return min(b.MinViews, 5)
}

// LookbackDays is how far back views count, 1 to 60.
func (s *AutomationStockTrigger) LookbackDays() int {
	if s == nil || s.Days < 1 {
		return DefaultStockDays
	}
	return min(s.Days, 60)
}

// DropPercent is the smallest price drop that counts, 1 to 90.
func (s *AutomationStockTrigger) DropPercent() int {
	if s == nil || s.MinDropPercent < 1 {
		return DefaultMinDropPercent
	}
	return min(s.MinDropPercent, 90)
}

// StoreOf is the store a store flow's trigger watches ("" for other kinds).
func (t *AutomationTrigger) StoreOf() string {
	switch {
	case t == nil:
		return ""
	case isBrowseTrigger(t.Kind) && t.Browse != nil:
		return t.Browse.StoreID
	case isStockTrigger(t.Kind) && t.Stock != nil:
		return t.Stock.StoreID
	}
	return ""
}

func isBrowseTrigger(kind string) bool {
	return kind == TriggerViewedProduct || kind == TriggerAddedToCart
}

func isStockTrigger(kind string) bool { return kind == TriggerBackInStock || kind == TriggerPriceDrop }

// isTrackerTrigger: the store flows of this file.
func isTrackerTrigger(kind string) bool { return isBrowseTrigger(kind) || isStockTrigger(kind) }

var (
	mTriggerViewedProduct = msg("trigger.viewedProduct")
	mTriggerAddedToCart   = msg("trigger.addedToCart")
	mTriggerBackInStock   = msg("trigger.backInStock")
	mTriggerPriceDrop     = msg("trigger.priceDrop")

	mRefBrowseAfter = msg("ref.browseAfter")
	mRefPriceDrop   = msg("ref.priceDrop")

	mProblemBrowseWait        = msg("problem.browseWait")
	mProblemBrowseViews       = msg("problem.browseViews")
	mProblemBrowseCollections = msg("problem.browseCollections")
	mProblemStockDays         = msg("problem.stockDays")
	mProblemDropPercent       = msg("problem.dropPercent")
)

// checkTrackerTrigger checks a store flow's settings: a connected store,
// the delay and views of a browse or cart trigger, the days and the drop
// of a stock trigger.
func checkTrackerTrigger(t *AutomationTrigger, refs *Refs) []configProblem {
	var out []configProblem
	id := t.StoreOf()
	if id == "" {
		return append(out, configProblem{"missing_ref", txt(mProblemPickStore)})
	}
	if refs != nil && refs.Stores != nil {
		if _, ok := refs.Stores[id]; !ok {
			out = append(out, configProblem{"missing_ref", txt(mProblemStoreGone)})
		}
	}
	if b := t.Browse; isBrowseTrigger(t.Kind) {
		if b.AfterHours < 1 || b.AfterHours > 24 {
			out = append(out, configProblem{"invalid_delay", txt(mProblemBrowseWait)})
		}
		if b.MinViews != 0 && (b.MinViews < 1 || b.MinViews > 5) {
			out = append(out, configProblem{"invalid_views", txt(mProblemBrowseViews)})
		}
		if len(b.CollectionIDs) > 20 {
			out = append(out, configProblem{"invalid_collections", txt(mProblemBrowseCollections)})
		}
	}
	if s := t.Stock; isStockTrigger(t.Kind) {
		if s.Days != 0 && (s.Days < 1 || s.Days > 60) {
			out = append(out, configProblem{"invalid_days", txt(mProblemStockDays)})
		}
		if s.MinDropPercent != 0 && (s.MinDropPercent < 1 || s.MinDropPercent > 90) {
			out = append(out, configProblem{"invalid_drop", txt(mProblemDropPercent)})
		}
	}
	return out
}

// trackerEvent is what a store flow's event carries.
type trackerEvent struct {
	Automation string `json:"automation"`
	Store      string `json:"store"`
}

// trackerTriggerMatches: the event is about the trigger's store.
func trackerTriggerMatches(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	var p trackerEvent
	return json.Unmarshal(ev.Payload, &p) == nil && p.Store != "" && p.Store == t.StoreOf()
}

// trackerTriggerAllows: the event names the automation it was found for.
func trackerTriggerAllows(flow uuid.UUID, ev store.ClaimAutomationEventsRow) bool {
	var p trackerEvent
	return json.Unmarshal(ev.Payload, &p) == nil && p.Automation == flow.String()
}

// trackerTriggerRef reads "Acme Merch · after 4 hours", "Acme Merch · 10%
// or more", or the store's name.
func trackerTriggerRef(t *AutomationTrigger, refs *Refs) i18n.Text {
	st := txt(mRefAStore)
	if refs != nil {
		if n, ok := refs.Stores[t.StoreOf()]; ok && n != "" {
			st = name(n)
		}
	}
	switch t.Kind {
	case TriggerViewedProduct, TriggerAddedToCart:
		return txt(mRefBrowseAfter, "store", st, "hours", t.Browse.Hours())
	case TriggerPriceDrop:
		return txt(mRefPriceDrop, "store", st, "percent", t.Stock.DropPercent())
	}
	return st
}
