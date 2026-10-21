package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// Trigger kinds a store starts (automation_events.kind and the
// automations' trigger kinds).
const (
	TriggerOrderPlaced       = "order_placed"
	TriggerFirstOrder        = "first_order"
	TriggerAbandonedCheckout = "abandoned_checkout"
)

// orderItems is an order's or a checkout's lines as stored.
type orderItem struct {
	ProductID string `json:"productId"`
	Title     string `json:"title"`
	Variant   string `json:"variant,omitempty"`
	Quantity  int    `json:"quantity"`
	Price     int64  `json:"price"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

func itemsOf(items []stores.LineItem) []orderItem {
	out := make([]orderItem, 0, len(items))
	for _, it := range items {
		out = append(out, orderItem{ProductID: it.ProductID, Title: it.Title, Variant: it.Variant, Quantity: it.Quantity, Price: it.Price, ImageURL: it.ImageURL})
	}
	return out
}

// applyOrder stores an order: the buyer's consent first (only a customer
// who accepted marketing joins the list; a contact's orders show on their
// profile; anyone else counts without their email), then the order with
// its refunds, its credit, the checkout it completed, and, for an order
// seen for the first time after the import, its triggers.
func (s *Service) applyOrder(ctx context.Context, q *store.Queries, st store.Store, o stores.Order, source, topic string) (applied, error) {
	now := s.now()
	existing, err := q.CommerceOrderForUpdate(ctx, store.CommerceOrderForUpdateParams{StoreID: st.ID, ExternalID: o.ID})
	isNew := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !isNew {
		return applied{}, err
	}
	if !isNew && existing.UpdatedAtStore != nil && !o.UpdatedAt.IsZero() && o.UpdatedAt.Before(*existing.UpdatedAtStore) {
		// An older version than the one stored (out of order, or replayed).
		return applied{status: "stale", outcome: "older than the stored order"}, nil
	}
	var contactID *uuid.UUID
	var j joined
	accepted := o.Customer != nil && o.Customer.Consent.Accepted()
	if o.Customer != nil {
		if accepted && st.SyncCustomers {
			if j, err = s.consent(ctx, q, st, *o.Customer, o.PlacedAt); err != nil {
				return applied{}, err
			}
			contactID = j.contactID
		} else if c, ok := s.contactOf(ctx, q, st.WorkspaceID, o.Email); ok {
			id := c.ID
			contactID = &id
			if err := s.link(ctx, q, st, *o.Customer, id); err != nil {
				return applied{}, err
			}
		}
	}
	refundIDs, refunded := mergeRefunds(nil, 0, o.Refunds, o.Refunded)
	if !isNew {
		refundIDs, refunded = mergeRefunds(existing.RefundIds, existing.RefundedCents, o.Refunds, o.Refunded)
	}
	landing := stores.StripLanding(o.LandingPage)
	items := mustJSON(itemsOf(o.Items))
	discounts := o.Discounts
	if discounts == nil {
		discounts = []string{}
	}
	if !isNew {
		if err := q.CommerceOrderUpdate(ctx, store.CommerceOrderUpdateParams{ID: existing.ID, ContactID: contactID, AcceptedMarketing: accepted || existing.AcceptedMarketing,
			Currency: o.Currency, SubtotalCents: o.Subtotal, TotalCents: o.Total, RefundedCents: refunded, RefundIds: refundIDs, DiscountCodes: discounts,
			Items: items, LandingPage: orKeep(landing, existing.LandingPage), UtmSource: orKeep(o.UTM.Source, existing.UtmSource),
			UtmMedium: orKeep(o.UTM.Medium, existing.UtmMedium), UtmCampaign: orKeep(o.UTM.Campaign, existing.UtmCampaign),
			UtmContent: orKeep(o.UTM.Content, existing.UtmContent), UtmTerm: orKeep(o.UTM.Term, existing.UtmTerm),
			CheckoutExternalID: orKeep(o.CheckoutToken, existing.CheckoutExternalID), Cancelled: o.Cancelled, Test: o.Test,
			UpdatedAtStore: nonZero(o.UpdatedAt), At: now}); err != nil {
			return applied{}, err
		}
		if refunded > existing.RefundedCents && source != sourceImport {
			if err := s.activity(ctx, q, st, activity{Kind: "refund", Tone: "neutral", Order: existing.Name, Amount: refunded - existing.RefundedCents,
				Currency: o.Currency, OrderID: &existing.ID, Topic: topic}); err != nil {
				return applied{}, err
			}
		}
		return applied{status: "applied", outcome: "order updated"}, nil
	}
	first := false
	if contactID != nil && !o.Test && !o.Cancelled {
		had, err := q.CommerceContactHasOrder(ctx, store.CommerceContactHasOrderParams{StoreID: st.ID, ContactID: contactID, PlacedAt: o.PlacedAt})
		if err != nil {
			return applied{}, err
		}
		first = !had
	}
	custKey := ""
	if o.Customer != nil && (contactID != nil || accepted) {
		custKey = customerKey(*o.Customer, o.Email)
	}
	row, err := q.CommerceOrderInsert(ctx, store.CommerceOrderInsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID, ExternalID: o.ID,
		Name: orderName(o), ContactID: contactID, CustomerExternalID: custKey, AcceptedMarketing: accepted, PlacedAt: o.PlacedAt, Currency: o.Currency,
		SubtotalCents: o.Subtotal, TotalCents: o.Total, RefundedCents: refunded, RefundIds: refundIDs, DiscountCodes: discounts, Items: items,
		LandingPage: landing, UtmSource: o.UTM.Source, UtmMedium: o.UTM.Medium, UtmCampaign: o.UTM.Campaign, UtmContent: o.UTM.Content, UtmTerm: o.UTM.Term,
		CheckoutExternalID: o.CheckoutToken, Cancelled: o.Cancelled, Test: o.Test, FirstOrder: first, Source: source, UpdatedAtStore: nonZero(o.UpdatedAt),
		At: now})
	if err != nil {
		return applied{}, err
	}
	// The import matches past orders after they're all in (its last phase).
	var cr credit
	if source != sourceImport && !o.Test {
		cr = s.match(ctx, q, st, o, contactID)
		if err := s.record(ctx, q, row.ID, cr); err != nil {
			return applied{}, err
		}
	}
	completed, err := q.CommerceCheckoutComplete(ctx, store.CommerceCheckoutCompleteParams{StoreID: st.ID, ExternalID: o.CheckoutToken, ContactID: contactID,
		PlacedAt: o.PlacedAt, OrderID: &row.ID, At: now})
	if err != nil {
		return applied{}, err
	}
	res := applied{status: "applied", outcome: "order", newOrder: true}
	if source == sourceImport {
		return res, nil
	}
	flows := []string{}
	if contactID != nil && st.Status == StatusConnected && !o.Test && !o.Cancelled {
		names, err := s.emitOrder(ctx, q, st, row, o, first)
		if err != nil {
			return applied{}, err
		}
		flows = names
		res.emitted = true
	}
	a := activity{Kind: "order", Tone: "neutral", Order: row.Name, Amount: o.Total, Currency: o.Currency, Accepted: accepted, Contact: contactID != nil,
		OrderID: &row.ID, ContactID: contactID, Topic: topic, Source: source, Window: int(st.AttributionDays), Matched: cr.matched, Flows: flows, At: o.PlacedAt}
	if a.Matched == "" {
		a.Matched = "none"
	}
	if contactID != nil {
		a.Customer = s.contactName(ctx, q, *contactID, o.Customer)
	}
	if j.joined {
		a.Lists = []string{j.listName}
	}
	if !cr.none() {
		a.Credited, a.CreditedKind, a.CreditedStep = s.creditName(ctx, q, cr)
		a.Tone = "ok"
		a.Recovered = len(completed) > 0 && cr.automationID != nil
	}
	if err := s.activity(ctx, q, st, a); err != nil {
		return applied{}, err
	}
	return res, nil
}

func orKeep(v, old string) string {
	if v == "" {
		return old
	}
	return v
}

func orderName(o stores.Order) string {
	if o.Name != "" {
		return o.Name
	}
	return "#" + o.Number
}

// mergeRefunds counts each refund once: by its id when the platform names
// it, else the order's refunded total.
func mergeRefunds(ids []string, refunded int64, refunds []stores.Refund, total int64) ([]string, int64) {
	out := slices.Clone(ids)
	if out == nil {
		out = []string{}
	}
	sum := refunded
	for _, r := range refunds {
		if r.ID == "" || slices.Contains(out, r.ID) {
			continue
		}
		out = append(out, r.ID)
		sum += r.Amount
	}
	if len(refunds) == 0 && total > sum {
		sum = total
	}
	return out, sum
}

// contactName is how the activity log names a contact.
func (s *Service) contactName(ctx context.Context, q *store.Queries, id uuid.UUID, c *stores.Customer) string {
	if c != nil {
		if n := strings.TrimSpace(c.FirstName + " " + c.LastName); n != "" {
			return n
		}
	}
	return ""
}

// emitOrder queues the order's triggers for the automations (Order placed;
// First order for a contact's first order in the store). It returns the
// names of the live automations they start.
func (s *Service) emitOrder(ctx context.Context, q *store.Queries, st store.Store, row store.StoreOrder, o stores.Order, first bool) ([]string, error) {
	products := []string{}
	for _, it := range o.Items {
		if it.ProductID != "" && !slices.Contains(products, it.ProductID) {
			products = append(products, it.ProductID)
		}
	}
	collections := []string{}
	if rows, err := q.CommerceProductCollections(ctx, store.CommerceProductCollectionsParams{StoreID: st.ID, Ids: products}); err == nil {
		for _, r := range rows {
			for _, c := range r.Collections {
				if !slices.Contains(collections, c) {
					collections = append(collections, c)
				}
			}
		}
	}
	payload := map[string]any{"store": st.ID.String(), "order": row.ID.String(), "name": row.Name, "total": o.Total, "currency": o.Currency,
		"products": products, "collections": collections, "first": first, "ctx": map[string]string{"store": st.ID.String(), "order": row.ID.String()}}
	kinds := []string{TriggerOrderPlaced}
	if first {
		kinds = append(kinds, TriggerFirstOrder)
	}
	names := []string{}
	for _, k := range kinds {
		if err := q.CommerceEmitEvent(ctx, store.CommerceEmitEventParams{WorkspaceID: st.WorkspaceID, Kind: k, ContactID: *row.ContactID,
			Ref: "store:" + st.ID.String(), Payload: mustJSON(payload), OccurredAt: o.PlacedAt}); err != nil {
			return nil, err
		}
	}
	if flows, err := q.CommerceStoreFlows(ctx, store.CommerceStoreFlowsParams{WorkspaceID: st.WorkspaceID, Store: st.ID.String()}); err == nil {
		for _, f := range flows {
			var t struct {
				Kind string `json:"kind"`
			}
			_ = json.Unmarshal(f.Trigger, &t)
			if f.Status == "live" && slices.Contains(kinds, t.Kind) {
				names = append(names, f.Name)
			}
		}
	}
	return names, nil
}

// applyRefund adds a refund to its order, once.
func (s *Service) applyRefund(ctx context.Context, q *store.Queries, st store.Store, r stores.Refund) (applied, error) {
	o, err := q.CommerceOrderForUpdate(ctx, store.CommerceOrderForUpdateParams{StoreID: st.ID, ExternalID: r.OrderID})
	if errors.Is(err, pgx.ErrNoRows) {
		return applied{status: "ignored", outcome: "the order isn't synced"}, nil
	}
	if err != nil {
		return applied{}, err
	}
	if slices.Contains(o.RefundIds, r.ID) {
		return applied{status: "ignored", outcome: "refund counted already"}, nil
	}
	ids := append(slices.Clone(o.RefundIds), r.ID)
	if err := q.CommerceOrderRefunds(ctx, store.CommerceOrderRefundsParams{ID: o.ID, RefundedCents: o.RefundedCents + r.Amount, RefundIds: ids, At: s.now()}); err != nil {
		return applied{}, err
	}
	if err := s.activity(ctx, q, st, activity{Kind: "refund", Order: o.Name, Amount: r.Amount, Currency: o.Currency, OrderID: &o.ID, Topic: "refunds/create"}); err != nil {
		return applied{}, err
	}
	return applied{status: "applied", outcome: "refund"}, nil
}

// applyCustomer applies a customer's consent: accepting marketing joins the
// list; unsubscribing in the store leaves it.
func (s *Service) applyCustomer(ctx context.Context, q *store.Queries, st store.Store, c stores.Customer, source string) (applied, error) {
	existing, err := q.CommerceCustomer(ctx, store.CommerceCustomerParams{StoreID: st.ID, ExternalID: c.ID})
	found := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return applied{}, err
	}
	if found && existing.UpdatedAtStore != nil && !c.UpdatedAt.IsZero() && c.UpdatedAt.Before(*existing.UpdatedAtStore) {
		return applied{status: "stale", outcome: "older than the stored customer"}, nil
	}
	if c.Consent.Accepted() {
		if !st.SyncCustomers {
			return applied{status: "ignored", outcome: "customers don't sync"}, nil
		}
		if _, err := s.consent(ctx, q, st, c, s.now()); err != nil {
			return applied{}, err
		}
		return applied{status: "applied", outcome: "accepted marketing"}, nil
	}
	contactID := (*uuid.UUID)(nil)
	if found && existing.ContactID != nil {
		contactID = existing.ContactID
	} else if ct, ok := s.contactOf(ctx, q, st.WorkspaceID, c.Email); ok {
		id := ct.ID
		contactID = &id
	}
	if contactID == nil {
		// Didn't accept marketing and isn't a contact: nothing is kept.
		return applied{status: "applied", outcome: "not kept"}, nil
	}
	if found && existing.Marketing == stores.ConsentSubscribed && c.Consent.State == stores.ConsentUnsubscribed {
		at := s.now()
		if c.Consent.UpdatedAt != nil {
			at = *c.Consent.UpdatedAt
		}
		left, err := s.leave(ctx, q, st, *contactID, at)
		if err != nil {
			return applied{}, err
		}
		if left {
			name := strings.TrimSpace(existing.FirstName + " " + existing.LastName)
			lists := []string{}
			if st.ListID != nil {
				if n, err := q.CommerceListName(ctx, *st.ListID); err == nil {
					lists = append(lists, n)
				}
			}
			if err := s.activity(ctx, q, st, activity{Kind: "customer_unsubscribed", Customer: name, ContactID: contactID, Lists: lists,
				Topic: "customers/update", Source: source}); err != nil {
				return applied{}, err
			}
		}
	}
	// Keep the link only: their email and name are the contact's.
	_, err = q.CommerceCustomerUpsert(ctx, store.CommerceCustomerUpsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID, ExternalID: c.ID,
		ContactID: contactID, Marketing: c.Consent.State, UpdatedAtStore: nonZero(c.UpdatedAt), At: s.now()})
	if err != nil {
		return applied{}, err
	}
	return applied{status: "applied", outcome: "customer " + c.Consent.State}, nil
}

// applyCheckout stores a checkout with an email: a candidate abandoned
// checkout until an order completes it.
func (s *Service) applyCheckout(ctx context.Context, q *store.Queries, st store.Store, ch stores.Checkout, source string) (applied, error) {
	if ch.ID == "" {
		return applied{status: "ignored", outcome: "checkout without a token"}, nil
	}
	accepted := ch.AcceptsMarketing || (ch.Customer != nil && ch.Customer.Consent.Accepted())
	var contactID *uuid.UUID
	if accepted && st.SyncCustomers && ch.Customer != nil {
		c := *ch.Customer
		if c.Email == "" {
			c.Email = ch.Email
		}
		if !c.Consent.Accepted() {
			c.Consent = stores.Consent{State: stores.ConsentSubscribed, Level: "single_opt_in"}
		}
		j, err := s.consent(ctx, q, st, c, ch.UpdatedAt)
		if err != nil {
			return applied{}, err
		}
		contactID = j.contactID
	} else if ct, ok := s.contactOf(ctx, q, st.WorkspaceID, ch.Email); ok {
		id := ct.ID
		contactID = &id
	}
	custKey := ""
	if ch.Customer != nil && contactID != nil {
		custKey = customerKey(*ch.Customer, ch.Email)
	}
	created, updated := ch.CreatedAt, ch.UpdatedAt
	if created.IsZero() {
		created = s.now()
	}
	if updated.IsZero() {
		updated = created
	}
	_, err := q.CommerceCheckoutUpsert(ctx, store.CommerceCheckoutUpsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID, ExternalID: ch.ID,
		ContactID: contactID, CustomerExternalID: custKey, HasEmail: ch.Email != "", AcceptedMarketing: accepted, Items: mustJSON(itemsOf(ch.Items)),
		SubtotalCents: ch.Subtotal, TotalCents: ch.Total, Currency: ch.Currency, RecoveryUrl: ch.RecoveryURL, CreatedAtStore: created,
		UpdatedAtStore: updated, CompletedAt: ch.CompletedAt, Source: source, At: s.now()})
	if errors.Is(err, pgx.ErrNoRows) {
		return applied{status: "stale", outcome: "older than the stored checkout"}, nil
	}
	if err != nil {
		return applied{}, err
	}
	return applied{status: "applied", outcome: "checkout"}, nil
}

// applyProduct stores a product. Webhooks don't carry collections, so they
// keep the stored ones.
func (s *Service) applyProduct(ctx context.Context, q *store.Queries, st store.Store, p stores.Product, keepCollections bool) (applied, error) {
	status := p.Status
	switch status {
	case "active", "draft", "archived":
	default:
		status = "active"
	}
	cur := p.Currency
	if cur == "" {
		cur = st.Currency
	}
	url := p.URL
	if url == "" && p.Handle != "" && st.Platform == Shopify {
		host := st.PublicDomain
		if host == "" {
			host = st.Domain
		}
		url = "https://" + host + "/products/" + p.Handle
	}
	cols := p.Collections
	if cols == nil {
		cols = []string{}
	}
	var inv *int32
	if p.Inventory != nil {
		inv = ptr(int32(*p.Inventory))
	}
	_, err := q.CommerceProductUpsert(ctx, store.CommerceProductUpsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID, ExternalID: p.ID,
		Title: p.Title, Handle: p.Handle, Url: url, ImageUrl: p.ImageURL, ImageAlt: p.ImageAlt, PriceCents: p.Price, CompareAtCents: p.CompareAt,
		Currency: cur, Available: p.Available, Inventory: inv, Status: status, Collections: cols, CreatedAtStore: nonZero(p.CreatedAt),
		UpdatedAtStore: nonZero(p.UpdatedAt), At: s.now(), KeepCollections: keepCollections})
	if errors.Is(err, pgx.ErrNoRows) {
		return applied{status: "stale", outcome: "older than the stored product"}, nil
	}
	if err != nil {
		return applied{}, err
	}
	return applied{status: "applied", outcome: "product"}, nil
}

// revoke marks a store disconnected because it refused ReputeCast (the app
// was uninstalled, the token revoked): its automations pause and say why,
// and the people who manage integrations are told.
func (s *Service) revoke(ctx context.Context, st store.Store, reason string, at time.Time) error {
	if at.IsZero() {
		at = s.now()
	}
	var flows []string
	var revoked bool
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		st2, err := q.CommerceStoreRevoked(ctx, store.CommerceStoreRevokedParams{ID: st.ID, Reason: reason, At: at})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		revoked = true
		if flows, err = s.pauseFlows(ctx, q, st2); err != nil {
			return err
		}
		if err := s.activity(ctx, q, st2, activity{Kind: "revoked", Tone: "bad", Reason: reason, Flows: flows, At: at}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &st.WorkspaceID, Action: "store.revoked", TargetType: "store", TargetID: st.ID.String(),
			TargetName: st.Name, Reason: reason, Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"},
			Metadata: map[string]any{"paused": flows}}); err != nil {
			return err
		}
		return s.notifyRevoked(ctx, tx, st2, reason, at, len(flows))
	})
	if err != nil || !revoked {
		return err
	}
	return nil
}

// pauseFlows pauses the live automations a store's triggers start, with the
// reason they show.
func (s *Service) pauseFlows(ctx context.Context, q *store.Queries, st store.Store) ([]string, error) {
	flows, err := q.CommerceStoreFlows(ctx, store.CommerceStoreFlowsParams{WorkspaceID: st.WorkspaceID, Store: st.ID.String()})
	if err != nil {
		return nil, err
	}
	reason := mFlowPaused.Text(i18n.Default, i18n.Args{"store": st.Name})
	out := []string{}
	for _, f := range flows {
		if f.Status != "live" {
			continue
		}
		n, err := q.CommercePauseFlow(ctx, store.CommercePauseFlowParams{ID: f.ID, At: s.now(), Reason: reason})
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}
		out = append(out, f.Name)
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &st.WorkspaceID, Action: "automation.pause", TargetType: "automation", TargetID: f.ID.String(),
			TargetName: f.Name, Reason: reason, Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// resumeFlows resumes the automations a store's disconnection paused.
func (s *Service) resumeFlows(ctx context.Context, q *store.Queries, st store.Store) error {
	flows, err := q.CommerceStoreFlows(ctx, store.CommerceStoreFlowsParams{WorkspaceID: st.WorkspaceID, Store: st.ID.String()})
	if err != nil {
		return err
	}
	reason := mFlowPaused.Text(i18n.Default, i18n.Args{"store": st.Name})
	for _, f := range flows {
		n, err := q.CommerceResumeFlow(ctx, store.CommerceResumeFlowParams{ID: f.ID, Reason: reason})
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &st.WorkspaceID, Action: "automation.resume", TargetType: "automation", TargetID: f.ID.String(),
			TargetName: f.Name, Reason: "store reconnected", Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}}); err != nil {
			return err
		}
	}
	return nil
}

// notifyRevoked tells the people who manage integrations.
func (s *Service) notifyRevoked(ctx context.Context, tx pgx.Tx, st store.Store, reason string, at time.Time, flows int) error {
	ws, err := store.New(tx).GetWorkspace(ctx, st.WorkspaceID)
	if err != nil {
		return err
	}
	l := i18n.Default
	date := at.In(location(ws.Timezone)).Format("2 Jan 15:04")
	return notify.ToMembersWith(ctx, tx, st.WorkspaceID, authz.IntegrationsManage, notify.Notification{
		Kind: KindStoreProblem, Severity: "bad",
		Title: mNotifyDiscTitle.Text(l, i18n.Args{"store": st.Name}),
		Body:  mNotifyDiscBody.Text(l, i18n.Args{"platform": platformName(st.Platform), "date": date, "reason": reason, "flows": flows}),
		Link:  "/w/" + ws.Slug + "/settings/integrations/stores/" + st.ID.String(),
		Data:  map[string]any{"store": st.ID.String(), "problem": "disconnected"},
	})
}

// KindStoreProblem is the notification kind: a store disconnected, or its
// webhooks missing.
const KindStoreProblem = notify.KindStoreProblem

func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil && tz != "" {
		return loc
	}
	return time.UTC
}
