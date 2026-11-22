package automations_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// The actual starter must receive the policy's tag. With the old tag
// supplied explicitly, its timeout must still never revoke list consent.
func TestSunsetReengagement(t *testing.T) {
	for _, manual := range []bool{false, true} {
		name := "policy tag starts the starter"
		if manual {
			name = "timeout preserves subscriptions"
		}
		t.Run(name, func(t *testing.T) {
			env, c := signedIn(t, "linh@acme.io")
			ctx := context.Background()
			clk := &clock{t: time.Now().UTC().Truncate(24 * time.Hour).Add(17 * time.Hour)}
			svc := engine(env, clk)
			var created struct{ ID string }
			c.Do("POST", wsPath, map[string]any{"template": "re-engagement"}).MustStatus(t, 201).JSON(t, &created)
			var detail automations.AutomationDetail
			c.Do("GET", wsPath+"/"+created.ID, nil).MustStatus(t, 200).JSON(t, &detail)
			c.Do("POST", wsPath+"/"+created.ID+"/publish", map[string]any{"version": detail.DraftVersion, "migration": "finish"}).MustStatus(t, 200)
			who := addContact(t, env, "sunset@example.com", nil)
			for _, list := range []string{"Product Updates", "Weekly Digest"} {
				joinList(t, env, listID(t, env, list), who)
			}
			if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET created_at = $2::timestamptz - interval '200 days' WHERE id = $1`, who, clk.now()); err != nil {
				t.Fatal(err)
			}
			ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := env.App.Pool.Exec(ctx, `INSERT INTO paid_subscriptions (id, workspace_id, contact_id, mode, kind, status, started_at)
				VALUES ($1, $2, $3, 'live', 'stripe', 'active', $4)`, uuid.Must(uuid.NewV7()), ws.ID, who, clk.now()); err != nil {
				t.Fatal(err)
			}
			aud := env.App.Audience()
			aud.Now = clk.now
			if _, err := aud.RecomputeTiers(ctx, ws.ID); err != nil {
				t.Fatal(err)
			}
			if manual {
				if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tags = array_append(tags, $2) WHERE id = $1 AND NOT ($2 = ANY(tags))`, who, detail.Graph.Trigger().Data.Trigger.Tag); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := svc.Dispatch(ctx); err != nil {
				t.Fatal(err)
			}
			if n := countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, created.ID, who); n != 1 {
				t.Fatalf("policy entered sunset; starter runs = %d, want 1", n)
			}
			if _, err := svc.Tick(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := env.App.Pool.Exec(ctx, `UPDATE messages SET status = 'sent', sent_at = $3 WHERE automation_id = $1 AND contact_id = $2`, created.ID, who, clk.now()); err != nil {
				t.Fatal(err)
			}
			clk.advance(15 * 24 * time.Hour)
			if _, err := svc.Tick(ctx); err != nil {
				t.Fatal(err)
			}
			if n := countWhere(t, env, `SELECT count(*) FROM list_memberships WHERE contact_id = $1 AND status = 'subscribed'`, who); n != 2 {
				t.Errorf("after 15 days: subscribed lists = %d, want 2; silence is not an unsubscribe", n)
			}
			if n := countWhere(t, env, `SELECT count(*) FROM contacts WHERE id = $1 AND broadcasts_stopped_at IS NOT NULL`, who); n != 0 {
				t.Error("starter stopped broadcasts before the policy's 30 days")
			}
			clk.advance(15 * 24 * time.Hour)
			if _, err := aud.RecomputeTiers(ctx, ws.ID); err != nil {
				t.Fatal(err)
			}
			if n := countWhere(t, env, `SELECT count(*) FROM contacts WHERE id = $1 AND broadcasts_stopped_at IS NOT NULL AND status = 'active'`, who); n != 1 {
				t.Error("at 30 days only broadcasts should stop, leaving the contact active")
			}
			if n := countWhere(t, env, `SELECT count(*) FROM list_memberships WHERE contact_id = $1 AND status = 'subscribed'`, who); n != 2 {
				t.Errorf("policy changed list subscriptions: %d, want 2", n)
			}
			if n := countWhere(t, env, `SELECT count(*) FROM suppressions WHERE workspace_id = $1 AND email = 'sunset@example.com'`, ws.ID); n != 0 {
				t.Error("sunset created a global suppression")
			}
			if n := countWhere(t, env, `SELECT count(*) FROM paid_subscriptions WHERE contact_id = $1 AND earn_paid_access(kind, status, ended_at, access_paused, paused_at)`, who); n != 1 {
				t.Error("sunset changed paid access")
			}
		})
	}
}

func TestSunsetStarterRecognizesHumanEngagement(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now().UTC().Truncate(24 * time.Hour).Add(17 * time.Hour)}
	svc := engine(env, clk)
	ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ ID string }
	c.Do("POST", wsPath, map[string]any{"template": "re-engagement"}).MustStatus(t, 201).JSON(t, &created)
	var d automations.AutomationDetail
	c.Do("GET", wsPath+"/"+created.ID, nil).MustStatus(t, 200).JSON(t, &d)
	c.Do("POST", wsPath+"/"+created.ID+"/publish", map[string]any{"version": d.DraftVersion, "migration": "finish"}).MustStatus(t, 200)
	var body string
	if err := env.App.Pool.QueryRow(ctx, `SELECT v.html FROM automation_email_versions v JOIN automation_emails e ON e.id = v.email_id WHERE e.automation_id = $1`, created.ID).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "cta_url") || !strings.Contains(body, "/p/acme-newsletter/archive") {
		t.Fatal("starter must have a real archive link, not an unset CTA token")
	}
	if strings.Contains(body, "workspace_name") || strings.Contains(body, "footer_address") || !strings.Contains(body, "500 Market St, Suite 1200") {
		t.Fatal("starter must have the workspace's editable footer, not unset recipient variables")
	}
	people := map[string]uuid.UUID{}
	for _, kind := range []string{"open", "click", "reply", "auto", "machine", "scanner", "quiet"} {
		id := addContact(t, env, kind+"@sunset.example", nil)
		people[kind] = id
		joinList(t, env, listID(t, env, "Product Updates"), id)
		if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET created_at = $2::timestamptz - interval '200 days' WHERE id = $1`, id, clk.now()); err != nil {
			t.Fatal(err)
		}
	}
	aud := env.App.Audience()
	aud.Now = clk.now
	if _, err := aud.RecomputeTiers(ctx, ws.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE messages SET status = 'sent', sent_at = $2 WHERE automation_id = $1`, created.ID, clk.now()); err != nil {
		t.Fatal(err)
	}
	clk.advance(5 * time.Minute)
	tr := env.App.SendingEngine().Recorder.(*tracking.Service)
	const chrome = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"
	for kind, id := range people {
		var mid uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM messages WHERE automation_id = $1 AND contact_id = $2`, created.ID, id).Scan(&mid); err != nil {
			t.Fatal(err)
		}
		switch kind {
		case "reply", "auto":
			receiveReply(t, env, mid, id, created.ID, kind == "auto", clk.now())
		case "open", "click", "scanner", "machine":
			e := sending.Engagement{MessageID: mid, Click: kind == "click" || kind == "scanner", UserAgent: chrome, IP: "198.51.100.41", Method: "GET", At: clk.now()}
			if kind == "machine" {
				e.UserAgent = "Mozilla/5.0" // Apple Mail Privacy Protection prefetch.
			}
			events := []sending.Engagement{e}
			if kind == "scanner" {
				for i := 1; i < 3; i++ {
					e.Link, e.At = i, clk.now().Add(time.Duration(i)*time.Second)
					events = append(events, e)
				}
			}
			if err := tr.RecordEngagement(ctx, events); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if _, status, _ := run(t, env, created.ID, people["click"]); status != "waiting" {
		t.Fatalf("pending click counted as engagement: %s", status)
	}
	clk.advance(3 * time.Minute)
	if _, err := tr.SettleEngagement(ctx, clk.now()); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	for kind, id := range people {
		_, status, _ := run(t, env, created.ID, id)
		want := "waiting"
		switch kind {
		case "open":
			want = "completed"
		case "click", "reply":
			want = "goal"
		}
		if status != want {
			t.Errorf("%s: status=%s, want %s", kind, status, want)
		}
	}
	if _, err := aud.RecomputeTiers(ctx, ws.ID); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"open", "click", "scanner", "machine"} {
		want := "sunset"
		if kind == "open" || kind == "click" {
			want = "active"
		}
		if n := countWhere(t, env, `SELECT count(*) FROM contacts WHERE id = $1 AND tier = $2`, people[kind], want); n != 1 {
			t.Errorf("%s: want tier %s", kind, want)
		}
	}
	// A later Sunset entry starts once more, but repeated nightly runs do not.
	clk.advance(182 * 24 * time.Hour)
	for range 2 {
		if _, err := aud.RecomputeTiers(ctx, ws.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Dispatch(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, created.ID, people["click"]); n != 2 {
		t.Errorf("second sunset entry: runs=%d, want 2", n)
	}
}

func TestLegacySunsetWarnsWithoutRewriting(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	var id string
	var before string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT a.id, v.graph::text FROM automations a JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.live_version WHERE a.name = 'Re-engagement'`).Scan(&id, &before); err != nil {
		t.Fatal(err)
	}
	var d automations.AutomationDetail
	c.Do("GET", wsPath+"/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	for _, code := range []string{"sunset_trigger", "sunset_unsubscribe"} {
		found := false
		for _, issue := range d.Issues {
			found = found || issue.Code == code && issue.Severity == "warning"
		}
		if !found {
			t.Errorf("missing %s warning: %+v", code, d.Issues)
		}
	}
	var after string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT graph::text FROM automation_versions WHERE automation_id = $1 AND version = $2`, id, d.LiveVersion).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Error("reading warnings rewrote the published graph")
	}
	// Repairing a draft does not hide the live version's warning.
	d.Graph.Trigger().Data.Trigger.Tag = "sunset"
	for i := range d.Graph.Nodes {
		n := &d.Graph.Nodes[i]
		if n.Type == automations.NodeList && n.Data.Action != nil && n.Data.Action.Kind == "unsubscribe_all" {
			n.Type, n.Data.Action = automations.NodeExit, nil
		}
	}
	var saved automations.AutomationDraftSaved
	c.Do("PUT", wsPath+"/"+id+"/draft", map[string]any{"graph": d.Graph, "revision": d.Revision}).MustStatus(t, 200).JSON(t, &saved)
	c.Do("GET", wsPath+"/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	for _, issues := range [][]automations.AutomationGraphIssue{saved.Issues, d.Issues} {
		if len(issues) != 1 || issues[0].Code != "sunset_live" || issues[0].Severity != "warning" {
			t.Errorf("fixed draft must still warn about the live version: %+v", issues)
		}
	}
	var preview automations.AutomationPublishPreview
	c.Do("GET", wsPath+"/"+id+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &preview)
	found := false
	for _, check := range preview.Checks {
		found = found || check.Status == "warn" && strings.Contains(check.Label, "published version still")
	}
	if !found {
		t.Error("publish preview omitted the live version warning")
	}
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT graph::text FROM automation_versions WHERE automation_id = $1 AND version = $2`, id, d.LiveVersion).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Error("saving a draft rewrote the published graph")
	}
}
