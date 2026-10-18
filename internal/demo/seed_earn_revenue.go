package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
)

// The revenue side of the Earn sample (brief 4): the offers, the cancel
// page's visits, the MRR ledger's history and the conversions.

// offersSeed adds FALL26 (signup, 20% off the first year of Insider
// yearly, 100 codes until 31 Oct), "Stay for half price" (retention, 50%
// off Insider monthly for 3 months) and the referral reward of Grow's
// 10-referral milestone with its two codes.
func (e *earnMore) offersSeed(ctx context.Context) error {
	s := e.s
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM earn_tiers WHERE workspace_id = $1 AND name = 'Weekly Digest Insider'`, e.ws.ID).Scan(&e.insider); err != nil {
		return fmt.Errorf("insider tier: %w", err)
	}
	e.fall26, e.stay, e.referral = ID(), ID(), ID()
	applies := func(period string) []byte {
		b, _ := json.Marshal([]map[string]string{{"tier": e.insider.String(), "period": period}})
		return b
	}
	stripe := func(coupon, promo string) []byte {
		live := map[string]any{"coupon": coupon, "coupons": []string{coupon}}
		if promo != "" {
			live["promo"] = promo
		}
		b, _ := json.Marshal(map[string]any{"live": live})
		return b
	}
	ends := s.earnDay(2026, 10, 31, 23, 59)
	for _, o := range []struct {
		id                     uuid.UUID
		name, kind, duration   string
		percent, months, limit int
		applies, stripe        []byte
		code                   string
		ends                   *time.Time
		newOnly                bool
		minMonths              int
		started                time.Time
	}{
		{e.fall26, "FALL26", "signup", "once", 20, 0, 100, applies("year"), stripe("FALL26-20", "promo_1Q9bX2rTFALL26"), "FALL26", &ends, true, 0, s.earnDay(2026, 10, 1, 9, 0)},
		{e.stay, "Stay for half price", "retention", "months", 50, 3, 0, applies("month"), stripe("STAY50", ""), "", nil, false, 2, s.earnDay(2026, 9, 18, 14, 40)},
		{e.referral, "Referral reward · 3 months of Insider free", "referral", "months", 100, 3, 0, applies("month"), stripe("REFER-3M", ""), "", nil, true, 0,
			s.earnDay(2026, 9, 18, 14, 50)},
	} {
		var months, limit *int
		if o.months > 0 {
			months = &o.months
		}
		if o.limit > 0 {
			limit = &o.limit
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_offers (id, workspace_id, name, kind, discount_type, percent_off, currency, duration, duration_months, applies,
				by_code, code, by_link, redeem_limit, ends_at, new_only, min_paid_months, status, stripe, sync_status, started_at, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'percent', $5, 'usd', $6, $7, $8, $9, $10, $9, $11, $12, $13, $14, 'live', $15, 'synced', $16, $17, $16, $16)`,
			o.id, e.ws.ID, o.name, o.kind, o.percent, o.duration, months, o.applies, o.code != "", o.code, limit, o.ends, o.newOnly, o.minMonths, o.stripe, o.started, e.marcus); err != nil {
			return fmt.Errorf("offer %s: %w", o.name, err)
		}
	}
	// Grow issued two referral reward codes; neither is redeemed yet.
	var referrers []uuid.UUID
	rows, err := s.Tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL ORDER BY email LIMIT 2 OFFSET 40`, e.ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		referrers = append(referrers, id)
	}
	rows.Close()
	for i, c := range []struct {
		code string
		at   time.Time
	}{{"INSIDER-7KQ2", s.earnDay(2026, 10, 3, 18, 12)}, {"INSIDER-M4XD", s.earnDay(2026, 10, 10, 8, 47)}} {
		var contact *uuid.UUID
		if i < len(referrers) {
			contact = &referrers[i]
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_offer_codes (id, workspace_id, offer_id, code, contact_id, issued_to, issued_at, expires_at, stripe, source)
			VALUES ($1, $2, $3, $4, $5, 'referral milestone · 10 referrals', $6, $7, $8, 'referral')`,
			ID(), e.ws.ID, e.referral, c.code, contact, c.at, c.at.AddDate(0, 0, 60), []byte(`{"live":{"promo":"promo_`+c.code+`"}}`)); err != nil {
			return err
		}
	}
	return nil
}

// cancelVisitsSeed: 43 visits to the cancel page in 30 days. 12 kept
// Insider with "Stay for half price", 26 went on to cancel and 5 left
// without deciding; the offer was shown 34 times. Reasons: not reading
// enough 14, too expensive 11, not for me 6, switching 3, other 4, no
// answer 5.
func (e *earnMore) cancelVisitsSeed(ctx context.Context) error {
	s := e.s
	rng := rand.New(rand.NewPCG(4317, 1252)) //nolint:gosec // deterministic sample data
	type sub struct {
		id, contact uuid.UUID
		reason      string
		at          time.Time
		monthly     bool
	}
	read := func(sql string, args ...any) ([]sub, error) {
		rows, err := s.Tx.Query(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []sub
		for rows.Next() {
			var x sub
			var at *time.Time
			var period string
			if err := rows.Scan(&x.id, &x.contact, &x.reason, &at, &period); err != nil {
				return nil, err
			}
			if at != nil {
				x.at = *at
			}
			x.monthly = period == "month"
			out = append(out, x)
		}
		return out, rows.Err()
	}
	kept, err := read(`SELECT id, contact_id, '', NULL::timestamptz, period FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'live'
		AND discount->>'label' = 'Stay for half price' ORDER BY id`, e.ws.ID)
	if err != nil {
		return err
	}
	// Olga Bennett canceled on the cancel page on 12 Oct (her access ends
	// on 2 Nov); the others' access has ended. Marta Kowalski's was
	// canceled by Marcus Chen, not on the cancel page.
	canceled, err := read(`SELECT s.id, s.contact_id, s.cancel_reason, coalesce(s.canceled_at, s.ended_at), s.period FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND s.mode = 'live' AND s.kind = 'stripe' AND s.canceled_by IS NULL
		  AND ((s.status = 'canceled' AND s.end_reason = 'canceled') OR c.email = 'olga.bennett@gmail.com')
		ORDER BY (c.email = 'olga.bennett@gmail.com') DESC, s.ended_at DESC LIMIT 26`, e.ws.ID)
	if err != nil {
		return err
	}
	undecided, err := read(`SELECT id, contact_id, '', NULL::timestamptz, period FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'live'
		AND status = 'active' AND period = 'month' AND discount IS NULL AND cancel_at IS NULL ORDER BY id LIMIT 5 OFFSET 7`, e.ws.ID)
	if err != nil {
		return err
	}
	need := map[string]int{"not_reading": 14, "too_expensive": 11, "not_for_me": 6, "switching": 3, "other": 4, "": 5}
	order := []string{"not_reading", "too_expensive", "not_for_me", "switching", "other", ""}
	take := func(want string) string {
		if need[want] > 0 {
			need[want]--
			return want
		}
		for _, r := range order {
			if need[r] > 0 {
				need[r]--
				return r
			}
		}
		return ""
	}
	start := s.earnDay(2026, 9, 16, 9, 0)
	span := s.earnDay(2026, 10, 15, 9, 0).Sub(start)
	when := func() time.Time { return start.Add(time.Duration(rng.Int64N(int64(span)))) }
	shown := 0
	visit := func(x sub, decision, reason string, at time.Time, offer bool) error {
		var offerID *uuid.UUID
		if offer {
			offerID = &e.stay
			shown++
		}
		var decided *time.Time
		if decision != "" {
			d := at.Add(time.Duration(30+rng.IntN(240)) * time.Second)
			decided = &d
		}
		_, err := s.Tx.Exec(ctx, `INSERT INTO earn_cancel_visits (id, workspace_id, contact_id, subscription_id, mode, visited_at, offer_id, reason, decision, decided_at)
			VALUES ($1, $2, $3, $4, 'live', $5, $6, $7, $8, $9)`, ID(), e.ws.ID, x.contact, x.id, at, offerID, reason, decision, decided)
		return err
	}
	// Continued first, so their stored reasons count.
	for _, x := range canceled {
		at := x.at
		if at.IsZero() || at.Before(start) {
			at = when()
		}
		at = at.Add(-time.Duration(2+rng.IntN(20)) * time.Minute)
		if err := visit(x, "continued", take(x.reason), at, x.monthly && shown < 22); err != nil {
			return err
		}
	}
	for i, x := range kept {
		at := when()
		if i == 0 {
			// Ben Okoro took it on 12 Oct.
			at = s.earnDay(2026, 10, 12, 19, 4)
		}
		if err := visit(x, "kept", take(""), at, true); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_offer_uses (id, workspace_id, offer_id, contact_id, subscription_id, mode, used_at, source)
			VALUES ($1, $2, $3, $4, $5, 'live', $6, 'cancel_page')`, ID(), e.ws.ID, e.stay, x.contact, x.id, at.Add(time.Minute)); err != nil {
			return err
		}
		e.keptAt = append(e.keptAt, keptOffer{sub: x.id, at: at.Add(time.Minute)})
	}
	for _, x := range undecided {
		if err := visit(x, "", take(""), when(), shown < 34); err != nil {
			return err
		}
	}
	return nil
}

// keptOffer is a subscription that took the retention offer, and when.
type keptOffer struct {
	sub uuid.UUID
	at  time.Time
}

// ledgerSeed gives the MRR ledger the 30 days of the sample: the 31 paid
// members who churned, the 12 who took the retention offer, 9 who moved
// from monthly to yearly and 3 who moved up to Founding member. Their
// earlier rows are what they paid before the change, so today's MRR stays
// $8,960 and the start of the period is $8,542.
func (e *earnMore) ledgerSeed(ctx context.Context) error {
	s := e.s
	rng := rand.New(rand.NewPCG(8542, 8960)) //nolint:gosec // deterministic sample data
	start := s.earnDay(2026, 9, 16, 9, 0)
	span := s.earnDay(2026, 10, 15, 9, 0).Sub(start)
	when := func() time.Time { return start.Add(time.Duration(rng.Int64N(int64(span)))) }
	// The churned: what they paid from when they started, until access ended.
	if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_mrr_changes (workspace_id, subscription_id, contact_id, tier_id, period, at, kind, arr_before, arr_after, source)
		SELECT workspace_id, id, contact_id, tier_id, period, least(started_at, created_at), 'new', 0,
		       CASE WHEN period = 'year' THEN amount_cents ELSE amount_cents * 12 END, CASE WHEN import_id IS NOT NULL THEN 'import' ELSE 'event' END
		FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'live' AND kind = 'stripe' AND status = 'canceled' AND end_reason = 'canceled'`, e.ws.ID); err != nil {
		return fmt.Errorf("churned history: %w", err)
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_mrr_changes (workspace_id, subscription_id, contact_id, tier_id, period, at, kind, arr_before, arr_after, source)
		SELECT workspace_id, id, contact_id, tier_id, period, coalesce(ended_at, updated_at), 'churn', CASE WHEN period = 'year' THEN amount_cents ELSE amount_cents * 12 END, 0, 'event'
		FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'live' AND kind = 'stripe' AND status = 'canceled' AND end_reason = 'canceled'`, e.ws.ID); err != nil {
		return fmt.Errorf("churn: %w", err)
	}
	// The retention offer: $8 until they took it, $4 since.
	for _, k := range e.keptAt {
		if _, err := s.Tx.Exec(ctx, `UPDATE earn_mrr_changes SET arr_after = 9600 WHERE subscription_id = $1 AND kind = 'new'`, k.sub); err != nil {
			return err
		}
		if err := e.ledgerChange(ctx, k.sub, k.at, "contraction", 9600, 4800, ""); err != nil {
			return err
		}
	}
	// Nine paid members moved from monthly to yearly, three from Insider to
	// Founding member (before the period they paid $8 a month).
	type move struct {
		sql, kind, period string
		after             int64
	}
	for _, m := range []move{
		{`SELECT id FROM paid_subscriptions s WHERE workspace_id = $1 AND mode = 'live' AND kind = 'stripe' AND status = 'active' AND period = 'year' AND discount IS NULL
			AND tier_id = $2 AND started_at < $3 AND EXISTS (SELECT 1 FROM earn_mrr_changes c WHERE c.subscription_id = s.id AND c.kind = 'new') ORDER BY id LIMIT 9`, "contraction", "month", 8000},
		{`SELECT id FROM paid_subscriptions s WHERE workspace_id = $1 AND mode = 'live' AND kind = 'stripe' AND status = 'active' AND period = 'year' AND discount IS NULL
			AND tier_id <> $2 AND started_at < $3 AND EXISTS (SELECT 1 FROM earn_mrr_changes c WHERE c.subscription_id = s.id AND c.kind = 'new') ORDER BY id LIMIT 3`, "expansion", "month", 20000},
	} {
		rows, err := s.Tx.Query(ctx, m.sql, e.ws.ID, e.insider, start)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			if _, err := s.Tx.Exec(ctx, `UPDATE earn_mrr_changes SET arr_after = 9600, period = 'month', tier_id = $2 WHERE subscription_id = $1 AND kind = 'new'`, id, e.insider); err != nil {
				return err
			}
			if err := e.ledgerChange(ctx, id, when(), m.kind, 9600, m.after, "year"); err != nil {
				return err
			}
		}
	}
	return nil
}

// sourcesSeed credits the 30 days' 96 new paid members to the emails of
// the sample: #139 19 ($140), #140 16 ($116), #141 9 ($82), Welcome to
// news.acme.io 11 ($88), the Welcome series 12 ($72) and no email click 29
// ($194). Sam Rivera stays #141's.
func (e *earnMore) sourcesSeed(ctx context.Context) error {
	s := e.s
	rows, err := s.Tx.Query(ctx, `SELECT s.id, CASE WHEN t.access_rank > 1 THEN 'founding' WHEN s.discount->>'label' = 'FALL26' THEN 'fall26' WHEN s.period = 'year' THEN 'yearly' ELSE 'monthly' END
		FROM paid_subscriptions s JOIN earn_tiers t ON t.id = s.tier_id JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND s.mode = 'live' AND EXISTS (SELECT 1 FROM earn_mrr_changes m WHERE m.subscription_id = s.id AND m.kind = 'new' AND m.at >= $2)
		ORDER BY (c.email = 'sam.rivera@outlook.com') DESC, s.started_at DESC, s.id`, e.ws.ID, s.earnDay(2026, 9, 15, 10, 0))
	if err != nil {
		return err
	}
	byType := map[string][]uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		var typ string
		if err := rows.Scan(&id, &typ); err != nil {
			rows.Close()
			return err
		}
		byType[typ] = append(byType[typ], id)
	}
	rows.Close()
	take := func(typ string, n int) []uuid.UUID {
		n = min(n, len(byType[typ]))
		out := byType[typ][:n]
		byType[typ] = byType[typ][n:]
		return out
	}
	for _, g := range []struct {
		name, kind                        string
		monthly, yearly, fall26, founding int
	}{
		{"Weekly Digest #141", "campaign", 0, 0, 6, 3},
		{"Weekly Digest #139", "campaign", 14, 1, 4, 0},
		{"Weekly Digest #140", "campaign", 10, 3, 3, 0},
		{"Welcome to news.acme.io", "campaign", 11, 0, 0, 0},
		{"Welcome series", "automation", 3, 0, 9, 0},
	} {
		var ids []uuid.UUID
		ids = append(ids, take("monthly", g.monthly)...)
		ids = append(ids, take("yearly", g.yearly)...)
		ids = append(ids, take("fall26", g.fall26)...)
		ids = append(ids, take("founding", g.founding)...)
		var camp *uuid.UUID
		if g.kind == "campaign" {
			if id, ok := s.Campaigns[g.name]; ok {
				camp = &id
			}
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE paid_subscriptions SET source_kind = $2, source_campaign_id = $3, source_label = $4 WHERE id = ANY($1)`, ids, g.kind, camp, g.name); err != nil {
			return err
		}
	}
	// The rest came through the FALL26 link or the upgrade page itself.
	if _, err := s.Tx.Exec(ctx, `UPDATE paid_subscriptions SET source_kind = 'offer', source_campaign_id = NULL, source_label = 'FALL26 link' WHERE id = ANY($1)`,
		byType["fall26"]); err != nil {
		return err
	}
	var rest []uuid.UUID
	for _, typ := range []string{"monthly", "yearly", "founding"} {
		rest = append(rest, byType[typ]...)
	}
	_, err = s.Tx.Exec(ctx, `UPDATE paid_subscriptions SET source_kind = 'page', source_campaign_id = NULL, source_label = 'Upgrade page' WHERE id = ANY($1)`, rest)
	return err
}

// checkoutsSeed credits the upgrade page's checkouts to the emails readers
// clicked first: Sam Rivera's and 57 more to Weekly Digest #141 (58
// started), and to Welcome to news.acme.io, #139 and #140 twice as many as
// their paid members (as many abandoned as completed), from the week after
// each was sent. Only the credit changes, so the upgrade page's numbers
// stay as they are.
func (e *earnMore) checkoutsSeed(ctx context.Context) error {
	s := e.s
	for _, name := range []string{"Welcome to news.acme.io", "Weekly Digest #139", "Weekly Digest #140", "Weekly Digest #141"} {
		id, ok := s.Campaigns[name]
		if !ok {
			continue
		}
		var sent *time.Time
		var credited int
		if err := s.Tx.QueryRow(ctx, `SELECT c.started_at, (SELECT count(*) FROM paid_subscriptions p WHERE p.source_campaign_id = c.id AND p.mode = 'live')
			FROM campaigns c WHERE c.id = $1`, id).Scan(&sent, &credited); err != nil {
			return err
		}
		if sent == nil {
			continue
		}
		attr := mustJSON(map[string]any{"kind": "campaign", "campaign_id": id.String(), "label": name})
		tag, err := s.Tx.Exec(ctx, `UPDATE earn_checkouts SET attribution = $2 WHERE workspace_id = $1 AND attribution->>'label' = $3`, e.ws.ID, attr, name)
		if err != nil {
			return err
		}
		want := 2*credited - int(tag.RowsAffected())
		if name == "Weekly Digest #141" {
			want = 58 - int(tag.RowsAffected())
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE earn_checkouts SET attribution = $2 WHERE id IN (
			SELECT id FROM earn_checkouts WHERE workspace_id = $1 AND mode = 'live' AND contact_id IS NULL AND attribution = '{}'
			  AND created_at >= $3 AND created_at < $3 + interval '7 days' ORDER BY created_at LIMIT $4)`, e.ws.ID, attr, *sent, max(want, 0)); err != nil {
			return err
		}
	}
	return nil
}

// ledgerChange records one change of a subscription's value.
func (e *earnMore) ledgerChange(ctx context.Context, sub uuid.UUID, at time.Time, kind string, before, after int64, period string) error {
	_, err := e.s.Tx.Exec(ctx, `INSERT INTO earn_mrr_changes (workspace_id, subscription_id, contact_id, tier_id, period, at, kind, arr_before, arr_after, source)
		SELECT workspace_id, id, contact_id, tier_id, CASE WHEN $5 = '' THEN period ELSE $5 END, $2, $3, $4, $6, 'event' FROM paid_subscriptions WHERE id = $1`,
		sub, at, kind, before, period, after)
	return err
}

// conversionsSeed: the "Acme Cloud · conversions" key Marcus Chen made on 6
// Oct, the click token on app.acme.io, and 64 "Upgraded to Team plan"
// conversions at $29 since: 23 credited to October Product Update (19 by
// click token, 4 by email) and 41 not credited.
func (e *earnMore) conversionsSeed(ctx context.Context) error {
	s := e.s
	keyID := ID()
	_, hash := demoKey("rc_live_", "3f8a", "c21d")
	if _, err := s.Tx.Exec(ctx, `INSERT INTO api_keys (id, workspace_id, kind, name, prefix, token_hash, permissions, scope, created_by, created_at,
			last_used_at, last_used_ip, last_used_client, last_call, mode, rate_per_second)
		VALUES ($1, $2, 'api_key', 'Acme Cloud · conversions', 'rc_live_3f8a…c21d', $3, $4, '{}', $5, $6, $7, $8, 'acme-cloud/2.14 (node)', 'POST /conversions', 'live', 50)`,
		keyID, e.ws.ID, hash, []string{string(authz.ConversionsCreate)}, e.marcus, s.earnDay(2026, 10, 6, 16, 10), s.Ago(12*time.Minute), addr("198.51.100.24")); err != nil {
		return fmt.Errorf("conversions key: %w", err)
	}
	s.Extra["key:Acme Cloud · conversions"] = keyID
	if _, err := s.Tx.Exec(ctx, `UPDATE earn_settings SET click_token_on = true, click_token_domains = '{app.acme.io}', attribution_days = 7 WHERE workspace_id = $1`,
		e.ws.ID); err != nil {
		return err
	}
	campaign := s.Campaigns["October Product Update"]
	var contacts []uuid.UUID
	rows, err := s.Tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL AND email LIKE '%@%' ORDER BY id LIMIT 64 OFFSET 300`, e.ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		contacts = append(contacts, id)
	}
	rows.Close()
	if len(contacts) < 64 {
		return nil
	}
	rng := rand.New(rand.NewPCG(64, 1856)) //nolint:gosec // deterministic sample data
	first := s.earnDay(2026, 10, 6, 17, 30)
	span := s.Ago(2 * time.Hour).Sub(first)
	type conv struct {
		at      time.Time
		matched string
		credit  bool
	}
	// The ones the log shows first, newest first.
	list := []conv{
		{s.earnDay(2026, 10, 15, 9, 48), "click_token", true},
		{s.earnDay(2026, 10, 15, 9, 20), "email", false},
		{s.earnDay(2026, 10, 14, 17, 2), "email", true},
		{s.earnDay(2026, 10, 14, 11, 40), "click_token", true},
		{s.earnDay(2026, 10, 13, 15, 26), "email", false},
	}
	byToken, byEmail := 17, 3
	for len(list) < 64 {
		c := conv{at: first.Add(time.Duration(rng.Int64N(int64(span))))}
		switch {
		case byToken > 0:
			c.matched, c.credit = "click_token", true
			byToken--
		case byEmail > 0:
			c.matched, c.credit = "email", true
			byEmail--
		default:
			c.matched = "email"
		}
		list = append(list, c)
	}
	for i, c := range list {
		at := c.at
		if at.After(s.Now) {
			at = s.Ago(time.Duration(i+1) * time.Minute)
		}
		var camp *uuid.UUID
		var clicked *time.Time
		if c.credit && campaign != uuid.Nil {
			camp = &campaign
			ck := at.Add(-time.Duration(10+rng.IntN(3000)) * time.Minute)
			clicked = &ck
		}
		contact := contacts[i]
		if _, err := s.Tx.Exec(ctx, `INSERT INTO conversions (id, workspace_id, api_key_id, key_name, name, value, value_cents, currency, occurred_at, received_at,
				contact_id, matched, campaign_id, clicked_at, metadata, idempotency_key, request_hash, test)
			VALUES ($1, $2, $3, 'Acme Cloud · conversions', 'Upgraded to Team plan', '29.00', 2900, 'usd', $4, $4, $5, $6, $7, $8, '{"plan":"team"}', $9, '', false)`,
			ID(), e.ws.ID, keyID, at, contact, c.matched, camp, clicked, fmt.Sprintf("upgrade-acct-%04d", 2100+i)); err != nil {
			return fmt.Errorf("conversion %d: %w", i, err)
		}
	}
	// Weekly Digest #141's upgrade links: 386 human clicks.
	if id, ok := s.Campaigns["Weekly Digest #141"]; ok {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_campaign_clicks (campaign_id, workspace_id, upgrade_clicks) VALUES ($1, $2, 386)
			ON CONFLICT (campaign_id) DO UPDATE SET upgrade_clicks = 386`, id, e.ws.ID); err != nil {
			return err
		}
	}
	return nil
}
