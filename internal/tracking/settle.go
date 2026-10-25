package tracking

import (
	"context"
	"errors"
	"net/netip"
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

// Clicks settle before they count (review AN-2). A scanner that follows
// links clicks every link of an email within seconds of its arrival, and
// follows the hidden link no person can see; a click alone can look like a
// person's. So a click that looks like a person's is recorded as pending,
// and settles after settleAfter: the clicks around it decide. Settled as a
// person's, it counts once (the human open, the unique click on its link,
// the automation's "clicked", Earn's credit, the transactional webhook);
// settled as a bot's, it is filed with the bot clicks the report shows
// apart. Nothing has to be taken back.
const (
	// settleAfter is how long a click waits before it counts.
	settleAfter = time.Minute
	// burstWindow and burstLinks: this many different links clicked within
	// the window around a click is a scanner following every link.
	burstWindow = 5 * time.Second
	burstLinks  = 3
	// honeypotWindow: clicks this close to the hidden link being followed
	// are the same scanner's.
	honeypotWindow = time.Minute
	// HoneypotLink is the position the hidden link's clicks are stored with.
	HoneypotLink = -1
)

// settleBatch is how many pending clicks one transaction settles.
const settleBatch = 200

// SettleEngagement settles the clicks that are due at now and returns when
// the next one is due (nil when none waits). The demo transport's pump and
// the tests call it with their own clock; the tracking.settle job and the
// writer call it with the real one.
func (s *Service) SettleEngagement(ctx context.Context, now time.Time) (*time.Time, error) {
	for range 50 {
		n, err := s.settle(ctx, now)
		if err != nil {
			return nil, err
		}
		if n < settleBatch {
			break
		}
	}
	next, err := s.Q.TrackingNextSettle(ctx)
	if err != nil || next.Year() < 2000 {
		return nil, err
	}
	return &next, nil
}

// settle settles up to settleBatch due clicks in one transaction and
// returns how many.
func (s *Service) settle(ctx context.Context, now time.Time) (int, error) {
	n := 0
	err := sending.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		n = 0
		q := store.New(tx)
		rows, err := q.TrackingDueClicks(ctx, store.TrackingDueClicksParams{Now: now, Lim: settleBatch})
		if err != nil || len(rows) == 0 {
			return err
		}
		n = len(rows)
		b := &batch{}
		done := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			if err := s.settleOne(ctx, tx, q, b, r); err != nil {
				return err
			}
			done = append(done, r.EventID)
		}
		if err := q.TrackingSettled(ctx, done); err != nil {
			return err
		}
		if err := b.ledger.Flush(ctx, q); err != nil {
			return err
		}
		return b.touches.flush(ctx, q)
	})
	return n, err
}

// settleOne decides one pending click and counts it.
func (s *Service) settleOne(ctx context.Context, tx pgx.Tx, q *store.Queries, b *batch, r store.TrackingDueClicksRow) error {
	m, err := q.TrackingMessage(ctx, r.MessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // the message is gone: nothing to count
	}
	if err != nil {
		return err
	}
	if err := q.CountersLockMessage(ctx, m.ID); err != nil {
		return err
	}
	v := s.verdictAt(ctx, q, m, r)
	if v.Machine {
		if err := q.TrackingSettleEvent(ctx, store.TrackingSettleEventParams{ID: r.EventID, Class: ClassBot, Machine: true, Diagnostic: clip(v.Reason, 1000)}); err != nil {
			return err
		}
		if m.CampaignID != nil {
			b.ledger.Click(*m.CampaignID, int(r.Link), true, false)
		}
		return nil
	}
	if err := q.TrackingSettleEvent(ctx, store.TrackingSettleEventParams{ID: r.EventID, Class: ClassHuman, Machine: false, Diagnostic: clip(v.Client, 1000)}); err != nil {
		return err
	}
	return s.countClick(ctx, tx, q, b, m, int(r.Link), r.Url, r.ClickedAt)
}

// verdictAt classifies a pending click with what is known now: the
// message's delivery (windows from arrival), the clicks around it (a burst,
// the hidden link), the instance's networks.
func (s *Service) verdictAt(ctx context.Context, q *store.Queries, m store.TrackingMessageRow, r store.TrackingDueClicksRow) Verdict {
	at := r.ClickedAt
	clicks, err := q.TrackingMessageClicks(ctx, store.TrackingMessageClicksParams{MessageID: &m.ID, FromAt: at.Add(-honeypotWindow), UntilAt: at.Add(honeypotWindow)})
	if err != nil {
		s.log().WarnContext(ctx, "read a message's clicks", "message", m.ID, "err", err)
	}
	links := map[int32]bool{}
	for _, c := range clicks {
		if c.Link == HoneypotLink {
			return ClassifyHoneypot()
		}
		if d := c.OccurredAt.Sub(at); d >= -burstWindow && d <= burstWindow && c.Link >= 0 {
			links[c.Link] = true
		}
	}
	sig := s.signal(ctx, m, r.UserAgent, r.Ip, "GET", at)
	return ClassifyClick(sig, len(links) >= burstLinks)
}

// signal is what a request tells about who made it, with the windows
// measured from the message's delivery when it is known.
func (s *Service) signal(ctx context.Context, m store.TrackingMessageRow, ua string, ip *netip.Addr, method string, at time.Time) Signal {
	sig := Signal{UserAgent: ua, Method: method, SinceSend: -1}
	if ip != nil {
		sig.IP = *ip
	}
	if m.SentAt != nil {
		sig.SinceSend = at.Sub(*m.SentAt)
	}
	if m.DeliveredAt != nil {
		sig.SinceDelivery, sig.Delivered = at.Sub(*m.DeliveredAt), true
	}
	sig.Net = s.networkOf(ctx, sig.IP)
	return sig
}

// countClick counts a person's click: the message moves (a click is a
// human open too), the link's clicks, the contact's activity, and what
// listens to clicks.
func (s *Service) countClick(ctx context.Context, tx pgx.Tx, q *store.Queries, b *batch, m store.TrackingMessageRow, link int, url string, at time.Time) error {
	mv, err := q.CountersMoveEngagement(ctx, store.CountersMoveEngagementParams{ID: m.ID, At: at, Click: true, Class: ClassHuman})
	if err != nil {
		return err
	}
	moved := counters.FromEngagement(mv)
	b.ledger.Add(moved)
	firstOpen := moved.Before.OpenedAt == nil
	firstClick := moved.Before.ClickedAt == nil
	if m.CampaignID != nil {
		unique := false
		if _, err := q.TrackingFirstLinkClick(ctx, store.TrackingFirstLinkClickParams{MessageID: m.ID, Link: int32(link), At: at}); err == nil {
			unique = true
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		b.ledger.Click(*m.CampaignID, link, false, unique)
	} else if ref := refOf(m.ContentRef); ref.Email != uuid.Nil {
		if err := q.TrackingBumpAutomationLink(ctx, store.TrackingBumpAutomationLinkParams{EmailID: ref.Email, Version: int32(ref.Version), Language: ref.Language, Position: int32(link)}); err != nil {
			return err
		}
	}
	if m.ContactID != nil {
		b.touches.add(*m.ContactID, at, true, true, true)
	}
	if s.Clicked != nil {
		if err := s.Clicked(ctx, q, HumanClick{WorkspaceID: m.WorkspaceID, MessageID: m.ID, CampaignID: m.CampaignID, ContactID: m.ContactID, Stream: m.Stream,
			URL: url, At: at}); err != nil {
			return err
		}
	}
	if m.Stream == "transactional" && s.Transactional != nil && (firstOpen || firstClick) {
		if err := s.Transactional.Engagement(ctx, tx, m.ID, firstClick, at); err != nil {
			return err
		}
	}
	if m.Stream == "automation" && s.Automations != nil {
		if firstOpen {
			if err := s.Automations.RecordMessageEvent(ctx, q, automations.MessageEvent{MessageID: m.ID, Type: "opened", At: at}); err != nil {
				return err
			}
		}
		if firstClick {
			if err := s.Automations.RecordMessageEvent(ctx, q, automations.MessageEvent{MessageID: m.ID, Type: "clicked", At: at}); err != nil {
				return err
			}
		}
	}
	return nil
}

// SettleArgs settles the clicks that are due (periodic).
type SettleArgs struct{}

// Kind implements river.JobArgs.
func (SettleArgs) Kind() string { return "tracking.settle" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SettleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Second}}
}

// SettleWorker runs SettleArgs.
type SettleWorker struct {
	river.WorkerDefaults[SettleArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SettleWorker) Work(ctx context.Context, _ *river.Job[SettleArgs]) error {
	_, err := w.S.SettleEngagement(ctx, w.S.now())
	return err
}
