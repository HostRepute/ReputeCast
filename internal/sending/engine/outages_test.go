package engine_test

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// setClock sets the engine's clock (people's decisions) to at.
func setClock(t *testing.T, f *sendtest.Fixture, at time.Time) {
	t.Helper()
	old := f.E.Now
	f.E.Now = func() time.Time { return at }
	t.Cleanup(func() { f.E.Now = old })
}

// outageIncident is the send's open outage incident; it fails the test
// when there is none or it offers no way to continue.
func outageIncident(t *testing.T, f *sendtest.Fixture) *engine.LiveIncident {
	t.Helper()
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	in := snap.Incident
	if in == nil || in.Kind != "provider_outage" {
		f.Dump()
		t.Fatalf("no open outage incident: %+v", in)
	}
	keys := map[string]bool{}
	for _, o := range in.Detail.Options {
		keys[o.Key] = true
	}
	if !keys["retry"] || !keys["cancel_share"] {
		t.Fatalf("options %v, want retry and cancel_share", keys)
	}
	return in
}

// heldOnFault sends n Gmail readers through a fault provider that refuses
// every call until the outage path holds its share (after three minutes).
func heldOnFault(t *testing.T, f *sendtest.Fixture, n int) (uuid.UUID, time.Time) {
	t.Helper()
	gmailOnly(f, n)
	sendtest.Fault.Reset(sendtest.FaultRefuse)
	pid := f.UseFaultProvider(false)
	f.StartSend(engine.StartOptions{})
	at, _ := f.Drive(6 * time.Minute)
	if st, _ := routeOf(t, f, pid); st != "held" {
		f.Dump()
		t.Fatalf("route %s, want held", st)
	}
	return pid, at
}

// seedInbox is the demo workspace's connected seed inbox the test messages
// go to.
func seedInbox(t *testing.T, f *sendtest.Fixture) string {
	t.Helper()
	var email string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT email FROM placement_seeds WHERE workspace_id = $1 AND status = 'connected' ORDER BY email LIMIT 1`, f.WS).Scan(&email); err != nil {
		t.Fatal(err)
	}
	return email
}

// testHealthJobOutage: an outage the provider health job finds goes through
// the engine's outage path: the share is held with the engine's analysis
// (what the provider answered, from its health record) and its ways to
// continue. A hold whose incident offers none (from before) gets them.
func testHealthJobOutage(t *testing.T, f *sendtest.Fixture) {
	gmailOnly(f, 300)
	sendtest.Fault.Reset(sendtest.FaultOK)
	pid := f.UseFaultProvider(false)
	f.StartSend(engine.StartOptions{})
	at, _ := f.Drive(3 * time.Second)
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'sent'`, f.Campaign); n == 0 {
		t.Fatal("nothing was sent before the outage")
	}
	p, err := f.Env.App.Q.ProvGetAny(f.Ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 6 {
		ce := &providers.CallError{Op: "POST /send", Code: "ECONNREFUSED", Message: "ECONNREFUSED mta.example.net:443", Temporary: true}
		if err := manage.RecordCall(f.Ctx, f.Env.App.Pool, p, "POST /send", ce, 0, at.Add(time.Duration(i)*20*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	now := at.Add(2 * time.Minute)
	setClock(t, f, now)
	svc := &manage.Service{Pool: f.Env.App.Pool, Q: f.Env.App.Q, Audit: f.Env.App.Audit, URLs: f.Env.App.URLs,
		Factory: &manage.Factory{Box: f.Env.App.Secrets, HTTP: http.DefaultClient}, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Jobs: func() *jobs.Client { return f.Env.App.Jobs }, Sending: func() sending.Starter { return f.E }, Now: func() time.Time { return now }}
	down, err := svc.Check(f.Ctx, p, false)
	if err != nil {
		t.Fatal(err)
	}
	if down.Status != state.Down {
		t.Fatalf("provider %s, want down", down.Status)
	}
	if st, _ := routeOf(t, f, pid); st != "held" {
		f.Dump()
		t.Fatalf("route %s, want held by the health job", st)
	}
	in := outageIncident(t, f)
	if in.Detail.Reason != "provider_down" || len(in.Detail.Causes) == 0 || in.Held == 0 {
		t.Fatalf("incident %+v", in.Detail)
	}
	logged := false
	for _, l := range in.Detail.Log {
		logged = logged || strings.Contains(l.Text, "ECONNREFUSED mta.example.net:443")
	}
	if !logged || in.Detail.ErrorKind != "connection errors" {
		t.Fatalf("log %+v (%s), want what the provider answered", in.Detail.Log, in.Detail.ErrorKind)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was asked to decide")
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.share_held'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d share_held audit entries", n)
	}

	// A hold from before the health job used this path: no options.
	f.Exec(`UPDATE campaign_incidents SET detail = '{}' WHERE campaign_id = $1 AND resolved_at IS NULL`, f.Campaign)
	if err := f.E.HoldShares(f.Ctx, pid, "provider_down"); err != nil {
		t.Fatal(err)
	}
	outageIncident(t, f)
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND resolved_at IS NULL`, f.Campaign); n != 1 {
		t.Fatalf("%d open incidents, want the one", n)
	}
}

// testRetryAsksAgain: Retry keeps the share on hold and sends a test message
// to a seed inbox every minute; with the provider still down a person is
// asked again after 15 minutes, with the options, never left waiting.
func testRetryAsksAgain(t *testing.T, f *sendtest.Fixture) {
	pid, heldAt := heldOnFault(t, f, 200)
	setClock(t, f, heldAt)
	if err := f.E.RetryRoute(f.Ctx, f.Campaign, f.Sofia, pid); err != nil {
		t.Fatal(err)
	}
	st, d := routeOf(t, f, pid)
	if st != "held" || !strings.Contains(d, `"decision":"retry"`) || !strings.Contains(d, `"seed":true`) {
		t.Fatalf("route %s %s, want held and retried with a seed inbox", st, d)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident != nil {
		t.Fatalf("incident %+v still open after Retry", snap.Incident)
	}
	for _, p := range snap.Providers {
		if p.ID == pid.String() && (!p.Retrying || p.ProbesNeeded != 3 || !p.ProbeSeed) {
			t.Fatalf("provider %+v, want retrying with 3 test messages to a seed inbox", p)
		}
	}
	calls := sendtest.Fault.Calls()
	end, _, err := f.E.Drive(f.Ctx, f.Campaign, heldAt, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if probes := sendtest.Fault.Calls() - calls; probes < 15 || probes > 17 {
		t.Fatalf("%d test messages in 15 minutes, want one a minute", probes)
	}
	if asked := end.Sub(heldAt); asked < 15*time.Minute || asked > 17*time.Minute {
		t.Fatalf("asked again after %s, want 15 minutes", asked)
	}
	in := outageIncident(t, f)
	if !in.Detail.Retried || len(in.Detail.Causes) == 0 || in.Detail.Causes[0].Lead != "Retried for 15 minutes." {
		t.Fatalf("incident %+v, want asked again", in.Detail)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'send_paused' AND data->>'campaignId' = $1 AND title LIKE '%again%'`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was asked again")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status IN ('failed', 'rejected', 'sent')`, f.Campaign); n != 0 {
		t.Fatalf("%d messages left the queue while the provider was down", n)
	}
}

// testRetryResumes: the share resumes by itself after three test messages
// in a row went through, and the rest of it goes out once. With a seed
// inbox the tests go there; without one, one of the share's own messages
// goes every minute.
func testRetryResumes(t *testing.T, f *sendtest.Fixture, seed bool) {
	if !seed {
		f.Exec(`UPDATE placement_seeds SET status = 'unchecked' WHERE workspace_id = $1 AND status = 'connected'`, f.WS)
		t.Cleanup(func() {
			f.Exec(`UPDATE placement_seeds SET status = 'connected' WHERE workspace_id = $1 AND status = 'unchecked'`, f.WS)
		})
	}
	pid, heldAt := heldOnFault(t, f, 200)
	setClock(t, f, heldAt)
	if err := f.E.RetryRoute(f.Ctx, f.Campaign, f.Sofia, pid); err != nil {
		t.Fatal(err)
	}
	sendtest.Fault.SetMode(sendtest.FaultOK)
	// Two test messages go through (now and a minute later): still held.
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, heldAt, 90*time.Second); err != nil {
		t.Fatal(err)
	}
	st, d := routeOf(t, f, pid)
	if st != "held" || !strings.Contains(d, `"probes":2`) {
		f.Dump()
		t.Fatalf("route %s %s, want held after 2 test messages", st, d)
	}
	sent := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'sent'`, f.Campaign)
	if seed {
		if n := sendtest.Fault.TookFor(seedInbox(t, f)); n != 2 || sent != 0 {
			t.Fatalf("%d test messages to the seed inbox, %d sent, want 2 and 0", n, sent)
		}
	} else if sent != 2 {
		t.Fatalf("%d of the share's messages sent as tests, want 2", sent)
	}
	// The third resumes the share; the rest goes out.
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, heldAt.Add(90*time.Second), 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	if s := f.Status(); s != "sent" {
		f.Dump()
		t.Fatalf("campaign %s, want sent", s)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'sent'`, f.Campaign); n != 200 {
		t.Fatalf("%d sent, want 200", n)
	}
	if n := tookTwice(); n != 0 {
		t.Fatalf("%d messages delivered twice", n)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.share_resumed'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d share_resumed audit entries", n)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.share_retried'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d share_retried audit entries", n)
	}
}
