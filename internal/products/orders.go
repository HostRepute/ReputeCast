package products

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// ProductDownloadState is an order's download link as the Download column
// and the order panel show it.
type ProductDownloadState struct {
	// State: delivered (the email arrived), sent (it left), pending (being
	// sent), failed, expired, used_up, revoked (by a person or a refund),
	// revoked_auto (a dispute, as the product says).
	State       string     `json:"state" enum:"pending,sent,delivered,failed,expired,used_up,revoked,revoked_auto"`
	Provider    string     `json:"provider"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	Error       string     `json:"error,omitempty"`
	Used        int32      `json:"used"`
	Allowed     int32      `json:"allowed"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	// NewLink: the current link replaced an expired one, sent by a person
	// (or the buyer from the expired link's page).
	NewLink   bool       `json:"newLink"`
	NewLinkAt *time.Time `json:"newLinkAt,omitempty"`
	NewLinkBy string     `json:"newLinkBy"`
	// The revocation: when, by whom ("" for the system) and why.
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
	RevokedBy    string     `json:"revokedBy"`
	RevokeReason string     `json:"revokeReason" enum:",manual,refund,dispute"`
}

// ProductOrderItem is one row of the orders table.
type ProductOrderItem struct {
	ID             string               `json:"id"`
	Number         int64                `json:"number"`
	PaidAt         time.Time            `json:"paidAt"`
	BuyerEmail     string               `json:"buyerEmail"`
	ProductID      string               `json:"productId"`
	Product        string               `json:"product" doc:"The product's short name (Field Guide)"`
	ProductName    string               `json:"productName"`
	AmountCents    int64                `json:"amountCents"`
	Currency       string               `json:"currency"`
	PayWhatYouWant bool                 `json:"payWhatYouWant"`
	Status         string               `json:"status" enum:"paid,refunded,partially_refunded,disputed"`
	Test           bool                 `json:"test"`
	RefundedCents  int64                `json:"refundedCents"`
	RefundedAt     *time.Time           `json:"refundedAt,omitempty"`
	DisputeStatus  string               `json:"disputeStatus" enum:",open,won,lost"`
	DisputeReason  string               `json:"disputeReason"`
	DisputedAt     *time.Time           `json:"disputedAt,omitempty"`
	Download       ProductDownloadState `json:"download"`
}

// ProductOrderList is a page of orders.
type ProductOrderList struct {
	Items    []ProductOrderItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	// Products names the products the filter offers.
	Products []ProductCount `json:"products"`
}

// OrdersQuery filters orders.
type OrdersQuery struct {
	Q        string
	Product  string
	Status   string
	Period   string
	Page     int
	PageSize int
}

func (s *Service) since(period string) time.Time {
	now := s.now()
	switch period {
	case "7d":
		return now.AddDate(0, 0, -7)
	case "90d":
		return now.AddDate(0, 0, -90)
	case "all":
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return now.AddDate(0, 0, -30)
}

// Orders lists orders, newest first.
func (s *Service) Orders(ctx context.Context, ws *store.Workspace, in OrdersQuery) (ProductOrderList, error) {
	if in.PageSize <= 0 {
		in.PageSize = 10
	}
	if in.Page <= 0 {
		in.Page = 1
	}
	var product *uuid.UUID
	if id, err := uuid.Parse(in.Product); err == nil {
		product = &id
	}
	q := strings.TrimSpace(in.Q)
	since := s.since(in.Period)
	in.Status = s.orderStatus(ctx, ws, in.Status)
	out := ProductOrderList{Items: []ProductOrderItem{}, Page: in.Page, PageSize: in.PageSize, Products: []ProductCount{}}
	rows, err := s.Q.ProductOrdersList(ctx, store.ProductOrdersListParams{WorkspaceID: ws.ID, Q: q, ProductID: product, Status: in.Status, Since: since,
		Lim: int32(in.PageSize), Off: int32((in.Page - 1) * in.PageSize)})
	if err != nil {
		return out, err
	}
	total, err := s.Q.ProductOrdersCount(ctx, store.ProductOrdersCountParams{WorkspaceID: ws.ID, Q: q, ProductID: product, Status: in.Status, Since: since})
	if err != nil {
		return out, err
	}
	out.Total = total
	orders := make([]store.ProductOrder, len(rows))
	for i, r := range rows {
		orders[i] = r.ProductOrder
	}
	states, err := s.downloadStates(ctx, ws, orders)
	if err != nil {
		return out, err
	}
	for i, r := range rows {
		out.Items = append(out.Items, orderItem(r.ProductOrder, r.CurrentProductName, states[i]))
	}
	prods, err := s.Q.ProductsList(ctx, ws.ID)
	if err == nil {
		for _, p := range prods {
			out.Products = append(out.Products, ProductCount{ProductID: p.Product.ID.String(), Name: p.Product.Name})
		}
	}
	return out, nil
}

func orderItem(o store.ProductOrder, name string, d ProductDownloadState) ProductOrderItem {
	return ProductOrderItem{ID: o.ID.String(), Number: o.Number, PaidAt: o.PaidAt, BuyerEmail: o.BuyerEmail, ProductID: o.ProductID.String(),
		Product: ShortName(name), ProductName: name, AmountCents: o.AmountCents, Currency: o.Currency, PayWhatYouWant: o.PayWhatYouWant,
		Status: o.Status, Test: o.Mode == string(payments.Test), RefundedCents: o.RefundedCents, RefundedAt: o.RefundedAt,
		DisputeStatus: o.DisputeStatus, DisputeReason: o.DisputeReason, DisputedAt: o.DisputedAt, Download: d}
}

// downloadStates reads each order's current link and its email.
func (s *Service) downloadStates(ctx context.Context, ws *store.Workspace, orders []store.ProductOrder) ([]ProductDownloadState, error) {
	out := make([]ProductDownloadState, len(orders))
	if len(orders) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	links, err := s.Q.ProductOrderLatestLinks(ctx, ids)
	if err != nil {
		return nil, err
	}
	byOrder := map[uuid.UUID]store.ProductOrderLatestLinksRow{}
	var msgIDs []uuid.UUID
	for _, l := range links {
		byOrder[l.ProductOrderLink.OrderID] = l
		if l.ProductOrderLink.MessageID != nil {
			msgIDs = append(msgIDs, *l.ProductOrderLink.MessageID)
		}
	}
	msgs, err := s.messages(ctx, ws, msgIDs)
	if err != nil {
		return nil, err
	}
	revokers := map[uuid.UUID]string{}
	for i, o := range orders {
		l, ok := byOrder[o.ID]
		var m *store.ProductDeliveryMessagesRow
		if ok && l.ProductOrderLink.MessageID != nil {
			if mm, ok := msgs[*l.ProductOrderLink.MessageID]; ok {
				m = &mm
			}
		}
		var link *store.ProductOrderLatestLinksRow
		if ok {
			link = &l
		}
		if o.AccessRevokedBy != nil {
			if _, done := revokers[*o.AccessRevokedBy]; !done {
				n, _ := s.Q.ProductOrderNames(ctx, o.AccessRevokedBy)
				revokers[*o.AccessRevokedBy] = n
			}
		}
		out[i] = s.downloadState(o, link, m, revokers)
		out[i].Error = deliveryError(lang(ctx), out[i].Error)
	}
	return out, nil
}

func (s *Service) messages(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) (map[uuid.UUID]store.ProductDeliveryMessagesRow, error) {
	out := map[uuid.UUID]store.ProductDeliveryMessagesRow{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Q.ProductDeliveryMessages(ctx, store.ProductDeliveryMessagesParams{WorkspaceID: ws.ID, Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

// downloadState is an order's link as the Download column says it.
func (s *Service) downloadState(o store.ProductOrder, link *store.ProductOrderLatestLinksRow, m *store.ProductDeliveryMessagesRow, revokers map[uuid.UUID]string) ProductDownloadState {
	d := ProductDownloadState{State: "pending", RevokeReason: o.AccessRevokeReason}
	if o.AccessRevokedAt != nil {
		d.RevokedAt = o.AccessRevokedAt
		if o.AccessRevokedBy != nil {
			d.RevokedBy = revokers[*o.AccessRevokedBy]
		}
		d.State = "revoked"
		if o.AccessRevokeReason == "dispute" {
			d.State = "revoked_auto"
		}
	}
	if link == nil {
		return d
	}
	l := link.ProductOrderLink
	d.Used, d.Allowed = l.DownloadsUsed, l.DownloadsAllowed
	exp := l.ExpiresAt
	d.ExpiresAt = &exp
	if l.Reason == "person" || l.Reason == "buyer" || l.Reason == "account" {
		d.NewLink, d.NewLinkBy = true, link.CreatedByName
		at := l.CreatedAt
		d.NewLinkAt = &at
	}
	if d.RevokedAt != nil {
		return d
	}
	switch {
	case l.DownloadsUsed >= l.DownloadsAllowed:
		d.State = "used_up"
	case !s.now().Before(l.ExpiresAt):
		d.State = "expired"
	case l.Delivery == "failed":
		d.State, d.Error = "failed", l.DeliveryError
	case m != nil:
		d.Provider = m.Provider
		switch m.Status {
		case "delivered":
			d.State, d.DeliveredAt = "delivered", m.DeliveredAt
			d.SentAt = m.SentAt
		case "bounced", "complained", "rejected", "failed", "cancelled":
			d.State, d.Error = "failed", m.Status
		default:
			d.State = "sent"
			at := m.QueuedAt
			if m.SentAt != nil {
				at = *m.SentAt
			}
			d.SentAt = &at
		}
	case l.Delivery == "sent":
		d.State = "sent"
		at := l.CreatedAt
		d.SentAt = &at
	}
	return d
}

// ProductOrderBuyer is who bought, and how they relate to contacts.
type ProductOrderBuyer struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	// Kind: a contact already, joined at checkout (the list offer), or not
	// a contact (the order keeps the address for delivery and support only).
	Kind         string     `json:"kind" enum:"contact,joined,none"`
	ContactID    string     `json:"contactId,omitempty"`
	ContactSince *time.Time `json:"contactSince,omitempty"`
	// The list offer's list and when its double opt-in was confirmed.
	List        string     `json:"list"`
	ListStatus  string     `json:"listStatus" enum:",pending,subscribed,unsubscribed"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	DoubleOptIn bool       `json:"doubleOptIn"`
}

// ProductOrderConsent is the consent given at checkout.
type ProductOrderConsent struct {
	Offered bool       `json:"offered"`
	Ticked  bool       `json:"ticked"`
	Text    string     `json:"text"`
	List    string     `json:"list"`
	Page    string     `json:"page"`
	IP      string     `json:"ip"`
	At      *time.Time `json:"at,omitempty"`
}

// ProductOrderPayment is the payment as Stripe reports it.
type ProductOrderPayment struct {
	Provider        string     `json:"provider"`
	Account         string     `json:"account"`
	Mode            string     `json:"mode" enum:"live,test"`
	AmountCents     int64      `json:"amountCents"`
	TaxCents        int64      `json:"taxCents"`
	Currency        string     `json:"currency"`
	RefundedCents   int64      `json:"refundedCents"`
	RefundedAt      *time.Time `json:"refundedAt,omitempty"`
	DisputeStatus   string     `json:"disputeStatus" enum:",open,won,lost"`
	DisputeReason   string     `json:"disputeReason"`
	DisputeFeeCents int64      `json:"disputeFeeCents"`
	DisputeDueBy    *time.Time `json:"disputeDueBy,omitempty"`
	// URL opens the payment in Stripe's dashboard (refunds happen there).
	URL        string `json:"url"`
	DisputeURL string `json:"disputeUrl"`
}

// ProductOrderDelivery is the delivery email.
type ProductOrderDelivery struct {
	MessageID   string     `json:"messageId,omitempty"`
	Template    string     `json:"template"`
	Version     int32      `json:"version"`
	Status      string     `json:"status"`
	Provider    string     `json:"provider"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	Seconds     float64    `json:"seconds"`
	// NewLink: this email carried a new link, sent by a person.
	NewLink bool `json:"newLink"`
}

// ProductOrderDownloads sums up the downloads.
type ProductOrderDownloads struct {
	Count     int64      `json:"count"`
	Addresses int64      `json:"addresses" doc:"Distinct IP addresses (kept only as hashes)"`
	First     *time.Time `json:"first,omitempty"`
	Last      *time.Time `json:"last,omitempty"`
}

// ProductOrderEvent is one line of the order's activity.
type ProductOrderEvent struct {
	Kind string    `json:"kind" enum:"checkout_started,paid,delivery_sent,delivery_delivered,delivery_failed,downloaded,refunded,dispute_opened,dispute_closed,access_revoked,access_restored,new_link,buyer_asked"`
	At   time.Time `json:"at"`
	// By is the person, "" for the system or the buyer.
	By string `json:"by"`
	// What the line says: the page and the list offer, Stripe's event,
	// the template, provider and seconds, the downloads, amounts, reasons.
	Page      string     `json:"page,omitempty"`
	Ticked    bool       `json:"ticked,omitempty"`
	Stripe    string     `json:"stripe,omitempty"`
	Template  string     `json:"template,omitempty"`
	Version   int32      `json:"version,omitempty"`
	Provider  string     `json:"provider,omitempty"`
	Seconds   float64    `json:"seconds,omitempty"`
	Count     int64      `json:"count,omitempty"`
	Used      int32      `json:"used,omitempty"`
	Allowed   int32      `json:"allowed,omitempty"`
	File      string     `json:"file,omitempty"`
	Bytes     int64      `json:"bytes,omitempty"`
	Amount    int64      `json:"amount,omitempty"`
	Currency  string     `json:"currency,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	RespondBy *time.Time `json:"respondBy,omitempty"`
	Outcome   string     `json:"outcome,omitempty"`
	Email     string     `json:"email,omitempty"`
	Note      string     `json:"note,omitempty"`
	ByStripe  bool       `json:"byStripe,omitempty" doc:"Done in Stripe's dashboard"`
	Auto      bool       `json:"auto,omitempty" doc:"Done by the system, as the product's setting says"`
}

// ProductOrderDetail is the order panel.
type ProductOrderDetail struct {
	ProductOrderItem
	ProductCover string                `json:"productCover"`
	ProductAlt   string                `json:"productAlt"`
	Files        ProductFilesSummary   `json:"files"`
	Price        ProductPrice          `json:"price"`
	Buyer        ProductOrderBuyer     `json:"buyer"`
	Consent      ProductOrderConsent   `json:"consent"`
	Payment      ProductOrderPayment   `json:"payment"`
	Delivery     *ProductOrderDelivery `json:"delivery,omitempty"`
	Downloads    ProductOrderDownloads `json:"downloads"`
	// PreviousExpired: the link before the current one had expired.
	Activity []ProductOrderEvent `json:"activity"`
	// What a member may do: send a new link (never while revoked), revoke,
	// restore.
	CanSendLink bool     `json:"canSendLink"`
	CanRevoke   bool     `json:"canRevoke"`
	CanRestore  bool     `json:"canRestore"`
	Route       string   `json:"route"`
	Helpers     []string `json:"helpers"`
}

// Order is one order for its panel.
func (s *Service) Order(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ProductOrderDetail, error) {
	o, err := s.Q.ProductOrderGet(ctx, store.ProductOrderGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ProductOrderDetail{}, mapErr(ctx, err, "order")
	}
	return s.order(ctx, ws, o)
}

func (s *Service) order(ctx context.Context, ws *store.Workspace, o store.ProductOrder) (ProductOrderDetail, error) {
	states, err := s.downloadStates(ctx, ws, []store.ProductOrder{o})
	if err != nil {
		return ProductOrderDetail{}, err
	}
	p, err := s.Q.ProductAny(ctx, o.ProductID)
	if err != nil {
		return ProductOrderDetail{}, err
	}
	live, ok := liveOf(p)
	if !ok {
		live = draftOf(p)
	}
	out := ProductOrderDetail{ProductOrderItem: orderItem(o, live.Name, states[0]), ProductCover: live.CoverURL, ProductAlt: live.CoverAlt,
		Price:    ProductPrice{Kind: live.PriceKind, AmountCents: live.PriceCents, MinimumCents: live.MinCents, SuggestedCents: live.SuggestedCents},
		Activity: []ProductOrderEvent{}, Helpers: s.helpers(ctx, ws, authz.OrdersManage)}
	files, err := s.Q.DownloadFilesOf(ctx, store.DownloadFilesOfParams{OwnerKind: OwnerProduct, OwnerID: o.ProductID})
	if err != nil {
		return out, err
	}
	fs := make([]store.DownloadFile, len(files))
	for i, f := range files {
		fs[i] = f.DownloadFile
	}
	out.Files = summary(fs)

	// The buyer.
	out.Buyer = ProductOrderBuyer{Email: o.BuyerEmail, Name: o.BuyerName, Kind: o.Buyer}
	var consent struct {
		Offered bool       `json:"offered"`
		Ticked  bool       `json:"ticked"`
		Text    string     `json:"text"`
		ListID  string     `json:"listId"`
		Page    string     `json:"page"`
		IP      string     `json:"ip"`
		At      *time.Time `json:"at"`
	}
	_ = json.Unmarshal(o.Consent, &consent)
	out.Consent = ProductOrderConsent{Offered: consent.Offered, Ticked: consent.Ticked, Text: consent.Text, Page: consent.Page, IP: consent.IP, At: consent.At}
	if lid, err := uuid.Parse(consent.ListID); err == nil {
		if lst, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: lid}); err == nil {
			out.Consent.List, out.Buyer.List, out.Buyer.DoubleOptIn = lst.Name, lst.Name, lst.DoubleOptIn
		}
		if o.ContactID != nil {
			if m, err := s.Q.ProductBuyerMembership(ctx, store.ProductBuyerMembershipParams{ListID: lid, ContactID: *o.ContactID}); err == nil {
				out.Buyer.ListStatus, out.Buyer.ConfirmedAt = m.Status, m.SubscribedAt
			}
			// A buyer who joined at checkout confirmed the offer's double
			// opt-in: when they did.
			if o.Buyer == "joined" {
				if at, err := s.Q.ProductBuyerConfirmed(ctx, store.ProductBuyerConfirmedParams{ContactID: *o.ContactID, ListID: &lid}); err == nil {
					out.Buyer.ConfirmedAt = &at
				}
			}
		}
	}
	if o.ContactID != nil {
		if c, err := s.Q.ProductBuyerContact(ctx, store.ProductBuyerContactParams{WorkspaceID: ws.ID, ID: *o.ContactID}); err == nil {
			out.Buyer.ContactID = c.ID.String()
			at := c.CreatedAt
			if o.Buyer == "joined" {
				// They became a contact when the order was paid.
				at = o.PaidAt
			}
			out.Buyer.ContactSince = &at
			if n := strings.TrimSpace(c.FirstName + " " + c.LastName); n != "" {
				out.Buyer.Name = n
			}
		}
	}

	// The payment.
	out.Payment = ProductOrderPayment{Provider: o.Provider, Mode: o.Mode, AmountCents: o.AmountCents, TaxCents: o.TaxCents, Currency: o.Currency,
		RefundedCents: o.RefundedCents, RefundedAt: o.RefundedAt, DisputeStatus: o.DisputeStatus, DisputeReason: o.DisputeReason,
		DisputeFeeCents: o.DisputeFeeCents, DisputeDueBy: o.DisputeDueBy}
	if o.PaymentIntent != "" {
		out.Payment.URL = earn.Dashboard(o.Provider, payments.Mode(o.Mode), "payments", o.PaymentIntent)
	}
	if o.DisputeID != "" {
		out.Payment.DisputeURL = earn.Dashboard(o.Provider, payments.Mode(o.Mode), "disputes", o.DisputeID)
	}
	if s.Earn != nil && s.Earn() != nil {
		if sel, ok, err := s.Earn().SellingIn(ctx, ws, payments.Mode(o.Mode)); err == nil && ok {
			out.Payment.Account = sel.Account
		}
	}
	route, _, err := s.route(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Route = route

	// Links, the delivery email and downloads.
	links, err := s.Q.ProductOrderLinks(ctx, o.ID)
	if err != nil {
		return out, err
	}
	var msgIDs []uuid.UUID
	for _, l := range links {
		if l.ProductOrderLink.MessageID != nil {
			msgIDs = append(msgIDs, *l.ProductOrderLink.MessageID)
		}
	}
	msgs, err := s.messages(ctx, ws, msgIDs)
	if err != nil {
		return out, err
	}
	if len(links) > 0 {
		cur := links[0].ProductOrderLink
		if cur.MessageID != nil {
			out.Delivery = &ProductOrderDelivery{MessageID: cur.MessageID.String(), NewLink: cur.Reason != "purchase"}
			if m, ok := msgs[*cur.MessageID]; ok {
				out.Delivery.Template, out.Delivery.Version, out.Delivery.Status, out.Delivery.Provider = m.TemplateKey, m.TemplateVersion, m.Status, m.Provider
				out.Delivery.SentAt, out.Delivery.DeliveredAt = m.SentAt, m.DeliveredAt
				if m.DeliveredAt != nil {
					out.Delivery.Seconds = deliverySeconds(m)
				}
			}
		}
	}
	dl, err := s.Q.ProductOrderDownloads(ctx, o.ID)
	if err != nil {
		return out, err
	}
	out.Downloads = ProductOrderDownloads{Count: dl.Downloads, Addresses: dl.Addresses}
	if t, ok := dl.FirstAt.(time.Time); ok {
		out.Downloads.First = &t
	}
	if t, ok := dl.LastAt.(time.Time); ok {
		out.Downloads.Last = &t
	}

	// The activity: what happened, oldest first.
	events, err := s.Q.ProductOrderEvents(ctx, o.ID)
	if err != nil {
		return out, err
	}
	for _, e := range events {
		var ev ProductOrderEvent
		_ = json.Unmarshal(e.ProductOrderEvent.Detail, &ev)
		ev.Kind, ev.At, ev.By = e.ProductOrderEvent.Kind, e.ProductOrderEvent.At, e.ActorName
		out.Activity = append(out.Activity, ev)
	}
	for i := len(links) - 1; i >= 0; i-- {
		l := links[i].ProductOrderLink
		if l.MessageID == nil {
			if l.Delivery == "failed" {
				out.Activity = append(out.Activity, ProductOrderEvent{Kind: "delivery_failed", At: l.CreatedAt, Note: deliveryError(lang(ctx), l.DeliveryError)})
			}
			continue
		}
		m, ok := msgs[*l.MessageID]
		if !ok {
			continue
		}
		switch {
		case m.DeliveredAt != nil:
			out.Activity = append(out.Activity, ProductOrderEvent{Kind: "delivery_delivered", At: *m.DeliveredAt, Template: m.TemplateKey, Version: m.TemplateVersion,
				Provider: m.Provider, Seconds: deliverySeconds(m)})
		case m.Status == "bounced" || m.Status == "rejected" || m.Status == "failed" || m.Status == "complained":
			out.Activity = append(out.Activity, ProductOrderEvent{Kind: "delivery_failed", At: m.QueuedAt, Template: m.TemplateKey, Provider: m.Provider, Note: m.Status})
		default:
			at := m.QueuedAt
			if m.SentAt != nil {
				at = *m.SentAt
			}
			out.Activity = append(out.Activity, ProductOrderEvent{Kind: "delivery_sent", At: at, Template: m.TemplateKey, Version: m.TemplateVersion, Provider: m.Provider})
		}
	}
	if dl.Downloads > 0 {
		first := ""
		var size int64
		if len(files) > 0 {
			first, size = kindLabel(files[0].DownloadFile.Kind, files[0].DownloadFile.Name), files[0].DownloadFile.SizeBytes
		}
		used, allowed := int32(0), int32(0)
		if len(links) > 0 {
			used, allowed = links[0].ProductOrderLink.DownloadsUsed, links[0].ProductOrderLink.DownloadsAllowed
		}
		at := s.now()
		if out.Downloads.Last != nil {
			at = *out.Downloads.Last
		}
		out.Activity = append(out.Activity, ProductOrderEvent{Kind: "downloaded", At: at, Count: dl.Downloads, Used: used, Allowed: allowed, File: first, Bytes: size})
	}
	slices.SortStableFunc(out.Activity, func(a, b ProductOrderEvent) int { return a.At.Compare(b.At) })

	revoked := o.AccessRevokedAt != nil
	fullRefund := o.RefundedCents > 0 && o.RefundedCents >= o.AmountCents
	out.CanSendLink = !revoked
	out.CanRevoke = !revoked
	out.CanRestore = revoked && !fullRefund
	return out, nil
}

// deliverySeconds is how long a delivery email took, from the moment it
// was queued to its delivery, as Transactional › Messages counts it.
func deliverySeconds(m store.ProductDeliveryMessagesRow) float64 {
	if m.DeliveredAt == nil {
		return 0
	}
	d := m.DeliveredAt.Sub(m.QueuedAt).Seconds()
	if d < 0 {
		return 0
	}
	return math.Round(d*10) / 10
}

// event records a line of an order's activity.
func (s *Service) event(ctx context.Context, q *store.Queries, orderID uuid.UUID, kind string, at time.Time, by *uuid.UUID, detail map[string]any) error {
	if detail == nil {
		detail = map[string]any{}
	}
	b, _ := json.Marshal(detail)
	return q.ProductOrderEventInsert(ctx, store.ProductOrderEventInsertParams{ID: newID(), OrderID: orderID, Kind: kind, At: at, ActorID: by, Detail: b})
}

// orderName is "order 1311 (The Deliverability Field Guide)" for the audit log.
func orderName(o store.ProductOrder) string {
	return fmt.Sprintf("Order %d · %s · %s", o.Number, o.ProductName, moneyText(o.AmountCents, o.Currency))
}

// NewLinkInput sends a new download link.
type NewLinkInput struct {
	// RequestedBy notes who asked ("the buyer, by replying to the delivery email").
	RequestedBy string `json:"requestedBy,omitempty" maxLength:"200"`
}

// SendNewLink sends a new download link, to the address on the order and
// never another one; the older links stop working.
func (s *Service) SendNewLink(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in NewLinkInput) (ProductOrderDetail, error) {
	l := lang(ctx)
	var linkID uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		o, err := q.ProductOrderLock(ctx, store.ProductOrderLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapErr(ctx, err, "order")
		}
		if o.AccessRevokedAt != nil {
			return api.Conflict(word(l, "order.revoked", nil))
		}
		linkID, err = s.newLink(ctx, tx, ws, o, "person", actor(p))
		if err != nil {
			return err
		}
		now := s.now()
		if err := s.event(ctx, q, o.ID, "new_link", now, actor(p), map[string]any{"email": o.BuyerEmail, "note": strings.TrimSpace(in.RequestedBy)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "order.link_sent", TargetType: "product_order", TargetID: o.ID.String(),
			TargetName: orderName(o), After: map[string]any{"to": o.BuyerEmail, "link": linkID.String()},
			Metadata: map[string]any{"summary": fmt.Sprintf("New download link sent to the address on order %d", o.Number), "requested_by": strings.TrimSpace(in.RequestedBy)}})
	})
	if err != nil {
		return ProductOrderDetail{}, err
	}
	_ = linkID
	return s.Order(ctx, ws, id)
}

// Revoke stops an order's downloads; Restore lets them work again (not
// after a full refund).
func (s *Service) Revoke(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, revoke bool) (ProductOrderDetail, error) {
	l := lang(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		o, err := q.ProductOrderLock(ctx, store.ProductOrderLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapErr(ctx, err, "order")
		}
		now := s.now()
		if revoke {
			if o.AccessRevokedAt != nil {
				return api.Conflict(word(l, "order.alreadyRevoked", nil))
			}
			if err := q.ProductOrderRevoke(ctx, store.ProductOrderRevokeParams{ID: o.ID, At: now, Actor: actor(p), Reason: "manual"}); err != nil {
				return err
			}
			if err := s.event(ctx, q, o.ID, "access_revoked", now, actor(p), map[string]any{"reason": "manual"}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "order.access_revoked", TargetType: "product_order", TargetID: o.ID.String(),
				TargetName: orderName(o), Before: map[string]any{"access": "on"}, After: map[string]any{"access": "revoked"}})
		}
		if o.AccessRevokedAt == nil {
			return api.Conflict(word(l, "order.notRevoked", nil))
		}
		if o.RefundedCents > 0 && o.RefundedCents >= o.AmountCents {
			return api.Conflict(word(l, "order.refundedNoRestore", nil))
		}
		if err := q.ProductOrderRestore(ctx, o.ID); err != nil {
			return err
		}
		if err := s.event(ctx, q, o.ID, "access_restored", now, actor(p), nil); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "order.access_restored", TargetType: "product_order", TargetID: o.ID.String(),
			TargetName: orderName(o), Before: map[string]any{"access": "revoked", "reason": o.AccessRevokeReason}, After: map[string]any{"access": "on"}})
	})
	if err != nil {
		return ProductOrderDetail{}, err
	}
	return s.Order(ctx, ws, id)
}

// Export writes the orders a filter shows as CSV, and records the export.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, p *authz.Principal, in OrdersQuery) ([]byte, int, error) {
	var product *uuid.UUID
	if id, err := uuid.Parse(in.Product); err == nil {
		product = &id
	}
	in.Status = s.orderStatus(ctx, ws, in.Status)
	rows, err := s.Q.ProductOrdersExport(ctx, store.ProductOrdersExportParams{WorkspaceID: ws.ID, Q: strings.TrimSpace(in.Q), ProductID: product, Status: in.Status, Since: s.since(in.Period)})
	if err != nil {
		return nil, 0, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"order", "paid_at", "buyer_email", "product", "amount", "currency", "pay_what_you_want", "status", "refunded", "dispute", "test", "source", "list_offer_ticked", "contact"})
	for _, r := range rows {
		o := r.ProductOrder
		var consent struct {
			Ticked bool `json:"ticked"`
		}
		_ = json.Unmarshal(o.Consent, &consent)
		_ = w.Write([]string{strconv.FormatInt(o.Number, 10), o.PaidAt.UTC().Format(time.RFC3339), o.BuyerEmail, r.CurrentProductName,
			centsText(o.AmountCents), strings.ToUpper(o.Currency), strconv.FormatBool(o.PayWhatYouWant), o.Status, centsText(o.RefundedCents),
			o.DisputeStatus, strconv.FormatBool(o.Mode == string(payments.Test)), o.Source, strconv.FormatBool(consent.Ticked), o.Buyer})
	}
	w.Flush()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "order.exported", TargetType: "workspace", TargetID: ws.ID.String(),
			TargetName: ws.Name, Metadata: map[string]any{"rows": len(rows), "filter": map[string]any{"q": in.Q, "product": in.Product, "status": in.Status, "period": in.Period},
				"summary": fmt.Sprintf("Exported %d orders as CSV", len(rows))}})
	})
	return buf.Bytes(), len(rows), err
}

// centsText is an amount in major units ("29.00").
func centsText(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s%d.%02d", sign, c/100, c%100)
}

// orderStatus is the orders a filter shows: every live order by default;
// the test orders while only a test key is connected (they're left out of
// every number, and shown on their own with the "test" filter).
func (s *Service) orderStatus(ctx context.Context, ws *store.Workspace, status string) string {
	if status != "" {
		return status
	}
	pay, _, err := s.payments(ctx, ws)
	if err == nil && numbersMode(pay) == string(payments.Test) {
		return "test"
	}
	return "live"
}
