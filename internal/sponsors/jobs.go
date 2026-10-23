package sponsors

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// TickArgs is the periodic sponsorship job: bookings whose issue went out
// become sent, holds past their date release the issue, reports still
// coming in are counted again, and the people with sponsors:manage hear
// about a creative due tomorrow or late and a payment overdue.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "sponsors.tick" }

// InsertOpts runs on the maintenance queue, once at a time.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
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

// RegisterJobs adds the sponsorship job, every 5 minutes.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	reg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, true)
}

// Tick runs the periodic work.
func (s *Service) Tick(ctx context.Context) error {
	if err := s.markSent(ctx); err != nil {
		return err
	}
	if err := s.lapseHolds(ctx); err != nil {
		return err
	}
	if err := s.refreshReports(ctx); err != nil {
		return err
	}
	return s.remind(ctx)
}

// markSent turns a booked issue whose campaign went out into a sent one.
func (s *Service) markSent(ctx context.Context) error {
	rows, err := s.Q.SponsorBookingsToSend(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.StartedAt == nil {
			continue
		}
		if err := s.Q.SponsorBookingSent(ctx, store.SponsorBookingSentParams{ID: r.ID, SentAt: *r.StartedAt, At: s.now()}); err != nil {
			return err
		}
	}
	return nil
}

// lapseHolds releases holds past their date, in each workspace's time zone.
func (s *Service) lapseHolds(ctx context.Context) error {
	// Holds lapse the day after their date: read every hold up to
	// tomorrow in UTC and check each against its workspace's calendar.
	rows, err := s.Q.SponsorHoldsLapsed(ctx, civil(s.now().UTC()).AddDate(0, 0, 1))
	if err != nil {
		return err
	}
	wss := map[uuid.UUID]*store.Workspace{}
	for _, b := range rows {
		ws, ok := wss[b.WorkspaceID]
		if !ok {
			w, err := s.Q.GetWorkspace(ctx, b.WorkspaceID)
			if err != nil {
				continue
			}
			ws, wss[b.WorkspaceID] = &w, &w
		}
		if b.HoldUntil == nil || !civil(*b.HoldUntil).Before(s.today(ws)) {
			continue
		}
		slot, err := s.Q.SponsorSlotGet(ctx, store.SponsorSlotGetParams{ID: b.SlotID, WorkspaceID: b.WorkspaceID})
		if err != nil {
			continue
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.SponsorBookingCancel(ctx, store.SponsorBookingCancelParams{ID: b.ID, WorkspaceID: b.WorkspaceID, CancelReason: "hold_lapsed", At: s.now()}); err != nil {
				return err
			}
			if err := s.event(ctx, q, b.ID, nil, "hold_lapsed", map[string]any{"holdUntil": dayString(b.HoldUntil)}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &b.WorkspaceID, Action: "sponsors.hold_lapsed", TargetType: "sponsor_booking", TargetID: b.ID.String(),
				TargetName: issueName(slot.ListName, int(b.IssueNumber), slot.Name), Before: map[string]any{"status": "hold", "holdUntil": dayString(b.HoldUntil)},
				Actor: &audit.Actor{Type: authz.ActorSystem, Name: "ReputeCast"}})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// refreshReports counts the reports still coming in again; each freezes
// once final.
func (s *Service) refreshReports(ctx context.Context) error {
	rows, err := s.Q.SponsorReportsToRefresh(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		b, err := s.Q.SponsorBookingGet(ctx, store.SponsorBookingGetParams(r))
		if err != nil {
			continue
		}
		if b.CampaignStatus != "sent" {
			continue
		}
		p, err := s.computeStats(ctx, b)
		if err != nil {
			s.log().WarnContext(ctx, "sponsor report", "booking", r.ID, "err", err)
			continue
		}
		if err := s.Q.SponsorReportStatsUpsert(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// remind tells the people with sponsors:manage about a creative due
// tomorrow or late, and a payment overdue, once each.
func (s *Service) remind(ctx context.Context) error {
	rows, err := s.Q.SponsorBookingsDue(ctx)
	if err != nil {
		return err
	}
	for _, b := range rows {
		ws := &store.Workspace{ID: b.WorkspaceID, Slug: b.WorkspaceSlug, Timezone: b.WorkspaceTimezone}
		today := s.today(ws)
		issue := map[string]any{"issue": b.IssueNumber, "sponsor": b.SponsorName}
		link := "/w/" + b.WorkspaceSlug + "/earn/sponsors?booking=" + b.ID.String()
		// Kept as messages, so each member reads them in their own language.
		send := func(what, kind, severity string, title, body i18n.Text) error {
			if containsStr(b.Notified, what) {
				return nil
			}
			return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				q := store.New(tx)
				if err := notify.ToMembersWith(ctx, tx, b.WorkspaceID, authz.SponsorsManage, notify.Notification{WorkspaceID: &b.WorkspaceID, Kind: kind, Severity: severity,
					TitleText: title, BodyText: body, Link: link, Data: map[string]any{"booking": b.ID.String()}}); err != nil {
					return err
				}
				return q.SponsorBookingNotified(ctx, store.SponsorBookingNotifiedParams{ID: b.ID, What: what})
			})
		}
		if b.Status == "booked" && b.CreativeStatus == "waiting" && b.CreativeDue != nil && !civil(b.IssueOn).Before(today) {
			due := civil(*b.CreativeDue)
			switch {
			case due.Equal(today.AddDate(0, 0, 1)):
				if err := send("creative_due", notify.KindSponsorCreative, "warn", msgs.M("notify.creativeDueTitle").Keep(issue),
					msgs.M("notify.creativeDueBody").Keep(issue)); err != nil {
					return err
				}
			case !due.After(today):
				if err := send("creative_late", notify.KindSponsorCreative, "warn", msgs.M("notify.creativeLateTitle").Keep(issue),
					msgs.M("notify.creativeLateBody").Keep(issue)); err != nil {
					return err
				}
			}
		}
		if (b.Status == "booked" || b.Status == "sent") && b.PaymentStatus == "unpaid" && b.PaymentDue != nil && civil(*b.PaymentDue).Before(today) {
			// The due day at noon in the workspace's zone: the date reads
			// the same wherever the reader is.
			due := civil(*b.PaymentDue)
			args := i18n.Args{"issue": b.IssueNumber, "sponsor": b.SponsorName, "amount": money(b.PriceCents, b.Currency),
				"at": time.Date(due.Year(), due.Month(), due.Day(), 12, 0, 0, 0, location(ws))}
			if err := send("payment_overdue", notify.KindSponsorPayment, "warn", msgs.M("notify.paymentTitle").Keep(args),
				msgs.M("notify.paymentBody").Keep(args)); err != nil {
				return err
			}
		}
	}
	return nil
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
