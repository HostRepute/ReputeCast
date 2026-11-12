package earn

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Teams (M65, G-38): a tier's group prices are per seat, for a range of
// seats. A team checkout is one Stripe subscription with a quantity, the
// buyer's; it carries a team (earn_groups) whose buyer uses one seat and
// invites people to the others (earn_group_seats). Someone who accepts gets
// their own paid_subscriptions row (kind and status group, amount 0) that
// follows the buyer's: syncGroupSeats copies its tier, pauses and end after
// every write to it, in the same transaction. Seats come from the
// subscription's quantity: when Stripe lowers it below the people in the
// team, nobody is removed automatically; the buyer is told, and the manage
// page asks them to remove people.

// Metadata of a team checkout and its subscription.
const (
	MetaGroup = "reputecast_group"
	MetaTeam  = "reputecast_team"
)

// How long the team links work: the buyer's manage page a week, an
// invite two weeks.
const (
	GroupLinkValid = 7 * 24 * time.Hour
	InviteValid    = 14 * 24 * time.Hour
)

// Audiences of a price.
const (
	AudienceIndividual = "individual"
	AudienceGroup      = "group"
)

// MaxInvites is how many addresses one invite may carry.
const MaxInvites = 50

// groupPrice is a tier's group price for a period in a mode.
type groupPrice struct {
	price    store.EarnPrice
	stripeID string
}

// groupPricesOf are a tier's current group prices by period, those a mode
// can sell (mode "": all of them, the editor's preview).
func groupPricesOf(prices []store.EarnPrice, tier uuid.UUID, mode string) map[string]groupPrice {
	out := map[string]groupPrice{}
	for _, p := range prices {
		if p.TierID != tier || p.Status != "current" || p.Audience != AudienceGroup {
			continue
		}
		sid := p.StripeLiveID
		if mode == "test" {
			sid = p.StripeTestID
		}
		if mode != "" && sid == "" {
			continue
		}
		out[p.Period] = groupPrice{price: p, stripeID: sid}
	}
	return out
}

// managePath is a team's manage page: its link works for GroupLinkValid.
func (s *Service) manageURL(ctx context.Context, ws *store.Workspace, g store.EarnGroup) string {
	t := s.Links.Sign(linktoken.Claims{Kind: TokenGroupManage, WorkspaceID: ws.ID, MessageID: g.ID, ContactID: g.OwnerContactID, Link: linktoken.IssueDay(s.now())})
	return s.pageURL(ctx, ws, "team/"+t)
}

// ManageTeamURL is a team's manage link, for the buyer's account page.
func (s *Service) ManageTeamURL(ctx context.Context, ws *store.Workspace, g store.EarnGroup) string {
	return s.manageURL(ctx, ws, g)
}

// inviteToken signs a seat's invite; its hash is what the seat keeps.
func (s *Service) inviteToken(ws uuid.UUID, seat uuid.UUID) (string, []byte) {
	t := s.Links.Sign(linktoken.Claims{Kind: TokenGroupJoin, WorkspaceID: ws, MessageID: seat, Link: linktoken.IssueDay(s.now())})
	h := sha256.Sum256([]byte(t))
	return t, h[:]
}

// syncGroupSeats keeps a buyer's team in step with the buyer's
// subscription after a write to it, in the write's transaction: the team
// is made the first time a team subscription is seen (sub's metadata), its
// seats follow the quantity, and every seat copies the buyer's tier,
// access rule pause, pause and end. It returns the emails to send once
// the transaction commits: the end of a seat's access, and the buyer's
// notice when the seats are fewer than the people in the team.
func (s *Service) syncGroupSeats(ctx context.Context, q *store.Queries, ws *store.Workspace, buyerID uuid.UUID, sub *payments.Subscription, at time.Time) ([]memberEmail, error) {
	buyer, err := q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: buyerID, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil || buyer.Kind != "stripe" {
		return nil, err
	}
	g, err := q.EarnGroupBySubscription(ctx, buyer.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		if sub == nil || sub.Metadata[MetaGroup] != "1" {
			return nil, nil
		}
		name := strings.TrimSpace(sub.Metadata[MetaTeam])
		g, err = q.EarnGroupInsert(ctx, store.EarnGroupInsertParams{ID: newID(), WorkspaceID: ws.ID, SubscriptionID: buyer.ID, OwnerContactID: buyer.ContactID,
			Name: clip(firstNonEmpty(name, "Team"), 80), Seats: buyer.Quantity, At: at})
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	var out []memberEmail
	if g.Seats != buyer.Quantity {
		people, err := q.EarnGroupPeople(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		fits := people.People+1 <= int64(buyer.Quantity)
		if err := q.EarnGroupSetSeats(ctx, store.EarnGroupSetSeatsParams{ID: g.ID, Seats: buyer.Quantity, Fits: fits, At: at}); err != nil {
			return nil, err
		}
		if !fits {
			if _, err := q.EarnGroupOverNotified(ctx, store.EarnGroupOverNotifiedParams{ID: g.ID, At: at}); err == nil {
				g.Seats = buyer.Quantity
				out = append(out, s.seatsOverEmail(ctx, q, ws, g, buyer, int(people.People+1)-int(buyer.Quantity)))
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
	}
	changed, err := q.EarnGroupSyncSeats(ctx, store.EarnGroupSyncSeatsParams{GroupID: g.ID, At: at})
	if err != nil {
		return nil, err
	}
	tier := s.tierShortName(ctx, q, ws.ID, buyer.TierID)
	for _, c := range changed {
		if !c.Ended {
			continue
		}
		row, err := q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: c.ID, WorkspaceID: ws.ID})
		if err != nil {
			return nil, err
		}
		if err := s.timeline(ctx, q, ws.ID, c.ContactID, &c.ID, row.Mode, "access_ended", at, nil, "", nil, "", nil,
			map[string]any{"reason": "group_ended", "team": g.Name, "tier": tier}); err != nil {
			return nil, err
		}
		m := s.memberEmailFor(EmailGroupAccessEnded, row, ws, tier, at)
		m.data["team"] = g.Name
		out = append(out, m)
	}
	if err := q.EarnGroupSetEnded(ctx, store.EarnGroupSetEndedParams{ID: g.ID, EndedAt: buyer.EndedAt, At: at}); err != nil {
		return nil, err
	}
	return out, nil
}

// afterWrite runs after every write to a subscription: its team's seats
// follow it, then its automation events are queued. The returned emails
// are sent once the write commits.
func (s *Service) afterWrite(ctx context.Context, q *store.Queries, ws *store.Workspace, mode string, before *store.PaidSubscription, subID uuid.UUID,
	at time.Time, sub *payments.Subscription) ([]memberEmail, error) {
	emails, err := s.syncGroupSeats(ctx, q, ws, subID, sub, at)
	if err != nil {
		return nil, err
	}
	if err := s.paidTransitions(ctx, q, ws, mode, before, subID, at); err != nil {
		return nil, err
	}
	return emails, nil
}

// seatsOverEmail tells a buyer their team has more people than seats.
func (s *Service) seatsOverEmail(ctx context.Context, q *store.Queries, ws *store.Workspace, g store.EarnGroup, buyer store.PaidSubscription, over int) memberEmail {
	tier := s.tierShortName(ctx, q, ws.ID, buyer.TierID)
	m := memberEmail{system: true, email: "team_seats_over", sub: buyer.ID, contact: buyer.ContactID, key: fmt.Sprintf("earn:%s:seats_over:%d", g.ID, buyer.Quantity),
		data: map[string]any{"team": g.Name, "tier": tier, "seats": buyer.Quantity, "over": over, "manage": s.manageURL(ctx, ws, g)}}
	return m
}

// EarnGroupPriceInput is a tier's group prices: per seat, for seatsMin to
// seatsMax seats (2–1000). An amount of 0 turns that period off.
type EarnGroupPriceInput struct {
	// Off removes the tier's group prices (teams that pay them keep them).
	Off      bool  `json:"off,omitempty"`
	Monthly  int64 `json:"monthly,omitempty" minimum:"0" maximum:"10000000" doc:"Per seat per month, in minor units"`
	Yearly   int64 `json:"yearly,omitempty" minimum:"0" maximum:"100000000" doc:"Per seat per year, in minor units"`
	SeatsMin int   `json:"seatsMin,omitempty" minimum:"0" maximum:"1000"`
	SeatsMax int   `json:"seatsMax,omitempty" minimum:"0" maximum:"1000"`
}

// EarnGroupPrices is a tier's group prices.
type EarnGroupPrices struct {
	Monthly  *EarnPrice `json:"monthly,omitempty"`
	Yearly   *EarnPrice `json:"yearly,omitempty"`
	SeatsMin int        `json:"seatsMin"`
	SeatsMax int        `json:"seatsMax"`
	// Teams is how many teams buy the tier now, and Seats their seats.
	Teams int64 `json:"teams"`
	Seats int64 `json:"seats"`
}

// validateGroup checks a tier's group prices.
func validateGroup(ctx context.Context, g *EarnGroupPriceInput) error {
	if g == nil || g.Off {
		return nil
	}
	if g.Monthly <= 0 && g.Yearly <= 0 {
		return invalid(ctx, "body.group.monthly", "tier.groupNoPrice", nil)
	}
	if g.SeatsMin < 2 || g.SeatsMax > 1000 || g.SeatsMin > g.SeatsMax {
		return invalid(ctx, "body.group.seatsMax", "tier.groupSeats", nil)
	}
	return nil
}

// saveGroupPrices writes a tier's group prices: a changed amount is a new
// price (the old one becomes legacy for the teams that pay it), a changed
// seat range changes in place. It returns the audit lines.
func saveGroupPrices(ctx context.Context, q *store.Queries, ws *store.Workspace, tier uuid.UUID, cur string, g *EarnGroupPriceInput, prices []store.EarnPrice, now time.Time) ([]map[string]any, error) {
	out := []map[string]any{}
	for _, pr := range []struct {
		period string
		amount int64
	}{{"month", groupAmount(g, "month")}, {"year", groupAmount(g, "year")}} {
		var current *store.EarnPrice
		for i := range prices {
			if prices[i].Period == pr.period && prices[i].Status == "current" && prices[i].Audience == AudienceGroup {
				current = &prices[i]
			}
		}
		if current != nil && current.AmountCents == pr.amount && g != nil {
			if int(deref32(current.SeatsMin)) != g.SeatsMin || int(deref32(current.SeatsMax)) != g.SeatsMax {
				if err := q.EarnPriceSetSeats(ctx, store.EarnPriceSetSeatsParams{ID: current.ID, SeatsMin: ptr(int32(g.SeatsMin)), SeatsMax: ptr(int32(g.SeatsMax))}); err != nil {
					return nil, err
				}
				out = append(out, map[string]any{"period": "group_" + pr.period, "seats": fmt.Sprintf("%d–%d", g.SeatsMin, g.SeatsMax)})
			}
			continue
		}
		line := map[string]any{"period": "group_" + pr.period}
		if current != nil {
			if err := q.EarnPriceRetire(ctx, store.EarnPriceRetireParams{ID: current.ID, At: &now}); err != nil {
				return nil, err
			}
			line["old"] = priceText(current.AmountCents, current.Currency, pr.period)
		}
		if pr.amount > 0 {
			if err := q.EarnPriceInsert(ctx, store.EarnPriceInsertParams{ID: newID(), WorkspaceID: ws.ID, TierID: tier, Period: pr.period, AmountCents: pr.amount,
				Currency: cur, Status: "current", Origin: "tier", At: now, Audience: ptr(AudienceGroup), SeatsMin: ptr(int32(g.SeatsMin)), SeatsMax: ptr(int32(g.SeatsMax))}); err != nil {
				return nil, err
			}
			line["new"] = priceText(pr.amount, cur, pr.period)
		}
		if line["old"] != nil || line["new"] != nil {
			out = append(out, line)
		}
	}
	return out, nil
}

func groupAmount(g *EarnGroupPriceInput, period string) int64 {
	if g == nil {
		return 0
	}
	if period == "month" {
		return g.Monthly
	}
	return g.Yearly
}

func deref32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

// TeamCheckoutRequest is a team's checkout from the team page.
type TeamCheckoutRequest struct {
	Tier   string // the tier's slug
	Period string
	// Seats is the number typed (the stepper's).
	Seats     string
	Team      string
	Email     string
	Token     string
	TestToken string
	// Lang is the page's: the form's errors are written in it.
	Lang i18n.Lang
}

// TeamFormError is what the team page's form got wrong: a sentence per
// field, in the page's language.
type TeamFormError struct {
	Fields map[string]string
}

func (e *TeamFormError) Error() string { return "earn: the team form has errors" }

// teamTier finds the tier a team page sells: by slug, not archived, with a
// group price in the mode.
func (s *Service) teamTier(ctx context.Context, ws *store.Workspace, slug, mode string) (store.EarnTier, map[string]groupPrice, error) {
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return store.EarnTier{}, nil, err
	}
	for _, t := range tiers {
		if t.Slug != slug || t.Visibility == VisibilityArchived {
			continue
		}
		prices, err := s.Q.EarnTierPrices(ctx, t.ID)
		if err != nil {
			return store.EarnTier{}, nil, err
		}
		if gp := groupPricesOf(prices, t.ID, mode); len(gp) > 0 {
			return t, gp, nil
		}
	}
	return store.EarnTier{}, nil, ErrUnavailable
}

// StartTeamCheckout opens Stripe Checkout for a team: the group price per
// seat with the seats as its quantity, the buyer's tax ID and billing
// address collected, no trial. The team is made when the subscription
// arrives (its metadata names it).
func (s *Service) StartTeamCheckout(ctx context.Context, ws *store.Workspace, r TeamCheckoutRequest) (string, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return "", err
	}
	conn, ok := active(conns)
	test := r.TestToken != "" && s.TestMode(ws, r.TestToken)
	if test {
		conn, ok = connected(conns, "test")
	}
	if !ok {
		return "", ErrUnavailable
	}
	t, gp, err := s.teamTier(ctx, ws, r.Tier, conn.Mode)
	if err != nil {
		return "", err
	}
	p, ok := gp[r.Period]
	if !ok {
		for _, per := range []string{"year", "month"} {
			if x, has := gp[per]; has {
				p, ok, r.Period = x, true, per
				break
			}
		}
	}
	if !ok {
		return "", ErrUnavailable
	}
	team := strings.TrimSpace(r.Team)
	email := strings.ToLower(strings.TrimSpace(r.Email))
	lo, hi := int(deref32(p.price.SeatsMin)), int(deref32(p.price.SeatsMax))
	seats, serr := strconv.Atoi(strings.TrimSpace(r.Seats))
	bad := map[string]string{}
	if serr != nil || seats < lo || seats > hi {
		bad["seats"] = s.t(r.Lang, "team.form.seats", i18n.Args{"min": lo, "max": hi})
	}
	switch {
	case team == "":
		bad["team"] = s.t(r.Lang, "team.form.teamName", nil)
	case len([]rune(team)) > 80:
		bad["team"] = s.t(r.Lang, "team.form.teamLong", nil)
	}
	if !emailaddr.Valid(email) {
		bad["email"] = s.t(r.Lang, "team.form.email", nil)
	}
	if len(bad) > 0 {
		return "", &TeamFormError{Fields: bad}
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return "", err
	}
	chkID := newID()
	req := payments.CheckoutRequest{PriceID: p.stripeID, Quantity: seats, Business: true, AllowCodes: true, Tax: st.TaxAtCheckout && conn.TaxEnabled,
		CustomerEmail: email, ClientRef: chkID.String(),
		SuccessURL: s.pageURL(ctx, ws, "upgrade/welcome?session={CHECKOUT_SESSION_ID}"),
		CancelURL:  s.pageURL(ctx, ws, "upgrade/"+t.Slug+"/team") + "?canceled=1",
		Metadata:   map[string]string{"reputecast_checkout": chkID.String(), "reputecast_tier": t.ID.String(), MetaGroup: "1", MetaTeam: team}}
	attribution := map[string]any{"kind": "page"}
	var contact *uuid.UUID
	if ct, claims, ok := s.readerOf(ctx, ws, r.Token); ok {
		id := ct.ID
		contact = &id
		if claims.MessageID != uuid.Nil {
			if a := s.emailAttribution(ctx, claims.MessageID); a != nil {
				attribution = a
			}
		}
	}
	if test {
		attribution = map[string]any{"kind": "test"}
	}
	attr, _ := json.Marshal(attribution)
	now := s.now()
	pid := p.price.ID
	if err := s.Q.EarnCheckoutInsert(ctx, store.EarnCheckoutInsertParams{ID: chkID, WorkspaceID: ws.ID, Mode: conn.Mode, TierID: &t.ID, PriceID: &pid, Period: r.Period,
		ContactID: contact, Email: email, Status: "open", Attribution: attr, CreatedAt: now, Quantity: ptr(int32(seats)), TeamName: &team}); err != nil {
		return "", err
	}
	client, err := s.open(conn.PaymentConnection)
	if err != nil {
		return "", ErrUnavailable
	}
	cs, err := client.CreateCheckout(ctx, req)
	if err != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE earn_checkouts SET status = 'failed' WHERE id = $1`, chkID)
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: conn.ID, At: now})
		}
		return "", fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if err := s.Q.EarnCheckoutSetSession(ctx, store.EarnCheckoutSetSessionParams{ID: chkID, StripeSessionID: cs.ID, Status: "open"}); err != nil {
		return "", err
	}
	return cs.URL, nil
}

// EarnTeamSeat is one person of a team in the member panel.
type EarnTeamSeat struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	// Status is invited or active; the buyer is the team's first row.
	Status     string     `json:"status" enum:"buyer,invited,active"`
	InvitedAt  time.Time  `json:"invitedAt"`
	AcceptedAt *time.Time `json:"acceptedAt,omitempty"`
	// InviteExpiresAt is when an invite stops working.
	InviteExpiresAt *time.Time `json:"inviteExpiresAt,omitempty"`
	// MemberID opens an active seat holder's own panel.
	MemberID string `json:"memberId,omitempty"`
}

// EarnMemberTeam is a paid member's team: the buyer's ("Team · 8 of 10")
// or the one a seat holder belongs to ("Seat in Acme Inc").
type EarnMemberTeam struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role" enum:"buyer,seat"`
	Seats int    `json:"seats"`
	// Used counts the buyer and every invited or active seat.
	Used  int        `json:"used"`
	Ended *time.Time `json:"endedAt,omitempty"`
	// BuyerID opens the buyer's panel from a seat holder's; BuyerName
	// names them. Invites counts the invites still open.
	BuyerID   string `json:"buyerId,omitempty"`
	BuyerName string `json:"buyerName,omitempty"`
	Invites   int    `json:"invites"`
	// RenewsAt is when the team's subscription renews (its seats last
	// with it).
	RenewsAt *time.Time `json:"renewsAt,omitempty"`
	// SeatID is a seat holder's seat (Remove seat from their panel).
	SeatID string `json:"seatId,omitempty"`
}

// teamsOf names the teams of a page of members, by subscription id.
func (s *Service) teamsOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*EarnMemberTeam, error) {
	out := map[uuid.UUID]*EarnMemberTeam{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Q.EarnGroupsOfSubscriptions(ctx, ids)
	if err != nil {
		return nil, err
	}
	byGroup := map[uuid.UUID]store.EarnGroupsOfSubscriptionsRow{}
	for _, r := range rows {
		byGroup[r.ID] = r
		if slices.Contains(ids, r.SubscriptionID) {
			out[r.SubscriptionID] = &EarnMemberTeam{ID: r.ID.String(), Name: r.Name, Role: "buyer", Seats: int(r.Seats), Used: int(r.People) + 1, Ended: r.EndedAt,
				Invites: int(r.Invites), BuyerName: r.BuyerName, RenewsAt: r.RenewsAt}
		}
	}
	if len(byGroup) == 0 {
		return out, nil
	}
	seats, err := s.Pool.Query(ctx, `SELECT s.id, s.group_id, coalesce(g.id::text, '') FROM paid_subscriptions s
		LEFT JOIN earn_group_seats g ON g.subscription_id = s.id AND g.status = 'active'
		WHERE s.id = ANY($1) AND s.group_id IS NOT NULL AND s.kind = 'group'`, ids)
	if err != nil {
		return nil, err
	}
	defer seats.Close()
	for seats.Next() {
		var id, gid uuid.UUID
		var seat string
		if err := seats.Scan(&id, &gid, &seat); err != nil {
			return nil, err
		}
		if r, ok := byGroup[gid]; ok {
			out[id] = &EarnMemberTeam{ID: r.ID.String(), Name: r.Name, Role: "seat", Seats: int(r.Seats), Used: int(r.People) + 1, Ended: r.EndedAt,
				BuyerID: r.SubscriptionID.String(), BuyerName: r.BuyerName, Invites: int(r.Invites), RenewsAt: r.RenewsAt, SeatID: seat}
		}
	}
	return out, seats.Err()
}

// teamSeats lists a buyer's team for the member panel: the buyer first.
func (s *Service) teamSeats(ctx context.Context, buyer store.PaidSubscription, buyerEmail, buyerName string, hidden bool) ([]EarnTeamSeat, error) {
	g, err := s.Q.EarnGroupBySubscription(ctx, buyer.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := []EarnTeamSeat{{ID: buyer.ID.String(), Email: buyerEmail, Name: buyerName, Status: "buyer", InvitedAt: g.CreatedAt, MemberID: buyer.ID.String()}}
	rows, err := s.Q.EarnGroupSeats(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		seat := EarnTeamSeat{ID: r.ID.String(), Email: r.Email, Name: strings.TrimSpace(r.FirstName + " " + r.LastName), Status: r.Status, InvitedAt: r.InvitedAt,
			AcceptedAt: r.AcceptedAt, InviteExpiresAt: r.InviteExpiresAt}
		if r.SubscriptionID != nil {
			seat.MemberID = r.SubscriptionID.String()
		}
		if hidden {
			seat.Email, seat.Name = "", ""
		}
		out = append(out, seat)
	}
	return out, nil
}

// RemoveSeat removes a person from a team from Earn › Members (the
// buyer's panel): an invite stops working, an active seat's access ends now.
func (s *Service) RemoveSeat(ctx context.Context, ws *store.Workspace, p *authz.Principal, memberID, seatID string, email bool) (EarnMember, error) {
	sid, err1 := uuid.Parse(memberID)
	seat, err2 := uuid.Parse(seatID)
	if err1 != nil || err2 != nil {
		return EarnMember{}, notFound(ctx, "paidMember")
	}
	buyer, err := s.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sid, WorkspaceID: ws.ID})
	if err != nil {
		return EarnMember{}, mapErr(ctx, err, "paidMember")
	}
	g, err := s.Q.EarnGroupBySubscription(ctx, buyer.ID)
	if err != nil {
		return EarnMember{}, mapErr(ctx, err, "team")
	}
	var emails []memberEmail
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, m, err := s.removeSeat(ctx, q, ws, g, seat, s.now())
		if err != nil {
			return err
		}
		emails = m
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.seat_removed", TargetType: "contact", TargetID: buyer.ContactID.String(),
			Before: map[string]any{"seat": row.Email, "status": row.Status}, Metadata: map[string]any{"team": g.Name, "group": g.ID.String()}})
	})
	if err != nil {
		return EarnMember{}, err
	}
	for _, m := range emails {
		if email {
			s.sendMemberEmail(ctx, ws, buyer.Mode, m, nil)
		}
	}
	return s.Member(ctx, ws, p, buyer.ID.String())
}

// removeSeat removes a seat in a transaction and ends its access.
func (s *Service) removeSeat(ctx context.Context, q *store.Queries, ws *store.Workspace, g store.EarnGroup, seatID uuid.UUID, now time.Time) (store.EarnGroupSeat, []memberEmail, error) {
	row, err := q.EarnGroupSeatForUpdate(ctx, store.EarnGroupSeatForUpdateParams{ID: seatID, GroupID: g.ID})
	if err != nil {
		return row, nil, mapErr(ctx, err, "seat")
	}
	if row.Status == "removed" {
		return row, nil, problem(ctx, http.StatusConflict, "seat_removed", "team.seatGone", nil)
	}
	if err := q.EarnGroupSeatRemove(ctx, store.EarnGroupSeatRemoveParams{ID: row.ID, At: now}); err != nil {
		return row, nil, err
	}
	var out []memberEmail
	if row.SubscriptionID != nil {
		if _, err := q.EarnGroupSeatEndAccess(ctx, store.EarnGroupSeatEndAccessParams{ID: *row.SubscriptionID, At: now, EndReason: "seat_removed"}); err == nil {
			sub, err := q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: *row.SubscriptionID, WorkspaceID: ws.ID})
			if err != nil {
				return row, nil, err
			}
			tier := s.tierShortName(ctx, q, ws.ID, sub.TierID)
			if err := s.timeline(ctx, q, ws.ID, sub.ContactID, &sub.ID, sub.Mode, "access_ended", now, nil, "", nil, "", nil,
				map[string]any{"reason": "seat_removed", "team": g.Name, "tier": tier}); err != nil {
				return row, nil, err
			}
			m := s.memberEmailFor(EmailGroupAccessEnded, sub, ws, tier, now)
			m.data["team"] = g.Name
			out = append(out, m)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return row, nil, err
		}
	}
	// Removing someone may make the team fit its seats again.
	people, err := q.EarnGroupPeople(ctx, g.ID)
	if err != nil {
		return row, nil, err
	}
	if people.People+1 <= int64(g.Seats) {
		if err := q.EarnGroupSetSeats(ctx, store.EarnGroupSetSeatsParams{ID: g.ID, Seats: g.Seats, Fits: true, At: now}); err != nil {
			return row, nil, err
		}
	}
	return row, out, nil
}

// groupView adds a current group price to a tier's group prices.
func (s *Service) groupView(g *EarnGroupPrices, p store.EarnPrice, mode payments.Mode, counts map[string]int64) *EarnGroupPrices {
	if g == nil {
		g = &EarnGroupPrices{}
	}
	ep := EarnPrice{ID: p.ID.String(), Period: p.Period, AmountCents: p.AmountCents, Currency: p.Currency, Status: p.Status, StripeLiveID: p.StripeLiveID,
		StripeTestID: p.StripeTestID, Origin: p.Origin, Paying: counts[StatusActive] + counts[StatusPastDue], Offers: []EarnPriceOffer{}}
	ep.Members = ep.Paying + counts[StatusTrialing]
	ep.StripeID = p.StripeLiveID
	if mode == payments.Test {
		ep.StripeID = p.StripeTestID
	}
	if ep.StripeID != "" {
		ep.StripeURL = dashboard(DefaultProvider, mode, "prices", ep.StripeID)
	}
	if p.Period == "month" {
		g.Monthly = &ep
	} else {
		g.Yearly = &ep
	}
	g.SeatsMin, g.SeatsMax = int(deref32(p.SeatsMin)), int(deref32(p.SeatsMax))
	return g
}

// teamsOfTier counts the teams that pay for a tier now and their seats.
func (s *Service) teamsOfTier(ctx context.Context, tier uuid.UUID) (int64, int64) {
	var teams, seats int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*), coalesce(sum(g.seats), 0) FROM earn_groups g JOIN paid_subscriptions s ON s.id = g.subscription_id
		WHERE s.tier_id = $1 AND g.ended_at IS NULL AND s.ended_at IS NULL`, tier).Scan(&teams, &seats)
	return teams, seats
}
