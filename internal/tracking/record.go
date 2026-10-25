package tracking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// Opens and clicks are recorded in batches (group commit), each in one
// transaction that takes its locks in the one order every counter writer
// uses: the batch's messages in id order, then the campaign counters, the
// hourly engagement and the links (internal/counters), then the contacts
// in id order. A batch that fails is retried; one that keeps failing is
// kept in tracking_retry and applied again by the tracking.retry job, so a
// pixel or a redirect never loses its event.

// RecordEngagement implements sending.Recorder: it classifies and records a
// batch of opens and clicks in one transaction (retried on deadlocks and
// serialization failures).
func (s *Service) RecordEngagement(ctx context.Context, evs []sending.Engagement) error {
	if len(evs) == 0 {
		return nil
	}
	// The batch in message order: every writer locks messages in id order.
	sorted := append([]sending.Engagement(nil), evs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if c := bytes.Compare(sorted[i].MessageID[:], sorted[j].MessageID[:]); c != 0 {
			return c < 0
		}
		return sorted[i].At.Before(sorted[j].At)
	})
	return sending.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		b := &batch{}
		for _, ev := range sorted {
			if err := s.apply(ctx, tx, q, b, ev); err != nil {
				return err
			}
		}
		if err := b.ledger.Flush(ctx, q); err != nil {
			return err
		}
		return b.touches.flush(ctx, q)
	})
}

// batch is what a batch of opens and clicks changes beyond its messages.
type batch struct {
	ledger  counters.Ledger
	touches touches
}

// touches are a batch's opens and clicks per contact, applied last in
// contact order.
type touches map[uuid.UUID]*touch

type touch struct {
	opened, human, clicked, activity time.Time
	machine                          bool
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func (t *touches) add(contact uuid.UUID, at time.Time, open, human, click bool) {
	if *t == nil {
		*t = touches{}
	}
	c := (*t)[contact]
	if c == nil {
		c = &touch{}
		(*t)[contact] = c
	}
	if open && !at.Before(c.opened) {
		c.opened, c.machine = at, !human
	}
	if human {
		c.human = later(c.human, at)
	}
	if click {
		c.clicked = later(c.clicked, at)
	}
	c.activity = later(c.activity, at)
}

// none stands for "no time" in TrackingTouchContacts.
var none = time.Unix(0, 0).UTC()

func orNone(t time.Time) time.Time {
	if t.IsZero() {
		return none
	}
	return t
}

func (t touches) flush(ctx context.Context, q *store.Queries) error {
	if len(t) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(t))
	for id := range t {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 })
	p := store.TrackingTouchContactsParams{}
	for _, id := range ids {
		c := t[id]
		p.Ids = append(p.Ids, id)
		p.Opened = append(p.Opened, orNone(c.opened))
		p.Human = append(p.Human, orNone(c.human))
		p.Clicked = append(p.Clicked, orNone(c.clicked))
		p.Machine = append(p.Machine, c.machine)
		p.Activity = append(p.Activity, c.activity)
	}
	return q.TrackingTouchContacts(ctx, p)
}

// apply records one open or click. An open counts at once; a click that
// looks like a person's waits to settle (settle.go), one that doesn't is
// filed with the bot clicks.
func (s *Service) apply(ctx context.Context, tx pgx.Tx, q *store.Queries, b *batch, ev sending.Engagement) error {
	m, err := q.TrackingMessage(ctx, ev.MessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if m.Kind == placementKind {
		return nil // placement test copies: opens and clicks are never counted
	}
	// The message first, before its event refers to it (lock order).
	if err := q.CountersLockMessage(ctx, m.ID); err != nil {
		return err
	}
	ip, _ := netip.ParseAddr(ev.IP)
	var ipPtr *netip.Addr
	if ip.IsValid() {
		ipPtr = &ip
	}
	sig := s.signal(ctx, m, ev.UserAgent, ipPtr, ev.Method, ev.At)
	var v Verdict
	link := ""
	switch {
	case ev.Click && ev.Link == HoneypotLink:
		v = ClassifyHoneypot()
	case ev.Click:
		v = ClassifyClick(sig, false)
		if urls, err := s.linksOf(ctx, q, m); err == nil {
			link = urls[ev.Link]
		}
	default:
		v = ClassifyOpen(sig)
	}
	typ, class := "open", v.Class
	var linkPos *int32
	if ev.Click {
		typ = "click"
		p := int32(ev.Link)
		linkPos = &p
		if !v.Machine {
			class = classPending // counted once it settles
		}
	}
	diag := v.Client
	if v.Machine {
		diag = v.Reason
	}
	label, _ := q.TrackingSourceLabel(ctx, m.ID)
	mid := m.ID
	eventID := uuid.Must(uuid.NewV7())
	if err := q.TrackingInsertEvent(ctx, store.TrackingInsertEventParams{
		ID: eventID, WorkspaceID: m.WorkspaceID, MessageID: &mid, CampaignID: m.CampaignID, ContactID: m.ContactID, ProviderID: m.ProviderID,
		Type: typ, OccurredAt: ev.At, ReceivedAt: ev.At, MailboxProvider: m.MailboxProvider, OriginalType: "ReputeCast tracking", Diagnostic: clip(diag, 1000),
		Machine: v.Machine, Url: clip(link, 2048), UserAgent: clip(ev.UserAgent, 500), Ip: ipPtr, Recipient: m.Email, Source: label, Class: class, Link: linkPos,
	}); err != nil {
		return err
	}
	if ev.Click {
		switch {
		case ev.Link == HoneypotLink:
			// The hidden link isn't one of the email's: nothing to count on
			// the map; the clicks around it settle as the scanner's.
		case v.Machine:
			// Bot clicks are logged and shown apart (the click map's Bot
			// clicks), never counted as clicks.
			if m.CampaignID != nil {
				b.ledger.Click(*m.CampaignID, ev.Link, true, false)
			}
		default:
			return q.TrackingPendingClick(ctx, store.TrackingPendingClickParams{EventID: eventID, MessageID: m.ID, Link: int32(ev.Link), Url: clip(link, 2048),
				ClickedAt: ev.At, SettleAt: ev.At.Add(settleAfter)})
		}
		return nil
	}
	mv, err := q.CountersMoveEngagement(ctx, store.CountersMoveEngagementParams{ID: m.ID, At: ev.At, Click: false, Class: class})
	if err != nil {
		return err
	}
	moved := counters.FromEngagement(mv)
	b.ledger.Add(moved)
	firstOpen := moved.Before.OpenedAt == nil
	if m.ContactID != nil {
		b.touches.add(*m.ContactID, ev.At, true, v.Class == ClassHuman, false)
	}
	if m.Stream == "transactional" && s.Transactional != nil && firstOpen {
		if err := s.Transactional.Engagement(ctx, tx, m.ID, false, ev.At); err != nil {
			return err
		}
	}
	if m.Stream == "automation" && s.Automations != nil && firstOpen {
		if err := s.Automations.RecordMessageEvent(ctx, q, automations.MessageEvent{MessageID: m.ID, Type: "opened", Machine: v.Machine, At: ev.At}); err != nil {
			return err
		}
	}
	return nil
}

// classPending marks a click event waiting to settle.
const classPending = "pending"

// Retries of opens and clicks that couldn't be recorded.
const (
	// retryMaxAttempts: an event that failed this often stays in
	// tracking_retry (with its error) until it is purged.
	retryMaxAttempts = 20
	// retryBacklog caps the events kept in memory while the database can't
	// even take them for a later retry.
	retryBacklog = 50_000
)

// retryAfter is the backoff before the nth retry: a minute, then doubling
// up to six hours.
func retryAfter(attempt int) time.Duration {
	d := time.Minute << min(max(attempt-1, 0), 9)
	return min(d, 6*time.Hour)
}

// park keeps a batch that failed for the tracking.retry job; when even
// that fails the batch stays in memory and is tried again by the writer.
func (s *Service) park(evs []sending.Engagement, cause error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ids := make([]uuid.UUID, 0, len(evs))
	raw := make([]json.RawMessage, 0, len(evs))
	for _, ev := range evs {
		b, _ := json.Marshal(ev)
		ids = append(ids, ev.MessageID)
		raw = append(raw, b)
	}
	err := s.Q.TrackingRetryPark(ctx, store.TrackingRetryParkParams{MessageIds: ids, Events: raw, NextAt: s.now().Add(retryAfter(1)), Error: clip(cause.Error(), 500)})
	if err == nil {
		s.log().Warn("opens and clicks kept for a retry", "events", len(evs), "err", cause)
		return
	}
	s.log().Error("opens and clicks not recorded yet; kept in memory", "events", len(evs), "err", cause, "park_err", err)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backlog = append(s.backlog, evs...)
	if over := len(s.backlog) - retryBacklog; over > 0 {
		s.log().Error("opens and clicks dropped: the retry backlog is full", "events", over)
		s.backlog = append([]sending.Engagement(nil), s.backlog[over:]...)
	}
}

// retryBacklog tries the in-memory backlog again.
func (s *Service) retryMemory() {
	s.mu.Lock()
	evs := s.backlog
	s.backlog = nil
	s.mu.Unlock()
	if len(evs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for len(evs) > 0 {
		n := min(len(evs), batchMax)
		if err := s.RecordEngagement(ctx, evs[:n]); err != nil {
			s.park(evs, err)
			return
		}
		evs = evs[n:]
	}
}

// RetryResult says what a retry run did.
type RetryResult struct {
	Recorded int
	Failed   int
}

// RetryParked applies the kept opens and clicks that are due, a batch at a
// time; a batch that fails again is retried event by event so one bad
// event doesn't hold the others back.
func (s *Service) RetryParked(ctx context.Context) (RetryResult, error) {
	var out RetryResult
	for range 20 {
		now := s.now()
		rows, err := s.Q.TrackingRetryClaim(ctx, store.TrackingRetryClaimParams{Now: now, NextAt: now.Add(retryAfter(2)), MaxAttempts: retryMaxAttempts, Lim: batchMax})
		if err != nil {
			return out, err
		}
		if len(rows) == 0 {
			break
		}
		evs := make([]sending.Engagement, 0, len(rows))
		ids := make([]int64, 0, len(rows))
		for _, r := range rows {
			var ev sending.Engagement
			if err := json.Unmarshal(r.Event, &ev); err != nil || ev.MessageID == uuid.Nil {
				ev = sending.Engagement{} // unreadable: dropped below
			}
			evs = append(evs, ev)
			ids = append(ids, r.ID)
		}
		if ok := nonEmpty(evs); s.RecordEngagement(ctx, ok) == nil {
			out.Recorded += len(ok)
			if err := s.Q.TrackingRetryDone(ctx, ids); err != nil {
				return out, err
			}
			continue
		}
		for i, ev := range evs {
			if ev.MessageID != uuid.Nil {
				if err := s.RecordEngagement(ctx, []sending.Engagement{ev}); err != nil {
					out.Failed++
					if err := s.Q.TrackingRetryFailed(ctx, store.TrackingRetryFailedParams{Ids: []int64{ids[i]}, Error: clip(err.Error(), 500)}); err != nil {
						return out, err
					}
					continue
				}
				out.Recorded++
			}
			if err := s.Q.TrackingRetryDone(ctx, []int64{ids[i]}); err != nil {
				return out, err
			}
		}
	}
	if _, err := s.Q.TrackingRetryPurge(ctx, s.now().AddDate(0, 0, -30)); err != nil {
		return out, err
	}
	return out, nil
}

func nonEmpty(evs []sending.Engagement) []sending.Engagement {
	out := evs[:0:0]
	for _, ev := range evs {
		if ev.MessageID != uuid.Nil {
			out = append(out, ev)
		}
	}
	return out
}

// RetryArgs applies the opens and clicks kept for a retry (periodic).
type RetryArgs struct{}

// Kind implements river.JobArgs.
func (RetryArgs) Kind() string { return "tracking.retry" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (RetryArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// RetryWorker runs RetryArgs.
type RetryWorker struct {
	river.WorkerDefaults[RetryArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RetryWorker) Work(ctx context.Context, _ *river.Job[RetryArgs]) error {
	_, err := w.S.RetryParked(ctx)
	return err
}
