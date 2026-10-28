package engine_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testComplaintWatch: complaints keep arriving after a send ends. The send
// is watched for 72 hours; Outlook passing its complaint limit with enough
// evidence alerts the people who can resume sends, once, and they
// acknowledge it. Gmail's complaints read "not reported", never 0%.
func testComplaintWatch(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: 700, Mailbox: "outlook", Domain: "hotmail.com", Engaged: true},
		sendtest.ContactSpec{N: 300, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	end, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("send not finished: %s", f.Status())
	}
	var until *time.Time
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT watch_until FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&until); err != nil {
		t.Fatal(err)
	}
	if until == nil || until.Sub(end) < 71*time.Hour || until.Sub(end) > 73*time.Hour {
		t.Fatalf("watch until %v, want 72 h after the send ended at %v", until, end)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Watch == nil || !snap.Watch.Active {
		t.Fatalf("watch %+v", snap.Watch)
	}
	for _, m := range snap.Mailboxes {
		if m.Key == "gmail" && (m.ComplaintRate != nil || m.ComplaintReason != "not_reported") {
			t.Fatalf("Gmail's complaints %+v, want not reported", m)
		}
	}
	// A day later, Outlook readers complained.
	f.Exec(`UPDATE campaign_counters SET complained = 12 WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND sent > 0
		AND provider_id = (SELECT provider_id FROM campaign_counters WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND sent > 0 LIMIT 1)`, f.Campaign)
	setClock(t, f, end.Add(24*time.Hour))
	for range 2 {
		if err := f.E.WatchComplaints(f.Ctx); err != nil {
			t.Fatal(err)
		}
	}
	snap, err = f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	var alert *engine.LiveIncident
	for i := range snap.Incidents {
		if snap.Incidents[i].Kind == "complaint_watch" {
			if alert != nil {
				t.Fatal("alerted twice for Outlook")
			}
			alert = &snap.Incidents[i]
		}
	}
	if alert == nil || alert.Mailbox != "outlook" || alert.ResolvedAt != nil || len(alert.Detail.Causes) == 0 {
		t.Fatalf("alert %+v", alert)
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE kind = 'complaint_watch' AND data->>'campaignId' = $1`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was alerted")
	}
	// The alert names the mailbox provider and writes its rates as
	// percentages.
	var body string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT body FROM notifications WHERE kind = 'complaint_watch' AND data->>'campaignId' = $1 LIMIT 1`,
		f.Campaign.String()).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^Outlook complaints reached \d+\.\d+% \(limit 0\.10%\) after the send ended\.`).MatchString(body) {
		t.Fatalf("alert %q", body)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.complaint_alert'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d complaint alert audit entries", n)
	}
	if err := f.E.Acknowledge(f.Ctx, f.Campaign, f.Sofia, uuid.MustParse(alert.ID)); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'complaint_watch' AND resolution = 'acknowledged'`, f.Campaign); n != 1 {
		t.Fatal("alert not acknowledged")
	}
	// After the watch, nothing is watched any more.
	setClock(t, f, end.Add(73*time.Hour))
	if err := f.E.WatchComplaints(f.Ctx); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_sends WHERE campaign_id = $1 AND watch_until IS NOT NULL`, f.Campaign); n != 0 {
		t.Fatal("the watch didn't end")
	}
}

// testPostmasterWatch (review GS-6, R2.A4): Gmail reports no complaints
// per message, only a daily spam rate for the sending domain in Postmaster
// Tools. With Traffic rules › After the send › Postmaster Tools on, the
// watch alerts once when Gmail's rate on a day of the send passes the
// complaint limit; a day before the send, a rate under the limit or the
// setting off raise nothing, and the monitor shows the rate only while
// the setting is on.
func testPostmasterWatch(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 600, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{})
	end, done, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("send not finished: %s", f.Status())
	}
	var started time.Time
	var domain string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT c.started_at, split_part(s.email, '@', 2) FROM campaigns c JOIN senders s ON s.id = c.sender_id WHERE c.id = $1`,
		f.Campaign).Scan(&started, &domain); err != nil {
		t.Fatal(err)
	}
	// Gmail's days run on Pacific time: the first one that can hold the
	// send is the day of its start, 7 hours earlier.
	sendDay := started.UTC().Add(-7 * time.Hour).Truncate(24 * time.Hour)
	postmaster := func(day time.Time, rate float64) {
		f.Exec(`INSERT INTO postmaster_daily (workspace_id, domain, day, spam_rate) VALUES ($1, $2, $3::date, $4)
			ON CONFLICT (workspace_id, domain, day) DO UPDATE SET spam_rate = EXCLUDED.spam_rate`, f.WS, domain, day.Format(time.DateOnly), rate)
	}
	t.Cleanup(func() {
		f.Exec(`DELETE FROM postmaster_daily WHERE workspace_id = $1 AND domain = $2`, f.WS, domain)
		f.Exec(`UPDATE traffic_rules SET watch = watch - 'postmaster' WHERE workspace_id = $1`, f.WS)
	})
	setPostmaster := func(on bool) {
		v := `{"postmaster":false}`
		if on {
			v = `{"postmaster":true}`
		}
		f.Exec(`INSERT INTO traffic_rules (workspace_id, watch) VALUES ($1, $2::jsonb)
			ON CONFLICT (workspace_id) DO UPDATE SET watch = traffic_rules.watch || EXCLUDED.watch`, f.WS, v)
	}
	alerts := func() int64 {
		return f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'complaint_watch'`, f.Campaign)
	}
	watch := func(at time.Time) {
		t.Helper()
		setClock(t, f, at)
		if err := f.E.WatchComplaints(f.Ctx); err != nil {
			t.Fatal(err)
		}
	}

	// The day before the send was bad, and the send's day so far is fine.
	postmaster(sendDay.AddDate(0, 0, -1), 0.0050)
	postmaster(sendDay, 0.0008)
	watch(end.Add(6 * time.Hour))
	if n := alerts(); n != 0 {
		t.Fatalf("%d alerts from a day before the send or a rate under the limit", n)
	}
	// Gmail's report for the send's day passes the limit, with the
	// setting off: no alert, and the monitor doesn't show it.
	postmaster(sendDay, 0.0025)
	setPostmaster(false)
	watch(end.Add(20 * time.Hour))
	if n := alerts(); n != 0 {
		t.Fatalf("%d alerts with Postmaster Tools off", n)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Postmaster != nil {
		t.Fatalf("Postmaster Tools is off but the monitor shows %+v", snap.Postmaster)
	}
	// On: one alert, said as Gmail's rate for the domain on that day.
	setPostmaster(true)
	watch(end.Add(21 * time.Hour))
	watch(end.Add(22 * time.Hour))
	if n := alerts(); n != 1 {
		t.Fatalf("%d alerts, want 1", n)
	}
	snap, err = f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Postmaster == nil || snap.Postmaster.SpamRate != 0.25 {
		t.Fatalf("monitor's Postmaster line %+v, want 0.25%%", snap.Postmaster)
	}
	var alert *engine.LiveIncident
	for i := range snap.Incidents {
		if snap.Incidents[i].Kind == "complaint_watch" {
			alert = &snap.Incidents[i]
		}
	}
	if alert == nil || alert.Mailbox != "gmail" || alert.Detail.Reason != "postmaster" || alert.Value == nil || *alert.Value != 0.25 || len(alert.Detail.Causes) == 0 ||
		!strings.Contains(alert.Detail.Causes[0].Body, "Postmaster Tools reports a 0.25% spam rate for "+domain) {
		t.Fatalf("alert %+v", alert)
	}
	var body string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT body FROM notifications WHERE kind = 'complaint_watch' AND data->>'campaignId' = $1 LIMIT 1`,
		f.Campaign.String()).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^Gmail's spam rate for ` + regexp.QuoteMeta(domain) + ` reached 0\.25% on .+ \(limit 0\.10%\), from Postmaster Tools\.`).MatchString(body) {
		t.Fatalf("alert %q", body)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.complaint_alert' AND metadata->>'source' = 'Postmaster Tools'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d Postmaster alert audit entries", n)
	}
}
