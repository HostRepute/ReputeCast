package engine_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// The guard on small sends and on block/policy bounces (review R2.L7c,
// GS-11): the floors (5 events, 500 sent or delivered) keep a rate from
// tripping on a handful of events, but complaints and block/policy bounces
// that are clearly over their limit trip at any volume, and a burst of
// block/policy bounces at one mailbox provider pauses it even when its
// whole-send rate is still diluted.

// smallSend starts a send to Outlook and Gmail (which keeps going) and
// drives it for a while; it returns where the clock stopped.
func smallSend(t *testing.T, f *sendtest.Fixture, outlook, gmail int, d time.Duration) time.Time {
	t.Helper()
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: outlook, Mailbox: "outlook", Domain: "hotmail.com", Engaged: true},
		sendtest.ContactSpec{N: gmail, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	at, _ := f.Drive(d)
	setClock(t, f, at)
	return at
}

// outlookIncident is the send's threshold incident at Outlook.
func outlookIncident(t *testing.T, f *sendtest.Fixture, metric string) *engine.LiveIncident {
	t.Helper()
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	for i := range snap.Incidents {
		if in := &snap.Incidents[i]; in.Kind == "threshold" && in.Metric == metric && in.Mailbox == "outlook" {
			return in
		}
	}
	f.Dump()
	t.Fatalf("no %s incident at Outlook: %+v", metric, snap.Incidents)
	return nil
}

// testSmallSendComplaints: 10 complaints among the first ~150 delivered to
// Outlook (far under the 500 delivered the floors need) pause Outlook, and
// the incident says it tripped clearly over the limit; Gmail keeps going.
func testSmallSendComplaints(t *testing.T, f *sendtest.Fixture) {
	at := smallSend(t, f, 400, 3000, 40*time.Second)
	delivered := f.Count(`SELECT coalesce(sum(delivered), 0) FROM campaign_counters WHERE campaign_id = $1 AND mailbox_provider = 'outlook'`, f.Campaign)
	if delivered == 0 || delivered >= 500 {
		f.Dump()
		t.Fatalf("%d delivered to Outlook, want a small send's first few hundred", delivered)
	}
	f.Exec(`UPDATE campaign_counters SET complained = 10 WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND delivered > 0
		AND provider_id = (SELECT provider_id FROM campaign_counters WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND delivered > 0 LIMIT 1)`, f.Campaign)
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 15*time.Second); err != nil {
		t.Fatal(err)
	}
	in := outlookIncident(t, f, "complaint")
	if !in.Detail.Clear || in.Detail.Events != 10 || in.Detail.Base >= 500 {
		t.Fatalf("incident detail %+v, want 10 complaints clearly over the limit on a small base", in.Detail)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_sends WHERE campaign_id = $1 AND holds ? 'outlook'`, f.Campaign); n != 1 {
		t.Fatal("Outlook isn't held")
	}
	if s := f.Status(); s != "sending" {
		t.Fatalf("campaign %s, want Gmail still sending", s)
	}
}

// testSmallSendWatch: after a send of 300 to Outlook, 3 late complaints
// (1%) alert the people who can resume sends, though 300 is under the
// floors' 500 delivered.
func testSmallSendWatch(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	limitsOutOfReach(f)
	f.AddContacts(sendtest.ContactSpec{N: 300, Mailbox: "outlook", Domain: "hotmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{})
	end, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("send not finished: %s", f.Status())
	}
	defaultLimits(f)
	setClock(t, f, end.Add(12*time.Hour))
	f.Exec(`UPDATE campaign_counters SET complained = 2 WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND delivered > 0
		AND provider_id = (SELECT provider_id FROM campaign_counters WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND delivered > 0 LIMIT 1)`, f.Campaign)
	if err := f.E.WatchComplaints(f.Ctx); err != nil {
		t.Fatal(err)
	}
	alerts := func(scope string) int64 {
		return f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'complaint_watch' AND scope LIKE $2`, f.Campaign, scope)
	}
	if n := alerts("%"); n != 0 {
		t.Fatalf("%d alerts for 2 complaints", n)
	}
	f.Exec(`UPDATE campaign_counters SET complained = 3 WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND complained = 2`, f.Campaign)
	for range 2 {
		if err := f.E.WatchComplaints(f.Ctx); err != nil {
			t.Fatal(err)
		}
	}
	// Outlook's alert, once; the whole send, which is Outlook's mail, doesn't
	// alert again for the same complaints.
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'complaint_watch' AND mailbox_provider = 'outlook'`, f.Campaign); n != 1 {
		t.Fatalf("%d alerts for Outlook's 3 complaints in 300 delivered, want 1", n)
	}
	if n := alerts("campaign"); n != 0 {
		t.Fatalf("%d campaign-wide alerts for the complaints Outlook's alert told", n)
	}
}

// faultOnly sends the fixture's campaign through the fault provider, which
// takes the mail and reports nothing of its own: the demo's simulated
// providers bounce about 0.13% of Outlook's mail on the sender's
// reputation, a block too, and some readers complain, at random, so a
// scenario that counts blocks would count theirs some runs and not others.
func faultOnly(t *testing.T, f *sendtest.Fixture) {
	t.Helper()
	sendtest.Fault.Reset(sendtest.FaultOK)
	f.UseFaultProvider(false)
}

// testBlockPolicyFloor: block/policy bounces trip on less evidence than
// unknown users (bytype.go). Three among the first ~150 sent to Outlook
// pause it at once; a burst, 8 in the last minutes while Outlook's
// whole-send rate is still under its limit, pauses it too.
func testBlockPolicyFloor(t *testing.T, f *sendtest.Fixture) {
	faultOnly(t, f)
	at := smallSend(t, f, 400, 400, 30*time.Second)
	blocks(t, f, at, 3, at.Add(-20*time.Second))
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 15*time.Second); err != nil {
		t.Fatal(err)
	}
	in := outlookIncident(t, f, "block_bounce")
	if !in.Detail.Clear || in.Detail.Window != 0 || in.Detail.Events != 3 || in.Detail.Base >= 500 ||
		!strings.Contains(in.Detail.Note, "refusing the sender itself") {
		t.Fatalf("incident detail %+v, want 3 block/policy bounces clearly over the limit", in.Detail)
	}
}

// testBlockBurst: Outlook took 1,800 messages over half an hour without a
// block, then refused 8 of the last ten minutes' 600: 0.44% of the whole
// send is under the 0.5% limit, but the burst pauses Outlook.
func testBlockBurst(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	faultOnly(t, f)
	f.AddContacts(sendtest.ContactSpec{N: 2400, Mailbox: "outlook", Domain: "hotmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{})
	f.Exec(`UPDATE campaign_sends SET throttles = '{"outlook":1}' WHERE campaign_id = $1`, f.Campaign)
	at, _ := f.Drive(30 * time.Minute)
	setClock(t, f, at)
	sent := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign)
	recent := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at >= $2`, f.Campaign, at.Add(-10*time.Minute))
	if sent < 1500 || recent > 700 {
		f.Dump()
		t.Fatalf("%d sent, %d in the last 10 minutes: want a long send at about 1 /s", sent, recent)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'threshold'`, f.Campaign); n != 0 {
		t.Fatalf("%d threshold incidents before the burst", n)
	}
	blocks(t, f, at, 8, at.Add(-3*time.Minute))
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 15*time.Second); err != nil {
		t.Fatal(err)
	}
	in := outlookIncident(t, f, "block_bounce")
	if in.Detail.Window != 10 || in.Detail.Events != 8 || in.Detail.Base > 700 || !in.Detail.Clear ||
		!strings.Contains(in.Detail.Note, "within 10 minutes") {
		t.Fatalf("incident detail %+v, want the burst of 8 in the last 10 minutes", in.Detail)
	}
}

// blocks records n policy-block bounces for messages sent to Outlook
// lately, at a time.
func blocks(t *testing.T, f *sendtest.Fixture, at time.Time, n int, when time.Time) {
	t.Helper()
	f.Exec(`INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, provider_id, type, occurred_at, mailbox_provider, original_code, bounce_class)
		SELECT gen_random_uuid(), m.workspace_id, m.id, m.campaign_id, m.provider_id, 'hard_bounce', $3, 'outlook', '550 5.7.1', 'policy_block'
		FROM messages m WHERE m.campaign_id = $1 AND m.mailbox_provider = 'outlook' AND m.sent_at IS NOT NULL AND m.sent_at <= $2
		ORDER BY m.sent_at DESC LIMIT $4`, f.Campaign, at, when, n)
}
