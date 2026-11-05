package campaigns_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// A member limited to some sender domains sends only from them, by every
// path that starts, schedules, moves or resumes a send, whoever picked the
// sender; their new drafts and copies start from a sender of theirs.
func TestScopedMemberSendsFromTheirDomainsOnly(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID, editor, digest uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, r.id, s.id FROM workspaces w
		JOIN roles r ON r.workspace_id = w.id AND r.key = 'editor'
		JOIN senders s ON s.workspace_id = w.id AND s.email = 'digest@news.acme.io'
		WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &editor, &digest); err != nil {
		t.Fatal(err)
	}
	// Kim may send from acme.io only (Marcus's sender), not news.acme.io.
	u := env.CreateUser(apptest.UserOpts{Email: "kim@contractor.dev", Name: "Kim Contractor"})
	scope, _ := json.Marshal(authz.Scope{SenderDomains: []string{"acme.io"}})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: editor, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	kim, linh := env.Client(), env.Client()
	kim.SignInDemo("kim@contractor.dev")
	linh.SignInDemo("linh@acme.io")
	refused := func(what string, r *apptest.Response) {
		t.Helper()
		b, _ := json.Marshal(r.MustStatus(t, 403).Problem(t))
		if !strings.Contains(string(b), "Your access is limited to some sender domains, and this campaign sends from news.acme.io.") {
			t.Fatalf("%s: %s", what, b)
		}
	}

	// A new draft and a copy start from acme.io, whatever the workspace's
	// default and the source's sender; Linh's copy keeps the source's.
	weekly := campaignID(t, env, "Weekly Digest #143")
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET sender_id = $2 WHERE id = $1`, weekly, digest); err != nil {
		t.Fatal(err)
	}
	var blank, copied, linhs detail
	kim.Do("POST", ws+"/campaigns", map[string]any{}).MustStatus(t, 201).JSON(t, &blank)
	kim.Do("POST", ws+"/campaigns", map[string]any{"fromCampaignId": weekly}).MustStatus(t, 201).JSON(t, &copied)
	linh.Do("POST", ws+"/campaigns", map[string]any{"fromCampaignId": weekly}).MustStatus(t, 201).JSON(t, &linhs)
	for what, d := range map[string]detail{"Kim's new draft": blank, "Kim's copy": copied} {
		if d.Sender == nil || d.Sender.Email != "marcus@acme.io" {
			t.Fatalf("%s starts from %+v", what, d.Sender)
		}
	}
	if linhs.Sender == nil || linhs.Sender.Email != "digest@news.acme.io" {
		t.Fatalf("Linh's copy keeps its sender: %+v", linhs.Sender)
	}

	// Sending a campaign from news.acme.io is refused, and so is letting
	// its mail go out again from the live monitor; stopping it isn't.
	refused("send", kim.Do("POST", ws+"/campaigns/"+weekly+"/send", map[string]any{"runId": uuid.NewString()}))
	live := ws + "/campaigns/" + weekly + "/live"
	refused("resume", kim.Do("POST", live+"/resume", map[string]any{}))
	refused("retry", kim.Do("POST", live+"/retry", map[string]any{"providerId": uuid.NewString()}))
	refused("move", kim.Do("POST", live+"/move", map[string]any{"fromProviderId": uuid.NewString(), "toProviderId": uuid.NewString()}))
	refused("send again", kim.Do("POST", live+"/in-doubt", map[string]any{"action": "resend"}))
	if r := kim.Do("POST", live+"/cancel", map[string]any{}); r.Status == 403 {
		t.Fatalf("cancelling isn't refused for the scope: %s", r.Body)
	}

	// An approved campaign is scheduled on the scheduler's behalf: their
	// scope applies, whoever approved it.
	digest142 := campaignID(t, env, "Weekly Digest #142")
	var sender string
	if err := env.App.Pool.QueryRow(ctx, `SELECT s.email FROM campaigns c JOIN senders s ON s.id = c.sender_id WHERE c.id = $1`, digest142).Scan(&sender); err != nil || sender != "digest@news.acme.io" {
		t.Fatalf("Weekly Digest #142 sends from %q: %v", sender, err)
	}
	linh.Do("POST", ws+"/campaigns/"+digest142+"/unschedule", nil).MustStatus(t, 204)
	var request string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM approval_requests WHERE campaign_id = $1 AND status = 'approved'`, digest142).Scan(&request); err != nil {
		t.Fatal(err)
	}
	refused("schedule as approved", kim.Do("POST", ws+"/approvals/"+request+"/schedule", map[string]any{"runId": uuid.NewString()}))

	// Moving a scheduled send to another time is scheduling it again.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET status = 'scheduled', scheduled_at = now() + interval '3 days' WHERE id = $1`, weekly); err != nil {
		t.Fatal(err)
	}
	future := time.Now().AddDate(0, 0, 10).Format(time.DateOnly)
	refused("reschedule", kim.Do("POST", ws+"/campaigns/"+weekly+"/reschedule", map[string]any{"mode": "at", "date": future, "time": "10:30", "timezone": "America/New_York"}))
}
