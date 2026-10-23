package sponsors

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

func (s *Service) libraryRefs(ctx context.Context, ws *store.Workspace) []SponsorRef {
	out := []SponsorRef{}
	rows, err := s.Q.SponsorList(ctx, ws.ID)
	if err != nil {
		return out
	}
	for _, r := range rows {
		out = append(out, *refOf(r.ID, r.Name, r.LogoUrl))
	}
	return out
}

// Booking is the booking panel.
func (s *Service) Booking(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SponsorBookingDetail, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	today := s.today(ws)
	status := b.Status
	if status == "booked" && b.CampaignStatus == "sent" {
		status = "sent"
	}
	out := SponsorBookingDetail{ID: b.ID.String(), SlotID: slot.ID.String(), SlotName: slot.Name, ListName: slot.ListName, Placement: slot.Placement,
		Issue: int(b.IssueNumber), IssueOn: noonOn(b.IssueOn, ws), Status: status, HoldUntil: noonPtr(b.HoldUntil, ws), PriceCents: b.PriceCents,
		SlotPrice: slot.PriceCents, Currency: b.Currency, Creative: parseCreative(b.Creative), CreativeSt: b.CreativeStatus, CreativeDue: noonPtr(b.CreativeDue, ws),
		ReadyAt: b.ReadyAt, Payment: paymentOf(ws, b.PaymentStatus, b.PaymentDue, b.PaidOn, b.PaidCents, b.PaidNote, today), Notes: b.Notes,
		Activity: []SponsorBookingEvent{}, Shares: []SponsorShareLink{}, CanceledAt: b.CanceledAt,
		CanManage: p.Can(authz.SponsorsManage), CanShare: p.Can(authz.ReportsShare), Library: s.libraryRefs(ctx, ws)}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, authz.SponsorsManage, p.UserID, 3)
	} else {
		out.Helpers = []string{}
	}
	if b.PaidBy != nil {
		if n, err := s.Q.EarnUserName(ctx, *b.PaidBy); err == nil {
			out.Payment.RecordedBy = n
		}
		out.Payment.RecordedAt = b.PaidRecordedAt
	}
	if b.SponsorID != nil {
		if it, err := s.sponsorItem(ctx, ws, *b.SponsorID); err == nil {
			out.Sponsor = &it
		} else {
			out.Sponsor = &SponsorItem{ID: b.SponsorID.String(), Name: b.SponsorName, Initials: initials(b.SponsorName), Issues: []int{}, Hold: []int{}}
		}
		out.Label = text(ctx, "block.labelBy", map[string]any{"sponsor": b.SponsorName})
	}
	if b.CampaignID != nil && b.CampaignName != "" {
		out.Campaign = s.campaignRef(ctx, ws, *b.CampaignID, b.CampaignByHand)
	} else if cid := s.campaignOn(ctx, ws, slot.ListID, b.IssueOn); cid != nil {
		out.Campaign = s.campaignRef(ctx, ws, *cid, false)
	}
	if b.Status == "proposed" {
		out.Proposal = proposalOf(b.ID, b.ProposedName, b.ProposedNote, b.CreatedAt, b.SponsorID, b.SponsorName, b.PriceCents)
	}
	events, err := s.Q.SponsorBookingEvents(ctx, b.ID)
	if err != nil {
		return out, err
	}
	for _, e := range events {
		var d map[string]any
		_ = json.Unmarshal(e.Data, &d)
		if d == nil {
			d = map[string]any{}
		}
		out.Activity = append(out.Activity, SponsorBookingEvent{Kind: e.Kind, At: e.OccurredAt, Actor: e.ActorName, Data: d})
	}
	if r, err := s.report(ctx, ws, b, slot); err != nil {
		return out, err
	} else if r != nil {
		out.Report = r
	}
	if out.CanShare {
		bid := b.ID
		if out.Shares, err = s.shares(ctx, ws, &bid, nil); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Issue is an open issue's panel: its dates and price, its campaign, and
// any proposal; a booked or held issue names its booking.
func (s *Service) Issue(ctx context.Context, ws *store.Workspace, p *authz.Principal, slotID uuid.UUID, number int) (SponsorIssueDetail, error) {
	slot, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: slotID, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorIssueDetail{}, mapErr(ctx, err, "slot")
	}
	if number < 1 {
		return SponsorIssueDetail{}, notFound(ctx, "issue")
	}
	sl := store.SponsorSlot{ID: slot.ID, FirstNumber: slot.FirstNumber, FirstOn: slot.FirstOn, EveryDays: slot.EveryDays}
	on := IssueOn(sl, number)
	out := SponsorIssueDetail{SlotID: slot.ID.String(), SlotName: slot.Name, ListName: slot.ListName, Placement: slot.Placement, Issue: number,
		IssueOn: noonOn(on, ws), PriceCents: slot.PriceCents, Currency: slot.Currency, CreativeDue: noonOn(on.AddDate(0, 0, -int(slot.CreativeDays)), ws),
		TermsDays: 14, Proposals: []SponsorPropose{}, Library: s.libraryRefs(ctx, ws), CanManage: p.Can(authz.SponsorsManage)}
	out.PaymentDue = noonOn(on.AddDate(0, 0, out.TermsDays), ws)
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, authz.SponsorsManage, p.UserID, 3)
	} else {
		out.Helpers = []string{}
	}
	if b, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: slot.ID, IssueNumber: int32(number)}); err == nil {
		out.BookingID = b.ID.String()
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if cid := s.campaignOn(ctx, ws, slot.ListID, on); cid != nil {
		out.Campaign = s.campaignRef(ctx, ws, *cid, false)
	}
	props, err := s.Q.SponsorBookingProposals(ctx, store.SponsorBookingProposalsParams{SlotID: slot.ID, IssueNumber: int32(number)})
	if err != nil {
		return out, err
	}
	for _, pr := range props {
		name := ""
		if pr.SponsorID != nil {
			if sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: *pr.SponsorID, WorkspaceID: ws.ID}); err == nil {
				name = sp.Name
			}
		}
		out.Proposals = append(out.Proposals, *proposalOf(pr.ID, pr.ProposedName, pr.ProposedNote, pr.CreatedAt, pr.SponsorID, name, pr.PriceCents))
	}
	return out, nil
}
