package products_test

// The demo workspace selling its products, for the buying tests. Stripe is
// an in-process fake (stripetest) holding the demo's products and prices:
// buyers pay on the fake, and its events reach the live connection's
// webhook signed as Stripe signs them. The delivery email goes through the
// Transactional route to an SMTP sink. Nothing leaves the machine.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/db"
	demodata "github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
)

// buyerIP is where the tests' Buy buttons are pressed from.
const buyerIP = "203.0.113.9"

// clock is the products domain's time: tests move it to expire links.
type clock struct{ ns atomic.Int64 }

func (c *clock) now() time.Time      { return time.Unix(0, c.ns.Load()) }
func (c *clock) add(d time.Duration) { c.ns.Add(int64(d)) }

// shop is the demo workspace ready to sell.
type shop struct {
	t     *testing.T
	ctx   context.Context
	env   *apptest.Env
	owner *apptest.Client // Linh Nguyen, the Owner
	ws    store.Workspace
	svc   *products.Service
	// stripe is the workspace's Stripe account.
	stripe *stripetest.Fake
	clock  *clock
	// The demo's products: The Deliverability Field Guide ($29, one-time),
	// the Newsletter Launch Kit (pay what you want) and the Warm-up planner
	// (a draft).
	fieldGuide, launchKit, planner uuid.UUID
	// digest is the Weekly Digest, the list offer's list.
	digest uuid.UUID
	// live is the live payment connection; its webhook's address and secret.
	live                  uuid.UUID
	hookToken, hookSecret string
	// sink receives the mail once deliverable() ran.
	sink *apptest.SMTPSink
}

func newShop(t *testing.T) *shop {
	t.Helper()
	env, owner := demo(t)
	s := &shop{t: t, ctx: context.Background(), env: env, owner: owner, svc: env.App.Products(), clock: &clock{}}
	s.clock.ns.Store(time.Now().Truncate(time.Second).UnixNano())
	s.svc.Now = s.clock.now
	s.stripe = stripetest.New()
	s.stripe.Permissive = true
	s.stripe.AddKey(demodata.EarnLiveKey, stripetest.KeySpec{})
	s.stripe.AddKey(demodata.EarnTestKey, stripetest.KeySpec{})
	// The fake reached in process, as the demo's connections reach theirs.
	env.App.Earn().Demo = func() *http.Client { return s.stripe.HTTPClient() }
	var wsID uuid.UUID
	s.scan(`SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`, nil, &wsID)
	ws, err := env.App.Q.GetWorkspace(s.ctx, wsID)
	if err != nil {
		t.Fatal(err)
	}
	s.ws = ws
	s.fieldGuide = s.productID("deliverability-field-guide")
	s.launchKit = s.productID("newsletter-launch-kit")
	s.planner = s.productID("warm-up-planner")
	s.scan(`SELECT id FROM lists WHERE workspace_id = $1 AND name = 'Weekly Digest' AND deleted_at IS NULL`, []any{wsID}, &s.digest)
	var sealed []byte
	s.scan(`SELECT id, webhook_token, webhook_secret_sealed FROM payment_connections WHERE workspace_id = $1 AND mode = 'live' AND status = 'connected'`,
		[]any{wsID}, &s.live, &s.hookToken, &sealed)
	if s.hookSecret, err = env.App.Secrets.OpenString(sealed, "payment_connection:"+s.live.String()+":webhook_secret"); err != nil {
		t.Fatalf("webhook secret: %v", err)
	}
	s.stripeHolds(s.fieldGuide)
	s.stripeHolds(s.launchKit)
	return s
}

// stripeHolds puts a product's Stripe objects (the ids the demo recorded) in
// the fake, as the real account holds them.
func (s *shop) stripeHolds(product uuid.UUID) {
	s.t.Helper()
	var raw []byte
	var name string
	s.scan(`SELECT stripe, name FROM products WHERE id = $1`, []any{product}, &raw, &name)
	var modes map[string]struct {
		Product, Price, Kind   string
		Amount, Min, Suggested int64
	}
	if err := json.Unmarshal(raw, &modes); err != nil {
		s.t.Fatalf("stripe ids of %s: %v", name, err)
	}
	for mode, m := range modes {
		pm := payments.Mode(mode)
		if m.Product != "" {
			s.stripe.Put(pm, map[string]any{"id": m.Product, "object": "product", "name": name, "active": true, "metadata": map[string]any{"reputecast_product": product.String()}})
		}
		if m.Price == "" {
			continue
		}
		price := map[string]any{"id": m.Price, "object": "price", "product": m.Product, "currency": "usd", "active": true, "type": "one_time", "unit_amount": m.Amount}
		if m.Kind == products.PricePWYW {
			price["unit_amount"] = nil
			price["custom_unit_amount"] = map[string]any{"minimum": m.Min, "preset": m.Suggested, "maximum": nil}
		}
		s.stripe.Put(pm, price)
	}
}

func (s *shop) scan(sql string, args []any, dest ...any) {
	s.t.Helper()
	if err := s.env.App.Pool.QueryRow(s.ctx, sql, args...).Scan(dest...); err != nil {
		s.t.Fatalf("%s: %v", sql, err)
	}
}

func (s *shop) count(sql string, args ...any) int {
	s.t.Helper()
	var n int64
	s.scan(sql, args, &n)
	return int(n)
}

func (s *shop) exec(sql string, args ...any) {
	s.t.Helper()
	if _, err := s.env.App.Pool.Exec(s.ctx, sql, args...); err != nil {
		s.t.Fatalf("%s: %v", sql, err)
	}
}

func (s *shop) productID(slug string) uuid.UUID {
	s.t.Helper()
	var id uuid.UUID
	s.scan(`SELECT id FROM products WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, []any{s.ws.ID, slug}, &id)
	return id
}

// slug is a product's address ("product" for one that doesn't exist).
func (s *shop) slug(product uuid.UUID) string {
	s.t.Helper()
	slug := "product"
	if err := s.env.App.Pool.QueryRow(s.ctx, `SELECT slug FROM products WHERE id = $1`, product).Scan(&slug); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.t.Fatal(err)
	}
	return slug
}

// site is the workspace's public site.
func (s *shop) site() string { return s.env.Server.URL + "/p/" + s.ws.Slug }

// contactID is the workspace's contact with an email (uuid.Nil for none).
func (s *shop) contactID(email string) uuid.UUID {
	s.t.Helper()
	var id uuid.UUID
	err := s.env.App.Pool.QueryRow(s.ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = lower($2)`, s.ws.ID, email).Scan(&id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.t.Fatal(err)
	}
	return id
}

// addContact adds an active contact (a reader who never bought anything).
func (s *shop) addContact(email, first string) uuid.UUID {
	s.t.Helper()
	id := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO contacts (id, workspace_id, email, first_name, timezone) VALUES ($1, $2, $3, $4, 'America/New_York')`, id, s.ws.ID, email, first)
	return id
}

// press is a Buy button on a product's default page: email is a signed-in
// member's ("" for a visitor), ticked the list offer's checkbox.
func (s *shop) press(product uuid.UUID, email string, ticked bool) (products.CheckoutHandoff, error) {
	s.t.Helper()
	page := s.site() + "/products/" + s.slug(product)
	return s.svc.StartCheckout(s.ctx, &s.ws, products.CheckoutStart{ProductID: product, Ticked: ticked, Source: products.SourceProductPage,
		PageURL: page, PageLabel: strings.TrimPrefix(page, "http://"), IP: buyerIP, Email: email,
		ThanksURL: page + "/thanks?session_id={CHECKOUT_SESSION_ID}"})
}

// checkout presses Buy and returns the handoff and Stripe's session id.
func (s *shop) checkout(product uuid.UUID, email string, ticked bool) (products.CheckoutHandoff, string) {
	s.t.Helper()
	h, err := s.press(product, email, ticked)
	if err != nil {
		s.t.Fatalf("checkout: %v", err)
	}
	u, err := url.Parse(h.URL)
	if err != nil || u.Host != "checkout.stripe.com" {
		s.t.Fatalf("checkout address %q", h.URL)
	}
	return h, path.Base(u.Path)
}

// buyer is who pays on Stripe's page.
type buyer struct {
	Email, Name string
	// Ticked is the list offer's checkbox, on the page before Stripe.
	Ticked bool
	// Amount is what a pay-what-you-want buyer chooses (0: the preset).
	Amount int64
	// Tax is what Stripe Tax adds.
	Tax int64
}

// pay completes a checkout session on Stripe's page.
func (s *shop) pay(session string, b buyer) stripetest.Paid {
	s.t.Helper()
	p, err := s.stripe.CompletePayment(payments.Live, session, stripetest.PaymentOptions{Email: b.Email, Name: b.Name, AmountCents: b.Amount, TaxCents: b.Tax})
	if err != nil {
		s.t.Fatalf("pay: %v", err)
	}
	return p
}

// buy is a whole purchase: Buy, pay on Stripe, and Stripe's
// checkout.session.completed on the webhook.
func (s *shop) buy(product uuid.UUID, b buyer) (store.ProductOrder, stripetest.Paid) {
	s.t.Helper()
	_, session := s.checkout(product, "", b.Ticked)
	paid := s.pay(session, b)
	if st, out := s.hook(paid.Events[0]); st != "applied" || out != "order_paid" {
		s.t.Fatalf("checkout.session.completed: %s %s", st, out)
	}
	return s.orderOf(session), paid
}

// hook delivers a Stripe event to the live connection's webhook, signed as
// Stripe signs it, and applies it as Earn's job does. It returns the
// payments event log's status and outcome.
func (s *shop) hook(e stripetest.Event) (status, outcome string) {
	s.t.Helper()
	id := s.deliverHook(e)
	if err := s.env.App.Earn().Apply(s.ctx, id); err != nil {
		s.t.Fatalf("apply %s: %v", e.Type, err)
	}
	s.scan(`SELECT status, coalesce(outcome, '') FROM payment_events WHERE id = $1`, []any{id}, &status, &outcome)
	return status, outcome
}

// deliverHook posts an event to the webhook and returns its row in the
// payments event log (not applied yet).
func (s *shop) deliverHook(e stripetest.Event) uuid.UUID {
	s.t.Helper()
	code, err := stripetest.Deliver(s.ctx, nil, s.env.Server.URL+"/hooks/stripe/"+s.hookToken, s.hookSecret, e, time.Now())
	if err != nil || code != http.StatusOK {
		s.t.Fatalf("webhook %s: %d %v", e.Type, code, err)
	}
	var id uuid.UUID
	s.scan(`SELECT id FROM payment_events WHERE provider_event_id = $1`, []any{e.ID}, &id)
	return id
}

// consume offers an event to the products consumer directly, as Earn does
// inside the transaction that files it.
func (s *shop) consume(e stripetest.Event) (earn.Consumed, bool) {
	s.t.Helper()
	p, err := stripe.Verifier{}.ParsePayment(e.Body)
	if err != nil {
		s.t.Fatalf("parse %s: %v", e.Type, err)
	}
	var out earn.Consumed
	var ok bool
	err = db.Tx(s.ctx, s.env.App.Pool, func(tx pgx.Tx) error {
		var err error
		out, ok, err = s.svc.Consume(s.ctx, tx, earn.PaymentEvent{Workspace: &s.ws, ConnectionID: s.live, Provider: "stripe", Mode: payments.Live,
			Type: e.Type, Payment: p})
		return err
	})
	if err != nil {
		s.t.Fatalf("consume %s: %v", e.Type, err)
	}
	return out, ok
}

// orderOf is the order a checkout session became.
func (s *shop) orderOf(session string) store.ProductOrder {
	s.t.Helper()
	o, err := s.env.App.Q.ProductOrderBySession(s.ctx, session)
	if err != nil {
		s.t.Fatalf("order of %s: %v", session, err)
	}
	return o
}

// order reads an order again.
func (s *shop) order(id uuid.UUID) store.ProductOrder {
	s.t.Helper()
	o, err := s.env.App.Q.ProductOrderGet(s.ctx, store.ProductOrderGetParams{WorkspaceID: s.ws.ID, ID: id})
	if err != nil {
		s.t.Fatalf("order %s: %v", id, err)
	}
	return o
}

// links are an order's download links, newest first.
func (s *shop) links(order uuid.UUID) []store.ProductOrderLink {
	s.t.Helper()
	rows, err := s.env.App.Q.ProductOrderLinks(s.ctx, order)
	if err != nil {
		s.t.Fatal(err)
	}
	out := make([]store.ProductOrderLink, len(rows))
	for i, r := range rows {
		out[i] = r.ProductOrderLink
	}
	return out
}

// link is an order's current link.
func (s *shop) link(order uuid.UUID) store.ProductOrderLink {
	s.t.Helper()
	ls := s.links(order)
	if len(ls) == 0 {
		s.t.Fatalf("order %s has no download link", order)
	}
	return ls[0]
}

// token is a link's download token, as its emails and pages carry it.
func (s *shop) token(order uuid.UUID, link uuid.UUID) string {
	return s.svc.DownloadToken(s.ws.ID, order, link)
}

// events are an order's activity, oldest first.
func (s *shop) events(order uuid.UUID) []string {
	s.t.Helper()
	rows, err := s.env.App.Q.ProductOrderEvents(s.ctx, order)
	if err != nil {
		s.t.Fatal(err)
	}
	out := []string{}
	for _, r := range rows {
		out = append(out, r.ProductOrderEvent.Kind)
	}
	return out
}

// event is the detail of an order's last event of a kind (nil for none).
func (s *shop) event(order uuid.UUID, kind string) map[string]any {
	s.t.Helper()
	rows, err := s.env.App.Q.ProductOrderEvents(s.ctx, order)
	if err != nil {
		s.t.Fatal(err)
	}
	var out map[string]any
	for _, r := range rows {
		if r.ProductOrderEvent.Kind == kind {
			out = map[string]any{}
			_ = json.Unmarshal(r.ProductOrderEvent.Detail, &out)
		}
	}
	return out
}

// audited is the audit log's latest entry of an action on a target.
type auditEntry struct {
	ActorType     string
	Before, After map[string]any
	Metadata      map[string]any
}

func (s *shop) audited(action string, target uuid.UUID) (auditEntry, bool) {
	s.t.Helper()
	var e auditEntry
	var before, after, meta []byte
	err := s.env.App.Pool.QueryRow(s.ctx, `SELECT actor_type, coalesce(before, 'null'), coalesce(after, 'null'), coalesce(metadata, 'null') FROM audit_events
		WHERE workspace_id = $1 AND action = $2 AND target_id = $3 ORDER BY occurred_at DESC LIMIT 1`, s.ws.ID, action, target.String()).Scan(&e.ActorType, &before, &after, &meta)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, false
	}
	if err != nil {
		s.t.Fatal(err)
	}
	_ = json.Unmarshal(before, &e.Before)
	_ = json.Unmarshal(after, &e.After)
	_ = json.Unmarshal(meta, &e.Metadata)
	return e, true
}

// deliveryJobs counts the products.deliver jobs queued for a link.
func (s *shop) deliveryJobs(link uuid.UUID) int {
	s.t.Helper()
	return s.count(`SELECT count(*) FROM river_job WHERE kind = 'products.deliver' AND args->>'link_id' = $1`, link.String())
}

// deliver runs a link's queued delivery job, as the worker would.
func (s *shop) deliver(link uuid.UUID) {
	s.t.Helper()
	if n := s.deliveryJobs(link); n != 1 {
		s.t.Fatalf("link %s has %d delivery jobs, want 1", link, n)
	}
	if err := s.svc.Deliver(s.ctx, link); err != nil {
		s.t.Fatalf("deliver: %v", err)
	}
}

// deliverable lets the demo send delivery emails: a verified Transactional
// sender, the Transactional route Postmark then Amazon SES, and the demo
// providers delivering to an SMTP sink standing in for the development mail
// catcher.
func (s *shop) deliverable() {
	s.t.Helper()
	s.sink = apptest.NewSMTPSink(s.t)
	s.env.App.TransactionalOutbox().DevSMTP = s.sink.Addr
	dom := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO sending_domains (id, workspace_id, domain, status, transactional) VALUES ($1, $2, 'orders.acme-x19.test', 'verified', true)`, dom, s.ws.ID)
	s.exec(`INSERT INTO senders (id, workspace_id, domain_id, name, email, reply_to) VALUES ($1, $2, $3, 'Acme orders', 'orders@orders.acme-x19.test', 'support@acme.io')`,
		uuid.Must(uuid.NewV7()), s.ws.ID, dom)
	prov := map[string]uuid.UUID{}
	rows, err := s.env.App.Pool.Query(s.ctx, `SELECT adapter, id FROM providers WHERE workspace_id = $1`, s.ws.ID)
	if err != nil {
		s.t.Fatal(err)
	}
	for rows.Next() {
		var a string
		var id uuid.UUID
		_ = rows.Scan(&a, &id)
		prov[a] = id
	}
	rows.Close()
	s.route(prov["postmark"], prov["ses"])
}

// route sets the Transactional route: primary, then fallback (automatic).
func (s *shop) route(primary, fallback uuid.UUID) {
	s.t.Helper()
	var c struct {
		Streams []struct {
			Stream string
			Splits []struct {
				ProviderID string
				Weight     int
			}
			Fallback   string
			Failover   string
			Conditions []struct {
				Field      string
				Values     []string
				ProviderID string
				Note       string
			}
		}
	}
	s.owner.Do("GET", ws+"/routing", nil).MustStatus(s.t, 200).JSON(s.t, &c)
	streams := []map[string]any{}
	for _, st := range c.Streams {
		in := map[string]any{"stream": st.Stream, "fallback": st.Fallback, "failover": st.Failover}
		splits, conds := []map[string]any{}, []map[string]any{}
		for _, sp := range st.Splits {
			splits = append(splits, map[string]any{"providerId": sp.ProviderID, "weight": sp.Weight})
		}
		for _, cd := range st.Conditions {
			conds = append(conds, map[string]any{"field": cd.Field, "values": cd.Values, "providerId": cd.ProviderID, "note": cd.Note})
		}
		if st.Stream == "transactional" {
			splits = []map[string]any{{"providerId": primary.String(), "weight": 100}}
			in["fallback"], in["failover"] = fallback.String(), "automatic"
		}
		in["splits"], in["conditions"] = splits, conds
		streams = append(streams, in)
	}
	s.owner.Do("PUT", ws+"/routing", map[string]any{"streams": streams, "acknowledgeBulkTerms": true}).MustStatus(s.t, 200)
}

// sendMail hands the queued transactional mail to the providers, step after
// step (each sends a batch within the providers' rates) until none waits.
func (s *shop) sendMail() int {
	s.t.Helper()
	total, at := 0, time.Now()
	for i := 0; i < 30; i++ {
		n, err := s.env.App.TransactionalOutbox().Step(s.ctx, at)
		if err != nil {
			s.t.Fatalf("outbox: %v", err)
		}
		total += n
		if s.count(`SELECT count(*) FROM transactional_outbox t JOIN messages m ON m.id = t.message_id WHERE m.workspace_id = $1`, s.ws.ID) == 0 {
			break
		}
		at = at.Add(2 * time.Second)
	}
	return total
}

// unfold undoes quoted-printable soft line breaks and folded headers so a
// test can look for a value in a received message.
func unfold(data string) string {
	return strings.NewReplacer("=\r\n", "", "=3D", "=", "\r\n ", "", "\r\n\t", "").Replace(data)
}

// deliveryMessage is the delivery email a link was sent with: its message id
// and recipient.
func (s *shop) deliveryMessage(link uuid.UUID) (uuid.UUID, string) {
	s.t.Helper()
	var id uuid.UUID
	var to string
	s.scan(`SELECT m.id, m.email FROM product_order_links l JOIN messages m ON m.id = l.message_id WHERE l.id = $1`, []any{link}, &id, &to)
	return id, to
}

// holding names the rows of an order (its checkout, links, events, audit
// entries, delivery emails and their log) and of the jobs, webhooks and
// notifications since the test began that hold text anywhere.
func (s *shop) holding(order uuid.UUID, text string, since time.Time) []string {
	s.t.Helper()
	checks := []struct{ name, sql string }{
		{"product_orders", `SELECT count(*) FROM product_orders t WHERE t.id = $1 AND strpos(t::text, $2::text) > 0`},
		{"product_checkouts", `SELECT count(*) FROM product_checkouts t WHERE t.order_id = $1 AND strpos(t::text, $2::text) > 0`},
		{"product_order_links", `SELECT count(*) FROM product_order_links t WHERE t.order_id = $1 AND strpos(t::text, $2::text) > 0`},
		{"product_order_events", `SELECT count(*) FROM product_order_events t WHERE t.order_id = $1 AND strpos(t::text, $2::text) > 0`},
		{"audit_events", `SELECT count(*) FROM audit_events t WHERE t.target_id = $1::text AND strpos(t::text, $2::text) > 0`},
		{"messages", `SELECT count(*) FROM messages t WHERE t.id IN (SELECT message_id FROM product_order_links WHERE order_id = $1) AND strpos(t::text, $2::text) > 0`},
		{"transactional_messages", `SELECT count(*) FROM transactional_messages t WHERE t.message_id IN (SELECT message_id FROM product_order_links WHERE order_id = $1) AND strpos(t::text, $2::text) > 0`},
		{"transactional_message_content", `SELECT count(*) FROM transactional_message_content t WHERE t.message_id IN (SELECT message_id FROM product_order_links WHERE order_id = $1) AND strpos(t::text, $2::text) > 0`},
		{"transactional_outbox", `SELECT count(*) FROM transactional_outbox t WHERE t.message_id IN (SELECT message_id FROM product_order_links WHERE order_id = $1) AND strpos(t::text, $2::text) > 0`},
	}
	var out []string
	for _, c := range checks {
		if s.count(c.sql, order, text) > 0 {
			out = append(out, c.name)
		}
	}
	for _, c := range []struct{ name, sql string }{
		{"river_job", `SELECT count(*) FROM river_job t WHERE t.created_at >= $1 AND strpos(t::text, $2::text) > 0`},
		{"webhook_deliveries", `SELECT count(*) FROM webhook_deliveries t WHERE t.created_at >= $1 AND strpos(t::text, $2::text) > 0`},
		{"notifications", `SELECT count(*) FROM notifications t WHERE t.created_at >= $1 AND strpos(t::text, $2::text) > 0`},
		{"payment_events", `SELECT count(*) FROM payment_events t WHERE t.received_at >= $1 AND strpos(t::text, $2::text) > 0`},
	} {
		if s.count(c.sql, since, text) > 0 {
			out = append(out, c.name)
		}
	}
	return out
}

// subscribeOrders adds a webhook endpoint for every order.* event (its
// deliveries are queued, never sent: no worker runs in tests).
func (s *shop) subscribeOrders() uuid.UUID {
	s.t.Helper()
	id := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO webhooks (id, workspace_id, name, url, events, secret_enc) VALUES ($1, $2, 'Orders', 'https://app.acme.io/hooks/orders', '{order.*}', '\x00')`, id, s.ws.ID)
	return id
}

// hooked are the order.* webhook payloads queued for an endpoint, oldest first.
func (s *shop) hooked(webhook uuid.UUID) []map[string]any {
	s.t.Helper()
	rows, err := s.env.App.Pool.Query(s.ctx, `SELECT payload FROM webhook_deliveries WHERE webhook_id = $1 ORDER BY created_at, id`, webhook)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			s.t.Fatal(err)
		}
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		out = append(out, m)
	}
	return out
}

// lastRequest is the fake's latest request to a method and path.
func (s *shop) lastRequest(method, p string) (stripetest.Recorded, bool) {
	reqs := s.stripe.Requests()
	for i := len(reqs) - 1; i >= 0; i-- {
		if reqs[i].Method == method && reqs[i].Path == p {
			return reqs[i], true
		}
	}
	return stripetest.Recorded{}, false
}

// requestsSince are the fake's requests after the first n.
func (s *shop) requestsSince(n int) []stripetest.Recorded {
	reqs := s.stripe.Requests()
	if n > len(reqs) {
		return nil
	}
	return reqs[n:]
}
