package texting_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Review 2026-11 CH-2: an SMS send whose failed rate passes its threshold
// pauses itself, holds what is left, says why, tells the senders and is
// audited; one under the threshold goes on.
func TestSMSAutoPause(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := workspace(t, env)
	s, _, _ := service(env, time.Now)
	campaign := one[uuid.UUID](t, env, `SELECT id FROM campaigns WHERE name = 'Summit: 8 days to go'`)
	// The demo's sent campaign, back in its send: a few texts still held.
	for _, q := range []string{
		`UPDATE sms_sends SET phase = 'sending', paused_kind = '', paused_at = NULL WHERE campaign_id = $1`,
		`UPDATE channel_messages SET status = 'held' WHERE id IN (SELECT id FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign' AND status = 'delivered' ORDER BY id LIMIT 20)`,
	} {
		if _, err := env.App.Pool.Exec(ctx, q, campaign); err != nil {
			t.Fatal(err)
		}
	}
	phase := func() (string, string) {
		var p, k string
		if err := env.App.Pool.QueryRow(ctx, `SELECT phase, coalesce(paused_kind, '') FROM sms_sends WHERE campaign_id = $1`, campaign).Scan(&p, &k); err != nil {
			t.Fatal(err)
		}
		return p, k
	}
	// About 1.6% failed: under the threshold, the send goes on.
	s.CheckAutoPause(ctx, w, campaign)
	if p, _ := phase(); p != "sending" {
		t.Fatalf("paused under the threshold: %s", p)
	}
	// 100 more failures: over it.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'failed', reason = '' WHERE id IN (
		SELECT id FROM channel_messages WHERE campaign_id = $1 AND kind = 'campaign' AND status = 'delivered' ORDER BY id LIMIT 100)`, campaign); err != nil {
		t.Fatal(err)
	}
	s.CheckAutoPause(ctx, w, campaign)
	if p, k := phase(); p != "paused" || k != "auto" {
		t.Fatalf("over the threshold: %s %s", p, k)
	}
	var audited, notified, hooked int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'campaign.auto_paused' AND target_id = $1`, campaign.String()).Scan(&audited)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'sms_send_paused'`).Scan(&notified)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries d JOIN webhook_events e ON e.id = d.event_id WHERE e.type = 'campaign.paused'`).Scan(&hooked)
	if audited != 1 || notified == 0 {
		t.Fatalf("audited %d, notified %d", audited, notified)
	}
	// A second check on a paused send changes nothing.
	s.CheckAutoPause(ctx, w, campaign)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'campaign.auto_paused' AND target_id = $1`, campaign.String()).Scan(&audited)
	if audited != 1 {
		t.Fatalf("paused twice: %d", audited)
	}
}

// Review 2026-11 CH-2: the SMS sweep clears texts' bodies after the
// workspace's retention and removes delivery detail after 90 days, keeping
// texts still waiting.
func TestSMSRetentionSweep(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	s, _, _ := service(env, time.Now)
	campaign := one[uuid.UUID](t, env, `SELECT id FROM campaigns WHERE name = 'Summit: 8 days to go'`)
	ids := func(q string) []uuid.UUID {
		rows, err := env.App.Pool.Query(ctx, q, campaign)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			out = append(out, id)
		}
		return out
	}
	// Texts whose short link no longer works (a working link keeps its text).
	sample := ids(`SELECT m.id FROM channel_messages m WHERE m.campaign_id = $1 AND m.status = 'delivered'
		AND NOT EXISTS (SELECT 1 FROM sms_links l WHERE l.message_id = m.id AND l.expires_at > now()) ORDER BY m.id LIMIT 3`)
	if len(sample) != 3 {
		t.Fatalf("sample: %v", sample)
	}
	old, ancient, waiting := sample[0], sample[1], sample[2]
	for _, q := range []struct {
		sql string
		id  uuid.UUID
	}{
		{`UPDATE channel_messages SET body = 'Doors open at 8:30', created_at = now() - interval '45 days' WHERE id = $1`, old},
		{`UPDATE channel_messages SET created_at = now() - interval '120 days' WHERE id = $1`, ancient},
		{`UPDATE channel_messages SET body = 'Still waiting', status = 'queued', created_at = now() - interval '120 days' WHERE id = $1`, waiting},
	} {
		if _, err := env.App.Pool.Exec(ctx, q.sql, q.id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	var body *string
	if err := env.App.Pool.QueryRow(ctx, `SELECT body FROM channel_messages WHERE id = $1`, old).Scan(&body); err != nil || body != nil {
		t.Fatalf("a 45-day-old text keeps its body past the 30-day retention: %v %v", body, err)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE id = $1`, ancient).Scan(&n)
	if n != 0 {
		t.Fatal("a 120-day-old text stays past the 90 days")
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT body FROM channel_messages WHERE id = $1`, waiting).Scan(&body); err != nil || body == nil || *body != "Still waiting" {
		t.Fatalf("a text still waiting lost its body: %v %v", body, err)
	}
}
