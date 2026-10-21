package commerce

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactStores is the contact profile's Store orders card: the stores
// they're a customer of, with their consent there, and their latest
// orders.
func (s *Service) ContactStores(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) (CommerceContactStores, error) {
	out := CommerceContactStores{Stores: []CommerceContactStore{}, Orders: []CommerceContactOrder{}}
	totals := map[uuid.UUID]store.CommerceContactOrderTotalsRow{}
	trows, err := s.Q.CommerceContactOrderTotals(ctx, store.CommerceContactOrderTotalsParams{ContactID: &contactID, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	for _, t := range trows {
		totals[t.StoreID] = t
	}
	seen := map[uuid.UUID]bool{}
	customers, err := s.Q.CommerceContactStoreCustomers(ctx, store.CommerceContactStoreCustomersParams{ContactID: &contactID, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	stores := map[uuid.UUID]store.Store{}
	storeOf := func(id uuid.UUID) store.Store {
		if st, ok := stores[id]; ok {
			return st
		}
		st, _ := s.Q.CommerceStoreByID(ctx, id)
		stores[id] = st
		return st
	}
	for _, c := range customers {
		if seen[c.StoreID] {
			continue
		}
		seen[c.StoreID] = true
		st := storeOf(c.StoreID)
		v := CommerceContactStore{StoreID: c.StoreID.String(), Name: c.StoreName, Platform: c.Platform, Currency: st.Currency, Marketing: c.Marketing, ConsentAt: c.ConsentAt}
		if t, ok := totals[c.StoreID]; ok {
			v.Orders, v.Spent = int(t.Orders), t.Spent
		}
		if c.StoreListID != nil {
			if name, err := s.Q.CommerceListName(ctx, *c.StoreListID); err == nil {
				v.ListName = name
			}
			if m, err := s.Q.CommerceMembership(ctx, store.CommerceMembershipParams{ListID: *c.StoreListID, ContactID: contactID}); err == nil {
				v.OnList = m.Status == "subscribed"
			}
		}
		out.Stores = append(out.Stores, v)
	}
	orders, err := s.Q.CommerceContactOrders(ctx, store.CommerceContactOrdersParams{ContactID: &contactID, WorkspaceID: ws.ID, Lim: 20})
	if err != nil {
		return out, err
	}
	for _, o := range orders {
		if !seen[o.StoreID] {
			// Orders from a store that never synced them as a customer (a
			// guest checkout).
			seen[o.StoreID] = true
			st := storeOf(o.StoreID)
			v := CommerceContactStore{StoreID: o.StoreID.String(), Name: o.StoreName, Platform: o.Platform, Currency: st.Currency, Marketing: "none"}
			if t, ok := totals[o.StoreID]; ok {
				v.Orders, v.Spent = int(t.Orders), t.Spent
			}
			if o.StoreListID != nil {
				if name, err := s.Q.CommerceListName(ctx, *o.StoreListID); err == nil {
					v.ListName = name
				}
			}
			out.Stores = append(out.Stores, v)
		}
		if o.Test {
			continue
		}
		v := CommerceContactOrder{ID: o.ID.String(), StoreID: o.StoreID.String(), Store: o.StoreName, Platform: o.Platform, Name: o.Name, PlacedAt: o.PlacedAt,
			Total: o.TotalCents, Refunded: o.RefundedCents, Currency: o.Currency, Items: itemTitles(o.Items), Accepted: o.AcceptedMarketing}
		switch {
		case o.CampaignName != "":
			v.Credited, v.CreditedKind = o.CampaignName, "campaign"
		case o.AutomationName != "":
			v.Credited, v.CreditedKind = o.AutomationName, "automation"
		}
		out.Orders = append(out.Orders, v)
	}
	return out, nil
}

func itemTitles(raw json.RawMessage) []string {
	var items []orderItem
	_ = json.Unmarshal(raw, &items)
	out := []string{}
	for _, it := range items {
		out = append(out, it.Title)
	}
	return out
}

var _ audience.ActivitySource = (*Service)(nil)

// ContactActivity implements audience.ActivitySource: a contact's store
// orders and the abandoned checkouts a flow looked at, on their timeline.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	p := store.CommerceContactTimelineParams{ContactID: &contactID, WorkspaceID: ws.ID, Lim: int32(limit)}
	if before != nil {
		id, err := uuid.Parse(beforeID)
		if err != nil {
			id = uuid.Max
		}
		p.BeforeAt, p.BeforeID = before, &id
	}
	rows, err := s.Q.CommerceContactTimeline(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]audience.ContactActivityItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, audience.ContactActivityItem{ID: r.ID.String(), Category: "engagement", Type: r.Kind, At: r.At, Subject: r.StoreName,
			Store: &audience.ContactActivityStore{Name: r.StoreName, Platform: r.Platform, Order: r.OrderName, Total: r.TotalCents, Currency: r.Currency,
				Items: itemTitles(r.Items), Accepted: r.AcceptedMarketing, Credited: r.Credited, Flow: r.Flow}})
	}
	n, err := s.Q.CommerceContactTimelineCount(ctx, store.CommerceContactTimelineCountParams{ContactID: &contactID, WorkspaceID: ws.ID})
	if err != nil {
		return nil, 0, err
	}
	return out, n, nil
}
