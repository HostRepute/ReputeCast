package revenue_test

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/revenue"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracking"
)

const day = 24 * time.Hour

// fixture is a workspace with paid subscriptions written straight to the
// database at chosen moments: the trigger keeps the ledger as Stripe's
// events would.
type fixture struct {
	t   *testing.T
	env *apptest.Env
	ws  store.Workspace
	now time.Time
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		f.t.Fatalf("%s: %v", sql, err)
	}
}

func (f *fixture) ago(days float64) time.Time { return f.now.Add(-time.Duration(days * float64(day))) }

func (f *fixture) contact(email string) uuid.UUID {
	id := uuid.New()
	f.exec(`INSERT INTO contacts (id, workspace_id, email) VALUES ($1, $2, $3)`, id, f.ws.ID, email)
	return id
}

func (f *fixture) sub(contact, tier uuid.UUID, mode, status, period string, amount int64, at time.Time) uuid.UUID {
	id := uuid.New()
	f.exec(`INSERT INTO paid_subscriptions (id, workspace_id, contact_id, mode, kind, tier_id, status, period, amount_cents, currency, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'stripe', $5, $6, $7, $8, 'usd', $9, $9, $9)`, id, f.ws.ID, contact, mode, tier, status, period, amount, at)
	return id
}

func near(got, want int64) bool { return math.Abs(float64(got-want)) <= 1 }

// fresh resets a result before decoding into it: an answer without
// "credited" must not keep the previous one's.
func fresh(r *revenue.ConversionResult) *revenue.ConversionResult {
	*r = revenue.ConversionResult{}
	return r
}

func trackingTarget(ws, msg uuid.UUID, u string) tracking.ClickTarget {
	return tracking.ClickTarget{WorkspaceID: ws, MessageID: msg, URL: u, At: time.Now()}
}

// MRR, its movement and churn follow from the ledger: new, reactivated,
// expansion, contraction and churn add up from the start to the end, a
// scheduled cancellation still counts until access ends, and trials,
// complimentary access and test mode count nothing.
func TestMRRMovementAndChurnFromTheLedger(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@revenue-x17.example", Name: "Rita Owner"})
	f := &fixture{t: t, env: env, ws: env.CreateWorkspace("Revenue X17", owner.ID), now: time.Now()}
	f.exec(`INSERT INTO earn_settings (workspace_id, currency) VALUES ($1, 'usd')`, f.ws.ID)
	insider, founding := uuid.New(), uuid.New()
	f.exec(`INSERT INTO earn_tiers (id, workspace_id, name, slug, access_rank) VALUES ($1, $3, 'Insider', 'insider', 1), ($2, $3, 'Founding member', 'founding', 2)`,
		insider, founding, f.ws.ID)

	// Paying since before the window.
	f.sub(f.contact("a@x17.example"), insider, "live", "active", "month", 800, f.ago(60))
	f.sub(f.contact("past-due@x17.example"), insider, "live", "past_due", "month", 800, f.ago(70))
	scheduled := f.sub(f.contact("scheduled@x17.example"), insider, "live", "active", "month", 800, f.ago(45))
	f.exec(`UPDATE paid_subscriptions SET cancel_at = $2, updated_at = $3 WHERE id = $1`, scheduled, f.now.Add(10*day), f.ago(2))
	churner := f.contact("churn@x17.example")
	churned := f.sub(churner, insider, "live", "active", "month", 800, f.ago(50))
	discounted := f.sub(f.contact("offer@x17.example"), insider, "live", "active", "month", 800, f.ago(40))
	upgraded := f.sub(f.contact("upgrade@x17.example"), insider, "live", "active", "month", 800, f.ago(40))
	// In the window: a new yearly member, a churn, an offer, an upgrade and
	// the churned reader paying again.
	f.sub(f.contact("new@x17.example"), insider, "live", "active", "year", 8000, f.ago(10))
	f.exec(`UPDATE paid_subscriptions SET status = 'canceled', ended_at = $2, end_reason = 'canceled', updated_at = $2 WHERE id = $1`, churned, f.ago(5))
	f.exec(`UPDATE paid_subscriptions SET discount = '{"label":"Stay for half price","amountCents":400,"duration":"repeating","months":3}', updated_at = $2 WHERE id = $1`,
		discounted, f.ago(3))
	f.exec(`UPDATE paid_subscriptions SET tier_id = $2, period = 'year', amount_cents = 20000, updated_at = $3 WHERE id = $1`, upgraded, founding, f.ago(2))
	f.sub(churner, insider, "live", "active", "month", 800, f.ago(1))
	// Never in MRR.
	f.sub(f.contact("trial@x17.example"), insider, "live", "trialing", "month", 800, f.ago(4))
	f.sub(f.contact("test@x17.example"), insider, "test", "active", "month", 800, f.ago(8))
	f.exec(`INSERT INTO paid_subscriptions (id, workspace_id, contact_id, mode, kind, tier_id, status, started_at) VALUES ($1, $2, $3, 'live', 'complimentary', $4, 'complimentary', $5)`,
		uuid.New(), f.ws.ID, f.contact("comp@x17.example"), insider, f.ago(20))

	svc := env.App.Revenue()
	p := &authz.Principal{Type: authz.ActorUser, UserID: owner.ID, Permissions: authz.NewSet(authz.EarnView, authz.EarnEdit)}
	ov, err := svc.Overview(ctx, &f.ws, p, revenue.OverviewQuery{Period: "30d"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Failed) > 0 {
		t.Fatalf("failed cards: %v", ov.Failed)
	}
	// Now: a, past due, scheduled ($8 each), new ($80/year), offer ($4),
	// upgraded ($200/year), the reactivated reader ($8).
	wantArr := int64(9600 + 9600 + 9600 + 8000 + 4800 + 20000 + 9600)
	m := ov.Movement
	if !near(m.ToCents, wantArr/12) || ov.Tiles.ARRCents != wantArr || m.ToMembers != 7 {
		t.Fatalf("end: %+v, ARR %d", m, ov.Tiles.ARRCents)
	}
	// At the start: a, past due, scheduled, churner, offer, upgraded at $8.
	if !near(m.FromCents, 6*800) || m.FromMembers != 6 {
		t.Fatalf("start: %+v", m)
	}
	if m.New.Count != 1 || !near(m.New.Cents, 8000/12) {
		t.Errorf("new: %+v", m.New)
	}
	if m.Reactivated.Count != 1 || !near(m.Reactivated.Cents, 800) {
		t.Errorf("reactivated: %+v", m.Reactivated)
	}
	if m.Churned.Count != 1 || !near(m.Churned.Cents, -800) {
		t.Errorf("churned: %+v", m.Churned)
	}
	if m.Contraction.Count != 1 || !near(m.Contraction.Cents, -400) {
		t.Errorf("contraction: %+v", m.Contraction)
	}
	if m.Expansion.Count != 1 || !near(m.Expansion.Cents, (20000-9600)/12) {
		t.Errorf("expansion: %+v", m.Expansion)
	}
	if sum := m.FromCents + m.New.Cents + m.Reactivated.Cents + m.Expansion.Cents + m.Contraction.Cents + m.Churned.Cents; !near(sum, m.ToCents) && math.Abs(float64(sum-m.ToCents)) > 3 {
		t.Errorf("the movement doesn't add up: %d → %d", sum, m.ToCents)
	}
	tl := ov.Tiles
	if tl.Members != 7 || tl.MembersDelta != 1 || tl.New != 2 || tl.Churned != 1 || tl.ChurnBase != 6 {
		t.Errorf("tiles: %+v", tl)
	}
	if tl.ChurnRate == nil || math.Abs(*tl.ChurnRate-100.0/6) > 0.01 {
		t.Errorf("churn rate %v, want 16.67%%", tl.ChurnRate)
	}
	if tl.Trialing != 1 {
		t.Errorf("trialing %d", tl.Trialing)
	}
	// By tier: Insider monthly holds the offer; Founding member includes Insider.
	var monthly, yearlyFounding *revenue.RevenueTierRow
	for i := range ov.Tiers {
		r := &ov.Tiers[i]
		if r.Tier == "Insider" && r.Period == "month" {
			monthly = r
		}
		if r.Tier == "Founding member" && r.Period == "year" {
			yearlyFounding = r
		}
	}
	if monthly == nil || monthly.Members != 5 || len(monthly.Offers) != 1 || monthly.Offers[0].Name != "Stay for half price" || monthly.Offers[0].PaidCents != 400 ||
		monthly.Offers[0].Duration != "months" || monthly.Offers[0].Months != 3 {
		t.Errorf("Insider monthly: %+v", monthly)
	}
	if yearlyFounding == nil || yearlyFounding.Members != 1 || len(yearlyFounding.Includes) != 1 || yearlyFounding.Includes[0] != "Insider" {
		t.Errorf("Founding member: %+v", yearlyFounding)
	}
	// The line: twelve months back, ending at today's MRR.
	if n := len(ov.Chart.Points); n < 13 || !near(ov.Chart.Points[n-1].MRRCents, wantArr/12) {
		t.Errorf("chart: %+v", ov.Chart.Points)
	}
	// Ending a scheduled cancellation is churn when access ends.
	f.exec(`UPDATE paid_subscriptions SET status = 'canceled', ended_at = $2, end_reason = 'canceled', updated_at = $2 WHERE id = $1`, scheduled, f.now.Add(-time.Hour))
	ov, err = svc.Overview(ctx, &f.ws, p, revenue.OverviewQuery{Period: "30d"})
	if err != nil {
		t.Fatal(err)
	}
	if ov.Movement.Churned.Count != 2 || ov.Tiles.Members != 6 {
		t.Errorf("after the scheduled end: %+v, %d members", ov.Movement.Churned, ov.Tiles.Members)
	}
	// Read-only people are told who can change Earn.
	viewer := &authz.Principal{Type: authz.ActorUser, UserID: uuid.New(), Permissions: authz.NewSet(authz.EarnView)}
	ov, err = svc.Overview(ctx, &f.ws, viewer, revenue.OverviewQuery{Period: "90d"})
	if err != nil {
		t.Fatal(err)
	}
	if !ov.ReadOnly || len(ov.Helpers) == 0 || ov.Helpers[0] != "Rita Owner" {
		t.Errorf("read-only: %v %v", ov.ReadOnly, ov.Helpers)
	}
}

// A workspace connected days ago, with no older subscriptions, is too
// early for 30-day numbers: churn and free-to-paid wait.
func TestTooEarly(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "early@revenue-x17.example", Name: "Eli Early"})
	f := &fixture{t: t, env: env, ws: env.CreateWorkspace("Early X17", owner.ID), now: time.Now()}
	f.exec(`INSERT INTO earn_settings (workspace_id, currency) VALUES ($1, 'usd')`, f.ws.ID)
	f.exec(`INSERT INTO payment_connections (id, workspace_id, mode, status, account_name, connected_at) VALUES ($1, $2, 'live', 'connected', 'Early Media', $3)`,
		uuid.New(), f.ws.ID, f.ago(9))
	tier := uuid.New()
	f.exec(`INSERT INTO earn_tiers (id, workspace_id, name, slug) VALUES ($1, $2, 'Insider', 'insider')`, tier, f.ws.ID)
	for i := range 3 {
		f.sub(f.contact(fmt.Sprintf("m%d@early.example", i)), tier, "live", "active", "month", 800, f.ago(float64(8-i)))
	}
	p := &authz.Principal{Type: authz.ActorUser, UserID: owner.ID, Permissions: authz.NewSet(authz.EarnView)}
	ov, err := env.App.Revenue().Overview(ctx, &f.ws, p, revenue.OverviewQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if !ov.TooEarly || ov.ShownFrom == nil || ov.Tiles.ChurnRate != nil || ov.Tiles.FreeToPaidRate != nil || ov.Account != "Early Media" {
		t.Fatalf("too early: %+v", ov)
	}
	if ov.Movement.New.Count != 3 || ov.Movement.FromCents != 0 || !near(ov.Movement.ToCents, 2400) {
		t.Errorf("so far: %+v", ov.Movement)
	}
	if got := ov.ShownFrom.Sub(*ov.DataSince); got < 29*day || got > 31*day {
		t.Errorf("shown from %v, data since %v", ov.ShownFrom, ov.DataSince)
	}
}

// demoClick finds a campaign message of the demo and records a person's
// click on it at a chosen moment.
func demoClick(t *testing.T, env *apptest.Env, at time.Time) (wsID, message, campaign, contact uuid.UUID, email string) {
	t.Helper()
	ctx := context.Background()
	err := env.App.Pool.QueryRow(ctx, `SELECT m.workspace_id, m.id, m.campaign_id, m.contact_id, c.email FROM messages m JOIN contacts c ON c.id = m.contact_id
		JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme-newsletter' AND m.stream = 'broadcast' AND m.campaign_id IS NOT NULL AND c.email LIKE '%@%' AND c.erased_at IS NULL
		ORDER BY m.id LIMIT 1`).Scan(&wsID, &message, &campaign, &contact, &email)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM delivery_events WHERE contact_id = $1 AND type = 'click'`, contact); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, type, occurred_at, received_at, machine)
		VALUES ($1, $2, $3, $4, $5, 'click', $6, $6, false)`, uuid.New(), wsID, message, campaign, contact, at); err != nil {
		t.Fatal(err)
	}
	return
}

// The conversion API credits a conversion to the email clicked within the
// window: by click token first, else by address; outside the window it's
// kept but not credited. Its errors have their own codes, a reused key
// answers with the first conversion, and a key without conversions:create
// can't record.
func TestConversionAttribution(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	const ws = "/workspaces/acme-newsletter"
	type created struct {
		Secret string `json:"secret"`
	}
	var key created
	owner.Do("POST", ws+"/api-keys", map[string]any{"name": "Product X17 · conversions", "permissions": []string{"conversions:create"}}).MustStatus(t, 201).JSON(t, &key)
	k := env.Client()
	k.Bearer = key.Secret
	now := time.Now()
	_, _, campaign, contact, email := demoClick(t, env, now.Add(-2*day))

	// By address: the click two days ago is within the 7-day window.
	var r revenue.ConversionResult
	k.Do("POST", ws+"/conversions", map[string]any{"email": strings.ToUpper(email), "name": "Upgraded to Team plan", "value": "29.00", "currency": "usd"}).
		MustStatus(t, 202).JSON(t, &r)
	if r.Matched != "email" || r.Credited == nil || r.Credited.ID != campaign.String() || r.Credited.Kind != "campaign" {
		t.Fatalf("by email: %+v", r)
	}
	var cents int64
	var cur string
	if err := env.App.Pool.QueryRow(ctx, `SELECT value_cents, currency FROM conversions WHERE id = $1`, r.ID).Scan(&cents, &cur); err != nil || cents != 2900 || cur != "usd" {
		t.Fatalf("stored %d %s: %v", cents, cur, err)
	}
	// A click token signed for this click credits it without an address.
	svc := env.App.Revenue()
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	var msg uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT message_id FROM delivery_events WHERE contact_id = $1 AND type = 'click'`, contact).Scan(&msg); err != nil {
		t.Fatal(err)
	}
	tok := svc.ClickTokenFor(wsID, msg, contact, now.Add(-time.Hour))
	k.Do("POST", ws+"/conversions", map[string]any{"click_token": tok, "name": "Upgraded to Team plan", "value": "29.00", "currency": "USD"}).MustStatus(t, 202).JSON(t, fresh(&r))
	if r.Matched != "click_token" || r.Credited == nil || r.Credited.ID != campaign.String() {
		t.Fatalf("by token: %+v", r)
	}
	// The campaign report's Revenue section counts them with their value.
	var cr revenue.CampaignRevenue
	owner.Do("GET", ws+"/campaigns/"+campaign.String()+"/revenue", nil).MustStatus(t, 200).JSON(t, &cr)
	var n, sum int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*), coalesce(sum(value_cents), 0) FROM conversions WHERE campaign_id = $1 AND NOT test`, campaign).Scan(&n, &sum); err != nil {
		t.Fatal(err)
	}
	if !cr.Active || cr.Conversions.Count != n || cr.Conversions.ValueCents != sum || cr.Conversions.ByToken < 1 || cr.Conversions.ByEmail < 1 {
		t.Fatalf("campaign revenue: %+v, want %d worth %d", cr.Conversions, n, sum)
	}
	// Shorten the window to a day: the token from three days ago and the
	// click from two days ago are both outside it.
	owner.Do("PUT", ws+"/earn/attribution", map[string]any{"days": 1, "clickToken": true, "domains": []string{"https://App.Acme.io/pricing"}}).MustStatus(t, 200)
	old := svc.ClickTokenFor(wsID, msg, contact, now.Add(-3*day))
	k.Do("POST", ws+"/conversions", map[string]any{"click_token": old, "name": "Upgraded to Team plan", "value": "29.00", "currency": "USD"}).MustStatus(t, 202).JSON(t, fresh(&r))
	if r.Matched != "none" || r.Credited != nil {
		t.Fatalf("expired token: %+v", r)
	}
	k.Do("POST", ws+"/conversions", map[string]any{"email": email, "name": "Upgraded to Team plan", "value": "29.00", "currency": "USD"}).MustStatus(t, 202).JSON(t, fresh(&r))
	if r.Matched != "email" || r.Credited != nil {
		t.Fatalf("outside the window: %+v", r)
	}
	var set revenue.AttributionSettings
	owner.Do("GET", ws+"/earn/attribution", nil).MustStatus(t, 200).JSON(t, &set)
	if set.Days != 1 || !set.ClickToken || len(set.Domains) != 1 || set.Domains[0] != "app.acme.io" || len(set.Keys) == 0 || !set.Keys[len(set.Keys)-1].Only {
		t.Fatalf("settings: %+v", set)
	}
	var audited int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'earn.attribution_updated'`).Scan(&audited); err != nil || audited != 1 {
		t.Fatalf("audited %d: %v", audited, err)
	}

	// Errors.
	for _, c := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"name": "x", "value": "1.00", "currency": "USD"}, revenue.CodeMissingIdentity},
		{map[string]any{"email": email, "name": "x", "value": 29, "currency": "USD"}, revenue.CodeInvalidValue},
		{map[string]any{"email": email, "name": "x", "value": "29.001", "currency": "USD"}, revenue.CodeInvalidValue},
		{map[string]any{"email": email, "name": "x", "value": "29.00", "currency": "XQZ"}, revenue.CodeInvalidCurrency},
		{map[string]any{"email": email, "name": "x", "value": "29.00", "currency": "USD", "occurred_at": now.Add(-40 * day).Format(time.RFC3339)}, revenue.CodeOccurredAtRange},
	} {
		res := k.Do("POST", ws+"/conversions", c.body).MustStatus(t, 422)
		if got := res.Problem(t)["code"]; got != c.code {
			t.Errorf("%v: code %v, want %s", c.body, got, c.code)
		}
	}
	// Idempotency: the same key answers with the first conversion; a
	// different body with the same key is a conflict.
	k.Header = map[string][]string{"Idempotency-Key": {"upgrade-42"}}
	var a, b revenue.ConversionResult
	body := map[string]any{"email": email, "name": "Upgraded to Team plan", "value": "29.00", "currency": "USD"}
	k.Do("POST", ws+"/conversions", body).MustStatus(t, 202).JSON(t, &a)
	k.Do("POST", ws+"/conversions", body).MustStatus(t, 202).JSON(t, &b)
	if a.ID != b.ID {
		t.Errorf("idempotent ids %s and %s", a.ID, b.ID)
	}
	body["value"] = "49.00"
	if got := k.Do("POST", ws+"/conversions", body).MustStatus(t, 409).Problem(t)["code"]; got != revenue.CodeIdempotencyReuse {
		t.Errorf("reused key: %v", got)
	}
	k.Header = nil
	// A key without conversions:create can't record.
	var reader created
	owner.Do("POST", ws+"/api-keys", map[string]any{"name": "Reader", "permissions": []string{"campaigns:view"}}).MustStatus(t, 201).JSON(t, &reader)
	rk := env.Client()
	rk.Bearer = reader.Secret
	rk.Do("POST", ws+"/conversions", map[string]any{"email": email, "name": "x", "value": "1.00", "currency": "USD"}).MustStatus(t, 403)
	// The log lists them, newest first, and none are test conversions.
	var log revenue.ConversionList
	owner.Do("GET", ws+"/conversions?limit=3", nil).MustStatus(t, 200).JSON(t, &log)
	if len(log.Items) != 3 || log.Next == nil || log.Window != 1 || log.Items[0].Test {
		t.Fatalf("log: %+v", log)
	}
}

// The click token goes on links to the publisher's own domains only, never
// on a sponsor's link, and a person's click on an upgrade link counts for
// the campaign's Revenue section.
func TestClickTokenAndUpgradeClicks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	svc := env.App.Revenue()
	var wsID, msg, campaign uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT m.workspace_id, m.id, m.campaign_id FROM messages m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme-newsletter' AND m.campaign_id IS NOT NULL ORDER BY m.id LIMIT 1`).Scan(&wsID, &msg, &campaign); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE earn_settings SET click_token_on = true, click_token_domains = '{app.acme.io}' WHERE workspace_id = $1`, wsID); err != nil {
		t.Fatal(err)
	}
	click := func(u string) string {
		return svc.Decorate(ctx, trackingTarget(wsID, msg, u))
	}
	if got := click("https://app.acme.io/pricing?plan=team"); !strings.HasPrefix(got, "https://app.acme.io/pricing?plan=team&rc_click=") {
		t.Errorf("own domain: %q", got)
	}
	if got := click("https://eu.app.acme.io/x"); !strings.Contains(got, "rc_click=") {
		t.Errorf("a subdomain: %q", got)
	}
	if got := click("https://northwind-analytics.example/trial"); got != "" {
		t.Errorf("another domain: %q", got)
	}
	// A sponsor's link on the publisher's domain still gets no token.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE sponsor_bookings SET campaign_id = $1, creative = jsonb_set(creative, '{link}', '"https://app.acme.io/partners/northwind"')
		WHERE id = (SELECT id FROM sponsor_bookings WHERE status = 'booked' ORDER BY issue_number LIMIT 1)`, campaign); err != nil {
		t.Fatal(err)
	}
	if got := click("https://app.acme.io/partners/northwind/"); got != "" {
		t.Errorf("sponsor link: %q", got)
	}
	// Upgrade clicks.
	for _, u := range []string{"https://news.acme.io/upgrade", "http://localhost:8080/p/acme-newsletter/upgrade/founding", "{{ upgrade_url }}"} {
		if !revenue.IsUpgradeLink(u) {
			t.Errorf("%s is an upgrade link", u)
		}
	}
	for _, u := range []string{"https://news.acme.io/upgrades-guide", "https://news.acme.io/upgrade/checkout", "https://app.acme.io/pricing"} {
		if revenue.IsUpgradeLink(u) {
			t.Errorf("%s isn't an upgrade link", u)
		}
	}
}
