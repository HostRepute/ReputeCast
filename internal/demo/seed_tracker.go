package demo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	mrand "math/rand/v2"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/mjml"
)

func init() { seedTracker = seedTrackerImpl }

// Website tracking (M62, TrackingSites, TrackingStates, AutomationsComplete
// strip 8, ContactProfile): Acme Merch (https://shop.acme.io, linked to the
// demo's Shopify store, added by Marcus Chen on 8 Oct) and Docs
// (https://docs.acme.io, no store); 30 days of events from 400 known
// contacts and 2,000 visitors, with about 1,240 and 312 events today; Sam
// Rivera's three product views two days ago; a live "Browse abandonment"
// flow (viewed a product, after 4 hours) with 30 days of step stats and a
// "Back in stock" flow with the requests waiting for Deploy Friday cap; and
// the products bought together, rebuilt from the store's orders.

const (
	trackerKnown    = 400
	trackerVisitors = 2000
	trackerDays     = 30
	merchToday      = 1240
	docsToday       = 312
)

type trackerSeed struct {
	s        *Seeder
	ctx      context.Context
	rng      *mrand.Rand
	ws       uuid.UUID
	storeID  uuid.UUID
	merch    uuid.UUID
	docs     uuid.UUID
	products []trackerProduct
	known    []uuid.UUID
	// visitors: the first trackerKnown are the known contacts'.
	visitors []uuid.UUID
}

type trackerProduct struct {
	id    uuid.UUID
	ext   string
	title string
	path  string
}

// interest is a known contact's views and add-to-carts of a product.
type trackerInterest struct {
	views, carts   int
	viewed, carted time.Time
}

func seedTrackerImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	marcus := s.Users["marcus@acme.io"].ID
	if ws.ID == uuid.Nil || marcus == uuid.Nil {
		return nil
	}
	g := &trackerSeed{s: s, ctx: ctx, rng: audRand(2026, 1108), ws: ws.ID, merch: ID(), docs: ID()}
	err := s.Tx.QueryRow(ctx, `SELECT id FROM stores WHERE workspace_id = $1 AND platform = 'shopify' ORDER BY created_at LIMIT 1`, ws.ID).Scan(&g.storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	steps := []func() error{g.sites, g.people, g.events, g.flows, g.pairs}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// sites adds Acme Merch and Docs, each with a sealed identify secret.
func (g *trackerSeed) sites() error {
	s, ctx := g.s, g.ctx
	marcus := s.Users["marcus@acme.io"].ID
	for _, st := range []struct {
		id           uuid.UUID
		name, origin string
		store        *uuid.UUID
		added        time.Time
	}{
		{g.merch, "Acme Merch", "https://shop.acme.io", &g.storeID, s.designDay(35, 10, 12)},
		{g.docs, "Docs", "https://docs.acme.io", nil, s.designDay(34, 15, 40)},
	} {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		var sealed []byte
		if s.Box != nil {
			var err error
			if sealed, err = s.Box.Seal([]byte("rcid_"+base64.RawURLEncoding.EncodeToString(b)), "tracker_site:"+st.id.String()+":identify_secret"); err != nil {
				return err
			}
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO tracker_sites (id, workspace_id, name, origin, store_id, wait_consent, identify_secret_sealed, created_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,true,$6,$7,$8,$8)`, st.id, g.ws, st.name, st.origin, st.store, sealed, marcus, st.added); err != nil {
			return err
		}
	}
	return nil
}

// people picks the 400 known contacts (Sam Rivera and Maya Patel among
// them) and the store's products, and adds the 2,400 visitors.
func (g *trackerSeed) people() error {
	s, ctx := g.s, g.ctx
	rows, err := s.Tx.Query(ctx, `SELECT id, external_id, title, handle FROM store_products
		WHERE store_id = $1 AND removed_at IS NULL AND status = 'active' AND price_cents >= 800 ORDER BY created_at_store DESC NULLS LAST, external_id`, g.storeID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p trackerProduct
		var handle string
		if err := rows.Scan(&p.id, &p.ext, &p.title, &handle); err != nil {
			rows.Close()
			return err
		}
		p.path = "/products/" + handle
		g.products = append(g.products, p)
	}
	rows.Close()
	if len(g.products) == 0 {
		return nil
	}
	rows, err = s.Tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND status = 'active' AND erased_at IS NULL
		ORDER BY (lower(first_name || ' ' || last_name) IN ('sam rivera', 'maya patel')) DESC, id LIMIT $2`, g.ws, trackerKnown)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		g.known = append(g.known, id)
	}
	rows.Close()
	ids, hashes, contacts, by, first, last, identified := []uuid.UUID{}, [][]byte{}, []*uuid.UUID{}, []string{}, []time.Time{}, []time.Time{}, []*time.Time{}
	for i := range len(g.known) + trackerVisitors {
		id := ID()
		g.visitors = append(g.visitors, id)
		h := sha256.Sum256([]byte(fmt.Sprintf("demo-visitor-%d-%s", i, g.ws)))
		f := s.Ago(time.Duration(trackerDays*24-g.rng.IntN(24*25)) * time.Hour)
		ids, hashes, first, last = append(ids, id), append(hashes, h[:]), append(first, f), append(last, f)
		if i < len(g.known) {
			c, at := g.known[i], f.Add(time.Hour)
			contacts, by, identified = append(contacts, &c), append(by, []string{"click", "click", "form", "identify"}[i%4]), append(identified, &at)
		} else {
			contacts, by, identified = append(contacts, nil), append(by, ""), append(identified, nil)
		}
	}
	_, err = s.Tx.Exec(ctx, `INSERT INTO tracker_visitors (id, workspace_id, vid_hash, contact_id, identified_by, first_seen, last_seen, identified_at)
		SELECT u.id, $1, u.h, u.c, u.b, u.f, u.l, u.i FROM unnest($2::uuid[], $3::bytea[], $4::uuid[], $5::text[], $6::timestamptz[], $7::timestamptz[], $8::timestamptz[])
		AS u (id, h, c, b, f, l, i)`, g.ws, ids, hashes, contacts, by, first, last, identified)
	return err
}

// events writes 30 days of page views, product views and add-to-carts on
// both sites, today's counts as the boards show them, the newest events
// minutes ago, and the known contacts' product interest.
func (g *trackerSeed) events() error {
	s, ctx := g.s, g.ctx
	if len(g.products) == 0 || len(g.visitors) == 0 {
		return nil
	}
	loc, err := time.LoadLocation(s.WS().Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := s.Now
	n := now.In(loc)
	midnight := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	interest := map[[2]uuid.UUID]*trackerInterest{}
	seen := map[uuid.UUID]time.Time{}
	var rows [][]any
	add := func(site uuid.UUID, visitor int, kind int16, p *trackerProduct, path, ref string, at time.Time) {
		v := g.visitors[visitor]
		var contact, product any
		if visitor < len(g.known) {
			c := g.known[visitor]
			contact = c
			if p != nil && kind != 1 {
				k := [2]uuid.UUID{c, p.id}
				it := interest[k]
				if it == nil {
					it = &trackerInterest{}
					interest[k] = it
				}
				if kind == 2 {
					it.views++
					it.viewed = trackerLater(it.viewed, at)
				} else {
					it.carts++
					it.carted = trackerLater(it.carted, at)
				}
			}
		}
		if p != nil {
			product = p.id
		}
		qty := 0
		if kind == 3 {
			qty = 1
		}
		seen[v] = trackerLater(seen[v], at)
		rows = append(rows, []any{ID(), g.ws, site, v, contact, kind, product, qty, path, ref, "", nil, at})
	}
	docsPaths := []string{"/", "/guides/feature-flags", "/guides/rollouts", "/api/webhooks", "/guides/canary-releases", "/changelog"}
	refs := []string{"", "", "", "google.com", "news.acme.io", "duckduckgo.com", "github.com"}
	// One visit: a page, then on the shop maybe a product and maybe the cart.
	visit := func(site uuid.UUID, at time.Time, budget *int) {
		visitor := g.rng.IntN(len(g.visitors))
		if g.rng.IntN(3) == 0 {
			visitor = g.rng.IntN(len(g.known))
		}
		if site == g.docs {
			add(site, visitor, 1, nil, docsPaths[g.rng.IntN(len(docsPaths))], refs[g.rng.IntN(len(refs))], at)
			*budget--
			if *budget > 0 && g.rng.IntN(4) == 0 {
				p := &g.products[g.rng.IntN(len(g.products))]
				add(site, visitor, 2, nil, p.path, "", at.Add(40*time.Second))
				*budget--
			}
			return
		}
		add(site, visitor, 1, nil, []string{"/", "/collections/apparel", "/collections/new-arrivals", "/collections/desk"}[g.rng.IntN(4)], refs[g.rng.IntN(len(refs))], at)
		*budget--
		if *budget > 0 && g.rng.IntN(10) < 7 {
			p := &g.products[min(g.rng.IntN(len(g.products)), g.rng.IntN(len(g.products)))]
			add(site, visitor, 2, p, p.path, "", at.Add(time.Minute))
			*budget--
			// Now and then the shop's own event: the size guide opened.
			if *budget > 0 && g.rng.IntN(25) == 0 {
				var contact any
				if visitor < len(g.known) {
					contact = g.known[visitor]
				}
				rows = append(rows, []any{ID(), g.ws, site, g.visitors[visitor], contact, int16(4), nil, 0, p.path, "", "size_guide_opened", nil, at.Add(90 * time.Second)})
				*budget--
			}
			if *budget > 0 && g.rng.IntN(5) == 0 {
				add(site, visitor, 3, p, p.path, "", at.Add(2*time.Minute))
				*budget--
			}
		}
	}
	for day := trackerDays; day >= 0; day-- {
		start := midnight.AddDate(0, 0, -day)
		end := start.AddDate(0, 0, 1)
		merch, docs := 700+g.rng.IntN(600), 160+g.rng.IntN(180)
		if day == 0 {
			// Today: up to the boards' counts, the last event 2 minutes ago
			// on the shop and 14 on Docs.
			merch, docs = merchToday, docsToday
			end = now.Add(-2 * time.Minute)
		}
		for _, site := range []struct {
			id    uuid.UUID
			count int
			end   time.Time
		}{{g.merch, merch, end}, {g.docs, docs, trackerEarlier(end, now.Add(-14*time.Minute))}} {
			span := site.end.Sub(start)
			if span <= 0 {
				continue
			}
			budget := site.count
			// A visit's events take two minutes: the first one ends at the
			// day's last event, the others before it.
			latest := site.end.Add(-2 * time.Minute)
			for budget > 0 {
				at := start.Add(time.Duration(g.rng.Int64N(int64(span))))
				if budget == site.count || at.After(latest) {
					at = latest
				}
				visit(site.id, at, &budget)
			}
		}
	}
	// Sam Rivera's three product views two days ago (ContactProfile).
	if len(g.known) > 0 {
		day := s.designDay(2, 14, 5)
		for i, ext := range []string{"8841203", "8841207", "8841162"} {
			for j := range g.products {
				if g.products[j].ext == ext {
					add(g.merch, 0, 2, &g.products[j], g.products[j].path, "", day.Add(time.Duration(i*3)*time.Minute))
				}
			}
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"tracker_events"},
		[]string{"id", "workspace_id", "site_id", "visitor_id", "contact_id", "kind", "product_id", "quantity", "path", "referrer_domain", "name", "props", "occurred_at"},
		pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	// The visitors' last time seen, and each site's last event.
	vis, at := make([]uuid.UUID, 0, len(seen)), make([]time.Time, 0, len(seen))
	for v, t := range seen {
		vis, at = append(vis, v), append(at, t)
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE tracker_visitors v SET last_seen = u.at FROM unnest($1::uuid[], $2::timestamptz[]) AS u (id, at) WHERE v.id = u.id`, vis, at); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE tracker_sites t SET last_event_at = (SELECT max(occurred_at) FROM tracker_events e WHERE e.site_id = t.id) WHERE t.workspace_id = $1`, g.ws); err != nil {
		return err
	}
	keys := make([][2]uuid.UUID, 0, len(interest))
	for k := range interest {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i][0].String()+keys[i][1].String() < keys[j][0].String()+keys[j][1].String()
	})
	var cs, ps []uuid.UUID
	var views, carts []int32
	var viewed, carted []*time.Time
	for _, k := range keys {
		it := interest[k]
		cs, ps = append(cs, k[0]), append(ps, k[1])
		views, carts = append(views, int32(it.views)), append(carts, int32(it.carts))
		viewed, carted = append(viewed, trackerTimeOrNil(it.viewed)), append(carted, trackerTimeOrNil(it.carted))
	}
	_, err = s.Tx.Exec(ctx, `INSERT INTO tracker_product_views (workspace_id, contact_id, product_id, views, carts, last_viewed_at, last_carted_at)
		SELECT $1, u.c, u.p, u.v, u.k, u.vt, u.kt FROM unnest($2::uuid[], $3::uuid[], $4::int[], $5::int[], $6::timestamptz[], $7::timestamptz[]) AS u (c, p, v, k, vt, kt)`,
		g.ws, cs, ps, views, carts, viewed, carted)
	return err
}

func trackerLater(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func trackerEarlier(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func trackerTimeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// flows adds the live Browse abandonment flow (viewed a product, after 4
// hours) with 30 days of numbers and the Back in stock flow, with the
// people waiting for Deploy Friday cap.
func (g *trackerSeed) flows() error {
	s, ctx := g.s, g.ctx
	ws := s.WS()
	marcus := s.Users["marcus@acme.io"].ID
	svc := &commerce.Service{Q: s.Q}
	draw := func(b commerce.CommerceBlock) (string, error) {
		r, err := svc.RenderBlock(ctx, &ws, b)
		if err != nil {
			return "", err
		}
		return storeGridSection(g.storeID, r.HTML), nil
	}
	viewed, err := draw(commerce.CommerceBlock{Store: g.storeID.String(), Mode: "viewed", Count: 3, SoldOut: "label", Button: "Take another look", Heading: "What you looked at"})
	if err != nil {
		return err
	}
	best, err := draw(commerce.CommerceBlock{Store: g.storeID.String(), Mode: "best_sellers", Count: 3, SoldOut: "hide", Button: "Shop now", Heading: "Best sellers this month"})
	if err != nil {
		return err
	}
	recent, err := draw(commerce.CommerceBlock{Store: g.storeID.String(), Mode: "recently_viewed", Count: 3, SoldOut: "hide", Button: "Shop now", Heading: "Recently viewed"})
	if err != nil {
		return err
	}
	browse, stock := ID(), ID()
	oneID, twoID, backID := ID(), ID(), ID()
	created, published := s.designDay(33, 9, 40), s.designDay(33, 10, 5)
	window := &automations.AutomationSendWindow{Enabled: false}
	browseGraph := automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			autoNode("n1", automations.NodeTrigger, 300, 24, autoFlowData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerViewedProduct,
				Browse: &automations.AutomationBrowseTrigger{StoreID: g.storeID.String(), AfterHours: 4, MinViews: 1}}}),
			autoNode("n2", automations.NodeEmail, 300, 180, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: oneID.String(), EmailVersion: 1, Subject: "Still thinking it over?", SendWindow: window}}),
			autoNode("n3", automations.NodeDelay, 300, 316, autoFlowData{Delay: autoDur(2, "days")}),
			autoNode("n4", automations.NodeEmail, 300, 430, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: twoID.String(), EmailVersion: 1, Subject: "Picked for you", SendWindow: window}}),
			autoNode("n5", automations.NodeExit, 300, 566, autoFlowData{Label: "Leave the flow"}),
		},
		Edges: []automations.AutomationEdge{autoEdge("n1", "n2", ""), autoEdge("n2", "n3", ""), autoEdge("n3", "n4", ""), autoEdge("n4", "n5", "")},
	}
	stockGraph := automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			autoNode("n1", automations.NodeTrigger, 300, 24, autoFlowData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerBackInStock,
				Stock: &automations.AutomationStockTrigger{StoreID: g.storeID.String(), Days: 14}}}),
			autoNode("n2", automations.NodeEmail, 300, 180, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: backID.String(), EmailVersion: 1, Subject: "It's back", SendWindow: window}}),
			autoNode("n3", automations.NodeExit, 300, 316, autoFlowData{Label: "Leave the flow"}),
		},
		Edges: []automations.AutomationEdge{autoEdge("n1", "n2", ""), autoEdge("n2", "n3", "")},
	}
	for _, f := range []autoFlowSeed{
		{id: browse, name: "Browse abandonment", description: "2 emails over 2 days, from Website tracking", status: "live", live: 1, createdBy: marcus, created: created, updated: published,
			versions: []autoVersionSeed{{graph: browseGraph, by: marcus, created: created, published: &published, summary: "created"}}},
		{id: stock, name: "Back in stock", description: "One email when a sold-out product returns", status: "live", live: 1, createdBy: marcus, created: created, updated: published,
			versions: []autoVersionSeed{{graph: stockGraph, by: marcus, created: created, published: &published, summary: "created"}}},
	} {
		if err := autoInsertFlow(ctx, s, f); err != nil {
			return err
		}
	}
	for _, e := range []struct {
		id, flow   uuid.UUID
		name, subj string
		mj         string
	}{
		{oneID, browse, `Browse abandonment · email 1 "Still thinking it over?"`, "Still thinking it over?",
			storeEmailMJML("Still thinking it over?", "What you looked at is still here.", `Hi {{ first_name | default: "there" }}, you were looking at these in the Acme Merch shop:`, viewed)},
		{twoID, browse, `Email 2 "Picked for you"`, "Picked for you",
			storeEmailMJML("Picked for you", "From what you looked at, and what everyone buys.", `A few more from the shop, {{ first_name | default: "there" }}:`, recent+"\n  "+best)},
		{backID, stock, `Back in stock · "It's back"`, "It's back",
			storeEmailMJML("It's back", "You asked us to tell you.", `{{ stock.product.title }} is back in the Acme Merch shop, at {{ stock.product.price }}. <a href="{{ stock.product.url }}" style="color:#0F766E">Get yours</a> before it sells out again.`, best)},
	} {
		res := mjml.Compile(e.mj)
		if len(res.Errors) > 0 {
			return fmt.Errorf("compile %s: %s", e.name, res.Errors[0].Message)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_emails (id, workspace_id, automation_id, name, latest_version, created_by, created_at, updated_at) VALUES ($1,$2,$3,$4,1,$5,$6,$6)`,
			e.id, g.ws, e.flow, e.name, marcus, created); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at) VALUES ($1,1,$2,'',$3,$4,$5,$6,$7)`,
			e.id, e.subj, e.mj, res.HTML, e.subj, marcus, created); err != nil {
			return err
		}
	}
	// 30 days of numbers: 186 people entered browse abandonment (one entry
	// per person a week), 152 got email 2.
	ses := s.Providers["ses"].String()
	if err := autoInsertStats(ctx, s, []autoNodeStats{
		{a: browse, v: 1, node: "n1", from: trackerDays, c: autoCounters{"entered": 186, "completed": 186}},
		{a: browse, v: 1, node: "n2", from: trackerDays, c: autoCounters{"entered": 186, "completed": 186, "sent": 186, "delivered": 184, "opened": 118, "opened_human": 101, "clicked": 41},
			providers: map[string]int64{ses: 186}},
		{a: browse, v: 1, node: "n3", from: trackerDays, c: autoCounters{"entered": 186, "completed": 152, "waiting": 186}},
		{a: browse, v: 1, node: "n4", from: trackerDays - 2, c: autoCounters{"entered": 152, "completed": 152, "sent": 152, "delivered": 151, "opened": 87, "opened_human": 74, "clicked": 22},
			providers: map[string]int64{ses: 152}},
		{a: browse, v: 1, node: "n5", from: trackerDays - 2, c: autoCounters{"entered": 152, "exited": 152}},
	}); err != nil {
		return err
	}
	// Who entered the flow lately (the 7-day rule), and who waits for
	// Deploy Friday cap (sold out): 23 people.
	if len(g.known) > 30 {
		enter := g.known[1:31]
		ats := make([]time.Time, len(enter))
		for i := range enter {
			ats[i] = s.Ago(time.Duration(4+i*5) * time.Hour)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO tracker_trigger_entries (automation_id, contact_id, kind, entered_at)
			SELECT $1, u.c, 'viewed_product', u.at FROM unnest($2::uuid[], $3::timestamptz[]) AS u (c, at)`, browse, enter, ats); err != nil {
			return err
		}
	}
	if len(g.known) > 60 {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO store_stock_requests (workspace_id, contact_id, product_id, created_at)
			SELECT $1, c, p.id, $4::timestamptz - (row_number() OVER ()) * interval '7 hours'
			FROM unnest($2::uuid[]) AS c, store_products p WHERE p.store_id = $3 AND p.external_id = '8841177'`, g.ws, g.known[37:60], g.storeID, s.Now); err != nil {
			return err
		}
	}
	return nil
}

// pairs rebuilds the store's products bought together from its orders.
func (g *trackerSeed) pairs() error {
	svc := &commerce.Service{Q: g.s.Q, Now: func() time.Time { return g.s.Now }}
	return svc.RebuildPairs(g.ctx, g.storeID)
}
