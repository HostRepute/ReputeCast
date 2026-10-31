package publicpages_test

// Buying and downloading on the publication's site (news.acme.io), as a
// buyer's browser does: a product page's Buy button, "Taking you to
// Stripe…", Stripe's page (stripetest, in process), the thank-you page, the
// download page and its files, and a lead magnet's link. Stripe's events
// reach the live connection's webhook signed as Stripe signs them; delivery
// emails go to an SMTP sink. Nothing leaves the machine.

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	demodata "github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
)

// sfClock is the products domain's time: tests move it to expire links.
type sfClock struct{ ns atomic.Int64 }

func (c *sfClock) now() time.Time      { return time.Unix(0, c.ns.Load()) }
func (c *sfClock) add(d time.Duration) { c.ns.Add(int64(d)) }

// sfEnv is the demo selling on news.acme.io.
type sfEnv struct {
	t      *testing.T
	ctx    context.Context
	env    *apptest.Env
	ids    demoIDs
	ws     store.Workspace
	svc    *products.Service
	stripe *stripetest.Fake
	clock  *sfClock
	// The Deliverability Field Guide, the Newsletter Launch Kit and the
	// Warm-up planner (a draft).
	fieldGuide, launchKit, planner uuid.UUID
	hookToken, hookSecret          string
	sink                           *apptest.SMTPSink
}

func sfDemo(t *testing.T) *sfEnv {
	t.Helper()
	env, ids := demo(t)
	e := &sfEnv{t: t, ctx: context.Background(), env: env, ids: ids, svc: env.App.Products(), clock: &sfClock{}}
	e.clock.ns.Store(time.Now().Truncate(time.Second).UnixNano())
	e.svc.Now = e.clock.now
	e.stripe = stripetest.New()
	e.stripe.Permissive = true
	e.stripe.AddKey(demodata.EarnLiveKey, stripetest.KeySpec{})
	e.stripe.AddKey(demodata.EarnTestKey, stripetest.KeySpec{})
	// The fake reached in process, as the demo's connections reach theirs.
	env.App.Earn().Demo = func() *http.Client { return e.stripe.HTTPClient() }
	ws, err := env.App.Q.GetWorkspace(e.ctx, ids.ws)
	if err != nil {
		t.Fatal(err)
	}
	e.ws = ws
	e.fieldGuide, e.launchKit, e.planner = e.product("deliverability-field-guide"), e.product("newsletter-launch-kit"), e.product("warm-up-planner")
	var live uuid.UUID
	var sealed []byte
	if err := env.App.Pool.QueryRow(e.ctx, `SELECT id, webhook_token, webhook_secret_sealed FROM payment_connections WHERE workspace_id = $1 AND mode = 'live' AND status = 'connected'`,
		ids.ws).Scan(&live, &e.hookToken, &sealed); err != nil {
		t.Fatal(err)
	}
	if e.hookSecret, err = env.App.Secrets.OpenString(sealed, "payment_connection:"+live.String()+":webhook_secret"); err != nil {
		t.Fatal(err)
	}
	e.stripeHolds(e.fieldGuide)
	e.stripeHolds(e.launchKit)
	return e
}

func (e *sfEnv) product(slug string) uuid.UUID {
	return one[uuid.UUID](e.t, e.env, `SELECT id FROM products WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, e.ids.ws, slug)
}

// stripeHolds puts a product's Stripe objects (the ids the demo recorded)
// in the fake.
func (e *sfEnv) stripeHolds(product uuid.UUID) {
	e.t.Helper()
	var modes map[string]struct {
		Product, Price, Kind   string
		Amount, Min, Suggested int64
	}
	if err := json.Unmarshal(one[[]byte](e.t, e.env, `SELECT stripe FROM products WHERE id = $1`, product), &modes); err != nil {
		e.t.Fatal(err)
	}
	for mode, m := range modes {
		pm := payments.Mode(mode)
		if m.Product != "" {
			e.stripe.Put(pm, map[string]any{"id": m.Product, "object": "product", "name": "product", "active": true, "metadata": map[string]any{}})
		}
		if m.Price == "" {
			continue
		}
		price := map[string]any{"id": m.Price, "object": "price", "product": m.Product, "currency": "usd", "active": true, "type": "one_time", "unit_amount": m.Amount}
		if m.Kind == products.PricePWYW {
			price["unit_amount"] = nil
			price["custom_unit_amount"] = map[string]any{"minimum": m.Min, "preset": m.Suggested, "maximum": nil}
		}
		e.stripe.Put(pm, price)
	}
}

// news requests a path of the publication's site on its custom domain.
func (e *sfEnv) news(method, p string, form url.Values, h hdr) resp {
	e.t.Helper()
	if h == nil {
		h = hdr{}
	}
	h["Host"] = "news.acme.io"
	return do(e.t, e.env, method, p, form, h)
}

// on is a link of the site as a path on it ("http://news.acme.io/d/…" is
// "/d/…").
func on(link string) string {
	u, err := url.Parse(html.UnescapeString(link))
	if err != nil {
		return link
	}
	return u.RequestURI()
}

// bought buys a product as the Buy button and Stripe do (the thank-you page
// applies the payment) and returns the order.
func (e *sfEnv) bought(product uuid.UUID, email string) (store.ProductOrder, stripetest.Paid) {
	e.t.Helper()
	h, err := e.svc.StartCheckout(e.ctx, &e.ws, products.CheckoutStart{ProductID: product, Source: products.SourceProductPage, PageURL: "http://news.acme.io/products",
		ThanksURL: "http://news.acme.io/products/thanks?session_id={CHECKOUT_SESSION_ID}"})
	if err != nil {
		e.t.Fatal(err)
	}
	session := path.Base(h.URL)
	paid, err := e.stripe.PayCheckout(payments.Live, session, stripetest.PaymentOptions{Email: email, Name: "Sky Buyer"})
	if err != nil {
		e.t.Fatal(err)
	}
	v, ok, err := e.svc.Thanks(e.ctx, &e.ws, product, session)
	if err != nil || !ok {
		e.t.Fatalf("thanks: %v %v", ok, err)
	}
	return v.Order, paid
}

// hook delivers a Stripe event to the live connection's webhook and
// applies it as Earn's job does.
func (e *sfEnv) hook(ev stripetest.Event) {
	e.t.Helper()
	code, err := stripetest.Deliver(e.ctx, nil, e.env.Server.URL+"/hooks/stripe/"+e.hookToken, e.hookSecret, ev, time.Now())
	if err != nil || code != http.StatusOK {
		e.t.Fatalf("webhook %s: %d %v", ev.Type, code, err)
	}
	id := one[uuid.UUID](e.t, e.env, `SELECT id FROM payment_events WHERE provider_event_id = $1`, ev.ID)
	if err := e.env.App.Earn().Apply(e.ctx, id); err != nil {
		e.t.Fatal(err)
	}
}

// links are an order's download links, newest first.
func (e *sfEnv) links(order uuid.UUID) []store.ProductOrderLink {
	e.t.Helper()
	rows, err := e.env.App.Q.ProductOrderLinks(e.ctx, order)
	if err != nil {
		e.t.Fatal(err)
	}
	out := make([]store.ProductOrderLink, len(rows))
	for i, r := range rows {
		out[i] = r.ProductOrderLink
	}
	return out
}

// deliverable lets the demo send delivery emails: a verified Transactional
// sender, the Transactional route on the demo's Postmark, and the demo
// providers delivering to an SMTP sink.
func (e *sfEnv) deliverable() {
	e.t.Helper()
	e.sink = apptest.NewSMTPSink(e.t)
	e.env.App.TransactionalOutbox().DevSMTP = e.sink.Addr
	dom := uuid.Must(uuid.NewV7())
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO sending_domains (id, workspace_id, domain, status, transactional) VALUES ($1, $2, 'orders.acme-x19.test', 'verified', true)`, []any{dom, e.ids.ws}},
		{`INSERT INTO senders (id, workspace_id, domain_id, name, email, reply_to) VALUES ($1, $2, $3, 'Acme orders', 'orders@orders.acme-x19.test', 'support@acme.io')`,
			[]any{uuid.Must(uuid.NewV7()), e.ids.ws, dom}},
	} {
		if _, err := e.env.App.Pool.Exec(e.ctx, q.sql, q.args...); err != nil {
			e.t.Fatal(err)
		}
	}
	postmark := one[uuid.UUID](e.t, e.env, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = 'postmark'`, e.ids.ws)
	ses := one[uuid.UUID](e.t, e.env, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = 'ses'`, e.ids.ws)
	c := owner(e.env)
	var routing struct {
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
	c.Do(http.MethodGet, wsAPI+"/routing", nil).MustStatus(e.t, 200).JSON(e.t, &routing)
	streams := []map[string]any{}
	for _, st := range routing.Streams {
		in := map[string]any{"stream": st.Stream, "fallback": st.Fallback, "failover": st.Failover}
		splits, conds := []map[string]any{}, []map[string]any{}
		for _, sp := range st.Splits {
			splits = append(splits, map[string]any{"providerId": sp.ProviderID, "weight": sp.Weight})
		}
		for _, cd := range st.Conditions {
			conds = append(conds, map[string]any{"field": cd.Field, "values": cd.Values, "providerId": cd.ProviderID, "note": cd.Note})
		}
		if st.Stream == "transactional" {
			splits = []map[string]any{{"providerId": postmark.String(), "weight": 100}}
			in["fallback"], in["failover"] = ses.String(), "automatic"
		}
		in["splits"], in["conditions"] = splits, conds
		streams = append(streams, in)
	}
	c.Do(http.MethodPut, wsAPI+"/routing", map[string]any{"streams": streams, "acknowledgeBulkTerms": true}).MustStatus(e.t, 200)
}

var sfDownloadLink = regexp.MustCompile(`href="(http://news\.acme\.io/d/[A-Za-z0-9_-]+)"`)

// firstFile is a product's first file.
func (e *sfEnv) firstFile(product uuid.UUID) uuid.UUID {
	return one[uuid.UUID](e.t, e.env, `SELECT id FROM download_files WHERE owner_kind = 'product' AND owner_id = $1 AND removed_at IS NULL ORDER BY position, uploaded_at LIMIT 1`, product)
}

// A reader buys on news.acme.io: the product page's Buy button with the
// list offer ticked, "Taking you to Stripe…" (which creates nothing yet),
// then Stripe's Checkout; a cancelled checkout comes back to the page. Back
// from Stripe before its payment arrives, the thank-you page waits; paid, it
// has "Download now" and says where the link went. The download link's page
// is kept out of caches and search engines, and its file downloads as an
// attachment, counted from its start but not when resumed.
func TestBuyAndDownloadOnTheSite(t *testing.T) {
	e := sfDemo(t)
	fg := e.fieldGuide.String()
	page := e.news(http.MethodGet, "/products/deliverability-field-guide", nil, nil)
	must(t, page, http.StatusOK, "The Deliverability Field Guide", `action="http://news.acme.io/buy/`+fg+`"`, `name="offer" value="1"`, "Also send me the Weekly Digest")

	checkouts := count(t, e.env, `SELECT count(*) FROM product_checkouts WHERE workspace_id = $1`, e.ids.ws)
	buy := url.Values{"offer": {"1"}, "_src": {"product_page"}}
	r := e.news(http.MethodPost, "/buy/"+fg, buy, nil)
	must(t, r, http.StatusOK, "Taking you to Stripe…", "Acme Media", "The Deliverability Field Guide", `action="http://news.acme.io/buy/`+fg+`/go"`,
		`name="offer" value="1"`, `name="_src" value="product_page"`)
	if !strings.Contains(r.Header.Get("Cache-Control"), "no-store") {
		t.Errorf("handoff cached: %q", r.Header.Get("Cache-Control"))
	}
	if n := count(t, e.env, `SELECT count(*) FROM product_checkouts WHERE workspace_id = $1`, e.ids.ws); n != checkouts {
		t.Fatal("the handoff page made a checkout")
	}
	r = e.news(http.MethodPost, "/buy/"+fg+"/go", buy, nil)
	to := r.Header.Get("Location")
	if r.Status != http.StatusSeeOther || !strings.HasPrefix(to, "https://checkout.stripe.com/c/pay/cs_live_") {
		t.Fatalf("to Stripe: %d %q\n%s", r.Status, to, clip(r.Body))
	}
	session := path.Base(to)
	var consent map[string]any
	if err := json.Unmarshal(one[[]byte](t, e.env, `SELECT consent FROM product_checkouts WHERE session_id = $1`, session), &consent); err != nil {
		t.Fatal(err)
	}
	if consent["ticked"] != true || consent["page"] != "news.acme.io/products/deliverability-field-guide" || consent["ip"] != "127.0.0.1" {
		t.Errorf("consent at checkout: %v", consent)
	}
	var req stripetest.Recorded
	for _, rq := range e.stripe.Requests() {
		if rq.Path == "/v1/checkout/sessions" {
			req = rq
		}
	}
	if req.Form.Get("success_url") != "http://news.acme.io/products/deliverability-field-guide/thanks?session_id={CHECKOUT_SESSION_ID}" ||
		req.Form.Get("cancel_url") != "http://news.acme.io/products/deliverability-field-guide?checkout=cancelled" || req.Form.Get("mode") != "payment" {
		t.Errorf("session: %v", req.Form)
	}

	// Cancelled on Stripe's page.
	must(t, e.news(http.MethodGet, "/products/deliverability-field-guide?checkout=cancelled", nil, nil), http.StatusOK, "Checkout cancelled. No payment was taken.")
	// Back before the payment arrived.
	thanks := "/products/deliverability-field-guide/thanks?session_id=" + session
	must(t, e.news(http.MethodGet, thanks, nil, nil), http.StatusOK, "Your payment is being confirmed")
	lacks(t, e.news(http.MethodGet, "/products/deliverability-field-guide/thanks", nil, nil), "Download now")

	// Paid.
	if _, err := e.stripe.PayCheckout(payments.Live, session, stripetest.PaymentOptions{Email: "sky.reader@fastmail.com", Name: "Sky Reader"}); err != nil {
		t.Fatal(err)
	}
	r = e.news(http.MethodGet, thanks, nil, nil)
	must(t, r, http.StatusOK, "Thanks, your download is ready", "Download now", "s•••@fastmail.com", "The Deliverability Field Guide")
	if double := one[bool](t, e.env, `SELECT double_opt_in FROM lists WHERE id = $1`, e.ids.digest); double {
		must(t, r, http.StatusOK, "One more step for the Weekly Digest")
	}
	if !strings.Contains(r.Header.Get("Cache-Control"), "no-store") || r.Header.Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("thank-you page headers: %v", r.Header)
	}
	m := sfDownloadLink.FindStringSubmatch(r.Body)
	if m == nil {
		t.Fatalf("no download link:\n%s", clip(r.Body))
	}
	o := one[uuid.UUID](t, e.env, `SELECT id FROM product_orders WHERE session_id = $1 AND status = 'paid'`, session)
	links := e.links(o)
	if len(links) != 1 || on(m[1]) != "/d/"+e.svc.DownloadToken(e.ids.ws, o, links[0].ID) {
		t.Fatalf("download link %s", m[1])
	}

	// The download link: its page out of caches and search engines, its file
	// an attachment (TestDownloadPageSaysWhy checks what the page shows).
	dl := e.news(http.MethodGet, on(m[1]), nil, nil)
	if dl.Status != http.StatusOK || !strings.Contains(dl.Body, `content="noindex, nofollow"`) || !strings.Contains(dl.Header.Get("Cache-Control"), "no-store") ||
		dl.Header.Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("download page: %d %v", dl.Status, dl.Header)
	}
	fileURL := on(m[1]) + "/" + e.firstFile(e.fieldGuide).String()
	file := e.news(http.MethodGet, fileURL, nil, nil)
	if file.Status != http.StatusOK || file.Header.Get("Content-Type") != "application/pdf" || !strings.HasPrefix(file.Header.Get("Content-Disposition"), "attachment;") ||
		!strings.HasPrefix(file.Body, "%PDF-") || !strings.Contains(file.Header.Get("Cache-Control"), "no-store") {
		t.Fatalf("file: %d %v\n%s", file.Status, file.Header, clip(file.Body))
	}
	resumed := e.news(http.MethodGet, fileURL, nil, hdr{"Range": "bytes=5-"})
	if resumed.Status != http.StatusPartialContent || resumed.Body != file.Body[5:] {
		t.Errorf("resumed download: %d", resumed.Status)
	}
	again := e.news(http.MethodGet, fileURL, nil, hdr{"Range": "bytes=0-"})
	if again.Status != http.StatusPartialContent && again.Status != http.StatusOK {
		t.Errorf("download from the start with a range: %d", again.Status)
	}
	if used := e.links(o)[0].DownloadsUsed; used != 2 {
		t.Errorf("%d downloads counted, want 2 (the resumed one doesn't count)", used)
	}
	// A link that isn't one.
	must(t, e.news(http.MethodGet, "/d/not-a-download-link", nil, nil), http.StatusNotFound, "This download link doesn't work")
	must(t, e.news(http.MethodGet, "/d/not-a-download-link/"+e.firstFile(e.fieldGuide).String(), nil, nil), http.StatusNotFound)
}

// What a download link does once it doesn't work any more. Used up, a file
// request goes back to the link's page; "Send a new link" there, whatever
// address is typed in, sends a new link to the order's address only, and
// the old link's files stop. Expired, refunded and revoked links serve no
// file, and a refunded or revoked order takes no request for a new link.
func TestDownloadLinksOnTheSite(t *testing.T) {
	e := sfDemo(t)
	e.deliverable()
	const buyerEmail = "sky.buyer@fastmail.com"
	o, paid := e.bought(e.fieldGuide, buyerEmail)
	link := e.links(o.ID)[0]
	tok := e.svc.DownloadToken(e.ids.ws, o.ID, link.ID)
	file := e.firstFile(e.fieldGuide).String()
	for i := 0; i < int(link.DownloadsAllowed); i++ {
		if r := e.news(http.MethodGet, "/d/"+tok+"/"+file, nil, nil); r.Status != http.StatusOK {
			t.Fatalf("download %d: %d", i+1, r.Status)
		}
	}
	// Used up.
	r := e.news(http.MethodGet, "/d/"+tok+"/"+file, nil, nil)
	if r.Status != http.StatusSeeOther || r.Header.Get("Location") != "http://news.acme.io/d/"+tok {
		t.Fatalf("a used-up link's file: %d %q", r.Status, r.Header.Get("Location"))
	}
	if r := e.news(http.MethodGet, "/d/"+tok, nil, nil); r.Status != http.StatusOK || strings.Contains(r.Text(), buyerEmail) {
		t.Errorf("the used-up link's page: %d", r.Status)
	}

	// Send a new link, whatever address is typed in.
	e.clock.add(time.Minute) // links of the same instant have no order
	if r := e.news(http.MethodPost, "/d/"+tok+"/new-link", url.Values{"email": {"attacker@evil.example"}, "to": {"attacker@evil.example"}}, nil); r.Status != http.StatusOK {
		t.Fatalf("send a new link: %d", r.Status)
	}
	links := e.links(o.ID)
	if len(links) != 2 || links[0].Reason != "buyer" || links[1].ID != link.ID || links[1].SupersededAt == nil {
		t.Fatalf("links: %+v", links)
	}
	if err := e.svc.Deliver(e.ctx, links[0].ID); err != nil {
		t.Fatal(err)
	}
	if n, err := e.env.App.TransactionalOutbox().Step(e.ctx, time.Now()); err != nil || n != 1 {
		t.Fatalf("sent %d: %v", n, err)
	}
	mail := e.sink.Messages()
	if len(mail) != 1 {
		t.Fatalf("the sink got %d messages", len(mail))
	}
	for _, msg := range mail {
		if len(msg.To) != 1 || msg.To[0] != buyerEmail {
			t.Errorf("a new link went to %v", msg.To)
		}
		if strings.Contains(msg.Data, "attacker@evil.example") {
			t.Error("the address typed on the page is in the email")
		}
	}
	if n := count(t, e.env, `SELECT count(*) FROM messages WHERE workspace_id = $1 AND email = 'attacker@evil.example'`, e.ids.ws); n != 0 {
		t.Errorf("%d messages to the address typed on the page", n)
	}
	fresh := e.svc.DownloadToken(e.ids.ws, o.ID, links[0].ID)
	if r := e.news(http.MethodGet, "/d/"+tok+"/"+file, nil, nil); r.Status != http.StatusSeeOther {
		t.Errorf("the replaced link's file: %d", r.Status)
	}
	if r := e.news(http.MethodGet, "/d/"+fresh+"/"+file, nil, nil); r.Status != http.StatusOK {
		t.Errorf("the new link's file: %d", r.Status)
	}

	// Expired.
	e.clock.add(time.Duration(8) * 24 * time.Hour)
	if r := e.news(http.MethodGet, "/d/"+fresh+"/"+file, nil, nil); r.Status != http.StatusSeeOther {
		t.Errorf("an expired link's file: %d", r.Status)
	}

	// Refunded in Stripe: revoked, as the product says.
	ev, err := e.stripe.Refund(payments.Live, paid.Charge, 0)
	if err != nil {
		t.Fatal(err)
	}
	e.hook(ev)
	if r := e.news(http.MethodGet, "/d/"+fresh+"/"+file, nil, nil); r.Status != http.StatusSeeOther {
		t.Errorf("a refunded order's file: %d", r.Status)
	}
	before := len(e.links(o.ID))
	must(t, e.news(http.MethodPost, "/d/"+fresh+"/new-link", url.Values{}, nil), http.StatusNotFound)
	if len(e.links(o.ID)) != before {
		t.Error("a refunded order got a new link")
	}

	// Revoked by a person.
	o2, _ := e.bought(e.fieldGuide, "jamie.revoked@fastmail.com")
	tok2 := e.svc.DownloadToken(e.ids.ws, o2.ID, e.links(o2.ID)[0].ID)
	owner(e.env).Do(http.MethodPost, wsAPI+"/products/orders/"+o2.ID.String()+"/revoke", nil).MustStatus(t, 200)
	if r := e.news(http.MethodGet, "/d/"+tok2+"/"+file, nil, nil); r.Status != http.StatusSeeOther {
		t.Errorf("a revoked order's file: %d", r.Status)
	}
	must(t, e.news(http.MethodPost, "/d/"+tok2+"/new-link", url.Values{}, nil), http.StatusNotFound)
}

// The download page shows the files with the link's expiry and the
// downloads left; once the link doesn't work, it says why, with "Send a new
// link" to the order's masked address where a new link helps.
func TestDownloadPageSaysWhy(t *testing.T) {
	e := sfDemo(t)
	const buyerEmail = "sky.buyer@fastmail.com"
	o, paid := e.bought(e.fieldGuide, buyerEmail)
	link := e.links(o.ID)[0]
	tok := e.svc.DownloadToken(e.ids.ws, o.ID, link.ID)
	file := e.firstFile(e.fieldGuide).String()
	must(t, e.news(http.MethodGet, "/d/"+tok, nil, nil), http.StatusOK, "The Deliverability Field Guide", "5 of 5 downloads left",
		`href="http://news.acme.io/d/`+tok+`/`+file+`"`)
	for i := 0; i < int(link.DownloadsAllowed); i++ {
		e.news(http.MethodGet, "/d/"+tok+"/"+file, nil, nil)
	}
	r := e.news(http.MethodGet, "/d/"+tok, nil, nil)
	must(t, r, http.StatusOK, "This link has reached its 5 downloads", "Send a new link to s•••@fastmail.com", "A new link goes only to the address on the order.",
		`action="http://news.acme.io/d/`+tok+`/new-link"`)
	lacks(t, r, buyerEmail)
	e.clock.add(time.Minute)
	must(t, e.news(http.MethodPost, "/d/"+tok+"/new-link", url.Values{}, nil), http.StatusOK, "Done: a new link is on its way to s•••@fastmail.com.",
		"A newer link replaced this one")
	fresh := e.svc.DownloadToken(e.ids.ws, o.ID, e.links(o.ID)[0].ID)
	must(t, e.news(http.MethodGet, "/d/"+fresh, nil, nil), http.StatusOK, "5 of 5 downloads left")
	e.clock.add(time.Duration(8) * 24 * time.Hour)
	must(t, e.news(http.MethodGet, "/d/"+fresh, nil, nil), http.StatusOK, "This link has expired", "Download links work for 7 days.", "Send a new link to s•••@fastmail.com")
	ev, err := e.stripe.Refund(payments.Live, paid.Charge, 0)
	if err != nil {
		t.Fatal(err)
	}
	e.hook(ev)
	r = e.news(http.MethodGet, "/d/"+fresh, nil, nil)
	must(t, r, http.StatusOK, "This download is no longer available", "This purchase was refunded, so the download is no longer available.")
	lacks(t, r, "Send a new link")
	o2, _ := e.bought(e.fieldGuide, "jamie.revoked@fastmail.com")
	owner(e.env).Do(http.MethodPost, wsAPI+"/products/orders/"+o2.ID.String()+"/revoke", nil).MustStatus(t, 200)
	must(t, e.news(http.MethodGet, "/d/"+e.svc.DownloadToken(e.ids.ws, o2.ID, e.links(o2.ID)[0].ID), nil, nil), http.StatusOK,
		"This download is no longer available", "The seller turned off the downloads of this purchase.")
}

// Only a product on sale can be bought: a draft, an archived product or an
// unknown one answers "isn't for sale right now", and so does every product
// once Stripe is disconnected; Stripe failing answers that nothing was
// charged. None of them reaches Stripe's Checkout.
func TestBuyButtonForProductsNotOnSale(t *testing.T) {
	e := sfDemo(t)
	closed := "This product isn't for sale right now. Nothing was charged."
	sessions := count(t, e.env, `SELECT count(*) FROM product_checkouts WHERE workspace_id = $1 AND session_id <> ''`, e.ids.ws)
	for _, id := range []uuid.UUID{e.planner, uuid.New()} {
		must(t, e.news(http.MethodPost, "/buy/"+id.String(), url.Values{}, nil), http.StatusConflict, closed)
		must(t, e.news(http.MethodPost, "/buy/"+id.String()+"/go", url.Values{}, nil), http.StatusConflict, closed)
	}
	c := owner(e.env)
	c.Do(http.MethodPost, wsAPI+"/products/"+e.launchKit.String()+"/archive", nil).MustStatus(t, 200)
	must(t, e.news(http.MethodPost, "/buy/"+e.launchKit.String()+"/go", url.Values{}, nil), http.StatusConflict, closed)
	must(t, e.news(http.MethodGet, "/products/newsletter-launch-kit", nil, nil), http.StatusNotFound)

	// Stripe down (the client tries three times).
	e.stripe.FailNext(http.MethodPost, "/v1/checkout/sessions", 503, 3)
	must(t, e.news(http.MethodPost, "/buy/"+e.fieldGuide.String()+"/go", url.Values{}, nil), http.StatusServiceUnavailable, "Stripe didn't answer. Nothing was charged.")

	rows, err := e.env.App.Pool.Query(e.ctx, `SELECT id, account_name FROM payment_connections WHERE workspace_id = $1 AND status = 'connected' ORDER BY mode DESC`, e.ids.ws)
	if err != nil {
		t.Fatal(err)
	}
	type conn struct {
		id      uuid.UUID
		account string
	}
	var conns []conn
	for rows.Next() {
		var cn conn
		_ = rows.Scan(&cn.id, &cn.account)
		conns = append(conns, cn)
	}
	rows.Close()
	for _, cn := range conns {
		c.Do(http.MethodPost, wsAPI+"/earn/connections/"+cn.id.String()+"/disconnect", map[string]any{"confirm": cn.account}).MustStatus(t, 204)
	}
	must(t, e.news(http.MethodPost, "/buy/"+e.fieldGuide.String()+"/go", url.Values{}, nil), http.StatusConflict, closed)
	if n := count(t, e.env, `SELECT count(*) FROM product_checkouts WHERE workspace_id = $1 AND session_id <> ''`, e.ids.ws); n != sessions {
		t.Errorf("%d Checkout sessions for products not on sale", n-sessions)
	}
}

// A lead magnet's link (/l/{token}) sends the form's file as an attachment
// while it works; past its days it answers 410 with a way to sign up again.
func TestLeadMagnetFileOnTheSite(t *testing.T) {
	e := sfDemo(t)
	form := one[uuid.UUID](t, e.env, `SELECT id FROM forms WHERE workspace_id = $1 AND download_file_id IS NOT NULL ORDER BY created_at LIMIT 1`, e.ids.ws)
	days := one[int32](t, e.env, `SELECT download_days FROM forms WHERE id = $1`, form)
	contact, ok := contactID(t, e.env, e.ids.ws, "sam.rivera@outlook.com")
	if !ok {
		t.Fatal("no Sam Rivera")
	}
	offer, ok, err := e.svc.LeadMagnetFor(e.ctx, &e.ws, "http://news.acme.io", form, contact)
	if err != nil || !ok || !strings.HasPrefix(offer.URL, "http://news.acme.io/l/") {
		t.Fatalf("offer: %+v %v %v", offer, ok, err)
	}
	r := e.news(http.MethodGet, on(offer.URL), nil, nil)
	if r.Status != http.StatusOK || !strings.HasPrefix(r.Header.Get("Content-Disposition"), "attachment;") || r.Header.Get("Content-Type") != "application/pdf" ||
		!strings.HasPrefix(r.Body, "%PDF-") {
		t.Fatalf("lead magnet file: %d %v", r.Status, r.Header)
	}
	e.clock.add(time.Duration(days) * 24 * time.Hour)
	must(t, e.news(http.MethodGet, on(offer.URL), nil, nil), http.StatusGone, "This download link expired. Sign up again to get a new one.")
	must(t, e.news(http.MethodGet, "/l/not-a-link", nil, nil), http.StatusNotFound)
}

// sfLeadLink is a lead magnet link on the site the page was seen on: the
// custom domain's (http://news.acme.io/l/…) or the instance's
// (/p/acme-newsletter/l/…).
var sfLeadLink = regexp.MustCompile(`href="(https?://[^"/]+(?:/p/acme-newsletter)?/l/[A-Za-z0-9_-]+)"`)

// follow requests a link of the site, on the host it names.
func (e *sfEnv) follow(link string) resp {
	e.t.Helper()
	if strings.HasPrefix(link, "http://news.acme.io/") {
		return e.news(http.MethodGet, on(link), nil, nil)
	}
	return get(e.t, e.env, on(link))
}

// After confirming through a form that offers a download (the DNS records
// checklist page's form), the confirmation page offers the file with a link
// of the reader's own: its name, kind and size, and how many days the link
// works. The link sends the file, and the page shows the same link again
// while it works. A reader who confirmed through a form without a download
// is offered none.
func TestConfirmationPageOffersTheFormDownload(t *testing.T) {
	e := sfDemo(t)
	const email = "lina.lead@fastmail.com"
	form := one[uuid.UUID](t, e.env, `SELECT id FROM forms WHERE workspace_id = $1 AND public_id = 'frm_d4c9a1'`, e.ids.ws)
	name := one[string](t, e.env, `SELECT f.name FROM download_files f JOIN forms fo ON fo.download_file_id = f.id WHERE fo.id = $1`, form)
	size := products.HumanSize(one[int64](t, e.env, `SELECT f.size_bytes FROM download_files f JOIN forms fo ON fo.download_file_id = f.id WHERE fo.id = $1`, form))
	days := one[int32](t, e.env, `SELECT download_days FROM forms WHERE id = $1`, form)
	r := do(t, e.env, http.MethodPost, "/p/acme-newsletter/dns-checklist/form", url.Values{"_t": {formToken(e.env, "frm_d4c9a1", time.Minute)}, "_form": {form.String()},
		"_block": {"form"}, "email": {email}, "first_name": {"Lina"}, "consent": {"1"}, "website": {""}}, hdr{"Origin": e.env.Server.URL})
	must(t, r, http.StatusOK, "Check your inbox")
	contact, ok := contactID(t, e.env, e.ids.ws, email)
	if !ok {
		t.Fatal("no contact")
	}
	confirm := "/p/acme-newsletter/confirm/" + sign(e.env, linktoken.Confirm, e.ids.ws, contact, uuid.Nil)
	r = confirmClick(t, e.env, confirm, nil)
	works := fmt.Sprintf("The download link on this page works for %d days.", days)
	if days == 1 {
		works = "The download link on this page works for 1 day."
	}
	must(t, r, http.StatusOK, name, "Download (PDF, "+size+")", works)
	m := sfLeadLink.FindStringSubmatch(r.Body)
	if m == nil {
		t.Fatalf("no download link on the confirmation page:\n%s", clip(r.Body))
	}
	if n := count(t, e.env, `SELECT count(*) FROM lead_magnet_links WHERE form_id = $1 AND contact_id = $2`, form, contact); n != 1 {
		t.Errorf("%d lead magnet links for the confirmed contact", n)
	}
	file := e.follow(m[1])
	if file.Status != http.StatusOK || file.Header.Get("Content-Type") != "application/pdf" || !strings.HasPrefix(file.Header.Get("Content-Disposition"), "attachment;") ||
		!strings.HasPrefix(file.Body, "%PDF-") {
		t.Fatalf("the form's file %s: %d %v", m[1], file.Status, file.Header)
	}
	if n := one[int32](t, e.env, `SELECT downloads FROM lead_magnet_links WHERE form_id = $1 AND contact_id = $2`, form, contact); n != 1 {
		t.Errorf("%d downloads counted", n)
	}
	// The page again: the same link.
	if again := sfLeadLink.FindStringSubmatch(get(t, e.env, confirm).Body); again == nil || again[1] != m[1] {
		t.Errorf("the confirmation page made another link: %v", again)
	}

	// Through the hosted subscribe page's form, which offers no download.
	must(t, signUp(t, e.env, e.ids, "no.download@fastmail.com"), http.StatusOK, "Check your inbox")
	other, ok := contactID(t, e.env, e.ids.ws, "no.download@fastmail.com")
	if !ok {
		t.Fatal("no contact")
	}
	r = confirmClick(t, e.env, "/p/acme-newsletter/confirm/"+sign(e.env, linktoken.Confirm, e.ids.ws, other, uuid.Nil), nil)
	must(t, r, http.StatusOK)
	lacks(t, r, "Download (")
	if sfLeadLink.MatchString(r.Body) {
		t.Error("a form without a download offers one")
	}
}
