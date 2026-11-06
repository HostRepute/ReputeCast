package push_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/push"
	"github.com/hostrepute/reputecast/internal/store"
)

// A member limited to some lists can't move a scheduled push campaign that
// reaches every subscriber of the sites, as they couldn't send it; the
// owner still can.
func TestScopedMemberCantMovePushBeyondTheirLists(t *testing.T) {
	f := newFixture(t)
	env := f.env
	ctx := context.Background()
	f.pushOn(t, 5)
	f.subscribe(t, "chrome", chromeUA, nil)
	var c push.PushCampaign
	f.owner.Do(http.MethodPost, f.path+"/push-campaigns", map[string]any{"name": "X22 everyone", "title": "Later",
		"message": "How we ship to 2% of accounts first.", "link": "help.x22.test/blog/flags"}).MustStatus(t, 201).JSON(t, &c)
	day := time.Now().AddDate(0, 0, 10)
	var rv push.PushReview
	f.owner.Do(http.MethodPut, f.path+"/push-campaigns/"+c.ID+"/send-settings", map[string]any{"mode": "at",
		"sendAt": time.Date(day.Year(), day.Month(), day.Day(), 15, 0, 0, 0, time.UTC), "outside": "hold"}).MustStatus(t, 200).JSON(t, &rv)
	var res push.PushSendResult
	f.owner.Do(http.MethodPost, f.path+"/push-campaigns/"+c.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "scheduled" {
		t.Fatalf("send: %+v", res)
	}

	// Kim, an Editor limited to a list of the workspace.
	list := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, 'X22 beta')`, list, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	role, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: f.ws.ID, Key: authz.RoleEditor})
	if err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "kim@x22-team.test", Name: "Kim Contractor"})
	scope, _ := json.Marshal(authz.Scope{Lists: []uuid.UUID{list}})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: f.ws.ID, UserID: u.ID, RoleID: role.ID, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	kim := env.Client()
	kim.SignInDemo(u.Email)

	move := f.path + "/campaigns/" + c.ID + "/reschedule"
	to := map[string]any{"mode": "at", "date": day.AddDate(0, 0, 1).Format(time.DateOnly), "time": "10:30", "timezone": "UTC"}
	b, _ := json.Marshal(kim.Do(http.MethodPost, move, to).MustStatus(t, http.StatusForbidden).Problem(t))
	if !strings.Contains(string(b), "Your access is limited to some lists: send only to those lists, not to other lists or a segment.") {
		t.Fatalf("Kim moves it: %s", b)
	}
	id := uuid.MustParse(c.ID)
	if at := one[time.Time](t, env, `SELECT scheduled_at FROM push_sends WHERE campaign_id = $1`, id); !at.Equal(res.ScheduledAt) {
		t.Fatalf("Kim's refused move moved the send to %v", at)
	}
	f.owner.Do(http.MethodPost, move, to).MustStatus(t, http.StatusNoContent)
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.rescheduled' AND target_id = $1`, c.ID); n != 1 {
		t.Fatalf("audited moves: %d", n)
	}
}
