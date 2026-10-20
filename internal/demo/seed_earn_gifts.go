package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// giftsSeed: the gift page went live with the upgrade page on 18 Sep,
// selling Insider for 3 months ($24) and for a year ($80). In the last 30
// days 11 gifts sold for $656 (7 for a year, 4 for 3 months): 9
// delivered, 8 redeemed, 2 scheduled (brief 4, Gifts). The redeemed ones
// are Earn's gift access (seed_earn.go): Lena Fischer's from Priya
// Natarajan, Marisol Alvarez's from Diego Alvarez, and six more.
func (e *earnMore) giftsSeed(ctx context.Context) error {
	s := e.s
	live := s.earnDay(2026, 9, 18, 11, 30)
	three, year := ID(), ID()
	for _, o := range []struct {
		id     uuid.UUID
		months int
		amount int64
		pos    int
	}{{three, 3, 2400, 0}, {year, 12, 8000, 1}} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_gift_options (id, workspace_id, tier_id, months, amount_cents, currency, position, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 'usd', $6, $7, $8, $8)`, o.id, e.ws.ID, e.insider, o.months, o.amount, o.pos, e.linh, live); err != nil {
			return fmt.Errorf("gift option: %w", err)
		}
	}
	page, _ := json.Marshal(map[string]string{"headline": "Give the Weekly Digest Insider",
		"intro": "A members-only deep dive every Tuesday, the full archive and quarterly roadmap calls with the Acme team. You pay once; nothing renews."})
	if _, err := s.Tx.Exec(ctx, `UPDATE earn_settings SET gift_page_on = true, gift_page_at = $2, gift_page_by = $3, gift_link_months = 12, gift_page = $4
		WHERE workspace_id = $1`, e.ws.ID, live, e.linh, page); err != nil {
		return fmt.Errorf("gift page: %w", err)
	}
	// The gift access Earn seeded, by giver: who redeemed and when.
	type access struct {
		sub, contact     uuid.UUID
		email, name      string
		started, periodE time.Time
		months           int
	}
	rows, err := s.Tx.Query(ctx, `SELECT s.id, s.contact_id, c.email, trim(c.first_name || ' ' || c.last_name), s.started_at, s.current_period_end,
			s.source_detail->>'from', coalesce((s.source_detail->>'months')::int, 0)
		FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND s.kind = 'gift' AND s.mode = 'live' ORDER BY s.started_at`, e.ws.ID)
	if err != nil {
		return err
	}
	var redeemed []struct {
		from string
		a    access
	}
	for rows.Next() {
		var a access
		var from string
		if err := rows.Scan(&a.sub, &a.contact, &a.email, &a.name, &a.started, &a.periodE, &from, &a.months); err != nil {
			rows.Close()
			return err
		}
		redeemed = append(redeemed, struct {
			from string
			a    access
		}{from, a})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	buyers := map[string]string{"Omar Haddad": "omar.haddad@gmail.com", "Hannah Berg": "hannah@bergconsulting.de", "Kenji Watanabe": "kenji.w@icloud.com",
		"Priya Natarajan": "priya.n@gmail.com", "Diego Alvarez": "d.alvarez@yahoo.com", "Sam Rivera": "sam.rivera@outlook.com", "Tomasz Nowak": "t.nowak@gmail.com"}
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		chicago = s.earnLoc()
	}
	// at is a design day's time in a zone ("24 Dec, 08:00" Chicago time).
	at := func(m time.Month, d, hour, minute int, loc *time.Location) time.Time {
		day := s.earnDay(2026, m, d, 12, 0).In(s.earnLoc())
		return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
	}
	type gift struct {
		buyer, toName, toEmail, message string
		inPerson                        bool
		months                          int
		bought                          time.Time
		deliverAt                       *time.Time
		tz                              string
		status                          string
		code                            string
		redeemedBy                      string // the recipient's email, for the gift access it matches
		from                            string
	}
	karim := at(12, 24, 8, 0, chicago)
	yuki := at(11, 1, 9, 0, s.earnLoc())
	gifts := []gift{
		{buyer: "Omar Haddad", toName: "Karim Haddad", toEmail: "karim.haddad@example.com", message: "Merry Christmas, Karim! Now you'll know what I keep quoting at dinner.",
			months: 12, bought: s.earnDay(2026, 10, 14, 19, 12), deliverAt: &karim, tz: "America/Chicago", status: "paid", code: "K4RM-HD24-DEC8"},
		{buyer: "Hannah Berg", toEmail: "jonas.berg@example.de", months: 3, bought: s.earnDay(2026, 10, 13, 8, 40), tz: "Europe/Berlin", status: "delivered", code: "JN8B-3MTH-QX2P"},
		{buyer: "Kenji Watanabe", toEmail: "yuki.watanabe@example.jp", message: "For the train rides. Enjoy the deep dives!", months: 3, bought: s.earnDay(2026, 10, 12, 21, 5),
			deliverAt: &yuki, tz: "America/New_York", status: "paid", code: "YK9W-NV2K-TKY3"},
		{buyer: "Priya Natarajan", toName: "Lena Fischer", toEmail: "lena.fischer@hotmail.com",
			message: "Lena, you've been reading over my shoulder for months, so here's your own. Happy birthday!", months: 12, bought: s.earnDay(2026, 10, 11, 17, 40),
			tz: "America/New_York", status: "redeemed", code: "G7K2-M9QX-4TBH", redeemedBy: "lena.fischer@hotmail.com", from: "Priya Natarajan"},
		{buyer: "Diego Alvarez", inPerson: true, months: 3, bought: s.earnDay(2026, 10, 9, 18, 20), tz: "America/New_York", status: "redeemed", code: "DG3A-PV9Z-PRSN",
			redeemedBy: "m.alvarez@yahoo.com", from: "Diego Alvarez"},
	}
	// The six others, all redeemed, one per gift access left.
	used := map[uuid.UUID]bool{}
	for _, r := range redeemed {
		if r.a.email == "lena.fischer@hotmail.com" || r.a.email == "m.alvarez@yahoo.com" {
			used[r.a.sub] = true
		}
	}
	n := 0
	for _, r := range redeemed {
		if used[r.a.sub] || r.from == "" {
			continue
		}
		used[r.a.sub] = true
		n++
		gifts = append(gifts, gift{buyer: r.from, toName: r.a.name, toEmail: r.a.email, months: r.a.months, bought: r.a.started.Add(-2*time.Hour - 15*time.Minute),
			tz: "America/New_York", status: "redeemed", code: strings.ToUpper(fmt.Sprintf("RD%02d-%s-%s", n, r.a.sub.String()[24:28], r.a.sub.String()[28:32])), redeemedBy: r.a.email, from: r.from})
	}
	byEmail := map[string]access{}
	for _, r := range redeemed {
		byEmail[r.a.email] = r.a
	}
	for i, g := range gifts {
		id := ID()
		amount, option := int64(2400), three
		if g.months == 12 {
			amount, option = 8000, year
		}
		email := buyers[g.buyer]
		pi := fmt.Sprintf("pi_3Q%02dGift%sdemo", i+1, id.String()[26:])
		var deliveredAt, linkEnds, redeemedAt, receipt *time.Time
		var contact, sub *uuid.UUID
		receiptAt := g.bought.Add(40 * time.Second)
		receipt = &receiptAt
		switch g.status {
		case "delivered", "redeemed":
			d := g.bought.Add(30 * time.Second)
			deliveredAt = &d
			l := d.AddDate(1, 0, 0)
			linkEnds = &l
		}
		if g.status == "redeemed" {
			a, ok := byEmail[g.redeemedBy]
			if !ok {
				return fmt.Errorf("gift access of %s", g.redeemedBy)
			}
			r := a.started
			redeemedAt, contact, sub = &r, &a.contact, &a.sub
			if deliveredAt != nil && deliveredAt.After(r) {
				d := r.Add(-time.Hour)
				deliveredAt = &d
			}
		}
		code := g.code
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_gifts (id, workspace_id, mode, option_id, tier_id, months, amount_cents, currency, status, buyer_name, buyer_email,
				in_person, recipient_name, recipient_email, message, deliver_at, timezone, code, stripe_session_id, payment_intent, stripe_customer, paid_at, delivered_at,
				link_expires_at, redeemed_at, redeemed_email, redeemed_contact, subscription_id, receipt_sent_at, email_status, created_at, updated_at)
			VALUES ($1, $2, 'live', $3, $4, $5, $6, 'usd', $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27,
				$28, $29, $29)`,
			id, e.ws.ID, option, e.insider, g.months, amount, g.status, g.buyer, email, g.inPerson, g.toName, g.toEmail, g.message, g.deliverAt, g.tz, code,
			"cs_live_gift"+id.String()[24:], pi, "cus_gift"+id.String()[26:], g.bought, deliveredAt, linkEnds, redeemedAt, g.redeemedBy, contact, sub, receipt,
			map[bool]string{true: "sent", false: ""}[deliveredAt != nil], g.bought.Add(-3*time.Minute)); err != nil {
			return fmt.Errorf("gift from %s: %w", g.buyer, err)
		}
		events := []struct {
			kind string
			at   time.Time
			data map[string]any
		}{{"paid", g.bought, map[string]any{"amountCents": amount, "currency": "usd"}},
			{"receipt", receiptAt, map[string]any{"email": "gift_purchased", "status": "sent", "template": "gift-purchased", "version": 1}}}
		if deliveredAt != nil {
			to := "recipient"
			if g.inPerson {
				to = "buyer"
			}
			events = append(events, struct {
				kind string
				at   time.Time
				data map[string]any
			}{"delivered", *deliveredAt, map[string]any{"email": "gift_received", "status": "sent", "template": "gift-received", "version": 1, "to": to}})
		}
		if redeemedAt != nil {
			events = append(events, struct {
				kind string
				at   time.Time
				data map[string]any
			}{"redeemed", *redeemedAt, map[string]any{"email": g.redeemedBy}})
		}
		for _, ev := range events {
			b, _ := json.Marshal(ev.data)
			if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_gift_events (id, gift_id, kind, occurred_at, data) VALUES ($1, $2, $3, $4, $5)`, ID(), id, ev.kind, ev.at, b); err != nil {
				return err
			}
		}
	}
	return nil
}
