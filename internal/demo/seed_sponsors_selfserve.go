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

// Self-serve sponsor booking (M65): the media kit at /p/acme-newsletter/
// sponsor is on (all three numbers, About, Audience, sponsors@acme.io) and
// the Main sponsor slot is self-serve. Four requests came from it: Fabrikam
// Studio (#147) and Tailspin Toys (#148) wait for a decision, Wide World
// Importers (#148) was approved by Sofia Rossi on 14 Oct and has until Fri
// 17 Oct to pay, and Adventure Works (#150) paid through Stripe on 13 Oct.

// The demo's sponsor status links (/p/acme-newsletter/sponsor/b/{token}),
// for the boards' frames: public demo links, not secrets.
const (
	SponsorStatusRequested = "dmFicmlrYW0tcmVxdWVzdGVkLTE0Ny1kZW1vLWxpbms"  //nolint:gosec // a public demo link
	SponsorStatusDeclined  = "dGFpbHNwaW4tcmVxdWVzdGVkLTE0OC1kZW1vLWxpbms"  //nolint:gosec // a public demo link
	SponsorStatusApproved  = "d2lkZS13b3JsZC1hcHByb3ZlZC0xNDgtZGVtby1saW5r" //nolint:gosec // a public demo link
	SponsorStatusBooked    = "YWR2ZW50dXJlLXdvcmtzLWJvb2tlZC0xNTAtZGVtbw"   //nolint:gosec // a public demo link
)

type sponsorRequestSeed struct {
	issue                     int
	company, web, name, email string
	notes                     string
	creative                  sponsorCreativeSeed
	at                        time.Time
	token                     string
	approved, paid            *time.Time
	payBy                     time.Time
}

// requestEventSeed is a line of a request's activity.
type requestEventSeed struct {
	kind string
	at   time.Time
	by   *uuid.UUID
	name string
	data map[string]any
}

func (e *earnMore) selfServe(ctx context.Context) error {
	s := e.s
	kit, _ := json.Marshal(map[string]any{"subscribers": true, "openRate": true, "clickRate": true,
		"about":    "The Weekly Digest is Acme's Tuesday letter about what shipped, what's next and how teams put it to work. Each issue has one sponsor, placed after the first story, and readers can tell it's sponsored.",
		"audience": "Product managers, engineers and founders at software companies, most of them in North America and Europe. About a third read on a phone."})
	if err := e.q.SponsorSettingsSave(ctx, store.SponsorSettingsSaveParams{WorkspaceID: e.ws.ID, SelfServe: true, MediaKit: kit, ContactEmail: "sponsors@acme.io",
		UpdatedBy: e.sofia, At: s.earnDay(2026, 10, 9, 15, 20)}); err != nil {
		return fmt.Errorf("sponsor page: %w", err)
	}
	approved, paidOn := s.earnDay(2026, 10, 14, 10, 2), s.earnDay(2026, 10, 13, 11, 18)
	awApproved := s.earnDay(2026, 10, 11, 9, 40)
	// The sponsors' notes name dates on the demo's calendar.
	launch, ships, offer := s.earnDay(2026, 11, 18, 12, 0), s.earnDay(2026, 11, 20, 12, 0), s.earnDay(2026, 12, 1, 12, 0)
	list := []sponsorRequestSeed{
		{issue: 147, company: "Fabrikam Studio", web: "fabrikam-studio.example", name: "Lucía Fernández", email: "lucia@fabrikam-studio.example",
			notes: "We're launching our review plugin on " + launch.Format("2 Jan") + ". Happy to move a week if that issue goes.",
			creative: sponsorCreativeSeed{Headline: "Design reviews without the screenshot pile", Text: "Fabrikam Studio collects comments on your live pages, so feedback lands where the work is. Free for teams of five.",
				Button: "Try Fabrikam free", Link: "https://fabrikam-studio.example/reviews", UTM: utmOf(147)},
			at: s.earnDay(2026, 10, 14, 11, 42), token: SponsorStatusRequested},
		{issue: 148, company: "Tailspin Toys", web: "tailspintoys.example", name: "Omar Siddiqui", email: "omar@tailspintoys.example",
			notes: "Our holiday kits ship from " + ships.Format("2 Jan") + ", so late " + ships.Format("January") + " works best for us.",
			creative: sponsorCreativeSeed{Headline: "A drone kit for the builders at home", Text: "Kids build and code a working drone in a weekend. Weekly Digest readers get 15% off until " + offer.Format("2 Jan") + ".",
				Button: "See the kits", Link: "https://tailspintoys.example/kits", UTM: utmOf(148)},
			at: s.earnDay(2026, 10, 13, 16, 5), token: SponsorStatusDeclined},
		{issue: 148, company: "Wide World Importers", web: "wideworldimporters.example", name: "Hannah Becker", email: "hannah@wideworldimporters.example",
			creative: sponsorCreativeSeed{Headline: "Ship to 40 countries from one dashboard", Text: "Wide World Importers handles customs forms, duties and returns for online stores. First 100 shipments free.",
				Button: "Start shipping", Link: "https://wideworldimporters.example/shipping", UTM: utmOf(148)},
			at: s.earnDay(2026, 10, 12, 9, 30), token: SponsorStatusApproved, approved: &approved, payBy: s.earnDay(2026, 10, 17, 10, 2)},
		{issue: 150, company: "Adventure Works", web: "adventure-works.example", name: "Grace Liu", email: "grace@adventure-works.example",
			creative: sponsorCreativeSeed{Headline: "Status pages your customers trust", Text: "Adventure Works watches your services from 12 regions and tells customers before they ask. Free for one page.",
				Button: "Make a status page", Link: "https://adventure-works.example/status", UTM: utmOf(150)},
			at: s.earnDay(2026, 10, 10, 14, 12), token: SponsorStatusBooked, approved: &awApproved, paid: &paidOn, payBy: s.earnDay(2026, 10, 14, 9, 40)},
	}
	for _, r := range list {
		sponsor := ID()
		if err := e.q.SponsorInsert(ctx, store.SponsorInsertParams{ID: sponsor, WorkspaceID: e.ws.ID, Name: r.company, Website: r.web, ContactName: r.name,
			ContactEmail: r.email, TermsDays: 0, Notes: "", At: r.at}); err != nil {
			return fmt.Errorf("sponsor %s: %w", r.company, err)
		}
		design := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC).AddDate(0, 0, (r.issue-142)*7)
		on := e.day(design.Month(), design.Day())
		creativeDue := on.AddDate(0, 0, -4)
		cr, _ := json.Marshal(r.creative)
		doc, _ := json.Marshal(map[string]any{"company": r.company, "name": r.name, "email": r.email, "website": r.web, "notes": r.notes, "lang": "en",
			"slot": e.slot.String(), "issue": r.issue})
		id := ID()
		if err := e.q.SponsorRequestInsert(ctx, store.SponsorRequestInsertParams{ID: id, WorkspaceID: e.ws.ID, SlotID: e.slot, IssueNumber: int32(r.issue), IssueOn: on,
			SponsorID: &sponsor, PriceCents: 75000, Currency: "usd", Creative: cr, CreativeDue: &creativeDue, Request: doc, StatusTokenHash: authn.HashToken(r.token),
			At: r.at}); err != nil {
			return fmt.Errorf("request %s: %w", r.company, err)
		}
		events := []requestEventSeed{{"requested", r.at, nil, r.name, map[string]any{"company": r.company, "name": r.name, "sponsor": r.company}}}
		if r.approved != nil {
			payDue := r.payBy
			if _, err := e.q.SponsorBookingApprove(ctx, store.SponsorBookingApproveParams{SlotID: e.slot, IssueNumber: int32(r.issue), IssueOn: on, PriceCents: 75000,
				Currency: "usd", Creative: cr, CreativeDue: &creativeDue, PaymentDue: &payDue, PaymentDueAt: r.payBy, StatusTokenHash: authn.HashToken(r.token),
				At: *r.approved, ID: id, WorkspaceID: e.ws.ID}); err != nil {
				return fmt.Errorf("approve %s: %w", r.company, err)
			}
			events = append(events, requestEventSeed{"approved", *r.approved, e.sofia, "Sofia Rossi", map[string]any{"paymentDueAt": r.payBy.UTC().Format(time.RFC3339), "priceCents": 75000}})
		}
		if r.paid != nil {
			amount := int64(75000)
			if err := e.q.SponsorBookingStripePaid(ctx, store.SponsorBookingStripePaidParams{Status: "booked", PaidCents: &amount, PaidOn: *r.paid, At: *r.paid,
				CheckoutID: "cs_live_a1Rw7PxT2mKq", ID: id}); err != nil {
				return fmt.Errorf("paid %s: %w", r.company, err)
			}
			events = append(events, requestEventSeed{"paid", *r.paid, nil, r.name, map[string]any{"amountCents": amount, "source": "stripe"}}, requestEventSeed{"booked", *r.paid, nil, r.name, map[string]any{"priceCents": amount, "paidBy": "stripe"}})
		}
		for _, ev := range events {
			raw, _ := json.Marshal(ev.data)
			if err := e.q.SponsorBookingEventInsert(ctx, store.SponsorBookingEventInsertParams{ID: ID(), BookingID: id, Kind: ev.kind, OccurredAt: ev.at,
				ActorID: ev.by, ActorName: ev.name, Data: raw}); err != nil {
				return err
			}
		}
	}
	return nil
}
