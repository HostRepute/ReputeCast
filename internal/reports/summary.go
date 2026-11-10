package reports

import (
	"container/list"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/membermail"
	"github.com/hostrepute/reputecast/internal/notify"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sending summary by email (M57 G-14, Notifications board): a workspace
// member who can view reports asks for a weekly or monthly email of the
// campaigns started in the period and, as far as their role shows them,
// the audience's change and the sender health. An hourly tick claims each
// member's period once, in the transaction that queues its email; the
// email job checks again that the member may still read reports, then
// sends through the System stream. Only members get it, never other
// addresses.

// ReportSummarySetting is the signed-in member's sending summary of the
// workspace.
type ReportSummarySetting struct {
	Frequency string `json:"frequency" enum:"off,weekly,monthly" doc:"weekly: Mondays at 09:00; monthly: on the 1st at 09:00, in your time zone"`
	// NextAt is when the next summary goes out.
	NextAt *time.Time `json:"nextAt,omitempty" readOnly:"true" doc:"When the next summary goes out (ignored in requests)"`
}

type reportSummaryInput struct {
	api.WorkspaceInput
	Body ReportSummarySetting
}

// summarySendHour is the local hour summaries go out: Mondays (weekly) and
// the 1st (monthly) at 09:00.
const summarySendHour = 9

// summaryTop is how many of the most opened campaigns the email lists.
const summaryTop = 5

// Inserter queues jobs in a transaction (River's client).
type Inserter interface {
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

func registerSummary(h huma.API, s *Service) {
	tags := []string{"Reports"}
	path := "/workspaces/{workspace}/report-summary"
	// A person's own email: API keys and agents have no inbox here.
	access := api.WorkspaceHuman(authz.ReportsView)
	api.Get(h, api.Op{ID: "get-report-summary", Path: path, Summary: "Your sending summary by email: off, weekly or monthly",
		Tags: tags, Access: access},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[ReportSummarySetting], error) {
			out, err := s.Summary(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-report-summary", Path: path, Summary: "Turn your sending summary by email on (weekly or monthly) or off",
		Description: "Campaigns, audience and sender health of the last week or month, as far as your role shows them, emailed to you through the System stream. Only people, not API keys.",
		Tags:        tags, Access: access},
		func(ctx context.Context, in *reportSummaryInput) (*api.Body[ReportSummarySetting], error) {
			out, err := s.SetSummary(ctx, api.WorkspaceFrom(ctx), in.Body.Frequency)
			return api.OK(out), err
		})
}

// summaryPeriod is one period of a summary in a member's zone: [From, To),
// sent from SendAt.
type summaryPeriod struct{ From, To, SendAt time.Time }

// lastSummaryPeriod is the latest period whose summary is due at now in
// loc: weekly, the previous Monday 00:00 to this Monday 00:00, due from
// Monday 09:00; monthly, the previous month's 1st to this month's 1st, due
// from 09:00 on the 1st. Bounds are local calendar ones (time.Date
// normalises day and month overflow), so a week with a daylight saving
// change lasts 167 or 169 hours.
func lastSummaryPeriod(freq string, now time.Time, loc *time.Location) summaryPeriod {
	t := now.In(loc)
	y, m, d := t.Date()
	if freq == "monthly" {
		if now.Before(time.Date(y, m, 1, summarySendHour, 0, 0, 0, loc)) {
			m--
		}
		return summaryPeriod{From: time.Date(y, m-1, 1, 0, 0, 0, 0, loc), To: time.Date(y, m, 1, 0, 0, 0, 0, loc),
			SendAt: time.Date(y, m, 1, summarySendHour, 0, 0, 0, loc)}
	}
	d -= (int(t.Weekday()) + 6) % 7 // back to Monday
	if now.Before(time.Date(y, m, d, summarySendHour, 0, 0, 0, loc)) {
		d -= 7
	}
	return summaryPeriod{From: time.Date(y, m, d-7, 0, 0, 0, 0, loc), To: time.Date(y, m, d, 0, 0, 0, 0, loc),
		SendAt: time.Date(y, m, d, summarySendHour, 0, 0, 0, loc)}
}

// nextSummaryAt is when the summary after the last due one goes out.
func nextSummaryAt(freq string, now time.Time, loc *time.Location) time.Time {
	p := lastSummaryPeriod(freq, now, loc)
	y, m, d := p.To.Date()
	if freq == "monthly" {
		return time.Date(y, m+1, 1, summarySendHour, 0, 0, 0, loc)
	}
	return time.Date(y, m, d+7, summarySendHour, 0, 0, 0, loc)
}

// summaryZone is the zone a member's periods are counted in: theirs, else
// the workspace's.
func summaryZone(userTZ, workspaceTZ string) *time.Location {
	if userTZ != "" {
		if loc, err := time.LoadLocation(userTZ); err == nil {
			return loc
		}
	}
	return provstate.Location(workspaceTZ)
}

// localDate is a period's start as the date stored in last_period_start.
func localDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func summarySetting(freq string, now time.Time, loc *time.Location) ReportSummarySetting {
	out := ReportSummarySetting{Frequency: freq}
	if freq != "off" {
		at := nextSummaryAt(freq, now, loc).UTC()
		out.NextAt = &at
	}
	return out
}

// humansOnly is the access layer's refusal (api.WorkspaceHuman), which the
// operations already get: the services read the signed-in person.
func humansOnly() error {
	return api.Err(http.StatusForbidden, api.CodeHumansOnly, "This action can only be taken by a signed-in person, not an API key or agent.")
}

// Summary reads the signed-in member's sending summary.
func (s *Service) Summary(ctx context.Context, ws *store.Workspace) (ReportSummarySetting, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return ReportSummarySetting{}, humansOnly()
	}
	row, err := s.Q.ReportSummaryGet(ctx, store.ReportSummaryGetParams{UserID: u.ID, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportSummarySetting{Frequency: "off"}, nil
	}
	if err != nil {
		return ReportSummarySetting{}, err
	}
	return summarySetting(row.Frequency, s.now(), summaryZone(u.Timezone, ws.Timezone)), nil
}

// SetSummary turns the member's summary on (weekly or monthly) or off. The
// period the member is in counts as already behind them: the first email
// is the next one, not one about the week they turned it on in.
func (s *Service) SetSummary(ctx context.Context, ws *store.Workspace, freq string) (ReportSummarySetting, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return ReportSummarySetting{}, humansOnly()
	}
	now, loc := s.now(), summaryZone(u.Timezone, ws.Timezone)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		key := store.ReportSummaryGetParams{UserID: u.ID, WorkspaceID: ws.ID}
		before := "off"
		cur, err := q.ReportSummaryGet(ctx, key)
		switch {
		case err == nil:
			before = cur.Frequency
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		if before == freq {
			return nil
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "report_summary.set", TargetType: "user", TargetID: u.ID.String(), TargetName: u.Name,
			Before: map[string]any{"frequency": before}, After: map[string]any{"frequency": freq}}
		if freq == "off" {
			if _, err := q.ReportSummaryRemove(ctx, store.ReportSummaryRemoveParams(key)); err != nil {
				return err
			}
			ev.Action, ev.After = "report_summary.remove", nil
		} else {
			last := localDate(lastSummaryPeriod(freq, now, loc).From)
			if err := q.ReportSummarySet(ctx, store.ReportSummarySetParams{UserID: u.ID, WorkspaceID: ws.ID, Frequency: freq, LastPeriodStart: &last}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		return ReportSummarySetting{}, err
	}
	return summarySetting(freq, now, loc), nil
}

// QueueSummaries claims every member's period that is due and queues its
// email in the same transaction, so overlapping ticks claim each period
// once. It returns how many emails it queued.
func (s *Service) QueueSummaries(ctx context.Context) (int, error) {
	rows, err := s.Q.ReportSummariesList(ctx)
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	if s.Jobs == nil {
		return 0, errors.New("report summaries: no job client")
	}
	now := s.now()
	queued := 0
	for _, r := range rows {
		loc := summaryZone(r.UserTimezone, r.WorkspaceTimezone)
		p := lastSummaryPeriod(r.Frequency, now, loc)
		from := localDate(p.From)
		if r.LastPeriodStart != nil && !r.LastPeriodStart.Before(from) {
			continue
		}
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			_, err := store.New(tx).ReportSummaryClaim(ctx, store.ReportSummaryClaimParams{FromDate: from, UserID: r.UserID, WorkspaceID: r.WorkspaceID, Frequency: r.Frequency})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil // another tick claimed it, or the member changed it since
			}
			if err != nil {
				return err
			}
			args := SummarySendArgs{UserID: r.UserID, WorkspaceID: r.WorkspaceID, Frequency: r.Frequency, From: p.From.UTC(), To: p.To.UTC()}
			if _, err := s.Jobs.InsertTx(ctx, tx, args, nil); err != nil {
				return err
			}
			queued++
			return nil
		})
		if err != nil {
			return queued, err
		}
	}
	return queued, nil
}

// SummaryArgs claims the sending summaries that are due (periodic).
type SummaryArgs struct{}

// Kind implements river.JobArgs.
func (SummaryArgs) Kind() string { return "reports.summary" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SummaryArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSystem, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

// SummaryWorker runs SummaryArgs.
type SummaryWorker struct {
	river.WorkerDefaults[SummaryArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SummaryWorker) Work(ctx context.Context, _ *river.Job[SummaryArgs]) error {
	_, err := w.S.QueueSummaries(ctx)
	return err
}

// SummarySendArgs emails one member the summary of one period ([From, To),
// in UTC).
type SummarySendArgs struct {
	UserID      uuid.UUID `json:"user_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Frequency   string    `json:"frequency"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
}

// Kind implements river.JobArgs.
func (SummarySendArgs) Kind() string { return "reports.summary_send" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SummarySendArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSystem, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// SummarySendWorker runs SummarySendArgs. A provider that accepted the
// email just before a crash may see it once more: delivery is at least
// once.
type SummarySendWorker struct {
	river.WorkerDefaults[SummarySendArgs]
	S   *Service
	Log *slog.Logger
	// memo keeps each period's numbers for the members who share it.
	memo periodMemo
}

// Work implements river.Worker.
func (w *SummarySendWorker) Work(ctx context.Context, job *river.Job[SummarySendArgs]) error {
	s, a := w.S, job.Args
	key := store.ReportSummaryGetParams{UserID: a.UserID, WorkspaceID: a.WorkspaceID}
	active, err := s.Q.ReportSummaryMember(ctx, store.ReportSummaryMemberParams{WorkspaceID: a.WorkspaceID, UserID: a.UserID})
	if err != nil {
		return err
	}
	if !active {
		// Former members get nothing more from the workspace.
		_, err := s.Q.ReportSummaryRemove(ctx, store.ReportSummaryRemoveParams(key))
		return err
	}
	// Turned off or changed since the period was claimed.
	if cur, err := s.Q.ReportSummaryGet(ctx, key); errors.Is(err, pgx.ErrNoRows) || err == nil && cur.Frequency != a.Frequency {
		return nil
	} else if err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, a.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // deleted since
	}
	if err != nil {
		return err
	}
	can := func(perm authz.Permission) (bool, error) {
		ids, err := notify.MembersWith(ctx, s.Pool, ws.ID, perm)
		return slices.Contains(ids, a.UserID), err
	}
	// A member whose role no longer reads reports keeps the setting but
	// gets nothing until it does again.
	if ok, err := can(authz.ReportsView); err != nil || !ok {
		return err
	}
	e := w.memo.entry(periodKey{workspace: ws.ID, frequency: a.Frequency, from: a.From.UnixNano(), to: a.To.UnixNano()}, s.now())
	n, err := e.numbers(ctx, s, &ws, a.From, a.To)
	if err != nil {
		return err
	}
	if n.campaigns.Started == 0 && n.audience.Joined == n.audience.Departed {
		return nil // nothing happened
	}
	u, err := s.Q.GetUser(ctx, a.UserID)
	if err != nil {
		return err
	}
	loc := summaryZone(u.Timezone, ws.Timezone)
	app := s.appURL(ws.Slug)
	d := membermail.ReportSummaryEmail{Workspace: ws.Name, Frequency: a.Frequency, From: a.From.In(loc), To: a.To.In(loc),
		Campaigns: n.campaigns, Top: n.top,
		Links: membermail.SummaryLinks{Campaigns: app + "/campaigns", Preferences: app + "/notifications#summary"}}
	if ok, err := can(authz.SubscribersView); err != nil {
		return err
	} else if ok {
		aud := n.audience
		d.Audience = &aud
	}
	if ok, err := can(authz.DeliverabilityView); err != nil {
		return err
	} else if ok {
		h, err := e.health(ctx, s, &ws)
		if err != nil && w.Log != nil {
			// A summary without its health line beats no summary.
			w.Log.WarnContext(ctx, "sending summary: sender health unavailable", "workspace", ws.ID, "err", err)
		}
		if h != nil {
			d.Health, d.Links.Deliverability = h, app+"/deliverability"
		}
	}
	msg := membermail.ReportSummary(applang.ForUser(ctx, s.Q, u), d)
	msg.WorkspaceID, msg.To, msg.ToName = &ws.ID, u.Email, u.Name
	var m mailer.Sender = mailer.Unavailable{}
	if s.Mail != nil {
		m = s.Mail()
	}
	if _, err := m.SendSystem(ctx, msg); err != nil && !errors.Is(err, mailer.ErrNoSystemProvider) {
		return err
	}
	// Without a System-stream provider the summary is skipped, as
	// notification emails are.
	return nil
}

// appURL is the workspace's screens on the instance's public URL.
func (s *Service) appURL(slug string) string {
	return strings.TrimRight(s.URLs.Base(), "/") + "/w/" + slug
}

// periodKey names one period's numbers: they depend only on these.
type periodKey struct {
	workspace uuid.UUID
	frequency string
	from, to  int64
}

// periodNumbers are the numbers of one workspace's period, the same for
// every member who gets its summary; the sender health is read only for
// the first member who may see it.
type periodNumbers struct {
	campaigns membermail.SummaryCampaigns
	top       []membermail.SummaryCampaign
	audience  membermail.SummaryAudience
}

type periodEntry struct {
	key  periodKey
	made time.Time

	mu         sync.Mutex
	nums       *periodNumbers
	healthRead bool
	healthOf   *membermail.SummaryHealth
}

// periodMemo is a small least-recently-used cache of period numbers: the
// members of a workspace who share a time zone get one computation.
type periodMemo struct {
	mu    sync.Mutex
	items map[periodKey]*list.Element
	order *list.List // most recently used first
}

const (
	periodMemoSize = 64
	periodMemoTTL  = 10 * time.Minute
)

func (m *periodMemo) entry(k periodKey, now time.Time) *periodEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.items == nil {
		m.items, m.order = map[periodKey]*list.Element{}, list.New()
	}
	if el, ok := m.items[k]; ok {
		e := el.Value.(*periodEntry)
		if now.Sub(e.made) < periodMemoTTL {
			m.order.MoveToFront(el)
			return e
		}
		m.order.Remove(el)
		delete(m.items, k)
	}
	e := &periodEntry{key: k, made: now}
	m.items[k] = m.order.PushFront(e)
	for m.order.Len() > periodMemoSize {
		last := m.order.Back()
		m.order.Remove(last)
		delete(m.items, last.Value.(*periodEntry).key)
	}
	return e
}

func (e *periodEntry) numbers(ctx context.Context, s *Service, ws *store.Workspace, from, to time.Time) (*periodNumbers, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.nums == nil {
		n, err := s.periodNumbers(ctx, ws, from, to)
		if err != nil {
			return nil, err
		}
		e.nums = n
	}
	return e.nums, nil
}

func (e *periodEntry) health(ctx context.Context, s *Service, ws *store.Workspace) (*membermail.SummaryHealth, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.healthRead || s.Health == nil {
		return e.healthOf, nil
	}
	score, label, ok, err := s.Health(ctx, ws)
	if err != nil {
		return nil, err
	}
	e.healthRead = true
	if ok {
		e.healthOf = &membermail.SummaryHealth{Score: score, Label: label}
	}
	return e.healthOf, nil
}

// periodNumbers counts a period: the totals of the email campaigns started
// in it from campaign_counters, the most opened of them, and the
// audience's change.
func (s *Service) periodNumbers(ctx context.Context, ws *store.Workspace, from, to time.Time) (*periodNumbers, error) {
	rows, err := s.Q.ReportSummaryCampaigns(ctx, store.ReportSummaryCampaignsParams{WorkspaceID: ws.ID, PeriodFrom: from, PeriodTo: to})
	if err != nil {
		return nil, err
	}
	out := &periodNumbers{top: []membermail.SummaryCampaign{}}
	now, app := s.now(), s.appURL(ws.Slug)
	byID := map[uuid.UUID]int{}
	var all []membermail.SummaryCampaign
	var reportedDelivered int64
	c := &out.campaigns
	for _, r := range rows {
		i, ok := byID[r.ID]
		if !ok {
			i = len(all)
			byID[r.ID] = i
			all = append(all, membermail.SummaryCampaign{Name: r.Name, Link: app + "/campaigns/" + r.ID.String() + "/report"})
			c.Started++
			// Opens and clicks keep counting for 72 hours after the send.
			if r.CompletedAt == nil || now.Before(r.CompletedAt.Add(FinalAfter)) {
				c.Collecting++
			}
		}
		all[i].Sent += r.Sent
		all[i].HumanOpens += r.OpenedHuman
		all[i].Clicks += r.Clicked
		c.Sent += r.Sent
		c.Delivered += r.Delivered
		c.HumanOpens += r.OpenedHuman
		c.Clicks += r.Clicked
		c.Unsubscribes += r.Unsubscribed
		c.Bounces += r.Bounced
		// Complaints where they are reported (review F-2): Gmail and
		// iCloud never tell senders.
		if r.MailboxProvider != "" && mailbox.ReportsComplaints(r.MailboxProvider) {
			reportedDelivered += r.Delivered
			c.Complaints += r.Complained
		}
	}
	if reportedDelivered > 0 {
		c.Reported = true
		c.ComplaintRate = float64(c.Complaints) / float64(reportedDelivered) * 100
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].HumanOpens > all[j].HumanOpens })
	out.top = append(out.top, all[:min(len(all), summaryTop)]...)
	a, err := s.Q.ReportSummaryAudience(ctx, store.ReportSummaryAudienceParams{WorkspaceID: ws.ID, PeriodFrom: from, PeriodTo: to})
	if err != nil {
		return nil, err
	}
	out.audience = membermail.SummaryAudience{Joined: a.Joined, Departed: a.Departed, Subscribers: a.Subscribers}
	return out, nil
}
