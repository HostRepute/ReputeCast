package earn_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
	"github.com/hostrepute/reputecast/internal/store"
)

// A live restricted key the fake Stripe knows: the MRR ledger and the
// automation events count live subscriptions only.
const liveKey = "rk_live_51QFlowTestKeyForTheFakeStripe0002" //nolint:gosec // a fake key for the in-process Stripe

// connectLive runs the wizard with the live key.
func (f *flow) connectLive() {
	t := f.t
	t.Helper()
	f.fake.AddKey(liveKey, stripetest.KeySpec{})
	var conn struct {
		ID, Mode string
		Webhook  struct{ URL string }
	}
	f.c.Do(http.MethodPost, f.base+"/connections", map[string]any{"provider": "stripe", "key": liveKey}).MustStatus(t, 200).JSON(t, &conn)
	if conn.Mode != "live" {
		t.Fatalf("connect: %+v", conn)
	}
	f.conn = conn.ID
	f.c.Do(http.MethodPost, f.base+"/connections/"+conn.ID+"/webhook", nil).MustStatus(t, 200).JSON(t, &conn)
	f.hook = conn.Webhook.URL
	eps := f.fake.Objects(payments.Live, "webhook_endpoint")
	if len(eps) != 1 {
		t.Fatalf("webhook endpoints in Stripe: %v", eps)
	}
	f.secret = fmt.Sprint(eps[0]["secret"])
	f.c.Do(http.MethodPut, f.base+"/connections/"+conn.ID+"/step", map[string]any{"step": "settings"}).MustStatus(t, 200)
	f.c.Do(http.MethodPost, f.base+"/connections/"+conn.ID+"/finish", map[string]any{"taxAtCheckout": false, "pricesIncludeTax": false,
		"portal": map[string]any{"updateCard": true, "invoices": true, "billingEmail": true, "switchPeriod": true, "cancel": true}}).MustStatus(t, 200)
}

// tierBody is Flow Weekly Insider at $8 a month or $80 a year, without a
// trial, with group prices when group isn't nil.
func (f *flow) tierBody(group map[string]any) map[string]any {
	b := map[string]any{"name": "Flow Weekly Insider", "benefits": []string{"A deep dive every week"}, "monthly": 800, "yearly": 8000,
		"trialDays": 0, "lists": []string{f.list}, "accessRank": 1, "visibility": "public"}
	if group != nil {
		b["group"] = group
	}
	return b
}

// createLiveTier creates the tier, with group prices of $6 a seat a month
// for 3 to 50 seats, and returns its slug.
func (f *flow) createLiveTier() string {
	t := f.t
	t.Helper()
	var saved struct {
		Tier struct {
			ID, Slug, SyncStatus string
			Group                *struct {
				Monthly            *struct{ AmountCents int64 }
				SeatsMin, SeatsMax int
			}
		}
		SyncError string
	}
	f.c.Do(http.MethodPost, f.base+"/tiers", f.tierBody(map[string]any{"monthly": 600, "seatsMin": 3, "seatsMax": 50})).MustStatus(t, 200).JSON(t, &saved)
	if saved.Tier.SyncStatus != "synced" || saved.SyncError != "" || saved.Tier.Group == nil || saved.Tier.Group.Monthly == nil ||
		saved.Tier.Group.Monthly.AmountCents != 600 || saved.Tier.Group.SeatsMin != 3 || saved.Tier.Group.SeatsMax != 50 {
		t.Fatalf("tier: %+v", saved)
	}
	f.tier = saved.Tier.ID
	return saved.Tier.Slug
}

// subscriptionOf is a contact's subscription row of a kind.
func (f *flow) subscriptionOf(email, kind string) store.PaidSubscription {
	f.t.Helper()
	var id uuid.UUID
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT s.id FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND lower(c.email) = lower($2) AND s.kind = $3 ORDER BY s.created_at DESC LIMIT 1`, f.ws.ID, email, kind).Scan(&id); err != nil {
		f.t.Fatalf("%s's %s subscription: %v", email, kind, err)
	}
	sub, err := store.New(f.env.App.Pool).EarnSubscription(context.Background(), store.EarnSubscriptionParams{ID: id, WorkspaceID: f.ws.ID})
	if err != nil {
		f.t.Fatal(err)
	}
	return sub
}

// hasAccess asks the SQL rule whether a contact has paid access now.
func (f *flow) hasAccess(email string) bool {
	f.t.Helper()
	return f.count(`SELECT count(*) FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND lower(c.email) = lower($2) AND earn_paid_access(s.kind, s.status, s.ended_at, s.access_paused, s.paused_at)`, f.ws.ID, email) > 0
}

// mrr is the live MRR, in cents: the ledger's last ARR of every subscription, over 12.
func (f *flow) mrr() int64 {
	f.t.Helper()
	var arr int64
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT coalesce(sum(earn_arr_cents(kind, mode, status, ended_at, period, amount_cents, discount, quantity, paused_at)), 0)::bigint
		FROM paid_subscriptions WHERE workspace_id = $1`, f.ws.ID).Scan(&arr); err != nil {
		f.t.Fatal(err)
	}
	return arr / 12
}

// emailed reports whether a member email was triggered for a contact (a
// draft writes "email_not_sent", a published one "email_sent").
func (f *flow) emailed(email, which string) int {
	f.t.Helper()
	return f.count(`SELECT count(*) FROM paid_events e JOIN contacts c ON c.id = e.contact_id WHERE e.workspace_id = $1 AND lower(c.email) = lower($2)
		AND e.kind IN ('email_sent','email_not_sent') AND e.data->>'email' = $3`, f.ws.ID, email, which)
}

// The plan's team test (M65): a team checkout for 10 seats, two invites
// and one acceptance, a quantity change in Stripe and the end of the
// buyer's subscription.
func TestTeamSubscription(t *testing.T) {
	f := newFlow(t)
	f.connectLive()
	slug := f.createLiveTier()
	ctx := context.Background()
	// The team page opens Checkout for 10 seats.
	form := url.Values{"period": {"month"}, "seats": {"10"}, "team": {"Acme Inc"}, "email": {"buyer@flow.example"}}
	r := f.c.DoRaw(http.MethodPost, "/p/"+f.ws.Slug+"/upgrade/"+slug+"/team", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	loc := r.Header.Get("Location")
	if r.Status != http.StatusSeeOther || !strings.HasPrefix(loc, "https://checkout.stripe.com/c/pay/cs_") {
		t.Fatalf("team checkout: %d %s %s", r.Status, loc, r.Body)
	}
	session := strings.TrimPrefix(loc, "https://checkout.stripe.com/c/pay/")
	if q := f.fake.Quantity(payments.Live, session); q != 10 {
		t.Fatalf("checkout quantity: %d", q)
	}
	evs, err := f.fake.CompleteCheckout(payments.Live, session, stripetest.CheckoutOptions{Email: "buyer@flow.example"})
	if err != nil {
		t.Fatal(err)
	}
	f.deliver(time.Now(), evs...)
	f.apply()
	buyer := f.subscriptionOf("buyer@flow.example", "stripe")
	if buyer.Quantity != 10 || buyer.Status != earn.StatusActive {
		t.Fatalf("buyer: quantity %d, %s", buyer.Quantity, buyer.Status)
	}
	var group uuid.UUID
	var seats int
	var name string
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT id, seats, name FROM earn_groups WHERE subscription_id = $1`, buyer.ID).Scan(&group, &seats, &name); err != nil {
		t.Fatal(err)
	}
	if seats != 10 || name != "Acme Inc" {
		t.Fatalf("team: %d seats, %q", seats, name)
	}
	if m := f.mrr(); m != 600*10 {
		t.Fatalf("MRR: %d, want the unit price × 10", m)
	}
	// The buyer invites two people from the manage page; one accepts.
	g, err := store.New(f.env.App.Pool).EarnGroup(ctx, store.EarnGroupParams{ID: group, WorkspaceID: f.ws.ID})
	if err != nil {
		t.Fatal(err)
	}
	manage := f.svc.ManageTeamURL(ctx, &f.ws, g)
	token := manage[strings.LastIndex(manage, "/")+1:]
	if _, err := f.svc.TeamInvite(ctx, &f.ws, token, "ana@acme.example, ben@acme.example", i18n.Default); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT count(*) FROM earn_group_seats WHERE group_id = $1 AND status = 'invited'`, group); n != 2 {
		t.Fatalf("invited: %d", n)
	}
	var seat uuid.UUID
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT id FROM earn_group_seats WHERE group_id = $1 AND email = 'ana@acme.example'`, group).Scan(&seat); err != nil {
		t.Fatal(err)
	}
	join := f.svc.Links.Sign(linktoken.Claims{Kind: earn.TokenGroupJoin, WorkspaceID: f.ws.ID, MessageID: seat, Link: linktoken.IssueDay(time.Now())})
	if _, err := f.svc.AcceptTeamInvite(ctx, &f.ws, join, earn.JoinInput{IP: "203.0.113.9", UserAgent: "test"}, i18n.Default); err != nil {
		t.Fatal(err)
	}
	ana := f.subscriptionOf("ana@acme.example", earn.KindGroup)
	if ana.Status != earn.StatusGroup || !f.hasAccess("ana@acme.example") {
		t.Fatalf("Ana's seat: %s, access %v", ana.Status, f.hasAccess("ana@acme.example"))
	}
	if f.hasAccess("ben@acme.example") {
		t.Fatal("an invite that wasn't accepted has access")
	}
	// A seat pays nothing of its own: the MRR is still the team's.
	if m := f.mrr(); m != 600*10 {
		t.Fatalf("MRR with a seat: %d", m)
	}
	// The quantity changes in Stripe (the billing portal): the team has 12 seats.
	sub := f.fake.Get(payments.Live, buyer.StripeSubscriptionID)
	items := sub["items"].(map[string]any)["data"].([]map[string]any)
	prev := map[string]any{"items": map[string]any{"data": []map[string]any{{"id": items[0]["id"], "quantity": items[0]["quantity"]}}}}
	items[0]["quantity"] = int64(12)
	f.fake.Put(payments.Live, sub)
	f.deliver(time.Now(), f.fake.Emit(payments.Live, "customer.subscription.updated", clone(sub), prev))
	f.apply()
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT seats FROM earn_groups WHERE id = $1`, group).Scan(&seats); err != nil {
		t.Fatal(err)
	}
	if seats != 12 {
		t.Fatalf("seats after the change in Stripe: %d", seats)
	}
	if m := f.mrr(); m != 600*12 {
		t.Fatalf("MRR after the change: %d", m)
	}
	if n := f.count(`SELECT count(*) FROM earn_mrr_changes WHERE subscription_id = $1 AND kind = 'seats'`, buyer.ID); n != 1 {
		t.Fatalf("seats movements: %d", n)
	}
	// The buyer's subscription ends: the seat's access ends with it, and Ana is told.
	sub = f.fake.Get(payments.Live, buyer.StripeSubscriptionID)
	sub["status"], sub["ended_at"], sub["canceled_at"] = "canceled", time.Now().Unix(), time.Now().Unix()
	f.deliver(time.Now(), f.fake.Emit(payments.Live, "customer.subscription.deleted", clone(sub), nil))
	f.apply()
	if f.hasAccess("ana@acme.example") || f.hasAccess("buyer@flow.example") {
		t.Fatal("access outlived the team's subscription")
	}
	if n := f.emailed("ana@acme.example", earn.EmailGroupAccessEnded); n != 1 {
		t.Fatalf("group_access_ended for Ana: %d", n)
	}
	if m := f.mrr(); m != 0 {
		t.Fatalf("MRR after the end: %d", m)
	}
}

// The plan's pause test (M65): pausing for 3 months voids Stripe's
// collection, ends access, moves the MRR ledger and starts "paid_paused"
// once; resuming brings everything back. The cancel page offers a pause
// only when pausing is on and the member hasn't paused in 12 months.
func TestPauseAndResume(t *testing.T) {
	f := newFlow(t)
	f.connectLive()
	f.createLiveTier()
	ctx := context.Background()
	form := url.Values{"tier": {f.tier}, "period": {"month"}}
	r := f.c.DoRaw(http.MethodPost, "/p/"+f.ws.Slug+"/upgrade/checkout", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	session := strings.TrimPrefix(r.Header.Get("Location"), "https://checkout.stripe.com/c/pay/")
	evs, err := f.fake.CompleteCheckout(payments.Live, session, stripetest.CheckoutOptions{Email: "reader@flow.example"})
	if err != nil {
		t.Fatal(err)
	}
	f.deliver(time.Now(), evs...)
	f.apply()
	sub := f.subscriptionOf("reader@flow.example", "stripe")
	if sub.Status != earn.StatusActive || f.mrr() != 800 {
		t.Fatalf("member: %s, MRR %d", sub.Status, f.mrr())
	}
	// Pausing is off: the cancel page offers no pause.
	if got := f.svc.CancelPauseChoices(ctx, &f.ws, sub); len(got) != 0 {
		t.Fatalf("pause offered while off: %v", got)
	}
	f.c.Do(http.MethodPut, f.base+"/settings/pausing", map[string]any{"months": []int{1, 3}}).MustStatus(t, 200)
	if got := f.svc.CancelPauseChoices(ctx, &f.ws, sub); !slices.Equal(got, []int{1, 3}) {
		t.Fatalf("pause choices: %v", got)
	}
	// Pause for 3 months from the member panel.
	before := len(f.fake.Requests())
	f.c.Do(http.MethodPost, f.base+"/members/"+sub.ID.String()+"/pause", map[string]any{"months": 3}).MustStatus(t, 200)
	voided := false
	for _, rq := range f.fake.Requests()[before:] {
		if rq.Method == http.MethodPost && strings.HasSuffix(rq.Path, "/subscriptions/"+sub.StripeSubscriptionID) && rq.Form.Get("pause_collection[behavior]") == "void" {
			voided = true
		}
	}
	if !voided {
		t.Fatalf("Stripe wasn't asked to void collection: %+v", f.fake.Requests()[before:])
	}
	sub = f.subscriptionOf("reader@flow.example", "stripe")
	if sub.PausedAt == nil || sub.PauseResumesAt == nil || f.hasAccess("reader@flow.example") {
		t.Fatalf("paused: at %v, resumes %v, access %v", sub.PausedAt, sub.PauseResumesAt, f.hasAccess("reader@flow.example"))
	}
	if want := time.Now().AddDate(0, 3, 0); sub.PauseResumesAt.Sub(want).Abs() > 48*time.Hour {
		t.Fatalf("resumes %v, want about %v", sub.PauseResumesAt, want)
	}
	if n := f.count(`SELECT count(*) FROM earn_mrr_changes WHERE subscription_id = $1 AND kind = 'paused'`, sub.ID); n != 1 || f.mrr() != 0 {
		t.Fatalf("paused movements %d, MRR %d", n, f.mrr())
	}
	if n := f.count(`SELECT count(*) FROM automation_events WHERE workspace_id = $1 AND kind = 'paid_paused' AND ref = $2`, f.ws.ID, "sub:"+sub.ID.String()); n != 1 {
		t.Fatalf("paid_paused events: %d", n)
	}
	// Stripe confirms the pause: nothing more happens.
	f.deliver(time.Now(), f.fake.Emit(payments.Live, "customer.subscription.updated", clone(f.fake.Get(payments.Live, sub.StripeSubscriptionID)), map[string]any{"pause_collection": nil}))
	f.apply()
	if n := f.count(`SELECT count(*) FROM automation_events WHERE workspace_id = $1 AND kind = 'paid_paused'`, f.ws.ID); n != 1 {
		t.Fatalf("paid_paused fired %d times", n)
	}
	// A paused member isn't offered another pause.
	if got := f.svc.CancelPauseChoices(ctx, &f.ws, sub); len(got) != 0 {
		t.Fatalf("pause offered while paused: %v", got)
	}
	// Resume now.
	before = len(f.fake.Requests())
	f.c.Do(http.MethodPost, f.base+"/members/"+sub.ID.String()+"/resume", nil).MustStatus(t, 200)
	cleared := false
	for _, rq := range f.fake.Requests()[before:] {
		if rq.Method == http.MethodPost && strings.HasSuffix(rq.Path, "/subscriptions/"+sub.StripeSubscriptionID) && rq.Form.Has("pause_collection") {
			cleared = true
		}
	}
	if !cleared {
		t.Fatalf("Stripe wasn't asked to resume: %+v", f.fake.Requests()[before:])
	}
	sub = f.subscriptionOf("reader@flow.example", "stripe")
	if sub.PausedAt != nil || !f.hasAccess("reader@flow.example") || f.mrr() != 800 {
		t.Fatalf("resumed: paused %v, access %v, MRR %d", sub.PausedAt, f.hasAccess("reader@flow.example"), f.mrr())
	}
	if n := f.count(`SELECT count(*) FROM earn_mrr_changes WHERE subscription_id = $1 AND kind = 'resumed'`, sub.ID); n != 1 {
		t.Fatalf("resumed movements: %d", n)
	}
	if n := f.count(`SELECT count(*) FROM automation_events WHERE workspace_id = $1 AND kind = 'paid_resumed'`, f.ws.ID); n != 1 {
		t.Fatalf("paid_resumed events: %d", n)
	}
	// Paused in the last 12 months: the cancel page offers no pause.
	if got := f.svc.CancelPauseChoices(ctx, &f.ws, sub); len(got) != 0 {
		t.Fatalf("a second pause within 12 months was offered: %v", got)
	}
}

// The SQL earn_paid_access and Go's hasAccess agree on every combination.
func TestPaidAccessAgreesWithSQL(t *testing.T) {
	f := newFlow(t)
	ctx := context.Background()
	kinds := []string{"stripe", "complimentary", "gift", earn.KindGroup}
	statuses := []string{earn.StatusTrialing, earn.StatusActive, earn.StatusPastDue, earn.StatusCanceled, earn.StatusComplimentary, earn.StatusGift, earn.StatusGroup, "incomplete", "unpaid"}
	for _, kind := range kinds {
		for _, status := range statuses {
			for _, ended := range []bool{false, true} {
				for _, accessPaused := range []bool{false, true} {
					for _, paused := range []bool{false, true} {
						var endedAt, pausedAt *time.Time
						now := time.Now()
						if ended {
							endedAt = &now
						}
						if paused {
							pausedAt = &now
						}
						var sql bool
						if err := f.env.App.Pool.QueryRow(ctx, `SELECT earn_paid_access($1, $2, $3, $4, $5)`, kind, status, endedAt, accessPaused, pausedAt).Scan(&sql); err != nil {
							t.Fatal(err)
						}
						if goes := earn.HasAccess(kind, status, ended, accessPaused, paused); goes != sql {
							t.Errorf("%s %s ended=%v accessPaused=%v paused=%v: SQL %v, Go %v", kind, status, ended, accessPaused, paused, sql, goes)
						}
					}
				}
			}
		}
	}
}
