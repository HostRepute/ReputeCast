package whatsapp_test

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
	"github.com/hostrepute/reputecast/internal/whatsapp"
)

// A member limited to some lists moves a scheduled WhatsApp campaign only
// when its audience is on their lists, as they would send it.
func TestScopedMemberMovesWhatsAppOnTheirListsOnly(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	ctx := context.Background()
	f.connect(t)
	list, ids := f.people(t)
	c.Do(http.MethodPut, p+"/whatsapp/settings", map[string]any{"quietHours": false}).MustStatus(t, 200)
	for _, k := range []string{"sam", "ana"} {
		c.Do(http.MethodPut, p+"/contacts/"+ids[k].String()+"/whatsapp/consent", map[string]any{"listId": list, "status": "subscribed", "evidence": evidence()}).
			MustStatus(t, 200)
	}
	var ov whatsapp.WhatsAppOverview
	c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	var tpl string
	for _, tp := range ov.Templates {
		if tp.Name == "summit_day_before" {
			tpl = tp.ID
		}
	}
	var camp whatsapp.WhatsAppCampaign
	c.Do(http.MethodPost, p+"/whatsapp-campaigns", map[string]any{"name": "X22 scoped", "templateId": tpl, "lists": []string{list.String()}}).
		MustStatus(t, 201).JSON(t, &camp)
	day := time.Now().AddDate(0, 0, 10)
	c.Do(http.MethodPatch, p+"/whatsapp-campaigns/"+camp.ID, map[string]any{"name": camp.Name, "templateId": tpl, "lists": []string{list.String()}, "mode": "at",
		"sendAt": time.Date(day.Year(), day.Month(), day.Day(), 15, 0, 0, 0, time.UTC), "version": camp.Version,
		"variables": []map[string]any{{"key": "1", "source": "field", "field": "first_name", "fallback": "there"}, {"key": "2", "source": "text", "text": "8:30"}}}).
		MustStatus(t, 200).JSON(t, &camp)
	var rv whatsapp.WhatsAppReview
	c.Do(http.MethodGet, p+"/whatsapp-campaigns/"+camp.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	var res whatsapp.WhatsAppSendResult
	c.Do(http.MethodPost, p+"/whatsapp-campaigns/"+camp.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "scheduled" {
		t.Fatalf("send: %+v", res)
	}

	// Kim's lists don't hold the campaign's; Lee's do.
	other := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, 'X22 beta')`, other, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	role, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: f.ws.ID, Key: authz.RoleEditor})
	if err != nil {
		t.Fatal(err)
	}
	editor := func(email string, lists ...uuid.UUID) *apptest.Client {
		t.Helper()
		u := env.CreateUser(apptest.UserOpts{Email: email, Name: "Contractor " + email})
		scope, _ := json.Marshal(authz.Scope{Lists: lists})
		if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: f.ws.ID, UserID: u.ID, RoleID: role.ID, Scope: scope,
			Source: "invite"}); err != nil {
			t.Fatal(err)
		}
		cl := env.Client()
		cl.SignInDemo(email)
		return cl
	}
	kim, lee := editor("kim@x22wa-team.test", other), editor("lee@x22wa-team.test", list)

	move := p + "/campaigns/" + camp.ID + "/reschedule"
	to := map[string]any{"mode": "at", "date": day.AddDate(0, 0, 1).Format(time.DateOnly), "time": "16:45", "timezone": "UTC"}
	b, _ := json.Marshal(kim.Do(http.MethodPost, move, to).MustStatus(t, http.StatusForbidden).Problem(t))
	if !strings.Contains(string(b), "Your access is limited to some lists: send only to those lists, not to other lists or a segment.") {
		t.Fatalf("Kim moves it: %s", b)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.rescheduled' AND target_id = $1`, camp.ID); n != 0 {
		t.Fatalf("a refused move is audited: %d", n)
	}
	lee.Do(http.MethodPost, move, to).MustStatus(t, http.StatusNoContent)
	want := time.Date(day.Year(), day.Month(), day.Day()+1, 16, 45, 0, 0, time.UTC)
	if at := one[time.Time](t, env, `SELECT scheduled_at FROM whatsapp_sends WHERE campaign_id = $1`, uuid.MustParse(camp.ID)); !at.Equal(want) {
		t.Fatalf("Lee's move: the send starts at %v, want %v", at, want)
	}
}
