package engine_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// inDoubtOnly sends n Gmail readers through a fault provider that takes the
// mail but never answers, until only messages in doubt are left and the
// send waits for a decision on them. It returns the provider and where the
// clock stopped.
func inDoubtOnly(t *testing.T, f *sendtest.Fixture, n int) (uuid.UUID, time.Time) {
	t.Helper()
	gmailOnly(f, n)
	sendtest.Fault.Reset(sendtest.FaultTakeNoAnswer)
	pid := f.UseFaultProvider(false)
	f.E.CallTimeout = 200 * time.Millisecond
	t.Cleanup(func() { f.E.CallTimeout = 0 })
	f.StartSend(engine.StartOptions{})
	at, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	sendtest.Fault.SetMode(sendtest.FaultOK)
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	if s := f.Status(); s != "paused" || statusCounts(t, f)["in_doubt"] == 0 {
		f.Dump()
		t.Fatalf("campaign %s, %v: want only messages in doubt left, waiting", s, statusCounts(t, f))
	}
	return pid, at.Add(2 * time.Minute)
}

// testInDoubtDecision: with only messages in doubt left, the send waits for
// the decision on them (review GS-1). The monitor says so, with the ways to
// continue worked out as of now; resuming can't send anything and says so;
// resuming after a person's pause doesn't decide on them or tell the team
// again; leaving them out ends the send, the report says "not sent, in
// doubt", and an event from the provider for one still counts it as sent.
func testInDoubtDecision(t *testing.T, f *sendtest.Fixture) {
	pid, at := inDoubtOnly(t, f, 20)
	setClock(t, f, at)
	doubt := statusCounts(t, f)["in_doubt"]
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	d := snap.InDoubt
	if d == nil || !d.Waiting || d.Count != doubt || d.Safe || d.IncidentID == "" || snap.Options.CanResume {
		t.Fatalf("in doubt %+v, can resume %v: want waiting for the decision, nothing to resume", d, snap.Options.CanResume)
	}
	if sh := d.Providers[0]; sh.ProviderID != pid.String() || sh.Idempotent || sh.SafeUntil != nil || sh.Safe {
		t.Fatalf("share %+v: the fault provider doesn't recognize a message it already took", sh)
	}
	var in *engine.LiveIncident
	for i := range snap.Incidents {
		if snap.Incidents[i].ID == d.IncidentID {
			in = &snap.Incidents[i]
		}
	}
	if in == nil || in.ResolvedAt != nil || len(in.Detail.Options) != 2 || in.Detail.Options[0].Key != "resend" || in.Detail.Options[0].Recommended ||
		!in.Detail.Options[1].Recommended || !strings.Contains(in.Detail.Options[1].Body, `"not sent, in doubt"`) {
		t.Fatalf("in-doubt incident %+v", in)
	}
	if snap.Incident != nil {
		t.Fatalf("incident %+v: the in-doubt decision has its own panel", snap.Incident)
	}

	path := "/workspaces/acme-newsletter/campaigns/" + f.Campaign.String() + "/live"
	linh := f.Env.Client()
	linh.SignInDemo("linh@acme.io")
	res := linh.Do("POST", path+"/resume", map[string]any{}).MustStatus(t, 409)
	if detail, _ := res.Problem(t)["detail"].(string); !strings.Contains(detail, "in doubt") {
		t.Fatalf("resume problem %q, want it to name the messages in doubt", detail)
	}
	notified := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1`, f.Campaign.String())
	// A person's pause, then resuming it: the messages in doubt still wait
	// for their own decision, and nobody is told about them again.
	if err := f.E.Pause(f.Ctx, f.Campaign, f.Sofia, "checking"); err != nil {
		t.Fatal(err)
	}
	if err := f.E.Resume(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'in_doubt'`, f.Campaign); n != 1 {
		t.Fatalf("%d in-doubt incidents: resuming closed it and a new one opened", n)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1`, f.Campaign.String()); n != notified {
		t.Fatalf("%d notifications after resuming, want %d: the team was told again", n, notified)
	}
	if s := f.Status(); s != "paused" {
		t.Fatalf("campaign %s, want waiting for the decision again", s)
	}

	var out engine.LiveActionResult
	linh.Do("POST", path+"/in-doubt", map[string]any{"action": "leave_out"}).MustStatus(t, 200).JSON(t, &out)
	if out.Decided != doubt || out.Snapshot.InDoubt != nil {
		t.Fatalf("left out %d of %d; in doubt %+v", out.Decided, doubt, out.Snapshot.InDoubt)
	}
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, at.Add(30*time.Second), time.Minute); err != nil || !done {
		f.Dump()
		t.Fatalf("the send didn't end after leaving them out: done=%v err=%v", done, err)
	}
	var rep struct {
		Campaign struct{ Notes []string } `json:"campaign"`
	}
	linh.Do("GET", "/workspaces/acme-newsletter/campaigns/"+f.Campaign.String()+"/report", nil).MustStatus(t, 200).JSON(t, &rep)
	want := fmt.Sprintf("%d messages not sent, in doubt", doubt)
	if doubt == 1 {
		want = "1 message not sent, in doubt"
	}
	found := false
	for _, n := range rep.Campaign.Notes {
		found = found || n == want
	}
	if !found {
		t.Fatalf("report notes %q, want %q", rep.Campaign.Notes, want)
	}

	// The provider reports a delivery for one of them after all.
	var id uuid.UUID
	var email string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT id, email FROM messages WHERE campaign_id = $1 AND status = 'cancelled' AND doubt_at IS NOT NULL LIMIT 1`, f.Campaign).Scan(&id, &email); err != nil {
		t.Fatal(err)
	}
	p, err := f.Env.App.Q.ProvGetAny(f.Ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	sent := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign)
	err = pgx.BeginFunc(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
		ev := providers.Event{Type: providers.EventDelivered, MessageID: &id, ProviderMessageID: "fault-" + id.String(), Recipient: email, OccurredAt: at.Add(time.Minute),
			OriginalType: "Delivery", OriginalCode: "250 OK"}
		_, err := f.E.Normalizer.Apply(f.Ctx, tx, p, nil, "webhook", at.Add(time.Minute), ev)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var st string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT status FROM messages WHERE id = $1`, id).Scan(&st); err != nil || st != "delivered" {
		t.Fatalf("a left-out message the provider delivered is %s (%v)", st, err)
	}
	if n := f.Count(`SELECT coalesce(sum(sent), 0) FROM campaign_counters WHERE campaign_id = $1`, f.Campaign); n != sent+1 {
		t.Fatalf("counters say %d sent, want %d: it counts as sent after all", n, sent+1)
	}
}

// testInDoubtWindowPassed: a provider that recognizes a message it already
// took does so only for its idempotency window. Messages whose call is older
// than that are in doubt like any other: the monitor says the window passed
// and doesn't call sending them again safe (review GS-1, F-6).
func testInDoubtWindowPassed(t *testing.T, f *sendtest.Fixture) {
	gmailOnly(f, 12)
	sendtest.Fault.Reset(sendtest.FaultOK)
	pid := f.UseFaultProvider(true)
	fn := "gs1_window_" + f.Campaign.String()[:8]
	f.Exec(`CREATE FUNCTION ` + fn + `() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
		IF NEW.status = 'sent' AND NEW.campaign_id = '` + f.Campaign.String() + `' THEN RAISE EXCEPTION 'the database is unavailable'; END IF; RETURN NEW; END $$`)
	f.Exec(`CREATE TRIGGER ` + fn + ` BEFORE UPDATE ON messages FOR EACH ROW EXECUTE FUNCTION ` + fn + `()`)
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 3*time.Second); err == nil {
		t.Fatal("recording the batch should have failed")
	}
	f.Exec(`DROP TRIGGER ` + fn + ` ON messages`)
	f.Exec(`DROP FUNCTION ` + fn + `()`)
	// The guard comes back after the provider's 30-minute window.
	later := f.Start.Add(40 * time.Minute)
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, later, time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := statusCounts(t, f)["in_doubt"]; n == 0 {
		f.Dump()
		t.Fatal("nothing in doubt once the window passed")
	}
	setClock(t, f, later.Add(time.Minute))
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.InDoubt == nil || len(snap.InDoubt.Providers) != 1 {
		t.Fatalf("in doubt %+v", snap.InDoubt)
	}
	sh := snap.InDoubt.Providers[0]
	if sh.ProviderID != pid.String() || !sh.Idempotent || sh.WindowSeconds != 1800 || sh.SafeUntil == nil || sh.Safe || snap.InDoubt.Safe {
		t.Fatalf("share %+v: want a 30-minute window that has passed", sh)
	}
	if want := sh.Since.Add(25 * time.Minute); !sh.SafeUntil.Equal(want) {
		t.Fatalf("safe until %v, want %v (the window less its margin)", sh.SafeUntil, want)
	}
	for _, in := range snap.Incidents {
		if in.Kind == "in_doubt" && in.ResolvedAt == nil {
			if o := in.Detail.Options[0]; o.Recommended || o.Body != "Anyone whose copy the provider did take gets the email twice." {
				t.Fatalf("resend option %+v after the window", o)
			}
			return
		}
	}
	t.Fatal("no open in-doubt incident")
}

// testCancelLeavesInDoubtOut: cancelling a send sends nothing more,
// messages in doubt included: they are left out, never left in doubt.
func testCancelLeavesInDoubtOut(t *testing.T, f *sendtest.Fixture) {
	_, at := inDoubtOnly(t, f, 10)
	setClock(t, f, at)
	doubt := statusCounts(t, f)["in_doubt"]
	if err := f.E.Cancel(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if st := statusCounts(t, f); st["in_doubt"] != 0 {
		t.Fatalf("after cancelling: %v", st)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'cancelled' AND doubt_at IS NOT NULL`, f.Campaign); n != doubt {
		t.Fatalf("%d left out while in doubt, want %d", n, doubt)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.cancelled' AND (after->>'in_doubt_left_out')::int = $2`, f.Campaign.String(), doubt); n != 1 {
		t.Fatal("the cancel's audit entry doesn't count the messages in doubt it left out")
	}
}
