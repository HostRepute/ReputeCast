package engine_test

import (
	"regexp"
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
