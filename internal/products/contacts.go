package products

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// A contact's purchases (design expansion phase 7, X19): the contact
// profile's Purchases card (what they bought, each download's state, "Send
// a new link") and the timeline's bought and refunded events.

// ContactPurchase is one order of the contact.
type ContactPurchase struct {
	ProductOrderItem
	CoverURL string `json:"coverUrl"`
	CoverAlt string `json:"coverAlt"`
}

// ContactPurchases is the Purchases card.
type ContactPurchases struct {
	Items []ContactPurchase `json:"items"`
	// SpentCents is what the live orders came to after refunds, in
	// Currency (the workspace's).
	SpentCents int64  `json:"spentCents"`
	Currency   string `json:"currency"`
	// CanManage is orders:manage (Send a new link, revoke and restore).
	CanManage bool `json:"canManage"`
}

// ContactPath is a contact of the workspace (exported: Huma reads embedded
// path structs only when their type is exported).
type ContactPath struct {
	api.WorkspaceInput
	Contact string `path:"contact" format:"uuid"`
}

// Purchases lists a contact's orders, newest first; test orders are left
// out.
func (s *Service) Purchases(ctx context.Context, ws *store.Workspace, p *authz.Principal, contactID uuid.UUID) (ContactPurchases, error) {
	out := ContactPurchases{Items: []ContactPurchase{}, Currency: "usd", CanManage: p.Can(authz.OrdersManage)}
	if _, err := s.Q.ProductBuyerContact(ctx, store.ProductBuyerContactParams{WorkspaceID: ws.ID, ID: contactID}); err != nil {
		return out, api.MapError(err, "Contact")
	}
	cid := contactID
	rows, err := s.Q.ProductOrdersOfContact(ctx, store.ProductOrdersOfContactParams{WorkspaceID: ws.ID, ContactID: &cid, Email: ""})
	if err != nil {
		return out, err
	}
	orders := []store.ProductOrder{}
	kept := rows[:0]
	for _, r := range rows {
		if r.ProductOrder.Mode == string(payments.Test) {
			continue
		}
		kept = append(kept, r)
		orders = append(orders, r.ProductOrder)
	}
	states, err := s.downloadStates(ctx, ws, orders)
	if err != nil {
		return out, err
	}
	for i, r := range kept {
		o := r.ProductOrder
		out.Items = append(out.Items, ContactPurchase{ProductOrderItem: orderItem(o, r.CurrentProductName, states[i]), CoverURL: r.CoverUrl, CoverAlt: r.CoverAlt})
		if i == 0 {
			out.Currency = o.Currency
		}
		if o.Currency == out.Currency {
			out.SpentCents += o.AmountCents - o.RefundedCents
		}
	}
	return out, nil
}

// ContactActivity implements audience.ActivitySource: the contact bought a
// product, and a refund. Test orders are left out.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	cid := contactID
	rows, err := s.Q.ProductOrdersOfContact(ctx, store.ProductOrdersOfContactParams{WorkspaceID: ws.ID, ContactID: &cid, Email: ""})
	if err != nil {
		return nil, 0, err
	}
	var all []audience.ContactActivityItem
	for _, r := range rows {
		o := r.ProductOrder
		if o.Mode == string(payments.Test) {
			continue
		}
		name := r.CurrentProductName
		all = append(all, audience.ContactActivityItem{ID: o.ID.String(), Category: "engagement", Type: "bought", At: o.PaidAt, Subject: name,
			Detail: word(lang(ctx), "activity.bought", map[string]any{"amount": moneyText(o.AmountCents, o.Currency), "number": orderNo(o.Number)})})
		if o.RefundedAt != nil && o.RefundedCents > 0 {
			all = append(all, audience.ContactActivityItem{ID: uuid.NewSHA1(o.ID, []byte("refunded")).String(), Category: "engagement", Type: "refunded",
				At: *o.RefundedAt, Subject: name, Detail: word(lang(ctx), "activity.refunded", map[string]any{"amount": moneyText(o.RefundedCents, o.Currency), "number": orderNo(o.Number)})})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].At.Equal(all[j].At) {
			return all[i].At.After(all[j].At)
		}
		return all[i].ID > all[j].ID
	})
	out := []audience.ContactActivityItem{}
	for _, it := range all {
		if before != nil && !it.At.Before(*before) && (!it.At.Equal(*before) || it.ID >= beforeID) {
			continue
		}
		if len(out) < limit {
			out = append(out, it)
		}
	}
	return out, int64(len(all)), nil
}

var _ audience.ActivitySource = (*Service)(nil)

// registerContacts adds the contact profile's Purchases card.
func registerContacts(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "get-contact-purchases", Path: "/workspaces/{workspace}/contacts/{contact}/purchases",
		Summary: "A contact's purchases: products, each download's state and what they spent", Tags: []string{"Products"}, Access: api.Workspace(authz.ProductsView)},
		func(ctx context.Context, in *ContactPath) (*api.Body[ContactPurchases], error) {
			out, err := s.Purchases(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), uuid.MustParse(in.Contact))
			return api.OK(out), err
		})
}

// orderNo is an order's number as messages write it: "1311", never
// grouped like an amount ("1,311").
func orderNo(n int64) string { return strconv.FormatInt(n, 10) }
