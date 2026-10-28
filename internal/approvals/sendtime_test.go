package approvals_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

// SR-3 (C3): a scheduled send is held on what nobody confirmed when they
// scheduled it (a new warning, the audience changed by more than 10%, an
// edit to the content, sender or route), a person confirms it again on the
// fresh count, and a hold nobody reviews expires back to Draft.
func TestSendTimeRecheckHolds(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	ctx := context.Background()
	n := counter{t, env}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	recheck := app.SendRecheck(env.App)
	digest := campaignID(t, linh, "Weekly Digest #142")
	id := uuid.MustParse(digest)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	holds := func() []string {
		t.Helper()
		r, err := recheck(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if r.Verdict != "blocked" {
			return nil
		}
		return r.Blocking
	}

	// No approval in the way: this is about what the person confirmed.
	var policy struct{ Policy map[string]any }
	linh.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	for _, k := range []string{"warmingProvider", "readyWithWarnings", "authorLacksSend", "approvalLists"} {
		policy.Policy[k] = false
	}
	policy.Policy["audienceMin"] = map[string]any{"enabled": false, "n": 10000}
	linh.Do("PUT", ws+"/approval-policy", policy.Policy).MustStatus(t, 200)
	exec(`UPDATE approval_requests SET status = 'cancelled' WHERE campaign_id = $1 AND status IN ('pending', 'approved')`, id)

	// Scheduled at its own time on a fresh run, confirming its warnings
	// and typing the count.
	linh.Do("POST", ws+"/campaigns/"+digest+"/unschedule", nil).MustStatus(t, 204)
	readinesstest.Post(t, env, linh, ws+"/campaigns/"+digest+"/readiness").MustStatus(t, 200)
	r := getReview(t, linh, digest)
	confirmed := r.Run.Estimates.Audience.Final
	schedule := func(r review, confirm int64) *apptest.Response {
		return linh.Do("POST", ws+"/campaigns/"+digest+"/send", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r), "confirm": fmt.Sprint(confirm)})
	}
	var sent struct {
		Status      string
		ScheduledAt *time.Time
	}
	schedule(r, confirmed).MustStatus(t, 200).JSON(t, &sent)
	if sent.Status != "scheduled" || sent.ScheduledAt == nil {
		t.Fatalf("scheduled: %+v", sent)
	}
	at := *sent.ScheduledAt
	var kept int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT recipients FROM campaign_send_confirmations WHERE campaign_id = $1`, id).Scan(&kept); err != nil || kept != confirmed {
		t.Fatalf("what was confirmed: %d (%v), want %d", kept, err, confirmed)
	}
	// Nothing changed: the send may start.
	if h := holds(); h != nil {
		t.Fatalf("unchanged: %v", h)
	}

	// The audience grows by more than 10% overnight (an import).
	var list uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM lists WHERE name = 'Weekly Digest' AND workspace_id = (SELECT workspace_id FROM campaigns WHERE id = $1)`, id).Scan(&list); err != nil {
		t.Fatal(err)
	}
	grow := confirmed*12/100 + 50
	exec(`WITH n AS (SELECT gen_random_uuid() AS id, g FROM generate_series(1, $3::int) g),
		c AS (INSERT INTO contacts (id, workspace_id, email, status, tier, mailbox_provider)
			SELECT id, (SELECT workspace_id FROM lists WHERE id = $1), 'import.' || g || '@example.org', 'active', 'active', 'other' FROM n RETURNING id)
		INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at) SELECT $1, id, 'subscribed', $2 FROM c`, list, time.Now(), grow)
	if h := holds(); len(h) != 1 || !strings.HasPrefix(h[0], "The audience grew to ") || !strings.Contains(h[0], "since you reviewed") {
		t.Fatalf("growth: %v", h)
	}

	// The recheck before the send holds it and tells people; the review
	// shows why, both counts and when the hold expires.
	told := n.notes("linh@acme.io", "send_blocked")
	if err := app.ApprovalsTickAt(env.App, at.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n.notes("linh@acme.io", "send_blocked") != told+1 {
		t.Fatal("linh wasn't told about the hold")
	}
	// The notification says when a hold nobody reviews expires.
	var body string
	if err := env.App.Pool.QueryRow(ctx, `SELECT n.body FROM notifications n JOIN users u ON u.id = n.user_id WHERE u.email = 'linh@acme.io' AND n.kind = 'send_blocked'
		ORDER BY n.created_at DESC LIMIT 1`).Scan(&body); err != nil || !strings.Contains(body, "and goes back to Draft if it's still held at ") ||
		!strings.HasSuffix(body, ", 3 hours after its time.") {
		t.Fatalf("the hold notification says when it expires: %q (%v)", body, err)
	}
	var rv struct {
		Held *struct {
			Reason    string
			Holds     []struct{ Kind, Text string }
			Reviewed  *struct{ Recipients int64 }
			Rechecked *struct{ Recipients int64 }
			ExpiresAt *time.Time
		}
	}
	linh.Do("GET", ws+"/campaigns/"+digest+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	h := rv.Held
	if h == nil || len(h.Holds) == 0 || h.Holds[0].Kind != "audience_change" || h.Reviewed == nil || h.Reviewed.Recipients != confirmed ||
		h.Rechecked == nil || h.Rechecked.Recipients <= confirmed || h.ExpiresAt == nil || !h.ExpiresAt.Equal(at.Add(3*time.Hour)) {
		t.Fatalf("held: %+v", h)
	}

	// Confirming it again takes the fresh count, not the one reviewed.
	readinesstest.Work(t, env)
	r = getReview(t, linh, digest)
	fresh := r.Run.Estimates.Audience.Final
	if fresh <= confirmed {
		t.Fatalf("the review should show the fresh count: %d", fresh)
	}
	if p := schedule(r, confirmed).MustStatus(t, 422).Problem(t); !strings.Contains(fmt.Sprint(p), fmt.Sprint(fresh)) {
		t.Fatalf("the typed count must be the fresh one: %v", p)
	}
	schedule(r, fresh).MustStatus(t, 200)
	if h := holds(); h != nil {
		t.Fatalf("confirmed again: %v", h)
	}

	// Edits after scheduling: the content and the sender.
	exec(`UPDATE campaigns SET content_hash = content_hash || '-edited' WHERE id = $1`, id)
	if h := holds(); len(h) != 1 || h[0] != "The content changed after it was scheduled" {
		t.Fatalf("content edit: %v", h)
	}
	exec(`UPDATE campaigns SET content_hash = left(content_hash, length(content_hash) - 7) WHERE id = $1`, id)
	exec(`UPDATE campaigns SET sender_id = (SELECT s.id FROM senders s WHERE s.workspace_id = campaigns.workspace_id AND s.id <> campaigns.sender_id ORDER BY s.id LIMIT 1)
		WHERE id = $1`, id)
	if h := holds(); len(h) == 0 || !strings.Contains(strings.Join(h, " | "), "The sender changed after it was scheduled") {
		t.Fatalf("sender edit: %v", h)
	}

	// The audience edited after scheduling (R1.L3c): an exclusion in the
	// composer holds the send even when it moves the count by nothing.
	linh.Do("POST", ws+"/campaigns/"+digest+"/unschedule", nil).MustStatus(t, 204)
	readinesstest.Post(t, env, linh, ws+"/campaigns/"+digest+"/readiness").MustStatus(t, 200)
	r = getReview(t, linh, digest)
	schedule(r, r.Run.Estimates.Audience.Final).MustStatus(t, 200)
	if h := holds(); h != nil {
		t.Fatalf("confirmed with the new sender: %v", h)
	}
	var partners struct{ ID string }
	linh.Do("POST", ws+"/lists", map[string]any{"name": "Partners (excluded)"}).MustStatus(t, 201).JSON(t, &partners)
	linh.Do("PATCH", ws+"/campaigns/"+digest, map[string]any{"audience": map[string]any{"lists": []string{list.String()}, "excludeLists": []string{partners.ID}}}).
		MustStatus(t, 200)
	if h := holds(); len(h) != 1 || h[0] != "The audience changed after it was scheduled" {
		t.Fatalf("audience edit: %v", h)
	}

	// Nobody reviews the hold: three hours after its time it expires back
	// to Draft (its schedule kept), and people are told.
	if err := app.ApprovalsTickAt(env.App, at.Add(-5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	told = n.notes("linh@acme.io", "send_blocked")
	if err := app.ApprovalsTickAt(env.App, at.Add(3*time.Hour+time.Minute)); err != nil {
		t.Fatal(err)
	}
	var status, resolution string
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.status, coalesce((SELECT resolution FROM campaign_incidents WHERE campaign_id = c.id AND kind = 'recheck_blocked'
		ORDER BY tripped_at DESC LIMIT 1), '') FROM campaigns c WHERE c.id = $1`, id).Scan(&status, &resolution); err != nil {
		t.Fatal(err)
	}
	if status != "draft" || resolution != "expired" {
		t.Fatalf("expired: status %s, hold %q", status, resolution)
	}
	if n.audit("campaign.send_expired", digest) != 1 || n.notes("linh@acme.io", "send_blocked") != told+1 {
		t.Fatal("the expiry is audited and people are told")
	}
}
