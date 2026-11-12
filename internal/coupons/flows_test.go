package coupons_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/coupons"
	"github.com/hostrepute/reputecast/internal/liquid"
)

// clock is a controllable "now" for the automation engine.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

const flowsPath = "/workspaces/acme-newsletter/automations"

// flowEmail is an automation email whose first version's HTML is html.
func flowEmail(t *testing.T, env *apptest.Env, c *apptest.Client, name, html string) string {
	t.Helper()
	var e struct{ ID string }
	c.Do("POST", "/workspaces/acme-newsletter/automation-emails", map[string]any{"name": name, "subject": name}).MustStatus(t, 201).JSON(t, &e)
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE automation_email_versions SET html = $2 WHERE email_id = $1`, e.ID, html); err != nil {
		t.Fatal(err)
	}
	return e.ID
}

// codesFlow publishes a flow: joining list, then the email, then a tag.
func codesFlow(t *testing.T, c *apptest.Client, name, list, email string) string {
	t.Helper()
	g := automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			{ID: "t", Type: automations.NodeTrigger, Data: automations.AutomationNodeData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: list}}},
			{ID: "m", Type: automations.NodeEmail, Data: automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: email, EmailVersion: 1, Subject: name}}},
			{ID: "tag", Type: automations.NodeList, Data: automations.AutomationNodeData{Action: &automations.AutomationListAction{Kind: "add_tag", Tag: "after-code"}}},
		},
		Edges: []automations.AutomationEdge{
			{ID: "t-m", Source: "t", Target: "m"},
			{ID: "m-tag", Source: "m", Target: "tag"},
		},
	}
	var created struct{ ID string }
	c.Do("POST", flowsPath, map[string]any{"name": name}).MustStatus(t, 201).JSON(t, &created)
	var saved struct {
		DraftVersion int
		Issues       []struct{ Code, Message string }
	}
	c.Do("PUT", flowsPath+"/"+created.ID+"/draft", map[string]any{"graph": g, "revision": 0}).MustStatus(t, 200).JSON(t, &saved)
	if len(saved.Issues) > 0 {
		t.Fatalf("flow %q has issues: %+v", name, saved.Issues)
	}
	c.Do("POST", flowsPath+"/"+created.ID+"/publish", map[string]any{"version": saved.DraftVersion, "migration": "finish"}).MustStatus(t, 200)
	return created.ID
}

// newList is an empty list of the demo workspace, so only the test's
// contacts join it.
func newList(t *testing.T, env *apptest.Env, name string) string {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO lists (id, workspace_id, name) SELECT $1, id, $2 FROM workspaces WHERE slug = 'acme-newsletter'`, id, name); err != nil {
		t.Fatal(err)
	}
	return id.String()
}

// joiner adds a contact and subscribes it to list.
func joiner(t *testing.T, env *apptest.Env, email, list string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name, timezone, attributes)
		SELECT $1, id, $2, 'Test', 'America/New_York', '{}' FROM workspaces WHERE slug = 'acme-newsletter'`, id, email); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at) VALUES ($1, $2, 'subscribed', 'form:test', now())`, list, id); err != nil {
		t.Fatal(err)
	}
	return id
}

type flowRun struct {
	node, status string
	state        struct {
		Wait      string            `json:"wait"`
		HeldSince *time.Time        `json:"heldSince"`
		Skipped   map[string]string `json:"skipped"`
	}
}

func runOf(t *testing.T, env *apptest.Env, flow string, contact uuid.UUID) flowRun {
	t.Helper()
	var r flowRun
	var raw []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT node_id, status, state FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, flow, contact).
		Scan(&r.node, &r.status, &raw); err != nil {
		t.Fatalf("run of %s: %v", contact, err)
	}
	if err := json.Unmarshal(raw, &r.state); err != nil {
		t.Fatal(err)
	}
	return r
}

// lowAlerts counts a member's "running low" notifications about a pool,
// by severity.
func lowAlerts(t *testing.T, env *apptest.Env, member, pool string) map[string]int64 {
	t.Helper()
	rows, err := env.App.Pool.Query(context.Background(), `SELECT n.severity, count(*) FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE u.email = $1 AND n.kind = 'coupon_codes_low' AND n.data ->> 'pool' = $2 GROUP BY n.severity`, member, pool)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var sev string
		var n int64
		if err := rows.Scan(&sev, &n); err != nil {
			t.Fatal(err)
		}
		out[sev] = n
	}
	return out
}

func newPool(t *testing.T, c *apptest.Client, in map[string]any) coupons.CouponPool {
	t.Helper()
	var p coupons.CouponPool
	c.Do("POST", base+"/pools", in).MustStatus(t, 201).JSON(t, &p)
	return p
}

// An automation email that shows a pool's code waits while the pool has
// none: the run looks again every 15 minutes and sends once codes come;
// after 24 hours it goes on without the email (no_codes in the run's
// state). The pool's managers hear once that it is out of codes.
func TestFlowEmailsWaitForCodesThenSkip(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	later := newPool(t, c, map[string]any{"name": "Comes later", "slug": "comes_later", "source": "upload", "prefix": "LATE-"})
	never := newPool(t, c, map[string]any{"name": "Never filled", "slug": "never_filled", "source": "upload", "prefix": "NEVER-"})
	list := newList(t, env, "M59 flow codes")
	flowLater := codesFlow(t, c, "Codes come later", list, flowEmail(t, env, c, "Later", `<p>{{ coupons.comes_later.code }}</p>`))
	// The Discount code block counts as showing the pool's code too.
	flowNever := codesFlow(t, c, "Codes never come", list, flowEmail(t, env, c, "Never",
		`<!-- rc:coupon {"pool":"never_filled","label":"Your code","button":"Shop now","buttonUrl":false} -->`))
	contacts := []uuid.UUID{joiner(t, env, "codes-a@example.com", list), joiner(t, env, "codes-b@example.com", list)}

	clk := &clock{t: time.Now()}
	svc := &automations.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Log: env.App.Log, Now: clk.now,
		EmailHolds: []automations.EmailHold{env.App.Coupons().EmailHold}}
	tick := func() {
		t.Helper()
		if _, err := svc.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	tick()
	stat := func(flow, col string) int64 {
		t.Helper()
		return countRows(t, env, `SELECT coalesce(sum(`+col+`), 0) FROM automation_step_stats WHERE automation_id = $1 AND node_id = 'm'`, flow)
	}
	for _, flow := range []string{flowLater, flowNever} {
		for _, contact := range contacts {
			if r := runOf(t, env, flow, contact); r.node != "m" || r.status != "waiting" || r.state.Wait != "hold" || r.state.HeldSince == nil {
				t.Fatalf("a run without a code: %s %s %+v", r.node, r.status, r.state)
			}
		}
		if n := stat(flow, "waiting"); n != 2 {
			t.Fatalf("%d runs waiting at the email", n)
		}
	}
	for _, p := range []coupons.CouponPool{later, never} {
		if got := lowAlerts(t, env, "linh@acme.io", p.ID); got["bad"] != 1 || len(got) != 1 {
			t.Fatalf("%s: out-of-codes alerts %v", p.Name, got)
		}
		if got := lowAlerts(t, env, "aisha@acme.io", p.ID); len(got) != 0 {
			t.Fatalf("a member without coupons:manage was told: %v", got)
		}
	}
	var title string
	if err := env.App.Pool.QueryRow(ctx, `SELECT title FROM notifications WHERE kind = 'coupon_codes_low' AND data ->> 'pool' = $1 LIMIT 1`, never.ID).Scan(&title); err != nil ||
		title != "Never filled has no discount codes left" {
		t.Fatalf("title %q (%v)", title, err)
	}

	// 15 minutes on, still no code: the runs keep waiting, nobody is told again.
	clk.advance(automations.EmailHoldRecheck)
	tick()
	if r := runOf(t, env, flowLater, contacts[0]); r.status != "waiting" || r.state.Wait != "hold" {
		t.Fatalf("after 15 minutes: %s %+v", r.status, r.state)
	}
	if got := lowAlerts(t, env, "linh@acme.io", never.ID); got["bad"] != 1 {
		t.Fatalf("told again: %v", got)
	}

	// Codes come: at the next look both emails of the first flow go.
	upload(t, c, later.ID, "LATE-AAA1\nLATE-AAA2")
	clk.advance(automations.EmailHoldRecheck)
	tick()
	for _, contact := range contacts {
		if r := runOf(t, env, flowLater, contact); r.node == "m" || r.state.HeldSince != nil || r.state.Skipped != nil {
			t.Fatalf("with codes the run is at %s %s %+v", r.node, r.status, r.state)
		}
	}
	if n := stat(flowLater, "sent"); n != 2 {
		t.Fatalf("%d emails sent once codes came", n)
	}
	if r := runOf(t, env, flowNever, contacts[0]); r.node != "m" || r.status != "waiting" {
		t.Fatalf("the other flow: %s %s", r.node, r.status)
	}

	// A day after the hold began the second flow goes on without its email.
	clk.advance(automations.EmailHoldMax)
	tick()
	for _, contact := range contacts {
		r := runOf(t, env, flowNever, contact)
		if r.node == "m" || r.state.Skipped["m"] != liquid.ShortNoCodes || r.state.HeldSince != nil {
			t.Fatalf("after a day: %s %s %+v", r.node, r.status, r.state)
		}
	}
	if sent, skipped := stat(flowNever, "sent"), stat(flowNever, "skipped"); sent != 0 || skipped != 2 {
		t.Fatalf("the skipped flow: %d sent, %d skipped", sent, skipped)
	}
}

// codeRows adds n codes to a pool straight in the table (the refill
// counts them, it never opens them); assignedAgo > 0 marks them given
// that long ago.
func codeRows(t *testing.T, env *apptest.Env, pool string, n int, assignedAgo time.Duration) {
	t.Helper()
	var assigned *time.Time
	if assignedAgo > 0 {
		at := time.Now().Add(-assignedAgo)
		assigned = &at
	}
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO coupon_codes (id, workspace_id, pool_id, code_sealed, code_hash, assigned_at)
		SELECT gen_random_uuid(), p.workspace_id, p.id, '\x00'::bytea, sha256((p.id::text || $4 || g)::bytea), $3
		FROM coupon_pools p, generate_series(1, $2::int) g WHERE p.id = $1`, pool, n, assigned, fmt.Sprintf(":%d:", assignedAgo)); err != nil {
		t.Fatal(err)
	}
}

// The hourly refill keeps store pools that live automations send with
// max(low mark, 1.2 × the last 7 days' codes) free: a pool with 50 free
// that gave 250 codes this week gets a batch of 250 more for its flow,
// once. A store pool no live flow uses isn't refilled; both are under
// their low mark, so their managers hear it once.
func TestRefillKeepsLiveFlowPoolsStocked(t *testing.T) {
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
	shop := func(name, slug string) coupons.CouponPool {
		return newPool(t, c, map[string]any{"name": name, "slug": slug, "source": "shopify", "storeId": storeID.String(), "prefix": "R-", "lowAt": 100,
			"settings": map[string]any{"kind": "percentage", "value": 10}})
	}
	live, idle := shop("Flow refill", "flow_refill"), shop("Idle store pool", "idle_store")
	codeRows(t, env, live.ID, 50, 0)
	codeRows(t, env, live.ID, 250, 24*time.Hour)
	codeRows(t, env, live.ID, 400, 10*24*time.Hour) // before the 7 days
	codeRows(t, env, idle.ID, 10, 0)
	codeRows(t, env, idle.ID, 500, 24*time.Hour)
	flow := codesFlow(t, c, "Refill flow", newList(t, env, "M59 refill"), flowEmail(t, env, c, "Refill", `<p>{{ coupons.flow_refill.code }}</p>`))

	svc := *env.App.Coupons()
	svc.Jobs = nil
	for range 2 {
		if err := svc.Refill(ctx); err != nil {
			t.Fatal(err)
		}
	}
	type batch struct {
		reason, status string
		requested      int32
		automation     *uuid.UUID
		by             *uuid.UUID
	}
	batches := func(pool string) []batch {
		t.Helper()
		rows, err := env.App.Pool.Query(ctx, `SELECT reason, status, requested, automation_id, created_by FROM coupon_batches WHERE pool_id = $1`, pool)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []batch
		for rows.Next() {
			var b batch
			if err := rows.Scan(&b.reason, &b.status, &b.requested, &b.automation, &b.by); err != nil {
				t.Fatal(err)
			}
			out = append(out, b)
		}
		return out
	}
	got := batches(live.ID)
	if len(got) != 1 || got[0].reason != "refill" || got[0].status != "running" || got[0].requested != 250 || got[0].by != nil ||
		got[0].automation == nil || got[0].automation.String() != flow {
		t.Fatalf("refill batches %+v", got)
	}
	if got := batches(idle.ID); len(got) != 0 {
		t.Fatalf("a pool no live flow uses was refilled: %+v", got)
	}
	if n := countRows(t, env, `SELECT count(*) FROM coupon_pools WHERE id = $1 AND refill_checked_at IS NOT NULL`, live.ID); n != 1 {
		t.Fatal("the refill didn't note its check")
	}
	if n := countRows(t, env, `SELECT count(*) FROM audit_events WHERE action = 'coupon_pool.codes_requested' AND target_id = $1 AND actor_type = 'system'`, live.ID); n != 1 {
		t.Fatalf("%d system audit entries", n)
	}
	var pool coupons.CouponPool
	c.Do("GET", base+"/pools/"+live.ID, nil).MustStatus(t, 200).JSON(t, &pool)
	if pool.Batch == nil || pool.Batch.Reason != "refill" || pool.Batch.Automation != "Refill flow" {
		t.Fatalf("the pool page's batch %+v", pool.Batch)
	}
	for _, p := range []coupons.CouponPool{live, idle} {
		if got := lowAlerts(t, env, "linh@acme.io", p.ID); got["warn"] != 1 || len(got) != 1 {
			t.Fatalf("%s: low alerts %v", p.Name, got)
		}
	}
}

// A pool under its low mark tells its managers once (the hourly pass),
// not again while it stays low; codes above the mark reset it, and a
// reader left without a code tells them again, as out of codes.
func TestLowCodesAlertOnceUntilRefilled(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ws := workspace(t, env)
	pool := newPool(t, c, map[string]any{"name": "Small pool", "slug": "small_pool", "source": "upload", "prefix": "SM-", "lowAt": 3, "onePerContact": false})
	upload(t, c, pool.ID, "SM-AAA1\nSM-AAA2\nSM-AAA3\nSM-AAA4")

	var contact uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL ORDER BY id LIMIT 1`, ws.ID).Scan(&contact); err != nil {
		t.Fatal(err)
	}
	give := func(n int) (short int) {
		t.Helper()
		for range n {
			r := liquid.Recipient{ContactID: contact, MessageID: uuid.Must(uuid.NewV7())}
			got := liquid.FillBatch(liquid.WithAllocation(ctx), []liquid.RecipientData{env.App.Coupons()}, ws.ID, []liquid.Recipient{r}, "{{ coupons.small_pool.code }}")
			if got[contact][liquid.ShortKey] == liquid.ShortNoCodes {
				short++
			}
		}
		return short
	}
	svc := *env.App.Coupons()
	svc.Jobs = nil
	hourly := func() {
		t.Helper()
		if err := svc.Refill(ctx); err != nil {
			t.Fatal(err)
		}
	}
	hourly()
	if got := lowAlerts(t, env, "linh@acme.io", pool.ID); len(got) != 0 {
		t.Fatalf("4 codes, low mark 3: %v", got)
	}
	if short := give(2); short != 0 {
		t.Fatalf("%d readers without a code", short)
	}
	hourly()
	hourly()
	if got := lowAlerts(t, env, "linh@acme.io", pool.ID); got["warn"] != 1 || len(got) != 1 {
		t.Fatalf("2 codes left: %v", got)
	}
	for _, who := range []string{"aisha@acme.io", "daniel@acme.io"} {
		if got := lowAlerts(t, env, who, pool.ID); len(got) != 0 {
			t.Fatalf("%s was told: %v", who, got)
		}
	}
	var link, body string
	if err := env.App.Pool.QueryRow(ctx, `SELECT link, body FROM notifications WHERE kind = 'coupon_codes_low' AND data ->> 'pool' = $1`, pool.ID).Scan(&link, &body); err != nil {
		t.Fatal(err)
	}
	if link != "/w/acme-newsletter/campaigns/codes/"+pool.ID || !strings.HasPrefix(body, "2 codes are left, under its low mark of 3.") {
		t.Fatalf("link %q, body %q", link, body)
	}

	// Codes above the mark reset the alert; running out tells them again.
	upload(t, c, pool.ID, "SM-BBB1\nSM-BBB2\nSM-BBB3\nSM-BBB4\nSM-BBB5")
	if n := countRows(t, env, `SELECT count(*) FROM coupon_pools WHERE id = $1 AND notified_low_at IS NULL`, pool.ID); n != 1 {
		t.Fatal("codes above the low mark didn't reset the alert")
	}
	if short := give(8); short != 1 {
		t.Fatalf("%d readers without a code, want 1", short)
	}
	if got := lowAlerts(t, env, "linh@acme.io", pool.ID); got["warn"] != 1 || got["bad"] != 1 {
		t.Fatalf("after running out: %v", got)
	}
}

// The data export opens the codes a contact was given; erasing the
// contact clears who got the code and in which message, and the code
// stays given, so it's never handed out again.
func TestCodesInDataExportAndErasure(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ws := workspace(t, env)
	pool := newPool(t, c, map[string]any{"name": "Export pool", "slug": "export_pool", "source": "upload", "prefix": "EXP-"})
	upload(t, c, pool.ID, "EXP-AAA1\nEXP-AAA2")

	var contact uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL ORDER BY id DESC LIMIT 1`, ws.ID).Scan(&contact); err != nil {
		t.Fatal(err)
	}
	r := liquid.Recipient{ContactID: contact, MessageID: uuid.Must(uuid.NewV7())}
	vars := liquid.FillBatch(liquid.WithAllocation(ctx), []liquid.RecipientData{env.App.Coupons()}, ws.ID, []liquid.Recipient{r}, "{{ coupons.export_pool.code }}")
	v, _ := vars[contact][coupons.Root].(map[string]any)["export_pool"].(map[string]any)
	given, _ := v["code"].(string)
	if !strings.HasPrefix(given, "EXP-AAA") {
		t.Fatalf("given %q", given)
	}

	var doc struct {
		Sections map[string]json.RawMessage `json:"sections"`
		Complete bool                       `json:"complete"`
	}
	c.Do("GET", "/workspaces/acme-newsletter/contacts/"+contact.String()+"/export", nil).MustStatus(t, 200).JSON(t, &doc)
	var rows []map[string]any
	if err := json.Unmarshal(doc.Sections["discountCodes"], &rows); err != nil {
		t.Fatalf("discountCodes: %v (%s)", err, doc.Sections["discountCodes"])
	}
	var found map[string]any
	for _, row := range rows {
		if row["pool"] == "Export pool" {
			found = row
		}
	}
	if !doc.Complete || found == nil || found["code"] != given || found["assigned_at"] == nil || found["code_ad"] != nil {
		t.Fatalf("complete %v, rows %v", doc.Complete, rows)
	}

	c.Do("POST", "/workspaces/acme-newsletter/contacts/erase", map[string]any{"ids": []string{contact.String()}, "confirm": 1}).MustStatus(t, 200)
	if n := countRows(t, env, `SELECT count(*) FROM coupon_codes WHERE contact_id = $1 OR message_id = $2`, contact, r.MessageID); n != 0 {
		t.Fatalf("%d codes still name the erased contact", n)
	}
	if n := countRows(t, env, `SELECT count(*) FROM coupon_codes WHERE pool_id = $1 AND assigned_at IS NOT NULL`, uuid.MustParse(pool.ID)); n != 1 {
		t.Fatalf("%d codes given after the erasure, want 1", n)
	}
	var after coupons.CouponPool
	c.Do("GET", base+"/pools/"+pool.ID, nil).MustStatus(t, 200).JSON(t, &after)
	if after.Free != 1 || after.Sent != 1 {
		t.Fatalf("after the erasure: %d free, %d sent", after.Free, after.Sent)
	}
}
