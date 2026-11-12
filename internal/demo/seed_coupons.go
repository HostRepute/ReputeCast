package demo

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/coupons"
	"github.com/hostrepute/reputecast/internal/mjml"
)

func init() { seedCoupons = seedCouponsImpl }

// The demo's discount codes (M59): "Welcome 15%", created in Acme Merch
// (15% off, WELCOME-, one per contact, low at 200): 12,400 codes, 9,860
// sent, 312 redeemed in orders worth $18,240.00; "Partner perks", uploaded
// with the apply address https://shop.acme.io/discount/{code}: 2,000 codes,
// 120 sent. Step 2 of the live Welcome series shows a Welcome code
// (DiscountCodePool).
const (
	couponWelcomeTotal    = 12400
	couponWelcomeSent     = 9860
	couponWelcomeRedeemed = 312
	couponWelcomeRevenue  = 1_824_000 // cents
	couponPartnerTotal    = 2000
	couponPartnerSent     = 120
	// CouponExample is the code the boards show.
	CouponExample = "WELCOME-7KQ4M2XA"
)

type couponOrder struct {
	id, contact uuid.UUID
	total       int64
	placed      time.Time
}

func seedCouponsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	var storeID uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM stores WHERE workspace_id = $1 AND name = 'Acme Merch'`, ws.ID).Scan(&storeID); err != nil {
		return fmt.Errorf("coupons: Acme Merch: %w", err)
	}
	marcus := s.Users["marcus@acme.io"].ID
	rng := rand.New(rand.NewPCG(59, 22)) //nolint:gosec // deterministic sample data
	welcome, partner := ID(), ID()
	created := s.Ago(300 * 24 * time.Hour)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO coupon_pools (id, workspace_id, name, slug, source, store_id, settings, prefix, one_per_contact, low_at, status,
		external_discount_id, created_by, created_at, updated_at) VALUES
		($1, $3, 'Welcome 15%', 'welcome15', 'shopify', $4, '{"kind":"percentage","value":15}', 'WELCOME-', true, 200, 'active', 'gid://shopify/DiscountCodeNode/1290000001', $5, $6, $6),
		($2, $3, 'Partner perks', 'partner_perks', 'upload', NULL, '{"applyUrl":"https://shop.acme.io/discount/{code}"}', '', true, 200, 'active', '', $5, $7, $7)`,
		welcome, partner, ws.ID, storeID, marcus, created, s.Ago(60*24*time.Hour)); err != nil {
		return err
	}
	// The redeemed codes' orders: one each for 312 contacts, older than the
	// last 30 days and newer than the pool, adding up to $18,240.00.
	orders, err := couponOrders(ctx, s, storeID, created)
	if err != nil {
		return err
	}
	// The other readers who got a Welcome code, and the Partner perks ones.
	taken := map[uuid.UUID]bool{}
	for _, o := range orders {
		taken[o.contact] = true
	}
	rows, err := s.Tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND status = 'active' ORDER BY id LIMIT $2`, ws.ID, couponWelcomeSent+couponPartnerSent+len(orders))
	if err != nil {
		return err
	}
	others, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	others = slices.DeleteFunc(others, func(id uuid.UUID) bool { return taken[id] })
	need := couponWelcomeSent - len(orders)
	if len(others) < need+couponPartnerSent {
		return fmt.Errorf("coupons: %d contacts, need %d", len(others), need+couponPartnerSent)
	}
	codes := demoCouponCodes("WELCOME-", couponWelcomeTotal, rng)
	codes[0] = CouponExample
	var batch [][]any
	add := func(pool uuid.UUID, code string, at time.Time, contact *uuid.UUID, assigned, redeemed *time.Time, order *uuid.UUID) error {
		sealed, err := s.Box.SealString(code, "coupon_code:"+pool.String())
		if err != nil {
			return err
		}
		ext := ""
		if pool == welcome {
			ext = fmt.Sprintf("gid://shopify/DiscountRedeemCode/%d", 2290000000+len(batch))
		}
		batch = append(batch, []any{ID(), ws.ID, pool, sealed, coupons.CodeHash(ws.ID, code), ext, at, assigned, contact, redeemed, order})
		return nil
	}
	recent := s.Ago(5 * 24 * time.Hour)
	for i, code := range codes {
		at := created.Add(time.Duration(i/250) * 24 * time.Hour)
		switch {
		case i < len(orders):
			o := orders[i]
			assigned := o.placed.Add(-time.Duration(26+rng.IntN(48)) * time.Hour)
			if assigned.Before(at) {
				at = assigned.Add(-time.Hour)
			}
			placed := o.placed
			if err := add(welcome, code, at, &o.contact, &assigned, &placed, &o.id); err != nil {
				return err
			}
		case i < couponWelcomeSent:
			c := others[i-len(orders)]
			assigned := created.Add(time.Duration(float64(recent.Sub(created)) * float64(i) / couponWelcomeSent))
			if assigned.Before(at) {
				at = assigned.Add(-time.Hour)
			}
			if err := add(welcome, code, at, &c, &assigned, nil, nil); err != nil {
				return err
			}
		default:
			if err := add(welcome, code, s.Ago(time.Duration(10+i%30)*24*time.Hour), nil, nil, nil, nil); err != nil {
				return err
			}
		}
	}
	for i, code := range demoCouponCodes("PERK", couponPartnerTotal, rng) {
		var contact *uuid.UUID
		var assigned *time.Time
		if i < couponPartnerSent {
			c, a := others[need+i], s.Ago(time.Duration(50-i%45)*24*time.Hour)
			contact, assigned = &c, &a
		}
		if err := add(partner, code, s.Ago(60*24*time.Hour), contact, assigned, nil, nil); err != nil {
			return err
		}
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"coupon_codes"}, []string{"id", "workspace_id", "pool_id", "code_sealed", "code_hash", "external_id",
		"created_at", "assigned_at", "contact_id", "redeemed_at", "order_id"}, pgx.CopyFromRows(batch)); err != nil {
		return err
	}
	// The orders carry the codes they used, as Shopify reports them.
	ids, used := make([]uuid.UUID, len(orders)), make([]string, len(orders))
	for i, o := range orders {
		ids[i], used[i] = o.id, codes[i]
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE store_orders o SET discount_codes = ARRAY[x.code] FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::text[]) AS code) x WHERE o.id = x.id`,
		ids, used); err != nil {
		return err
	}
	if err := couponBatches(ctx, s, welcome, partner, created); err != nil {
		return err
	}
	return couponEmail(ctx, s)
}

// couponBatches are the pools' batches (DiscountCodePool): Welcome 15%'s
// first 10,000 (2 already in Acme Merch), two hourly refills for the
// Welcome series and Marcus's 500 under way (250 done), adding up to its
// 12,400; Partner perks' 2,000, uploaded by Linh.
func couponBatches(ctx context.Context, s *Seeder, welcome, partner uuid.UUID, created time.Time) error {
	ws := s.WS().ID
	marcus, linh := s.Users["marcus@acme.io"].ID, s.Users["linh@acme.io"].ID
	var flow uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM automations WHERE workspace_id = $1 AND name = 'Welcome series'`, ws).Scan(&flow); err != nil {
		return fmt.Errorf("coupons: the Welcome series: %w", err)
	}
	type b struct {
		pool                     uuid.UUID
		kind, reason, status     string
		by, automation           *uuid.UUID
		requested, made, refused int
		at                       time.Time
		finished                 *time.Time
	}
	done := func(t time.Time) *time.Time { f := t.Add(4 * time.Minute); return &f }
	first, refill1, refill2 := created.Add(40*time.Minute), s.Ago(11*24*time.Hour), s.Ago(24*time.Hour)
	upload := s.Ago(8 * 24 * time.Hour)
	rows := []b{
		{pool: welcome, kind: "generate", reason: "pool", status: "done", by: &marcus, requested: 10000, made: 9998, refused: 2, at: first, finished: done(first)},
		{pool: welcome, kind: "generate", reason: "refill", status: "done", automation: &flow, requested: 952, made: 952, at: refill1, finished: done(refill1)},
		{pool: welcome, kind: "generate", reason: "refill", status: "done", automation: &flow, requested: 1200, made: 1200, at: refill2, finished: done(refill2)},
		{pool: welcome, kind: "generate", reason: "request", status: "running", by: &marcus, requested: 500, made: 250, at: s.Ago(18 * time.Minute)},
		{pool: partner, kind: "upload", reason: "request", status: "done", by: &linh, requested: 2000, made: 2000, at: upload, finished: done(upload)},
	}
	for _, r := range rows {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO coupon_batches (id, workspace_id, pool_id, kind, reason, automation_id, requested, created, failed, status,
			created_by, created_at, finished_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			ID(), ws, r.pool, r.kind, r.reason, r.automation, r.requested, r.made, r.refused, r.status, r.by, r.at, r.finished); err != nil {
			return err
		}
	}
	// The hourly refill looked at Welcome 15% at the top of this hour.
	_, err := s.Tx.Exec(ctx, `UPDATE coupon_pools SET refill_checked_at = $2 WHERE id = $1`, welcome, s.Now.Truncate(time.Hour))
	return err
}

// couponOrders picks 312 orders of different contacts whose totals add up
// to the Welcome pool's revenue exactly.
func couponOrders(ctx context.Context, s *Seeder, storeID uuid.UUID, since time.Time) ([]couponOrder, error) {
	rows, err := s.Tx.Query(ctx, `SELECT DISTINCT ON (contact_id) id, contact_id, total_cents, placed_at FROM store_orders
		WHERE store_id = $1 AND contact_id IS NOT NULL AND NOT cancelled AND NOT test AND refunded_cents = 0 AND placed_at > $2 AND placed_at < $3
		ORDER BY contact_id, placed_at, id`, storeID, since.Add(72*time.Hour), s.Ago(30*24*time.Hour))
	if err != nil {
		return nil, err
	}
	var all []couponOrder
	for rows.Next() {
		var o couponOrder
		if err := rows.Scan(&o.id, &o.contact, &o.total, &o.placed); err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, o)
	}
	rows.Close()
	if len(all) < couponWelcomeRedeemed*2 {
		return nil, fmt.Errorf("coupons: %d orders to redeem codes in, need %d", len(all), couponWelcomeRedeemed*2)
	}
	// Start from the orders closest to the average, then swap one in and
	// one out until the sum is exact.
	avg := int64(couponWelcomeRevenue / couponWelcomeRedeemed)
	slices.SortFunc(all, func(a, b couponOrder) int {
		da, db := absCents(a.total-avg), absCents(b.total-avg)
		if da != db {
			return int(da - db)
		}
		return strings.Compare(a.id.String(), b.id.String())
	})
	picked, rest := slices.Clone(all[:couponWelcomeRedeemed]), slices.Clone(all[couponWelcomeRedeemed:])
	var sum int64
	for _, o := range picked {
		sum += o.total
	}
	for tries := 0; sum != couponWelcomeRevenue && tries < 10000; tries++ {
		diff := int64(couponWelcomeRevenue) - sum
		bi, bj, best := -1, -1, absCents(diff)
		for i := range picked {
			for j := range rest {
				if d := absCents(diff - (rest[j].total - picked[i].total)); d < best {
					bi, bj, best = i, j, d
				}
			}
			if best == 0 {
				break
			}
		}
		if bi < 0 {
			break
		}
		sum += rest[bj].total - picked[bi].total
		picked[bi], rest[bj] = rest[bj], picked[bi]
	}
	if sum != couponWelcomeRevenue {
		return nil, fmt.Errorf("coupons: redeemed orders add up to %d cents, not %d", sum, couponWelcomeRevenue)
	}
	slices.SortFunc(picked, func(a, b couponOrder) int { return a.placed.Compare(b.placed) })
	return picked, nil
}

func absCents(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// demoCouponCodes are n distinct codes with the prefix, as the pools
// create them (8 characters without look-alikes), from the demo's rng.
func demoCouponCodes(prefix string, n int, rng *rand.Rand) []string {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	out := make([]string, 0, n)
	seen := map[string]bool{CouponExample: true}
	for len(out) < n {
		b := []byte(prefix)
		for range 8 {
			b = append(b, alphabet[rng.IntN(len(alphabet))])
		}
		if c := string(b); !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

// couponEmail puts a Discount code block for Welcome 15% in the Welcome
// series' second email (live).
func couponEmail(ctx context.Context, s *Seeder) error {
	var id uuid.UUID
	var src string
	err := s.Tx.QueryRow(ctx, `SELECT e.id, v.mjml FROM automation_emails e JOIN automation_email_versions v ON v.email_id = e.id AND v.version = e.latest_version
		WHERE e.workspace_id = $1 AND e.name = $2`, s.WS().ID, `Step 2 "Three things to try first"`).Scan(&id, &src)
	if err != nil {
		return fmt.Errorf("coupons: the Welcome series email: %w", err)
	}
	block := `  <mj-section background-color="#ffffff" padding="0px 24px 16px"><mj-column>
    <mj-text font-size="15px" line-height="1.6" color="#374151" padding="0px 0px 4px">And a welcome gift: 15% off your first order in the Acme Merch store.</mj-text>
    <mj-raw><!-- rc:coupon {"pool":"welcome15","label":"Your code","button":"Shop now","buttonUrl":true} --></mj-raw>
  </mj-column></mj-section>
</mj-body>`
	next := strings.Replace(src, "</mj-body>", block, 1)
	res := mjml.Compile(next)
	if len(res.Errors) > 0 {
		return fmt.Errorf("coupons: compile the email: %s", res.Errors[0].Message)
	}
	_, err = s.Tx.Exec(ctx, `UPDATE automation_email_versions v SET mjml = $2, html = $3 FROM automation_emails e
		WHERE e.id = $1 AND v.email_id = e.id AND v.version = e.latest_version`, id, next, res.HTML)
	return err
}
