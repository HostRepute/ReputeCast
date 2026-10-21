package commerce

import "time"

// The API's types. Names start with Commerce: they become OpenAPI schema
// names, unique across the API.

// CommerceStoreSummary is a store in lists: the catalog's cards, the
// composer's product sources, the automation trigger's store picker.
type CommerceStoreSummary struct {
	ID       string `json:"id"`
	Platform string `json:"platform" enum:"shopify,woocommerce"`
	Name     string `json:"name"`
	// Address is what was typed (shop.acme.io); Domain the canonical host
	// (acme-merch.myshopify.com).
	Address  string            `json:"address"`
	Domain   string            `json:"domain"`
	Currency string            `json:"currency"`
	Status   string            `json:"status" enum:"connecting,importing,connected,disconnected"`
	Step     string            `json:"step"`
	Sync     CommerceSyncState `json:"sync"`
	Products int64             `json:"products"`
	ListID   string            `json:"listId,omitempty"`
	ListName string            `json:"listName,omitempty"`
	Demo     bool              `json:"demo"`
}

// CommerceSyncState is the store sync row: "Connected · last event 2 min
// ago", "Importing orders · 5,120 of 7,880", "Webhook missing · polling
// every 15 min", "Disconnected".
type CommerceSyncState struct {
	State       string     `json:"state" enum:"connecting,importing,connected,webhook_missing,rate_limited,no_plugin,disconnected"`
	LastEventAt *time.Time `json:"lastEventAt,omitempty"`
	// Missing are the webhooks gone from the store, polled every 15 minutes.
	Missing      []string   `json:"missing"`
	MissingSince *time.Time `json:"missingSince,omitempty"`
	// RateLimitedUntil: the store throttled ReputeCast; the sync slows down.
	RateLimitedUntil *time.Time      `json:"rateLimitedUntil,omitempty"`
	Import           *CommerceImport `json:"import,omitempty"`
	// RevokedReason: uninstalled (Shopify's app/uninstalled), revoked (a
	// refused token); empty when a person disconnected.
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	RevokedReason  string     `json:"revokedReason" enum:",uninstalled,revoked"`
	DisconnectedAt *time.Time `json:"disconnectedAt,omitempty"`
	DisconnectedBy string     `json:"disconnectedBy,omitempty"`
}

// CommerceImport is the first import's progress.
type CommerceImport struct {
	StartedAt time.Time `json:"startedAt"`
	StartedBy string    `json:"startedBy"`
	// Phase: products, customers, orders, matching, done.
	Phase       string           `json:"phase" enum:"products,customers,orders,matching,done"`
	Products    CommerceProgress `json:"products"`
	Collections int              `json:"collections"`
	Customers   CommerceProgress `json:"customers"`
	// Accepted joined the list (Known were contacts already, the rest are
	// new); NotAdded didn't accept marketing.
	Accepted int `json:"accepted"`
	Known    int `json:"known"`
	NotAdded int `json:"notAdded"`
	// CustomersSkipped: the platform has no customer list (WooCommerce:
	// consent comes with each order).
	CustomersSkipped bool             `json:"customersSkipped"`
	Orders           CommerceProgress `json:"orders"`
	Months           int              `json:"months"`
	Matching         CommerceProgress `json:"matching"`
	// MinutesLeft is the estimate for the orders still to import.
	MinutesLeft int        `json:"minutesLeft"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
}

// CommerceProgress is "5,120 of 7,880".
type CommerceProgress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// CommerceStoreList is a workspace's stores.
type CommerceStoreList struct {
	Stores []CommerceStoreSummary `json:"stores"`
	// CanManage: the caller holds integrations:manage; Helpers hold it.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// CommerceStorePage is a store's page in the integrations hub.
type CommerceStorePage struct {
	Store CommerceStoreSummary `json:"store"`
	// Facts of the header: the custom app, the API version, who connected.
	AppName      string     `json:"appName"`
	APIVersion   string     `json:"apiVersion"`
	ConnectedBy  string     `json:"connectedBy"`
	ConnectedAt  *time.Time `json:"connectedAt,omitempty"`
	Capabilities []string   `json:"capabilities"`
	PublicDomain string     `json:"publicDomain"`
	// Numbers for 30 days.
	Numbers CommerceNumbers `json:"numbers"`
	// UsedBy: automations started by the store's triggers, campaigns with
	// its blocks, and revenue in reports.
	UsedBy      []CommerceUse           `json:"usedBy"`
	Sync        CommerceSyncFacts       `json:"sync"`
	Consent     CommerceConsent         `json:"consent"`
	Attribution CommerceAttribution     `json:"attribution"`
	Webhooks    []CommerceWebhook       `json:"webhooks"`
	Activity    []CommerceActivityItem  `json:"activity"`
	Flows       []CommerceFlow          `json:"flows"`
	Plugin      *CommercePluginState    `json:"plugin,omitempty"`
	CanManage   bool                    `json:"canManage"`
	Helpers     []string                `json:"helpers"`
	Lists       []CommerceListOption    `json:"lists"`
	Recovered   CommerceRecoveredOrders `json:"recovered"`
}

// CommerceNumbers are the page's tiles.
type CommerceNumbers struct {
	Days int `json:"days"`
	// Orders and Revenue in the store (minor units of Currency).
	Orders   int64  `json:"orders"`
	Revenue  int64  `json:"revenue"`
	Currency string `json:"currency"`
	// Credited to email, and their share of the revenue (percent).
	Credited        int64   `json:"credited"`
	CreditedRevenue int64   `json:"creditedRevenue"`
	Share           float64 `json:"share"`
	// On the store's list.
	ListMembers int64 `json:"listMembers"`
}

// CommerceRecoveredOrders are the orders abandoned-cart emails brought back.
type CommerceRecoveredOrders struct {
	Orders  int64      `json:"orders"`
	Revenue int64      `json:"revenue"`
	Since   *time.Time `json:"since,omitempty"`
}

// CommerceUse is one thing that uses the store.
type CommerceUse struct {
	Kind   string `json:"kind" enum:"automation,campaign,reports"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

// CommerceSyncFacts is the Sync card.
type CommerceSyncFacts struct {
	Products       int64      `json:"products"`
	Collections    []string   `json:"collections"`
	Accepted       int64      `json:"accepted"`
	Known          int        `json:"known"`
	New            int        `json:"new"`
	NotAdded       int        `json:"notAdded"`
	OrdersImported int        `json:"ordersImported"`
	Months         int        `json:"months"`
	ImportFinished *time.Time `json:"importFinished,omitempty"`
	ImportMinutes  int        `json:"importMinutes"`
	Webhooks       int        `json:"webhooks"`
	WebhooksTotal  int        `json:"webhooksTotal"`
	ReconciledAt   *time.Time `json:"reconciledAt,omitempty"`
	Missed         int        `json:"missed"`
	// CustomersFromOrders: WooCommerce, where consent comes with orders.
	CustomersFromOrders bool `json:"customersFromOrders"`
}

// CommerceConsent is the Customers and consent card.
type CommerceConsent struct {
	ListID      string `json:"listId,omitempty"`
	ListName    string `json:"listName,omitempty"`
	ListMembers int64  `json:"listMembers"`
	// ListCreated: the store's connection created the list.
	ListCreated bool `json:"listCreated"`
	WriteBack   bool `json:"writeBack"`
	// WriteGranted: the app may pass unsubscribes back (Shopify's
	// write_customers); WriteBackSupported is false on WooCommerce.
	WriteGranted       bool `json:"writeGranted"`
	WriteBackSupported bool `json:"writeBackSupported"`
}

// CommerceAttribution is the Attribution card.
type CommerceAttribution struct {
	Days     int        `json:"days"`
	SetBy    string     `json:"setBy,omitempty"`
	SetAt    *time.Time `json:"setAt,omitempty"`
	TagLinks bool       `json:"tagLinks"`
	// TagHost is the storefront the links point to (shop.acme.io), and
	// TagExample a tagged link's query.
	TagHost    string `json:"tagHost"`
	TagExample string `json:"tagExample"`
	Param      string `json:"param"`
	// Last 30 days.
	Orders          int64                `json:"orders"`
	Revenue         int64                `json:"revenue"`
	Credited        int64                `json:"credited"`
	CreditedRevenue int64                `json:"creditedRevenue"`
	ByToken         int64                `json:"byToken"`
	ByEmail         int64                `json:"byEmail"`
	Rows            []CommerceRevenueRow `json:"rows"`
	// Others groups the campaigns past the top rows ("9 other campaigns").
	Others *CommerceRevenueRow `json:"others,omitempty"`
}

// CommerceRevenueRow is one email's credited orders.
type CommerceRevenueRow struct {
	Kind    string `json:"kind" enum:"campaign,automation,others"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Orders  int64  `json:"orders"`
	Revenue int64  `json:"revenue"`
}

// CommerceWebhook is one webhook of the store.
type CommerceWebhook struct {
	Topic        string     `json:"topic"`
	Status       string     `json:"status" enum:"pending,registered,missing,failed"`
	LastEventAt  *time.Time `json:"lastEventAt,omitempty"`
	MissingSince *time.Time `json:"missingSince,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// CommerceActivityItem is one entry of the store's activity log. The app
// words it from its facts.
type CommerceActivityItem struct {
	ID   string    `json:"id"`
	At   time.Time `json:"at"`
	Kind string    `json:"kind" enum:"order,refund,checkout_abandoned,checkout_not_emailed,reconciliation,webhook_missing,webhook_registered,customer_unsubscribed,unsubscribe_passed,import_started,import_finished,connected,reconnected,disconnected,revoked,rate_limited,sync_error"`
	Tone string    `json:"tone" enum:"neutral,ok,info,warn,bad"`
	// The order ("#4421") and its amount.
	Order    string `json:"order,omitempty"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency,omitempty"`
	// The person: a contact's name, or the customer's who accepted
	// marketing; never the email of anyone else.
	Customer  string `json:"customer,omitempty"`
	ContactID string `json:"contactId,omitempty"`
	// Accepted marketing at checkout; Contact: the buyer is a contact.
	Accepted bool `json:"accepted"`
	Contact  bool `json:"contact"`
	// Lists joined and automations started.
	Lists []string `json:"lists"`
	Flows []string `json:"flows"`
	// Attribution: how the order was matched and to which email.
	Matched      string `json:"matched,omitempty" enum:"click_token,email,none,"`
	Credited     string `json:"credited,omitempty"`
	CreditedKind string `json:"creditedKind,omitempty" enum:"campaign,automation,"`
	CreditedStep int    `json:"creditedStep,omitempty"`
	Window       int    `json:"window,omitempty"`
	Recovered    bool   `json:"recovered"`
	// Topic is the webhook the event came with.
	Topic  string `json:"topic,omitempty"`
	Source string `json:"source,omitempty"`
	By     string `json:"by,omitempty"`
	// Reconciliation: orders missed, products checked.
	Missed   int `json:"missed"`
	Products int `json:"products"`
	// Abandoned checkout: the automation and its delay.
	Flow  string `json:"flow,omitempty"`
	Hours int    `json:"hours,omitempty"`
	// Webhook registered again after polling while it was missing.
	Polled bool `json:"polled"`
	// Revoked: uninstalled or revoked; RateLimited: until.
	Reason  string     `json:"reason,omitempty"`
	Until   *time.Time `json:"until,omitempty"`
	Message string     `json:"message,omitempty"`
}

// CommerceActivityList is a page of the activity log.
type CommerceActivityList struct {
	Items []CommerceActivityItem `json:"items"`
	Next  string                 `json:"next,omitempty"`
}

// CommerceFlow is an automation started by the store's triggers.
type CommerceFlow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Kind   string `json:"kind"`
	// PauseReason says why the store paused it.
	PauseReason string `json:"pauseReason,omitempty"`
}

// CommercePluginState is WooCommerce's ReputeCast plugin on the store's site.
type CommercePluginState struct {
	Found     bool       `json:"found"`
	Skipped   bool       `json:"skipped"`
	Host      string     `json:"host"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
	Version   string     `json:"version,omitempty"`
	// Consent and Checkouts: the plugin's checkout checkbox and abandoned
	// checkout reporting are on.
	Consent   bool `json:"consent"`
	Checkouts bool `json:"checkouts"`
	// SiteID is the WordPress site's connection, for "Open the site".
	SiteID string `json:"siteId,omitempty"`
}

// CommerceListOption is a list customers can join.
type CommerceListOption struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Members int64  `json:"members"`
	Store   bool   `json:"store"`
}

// CommerceConnectState is the connect dialog.
type CommerceConnectState struct {
	Store CommerceStoreSummary `json:"store"`
	Step  string               `json:"step" enum:"address,app,keys,install,sync,key,webhooks,plugin,done"`
	// Shopify: the values to paste into the app.
	AppURL        string   `json:"appUrl"`
	RedirectURL   string   `json:"redirectUrl"`
	Scopes        []string `json:"scopes"`
	OptionalScope string   `json:"optionalScope"`
	// Reachable: the instance's public URL answers from the internet.
	Reachable          bool       `json:"reachable"`
	ReachableCheckedAt *time.Time `json:"reachableCheckedAt,omitempty"`
	ClientIDHint       string     `json:"clientIdHint,omitempty"`
	InstalledAt        *time.Time `json:"installedAt,omitempty"`
	Granted            []string   `json:"granted"`
	WriteGranted       bool       `json:"writeGranted"`
	// Error: the last refused client secret or key.
	Error string `json:"error" enum:",secret_refused,state_expired,not_installed"`
	// Step 5: the lists customers can join, and the new list's name.
	Lists       []CommerceListOption `json:"lists"`
	NewListName string               `json:"newListName"`
	Months      int                  `json:"months"`
	// WooCommerce.
	KeyHint  string               `json:"keyHint,omitempty"`
	Webhooks int                  `json:"webhooks"`
	Plugin   *CommercePluginState `json:"plugin,omitempty"`
	Demo     bool                 `json:"demo"`
	// Syncs is what syncs now, for "Change what syncs" on a connected store.
	Syncs CommerceSyncChoice `json:"syncs"`
}

// CommerceSyncChoice is what a store syncs and the list customers join.
type CommerceSyncChoice struct {
	Products  bool   `json:"products"`
	Customers bool   `json:"customers"`
	Orders    bool   `json:"orders"`
	Checkouts bool   `json:"checkouts"`
	WriteBack bool   `json:"writeBack"`
	ListID    string `json:"listId,omitempty"`
}

// CommerceInstall is where to approve the app.
type CommerceInstall struct {
	URL string `json:"url"`
}

// CommerceCatalog lists the stores' catalogs for the composer's picker.
type CommerceCatalog struct {
	Stores []CommerceCatalogStore `json:"stores"`
}

// CommerceCatalogStore is one store's catalog.
type CommerceCatalogStore struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Platform    string               `json:"platform"`
	Currency    string               `json:"currency"`
	Products    int64                `json:"products"`
	Collections []CommerceCollection `json:"collections"`
	Status      string               `json:"status"`
}

// CommerceCollection is a collection with its products.
type CommerceCollection struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Products int64  `json:"products"`
}

// CommerceProduct is a product, with its price and availability now.
type CommerceProduct struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	URL       string     `json:"url"`
	ImageURL  string     `json:"imageUrl"`
	ImageAlt  string     `json:"imageAlt"`
	Price     int64      `json:"price"`
	CompareAt int64      `json:"compareAt"`
	Currency  string     `json:"currency"`
	Available bool       `json:"available"`
	Inventory *int       `json:"inventory,omitempty"`
	Removed   bool       `json:"removed"`
	RemovedAt *time.Time `json:"removedAt,omitempty"`
}

// CommerceProductList is a page of a catalog.
type CommerceProductList struct {
	Items []CommerceProduct `json:"items"`
	Total int64             `json:"total"`
	Next  int               `json:"next,omitempty"`
}

// CommerceBlock is a store block's settings, as the composer keeps them on
// the block (rc-store).
type CommerceBlock struct {
	Store string `json:"store" format:"uuid"`
	// Mode: card (one product), chosen (picked products), collection (the
	// newest of a collection).
	Mode       string   `json:"mode" enum:"card,chosen,collection"`
	Products   []string `json:"products" maxItems:"12"`
	Collection string   `json:"collection,omitempty" maxLength:"64"`
	Count      int      `json:"count,omitempty" minimum:"0" maximum:"6"`
	SoldOut    string   `json:"soldOut,omitempty" enum:"hide,label,"`
	CompareAt  bool     `json:"compareAt,omitempty"`
	Button     string   `json:"button,omitempty" maxLength:"40"`
	Heading    string   `json:"heading,omitempty" maxLength:"120"`
}

// CommerceBlockRender is a store block drawn now, with what to flag.
type CommerceBlockRender struct {
	HTML      string                   `json:"html"`
	StoreName string                   `json:"storeName"`
	Platform  string                   `json:"platform"`
	Products  []CommerceProduct        `json:"products"`
	Removed   []CommerceRemovedProduct `json:"removed"`
	// SoldOut are products hidden or labelled because they're sold out.
	SoldOut []string `json:"soldOut"`
	// Disconnected: the store isn't connected; the block keeps its last prices.
	Disconnected bool `json:"disconnected"`
}

// CommerceRemovedProduct is a product a block names that the store deleted.
type CommerceRemovedProduct struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	RemovedAt *time.Time `json:"removedAt,omitempty"`
	Platform  string     `json:"platform"`
}

// CommerceContactStores is the contact profile's Store orders card.
type CommerceContactStores struct {
	Stores []CommerceContactStore `json:"stores"`
	Orders []CommerceContactOrder `json:"orders"`
}

// CommerceContactStore is the contact as a store's customer.
type CommerceContactStore struct {
	StoreID  string `json:"storeId"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Orders   int    `json:"orders"`
	Spent    int64  `json:"spent"`
	Currency string `json:"currency"`
	// Marketing is their consent in the store; OnList: on the store's list.
	Marketing string     `json:"marketing" enum:"subscribed,not_subscribed,unsubscribed,pending,none"`
	OnList    bool       `json:"onList"`
	ListName  string     `json:"listName,omitempty"`
	ConsentAt *time.Time `json:"consentAt,omitempty"`
}

// CommerceContactOrder is one of the contact's store orders.
type CommerceContactOrder struct {
	ID           string    `json:"id"`
	StoreID      string    `json:"storeId"`
	Store        string    `json:"store"`
	Platform     string    `json:"platform"`
	Name         string    `json:"name"`
	PlacedAt     time.Time `json:"placedAt"`
	Total        int64     `json:"total"`
	Refunded     int64     `json:"refunded"`
	Currency     string    `json:"currency"`
	Items        []string  `json:"items"`
	Accepted     bool      `json:"accepted"`
	Credited     string    `json:"credited,omitempty"`
	CreditedKind string    `json:"creditedKind,omitempty" enum:"campaign,automation,"`
}

// CommerceCampaignRevenue is the campaign report's store rows.
type CommerceCampaignRevenue struct {
	Rows []CommerceCampaignStoreRow `json:"rows"`
}

// CommerceCampaignStoreRow is one store's orders credited to a campaign.
type CommerceCampaignStoreRow struct {
	StoreID  string `json:"storeId"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Orders   int64  `json:"orders"`
	Revenue  int64  `json:"revenue"`
	Currency string `json:"currency"`
	Days     int    `json:"days"`
	ByToken  int64  `json:"byToken"`
	ByEmail  int64  `json:"byEmail"`
}

// CommerceRevenueSummary is Earn › Overview's "Store revenue from email".
type CommerceRevenueSummary struct {
	Days            int     `json:"days"`
	Stores          int     `json:"stores"`
	Orders          int64   `json:"orders"`
	Revenue         int64   `json:"revenue"`
	Credited        int64   `json:"credited"`
	CreditedRevenue int64   `json:"creditedRevenue"`
	Share           float64 `json:"share"`
	Currency        string  `json:"currency"`
}

// CommerceAutomationOptions are the store triggers' choices.
type CommerceAutomationOptions struct {
	Stores []CommerceTriggerStore `json:"stores"`
}

// CommerceTriggerStore is a store a trigger can watch.
type CommerceTriggerStore struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Address     string               `json:"address"`
	Platform    string               `json:"platform"`
	Status      string               `json:"status"`
	Currency    string               `json:"currency"`
	Checkouts   bool                 `json:"checkouts"`
	Products    []CommerceProductRef `json:"products"`
	Collections []CommerceCollection `json:"collections"`
}

// CommerceProductRef names a product.
type CommerceProductRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// CommerceTriggerStats are an abandoned-checkout trigger's numbers since
// its automation went live.
type CommerceTriggerStats struct {
	Since     *time.Time `json:"since,omitempty"`
	Abandoned int64      `json:"abandoned"`
	WithEmail int64      `json:"withEmail"`
	Entered   int64      `json:"entered"`
	NoConsent int64      `json:"noConsent"`
	Skipped   int64      `json:"skipped"`
	// Recovered orders credited to the automation, and their revenue.
	Orders   int64  `json:"orders"`
	Revenue  int64  `json:"revenue"`
	Currency string `json:"currency"`
}
