package earn_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
	"github.com/hostrepute/reputecast/internal/store"
)

// The key a fresh workspace connects with: a restricted test key the fake
// Stripe knows.
const flowKey = "rk_test_51QFlowTestKeyForTheFakeStripe0001" //nolint:gosec // a fake key for the in-process Stripe

// flow is a fresh workspace connected to a fake Stripe in test mode.
type flow struct {
	t      *testing.T
	env    *apptest.Env
	c      *apptest.Client
	fake   *stripetest.Fake
	svc    *earn.Service
	ws     store.Workspace
	base   string
	list   string
	tier   string
	conn   string
	hook   string
	secret string
}

func newFlow(t *testing.T) *flow {
	t.Helper()
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "owner@flow.example", Name: "Ola Owner"})
	ws := env.CreateWorkspace("Flow Weekly", u.ID)
	c := env.Client()
	c.SignIn("owner@flow.example", "")
	fake := stripetest.New()
	fake.AddKey(flowKey, stripetest.KeySpec{})
	svc := env.App.Earn()
	svc.StripeHTTP, svc.StripeBase = fake.HTTPClient(), "https://api.stripe.com"
	svc.Reach = func(context.Context, string, string) earn.Reach { return earn.Reach{OK: true, At: time.Now()} }
	f := &flow{t: t, env: env, c: c, fake: fake, svc: svc, ws: ws, base: "/workspaces/" + ws.Slug + "/earn"}
	var list struct{ ID string }
	c.Do(http.MethodPost, "/workspaces/"+ws.Slug+"/lists", map[string]any{"name": "Flow Weekly"}).MustStatus(t, 201).JSON(t, &list)
	f.list = list.ID
	return f
}

// connect runs the wizard: the key, the webhook, checkout and tax.
func (f *flow) connect() {
	t := f.t
	t.Helper()
	var conn struct {
		ID, Mode, KeyHint, Status, Step string
		Webhook                         struct {
			URL                      string
			Registered, SecretStored bool
		}
	}
	f.c.Do(http.MethodPost, f.base+"/connections", map[string]any{"provider": "stripe", "key": flowKey}).MustStatus(t, 200).JSON(t, &conn)
	if conn.Mode != "test" || conn.Step != "webhook" || !strings.HasSuffix(conn.KeyHint, "0001") || strings.Contains(conn.KeyHint, flowKey) {
		t.Fatalf("connect: %+v", conn)
	}
	f.conn = conn.ID
	f.c.Do(http.MethodPost, f.base+"/connections/"+conn.ID+"/webhook", nil).MustStatus(t, 200).JSON(t, &conn)
	if !conn.Webhook.Registered || !conn.Webhook.SecretStored || !strings.Contains(conn.Webhook.URL, "/hooks/stripe/") {
		t.Fatalf("webhook: %+v", conn.Webhook)
	}
	f.hook = conn.Webhook.URL
	eps := f.fake.Objects(payments.Test, "webhook_endpoint")
	if len(eps) != 1 {
		t.Fatalf("webhook endpoints in Stripe: %v", eps)
	}
	f.secret = fmt.Sprint(eps[0]["secret"])
	f.c.Do(http.MethodPut, f.base+"/connections/"+conn.ID+"/step", map[string]any{"step": "settings"}).MustStatus(t, 200)
	var fin struct {
		Connection struct{ Status string }
		Drafts     int
	}
	f.c.Do(http.MethodPost, f.base+"/connections/"+conn.ID+"/finish", map[string]any{"taxAtCheckout": true, "pricesIncludeTax": false,
		"portal": map[string]any{"updateCard": true, "invoices": true, "billingEmail": true, "switchPeriod": true, "cancel": true}}).MustStatus(t, 200).JSON(t, &fin)
	if fin.Connection.Status != "connected" || fin.Drafts != 8 {
		t.Fatalf("finish: %+v", fin)
	}
}

// createTier creates Flow Weekly Insider at $8 a month or $80 a year with a 14-day trial.
func (f *flow) createTier() {
	t := f.t
	t.Helper()
	var saved struct {
		Tier struct {
			ID, SyncStatus string
			Prices         []struct{ Period, StripeTestID string }
		}
		SyncError string
	}
	f.c.Do(http.MethodPost, f.base+"/tiers", map[string]any{"name": "Flow Weekly Insider", "benefits": []string{"A deep dive every week"}, "monthly": 800, "yearly": 8000,
		"trialDays": 14, "lists": []string{f.list}, "accessRank": 1, "visibility": "public", "highlight": "Most popular"}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Tier.SyncStatus != "synced" || saved.SyncError != "" || len(saved.Tier.Prices) != 2 || saved.Tier.Prices[0].StripeTestID == "" {
		t.Fatalf("tier: %+v", saved)
	}
	f.tier = saved.Tier.ID
}

// deliver posts events to the webhook, signed at ts.
func (f *flow) deliver(ts time.Time, evs ...stripetest.Event) {
	f.t.Helper()
	for _, ev := range evs {
		code, err := stripetest.Deliver(context.Background(), nil, f.hook, f.secret, ev, ts)
		if err != nil || code != http.StatusOK {
			f.t.Fatalf("deliver %s: %d %v", ev.Type, code, err)
		}
	}
}

// apply applies the stored events, as the job does.
func (f *flow) apply() {
	f.t.Helper()
	if _, err := f.svc.ApplyPending(context.Background(), time.Now().Add(time.Hour)); err != nil {
		f.t.Fatal(err)
	}
}

type memberRow struct {
	ID, Email, Status, Period, Kind string
	Test, AccessPaused              bool
	LTVCents                        int64 `json:"ltvCents"`
	Source                          struct{ Kind string }
}

func (f *flow) members(status string) []memberRow {
	f.t.Helper()
	var list struct{ Items []memberRow }
	f.c.Do(http.MethodGet, f.base+"/members?test=true&status="+status, nil).MustStatus(f.t, 200).JSON(f.t, &list)
	return list.Items
}

func (f *flow) count(sql string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		f.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func clone(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Secret keys are refused before anything is sent to Stripe, and neither
// they nor a key Stripe rejects are stored.
func TestSecretKeysAreRefused(t *testing.T) {
	f := newFlow(t)
	p := f.c.Do(http.MethodPost, f.base+"/connections", map[string]any{"provider": "stripe", "key": "sk_test_51QThisIsASecretKeyNotRestricted00"}).MustStatus(t, 422).Problem(t)
	if p["code"] != "secret_key" || !strings.Contains(fmt.Sprint(p["detail"]), "Create a restricted key") {
		t.Fatalf("problem: %v", p)
	}
	if n := len(f.fake.Requests()); n != 0 {
		t.Fatalf("a secret key reached Stripe: %d requests", n)
	}
	f.fake.AddKey("rk_test_51QRolledKeyThatStripeRejects00", stripetest.KeySpec{Rejected: true})
	p = f.c.Do(http.MethodPost, f.base+"/connections", map[string]any{"provider": "stripe", "key": "rk_test_51QRolledKeyThatStripeRejects00"}).MustStatus(t, 422).Problem(t)
	if p["code"] != "key_rejected" {
		t.Fatalf("problem: %v", p)
	}
	if n := f.count(`SELECT count(*) FROM payment_connections WHERE workspace_id = $1`, f.ws.ID); n != 0 {
		t.Fatalf("%d connections stored", n)
	}
	// A viewer can't connect at all.
	viewer := f.env.CreateUser(apptest.UserOpts{Email: "viewer@flow.example", Name: "Vic Viewer"})
	f.env.AddMember(f.ws, viewer.ID, "viewer")
	vc := f.env.Client()
	vc.SignIn("viewer@flow.example", "")
	vc.Do(http.MethodPost, f.base+"/connections", map[string]any{"provider": "stripe", "key": flowKey}).MustStatus(t, 403)
}

// A checkout from the upgrade page, then the subscription's life through
// Stripe's webhooks: trial, active, past due, cancel scheduled, ended and
// a refund, with replays, bad signatures and an event out of order.
func TestCheckoutAndWebhooks(t *testing.T) {
	f := newFlow(t)
	f.connect()
	f.createTier()
	ctx := context.Background()
	// Neither the key nor the signing secret ever comes back.
	pay := f.c.Do(http.MethodGet, f.base+"/payments", nil).MustStatus(t, 200)
	if strings.Contains(string(pay.Body), flowKey) || strings.Contains(string(pay.Body), f.secret) {
		t.Fatal("a secret is in Settings › Payments")
	}
	// The upgrade page sells the tier, in test mode.
	page := f.c.Do(http.MethodGet, "/p/"+f.ws.Slug+"/upgrade", nil).MustStatus(t, 200)
	if body := string(page.Body); !strings.Contains(body, "Flow Weekly Insider") || !strings.Contains(body, "Test mode") {
		t.Fatalf("upgrade page: %s", body)
	}
	// Checkout opens with the monthly price, the trial and tax.
	form := url.Values{"tier": {f.tier}, "period": {"month"}}
	r := f.c.DoRaw(http.MethodPost, "/p/"+f.ws.Slug+"/upgrade/checkout", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	loc := r.Header.Get("Location")
	if r.Status != http.StatusSeeOther || !strings.HasPrefix(loc, "https://checkout.stripe.com/c/pay/cs_") {
		t.Fatalf("checkout: %d %s %s", r.Status, loc, r.Body)
	}
	session := strings.TrimPrefix(loc, "https://checkout.stripe.com/c/pay/")
	if price, trial, _, tax := f.fake.Session(payments.Test, session); price == "" || trial != 14 || !tax {
		t.Fatalf("session: price %q, trial %d, tax %v", price, trial, tax)
	}
	auditBefore := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1`, f.ws.ID)
	// The reader pays; Stripe sends its events.
	evs, err := f.fake.CompleteCheckout(payments.Test, session, stripetest.CheckoutOptions{Email: "reader@flow.example"})
	if err != nil {
		t.Fatal(err)
	}
	f.deliver(time.Now(), evs...)
	f.apply()
	rows := f.members("trialing")
	if len(rows) != 1 || rows[0].Email != "reader@flow.example" || !rows[0].Test || rows[0].Period != "month" {
		t.Fatalf("trialing: %+v", rows)
	}
	member := rows[0].ID
	// The welcome email is a draft: it isn't sent, and the timeline says so.
	var m struct {
		Status        string
		LTVCents      int64 `json:"ltvCents"`
		RefundedCents int64 `json:"refundedCents"`
		Timeline      []struct {
			Kind string
			Data map[string]any
		}
	}
	f.c.Do(http.MethodGet, f.base+"/members/"+member, nil).MustStatus(t, 200).JSON(t, &m)
	notSent := false
	for _, x := range m.Timeline {
		if x.Kind == "email_not_sent" && x.Data["email"] == "welcome" {
			notSent = true
		}
		if x.Kind == "email_sent" {
			t.Fatalf("a draft member email was sent: %+v", x)
		}
	}
	if !notSent {
		t.Fatalf("timeline: %+v", m.Timeline)
	}
	// A replay is stored once and changes nothing.
	f.deliver(time.Now(), evs[0])
	if n := f.count(`SELECT count(*) FROM payment_events WHERE workspace_id = $1 AND provider_event_id = $2`, f.ws.ID, evs[0].ID); n != 1 {
		t.Fatalf("stored %d times", n)
	}
	// A bad signature, or a signature older than five minutes, is refused.
	other := f.fake.Emit(payments.Test, "customer.updated", map[string]any{"id": "cus_nobody", "object": "customer", "email": "nobody@flow.example"}, nil)
	if code, _ := stripetest.Deliver(ctx, nil, f.hook, "whsec_notTheSigningSecret0000", other, time.Now()); code != http.StatusBadRequest {
		t.Fatalf("bad signature: %d", code)
	}
	if code, _ := stripetest.Deliver(ctx, nil, f.hook, f.secret, other, time.Now().Add(-time.Hour)); code != http.StatusBadRequest {
		t.Fatalf("old signature: %d", code)
	}
	if n := f.count(`SELECT count(*) FROM payment_events WHERE workspace_id = $1 AND status = 'rejected'`, f.ws.ID); n != 2 {
		t.Fatalf("rejected: %d", n)
	}
	subs := f.fake.Objects(payments.Test, "subscription")
	if len(subs) != 1 {
		t.Fatalf("subscriptions: %v", subs)
	}
	sub := subs[0]
	end := time.Now().AddDate(0, 1, 0).Unix()
	// The trial ends and the first payment goes through: active.
	sub["status"], sub["trial_end"], sub["current_period_end"] = "active", nil, end
	f.deliver(time.Now(), f.fake.Emit(payments.Test, "customer.subscription.updated", clone(sub), map[string]any{"status": "trialing"}),
		f.fake.Emit(payments.Test, "invoice.paid", map[string]any{"id": "in_flow1", "object": "invoice", "customer": sub["customer"], "subscription": sub["id"],
			"amount_paid": 800, "amount_due": 800, "currency": "usd", "billing_reason": "subscription_cycle", "status": "paid", "charge": "ch_flow1"}, nil))
	f.apply()
	if rows := f.members("paying"); len(rows) != 1 || rows[0].Status != "active" || rows[0].LTVCents != 800 {
		t.Fatalf("paying: %+v", rows)
	}
	// A payment fails: past due, with access while Stripe retries.
	sub["status"] = "past_due"
	f.deliver(time.Now(), f.fake.Emit(payments.Test, "invoice.payment_failed", map[string]any{"id": "in_flow2", "object": "invoice", "customer": sub["customer"],
		"subscription": sub["id"], "amount_due": 800, "currency": "usd", "status": "open", "next_payment_attempt": time.Now().Add(72 * time.Hour).Unix()}, nil),
		f.fake.Emit(payments.Test, "customer.subscription.updated", clone(sub), map[string]any{"status": "active"}))
	f.apply()
	if rows := f.members("past_due"); len(rows) != 1 || rows[0].AccessPaused {
		t.Fatalf("past due: %+v", rows)
	}
	// An event from two days ago arrives late: it changes nothing.
	f.fake.Now = func() time.Time { return time.Now().Add(-48 * time.Hour) }
	stale := clone(sub)
	stale["status"] = "active"
	late := f.fake.Emit(payments.Test, "customer.subscription.updated", stale, nil)
	f.fake.Now = time.Now
	f.deliver(time.Now(), late)
	f.apply()
	if rows := f.members("past_due"); len(rows) != 1 {
		t.Fatalf("an event out of order moved the paid member: %+v", rows)
	}
	if n := f.count(`SELECT count(*) FROM payment_events WHERE workspace_id = $1 AND provider_event_id = $2 AND outcome = 'stale'`, f.ws.ID, late.ID); n != 1 {
		t.Fatalf("the late event isn't marked stale: %d", n)
	}
	// Payment goes through, then the reader cancels at the period's end.
	sub["status"], sub["cancel_at_period_end"], sub["cancel_at"] = "active", true, end
	f.deliver(time.Now(), f.fake.Emit(payments.Test, "customer.subscription.updated", clone(sub), map[string]any{"cancel_at_period_end": false, "status": "past_due"}))
	f.apply()
	if rows := f.members("cancels_soon"); len(rows) != 1 {
		t.Fatalf("cancels soon: %+v", rows)
	}
	// The subscription ends.
	sub["status"], sub["ended_at"], sub["canceled_at"] = "canceled", time.Now().Unix(), time.Now().Unix()
	f.deliver(time.Now(), f.fake.Emit(payments.Test, "customer.subscription.deleted", clone(sub), nil))
	f.apply()
	if rows := f.members("ended"); len(rows) != 1 || rows[0].Status != "canceled" {
		t.Fatalf("ended: %+v", rows)
	}
	// A refund comes off the lifetime value.
	f.deliver(time.Now(), f.fake.Emit(payments.Test, "charge.refunded", map[string]any{"id": "ch_flow1", "object": "charge", "customer": sub["customer"],
		"amount": 800, "amount_refunded": 800, "currency": "usd", "refunded": true}, map[string]any{"amount_refunded": 0}))
	f.apply()
	f.c.Do(http.MethodGet, f.base+"/members/"+member, nil).MustStatus(t, 200).JSON(t, &m)
	if m.RefundedCents != 800 || m.LTVCents != 0 {
		t.Fatalf("refund: %+v", m)
	}
	// Webhook-driven changes stay in the payments event log, never in the audit log.
	if after := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1`, f.ws.ID); after != auditBefore {
		t.Fatalf("payment events wrote %d audit entries", after-auditBefore)
	}
	var log struct{ Total int }
	f.c.Do(http.MethodGet, f.base+"/payments/events?test=true&range=24h", nil).MustStatus(t, 200).JSON(t, &log)
	if log.Total < 9 {
		t.Fatalf("event log: %d events", log.Total)
	}
	// Connecting (the first key, so "connected") and the tier did write the audit log.
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action IN ('payments.connected', 'earn.tier_created')`, f.ws.ID); n != 2 {
		t.Fatalf("audit: %d", n)
	}
}

// Complimentary access: given, refused twice, changed and removed, each
// in the audit log.
func TestComplimentaryAccess(t *testing.T) {
	f := newFlow(t)
	f.connect()
	f.createTier()
	f.c.Do(http.MethodPost, "/workspaces/"+f.ws.Slug+"/contacts", map[string]any{"email": "guest@flow.example", "lists": []string{f.list}, "consent": "single"}).MustStatus(t, 201)
	var m struct{ ID, Status, Next string }
	f.c.Do(http.MethodPost, f.base+"/members/complimentary", map[string]any{"email": "guest@flow.example", "tierId": f.tier, "reason": "press", "note": "Covers us",
		"sendWelcome": false}).MustStatus(t, 200).JSON(t, &m)
	if m.Status != "complimentary" || m.Next != "no_end" {
		t.Fatalf("given: %+v", m)
	}
	f.c.Do(http.MethodPost, f.base+"/members/complimentary", map[string]any{"email": "guest@flow.example", "tierId": f.tier, "reason": "press", "sendWelcome": false}).MustStatus(t, 409)
	f.c.Do(http.MethodPut, f.base+"/members/"+m.ID+"/complimentary", map[string]any{"tierId": f.tier, "reason": "press", "until": "2001-01-01"}).MustStatus(t, 422)
	until := time.Now().AddDate(0, 2, 0).Format("2006-01-02")
	f.c.Do(http.MethodPut, f.base+"/members/"+m.ID+"/complimentary", map[string]any{"tierId": f.tier, "reason": "press", "until": until}).MustStatus(t, 200).JSON(t, &m)
	if m.Next != "ends" {
		t.Fatalf("changed: %+v", m)
	}
	f.c.Do(http.MethodDelete, f.base+"/members/"+m.ID+"/complimentary", nil).MustStatus(t, 204)
	if rows := f.members("complimentary"); len(rows) != 0 {
		t.Fatalf("still complimentary: %+v", rows)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action LIKE 'paid_member.complimentary_%'`, f.ws.ID); n != 3 {
		t.Fatalf("audit: %d", n)
	}
}

// An export from another platform is matched with Stripe by customer id,
// then by email; ended subscriptions become free subscribers, rows without
// one complimentary until their paid-through date, and a conflict waits
// for a choice. Nothing is written to Stripe.
func TestImportMatchesStripe(t *testing.T) {
	f := newFlow(t)
	f.connect()
	f.createTier()
	price := f.fake.Objects(payments.Test, "price")[0]
	created := time.Now().AddDate(-1, 0, 0).Unix()
	customer := func(id, email string) {
		f.fake.Put(payments.Test, map[string]any{"id": id, "object": "customer", "email": email, "created": created})
	}
	subscription := func(id, customer, status string) {
		f.fake.Put(payments.Test, map[string]any{"id": id, "object": "subscription", "customer": customer, "status": status, "start_date": time.Now().AddDate(0, -3, 0).Unix(),
			"created": created, "current_period_end": time.Now().AddDate(0, 1, 0).Unix(), "ended_at": nil, "cancel_at": nil, "cancel_at_period_end": false,
			"items": map[string]any{"object": "list", "data": []any{map[string]any{"price": price}}}})
	}
	customer("cus_flowById", "one@flow.example")
	subscription("sub_flowById", "cus_flowById", "active")
	customer("cus_flowByEmail", "two@flow.example")
	subscription("sub_flowByEmail", "cus_flowByEmail", "active")
	customer("cus_flowEnded", "three@flow.example")
	subscription("sub_flowEnded", "cus_flowEnded", "canceled")
	customer("cus_flowDup1", "dup@flow.example")
	customer("cus_flowDup2", "dup@flow.example")
	csv := "email,stripe_customer_id,plan,paid_through,name\n" +
		"one@flow.example,cus_flowById,Insider,,One Reader\n" +
		"two@flow.example,,Insider,,Two Reader\n" +
		"three@flow.example,cus_flowEnded,Insider,,Three Reader\n" +
		"four@flow.example,,Insider," + time.Now().AddDate(0, 6, 0).Format("2006-01-02") + ",Four Reader\n" +
		"dup@flow.example,,Insider,,Dup Reader\n"
	type group struct{ Count, ByCustomer, ByEmail int }
	var imp struct {
		ID, Status, Problem string
		Rows                int
		Active, Ended, None *group
		Conflicts           []struct{ Row int }
		Stats               *struct{ Paid, Free, Complimentary, New, Skipped int }
	}
	f.c.DoRaw(http.MethodPost, f.base+"/imports?source=csv&fileName=export.csv", "application/octet-stream", strings.NewReader(csv)).MustStatus(t, 200).JSON(t, &imp)
	if imp.Rows != 5 || imp.Problem != "" {
		t.Fatalf("upload: %+v", imp)
	}
	writes := func() int {
		n := 0
		for _, r := range f.fake.Requests() {
			if r.Method != http.MethodGet {
				n++
			}
		}
		return n
	}
	before := writes()
	f.c.Do(http.MethodPost, f.base+"/imports/"+imp.ID+"/match", nil).MustStatus(t, 200).JSON(t, &imp)
	if imp.Active == nil || imp.Active.Count != 2 || imp.Active.ByCustomer != 1 || imp.Active.ByEmail != 1 || imp.Ended == nil || imp.Ended.Count != 1 ||
		imp.None == nil || imp.None.Count != 1 || len(imp.Conflicts) != 1 {
		t.Fatalf("match: %+v", imp)
	}
	// The conflict needs a choice first.
	p := f.c.Do(http.MethodPost, f.base+"/imports/"+imp.ID+"/start", nil).MustStatus(t, 422).Problem(t)
	if p["code"] != "unsettled_conflict" {
		t.Fatalf("start: %v", p)
	}
	f.c.Do(http.MethodPut, f.base+"/imports/"+imp.ID+"/decisions", map[string]any{"noSubscription": "complimentary",
		"conflicts": map[string]string{fmt.Sprint(imp.Conflicts[0].Row): "cus_flowDup1"}, "sendWelcome": false}).MustStatus(t, 200)
	f.c.Do(http.MethodPost, f.base+"/imports/"+imp.ID+"/start", nil).MustStatus(t, 200)
	if err := f.svc.RunImport(context.Background(), uuid.MustParse(imp.ID)); err != nil {
		t.Fatal(err)
	}
	f.c.Do(http.MethodGet, f.base+"/imports/"+imp.ID, nil).MustStatus(t, 200).JSON(t, &imp)
	if imp.Status != "done" || imp.Stats == nil || imp.Stats.Paid != 2 || imp.Stats.Free != 2 || imp.Stats.Complimentary != 1 || imp.Stats.New != 5 || imp.Stats.Skipped != 0 {
		t.Fatalf("import: %+v %+v", imp, imp.Stats)
	}
	if n := writes(); n != before {
		t.Fatalf("the import wrote to Stripe %d times", n-before)
	}
	paying := f.members("paying")
	if len(paying) != 2 || paying[0].Source.Kind != "import" {
		t.Fatalf("paying: %+v", paying)
	}
	if rows := f.members("complimentary"); len(rows) != 1 || rows[0].Email != "four@flow.example" {
		t.Fatalf("complimentary: %+v", rows)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'paid_member.imported'`, f.ws.ID); n != 1 {
		t.Fatalf("audit: %d", n)
	}
}

// Member emails go out through Transactional when they're published: a
// failed payment for Sam Rivera in the demo sends "payment failed".
func TestMemberEmailsAreSent(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	svc := env.App.Earn()
	ctx := context.Background()
	// The event as Stripe sends it to the demo's live webhook.
	ev := stripetest.New().Emit(payments.Live, "invoice.payment_failed", map[string]any{"id": "in_flowSamFailed", "object": "invoice", "customer": "cus_QmT4wE8sLd",
		"subscription": "sub_1Q8xRb2Va7", "amount_due": 8000, "currency": "usd", "status": "open", "next_payment_attempt": time.Now().Add(72 * time.Hour).Unix()}, nil)
	hook := env.Server.URL + "/hooks/stripe/7Gk2pQx9LmT4vB8sNw3eRfJh"
	if code, err := stripetest.Deliver(ctx, nil, hook, "whsec_demoAcmeLiveSigningSecret0001", ev, time.Now()); err != nil || code != http.StatusOK {
		t.Fatalf("deliver: %d %v", code, err)
	}
	if _, err := svc.ApplyPending(ctx, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var list struct{ Items []struct{ ID, Status string } }
	c.Do(http.MethodGet, "/workspaces/acme-newsletter/earn/members?status=past_due&q=sam.rivera", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 {
		t.Fatalf("Sam isn't past due: %+v", list.Items)
	}
	var m struct {
		Timeline []struct {
			Kind      string
			MessageID string `json:"messageId"`
			Data      map[string]any
		}
	}
	c.Do(http.MethodGet, "/workspaces/acme-newsletter/earn/members/"+list.Items[0].ID, nil).MustStatus(t, 200).JSON(t, &m)
	sent := false
	for _, x := range m.Timeline {
		if x.Kind == "email_sent" && x.Data["email"] == "payment_failed" && x.MessageID != "" {
			sent = true
		}
		if x.Kind == "email_not_sent" && x.Data["email"] == "payment_failed" {
			t.Fatalf("not sent: %+v", x)
		}
	}
	if !sent {
		t.Fatalf("timeline: %+v", m.Timeline)
	}
	// The same event again sends nothing more.
	if code, _ := stripetest.Deliver(ctx, nil, hook, "whsec_demoAcmeLiveSigningSecret0001", ev, time.Now()); code != http.StatusOK {
		t.Fatalf("replay: %d", code)
	}
	if _, err := svc.ApplyPending(ctx, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM paid_events WHERE kind = 'email_sent' AND data->>'email' = 'payment_failed' AND contact_id =
		(SELECT id FROM contacts WHERE email = 'sam.rivera@outlook.com' AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter'))`).Scan(&n)
	if n != 1 {
		t.Fatalf("payment failed sent %d times", n)
	}
}
