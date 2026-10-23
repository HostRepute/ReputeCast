package earn

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Notification kinds of Earn (registered in internal/notify).
const (
	KindPaymentsAttention = notify.KindPaymentsAttention
	KindUnmappedPrice     = notify.KindUnmappedPrice
	KindPaidImportDone    = notify.KindPaidImportDone
	KindFailedPayments    = notify.KindFailedPayments
)

// RetentionDays is how long raw payment events are kept.
const RetentionDays = 90

// HourlyArgs is the hourly check: every connected account's missed
// events, its webhook and its key.
type HourlyArgs struct{}

// Kind implements river.JobArgs.
func (HourlyArgs) Kind() string { return "earn.hourly" }

// InsertOpts runs on the events queue, once at a time.
func (HourlyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// HourlyWorker runs HourlyArgs.
type HourlyWorker struct {
	river.WorkerDefaults[HourlyArgs]
	S *Service
}

// Timeout bounds one run.
func (w *HourlyWorker) Timeout(*river.Job[HourlyArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *HourlyWorker) Work(ctx context.Context, _ *river.Job[HourlyArgs]) error {
	return w.S.Hourly(ctx)
}

// SweepArgs applies events whose job never ran, ends access that has run
// out and purges raw events past retention.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "earn.sweep" }

// InsertOpts runs on the maintenance queue.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

// SweepWorker runs SweepArgs.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error { return w.S.Sweep(ctx) }

// DailyArgs sends the daily summary of failed payments to the people who
// turned it on.
type DailyArgs struct{}

// Kind implements river.JobArgs.
func (DailyArgs) Kind() string { return "earn.daily" }

// InsertOpts runs on the system queue, once a day.
func (DailyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSystem, UniqueOpts: river.UniqueOpts{ByPeriod: 20 * time.Hour}}
}

// DailyWorker runs DailyArgs.
type DailyWorker struct {
	river.WorkerDefaults[DailyArgs]
	S *Service
}

// Work implements river.Worker.
func (w *DailyWorker) Work(ctx context.Context, _ *river.Job[DailyArgs]) error {
	return w.S.DailyFailed(ctx)
}

// RegisterJobs adds Earn's workers and periodic jobs.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &ApplyWorker{S: s})
	river.AddWorker(reg.Workers, &HourlyWorker{S: s})
	river.AddWorker(reg.Workers, &SweepWorker{S: s})
	river.AddWorker(reg.Workers, &DailyWorker{S: s})
	river.AddWorker(reg.Workers, &ImportWorker{S: s})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return HourlyArgs{}, nil }, false)
	reg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return SweepArgs{}, nil }, true)
	reg.Every(24*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return DailyArgs{}, nil }, false)
}

// Hourly checks every connected account: it reads the events the webhook
// missed and applies them, notices a webhook deleted in the provider or a
// rejected key, and tells the people with payments:manage once per problem.
func (s *Service) Hourly(ctx context.Context) error {
	conns, err := s.Q.PayConnectionsActive(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, c := range conns {
		if _, err := s.gapCheck(ctx, c); err != nil && !errors.Is(err, payments.ErrUnavailable) {
			errs = append(errs, err)
		}
		if err := s.healthCheck(ctx, c); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// gapCheck reads the provider's list of events since the last one (an
// hour of overlap) and stores and applies any the webhook didn't deliver.
func (s *Service) gapCheck(ctx context.Context, c store.PaymentConnection) (int, error) {
	client, err := s.open(c)
	if err != nil {
		return 0, err
	}
	now := s.now()
	since := now.Add(-2 * time.Hour)
	if c.GapCheckedAt != nil && c.GapCheckedAt.Add(-time.Hour).Before(since) {
		since = c.GapCheckedAt.Add(-time.Hour)
	}
	if c.WebhookMissingAt != nil && c.WebhookMissingAt.Before(since) {
		since = c.WebhookMissingAt.Add(-5 * time.Minute)
	}
	if c.LastEventAt != nil && c.LastEventAt.Before(since) {
		since = c.LastEventAt.Add(-5 * time.Minute)
	}
	if now.Sub(since) > 72*time.Hour {
		since = now.Add(-72 * time.Hour)
	}
	d, _ := payments.Lookup(c.Provider)
	events, err := client.ListEvents(ctx, since, d.WebhookEvents)
	if err != nil {
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: c.ID, At: now})
		}
		return 0, err
	}
	applied := 0
	types := map[string]int{}
	for _, ev := range events {
		if ev.Livemode != (c.Mode == "live") {
			continue
		}
		dup, err := s.store(ctx, c, ev, ev.Raw, "gap_check", now)
		if err != nil {
			return applied, err
		}
		if dup {
			continue
		}
		applied++
		types[ev.Type]++
	}
	if applied > 0 {
		if _, err := s.ApplyPending(ctx, now.Add(time.Second)); err != nil {
			return applied, err
		}
	}
	var hist []EarnGapRun
	_ = json.Unmarshal(c.GapHistory, &hist)
	hist = append([]EarnGapRun{{At: now, Applied: applied, Types: types}}, hist...)
	if len(hist) > 6 {
		hist = hist[:6]
	}
	b, _ := json.Marshal(hist)
	return applied, s.Q.PayConnectionGapChecked(ctx, store.PayConnectionGapCheckedParams{ID: c.ID, At: &now, GapHistory: b})
}

// healthCheck notices a deleted webhook, a rejected key and a silent
// webhook, and notifies the people who manage payments once per problem.
func (s *Service) healthCheck(ctx context.Context, c store.PaymentConnection) error {
	now := s.now()
	ws, err := s.Q.GetWorkspace(ctx, c.WorkspaceID)
	if err != nil {
		return err
	}
	client, err := s.open(c)
	if errors.Is(err, errNoKey) {
		return nil
	}
	if err != nil {
		return err
	}
	if c.WebhookID != "" && !c.WebhookManual && c.WebhookMissingAt == nil {
		ok, err := client.WebhookExists(ctx, c.WebhookID)
		switch {
		case errors.Is(err, payments.ErrKeyRejected):
			if c.KeyRejectedAt == nil {
				_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: c.ID, At: now})
				return s.notifyAttention(ctx, &ws, c, "key_rejected", now)
			}
		case err == nil && !ok:
			_ = s.Q.PayConnectionWebhookMissing(ctx, store.PayConnectionWebhookMissingParams{ID: c.ID, At: now})
			return s.notifyAttention(ctx, &ws, c, "webhook_missing", now)
		}
	}
	if c.LastEventAt != nil && now.Sub(*c.LastEventAt) > SilentAfter && s.activeSubscriptions(ctx, c.WorkspaceID, c.Mode) >= SilentMinimum {
		var notified bool
		_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM notifications WHERE workspace_id = $1 AND kind = $2 AND data->>'reason' = 'webhook_silent'
			AND data->>'connection' = $3 AND created_at > $4)`, c.WorkspaceID, KindPaymentsAttention, c.ID.String(), *c.LastEventAt).Scan(&notified)
		if !notified {
			return s.notifyAttention(ctx, &ws, c, "webhook_silent", now)
		}
	}
	return nil
}

func (s *Service) notifyAttention(ctx context.Context, ws *store.Workspace, c store.PaymentConnection, reason string, now time.Time) error {
	// Kept as messages, so each member reads it in their own language; the
	// time in the workspace's zone.
	args := i18n.Args{"account": c.AccountName, "at": now.In(location(ws))}
	n := notify.Notification{WorkspaceID: &ws.ID, Kind: KindPaymentsAttention, Severity: "bad",
		TitleText: msgs.M("notify.attention." + reason + ".title").Keep(nil), BodyText: msgs.M("notify.attention." + reason + ".body").Keep(args),
		Link: "/w/" + ws.Slug + "/earn/settings?tab=payments", Data: map[string]any{"reason": reason, "connection": c.ID.String(), "mode": c.Mode}}
	if reason == "webhook_silent" {
		n.Severity = "warn"
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return notify.ToMembersWith(ctx, tx, ws.ID, authz.PaymentsManage, n)
	})
}

// Sweep applies events stored more than two minutes ago whose job never
// ran, ends access that has run out (complimentary access past its date,
// gifts past their end, and paid access a disconnected account can no
// longer renew) and purges raw events past 90 days.
func (s *Service) Sweep(ctx context.Context) error {
	now := s.now()
	if _, err := s.ApplyPending(ctx, now.Add(-2*time.Minute)); err != nil {
		return err
	}
	if err := s.EndRunOut(ctx); err != nil {
		return err
	}
	_, err := s.Q.PayEventsPurge(ctx, now.AddDate(0, 0, -RetentionDays))
	return err
}

// EndRunOut ends access that has run out.
func (s *Service) EndRunOut(ctx context.Context) error {
	now := s.now()
	rows, err := s.Q.EarnGrantsToEnd(ctx, store.EarnGrantsToEndParams{Now: now, GraceBefore: now.Add(-24 * time.Hour)})
	if err != nil {
		return err
	}
	for _, r := range rows {
		reason := "grant_ended"
		switch r.Kind {
		case "gift":
			reason = "gift_ended"
		case "stripe":
			reason = "disconnected"
		}
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.EarnSubscriptionEnd(ctx, store.EarnSubscriptionEndParams{ID: r.ID, At: now, EndReason: reason}); err != nil {
				return err
			}
			return s.timeline(ctx, q, r.WorkspaceID, r.ContactID, &r.ID, r.Mode, "access_ended", now, nil, "", nil, "", nil, map[string]any{"reason": reason})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// DailyFailed tells the people who turned on the daily summary how many
// payments failed in the last day (earn:view; off by default).
func (s *Service) DailyFailed(ctx context.Context) error {
	now := s.now()
	rows, err := s.Pool.Query(ctx, `SELECT s.workspace_id, count(*) FROM paid_subscriptions s
		WHERE s.mode = 'live' AND s.status = 'past_due' AND s.failed_at >= $1 GROUP BY s.workspace_id`, now.Add(-24*time.Hour))
	if err != nil {
		return err
	}
	type wsCount struct {
		ws store.Workspace
		n  int64
	}
	var todo []wsCount
	for rows.Next() {
		var w wsCount
		var id [16]byte
		if err := rows.Scan(&id, &w.n); err != nil {
			rows.Close()
			return err
		}
		ws, err := s.Q.GetWorkspace(ctx, id)
		if err != nil {
			continue
		}
		w.ws = ws
		todo = append(todo, w)
	}
	rows.Close()
	for _, w := range todo {
		ids, err := notify.MembersWith(ctx, s.Pool, w.ws.ID, authz.EarnView)
		if err != nil {
			return err
		}
		n := notify.Notification{WorkspaceID: &w.ws.ID, Kind: KindFailedPayments, Severity: "warn",
			TitleText: msgs.M("notify.failedTitle").Keep(i18n.Args{"count": w.n}),
			BodyText:  msgs.M("notify.failedBody").Keep(i18n.Args{"count": w.n}),
			Link:      "/w/" + w.ws.Slug + "/earn/members?status=past_due"}
		for _, uid := range ids {
			// Off by default: only people who turned it on get it.
			var on bool
			_ = s.Pool.QueryRow(ctx, `SELECT coalesce(in_app OR email, false) FROM notification_preferences WHERE user_id = $1 AND event_type = $2`, uid, KindFailedPayments).Scan(&on)
			if !on {
				continue
			}
			if err := notify.ToUser(ctx, s.Pool, uid, n); err != nil {
				return err
			}
		}
	}
	return nil
}
