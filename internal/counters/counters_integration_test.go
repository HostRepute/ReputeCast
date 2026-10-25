package counters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracking"
)

func TestCounters(t *testing.T) {
	s := sendtest.NewSuite(t)
	t.Run("SendThenRebuild", func(t *testing.T) { testSendThenRebuild(t, s.Fixture(t, "Counters rebuild")) })
	t.Run("Transitions", func(t *testing.T) { testTransitions(t, s.Fixture(t, "Counter transitions")) })
	t.Run("OneLockOrder", func(t *testing.T) { testLockOrder(t, s.Fixture(t, "Counter lock order")) })
	t.Run("TrackingRetry", func(t *testing.T) { testTrackingRetry(t, s.Fixture(t, "Tracking retry")) })
}

// snapshot is everything counted for a campaign, as text.
func snapshot(t *testing.T, f *sendtest.Fixture) string {
	t.Helper()
	var b strings.Builder
	read := func(sql string) {
		rows, err := f.Env.App.Pool.Query(f.Ctx, sql, f.Campaign)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			vals, err := rows.Values()
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(&b, vals...)
		}
	}
	read(`SELECT provider_id, mailbox_provider, variant, sent, delivered, deferred, hard_bounced, soft_bounced, complained, rejected,
		opened_human, opened_privacy, opened_machine, clicked, unsubscribed
		FROM campaign_counters WHERE campaign_id = $1 AND (sent + delivered + deferred + hard_bounced + soft_bounced + complained + rejected
			+ opened_human + opened_privacy + opened_machine + clicked + unsubscribed) > 0 ORDER BY 1, 2, 3`)
	read(`SELECT hour, opened_human, opened_privacy, opened_machine, clicked FROM campaign_hourly WHERE campaign_id = $1
		AND (opened_human + opened_privacy + opened_machine + clicked) > 0 ORDER BY hour`)
	read(`SELECT position, clicks, unique_clicks, bot_clicks FROM campaign_links WHERE campaign_id = $1 ORDER BY position`)
	return b.String()
}

// testSendThenRebuild: after a production-shape send (deliveries through
// webhook normalization, opens and clicks through the tracking recorder),
// rebuilding the counters from the stored messages and events finds what
// the writers counted as the events came in.
func testSendThenRebuild(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: 260, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 120, Mailbox: "outlook", Domain: "outlook.com", Engaged: true},
		sendtest.ContactSpec{N: 60, Mailbox: "yahoo", Domain: "yahoo.com"},
		sendtest.ContactSpec{N: 80, Mailbox: "icloud", Domain: "icloud.com", Engaged: true},
		sendtest.ContactSpec{N: 40, Mailbox: "other", Domain: "example.org"},
	)
	f.StartSend(engine.StartOptions{})
	if _, done := f.Drive(30 * time.Minute); !done {
		f.Dump()
		t.Fatal("the send didn't finish")
	}
	f.Drive(6 * time.Hour) // deliveries, opens and clicks keep arriving
	live := snapshot(t, f)
	if !strings.Contains(live, " ") {
		t.Fatal("nothing counted")
	}
	res, err := counters.Rebuild(f.Ctx, f.Env.App.Pool, f.Campaign, time.Now())
	if err != nil || !res.Rebuilt {
		t.Fatalf("rebuild: %+v %v", res, err)
	}
	if rebuilt := snapshot(t, f); rebuilt != live {
		t.Fatalf("rebuilt counters differ from the live ones\nlive:\n%s\nrebuilt:\n%s", live, rebuilt)
	}
	if n := f.Count(`SELECT version FROM campaign_counter_builds WHERE campaign_id = $1 AND skipped = ''`, f.Campaign); n != counters.Version {
		t.Fatalf("build recorded with version %d", n)
	}
	// Every sent message is delivered, bounced or still undelivered, once.
	var sent, delivered, hard, soft, pending int64
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT coalesce(sum(sent), 0), coalesce(sum(delivered), 0), coalesce(sum(hard_bounced), 0), coalesce(sum(soft_bounced), 0)
		FROM campaign_counters WHERE campaign_id = $1`, f.Campaign).Scan(&sent, &delivered, &hard, &soft); err != nil {
		t.Fatal(err)
	}
	pending = f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL AND status IN ('sent', 'deferred') AND delivered_at IS NULL
		AND soft_bounced_at IS NULL`, f.Campaign)
	if sent != delivered+hard+soft+pending {
		t.Fatalf("sent %d ≠ delivered %d + hard %d + soft %d + undelivered %d", sent, delivered, hard, soft, pending)
	}
}

type script struct {
	t   *testing.T
	f   *sendtest.Fixture
	ids []uuid.UUID
	p   map[uuid.UUID]store.Provider
}

func (s *script) provider(msg uuid.UUID) store.Provider {
	s.t.Helper()
	var pid uuid.UUID
	if err := s.f.Env.App.Pool.QueryRow(s.f.Ctx, `SELECT provider_id FROM messages WHERE id = $1`, msg).Scan(&pid); err != nil {
		s.t.Fatal(err)
	}
	if p, ok := s.p[pid]; ok {
		return p
	}
	p, err := s.f.Env.App.Q.ProvGetAny(s.f.Ctx, pid)
	if err != nil {
		s.t.Fatal(err)
	}
	s.p[pid] = p
	return p
}

// event applies a provider event to a message through the webhook
// normalizer, as a webhook would.
func (s *script) event(msg uuid.UUID, typ providers.EventType, after time.Duration) {
	s.t.Helper()
	p := s.provider(msg)
	at := s.f.Start.Add(after)
	id := msg
	ev := providers.Event{Type: typ, MessageID: &id, OccurredAt: at, OriginalType: string(typ), OriginalCode: "250 2.0.0"}
	switch typ {
	case providers.EventHardBounce:
		ev.OriginalCode, ev.BounceClass = "550 5.1.1", "mailbox_unknown"
	case providers.EventSoftBounce:
		ev.OriginalCode, ev.BounceClass = "452 4.2.2", "mailbox_full"
	case providers.EventDeferred:
		ev.OriginalCode = "421 4.7.0"
	}
	if err := sending.Tx(s.f.Ctx, s.f.Env.App.Pool, func(tx pgx.Tx) error {
		_, err := s.f.E.Normalizer.Apply(s.f.Ctx, tx, p, nil, "webhook", at, ev)
		return err
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s *script) sum(col string) int64 {
	s.t.Helper()
	return s.f.Count(`SELECT coalesce(sum(`+col+`), 0) FROM campaign_counters WHERE campaign_id = $1`, s.f.Campaign)
}

// sentMessages starts a small send, lets it hand everything to the
// providers and drops what the demo transport planned for it, so only the
// test's own events move the messages.
func sentMessages(t *testing.T, f *sendtest.Fixture, n int) *script {
	t.Helper()
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: n, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{})
	f.Drive(20 * time.Second)
	if got := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); got != int64(n) {
		f.Dump()
		t.Fatalf("%d of %d sent", got, n)
	}
	f.Exec(`DELETE FROM sending_sim_events WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`UPDATE messages SET status = 'sent', delivered_at = NULL, opened_at = NULL, human_opened_at = NULL, clicked_at = NULL
		WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`UPDATE campaign_counters SET delivered = 0, deferred = 0, hard_bounced = 0, soft_bounced = 0, complained = 0, rejected = 0,
		opened_human = 0, opened_privacy = 0, opened_machine = 0, clicked = 0, unsubscribed = 0 WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`DELETE FROM campaign_hourly WHERE campaign_id = $1`, f.Campaign)
	f.Exec(`DELETE FROM delivery_events WHERE campaign_id = $1`, f.Campaign)
	s := &script{t: t, f: f, p: map[uuid.UUID]store.Provider{}}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT id FROM messages WHERE campaign_id = $1 ORDER BY id`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		s.ids = append(s.ids, id)
	}
	return s
}

// testTransitions moves messages through the provider events that used to
// count twice, and checks that each counts once, where it ends up.
func testTransitions(t *testing.T, f *sendtest.Fixture) {
	s := sentMessages(t, f, 12)
	m := s.ids
	sent := s.sum("sent")
	if sent != 12 {
		t.Fatalf("%d sent", sent)
	}
	// A soft bounce, then the provider's retry delivers it.
	s.event(m[0], providers.EventSoftBounce, time.Minute)
	if s.sum("soft_bounced") != 1 {
		t.Fatal("a soft bounce wasn't counted")
	}
	s.event(m[0], providers.EventDelivered, 30*time.Minute)
	if s.sum("soft_bounced") != 0 || s.sum("delivered") != 1 {
		t.Fatalf("soft bounce then delivered: %d soft, %d delivered; want 0 and 1", s.sum("soft_bounced"), s.sum("delivered"))
	}
	// Deferred twice, then delivered: one deferral.
	s.event(m[1], providers.EventDeferred, time.Minute)
	s.event(m[1], providers.EventDeferred, 6*time.Minute)
	s.event(m[1], providers.EventDelivered, 12*time.Minute)
	if s.sum("deferred") != 1 || s.sum("delivered") != 2 {
		t.Fatalf("deferred twice: %d deferrals, %d delivered; want 1 and 2", s.sum("deferred"), s.sum("delivered"))
	}
	// The same delivery reported again (a redelivered webhook) and a
	// second delivery event for the same message.
	s.event(m[2], providers.EventDelivered, time.Minute)
	s.event(m[2], providers.EventDelivered, time.Minute)
	s.event(m[2], providers.EventDelivered, 3*time.Minute)
	if s.sum("delivered") != 3 {
		t.Fatalf("%d delivered, want 3", s.sum("delivered"))
	}
	// Delivered, then a hard bounce: a bounce, not a delivery.
	s.event(m[3], providers.EventDelivered, time.Minute)
	s.event(m[3], providers.EventHardBounce, 9*time.Minute)
	if s.sum("delivered") != 3 || s.sum("hard_bounced") != 1 {
		t.Fatalf("delivered then bounced: %d delivered, %d hard", s.sum("delivered"), s.sum("hard_bounced"))
	}
	// The provider drops a message it took: it was never sent.
	s.event(m[4], providers.EventRejected, time.Minute)
	if s.sum("sent") != 11 || s.sum("rejected") != 1 {
		t.Fatalf("late rejection: %d sent, %d rejected; want 11 and 1", s.sum("sent"), s.sum("rejected"))
	}
	// A complaint proves the delivery.
	s.event(m[5], providers.EventComplaint, 20*time.Minute)
	if s.sum("complained") != 1 || s.sum("delivered") != 4 {
		t.Fatalf("complaint: %d complained, %d delivered", s.sum("complained"), s.sum("delivered"))
	}
	// The provider reports an unsubscribe, and the reader also used ours.
	s.event(m[6], providers.EventDelivered, time.Minute)
	s.event(m[6], providers.EventUnsubscribe, time.Hour)
	if err := db.Tx(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
		return counters.Unsubscribed(f.Ctx, store.New(tx), m[6], f.Start.Add(2*time.Hour))
	}); err != nil {
		t.Fatal(err)
	}
	if s.sum("unsubscribed") != 1 {
		t.Fatalf("%d unsubscribes, want 1", s.sum("unsubscribed"))
	}
	// A scanner opens it first, then the reader: one human open.
	s.event(m[7], providers.EventDelivered, time.Minute)
	rec := f.E.Recorder
	if err := rec.RecordEngagement(f.Ctx, []sending.Engagement{
		{MessageID: m[7], UserAgent: "Mozilla/5.0 Barracuda Sentinel", Method: "GET", At: f.Start.Add(2 * time.Minute)},
		{MessageID: m[7], UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
			IP: "198.51.100.7", Method: "GET", At: f.Start.Add(3 * time.Hour)},
	}); err != nil {
		t.Fatal(err)
	}
	if s.sum("opened_human") != 1 || s.sum("opened_machine") != 0 {
		t.Fatalf("scanner then person: %d human, %d machine; want 1 and 0", s.sum("opened_human"), s.sum("opened_machine"))
	}
	if h := f.Count(`SELECT coalesce(sum(opened_human), 0) FROM campaign_hourly WHERE campaign_id = $1 AND hour = 3`, f.Campaign); h != 1 {
		t.Fatalf("the human open isn't in hour 3: %d", h)
	}
	if m := f.Count(`SELECT coalesce(sum(opened_machine), 0) FROM campaign_hourly WHERE campaign_id = $1`, f.Campaign); m != 0 {
		t.Fatalf("the scanner's open is still in the chart: %d", m)
	}
	// Undelivered is what was sent and neither delivered nor bounced.
	undelivered := s.sum("sent") - s.sum("delivered") - s.sum("hard_bounced") - s.sum("soft_bounced")
	if want := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'sent'`, f.Campaign); undelivered != want {
		t.Fatalf("undelivered %d, want %d (messages still only sent)", undelivered, want)
	}
	live := snapshot(t, f)
	if _, err := counters.Rebuild(f.Ctx, f.Env.App.Pool, f.Campaign, time.Now()); err != nil {
		t.Fatal(err)
	}
	if rebuilt := snapshot(t, f); rebuilt != live {
		t.Fatalf("rebuilt counters differ\nlive:\n%s\nrebuilt:\n%s", live, rebuilt)
	}
}

// testLockOrder runs webhook batches and tracking batches over the same
// messages at once, in different orders and without retries: writers take
// their locks in one order, so none of them deadlocks.
func testLockOrder(t *testing.T, f *sendtest.Fixture) {
	s := sentMessages(t, f, 40)
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for w := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := range 5 {
				// Each worker walks the messages from a different start.
				evs := make([]providers.Event, 0, len(s.ids))
				eng := make([]sending.Engagement, 0, len(s.ids))
				for i := range s.ids {
					id := s.ids[(i*7+w*11+round*3)%len(s.ids)]
					at := f.Start.Add(time.Duration(w*10+round+1) * time.Minute)
					evs = append(evs, providers.Event{Type: providers.EventDelivered, MessageID: &id, OccurredAt: at, OriginalCode: fmt.Sprintf("250 %d-%d", w, round)})
					eng = append(eng, sending.Engagement{MessageID: id, UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/129.0", IP: "198.51.100.9", Method: "GET",
						At: at.Add(time.Hour)})
				}
				p := s.provider(s.ids[0])
				if err := db.Tx(ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
					_, err := f.E.Normalizer.ApplyAll(ctx, tx, p, nil, "webhook", f.Start, evs)
					return err
				}); err != nil {
					errs <- fmt.Errorf("webhook batch: %w", err)
					return
				}
				if err := f.E.Recorder.RecordEngagement(ctx, eng); err != nil {
					errs <- fmt.Errorf("tracking batch: %w", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if got := s.sum("delivered"); got != 40 {
		t.Fatalf("%d delivered, want 40 (each message once)", got)
	}
	if got := s.sum("opened_human"); got != 40 {
		t.Fatalf("%d human opens, want 40", got)
	}
}

// testTrackingRetry: opens and clicks kept for a retry (the writer couldn't
// record them) are recorded by the retry job, and dropped events don't hold
// the others back.
func testTrackingRetry(t *testing.T, f *sendtest.Fixture) {
	s := sentMessages(t, f, 3)
	open := sending.Engagement{MessageID: s.ids[0], UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/18.0 Safari/605.1.15",
		IP: "198.51.100.20", Method: "GET", At: f.Start.Add(40 * time.Minute)}
	ev, _ := json.Marshal(open)
	f.Exec(`INSERT INTO tracking_retry (message_id, event, next_at) VALUES ($1, $2, now() - interval '1 minute'), ($3, '{"MessageID":"not a uuid"}', now() - interval '1 minute')`,
		s.ids[0], ev, uuid.Must(uuid.NewV7()))
	tr, ok := f.E.Recorder.(*tracking.Service)
	if !ok {
		t.Fatalf("the engine records engagement with %T", f.E.Recorder)
	}
	res, err := tr.RetryParked(f.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Recorded != 1 {
		t.Fatalf("retry recorded %d events, want 1", res.Recorded)
	}
	if n := f.Count(`SELECT count(*) FROM tracking_retry WHERE message_id = ANY($1)`, s.ids); n != 0 {
		t.Fatalf("%d kept events left", n)
	}
	if s.sum("opened_human") != 1 {
		t.Fatal("the kept open wasn't counted")
	}
}
