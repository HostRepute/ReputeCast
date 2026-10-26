package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// TestGuardedSending covers the October 2026 review's guarded sending
// (M34), one subtest per ID, on the production-shape harness.
func TestGuardedSending(t *testing.T) {
	s := sendtest.NewSuite(t)
	// GS-1: no double sends.
	t.Run("TimeoutLeavesMessagesInDoubt", func(t *testing.T) { testTimeoutInDoubt(t, s.Fixture(t, "GS-1 timeout")) })
	t.Run("StoppedWorkerRecordsItsBatch", func(t *testing.T) { testStoppedWorker(t, s.Fixture(t, "GS-1 stop")) })
	t.Run("DatabaseErrorLeavesMessagesInDoubt", func(t *testing.T) { testDBErrorInDoubt(t, s.Fixture(t, "GS-1 db error"), false) })
	t.Run("DatabaseErrorWithIdempotentProvider", func(t *testing.T) { testDBErrorInDoubt(t, s.Fixture(t, "GS-1 db error idem"), true) })
	t.Run("SendInDoubtAgain", func(t *testing.T) { testResendInDoubt(t, s.Fixture(t, "GS-1 resend")) })
	t.Run("EventSettlesInDoubt", func(t *testing.T) { testEventSettlesInDoubt(t, s.Fixture(t, "GS-1 event")) })
	t.Run("StuckJobsRescued", func(t *testing.T) { testRescueStuckJobs(t, s.Fixture(t, "GS-1 rescue")) })
	// GS-4: the provider-errors threshold; throttling and quota are outages.
	t.Run("APIErrorThresholdHoldsRoute", func(t *testing.T) {
		testRouteHeldFor(t, s.Fixture(t, "GS-4 api errors"), sendtest.FaultFlaky, "api_errors")
	})
	t.Run("LastingThrottlingHoldsRoute", func(t *testing.T) {
		testRouteHeldFor(t, s.Fixture(t, "GS-4 throttled"), sendtest.FaultThrottle, "throttled")
	})
	t.Run("UsedUpQuotaHoldsRoute", func(t *testing.T) { testRouteHeldFor(t, s.Fixture(t, "GS-4 quota"), sendtest.FaultQuota, "quota") })
	// C6 / PR-7 (engine part): an unverified sender holds the route.
	t.Run("IdentityErrorHoldsRoute", func(t *testing.T) {
		testRouteHeldFor(t, s.Fixture(t, "PR-7 identity"), sendtest.FaultIdentity, "identity")
	})
	// GS-3: one outage path, Retry with test messages.
	t.Run("HealthJobOutageHasOptions", func(t *testing.T) { testHealthJobOutage(t, s.Fixture(t, "GS-3 health job")) })
	t.Run("RetryAsksAgainWhileDown", func(t *testing.T) { testRetryAsksAgain(t, s.Fixture(t, "GS-3 ask again")) })
	t.Run("RetryResumesAfterThreeTests", func(t *testing.T) { testRetryResumes(t, s.Fixture(t, "GS-3 resume seed"), true) })
	t.Run("RetryTestsWithTheShareWithoutSeeds", func(t *testing.T) { testRetryResumes(t, s.Fixture(t, "GS-3 resume own"), false) })
	// GS-5: moves are vetted, holdouts stay held, a person's pause stays.
	t.Run("MoveKeepsHoldoutsAndPause", func(t *testing.T) { testMoveKeepsHoldouts(t, s.Fixture(t, "GS-5 move")) })
	// GS-2: resuming re-arms; changes at every scope; thresholds while paused.
	t.Run("ResumeDoesNotRetripOnOldEvents", func(t *testing.T) { testResumeRearms(t, s.Fixture(t, "GS-2 mailbox")) })
	t.Run("CampaignResumeRearms", func(t *testing.T) { testCampaignResumeRearms(t, s.Fixture(t, "GS-2 campaign")) })
	t.Run("RouteResumeWithChanges", func(t *testing.T) { testRouteResumeWithChanges(t, s.Fixture(t, "GS-2 route")) })
	t.Run("ThresholdsWhilePaused", func(t *testing.T) { testThresholdsWhilePaused(t, s.Fixture(t, "GS-2 thresholds")) })
	// GS-6: complaints count where reported; the complaint watch after the send.
	t.Run("ComplaintWatchAfterTheSend", func(t *testing.T) { testComplaintWatch(t, s.Fixture(t, "GS-6 watch")) })
	// GS-7: plain SMTP's deliveries are inferred from its acceptance.
	t.Run("SMTPInfersDelivery", func(t *testing.T) { testSMTPInfersDelivery(t, s.Fixture(t, "GS-7 smtp")) })
	// GS-8: throttles shared by sends going out at once; ETAs.
	t.Run("SendsShareThrottles", func(t *testing.T) { testSharedThrottles(t, s) })
}

// routeOf is the fixture's route on a provider: its status and detail.
func routeOf(t *testing.T, f *sendtest.Fixture, pid uuid.UUID) (string, string) {
	t.Helper()
	var st, d string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT status, status_detail FROM campaign_routes WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, pid).Scan(&st, &d); err != nil {
		t.Fatal(err)
	}
	return st, d
}

// testRouteHeldFor: a provider whose answers are about the route (too many
// failed calls, throttling that lasts, a used-up quota) gets its share held
// through the outage path with ways to continue, and no message is marked
// failed or uses up its attempts on answers that aren't about it.
func testRouteHeldFor(t *testing.T, f *sendtest.Fixture, mode, reason string) {
	gmailOnly(f, 400)
	sendtest.Fault.Reset(mode)
	pid := f.UseFaultProvider(false)
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	st, _ := routeOf(t, f, pid)
	if st != "held" {
		f.Dump()
		t.Fatalf("route %s, want held for %s", st, reason)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	in := snap.Incident
	if in == nil || in.Kind != "provider_outage" || in.Detail.Reason != reason || len(in.Detail.Options) == 0 || len(in.Detail.Causes) == 0 {
		t.Fatalf("incident %+v", in)
	}
	keys := map[string]bool{}
	for _, o := range in.Detail.Options {
		keys[o.Key] = true
	}
	if !keys["retry"] || !keys["cancel_share"] {
		t.Fatalf("options %v, want retry and cancel_share", keys)
	}
	if reason == "identity" && in.Detail.ErrorKind != "unverified sender" {
		t.Fatalf("error kind %q, want an unverified sender", in.Detail.ErrorKind)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status IN ('failed', 'rejected')`, f.Campaign); n != 0 {
		t.Fatalf("%d messages failed on answers about the route", n)
	}
	if reason != "api_errors" {
		// Throttling and quota answers never use up a message's attempts.
		if n := f.Count(`SELECT coalesce(max(attempts), 0) FROM messages WHERE campaign_id = $1`, f.Campaign); n != 0 {
			t.Fatalf("messages used %d attempts on %s answers", n, reason)
		}
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was asked to decide")
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.share_held'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d share_held audit entries", n)
	}
}

// gmailOnly makes a fixture without an A/B test sending to n engaged Gmail
// readers.
func gmailOnly(f *sendtest.Fixture, n int) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: n, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
}

// statusCounts is the campaign's messages per status.
func statusCounts(t *testing.T, f *sendtest.Fixture) map[string]int64 {
	t.Helper()
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT status, count(*) FROM messages WHERE campaign_id = $1 GROUP BY status`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var st string
		var n int64
		if err := rows.Scan(&st, &n); err != nil {
			t.Fatal(err)
		}
		out[st] = n
	}
	return out
}

// tookTwice is how many messages the fault provider took more than once.
func tookTwice() int {
	n := 0
	for _, c := range sendtest.Fault.Took() {
		if c > 1 {
			n++
		}
	}
	return n
}

// testTimeoutInDoubt: the provider takes the mail but its answer never comes
// (the call times out). Those messages are in doubt, never sent again on a
// guess; the send waits for a decision, and leaving them out ends it.
func testTimeoutInDoubt(t *testing.T, f *sendtest.Fixture) {
	gmailOnly(f, 30)
	sendtest.Fault.Reset(sendtest.FaultTakeNoAnswer)
	pid := f.UseFaultProvider(false)
	f.E.CallTimeout = 200 * time.Millisecond
	defer func() { f.E.CallTimeout = 0 }()
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 20*time.Second); err != nil {
		t.Fatal(err)
	}
	took := len(sendtest.Fault.Took())
	if took == 0 {
		f.Dump()
		t.Fatal("the provider took nothing")
	}
	st := statusCounts(t, f)
	if st["in_doubt"] != int64(took) {
		f.Dump()
		t.Fatalf("%d in doubt, the provider took %d: %v", st["in_doubt"], took, st)
	}
	// Later passes, after the stale time too, never send them again.
	sendtest.Fault.SetMode(sendtest.FaultOK)
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(20*time.Second), 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := tookTwice(); n != 0 {
		t.Fatalf("%d messages delivered twice", n)
	}
	if f.Status() != "paused" {
		f.Dump()
		t.Fatalf("status %s; with only messages in doubt left the send waits for a decision", f.Status())
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.InDoubt == nil || snap.InDoubt.Count != int64(took) || snap.InDoubt.Safe || len(snap.InDoubt.Providers) != 1 || snap.InDoubt.Providers[0].ProviderID != pid.String() {
		t.Fatalf("snapshot in doubt %+v", snap.InDoubt)
	}
	for _, p := range snap.Providers {
		if p.ID == pid.String() && p.Idempotent {
			t.Fatal("the fault provider doesn't dedupe; the monitor says it does")
		}
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'in_doubt' AND resolved_at IS NULL AND held = $2`, f.Campaign, took); n != 1 {
		t.Fatalf("%d open in-doubt incidents holding %d", n, took)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE data->>'campaignId' = $1 AND kind = 'send_paused'`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was told about the messages in doubt")
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.messages_in_doubt'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d audit entries for the messages in doubt", n)
	}
	n, err := f.E.ResolveInDoubt(f.Ctx, f.Campaign, f.Sofia, false, nil)
	if err != nil || n != int64(took) {
		t.Fatalf("left out %d: %v", n, err)
	}
	if _, err := f.E.ResolveInDoubt(f.Ctx, f.Campaign, f.Sofia, false, nil); !errors.Is(err, engine.ErrState) {
		t.Fatalf("a second decision: %v, want a state error", err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'in_doubt' AND resolution = 'cancelled'`, f.Campaign); n != 1 {
		t.Fatal("the in-doubt incident wasn't resolved")
	}
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(11*time.Minute), 10*time.Minute); err != nil || !done {
		f.Dump()
		t.Fatalf("the send didn't end after the decision: done=%v err=%v", done, err)
	}
	if n := tookTwice(); n != 0 {
		t.Fatalf("%d messages delivered twice", n)
	}
	if st := statusCounts(t, f); st["in_doubt"] != 0 || st["queued"] != 0 {
		t.Fatalf("after the decision: %v", st)
	}
}

// testStoppedWorker: the worker is told to stop (SIGTERM) while a call is
// in flight. The call finishes, its result is recorded, and nothing of that
// batch is ever sent again.
func testStoppedWorker(t *testing.T, f *sendtest.Fixture) {
	gmailOnly(f, 15)
	sendtest.Fault.Reset(sendtest.FaultHold)
	f.UseFaultProvider(false)
	f.StartSend(engine.StartOptions{})
	ctx, cancel := context.WithCancel(f.Ctx)
	done := make(chan error, 1)
	go func() {
		_, _, err := f.E.Run(ctx, f.Campaign, time.Minute)
		done <- err
	}()
	select {
	case <-sendtest.Fault.Started():
	case <-time.After(30 * time.Second):
		t.Fatal("no call reached the provider")
	}
	cancel() // the worker is stopping
	time.Sleep(200 * time.Millisecond)
	sendtest.Fault.Release() // the provider answers after all
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("the loop ended with %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the loop didn't stop")
	}
	took := len(sendtest.Fault.Took())
	if took == 0 {
		t.Fatal("the held call took nothing")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'sent' AND claimed_at IS NULL`, f.Campaign); n != int64(took) {
		f.Dump()
		t.Fatalf("%d messages recorded as sent, the provider took %d", n, took)
	}
	sendtest.Fault.SetMode(sendtest.FaultOK)
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(time.Second), 10*time.Minute); err != nil || !done {
		f.Dump()
		t.Fatalf("the send didn't finish: done=%v err=%v", done, err)
	}
	if n := tookTwice(); n != 0 || len(sendtest.Fault.Took()) != 15 {
		t.Fatalf("%d of 15 delivered, %d twice", len(sendtest.Fault.Took()), n)
	}
	if f.Status() != "sent" {
		t.Fatalf("status %s", f.Status())
	}
}

// testDBErrorInDoubt: the database refuses the batch's result after the
// provider took it. The messages stay claimed; past the stale time they are
// in doubt, or, with a provider that dedupes by message id, sent again and
// dropped by the provider.
func testDBErrorInDoubt(t *testing.T, f *sendtest.Fixture, dedupe bool) {
	gmailOnly(f, 12)
	sendtest.Fault.Reset(sendtest.FaultOK)
	pid := f.UseFaultProvider(dedupe)
	fn := "gs1_refuse_" + f.Campaign.String()[:8]
	f.Exec(`CREATE FUNCTION ` + fn + `() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
		IF NEW.status = 'sent' AND NEW.campaign_id = '` + f.Campaign.String() + `' THEN RAISE EXCEPTION 'the database is unavailable'; END IF; RETURN NEW; END $$`)
	f.Exec(`CREATE TRIGGER ` + fn + ` BEFORE UPDATE ON messages FOR EACH ROW EXECUTE FUNCTION ` + fn + `()`)
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 3*time.Second); err == nil {
		t.Fatal("recording the batch should have failed")
	}
	f.Exec(`DROP TRIGGER ` + fn + ` ON messages`)
	f.Exec(`DROP FUNCTION ` + fn + `()`)
	took := len(sendtest.Fault.Took())
	if took == 0 {
		t.Fatal("the provider took nothing")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'queued' AND claimed_at IS NOT NULL`, f.Campaign); n != int64(took) {
		t.Fatalf("%d messages still claimed, the provider took %d", n, took)
	}
	// Before the stale time nothing touches them; after it they settle.
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(6*time.Minute), 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	st := statusCounts(t, f)
	if dedupe {
		if st["in_doubt"] != 0 || st["sent"] != 12 {
			f.Dump()
			t.Fatalf("with a provider that dedupes: %v", st)
		}
		if n := tookTwice(); n != 0 || len(sendtest.Fault.Took()) != 12 {
			t.Fatalf("%d delivered, %d twice", len(sendtest.Fault.Took()), n)
		}
		if sendtest.Fault.Calls() <= 2 {
			t.Fatal("the stale messages weren't sent again")
		}
		return
	}
	if st["in_doubt"] != int64(took) {
		f.Dump()
		t.Fatalf("%d in doubt, the provider took %d: %v", st["in_doubt"], took, st)
	}
	if n := tookTwice(); n != 0 {
		t.Fatalf("%d messages delivered twice", n)
	}
	var detail []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT detail FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'in_doubt'`, f.Campaign).Scan(&detail); err != nil {
		t.Fatal(err)
	}
	var d engine.IncidentDetail
	_ = json.Unmarshal(detail, &d)
	if len(d.InDoubt) != 1 || d.InDoubt[0].ProviderID != pid.String() || d.InDoubt[0].Idempotent || len(d.Options) != 2 || d.Options[1].Key != "leave_out" || !d.Options[1].Recommended {
		t.Fatalf("in-doubt detail %+v", d)
	}
}

// testResendInDoubt: a person sends the messages in doubt again; they go
// out (once more) and the send finishes.
func testResendInDoubt(t *testing.T, f *sendtest.Fixture) {
	gmailOnly(f, 10)
	sendtest.Fault.Reset(sendtest.FaultTakeNoAnswer)
	f.UseFaultProvider(false)
	f.E.CallTimeout = 200 * time.Millisecond
	defer func() { f.E.CallTimeout = 0 }()
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	doubt := statusCounts(t, f)["in_doubt"]
	if doubt == 0 {
		t.Fatal("nothing in doubt")
	}
	sendtest.Fault.SetMode(sendtest.FaultOK)
	// Only people who can send decide; the decision is theirs in the log.
	path := "/workspaces/acme-newsletter/campaigns/" + f.Campaign.String() + "/live/in-doubt"
	viewer := f.Env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("POST", path, map[string]any{"action": "resend"}).MustStatus(t, 403)
	writer := f.Env.Client()
	writer.SignInDemo("daniel@acme.io")
	writer.Do("POST", path, map[string]any{"action": "resend"}).MustStatus(t, 403)
	owner := f.Env.Client()
	owner.SignInDemo("linh@acme.io")
	owner.Do("POST", path, map[string]any{"action": "send_twice"}).MustStatus(t, 422)
	var res engine.LiveActionResult
	owner.Do("POST", path, map[string]any{"action": "resend"}).MustStatus(t, 200).JSON(t, &res)
	if res.Decided != doubt || res.Snapshot.InDoubt != nil {
		t.Fatalf("sent %d of %d again; snapshot in doubt %+v", res.Decided, doubt, res.Snapshot.InDoubt)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.in_doubt_resent' AND actor_name = 'Linh Nguyen'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d audit entries by Linh for sending them again", n)
	}
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start.Add(10*time.Second), 10*time.Minute); err != nil || !done {
		f.Dump()
		t.Fatalf("the send didn't finish: done=%v err=%v", done, err)
	}
	if st := statusCounts(t, f); st["sent"] != 10 {
		t.Fatalf("after sending them again: %v", st)
	}
}

// testEventSettlesInDoubt: the provider reports an event for a message in
// doubt; it took it after all, so the message counts as sent and the
// incident closes once nothing is left in doubt.
func testEventSettlesInDoubt(t *testing.T, f *sendtest.Fixture) {
	gmailOnly(f, 5)
	sendtest.Fault.Reset(sendtest.FaultTakeNoAnswer)
	pid := f.UseFaultProvider(false)
	f.E.CallTimeout = 200 * time.Millisecond
	defer func() { f.E.CallTimeout = 0 }()
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT id, email FROM messages WHERE campaign_id = $1 AND status = 'in_doubt'`, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	type msg struct {
		id    uuid.UUID
		email string
	}
	var doubt []msg
	for rows.Next() {
		var m msg
		if err := rows.Scan(&m.id, &m.email); err != nil {
			t.Fatal(err)
		}
		doubt = append(doubt, m)
	}
	rows.Close()
	if len(doubt) == 0 {
		t.Fatal("nothing in doubt")
	}
	p, err := f.Env.App.Q.ProvGetAny(f.Ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	sentBefore := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign)
	err = pgx.BeginFunc(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
		for _, m := range doubt {
			id := m.id
			ev := providers.Event{Type: providers.EventDelivered, MessageID: &id, ProviderMessageID: "fault-" + id.String(), Recipient: m.email, OccurredAt: f.Start.Add(time.Minute),
				OriginalType: "Delivery", OriginalCode: "250 OK"}
			if _, err := f.E.Normalizer.Apply(f.Ctx, tx, p, nil, "webhook", f.Start.Add(time.Minute), ev); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if st := statusCounts(t, f); st["in_doubt"] != 0 || st["delivered"] != int64(len(doubt)) {
		t.Fatalf("after the events: %v", st)
	}
	if n := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign); n != sentBefore+int64(len(doubt)) {
		t.Fatalf("counters say %d sent, want %d", n, sentBefore+int64(len(doubt)))
	}
	if err := f.E.Watch(f.Ctx); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'in_doubt' AND resolution = 'auto_recovered'`, f.Campaign); n != 1 {
		t.Fatal("the in-doubt incident didn't close once events settled every message")
	}
}

// testRescueStuckJobs: a send job whose worker died stays "running" in
// River; the watchdog gives it back within minutes, unless a process still
// holds the send.
func testRescueStuckJobs(t *testing.T, f *sendtest.Fixture) {
	other := uuid.Must(uuid.NewV7())
	insert := func(campaign uuid.UUID, ago time.Duration) int64 {
		var id int64
		args, _ := json.Marshal(map[string]string{"campaign_id": campaign.String()})
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `INSERT INTO river_job (args, attempt, attempted_at, kind, max_attempts, priority, queue, state, scheduled_at, metadata, tags)
			VALUES ($1, 1, now() - make_interval(secs => $2), 'sending.run', 12, 1, 'sending', 'running', now() - make_interval(secs => $2), '{}', '{}') RETURNING id`,
			args, ago.Seconds()).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	stuck := insert(f.Campaign, 4*time.Minute)
	fresh := insert(other, time.Minute)
	held := insert(uuid.Must(uuid.NewV7()), 10*time.Minute)
	// A process still runs the third one's send: it keeps the lock.
	var heldCampaign uuid.UUID
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT (args->>'campaign_id')::uuid FROM river_job WHERE id = $1`, held).Scan(&heldCampaign); err != nil {
		t.Fatal(err)
	}
	unlock, err := engine.LockSendForTest(f.E, f.Ctx, heldCampaign)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if err := f.E.Watch(f.Ctx); err != nil {
		t.Fatal(err)
	}
	state := func(id int64) string {
		var s string
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT state::text FROM river_job WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if s := state(stuck); s != "retryable" && s != "available" {
		t.Fatalf("the stuck job is %s, want rescued", s)
	}
	if s := state(fresh); s != "running" {
		t.Fatalf("a job still inside its time is %s", s)
	}
	if s := state(held); s != "running" {
		t.Fatalf("a job whose send is held by a process is %s", s)
	}
	f.Exec(`DELETE FROM river_job WHERE id = ANY($1)`, []int64{stuck, fresh, held})
}
