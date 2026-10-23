package commerce

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// RegisterJobs adds the stores' workers and their schedules: deliveries
// applied as they arrive, first imports and reconciliations, abandoned
// checkouts and unsubscribes passed back every minute, webhooks checked
// every 15 minutes (a missing one's data polled meanwhile), the nightly
// reconciliation at 03:00 in each workspace's timezone.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &ApplyWorker{S: s})
	river.AddWorker(reg.Workers, &ImportWorker{S: s})
	river.AddWorker(reg.Workers, &ReconcileWorker{S: s})
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	river.AddWorker(reg.Workers, &HealthWorker{S: s})
	river.AddWorker(reg.Workers, &NightlyWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, false)
	reg.Every(HealthEvery, func() (river.JobArgs, *river.InsertOpts) { return HealthArgs{}, nil }, false)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return NightlyArgs{}, nil }, true)
}

// TickArgs runs every minute.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "commerce.tick" }

// InsertOpts: one tick at a time.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// TickWorker finds abandoned checkouts, passes unsubscribes back and
// requeues deliveries left unapplied.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error {
	return w.S.Tick(ctx)
}

// Tick is one minute's work.
func (s *Service) Tick(ctx context.Context) error {
	var errs []error
	if _, err := s.DetectAbandoned(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.writeBack(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.requeue(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// requeue applies deliveries still pending two minutes after they arrived
// (a lost job).
func (s *Service) requeue(ctx context.Context) error {
	jc := s.jobs()
	if jc == nil {
		return nil
	}
	ids, err := s.Q.CommerceEventsPending(ctx, s.now().Add(-2*time.Minute))
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := jc.Insert(ctx, ApplyArgs{EventID: id.String()}, nil); err != nil {
			return err
		}
	}
	return nil
}

// HealthArgs checks every store's webhooks.
type HealthArgs struct{}

// Kind implements river.JobArgs.
func (HealthArgs) Kind() string { return "commerce.health" }

// InsertOpts: one check at a time.
func (HealthArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Minute}}
}

// HealthWorker checks stores' webhooks.
type HealthWorker struct {
	river.WorkerDefaults[HealthArgs]
	S *Service
}

// Work implements river.Worker.
func (w *HealthWorker) Work(ctx context.Context, _ *river.Job[HealthArgs]) error {
	return w.S.Health(ctx)
}

// Health checks every connected store (the demo's are never polled): its
// webhooks are still there, its credentials still work. A missing webhook
// is marked and told about once; its data is polled every 15 minutes until
// it's registered again, so nothing is missed. A refused token disconnects
// the store.
func (s *Service) Health(ctx context.Context) error {
	sts, err := s.Q.CommerceStoresActive(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, st := range sts {
		if err := s.CheckStore(ctx, st); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// CheckStore checks one store's webhooks and polls the missing ones' data.
func (s *Service) CheckStore(ctx context.Context, st store.Store) error {
	c, err := s.client(st)
	if err != nil {
		return nil //nolint:nilerr // a store without credentials has nothing to check
	}
	now := s.now()
	hooks, err := c.Webhooks(ctx)
	if err != nil {
		switch {
		case errors.Is(err, stores.ErrUnauthorized):
			return s.revoke(ctx, st, "revoked", now)
		default:
			if wait, ok := stores.IsRateLimited(err); ok {
				return s.rateLimited(ctx, st, wait)
			}
		}
		return nil
	}
	rows, err := s.Q.CommerceWebhooks(ctx, st.ID)
	if err != nil {
		return err
	}
	addr := s.WebhookURL(st)
	present := func(topic string) bool {
		return slices.ContainsFunc(hooks, func(w stores.Webhook) bool { return w.Topic == topic && w.Address == addr })
	}
	var newly []string
	resources := map[string]time.Time{}
	for _, r := range rows {
		missing := r.Status == "missing"
		if r.Status == "registered" && !present(r.Topic) {
			if r.Topic == shopify.LifecycleTopic {
				// Not a data webhook: registered again quietly.
				_ = s.registerWebhooks(ctx, st, nil, true)
				continue
			}
			if err := s.Q.CommerceWebhookMissing(ctx, store.CommerceWebhookMissingParams{StoreID: st.ID, Topic: r.Topic, At: now}); err != nil {
				return err
			}
			newly = append(newly, r.Topic)
			missing = true
		}
		if !missing {
			continue
		}
		since := now.Add(-HealthEvery)
		switch {
		case r.LastPolledAt != nil:
			since = *r.LastPolledAt
		case r.MissingSince != nil:
			since = *r.MissingSince
		}
		if res := topicResource(r.Topic); res != "" {
			if at, ok := resources[res]; !ok || since.Before(at) {
				resources[res] = since
			}
		}
		if err := s.Q.CommerceWebhookPolled(ctx, store.CommerceWebhookPolledParams{StoreID: st.ID, Topic: r.Topic, At: now}); err != nil {
			return err
		}
	}
	for res, since := range resources {
		rng := stores.Range{UpdatedSince: since.Add(-time.Minute)}
		var err error
		switch res {
		case "orders":
			_, err = s.pullOrders(ctx, c, st, rng, sourcePoll)
		case "checkouts":
			err = s.pullCheckouts(ctx, c, st, rng, sourcePoll)
		case "customers":
			err = s.pullCustomers(ctx, c, st, rng, sourcePoll)
		case "products":
			err = s.pullProducts(ctx, c, st, rng)
		}
		if wait, ok := stores.IsRateLimited(err); ok {
			_ = s.rateLimited(ctx, st, wait)
		}
	}
	if len(newly) > 0 {
		if err := s.webhooksMissing(ctx, st, newly, now); err != nil {
			return err
		}
	}
	return s.Q.CommerceStoreHealthChecked(ctx, store.CommerceStoreHealthCheckedParams{ID: st.ID, At: now})
}

func (s *Service) pullProducts(ctx context.Context, c stores.Client, st store.Store, r stores.Range) error {
	cursor := ""
	for {
		page, err := c.Products(ctx, cursor, r)
		if err != nil {
			return err
		}
		for _, p := range page.Items {
			if _, err := s.applyProduct(ctx, s.Q, st, p, true); err != nil {
				return err
			}
		}
		if page.Next == "" {
			return nil
		}
		cursor = page.Next
	}
}

// webhooksMissing records and tells about webhooks gone from the store.
func (s *Service) webhooksMissing(ctx context.Context, st store.Store, topics []string, at time.Time) error {
	ws, err := s.Q.GetWorkspace(ctx, st.WorkspaceID)
	if err != nil {
		return err
	}
	for _, t := range topics {
		if err := s.activity(ctx, s.Q, st, activity{Kind: "webhook_missing", Tone: "warn", Topic: t, At: at}); err != nil {
			return err
		}
	}
	// Kept as messages, so each member reads it in their own language; the
	// time in the workspace's zone.
	return notify.ToMembersWith(ctx, s.Pool, st.WorkspaceID, authz.IntegrationsManage, notify.Notification{
		Kind: KindStoreProblem, Severity: "warn",
		TitleText: mNotifyWebhookTitle.Keep(i18n.Args{"topic": topics[0], "store": st.Name}),
		BodyText:  mNotifyWebhookBody.Keep(i18n.Args{"platform": platformName(st.Platform), "at": at.In(location(ws.Timezone))}),
		Link:      "/w/" + ws.Slug + "/settings/integrations/stores/" + st.ID.String(),
		Data:      map[string]any{"store": st.ID.String(), "problem": "webhook_missing", "topics": topics},
	})
}

// NightlyArgs runs every hour: each store is reconciled once a night, at
// 03:00 in its workspace's timezone; old events and activity are purged.
type NightlyArgs struct{}

// Kind implements river.JobArgs.
func (NightlyArgs) Kind() string { return "commerce.nightly" }

// InsertOpts: one run at a time.
func (NightlyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

// NightlyWorker schedules the reconciliations.
type NightlyWorker struct {
	river.WorkerDefaults[NightlyArgs]
	S *Service
}

// Work implements river.Worker.
func (w *NightlyWorker) Work(ctx context.Context, _ *river.Job[NightlyArgs]) error {
	return w.S.Nightly(ctx)
}

// Nightly queues the reconciliations due and purges what's older than 30
// days.
func (s *Service) Nightly(ctx context.Context) error {
	now := s.now()
	if _, err := s.Q.CommerceEventsPurge(ctx, now.Add(-Kept)); err != nil {
		return err
	}
	if _, err := s.Q.CommerceActivityPurge(ctx, now.Add(-Kept)); err != nil {
		return err
	}
	sts, err := s.Q.CommerceStoresActive(ctx)
	if err != nil {
		return err
	}
	jc := s.jobs()
	if jc == nil {
		return nil
	}
	tz := map[uuid.UUID]*time.Location{}
	for _, st := range sts {
		if st.Status != StatusConnected {
			continue
		}
		loc, ok := tz[st.WorkspaceID]
		if !ok {
			ws, err := s.Q.GetWorkspace(ctx, st.WorkspaceID)
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			loc = location(ws.Timezone)
			tz[st.WorkspaceID] = loc
		}
		if !nightlyDue(now, loc, st.ReconciledAt) {
			continue
		}
		if _, err := jc.Insert(ctx, ReconcileArgs{StoreID: st.ID.String()}, nil); err != nil {
			return err
		}
	}
	return nil
}

// nightlyDue reports whether a store's reconciliation for tonight hasn't
// run: it's past 03:00 local and the last one was before.
func nightlyDue(now time.Time, loc *time.Location, last *time.Time) bool {
	local := now.In(loc)
	at := time.Date(local.Year(), local.Month(), local.Day(), NightlyHour, 0, 0, 0, loc)
	if local.Before(at) {
		return false
	}
	return last == nil || last.Before(at)
}
