package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/holdout"
	"github.com/hostrepute/reputecast/internal/store"
)

// The holdout group (M63 G-33, AudienceHoldout, SettingsSendingRules):
// Acme keeps a random 5% of its contacts out of campaigns and automations
// since 60 days ago (Fri 15 Aug on the boards' calendar). The Weekly
// Digests sent since went to the holdout group too (subscribers get the
// digest they signed up for, so "Campaigns that included the holdout: 8"),
// and so does #142 (seed_campaigns.go, seed_readiness.go). The Welcome
// series let the group pass its emails (67 · 41 · 26 held out).
//
// What the group did is measured from the demo's own orders, conversions
// and paid subscriptions, plus Acme Cloud's "Upgraded to Team plan"
// conversions of the period's first month: enough converters for a
// measurable answer, placed before the last 30 days the Earn and store
// screens count. Their number per group is solved from what the demo
// already holds, so the measurement says what the boards say: marketing
// added 18% more revenue per contact, with 96% confidence.

func init() { seedHoldout = seedHoldoutImpl }

// SkipHoldout leaves the holdout group out of the demo. apptest sets it:
// a 5% holdout would leave a random part of every test's audience out.
// The holdout tests add it to their own copy with SeedHoldout.
var SkipHoldout bool

// The demo's holdout group.
const (
	holdoutDemoSeed  int64 = 20260815
	holdoutDemoBP    int32 = 500
	holdoutDemoDays        = 60
	holdoutDemoValue int64 = 2900 // Acme Cloud's Team plan, in cents
	// The verdict the boards show.
	holdoutDemoLift = 18.0
)

func seedHoldoutImpl(ctx context.Context, s *Seeder) error {
	if SkipHoldout {
		return nil
	}
	// After every other seeder: the orders, conversions, digests and flows
	// it measures and marks are there by then.
	s.After(func(ctx context.Context, s *Seeder) error {
		return seedHoldoutGroup(ctx, s.Tx, s.WS().ID, s.Now)
	})
	return nil
}

// SeedHoldout adds the demo's holdout group to an already seeded workspace
// (the holdout tests' copy of the demo), as of now.
func SeedHoldout(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID, now time.Time) error {
	return seedHoldoutGroup(ctx, tx, workspaceID, now)
}

func seedHoldoutGroup(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, now time.Time) error {
	q := store.New(tx)
	if open, err := holdout.Open(ctx, q, wsID); err != nil || open != nil {
		return err
	}
	start := now.AddDate(0, 0, -holdoutDemoDays)
	var linh *uuid.UUID
	var linhName string
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT u.id, u.name FROM users u WHERE u.email = 'linh@acme.io'`).Scan(&id, &linhName); err == nil {
		linh = &id
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	period := uuid.Must(uuid.NewV7())
	if err := q.HoldoutInsertPeriod(ctx, store.HoldoutInsertPeriodParams{ID: period, WorkspaceID: wsID, Seed: holdoutDemoSeed, BasisPoints: holdoutDemoBP,
		Campaigns: true, Automations: true, StartedAt: start, StartedBy: linh, Reason: ""}); err != nil {
		return err
	}
	if linh != nil {
		after, _ := json.Marshal(map[string]any{"on": true, "basis_points": holdoutDemoBP, "campaigns": true, "automations": true, "period": period.String()})
		if _, err := tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
				result, ip, before, after, metadata)
			SELECT $1, w.id, $2, 'user', $3, $4, 'holdout.updated', 'workspace', w.id::text, w.name, 'allowed', '203.0.113.9'::inet, '{"on":false}', $5, '{}'
			FROM workspaces w WHERE w.id = $6`, ID(), start, *linh, linhName, after, wsID); err != nil {
			return err
		}
	}
	// The digests sent since went to everyone, the holdout group included.
	if _, err := tx.Exec(ctx, `UPDATE campaigns SET audience = audience || '{"include_holdout": true}'
		WHERE workspace_id = $1 AND name LIKE 'Weekly Digest #%' AND started_at >= $2 AND started_at < $3`, wsID, start, now); err != nil {
		return err
	}
	if err := holdoutWelcomeStats(ctx, tx, wsID); err != nil {
		return err
	}
	// Measured 6 hours ago ("Computed 6 h ago"), as the nightly job does.
	measured := now.Add(-6 * time.Hour)
	if err := holdoutConversions(ctx, tx, wsID, start, measured); err != nil {
		return err
	}
	svc := &holdout.Service{Q: q, Now: func() time.Time { return measured }}
	return svc.Measure(ctx, wsID)
}

// holdoutWelcomeStats: the Welcome series' email steps let the group pass
// (AutomationFlow: 67 · 41 · 26 held out), counted as skips too, as the
// engine counts them.
func holdoutWelcomeStats(ctx context.Context, tx pgx.Tx, wsID uuid.UUID) error {
	var flow uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM automations WHERE workspace_id = $1 AND name = 'Welcome series'`, wsID).Scan(&flow)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	// The draft on the AutomationFlow board: each email step with sends has
	// a second stat line, so the steps under it move down 21 px per line.
	var draft int32
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT v.version, v.graph FROM automation_versions v JOIN automations a ON a.id = v.automation_id
		WHERE a.id = $1 AND v.version = a.draft_version AND v.published_at IS NULL`, flow).Scan(&draft, &raw); err == nil {
		var g automations.AutomationGraph
		if err := json.Unmarshal(raw, &g); err != nil {
			return err
		}
		down := map[string]float64{"n4": 21, "n5": 21, "n6": 21, "n11": 21, "n12": 21, "n7": 42, "n8": 42, "n9": 42, "n10": 42, "n13": 42}
		for i := range g.Nodes {
			g.Nodes[i].Position.Y += down[g.Nodes[i].ID]
		}
		if _, err := tx.Exec(ctx, `UPDATE automation_versions SET graph = $3 WHERE automation_id = $1 AND version = $2`, flow, draft, g.MustJSON()); err != nil {
			return err
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	for node, total := range map[string]int64{"n3": 67, "n6": 41, "n12": 26} {
		rows, err := tx.Query(ctx, `SELECT version, day, sent FROM automation_step_stats WHERE automation_id = $1 AND node_id = $2 ORDER BY day, version`, flow, node)
		if err != nil {
			return err
		}
		type cell struct {
			version int32
			day     time.Time
		}
		var cells []cell
		var weights []float64
		for rows.Next() {
			var c cell
			var sent int64
			if err := rows.Scan(&c.version, &c.day, &sent); err != nil {
				rows.Close()
				return err
			}
			cells = append(cells, c)
			weights = append(weights, float64(sent)+0.01)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for i, n := range Split(total, weights) {
			if n == 0 {
				continue
			}
			if _, err := tx.Exec(ctx, `UPDATE automation_step_stats SET held_out = held_out + $5, skipped = skipped + $5
				WHERE automation_id = $1 AND version = $2 AND node_id = $3 AND day = $4`, flow, cells[i].version, node, cells[i].day, n); err != nil {
				return err
			}
		}
	}
	return nil
}

// holdoutOutcome is what one contact did in the period, as the measurement
// counts it.
type holdoutOutcome struct {
	id        uuid.UUID
	held      bool
	revenue   float64
	orders    float64
	converted bool
}

// holdoutConversions adds Acme Cloud's upgrades of the period's first
// month: one "Upgraded to Team plan" each for as many contacts of each
// group as make marketing's revenue per contact 18% higher than the
// holdout's with 96% confidence, given what the demo already holds up to
// the measurement (until).
func holdoutConversions(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, start, until time.Time) error {
	q := store.New(tx)
	currency, err := q.HoldoutCurrency(ctx, wsID)
	if err != nil {
		return err
	}
	// Acme Cloud's conversions key, as the demo's other conversions use it.
	var keyID *uuid.UUID
	keyName := "Acme Cloud · conversions"
	if err := tx.QueryRow(ctx, `SELECT api_key_id, key_name FROM conversions WHERE workspace_id = $1 AND NOT test ORDER BY occurred_at LIMIT 1`, wsID).Scan(&keyID, &keyName); err != nil &&
		!errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	rows, err := tx.Query(ctx, holdoutOutcomesSQL, wsID, holdoutDemoSeed, holdoutDemoBP, start, until, currency)
	if err != nil {
		return err
	}
	var all []holdoutOutcome
	for rows.Next() {
		var o holdoutOutcome
		if err := rows.Scan(&o.id, &o.held, &o.revenue, &o.orders, &o.converted); err != nil {
			rows.Close()
			return err
		}
		all = append(all, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var base [2]holdout.Group // 0: holdout, 1: marketing
	var free [2][]uuid.UUID   // who hasn't converted yet
	for _, o := range all {
		g := 1
		if o.held {
			g = 0
		}
		b := &base[g]
		b.Contacts++
		b.Revenue += o.revenue
		b.RevenueSq += o.revenue * o.revenue
		b.Orders += o.orders
		b.OrdersSq += o.orders * o.orders
		if o.converted {
			b.Converters++
		} else {
			free[g] = append(free[g], o.id)
		}
	}
	if base[0].Contacts < holdout.MinHoldout {
		return nil // a demo too small to measure
	}
	kh, km, ok := holdoutSolve(base, int64(len(free[0])), int64(len(free[1])))
	if !ok {
		return fmt.Errorf("holdout demo: no number of upgrades gives +%.0f%% at 96%% confidence", holdoutDemoLift)
	}
	rng := rand.New(rand.NewPCG(uint64(holdoutDemoSeed), 63)) //nolint:gosec // deterministic sample data
	for g := range free {
		rng.Shuffle(len(free[g]), func(i, j int) { free[g][i], free[g][j] = free[g][j], free[g][i] })
	}
	// The period's first month: before the last 30 days the Earn and store
	// screens count.
	from, to := start.Add(2*time.Hour), until.AddDate(0, 0, -31)
	span := to.Sub(from)
	var copyRows [][]any
	add := func(contacts []uuid.UUID) {
		for _, c := range contacts {
			n := len(copyRows)
			at := from.Add(time.Duration(rng.Int64N(int64(span))))
			copyRows = append(copyRows, []any{ID(), wsID, keyID, keyName, "Upgraded to Team plan", "29.00", holdoutDemoValue, currency, at, at, c, "email",
				json.RawMessage(`{"plan":"team"}`), fmt.Sprintf("upgrade-acct-h%05d", n), "", false})
		}
	}
	add(free[0][:kh])
	add(free[1][:km])
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"conversions"}, []string{"id", "workspace_id", "api_key_id", "key_name", "name", "value", "value_cents", "currency",
		"occurred_at", "received_at", "contact_id", "matched", "metadata", "idempotency_key", "request_hash", "test"}, pgx.CopyFromRows(copyRows))
	return err
}

// holdoutSolve finds how many upgrades each group gets: the fewest for the
// holdout group whose revenue per contact the marketing group beats by
// holdoutDemoLift percent with a confidence that rounds down to 96%.
func holdoutSolve(base [2]holdout.Group, freeH, freeM int64) (int64, int64, bool) {
	v := float64(holdoutDemoValue)
	with := func(g holdout.Group, k int64) holdout.Group {
		g.Converters += k
		g.Revenue += v * float64(k)
		g.RevenueSq += v * v * float64(k)
		return g
	}
	for kh := int64(1); kh <= freeH; kh++ {
		h := with(base[0], kh)
		// The marketing group's revenue for the lift, in whole upgrades.
		want := h.Revenue / float64(h.Contacts) * (1 + holdoutDemoLift/100) * float64(base[1].Contacts)
		km := int64(math.Round((want - base[1].Revenue) / v))
		if km < 0 || km > freeM {
			continue
		}
		m := holdout.Measured{Holdout: h, Marketing: with(base[1], km)}
		for _, x := range m.Metrics() {
			if x.Key != holdout.MetricRevenue {
				continue
			}
			if math.Round(x.Lift) != holdoutDemoLift {
				break
			}
			switch {
			case x.Confidence >= 0.9605 && x.Confidence < 0.9695:
				return kh, km, true
			case x.Confidence >= 0.9695:
				return 0, 0, false
			}
		}
	}
	return 0, 0, false
}

// holdoutOutcomesSQL is the measurement's population and outcomes
// (queries/holdout.sql, HoldoutMeasureGroups) per contact.
const holdoutOutcomesSQL = `
WITH pop AS (
    SELECT c.id, (rc_holdout_bucket(c.id, $2::bigint) < $3::int) AS held
    FROM contacts c
    WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.created_at < $5::timestamptz
      AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email AND s.created_at < $4::timestamptz)
), ord AS (
    SELECT o.contact_id, count(*) AS orders,
           coalesce(sum(o.total_cents - o.refunded_cents) FILTER (WHERE lower(o.currency) = $6::text), 0) AS cents
    FROM store_orders o
    WHERE o.workspace_id = $1 AND o.contact_id IS NOT NULL AND NOT o.test AND NOT o.cancelled
      AND o.placed_at >= $4::timestamptz AND o.placed_at < $5::timestamptz
    GROUP BY o.contact_id
), conv AS (
    SELECT v.contact_id, coalesce(sum(v.value_cents) FILTER (WHERE lower(v.currency) = $6::text), 0) AS cents
    FROM conversions v
    WHERE v.workspace_id = $1 AND v.contact_id IS NOT NULL AND NOT v.test
      AND v.occurred_at >= $4::timestamptz AND v.occurred_at < $5::timestamptz
    GROUP BY v.contact_id
), paid AS (
    SELECT DISTINCT p.contact_id FROM paid_subscriptions p
    WHERE p.workspace_id = $1 AND p.mode = 'live' AND p.kind = 'stripe' AND p.status <> 'incomplete'
      AND p.started_at >= $4::timestamptz AND p.started_at < $5::timestamptz
)
SELECT pop.id, pop.held,
       (coalesce(ord.cents, 0) + coalesce(conv.cents, 0))::float8,
       coalesce(ord.orders, 0)::float8,
       (ord.contact_id IS NOT NULL OR conv.contact_id IS NOT NULL OR paid.contact_id IS NOT NULL)
FROM pop
LEFT JOIN ord ON ord.contact_id = pop.id
LEFT JOIN conv ON conv.contact_id = pop.id
LEFT JOIN paid ON paid.contact_id = pop.id
ORDER BY pop.id`
