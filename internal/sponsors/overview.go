package sponsors

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// OverviewQuery picks the slot and the issues of Earn › Sponsors.
type OverviewQuery struct {
	Slot string `query:"slot" doc:"A slot's id; the first slot when absent."`
	// Range: around (the last 3 and the next 5 issues), upcoming (the next
	// 10) or past (the last 12).
	Range string `query:"range" enum:"around,upcoming,past" default:"around"`
}

// noticeWindow is how close an issue must be for "no campaign yet".
const noticeWindow = 10

// bookingView is a booking row in the API's shape.
func (s *Service) bookingBrief(ws *store.Workspace, b store.SponsorBookingsOfSlotRow, stats map[uuid.UUID]store.SponsorReportStat, shared map[uuid.UUID]time.Time,
	today time.Time) *SponsorBookingBrief {
	status := b.Status
	if status == "booked" && b.CampaignStatus == "sent" {
		status = "sent"
	}
	out := &SponsorBookingBrief{ID: b.ID.String(), Status: status, HoldUntil: noonPtr(b.HoldUntil, ws), PriceCents: b.PriceCents, CreativeStatus: b.CreativeStatus,
		CreativeDue: noonPtr(b.CreativeDue, ws), Payment: paymentOf(ws, b.PaymentStatus, b.PaymentDue, b.PaidOn, b.PaidCents, b.PaidNote, today)}
	if b.SponsorID != nil {
		out.Sponsor = refOf(*b.SponsorID, b.SponsorName, "")
	}
	sent := firstTime(b.SentAt, b.CampaignStartedAt)
	if status == "sent" && sent != nil {
		out.SentAt = sent
		fin := sent.Add(FinalAfter)
		out.ResultsFinalAt = &fin
		if st, ok := stats[b.ID]; ok {
			out.Final = st.Final
			if st.SentAt != nil {
				fin := st.SentAt.Add(FinalAfter)
				out.ResultsFinalAt = &fin
			}
			if st.Delivered > 0 {
				r := float64(st.UniqueClickers) / float64(st.Delivered) * 100
				out.ClickRate = &r
			}
		}
	}
	if at, ok := shared[b.ID]; ok {
		out.SharedAt = &at
	}
	return out
}

func firstTime(ts ...*time.Time) *time.Time {
	for _, t := range ts {
		if t != nil && !t.IsZero() {
			return t
		}
	}
	return nil
}

// paymentOf is a payment's status: overdue once an unpaid booking's due
// date has passed.
func paymentOf(ws *store.Workspace, status string, due, paidOn *time.Time, paid *int64, note string, today time.Time) SponsorPayment {
	out := SponsorPayment{Status: status, Due: noonPtr(due, ws), PaidOn: noonPtr(paidOn, ws), PaidCents: paid, Note: note}
	if status == "unpaid" && due != nil && civil(*due).Before(today) {
		out.Status = "overdue"
	}
	return out
}

// Overview is Earn › Sponsors: the numbers, a slot's issues with their
// bookings, the sponsor library, the slots and what needs a person.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal, in OverviewQuery) (SponsorOverview, error) {
	out := SponsorOverview{Slots: []SponsorSlotItem{}, Past: []SponsorIssueRow{}, Upcoming: []SponsorIssueRow{}, Library: []SponsorItem{}, Notices: []SponsorNotice{},
		Range: in.Range, CanManage: p.Can(authz.SponsorsManage), Currency: s.currency(ctx, ws), Tiles: SponsorTiles{AvgClickIssues: []int{}, BookedAheadIssues: []int{}}}
	if out.Range == "" {
		out.Range = "around"
	}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, authz.SponsorsManage, p.UserID, 3)
	} else {
		out.Helpers = []string{}
	}
	lib, err := s.library(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Library = lib
	slots, err := s.Q.SponsorSlots(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	all, err := s.Q.SponsorBookingsOfWorkspace(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	counts := map[uuid.UUID]int{}
	for _, b := range all {
		if b.Status == "booked" || b.Status == "sent" {
			counts[b.SlotID]++
		}
	}
	for _, r := range slots {
		out.Slots = append(out.Slots, slotItem(r, ws, counts[r.ID]))
	}
	today := s.today(ws)
	if err := s.tiles(ctx, ws, &out, all, today); err != nil {
		return out, err
	}
	out.Notices = s.notices(ws, all, slots, today)
	if len(slots) == 0 {
		return out, nil
	}
	sel := slots[0]
	for _, r := range slots {
		if r.ID.String() == in.Slot {
			sel = r
		}
	}
	out.SlotID = sel.ID.String()
	rows, err := s.issueRows(ctx, ws, sel, out.Range, today)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.Past {
			out.Past = append(out.Past, r)
		} else {
			out.Upcoming = append(out.Upcoming, r)
		}
	}
	// The next 5 issues of the slot.
	next := s.nextNumber(ctx, sel, today)
	slot := slotRow(sel)
	nextRows, err := s.issueRowsBetween(ctx, ws, sel, next, next+4, today)
	if err != nil {
		return out, err
	}
	t := &out.Tiles
	t.NextCount, t.NextFrom, t.NextTo = len(nextRows), next, next+4
	for _, r := range nextRows {
		switch {
		case r.Booking == nil:
			t.NextOpen++
		case r.Booking.Status == "hold":
			t.NextHold++
		default:
			t.NextBooked++
		}
	}
	_ = slot
	return out, nil
}

func slotRow(r store.SponsorSlotsRow) store.SponsorSlot {
	return store.SponsorSlot{ID: r.ID, WorkspaceID: r.WorkspaceID, ListID: r.ListID, Name: r.Name, Placement: r.Placement, PriceCents: r.PriceCents, Currency: r.Currency,
		FirstNumber: r.FirstNumber, FirstOn: r.FirstOn, EveryDays: r.EveryDays, CreativeDays: r.CreativeDays, Selling: r.Selling}
}

// nextNumber is the slot's next issue: the first on or after today, past
// today's once it has gone out.
func (s *Service) nextNumber(ctx context.Context, sl store.SponsorSlotsRow, today time.Time) int {
	slot := slotRow(sl)
	n := NextNumber(slot, today)
	if IssueOn(slot, n).Equal(today) {
		if b, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: sl.ID, IssueNumber: int32(n)}); err == nil && b.Status == "sent" {
			n++
		}
	}
	return n
}

func (s *Service) issueRows(ctx context.Context, ws *store.Workspace, sl store.SponsorSlotsRow, rng string, today time.Time) ([]SponsorIssueRow, error) {
	next := s.nextNumber(ctx, sl, today)
	from, to := next-3, next+4
	switch rng {
	case "upcoming":
		from, to = next, next+9
	case "past":
		from, to = next-12, next-1
	}
	if from < 1 {
		from = 1
	}
	return s.issueRowsBetween(ctx, ws, sl, from, to, today)
}

// issueRowsBetween builds a slot's issues from one number to another,
// with their bookings, proposals, campaigns and reports.
func (s *Service) issueRowsBetween(ctx context.Context, ws *store.Workspace, sl store.SponsorSlotsRow, from, to int, today time.Time) ([]SponsorIssueRow, error) {
	slot := slotRow(sl)
	bookings, err := s.Q.SponsorBookingsOfSlot(ctx, store.SponsorBookingsOfSlotParams{SlotID: sl.ID, FromNumber: int32(from), ToNumber: int32(to)})
	if err != nil {
		return nil, err
	}
	byIssue := map[int]store.SponsorBookingsOfSlotRow{}
	proposals := map[int]store.SponsorBookingsOfSlotRow{}
	ids := []uuid.UUID{}
	for _, b := range bookings {
		switch b.Status {
		case "proposed":
			if _, ok := proposals[int(b.IssueNumber)]; !ok {
				proposals[int(b.IssueNumber)] = b
			}
		default:
			byIssue[int(b.IssueNumber)] = b
			ids = append(ids, b.ID)
		}
	}
	stats := map[uuid.UUID]store.SponsorReportStat{}
	shared := map[uuid.UUID]time.Time{}
	if len(ids) > 0 {
		st, err := s.Q.SponsorReportStatsOf(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, r := range st {
			stats[r.BookingID] = r
		}
		rows, err := s.Pool.Query(ctx, `SELECT booking_id, min(created_at) FROM sponsor_report_shares WHERE booking_id = ANY($1) GROUP BY booking_id`, ids)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id uuid.UUID
			var at time.Time
			if rows.Scan(&id, &at) == nil {
				shared[id] = at
			}
		}
		rows.Close()
	}
	out := []SponsorIssueRow{}
	for n := from; n <= to; n++ {
		on := IssueOn(slot, n)
		r := SponsorIssueRow{SlotID: sl.ID.String(), SlotName: sl.Name, Number: n, IssueOn: noonOn(on, ws), Past: on.Before(today)}
		if b, ok := byIssue[n]; ok {
			r.Booking = s.bookingBrief(ws, b, stats, shared, today)
			if r.Booking.Status == "sent" {
				r.Past = true
			}
			if b.CampaignID != nil && b.CampaignName != "" {
				r.Campaign = &SponsorCampaignRef{ID: b.CampaignID.String(), Name: b.CampaignName, Status: b.CampaignStatus, ByHand: b.CampaignByHand}
			}
		}
		if r.Campaign == nil {
			if cid := s.campaignOn(ctx, ws, sl.ListID, on); cid != nil {
				if cp, err := s.Q.SponsorCampaignBrief(ctx, store.SponsorCampaignBriefParams{ID: *cid, WorkspaceID: ws.ID}); err == nil {
					r.Campaign = &SponsorCampaignRef{ID: cp.ID.String(), Name: cp.Name, Status: cp.Status}
				}
			}
		}
		if pr, ok := proposals[n]; ok && r.Booking == nil {
			r.Proposal = proposalOf(pr.ID, pr.ProposedName, pr.ProposedNote, pr.CreatedAt, pr.SponsorID, pr.SponsorName, pr.PriceCents)
		}
		out = append(out, r)
	}
	return out, nil
}

func proposalOf(id uuid.UUID, agent, note string, at time.Time, sponsor *uuid.UUID, sponsorName string, price int64) *SponsorPropose {
	p := &SponsorPropose{ID: id.String(), Agent: agent, Note: note, At: at, PriceCents: price}
	if sponsor != nil {
		p.Sponsor = refOf(*sponsor, sponsorName, "")
	}
	return p
}

// tiles are the tab's four numbers.
func (s *Service) tiles(ctx context.Context, ws *store.Workspace, out *SponsorOverview, all []store.SponsorBookingsOfWorkspaceRow, today time.Time) error {
	t := &out.Tiles
	since := today.AddDate(0, 0, -30)
	var reports []uuid.UUID
	for _, b := range all {
		sent := b.Status == "sent" || (b.Status == "booked" && b.CampaignStatus == "sent")
		on := civil(b.IssueOn)
		if sent && !on.Before(since) && !on.After(today) {
			t.Income30dCents += b.PriceCents
			t.IncomeIssues++
			if b.PaymentStatus == "paid" {
				t.Paid30dCents += b.PriceCents
			} else {
				t.Unpaid30dCents += b.PriceCents
			}
		}
		if sent {
			reports = append(reports, b.ID)
		}
		if !on.Before(today) && !sent {
			switch b.Status {
			case "booked":
				t.BookedAheadCents += b.PriceCents
				t.BookedAheadIssues = append(t.BookedAheadIssues, int(b.IssueNumber))
			case "hold":
				t.HoldAheadCents += b.PriceCents
			}
		}
	}
	if len(reports) > 0 {
		st, err := s.Q.SponsorReportStatsOf(ctx, reports)
		if err != nil {
			return err
		}
		issues := map[uuid.UUID]int{}
		for _, b := range all {
			issues[b.ID] = int(b.IssueNumber)
		}
		var clickers, delivered int64
		cut := s.now().AddDate(0, 0, -90)
		for _, r := range st {
			if !r.Final || r.Delivered == 0 || (r.SentAt != nil && r.SentAt.Before(cut)) {
				continue
			}
			clickers += r.UniqueClickers
			delivered += r.Delivered
			t.AvgClickIssues = append(t.AvgClickIssues, issues[r.BookingID])
		}
		if delivered > 0 {
			rate := float64(clickers) / float64(delivered) * 100
			t.AvgClickRate = &rate
		}
		slices.Sort(t.AvgClickIssues)
	}
	slices.Sort(t.BookedAheadIssues)
	return nil
}

// notices are what needs a person: a creative due or late before its
// issue, an overdue payment, a hold ending tomorrow, a booked issue soon
// without a campaign.
func (s *Service) notices(ws *store.Workspace, all []store.SponsorBookingsOfWorkspaceRow, slots []store.SponsorSlotsRow, today time.Time) []SponsorNotice {
	lists := map[uuid.UUID]string{}
	for _, sl := range slots {
		lists[sl.ID] = sl.ListName
	}
	var late, overdue, lapsing, nocamp []SponsorNotice
	for _, b := range all {
		on := civil(b.IssueOn)
		days := int(on.Sub(today).Hours() / 24)
		n := SponsorNotice{BookingID: b.ID.String(), SlotID: b.SlotID.String(), Issue: int(b.IssueNumber), IssueOn: noonOn(on, ws), Sponsor: b.SponsorName,
			ContactName: b.SponsorContact, ContactEmail: b.SponsorEmail, AmountCents: b.PriceCents, DaysToIssue: days, ListName: lists[b.SlotID]}
		sent := b.Status == "sent" || b.CampaignStatus == "sent"
		switch {
		case b.Status == "booked" && !sent && b.CreativeStatus == "waiting" && b.CreativeDue != nil && !civil(*b.CreativeDue).After(today) && days >= 0:
			n.Kind, n.Due = "creative_late", noonPtr(b.CreativeDue, ws)
			late = append(late, n)
		case b.Status == "hold" && b.HoldUntil != nil && civil(*b.HoldUntil).Sub(today) <= 24*time.Hour && !civil(*b.HoldUntil).Before(today):
			n.Kind, n.Due = "hold_lapsing", noonPtr(b.HoldUntil, ws)
			lapsing = append(lapsing, n)
		case b.Status == "booked" && !sent && b.CampaignID == nil && days >= 0 && days <= noticeWindow:
			n.Kind = "no_campaign"
			nocamp = append(nocamp, n)
		}
		if (b.Status == "booked" || b.Status == "sent") && b.PaymentStatus == "unpaid" && b.PaymentDue != nil && civil(*b.PaymentDue).Before(today) {
			m := n
			m.Kind, m.Due = "payment_overdue", noonPtr(b.PaymentDue, ws)
			overdue = append(overdue, m)
		}
	}
	out := []SponsorNotice{}
	out = append(out, late...)
	out = append(out, overdue...)
	out = append(out, lapsing...)
	out = append(out, nocamp...)
	return out
}
