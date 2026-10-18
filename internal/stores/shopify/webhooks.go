package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
)

// Webhooks verifies and parses Shopify's deliveries. Shopify signs the raw
// body with the app's client secret (X-Shopify-Hmac-Sha256, base64).
type Webhooks struct{}

var _ stores.WebhookParser = Webhooks{}

// Sign is the X-Shopify-Hmac-Sha256 value of a body (tests and the fake).
func Sign(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

// Verify implements stores.WebhookParser.
func (Webhooks) Verify(h http.Header, body []byte, secret string) error {
	got := strings.TrimSpace(h.Get("X-Shopify-Hmac-Sha256"))
	if got == "" || secret == "" {
		return stores.ErrBadSignature
	}
	sig, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		return stores.ErrBadSignature
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	if !hmac.Equal(sig, m.Sum(nil)) {
		return stores.ErrBadSignature
	}
	return nil
}

// Topic implements stores.WebhookParser. Shopify's event id identifies an
// event across its retries; the older webhook id is used when it's absent.
func (Webhooks) Topic(h http.Header) (string, string) {
	delivery := h.Get("X-Shopify-Event-Id")
	if delivery == "" {
		delivery = h.Get("X-Shopify-Webhook-Id")
	}
	return strings.ToLower(strings.TrimSpace(h.Get("X-Shopify-Topic"))), delivery
}

// The REST-shaped payloads webhooks carry, with only the fields
// ReputeCast reads: no address, phone, payment detail or note has a field.

type restConsent struct {
	State            string     `json:"state"`
	OptInLevel       string     `json:"opt_in_level"`
	ConsentUpdatedAt *time.Time `json:"consent_updated_at"`
}

type restCustomer struct {
	ID                    json.Number  `json:"id"`
	Email                 string       `json:"email"`
	FirstName             string       `json:"first_name"`
	LastName              string       `json:"last_name"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
	EmailMarketingConsent *restConsent `json:"email_marketing_consent"`
	DefaultAddress        *struct {
		CountryCode string `json:"country_code"`
	} `json:"default_address"`
}

func (c *restCustomer) customer() *stores.Customer {
	if c == nil || c.ID == "" {
		return nil
	}
	out := &stores.Customer{ID: c.ID.String(), Email: strings.ToLower(strings.TrimSpace(c.Email)), FirstName: c.FirstName, LastName: c.LastName,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, Consent: stores.Consent{State: stores.ConsentNotSubscribed, Level: "unknown"}}
	if e := c.EmailMarketingConsent; e != nil {
		out.Consent = stores.Consent{State: consentState(e.State), Level: optInLevel(e.OptInLevel), UpdatedAt: e.ConsentUpdatedAt}
	}
	if c.DefaultAddress != nil {
		out.Country = c.DefaultAddress.CountryCode
	}
	return out
}

type restLineItem struct {
	ProductID    *json.Number `json:"product_id"`
	VariantID    *json.Number `json:"variant_id"`
	Title        string       `json:"title"`
	VariantTitle string       `json:"variant_title"`
	Quantity     int          `json:"quantity"`
	Price        string       `json:"price"`
}

func restItems(items []restLineItem, currency string) []stores.LineItem {
	out := make([]stores.LineItem, 0, len(items))
	for _, it := range items {
		li := stores.LineItem{Title: it.Title, Variant: variantTitle(it.VariantTitle), Quantity: it.Quantity, Price: stores.ParseAmount(it.Price, currency)}
		if it.ProductID != nil {
			li.ProductID = it.ProductID.String()
		}
		if it.VariantID != nil {
			li.VariantID = it.VariantID.String()
		}
		out = append(out, li)
	}
	return out
}

type restTransaction struct {
	Amount string `json:"amount"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type restRefund struct {
	ID           json.Number       `json:"id"`
	OrderID      json.Number       `json:"order_id"`
	CreatedAt    time.Time         `json:"created_at"`
	Transactions []restTransaction `json:"transactions"`
}

func (r restRefund) amount(currency string) int64 {
	var sum int64
	for _, t := range r.Transactions {
		if t.Kind == "refund" && (t.Status == "" || t.Status == "success") {
			sum += stores.ParseAmount(t.Amount, currency)
		}
	}
	return sum
}

type restOrder struct {
	ID                    json.Number             `json:"id"`
	Name                  string                  `json:"name"`
	OrderNumber           json.Number             `json:"order_number"`
	Email                 string                  `json:"email"`
	CreatedAt             time.Time               `json:"created_at"`
	UpdatedAt             time.Time               `json:"updated_at"`
	CancelledAt           *time.Time              `json:"cancelled_at"`
	Currency              string                  `json:"currency"`
	SubtotalPrice         string                  `json:"subtotal_price"`
	TotalPrice            string                  `json:"total_price"`
	DiscountCodes         []struct{ Code string } `json:"discount_codes"`
	LandingSite           string                  `json:"landing_site"`
	CheckoutToken         string                  `json:"checkout_token"`
	BuyerAcceptsMarketing bool                    `json:"buyer_accepts_marketing"`
	Test                  bool                    `json:"test"`
	Customer              *restCustomer           `json:"customer"`
	LineItems             []restLineItem          `json:"line_items"`
	Refunds               []restRefund            `json:"refunds"`
}

func (o restOrder) order() stores.Order {
	out := stores.Order{
		ID: o.ID.String(), Name: o.Name, Number: o.OrderNumber.String(), Email: strings.ToLower(strings.TrimSpace(o.Email)),
		Customer: o.Customer.customer(), AcceptsMarketing: o.BuyerAcceptsMarketing, PlacedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		Currency: o.Currency, Subtotal: stores.ParseAmount(o.SubtotalPrice, o.Currency), Total: stores.ParseAmount(o.TotalPrice, o.Currency),
		Items: restItems(o.LineItems, o.Currency), LandingPage: o.LandingSite, CheckoutToken: o.CheckoutToken,
		Cancelled: o.CancelledAt != nil, Test: o.Test, Discounts: []string{},
	}
	if out.Number == "" {
		out.Number = strings.TrimPrefix(o.Name, "#")
	}
	for _, d := range o.DiscountCodes {
		out.Discounts = append(out.Discounts, d.Code)
	}
	for _, r := range o.Refunds {
		amt := r.amount(o.Currency)
		out.Refunded += amt
		out.Refunds = append(out.Refunds, stores.Refund{ID: r.ID.String(), OrderID: out.ID, Amount: amt, At: r.CreatedAt})
	}
	out.UTM, out.ClickToken = stores.ParseLanding(o.LandingSite)
	if out.Email == "" && out.Customer != nil {
		out.Email = out.Customer.Email
	}
	out.Customer = guest(out.Customer, out.Email, out.AcceptsMarketing)
	return out
}

type restCheckout struct {
	ID                    json.Number    `json:"id"`
	Token                 string         `json:"token"`
	Email                 string         `json:"email"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	CompletedAt           *time.Time     `json:"completed_at"`
	AbandonedCheckoutURL  string         `json:"abandoned_checkout_url"`
	Currency              string         `json:"currency"`
	PresentmentCurrency   string         `json:"presentment_currency"`
	SubtotalPrice         string         `json:"subtotal_price"`
	TotalPrice            string         `json:"total_price"`
	LandingSite           string         `json:"landing_site"`
	BuyerAcceptsMarketing bool           `json:"buyer_accepts_marketing"`
	Customer              *restCustomer  `json:"customer"`
	LineItems             []restLineItem `json:"line_items"`
}

func (c restCheckout) checkout() stores.Checkout {
	cur := c.Currency
	if cur == "" {
		cur = c.PresentmentCurrency
	}
	id := c.Token
	if id == "" {
		id = c.ID.String()
	}
	out := stores.Checkout{ID: id, Email: strings.ToLower(strings.TrimSpace(c.Email)), Customer: c.Customer.customer(), AcceptsMarketing: c.BuyerAcceptsMarketing,
		Items: restItems(c.LineItems, cur), Subtotal: stores.ParseAmount(c.SubtotalPrice, cur), Total: stores.ParseAmount(c.TotalPrice, cur), Currency: cur,
		RecoveryURL: c.AbandonedCheckoutURL, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, CompletedAt: c.CompletedAt}
	_, out.ClickToken = stores.ParseLanding(c.LandingSite)
	if out.Customer != nil {
		if out.Email == "" {
			out.Email = out.Customer.Email
		}
		if out.Customer.Consent.Accepted() {
			out.AcceptsMarketing = true
		}
	}
	return out
}

type restProduct struct {
	ID          json.Number `json:"id"`
	Title       string      `json:"title"`
	Handle      string      `json:"handle"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	PublishedAt *time.Time  `json:"published_at"`
	Image       *struct {
		Src string `json:"src"`
		Alt string `json:"alt"`
	} `json:"image"`
	Variants []struct {
		Price               string `json:"price"`
		CompareAtPrice      string `json:"compare_at_price"`
		InventoryQuantity   int    `json:"inventory_quantity"`
		InventoryManagement string `json:"inventory_management"`
		InventoryPolicy     string `json:"inventory_policy"`
	} `json:"variants"`
}

// product maps a product payload; currency is the store's (product
// payloads don't name it).
func (p restProduct) product(currency string) stores.Product {
	out := stores.Product{ID: p.ID.String(), Title: p.Title, Handle: p.Handle, Status: strings.ToLower(p.Status), Currency: currency,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Collections: []string{}}
	if p.Image != nil {
		out.ImageURL, out.ImageAlt = p.Image.Src, p.Image.Alt
	}
	tracked := false
	stock := 0
	sellable := false
	for i, v := range p.Variants {
		price := stores.ParseAmount(v.Price, currency)
		if i == 0 || price < out.Price {
			out.Price = price
			out.CompareAt = 0
			if ca := stores.ParseAmount(v.CompareAtPrice, currency); ca > price {
				out.CompareAt = ca
			}
		}
		if v.InventoryManagement != "" {
			tracked = true
			if v.InventoryQuantity > 0 {
				stock += v.InventoryQuantity
				sellable = true
			} else if v.InventoryPolicy == "continue" {
				sellable = true
			}
		} else {
			sellable = true
		}
	}
	out.Available = sellable || len(p.Variants) == 0
	if tracked {
		out.Inventory = &stock
	}
	return out
}

// Parse implements stores.WebhookParser.
func (w Webhooks) Parse(h http.Header, body []byte) (stores.Event, error) {
	topic, delivery := w.Topic(h)
	ev := stores.Event{Topic: topic, DeliveryID: delivery}
	if at := h.Get("X-Shopify-Triggered-At"); at != "" {
		if t, err := time.Parse(time.RFC3339Nano, at); err == nil {
			ev.TriggeredAt = t
		}
	}
	switch topic {
	case "orders/create", "orders/paid", "orders/updated", "orders/cancelled", "orders/fulfilled":
		var o restOrder
		if err := json.Unmarshal(body, &o); err != nil {
			return ev, fmt.Errorf("shopify: unreadable order: %w", err)
		}
		order := o.order()
		ev.Kind, ev.Order = stores.EventOrder, &order
	case "orders/delete":
		var o struct {
			ID json.Number `json:"id"`
		}
		if err := json.Unmarshal(body, &o); err != nil {
			return ev, fmt.Errorf("shopify: unreadable order deletion: %w", err)
		}
		ev.Kind, ev.DeletedID = stores.EventOrderDeleted, o.ID.String()
	case "refunds/create":
		var r restRefund
		if err := json.Unmarshal(body, &r); err != nil {
			return ev, fmt.Errorf("shopify: unreadable refund: %w", err)
		}
		cur := h.Get(stores.CurrencyHeader)
		ev.Kind = stores.EventRefund
		ev.Refund = &stores.Refund{ID: r.ID.String(), OrderID: r.OrderID.String(), Amount: r.amount(cur), At: r.CreatedAt}
	case "customers/create", "customers/update", "customers/enable", "customers/disable", "customers_email_marketing_consent/update":
		var c restCustomer
		if err := json.Unmarshal(body, &c); err != nil {
			return ev, fmt.Errorf("shopify: unreadable customer: %w", err)
		}
		ev.Kind, ev.Customer = stores.EventCustomer, c.customer()
		if ev.Customer == nil {
			return ev, fmt.Errorf("shopify: customer without an id")
		}
	case "checkouts/create", "checkouts/update":
		var c restCheckout
		if err := json.Unmarshal(body, &c); err != nil {
			return ev, fmt.Errorf("shopify: unreadable checkout: %w", err)
		}
		ch := c.checkout()
		ev.Kind, ev.Checkout = stores.EventCheckout, &ch
	case "products/create", "products/update":
		var p restProduct
		if err := json.Unmarshal(body, &p); err != nil {
			return ev, fmt.Errorf("shopify: unreadable product: %w", err)
		}
		prod := p.product(h.Get(stores.CurrencyHeader))
		ev.Kind, ev.Product = stores.EventProduct, &prod
	case "products/delete":
		var p struct {
			ID json.Number `json:"id"`
		}
		if err := json.Unmarshal(body, &p); err != nil {
			return ev, fmt.Errorf("shopify: unreadable product deletion: %w", err)
		}
		ev.Kind, ev.DeletedID = stores.EventProductDeleted, p.ID.String()
	case LifecycleTopic:
		ev.Kind = stores.EventUninstalled
	default:
		return ev, fmt.Errorf("shopify: topic %q isn't one ReputeCast subscribes to", topic)
	}
	return ev, nil
}
