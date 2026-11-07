package texting_test

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
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/texting"
)

// scopedEditor adds an Editor limited to some lists to the demo workspace
// and signs them in.
func scopedEditor(t *testing.T, env *apptest.Env, email string, lists ...uuid.UUID) *apptest.Client {
	t.Helper()
	ctx := context.Background()
	var wsID, role uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, r.id FROM workspaces w JOIN roles r ON r.workspace_id = w.id AND r.key = 'editor'
		WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &role); err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: email, Name: "Contractor " + email})
	scope, _ := json.Marshal(authz.Scope{Lists: lists})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: role, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo(email)
	return c
}

// A member limited to some lists moves or resumes a scheduled SMS send only
// when its audience is on their lists, as they would send it; whoever built
// the audience. Members whose lists hold it, and Sofia, still can.
func TestScopedMemberMovesAndResumesTextsOnTheirListsOnly(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	summit := one[uuid.UUID](t, env, `SELECT id FROM lists WHERE name = 'Acme Summit 2026'`)
	beta := one[uuid.UUID](t, env, `SELECT id FROM lists WHERE name = 'Beta Program'`)
	kim := scopedEditor(t, env, "kim@contractor.dev", beta)
	lee := scopedEditor(t, env, "lee@contractor.dev", summit)
	var c texting.SmsCampaign
	sofia.Do(http.MethodPost, ws+"/sms-campaigns", map[string]any{"name": "Summit: doors", "lists": []string{summit.String()},
		"body": "Doors open at 8:30. Bring your badge."}).MustStatus(t, 201).JSON(t, &c)
	day := time.Now().AddDate(0, 0, 10)
	future := day.Format(time.DateOnly)
	var rv texting.SmsReview
	sofia.Do(http.MethodPut, ws+"/sms-campaigns/"+c.ID+"/send-settings", map[string]any{"mode": "at", "sendAt": time.Date(day.Year(), day.Month(), day.Day(), 15, 0, 0, 0, time.UTC)}).
		MustStatus(t, 200).JSON(t, &rv)
	acks := []string{}
	for _, ch := range rv.Checks {
		if ch.Status == "warn" {
			acks = append(acks, ch.Key)
		}
	}
	var res texting.SmsSendResult
	sofia.Do(http.MethodPost, ws+"/sms-campaigns/"+c.ID+"/send", map[string]any{"fingerprint": rv.Fingerprint, "acknowledged": acks}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "scheduled" {
		t.Fatalf("send: %+v", res)
	}
	refused := func(what string, r *apptest.Response) {
		t.Helper()
		b, _ := json.Marshal(r.MustStatus(t, http.StatusForbidden).Problem(t))
		if !strings.Contains(string(b), "Your access is limited to some lists: send only to those lists, not to other lists or a segment.") {
			t.Fatalf("%s: %s", what, b)
		}
	}
	move := "/workspaces/acme-newsletter/campaigns/" + c.ID + "/reschedule"
	to := map[string]any{"mode": "at", "date": future, "time": "11:00", "timezone": "America/Chicago"}
	refused("Kim moves it", kim.Do(http.MethodPost, move, to))
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.rescheduled' AND target_id = $1`, c.ID); n != 0 {
		t.Fatalf("a refused move is audited: %d", n)
	}
	lee.Do(http.MethodPost, move, to).MustStatus(t, http.StatusNoContent)
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.rescheduled' AND target_id = $1`, c.ID); n != 1 {
		t.Fatalf("Lee's move isn't audited: %d", n)
	}

	// Paused, its texts go on only for a member whose lists hold them.
	sofia.Do(http.MethodPost, ws+"/sms-campaigns/"+c.ID+"/pause", map[string]any{"reason": "Venue change"}).MustStatus(t, http.StatusNoContent)
	refused("Kim resumes it", kim.Do(http.MethodPost, ws+"/sms-campaigns/"+c.ID+"/resume", map[string]any{}))
	if p := one[string](t, env, `SELECT phase FROM sms_sends WHERE campaign_id = $1`, c.ID); p != "paused" {
		t.Fatalf("after Kim's resume: %s", p)
	}
	sofia.Do(http.MethodPost, ws+"/sms-campaigns/"+c.ID+"/resume", map[string]any{}).MustStatus(t, http.StatusNoContent)
	if p := one[string](t, env, `SELECT phase FROM sms_sends WHERE campaign_id = $1`, c.ID); p == "paused" {
		t.Fatal("Sofia's resume left the send paused")
	}
}
