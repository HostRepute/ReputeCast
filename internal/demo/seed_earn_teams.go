package demo

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/emailaddr"
)

// Teams and pauses (M65): Insider's group prices ($6 a seat a month, $60 a
// year, 3 to 50 seats), the team Acme Inc that Priya Natarajan bought on 3
// Sep (10 seats: her own, seven people who accepted and Mei Chen's invite
// from 13 Oct), and Chloé Martin's Insider monthly, paused by Marcus Chen
// on 12 Oct until 12 Jan 2027. The cancel page offers pauses of 1 and 3
// months.

// teamPerson is a person of the demo's team.
type teamPerson struct {
	first, last, email string
	accepted           time.Time
}

// teamsAndPauses adds the group prices, the team's buyer and the paused
// member; their rows are written with the others (write). The seats come
// once the buyer's subscription is in place (teamSeats).
func (e *earnSeed) teamsAndPauses(ctx context.Context) error {
	s := e.s
	created := s.earnDay(2026, 9, 2, 16, 40)
	e.gMonth, e.gYear = ID(), ID()
	for _, p := range []struct {
		id         uuid.UUID
		period     string
		amount     int64
		live, test string
	}{
		{e.gMonth, "month", 600, "price_1Q9tA1eF", "price_test_1Q9tT1eF"},
		{e.gYear, "year", 6000, "price_1Q9tA4gH", "price_test_1Q9tT4gH"},
	} {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_prices (id, workspace_id, tier_id, period, amount_cents, currency, status, stripe_live_id, stripe_test_id, origin,
			created_at, audience, seats_min, seats_max)
			VALUES ($1, $2, $3, $4, $5, 'usd', 'current', $6, $7, 'tier', $8, 'group', 3, 50)`, p.id, e.ws.ID, e.insider, p.period, p.amount, p.live, p.test, created); err != nil {
			return fmt.Errorf("group price: %w", err)
		}
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE earn_settings SET pause_months = '{1,3}' WHERE workspace_id = $1`, e.ws.ID); err != nil {
		return fmt.Errorf("pause months: %w", err)
	}
	// The team's buyer, and the people who took a seat: Weekly Digest
	// subscribers stand for them, as for the other sample members.
	e.teamBuyer = e.stand(ctx, teamPerson{first: "Priya", last: "Natarajan", email: "priya@acme-inc.example"})
	e.teamPeople = []teamPerson{
		{"Jonas", "Weber", "jonas@acme-inc.example", s.earnDay(2026, 9, 4, 9, 10)},
		{"Fatima", "Zahra", "fatima@acme-inc.example", s.earnDay(2026, 9, 4, 11, 32)},
		{"Noah", "Kim", "noah@acme-inc.example", s.earnDay(2026, 9, 5, 8, 50)},
		{"Sara", "Okonkwo", "sara@acme-inc.example", s.earnDay(2026, 9, 8, 14, 3)},
		{"Ethan", "Brooks", "ethan@acme-inc.example", s.earnDay(2026, 9, 12, 10, 41)},
		{"Lucas", "Moreau", "lucas@acme-inc.example", s.earnDay(2026, 9, 22, 16, 15)},
		{"Yuki", "Sato", "yuki@acme-inc.example", s.earnDay(2026, 10, 12, 9, 2)},
	}
	e.teamContacts = make([]earnContact, 0, len(e.teamPeople))
	for _, p := range e.teamPeople {
		e.teamContacts = append(e.teamContacts, e.stand(ctx, p))
	}
	start := s.earnDay(2026, 9, 3, 10, 14)
	gy := e.gYear
	buyer := e.add(earnRow{key: "team", c: e.teamBuyer, kind: "stripe", tier: e.insider, price: &gy, stripePr: "price_1Q9tA4gH", status: "active", period: "year",
		amount: 6000, quantity: 10, itemID: "si_R4tLq2MzW8", started: start, periodEnd: tp(start.AddDate(1, 0, 0)), paid: 60000, srcKind: "page",
		srcLabel: "Upgrade page · team", srcDetail: map[string]any{"team": "Acme Inc"}, emailed: []string{earn.EmailGroupWelcome}, updated: s.earnDay(2026, 10, 13, 15, 40)})
	e.event(buyer, e.teamBuyer, "", "checkout_completed", start, nil, nil, map[string]any{"tier": "Weekly Digest Insider", "period": "year", "team": "Acme Inc", "seats": 10,
		"lists": "joined"}, nil)
	e.event(buyer, e.teamBuyer, "", "paid", start.Add(20*time.Second), i64(60000), nil, map[string]any{"tier": "Weekly Digest Insider", "period": "year", "seats": 10}, nil)
	welcome := e.memberMessage(ctx, e.teamBuyer, earn.EmailGroupWelcome, 1, "Your team's Insider is ready", start.Add(time.Minute))
	e.event(buyer, e.teamBuyer, "", "email_sent", start.Add(time.Minute), nil, nil, map[string]any{"email": earn.EmailGroupWelcome, "template": "group-welcome", "version": 1,
		"status": "sent"}, welcome)

	// Chloé Martin: Insider monthly since 4 Apr, from Substack; paused on
	// 12 Oct until 12 Jan 2027. Stripe's next invoice after the pause is 4
	// Nov's.
	chloe := e.stand(ctx, teamPerson{first: "Chloé", last: "Martin", email: "chloe.martin@gmail.com"})
	e.contacts["paused"] = chloe
	pm := e.pMonth
	pausedAt, until := s.earnDay(2026, 10, 12, 10, 20), s.earnDay(2027, 1, 12, 10, 20)
	paused := e.add(earnRow{key: "paused", c: chloe, kind: "stripe", tier: e.insider, price: &pm, stripePr: "price_1Q8mL2aB", status: "active", period: "month",
		amount: 800, started: time.Date(2026, 4, 4, 9, 30, 0, 0, s.earnLoc()), periodEnd: tp(s.earnDay(2026, 11, 4, 9, 30)), paid: 4800, srcKind: "import",
		srcLabel: "Imported from Substack", imported: true, pausedAt: &pausedAt, pauseResumes: &until, emailed: []string{earn.EmailPaused}, updated: pausedAt})
	e.event(paused, chloe, "", "paused", pausedAt, nil, e.marcus, map[string]any{"until": until.UTC().Format(time.RFC3339), "source": "team"}, nil)
	msg := e.memberMessage(ctx, chloe, earn.EmailPaused, 1, "Your Insider membership is paused until 12 Jan 2027", pausedAt.Add(30*time.Second))
	e.event(paused, chloe, "", "email_sent", pausedAt.Add(30*time.Second), nil, nil, map[string]any{"email": earn.EmailPaused, "template": "member-paused", "version": 1,
		"status": "sent"}, msg)
	return nil
}

// stand makes a Weekly Digest subscriber stand for a person of the sample
// (their address and name), as people does.
func (e *earnSeed) stand(ctx context.Context, p teamPerson) earnContact {
	c := e.draw()
	if _, err := e.s.Tx.Exec(ctx, `UPDATE contacts SET email = $2, first_name = $3, last_name = $4, role_address = $5, mailbox_provider = $6 WHERE id = $1`,
		c.id, p.email, p.first, p.last, emailaddr.IsRole(p.email), emailaddr.MailboxProvider(emailaddr.Domain(p.email))); err != nil {
		e.err = fmt.Errorf("contact %s: %w", p.email, err)
	}
	return earnContact{id: c.id, email: p.email, first: p.first, last: p.last}
}

// teamSeats adds the team, its seven seats with their access and Mei
// Chen's invite, once the buyer's subscription is written.
func (e *earnSeed) teamSeats(ctx context.Context) error {
	if e.err != nil {
		return e.err
	}
	s := e.s
	buyer := e.byKey["team"]
	group := ID()
	start := s.earnDay(2026, 9, 3, 10, 14)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_groups (id, workspace_id, subscription_id, owner_contact_id, name, seats, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'Acme Inc', 10, $5, $6)`, group, e.ws.ID, buyer, e.teamBuyer.id, start, s.earnDay(2026, 10, 13, 15, 40)); err != nil {
		return fmt.Errorf("team: %w", err)
	}
	ins := e.insider
	for i, p := range e.teamPeople {
		c := e.teamContacts[i]
		sub, seat := ID(), ID()
		invited := start.Add(time.Duration(10+i) * time.Minute)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO paid_subscriptions (id, workspace_id, contact_id, mode, kind, tier_id, status, period, amount_cents, currency, started_at,
			source_kind, source_label, source_detail, group_id, created_at, updated_at, automation_started_at)
			VALUES ($1, $2, $3, 'live', 'group', $4, 'group', '', 0, 'usd', $5, 'group', 'Invited by Priya Natarajan', $6, $7, $5, $5, $5)`,
			sub, e.ws.ID, c.id, ins, p.accepted, mustJSON(map[string]any{"group": group.String()}), group); err != nil {
			return fmt.Errorf("seat %s: %w", p.email, err)
		}
		hash := sha256.Sum256([]byte("demo-seat:" + seat.String()))
		if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_group_seats (id, group_id, workspace_id, email, contact_id, subscription_id, status, invite_hash, invited_at,
			invite_sent_at, invite_expires_at, accepted_at) VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8, $8, $9, $10)`,
			seat, group, e.ws.ID, p.email, c.id, sub, hash[:], invited, invited.Add(earn.InviteValid), p.accepted); err != nil {
			return fmt.Errorf("seat %s: %w", p.email, err)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO paid_events (id, workspace_id, contact_id, subscription_id, mode, kind, occurred_at, data)
			VALUES ($1, $2, $3, $4, 'live', 'team_joined', $5, $6), ($7, $2, $8, $9, 'live', 'team_accepted', $5, $10)`,
			ID(), e.ws.ID, c.id, sub, p.accepted, mustJSON(map[string]any{"team": "Acme Inc", "tier": "Insider"}),
			ID(), e.teamBuyer.id, buyer, mustJSON(map[string]any{"team": "Acme Inc", "email": p.email, "name": p.first + " " + p.last, "used": i + 2, "seats": 10})); err != nil {
			return fmt.Errorf("seat %s: %w", p.email, err)
		}
	}
	// Mei Chen's invite, sent on 13 Oct: it works for two weeks.
	invited := s.earnDay(2026, 10, 13, 15, 40)
	seat := ID()
	hash := sha256.Sum256([]byte("demo-seat:" + seat.String()))
	if _, err := s.Tx.Exec(ctx, `INSERT INTO earn_group_seats (id, group_id, workspace_id, email, status, invite_hash, invited_at, invite_sent_at, invite_expires_at)
		VALUES ($1, $2, $3, 'mei.chen@acme-inc.example', 'invited', $4, $5, $5, $6)`, seat, group, e.ws.ID, hash[:], invited, invited.Add(earn.InviteValid)); err != nil {
		return fmt.Errorf("invite: %w", err)
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO paid_events (id, workspace_id, contact_id, subscription_id, mode, kind, occurred_at, data)
		VALUES ($1, $2, $3, $4, 'live', 'team_invited', $5, $6)`, ID(), e.ws.ID, e.teamBuyer.id, buyer, invited,
		mustJSON(map[string]any{"team": "Acme Inc", "count": 1, "email": "mei.chen@acme-inc.example", "name": "Mei Chen",
			"until": invited.Add(earn.InviteValid).UTC().Format(time.RFC3339)})); err != nil {
		return fmt.Errorf("invite: %w", err)
	}
	return nil
}
