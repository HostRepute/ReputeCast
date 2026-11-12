package inbound

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/smtp/bouncebox"
	"github.com/hostrepute/reputecast/internal/store"
)

// Periodic work: reading mailbox routes, requeueing and purging, watching
// provider routes' MX records and telling members.

// Retention.
const (
	rawKeep      = 7 * 24 * time.Hour
	messagesKeep = 365 * 24 * time.Hour
	// staleAfter is how long a stored arrival may wait for its job before
	// the sweep queues it again.
	staleAfter = 2 * time.Minute
	// noticeEvery is how often members are told about new replies, at most.
	noticeEvery = 15 * time.Minute
	// mailboxBatch is how many messages one poll of a mailbox reads.
	mailboxBatch = 200
)

// MailboxArgs reads every mailbox route.
type MailboxArgs struct{}

// Kind implements river.JobArgs.
func (MailboxArgs) Kind() string { return "inbound.mailbox" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (MailboxArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// MailboxWorker runs MailboxArgs.
type MailboxWorker struct {
	river.WorkerDefaults[MailboxArgs]
	S *Service
}

// Timeout bounds one round of polls.
func (w *MailboxWorker) Timeout(*river.Job[MailboxArgs]) time.Duration { return 3 * time.Minute }

// Work implements river.Worker.
func (w *MailboxWorker) Work(ctx context.Context, _ *river.Job[MailboxArgs]) error {
	routes, err := w.S.Q.InboundRoutesMailbox(ctx)
	if err != nil {
		return err
	}
	for _, r := range routes {
		if _, err := w.S.Poll(ctx, r); err != nil {
			w.S.Log.WarnContext(ctx, "inbound mailbox", "route", r.ID, "err", err)
		}
	}
	return nil
}

// mailboxConfig is the bouncebox configuration of a mailbox route.
func (s *Service) mailboxConfig(r store.InboundRoute) (bouncebox.Config, error) {
	var m mailboxSettings
	_ = json.Unmarshal(r.MailboxSettings, &m)
	pw := ""
	if len(r.MailboxPasswordSealed) > 0 {
		var err error
		if pw, err = s.Box.OpenString(r.MailboxPasswordSealed, passwordAD(r.ID)); err != nil {
			return bouncebox.Config{}, err
		}
	}
	return bouncebox.Config{Protocol: m.Protocol, Host: m.Host, Port: m.Port, TLS: m.Security, Username: m.Username, Password: pw,
		Address: m.Address, Timeout: 30 * time.Second, Dialer: s.Guard.Dialer(30 * time.Second)}, nil
}

// Poll reads up to 200 new messages from a mailbox route, stores each with
// its job, and marks them read (IMAP) or removes them (POP3). A mailbox
// that can't be read makes the route failing, with the reason.
func (s *Service) Poll(ctx context.Context, r store.InboundRoute) (int, error) {
	cfg, err := s.mailboxConfig(r)
	if err != nil {
		return 0, err
	}
	open := s.OpenBox
	if open == nil {
		open = bouncebox.Open
	}
	box, err := open(ctx, cfg)
	if err != nil {
		s.mailboxFailed(ctx, r, err)
		return 0, err
	}
	defer func() { _ = box.Close() }()
	msgs, err := box.Fetch(ctx, mailboxBatch)
	if err != nil {
		s.mailboxFailed(ctx, r, err)
		return 0, err
	}
	acked := make([]string, 0, len(msgs))
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		acked = acked[:0]
		for _, m := range msgs {
			if len(m.Raw) == 0 {
				continue
			}
			body := m.Raw
			if len(body) > MaxHookBody {
				body = body[:MaxHookBody]
			}
			if err := s.storeRaw(ctx, tx, r, "mailbox", "message/rfc822", nil, body, true); err != nil {
				return err
			}
			acked = append(acked, m.ID)
		}
		return store.New(tx).InboundRoutePolled(ctx, store.InboundRoutePolledParams{ID: r.ID, At: now})
	})
	if err != nil {
		return 0, err
	}
	if len(acked) == 0 {
		return 0, nil
	}
	return len(acked), box.Ack(ctx, acked)
}

// mailboxFailed makes a mailbox route failing with the reason, and tells
// the members who manage domains once.
func (s *Service) mailboxFailed(ctx context.Context, r store.InboundRoute, err error) {
	key, args := "status.mailboxUnreachable", i18n.Args{"host": hostOf(r), "error": err.Error()}
	if errors.Is(err, bouncebox.ErrAuth) {
		key, args = "status.mailboxAuth", i18n.Args{"host": hostOf(r)}
	}
	detail := keepDetail(msgs.M(key).Keep(args))
	if e := s.Q.InboundRouteFailing(ctx, store.InboundRouteFailingParams{ID: r.ID, StatusDetail: detail, Now: s.now()}); e != nil {
		s.Log.WarnContext(ctx, "inbound route failing", "route", r.ID, "err", e)
		return
	}
	s.notifyFailing(ctx, r, key, args)
}

func hostOf(r store.InboundRoute) string {
	var m mailboxSettings
	_ = json.Unmarshal(r.MailboxSettings, &m)
	return m.Host
}

// notifyFailing tells the members who manage domains that receiving
// stopped, once until the route receives again.
func (s *Service) notifyFailing(ctx context.Context, r store.InboundRoute, key string, args i18n.Args) {
	if _, err := s.Q.InboundRouteMarkFailingNotified(ctx, store.InboundRouteMarkFailingNotifiedParams{ID: r.ID, At: s.now()}); err != nil {
		return // told already, or the route is gone
	}
	ws, err := s.Q.GetWorkspace(ctx, r.WorkspaceID)
	if err != nil {
		return
	}
	n := notify.Notification{Kind: notify.KindInboundFailing, Severity: "bad", TitleText: msgs.M("notice.failingTitle").Keep(nil),
		BodyText: msgs.M(key).Keep(args), Link: "/w/" + ws.Slug + "/settings/senders", Data: map[string]any{"route": r.ID.String()}}
	if err := notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.DomainsManage, n); err != nil {
		s.Log.WarnContext(ctx, "notify inbound failing", "route", r.ID, "err", err)
	}
}

// SweepArgs requeues stored arrivals whose job was lost and purges old ones.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "inbound.sweep" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// SweepWorker runs SweepArgs.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	return w.S.Sweep(ctx)
}

// Sweep requeues verified arrivals still waiting after two minutes, and
// deletes arrivals after 7 days and received messages after a year.
func (s *Service) Sweep(ctx context.Context) error {
	now := s.now()
	stale, err := s.Q.InboundRawStale(ctx, now.Add(-staleAfter))
	if err != nil {
		return err
	}
	if c := s.jobs(); c != nil && len(stale) > 0 {
		params := make([]river.InsertManyParams, 0, len(stale))
		for _, id := range stale {
			params = append(params, river.InsertManyParams{Args: ProcessArgs{RawID: id}})
		}
		if _, err := c.InsertMany(ctx, params); err != nil {
			return err
		}
	}
	if _, err := s.Q.InboundRawPurge(ctx, now.Add(-rawKeep)); err != nil {
		return err
	}
	_, err = s.Q.InboundMessagesPurge(ctx, now.Add(-messagesKeep))
	return err
}

func (s *Service) jobs() *jobs.Client {
	if s.Jobs == nil {
		return nil
	}
	return s.Jobs()
}

// HealthArgs checks provider routes' MX records.
type HealthArgs struct{}

// Kind implements river.JobArgs.
func (HealthArgs) Kind() string { return "inbound.health" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (HealthArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 14 * time.Minute}}
}

// HealthWorker runs HealthArgs.
type HealthWorker struct {
	river.WorkerDefaults[HealthArgs]
	S *Service
}

// Work implements river.Worker.
func (w *HealthWorker) Work(ctx context.Context, _ *river.Job[HealthArgs]) error {
	return w.S.Health(ctx)
}

// Health looks up each receiving provider route's MX records: a route
// whose MX no longer names its provider's host becomes failing, and the
// members who manage domains are told once. It receives again with the
// next message.
func (s *Service) Health(ctx context.Context) error {
	routes, err := s.Q.InboundRoutesProvider(ctx)
	if err != nil {
		return err
	}
	for _, r := range routes {
		if r.Status != "receiving" {
			continue
		}
		if err := s.checkRoute(ctx, r); err != nil {
			s.Log.WarnContext(ctx, "inbound health", "route", r.ID, "err", err)
		}
	}
	return nil
}

func (s *Service) checkRoute(ctx context.Context, r store.InboundRoute) error {
	since := s.now()
	rc, p, err := s.receiver(ctx, r)
	if err != nil || manage.IsDemo(p) {
		return nil // the provider is gone (the card says so) or answers in process
	}
	ar, err := s.adapterRoute(r)
	if err != nil {
		return err
	}
	want := rc.InboundSetup(ar).MX
	if len(want) == 0 {
		return nil // the provider shows its own MX per domain (Resend): nothing to compare
	}
	if s.DNS == nil {
		return nil
	}
	lctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	mxs, err := s.DNS.MX(lctx, strings.ToLower(r.Domain))
	if err != nil && !errors.Is(err, dnscheck.ErrNotFound) {
		return nil // DNS can't tell now
	}
	hosts := make([]string, 0, len(mxs))
	for _, m := range mxs {
		hosts = append(hosts, strings.ToLower(strings.TrimSuffix(m.Host, ".")))
	}
	for _, w := range want {
		if hasMX(hosts, w.Value) {
			return nil
		}
	}
	host := mxHost(want[0].Value)
	args := i18n.Args{"domain": r.Domain, "host": host}
	detail := keepDetail(msgs.M("status.mxMoved").Keep(args))
	if err := s.Q.InboundRouteFailingFor(ctx, store.InboundRouteFailingForParams{ID: r.ID, StatusDetail: detail, Now: s.now(), Since: since}); err != nil {
		return err
	}
	cur, err := s.Q.InboundRouteByID(ctx, r.ID)
	if err != nil || cur.Status != "failing" {
		return err
	}
	s.notifyFailing(ctx, cur, "status.mxMoved", args)
	return nil
}

// mxHost is the host of an MX record's value ("10 inbound.postmarkapp.com").
func mxHost(v string) string {
	f := strings.Fields(v)
	if len(f) == 0 {
		return v
	}
	return strings.TrimSuffix(f[len(f)-1], ".")
}

// NoticeArgs tells members about new replies.
type NoticeArgs struct{}

// Kind implements river.JobArgs.
func (NoticeArgs) Kind() string { return "inbound.notice" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (NoticeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// NoticeWorker runs NoticeArgs.
type NoticeWorker struct {
	river.WorkerDefaults[NoticeArgs]
	S *Service
}

// Work implements river.Worker.
func (w *NoticeWorker) Work(ctx context.Context, _ *river.Job[NoticeArgs]) error {
	return w.S.Notice(ctx)
}

// Notice tells the members who can view replies how many people replied
// since they were last told, at most every 15 minutes per workspace. Only
// people's replies count (not auto-replies, spam or other mail).
func (s *Service) Notice(ctx context.Context) error {
	now := s.now()
	routes, err := s.Q.InboundRoutesWithFreshReplies(ctx, now.Add(-noticeEvery))
	if err != nil {
		return err
	}
	for _, r := range routes {
		if err := s.noticeRoute(ctx, r, now); err != nil {
			s.Log.WarnContext(ctx, "notify replies", "route", r.ID, "err", err)
		}
	}
	return nil
}

func (s *Service) noticeRoute(ctx context.Context, r store.InboundRoutesWithFreshRepliesRow, now time.Time) error {
	since := r.CreatedAt
	if r.NotifiedAt != nil {
		since = *r.NotifiedAt
	}
	n, err := s.Q.InboundFreshReplies(ctx, store.InboundFreshRepliesParams{WorkspaceID: r.WorkspaceID, Since: since})
	if err != nil || n == 0 {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, r.WorkspaceID)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.InboundRouteClaimNotice(ctx, store.InboundRouteClaimNoticeParams{ID: r.ID, Now: now, Prev: r.NotifiedAt}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // another process told them
			}
			return err
		}
		args := i18n.Args{"count": n}
		return notify.ToMembersWith(ctx, tx, ws.ID, authz.RepliesView, notify.Notification{Kind: notify.KindReplyReceived, Severity: "info",
			TitleText: msgs.M("notice.repliesTitle").Keep(args), SummaryText: msgs.M("notice.repliesSummary").Keep(args),
			BodyText: msgs.M("notice.repliesBody").Keep(args), Link: "/w/" + ws.Slug + "/audience/replies",
			Data: map[string]any{"count": n}})
	})
}
