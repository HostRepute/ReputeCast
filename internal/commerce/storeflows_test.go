package commerce_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
	"github.com/hostrepute/reputecast/internal/stores/shopify/shopifytest"
	"github.com/hostrepute/reputecast/internal/tracker"
)

// Store flows from the site tracker (M62, G-29–G-31) on a connected
// Shopify store: browse abandonment from tracker.tick, back in stock and
// price drops from product changes (the inventory webhook included), the
// stock-request API, and the per-recipient store blocks.

// liveFlow adds a live automation started by trigger.
func (s *shop) liveFlow(trigger map[string]any) uuid.UUID {
	s.t.Helper()
	id := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO automations (id, workspace_id, name, status, live_version, draft_version, trigger) VALUES ($1, $2, $3, 'live', 1, 1, $4)`,
		id, s.ws.ID, trigger["kind"], mustJSON(trigger))
	s.exec(`INSERT INTO automation_versions (automation_id, version, graph, published_at) VALUES ($1, 1, '{"nodes":[],"edges":[]}', $2)`, id, s.clk.now().Add(-24*time.Hour))
	return id
}

func (s *shop) contact(email string) uuid.UUID {
	s.t.Helper()
	return one[uuid.UUID](s.t, s, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = $2`, s.ws.ID, email)
}

func (s *shop) product(external string) uuid.UUID {
	s.t.Helper()
	return one[uuid.UUID](s.t, s, `SELECT id FROM store_products WHERE store_id = $1 AND external_id = $2`, s.id, external)
}

// viewed records a contact's view of a product at a time, as the tracker's
// collect writer keeps it.
func (s *shop) viewed(contact, product uuid.UUID, views int, at time.Time) {
	s.t.Helper()
	s.exec(`INSERT INTO tracker_product_views (workspace_id, contact_id, product_id, views, last_viewed_at) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (contact_id, product_id) DO UPDATE SET views = EXCLUDED.views, last_viewed_at = EXCLUDED.last_viewed_at`, s.ws.ID, contact, product, views, at)
}

func (s *shop) events(kind string) int64 {
	s.t.Helper()
	return one[int64](s.t, s, `SELECT count(*) FROM automation_events WHERE kind = $1`, kind)
}

// changes are the queued commerce.product_changed jobs, oldest first.
func (s *shop) changes() []commerce.ProductChangedArgs {
	s.t.Helper()
	rows, err := s.env.App.Pool.Query(context.Background(), `SELECT args FROM river_job WHERE kind = 'commerce.product_changed' ORDER BY id`)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	var out []commerce.ProductChangedArgs
	for rows.Next() {
		var raw []byte
		var a commerce.ProductChangedArgs
		if err := rows.Scan(&raw); err != nil || json.Unmarshal(raw, &a) != nil {
			s.t.Fatalf("job args: %v", err)
		}
		out = append(out, a)
	}
	return out
}

func TestBrowseAbandonment(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	ctx := context.Background()
	s.liveFlow(map[string]any{"kind": "viewed_product", "browse": map[string]any{"storeId": s.id, "afterHours": 4, "minViews": 1}})
	flows := &tracker.Flows{Pool: s.env.App.Pool, Q: s.env.App.Q, Now: s.clk.now}
	graceID, samID := s.contact(grace), s.contact(sam)
	mug := s.product("8841207")
	gone := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO contacts (id, workspace_id, email, status, consent, source) VALUES ($1, $2, 'gone@example.com', 'unsubscribed', 'double', 'import:csv')`, gone, s.ws.ID)
	viewAt := s.clk.now()
	for _, c := range []uuid.UUID{graceID, samID, gone} {
		s.viewed(c, mug, 2, viewAt)
	}
	// Sam ordered after viewing.
	s.clk.add(time.Hour)
	if st := s.deliver(s.order(&shopifytest.Order{ID: "5800000000099", Number: 4499, Email: sam, CustomerID: "7003", CreatedAt: s.clk.now(), UpdatedAt: s.clk.now(),
		Subtotal: "18.0", Total: "18.0", Items: []shopifytest.Item{{ProductID: "8841207", Title: "Ship It mug", Quantity: 1, Price: "18.0"}}})); st != 200 {
		t.Fatalf("order: %d", st)
	}
	if n, err := flows.Tick(ctx); err != nil || n != 0 {
		t.Fatalf("an hour after the view: %d %v", n, err)
	}
	// Four hours after the view: Grace only (Sam ordered, the other isn't active).
	s.clk.add(3*time.Hour + time.Minute)
	if n, err := flows.Tick(ctx); err != nil || n != 1 {
		t.Fatalf("after 4 hours: %d %v", n, err)
	}
	var ref string
	var payload []byte
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT ref, payload FROM automation_events WHERE kind = 'viewed_product' AND contact_id = $1`, graceID).Scan(&ref, &payload); err != nil {
		t.Fatal(err)
	}
	if ref != "browse:"+graceID.String()+":"+s.clk.now().UTC().Format(time.DateOnly) {
		t.Errorf("ref %q", ref)
	}
	var p struct {
		Automation string            `json:"automation"`
		Store      string            `json:"store"`
		Ctx        map[string]string `json:"ctx"`
		Liquid     struct {
			Browse struct {
				Products []struct{ Title, URL, Image, Price string } `json:"products"`
			} `json:"browse"`
		} `json:"liquid"`
	}
	if err := json.Unmarshal(payload, &p); err != nil || p.Store != s.id || p.Ctx["products"] != mug.String() || len(p.Liquid.Browse.Products) != 1 ||
		p.Liquid.Browse.Products[0].Title != "Ship It mug" || p.Liquid.Browse.Products[0].Price != "$18" {
		t.Fatalf("payload: %s", payload)
	}
	// The next minute, and a new view within 7 days: no second event.
	if n, _ := flows.Tick(ctx); n != 0 {
		t.Fatalf("the next minute: %d", n)
	}
	s.viewed(graceID, mug, 3, s.clk.now())
	s.clk.add(4*time.Hour + time.Minute)
	if n, _ := flows.Tick(ctx); n != 0 || s.events("viewed_product") != 1 {
		t.Fatalf("within 7 days: %d, %d events", n, s.events("viewed_product"))
	}

	// The Viewed products block reads the run's products when the email
	// goes out, its links included.
	vars := s.svc.Vars(ctx, p.Ctx)
	if vars["rc_viewed_0"] != "https://shop.acme.io/products/ship-it-mug" || vars["rc_viewed_0_title"] != "Ship It mug" || vars["rc_viewed_1"] != "" {
		t.Fatalf("viewed vars: %v", vars)
	}
	block := `<!--rc:store-items {"store":"` + s.id + `","mode":"viewed","products":[],"count":3}--><p>canvas</p><!--/rc:store-items-->`
	html := s.svc.Resolve(ctx, s.ws.ID, block, true)
	out, err := liquid.Render(html, liquid.Vars(grace, "Grace", "", nil, vars))
	if err != nil || !strings.Contains(out, "Ship It mug") || !strings.Contains(out, `href="https://shop.acme.io/products/ship-it-mug"`) || strings.Contains(out, "canvas") {
		t.Fatalf("viewed block: %v\n%s", err, out)
	}
	if empty, _ := liquid.Render(html, liquid.Vars(grace, "Grace", "", nil, nil)); strings.Contains(empty, "<table") {
		t.Fatalf("a message without viewed products shows the block: %s", empty)
	}
}

func TestBackInStockAndPriceDrop(t *testing.T) {
	s := newShop(t)
	// The app gets the stock scope; the cap's one variant has an inventory item.
	s.fake.Scopes = append(s.fake.Scopes, shopify.InventoryScope)
	capProduct := s.fake.Products[3]
	capProduct.InventoryItemID = "45120099"
	s.connect(fullSync)
	ctx := context.Background()
	if n := one[int64](t, s, `SELECT count(*) FROM store_webhooks WHERE store_id = $1 AND topic = $2 AND status = 'registered'`, s.id, shopify.InventoryTopic); n != 1 {
		t.Fatalf("inventory webhook: %d", n)
	}
	capID := s.product("8841177")
	graceID := s.contact(grace)

	// "Email me when it's back", through the API: an existing contact only.
	var req commerce.CommerceStockRequest
	s.c.Do("POST", s.path+"/commerce/stock-requests", map[string]any{"email": grace, "productId": "8841177"}).MustStatus(t, 200).JSON(t, &req)
	if req.ProductID != capID.String() || req.Available {
		t.Fatalf("request: %+v", req)
	}
	s.c.Do("POST", s.path+"/commerce/stock-requests", map[string]any{"email": "nobody@example.com", "productId": "8841177"}).MustStatus(t, 404)
	s.c.Do("POST", s.path+"/commerce/stock-requests", map[string]any{"email": grace, "productId": "99999"}).MustStatus(t, 404)
	if n := one[int64](t, s, `SELECT count(*) FROM audit_events WHERE action = 'store_stock_request.created' AND target_id = $1`, capID.String()); n != 1 {
		t.Fatalf("audit: %d", n)
	}

	// Back in stock: the inventory webhook reads the cap again.
	s.liveFlow(map[string]any{"kind": "back_in_stock", "stock": map[string]any{"storeId": s.id}})
	s.clk.add(time.Minute)
	capProduct.Inventory, capProduct.UpdatedAt = 5, s.clk.now()
	if st := s.deliver(s.fake.Deliver(shopify.InventoryTopic, s.fake.InventoryLevelPayload(capProduct))); st != 200 {
		t.Fatalf("inventory delivery: %d", st)
	}
	if !one[bool](t, s, `SELECT available FROM store_products WHERE id = $1`, capID) {
		t.Fatal("the cap isn't back")
	}
	jobs := s.changes()
	if len(jobs) != 1 || jobs[0].ProductID != capID.String() || jobs[0].Change != "back_in_stock" {
		t.Fatalf("queued: %+v", jobs)
	}
	if n, err := s.svc.ProductChanged(ctx, jobs[0]); err != nil || n != 1 {
		t.Fatalf("back in stock: %d %v", n, err)
	}
	if one[*time.Time](t, s, `SELECT notified_at FROM store_stock_requests WHERE contact_id = $1 AND product_id = $2`, graceID, capID) == nil {
		t.Fatal("the request isn't answered")
	}
	ref := "stock:" + capID.String() + ":" + s.clk.now().UTC().Format(time.DateOnly)
	if n := one[int64](t, s, `SELECT count(*) FROM automation_events WHERE kind = 'back_in_stock' AND contact_id = $1 AND ref = $2`, graceID, ref); n != 1 {
		t.Fatalf("back-in-stock events: %d", n)
	}
	if n, _ := s.svc.ProductChanged(ctx, jobs[0]); n != 0 {
		t.Fatalf("told twice: %d", n)
	}

	// Price drop: people who viewed the hoodie in the last 14 days, when it
	// falls by 10% or more.
	s.liveFlow(map[string]any{"kind": "price_drop", "stock": map[string]any{"storeId": s.id, "minDropPercent": 10}})
	hoodie := s.product("8841203")
	samID := s.contact(sam)
	s.viewed(samID, hoodie, 1, s.clk.now().Add(-48*time.Hour))
	if n, _ := s.svc.ProductChanged(ctx, commerce.ProductChangedArgs{ProductID: hoodie.String(), Change: "price_drop", DropPercent: 5, OldPrice: 5800}); n != 0 {
		t.Fatalf("a 5%% cut: %d", n)
	}
	s.clk.add(time.Minute)
	h := s.fake.Products[0]
	h.Price, h.UpdatedAt = "52.20", s.clk.now()
	if st := s.deliver(s.fake.Deliver("products/update", s.fake.ProductPayload(h))); st != 200 {
		t.Fatalf("product delivery: %d", st)
	}
	jobs = s.changes()
	last := jobs[len(jobs)-1]
	if last.Change != "price_drop" || last.DropPercent != 10 || last.OldPrice != 5800 {
		t.Fatalf("price drop queued: %+v", jobs)
	}
	if n, err := s.svc.ProductChanged(ctx, last); err != nil || n != 1 {
		t.Fatalf("a 10%% cut: %d %v", n, err)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM automation_events WHERE kind = 'price_drop' AND contact_id = $1 AND payload -> 'liquid' -> 'stock' ->> 'previous_price' = '$58'`, samID); n != 1 {
		t.Fatalf("price-drop events: %d", n)
	}
}

// queryCounter counts the statements a pool runs.
type queryCounter struct{ n atomic.Int64 }

func (c *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.n.Add(1)
	return ctx
}
func (c *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestPerRecipientBlocks(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	ctx := context.Background()
	graceID, samID := s.contact(grace), s.contact(sam)
	mug, stickers, hoodie := s.product("8841207"), s.product("8841215"), s.product("8841203")
	site := uuid.Must(uuid.NewV7())
	event := func(contact, product uuid.UUID, ago time.Duration) {
		s.exec(`INSERT INTO tracker_events (id, workspace_id, site_id, visitor_id, contact_id, kind, product_id, occurred_at) VALUES ($1, $2, $3, $4, $5, 2, $6, $7)`,
			uuid.Must(uuid.NewV7()), s.ws.ID, site, uuid.Must(uuid.NewV7()), contact, product, s.clk.now().Add(-ago))
	}
	// Grace viewed the mug, then the stickers; Sam the hoodie, which he bought.
	event(graceID, mug, 3*time.Hour)
	event(graceID, stickers, time.Hour)
	event(samID, hoodie, 72*time.Hour)

	block := `<!--rc:store-items {"store":"` + s.id + `","mode":"recently_viewed","products":[],"count":3}--><p>canvas</p><!--/rc:store-items-->`
	html := s.svc.Resolve(ctx, s.ws.ID, block, false)

	// One query for the page, through a pool that counts.
	cfg := s.env.App.Pool.Config().Copy()
	counter := &queryCounter{}
	cfg.ConnConfig.Tracer = counter
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// The pool's connect hook (db.AfterConnect) runs its own statements:
	// open the connection first, then count.
	if _, err := pool.Exec(ctx, `SELECT 1`); err != nil {
		t.Fatal(err)
	}
	counter.n.Store(0)
	counted := &commerce.Service{Pool: pool, Q: store.New(pool), Now: s.clk.now}
	readers := []liquid.Recipient{{ContactID: graceID}, {ContactID: samID}}
	vars := liquid.FillBatch(ctx, []liquid.RecipientData{counted.Recipients()}, s.ws.ID, readers, html)
	if n := counter.n.Load(); n != 1 {
		t.Fatalf("%d queries for the page, want 1", n)
	}
	render := func(c uuid.UUID) string {
		out, err := liquid.Render(html, liquid.Merge(liquid.Vars("x@example.com", "", "", nil, nil), vars[c]))
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	g, sm := render(graceID), render(samID)
	if !strings.Contains(g, "Feature Flag sticker pack") || !strings.Contains(g, "Ship It mug") || strings.Index(g, "Feature Flag") > strings.Index(g, "Ship It mug") {
		t.Fatalf("Grace's recently viewed, newest first:\n%s", g)
	}
	// Sam bought what he viewed: he gets the best sellers instead.
	if !strings.Contains(sm, "Acme Logo Hoodie") || strings.Contains(g, "Acme Logo Hoodie") {
		t.Fatalf("Sam's block:\n%s", sm)
	}
	if strings.Count(g, `class="rc-product"`) != 2 {
		t.Fatalf("Grace has two products: %s", g)
	}

	// Bought together, from the store's orders.
	if err := s.svc.RebuildPairs(ctx, uuid.MustParse(s.id)); err != nil {
		t.Fatal(err)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_product_pairs WHERE store_id = $1 AND product_id = $2 AND other_id = $3`, s.id, hoodie, stickers); n != 1 {
		t.Fatalf("hoodie and stickers bought together: %d", n)
	}
	// The canvas shows best sellers, with a note.
	r, err := s.svc.RenderBlock(ctx, &s.ws, commerce.CommerceBlock{Store: s.id, Mode: commerce.ModeBoughtTogether, Products: []string{}, Count: 3})
	if err != nil || r.Note == "" || len(r.Products) == 0 || r.Products[0].Title != "Acme Logo Hoodie" {
		t.Fatalf("canvas: %+v %v", r, err)
	}
}
