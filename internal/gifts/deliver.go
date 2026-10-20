package gifts

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// The gift emails: "Gift receipt" to the buyer once paid, "Your gift" to
// the recipient on the delivery date (to the buyer for an in-person gift),
// again after a week if it isn't redeemed, and "Gift ending" a week before
// the access ends. They're the member emails' templates (Transactional,
// Memberships), sent only once published.

// RedeemURL is a gift's redeem link (news.acme.io/gift/redeem/{code}).
func (s *Service) RedeemURL(ctx context.Context, ws *store.Workspace, code string) string {
	return s.Earn.PageURL(ctx, ws, "gift/redeem/"+code)
}

// RedeemHost is the redeem link without its scheme.
func (s *Service) RedeemHost(ctx context.Context, ws *store.Workspace, code string) string {
	return s.Earn.PageHost(ctx, ws, "gift/redeem/"+code)
}

// deliveryText is "24 Dec at 08:00" in the buyer's time zone.
func deliveryText(at time.Time, tz string, ws *store.Workspace, now time.Time) (string, string) {
	loc := zoneOr(tz, ws)
	l := at.In(loc)
	date := l.Format("2 Jan")
	if l.Year() != now.In(loc).Year() {
		date = l.Format("2 Jan 2006")
	}
	return date, l.Format("15:04")
}

// tierOf is a gift's tier's full and short names, and its lists.
func (s *Service) tierOf(ctx context.Context, ws *store.Workspace, g store.EarnGift) (string, string, []uuid.UUID) {
	if g.TierID == nil {
		return "", "", nil
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: *g.TierID, WorkspaceID: ws.ID})
	if err != nil {
		return "", "", nil
	}
	return t.Name, s.tierShort(ctx, t.Name, t.ListIds), t.ListIds
}

// sendReceipt sends the buyer their receipt.
func (s *Service) sendReceipt(ctx context.Context, ws *store.Workspace, g store.EarnGift) earn.MemberEmailResult {
	return s.Earn.SendMemberEmailTo(ctx, ws, earn.EmailGiftPurchased, g.BuyerEmail, g.BuyerName, s.receiptData(ctx, ws, g), "gift:"+g.ID.String()+":receipt")
}

// sendGift sends "Your gift" with the redeem link: to the recipient, or
// to the buyer for an in-person gift. key tells the sends apart (the
// delivery, the reminder, a resend).
func (s *Service) sendGift(ctx context.Context, ws *store.Workspace, g store.EarnGift, key string) earn.MemberEmailResult {
	_, short, _ := s.tierOf(ctx, ws, g)
	data := map[string]any{"tier": short, "giver": firstNonEmpty(g.BuyerName, g.BuyerEmail), "gift": period(int(g.Months)), "message": g.Message,
		"redeem_url": s.RedeemURL(ctx, ws, g.Code)}
	to, name := g.RecipientEmail, g.RecipientName
	if g.InPerson || to == "" {
		to, name = g.BuyerEmail, g.BuyerName
	} else if f := firstName(g.RecipientName); f != "" {
		data["first_name"] = f
	}
	return s.Earn.SendMemberEmailTo(ctx, ws, earn.EmailGiftReceived, to, name, data, "gift:"+g.ID.String()+":"+key)
}

// emailData is what a gift event says about an email.
func emailData(email string, r earn.MemberEmailResult) map[string]any {
	d := map[string]any{"email": email, "status": r.Status, "template": r.Template}
	if r.Reason != "" {
		d["reason"] = r.Reason
	}
	if r.Version > 0 {
		d["version"] = r.Version
	}
	return d
}

// emailStatus is how a "Your gift" email went, as the gift keeps it.
func emailStatus(r earn.MemberEmailResult) string {
	if r.Sent() {
		return "sent"
	}
	return r.Reason
}

// TickArgs is the gifts job: receipts, deliveries on their date,
// reminders, expired links, gift ending emails and message purges.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "gifts.tick" }

// InsertOpts runs on the maintenance queue, once at a time.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// TickWorker runs TickArgs.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Timeout bounds one run.
func (w *TickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 5 * time.Minute }

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.S.Tick(ctx) }

// RegisterJobs adds the gifts job, every minute: a gift goes out at the
// minute its buyer picked.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, true)
}

// Tick does what's due.
func (s *Service) Tick(ctx context.Context) error {
	now := s.now()
	wss := map[uuid.UUID]*store.Workspace{}
	wsOf := func(id uuid.UUID) *store.Workspace {
		if ws, ok := wss[id]; ok {
			return ws
		}
		ws, err := s.Q.GetWorkspace(ctx, id)
		if err != nil {
			wss[id] = nil
			return nil
		}
		wss[id] = &ws
		return &ws
	}
	if _, err := s.Q.GiftsAbandon(ctx, store.GiftsAbandonParams{Now: now, DayAgo: now.Add(-AbandonAfter)}); err != nil {
		return err
	}
	// Receipts, as soon as the payment is in (not for payments of more than
	// a week ago: a gift paid while mail was down doesn't get one late).
	paid, err := s.Q.GiftsToReceipt(ctx, now.Add(-7*24*time.Hour))
	if err != nil {
		return err
	}
	for _, g := range paid {
		ws := wsOf(g.WorkspaceID)
		if ws == nil {
			continue
		}
		r := s.sendReceipt(ctx, ws, g)
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.GiftReceiptSent(ctx, store.GiftReceiptSentParams{At: now, ID: g.ID}); err != nil {
				return err
			}
			return event(ctx, q, g.ID, "receipt", now, nil, r.MessageID, emailData(earn.EmailGiftPurchased, r))
		}); err != nil {
			return err
		}
	}
	due, err := s.Q.GiftsDue(ctx, now)
	if err != nil {
		return err
	}
	for _, g := range due {
		ws := wsOf(g.WorkspaceID)
		if ws == nil {
			continue
		}
		if err := s.deliver(ctx, ws, g); err != nil {
			return err
		}
	}
	remind, err := s.Q.GiftsToRemind(ctx, store.GiftsToRemindParams{WeekAgo: now.Add(-RemindAfter), Now: now})
	if err != nil {
		return err
	}
	for _, g := range remind {
		ws := wsOf(g.WorkspaceID)
		if ws == nil {
			continue
		}
		r := s.sendGift(ctx, ws, g, "reminder")
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.GiftEmailed(ctx, store.GiftEmailedParams{EmailStatus: emailStatus(r), RemindedAt: &now, At: now, ID: g.ID}); err != nil {
				return err
			}
			return event(ctx, q, g.ID, "reminded", now, nil, r.MessageID, emailData(earn.EmailGiftReceived, r))
		}); err != nil {
			return err
		}
	}
	expired, err := s.Q.GiftsToExpire(ctx, now)
	if err != nil {
		return err
	}
	for _, g := range expired {
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.GiftExpired(ctx, store.GiftExpiredParams{At: now, ID: g.ID}); err != nil {
				return err
			}
			return event(ctx, q, g.ID, "expired", now, nil, nil, nil)
		}); err != nil {
			return err
		}
	}
	if err := s.endings(ctx, now, wsOf); err != nil {
		return err
	}
	_, err = s.Q.GiftsPurgeMessages(ctx, store.GiftsPurgeMessagesParams{Now: now, MonthAgo: now.Add(-MessageKeep)})
	return err
}

// deliver sends a gift on its date and starts its link's window.
func (s *Service) deliver(ctx context.Context, ws *store.Workspace, g store.EarnGift) error {
	now := s.now()
	months := 12
	if st, err := s.Q.EarnSettings(ctx, ws.ID); err == nil && st.GiftLinkMonths > 0 {
		months = int(st.GiftLinkMonths)
	}
	r := s.sendGift(ctx, ws, g, "delivery")
	expires := now.AddDate(0, months, 0)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.GiftDelivered(ctx, store.GiftDeliveredParams{At: now, LinkExpiresAt: expires, EmailStatus: emailStatus(r), ID: g.ID}); err != nil {
			return err
		}
		d := emailData(earn.EmailGiftReceived, r)
		d["to"] = map[bool]string{true: "buyer", false: "recipient"}[g.InPerson || g.RecipientEmail == ""]
		return event(ctx, q, g.ID, "delivered", now, nil, r.MessageID, d)
	})
}

// endings sends "Gift ending" a week before redeemed gifts end. While
// gift-ending isn't published nothing goes, and it's tried again until
// the gift ends.
func (s *Service) endings(ctx context.Context, now time.Time, wsOf func(uuid.UUID) *store.Workspace) error {
	rows, err := s.Q.GiftsEnding(ctx, store.GiftsEndingParams{Now: now, WeekAhead: now.Add(EndingBefore)})
	if err != nil {
		return err
	}
	for _, row := range rows {
		g := row.EarnGift
		ws := wsOf(g.WorkspaceID)
		if ws == nil || g.RedeemedContact == nil || row.AccessEnds == nil {
			continue
		}
		c, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: *g.RedeemedContact, WorkspaceID: ws.ID})
		if err != nil {
			continue
		}
		_, short, _ := s.tierOf(ctx, ws, g)
		data := map[string]any{"tier": short, "ends_at": s.Earn.DateText(ws, *row.AccessEnds), "upgrade_url": s.Earn.UpgradeURL(ctx, ws, c.ID, uuid.Nil)}
		if c.FirstName != "" {
			data["first_name"] = c.FirstName
		}
		r := s.Earn.SendMemberEmailTo(ctx, ws, earn.EmailGiftEnding, c.Email, c.FirstName+" "+c.LastName, data, "gift:"+g.ID.String()+":ending")
		switch r.Reason {
		case "not_published", "no_route", "no_template", "no_mailer":
			// Not now: tried again while the gift lasts.
			continue
		}
		if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.GiftEndingSent(ctx, store.GiftEndingSentParams{At: now, ID: g.ID}); err != nil {
				return err
			}
			if err := event(ctx, q, g.ID, "ending", now, nil, r.MessageID, emailData(earn.EmailGiftEnding, r)); err != nil {
				return err
			}
			kind := "email_sent"
			if !r.Sent() {
				kind = "email_not_sent"
			}
			return s.Earn.Timeline(ctx, q, ws.ID, c.ID, g.SubscriptionID, g.Mode, kind, now, nil, emailData(earn.EmailGiftEnding, r))
		}); err != nil {
			return err
		}
	}
	return nil
}
