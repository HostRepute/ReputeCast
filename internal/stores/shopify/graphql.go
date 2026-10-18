package shopify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
)

// Page sizes. Orders carry their line items, so they come 100 at a time to
// stay well under the query-cost limit.
const (
	pageProducts    = 100
	pageCollections = 100
	pageCustomers   = 250
	pageOrders      = 100
	pageCheckouts   = 100
)

// The operations, pinned to APIVersion. Operation names are what the fake
// answers by (shopifytest).
const (
	qShop = `query Shop { shop { name currencyCode myshopifyDomain primaryDomain { host } } }`

	qCount = `query Count($query: String) {
  productsCount(query: $query, limit: null) { count }
  customersCount(query: $query, limit: null) { count }
  ordersCount(query: $query, limit: null) { count }
}`

	qProducts = `query Products($first: Int!, $after: String, $query: String) {
  products(first: $first, after: $after, query: $query, sortKey: UPDATED_AT) {
    nodes {
      id title handle status onlineStoreUrl createdAt updatedAt
      featuredMedia { preview { image { url altText } } }
      priceRangeV2 { minVariantPrice { amount currencyCode } }
      compareAtPriceRange { minVariantCompareAtPrice { amount currencyCode } }
      tracksInventory totalInventory
      collections(first: 25) { nodes { id } }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

	qCollections = `query Collections($first: Int!, $after: String) {
  collections(first: $first, after: $after) {
    nodes { id title handle productsCount { count } }
    pageInfo { hasNextPage endCursor }
  }
}`

	customerFields = `id firstName lastName createdAt updatedAt
      defaultEmailAddress { emailAddress marketingState marketingOptInLevel marketingUpdatedAt }
      defaultAddress { countryCodeV2 }`

	qCustomers = `query Customers($first: Int!, $after: String, $query: String) {
  customers(first: $first, after: $after, query: $query, sortKey: UPDATED_AT) {
    nodes { ` + customerFields + ` }
    pageInfo { hasNextPage endCursor }
  }
}`

	lineItemFields = `title variantTitle quantity product { id } variant { id } originalUnitPriceSet { shopMoney { amount currencyCode } } image { url }`

	qOrders = `query Orders($first: Int!, $after: String, $query: String) {
  orders(first: $first, after: $after, query: $query, sortKey: CREATED_AT) {
    nodes {
      id name createdAt updatedAt cancelledAt test email currencyCode customerAcceptsMarketing discountCodes
      subtotalPriceSet { shopMoney { amount currencyCode } }
      totalPriceSet { shopMoney { amount currencyCode } }
      totalRefundedSet { shopMoney { amount currencyCode } }
      customer { ` + customerFields + ` }
      customerJourneySummary {
        firstVisit { landingPage utmParameters { source medium campaign content term } }
        lastVisit { landingPage utmParameters { source medium campaign content term } }
      }
      lineItems(first: 50) { nodes { ` + lineItemFields + ` } }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

	qCheckouts = `query Checkouts($first: Int!, $after: String, $query: String) {
  abandonedCheckouts(first: $first, after: $after, query: $query) {
    nodes {
      id abandonedCheckoutUrl createdAt updatedAt completedAt
      customer { ` + customerFields + ` }
      subtotalPriceSet { shopMoney { amount currencyCode } }
      totalPriceSet { shopMoney { amount currencyCode } }
      lineItems(first: 50) { nodes { ` + lineItemFields + ` } }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

	qWebhooks = `query Webhooks { webhookSubscriptions(first: 100) { nodes { id topic uri } } }`

	mSubscribe = `mutation Subscribe($topic: WebhookSubscriptionTopic!, $sub: WebhookSubscriptionInput!) {
  webhookSubscriptionCreate(topic: $topic, webhookSubscription: $sub) {
    webhookSubscription { id topic uri }
    userErrors { field message }
  }
}`

	mUnsubscribe = `mutation Unsubscribe($id: ID!) {
  webhookSubscriptionDelete(id: $id) { deletedWebhookSubscriptionId userErrors { field message } }
}`

	mConsent = `mutation Consent($input: CustomerEmailMarketingConsentUpdateInput!) {
  customerEmailMarketingConsentUpdate(input: $input) { customer { id } userErrors { field message } }
}`
)

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

func (p pageInfo) next() string {
	if p.HasNextPage {
		return p.EndCursor
	}
	return ""
}

type money struct {
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
}

type moneySet struct {
	ShopMoney money `json:"shopMoney"`
}

type userError struct {
	Field   []string `json:"field"`
	Message string   `json:"message"`
}

func userErrors(op string, errs []userError) error {
	if len(errs) == 0 {
		return nil
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Message
	}
	msg := strings.Join(msgs, "; ")
	if strings.Contains(strings.ToLower(msg), "access") && strings.Contains(strings.ToLower(msg), "denied") {
		return fmt.Errorf("%w: %s", stores.ErrPermission, msg)
	}
	return fmt.Errorf("shopify: %s: %s", op, msg)
}

// search builds a search query for a range ("created_at:>='…' AND updated_at:>='…'").
func search(r stores.Range) string {
	var parts []string
	if !r.CreatedSince.IsZero() {
		parts = append(parts, fmt.Sprintf("created_at:>='%s'", r.CreatedSince.UTC().Format(time.RFC3339)))
	}
	if !r.UpdatedSince.IsZero() {
		parts = append(parts, fmt.Sprintf("updated_at:>='%s'", r.UpdatedSince.UTC().Format(time.RFC3339)))
	}
	return strings.Join(parts, " AND ")
}

func pageVars(first int, cursor string, query string) map[string]any {
	v := map[string]any{"first": first}
	if cursor != "" {
		v["after"] = cursor
	}
	if query != "" {
		v["query"] = query
	}
	return v
}

// Shop implements stores.Client.
func (c *Client) Shop(ctx context.Context) (stores.Shop, error) {
	var d struct {
		Shop struct {
			Name            string `json:"name"`
			CurrencyCode    string `json:"currencyCode"`
			MyshopifyDomain string `json:"myshopifyDomain"`
			PrimaryDomain   struct {
				Host string `json:"host"`
			} `json:"primaryDomain"`
		} `json:"shop"`
	}
	if err := c.gql(ctx, "Shop", qShop, nil, &d); err != nil {
		return stores.Shop{}, err
	}
	return stores.Shop{Name: d.Shop.Name, Domain: d.Shop.MyshopifyDomain, PublicDomain: d.Shop.PrimaryDomain.Host, Currency: d.Shop.CurrencyCode}, nil
}

// Count implements stores.Client.
func (c *Client) Count(ctx context.Context, kind string, r stores.Range) (int, error) {
	var d struct {
		ProductsCount  struct{ Count int } `json:"productsCount"`
		CustomersCount struct{ Count int } `json:"customersCount"`
		OrdersCount    struct{ Count int } `json:"ordersCount"`
	}
	vars := map[string]any{}
	if q := search(r); q != "" {
		vars["query"] = q
	}
	if err := c.gql(ctx, "Count", qCount, vars, &d); err != nil {
		return 0, err
	}
	switch kind {
	case stores.CountProducts:
		return d.ProductsCount.Count, nil
	case stores.CountCustomers:
		return d.CustomersCount.Count, nil
	case stores.CountOrders:
		return d.OrdersCount.Count, nil
	}
	return 0, fmt.Errorf("shopify: nothing to count as %q", kind)
}

type gqlProduct struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Handle         string    `json:"handle"`
	Status         string    `json:"status"`
	OnlineStoreURL string    `json:"onlineStoreUrl"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	FeaturedMedia  *struct {
		Preview *struct {
			Image *struct {
				URL     string `json:"url"`
				AltText string `json:"altText"`
			} `json:"image"`
		} `json:"preview"`
	} `json:"featuredMedia"`
	PriceRangeV2 struct {
		MinVariantPrice money `json:"minVariantPrice"`
	} `json:"priceRangeV2"`
	CompareAtPriceRange *struct {
		MinVariantCompareAtPrice money `json:"minVariantCompareAtPrice"`
	} `json:"compareAtPriceRange"`
	TracksInventory bool `json:"tracksInventory"`
	TotalInventory  int  `json:"totalInventory"`
	Collections     struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	} `json:"collections"`
}

func (p gqlProduct) product() stores.Product {
	cur := p.PriceRangeV2.MinVariantPrice.CurrencyCode
	out := stores.Product{
		ID: numericID(p.ID), Title: p.Title, Handle: p.Handle, URL: p.OnlineStoreURL,
		Price: stores.ParseAmount(p.PriceRangeV2.MinVariantPrice.Amount, cur), Currency: cur,
		Status: strings.ToLower(p.Status), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Available: true,
		Collections: []string{},
	}
	if p.CompareAtPriceRange != nil {
		ca := stores.ParseAmount(p.CompareAtPriceRange.MinVariantCompareAtPrice.Amount, cur)
		if ca > out.Price {
			out.CompareAt = ca
		}
	}
	if p.FeaturedMedia != nil && p.FeaturedMedia.Preview != nil && p.FeaturedMedia.Preview.Image != nil {
		out.ImageURL, out.ImageAlt = p.FeaturedMedia.Preview.Image.URL, p.FeaturedMedia.Preview.Image.AltText
	}
	if p.TracksInventory {
		n := p.TotalInventory
		out.Inventory = &n
		out.Available = n > 0
	}
	for _, col := range p.Collections.Nodes {
		out.Collections = append(out.Collections, numericID(col.ID))
	}
	return out
}

// Products implements stores.Client.
func (c *Client) Products(ctx context.Context, cursor string, r stores.Range) (stores.Page[stores.Product], error) {
	var d struct {
		Products struct {
			Nodes    []gqlProduct `json:"nodes"`
			PageInfo pageInfo     `json:"pageInfo"`
		} `json:"products"`
	}
	if err := c.gql(ctx, "Products", qProducts, pageVars(pageProducts, cursor, search(r)), &d); err != nil {
		return stores.Page[stores.Product]{}, err
	}
	out := stores.Page[stores.Product]{Items: make([]stores.Product, 0, len(d.Products.Nodes)), Next: d.Products.PageInfo.next()}
	for _, p := range d.Products.Nodes {
		out.Items = append(out.Items, p.product())
	}
	return out, nil
}

// Collections implements stores.Client.
func (c *Client) Collections(ctx context.Context, cursor string) (stores.Page[stores.Collection], error) {
	var d struct {
		Collections struct {
			Nodes []struct {
				ID            string `json:"id"`
				Title         string `json:"title"`
				Handle        string `json:"handle"`
				ProductsCount struct {
					Count int `json:"count"`
				} `json:"productsCount"`
			} `json:"nodes"`
			PageInfo pageInfo `json:"pageInfo"`
		} `json:"collections"`
	}
	if err := c.gql(ctx, "Collections", qCollections, pageVars(pageCollections, cursor, ""), &d); err != nil {
		return stores.Page[stores.Collection]{}, err
	}
	out := stores.Page[stores.Collection]{Items: []stores.Collection{}, Next: d.Collections.PageInfo.next()}
	for _, n := range d.Collections.Nodes {
		out.Items = append(out.Items, stores.Collection{ID: numericID(n.ID), Title: n.Title, Handle: n.Handle, Products: n.ProductsCount.Count})
	}
	return out, nil
}

type gqlCustomer struct {
	ID                  string    `json:"id"`
	FirstName           string    `json:"firstName"`
	LastName            string    `json:"lastName"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
	DefaultEmailAddress *struct {
		EmailAddress        string     `json:"emailAddress"`
		MarketingState      string     `json:"marketingState"`
		MarketingOptInLevel string     `json:"marketingOptInLevel"`
		MarketingUpdatedAt  *time.Time `json:"marketingUpdatedAt"`
	} `json:"defaultEmailAddress"`
	DefaultAddress *struct {
		CountryCodeV2 string `json:"countryCodeV2"`
	} `json:"defaultAddress"`
}

func (g *gqlCustomer) customer() *stores.Customer {
	if g == nil {
		return nil
	}
	out := &stores.Customer{ID: numericID(g.ID), FirstName: g.FirstName, LastName: g.LastName, CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
		Consent: stores.Consent{State: stores.ConsentNotSubscribed, Level: "unknown"}}
	if e := g.DefaultEmailAddress; e != nil {
		out.Email = strings.ToLower(strings.TrimSpace(e.EmailAddress))
		out.Consent = stores.Consent{State: consentState(e.MarketingState), Level: optInLevel(e.MarketingOptInLevel), UpdatedAt: e.MarketingUpdatedAt}
	}
	if g.DefaultAddress != nil {
		out.Country = g.DefaultAddress.CountryCodeV2
	}
	return out
}

// consentState maps Shopify's marketing states (GraphQL's SUBSCRIBED and
// webhooks' subscribed alike) onto the contract's.
func consentState(s string) string {
	switch strings.ToLower(s) {
	case "subscribed":
		return stores.ConsentSubscribed
	case "unsubscribed", "redacted", "invalid":
		return stores.ConsentUnsubscribed
	case "pending":
		return stores.ConsentPending
	}
	return stores.ConsentNotSubscribed
}

func optInLevel(s string) string {
	switch strings.ToLower(s) {
	case "single_opt_in":
		return "single_opt_in"
	case "confirmed_opt_in":
		return "confirmed_opt_in"
	}
	return "unknown"
}

// Customers implements stores.Client.
func (c *Client) Customers(ctx context.Context, cursor string, r stores.Range) (stores.Page[stores.Customer], error) {
	var d struct {
		Customers struct {
			Nodes    []gqlCustomer `json:"nodes"`
			PageInfo pageInfo      `json:"pageInfo"`
		} `json:"customers"`
	}
	if err := c.gql(ctx, "Customers", qCustomers, pageVars(pageCustomers, cursor, search(r)), &d); err != nil {
		return stores.Page[stores.Customer]{}, err
	}
	out := stores.Page[stores.Customer]{Items: make([]stores.Customer, 0, len(d.Customers.Nodes)), Next: d.Customers.PageInfo.next()}
	for i := range d.Customers.Nodes {
		out.Items = append(out.Items, *d.Customers.Nodes[i].customer())
	}
	return out, nil
}

type gqlLineItem struct {
	Title        string `json:"title"`
	VariantTitle string `json:"variantTitle"`
	Quantity     int    `json:"quantity"`
	Product      *struct {
		ID string `json:"id"`
	} `json:"product"`
	Variant *struct {
		ID string `json:"id"`
	} `json:"variant"`
	OriginalUnitPriceSet moneySet `json:"originalUnitPriceSet"`
	Image                *struct {
		URL string `json:"url"`
	} `json:"image"`
}

func lineItems(nodes []gqlLineItem, currency string) []stores.LineItem {
	out := make([]stores.LineItem, 0, len(nodes))
	for _, n := range nodes {
		li := stores.LineItem{Title: n.Title, Variant: variantTitle(n.VariantTitle), Quantity: n.Quantity,
			Price: stores.ParseAmount(n.OriginalUnitPriceSet.ShopMoney.Amount, currency)}
		if n.Product != nil {
			li.ProductID = numericID(n.Product.ID)
		}
		if n.Variant != nil {
			li.VariantID = numericID(n.Variant.ID)
		}
		if n.Image != nil {
			li.ImageURL = n.Image.URL
		}
		out = append(out, li)
	}
	return out
}

// guest stands in for an order's missing customer (a guest checkout): the
// email, and the consent the checkout's box gave.
func guest(c *stores.Customer, email string, accepts bool) *stores.Customer {
	if c != nil || email == "" {
		if c != nil && c.Email == "" {
			c.Email = email
		}
		return c
	}
	consent := stores.Consent{State: stores.ConsentNotSubscribed, Level: "unknown"}
	if accepts {
		consent = stores.Consent{State: stores.ConsentSubscribed, Level: "single_opt_in"}
	}
	return &stores.Customer{Email: email, Consent: consent}
}

// variantTitle drops Shopify's placeholder for products without options.
func variantTitle(s string) string {
	if s == "Default Title" {
		return ""
	}
	return s
}

type visit struct {
	LandingPage   string `json:"landingPage"`
	UTMParameters *struct {
		Source   string `json:"source"`
		Medium   string `json:"medium"`
		Campaign string `json:"campaign"`
		Content  string `json:"content"`
		Term     string `json:"term"`
	} `json:"utmParameters"`
}

type gqlOrder struct {
	ID                       string       `json:"id"`
	Name                     string       `json:"name"`
	CreatedAt                time.Time    `json:"createdAt"`
	UpdatedAt                time.Time    `json:"updatedAt"`
	CancelledAt              *time.Time   `json:"cancelledAt"`
	Test                     bool         `json:"test"`
	Email                    string       `json:"email"`
	CurrencyCode             string       `json:"currencyCode"`
	CustomerAcceptsMarketing bool         `json:"customerAcceptsMarketing"`
	DiscountCodes            []string     `json:"discountCodes"`
	SubtotalPriceSet         moneySet     `json:"subtotalPriceSet"`
	TotalPriceSet            moneySet     `json:"totalPriceSet"`
	TotalRefundedSet         moneySet     `json:"totalRefundedSet"`
	Customer                 *gqlCustomer `json:"customer"`
	CustomerJourneySummary   *struct {
		FirstVisit *visit `json:"firstVisit"`
		LastVisit  *visit `json:"lastVisit"`
	} `json:"customerJourneySummary"`
	LineItems struct {
		Nodes []gqlLineItem `json:"nodes"`
	} `json:"lineItems"`
}

func (o gqlOrder) order() stores.Order {
	cur := o.TotalPriceSet.ShopMoney.CurrencyCode
	if cur == "" {
		cur = o.CurrencyCode
	}
	out := stores.Order{
		ID: numericID(o.ID), Name: o.Name, Number: strings.TrimPrefix(o.Name, "#"),
		Email: strings.ToLower(strings.TrimSpace(o.Email)), Customer: o.Customer.customer(), AcceptsMarketing: o.CustomerAcceptsMarketing,
		PlacedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt, Currency: cur,
		Subtotal: stores.ParseAmount(o.SubtotalPriceSet.ShopMoney.Amount, cur), Total: stores.ParseAmount(o.TotalPriceSet.ShopMoney.Amount, cur),
		Refunded: stores.ParseAmount(o.TotalRefundedSet.ShopMoney.Amount, cur), Discounts: o.DiscountCodes,
		Items: lineItems(o.LineItems.Nodes, cur), Cancelled: o.CancelledAt != nil, Test: o.Test,
	}
	if out.Discounts == nil {
		out.Discounts = []string{}
	}
	if out.Email == "" && out.Customer != nil {
		out.Email = out.Customer.Email
	}
	out.Customer = guest(out.Customer, out.Email, out.AcceptsMarketing)
	if j := o.CustomerJourneySummary; j != nil {
		v := j.LastVisit
		if v == nil || v.LandingPage == "" {
			v = j.FirstVisit
		}
		if v != nil {
			out.LandingPage = v.LandingPage
			out.UTM, out.ClickToken = stores.ParseLanding(v.LandingPage)
			if u := v.UTMParameters; u != nil && out.UTM == (stores.UTM{}) {
				out.UTM = stores.UTM{Source: u.Source, Medium: u.Medium, Campaign: u.Campaign, Content: u.Content, Term: u.Term}
			}
		}
	}
	return out
}

// Orders implements stores.Client.
func (c *Client) Orders(ctx context.Context, cursor string, r stores.Range) (stores.Page[stores.Order], error) {
	var d struct {
		Orders struct {
			Nodes    []gqlOrder `json:"nodes"`
			PageInfo pageInfo   `json:"pageInfo"`
		} `json:"orders"`
	}
	if err := c.gql(ctx, "Orders", qOrders, pageVars(pageOrders, cursor, search(r)), &d); err != nil {
		return stores.Page[stores.Order]{}, err
	}
	out := stores.Page[stores.Order]{Items: make([]stores.Order, 0, len(d.Orders.Nodes)), Next: d.Orders.PageInfo.next()}
	for _, o := range d.Orders.Nodes {
		out.Items = append(out.Items, o.order())
	}
	return out, nil
}

// Checkouts implements stores.Client: Shopify's abandoned checkouts.
func (c *Client) Checkouts(ctx context.Context, cursor string, r stores.Range) (stores.Page[stores.Checkout], error) {
	var d struct {
		AbandonedCheckouts struct {
			Nodes []struct {
				ID                   string       `json:"id"`
				AbandonedCheckoutURL string       `json:"abandonedCheckoutUrl"`
				CreatedAt            time.Time    `json:"createdAt"`
				UpdatedAt            time.Time    `json:"updatedAt"`
				CompletedAt          *time.Time   `json:"completedAt"`
				Customer             *gqlCustomer `json:"customer"`
				SubtotalPriceSet     moneySet     `json:"subtotalPriceSet"`
				TotalPriceSet        moneySet     `json:"totalPriceSet"`
				LineItems            struct {
					Nodes []gqlLineItem `json:"nodes"`
				} `json:"lineItems"`
			} `json:"nodes"`
			PageInfo pageInfo `json:"pageInfo"`
		} `json:"abandonedCheckouts"`
	}
	if err := c.gql(ctx, "Checkouts", qCheckouts, pageVars(pageCheckouts, cursor, search(r)), &d); err != nil {
		return stores.Page[stores.Checkout]{}, err
	}
	out := stores.Page[stores.Checkout]{Items: []stores.Checkout{}, Next: d.AbandonedCheckouts.PageInfo.next()}
	for _, n := range d.AbandonedCheckouts.Nodes {
		cur := n.TotalPriceSet.ShopMoney.CurrencyCode
		ch := stores.Checkout{ID: checkoutToken(n.AbandonedCheckoutURL, n.ID), Customer: n.Customer.customer(), Items: lineItems(n.LineItems.Nodes, cur),
			Subtotal: stores.ParseAmount(n.SubtotalPriceSet.ShopMoney.Amount, cur), Total: stores.ParseAmount(n.TotalPriceSet.ShopMoney.Amount, cur),
			Currency: cur, RecoveryURL: n.AbandonedCheckoutURL, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt, CompletedAt: n.CompletedAt}
		if ch.Customer != nil {
			ch.Email = ch.Customer.Email
			ch.AcceptsMarketing = ch.Customer.Consent.Accepted()
		}
		out.Items = append(out.Items, ch)
	}
	return out, nil
}

// checkoutToken is the token webhooks and orders use for a checkout: the
// last path segment of its recovery link (/…/checkouts/{token}/recover),
// else its numeric id.
func checkoutToken(recovery, id string) string {
	if i := strings.Index(recovery, "/checkouts/"); i >= 0 {
		rest := recovery[i+len("/checkouts/"):]
		if j := strings.IndexAny(rest, "/?"); j >= 0 {
			rest = rest[:j]
		}
		if rest != "" {
			return rest
		}
	}
	return numericID(id)
}

// Webhooks implements stores.Client.
func (c *Client) Webhooks(ctx context.Context) ([]stores.Webhook, error) {
	var d struct {
		WebhookSubscriptions struct {
			Nodes []struct {
				ID    string `json:"id"`
				Topic string `json:"topic"`
				URI   string `json:"uri"`
			} `json:"nodes"`
		} `json:"webhookSubscriptions"`
	}
	if err := c.gql(ctx, "Webhooks", qWebhooks, nil, &d); err != nil {
		return nil, err
	}
	out := []stores.Webhook{}
	for _, n := range d.WebhookSubscriptions.Nodes {
		out = append(out, stores.Webhook{ID: n.ID, Topic: TopicFromEnum(n.Topic), Address: n.URI})
	}
	return out, nil
}

// Subscribe implements stores.Client. Shopify signs deliveries with the
// app's client secret, so secret isn't sent.
func (c *Client) Subscribe(ctx context.Context, topic, address, _ string) (stores.Webhook, error) {
	var d struct {
		WebhookSubscriptionCreate struct {
			WebhookSubscription *struct {
				ID    string `json:"id"`
				Topic string `json:"topic"`
				URI   string `json:"uri"`
			} `json:"webhookSubscription"`
			UserErrors []userError `json:"userErrors"`
		} `json:"webhookSubscriptionCreate"`
	}
	vars := map[string]any{"topic": TopicEnum(topic), "sub": map[string]any{"uri": address, "format": "JSON"}}
	if err := c.gql(ctx, "Subscribe", mSubscribe, vars, &d); err != nil {
		return stores.Webhook{}, err
	}
	if err := userErrors("webhookSubscriptionCreate", d.WebhookSubscriptionCreate.UserErrors); err != nil {
		return stores.Webhook{}, err
	}
	w := d.WebhookSubscriptionCreate.WebhookSubscription
	if w == nil {
		return stores.Webhook{}, fmt.Errorf("shopify: webhookSubscriptionCreate returned no subscription")
	}
	return stores.Webhook{ID: w.ID, Topic: TopicFromEnum(w.Topic), Address: w.URI}, nil
}

// Unsubscribe implements stores.Client. A webhook that's already gone is
// not an error.
func (c *Client) Unsubscribe(ctx context.Context, id string) error {
	var d struct {
		WebhookSubscriptionDelete struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"webhookSubscriptionDelete"`
	}
	err := c.gql(ctx, "Unsubscribe", mUnsubscribe, map[string]any{"id": gid("WebhookSubscription", id)}, &d)
	if err != nil {
		return err
	}
	for _, e := range d.WebhookSubscriptionDelete.UserErrors {
		if strings.Contains(strings.ToLower(e.Message), "not exist") || strings.Contains(strings.ToLower(e.Message), "not found") {
			return nil
		}
	}
	return userErrors("webhookSubscriptionDelete", d.WebhookSubscriptionDelete.UserErrors)
}

// SetConsent implements stores.Client: the customer's email marketing
// consent becomes unsubscribed (write_customers).
func (c *Client) SetConsent(ctx context.Context, customerID, state string, at time.Time) error {
	var d struct {
		CustomerEmailMarketingConsentUpdate struct {
			UserErrors []userError `json:"userErrors"`
		} `json:"customerEmailMarketingConsentUpdate"`
	}
	ms := "UNSUBSCRIBED"
	if state == stores.ConsentSubscribed {
		ms = "SUBSCRIBED"
	}
	input := map[string]any{
		"customerId":            gid("Customer", customerID),
		"emailMarketingConsent": map[string]any{"marketingState": ms, "consentUpdatedAt": at.UTC().Format(time.RFC3339)},
	}
	if err := c.gql(ctx, "Consent", mConsent, map[string]any{"input": input}, &d); err != nil {
		return err
	}
	return userErrors("customerEmailMarketingConsentUpdate", d.CustomerEmailMarketingConsentUpdate.UserErrors)
}

// TopicEnum turns "orders/create" into ORDERS_CREATE.
func TopicEnum(topic string) string {
	return strings.ToUpper(strings.ReplaceAll(topic, "/", "_"))
}

// TopicFromEnum turns ORDERS_CREATE into "orders/create".
func TopicFromEnum(enum string) string {
	e := strings.ToLower(enum)
	i := strings.IndexByte(e, '_')
	if i < 0 {
		return e
	}
	return e[:i] + "/" + e[i+1:]
}
