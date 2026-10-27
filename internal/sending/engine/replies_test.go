package engine_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testReplyRules: a mailbox provider response rule acts when the replies
// containing its text reach its count in its window (review GS-12): Yahoo
// replying TSS04 ten times in ten minutes pauses Yahoo for 2 hours while
// Gmail keeps sending; after the 2 hours Yahoo sends again by itself, and
// the rule counts only new replies.
func testReplyRules(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(
		sendtest.ContactSpec{N: 1500, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
		sendtest.ContactSpec{N: 1500, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
	)
	f.Exec(`DELETE FROM traffic_reply_rules WHERE workspace_id = $1`, f.WS)
	rule := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO traffic_reply_rules (id, workspace_id, mailbox_provider, contains, min_count, window_seconds, action, duration_seconds, enabled)
		VALUES ($1, $2, 'yahoo', 'TSS04', 10, 600, 'pause', 7200, true)`, rule, f.WS)
	defer f.Exec(`DELETE FROM traffic_reply_rules WHERE id = $1`, rule)
	f.StartSend(engine.StartOptions{})
	at, _ := f.Drive(time.Minute)
	setClock(t, f, at)
	f.Exec(`INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, provider_id, type, occurred_at, mailbox_provider, original_code, diagnostic)
		SELECT gen_random_uuid(), m.workspace_id, m.id, m.campaign_id, m.provider_id, 'deferred', $2, 'yahoo', '421 4.7.0',
		       '[TSS04] Messages from 192.0.2.1 temporarily deferred due to unexpected volume or user complaints'
		FROM messages m WHERE m.campaign_id = $1 AND m.mailbox_provider = 'yahoo' AND m.sent_at IS NOT NULL ORDER BY m.id LIMIT 10`, f.Campaign, at.Add(-30*time.Second))
	var err error
	if at, _, err = f.E.Drive(f.Ctx, f.Campaign, at, 20*time.Second); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_sends WHERE campaign_id = $1 AND holds ? 'yahoo'`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatal("Yahoo isn't paused by the TSS04 rule")
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'reply_rule' AND resolved_at IS NULL`, f.Campaign); n != 1 {
		t.Fatalf("%d open reply rule incidents, want 1", n)
	}
	if n := f.Count(`SELECT count(*) FROM traffic_reply_rules WHERE id = $1 AND last_campaign_id = $2 AND last_matched_at IS NOT NULL`, rule, f.Campaign); n != 1 {
		t.Fatal("the rule doesn't say it acted")
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.reply_rule'`, f.Campaign.String()); n != 1 {
		t.Fatal("not audited")
	}
	before := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'gmail' AND sent_at IS NOT NULL`, f.Campaign)
	if at, _, err = f.E.Drive(f.Ctx, f.Campaign, at, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if after := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'gmail' AND sent_at IS NOT NULL`, f.Campaign); after <= before {
		t.Fatal("Gmail stopped with Yahoo")
	}
	// Two hours later Yahoo sends again by itself.
	at = at.Add(2 * time.Hour)
	setClock(t, f, at)
	yahoo := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'yahoo' AND sent_at IS NOT NULL`, f.Campaign)
	if _, _, err = f.E.Drive(f.Ctx, f.Campaign, at, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'reply_rule' AND resolution = 'expired'`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatal("the pause didn't end by itself")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'yahoo' AND sent_at IS NOT NULL`, f.Campaign); n <= yahoo {
		t.Fatal("Yahoo didn't send again after the rule's 2 hours")
	}
}
