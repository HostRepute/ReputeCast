package coupons_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/coupons"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
	"github.com/hostrepute/reputecast/internal/stores/shopify/shopifytest"
)

const base = "/workspaces/acme-newsletter/coupons"

func upload(t *testing.T, c *apptest.Client, pool, codes string) coupons.CouponCodesResult {
	t.Helper()
	var out coupons.CouponCodesResult
	c.DoRaw("POST", base+"/pools/"+pool+"/codes", "text/plain", strings.NewReader(codes)).MustStatus(t, 200).JSON(t, &out)
	return out
}

func workspace(t *testing.T, env *apptest.Env) store.Workspace {
	t.Helper()
	ws, err := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

var codeInMail = regexp.MustCompile(`\[(OCT-[A-Z0-9]+)\]`)

// An upload pool with 3 codes and a campaign to 5 readers: Send Readiness
// blocks it; with 2 more codes the send gives each reader a code of their
// own, a page sent again gets the same codes, view online shows the
// message's code and previews show the sample.
func TestUploadPoolGivesEachReaderACode(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "M59 codes")
	f.UseFaultProvider(false)
	sendtest.Fault.Reset(sendtest.FaultOK)
	c := s.Env.Client()
	c.SignInDemo("linh@acme.io")

	var pool coupons.CouponPool
	c.Do("POST", base+"/pools", map[string]any{"name": "Partner October", "slug": "partner_oct", "source": "upload", "prefix": "OCT-"}).
		MustStatus(t, 201).JSON(t, &pool)
	if got := upload(t, c, pool.ID, "code\nOCT-AAA1\nOCT-AAA2\noct-aaa2\nOCT-AAA3\nnot a code\n"); got.Added != 3 || got.Duplicates != 1 || got.Invalid != 1 || got.Free != 3 {
		t.Fatalf("upload: %+v", got)
	}
	f.Exec(`UPDATE campaigns SET ab_test = NULL, html = replace(html, '</body>', '<p>Your code: [{{ coupons.partner_oct.code }}]</p></body>') WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 5, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})

	ws := workspace(t, s.Env)
	var html string
	if err := s.Env.App.Pool.QueryRow(f.Ctx, `SELECT html FROM campaigns WHERE id = $1`, f.Campaign).Scan(&html); err != nil {
		t.Fatal(err)
	}
	svc := s.Env.App.Coupons()
	rows := svc.ReadinessRows(f.Ctx, &ws, f.Campaign, html, 5, func(t time.Time) any { return t })
	if len(rows) != 1 || rows[0].Status != "block" || rows[0].Title.In(i18n.Default) != "Not enough discount codes: 3 left for 5 recipients" {
		t.Fatalf("readiness: %+v", rows)
	}
	upload(t, c, pool.ID, "OCT-AAA4\nOCT-AAA5")
	if rows := svc.ReadinessRows(f.Ctx, &ws, f.Campaign, html, 5, func(t time.Time) any { return t }); len(rows) != 1 || rows[0].Status != "pass" {
		t.Fatalf("readiness with 5 codes: %+v", rows)
	}

	f.StartSend(engine.StartOptions{})
	f.Drive(5 * time.Minute)
	if st := f.Status(); st != "sent" {
		f.Dump()
		t.Fatalf("status %s", st)
	}
	type sent struct{ id, contact uuid.UUID }
	var msgs []sent
	addresses := map[uuid.UUID]string{}
	r, err := s.Env.App.Pool.Query(f.Ctx, `SELECT m.id, m.contact_id, c.email FROM messages m JOIN contacts c ON c.id = m.contact_id WHERE m.campaign_id = $1 ORDER BY m.id`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	for r.Next() {
		var m sent
		var email string
		if err := r.Scan(&m.id, &m.contact, &email); err != nil {
			t.Fatal(err)
		}
		msgs, addresses[m.id] = append(msgs, m), email
	}
	r.Close()
	if len(msgs) != 5 {
		t.Fatalf("%d messages", len(msgs))
	}
	given := map[uuid.UUID]string{}
	seen := map[string]bool{}
	for _, m := range msgs {
		got := codeInMail.FindStringSubmatch(sendtest.Fault.HTMLFor(addresses[m.id]))
		if got == nil || seen[got[1]] {
			t.Fatalf("message to %s shows %v (seen %v)", addresses[m.id], got, seen)
		}
		seen[got[1]], given[m.id] = true, got[1]
	}
	if n := f.Count(`SELECT count(*) FROM coupon_codes WHERE pool_id = $1 AND message_id IS NOT NULL AND contact_id IS NOT NULL`, uuid.MustParse(pool.ID)); n != 5 {
		t.Fatalf("%d codes given", n)
	}

	src := "<p>[{{ coupons.partner_oct.code }}]</p>"
	recipients := make([]liquid.Recipient, len(msgs))
	for i, m := range msgs {
		recipients[i] = liquid.Recipient{ContactID: m.contact, MessageID: m.id}
	}
	data := []liquid.RecipientData{svc}
	codeOf := func(vars map[uuid.UUID]map[string]any, contact uuid.UUID) string {
		v, _ := vars[contact][coupons.Root].(map[string]any)["partner_oct"].(map[string]any)
		s, _ := v["code"].(string)
		return s
	}
	// A page sent again (a retry) gets the codes it was given.
	again := liquid.FillBatch(liquid.WithAllocation(f.Ctx), data, ws.ID, recipients, src)
	for _, m := range msgs {
		if got := codeOf(again, m.contact); got != given[m.id] {
			t.Fatalf("sent again: %q, first %q", got, given[m.id])
		}
	}
	// View online reads the message's code.
	online := liquid.FillBatch(f.Ctx, data, ws.ID, recipients[:1], src)
	if got := codeOf(online, msgs[0].contact); got != given[msgs[0].id] {
		t.Fatalf("view online: %q, sent %q", got, given[msgs[0].id])
	}
	// A preview or a test send (no message) shows the sample.
	preview := liquid.FillBatch(f.Ctx, data, ws.ID, []liquid.Recipient{{ContactID: msgs[0].contact}}, src)
	if got := codeOf(preview, msgs[0].contact); got != "OCT-SAMPLE1" {
		t.Fatalf("preview: %q", got)
	}
	// No code is left: a new reader is told why.
	extra := liquid.Recipient{ContactID: msgs[0].contact, MessageID: uuid.Must(uuid.NewV7())}
	f.Exec(`UPDATE coupon_pools SET one_per_contact = false WHERE id = $1`, uuid.MustParse(pool.ID))
	short := liquid.FillBatch(liquid.WithAllocation(f.Ctx), data, ws.ID, []liquid.Recipient{extra}, src)
	if short[extra.ContactID][liquid.ShortKey] != liquid.ShortNoCodes {
		t.Fatalf("out of codes: %+v", short[extra.ContactID])
	}
}

// Two sends taking codes from one pool at once never share a code.
func TestConcurrentAllocationNeverSharesACode(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var pool coupons.CouponPool
	c.Do("POST", base+"/pools", map[string]any{"name": "Race", "slug": "race", "source": "upload", "onePerContact": false}).MustStatus(t, 201).JSON(t, &pool)
	var lines []string
	for i := range 300 {
		lines = append(lines, fmt.Sprintf("RACE-%04d", i))
	}
	upload(t, c, pool.ID, strings.Join(lines, "\n"))
	ws := workspace(t, env)
	var contacts []uuid.UUID
	r, err := env.App.Pool.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 ORDER BY id LIMIT 240`, ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	for r.Next() {
		var id uuid.UUID
		if err := r.Scan(&id); err != nil {
			t.Fatal(err)
		}
		contacts = append(contacts, id)
	}
	r.Close()
	svc := env.App.Coupons()
	src := "{{ coupons.race.code }}"
	var mu sync.Mutex
	codes := map[string]uuid.UUID{}
	var wg sync.WaitGroup
	for half := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range 4 {
				var rs []liquid.Recipient
				for _, contact := range contacts[half*120+page*30 : half*120+(page+1)*30] {
					rs = append(rs, liquid.Recipient{ContactID: contact, MessageID: uuid.Must(uuid.NewV7())})
				}
				got := liquid.FillBatch(liquid.WithAllocation(ctx), []liquid.RecipientData{svc}, ws.ID, rs, src)
				mu.Lock()
				for _, rc := range rs {
					v, _ := got[rc.ContactID][coupons.Root].(map[string]any)["race"].(map[string]any)
					code, _ := v["code"].(string)
					if code == "" {
						t.Errorf("contact %s got no code", rc.ContactID)
					} else if other, ok := codes[code]; ok {
						t.Errorf("%s given to %s and %s", code, other, rc.MessageID)
					}
					codes[code] = rc.MessageID
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(codes) != 240 {
		t.Fatalf("%d codes given", len(codes))
	}
}

// A Shopify pool creates its discount once, then adds codes 250 a call:
// 500 codes are the first code and two discountRedeemCodeBulkAdd calls.
// An order that used a code redeems it, once.
func TestShopifyPoolCreatesCodesAndOrdersRedeemThem(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ws := workspace(t, env)
	var storeID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM stores WHERE workspace_id = $1 AND name = 'Acme Merch'`, ws.ID).Scan(&storeID); err != nil {
		t.Fatal(err)
	}
	var pool coupons.CouponPool
	c.Do("POST", base+"/pools", map[string]any{"name": "Spring 10%", "slug": "spring10", "source": "shopify", "storeId": storeID.String(), "prefix": "SPRING-",
		"settings": map[string]any{"kind": "percentage", "value": 10}}).MustStatus(t, 201).JSON(t, &pool)

	fake := shopifytest.New()
	fake.Scopes = append(fake.Scopes, shopify.DiscountScopes...)
	fake.BulkPolls = 1
	fake.IssueToken("shpat_m59")
	srv := fake.Server()
	t.Cleanup(srv.Close)
	client, err := shopify.New(stores.Config{Domain: fake.Domain, BaseURL: srv.URL, Token: "shpat_m59", HTTP: srv.Client(), RetryWait: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	svc := *env.App.Coupons()
	svc.Jobs = nil
	svc.Discounts = func(context.Context, *store.Workspace, uuid.UUID) (stores.DiscountCreator, error) { return client, nil }
	var b coupons.CouponBatch
	c.Do("POST", base+"/pools/"+pool.ID+"/generate", map[string]any{"count": 500}).MustStatus(t, 202).JSON(t, &b)
	c.Do("POST", base+"/pools/"+pool.ID+"/generate", map[string]any{"count": 10}).MustStatus(t, 409)
	batch := uuid.MustParse(b.ID)
	for range 20 {
		wait, err := svc.Generate(ctx, batch)
		if err != nil {
			t.Fatal(err)
		}
		if wait == 0 {
			break
		}
	}
	adds := fake.BulkAdds()
	if len(adds) != 2 || len(adds[0]) != 250 || len(adds[1]) != 249 || len(fake.Discounts) != 1 {
		t.Fatalf("%d discounts, bulk adds %d", len(fake.Discounts), len(adds))
	}
	var got coupons.CouponPool
	c.Do("GET", base+"/pools/"+pool.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if got.Free != 500 || got.Batch == nil || got.Batch.Status != "done" || got.Batch.Created != 500 || got.Batch.Reason != "pool" || got.Batch.By != "Linh Nguyen" {
		t.Fatalf("pool: free %d, batch %+v", got.Free, got.Batch)
	}
	if !slices.ContainsFunc(fake.Codes(fake.Discounts[0].ID), func(code string) bool { return strings.HasPrefix(code, "SPRING-") && len(code) == 15 }) {
		t.Fatalf("codes %v", fake.Codes(fake.Discounts[0].ID)[:3])
	}

	// An order that used one of the codes redeems it, once.
	code := adds[0][7]
	var orderID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM store_orders WHERE store_id = $1 AND NOT test AND NOT cancelled ORDER BY placed_at DESC LIMIT 1`, storeID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if env.App.Commerce().Redeemed == nil {
		t.Fatal("commerce doesn't tell the pools about orders")
	}
	for range 2 {
		if err := env.App.Commerce().Redeemed(ctx, env.App.Q, ws.ID, orderID, []string{strings.ToLower(code), "SOMETHING-ELSE"}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if n := countRows(t, env, `SELECT count(*) FROM coupon_codes WHERE pool_id = $1 AND order_id = $2 AND redeemed_at IS NOT NULL`, uuid.MustParse(pool.ID), orderID); n != 1 {
		t.Fatalf("%d codes redeemed", n)
	}
}

func countRows(t *testing.T, env *apptest.Env, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Pools are seen with campaigns:view and changed with coupons:manage by
// people; codes are never listed; changes are audited.
func TestPoolAccessAndAudit(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var list coupons.CouponPoolList
	owner.Do("GET", base+"/pools", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 2 || list.Items[0].Name != "Welcome 15%" || list.Items[0].Free != 2540 || list.Items[0].Sent != 9860 || list.Items[0].Redeemed != 312 ||
		list.Items[0].RevenueCents != 1824000 || list.Items[0].Running == nil || list.Items[0].Running.Created != 250 {
		t.Fatalf("list: %+v", list.Items)
	}
	var welcome coupons.CouponPool
	owner.Do("GET", base+"/pools/"+list.Items[0].ID, nil).MustStatus(t, 200).JSON(t, &welcome)
	if welcome.Orders != 312 || len(welcome.Batches) != 4 || !welcome.Refill || welcome.SampleURL != "https://shop.acme.io/discount/WELCOME-SAMPLE1" {
		t.Fatalf("pool: orders %d, %d batches, refill %v, %q", welcome.Orders, len(welcome.Batches), welcome.Refill, welcome.SampleURL)
	}
	var campaign, flow bool
	for _, u := range welcome.UsedIn {
		campaign = campaign || (u.Kind == "campaign" && u.Name == "Black Friday Early Access" && u.Block && u.Status == "draft" && u.Lists == "Weekly Digest")
		flow = flow || (u.Kind == "automation" && u.Name == "Welcome series" && u.Live && u.Block)
	}
	if !campaign || !flow {
		t.Fatalf("used in %+v", welcome.UsedIn)
	}
	raw := owner.Do("GET", base+"/pools/"+list.Items[0].ID, nil).MustStatus(t, 200)
	var body map[string]any
	raw.JSON(t, &body)
	if b, _ := json.Marshal(body); strings.Contains(strings.ToUpper(string(b)), "WELCOME-7KQ4M2XA") {
		t.Fatal("a pool's page shows a code")
	}

	for _, who := range []string{"aisha@acme.io", "daniel@acme.io"} {
		c := env.Client()
		c.SignInDemo(who)
		c.Do("GET", base+"/pools", nil).MustStatus(t, 200)
		c.Do("POST", base+"/pools", map[string]any{"name": "Nope", "slug": "nope", "source": "upload"}).MustStatus(t, 403)
		c.Do("GET", base+"/pools/"+list.Items[1].ID+"/export.csv", nil).MustStatus(t, 403)
	}
	owner.Do("POST", base+"/pools", map[string]any{"name": "Bad", "slug": "Bad Slug", "source": "upload"}).MustStatus(t, 422)
	owner.Do("POST", base+"/pools", map[string]any{"name": "Again", "slug": "welcome15", "source": "upload"}).MustStatus(t, 422)

	csv := owner.Do("GET", base+"/pools/"+list.Items[1].ID+"/export.csv", nil).MustStatus(t, 200)
	if n := countRows(t, env, `SELECT count(*) FROM audit_events WHERE action = 'coupon_pool.codes_exported' AND target_id = $1`, list.Items[1].ID); n != 1 {
		t.Fatalf("%d export entries", n)
	}
	_ = csv
	var archived coupons.CouponPool
	owner.Do("POST", base+"/pools/"+list.Items[0].ID+"/archive", map[string]any{}).MustStatus(t, 409)
	owner.Do("POST", base+"/pools/"+list.Items[0].ID+"/archive", map[string]any{"confirm": true}).MustStatus(t, 200).JSON(t, &archived)
	if archived.Status != "archived" {
		t.Fatalf("status %s", archived.Status)
	}
	if n := countRows(t, env, `SELECT count(*) FROM audit_events WHERE action = 'coupon_pool.archived' AND target_id = $1`, list.Items[0].ID); n != 1 {
		t.Fatalf("%d archive entries", n)
	}
}
