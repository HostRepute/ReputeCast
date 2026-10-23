package gifts

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The team's changes to a gift (earn:manage), each in the audit log:
// moving a scheduled gift's delivery, sending "Your gift" again, and
// copying the redeem link (anyone with it can redeem the gift). Refunds
// are made in the provider.

// GiftDeliveryInput moves a scheduled gift.
type GiftDeliveryInput struct {
	Date     string `json:"date" format:"date" doc:"The day, in the time zone"`
	Time     string `json:"time" pattern:"^[0-2][0-9]:[0-5][0-9]$" doc:"The time of day, 24-hour"`
	Timezone string `json:"timezone,omitempty" doc:"An IANA time zone; the buyer's is the default"`
	// NotifyBuyer sends the buyer their receipt again with the new date.
	NotifyBuyer bool `json:"notifyBuyer,omitempty"`
}

// gift reads a gift of the workspace.
func (s *Service) giftOf(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.GiftGetRow, error) {
	row, err := s.Q.GiftGet(ctx, store.GiftGetParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return row, notFound(ctx, "gift")
	}
	return row, err
}

// Row reads one gift as the table shows it.
func (s *Service) Row(ctx context.Context, ws *store.Workspace, id uuid.UUID) (GiftRow, error) {
	row, err := s.giftOf(ctx, ws, id)
	if err != nil {
		return GiftRow{}, err
	}
	return s.row(ctx, gift(row.EarnGift, row.TierName, row.ListIds, row.AccessEnds), s.dashboard(ctx, ws, row.EarnGift.Mode)), nil
}

// ChangeDelivery moves a scheduled gift's delivery.
func (s *Service) ChangeDelivery(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in GiftDeliveryInput) (GiftRow, error) {
	row, err := s.giftOf(ctx, ws, id)
	if err != nil {
		return GiftRow{}, err
	}
	g := row.EarnGift
	if g.Status != StatusPaid {
		return GiftRow{}, api.Err(http.StatusConflict, api.CodeConflict, text(ctx, "problem.notScheduled", nil))
	}
	tz := firstNonEmpty(in.Timezone, g.Timezone)
	loc, err := time.LoadLocation(tz)
	if tz == "" || err != nil {
		return GiftRow{}, invalid("body.timezone", text(ctx, "problem.timezone", nil))
	}
	at, err := time.ParseInLocation("2006-01-02 15:04", in.Date+" "+in.Time, loc)
	if err != nil {
		return GiftRow{}, invalid("body.date", text(ctx, "problem.date", nil))
	}
	now := s.now()
	if !at.After(now.Add(5 * time.Minute)) {
		return GiftRow{}, invalid("body.date", text(ctx, "problem.datePast", nil))
	}
	if at.After(now.Add(MaxDeliverOut)) {
		return GiftRow{}, invalid("body.date", text(ctx, "problem.dateFar", nil))
	}
	before := map[string]any{"deliverAt": g.DeliverAt, "timezone": g.Timezone}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.GiftSetDelivery(ctx, store.GiftSetDeliveryParams{DeliverAt: at, Timezone: tz, At: now, ID: id}); err != nil {
			return err
		}
		if err := event(ctx, q, id, "delivery_changed", now, userID(p), nil, map[string]any{"from": g.DeliverAt, "to": at, "timezone": tz}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.gift_delivery_changed", TargetType: "earn_gift", TargetID: id.String(),
			TargetName: s.giftName(ctx, row), Before: before, After: map[string]any{"deliverAt": at, "timezone": tz},
			Metadata: map[string]any{"notifyBuyer": in.NotifyBuyer}})
	})
	if err != nil {
		return GiftRow{}, err
	}
	if in.NotifyBuyer {
		g.DeliverAt, g.Timezone = &at, tz
		texts := func(l i18n.Lang) map[string]any { return s.receiptData(ctx, ws, g, l) }
		r := s.Earn.SendMemberEmailIn(ctx, ws, earn.EmailGiftPurchased, g.BuyerEmail, g.BuyerName, texts(earn.PageLang),
			"gift:"+id.String()+":moved:"+at.UTC().Format(time.RFC3339), texts)
		_ = event(ctx, s.Q, id, "receipt", now, userID(p), r.MessageID, emailData(earn.EmailGiftPurchased, r))
	}
	return s.Row(ctx, ws, id)
}

// receiptData is the receipt's variables (sendReceipt's, for a resend).
// receiptData is the receipt's data, its lines in a language: the
// email's.
func (s *Service) receiptData(ctx context.Context, ws *store.Workspace, g store.EarnGift, l i18n.Lang) map[string]any {
	now := s.now()
	_, short, _ := s.tierOf(ctx, ws, g)
	data := map[string]any{"tier": short, "gift": period(l, int(g.Months)), "amount": earn.MoneyCents(g.AmountCents, g.Currency)}
	if f := firstName(g.BuyerName); f != "" {
		data["first_name"] = f
	}
	data["recipient"] = firstNonEmpty(firstName(g.RecipientName), g.RecipientEmail)
	switch {
	case g.InPerson:
		data["recipient"] = msgs.M("receipt.someone").Text(l, nil)
		data["delivery_line"] = msgs.M("receipt.inPerson").Text(l, nil)
	case g.DeliverAt != nil && g.DeliverAt.After(now):
		date, clock := deliveryText(l, *g.DeliverAt, g.Timezone, ws, now)
		data["delivery_line"] = msgs.M("receipt.scheduled").Text(l, i18n.Args{"date": date, "time": clock})
	default:
		data["delivery_line"] = msgs.M("receipt.now").Text(l, nil)
	}
	return data
}

// Resend sends "Your gift" again to a delivered gift's recipient (or the
// buyer, in person).
func (s *Service) Resend(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (GiftRow, error) {
	row, err := s.giftOf(ctx, ws, id)
	if err != nil {
		return GiftRow{}, err
	}
	g := row.EarnGift
	if g.Status != StatusDelivered {
		return GiftRow{}, api.Err(http.StatusConflict, api.CodeConflict, text(ctx, "problem.notDelivered", nil))
	}
	now := s.now()
	r := s.sendGift(ctx, ws, g, "resend:"+now.UTC().Format(time.RFC3339))
	if !r.Sent() {
		return GiftRow{}, api.Err(http.StatusConflict, api.CodeConflict, text(ctx, "problem.notSent."+r.Reason, nil))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.GiftEmailed(ctx, store.GiftEmailedParams{EmailStatus: emailStatus(r), At: now, ID: id}); err != nil {
			return err
		}
		if err := event(ctx, q, id, "resent", now, userID(p), r.MessageID, emailData(earn.EmailGiftReceived, r)); err != nil {
			return err
		}
		to := g.RecipientEmail
		if g.InPerson || to == "" {
			to = g.BuyerEmail
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.gift_email_resent", TargetType: "earn_gift", TargetID: id.String(),
			TargetName: s.giftName(ctx, row), Metadata: map[string]any{"to": to, "template": r.Template}})
	})
	if err != nil {
		return GiftRow{}, err
	}
	return s.Row(ctx, ws, id)
}

// GiftLink is a gift's redeem link, copied by the team.
type GiftLink struct {
	URL  string `json:"url"`
	Host string `json:"host"`
}

// CopyLink returns a gift's redeem link, recorded in the audit log: anyone
// with it can redeem the gift until it's used.
func (s *Service) CopyLink(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (GiftLink, error) {
	row, err := s.giftOf(ctx, ws, id)
	if err != nil {
		return GiftLink{}, err
	}
	g := row.EarnGift
	if g.Status != StatusPaid && g.Status != StatusDelivered {
		return GiftLink{}, api.Err(http.StatusConflict, api.CodeConflict, text(ctx, "problem.linkGone", nil))
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := event(ctx, q, id, "link_copied", now, userID(p), nil, nil); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.gift_link_copied", TargetType: "earn_gift", TargetID: id.String(),
			TargetName: s.giftName(ctx, row)})
	})
	if err != nil {
		return GiftLink{}, err
	}
	return GiftLink{URL: s.RedeemURL(ctx, ws, g.Code), Host: s.RedeemHost(ctx, ws, g.Code)}, nil
}

// giftName names a gift in the audit log: "Gift to Karim Haddad ·
// Weekly Digest Insider · 1 year".
func (s *Service) giftName(_ context.Context, row store.GiftGetRow) string {
	g := row.EarnGift
	to := firstNonEmpty(g.RecipientName, g.RecipientEmail)
	if g.InPerson || to == "" {
		return english("giftNamePerson", i18n.Args{"buyer": firstNonEmpty(g.BuyerName, g.BuyerEmail), "tier": row.TierName, "months": g.Months})
	}
	return english("giftName", i18n.Args{"to": to, "tier": row.TierName, "months": g.Months})
}
