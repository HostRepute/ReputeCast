package sponsors

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// BookInput books an open issue for a sponsor, or holds it until a date.
type BookInput struct {
	SponsorID  string `json:"sponsorId" format:"uuid"`
	PriceCents *int64 `json:"priceCents,omitempty" minimum:"0" doc:"The slot's default price when absent."`
	// Status: booked, or hold until HoldUntil (then the issue opens again).
	Status    string `json:"status" enum:"booked,hold"`
	HoldUntil string `json:"holdUntil,omitempty" format:"date"`
	Notes     string `json:"notes,omitempty" maxLength:"2000"`
}

// BookingInput changes a booking: its sponsor, price, status or hold date.
type BookingInput struct {
	SponsorID  string  `json:"sponsorId,omitempty" format:"uuid"`
	PriceCents *int64  `json:"priceCents,omitempty" minimum:"0"`
	Status     string  `json:"status,omitempty" enum:"booked,hold,"`
	HoldUntil  string  `json:"holdUntil,omitempty" format:"date"`
	Notes      *string `json:"notes,omitempty" maxLength:"2000"`
}

// PaymentInput records a booking's payment, by hand.
type PaymentInput struct {
	Paid        bool   `json:"paid"`
	PaidOn      string `json:"paidOn,omitempty" format:"date"`
	AmountCents *int64 `json:"amountCents,omitempty" minimum:"0"`
	Note        string `json:"note,omitempty" maxLength:"300" doc:"Shown on the booking, never to the sponsor."`
}

// CampaignInput links an issue to its campaign by hand (empty: by list and date).
type CampaignInput struct {
	CampaignID string `json:"campaignId,omitempty" format:"uuid"`
}

// CancelInput cancels a booking or releases a hold.
type CancelInput struct {
	Reason string `json:"reason,omitempty" maxLength:"300"`
}

// ProposeInput is an MCP agent's proposed booking.
type ProposeInput struct {
	SlotID     uuid.UUID
	Issue      int
	SponsorID  uuid.UUID
	PriceCents *int64
	Note       string
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// defaultUTM are the UTM parameters a new booking's link gets:
// utm_source=acme-weekly-digest, utm_medium=email, utm_campaign=143.
func defaultUTM(ws *store.Workspace, listName string, issue int) SponsorUTMs {
	brand := ""
	if f := strings.Fields(ws.Name); len(f) > 0 {
		brand = f[0]
	}
	src := slugify(brand + " " + listName)
	if strings.HasPrefix(slugify(listName), slugify(brand)) {
		src = slugify(listName)
	}
	return SponsorUTMs{Source: src, Medium: "email", Campaign: strconv.Itoa(issue)}
}

// WithUTM is a creative's link with its UTM parameters.
func WithUTM(link string, u SponsorUTMs) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	if !strings.Contains(link, "://") {
		link = "https://" + link
	}
	p, err := url.Parse(link)
	if err != nil {
		return link
	}
	q := p.Query()
	for k, v := range map[string]string{"utm_source": u.Source, "utm_medium": u.Medium, "utm_campaign": u.Campaign} {
		if v != "" {
			q.Set(k, v)
		}
	}
	p.RawQuery = q.Encode()
	return p.String()
}

func parseCreative(raw json.RawMessage) SponsorCreative {
	var c SponsorCreative
	_ = json.Unmarshal(raw, &c)
	return c
}

func parseDay(ctx context.Context, field, v string) (*time.Time, error) {
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, invalid(ctx, field, "problem.date", nil)
	}
	return &d, nil
}

// event records a line of a booking's activity.
func (s *Service) event(ctx context.Context, q *store.Queries, booking uuid.UUID, p *authz.Principal, kind string, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	b, _ := json.Marshal(data)
	return q.SponsorBookingEventInsert(ctx, store.SponsorBookingEventInsertParams{ID: newID(), BookingID: booking, Kind: kind, OccurredAt: s.now(),
		ActorID: userID(p), ActorName: actorName(p), Data: b})
}

// Book books an open issue or holds it.
func (s *Service) Book(ctx context.Context, ws *store.Workspace, p *authz.Principal, slotID uuid.UUID, issue int, in BookInput) (SponsorBookingDetail, error) {
	slot, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: slotID, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorBookingDetail{}, api.MapError(err, "Slot")
	}
	sid, err := uuid.Parse(in.SponsorID)
	if err != nil {
		return SponsorBookingDetail{}, invalid(ctx, "body.sponsorId", "problem.pickSponsor", nil)
	}
	sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: sid, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorBookingDetail{}, invalid(ctx, "body.sponsorId", "problem.pickSponsor", nil)
	}
	slotRow := store.SponsorSlot{ID: slot.ID, FirstNumber: slot.FirstNumber, FirstOn: slot.FirstOn, EveryDays: slot.EveryDays}
	on := IssueOn(slotRow, issue)
	if on.Before(s.today(ws)) {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.issuePast", map[string]any{"issue": issue})
	}
	if _, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: slotID, IssueNumber: int32(issue)}); err == nil {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.issueTaken", map[string]any{"issue": issue})
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return SponsorBookingDetail{}, err
	}
	status := in.Status
	if status == "" {
		status = "booked"
	}
	hold, err := parseDay(ctx, "body.holdUntil", in.HoldUntil)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	if status == "hold" {
		if hold == nil {
			return SponsorBookingDetail{}, invalid(ctx, "body.holdUntil", "problem.holdDate", nil)
		}
		if !hold.Before(on) || hold.Before(s.today(ws)) {
			return SponsorBookingDetail{}, invalid(ctx, "body.holdUntil", "problem.holdBeforeIssue", map[string]any{"issue": issue})
		}
	} else {
		hold = nil
	}
	price := slot.PriceCents
	if in.PriceCents != nil {
		price = *in.PriceCents
	}
	creativeDue := on.AddDate(0, 0, -int(slot.CreativeDays))
	paymentDue := on.AddDate(0, 0, int(sp.TermsDays))
	cr := SponsorCreative{UTM: defaultUTM(ws, slot.ListName, issue), Logo: sp.LogoUrl}
	if sp.Website != "" {
		cr.Link = "https://" + sp.Website
	}
	crRaw, _ := json.Marshal(cr)
	id := newID()
	now := s.now()
	campaign := s.campaignOn(ctx, ws, slot.ListID, on)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingInsert(ctx, store.SponsorBookingInsertParams{ID: id, WorkspaceID: ws.ID, SlotID: slotID, IssueNumber: int32(issue), IssueOn: on,
			SponsorID: &sid, CampaignID: campaign, PriceCents: price, Currency: slot.Currency, Status: status, HoldUntil: hold, Creative: crRaw,
			CreativeStatus: "waiting", CreativeDue: &creativeDue, PaymentDue: &paymentDue, Notes: strings.TrimSpace(in.Notes), CreatedBy: userID(p), At: now}); err != nil {
			if isUnique(err) {
				return problem(ctx, 409, api.CodeConflict, "problem.issueTaken", map[string]any{"issue": issue})
			}
			return err
		}
		kind := "booked"
		if status == "hold" {
			kind = "held"
		}
		if err := s.event(ctx, q, id, p, kind, map[string]any{"sponsor": sp.Name, "priceCents": price, "holdUntil": dayString(hold)}); err != nil {
			return err
		}
		action := "sponsors.booking_created"
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, issue, slot.Name),
			After:      map[string]any{"sponsor": sp.Name, "status": status, "priceCents": price, "holdUntil": dayString(hold), "issueOn": on.Format("2006-01-02")}})
	})
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	return s.Booking(ctx, ws, p, id)
}

func dayString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// issueName names an issue for the audit log: "Weekly Digest #143 · Main sponsor".
func issueName(list string, issue int, slot string) string {
	return list + " #" + strconv.Itoa(issue) + " · " + slot
}

// campaignOn finds the campaign of a list on an issue's date (by list and
// date): one scheduled or sent that day in the workspace's time zone.
func (s *Service) campaignOn(ctx context.Context, ws *store.Workspace, list uuid.UUID, day time.Time) *uuid.UUID {
	from, to := dayRange(day, ws)
	rows, err := s.Q.SponsorCampaignsOfList(ctx, store.SponsorCampaignsOfListParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to, ListID: list.String()})
	if err != nil || len(rows) == 0 {
		return nil
	}
	id := rows[0].ID
	return &id
}

// mustBooking reads a booking of the workspace for a change.
func (s *Service) mustBooking(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.SponsorBookingGetRow, store.SponsorSlotGetRow, error) {
	b, err := s.Q.SponsorBookingGet(ctx, store.SponsorBookingGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return b, store.SponsorSlotGetRow{}, api.MapError(err, "Booking")
	}
	slot, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: b.SlotID, WorkspaceID: ws.ID})
	if err != nil {
		return b, slot, api.MapError(err, "Slot")
	}
	return b, slot, nil
}

// UpdateBooking changes a booking's sponsor, price, status, hold date or notes.
func (s *Service) UpdateBooking(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in BookingInput) (SponsorBookingDetail, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	if b.Status == "canceled" || b.Status == "proposed" {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.bookingClosed", nil)
	}
	sponsorID := b.SponsorID
	sponsorName := b.SponsorName
	terms := b.SponsorTerms
	if in.SponsorID != "" {
		sid, err := uuid.Parse(in.SponsorID)
		if err != nil {
			return SponsorBookingDetail{}, invalid(ctx, "body.sponsorId", "problem.pickSponsor", nil)
		}
		sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: sid, WorkspaceID: ws.ID})
		if err != nil {
			return SponsorBookingDetail{}, invalid(ctx, "body.sponsorId", "problem.pickSponsor", nil)
		}
		sponsorID, sponsorName, terms = &sid, sp.Name, sp.TermsDays
	}
	price := b.PriceCents
	if in.PriceCents != nil {
		price = *in.PriceCents
	}
	status := b.Status
	if in.Status != "" && b.Status != "sent" {
		status = in.Status
	}
	hold := b.HoldUntil
	if in.HoldUntil != "" {
		if hold, err = parseDay(ctx, "body.holdUntil", in.HoldUntil); err != nil {
			return SponsorBookingDetail{}, err
		}
	}
	if status == "hold" {
		if hold == nil {
			return SponsorBookingDetail{}, invalid(ctx, "body.holdUntil", "problem.holdDate", nil)
		}
		if !hold.Before(b.IssueOn) {
			return SponsorBookingDetail{}, invalid(ctx, "body.holdUntil", "problem.holdBeforeIssue", map[string]any{"issue": b.IssueNumber})
		}
	} else {
		hold = nil
	}
	paymentDue := b.PaymentDue
	if sponsorID != nil && (b.SponsorID == nil || *sponsorID != *b.SponsorID) {
		d := civil(b.IssueOn).AddDate(0, 0, int(terms))
		paymentDue = &d
	}
	notes := b.Notes
	if in.Notes != nil {
		notes = strings.TrimSpace(*in.Notes)
	}
	before := map[string]any{"sponsor": b.SponsorName, "status": b.Status, "priceCents": b.PriceCents, "holdUntil": dayString(b.HoldUntil)}
	after := map[string]any{"sponsor": sponsorName, "status": status, "priceCents": price, "holdUntil": dayString(hold)}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingSetTerms(ctx, store.SponsorBookingSetTermsParams{ID: id, WorkspaceID: ws.ID, SponsorID: sponsorID, PriceCents: price, Status: status,
			HoldUntil: hold, CreativeDue: b.CreativeDue, PaymentDue: paymentDue, At: s.now()}); err != nil {
			return err
		}
		if notes != b.Notes {
			if err := q.SponsorBookingSetNotes(ctx, store.SponsorBookingSetNotesParams{ID: id, WorkspaceID: ws.ID, Notes: notes, At: s.now()}); err != nil {
				return err
			}
			before["notes"], after["notes"] = b.Notes, notes
		}
		switch {
		case b.Status == "hold" && status == "booked":
			if err := s.event(ctx, q, id, p, "booked", map[string]any{"sponsor": sponsorName, "priceCents": price, "fromHold": true}); err != nil {
				return err
			}
		case status == "hold" && dayString(hold) != dayString(b.HoldUntil):
			if err := s.event(ctx, q, id, p, "hold_changed", map[string]any{"holdUntil": dayString(hold)}); err != nil {
				return err
			}
		case price != b.PriceCents || sponsorName != b.SponsorName:
			if err := s.event(ctx, q, id, p, "changed", map[string]any{"sponsor": sponsorName, "priceCents": price}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.booking_updated", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name), Before: before, After: after})
	})
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	return s.Booking(ctx, ws, p, id)
}

// SetCampaign links an issue to its campaign by hand, or back to the
// campaign of its list and date.
func (s *Service) SetCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CampaignInput) (SponsorBookingDetail, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	var cid *uuid.UUID
	byHand := false
	name := ""
	if in.CampaignID != "" {
		c, err := uuid.Parse(in.CampaignID)
		if err != nil {
			return SponsorBookingDetail{}, invalid(ctx, "body.campaignId", "problem.campaign", nil)
		}
		cp, err := s.Q.SponsorCampaignBrief(ctx, store.SponsorCampaignBriefParams{ID: c, WorkspaceID: ws.ID})
		if err != nil {
			return SponsorBookingDetail{}, invalid(ctx, "body.campaignId", "problem.campaign", nil)
		}
		cid, byHand, name = &c, true, cp.Name
	} else {
		cid = s.campaignOn(ctx, ws, slot.ListID, b.IssueOn)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingSetCampaign(ctx, store.SponsorBookingSetCampaignParams{ID: id, WorkspaceID: ws.ID, CampaignID: cid, CampaignByHand: byHand, At: s.now()}); err != nil {
			return err
		}
		if err := s.event(ctx, q, id, p, "campaign_linked", map[string]any{"campaign": name, "byHand": byHand}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.booking_updated", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name), Before: map[string]any{"campaign": b.CampaignName},
			After: map[string]any{"campaign": name, "byHand": byHand}})
	})
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	return s.Booking(ctx, ws, p, id)
}

// creativeMissing names what a creative still needs before it's ready.
func creativeMissing(c SponsorCreative) []string {
	var out []string
	if strings.TrimSpace(c.Headline) == "" {
		out = append(out, "headline")
	}
	if strings.TrimSpace(c.Text) == "" {
		out = append(out, "text")
	}
	if strings.TrimSpace(c.Link) == "" {
		out = append(out, "link")
	}
	return out
}

func cleanCreative(ctx context.Context, c SponsorCreative) (SponsorCreative, error) {
	c.Headline = strings.TrimSpace(c.Headline)
	c.Text = strings.TrimSpace(c.Text)
	c.Button = strings.TrimSpace(c.Button)
	c.Link = strings.TrimSpace(c.Link)
	if len([]rune(c.Text)) > 280 {
		return c, invalid(ctx, "body.text", "problem.textLength", nil)
	}
	if c.Link != "" {
		l := c.Link
		if !strings.Contains(l, "://") {
			l = "https://" + l
		}
		u, err := url.Parse(l)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			return c, invalid(ctx, "body.link", "problem.link", nil)
		}
		c.Link = l
	}
	for _, v := range []string{c.UTM.Source, c.UTM.Medium, c.UTM.Campaign} {
		if strings.ContainsAny(v, "&?#= ") {
			return c, invalid(ctx, "body.utm", "problem.utm", nil)
		}
	}
	return c, nil
}

// SetCreative saves the creative's fields (the panel saves as you go). A
// ready creative stays ready.
func (s *Service) SetCreative(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SponsorCreative) (SponsorBookingDetail, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	if b.Status == "canceled" || b.Status == "sent" {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.creativeLocked", nil)
	}
	in, err = cleanCreative(ctx, in)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	old := parseCreative(b.Creative)
	raw, _ := json.Marshal(in)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingSetCreative(ctx, store.SponsorBookingSetCreativeParams{ID: id, WorkspaceID: ws.ID, Creative: raw, CreativeStatus: b.CreativeStatus,
			ReadyAt: b.ReadyAt, ReadyBy: b.ReadyBy, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.creative_updated", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name), Before: creativeAudit(old), After: creativeAudit(in)})
	})
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	return s.Booking(ctx, ws, p, id)
}

func creativeAudit(c SponsorCreative) map[string]any {
	return map[string]any{"headline": c.Headline, "text": c.Text, "button": c.Button, "link": c.Link, "utm": c.UTM, "logo": c.Logo != "", "image": c.Image != ""}
}

// MarkReady marks the creative ready: the Sponsor block shows it and the
// issue can go out with it.
func (s *Service) MarkReady(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SponsorBookingDetail, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	if b.Status != "booked" {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.readyNeedsBooking", nil)
	}
	c := parseCreative(b.Creative)
	if miss := creativeMissing(c); len(miss) > 0 {
		return SponsorBookingDetail{}, problem(ctx, 422, api.CodeValidation, "problem.creativeIncomplete", nil).WithMeta("missing", miss)
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingSetCreative(ctx, store.SponsorBookingSetCreativeParams{ID: id, WorkspaceID: ws.ID, Creative: b.Creative, CreativeStatus: "ready",
			ReadyAt: &now, ReadyBy: userID(p), At: now}); err != nil {
			return err
		}
		if err := s.event(ctx, q, id, p, "creative_ready", nil); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.creative_ready", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name), Before: map[string]any{"creative": b.CreativeStatus}, After: map[string]any{"creative": "ready"}})
	})
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	return s.Booking(ctx, ws, p, id)
}

// RecordPayment records a booking's payment by hand, or changes it.
func (s *Service) RecordPayment(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PaymentInput) (SponsorBookingDetail, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	if b.Status == "canceled" || b.Status == "proposed" || b.SponsorID == nil {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.bookingClosed", nil)
	}
	status := "unpaid"
	var paidOn *time.Time
	var amount *int64
	note := strings.TrimSpace(in.Note)
	now := s.now()
	var recorded *time.Time
	if in.Paid {
		status = "paid"
		if paidOn, err = parseDay(ctx, "body.paidOn", in.PaidOn); err != nil {
			return SponsorBookingDetail{}, err
		}
		if paidOn == nil {
			d := s.today(ws)
			paidOn = &d
		}
		if paidOn.After(s.today(ws)) {
			return SponsorBookingDetail{}, invalid(ctx, "body.paidOn", "problem.paidFuture", nil)
		}
		a := b.PriceCents
		if in.AmountCents != nil {
			a = *in.AmountCents
		}
		amount = &a
		recorded = &now
	}
	action := "sponsors.payment_recorded"
	if b.PaymentStatus == "paid" {
		action = "sponsors.payment_updated"
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		by := userID(p)
		if !in.Paid {
			by = nil
		}
		if err := q.SponsorBookingSetPayment(ctx, store.SponsorBookingSetPaymentParams{ID: id, WorkspaceID: ws.ID, PaymentStatus: status, PaidOn: paidOn, PaidCents: amount,
			PaidNote: note, PaidBy: by, PaidRecordedAt: recorded, At: now}); err != nil {
			return err
		}
		kind := "paid"
		if !in.Paid {
			kind = "payment_cleared"
		}
		if err := s.event(ctx, q, id, p, kind, map[string]any{"amountCents": amount, "paidOn": dayString(paidOn), "note": note}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name),
			Before:     map[string]any{"payment": b.PaymentStatus, "paidOn": dayString(b.PaidOn), "paidCents": b.PaidCents},
			After:      map[string]any{"payment": status, "paidOn": dayString(paidOn), "paidCents": amount, "note": note}})
	})
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	return s.Booking(ctx, ws, p, id)
}

// CancelBooking cancels a booking or releases a hold: the issue opens again
// and its Sponsor block is hidden.
func (s *Service) CancelBooking(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in CancelInput) error {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return err
	}
	if b.Status == "canceled" {
		return nil
	}
	if b.Status == "sent" {
		return problem(ctx, 409, api.CodeConflict, "problem.cancelSent", map[string]any{"issue": b.IssueNumber})
	}
	reason := strings.TrimSpace(in.Reason)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingCancel(ctx, store.SponsorBookingCancelParams{ID: id, WorkspaceID: ws.ID, CancelReason: reason, At: s.now(), CanceledBy: userID(p)}); err != nil {
			return err
		}
		kind := "canceled"
		action := "sponsors.booking_canceled"
		switch b.Status {
		case "hold":
			kind = "released"
		case "proposed":
			kind, action = "dismissed", "sponsors.booking_proposal_dismissed"
		}
		if err := s.event(ctx, q, id, p, kind, map[string]any{"reason": reason}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name), Reason: reason,
			Before: map[string]any{"status": b.Status, "sponsor": b.SponsorName, "priceCents": b.PriceCents}})
	})
}

// Propose records an MCP agent's proposed booking for a person to confirm.
func (s *Service) Propose(ctx context.Context, ws *store.Workspace, p *authz.Principal, in ProposeInput) (SponsorIssueDetail, error) {
	slot, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: in.SlotID, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorIssueDetail{}, api.MapError(err, "Slot")
	}
	sp, err := s.Q.SponsorGet(ctx, store.SponsorGetParams{ID: in.SponsorID, WorkspaceID: ws.ID})
	if err != nil {
		return SponsorIssueDetail{}, api.MapError(err, "Sponsor")
	}
	slotRow := store.SponsorSlot{ID: slot.ID, FirstNumber: slot.FirstNumber, FirstOn: slot.FirstOn, EveryDays: slot.EveryDays}
	on := IssueOn(slotRow, in.Issue)
	if on.Before(s.today(ws)) {
		return SponsorIssueDetail{}, problem(ctx, 409, api.CodeConflict, "problem.issuePast", map[string]any{"issue": in.Issue})
	}
	if _, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: slot.ID, IssueNumber: int32(in.Issue)}); err == nil {
		return SponsorIssueDetail{}, problem(ctx, 409, api.CodeConflict, "problem.issueTaken", map[string]any{"issue": in.Issue})
	}
	price := slot.PriceCents
	if in.PriceCents != nil {
		price = *in.PriceCents
	}
	creativeDue := on.AddDate(0, 0, -int(slot.CreativeDays))
	paymentDue := on.AddDate(0, 0, int(sp.TermsDays))
	cr, _ := json.Marshal(SponsorCreative{UTM: defaultUTM(ws, slot.ListName, in.Issue), Logo: sp.LogoUrl})
	id := newID()
	agent := p.ID
	note := strings.TrimSpace(in.Note)
	if len([]rune(note)) > 500 {
		note = string([]rune(note)[:500])
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingInsert(ctx, store.SponsorBookingInsertParams{ID: id, WorkspaceID: ws.ID, SlotID: slot.ID, IssueNumber: int32(in.Issue), IssueOn: on,
			SponsorID: &sp.ID, CampaignID: s.campaignOn(ctx, ws, slot.ListID, on), PriceCents: price, Currency: slot.Currency, Status: "proposed", Creative: cr,
			CreativeStatus: "waiting", CreativeDue: &creativeDue, PaymentDue: &paymentDue, ProposedBy: &agent, ProposedName: actorName(p), ProposedNote: note, At: s.now()}); err != nil {
			return err
		}
		if err := s.event(ctx, q, id, p, "proposed", map[string]any{"sponsor": sp.Name, "note": note}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.booking_proposed", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, in.Issue, slot.Name), After: map[string]any{"sponsor": sp.Name, "priceCents": price, "note": note}})
	})
	if err != nil {
		return SponsorIssueDetail{}, err
	}
	return s.Issue(ctx, ws, p, slot.ID, in.Issue)
}

// ConfirmProposal books an agent's proposal, if the issue is still open.
func (s *Service) ConfirmProposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (SponsorBookingDetail, error) {
	b, slot, err := s.mustBooking(ctx, ws, id)
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	if b.Status != "proposed" {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.notProposed", nil)
	}
	if _, err := s.Q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: b.SlotID, IssueNumber: b.IssueNumber}); err == nil {
		return SponsorBookingDetail{}, problem(ctx, 409, api.CodeConflict, "problem.issueTaken", map[string]any{"issue": b.IssueNumber})
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorBookingSetTerms(ctx, store.SponsorBookingSetTermsParams{ID: id, WorkspaceID: ws.ID, SponsorID: b.SponsorID, PriceCents: b.PriceCents, Status: "booked",
			CreativeDue: b.CreativeDue, PaymentDue: b.PaymentDue, At: s.now()}); err != nil {
			if isUnique(err) {
				return problem(ctx, 409, api.CodeConflict, "problem.issueTaken", map[string]any{"issue": b.IssueNumber})
			}
			return err
		}
		if err := s.event(ctx, q, id, p, "booked", map[string]any{"sponsor": b.SponsorName, "priceCents": b.PriceCents, "fromProposal": b.ProposedName}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsors.booking_created", TargetType: "sponsor_booking", TargetID: id.String(),
			TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name),
			After:      map[string]any{"sponsor": b.SponsorName, "status": "booked", "priceCents": b.PriceCents, "proposedBy": b.ProposedName}})
	})
	if err != nil {
		return SponsorBookingDetail{}, err
	}
	return s.Booking(ctx, ws, p, id)
}
