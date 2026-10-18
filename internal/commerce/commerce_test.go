package commerce_test

// Shopify against an in-memory Shopify (shopifytest), over the API and the
// webhook endpoint, on a fresh database: connecting, the first import,
// webhooks (signatures, replays, events out of order, refunds counted
// once), consent (only customers who accepted marketing join; unsubscribes
// both ways), attribution (the click token, then the email, within the
// window; one email per order), abandoned checkouts (the delay, the skip
// rule, an order before the delay), disconnecting and reconnecting, health
// (a missing webhook, rate limits, a revoked token) and store blocks at
// send time. Nothing reaches Shopify.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores/shopify/shopifytest"
)

// clock is the shop's and ReputeCast's "now".
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

type shop struct {
	t     *testing.T
	env   *apptest.Env
	fake  *shopifytest.Fake
	svc   *commerce.Service
	clk   *clock
	ws    store.Workspace
	c     *apptest.Client
	path  string // /workspaces/<slug>
	id    string // the store
	token string // its webhook token
}

const (
	grace = "grace.liu@example.com" // accepted marketing
	theo  = "theo.martin@example.com"
	sam   = "sam.rivera@example.com" // a contact already, didn't accept marketing
)

func newShop(t *testing.T) *shop {
	t.Helper()
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "marcus@acme.io", Name: "Marcus Chen"})
	ws := env.CreateWorkspace("Acme Newsletter", owner.ID)
	c := env.Client()
	c.SignIn("marcus@acme.io", "")
	clk := &clock{t: time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)}
	f := shopifytest.New()
	f.Now = clk.now
	seedShop(f, clk.now())
	srv := f.Server()
	t.Cleanup(srv.Close)
	svc := env.App.Commerce()
	svc.BaseURL, svc.HTTP, svc.Now = srv.URL, srv.Client(), clk.now
	s := &shop{t: t, env: env, fake: f, svc: svc, clk: clk, ws: ws, c: c, path: "/workspaces/" + ws.Slug}
	// Sam is a contact already, on a list of his own.
	s.exec(`INSERT INTO contacts (id, workspace_id, email, first_name, last_name, consent, source) VALUES ($1, $2, $3, 'Sam', 'Rivera', 'double', 'import:csv')`,
		uuid.Must(uuid.NewV7()), ws.ID, sam)
	return s
}

// seedShop fills the fake with Acme Merch's sample, dated before now.
func seedShop(f *shopifytest.Fake, now time.Time) {
	ago := func(d time.Duration) time.Time { return now.Add(-d) }
	day := 24 * time.Hour
	f.Collections = []*shopifytest.Collection{{ID: "501", Title: "Apparel", Handle: "apparel"}, {ID: "502", Title: "New arrivals", Handle: "new-arrivals"}}
	f.Products = []*shopifytest.Product{
		{ID: "8841203", Title: "Acme Logo Hoodie", Handle: "acme-logo-hoodie", Status: "ACTIVE", Price: "58.0", Tracked: true, Inventory: 21, Collections: []string{"501", "502"}, CreatedAt: ago(20 * day), UpdatedAt: ago(2 * day)},
		{ID: "8841207", Title: "Ship It mug", Handle: "ship-it-mug", Status: "ACTIVE", Price: "18.0", Collections: []string{"502"}, CreatedAt: ago(19 * day), UpdatedAt: ago(2 * day)},
		{ID: "8841215", Title: "Feature Flag sticker pack", Handle: "feature-flag-stickers", Status: "ACTIVE", Price: "9.0", Collections: []string{"502"}, CreatedAt: ago(18 * day), UpdatedAt: ago(2 * day)},
		{ID: "8841177", Title: "Deploy Friday cap", Handle: "deploy-friday-cap", Status: "ACTIVE", Price: "24.0", Tracked: true, Inventory: 0, Collections: []string{"501"}, CreatedAt: ago(90 * day), UpdatedAt: ago(2 * day)},
	}
	consent := ago(300 * day)
	f.Customers = []*shopifytest.Customer{
		{ID: "7001", Email: grace, FirstName: "Grace", LastName: "Liu", Country: "US", State: "SUBSCRIBED", Level: "SINGLE_OPT_IN", ConsentAt: &consent, CreatedAt: ago(300 * day), UpdatedAt: ago(3 * day)},
		{ID: "7002", Email: theo, FirstName: "Theo", LastName: "Martin", Country: "FR", State: "NOT_SUBSCRIBED", Level: "UNKNOWN", CreatedAt: ago(10 * day), UpdatedAt: ago(10 * day)},
		{ID: "7003", Email: sam, FirstName: "Sam", LastName: "Rivera", Country: "US", State: "NOT_SUBSCRIBED", Level: "UNKNOWN", CreatedAt: ago(5 * day), UpdatedAt: ago(5 * day)},
	}
	item := func(id, title, price string) shopifytest.Item {
		return shopifytest.Item{ProductID: id, Title: title, Quantity: 1, Price: price}
	}
	f.Orders = []*shopifytest.Order{
		// Too old for a 3-month import.
		{ID: "5800000000001", Number: 1201, Email: grace, CustomerID: "7001", CreatedAt: ago(200 * day), UpdatedAt: ago(200 * day), Subtotal: "28.0", Total: "28.0",
			Items: []shopifytest.Item{item("8841203", "Acme Logo Hoodie", "28.0")}},
		{ID: "5800000000002", Number: 4409, Email: sam, CustomerID: "7003", CreatedAt: ago(2 * day), UpdatedAt: ago(2 * day), Subtotal: "58.0", Total: "58.0",
			Items: []shopifytest.Item{item("8841203", "Acme Logo Hoodie", "58.0")}},
		{ID: "5800000000004", Number: 4421, Email: theo, CustomerID: "7002", CreatedAt: ago(time.Hour), UpdatedAt: ago(time.Hour), Subtotal: "64.0", Total: "64.0",
			Items: []shopifytest.Item{item("8841203", "Acme Logo Hoodie", "58.0"), item("8841215", "Feature Flag sticker pack", "6.0")}},
	}
}

func (s *shop) exec(sql string, args ...any) {
	s.t.Helper()
	if _, err := s.env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatalf("%s: %v", sql, err)
	}
}

func one[T any](t *testing.T, s *shop, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := s.env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// get fetches a browser navigation without following its redirect.
func (s *shop) get(target string) *http.Response {
	s.t.Helper()
	cl := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := cl.Get(target)
	if err != nil {
		s.t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp
}

// approve installs the app: the install call, Shopify's approval page and
// the redirect back.
func (s *shop) approve() *http.Response {
	s.t.Helper()
	var in commerce.CommerceInstall
	s.c.Do("POST", s.path+"/stores/"+s.id+"/install", nil).MustStatus(s.t, 200).JSON(s.t, &in)
	if !strings.Contains(in.URL, "/admin/oauth/authorize") || !strings.Contains(in.URL, "client_id="+s.fake.ClientID) {
		s.t.Fatalf("approval page: %s", in.URL)
	}
	u, _ := url.Parse(in.URL)
	return s.get(s.fake.Approve(s.svc.CallbackURL(), u.Query().Get("state")))
}

// connect runs the connect dialog to the end and the first import.
func (s *shop) connect(sync commerce.CommerceSyncInput) {
	s.t.Helper()
	var st commerce.CommerceConnectState
	s.c.Do("POST", s.path+"/stores", map[string]any{"platform": "shopify", "address": "acme-merch.myshopify.com"}).MustStatus(s.t, 200).JSON(s.t, &st)
	if st.Step != commerce.StepApp || st.AppURL == "" || st.RedirectURL == "" || st.Store.Name != "Acme Merch" {
		s.t.Fatalf("step 2: %+v", st)
	}
	s.id = st.Store.ID
	s.install()
	s.start(sync)
}

// install pastes the app's keys and approves it.
func (s *shop) install() {
	s.t.Helper()
	s.c.Do("PUT", s.path+"/stores/"+s.id+"/app-keys", map[string]any{"clientId": s.fake.ClientID, "clientSecret": s.fake.ClientSecret}).MustStatus(s.t, 200)
	if resp := s.approve(); resp.StatusCode != http.StatusSeeOther || strings.Contains(resp.Header.Get("Location"), "error=") {
		s.t.Fatalf("approval: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// start is step 5, then the first import.
func (s *shop) start(sync commerce.CommerceSyncInput) {
	s.t.Helper()
	s.c.Do("POST", s.path+"/stores/"+s.id+"/start", sync).MustStatus(s.t, 200)
	s.token = one[string](s.t, s, `SELECT webhook_token FROM stores WHERE id = $1`, s.id)
	s.importAll()
}

func (s *shop) importAll() {
	s.t.Helper()
	id := uuid.MustParse(s.id)
	for range 50 {
		if _, err := s.svc.Import(context.Background(), id, time.Minute); err != nil {
			s.t.Fatalf("import: %v", err)
		}
		if one[string](s.t, s, `SELECT status FROM stores WHERE id = $1`, s.id) == commerce.StatusConnected {
			return
		}
	}
	s.t.Fatal("the import didn't finish")
}

var fullSync = commerce.CommerceSyncInput{Products: true, Customers: true, NewList: true, Orders: true, Months: 3, Checkouts: true, WriteBack: true}

// deliver posts a signed webhook and applies what it stored.
func (s *shop) deliver(d shopifytest.Delivery) int {
	s.t.Helper()
	resp, err := http.DefaultClient.Do(d.Request(s.env.Server.URL + "/hooks/shopify/" + s.token))
	if err != nil {
		s.t.Fatal(err)
	}
	_ = resp.Body.Close()
	s.applyPending()
	return resp.StatusCode
}

// applyPending applies the stored events, as the River job does.
func (s *shop) applyPending() {
	s.t.Helper()
	rows, err := s.env.App.Pool.Query(context.Background(), `SELECT id FROM store_events WHERE status = 'pending' ORDER BY received_at`)
	if err != nil {
		s.t.Fatal(err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := s.svc.Apply(context.Background(), id); err != nil {
			s.t.Fatalf("apply: %v", err)
		}
	}
}

func (s *shop) order(o *shopifytest.Order) shopifytest.Delivery {
	return s.fake.Deliver("orders/create", s.fake.OrderPayload(o))
}

func TestConnectAndFirstImport(t *testing.T) {
	s := newShop(t)
	// A secret Shopify doesn't sign with is caught on the way back.
	var st commerce.CommerceConnectState
	s.c.Do("POST", s.path+"/stores", map[string]any{"platform": "shopify", "address": "acme-merch.myshopify.com"}).MustStatus(t, 200).JSON(t, &st)
	s.id = st.Store.ID
	s.c.Do("PUT", s.path+"/stores/"+s.id+"/app-keys", map[string]any{"clientId": s.fake.ClientID, "clientSecret": "not-a-secret"}).MustStatus(t, 422)
	s.c.Do("PUT", s.path+"/stores/"+s.id+"/app-keys", map[string]any{"clientId": s.fake.ClientID, "clientSecret": "shpss_wrong0000"}).MustStatus(t, 200)
	if resp := s.approve(); !strings.Contains(resp.Header.Get("Location"), "error=secret_refused") {
		t.Fatalf("a wrong secret: %s", resp.Header.Get("Location"))
	}
	s.c.Do("GET", s.path+"/stores/"+s.id+"/connect", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Step != commerce.StepKeys || st.Error != "secret_refused" {
		t.Fatalf("back at the keys: %s %s", st.Step, st.Error)
	}
	s.install()
	s.start(fullSync)
	topics := s.fake.WebhookTopics()
	if len(topics) != 12 {
		t.Fatalf("webhooks: %v", topics)
	}
	// What synced: the products, Grace (who accepted marketing) on the new
	// list, Theo nowhere, the orders of the last 3 months.
	if n := one[int64](t, s, `SELECT count(*) FROM store_products WHERE store_id = $1`, s.id); n != 4 {
		t.Fatalf("products: %d", n)
	}
	list := one[string](t, s, `SELECT l.name FROM lists l JOIN stores st ON st.list_id = l.id WHERE st.id = $1`, s.id)
	if list != "Acme Merch" {
		t.Fatalf("list: %s", list)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM list_memberships lm JOIN stores st ON st.list_id = lm.list_id WHERE st.id = $1 AND lm.status = 'subscribed'`, s.id); n != 1 {
		t.Fatalf("members: %d", n)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM contacts WHERE email = $1`, theo); n != 0 {
		t.Fatal("a customer who didn't accept marketing became a contact")
	}
	detail := one[string](t, s, `SELECT ce.detail FROM consent_events ce JOIN contacts c ON c.id = ce.contact_id WHERE c.email = $1 AND ce.action = 'subscribed'`, grace)
	if detail != "single opt-in" {
		t.Fatalf("consent record: %q", detail)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders WHERE store_id = $1`, s.id); n != 2 {
		t.Fatalf("orders imported: %d", n)
	}
	// Sam's order shows on his profile; Theo's counts without his email.
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders o JOIN contacts c ON c.id = o.contact_id WHERE c.email = $1`, sam); n != 1 {
		t.Fatal("Sam's order isn't on his profile")
	}
	for _, action := range []string{"store.connect_started", "store.keys_saved", "store.installed", "store.connected"} {
		if n := one[int64](t, s, `SELECT count(*) FROM audit_events WHERE action = $1`, action); n == 0 {
			t.Fatalf("no audit entry %s", action)
		}
	}
	// Secrets are write-only: nothing of them in any answer.
	page := s.c.Do("GET", s.path+"/stores/"+s.id, nil).MustStatus(t, 200)
	body := string(page.Body) + string(s.c.Do("GET", s.path+"/stores/"+s.id+"/connect", nil).Body)
	if strings.Contains(body, s.fake.ClientSecret) || strings.Contains(body, "shpat_") || strings.Contains(body, "shpss_") {
		t.Fatal("a secret came back")
	}
	var p commerce.CommerceStorePage
	page.JSON(t, &p)
	if p.Store.Sync.State != "connected" || p.Sync.Accepted != 1 || p.Sync.NotAdded != 2 || p.Sync.Products != 4 || len(p.Webhooks) != 11 {
		t.Fatalf("page: %+v %+v", p.Store.Sync, p.Sync)
	}
}

func TestWebhooksAreVerifiedStoredOnceAndAppliedInOrder(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	o := &shopifytest.Order{ID: "5800000000010", Number: 4430, Email: grace, CustomerID: "7001", CreatedAt: s.clk.now(), UpdatedAt: s.clk.now(),
		Subtotal: "76.0", Total: "76.0", Items: []shopifytest.Item{{ProductID: "8841203", Title: "Acme Logo Hoodie", Quantity: 1, Price: "58.0"}}}

	// A forged delivery is refused and never stored.
	d := s.order(o)
	d.Header.Set("X-Shopify-Hmac-Sha256", "bm90IGEgc2lnbmF0dXJl")
	if code := s.deliver(d); code != http.StatusUnauthorized {
		t.Fatalf("forged: %d", code)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_events WHERE topic = 'orders/create' AND status <> 'rejected'`); n != 0 {
		t.Fatalf("a forged event was kept to apply: %d", n)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders WHERE external_id = $1`, o.ID); n != 0 {
		t.Fatal("a forged order was applied")
	}

	// Delivered twice (Shopify retries): stored and applied once.
	d = s.fake.DeliverRaw("orders/create", mustJSON(s.fake.OrderPayload(o)), "evt-replay")
	if code := s.deliver(d); code != http.StatusOK {
		t.Fatalf("delivery: %d", code)
	}
	if code := s.deliver(d); code != http.StatusOK {
		t.Fatalf("replay: %d", code)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_events WHERE delivery_id = 'evt-replay'`); n != 1 {
		t.Fatalf("stored %d times", n)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders WHERE external_id = $1`, o.ID); n != 1 {
		t.Fatalf("orders: %d", n)
	}

	// A refund counts once, whichever event brings it first.
	o.Refunded, o.UpdatedAt = "18.0", s.clk.now().Add(time.Minute)
	s.deliver(s.fake.Deliver("refunds/create", s.fake.RefundPayload(o)))
	s.deliver(s.fake.Deliver("orders/updated", s.fake.OrderPayload(o)))
	if n := one[int64](t, s, `SELECT refunded_cents FROM store_orders WHERE external_id = $1`, o.ID); n != 1800 {
		t.Fatalf("refunded: %d", n)
	}
	// An older version of the order arriving late changes nothing.
	old := *o
	old.UpdatedAt, old.Total, old.Refunded = s.clk.now().Add(-time.Hour), "1.0", ""
	s.deliver(s.fake.Deliver("orders/updated", s.fake.OrderPayload(&old)))
	if n := one[int64](t, s, `SELECT total_cents FROM store_orders WHERE external_id = $1`, o.ID); n != 7600 {
		t.Fatalf("an older event won: %d", n)
	}
	if st := one[string](t, s, `SELECT status FROM store_events WHERE topic = 'orders/updated' ORDER BY id DESC LIMIT 1`); st != "stale" {
		t.Fatalf("stale event: %s", st)
	}
	// Nothing private is kept: the payload's address, phone, payment and note.
	raw := one[string](t, s, `SELECT row_to_json(o)::text FROM store_orders o WHERE external_id = $1`, o.ID)
	for _, private := range []string{"Example Street", "555 0100", "4242", "Leave at the door", grace} {
		if strings.Contains(raw, private) {
			t.Fatalf("kept %q: %s", private, raw)
		}
	}
	// A product deleted in Shopify is flagged, not forgotten.
	s.deliver(s.fake.Deliver("products/delete", map[string]any{"id": 8841215}))
	if one[*time.Time](t, s, `SELECT removed_at FROM store_products WHERE external_id = '8841215'`) == nil {
		t.Fatal("the deleted product isn't flagged")
	}
}

func TestConsentTravelsBothWays(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	listID := one[uuid.UUID](t, s, `SELECT list_id FROM stores WHERE id = $1`, s.id)
	status := func(email string) string {
		var st string
		_ = s.env.App.Pool.QueryRow(context.Background(), `SELECT lm.status FROM list_memberships lm JOIN contacts c ON c.id = lm.contact_id WHERE c.email = $1 AND lm.list_id = $2`, email, listID).Scan(&st)
		return st
	}
	// Theo accepts marketing in Shopify: he joins.
	theoC := s.fake.Customers[1]
	at := s.clk.now()
	theoC.State, theoC.Level, theoC.ConsentAt, theoC.UpdatedAt = "SUBSCRIBED", "CONFIRMED_OPT_IN", &at, at
	s.deliver(s.fake.Deliver("customers/update", s.fake.CustomerPayload(theoC)))
	if status(theo) != "subscribed" {
		t.Fatalf("Theo: %q", status(theo))
	}
	// A suppressed address never joins.
	s.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source) VALUES ($1, $2, 'blocked@example.com', 'complaint', 'manual')`, uuid.Must(uuid.NewV7()), s.ws.ID)
	blocked := &shopifytest.Customer{ID: "7009", Email: "blocked@example.com", FirstName: "B", State: "SUBSCRIBED", Level: "SINGLE_OPT_IN", ConsentAt: &at, CreatedAt: at, UpdatedAt: at}
	s.fake.Customers = append(s.fake.Customers, blocked)
	s.deliver(s.fake.Deliver("customers/create", s.fake.CustomerPayload(blocked)))
	if n := one[int64](t, s, `SELECT count(*) FROM contacts WHERE email = 'blocked@example.com'`); n != 0 {
		t.Fatal("a suppressed address became a contact")
	}

	// Grace unsubscribes in Shopify: she leaves the store's list.
	graceC := s.fake.Customers[0]
	s.clk.add(time.Minute)
	graceC.State, graceC.UpdatedAt = "UNSUBSCRIBED", s.clk.now()
	s.deliver(s.fake.Deliver("customers/update", s.fake.CustomerPayload(graceC)))
	if status(grace) != "unsubscribed" {
		t.Fatalf("Grace: %q", status(grace))
	}
	// An old consent arriving after doesn't bring her back.
	graceC.State, graceC.UpdatedAt = "SUBSCRIBED", s.clk.now().Add(-time.Hour)
	s.deliver(s.fake.Deliver("customers/update", s.fake.CustomerPayload(graceC)))
	if status(grace) != "unsubscribed" {
		t.Fatal("an older consent rejoined her")
	}

	// Theo unsubscribes here: Shopify is told (write_customers granted).
	theoID := one[uuid.UUID](t, s, `SELECT id FROM contacts WHERE email = $1`, theo)
	s.clk.add(time.Minute)
	s.exec(`UPDATE list_memberships SET status = 'unsubscribed', unsubscribed_at = $3 WHERE list_id = $1 AND contact_id = $2`, listID, theoID, s.clk.now())
	s.exec(`INSERT INTO consent_events (id, contact_id, list_id, action, source, occurred_at) VALUES ($1, $2, $3, 'unsubscribed', 'unsubscribe_page', $4)`,
		uuid.Must(uuid.NewV7()), theoID, listID, s.clk.now())
	s.clk.add(time.Minute)
	if err := s.svc.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := s.fake.ConsentUpdates(); len(got) != 1 || got[0] != "7002=UNSUBSCRIBED" {
		t.Fatalf("told Shopify: %v", got)
	}
	// Told once.
	if err := s.svc.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := s.fake.ConsentUpdates(); len(got) != 1 {
		t.Fatalf("told twice: %v", got)
	}
}

// click records a human click on a campaign email to contact, and the
// token the click redirect would add to a store link.
func (s *shop) click(contact uuid.UUID, campaign string, at time.Time) string {
	s.t.Helper()
	cid, mid := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO campaigns (id, workspace_id, name, status) VALUES ($1, $2, $3, 'sent')`, cid, s.ws.ID, campaign)
	s.exec(`INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, status) VALUES ($1, $2, 'broadcast', $3, $4, 'x@example.com', 'delivered')`,
		mid, s.ws.ID, cid, contact)
	s.exec(`INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, type, occurred_at, url) VALUES ($1, $2, $3, $4, $5, 'click', $6, 'https://shop.acme.io/')`,
		uuid.Must(uuid.NewV7()), s.ws.ID, mid, cid, contact, at)
	return clicktoken.Sign(s.env.App.Links, clicktoken.Click{WorkspaceID: s.ws.ID, MessageID: mid, ContactID: contact, At: at})
}

func TestAttribution(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	graceID := one[uuid.UUID](t, s, `SELECT id FROM contacts WHERE email = $1`, grace)
	samID := one[uuid.UUID](t, s, `SELECT id FROM contacts WHERE email = $1`, sam)
	now := s.clk.now()
	credited := func(number int) (matched, campaign string) {
		_ = s.env.App.Pool.QueryRow(context.Background(), `SELECT o.matched, coalesce(c.name, '') FROM store_orders o LEFT JOIN campaigns c ON c.id = o.campaign_id
			WHERE o.store_id = $1 AND o.name = $2`, s.id, "#"+itoa(number)).Scan(&matched, &campaign)
		return
	}
	place := func(number int, email, customer, landing string) {
		o := &shopifytest.Order{ID: "59000000" + itoa(number), Number: number, Email: email, CustomerID: customer, CreatedAt: s.clk.now(), UpdatedAt: s.clk.now(),
			Subtotal: "58.0", Total: "58.0", Landing: landing, Items: []shopifytest.Item{{ProductID: "8841203", Title: "Acme Logo Hoodie", Quantity: 1, Price: "58.0"}}}
		if code := s.deliver(s.order(o)); code != 200 {
			t.Fatalf("order %d: %d", number, code)
		}
	}

	// By the click token in the landing page, even for a guest.
	tok := s.click(graceID, "October Product Update", now.Add(-48*time.Hour))
	place(5001, "guest@example.com", "", "/products/acme-logo-hoodie?utm_source=acme-newsletter&"+clicktoken.Param+"="+tok)
	if m, c := credited(5001); m != "click_token" || c != "October Product Update" {
		t.Fatalf("by token: %s %s", m, c)
	}
	// By email: Sam's last click within 5 days.
	s.click(samID, "Weekly Digest #141", now.Add(-4*24*time.Hour))
	s.click(samID, "Weekly Digest #142", now.Add(-3*24*time.Hour))
	place(5002, sam, "7003", "")
	if m, c := credited(5002); m != "email" || c != "Weekly Digest #142" {
		t.Fatalf("by email, the last click: %s %s", m, c)
	}
	// One order, one email.
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders WHERE name = '#5002' AND campaign_id IS NOT NULL AND automation_id IS NOT NULL`); n != 0 {
		t.Fatal("an order credited twice")
	}
	// A window of 1 day leaves an older click out.
	s.c.Do("PATCH", s.path+"/stores/"+s.id+"/settings", map[string]any{"attributionDays": 1}).MustStatus(t, 200)
	place(5003, sam, "7003", "")
	if m, _ := credited(5003); m != "none" {
		t.Fatalf("outside the window: %s", m)
	}
	// A token from another workspace's signer, or a forged one, is ignored.
	place(5004, "guest2@example.com", "", "/?"+clicktoken.Param+"=forged")
	if m, _ := credited(5004); m != "none" {
		t.Fatalf("forged token: %s", m)
	}
	// The store page's numbers.
	var p commerce.CommerceStorePage
	s.c.Do("GET", s.path+"/stores/"+s.id, nil).MustStatus(t, 200).JSON(t, &p)
	if p.Attribution.ByToken != 1 || p.Attribution.ByEmail != 1 || p.Numbers.Credited != 2 {
		t.Fatalf("attribution: %+v", p.Attribution)
	}
}

// abandonedFlow is a live abandoned-cart automation of the store.
func (s *shop) abandonedFlow(hours int) uuid.UUID {
	s.t.Helper()
	id := uuid.Must(uuid.NewV7())
	trigger := mustJSON(map[string]any{"kind": "abandoned_checkout", "store": s.id, "afterHours": hours})
	s.exec(`INSERT INTO automations (id, workspace_id, name, status, live_version, draft_version, trigger) VALUES ($1, $2, 'Abandoned cart', 'live', 1, 1, $3)`, id, s.ws.ID, trigger)
	s.exec(`INSERT INTO automation_versions (automation_id, version, graph, published_at) VALUES ($1, 1, '{"nodes":[],"edges":[]}', $2)`, id, s.clk.now().Add(-24*time.Hour))
	return id
}

func TestAbandonedCheckouts(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	flow := s.abandonedFlow(1)
	// Checkouts from before the first import ended never start the flow.
	s.clk.add(3 * time.Hour)
	checkout := func(id, email, customer string, accepts bool, updated time.Time) {
		c := &shopifytest.Checkout{ID: id, Token: "tok" + id, Email: email, CustomerID: customer, AcceptsMarketing: accepts, Subtotal: "76.0", Total: "76.0",
			CreatedAt: updated, UpdatedAt: updated, Items: []shopifytest.Item{{ProductID: "8841203", Title: "Acme Logo Hoodie", Quantity: 1, Price: "58.0"}}}
		s.deliver(s.fake.Deliver("checkouts/update", s.fake.CheckoutPayload(c)))
	}
	outcome := func(id string) string {
		var o string
		_ = s.env.App.Pool.QueryRow(context.Background(), `SELECT e.outcome FROM store_checkout_entries e JOIN store_checkouts c ON c.id = e.checkout_id WHERE c.external_id = $1`, "tok"+id).Scan(&o)
		return o
	}
	now := s.clk.now()
	checkout("1", grace, "7001", true, now.Add(-90*time.Minute))   // after the hour: enters
	checkout("2", theo, "7002", false, now.Add(-90*time.Minute))   // didn't accept marketing
	checkout("3", "", "", false, now.Add(-90*time.Minute))         // no email
	checkout("4", grace, "7001", true, now.Add(-20*time.Minute))   // too early
	checkout("5", "x@example.com", "", true, now.Add(-2*time.Hour)) // ordered before the delay ran
	s.deliver(s.order(&shopifytest.Order{ID: "5900000005", Number: 5005, Email: "x@example.com", CheckoutToken: "tok5", CreatedAt: now.Add(-100 * time.Minute),
		UpdatedAt: now.Add(-100 * time.Minute), Subtotal: "76.0", Total: "76.0"}))
	if _, err := s.svc.DetectAbandoned(context.Background()); err != nil {
		t.Fatal(err)
	}
	if outcome("1") != "entered" || outcome("2") != "no_consent" || outcome("3") != "no_email" || outcome("4") != "" || outcome("5") != "" {
		t.Fatalf("outcomes: %q %q %q %q %q", outcome("1"), outcome("2"), outcome("3"), outcome("4"), outcome("5"))
	}
	if n := one[int64](t, s, `SELECT count(*) FROM automation_events WHERE kind = 'abandoned_checkout'`); n != 1 {
		t.Fatalf("events: %d", n)
	}
	// The skip rule: Grace got an abandoned-cart email in the last 7 days.
	graceID := one[uuid.UUID](t, s, `SELECT id FROM contacts WHERE email = $1`, grace)
	s.exec(`INSERT INTO messages (id, workspace_id, stream, automation_id, contact_id, email, status, queued_at) VALUES ($1, $2, 'automation', $3, $4, $5, 'delivered', $6)`,
		uuid.Must(uuid.NewV7()), s.ws.ID, flow, graceID, grace, now.Add(-time.Hour))
	s.clk.add(time.Hour)
	if _, err := s.svc.DetectAbandoned(context.Background()); err != nil {
		t.Fatal(err)
	}
	if outcome("4") != "skip_rule" {
		t.Fatalf("skip rule: %q", outcome("4"))
	}
	var stats commerce.CommerceTriggerStats
	s.c.Do("GET", s.path+"/automations/"+flow.String()+"/store-stats", nil).MustStatus(t, 200).JSON(t, &stats)
	if stats.Entered != 1 || stats.NoConsent != 1 || stats.Skipped != 1 {
		t.Fatalf("stats: %+v", stats)
	}
}

func TestDisconnectAndReconnect(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	flow := s.abandonedFlow(1)
	// Deleting the store data needs its name typed.
	s.c.Do("POST", s.path+"/stores/"+s.id+"/disconnect", map[string]any{"deleteData": true, "confirm": "Acme"}).MustStatus(t, 422)
	s.c.Do("POST", s.path+"/stores/"+s.id+"/disconnect", map[string]any{"deleteData": false}).MustStatus(t, 200)
	if topics := s.fake.WebhookTopics(); len(topics) != 0 {
		t.Fatalf("webhooks left in the store: %v", topics)
	}
	status := one[string](t, s, `SELECT status FROM automations WHERE id = $1`, flow)
	reason := one[string](t, s, `SELECT pause_reason FROM automations WHERE id = $1`, flow)
	if status != "paused" || reason != "Acme Merch is disconnected. Reconnect it to resume." {
		t.Fatalf("flow: %s %q", status, reason)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders WHERE store_id = $1`, s.id); n != 2 {
		t.Fatal("the data wasn't kept")
	}
	if n := one[int64](t, s, `SELECT count(*) FROM audit_events WHERE action IN ('store.disconnected', 'automation.pause')`); n != 2 {
		t.Fatalf("audit: %d", n)
	}
	// An order placed meanwhile is fetched on reconnecting.
	s.clk.add(time.Hour)
	s.fake.Orders = append(s.fake.Orders, &shopifytest.Order{ID: "5900000009", Number: 5009, Email: theo, CustomerID: "7002", CreatedAt: s.clk.now(), UpdatedAt: s.clk.now(),
		Subtotal: "18.0", Total: "18.0"})
	s.clk.add(time.Hour)
	// Disconnecting deleted the app's keys: they're pasted again.
	var st commerce.CommerceConnectState
	s.c.Do("POST", s.path+"/stores/"+s.id+"/reconnect", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Step != commerce.StepKeys {
		t.Fatalf("reconnect step: %s", st.Step)
	}
	s.install()
	if got := one[string](t, s, `SELECT status FROM automations WHERE id = $1`, flow); got != "live" {
		t.Fatalf("flow after reconnecting: %s", got)
	}
	if err := s.svc.Reconcile(context.Background(), uuid.MustParse(s.id), s.clk.now().Add(-3*time.Hour), nil); err != nil {
		t.Fatal(err)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders WHERE name = '#5009'`); n != 1 {
		t.Fatal("the order placed while disconnected wasn't fetched")
	}
	// Deleting the data: products and orders go, contacts stay.
	s.c.Do("POST", s.path+"/stores/"+s.id+"/disconnect", map[string]any{"deleteData": true, "confirm": "Acme Merch"}).MustStatus(t, 200)
	if n := one[int64](t, s, `SELECT count(*) FROM store_orders WHERE store_id = $1`, s.id); n != 0 {
		t.Fatal("orders kept")
	}
	if n := one[int64](t, s, `SELECT count(*) FROM contacts WHERE email = $1`, grace); n != 1 {
		t.Fatal("a contact the store added was deleted")
	}
}

func TestHealthRateLimitsAndRevokedTokens(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	ctx := context.Background()
	id := uuid.MustParse(s.id)
	// A webhook deleted in Shopify: marked missing and polled.
	s.fake.DeleteWebhook("checkouts/update")
	at := s.clk.now()
	s.fake.Checkouts = append(s.fake.Checkouts, &shopifytest.Checkout{ID: "36728419123499", Token: "polled", Email: grace, CustomerID: "7001", AcceptsMarketing: true,
		Subtotal: "18.0", Total: "18.0", CreatedAt: at, UpdatedAt: at})
	s.clk.add(time.Minute)
	if err := s.svc.Health(ctx); err != nil {
		t.Fatal(err)
	}
	if st := one[string](t, s, `SELECT status FROM store_webhooks WHERE store_id = $1 AND topic = 'checkouts/update'`, s.id); st != "missing" {
		t.Fatalf("webhook: %s", st)
	}
	if n := one[int64](t, s, `SELECT count(*) FROM store_checkouts WHERE external_id = 'polled'`); n != 1 {
		t.Fatal("the missing webhook's checkout wasn't polled")
	}
	if n := one[int64](t, s, `SELECT count(*) FROM notifications WHERE kind = 'store_problem'`); n == 0 {
		t.Fatal("nobody was told")
	}
	var p commerce.CommerceStorePage
	s.c.Do("POST", s.path+"/stores/"+s.id+"/webhooks", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Store.Sync.State != "connected" {
		t.Fatalf("after registering again: %s", p.Store.Sync.State)
	}
	// Rate limits slow the sync down and say so.
	s.fake.Throttle(20)
	_ = s.svc.Reconcile(ctx, id, time.Time{}, nil)
	s.c.Do("GET", s.path+"/stores/"+s.id, nil).MustStatus(t, 200).JSON(t, &p)
	if p.Store.Sync.State != "rate_limited" || p.Store.Sync.RateLimitedUntil == nil {
		t.Fatalf("rate limited: %+v", p.Store.Sync)
	}
	s.fake.Throttle(0)
	s.clk.add(time.Hour)
	// A revoked token disconnects the store, with Reconnect.
	s.fake.Revoke()
	if err := s.svc.Reconcile(ctx, id, time.Time{}, nil); err != nil {
		t.Fatal(err)
	}
	s.c.Do("GET", s.path+"/stores/"+s.id, nil).MustStatus(t, 200).JSON(t, &p)
	if p.Store.Status != commerce.StatusDisconnected || p.Store.Sync.RevokedReason != "revoked" {
		t.Fatalf("revoked: %s %+v", p.Store.Status, p.Store.Sync)
	}
}

func TestBlocksReadTheStoreAtSendTime(t *testing.T) {
	s := newShop(t)
	s.connect(fullSync)
	ctx := context.Background()
	block := commerce.CommerceBlock{Store: s.id, Mode: "chosen", Products: []string{"8841203", "8841177", "8841215"}, SoldOut: "hide", Button: "Shop now"}
	var r commerce.CommerceBlockRender
	s.c.Do("POST", s.path+"/commerce/blocks/render", map[string]any{"block": block}).MustStatus(t, 200).JSON(t, &r)
	if !strings.Contains(r.HTML, "Acme Logo Hoodie") || !strings.Contains(r.HTML, "$58") || len(r.SoldOut) != 1 {
		t.Fatalf("drawn: %+v", r.SoldOut)
	}
	email := "<p>Hi</p>" + r.HTML
	// The price changes, a product is deleted: the send reads them again.
	hoodie := s.fake.Products[0]
	hoodie.Price, hoodie.UpdatedAt = "52.0", s.clk.now()
	s.deliver(s.fake.Deliver("products/update", s.fake.ProductPayload(hoodie)))
	s.deliver(s.fake.Deliver("products/delete", map[string]any{"id": 8841215}))
	sent := s.svc.Resolve(ctx, s.ws.ID, email, false)
	if !strings.Contains(sent, "$52") || strings.Contains(sent, "$58") {
		t.Fatal("the send didn't read the new price")
	}
	if strings.Contains(sent, "Feature Flag sticker pack") || strings.Contains(sent, "Deploy Friday cap") {
		t.Fatal("a removed or sold-out product was sent")
	}
	// Labelled instead of hidden.
	block.SoldOut = "label"
	s.c.Do("POST", s.path+"/commerce/blocks/render", map[string]any{"block": block}).MustStatus(t, 200).JSON(t, &r)
	labelled := s.svc.Resolve(ctx, s.ws.ID, r.HTML, false)
	if !strings.Contains(labelled, "Deploy Friday cap") || !strings.Contains(labelled, "Sold out") {
		t.Fatal("the sold-out product isn't labelled")
	}
	// Automation emails keep every product in its place, the hidden ones invisible.
	stable := s.svc.Resolve(ctx, s.ws.ID, email, true)
	if !strings.Contains(stable, "Feature Flag sticker pack") || !strings.Contains(stable, "display:none") {
		t.Fatal("the stable drawing moved products")
	}
	// The composer's check and Send Readiness flag the removed product.
	issues := s.svc.RemovedProducts(ctx, contentcheck.Input{WorkspaceID: s.ws.ID, HTML: email})
	if len(issues) != 1 || issues[0].ID != commerce.IssueProductRemoved || !strings.Contains(issues[0].Title, "Feature Flag sticker pack is no longer in Acme Merch") {
		t.Fatalf("issues: %+v", issues)
	}
	// Another workspace can't read the store through a block.
	other := s.env.CreateWorkspace("Other", one[uuid.UUID](t, s, `SELECT id FROM users WHERE email = 'marcus@acme.io'`))
	if got := s.svc.RemovedProducts(ctx, contentcheck.Input{WorkspaceID: other.ID, HTML: email}); len(got) != 0 {
		t.Fatal("another workspace read the store")
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
