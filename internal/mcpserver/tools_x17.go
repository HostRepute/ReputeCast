package mcpserver

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/revenue"
	"github.com/hostrepute/reputecast/internal/sponsors"
	"github.com/hostrepute/reputecast/internal/store"
)

// Earn's revenue and sponsorships for agents (design expansion phase 6,
// X17): agents read revenue numbers and sponsor bookings, and can propose
// a booking for a person to confirm. They never see payment credentials,
// change prices or offers, grant access, cancel subscriptions or record
// payments.

// RevenueReader reads Earn › Overview (internal/revenue).
type RevenueReader interface {
	Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal, in revenue.OverviewQuery) (revenue.RevenueOverview, error)
}

// SponsorProposer records an agent's proposed booking (internal/sponsors).
type SponsorProposer interface {
	Propose(ctx context.Context, ws *store.Workspace, p *authz.Principal, in sponsors.ProposeInput) (sponsors.SponsorIssueDetail, error)
}

func init() {
	Tools = append(Tools,
		MCPToolInfo{"get_revenue_overview", "Earn's numbers for the last 30 days, 90 days or 12 months: MRR, ARR, paid members, new and churned, churn rate, free-to-paid rate, gift and sponsor income. Totals only, never names or emails.", string(authz.EarnView), false},
		MCPToolInfo{"list_sponsor_bookings", "Sponsor bookings by issue and slot, from 30 days ago on: the sponsor, the status, the creative's state and the payment's. Never contact data.", string(authz.EarnView), false},
		MCPToolInfo{"propose_sponsor_booking", "Propose a sponsor for an open issue of a slot. A team member confirms it before it's booked.", string(authz.SponsorsManage), true},
	)
}

type revenueIn struct {
	Period string `json:"period,omitempty" jsonschema:"30d (the default), 90d or 12m"`
}

type bookingsIn struct {
	Sponsor string `json:"sponsor,omitempty" jsonschema:"only this sponsor's bookings (its name)"`
}

type proposeIn struct {
	Slot    string `json:"slot" jsonschema:"the slot's name (Main sponsor) or id"`
	Issue   int    `json:"issue" jsonschema:"the issue number, such as 146"`
	Sponsor string `json:"sponsor" jsonschema:"the sponsor's name or id, from the sponsor library"`
	// PriceCents defaults to the slot's price.
	PriceCents *int64 `json:"price_cents,omitempty" jsonschema:"the price in cents; the slot's price by default"`
	Note       string `json:"note,omitempty" jsonschema:"why, for the person who confirms (up to 500 characters)"`
}

// addX17Tools registers the revenue and sponsor tools.
func (s *Service) addX17Tools(srv *mcp.Server) {
	tool(s, srv, "get_revenue_overview", authz.EarnView, func(ctx context.Context, c *agentCall, in revenueIn) (any, entry, error) {
		e := entry{action: "earn.revenue_read", targetType: "workspace", targetID: c.Workspace.ID.String(), targetName: c.Workspace.Name}
		if s.Revenue == nil {
			return map[string]any{"message": "Revenue isn't available on this server."}, e, nil
		}
		period := strings.TrimSpace(in.Period)
		switch period {
		case "":
			period = "30d"
		case "30d", "90d", "12m":
		default:
			return nil, e, errors.New("period is 30d, 90d or 12m")
		}
		ws := c.Workspace
		o, err := s.Revenue.Overview(ctx, &ws, c.Principal, revenue.OverviewQuery{Period: period})
		if err != nil {
			return nil, e, err
		}
		t := o.Tiles
		out := map[string]any{"period": o.Period, "from": o.From, "to": o.To, "currency": o.Currency, "mrr_cents": t.MRRCents, "mrr_change_cents": t.MRRDeltaCents,
			"arr_cents": t.ARRCents, "paid_members": t.Members, "paid_members_change": t.MembersDelta, "new": t.New, "churned": t.Churned, "trialing": t.Trialing,
			"free_to_paid": t.FreeToPaid, "gift_income_cents": t.GiftsCents, "sponsor_income_cents": t.SponsorsCents,
			"note": "MRR counts Active and Past due subscriptions after discounts, a yearly price as a twelfth; trials, free periods, complimentary access, gifts, sponsors and conversions add nothing to it. Churn counts when access ends."}
		if t.ChurnRate != nil {
			out["churn_rate_percent"] = *t.ChurnRate
		}
		if t.FreeToPaidRate != nil {
			out["free_to_paid_rate_percent"] = *t.FreeToPaidRate
		}
		if o.TooEarly {
			out["too_early"] = true
		}
		return out, e, nil
	}, nil)

	tool(s, srv, "list_sponsor_bookings", authz.EarnView, func(ctx context.Context, c *agentCall, in bookingsIn) (any, entry, error) {
		e := entry{action: "sponsors.bookings_read", targetType: "workspace", targetID: c.Workspace.ID.String(), targetName: c.Workspace.Name}
		rows, err := s.Q.SponsorBookingsOfWorkspace(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		slots, err := s.Q.SponsorSlots(ctx, c.Workspace.ID)
		if err != nil {
			return nil, e, err
		}
		slotName := map[uuid.UUID]string{}
		for _, sl := range slots {
			slotName[sl.ID] = sl.Name
		}
		since := s.now().AddDate(0, 0, -30)
		type booking struct {
			Issue          int        `json:"issue"`
			IssueOn        string     `json:"issue_on"`
			Slot           string     `json:"slot"`
			Sponsor        string     `json:"sponsor,omitempty"`
			Status         string     `json:"status"`
			HoldUntil      *time.Time `json:"hold_until,omitempty"`
			PriceCents     int64      `json:"price_cents"`
			Currency       string     `json:"currency"`
			CreativeStatus string     `json:"creative_status"`
			CreativeDue    string     `json:"creative_due,omitempty"`
			PaymentStatus  string     `json:"payment_status"`
			ProposedBy     string     `json:"proposed_by,omitempty"`
		}
		out := []booking{}
		filter := strings.ToLower(strings.TrimSpace(in.Sponsor))
		for _, b := range rows {
			if b.Status == "canceled" || b.IssueOn.Before(since) {
				continue
			}
			if filter != "" && !strings.Contains(strings.ToLower(b.SponsorName), filter) {
				continue
			}
			x := booking{Issue: int(b.IssueNumber), IssueOn: b.IssueOn.Format(time.DateOnly), Slot: slotName[b.SlotID], Sponsor: b.SponsorName, Status: b.Status,
				HoldUntil: b.HoldUntil, PriceCents: b.PriceCents, Currency: b.Currency, CreativeStatus: b.CreativeStatus, PaymentStatus: b.PaymentStatus, ProposedBy: b.ProposedName}
			if b.CreativeDue != nil {
				x.CreativeDue = b.CreativeDue.Format(time.DateOnly)
			}
			out = append(out, x)
		}
		return map[string]any{"bookings": out, "note": "Issues booked, on hold, proposed or sent from 30 days ago on. Open issues have no booking."}, e, nil
	}, nil)

	tool(s, srv, "propose_sponsor_booking", authz.SponsorsManage, func(ctx context.Context, c *agentCall, in proposeIn) (any, entry, error) {
		e := entry{action: "sponsors.booking_proposed", targetType: "sponsor_booking"}
		if s.Sponsors == nil {
			return nil, e, errors.New("sponsorships aren't available on this server")
		}
		slotID, slot, err := s.findSlot(ctx, c.Workspace.ID, in.Slot)
		if err != nil {
			return nil, e, err
		}
		sponsorID, sponsor, err := s.findSponsor(ctx, c.Workspace.ID, in.Sponsor)
		if err != nil {
			return nil, e, err
		}
		if in.Issue < 1 {
			return nil, e, errors.New("name the issue number, such as 146")
		}
		if in.PriceCents != nil && *in.PriceCents < 0 {
			return nil, e, errors.New("price_cents can't be negative")
		}
		ws := c.Workspace
		d, err := s.Sponsors.Propose(ctx, &ws, c.Principal, sponsors.ProposeInput{SlotID: slotID, Issue: in.Issue, SponsorID: sponsorID, PriceCents: in.PriceCents, Note: in.Note})
		if err != nil {
			return nil, e, err
		}
		e.targetName = slot + " · #" + itoa(in.Issue)
		e.detail = sponsor
		e.summaryText = keep("outcome.sponsor", i18n.Args{"sponsor": sponsor, "target": e.targetName})
		return map[string]any{"status": "proposed", "issue": in.Issue, "issue_on": d.IssueOn.Format(time.DateOnly), "slot": slot, "sponsor": sponsor,
			"message": "Proposed. A team member with sponsors:manage confirms it in Earn › Sponsors before it's booked."}, e, nil
	}, func(ctx context.Context, c *agentCall, in proposeIn) entry {
		return entry{action: "sponsors.booking_proposed", targetType: "sponsor_booking", targetName: in.Slot + " · #" + itoa(in.Issue), detail: in.Sponsor}
	})
}

// findSlot finds a slot by id or by name (case-insensitive).
func (s *Service) findSlot(ctx context.Context, ws uuid.UUID, ref string) (uuid.UUID, string, error) {
	slots, err := s.Q.SponsorSlots(ctx, ws)
	if err != nil {
		return uuid.Nil, "", err
	}
	ref = strings.TrimSpace(ref)
	for _, sl := range slots {
		if sl.ID.String() == ref || strings.EqualFold(sl.Name, ref) {
			return sl.ID, sl.Name, nil
		}
	}
	if len(slots) == 1 && ref == "" {
		return slots[0].ID, slots[0].Name, nil
	}
	return uuid.Nil, "", errors.New("no slot named " + quote(ref) + "; list_sponsor_bookings shows the slots")
}

// findSponsor finds a sponsor of the library by id or by name.
func (s *Service) findSponsor(ctx context.Context, ws uuid.UUID, ref string) (uuid.UUID, string, error) {
	list, err := s.Q.SponsorList(ctx, ws)
	if err != nil {
		return uuid.Nil, "", err
	}
	ref = strings.TrimSpace(ref)
	for _, sp := range list {
		if sp.ID.String() == ref || strings.EqualFold(sp.Name, ref) {
			return sp.ID, sp.Name, nil
		}
	}
	return uuid.Nil, "", errors.New("no sponsor named " + quote(ref) + " in the sponsor library")
}

func quote(s string) string { return "“" + s + "”" }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
