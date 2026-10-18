package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedEarnMore = seedEarnMoreImpl }

// earnMore carries the rest of the Earn sample (brief 4): the sponsors and
// their bookings, the offers and the cancel page, the gifts, the revenue
// history and the conversions.
type earnMore struct {
	s  *Seeder
	ws store.Workspace
	q  *store.Queries

	linh, marcus, sofia *uuid.UUID
	digest              uuid.UUID

	// Sponsors.
	slot               uuid.UUID
	northwind, contoso uuid.UUID
	bookings           map[int]uuid.UUID
}

func seedEarnMoreImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	digest := s.Lists["Weekly Digest"]
	if digest == uuid.Nil || ws.ID == uuid.Nil {
		return nil
	}
	e := &earnMore{s: s, ws: ws, q: store.New(s.Tx), digest: digest, bookings: map[int]uuid.UUID{}}
	for email, p := range map[string]**uuid.UUID{"linh@acme.io": &e.linh, "marcus@acme.io": &e.marcus, "sofia@acme.io": &e.sofia} {
		if u, ok := s.Users[email]; ok {
			id := u.ID
			*p = &id
		}
	}
	for _, step := range []func(context.Context) error{e.sponsors, e.bookingsSeed, e.sponsorAudit} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

// day is a calendar day of the Earn sample (Wed 15 Oct 2026 is today), as
// the workspace's local date.
func (e *earnMore) day(m time.Month, d int) time.Time {
	t := e.s.earnDay(2026, m, d, 12, 0).In(e.s.earnLoc())
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// sponsors: Northwind Analytics and Contoso Cloud, and Weekly Digest's
// "Main sponsor" slot after the first story, $750 an issue, every Tuesday
// numbered from #142 on 14 Oct 2026.
func (e *earnMore) sponsors(ctx context.Context) error {
	s := e.s
	e.northwind, e.contoso = ID(), ID()
	for _, sp := range []struct {
		id                               uuid.UUID
		name, web, contact, email, notes string
		at                               time.Time
	}{
		{e.northwind, "Northwind Analytics", "northwind-analytics.example", "Maya Lindqvist", "maya@northwind-analytics.example",
			"Books after each report. Prefers the trial link with UTM parameters.", s.earnDay(2026, 9, 26, 10, 12)},
		{e.contoso, "Contoso Cloud", "contoso.example", "Felix Osei", "felix@contoso.example", "Pays by wire transfer within two weeks.", s.earnDay(2026, 9, 19, 15, 40)},
	} {
		if err := e.q.SponsorInsert(ctx, store.SponsorInsertParams{ID: sp.id, WorkspaceID: e.ws.ID, Name: sp.name, Website: sp.web, ContactName: sp.contact,
			ContactEmail: sp.email, TermsDays: 14, Notes: sp.notes, CreatedBy: e.sofia, At: sp.at}); err != nil {
			return fmt.Errorf("sponsor %s: %w", sp.name, err)
		}
	}
	e.slot = ID()
	return e.q.SponsorSlotInsert(ctx, store.SponsorSlotInsertParams{ID: e.slot, WorkspaceID: e.ws.ID, ListID: e.digest, Name: "Main sponsor", Placement: "After the first story",
		PriceCents: 75000, Currency: "usd", FirstNumber: 142, FirstOn: e.day(10, 14), EveryDays: 7, CreativeDays: 4, Selling: true, CreatedBy: e.sofia,
		At: s.earnDay(2026, 9, 19, 15, 30)})
}

type sponsorCreativeSeed struct {
	Logo     string            `json:"logo"`
	Image    string            `json:"image"`
	Headline string            `json:"headline"`
	Text     string            `json:"text"`
	Button   string            `json:"button"`
	Link     string            `json:"link"`
	UTM      map[string]string `json:"utm"`
}

func utmOf(issue int) map[string]string {
	return map[string]string{"source": "acme-weekly-digest", "medium": "email", "campaign": fmt.Sprint(issue)}
}

// bookingsSeed: issues #140 to #147 as on the Sponsors board.
func (e *earnMore) bookingsSeed(ctx context.Context) error {
	s := e.s
	northwind := func(issue int) sponsorCreativeSeed {
		return sponsorCreativeSeed{Headline: "Find the dashboards nobody opens",
			Text:   "Northwind shows which reports your team actually uses, and which ones you can retire. Free for 14 days.",
			Button: "Try Northwind free", Link: "https://northwind-analytics.example/trial", UTM: utmOf(issue)}
	}
	contoso := func(issue int) sponsorCreativeSeed {
		return sponsorCreativeSeed{Headline: "Postgres that scales to zero", Text: "Managed Postgres with a branch for every pull request.", Button: "Start building",
			Link: "https://contoso.example/postgres", UTM: utmOf(issue)}
	}
	// A creative still to come has only its UTM parameters.
	waiting := func(issue int) sponsorCreativeSeed {
		return sponsorCreativeSeed{UTM: utmOf(issue)}
	}
	type paid struct {
		on     time.Time
		amount int64
		note   string
		by     *uuid.UUID
		at     time.Time
	}
	type booking struct {
		issue     int
		sponsor   uuid.UUID
		status    string
		hold      *time.Time
		campaign  string
		creative  sponsorCreativeSeed
		ready     bool
		readyAt   time.Time
		paid      *paid
		notes     string
		created   time.Time
		createdBy *uuid.UUID
		stats     []int64 // delivered, human opens, unique clickers, button clicks, logo clicks
		final     bool
	}
	hold := e.day(10, 20)
	list := []booking{
		{issue: 140, sponsor: e.contoso, status: "sent", campaign: "Weekly Digest #140", creative: contoso(140), ready: true, readyAt: s.earnDay(2026, 9, 26, 11, 0),
			paid: &paid{e.day(10, 3), 75000, "Wire transfer", e.sofia, s.earnDay(2026, 10, 3, 14, 12)}, created: s.earnDay(2026, 9, 19, 15, 45), createdBy: e.sofia,
			stats: []int64{31126, 13665, 342, 353, 51}, final: true},
		{issue: 141, sponsor: e.northwind, status: "sent", campaign: "Weekly Digest #141", creative: northwind(141), ready: true, readyAt: s.earnDay(2026, 10, 2, 16, 30),
			paid: &paid{e.day(10, 9), 75000, "Wire transfer, ref 5523", e.sofia, s.earnDay(2026, 10, 9, 15, 2)}, created: s.earnDay(2026, 9, 26, 10, 20), createdBy: e.sofia,
			notes: "First booking. Maya wants the report shared after each issue.", stats: []int64{31299, 14022, 412, 431, 67}, final: true},
		{issue: 142, sponsor: e.contoso, status: "sent", campaign: "Weekly Digest #142", creative: contoso(142), ready: true, readyAt: s.earnDay(2026, 10, 9, 10, 0),
			created: s.earnDay(2026, 9, 19, 15, 46), createdBy: e.sofia, stats: []int64{31342, 12108, 296, 309, 42}},
		{issue: 143, sponsor: e.northwind, status: "booked", campaign: "Weekly Digest #143", creative: waiting(143),
			notes:   "Second of three issues booked after #141. Maya asked for the same slot and the trial link again.",
			created: s.earnDay(2026, 10, 9, 16, 20), createdBy: e.sofia},
		{issue: 145, sponsor: e.contoso, status: "hold", hold: &hold, creative: waiting(145), created: s.earnDay(2026, 10, 13, 11, 30), createdBy: e.sofia},
		{issue: 146, sponsor: e.northwind, status: "booked", creative: waiting(146), created: s.earnDay(2026, 10, 9, 16, 20), createdBy: e.sofia},
	}
	for _, b := range list {
		// The issue's date in the sample's calendar, and the demo's.
		design := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC).AddDate(0, 0, (b.issue-142)*7)
		on := e.day(design.Month(), design.Day())
		id := ID()
		e.bookings[b.issue] = id
		cr, _ := json.Marshal(b.creative)
		creativeDue := on.AddDate(0, 0, -4)
		paymentDue := on.AddDate(0, 0, 14)
		var camp *uuid.UUID
		if c, ok := s.Campaigns[b.campaign]; ok && b.campaign != "" {
			camp = &c
		}
		sid := b.sponsor
		if err := e.q.SponsorBookingInsert(ctx, store.SponsorBookingInsertParams{ID: id, WorkspaceID: e.ws.ID, SlotID: e.slot, IssueNumber: int32(b.issue), IssueOn: on,
			SponsorID: &sid, CampaignID: camp, CampaignByHand: false, PriceCents: 75000, Currency: "usd", Status: b.status, HoldUntil: b.hold, Creative: cr,
			CreativeStatus: "waiting", CreativeDue: &creativeDue, PaymentDue: &paymentDue, Notes: b.notes, CreatedBy: b.createdBy, At: b.created}); err != nil {
			return fmt.Errorf("booking #%d: %w", b.issue, err)
		}
		if b.ready {
			at := b.readyAt
			if err := e.q.SponsorBookingSetCreative(ctx, store.SponsorBookingSetCreativeParams{ID: id, WorkspaceID: e.ws.ID, Creative: cr, CreativeStatus: "ready",
				ReadyAt: &at, ReadyBy: e.sofia, At: at}); err != nil {
				return err
			}
		}
		if b.paid != nil {
			on, amount, rec := b.paid.on, b.paid.amount, b.paid.at
			if err := e.q.SponsorBookingSetPayment(ctx, store.SponsorBookingSetPaymentParams{ID: id, WorkspaceID: e.ws.ID, PaymentStatus: "paid", PaidOn: &on,
				PaidCents: &amount, PaidNote: b.paid.note, PaidBy: b.paid.by, PaidRecordedAt: &rec, At: rec}); err != nil {
				return err
			}
		}
		if b.status == "sent" {
			sent := s.earnDay(design.Year(), design.Month(), design.Day(), 9, 0)
			if _, err := s.Tx.Exec(ctx, `UPDATE sponsor_bookings SET sent_at = $2 WHERE id = $1`, id, sent); err != nil {
				return err
			}
			links, _ := json.Marshal([]map[string]any{{"kind": "button", "label": b.creative.Button, "clicks": b.stats[3]}, {"kind": "logo", "label": "", "clicks": b.stats[4]}})
			computed := sent.Add(72 * time.Hour)
			if !b.final {
				computed = s.Ago(6 * time.Minute)
			}
			if err := e.q.SponsorReportStatsUpsert(ctx, store.SponsorReportStatsUpsertParams{BookingID: id, SentAt: &sent, Delivered: b.stats[0], HumanOpens: b.stats[1],
				UniqueClickers: b.stats[2], Clicks: b.stats[3] + b.stats[4], Links: links, Final: b.final, ComputedAt: computed}); err != nil {
				return err
			}
		}
	}
	// The activity of #143 and #141.
	for _, ev := range []struct {
		issue int
		kind  string
		at    time.Time
		by    *uuid.UUID
		name  string
		data  map[string]any
	}{
		{141, "booked", s.earnDay(2026, 9, 26, 10, 20), e.sofia, "Sofia Rossi", map[string]any{"sponsor": "Northwind Analytics", "priceCents": 75000}},
		{141, "creative_ready", s.earnDay(2026, 10, 2, 16, 30), e.sofia, "Sofia Rossi", nil},
		{141, "report_shared", s.earnDay(2026, 10, 9, 11, 5), e.sofia, "Sofia Rossi", map[string]any{"expiresAt": s.earnDay(2027, 1, 7, 11, 5).UTC().Format(time.RFC3339)}},
		{141, "paid", s.earnDay(2026, 10, 9, 15, 2), e.sofia, "Sofia Rossi", map[string]any{"amountCents": 75000, "paidOn": e.day(10, 9).Format(time.DateOnly), "note": "Wire transfer, ref 5523"}},
		{143, "booked", s.earnDay(2026, 10, 9, 16, 20), e.sofia, "Sofia Rossi", map[string]any{"sponsor": "Northwind Analytics", "priceCents": 75000, "with": []int{143, 146}, "after": 141}},
		{143, "creative_requested", s.earnDay(2026, 10, 10, 9, 40), e.sofia, "Sofia Rossi", map[string]any{"contact": "Maya Lindqvist"}},
		{146, "booked", s.earnDay(2026, 10, 9, 16, 20), e.sofia, "Sofia Rossi", map[string]any{"sponsor": "Northwind Analytics", "priceCents": 75000, "with": []int{143, 146}, "after": 141}},
		{145, "held", s.earnDay(2026, 10, 13, 11, 30), e.sofia, "Sofia Rossi", map[string]any{"sponsor": "Contoso Cloud", "holdUntil": e.day(10, 20).Format(time.DateOnly)}},
	} {
		data := ev.data
		if data == nil {
			data = map[string]any{}
		}
		raw, _ := json.Marshal(data)
		if err := e.q.SponsorBookingEventInsert(ctx, store.SponsorBookingEventInsertParams{ID: ID(), BookingID: e.bookings[ev.issue], Kind: ev.kind, OccurredAt: ev.at,
			ActorID: ev.by, ActorName: ev.name, Data: raw}); err != nil {
			return err
		}
	}
	// Sofia shared #141's report with Northwind Analytics on 9 Oct; it
	// expires on 7 Jan 2027 and was opened 6 times.
	token := SponsorReportToken
	sealedID := ID()
	sealed, err := s.Box.SealString(token, "sponsor_report_share:"+sealedID.String()+":token")
	if err != nil {
		return err
	}
	b141 := e.bookings[141]
	expires := s.earnDay(2027, 1, 7, 11, 5)
	if err := e.q.SponsorShareInsert(ctx, store.SponsorShareInsertParams{ID: sealedID, WorkspaceID: e.ws.ID, BookingID: &b141, TokenHash: authn.HashToken(token),
		TokenSealed: sealed, CreatedBy: e.sofia, CreatedAt: s.earnDay(2026, 10, 9, 11, 5), ExpiresAt: &expires}); err != nil {
		return err
	}
	_, err = s.Tx.Exec(ctx, `UPDATE sponsor_report_shares SET views = 6, last_viewed_at = $2 WHERE id = $1`, sealedID, s.earnDay(2026, 10, 14, 17, 40))
	return err
}

// SponsorReportToken is the demo's read-only link to #141's sponsor report
// (reputecast.acme.io/r/Qm7vT2pXk9LbN4sW8cYd1A).
const SponsorReportToken = "Qm7vT2pXk9LbN4sW8cYd1A" //nolint:gosec // a public demo link, not a secret

// sponsorAudit: the sponsorship entries of the audit log.
func (e *earnMore) sponsorAudit(ctx context.Context) error {
	s := e.s
	for _, a := range []auditSeed{
		{at: s.earnDay(2026, 9, 19, 15, 30), actorType: "user", actor: "sofia@acme.io", action: "sponsors.slot_created", targetType: "sponsor_slot", targetID: e.slot.String(),
			target: "Main sponsor", ip: "203.0.113.14", ua: uaChromeMac, after: map[string]any{"list": "Weekly Digest", "placement": "After the first story", "priceCents": 75000,
				"firstNumber": 142, "everyDays": 7, "creativeDays": 4}},
		{at: s.earnDay(2026, 9, 26, 10, 12), actorType: "user", actor: "sofia@acme.io", action: "sponsors.sponsor_created", targetType: "sponsor", targetID: e.northwind.String(),
			target: "Northwind Analytics", ip: "203.0.113.14", ua: uaChromeMac, after: map[string]any{"website": "northwind-analytics.example", "contact": "Maya Lindqvist", "termsDays": 14}},
		{at: s.earnDay(2026, 10, 9, 11, 5), actorType: "user", actor: "sofia@acme.io", action: "sponsors.report_link_created", targetType: "sponsor_report_share",
			targetID: e.bookings[141].String(), target: "Sponsor report · Weekly Digest #141 · Northwind Analytics", ip: "203.0.113.14", ua: uaChromeMac,
			after: map[string]any{"expiresAt": "2027-01-07", "readOnly": true}},
		{at: s.earnDay(2026, 10, 9, 15, 2), actorType: "user", actor: "sofia@acme.io", action: "sponsors.payment_recorded", targetType: "sponsor_booking",
			targetID: e.bookings[141].String(), target: "Weekly Digest #141 · Main sponsor", ip: "203.0.113.14", ua: uaChromeMac,
			before: map[string]any{"payment": "unpaid"}, after: map[string]any{"payment": "paid", "paidOn": e.day(10, 9).Format(time.DateOnly), "paidCents": 75000, "note": "Wire transfer, ref 5523"}},
		{at: s.earnDay(2026, 10, 9, 16, 20), actorType: "user", actor: "sofia@acme.io", action: "sponsors.booking_created", targetType: "sponsor_booking",
			targetID: e.bookings[143].String(), target: "Weekly Digest #143 · Main sponsor", ip: "203.0.113.14", ua: uaChromeMac,
			after: map[string]any{"sponsor": "Northwind Analytics", "status": "booked", "priceCents": 75000}},
		{at: s.earnDay(2026, 10, 13, 11, 30), actorType: "user", actor: "sofia@acme.io", action: "sponsors.booking_created", targetType: "sponsor_booking",
			targetID: e.bookings[145].String(), target: "Weekly Digest #145 · Main sponsor", ip: "203.0.113.14", ua: uaChromeMac,
			after: map[string]any{"sponsor": "Contoso Cloud", "status": "hold", "holdUntil": e.day(10, 20).Format(time.DateOnly), "priceCents": 75000}},
	} {
		if err := s.insertAudit(ctx, e.ws.ID, a); err != nil {
			return err
		}
	}
	return nil
}
