package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// The demo's restricted keys. They aren't Stripe keys: the demo's
// connections talk to an in-process Stripe (internal/app/wire_earn.go).
const (
	EarnLiveKey = "rk_live_51QAcmeDemoOnlyNotARealStripeKey9fQ2" //nolint:gosec // demo key for the in-process Stripe, documented in the README
	EarnTestKey = "rk_test_51QAcmeDemoOnlyNotARealStripeKey3mZ8" //nolint:gosec // demo key for the in-process Stripe
)

func init() { seedEarn = seedEarnImpl }

// earnSeed carries the Earn sample (brief 4): Acme Media's Stripe
// connection, the two tiers, 1,252 paid members with access, the import
// from Substack, the member emails and the payments event log.
type earnSeed struct {
	s   *Seeder
	ws  store.Workspace
	q   *store.Queries
	rng *rand.Rand

	linh, marcus *uuid.UUID
	digest       uuid.UUID

	live, test          uuid.UUID
	insider, founding   uuid.UUID
	pMonth, pYear, pFnd uuid.UUID
	imp                 uuid.UUID
	sender              uuid.UUID
	category            uuid.UUID
	templates           map[string]uuid.UUID

	// contacts by sample key; pool are Weekly Digest subscribers to draw
	// generated paid members from.
	contacts map[string]earnContact
	pool     []earnContact
	next     int

	subs   [][]any
	events [][]any
	// byKey are the sample rows' subscription ids.
	byKey map[string]uuid.UUID
	// monthly are generated monthly members, for renewals in the event log.
	monthly []earnRow
	yearly  []earnRow
	order   int
}

type earnContact struct {
	id                 uuid.UUID
	email, first, last string
}

func (c earnContact) name() string { return strings.TrimSpace(c.first + " " + c.last) }

// earnRow is one paid subscription of the sample.
type earnRow struct {
	key        string
	c          earnContact
	mode       string
	kind       string
	tier       uuid.UUID
	price      *uuid.UUID
	stripePr   string
	status     string
	period     string
	amount     int64
	discount   map[string]any
	started    time.Time
	trialEnd   *time.Time
	periodEnd  *time.Time
	cancelAt   *time.Time
	canceled   *time.Time
	canceledBy *uuid.UUID
	ended      *time.Time
	endReason  string
	cancelWhy  string
	paid       int64
	refunded   int64
	failedAt   *time.Time
	failure    string
	retryAt    *time.Time
	srcKind    string
	srcCamp    *uuid.UUID
	srcLabel   string
	srcDetail  map[string]any
	grantWhy   string
	grantNote  string
	grantedBy  *uuid.UUID
	grantTill  *time.Time
	imported   bool
	emailed    []string
	updated    time.Time
}

// earnDay is a moment of the Earn sample's calendar, where today is Wed 15
// Oct 2026 and the clock reads 10:00 in New York: earlier days as
// designDay dates them, today's times counted back from 10:00 (09:56 is
// four minutes ago), later days ahead of designDay's today.
func (s *Seeder) earnDay(y int, m time.Month, d, hour, minute int) time.Time {
	days := int(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC).Sub(time.Date(y, m, d, 0, 0, 0, 0, time.UTC)).Hours() / 24)
	if days == 0 {
		return s.Ago(time.Duration(10*60-(hour*60+minute)) * time.Minute)
	}
	return s.designDay(days, hour, minute)
}

func tp(t time.Time) *time.Time { return &t }

func seedEarnImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	digest := s.Lists["Weekly Digest"]
	if digest == uuid.Nil {
		return nil
	}
	e := &earnSeed{s: s, ws: ws, q: store.New(s.Tx), rng: rand.New(rand.NewPCG(1252, 8960)), //nolint:gosec // deterministic sample data
		digest: digest, templates: map[string]uuid.UUID{}, contacts: map[string]earnContact{}, byKey: map[string]uuid.UUID{}}
	if u, ok := s.Users["linh@acme.io"]; ok {
		e.linh = &u.ID
	}
	if u, ok := s.Users["marcus@acme.io"]; ok {
		e.marcus = &u.ID
	}
	for _, step := range []func(context.Context) error{e.settings, e.connections, e.tiers, e.memberEmails, e.people, e.importRun, e.members,
		e.write, e.eventLog, e.checkouts, e.visits, e.audit} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

// settings: USD, Stripe Tax at checkout with prices before tax, every
// portal feature, access kept while Stripe retries, the upgrade page.
func (e *earnSeed) settings(ctx context.Context) error {
	// The upgrade page is written by tiers(), once the tiers exist.
	_, err := e.s.Tx.Exec(ctx, `INSERT INTO earn_settings (workspace_id, currency, tax_at_checkout, prices_include_tax, portal, on_payment_failure, terms_url,
		support_email, attribution_days, updated_at, updated_by) VALUES ($1, 'usd', true, false, $2, 'keep', 'https://acme.io/terms-of-sale', 'hello@acme.io', 7, $3, $4)`,
		e.ws.ID, mustJSON(map[string]bool{"updateCard": true, "invoices": true, "billingEmail": true, "switchPeriod": true, "cancel": true}),
		e.s.earnDay(2026, 9, 17, 14, 18), e.linh)
	return err
}

func (e *earnSeed) connections(ctx context.Context) error {
	s := e.s
	perms := []map[string]any{}
	for _, p := range [][2]string{{"customers", "write"}, {"checkout_sessions", "write"}, {"customer_portal", "write"}, {"products", "write"}, {"prices", "write"},
		{"coupons", "write"}, {"promotion_codes", "write"}, {"subscriptions", "write"}, {"invoices", "read"}, {"charges", "read"}, {"webhook_endpoints", "write"},
		{"events", "read"}, {"tax_settings", "read"}} {
		perms = append(perms, map[string]any{"resource": p[0], "access": p[1], "ok": true})
	}
	events := []string{"checkout.session.completed", "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted",
		"customer.subscription.trial_will_end", "invoice.paid", "invoice.payment_failed", "invoice.upcoming", "charge.refunded", "charge.dispute.created",
		"customer.updated", "customer.deleted"}
	connected := s.earnDay(2026, 9, 17, 14, 10)
	// The hourly gap check: the last six runs found nothing missed.
	var gaps []map[string]any
	for i := 0; i < 6; i++ {
		gaps = append(gaps, map[string]any{"at": s.earnDay(2026, 10, 15, 9-i, 0).UTC().Format(time.RFC3339), "applied": 0, "types": map[string]int{}})
	}
	reach := mustJSON(map[string]any{"ok": true, "status": 200, "checkedAt": s.earnDay(2026, 9, 17, 14, 12).UTC().Format(time.RFC3339)})
	for _, c := range []struct {
		mode, key, hint, token, webhook, secret, portal string
		first, last                                     time.Time
		firstType, lastType                             string
	}{
		{"live", EarnLiveKey, "rk_live_…9fQ2", "7Gk2pQx9LmT4vB8sNw3eRfJh", "we_1Q8mAcmeLiveHk2p", "whsec_demoAcmeLiveSigningSecret0001", "bpc_1Q8mAcmeLivePortal",
			s.earnDay(2026, 9, 17, 14, 21), s.earnDay(2026, 10, 15, 9, 56), "invoice.paid", "invoice.paid"},
		{"test", EarnTestKey, "rk_test_…3mZ8", "Tz8mQ1vK5pW2xL9cRb6nHdQs", "we_1Q8mAcmeTestHk7v", "whsec_demoAcmeTestSigningSecret0001", "bpc_1Q8mAcmeTestPortal",
			s.earnDay(2026, 9, 17, 14, 20), s.earnDay(2026, 9, 18, 9, 41), "checkout.session.completed", "invoice.paid"},
	} {
		id := ID()
		if c.mode == "live" {
			e.live = id
		} else {
			e.test = id
		}
		key, err := s.Box.SealString(c.key, "payment_connection:"+id.String()+":key")
		if err != nil {
			return err
		}
		secret, err := s.Box.SealString(c.secret, "payment_connection:"+id.String()+":webhook_secret")
		if err != nil {
			return err
		}
		gapAt := s.earnDay(2026, 10, 15, 9, 0)
		at := connected
		if c.mode == "test" {
			at = connected.Add(-8 * time.Minute)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO payment_connections (id, workspace_id, provider, mode, status, step, key_sealed, key_hint, account_id, account_name,
			account_country, currency, tax_enabled, permissions, checked_at, webhook_token, webhook_id, webhook_secret_sealed, webhook_events, webhook_registered_at,
			webhook_reach, first_event_at, first_event_type, last_event_at, last_event_type, gap_checked_at, gap_history, portal_configuration, demo,
			connected_by, connected_at, created_by, created_at, updated_at)
			VALUES ($1, $2, 'stripe', $3, 'connected', 'done', $4, $5, 'acct_1AcmeMediaQ2x', 'Acme Media', 'US', 'usd', true, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, true, $21, $22, $21, $23, $24)`,
			id, e.ws.ID, c.mode, key, c.hint, mustJSON(perms), s.earnDay(2026, 10, 15, 9, 50), c.token, c.webhook, secret, events, at.Add(2*time.Minute),
			reach, c.first, c.firstType, c.last, c.lastType, gapAt, mustJSON(gaps), c.portal, e.linh, at, at.Add(-6*time.Minute), s.earnDay(2026, 10, 15, 9, 56)); err != nil {
			return fmt.Errorf("payment connection: %w", err)
		}
	}
	return nil
}

// The tiers: Weekly Digest Insider ($8/month, $80/year, 14-day trial,
// "Most popular") and Founding member ($200/year), both on Weekly Digest,
// with the prices Substack created in Acme Media.
func (e *earnSeed) tiers(ctx context.Context) error {
	s := e.s
	created := s.earnDay(2026, 9, 18, 10, 3)
	steps := mustJSON([]map[string]any{{"step": "product", "status": "done"}, {"step": "prices", "status": "done"}})
	e.insider, e.founding = ID(), ID()
	for _, t := range []struct {
		id                    uuid.UUID
		name, slug, desc      string
		benefits              []string
		trial, rank           int
		productLive, prodTest string
	}{
		{e.insider, "Weekly Digest Insider", "insider", "A members-only deep dive every Tuesday, on top of the free Weekly Digest.",
			[]string{"A members-only deep dive every Tuesday", "The full archive of deep dives", "Quarterly roadmap calls with the Acme team", "Reply to any issue and an editor answers"},
			14, 1, "prod_QmA8x2Lw", "prod_TestQmA8x2Lw"},
		{e.founding, "Founding member", "founding-member", "For readers who want to back Acme's writing, with Insider included.",
			[]string{"Everything in Insider", "Your name in the annual thank-you issue", "Early access to Acme betas"},
			0, 2, "prod_QmA9fM3k", "prod_TestQmA9fM3k"},
	} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_tiers (id, workspace_id, name, slug, description, benefits, trial_days, list_ids, access_rank, visibility,
			stripe_product_live, stripe_product_test, sync_status, sync_steps, synced_at, origin, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'public', $10, $11, 'synced', $12, $13, 'import', $14, $13, $15)`,
			t.id, e.ws.ID, t.name, t.slug, t.desc, t.benefits, t.trial, []uuid.UUID{e.digest}, t.rank, t.productLive, t.prodTest, steps, created, e.linh,
			s.earnDay(2026, 10, 1, 9, 12)); err != nil {
			return fmt.Errorf("tier: %w", err)
		}
	}
	e.pMonth, e.pYear, e.pFnd = ID(), ID(), ID()
	for _, p := range []struct {
		id         uuid.UUID
		tier       uuid.UUID
		period     string
		amount     int64
		live, test string
	}{
		{e.pMonth, e.insider, "month", 800, "price_1Q8mL2aB", "price_test_1Q8mT2aB"},
		{e.pYear, e.insider, "year", 8000, "price_1Q8mL5cD", "price_test_1Q8mT5cD"},
		{e.pFnd, e.founding, "year", 20000, "price_1Q8mM1eF", "price_test_1Q8mU1eF"},
	} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_prices (id, workspace_id, tier_id, period, amount_cents, currency, status, stripe_live_id, stripe_test_id, origin, created_at)
			VALUES ($1, $2, $3, $4, $5, 'usd', 'current', $6, $7, 'mapped', $8)`, p.id, e.ws.ID, p.tier, p.period, p.amount, p.live, p.test, created); err != nil {
			return fmt.Errorf("price: %w", err)
		}
	}
	page := earn.UpgradePage{
		Headline: "Get more from the Weekly Digest",
		Intro:    "The free Weekly Digest stays free. Insider adds a deep dive every Tuesday, and the people behind it.",
		Order:    []string{e.insider.String(), e.founding.String()}, Off: []string{}, DefaultPeriod: "year",
		Highlight: e.insider.String(), HighlightLabel: "Most popular",
		FreeList: e.digest.String(), FreeLine: "the Weekly Digest, every Tuesday",
		FreeText: "The weekly issue, with what shipped and what's next, and the public archive.",
		FAQ: []earn.UpgradeFAQ{
			{Q: "When am I charged?", A: "With the 14-day trial, on the day it ends, and we email you 3 days before. Without a trial, when you subscribe. Yearly plans renew once a year, with a reminder 7 days before."},
			{Q: "Can I switch between monthly and yearly?", A: "Yes, any time, from “Manage your subscription” in your email preferences. Stripe credits what you've already paid."},
			{Q: "Do I keep the free issues?", A: "Yes. Insider comes on top of the free Weekly Digest. If you cancel, the free issue keeps arriving every Tuesday."},
		},
		Welcome: "Welcome to Insider, {{ first_name }}. Your first deep dive arrives on Tuesday.",
		Buttons: map[string]string{e.founding.String(): "Become a founding member"},
	}
	_, err := s.Tx.Exec(ctx, `UPDATE earn_settings SET upgrade_page = $2, upgrade_published_at = $3, upgrade_published_by = $4 WHERE workspace_id = $1`,
		e.ws.ID, mustJSON(page), s.earnDay(2026, 9, 18, 11, 30), e.linh)
	return err
}

// memberEmails: the eight templates in the Essential "Memberships"
// category, from Acme Newsletter <digest@news.acme.io> with replies to
// hello@acme.io. member-welcome is at v2; gift-ending has never been
// published.
func (e *earnSeed) memberEmails(ctx context.Context) error {
	if SkipTransactional {
		return nil
	}
	s := e.s
	// They go out from the newsletter's own sender: another sender would
	// change the Senders boards, which phase 6 leaves as they are.
	e.sender = s.Senders["digest@news.acme.io"]
	if e.sender == uuid.Nil {
		return nil
	}
	created := s.earnDay(2026, 9, 17, 14, 30)
	e.category = ID()
	if err := e.q.TxnCategoryInsert(ctx, store.TxnCategoryInsertParams{ID: e.category, WorkspaceID: e.ws.ID, Name: "Memberships", Kind: transactional.KindEssential,
		Description: "Paid members' emails: welcome, trials, payments, renewals and gifts", CreatedBy: e.linh, CreatedAt: created}); err != nil {
		return fmt.Errorf("memberships category: %w", err)
	}
	sender := e.sender
	for i, st := range earn.Starters("Acme Newsletter", templates.WorkspaceAddress(&e.ws)) {
		vars := make([]transactional.TransactionalVariable, 0, len(st.Variables))
		for _, v := range st.Variables {
			vars = append(vars, transactional.TransactionalVariable{Name: v.Name, Type: v.Type, Required: v.Required, Sample: v.Sample, Default: v.Default, Sensitive: v.Sensitive})
		}
		c := transactional.DemoTemplate{Subject: st.Subject, Preheader: st.Preheader, MJML: st.MJML, Variables: vars}
		if err := c.Compile(); err != nil {
			return fmt.Errorf("%s: %w", st.Key, err)
		}
		testData := []transactional.TransactionalTestData{{Name: "Sample paid member", Data: st.Sample}}
		id := ID()
		e.templates[st.Email] = id
		at := created.Add(time.Duration(i) * time.Second)
		cat := e.category
		if err := e.q.TxnTemplateInsert(ctx, store.TxnTemplateInsertParams{ID: id, WorkspaceID: e.ws.ID, Key: st.Key, Name: st.Name, Description: st.Description, CategoryID: &cat,
			Subject: c.Subject, Preheader: c.Preheader, EditorMode: "visual", Mjml: c.MJML, Html: c.HTML, PlainText: "", Variables: mustJSON(c.Variables),
			TestData: mustJSON(testData), SenderID: &sender, ReplyTo: "hello@acme.io", TrackOpens: false, TrackClicks: false,
			DraftHash: transactional.DemoHash(c, &sender, "hello@acme.io", false, false), Now: at, CreatedBy: e.linh, Origin: json.RawMessage(`{"source":"earn"}`)}); err != nil {
			return fmt.Errorf("%s: %w", st.Key, err)
		}
		versions := 1
		switch st.Email {
		case earn.EmailGiftEnding:
			versions = 0
		case earn.EmailWelcome:
			versions = 2
		}
		for v := 1; v <= versions; v++ {
			pub := s.earnDay(2026, 9, 18, 9, 40).Add(time.Duration(i) * time.Minute)
			by, change := e.linh, "First version"
			if v == 2 {
				pub, by, change = s.earnDay(2026, 10, 1, 16, 20), e.marcus, "Mentions the deep dive and when the next one arrives"
			}
			version := int32(v)
			if err := e.q.TxnVersionInsert(ctx, store.TxnVersionInsertParams{TemplateID: id, Version: version, Subject: c.Subject, Preheader: c.Preheader, EditorMode: "visual",
				Mjml: c.MJML, Html: c.HTML, PlainText: "", Variables: mustJSON(c.Variables), TestData: mustJSON(testData), SenderID: &sender, ReplyTo: "hello@acme.io",
				ContentHash: transactional.DemoHash(c, &sender, "hello@acme.io", false, false), SizeBytes: int32(len(c.HTML)), Change: change, PublishedBy: by, PublishedAt: pub}); err != nil {
				return fmt.Errorf("%s v%d: %w", st.Key, v, err)
			}
			if err := e.q.TxnTemplatePublish(ctx, store.TxnTemplatePublishParams{WorkspaceID: e.ws.ID, ID: id, Version: &version, Now: pub, LiveBy: by, Variables: mustJSON(c.Variables)}); err != nil {
				return err
			}
		}
		tid := id
		if err := e.q.EarnMemberEmailSet(ctx, store.EarnMemberEmailSetParams{WorkspaceID: e.ws.ID, Email: st.Email, TemplateID: &tid}); err != nil {
			return err
		}
	}
	return nil
}

// people are the paid members the boards name. The audience seeder has
// some of them; the others take the place of generated Weekly Digest
// subscribers with the same mailbox provider, so every audience total the
// other boards count stays as it is.
func (e *earnSeed) people(ctx context.Context) error {
	s := e.s
	type person struct{ key, first, last, email string }
	people := []person{
		{"sam", "Sam", "Rivera", "sam.rivera@outlook.com"},
		{"priya", "Priya", "Natarajan", "priya.n@gmail.com"},
		{"ben", "Ben", "Okoro", "ben@okoro.example"},
		{"olga", "Olga", "Bennett", "olga.bennett@gmail.com"},
		{"grace", "Grace", "Okafor", "g.okafor@outlook.com"},
		{"diego", "Diego", "Alvarez", "d.alvarez@yahoo.com"},
		{"ravi", "Ravi", "Menon", "ravi@menon.example"},
		{"lena", "Lena", "Fischer", "lena.fischer@hotmail.com"},
		{"marta", "Marta", "Kowalski", "marta.k@gmail.com"},
		{"jonas", "Jonas", "Weber", "jonas.weber@example.de"},
		{"chloe", "Chloé", "Martin", "chloe.martin@example.fr"},
		{"tomas", "Tomás", "Herrera", "tomas.herrera@example.com"},
		{"freya", "Freya", "Lindgren", "freya.l@example.se"},
		{"marco", "Marco", "Bellini", "m.bellini@example.it"},
		{"aiko", "Aiko", "Tanaka", "aiko.tanaka@example.jp"},
		{"felipe", "Felipe", "Duarte", "felipe.duarte@example.com"},
		{"leo", "Leo", "Hartmann", "leo.hartmann@example.de"},
		{"press", "", "", "press@techweekly.example"},
		{"jmorales", "Javier", "Morales", "j.morales@example.com"},
		{"malvarez", "Marisol", "Alvarez", "m.alvarez@yahoo.com"},
	}
	// The test checkouts' addresses take the place of two subscribers too
	// (test paid members are left out of every number and audience).
	tests := []person{{"t-linh", "", "", "linh+insider@acme.io"}, {"t-marcus", "", "", "marcus+founding@acme.io"}}
	keep := []string{"kenji.w@icloud.com", "omar.haddad@gmail.com", "hannah@bergconsulting.de", "t.nowak@gmail.com", "aisha@rahmanlabs.io"}
	for _, p := range append(append([]person{}, people...), tests...) {
		keep = append(keep, p.email)
	}
	// Generated Weekly Digest subscribers, oldest first, by mailbox provider.
	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.email, c.first_name, c.last_name, c.mailbox_provider FROM contacts c
		JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $2 AND m.status = 'subscribed'
		WHERE c.workspace_id = $1 AND c.status = 'active' AND c.erased_at IS NULL AND c.first_name <> '' AND NOT (c.email = ANY($3))
		ORDER BY c.created_at, c.id LIMIT 3000`, e.ws.ID, e.digest, keep)
	if err != nil {
		return err
	}
	byMailbox := map[string][]earnContact{}
	var all []earnContact
	for rows.Next() {
		var c earnContact
		var mailbox string
		if err := rows.Scan(&c.id, &c.email, &c.first, &c.last, &mailbox); err != nil {
			rows.Close()
			return err
		}
		byMailbox[mailbox] = append(byMailbox[mailbox], c)
		all = append(all, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	used := map[uuid.UUID]bool{}
	for _, p := range append(append([]person{}, people...), tests...) {
		var existing earnContact
		err := s.Tx.QueryRow(ctx, `SELECT id, email, first_name, last_name FROM contacts WHERE workspace_id = $1 AND email = $2`, e.ws.ID, p.email).
			Scan(&existing.id, &existing.email, &existing.first, &existing.last)
		if err == nil {
			e.contacts[p.key] = existing
			continue
		}
		mailbox := emailaddr.MailboxProvider(emailaddr.Domain(p.email))
		var stand *earnContact
		for i := range byMailbox[mailbox] {
			if c := byMailbox[mailbox][i]; !used[c.id] {
				stand = &byMailbox[mailbox][i]
				break
			}
		}
		if stand == nil {
			return fmt.Errorf("earn: no %s subscriber to stand for %s", mailbox, p.email)
		}
		used[stand.id] = true
		if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET email = $2, first_name = $3, last_name = $4, role_address = $5 WHERE id = $1`,
			stand.id, p.email, p.first, p.last, emailaddr.IsRole(p.email)); err != nil {
			return fmt.Errorf("contact %s: %w", p.email, err)
		}
		e.contacts[p.key] = earnContact{id: stand.id, email: p.email, first: p.first, last: p.last}
	}
	for _, c := range all {
		if !used[c.id] {
			e.pool = append(e.pool, c)
		}
	}
	if len(e.pool) < 1350 {
		return fmt.Errorf("earn: only %d Weekly Digest subscribers to draw paid members from", len(e.pool))
	}
	return nil
}

func (e *earnSeed) draw() earnContact {
	c := e.pool[e.next]
	e.next++
	return c
}

// importRun: substack-acme-insider-export.zip, imported by Linh Nguyen on 18 Sep.
func (e *earnSeed) importRun(ctx context.Context) error {
	s := e.s
	e.imp = ID()
	cols := []map[string]string{
		{"name": "email", "usedAs": "email", "example": "priya.n@gmail.com"},
		{"name": "stripe_customer_id", "usedAs": "customer", "example": "cus_PqR8…"},
		{"name": "plan", "usedAs": "plan", "example": "Insider · yearly"},
		{"name": "paid_through", "usedAs": "paid_through", "example": "2027-04-14"},
		{"name": "name", "usedAs": "name", "example": "Priya Natarajan"},
	}
	mapping := map[string]string{"email": "email", "customer": "stripe_customer_id", "plan": "plan", "paid_through": "paid_through", "name": "name"}
	dec := map[string]any{"noSubscription": "complimentary", "sendWelcome": false, "prices": map[string]any{
		"price_1Q8mL2aB": map[string]any{"tierId": e.insider.String(), "legacy": false},
		"price_1Q8mL5cD": map[string]any{"tierId": e.insider.String(), "legacy": false},
		"price_1Q8mM1eF": map[string]any{"tierId": e.founding.String(), "legacy": false}}}
	stats := map[string]any{"paid": 1121, "free": 12, "complimentary": 17, "skipped": 0, "merged": 1083, "new": 67, "pricesMapped": 3, "emailsSent": 0,
		"compFirst": "2026-10-31T23:59:59Z", "compLast": "2027-06-30T23:59:59Z"}
	_, err := s.Tx.Exec(ctx, `INSERT INTO earn_imports (id, workspace_id, mode, source, status, file_name, file_size, columns, mapping, decisions, send_welcome, total,
		done_rows, stats, created_by, created_at, matched_at, started_at, finished_at)
		VALUES ($1, $2, 'live', 'substack', 'done', 'substack-acme-insider-export.zip', 421888, $3, $4, $5, false, 1150, 1150, $6, $7, $8, $9, $10, $11)`,
		e.imp, e.ws.ID, mustJSON(cols), mustJSON(mapping), mustJSON(dec), mustJSON(stats), e.linh,
		s.earnDay(2026, 9, 18, 10, 2), s.earnDay(2026, 9, 18, 10, 3), s.earnDay(2026, 9, 18, 10, 4), s.earnDay(2026, 9, 18, 10, 6))
	return err
}

func (e *earnSeed) cus() string { return "cus_" + e.code(10) }
func (e *earnSeed) sub() string { return "sub_1Q" + e.code(8) }

const earnAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

func (e *earnSeed) code(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = earnAlphabet[e.rng.IntN(len(earnAlphabet))]
	}
	return string(b)
}

// add queues a paid subscription and its timeline.
func (e *earnSeed) add(r earnRow) uuid.UUID {
	id := ID()
	if r.mode == "" {
		r.mode = "live"
	}
	if r.updated.IsZero() {
		e.order++
		r.updated = e.s.earnDay(2026, 9, 18, 10, 6).Add(-time.Duration(e.order) * time.Second)
	}
	var discount any
	if r.discount != nil {
		discount = mustJSON(r.discount)
	}
	if r.srcDetail == nil {
		r.srcDetail = map[string]any{}
	}
	var importID *uuid.UUID
	if r.imported {
		imp := e.imp
		importID = &imp
		r.srcDetail["platform"] = "substack"
		r.srcDetail["import_id"] = e.imp.String()
	}
	cus, sub := "", ""
	if r.kind == "stripe" {
		cus, sub = e.cus(), e.sub()
		if r.key == "sam" {
			cus, sub = "cus_QmT4wE8sLd", "sub_1Q8xRb2Va7"
		}
	}
	if r.emailed == nil {
		r.emailed = []string{}
	}
	e.subs = append(e.subs, []any{id, e.ws.ID, r.c.id, r.mode, r.kind, r.tier, r.price, r.stripePr, r.status, r.period, r.amount, "usd", discount,
		r.started, r.trialEnd, r.periodEnd, r.cancelAt, r.canceled, r.canceledBy, r.ended, r.endReason, r.cancelWhy, r.paid, r.refunded, r.failedAt,
		r.failure, r.retryAt, cus, sub, "", r.srcKind, r.srcCamp, r.srcLabel, mustJSON(r.srcDetail), r.grantWhy, r.grantNote, r.grantedBy, r.grantTill,
		importID, tp(r.updated), r.emailed, r.started, r.updated})
	if r.key != "" {
		e.byKey[r.key] = id
	}
	return id
}

// event queues a timeline entry.
func (e *earnSeed) event(sub uuid.UUID, c earnContact, mode, kind string, at time.Time, amount *int64, actor *uuid.UUID, data map[string]any, msg *uuid.UUID) {
	if data == nil {
		data = map[string]any{}
	}
	if mode == "" {
		mode = "live"
	}
	cur := ""
	if amount != nil {
		cur = "usd"
	}
	e.events = append(e.events, []any{ID(), e.ws.ID, c.id, sub, mode, kind, at, amount, cur, nil, "", msg, actor, mustJSON(data)})
}

func i64(v int64) *int64 { return &v }

// months adds n months to a date (Stripe's billing anchor).
func months(t time.Time, n int) time.Time { return t.AddDate(0, n, 0) }

// nextRenewal is the first anchor after today.
func (e *earnSeed) nextRenewal(start time.Time, everyMonths int) time.Time {
	t := start
	now := e.s.Now
	for !t.After(now) {
		t = months(t, everyMonths)
	}
	return t
}

func (e *earnSeed) members(ctx context.Context) error {
	s := e.s
	imported := s.earnDay(2026, 9, 18, 10, 6)
	moved := s.earnDay(2026, 9, 18, 10, 6)
	c141 := s.Campaigns["Weekly Digest #141"]
	c140 := s.Campaigns["Weekly Digest #140"]
	c139 := s.Campaigns["Weekly Digest #139"]
	camp := func(id uuid.UUID) *uuid.UUID {
		if id == uuid.Nil {
			return nil
		}
		return &id
	}
	pm, py, pf := e.pMonth, e.pYear, e.pFnd
	importedLabel := "Imported from Substack"
	// The board's rows, newest change first.
	top := s.Now.Add(-time.Minute)
	upd := func(i int) time.Time { return top.Add(-time.Duration(i) * time.Second) }

	// Sam Rivera: Insider yearly with FALL26 from Weekly Digest #141.
	samStart := s.earnDay(2026, 10, 7, 9, 52)
	sam := e.add(earnRow{key: "sam", c: e.contacts["sam"], kind: "stripe", tier: e.insider, price: &py, stripePr: "price_1Q8mL5cD", status: "active", period: "year",
		amount: 8000, discount: map[string]any{"label": "FALL26", "coupon": "FALL26", "duration": "once", "amountCents": 6400, "percent": 20, "endsAt": samStart.AddDate(1, 0, 0).UTC().Format(time.RFC3339)},
		started: samStart, periodEnd: tp(samStart.AddDate(1, 0, 0)), paid: 6400, srcKind: "campaign", srcCamp: camp(c141), srcLabel: "Weekly Digest #141",
		srcDetail: map[string]any{"clickedAt": s.earnDay(2026, 10, 7, 9, 44).UTC().Format(time.RFC3339), "offer": "FALL26"}, emailed: []string{"welcome"}, updated: upd(0)})
	welcomeMsg := e.memberMessage(ctx, e.contacts["sam"], earn.EmailWelcome, 2, "Welcome to Insider, Sam", s.earnDay(2026, 10, 7, 9, 53))
	e.event(sam, e.contacts["sam"], "", "clicked_upgrade", s.earnDay(2026, 10, 7, 9, 44), nil, nil, map[string]any{"campaign": "Weekly Digest #141", "credited": true}, nil)
	e.event(sam, e.contacts["sam"], "", "checkout_completed", s.earnDay(2026, 10, 7, 9, 52), nil, nil, map[string]any{"tier": "Weekly Digest Insider", "period": "year", "offer": "FALL26", "lists": "already"}, nil)
	e.event(sam, e.contacts["sam"], "", "paid", s.earnDay(2026, 10, 7, 9, 52).Add(20*time.Second), i64(6400), nil, map[string]any{"tier": "Weekly Digest Insider", "period": "year", "offer": "FALL26"}, nil)
	e.event(sam, e.contacts["sam"], "", "email_sent", s.earnDay(2026, 10, 7, 9, 53), nil, nil, map[string]any{"email": "welcome", "template": "member-welcome", "version": 2, "status": "sent"}, welcomeMsg)

	// Priya Natarajan: Founding member since 14 Apr 2025, from Substack.
	priyaStart := time.Date(2025, 4, 14, 15, 20, 0, 0, s.earnLoc())
	priya := e.add(earnRow{key: "priya", c: e.contacts["priya"], kind: "stripe", tier: e.founding, price: &pf, stripePr: "price_1Q8mM1eF", status: "active", period: "year",
		amount: 20000, started: priyaStart, periodEnd: tp(e.nextRenewal(priyaStart, 12)), paid: 40000, srcKind: "import", srcLabel: importedLabel, imported: true, updated: upd(1)})
	e.event(priya, e.contacts["priya"], "", "imported", imported, nil, e.linh, map[string]any{"platform": "substack", "matchedBy": "customer"}, nil)

	// Ben Okoro: Insider monthly, "Stay for half price" since 12 Oct.
	benStart := time.Date(2026, 3, 12, 11, 0, 0, 0, s.earnLoc())
	ben := e.add(earnRow{key: "ben", c: e.contacts["ben"], kind: "stripe", tier: e.insider, price: &pm, stripePr: "price_1Q8mL2aB", status: "active", period: "month",
		amount: 800, discount: map[string]any{"label": "Stay for half price", "coupon": "STAY50", "duration": "repeating", "months": 3, "amountCents": 400, "percent": 50,
			"endsAt": s.earnDay(2027, 1, 12, 11, 0).UTC().Format(time.RFC3339)},
		started: benStart, periodEnd: tp(s.earnDay(2026, 11, 12, 11, 0)), paid: 6000, srcKind: "import", srcLabel: importedLabel, imported: true, updated: upd(2)})
	e.event(ben, e.contacts["ben"], "", "imported", imported, nil, e.linh, map[string]any{"platform": "substack", "matchedBy": "customer"}, nil)
	e.event(ben, e.contacts["ben"], "", "offer_applied", s.earnDay(2026, 10, 12, 14, 20), nil, nil, map[string]any{"offer": "Stay for half price"}, nil)
	e.event(ben, e.contacts["ben"], "", "renewed", s.earnDay(2026, 10, 12, 14, 22), i64(400), nil, map[string]any{"tier": "Weekly Digest Insider", "period": "month", "offer": "Stay for half price"}, nil)

	// Olga Bennett: canceled on the cancel page on 12 Oct; ends 2 Nov.
	olgaStart := time.Date(2026, 7, 2, 18, 30, 0, 0, s.earnLoc())
	olga := e.add(earnRow{key: "olga", c: e.contacts["olga"], kind: "stripe", tier: e.insider, price: &pm, stripePr: "price_1Q8mL2aB", status: "active", period: "month",
		amount: 800, started: olgaStart, periodEnd: tp(s.earnDay(2026, 11, 2, 18, 30)), cancelAt: tp(s.earnDay(2026, 11, 2, 18, 30)), canceled: tp(s.earnDay(2026, 10, 12, 11, 5)),
		cancelWhy: "not_reading", paid: 3200, srcKind: "import", srcLabel: importedLabel, imported: true, emailed: []string{"canceled"}, updated: upd(3)})
	e.event(olga, e.contacts["olga"], "", "imported", imported, nil, e.linh, map[string]any{"platform": "substack", "matchedBy": "customer"}, nil)
	e.event(olga, e.contacts["olga"], "", "cancel_scheduled", s.earnDay(2026, 10, 12, 11, 5), nil, nil, map[string]any{"date": s.earnDay(2026, 11, 2, 18, 30).UTC().Format(time.RFC3339), "reason": "not_reading"}, nil)

	// Grace Okafor: past due since 13 Oct; Stripe tries again on 16 Oct.
	graceStart := time.Date(2026, 2, 13, 9, 12, 0, 0, s.earnLoc())
	grace := e.add(earnRow{key: "grace", c: e.contacts["grace"], kind: "stripe", tier: e.insider, price: &pm, stripePr: "price_1Q8mL2aB", status: "past_due", period: "month",
		amount: 800, started: graceStart, periodEnd: tp(s.earnDay(2026, 11, 13, 9, 12)), paid: 6400, failedAt: tp(s.earnDay(2026, 10, 13, 9, 12)),
		failure: "Your card has insufficient funds.", retryAt: tp(s.earnDay(2026, 10, 16, 9, 12)), srcKind: "import", srcLabel: importedLabel, imported: true,
		emailed: []string{}, updated: upd(4)})
	// Her Payment failed email isn't in the transactional log: the
	// Transactional boards' last seven days hold Acme Cloud's mail only.
	e.event(grace, e.contacts["grace"], "", "imported", imported, nil, e.linh, map[string]any{"platform": "substack", "matchedBy": "customer"}, nil)
	e.event(grace, e.contacts["grace"], "", "payment_failed", s.earnDay(2026, 10, 13, 9, 12), i64(800), nil, map[string]any{"reason": "Your card has insufficient funds.",
		"retry": s.earnDay(2026, 10, 16, 9, 12).UTC().Format(time.RFC3339)}, nil)
	e.event(grace, e.contacts["grace"], "", "email_sent", s.earnDay(2026, 10, 13, 9, 13), nil, nil, map[string]any{"email": "payment_failed", "template": "member-payment-failed", "version": 1, "status": "sent"}, nil)

	// Diego Alvarez: an Insider monthly trial from Weekly Digest #141.
	diegoStart := s.earnDay(2026, 10, 7, 10, 2)
	diego := e.add(earnRow{key: "diego", c: e.contacts["diego"], kind: "stripe", tier: e.insider, price: &pm, stripePr: "price_1Q8mL2aB", status: "trialing", period: "month",
		amount: 800, started: diegoStart, trialEnd: tp(s.earnDay(2026, 10, 21, 10, 2)), periodEnd: tp(s.earnDay(2026, 10, 21, 10, 2)), srcKind: "campaign", srcCamp: camp(c141),
		srcLabel: "Weekly Digest #141", emailed: []string{"welcome"}, updated: upd(5)})
	e.event(diego, e.contacts["diego"], "", "trial_started", diegoStart, nil, nil, map[string]any{"trialEnd": s.earnDay(2026, 10, 21, 10, 2).UTC().Format(time.RFC3339)}, nil)

	// Ravi Menon: complimentary Insider from Marcus Chen, a contributor.
	ravi := e.add(earnRow{key: "ravi", c: e.contacts["ravi"], kind: "complimentary", tier: e.insider, status: "complimentary", started: s.earnDay(2026, 10, 3, 15, 40),
		srcKind: "grant", srcLabel: "Marcus Chen", srcDetail: map[string]any{"reason": "contributor"}, grantWhy: "contributor", grantNote: "Writes the monthly Postgres column",
		grantedBy: e.marcus, updated: upd(6)})
	e.event(ravi, e.contacts["ravi"], "", "granted", s.earnDay(2026, 10, 3, 15, 40), nil, e.marcus, map[string]any{"tier": "Weekly Digest Insider", "reason": "contributor", "until": ""}, nil)

	// Lena Fischer: a year of Insider from Priya Natarajan.
	lena := e.add(earnRow{key: "lena", c: e.contacts["lena"], kind: "gift", tier: e.insider, status: "gift", started: s.earnDay(2026, 10, 11, 18, 5),
		periodEnd: tp(s.earnDay(2027, 10, 11, 18, 5)), srcKind: "gift", srcLabel: "Gift from Priya Natarajan", srcDetail: map[string]any{"from": "Priya Natarajan", "months": 12}, updated: upd(7)})
	e.event(lena, e.contacts["lena"], "", "gift_redeemed", s.earnDay(2026, 10, 11, 18, 5), nil, nil, map[string]any{"from": "Priya Natarajan", "months": 12}, nil)

	// Marta Kowalski: renewed on 10 Oct, refunded and canceled on 13 Oct.
	martaStart := time.Date(2025, 10, 10, 8, 0, 0, 0, s.earnLoc())
	marta := e.add(earnRow{key: "marta", c: e.contacts["marta"], kind: "stripe", tier: e.insider, price: &py, stripePr: "price_1Q8mL5cD", status: "canceled", period: "year",
		amount: 8000, started: martaStart, periodEnd: tp(s.earnDay(2027, 10, 10, 8, 0)), canceled: tp(s.earnDay(2026, 10, 13, 16, 41)), canceledBy: e.marcus,
		ended: tp(s.earnDay(2026, 10, 13, 16, 41)), endReason: "canceled", paid: 16000, refunded: 8000, srcKind: "import", srcLabel: importedLabel, imported: true,
		emailed: []string{"canceled"}})
	e.event(marta, e.contacts["marta"], "", "imported", imported, nil, e.linh, map[string]any{"platform": "substack", "matchedBy": "customer"}, nil)
	e.event(marta, e.contacts["marta"], "", "renewed", s.earnDay(2026, 10, 10, 8, 0), i64(8000), nil, map[string]any{"tier": "Weekly Digest Insider", "period": "year"}, nil)
	e.event(marta, e.contacts["marta"], "", "refunded", s.earnDay(2026, 10, 13, 16, 32), i64(-8000), nil, map[string]any{"amount": 8000}, nil)
	e.event(marta, e.contacts["marta"], "", "access_ended", s.earnDay(2026, 10, 13, 16, 41), nil, e.marcus, map[string]any{"reason": "canceled", "by": "Marcus Chen"}, nil)

	// Jonas Weber and Chloé Martin: their access ended in the last 30 days.
	jonas := e.add(earnRow{key: "jonas", c: e.contacts["jonas"], kind: "stripe", tier: e.insider, price: &pm, stripePr: "price_1Q8mL2aB", status: "canceled", period: "month",
		amount: 800, started: time.Date(2026, 1, 3, 12, 0, 0, 0, s.earnLoc()), canceled: tp(s.earnDay(2026, 9, 24, 20, 10)), ended: tp(s.earnDay(2026, 10, 3, 12, 0)),
		endReason: "canceled", cancelWhy: "not_reading", paid: 7200, srcKind: "import", srcLabel: importedLabel, imported: true, emailed: []string{"canceled"}})
	e.event(jonas, e.contacts["jonas"], "", "access_ended", s.earnDay(2026, 10, 3, 12, 0), nil, nil, map[string]any{"reason": "canceled"}, nil)
	chloe := e.add(earnRow{key: "chloe", c: e.contacts["chloe"], kind: "stripe", tier: e.insider, price: &py, stripePr: "price_1Q8mL5cD", status: "canceled", period: "year",
		amount: 8000, started: time.Date(2025, 9, 26, 9, 0, 0, 0, s.earnLoc()), canceled: tp(s.earnDay(2026, 9, 20, 9, 30)), ended: tp(s.earnDay(2026, 9, 26, 9, 0)),
		endReason: "canceled", cancelWhy: "", paid: 8000, srcKind: "import", srcLabel: importedLabel, imported: true})
	e.event(chloe, e.contacts["chloe"], "", "access_ended", s.earnDay(2026, 9, 26, 9, 0), nil, nil, map[string]any{"reason": "canceled", "didntRenew": true}, nil)

	// The other past due members: Tomás Herrera, Freya Lindgren, Marco Bellini.
	for i, p := range []struct {
		key           string
		tier          uuid.UUID
		price         uuid.UUID
		stripe, per   string
		amount, paid  int64
		start         time.Time
		failed, retry time.Time
	}{
		{"tomas", e.insider, pm, "price_1Q8mL2aB", "month", 800, 4000, time.Date(2026, 5, 10, 10, 0, 0, 0, s.earnLoc()), s.earnDay(2026, 10, 13, 8, 40), s.earnDay(2026, 10, 17, 8, 40)},
		{"freya", e.founding, pf, "price_1Q8mM1eF", "year", 20000, 20000, time.Date(2025, 10, 9, 14, 0, 0, 0, s.earnLoc()), s.earnDay(2026, 10, 13, 7, 55), s.earnDay(2026, 10, 18, 7, 55)},
		{"marco", e.insider, py, "price_1Q8mL5cD", "year", 8000, 16000, time.Date(2024, 10, 12, 16, 0, 0, 0, s.earnLoc()), s.earnDay(2026, 10, 12, 18, 20), s.earnDay(2026, 10, 19, 18, 20)},
	} {
		price := p.price
		id := e.add(earnRow{key: p.key, c: e.contacts[p.key], kind: "stripe", tier: p.tier, price: &price, stripePr: p.stripe, status: "past_due", period: p.per,
			amount: p.amount, started: p.start, periodEnd: tp(months(p.failed, 1)), paid: p.paid, failedAt: tp(p.failed), failure: "Your card was declined.",
			retryAt: tp(p.retry), srcKind: "import", srcLabel: importedLabel, imported: true, emailed: []string{}, updated: upd(20 + i)})
		e.event(id, e.contacts[p.key], "", "payment_failed", p.failed, i64(p.amount), nil, map[string]any{"reason": "Your card was declined.", "retry": p.retry.UTC().Format(time.RFC3339)}, nil)
	}

	// Aiko Tanaka renewed today; Leo Hartmann renews on 22 Oct; Felipe
	// Duarte's trial ends on 18 Oct (the event log's rows).
	aiko := e.add(earnRow{key: "aiko", c: e.contacts["aiko"], kind: "stripe", tier: e.insider, price: &py, stripePr: "price_1Q8mL5cD", status: "active", period: "year",
		amount: 8000, started: time.Date(2024, 10, 15, 9, 56, 0, 0, s.earnLoc()), periodEnd: tp(s.earnDay(2027, 10, 15, 9, 56)), paid: 16000, srcKind: "import", srcLabel: importedLabel, imported: true})
	e.event(aiko, e.contacts["aiko"], "", "renewed", s.earnDay(2026, 10, 15, 9, 56), i64(8000), nil, map[string]any{"tier": "Weekly Digest Insider", "period": "year",
		"until": s.earnDay(2027, 10, 15, 9, 56).UTC().Format(time.RFC3339)}, nil)
	e.add(earnRow{key: "leo", c: e.contacts["leo"], kind: "stripe", tier: e.insider, price: &py, stripePr: "price_1Q8mL5cD", status: "active", period: "year",
		amount: 8000, started: time.Date(2025, 10, 22, 8, 12, 0, 0, s.earnLoc()), periodEnd: tp(s.earnDay(2026, 10, 22, 8, 12)), paid: 8000, srcKind: "import", srcLabel: importedLabel, imported: true,
		emailed: []string{"renewal_reminder"}})
	felipe := e.add(earnRow{key: "felipe", c: e.contacts["felipe"], kind: "stripe", tier: e.insider, price: &pm, stripePr: "price_1Q8mL2aB", status: "trialing", period: "month",
		amount: 800, started: s.earnDay(2026, 10, 4, 9, 31), trialEnd: tp(s.earnDay(2026, 10, 18, 9, 31)), periodEnd: tp(s.earnDay(2026, 10, 18, 9, 31)), srcKind: "page",
		srcLabel: "Upgrade page", emailed: []string{"welcome", "trial_ending"}})
	e.event(felipe, e.contacts["felipe"], "", "trial_started", s.earnDay(2026, 10, 4, 9, 31), nil, nil, nil, nil)

	// Complimentary access from the import (until the paid-through date)
	// and by hand.
	press := e.add(earnRow{key: "press", c: e.contacts["press"], kind: "complimentary", tier: e.insider, status: "complimentary", started: moved, srcKind: "import",
		srcLabel: importedLabel, imported: true, grantWhy: "other", grantNote: "Paid-through date from the Substack export, with no subscription in Stripe",
		grantedBy: e.linh, grantTill: tp(time.Date(2027, 6, 30, 23, 59, 59, 0, time.UTC))})
	e.event(press, e.contacts["press"], "", "imported", imported, nil, e.linh, map[string]any{"platform": "substack", "complimentary": true, "until": "2027-06-30"}, nil)
	e.add(earnRow{key: "jmorales", c: e.contacts["jmorales"], kind: "complimentary", tier: e.insider, status: "complimentary", started: moved, srcKind: "import",
		srcLabel: importedLabel, imported: true, grantWhy: "other", grantNote: "Paid-through date from the Substack export, with no subscription in Stripe",
		grantedBy: e.linh, grantTill: tp(time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC))})
	compDates := []time.Time{}
	for i := 0; i < 15; i++ {
		d := time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC).AddDate(0, i%7, (i*3)%27)
		compDates = append(compDates, d)
	}
	for _, d := range compDates {
		c := e.draw()
		e.add(earnRow{c: c, kind: "complimentary", tier: e.insider, status: "complimentary", started: moved, srcKind: "import", srcLabel: importedLabel, imported: true,
			grantWhy: "other", grantNote: "Paid-through date from the Substack export, with no subscription in Stripe", grantedBy: e.linh, grantTill: tp(d)})
	}
	for i, g := range []struct {
		why, note string
		by        *uuid.UUID
		until     *time.Time
	}{
		{"team", "Acme's editorial team", e.marcus, nil},
		{"team", "Acme's editorial team", e.linh, nil},
		{"press", "Covers developer tools", e.marcus, tp(s.earnDay(2027, 3, 31, 23, 59))},
		{"partner", "Ops Weekly cross-promotion", e.marcus, tp(s.earnDay(2027, 1, 31, 23, 59))},
		{"giveaway", "Postgres meetup raffle", e.linh, tp(s.earnDay(2026, 12, 31, 23, 59))},
	} {
		c := e.draw()
		at := s.earnDay(2026, 9, 25+i, 11, 15)
		name := "Marcus Chen"
		if g.by == e.linh {
			name = "Linh Nguyen"
		}
		id := e.add(earnRow{c: c, kind: "complimentary", tier: e.insider, status: "complimentary", started: at, srcKind: "grant", srcLabel: name,
			srcDetail: map[string]any{"reason": g.why}, grantWhy: g.why, grantNote: g.note, grantedBy: g.by, grantTill: g.until})
		e.event(id, c, "", "granted", at, nil, g.by, map[string]any{"tier": "Weekly Digest Insider", "reason": g.why, "note": g.note}, nil)
	}

	// Gifts: m.alvarez@yahoo.com (3 months from Diego Alvarez) and six more.
	e.add(earnRow{key: "malvarez", c: e.contacts["malvarez"], kind: "gift", tier: e.insider, status: "gift", started: s.earnDay(2026, 10, 10, 12, 0),
		periodEnd: tp(s.earnDay(2027, 1, 10, 12, 0)), srcKind: "gift", srcLabel: "Gift from Diego Alvarez", srcDetail: map[string]any{"from": "Diego Alvarez", "months": 3}})
	givers := []string{"Hannah Berg", "Omar Haddad", "Kenji Watanabe", "Priya Natarajan", "Sam Rivera", "Tomasz Nowak"}
	for i, giver := range givers {
		c := e.draw()
		m := 12
		if i == 5 {
			m = 3
		}
		at := s.earnDay(2026, 9, 22+i*3, 19, 0)
		e.add(earnRow{c: c, kind: "gift", tier: e.insider, status: "gift", started: at, periodEnd: tp(months(at, m)), srcKind: "gift", srcLabel: "Gift from " + giver,
			srcDetail: map[string]any{"from": giver, "months": m}})
	}

	// Generated paying members. Insider monthly: 431 paying in all (12 on
	// "Stay for half price", 12 past due, 21 cancelling); Insider yearly: 693
	// (45 with FALL26, 1 past due, 5 cancelling); Founding member: 60 (1 past due).
	type plan struct {
		tier   uuid.UUID
		price  uuid.UUID
		stripe string
		period string
		amount int64
	}
	month := plan{e.insider, pm, "price_1Q8mL2aB", "month", 800}
	year := plan{e.insider, py, "price_1Q8mL5cD", "year", 8000}
	fnd := plan{e.founding, pf, "price_1Q8mM1eF", "year", 20000}
	gen := func(p plan, n int, opts func(i int, r *earnRow)) {
		for i := 0; i < n; i++ {
			c := e.draw()
			price := p.price
			// Imported members joined on Substack between three years
			// and a month before the move.
			start := moved.AddDate(0, 0, -(30 + e.rng.IntN(1000)))
			every := 1
			if p.period == "year" {
				every = 12
			}
			renew := e.nextRenewal(start, every)
			cycles := int64(0)
			for t := start; !t.After(s.Now); t = months(t, every) {
				cycles++
			}
			r := earnRow{c: c, kind: "stripe", tier: p.tier, price: &price, stripePr: p.stripe, status: "active", period: p.period, amount: p.amount,
				started: start, periodEnd: tp(renew), paid: cycles * p.amount, srcKind: "import", srcLabel: importedLabel, imported: true}
			if opts != nil {
				opts(i, &r)
			}
			id := e.add(r)
			if r.imported {
				e.event(id, c, "", "imported", imported, nil, e.linh, map[string]any{"platform": "substack", "matchedBy": "customer"}, nil)
			}
			if r.status == "active" && r.ended == nil && r.discount == nil {
				if p.period == "month" {
					e.monthly = append(e.monthly, r)
				} else {
					e.yearly = append(e.yearly, r)
				}
			}
		}
	}
	reasons := []string{"not_reading", "too_expensive", "not_for_me", "switching", "other", ""}
	newSources := []struct {
		kind, label string
		camp        uuid.UUID
	}{
		{"campaign", "Weekly Digest #139", c139}, {"campaign", "Weekly Digest #140", c140}, {"automation", "Welcome to news.acme.io", uuid.Nil},
		{"automation", "Welcome series", uuid.Nil}, {"page", "Upgrade page", uuid.Nil}, {"offer", "FALL26 link", uuid.Nil},
	}
	newStart := func(i int) time.Time {
		return s.earnDay(2026, 9, 18, 12, 0).Add(time.Duration(i*6+e.rng.IntN(5)) * time.Hour)
	}
	// Insider monthly: 431 − Ben, Olga, Grace, Tomás = 427 generated.
	gen(month, 427, func(i int, r *earnRow) {
		switch {
		case i < 11: // Stay for half price, accepted in the last 30 days.
			r.discount = map[string]any{"label": "Stay for half price", "coupon": "STAY50", "duration": "repeating", "months": 3, "amountCents": 400, "percent": 50,
				"endsAt": months(*r.periodEnd, 2).UTC().Format(time.RFC3339)}
		case i < 21: // past due: Stripe is retrying.
			f := s.earnDay(2026, 10, 12-(i-11)%9, 6+(i%5), 10*(i%6))
			r.status, r.failedAt, r.retryAt, r.failure = "past_due", tp(f), tp(f.AddDate(0, 0, 3+(i%4))), "Your card was declined."
		case i < 41: // cancelling at the end of the period.
			r.cancelAt, r.canceled, r.cancelWhy = r.periodEnd, tp(s.earnDay(2026, 10, 14-(i-21)%12, 10+(i%8), 5*(i%11))), reasons[i%len(reasons)]
		case i < 80: // 39 converted after a trial in the last 30 days.
			src := newSources[i%5]
			start := newStart(i - 41)
			r.started, r.imported, r.srcKind, r.srcLabel, r.srcCamp = start, false, src.kind, src.label, camp(src.camp)
			r.trialEnd = tp(start.AddDate(0, 0, 14))
			r.periodEnd, r.paid = tp(months(*r.trialEnd, 1)), 800
			if r.periodEnd.Before(s.Now) {
				r.periodEnd = tp(months(*r.periodEnd, 1))
				r.paid = 1600
			}
		}
	})
	// Insider yearly: 693 − Sam, Marco, Aiko, Leo = 689 generated.
	gen(year, 689, func(i int, r *earnRow) {
		switch {
		case i < 44: // FALL26: 6 from #141, the rest from the offer link and emails.
			src := newSources[5]
			if i < 6 {
				src = struct {
					kind, label string
					camp        uuid.UUID
				}{"campaign", "Weekly Digest #141", c141}
			} else if i%3 == 0 {
				src = newSources[i%4]
			}
			start := s.earnDay(2026, 10, 1, 9, 0).Add(time.Duration(i*5+e.rng.IntN(4)) * time.Hour)
			if i < 6 {
				start = s.earnDay(2026, 10, 7, 9, 10+i*7)
			}
			r.started, r.imported, r.srcKind, r.srcLabel, r.srcCamp = start, false, src.kind, src.label, camp(src.camp)
			r.discount = map[string]any{"label": "FALL26", "coupon": "FALL26", "duration": "once", "amountCents": 6400, "percent": 20, "endsAt": start.AddDate(1, 0, 0).UTC().Format(time.RFC3339)}
			r.periodEnd, r.paid = tp(start.AddDate(1, 0, 0)), 6400
			r.srcDetail = map[string]any{"offer": "FALL26"}
		case i < 50: // 6 converted after a trial.
			start := newStart(i)
			src := newSources[i%5]
			r.started, r.imported, r.srcKind, r.srcLabel, r.srcCamp = start, false, src.kind, src.label, camp(src.camp)
			r.trialEnd = tp(start.AddDate(0, 0, 14))
			r.periodEnd, r.paid = tp(r.trialEnd.AddDate(1, 0, 0)), 8000
		case i < 55: // cancelling at the end of the year.
			r.cancelAt, r.canceled, r.cancelWhy = r.periodEnd, tp(s.earnDay(2026, 10, 10-(i-50), 15, 30)), reasons[i%len(reasons)]
		}
	})
	// Founding members: 60 − Priya, Freya = 58 generated; 6 new (3 from #141).
	gen(fnd, 58, func(i int, r *earnRow) {
		if i < 6 {
			src := struct {
				kind, label string
				camp        uuid.UUID
			}{"campaign", "Weekly Digest #141", c141}
			start := s.earnDay(2026, 10, 7, 11, 5+i*9)
			if i >= 3 {
				src.kind, src.label, src.camp = "page", "Upgrade page", uuid.Nil
				start = newStart(i * 9)
			}
			r.started, r.imported, r.srcKind, r.srcLabel, r.srcCamp = start, false, src.kind, src.label, camp(src.camp)
			r.periodEnd, r.paid = tp(start.AddDate(1, 0, 0)), 20000
		}
	})

	// Trials: 37 in all. Diego and Felipe, 15 more from #141 ending 21 Oct
	// (12 monthly, 3 yearly), 12 more ending within the week, 8 later.
	for i := 0; i < 35; i++ {
		c := e.draw()
		p := month
		if i >= 12 && i < 15 {
			p = year
		}
		price := p.price
		var start time.Time
		src := struct {
			kind, label string
			camp        uuid.UUID
		}{"campaign", "Weekly Digest #141", c141}
		switch {
		case i < 15:
			start = s.earnDay(2026, 10, 7, 9, 20+i*3)
		case i < 27:
			start = s.earnDay(2026, 10, 2+(i-15)%5, 8+(i%9), 4*(i%13))
			src = newSources[(i+2)%5]
		default:
			start = s.earnDay(2026, 10, 9+(i-27)%5, 10+(i%7), 3*(i%17))
			src = newSources[(i+1)%5]
		}
		end := start.AddDate(0, 0, 14)
		id := e.add(earnRow{c: c, kind: "stripe", tier: e.insider, price: &price, stripePr: p.stripe, status: "trialing", period: p.period, amount: p.amount,
			started: start, trialEnd: tp(end), periodEnd: tp(end), srcKind: src.kind, srcLabel: src.label, srcCamp: camp(src.camp), emailed: []string{"welcome"}})
		e.event(id, c, "", "trial_started", start, nil, nil, map[string]any{"trialEnd": end.UTC().Format(time.RFC3339)}, nil)
	}

	// Ended in the last 30 days: 31 paying members churned (25 Insider
	// monthly, 6 Insider yearly, Marta, Jonas and Chloé among them), and
	// 30 trials that ended without a payment (not paid members: Ended
	// leaves them out).
	for i := 0; i < 24; i++ {
		c := e.draw()
		price := pm
		start := moved.AddDate(0, 0, -(40 + e.rng.IntN(500)))
		end := s.earnDay(2026, 10, 12-(i%26), 9+(i%10), 7*(i%8))
		if end.After(s.Now) {
			end = s.Now.Add(-time.Hour)
		}
		id := e.add(earnRow{c: c, kind: "stripe", tier: e.insider, price: &price, stripePr: "price_1Q8mL2aB", status: "canceled", period: "month", amount: 800,
			started: start, canceled: tp(end.AddDate(0, 0, -9)), ended: tp(end), endReason: "canceled", cancelWhy: reasons[i%len(reasons)],
			paid: 800 * int64(3+e.rng.IntN(20)), srcKind: "import", srcLabel: importedLabel, imported: true, emailed: []string{"canceled"}})
		e.event(id, c, "", "access_ended", end, nil, nil, map[string]any{"reason": "canceled"}, nil)
	}
	for i := 0; i < 4; i++ {
		c := e.draw()
		price := py
		end := s.earnDay(2026, 10, 12-i*5, 10, 0)
		id := e.add(earnRow{c: c, kind: "stripe", tier: e.insider, price: &price, stripePr: "price_1Q8mL5cD", status: "canceled", period: "year", amount: 8000,
			started: end.AddDate(-1-i%2, 0, 0), canceled: tp(end.AddDate(0, 0, -20)), ended: tp(end), endReason: "canceled", paid: 8000 * int64(1+i%2),
			srcKind: "import", srcLabel: importedLabel, imported: true})
		e.event(id, c, "", "access_ended", end, nil, nil, map[string]any{"reason": "canceled", "didntRenew": true}, nil)
	}
	for i := 0; i < 30; i++ {
		c := e.draw()
		price := pm
		start := s.earnDay(2026, 9, 18, 13, 0).Add(time.Duration(i*13) * time.Hour)
		end := start.AddDate(0, 0, 14)
		if end.After(s.Now) {
			start, end = start.AddDate(0, 0, -14), start
		}
		id := e.add(earnRow{c: c, kind: "stripe", tier: e.insider, price: &price, stripePr: "price_1Q8mL2aB", status: "canceled", period: "month", amount: 800,
			started: start, trialEnd: tp(end), canceled: tp(end.Add(-48 * time.Hour)), ended: tp(end), endReason: "trial_ended", srcKind: newSources[i%5].kind,
			srcLabel: newSources[i%5].label, srcCamp: camp(newSources[i%5].camp), emailed: []string{"welcome", "trial_ending"}})
		e.event(id, c, "", "access_ended", end, nil, nil, map[string]any{"reason": "trial_ended"}, nil)
	}

	// Test mode: two test checkouts from the upgrade page editor.
	tm := e.add(earnRow{key: "t-linh", c: e.contacts["t-linh"], mode: "test", kind: "stripe", tier: e.insider, stripePr: "price_test_1Q8mT2aB", status: "active",
		period: "month", amount: 800, started: s.earnDay(2026, 9, 17, 14, 44), periodEnd: tp(s.earnDay(2026, 10, 17, 14, 44)), paid: 800, srcKind: "page",
		srcLabel: "Test checkout from the upgrade page editor", updated: s.Now.Add(-30 * time.Second)})
	e.event(tm, e.contacts["t-linh"], "test", "subscribed", s.earnDay(2026, 9, 17, 14, 44), nil, nil, nil, nil)
	tf := e.add(earnRow{key: "t-marcus", c: e.contacts["t-marcus"], mode: "test", kind: "stripe", tier: e.founding, stripePr: "price_test_1Q8mU1eF", status: "active",
		period: "year", amount: 20000, started: s.earnDay(2026, 9, 18, 9, 41), periodEnd: tp(s.earnDay(2027, 9, 18, 9, 41)), paid: 20000, srcKind: "page",
		srcLabel: "Test checkout from the upgrade page editor", updated: s.Now.Add(-31 * time.Second)})
	e.event(tf, e.contacts["t-marcus"], "test", "subscribed", s.earnDay(2026, 9, 18, 9, 41), nil, nil, nil, nil)
	return nil
}

func (s *Seeder) earnLoc() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.UTC
	}
	return loc
}

// memberMessage is a member email in the transactional log, delivered by
// Postmark.
func (e *earnSeed) memberMessage(ctx context.Context, c earnContact, email string, version int, subject string, at time.Time) *uuid.UUID {
	tpl, ok := e.templates[email]
	postmark := e.s.Providers["postmark"]
	if !ok || postmark == uuid.Nil {
		return nil
	}
	id := ID()
	if _, err := e.s.Tx.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, contact_id, email, provider_id, provider_message_id, mailbox_provider, status, queued_at, sent_at, delivered_at)
		VALUES ($1, $2, 'transactional', $3, $4, $5, $6, $7, 'delivered', $8, $9, $10)`, id, e.ws.ID, c.id, c.email, postmark, "pm-"+id.String(),
		emailaddr.MailboxProvider(emailaddr.Domain(c.email)), at, at.Add(2*time.Second), at.Add(6*time.Second)); err != nil {
		return nil
	}
	cat := e.category
	if _, err := e.s.Tx.Exec(ctx, `INSERT INTO transactional_messages (message_id, workspace_id, source, template_id, template_key, template_name, template_version,
		category_id, category_name, category_kind, subject, from_email, from_name, reply_to, to_name, tags, metadata, accepted_at, rendered_at, provider_since)
		VALUES ($1, $2, 'api', $3, $4, $5, $6, $7, 'Memberships', 'essential', $8, 'digest@news.acme.io', 'Acme Newsletter', 'hello@acme.io', $9, '{earn}',
		'{"source":"earn"}', $10, $10, $10)`, id, e.ws.ID, tpl, earn.StarterKey(email), earnStarterName(email), version, cat, subject, c.name(), at); err != nil {
		return nil
	}
	return &id
}

func earnStarterName(email string) string {
	for _, st := range earn.Starters("", "") {
		if st.Email == email {
			return st.Name
		}
	}
	return ""
}

// write copies the paid subscriptions and their timelines.
func (e *earnSeed) write(ctx context.Context) error {
	if _, err := e.s.Tx.CopyFrom(ctx, pgx.Identifier{"paid_subscriptions"}, []string{"id", "workspace_id", "contact_id", "mode", "kind", "tier_id", "price_id",
		"stripe_price_id", "status", "period", "amount_cents", "currency", "discount", "started_at", "trial_end", "current_period_end", "cancel_at", "canceled_at",
		"canceled_by", "ended_at", "end_reason", "cancel_reason", "paid_cents", "refunded_cents", "failed_at", "failure_reason", "next_retry_at",
		"stripe_customer_id", "stripe_subscription_id", "billing_email", "source_kind", "source_campaign_id", "source_label", "source_detail", "grant_reason",
		"grant_note", "granted_by", "grant_until", "import_id", "stripe_synced_at", "emailed", "created_at", "updated_at"}, pgx.CopyFromRows(e.subs)); err != nil {
		return fmt.Errorf("paid subscriptions: %w", err)
	}
	if _, err := e.s.Tx.CopyFrom(ctx, pgx.Identifier{"paid_events"}, []string{"id", "workspace_id", "contact_id", "subscription_id", "mode", "kind", "occurred_at",
		"amount_cents", "currency", "payment_event_id", "stripe_event", "message_id", "actor_id", "data"}, pgx.CopyFromRows(e.events)); err != nil {
		return fmt.Errorf("paid events: %w", err)
	}
	return nil
}

// earnPayEvent is a line of the payments event log.
type earnPayEvent struct {
	at      time.Time
	typ     string
	amount  *int64
	outcome string
	detail  map[string]any
	contact *uuid.UUID
	sub     *uuid.UUID
}

// eventLog: the payments event log of the last week (486 events, 71 in the
// last 24 hours) with the board's rows.
func (e *earnSeed) eventLog(ctx context.Context) error {
	s := e.s
	rows := []earnPayEvent{}
	add := func(at time.Time, typ, key string, amount *int64, outcome string, detail map[string]any) {
		r := earnPayEvent{at: at, typ: typ, amount: amount, outcome: outcome, detail: detail}
		if c, ok := e.contacts[key]; ok {
			cid, sid := c.id, e.byKey[key]
			r.contact, r.sub = &cid, &sid
		}
		rows = append(rows, r)
	}
	add(s.earnDay(2026, 10, 15, 9, 56), "invoice.paid", "aiko", i64(8000), "renewed", map[string]any{"tier": "Weekly Digest Insider", "period": "year",
		"until": s.earnDay(2027, 10, 15, 9, 56).UTC().Format(time.RFC3339)})
	add(s.earnDay(2026, 10, 15, 9, 31), "customer.subscription.trial_will_end", "felipe", nil, "sent_trial_ending", map[string]any{"tier": "Weekly Digest Insider",
		"date": s.earnDay(2026, 10, 18, 9, 31).UTC().Format(time.RFC3339), "emailStatus": "sent"})
	add(s.earnDay(2026, 10, 15, 8, 12), "invoice.upcoming", "leo", i64(8000), "sent_renewal_reminder", map[string]any{"tier": "Weekly Digest Insider",
		"date": s.earnDay(2026, 10, 22, 8, 12).UTC().Format(time.RFC3339), "emailStatus": "sent"})
	add(s.earnDay(2026, 10, 13, 16, 41), "customer.subscription.deleted", "marta", nil, "access_ended", map[string]any{"reason": "canceled", "by": "Marcus Chen"})
	add(s.earnDay(2026, 10, 13, 16, 32), "charge.refunded", "marta", i64(-8000), "refunded", map[string]any{"ltv": 8000})
	add(s.earnDay(2026, 10, 13, 9, 12), "invoice.payment_failed", "grace", i64(800), "past_due", map[string]any{"emailStatus": "sent",
		"retry": s.earnDay(2026, 10, 16, 9, 12).UTC().Format(time.RFC3339), "reason": "Your card has insufficient funds."})
	add(s.earnDay(2026, 10, 12, 14, 22), "invoice.paid", "ben", i64(400), "renewed", map[string]any{"tier": "Weekly Digest Insider", "period": "month", "offer": "Stay for half price"})
	add(s.earnDay(2026, 10, 12, 11, 5), "customer.subscription.updated", "olga", nil, "cancel_scheduled", map[string]any{"date": s.earnDay(2026, 11, 2, 18, 30).UTC().Format(time.RFC3339),
		"reason": "not_reading"})
	// The rest of the last 24 hours, so the day reads 71 events: invoice.paid
	// 52, customer.subscription.updated 9, invoice.upcoming 6,
	// customer.subscription.trial_will_end 3, invoice.payment_failed 1.
	k := 0
	for _, d := range []struct {
		typ string
		n   int
	}{{"invoice.paid", 51}, {"customer.subscription.updated", 9}, {"invoice.upcoming", 5}, {"customer.subscription.trial_will_end", 2}, {"invoice.payment_failed", 1}} {
		for i := 0; i < d.n; i++ {
			k++
			rows = append(rows, e.generated(s.Now.Add(-time.Duration(12+k*19)*time.Minute), d.typ, k))
		}
	}
	// The six days before: 415 more, 486 in the week.
	for i := 0; i < 415; i++ {
		typ := "invoice.paid"
		switch {
		case i%10 == 6:
			typ = "customer.subscription.updated"
		case i%10 == 8:
			typ = "invoice.upcoming"
		case i%20 == 9:
			typ = "customer.subscription.trial_will_end"
		}
		rows = append(rows, e.generated(s.Now.Add(-25*time.Hour-time.Duration(i*20+e.rng.IntN(15))*time.Minute), typ, 1000+i))
	}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		evID := "evt_1Q" + e.code(14)
		payload := mustJSON(map[string]any{"id": evID, "object": "event", "type": r.typ, "livemode": true, "created": r.at.Unix(), "api_version": "2024-06-20",
			"data": map[string]any{"object": map[string]any{"object": strings.SplitN(r.typ, ".", 2)[0]}}})
		cur := ""
		if r.amount != nil {
			cur = "usd"
		}
		if r.detail == nil {
			r.detail = map[string]any{}
		}
		out = append(out, []any{ID(), e.ws.ID, e.live, "live", evID, r.typ, "webhook", r.at.Add(-time.Second), r.at, payload, true, "applied", r.outcome,
			mustJSON(r.detail), r.contact, r.sub, r.amount, cur, 1, "", tp(r.at.Add(2 * time.Second))})
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"payment_events"}, []string{"id", "workspace_id", "connection_id", "mode", "provider_event_id", "type", "source",
		"occurred_at", "received_at", "payload", "signature_ok", "status", "outcome", "outcome_detail", "contact_id", "subscription_id", "amount_cents", "currency",
		"attempts", "error", "applied_at"}, pgx.CopyFromRows(out)); err != nil {
		return fmt.Errorf("payment events: %w", err)
	}
	return nil
}

// generated is an event of a generated paid member: a renewal, a plan
// change, a reminder, a trial ending or a failed payment.
func (e *earnSeed) generated(at time.Time, typ string, k int) earnPayEvent {
	r := earnPayEvent{at: at, typ: typ, detail: map[string]any{}}
	pick := e.monthly
	if (typ == "invoice.upcoming" || k%4 == 0) && len(e.yearly) > 0 {
		pick = e.yearly
	}
	m := pick[k%len(pick)]
	cid, sid := m.c.id, e.subOf(m)
	r.contact, r.sub = &cid, sid
	tier := "Weekly Digest Insider"
	switch typ {
	case "invoice.paid":
		r.amount, r.outcome = i64(m.amount), "renewed"
		r.detail = map[string]any{"tier": tier, "period": m.period}
	case "customer.subscription.updated":
		r.outcome = "nothing_changed"
	case "invoice.upcoming":
		r.amount, r.outcome = i64(m.amount), "sent_renewal_reminder"
		r.detail = map[string]any{"tier": tier, "date": at.AddDate(0, 0, 7).UTC().Format(time.RFC3339), "emailStatus": "sent"}
	case "customer.subscription.trial_will_end":
		r.outcome = "sent_trial_ending"
		r.detail = map[string]any{"tier": tier, "date": at.AddDate(0, 0, 3).UTC().Format(time.RFC3339), "emailStatus": "sent"}
	case "invoice.payment_failed":
		r.amount, r.outcome = i64(m.amount), "past_due"
		r.detail = map[string]any{"emailStatus": "sent", "retry": at.AddDate(0, 0, 3).UTC().Format(time.RFC3339), "reason": "Your card was declined."}
	}
	return r
}

// subOf finds a generated row's subscription id (the ids are assigned
// when rows are queued, in order).
func (e *earnSeed) subOf(r earnRow) *uuid.UUID {
	for _, row := range e.subs {
		if row[2] == r.c.id {
			id := row[0].(uuid.UUID)
			return &id
		}
	}
	return nil
}

// checkouts: 312 started on the upgrade page in 30 days, 163 completed:
// 112 started a trial and 51 paid at once (Sam Rivera's among them).
func (e *earnSeed) checkouts(ctx context.Context) error {
	s := e.s
	rows := make([][]any, 0, 311)
	for i := 0; i < 311; i++ {
		at := s.Now.Add(-time.Duration(30*24*60*(i+1)/313) * time.Minute)
		status := "completed"
		switch {
		case i%2 == 1 && i < 260:
			status = "canceled"
		case i >= 290:
			status = "expired"
		}
		var done *time.Time
		if status == "completed" {
			done = tp(at.Add(3 * time.Minute))
		}
		tier, price, period, trial := e.insider, e.pMonth, "month", 14
		switch i % 6 {
		case 1, 2:
			price, period = e.pYear, "year"
		case 5:
			tier, price, period, trial = e.founding, e.pFnd, "year", 0
		}
		rows = append(rows, []any{ID(), e.ws.ID, "live", tier, price, period, trial, "cs_live_" + e.code(24), status, mustJSON(map[string]any{}), at, done})
	}
	// 162 completed with Sam's to come: count the completed above and top up.
	n := 0
	for _, r := range rows {
		if r[8] == "completed" {
			n++
		}
	}
	for i := 0; i < len(rows) && n < 162; i++ {
		if rows[i][8] == "canceled" {
			rows[i][8], rows[i][11] = "completed", tp(rows[i][10].(time.Time).Add(4*time.Minute))
			n++
		}
	}
	// 50 of them paid at once: Founding member has no trial, and readers
	// who'd had one before got none.
	once := 0
	for _, r := range rows {
		if r[8] == "completed" && r[6] == 0 {
			once++
		}
	}
	for _, step := range []int{3, 1} {
		for i := 0; i < len(rows) && once < 50; i += step {
			if rows[i][8] == "completed" && rows[i][6] != 0 {
				rows[i][6] = 0
				once++
			}
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"earn_checkouts"}, []string{"id", "workspace_id", "mode", "tier_id", "price_id", "period", "trial_days",
		"stripe_session_id", "status", "attribution", "created_at", "completed_at"}, pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	// Sam Rivera's checkout from Weekly Digest #141, for the welcome page:
	// /upgrade/welcome?session=cs_live_demoSamRiveraFALL26.
	sam, sub := e.contacts["sam"].id, e.byKey["sam"]
	_, err := s.Tx.Exec(ctx, `INSERT INTO earn_checkouts (id, workspace_id, mode, tier_id, price_id, period, trial_days, contact_id, email, stripe_session_id, status,
		attribution, offer, created_at, completed_at, subscription_id) VALUES ($1, $2, 'live', $3, $4, 'year', 0, $5, 'sam.rivera@outlook.com', 'cs_live_demoSamRiveraFALL26',
		'completed', $6, 'FALL26', $7, $8, $9)`, ID(), e.ws.ID, e.insider, e.pYear, sam, mustJSON(map[string]any{"kind": "campaign", "label": "Weekly Digest #141"}),
		s.earnDay(2026, 10, 7, 9, 45), s.earnDay(2026, 10, 7, 9, 52), sub)
	return err
}

// visits: 4,180 upgrade page visits and 43 cancel page visits in 30 days.
func (e *earnSeed) visits(ctx context.Context) error {
	s := e.s
	loc := s.earnLoc()
	today := s.Now.In(loc)
	upgrade, cancel := 4180, 43
	for d := 29; d >= 0; d-- {
		day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -d)
		u := upgrade / (d + 1)
		c := cancel / (d + 1)
		if d == 0 {
			u, c = upgrade, cancel
		}
		upgrade -= u
		cancel -= c
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_page_visits (workspace_id, day, page, visits) VALUES ($1, $2, 'upgrade', $3), ($1, $2, 'cancel', $4)`,
			e.ws.ID, day, u, c); err != nil {
			return err
		}
	}
	return nil
}

// audit: connecting Stripe, the import, the tiers and the upgrade page,
// Ravi Menon's complimentary access and Marta Kowalski's cancellation.
func (e *earnSeed) audit(ctx context.Context) error {
	s := e.s
	target := func(key string) string { return e.contacts[key].id.String() }
	for _, a := range []auditSeed{
		{at: s.earnDay(2026, 9, 17, 14, 4), actorType: "user", actor: "linh@acme.io", action: "payments.test_key_added", targetType: "payment_connection", targetID: e.test.String(),
			target: "Stripe · Acme Media (test)", ip: "203.0.113.9", ua: uaChromeMac, after: map[string]any{"key": "rk_test_…3mZ8", "account": "Acme Media"}},
		{at: s.earnDay(2026, 9, 17, 14, 12), actorType: "user", actor: "linh@acme.io", action: "payments.connected", targetType: "payment_connection", targetID: e.live.String(),
			target: "Stripe · Acme Media", ip: "203.0.113.9", ua: uaChromeMac, after: map[string]any{"key": "rk_live_…9fQ2", "account": "Acme Media", "permissions": "13 of 13",
				"webhook_events": 12, "tax_at_checkout": true, "prices_include_tax": false}},
		{at: s.earnDay(2026, 9, 18, 10, 3), actorType: "user", actor: "linh@acme.io", action: "earn.tier_created", targetType: "earn_tier", targetID: e.insider.String(),
			target: "Weekly Digest Insider", ip: "203.0.113.9", ua: uaChromeMac, after: map[string]any{"prices": "$8/month, $80/year", "trial_days": 14, "lists": "Weekly Digest"}},
		{at: s.earnDay(2026, 9, 18, 10, 3).Add(20 * time.Second), actorType: "user", actor: "linh@acme.io", action: "earn.tier_created", targetType: "earn_tier",
			targetID: e.founding.String(), target: "Founding member", ip: "203.0.113.9", ua: uaChromeMac, after: map[string]any{"prices": "$200/year", "lists": "Weekly Digest"}},
		{at: s.earnDay(2026, 9, 18, 10, 6), actorType: "user", actor: "linh@acme.io", action: "paid_member.imported", targetType: "earn_import", targetID: e.imp.String(),
			target: "substack-acme-insider-export.zip", ip: "203.0.113.9", ua: uaChromeMac, meta: map[string]any{"source": "substack", "rows": 1150, "paid": 1121, "free": 12,
				"complimentary": 17, "merged": 1083, "new": 67, "prices_mapped": 3, "emails_sent": 0}},
		{at: s.earnDay(2026, 9, 18, 11, 30), actorType: "user", actor: "linh@acme.io", action: "earn.upgrade_page_updated", targetType: "workspace", targetID: e.ws.ID.String(),
			target: "Upgrade page", ip: "203.0.113.9", ua: uaChromeMac, after: map[string]any{"headline": "Get more from the Weekly Digest", "highlight": "Weekly Digest Insider"}},
		{at: s.earnDay(2026, 10, 3, 15, 40), actorType: "user", actor: "marcus@acme.io", action: "paid_member.complimentary_granted", targetType: "contact", targetID: target("ravi"),
			target: "ravi@menon.example", reason: "contributor", ip: "198.51.100.24", ua: uaSafariMac, after: map[string]any{"tier": "Weekly Digest Insider", "until": "", "reason": "contributor"}},
		{at: s.earnDay(2026, 10, 13, 16, 40), actorType: "user", actor: "marcus@acme.io", action: "paid_member.subscription_canceled", targetType: "contact", targetID: target("marta"),
			target: "marta.k@gmail.com", ip: "198.51.100.24", ua: uaSafariMac, before: map[string]any{"status": "active"}, after: map[string]any{"when": "now", "email": false},
			meta: map[string]any{"note": "Refunded in Stripe on request", "tier": "Weekly Digest Insider"}},
	} {
		if err := s.insertAudit(ctx, e.ws.ID, a); err != nil {
			return err
		}
	}
	return nil
}
