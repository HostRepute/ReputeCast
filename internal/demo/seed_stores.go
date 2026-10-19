package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/mjml"
)

func init() { seedStores = seedStoresImpl }

// The demo's Shopify store (design expansion phase 7, brief 4 "Shopify:
// Acme Merch"): shop.acme.io connected by Marcus Chen on 6 Oct through the
// custom app "ReputeCast", 38 products in 4 collections (and Feature Flag
// sticker pack, deleted on 13 Oct), 2,318 customers who accepted marketing
// on the new list "Acme Merch" (1,947 contacts already, 371 new), 7,880
// orders imported from the last 12 months and the orders since, the last
// 30 days' 412 orders and $21,836 with the 96 credited to email, the
// Abandoned cart flow live since 7 Oct, Grace Liu's recovered checkout,
// Sam Rivera's and Theo's orders without consent, the 13 Oct webhook
// incident, the store's activity log, notifications and audit entries.
// The store is marked demo: it's never polled, and what a person does to
// it answers from an in-process Shopify.

// Demo store credentials: the app's client ID (shown as 4f1c…a9e2) and the
// sealed secret and token the in-process Shopify accepts.
const (
	StoreClientID     = "4f1c8e2b7d03a9e2"
	StoreClientSecret = "shpss_demo1c8e2b7d03a9e2b5f0c4d6e8a1b3c5" //nolint:gosec // the demo's sample app, answered by an in-process fake
	storeAccessToken  = "shpat_demo5c0d2e7a9b1f3c4d6e8a0b2c4d6e8f" //nolint:gosec // the demo's sample token
)

// storeProduct is one of Acme Merch's products.
type storeProduct struct {
	id, title, handle string
	price, compareAt  int64 // dollars
	collections       []string
	inventory         *int
	created           int // days before 15 Oct
	removed           bool
}

func intp(n int) *int { return &n }

// Collections: Apparel 14, Desk 9, Stickers 8 and New arrivals 7 (live
// products), as the product picker counts them.
var storeCollections = []struct{ id, title, handle string }{
	{"501", "Apparel", "apparel"}, {"502", "Desk", "desk"}, {"503", "Stickers", "stickers"}, {"504", "New arrivals", "new-arrivals"},
}

var storeProducts = []storeProduct{
	// The newest first, as the picker lists them.
	{id: "8841203", title: "Acme Logo Hoodie", handle: "acme-logo-hoodie", price: 58, collections: []string{"501", "504"}, inventory: intp(21), created: 14},
	{id: "8841207", title: "Ship It mug", handle: "ship-it-mug", price: 18, collections: []string{"502", "504"}, inventory: intp(64), created: 15},
	{id: "8841215", title: "Feature Flag sticker pack", handle: "feature-flag-stickers", price: 9, collections: []string{"503", "504"}, created: 16, removed: true},
	{id: "8841190", title: "Boring Infrastructure tee", handle: "boring-infrastructure-tee", price: 28, compareAt: 32, collections: []string{"501"}, created: 17},
	{id: "8841177", title: "Deploy Friday cap", handle: "deploy-friday-cap", price: 24, collections: []string{"501"}, inventory: intp(0), created: 18},
	{id: "8841162", title: "Postgres desk mat", handle: "postgres-desk-mat", price: 32, collections: []string{"502"}, inventory: intp(4), created: 19},
	{id: "8841158", title: "Status Page desk lamp", handle: "status-page-desk-lamp", price: 45, collections: []string{"502", "504"}, inventory: intp(12), created: 20},
	{id: "8841151", title: "Acme Windbreaker", handle: "acme-windbreaker", price: 78, collections: []string{"501", "504"}, inventory: intp(9), created: 21},
	{id: "8841147", title: "Dark mode sticker pack", handle: "dark-mode-stickers", price: 9, collections: []string{"503", "504"}, created: 22},
	{id: "8841140", title: "Canary Release socks", handle: "canary-release-socks", price: 14, collections: []string{"501", "504"}, created: 23},
	{id: "8841133", title: "Acme Field Notes", handle: "acme-field-notes", price: 14, collections: []string{"504"}, created: 24},
	{id: "8841126", title: "Ship It crewneck", handle: "ship-it-crewneck", price: 52, collections: []string{"501"}, created: 40},
	{id: "8841119", title: "Feature Flag tee", handle: "feature-flag-tee", price: 28, collections: []string{"501"}, created: 52},
	{id: "8841112", title: "Uptime Is a Feature tee", handle: "uptime-is-a-feature-tee", price: 28, collections: []string{"501"}, created: 64},
	{id: "8841105", title: "Acme Zip Hoodie", handle: "acme-zip-hoodie", price: 64, collections: []string{"501"}, created: 76},
	{id: "8841098", title: "Rollback beanie", handle: "rollback-beanie", price: 22, collections: []string{"501"}, created: 88},
	{id: "8841091", title: "Postgres Elephant tee", handle: "postgres-elephant-tee", price: 28, collections: []string{"501"}, created: 100},
	{id: "8841084", title: "On-call Survivor tee", handle: "on-call-survivor-tee", price: 28, collections: []string{"501"}, created: 112},
	{id: "8841077", title: "Acme Logo cap", handle: "acme-logo-cap", price: 24, collections: []string{"501"}, created: 124},
	{id: "8841070", title: "Incident Commander tee", handle: "incident-commander-tee", price: 30, collections: []string{"501"}, created: 136},
	{id: "8841063", title: "Acme Notebook", handle: "acme-notebook", price: 16, collections: []string{"502"}, created: 148},
	{id: "8841056", title: "Latency Matters coaster set", handle: "latency-matters-coasters", price: 12, collections: []string{"502"}, created: 160},
	{id: "8841049", title: "Merge Conflict mug", handle: "merge-conflict-mug", price: 18, collections: []string{"502"}, created: 172},
	{id: "8841042", title: "Acme water bottle", handle: "acme-water-bottle", price: 26, collections: []string{"502"}, created: 184},
	{id: "8841035", title: "Keyboard wrist rest", handle: "keyboard-wrist-rest", price: 29, collections: []string{"502"}, created: 196},
	{id: "8841028", title: "Cable organizer", handle: "cable-organizer", price: 15, collections: []string{"502"}, created: 208},
	{id: "8841021", title: "Ship It sticker", handle: "ship-it-sticker", price: 3, collections: []string{"503"}, created: 220},
	{id: "8841014", title: "Acme logo sticker", handle: "acme-logo-sticker", price: 3, collections: []string{"503"}, created: 232},
	{id: "8841007", title: "Works on My Machine sticker", handle: "works-on-my-machine-sticker", price: 3, collections: []string{"503"}, created: 244},
	{id: "8841000", title: "YAML Engineer sticker", handle: "yaml-engineer-sticker", price: 3, collections: []string{"503"}, created: 256},
	{id: "8840993", title: "99.99% sticker", handle: "four-nines-sticker", price: 3, collections: []string{"503"}, created: 268},
	{id: "8840986", title: "Laptop sticker sheet", handle: "laptop-sticker-sheet", price: 8, collections: []string{"503"}, created: 280},
	{id: "8840979", title: "Holographic Acme sticker", handle: "holographic-acme-sticker", price: 5, collections: []string{"503"}, created: 292},
	{id: "8840972", title: "Acme gift card", handle: "acme-gift-card", price: 25, created: 304},
	{id: "8840965", title: "Acme tote bag", handle: "acme-tote-bag", price: 22, created: 316},
	{id: "8840958", title: "Laptop sleeve", handle: "laptop-sleeve", price: 39, created: 328},
	{id: "8840951", title: "Enamel pin set", handle: "enamel-pin-set", price: 12, created: 340},
	{id: "8840944", title: "Poster: The Stack", handle: "poster-the-stack", price: 20, created: 352},
	{id: "8840937", title: "Acme backpack", handle: "acme-backpack", price: 89, created: 364},
}

// storeRow is who an order is credited to.
type storeCredit struct {
	campaign   string // a campaign's name
	automation string // or an automation's
	node       string
	orders     int
	revenue    int64 // dollars
}

// The last 30 days' credited orders (the Attribution table): 96 orders and
// $5,184. "Weekly Digest #142" of the brief is scheduled in the demo, so
// its 6 orders are credited to #140; the nine other campaigns are the
// demo's sends of the last five weeks.
var storeCredits = []storeCredit{
	{campaign: "October Product Update", orders: 31, revenue: 1612},
	{campaign: "Weekly Digest #141", orders: 18, revenue: 902},
	{automation: "Welcome series", node: "n3", orders: 11, revenue: 598},
	{automation: "Abandoned cart", node: "n2", orders: 8, revenue: 472},
	{campaign: "Weekly Digest #140", orders: 6, revenue: 318},
}

var storeOtherCampaigns = []string{"Engineering blog roundup #2", "Beta Program · launch invite", "Feature flags preview", "Beta Program: October check-in",
	"Engineering blog roundup #1", "Weekly Digest #139", "Welcome to news.acme.io", "Weekly Digest #138", "Weekly Digest #137"}

const (
	storeOtherOrders  = 22
	storeOtherRevenue = 1282
	// The last 30 days: 412 orders, $21,836.
	storeMonthOrders  = 412
	storeMonthRevenue = 21836
	storeImported     = 7880
	storeCustomers    = 6240
	storeAccepted     = 2318
	storeKnown        = 1947
	storeNotAdded     = 3922
	// Abandoned checkouts since the flow went live on 7 Oct.
	storeAbandoned  = 104
	storeWithEmail  = 78
	storeEntered    = 44
	storeNoConsent  = 34
	storeEmailTwo   = 30
	storeRecovered  = 8
	storeInFlowNow  = 6
	storeTokenMatch = 89
)

type storeOrderRow struct {
	id               uuid.UUID
	external, name   string
	contact          *uuid.UUID
	accepted         bool
	placed           time.Time
	total, refunded  int64 // cents
	refundIDs        []string
	items            []commerce.CheckoutLine
	landing          string
	utm              [3]string
	checkout         string
	first            bool
	matched          string
	message          *uuid.UUID
	campaign         *uuid.UUID
	automation       *uuid.UUID
	node             string
	clicked          *time.Time
	source           string
	customerExternal string
}

type storeSeedState struct {
	s         *Seeder
	ctx       context.Context
	rng       *rand.Rand
	storeID   uuid.UUID
	listID    uuid.UUID
	source    string
	products  map[string]storeProduct
	byPrice   []storeProduct
	accepted  []storeCustomer // contacts on the list, by customer id
	orders    []storeOrderRow
	campaigns map[string]uuid.UUID
	flows     map[string]uuid.UUID
}

type storeCustomer struct {
	contact            uuid.UUID
	external           string
	email, first, last string
}

// day is a moment of the sample's calendar (today is Wed 15 Oct, 10:00 in
// New York; see earnDay).
func (g *storeSeedState) day(d, hour, minute int) time.Time {
	return g.s.earnDay(2026, 10, d, hour, minute)
}

func (g *storeSeedState) month(m time.Month, d, hour, minute int) time.Time {
	return g.s.earnDay(2026, m, d, hour, minute)
}

func seedStoresImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	// The plugin's 1.4.0 (WooCommerce) ships with this release: the staging
	// site runs the latest, as the WordPress boards show it.
	if _, err := s.Tx.Exec(ctx, `UPDATE wordpress_sites SET plugin_version = '1.4.0' WHERE workspace_id = $1 AND address = 'staging.acme.io'`, ws.ID); err != nil {
		return err
	}
	marcus := s.Users["marcus@acme.io"]
	if marcus.ID == uuid.Nil || ws.ID == uuid.Nil {
		return nil
	}
	g := &storeSeedState{s: s, ctx: ctx, rng: audRand(2026, 1006), storeID: ID(), products: map[string]storeProduct{}, campaigns: map[string]uuid.UUID{}, flows: map[string]uuid.UUID{}}
	g.source = "store:shopify:" + g.storeID.String()
	for _, p := range storeProducts {
		g.products[p.id] = p
		if !p.removed && p.price >= 8 && p.id != "8840972" {
			g.byPrice = append(g.byPrice, p)
		}
	}
	for name, id := range s.Campaigns {
		g.campaigns[name] = id
	}
	rows, err := s.Tx.Query(ctx, `SELECT name, id FROM automations WHERE workspace_id = $1`, ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		var id uuid.UUID
		if err := rows.Scan(&name, &id); err != nil {
			rows.Close()
			return err
		}
		g.flows[name] = id
	}
	rows.Close()
	steps := []func() error{g.store, g.catalog, g.list, g.customers, g.flow, g.ordersAndCheckouts, g.blocks, g.history}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// store inserts Acme Merch, connected on 6 Oct at 16:04 by Marcus Chen.
func (g *storeSeedState) store() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	marcus := s.Users["marcus@acme.io"].ID
	sealed, err := commerce.SealCredentials(s.Box, g.storeID, StoreClientSecret, storeAccessToken)
	if err != nil {
		return err
	}
	g.listID = ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO lists (id, workspace_id, name, description, double_opt_in, requires_approval, origin, created_at)
		VALUES ($1,$2,'Acme Merch','',false,false,'store',$3)`, g.listID, ws.ID, g.day(6, 16, 4)); err != nil {
		return err
	}
	s.Lists["Acme Merch"] = g.listID
	progress, _ := json.Marshal(map[string]any{"phase": "done", "months": 12, "by": "Marcus Chen",
		"products": map[string]int{"done": 39, "total": 39}, "collections": len(storeCollections),
		"customers": map[string]int{"done": storeCustomers, "total": storeCustomers}, "accepted": storeAccepted, "known": storeKnown, "notAdded": storeNotAdded,
		"orders": map[string]int{"done": storeImported, "total": storeImported}, "matching": map[string]int{"done": 4120, "total": 4120},
		"since": g.month(10, 6, 16, 4).AddDate(-1, 0, 0)})
	_, err = s.Tx.Exec(ctx, `INSERT INTO stores (id, workspace_id, platform, status, step, name, address, domain, public_domain, currency, api_version, app_name,
		client_id, credentials_sealed, scopes, installed_at, webhook_token, import_months, list_id, write_back, tag_links, attribution_days,
		attribution_set_by, attribution_set_at, last_event_at, health_checked_at, import_progress, import_started_at, import_started_by, import_finished_at,
		writeback_through, reconciled_at, reconcile_missed, reconcile_products, demo, connected_by, connected_at, created_by, created_at, updated_at)
		VALUES ($1,$2,'shopify','connected','done','Acme Merch','shop.acme.io','acme-merch.myshopify.com','shop.acme.io','USD','2026-10','ReputeCast',
		$3,$4,'{read_customers,read_orders,read_products,write_customers}',$5,$6,12,$7,true,true,5,$8,$9,$10,$11,$12,$9,$8,$13,$14,$15,0,38,true,$8,$9,$8,$16,$10)`,
		g.storeID, ws.ID, StoreClientID, sealed, g.day(6, 16, 1), "demo-"+g.storeID.String()[24:], g.listID, marcus, g.day(6, 16, 4),
		s.Ago(2*time.Minute), s.Ago(4*time.Minute), progress, g.day(6, 16, 42), s.Ago(time.Minute), g.day(15, 3, 0), g.day(6, 15, 51))
	if err != nil {
		return err
	}
	// The 11 webhooks (and the app/uninstalled one Shopify keeps out of
	// sight), with their last events; checkouts/update was deleted in
	// Shopify on 13 Oct at 14:20 and registered again by Marcus at 16:05.
	last := map[string]time.Time{
		"orders/create": s.Ago(2 * time.Minute), "orders/paid": s.Ago(2 * time.Minute), "orders/updated": s.Ago(2 * time.Minute),
		"refunds/create": g.day(14, 16, 40), "customers/create": s.Ago(18 * time.Minute), "customers/update": s.Ago(6 * time.Minute),
		"checkouts/create": s.Ago(9 * time.Minute), "checkouts/update": s.Ago(9 * time.Minute), "products/create": g.day(12, 11, 2),
		"products/update": g.day(14, 9, 30), "products/delete": g.day(13, 17, 45),
	}
	topics := []string{"orders/create", "orders/paid", "orders/updated", "refunds/create", "customers/create", "customers/update", "checkouts/create",
		"checkouts/update", "products/create", "products/update", "products/delete"}
	for i, t := range topics {
		registered, by := g.day(6, 16, 4), marcus
		if t == "checkouts/update" {
			registered = g.day(13, 16, 5)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO store_webhooks (store_id, topic, external_id, status, registered_at, registered_by, last_event_at, last_polled_at, position)
			VALUES ($1,$2,$3,'registered',$4,$5,$6,$7,$8)`, g.storeID, t, fmt.Sprintf("gid://shopify/WebhookSubscription/%d", 1449012300+i),
			registered, by, last[t], nilTime(t == "checkouts/update", g.day(13, 16, 0)), i); err != nil {
			return err
		}
	}
	_, err = s.Tx.Exec(ctx, `INSERT INTO store_webhooks (store_id, topic, external_id, status, registered_at, registered_by, position) VALUES ($1,'app/uninstalled',$2,'registered',$3,$4,100)`,
		g.storeID, "gid://shopify/WebhookSubscription/1449012399", g.day(6, 16, 4), marcus)
	return err
}

func nilTime(ok bool, t time.Time) *time.Time {
	if !ok {
		return nil
	}
	return &t
}

// catalog inserts the collections and the 39 products (38 live).
func (g *storeSeedState) catalog() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	for i, c := range storeCollections {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO store_collections (store_id, external_id, title, handle, position) VALUES ($1,$2,$3,$4,$5)`,
			g.storeID, c.id, c.title, c.handle, i); err != nil {
			return err
		}
	}
	var rows [][]any
	for _, p := range storeProducts {
		created := g.day(15, 9, 0).AddDate(0, 0, -p.created)
		available := p.inventory == nil || *p.inventory > 0
		cols := p.collections
		if cols == nil {
			cols = []string{}
		}
		var compare int64
		if p.compareAt > 0 {
			compare = p.compareAt * 100
		}
		updated := g.day(14, 9, 30)
		if p.created > 30 {
			updated = created.AddDate(0, 0, 3)
		}
		rows = append(rows, []any{ID(), g.storeID, ws.ID, p.id, p.title, p.handle, "https://shop.acme.io/products/" + p.handle, "", p.title, p.price * 100, compare, "USD",
			available, p.inventory, "active", cols, created, updated, g.day(6, 16, 9)})
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"store_products"}, []string{"id", "store_id", "workspace_id", "external_id", "title", "handle", "url", "image_url",
		"image_alt", "price_cents", "compare_at_cents", "currency", "available", "inventory", "status", "collections", "created_at_store", "updated_at_store", "synced_at"},
		pgx.CopyFromRows(rows))
	return err
}

// list joins 2,318 customers who accepted marketing to "Acme Merch". The
// sample keeps the Audience boards' 48,213 contacts, so the 371 the first
// import counted as new to ReputeCast are sample contacts too (none of the
// Beta Program, nobody the boards name); Grace Liu, who comes back from an
// abandoned checkout on 15 Oct, is a sample Grace renamed.
func (g *storeSeedState) list() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	named := []string{"sam.rivera@outlook.com"}
	for _, b := range audBoardContacts {
		named = append(named, b.email)
	}
	const eligible = `c.workspace_id = $1 AND c.status = 'active' AND c.erased_at IS NULL AND c.consent IN ('double', 'single') AND NOT (c.email = ANY($2))
		  AND NOT EXISTS (SELECT 1 FROM suppressions su WHERE su.workspace_id = c.workspace_id AND su.email = c.email)
		  AND NOT EXISTS (SELECT 1 FROM list_memberships lm JOIN lists l ON l.id = lm.list_id WHERE lm.contact_id = c.id AND l.name = 'Beta Program')
		  AND EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.status = 'subscribed')`
	// Grace: a sample contact no other part of the demo tells a story about.
	var grace storeCustomer
	var oldEmail string
	err := s.Tx.QueryRow(ctx, `SELECT c.id, c.email FROM contacts c WHERE `+eligible+` AND c.first_name = 'Grace' AND c.email LIKE '%@gmail.com'
		  AND NOT EXISTS (SELECT 1 FROM automation_runs x WHERE x.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM paid_subscriptions x WHERE x.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM earn_checkouts x WHERE x.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM form_submissions x WHERE x.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM poll_answers x WHERE x.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM survey_responses x WHERE x.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM unsubscribe_feedback x WHERE x.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM website_members x WHERE x.contact_id = c.id)
		ORDER BY md5(c.email) LIMIT 1`, ws.ID, named).Scan(&grace.contact, &oldEmail)
	if err != nil {
		return fmt.Errorf("stores: a sample Grace: %w", err)
	}
	grace.email, grace.first, grace.last = "grace.liu@example.com", "Grace", "Liu"
	if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET last_name = 'Liu', email = $2 WHERE id = $1`, grace.contact, grace.email); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE messages SET email = $2 WHERE contact_id = $1 AND email = $3`, grace.contact, grace.email, oldEmail); err != nil {
		return err
	}
	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.email, c.first_name, c.last_name FROM contacts c WHERE `+eligible+` AND c.id <> $4
		ORDER BY md5(c.email) LIMIT $3`, ws.ID, named, storeAccepted-1, grace.contact)
	if err != nil {
		return err
	}
	g.accepted = []storeCustomer{grace}
	for rows.Next() {
		var c storeCustomer
		if err := rows.Scan(&c.contact, &c.email, &c.first, &c.last); err != nil {
			rows.Close()
			return err
		}
		g.accepted = append(g.accepted, c)
	}
	rows.Close()
	if len(g.accepted) < storeAccepted {
		return fmt.Errorf("stores: %d contacts to join Acme Merch, need %d", len(g.accepted), storeAccepted)
	}
	// The first 371 are the ones the import counted as new; a few of them
	// joined since, from webhooks.
	fresh := storeAccepted - storeKnown
	var members, consents, customers [][]any
	for i, c := range g.accepted {
		c.external = fmt.Sprint(7001 + i)
		g.accepted[i].external = c.external
		consentAt := g.day(6, 16, 4).AddDate(0, 0, -(3 + g.rng.IntN(360))).Add(time.Duration(g.rng.IntN(600)) * time.Minute)
		joined := g.day(6, 16, 10).Add(time.Duration(i%30) * time.Minute)
		if i < fresh && i%9 == 0 && i > 0 {
			joined = g.day(7+i%8, 9+i%10, i%60)
			consentAt = joined
		}
		if c.email == grace.email {
			consentAt = g.month(3, 2, 18, 40)
		}
		level, detail := "single_opt_in", "single opt-in"
		if i%7 == 3 {
			level, detail = "confirmed_opt_in", "confirmed opt-in"
		}
		country := []string{"US", "US", "US", "CA", "GB", "DE", "FR", "AU", "NL", "BR"}[i%10]
		members = append(members, []any{g.listID, c.contact, "subscribed", g.source, joined, joined})
		consents = append(consents, []any{ID(), c.contact, g.listID, "subscribed", g.source, detail, "", consentAt})
		customers = append(customers, []any{ID(), g.storeID, g.s.WS().ID, c.external, c.contact, c.email, c.first, c.last, country, "subscribed", level, consentAt, joined,
			consentAt, joined})
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"list_memberships"}, []string{"list_id", "contact_id", "status", "source", "subscribed_at", "created_at"},
		pgx.CopyFromRows(members)); err != nil {
		return err
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"consent_events"}, []string{"id", "contact_id", "list_id", "action", "source", "detail", "note", "occurred_at"},
		pgx.CopyFromRows(consents)); err != nil {
		return err
	}
	_, err = s.Tx.CopyFrom(ctx, pgx.Identifier{"store_customers"}, []string{"id", "store_id", "workspace_id", "external_id", "contact_id", "email", "first_name",
		"last_name", "country", "marketing", "opt_in_level", "consent_at", "joined_list_at", "updated_at_store", "created_at"}, pgx.CopyFromRows(customers))
	return err
}

// customers links Sam Rivera, who ordered without accepting marketing: his
// profile shows the order, and he isn't added to Acme Merch.
func (g *storeSeedState) customers() error {
	s, ctx := g.s, g.ctx
	sam, ok := s.contactByEmail(ctx, "sam.rivera@outlook.com")
	if !ok {
		return nil
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO store_customers (id, store_id, workspace_id, external_id, contact_id, marketing, updated_at_store, created_at)
		VALUES ($1,$2,$3,'9993',$4,'not_subscribed',$5,$5)`, ID(), g.storeID, s.WS().ID, sam, g.day(13, 11, 24))
	return err
}

// The abandoned-cart emails.
const (
	cartEmailOne = "You left something behind"
	cartEmailTwo = "Still deciding?"
)

// flow creates Abandoned cart, live since 7 Oct: the checkout's items,
// a day's wait, the goal "Placed an order", a second email with the
// newest products, with its numbers (44 entered, 44 and 30 sent, 8 orders).
func (g *storeSeedState) flow() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	marcus := s.Users["marcus@acme.io"].ID
	id := ID()
	g.flows["Abandoned cart"] = id
	created, published := g.day(7, 9, 12), g.day(7, 9, 30)
	one, two := ID(), ID()
	sample := []commerce.CheckoutLine{{ProductID: "8841203", Title: "Acme Logo Hoodie", Variant: "Size M", Quantity: 1, Price: 5800},
		{ProductID: "8841207", Title: "Ship It mug", Quantity: 1, Price: 1800}}
	checkout := commerce.CheckoutBlockHTML(sample, 7600, "USD", "Return to checkout")
	mjOne := storeEmailMJML(cartEmailOne, "Your cart is saved.", `Hi {{ first_name | default: "there" }}, your cart at Acme Merch is saved. Here's what's in it:`, checkout)
	svc := &commerce.Service{Q: s.Q}
	grid, err := svc.RenderBlock(ctx, &ws, commerce.CommerceBlock{Store: g.storeID.String(), Mode: "collection", Collection: "504", Count: 3, SoldOut: "hide",
		Button: "Shop now", Heading: "New in the Acme Merch shop"})
	if err != nil {
		return err
	}
	mjTwo := storeEmailMJML(cartEmailTwo, "Your cart is still saved.", `Still deciding, {{ first_name | default: "there" }}? Your cart is saved for a few more days. And here's what's new in the shop:`,
		storeGridSection(g.storeID, grid.HTML))
	skip := 7
	window := &automations.AutomationSendWindow{Enabled: false}
	graph := automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			autoNode("n1", automations.NodeTrigger, 300, 24, autoFlowData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerAbandonedCheckout,
				Store: g.storeID.String(), AfterHours: 1, SkipDays: &skip, Reentry: "always"}}),
			autoNode("n2", automations.NodeEmail, 300, 160, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: one.String(), EmailVersion: 1, Subject: cartEmailOne, SendWindow: window}}),
			autoNode("n3", automations.NodeDelay, 300, 330, autoFlowData{Delay: autoDur(23, "hours")}),
			autoNode("n4", automations.NodeGoal, 300, 430, autoFlowData{Label: "Placed an order", Condition: &automations.AutomationCondition{Kind: automations.ConditionPlacedOrder, Store: g.storeID.String()}}),
			autoNode("n5", automations.NodeEmail, 300, 560, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: two.String(), EmailVersion: 1, Subject: cartEmailTwo, SendWindow: window}}),
			autoNode("n6", automations.NodeExit, 300, 730, autoFlowData{Label: "Leave the flow"}),
		},
		Edges: []automations.AutomationEdge{autoEdge("n1", "n2", ""), autoEdge("n2", "n3", ""), autoEdge("n3", "n4", ""), autoEdge("n4", "n5", ""), autoEdge("n5", "n6", "")},
	}
	if err := autoInsertFlow(ctx, s, autoFlowSeed{id: id, name: "Abandoned cart", description: "2 emails over a day", status: "live", template: "abandoned-cart", live: 1,
		createdBy: marcus, created: created, updated: published,
		versions: []autoVersionSeed{{graph: graph, by: marcus, created: created, published: &published, summary: "created from the Abandoned cart template"}}}); err != nil {
		return err
	}
	for _, e := range []struct {
		id         uuid.UUID
		name, subj string
		mj         string
	}{{one, `Abandoned cart · email 1 "You left something behind"`, cartEmailOne, mjOne}, {two, `Email 2 "Still deciding?"`, cartEmailTwo, mjTwo}} {
		res := mjml.Compile(e.mj)
		if len(res.Errors) > 0 {
			return fmt.Errorf("compile %s: %s", e.name, res.Errors[0].Message)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_emails (id, workspace_id, automation_id, name, latest_version, created_by, created_at, updated_at) VALUES ($1,$2,$3,$4,1,$5,$6,$6)`,
			e.id, ws.ID, id, e.name, marcus, created); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at) VALUES ($1,1,$2,'',$3,$4,$5,$6,$7)`,
			e.id, e.subj, e.mj, res.HTML, e.subj, marcus, created); err != nil {
			return err
		}
	}
	// Numbers since 7 Oct, spread over the days (autoInsertStats).
	from := 8 // 7 Oct, counted back from 15 Oct
	ses := s.Providers["ses"].String()
	return autoInsertStats(ctx, s, []autoNodeStats{
		{a: id, v: 1, node: "n1", from: from, c: autoCounters{"entered": storeEntered, "completed": storeEntered}},
		{a: id, v: 1, node: "n2", from: from, c: autoCounters{"entered": storeEntered, "completed": storeEntered, "sent": storeEntered, "delivered": storeEntered,
			"opened": 31, "opened_human": 27, "clicked": 14}, providers: map[string]int64{ses: storeEntered}},
		{a: id, v: 1, node: "n3", from: from, c: autoCounters{"entered": storeEntered, "completed": storeEntered - storeInFlowNow - 3, "waiting": storeEntered}},
		{a: id, v: 1, node: "n4", from: from, c: autoCounters{"entered": storeEntered - storeInFlowNow - 3, "goal_reached": storeRecovered, "completed": storeEmailTwo}},
		{a: id, v: 1, node: "n5", from: from - 1, c: autoCounters{"entered": storeEmailTwo, "completed": storeEmailTwo, "sent": storeEmailTwo, "delivered": storeEmailTwo,
			"opened": 18, "opened_human": 15, "clicked": 5}, providers: map[string]int64{ses: storeEmailTwo}},
		{a: id, v: 1, node: "n6", from: from - 1, c: autoCounters{"entered": storeEmailTwo, "exited": storeEmailTwo}},
	})
}

// storeEmailMJML is an abandoned-cart email in Acme Merch's voice.
func storeEmailMJML(subject, preview, intro, block string) string {
	return `<mjml><mj-head><mj-title>` + subject + `</mj-title><mj-preview>` + preview + `</mj-preview><mj-attributes><mj-all font-family="IBM Plex Sans, Arial, sans-serif"></mj-all></mj-attributes></mj-head>
<mj-body background-color="#F3F4F6" width="600px">
  <mj-section background-color="#ffffff" padding="24px 24px 8px"><mj-column>
    <mj-text font-size="13px" font-weight="700" color="#0F766E" padding="0px 0px 12px">Acme Merch</mj-text>
    <mj-text font-size="22px" font-weight="700" color="#0B1220" padding="0px 0px 10px">` + subject + `</mj-text>
    <mj-text font-size="15px" line-height="1.6" color="#374151" padding="0px 0px 12px">` + intro + `</mj-text>
  </mj-column></mj-section>
  ` + storeRawSection(block) + `
  <mj-section background-color="#ffffff" padding="8px 24px 24px"><mj-column>
    <mj-text font-size="12px" line-height="1.5" color="#5B6472">You get this because you agreed to emails from Acme Merch at checkout. <a href="{{ unsubscribe_url }}" style="color:#5B6472">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#5B6472">Email preferences</a></mj-text>
  </mj-column></mj-section>
</mj-body></mjml>`
}

// storeRawSection wraps a drawn block: a raw block in a white section.
func storeRawSection(inner string) string {
	if strings.HasPrefix(strings.TrimSpace(inner), "<mj-section") {
		return inner
	}
	return `<mj-section background-color="#ffffff" padding="0px 24px"><mj-column><mj-raw>` + inner + `</mj-raw></mj-column></mj-section>`
}

// storeGridSection is a product grid as the composer keeps it: the block's
// settings on the section, its drawing in a raw block.
func storeGridSection(store uuid.UUID, drawing string) string {
	m := storeItemsConfig.FindStringSubmatch(drawing)
	cfg := ""
	if len(m) > 1 {
		cfg = m[1]
	}
	return `<mj-section background-color="#ffffff" padding="0px" rc-store='` + strings.ReplaceAll(cfg, "'", "&#39;") + `' rc-name="Product grid"><mj-column><mj-raw>` + drawing + `</mj-raw></mj-column></mj-section>`
}

// ordersAndCheckouts inserts the orders: the last 30 days' 412 orders and
// $21,836 (96 credited: 89 by click token, 7 by email), the rest of the
// 7,880 imported over 12 months, the abandoned checkouts since 7 Oct and
// the flow's runs and emails.
func (g *storeSeedState) ordersAndCheckouts() error {
	s, ctx := g.s, g.ctx
	now := s.Now
	monthStart := now.Add(-30 * 24 * time.Hour)
	importEnd := g.day(6, 16, 4)
	sent := map[string]time.Time{}
	rows, err := s.Tx.Query(ctx, `SELECT name, id, started_at FROM campaigns WHERE workspace_id = $1 AND started_at IS NOT NULL`, s.WS().ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var name string
		var id uuid.UUID
		var at time.Time
		if err := rows.Scan(&name, &id, &at); err != nil {
			rows.Close()
			return err
		}
		g.campaigns[name], sent[name] = id, at
	}
	rows.Close()
	var month []storeOrderRow
	emailMatched := map[string][]int{"Weekly Digest #141": {3, 9}, "October Product Update": {12}, "others": {1, 5, 10}}
	for _, c := range storeCredits {
		amounts := storeSplit(g.rng, c.revenue, c.orders)
		switch {
		case c.automation == "Abandoned cart":
			storePin(amounts, len(amounts)-1, 76)
			for i, a := range amounts {
				at := g.day(8+i%7, 10+i%9, (i*13)%60)
				if i == len(amounts)-1 {
					at = g.day(15, 8, 40)
				}
				o := g.newOrder(at, a*100)
				g.creditFlow(&o, g.flows["Abandoned cart"], "n2", at.Add(-time.Duration(2+i)*time.Hour))
				if i == len(amounts)-1 {
					g.grace(&o)
				}
				month = append(month, o)
			}
		case c.automation != "":
			flow, ok := g.flows[c.automation]
			for i, a := range amounts {
				at := monthStart.Add(time.Duration(i*61+7) * time.Hour)
				o := g.newOrder(at, a*100)
				if ok {
					g.creditFlow(&o, flow, c.node, at.Add(-time.Duration(3+i%5)*time.Hour))
				}
				month = append(month, o)
			}
		default:
			cid, ok := g.campaigns[c.campaign]
			start := sent[c.campaign]
			if c.campaign == "October Product Update" {
				storePin(amounts, 0, 58)
			}
			step := time.Duration(60/c.orders+1) * time.Hour
			for i, a := range amounts {
				at := start.Add(2*time.Hour + time.Duration(i)*step)
				if at.After(now.Add(-15 * time.Minute)) {
					at = now.Add(-time.Duration(30+i) * time.Minute)
				}
				o := g.newOrder(at, a*100)
				if ok {
					g.creditCampaign(&o, cid, at.Add(-time.Duration(5+i%30)*time.Hour), slices.Contains(emailMatched[c.campaign], i))
					if c.campaign == "October Product Update" && i == 0 {
						g.sam(&o, cid)
					}
				}
				month = append(month, o)
			}
		}
	}
	for i, a := range storeSplit(g.rng, storeOtherRevenue, storeOtherOrders) {
		name := storeOtherCampaigns[i%len(storeOtherCampaigns)]
		at := sent[name].Add(time.Duration(30+i*7) * time.Hour)
		if at.Before(monthStart) {
			at = monthStart.Add(time.Duration(1+i) * time.Hour)
		}
		if at.After(now) {
			at = now.Add(-time.Duration(3+i) * time.Hour)
		}
		o := g.newOrder(at, a*100)
		if cid, ok := g.campaigns[name]; ok {
			g.creditCampaign(&o, cid, at.Add(-26*time.Hour), slices.Contains(emailMatched["others"], i))
		}
		month = append(month, o)
	}
	// The rest isn't credited: 316 orders and $16,652, Theo's #4421 last.
	rest := storeMonthOrders - len(month)
	amounts := storeSplit(g.rng, storeMonthRevenue-5184, rest)
	storePin(amounts, rest-1, 64)
	for i, a := range amounts {
		at := monthStart.Add(time.Duration(float64(30*24*time.Hour) * (float64(i) + 0.5) / float64(rest)))
		if i == rest-1 {
			at = g.day(15, 9, 58)
		}
		o := g.newOrder(at, a*100)
		if i == rest-1 {
			g.theo(&o)
		} else {
			g.assignBuyer(&o, 0.4)
		}
		month = append(month, o)
	}
	// A mug refunded on 14 Oct at 16:40 (refunds/create): the 30 days'
	// revenue is net of it, so its order's total grows by as much.
	for i := range month {
		o := &month[i]
		if o.matched == "none" && o.placed.After(g.day(13, 12, 0)) && o.placed.Before(g.day(14, 12, 0)) {
			o.total += 1800
			o.refunded, o.refundIDs = 1800, []string{"9301240012"}
			o.items = append(o.items, commerce.CheckoutLine{ProductID: "8841207", Title: "Ship It mug", Quantity: 1, Price: 1800})
			break
		}
	}
	slices.SortFunc(month, func(a, b storeOrderRow) int { return a.placed.Compare(b.placed) })
	g.fixNames(month)
	imported := 0
	for i := range month {
		month[i].source = "webhook"
		if month[i].placed.Before(importEnd) {
			month[i].source = "import"
			imported++
		}
	}
	// The older imported orders, over the 12 months before the import.
	older := storeImported - imported
	yearStart := importEnd.AddDate(-1, 0, 0)
	span := monthStart.Sub(yearStart)
	for i := range older {
		at := yearStart.Add(time.Duration(float64(span) * (float64(i) + g.rng.Float64()) / float64(older)))
		o := g.newOrder(at, 0)
		o.name, o.source = fmt.Sprintf("#AM%d", 10001+i), "import"
		g.assignBuyer(&o, 0.45)
		g.orders = append(g.orders, o)
	}
	g.orders = append(g.orders, month...)
	g.firstOrders()
	if err := g.writeOrders(); err != nil {
		return err
	}
	return g.checkouts()
}

// storeSplit splits dollars over n orders: plausible totals that add up.
func storeSplit(rng *rand.Rand, total int64, n int) []int64 {
	out := make([]int64, n)
	if n == 0 {
		return out
	}
	avg := float64(total) / float64(n)
	var sum int64
	for i := range out {
		v := max(int64(avg*0.55+rng.Float64()*avg*0.9), 9)
		out[i] = v
		sum += v
	}
	storeSpread(out, total-sum, -1)
	return out
}

// storePin fixes one amount and spreads the difference over the others.
func storePin(amounts []int64, i int, value int64) {
	diff := amounts[i] - value
	amounts[i] = value
	storeSpread(amounts, diff, i)
}

// storeSpread adds diff dollars over the amounts (but skip), each staying
// at $9 or more.
func storeSpread(amounts []int64, diff int64, skip int) {
	for i := 0; diff != 0; i = (i + 1) % len(amounts) {
		if i == skip {
			continue
		}
		if diff > 0 {
			amounts[i]++
			diff--
		} else if amounts[i] > 9 {
			amounts[i]--
			diff++
		}
	}
}

// newOrder is an order of total cents (0: a random one) with items adding up.
func (g *storeSeedState) newOrder(at time.Time, total int64) storeOrderRow {
	if total == 0 {
		total = int64(12+g.rng.IntN(110)) * 100
	}
	o := storeOrderRow{id: ID(), external: fmt.Sprint(5800000000000 + g.rng.Int64N(99999999)), placed: at.Truncate(time.Second), total: total, matched: "none"}
	o.items = g.itemsFor(total)
	return o
}

// itemsFor fills an amount with the catalog's prices; the remainder goes
// to a gift card, as Shopify sells them in any amount.
func (g *storeSeedState) itemsFor(total int64) []commerce.CheckoutLine {
	var items []commerce.CheckoutLine
	left := total / 100
	for tries := 0; left > 0 && tries < 4; tries++ {
		var fits []storeProduct
		for _, p := range g.byPrice {
			if p.price <= left {
				fits = append(fits, p)
			}
		}
		if len(fits) == 0 {
			break
		}
		p := fits[g.rng.IntN(len(fits))]
		qty := int64(1)
		if p.price*2 <= left && g.rng.IntN(5) == 0 {
			qty = 2
		}
		variant := ""
		if slices.Contains(p.collections, "501") && strings.Contains(strings.ToLower(p.title), "hoodie") || strings.Contains(strings.ToLower(p.title), "tee") {
			variant = []string{"Size S", "Size M", "Size L", "Size XL"}[g.rng.IntN(4)]
		}
		items = append(items, commerce.CheckoutLine{ProductID: p.id, Title: p.title, Variant: variant, Quantity: int(qty), Price: p.price * 100})
		left -= p.price * qty
	}
	if left > 0 {
		items = append(items, commerce.CheckoutLine{ProductID: "8840972", Title: "Acme gift card", Variant: fmt.Sprintf("$%d", left), Quantity: 1, Price: left * 100})
	}
	return items
}

// assignBuyer makes share of orders a customer's who accepted marketing
// (a contact); the rest are guests or customers who didn't, kept without
// their email.
func (g *storeSeedState) assignBuyer(o *storeOrderRow, share float64) {
	if g.rng.Float64() < share {
		c := g.accepted[g.rng.IntN(len(g.accepted))]
		id := c.contact
		o.contact, o.accepted, o.customerExternal = &id, true, c.external
	}
}

func (g *storeSeedState) creditCampaign(o *storeOrderRow, campaign uuid.UUID, clicked time.Time, email bool) {
	c := g.accepted[g.rng.IntN(len(g.accepted))]
	id, msg := c.contact, ID()
	o.contact, o.accepted, o.customerExternal = &id, true, c.external
	o.campaign, o.message, o.clicked = &campaign, &msg, &clicked
	o.matched = "click_token"
	if email {
		o.matched = "email"
	} else {
		o.landing = "/collections/new-arrivals"
		o.utm = [3]string{"acme-newsletter", "email", "campaign"}
	}
}

func (g *storeSeedState) creditFlow(o *storeOrderRow, flow uuid.UUID, node string, clicked time.Time) {
	c := g.accepted[g.rng.IntN(len(g.accepted))]
	id, msg := c.contact, ID()
	o.contact, o.accepted, o.customerExternal = &id, true, c.external
	o.automation, o.node, o.message, o.clicked, o.matched = &flow, node, &msg, &clicked, "click_token"
	o.landing = "/cart"
	o.utm = [3]string{"acme-newsletter", "email", "abandoned-cart"}
}

// grace is Grace Liu's #4417: her checkout of 14 Oct 20:12, recovered by
// Abandoned cart's first email.
func (g *storeSeedState) grace(o *storeOrderRow) {
	c := g.accepted[0]
	id := c.contact
	o.name, o.contact, o.customerExternal = "#4417", &id, c.external
	o.items = []commerce.CheckoutLine{{ProductID: "8841203", Title: "Acme Logo Hoodie", Variant: "Size M", Quantity: 1, Price: 5800},
		{ProductID: "8841207", Title: "Ship It mug", Quantity: 1, Price: 1800}}
	o.checkout = "c4f2a9e7d1b34e6f8a0b2c3d4e5f6a7b"
	clicked := g.day(15, 8, 31)
	o.clicked = &clicked
}

// sam is Sam Rivera's #4409 of 13 Oct: no marketing box ticked, credited
// by email to October Product Update, whose link he clicked.
func (g *storeSeedState) sam(o *storeOrderRow, campaign uuid.UUID) {
	sam, ok := g.s.contactByEmail(g.ctx, "sam.rivera@outlook.com")
	if !ok {
		return
	}
	o.name, o.contact, o.customerExternal, o.accepted = "#4409", &sam, "9993", false
	o.placed = g.day(13, 11, 24)
	o.items = []commerce.CheckoutLine{{ProductID: "8841203", Title: "Acme Logo Hoodie", Variant: "Size L", Quantity: 1, Price: 5800}}
	o.total = 5800
	clicked := g.day(12, 10, 40)
	o.campaign, o.matched, o.clicked = &campaign, "email", &clicked
}

// theo is #4421, 15 Oct 09:58: theo.martin@example.com didn't accept
// marketing and isn't a contact, so ReputeCast keeps the order without him.
func (g *storeSeedState) theo(o *storeOrderRow) {
	o.name, o.contact, o.accepted = "#4421", nil, false
	o.items = []commerce.CheckoutLine{{ProductID: "8841203", Title: "Acme Logo Hoodie", Variant: "Size M", Quantity: 1, Price: 5800},
		{ProductID: "8841215", Title: "Feature Flag sticker pack", Quantity: 1, Price: 600}}
}

// fixNames keeps the named orders' numbers and gives the rest theirs.
func (g *storeSeedState) fixNames(month []storeOrderRow) {
	used := map[string]bool{}
	for _, o := range month {
		if o.name == "#4409" || o.name == "#4417" || o.name == "#4421" {
			used[o.name] = true
		}
	}
	n := 4421
	for i := len(month) - 1; i >= 0; i-- {
		switch month[i].name {
		case "#4409", "#4417", "#4421":
			continue
		}
		for used[fmt.Sprintf("#%d", n)] {
			n--
		}
		month[i].name = fmt.Sprintf("#%d", n)
		used[month[i].name] = true
		n--
	}
}

// firstOrders marks each contact's first order.
func (g *storeSeedState) firstOrders() {
	slices.SortFunc(g.orders, func(a, b storeOrderRow) int { return a.placed.Compare(b.placed) })
	seen := map[uuid.UUID]bool{}
	for i := range g.orders {
		if c := g.orders[i].contact; c != nil && !seen[*c] {
			seen[*c] = true
			g.orders[i].first = true
		}
	}
}

func (g *storeSeedState) writeOrders() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	var rows [][]any
	seen := map[string]bool{}
	for _, o := range g.orders {
		for seen[o.external] {
			o.external = fmt.Sprint(5800000000000 + g.rng.Int64N(99999999))
		}
		seen[o.external] = true
		items, _ := json.Marshal(o.items)
		refunds := o.refundIDs
		if refunds == nil {
			refunds = []string{}
		}
		var sub int64
		for _, it := range o.items {
			sub += it.Price * int64(it.Quantity)
		}
		rows = append(rows, []any{o.id, g.storeID, ws.ID, o.external, o.name, o.contact, o.customerExternal, o.accepted, o.placed, "USD", sub, o.total, o.refunded,
			refunds, []string{}, json.RawMessage(items), o.landing, o.utm[0], o.utm[1], o.utm[2], "", "", o.checkout, false, false, o.first, o.matched, o.message,
			o.campaign, o.automation, o.node, o.clicked, o.source, o.placed, o.placed, o.placed})
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"store_orders"}, []string{"id", "store_id", "workspace_id", "external_id", "name", "contact_id", "customer_external_id",
		"accepted_marketing", "placed_at", "currency", "subtotal_cents", "total_cents", "refunded_cents", "refund_ids", "discount_codes", "items", "landing_page",
		"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term", "checkout_external_id", "cancelled", "test", "first_order", "matched", "message_id",
		"campaign_id", "automation_id", "automation_node", "clicked_at", "source", "updated_at_store", "created_at", "updated_at"}, pgx.CopyFromRows(rows))
	return err
}

// checkouts inserts the abandoned checkouts since the flow went live (104:
// 78 with an email, 44 entered, 34 didn't accept marketing, 26 without an
// email; 8 recovered), the flow's runs, its emails and their deliveries.
func (g *storeSeedState) checkouts() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	flow := g.flows["Abandoned cart"]
	var recovered []storeOrderRow
	for _, o := range g.orders {
		if o.automation != nil && *o.automation == flow {
			recovered = append(recovered, o)
		}
	}
	slices.SortFunc(recovered, func(a, b storeOrderRow) int { return a.placed.Compare(b.placed) })
	var checkouts, entries, runs, messages, events [][]any
	add := func(token string, contact *uuid.UUID, external, email string, accepted bool, at time.Time, items []commerce.CheckoutLine, total int64, outcome string, order *storeOrderRow) uuid.UUID {
		id := ID()
		raw, _ := json.Marshal(items)
		var done *time.Time
		var orderID *uuid.UUID
		if order != nil {
			t, oid := order.placed, order.id
			done, orderID = &t, &oid
		}
		checkouts = append(checkouts, []any{id, g.storeID, ws.ID, token, contact, external, email != "", accepted, json.RawMessage(raw), total, total, "USD",
			"https://shop.acme.io/checkouts/" + token + "/recover", at, at, done, orderID, "webhook", at, at})
		entries = append(entries, []any{flow, id, outcome, at.Add(time.Hour)})
		return id
	}
	// The 44 who entered: Grace's checkout of 14 Oct 20:12 last; 8 of them
	// ordered.
	entered := 0
	for i := range storeEntered {
		var c storeCustomer
		var at time.Time
		var order *storeOrderRow
		items := g.itemsFor(int64(30+g.rng.IntN(90)) * 100)
		total := int64(0)
		if i < len(recovered) {
			order = &recovered[i]
			items = order.items
			c = g.customerOf(order.contact)
			at = order.placed.Add(-time.Duration(4+i%9) * time.Hour)
		} else {
			c = g.accepted[1+g.rng.IntN(len(g.accepted)-1)]
			at = g.day(7, 11, 0).Add(time.Duration(float64(7*24*time.Hour) * float64(i) / storeEntered))
		}
		if order != nil && order.name == "#4417" {
			at = g.day(14, 20, 12)
		}
		for _, it := range items {
			total += it.Price * int64(it.Quantity)
		}
		cid := c.contact
		token := fmt.Sprintf("%032x", g.rng.Uint64())[:32]
		if order != nil && order.checkout != "" {
			token = order.checkout
		}
		checkout := add(token, &cid, c.external, c.email, true, at, items, total, "entered", order)
		// Its run: goal for the 8 who ordered, finished after email 2 for 30,
		// waiting out the day for the last 6.
		emailed := at.Add(time.Hour)
		ctxJSON, _ := json.Marshal(map[string]any{"event": ID(), "ctx": map[string]string{"store": g.storeID.String(), "checkout": checkout.String()}})
		status, node := "completed", "n6"
		var next *time.Time
		var finished *time.Time
		switch {
		case order != nil:
			status, node = "goal", "n4"
			t := order.placed
			finished = &t
		case entered >= storeEntered-storeRecovered-storeInFlowNow:
			// Still waiting out the day when the demo opens.
			status, node = "waiting", "n3"
			t := emailed.Add(23 * time.Hour)
			if soon := g.s.Now.Add(time.Duration(2+entered%6) * time.Hour); t.Before(soon) {
				t = soon
			}
			next = &t
		default:
			t := emailed.Add(23 * time.Hour)
			finished = &t
		}
		if order == nil {
			entered++
		}
		runs = append(runs, []any{ID(), flow, 1, cid, node, status, json.RawMessage(ctxJSON), at.Add(time.Hour), at.Add(time.Hour), next, finished})
		msg := ID()
		if order != nil && order.message != nil {
			msg = *order.message
		}
		ref, _ := json.Marshal(map[string]any{"email": ID(), "version": 1, "ctx": map[string]string{"store": g.storeID.String(), "checkout": checkout.String()}})
		messages = append(messages, []any{msg, ws.ID, "automation", flow, "n2", cid, c.email, "delivered", emailed, emailed, emailed.Add(time.Minute), json.RawMessage(ref)})
		events = append(events, []any{ID(), ws.ID, msg, cid, "delivered", emailed.Add(time.Minute)})
		if status == "completed" {
			second := emailed.Add(23 * time.Hour)
			m2 := ID()
			messages = append(messages, []any{m2, ws.ID, "automation", flow, "n5", cid, c.email, "delivered", second, second, second.Add(time.Minute), json.RawMessage(ref)})
			events = append(events, []any{ID(), ws.ID, m2, cid, "delivered", second.Add(time.Minute)})
		}
		if order != nil && order.clicked != nil {
			events = append(events, []any{ID(), ws.ID, msg, cid, "click", *order.clicked})
		}
	}
	// 34 who didn't accept marketing: not emailed.
	for i := range storeNoConsent {
		at := g.day(7, 13, 0).Add(time.Duration(float64(8*24*time.Hour) * float64(i) / storeNoConsent))
		items := g.itemsFor(int64(20+g.rng.IntN(80)) * 100)
		var total int64
		for _, it := range items {
			total += it.Price * int64(it.Quantity)
		}
		add(fmt.Sprintf("%032x", g.rng.Uint64())[:32], nil, fmt.Sprint(9100+i), fmt.Sprintf("guest%d@example.net", i), false, at, items, total, "no_consent", nil)
	}
	// 26 without an email.
	for i := range storeAbandoned - storeWithEmail {
		at := g.day(7, 15, 0).Add(time.Duration(float64(8*24*time.Hour) * float64(i) / float64(storeAbandoned-storeWithEmail)))
		items := g.itemsFor(int64(15+g.rng.IntN(60)) * 100)
		var total int64
		for _, it := range items {
			total += it.Price * int64(it.Quantity)
		}
		add(fmt.Sprintf("%032x", g.rng.Uint64())[:32], nil, "", "", false, at, items, total, "no_email", nil)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"store_checkouts"}, []string{"id", "store_id", "workspace_id", "external_id", "contact_id", "customer_external_id",
		"has_email", "accepted_marketing", "items", "subtotal_cents", "total_cents", "currency", "recovery_url", "created_at_store", "updated_at_store", "completed_at",
		"order_id", "source", "created_at", "updated_at"}, pgx.CopyFromRows(checkouts)); err != nil {
		return err
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"store_checkout_entries"}, []string{"automation_id", "checkout_id", "outcome", "at"}, pgx.CopyFromRows(entries)); err != nil {
		return err
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"automation_runs"}, []string{"id", "automation_id", "version", "contact_id", "node_id", "status", "state", "entered_at",
		"node_entered_at", "next_at", "finished_at"}, pgx.CopyFromRows(runs)); err != nil {
		return err
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"messages"}, []string{"id", "workspace_id", "stream", "automation_id", "automation_node", "contact_id", "email",
		"status", "queued_at", "sent_at", "delivered_at", "content_ref"}, pgx.CopyFromRows(messages)); err != nil {
		return err
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"delivery_events"}, []string{"id", "workspace_id", "message_id", "contact_id", "type", "occurred_at"}, pgx.CopyFromRows(events))
	return err
}

func (g *storeSeedState) customerOf(id *uuid.UUID) storeCustomer {
	if id != nil {
		for _, c := range g.accepted {
			if c.contact == *id {
				return c
			}
		}
	}
	return g.accepted[1]
}

// blocks puts the product grid of the October Product Update (New
// arrivals, three products, as sent on the day) in its content. The
// Feature Flag sticker pack was deleted in Shopify afterwards, on 13 Oct.
func (g *storeSeedState) blocks() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	svc := &commerce.Service{Q: s.Q}
	r, err := svc.RenderBlock(ctx, &ws, commerce.CommerceBlock{Store: g.storeID.String(), Mode: "collection", Collection: "504", Count: 3, SoldOut: "hide",
		Button: "Shop now", Heading: "New in the Acme Merch shop"})
	if err != nil {
		return err
	}
	if id, ok := s.Campaigns["October Product Update"]; ok {
		if err := g.insertSection(id, storeGridSection(g.storeID, r.HTML)); err != nil {
			return err
		}
	}
	_, err = s.Tx.Exec(ctx, `UPDATE store_products SET removed_at = $2 WHERE store_id = $1 AND external_id = '8841215'`, g.storeID, g.day(13, 17, 45))
	return err
}

// insertSection adds a section before a sent campaign's footer, in its
// content and its last version.
func (g *storeSeedState) insertSection(id uuid.UUID, section string) error {
	s, ctx := g.s, g.ctx
	var src string
	if err := s.Tx.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE id = $1`, id).Scan(&src); err != nil {
		return err
	}
	at := strings.LastIndex(src, "<!-- rc:footer -->")
	if at < 0 {
		at = strings.LastIndex(src, "</mj-body>")
	}
	if at < 0 {
		return nil
	}
	next := src[:at] + section + "\n    " + src[at:]
	res := mjml.Compile(next)
	if len(res.Errors) > 0 {
		return fmt.Errorf("compile with the product grid: %s", res.Errors[0].Message)
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET mjml = $2, html = $3 WHERE id = $1`, id, next, res.HTML); err != nil {
		return err
	}
	_, err := s.Tx.Exec(ctx, `UPDATE campaign_versions SET mjml = $2, html = $3 WHERE campaign_id = $1 AND version = (SELECT max(version) FROM campaign_versions WHERE campaign_id = $1)`,
		id, next, res.HTML)
	return err
}

// history writes the store's activity log, the 13 Oct incident's
// notifications and the audit entries of connecting the store.
func (g *storeSeedState) history() error {
	s, ctx, ws := g.s, g.ctx, g.s.WS()
	marcus := s.Users["marcus@acme.io"]
	order := func(name string) (uuid.UUID, *uuid.UUID) {
		for _, o := range g.orders {
			if o.name == name {
				return o.id, o.contact
			}
		}
		return uuid.Nil, nil
	}
	entry := func(at time.Time, kind, tone string, f commerce.ActivityFacts, orderName string, contact *uuid.UUID) error {
		data, _ := json.Marshal(f)
		var oid *uuid.UUID
		if orderName != "" {
			id, c := order(orderName)
			if id != uuid.Nil {
				oid = &id
				if contact == nil {
					contact = c
				}
			}
		}
		_, err := s.Tx.Exec(ctx, `INSERT INTO store_activity (id, store_id, workspace_id, at, kind, tone, data, order_id, contact_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			ID(), g.storeID, ws.ID, at, kind, tone, data, oid, contact)
		return err
	}
	samID, _ := s.contactByEmail(ctx, "sam.rivera@outlook.com")
	grace := g.accepted[0].contact
	entries := []struct {
		at         time.Time
		kind, tone string
		f          commerce.ActivityFacts
		order      string
		contact    *uuid.UUID
	}{
		{g.day(6, 15, 51), "import_started", "info", commerce.ActivityFacts{By: marcus.Name}, "", nil},
		{g.day(6, 16, 4), "connected", "ok", commerce.ActivityFacts{By: marcus.Name}, "", nil},
		{g.day(6, 16, 42), "import_finished", "ok", commerce.ActivityFacts{Products: 38, Missed: 0}, "", nil},
		{g.day(13, 11, 24), "order", "neutral", commerce.ActivityFacts{Order: "#4409", Amount: 5800, Currency: "USD", Customer: "Sam Rivera", Contact: true,
			Matched: "email", Credited: "October Product Update", CreditedKind: "campaign", Window: 5, Topic: "orders/create", Source: "webhook"}, "#4409", &samID},
		{g.day(13, 14, 20), "webhook_missing", "warn", commerce.ActivityFacts{Topic: "checkouts/update"}, "", nil},
		{g.day(13, 16, 5), "webhook_registered", "ok", commerce.ActivityFacts{Topic: "checkouts/update", By: marcus.Name, Polled: true, Missed: 0}, "", nil},
		{g.day(14, 21, 12), "checkout_abandoned", "info", commerce.ActivityFacts{Amount: 7600, Currency: "USD", Customer: "Grace Liu", Accepted: true, Contact: true,
			Flow: "Abandoned cart", Hours: 1, Lists: []string{"Acme Merch"}, Flows: []string{"Abandoned cart"}}, "", &grace},
		{g.day(15, 3, 0), "reconciliation", "ok", commerce.ActivityFacts{Missed: 0, Products: 38}, "", nil},
		{g.day(15, 8, 40), "order", "ok", commerce.ActivityFacts{Order: "#4417", Amount: 7600, Currency: "USD", Customer: "Grace Liu", Accepted: true, Contact: true,
			Matched: "click_token", Credited: "Abandoned cart", CreditedKind: "automation", CreditedStep: 1, Window: 5, Recovered: true, Topic: "orders/create", Source: "webhook"},
			"#4417", &grace},
		{g.day(15, 9, 58), "order", "neutral", commerce.ActivityFacts{Order: "#4421", Amount: 6400, Currency: "USD", Topic: "orders/create", Source: "webhook", Matched: "none",
			Window: 5}, "#4421", nil},
	}
	for _, e := range entries {
		if err := entry(e.at, e.kind, e.tone, e.f, e.order, e.contact); err != nil {
			return err
		}
	}
	// The incident sent no notification the boards show: the Notifications
	// board's center stays as it is (store problems notify from the health
	// check, see commerce's jobs).
	// Audit: connecting the store, its list, the flow, the webhook
	// registered again.
	store := g.storeID.String()
	for _, a := range []auditSeed{
		{at: g.day(6, 15, 51), actorType: "user", actor: "marcus@acme.io", action: "store.connect_started", targetType: "store", targetID: store, target: "Acme Merch",
			after: map[string]any{"platform": "shopify", "address": "shop.acme.io"}},
		{at: g.day(6, 15, 56), actorType: "user", actor: "marcus@acme.io", action: "store.keys_saved", targetType: "store", targetID: store, target: "Acme Merch",
			after: map[string]any{"clientId": "4f1c…a9e2", "secret": "sealed"}},
		{at: g.day(6, 16, 1), actorType: "user", actor: "marcus@acme.io", action: "store.installed", targetType: "store", targetID: store, target: "Acme Merch",
			after: map[string]any{"scopes": []string{"read_customers", "read_orders", "read_products", "write_customers"}}},
		{at: g.day(6, 16, 4), actorType: "user", actor: "marcus@acme.io", action: "list.create", targetType: "list", targetID: g.listID.String(), target: "Acme Merch",
			after: map[string]any{"name": "Acme Merch", "origin": "store"}},
		{at: g.day(6, 16, 4), actorType: "user", actor: "marcus@acme.io", action: "store.connected", targetType: "store", targetID: store, target: "Acme Merch",
			after: map[string]any{"list": "Acme Merch", "orders": "12 months", "writeBack": true, "attributionDays": 5}},
		{at: g.day(7, 9, 12), actorType: "user", actor: "marcus@acme.io", action: "automation.create", targetType: "automation", targetID: g.flows["Abandoned cart"].String(),
			target: "Abandoned cart", after: map[string]any{"template": "abandoned-cart"}},
		{at: g.day(7, 9, 30), actorType: "user", actor: "marcus@acme.io", action: "automation.publish", targetType: "automation", targetID: g.flows["Abandoned cart"].String(),
			target: "Abandoned cart", after: map[string]any{"version": 1}},
		{at: g.day(13, 16, 5), actorType: "user", actor: "marcus@acme.io", action: "store.webhooks_registered", targetType: "store", targetID: store, target: "Acme Merch",
			after: map[string]any{"topics": []string{"checkouts/update"}}},
	} {
		if err := s.insertAudit(ctx, ws.ID, a); err != nil {
			return err
		}
	}
	// Sam clicked October Product Update before ordering (attribution by email).
	if opu, ok := s.Campaigns["October Product Update"]; ok && samID != uuid.Nil {
		var msg uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM messages WHERE campaign_id = $1 AND contact_id = $2 LIMIT 1`, opu, samID).Scan(&msg); err == nil {
			if _, err := s.Tx.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, type, occurred_at, url, mailbox_provider)
				VALUES ($1,$2,$3,$4,$5,'click',$6,'https://shop.acme.io/products/acme-logo-hoodie','outlook')`, ID(), ws.ID, msg, opu, samID, g.day(12, 10, 40)); err != nil {
				return err
			}
		}
	}
	return nil
}

// storeItemsConfig reads a drawn block's settings from its marker.
var storeItemsConfig = regexp.MustCompile(`<!--rc:store-items (\{.*?\})-->`)
