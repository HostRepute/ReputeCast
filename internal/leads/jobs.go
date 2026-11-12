package leads

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/leads/meta"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// RegisterJobs adds the lead ads workers and their schedules: deliveries
// applied as they arrive (and requeued every minute when a job was lost),
// every connection checked every 15 minutes (its token, its forms, and its
// active forms polled while the webhook is quiet), past leads imported on
// request.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &ApplyWorker{S: s})
	river.AddWorker(reg.Workers, &ImportWorker{S: s})
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	river.AddWorker(reg.Workers, &HealthWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, false)
	reg.Every(HealthEvery, func() (river.JobArgs, *river.InsertOpts) { return HealthArgs{}, nil }, false)
}

// TickArgs runs every minute.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "leads.tick" }

// InsertOpts: one tick at a time.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// TickWorker requeues deliveries left unapplied.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.S.Requeue(ctx) }

// Requeue applies deliveries still pending two minutes after they arrived
// (a lost job, or a connection that works again). ApplyEvent leaves those
// of a connection that's held as they are.
func (s *Service) Requeue(ctx context.Context) error {
	jc := s.jobs()
	if jc == nil {
		return nil
	}
	ids, err := s.Q.LeadEventsPending(ctx, s.now().Add(-2*time.Minute))
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := jc.Insert(ctx, ApplyArgs{EventID: id.String()}, &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 10 * time.Minute}}); err != nil {
			return err
		}
	}
	return nil
}

// requeueConnection applies a connection's held deliveries now (its
// credentials were replaced).
func (s *Service) requeueConnection(ctx context.Context, _ uuid.UUID) {
	if err := s.Requeue(ctx); err != nil {
		s.log().WarnContext(ctx, "lead ads: requeue deliveries", "err", err)
	}
}

// HealthArgs checks every connection.
type HealthArgs struct{}

// Kind implements river.JobArgs.
func (HealthArgs) Kind() string { return "leads.health" }

// InsertOpts: one check at a time.
func (HealthArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Minute}}
}

// HealthWorker checks connections.
type HealthWorker struct {
	river.WorkerDefaults[HealthArgs]
	S *Service
}

// Work implements river.Worker.
func (w *HealthWorker) Work(ctx context.Context, _ *river.Job[HealthArgs]) error {
	return w.S.Health(ctx)
}

// Health checks every connection (the demo's is never polled) and purges
// deliveries older than 30 days.
func (s *Service) Health(ctx context.Context) error {
	if _, err := s.Q.LeadEventsPurge(ctx, s.now().Add(-Kept)); err != nil {
		return err
	}
	conns, err := s.Q.LeadConnectionsActive(ctx)
	if err != nil {
		return err
	}
	_, reachable := s.reachable(ctx)
	var errs []error
	for _, c := range conns {
		if c.Demo {
			continue
		}
		if err := s.Check(ctx, c, reachable); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Check checks one connection: its token (a refused one needs attention
// and is told about once), its Pages' forms, and whether the webhook
// works. While Meta doesn't push leads (the subscription was refused, the
// instance isn't reachable, or nothing came for 24 hours while forms are
// active) the active forms are polled.
func (s *Service) Check(ctx context.Context, c store.LeadConnection, reachable bool) error {
	now := s.now()
	if c.RateLimitedUntil != nil && c.RateLimitedUntil.After(now) {
		return nil
	}
	cl, err := s.client(c)
	if err != nil {
		return nil //nolint:nilerr // nothing to check without credentials
	}
	if _, _, err := cl.Me(ctx); err != nil {
		if s.noteProblem(ctx, c, err) {
			return nil
		}
		if _, ok := s.rateLimited(ctx, c, err); ok {
			return nil
		}
		return nil //nolint:nilerr // Meta didn't answer: the next check tries again
	}
	if err := s.Q.LeadConnectionChecked(ctx, store.LeadConnectionCheckedParams{ID: c.ID, At: now}); err != nil {
		return err
	}
	if c.Status == StatusAttention {
		// The token works again (permissions given back, say).
		if err := s.Q.LeadConnectionSetStatus(ctx, store.LeadConnectionSetStatusParams{ID: c.ID, Status: StatusConnected, StatusDetail: "", At: now}); err != nil {
			return err
		}
		if err := s.Q.LeadConnectionNotified(ctx, store.LeadConnectionNotifiedParams{ID: c.ID}); err != nil {
			return err
		}
		c.Status = StatusConnected
	}
	if err := s.syncForms(ctx, c, cl); err != nil {
		if s.noteProblem(ctx, c, err) {
			return nil
		}
		if _, ok := s.rateLimited(ctx, c, err); ok {
			return nil
		}
	}
	forms, err := s.Q.LeadFormsCollecting(ctx, c.ID)
	if err != nil {
		return err
	}
	reason := pollingReason(c, reachable, len(forms) > 0, now)
	if reason != c.PollingReason {
		if err := s.Q.LeadConnectionPolling(ctx, store.LeadConnectionPollingParams{ID: c.ID, PollingReason: reason}); err != nil {
			return err
		}
	}
	if reason == "" {
		return nil
	}
	return s.poll(ctx, c, cl, forms)
}

// pollingReason is why leads must be polled ("" when the webhook works).
func pollingReason(c store.LeadConnection, reachable, active bool, now time.Time) string {
	switch {
	case c.SubscribedAt == nil:
		return PollingNotSubscribed
	case !reachable:
		return PollingUnreachable
	case !active:
		return ""
	}
	last := *c.SubscribedAt
	if c.LastEventAt != nil && c.LastEventAt.After(last) {
		last = *c.LastEventAt
	}
	if now.Sub(last) >= QuietAfter {
		return PollingQuiet
	}
	return ""
}

// maxPollPages bounds one form's poll (100 leads a page).
const maxPollPages = 10

// poll reads the active forms' leads since they were last polled (the
// day before, the first time) and records each once.
func (s *Service) poll(ctx context.Context, c store.LeadConnection, cl *meta.Client, forms []store.LeadFormsCollectingRow) error {
	for _, f := range forms {
		pg, err := s.pageOf(f.PageID, f.MetaPageID, f.PageTokenSealed)
		if err != nil {
			continue
		}
		start := s.now()
		since := start.Add(-QuietAfter - HealthEvery)
		if f.LastPolledAt != nil {
			// A minute's overlap: a lead made as the last poll ran.
			since = f.LastPolledAt.Add(-time.Minute)
		}
		if err := s.readLeads(ctx, c, cl, pg, collectOfRow(f), since, SourcePoll, maxPollPages, nil); err != nil {
			if s.noteProblem(ctx, c, err) {
				return nil
			}
			if _, ok := s.rateLimited(ctx, c, err); ok {
				return nil
			}
			return err
		}
		if err := s.Q.LeadFormPolled(ctx, store.LeadFormPolledParams{ID: f.ID, At: start}); err != nil {
			return err
		}
	}
	return nil
}

// readLeads pages through a form's leads since a moment, recording the
// ones not seen yet; progress, when given, hears how many were read.
func (s *Service) readLeads(ctx context.Context, c store.LeadConnection, cl *meta.Client, pg meta.Page, f collect, since time.Time, source string, maxPages int, progress func(read int)) error {
	after := ""
	read := 0
	for range maxPages {
		page, err := cl.FormLeads(ctx, pg, f.MetaID, since, after)
		if err != nil {
			return err
		}
		for _, lead := range page.Leads {
			read++
			seen, err := s.Q.LeadEntryExists(ctx, store.LeadEntryExistsParams{FormID: f.ID, LeadgenID: lead.ID})
			if err != nil {
				return err
			}
			if seen {
				continue
			}
			if _, err := s.record(ctx, f, lead, source); err != nil {
				return err
			}
		}
		if progress != nil {
			progress(read)
		}
		if page.Next == "" {
			return nil
		}
		after = page.Next
	}
	return nil
}

// noteProblem puts a connection whose token Meta refused in attention,
// and tells the people who can manage integrations once. It reports
// whether err was such a refusal.
func (s *Service) noteProblem(ctx context.Context, c store.LeadConnection, err error) bool {
	problem := problemOf(err)
	if problem == "" {
		return false
	}
	now := s.now()
	if err := s.Q.LeadConnectionSetStatus(ctx, store.LeadConnectionSetStatusParams{ID: c.ID, Status: StatusAttention, StatusDetail: problem, At: now}); err != nil {
		s.log().WarnContext(ctx, "lead ads: mark the connection", "connection", c.ID, "err", err)
		return true
	}
	if c.NotifiedAt != nil {
		return true
	}
	ws, err := s.Q.GetWorkspace(ctx, c.WorkspaceID)
	if err != nil {
		return true
	}
	if err := notify.ToMembersWith(ctx, s.Pool, c.WorkspaceID, authz.IntegrationsManage, notify.Notification{
		Kind: notify.KindLeadAdsAttention, Severity: "bad",
		TitleText: mNotifyTitle.Keep(nil),
		BodyText:  mNotifyBody.Keep(i18n.Args{"name": c.Name}),
		Link:      "/w/" + ws.Slug + "/settings/integrations/leads/" + c.ID.String(),
		Data:      map[string]any{"connection": c.ID.String(), "problem": problem},
	}); err != nil {
		s.log().WarnContext(ctx, "lead ads: notify", "connection", c.ID, "err", err)
		return true
	}
	if err := s.Q.LeadConnectionNotified(ctx, store.LeadConnectionNotifiedParams{ID: c.ID, At: &now}); err != nil {
		s.log().WarnContext(ctx, "lead ads: keep the notification", "connection", c.ID, "err", err)
	}
	return true
}

// ImportArgs imports a form's leads of the last 90 days.
type ImportArgs struct {
	FormID string `json:"form_id"`
}

// Kind implements river.JobArgs.
func (ImportArgs) Kind() string { return "leads.import" }

// InsertOpts runs on the default queue.
func (ImportArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 5}
}

// ImportWorker imports past leads.
type ImportWorker struct {
	river.WorkerDefaults[ImportArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ImportWorker) Work(ctx context.Context, job *river.Job[ImportArgs]) error {
	id, err := uuid.Parse(job.Args.FormID)
	if err != nil {
		return river.JobCancel(err)
	}
	return w.S.Import(ctx, id, job.Attempt >= job.MaxAttempts)
}

// maxImportPages bounds an import (100 leads a page).
const maxImportPages = 200

// Import reads a form's leads of the last 90 days and records each once.
// Leads already recorded are skipped, so a retried import carries on; a
// finished one clears the count of leads that waited for the mapping.
func (s *Service) Import(ctx context.Context, formID uuid.UUID, last bool) error {
	f, err := s.Q.LeadFormAny(ctx, formID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if f.ImportState != "running" {
		return nil
	}
	fail := func() error {
		return s.Q.LeadFormSetImport(ctx, store.LeadFormSetImportParams{ID: f.ID, ImportState: "failed", ImportCount: f.ImportCount, ImportedAt: f.ImportedAt})
	}
	c, err := s.Q.LeadConnectionAny(ctx, f.ConnectionID)
	if err != nil {
		return err
	}
	if c.Status == StatusDisconnected || f.MappedAt == nil {
		return fail()
	}
	cl, err := s.client(c)
	if err != nil {
		return fail()
	}
	pg, err := s.pageOf(f.PageID, f.MetaPageID, f.PageTokenSealed)
	if err != nil {
		return fail()
	}
	form, err := s.Q.LeadFormByMetaID(ctx, store.LeadFormByMetaIDParams{PageID: f.PageID, FormID: f.FormID})
	if err != nil {
		return err
	}
	count := int32(0)
	err = s.readLeads(ctx, c, cl, pg, collectOf(form), s.now().Add(-ImportWindow), SourceImport, maxImportPages, func(read int) {
		count = int32(read)
		_ = s.Q.LeadFormImportProgress(ctx, store.LeadFormImportProgressParams{ID: f.ID, ImportCount: count})
	})
	if err != nil {
		if s.noteProblem(ctx, c, err) {
			return fail()
		}
		if wait, ok := s.rateLimited(ctx, c, err); ok {
			return river.JobSnooze(wait)
		}
		if last {
			return fail()
		}
		return err
	}
	now := s.now()
	if err := s.Q.LeadFormSetImport(ctx, store.LeadFormSetImportParams{ID: f.ID, ImportState: "done", ImportCount: count, ImportedAt: &now}); err != nil {
		return err
	}
	return s.Q.LeadFormClearWaiting(ctx, f.ID)
}
