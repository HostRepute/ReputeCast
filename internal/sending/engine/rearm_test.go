package engine_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// outlookTrip sends to 2,500 at-risk Outlook imports and 600 engaged Gmail
// readers with the Outlook complaints scenario until the guard trips. It
// returns where the clock stopped.
func outlookTrip(t *testing.T, f *sendtest.Fixture) time.Time {
	t.Helper()
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: 2500, Mailbox: "outlook", Domain: "hotmail.com", Tier: "at_risk", Source: "import:mailchimp"},
		sendtest.ContactSpec{N: 600, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{Scenario: "outlook-complaints"})
	at, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'threshold'`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatalf("%d threshold incidents, want the one that tripped", n)
	}
	return at
}

// guardOf is the send's guard state.
func guardOf(t *testing.T, f *sendtest.Fixture) map[string]any {
	t.Helper()
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT guard FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var g map[string]any
	_ = json.Unmarshal(raw, &g)
	return g
}

// testResumeRearms: resuming a mailbox provider as is re-arms its threshold
// from the current counts. The complaints that tripped it, and those still
// arriving for mail delivered before the pause, don't pause it again.
func testResumeRearms(t *testing.T, f *sendtest.Fixture) {
	at := outlookTrip(t, f)
	setClock(t, f, at)
	if _, err := f.E.ResumeScope(f.Ctx, f.Campaign, f.Sofia, engine.ResumeRequest{Mailbox: "outlook"}); err != nil {
		t.Fatal(err)
	}
	g := guardOf(t, f)
	if rearm, _ := g["rearm"].(map[string]any); rearm["mailbox:outlook"] == nil || rearm["campaign"] == nil {
		t.Fatalf("guard %v, want Outlook re-armed and its events left out of the campaign-wide check", g)
	}
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'threshold'`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatalf("%d threshold incidents: resuming tripped again on old complaints", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND status = 'sent' AND sent_at > $2`, f.Campaign, at); n == 0 {
		t.Fatal("Outlook didn't send after resuming")
	}
}

// testCampaignResumeRearms: with the campaign scope a trip pauses the whole
// send; resuming it re-arms every threshold from the current counts, with
// a raised limit audited.
func testCampaignResumeRearms(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET thresholds = '{"scope":"campaign"}' WHERE id = $1`, f.Campaign)
	at := outlookTrip(t, f)
	if s := f.Status(); s != "paused" {
		t.Fatalf("campaign %s, want paused by the campaign-wide trip", s)
	}
	setClock(t, f, at)
	limit := 0.5
	if _, err := f.E.ResumeScope(f.Ctx, f.Campaign, f.Sofia, engine.ResumeRequest{Limit: &limit}); err != nil {
		t.Fatal(err)
	}
	var th struct {
		Complaint float64 `json:"complaint"`
	}
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT thresholds FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(raw, &th)
	if th.Complaint != 0.5 {
		t.Fatalf("complaint limit %.2f, want raised to 0.5", th.Complaint)
	}
	if rearm, _ := guardOf(t, f)["rearm"].(map[string]any); rearm["campaign"] == nil {
		t.Fatal("the campaign-wide thresholds didn't re-arm")
	}
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'threshold'`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatalf("%d threshold incidents: resuming tripped again on old complaints", n)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.resumed' AND metadata ? 'thresholds_after'`, f.Campaign.String()); n != 1 {
		t.Fatal("the raised limit isn't in the resume's audit entry")
	}
}

// testRouteResumeWithChanges: a sending provider paused for its bounces
// resumes with the recommended changes: it sends slower for the rest of
// the send, and its threshold re-arms.
func testRouteResumeWithChanges(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: 750, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 750, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
	)
	sendtest.Fault.Reset(sendtest.FaultOK)
	pid := f.UseFaultProvider(false)
	f.StartSend(engine.StartOptions{})
	at, _ := f.Drive(20 * time.Second)
	// Its bounces tripped the route's threshold: 5% of 600 sent, spread
	// over Gmail and Yahoo (neither has enough mail to trip alone).
	f.Exec(`UPDATE campaign_counters SET sent = 300, hard_bounced = 15 WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, pid)
	f.Exec(`UPDATE campaign_routes SET status = 'paused', status_detail = '{"reason":"threshold"}' WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, pid)
	f.Exec(`INSERT INTO campaign_incidents (id, campaign_id, kind, scope, provider_id, metric, value, threshold, held, detail, tripped_at)
		VALUES ($1, $2, 'threshold', 'route', $3, 'bounce', 10, 2, 900, '{"changes":{"rate":2}}', $4)`, uuid.Must(uuid.NewV7()), f.Campaign, pid, at)
	setClock(t, f, at)
	res, err := f.E.ResumeScope(f.Ctx, f.Campaign, f.Sofia, engine.ResumeRequest{ProviderID: &pid, WithChanges: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Rate != 2 {
		t.Fatalf("resumed at %.0f /s, want the recommended 2 /s", res.Rate)
	}
	g := guardOf(t, f)
	rates, _ := g["routeRates"].(map[string]any)
	rearm, _ := g["rearm"].(map[string]any)
	if rates[pid.String()] != 2.0 || rearm["route:"+pid.String()] == nil {
		t.Fatalf("guard %v, want the route slowed and re-armed", g)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND resolution = 'resumed_with_changes'`, f.Campaign); n != 1 {
		t.Fatal("incident not resolved as resumed with changes")
	}
	before := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'sent'`, f.Campaign)
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	sent := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'sent'`, f.Campaign) - before
	if sent == 0 || sent > 30 {
		t.Fatalf("%d sent in 10 s, want about 2 /s", sent)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'threshold'`, f.Campaign); n != 1 {
		t.Fatalf("%d threshold incidents: the route tripped again on its old bounces", n)
	}
}

// testThresholdsWhilePaused: a person who can send changes a waiting send's
// thresholds; the change is checked and audited, and others can't.
func testThresholdsWhilePaused(t *testing.T, f *sendtest.Fixture) {
	at := outlookTrip(t, f)
	setClock(t, f, at)
	path := "/workspaces/acme-newsletter/campaigns/" + f.Campaign.String() + "/live/thresholds"
	viewer := f.Env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("PUT", path, map[string]any{"mailboxComplaint": 0.2}).MustStatus(t, 403)
	linh := f.Env.Client()
	linh.SignInDemo("linh@acme.io")
	linh.Do("PUT", path, map[string]any{"mailboxComplaint": 9}).MustStatus(t, 422)
	linh.Do("PUT", path, map[string]any{"mailboxComplaint": 0.2, "deferral": 8}).MustStatus(t, 200)
	var raw []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT thresholds FROM campaign_sends WHERE campaign_id = $1`, f.Campaign).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var th struct {
		MailboxComplaint float64 `json:"mailbox_complaint"`
		Deferral         float64 `json:"deferral"`
	}
	_ = json.Unmarshal(raw, &th)
	if th.MailboxComplaint != 0.2 || th.Deferral != 8 {
		t.Fatalf("thresholds %s", raw)
	}
	var before, after []byte
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT before, after FROM audit_events WHERE target_id = $1 AND action = 'campaign.thresholds_changed'`, f.Campaign.String()).Scan(&before, &after); err != nil {
		t.Fatalf("not audited: %v", err)
	}
	if string(before) == string(after) {
		t.Fatal("the audit entry doesn't show what changed")
	}
}
