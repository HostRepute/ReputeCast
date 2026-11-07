package webhooks_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Every event's data has a view permission someone chose for it: an event
// added without one would fall back to needing contacts.
func TestEveryEventSaysWhatItNeeds(t *testing.T) {
	for _, e := range webhooks.Events {
		if !webhooks.Described(e.Name) {
			t.Errorf("%s: say what its data needs in internal/webhooks/access.go", e.Name)
		}
		if len(webhooks.Permissions(e.Name)) == 0 {
			t.Errorf("%s needs no permission", e.Name)
		}
	}
	for event, want := range map[string]string{
		"contact.updated": "subscribers:view", "email.batch": "subscribers:view", "form.submitted": "subscribers:view",
		"message.bounced": "transactional:view", "campaign.completed": "campaigns:view", "provider.down": "providers:view",
	} {
		if got := webhooks.Permissions(event); len(got) != 1 || string(got[0]) != want {
			t.Errorf("%s needs %v, want %s", event, got, want)
		}
	}
	if n := webhooks.NeedsOf("contact.updated"); !n.Contacts {
		t.Error("contact events carry every list's contacts")
	}
	if n := webhooks.NeedsOf("message.bounced"); n.Contacts {
		t.Error("transactional mail isn't kept per list")
	}
	if n := webhooks.NeedsOf("webhook.test"); len(n.Permissions) != 0 || n.Contacts {
		t.Error("a test event carries nothing")
	}
}

// webhooks:manage chooses where events go; what they carry needs its own
// view permission, and contacts can't go out from someone limited to some
// lists.
func TestSubscribingNeedsWhatEventsCarry(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	srv := httptest.NewServer(&receiver{})
	defer srv.Close()
	hook := func(events ...string) map[string]any {
		return map[string]any{"name": "Receiver", "url": srv.URL + "/hook", "events": events}
	}

	// A key that holds webhooks:manage alone can't send any event's data
	// anywhere; with campaigns:view it can take campaign events only.
	var k struct{ Secret string }
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Hooks only", "permissions": []string{"webhooks:manage"}}).MustStatus(t, 201).JSON(t, &k)
	key := env.Client()
	key.Bearer = k.Secret
	r := key.Do("POST", ws+"/webhooks", hook("contact.updated")).MustStatus(t, 403)
	if p := r.Problem(t); p["code"] != "permission_denied" || p["permission"] != "subscribers:view" || !strings.Contains(p["detail"].(string), "contact.updated") {
		t.Fatalf("contact.updated with webhooks:manage alone: %v", p)
	}
	r = key.Do("POST", ws+"/webhooks", hook("campaign.completed")).MustStatus(t, 403)
	if p := r.Problem(t); p["permission"] != "campaigns:view" {
		t.Fatalf("campaign.completed with webhooks:manage alone: %v", p)
	}
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Campaign hooks", "permissions": []string{"webhooks:manage", "campaigns:view"}}).MustStatus(t, 201).JSON(t, &k)
	key.Bearer = k.Secret
	key.Do("POST", ws+"/webhooks", hook("campaign.*")).MustStatus(t, 201)
	r = key.Do("POST", ws+"/webhooks", hook("campaign.completed", "email.*")).MustStatus(t, 403)
	if p := r.Problem(t); p["permission"] != "subscribers:view" || !strings.Contains(p["detail"].(string), "email.* includes email.sent") {
		t.Fatalf("email.* names the event it includes: %v", p)
	}
	r = key.Do("POST", ws+"/webhooks", hook("*")).MustStatus(t, 403)
	if p := r.Problem(t); p["code"] != "permission_denied" || !strings.Contains(p["detail"].(string), "* includes contact.subscribed") {
		t.Fatalf("every event: %v", p)
	}
	// Nor can it read contacts in the delivery log: a delivery's payload is
	// shown only to those who could see its data.
	if _, err := webhooks.Emit(ctx, env.App.Pool, w.ID, w.Slug, "email.bounced", map[string]any{"to": "hidden.person@example.com"}); err != nil {
		t.Fatal(err)
	}
	var bounced string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM webhook_deliveries WHERE payload -> 'data' ->> 'to' = 'hidden.person@example.com' LIMIT 1`).Scan(&bounced); err != nil {
		t.Fatal(err)
	}
	type detail struct {
		Payload       json.RawMessage
		PayloadHidden string
	}
	var seen detail
	r = key.Do("GET", ws+"/webhooks/deliveries/"+bounced, nil).MustStatus(t, 200)
	r.JSON(t, &seen)
	if strings.Contains(string(r.Body), "hidden.person@example.com") || string(seen.Payload) != "null" || !strings.Contains(seen.PayloadHidden, "View contacts") {
		t.Fatalf("the payload, for webhooks:manage alone: %s", r.Body)
	}
	var linhSees detail
	c.Do("GET", ws+"/webhooks/deliveries/"+bounced, nil).MustStatus(t, 200).JSON(t, &linhSees)
	if !strings.Contains(string(linhSees.Payload), "hidden.person@example.com") || linhSees.PayloadHidden != "" {
		t.Fatalf("the payload, for Linh: %+v", linhSees)
	}

	// A person who holds webhooks:manage alone: the same, and the problem
	// sits beside the events and says who can help.
	hana := member(t, env, w, "hana@acme.io", "Hana Hooks", []string{"webhooks:manage"})
	r = hana.Do("POST", ws+"/webhooks", hook("form.submitted")).MustStatus(t, 403)
	if p := r.Problem(t); p["permission"] != "subscribers:view" || !strings.Contains(p["detail"].(string), "Ask ") || !strings.Contains(string(r.Body), `"location":"body.events"`) {
		t.Fatalf("a person with webhooks:manage alone: %s", r.Body)
	}

	// Seeing contacts, but limited to Beta Program: no contact events,
	// campaign events still.
	ivy := member(t, env, w, "ivy@acme.io", "Ivy Scoped", []string{"webhooks:manage", "subscribers:view", "campaigns:view"})
	ivy.Do("POST", ws+"/webhooks", hook("contact.updated")).MustStatus(t, 201)
	scope(t, env, "ivy@acme.io", list(t, env, w, "Beta Program"))
	for _, events := range [][]string{{"contact.updated"}, {"email.bounced"}, {"form.*"}} {
		r := ivy.Do("POST", ws+"/webhooks", hook(events...)).MustStatus(t, 403)
		if p := r.Problem(t); p["code"] != "permission_denied" || !strings.Contains(p["detail"].(string), "some lists") {
			t.Fatalf("%v from a scoped member: %v", events, p)
		}
	}
	ivy.Do("POST", ws+"/webhooks", hook("campaign.completed")).MustStatus(t, 201)

	// Editing re-checks the events, as adding does.
	var list struct{ Items []struct{ ID, Name string } }
	c.Do("GET", ws+"/webhooks", nil).MustStatus(t, 200).JSON(t, &list)
	var editable string
	for _, it := range list.Items {
		if it.Name == "Receiver" {
			editable = it.ID
		}
	}
	ivy.Do("PUT", ws+"/webhooks/"+editable, hook("contact.subscribed")).MustStatus(t, 403)
	// Nothing was written for the refusals.
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhooks WHERE workspace_id = $1 AND name = 'Receiver'`, w.ID).Scan(&n); err != nil || n != 3 {
		t.Fatalf("endpoints made: %d %v", n, err)
	}
}

// Every delivery checks the person the endpoint sends for as they are at
// that moment: once they can't see what an event carries, ReputeCast
// pauses the endpoint, says why, and keeps the events until someone who
// can see them resumes it on their own behalf.
func TestDeliveriesStopWhenTheCreatorCantSeeThem(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	rcv := &receiver{}
	srv := httptest.NewServer(rcv)
	defer srv.Close()
	received := func() int {
		rcv.mu.Lock()
		defer rcv.mu.Unlock()
		return len(rcv.bodies)
	}
	s := service(env)

	ivy := member(t, env, w, "ivy@acme.io", "Ivy Hooks", []string{"webhooks:manage", "subscribers:view", "campaigns:view"})
	var made struct{ Webhook struct{ ID string } }
	ivy.Do("POST", ws+"/webhooks", map[string]any{"name": "CRM", "url": srv.URL + "/crm", "events": []string{"contact.*", "campaign.completed"}}).MustStatus(t, 201).JSON(t, &made)
	id := uuid.MustParse(made.Webhook.ID)
	emit := func(event string) uuid.UUID {
		t.Helper()
		if _, err := webhooks.Emit(ctx, env.App.Pool, w.ID, w.Slug, event, map[string]any{"email": "someone@example.com"}); err != nil {
			t.Fatal(err)
		}
		var d uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM webhook_deliveries WHERE webhook_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, id).Scan(&d); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Attempt(ctx, d); err != nil {
			t.Fatal(err)
		}
		return d
	}
	state := func(d uuid.UUID) string {
		t.Helper()
		var st string
		if err := env.App.Pool.QueryRow(ctx, `SELECT state FROM webhook_deliveries WHERE id = $1`, d).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	type endpoint struct {
		ID, Status, PausedReason string
	}
	view := func() endpoint {
		t.Helper()
		var l struct{ Items []endpoint }
		linh.Do("GET", ws+"/webhooks", nil).MustStatus(t, 200).JSON(t, &l)
		for _, it := range l.Items {
			if it.ID == made.Webhook.ID {
				return it
			}
		}
		t.Fatal("the endpoint is listed")
		return endpoint{}
	}

	// While Ivy can see contacts on every list, they go out.
	if d := emit("contact.updated"); state(d) != "delivered" || received() != 1 {
		t.Fatalf("Ivy as she was: %s, %d received", state(d), received())
	}

	// Narrowed to Beta Program, she can't see every list's contacts: the
	// delivery waits, the endpoint pauses and says why, as the system.
	scope(t, env, "ivy@acme.io", list(t, env, w, "Beta Program"))
	held := emit("contact.updated")
	if state(held) != "held" || received() != 1 {
		t.Fatalf("Ivy narrowed: %s, %d received", state(held), received())
	}
	if v := view(); v.Status != "paused" || !strings.Contains(v.PausedReason, "Ivy Hooks") || !strings.Contains(v.PausedReason, "some lists") {
		t.Fatalf("paused, with the reason: %+v", v)
	}
	var actor, reason string
	if err := env.App.Pool.QueryRow(ctx, `SELECT actor_type, reason FROM audit_events WHERE action = 'webhook.paused' AND target_id = $1`, made.Webhook.ID).Scan(&actor, &reason); err != nil || actor != string(authz.ActorSystem) || !strings.Contains(reason, "Ivy Hooks") {
		t.Fatalf("the pause is audited as the system's: %q %q %v", actor, reason, err)
	}
	// Events keep queuing while it is paused.
	if _, err := webhooks.Emit(ctx, env.App.Pool, w.ID, w.Slug, "contact.subscribed", map[string]any{}); err != nil {
		t.Fatal(err)
	}

	// Ivy can't resume it; Linh can, and from then on it sends for her:
	// what waited goes out.
	ivy.Do("POST", ws+"/webhooks/"+made.Webhook.ID+"/resume", nil).MustStatus(t, 403)
	linh.Do("POST", ws+"/webhooks/"+made.Webhook.ID+"/resume", nil).MustStatus(t, 204)
	if v := view(); v.Status == "paused" || v.PausedReason != "" {
		t.Fatalf("resumed: %+v", v)
	}
	var owner string
	if err := env.App.Pool.QueryRow(ctx, `SELECT u.email FROM webhooks h JOIN users u ON u.id = h.created_by WHERE h.id = $1`, id).Scan(&owner); err != nil || owner != "linh@acme.io" {
		t.Fatalf("it sends for %q (%v)", owner, err)
	}
	rows, err := env.App.Pool.Query(ctx, `SELECT id FROM webhook_deliveries WHERE webhook_id = $1 AND state = 'pending'`, id)
	if err != nil {
		t.Fatal(err)
	}
	var waiting []uuid.UUID
	for rows.Next() {
		var d uuid.UUID
		if err := rows.Scan(&d); err != nil {
			t.Fatal(err)
		}
		waiting = append(waiting, d)
	}
	rows.Close()
	for _, d := range waiting {
		if _, err := s.Attempt(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	if len(waiting) != 2 || state(held) != "delivered" || received() != 3 {
		t.Fatalf("after Linh resumed: %d waiting, %s, %d received", len(waiting), state(held), received())
	}

	// Ivy saves it again, so it sends for her, and then loses contacts
	// altogether: the reason names what the event needs, in words.
	scope(t, env, "ivy@acme.io", "")
	ivy.Do("PUT", ws+"/webhooks/"+made.Webhook.ID, map[string]any{"name": "CRM", "url": srv.URL + "/crm", "events": []string{"contact.*", "campaign.completed"}}).MustStatus(t, 200)
	role(t, env, w, "ivy@acme.io", []string{"webhooks:manage", "campaigns:view"})
	if d := emit("contact.updated"); state(d) != "held" {
		t.Fatalf("without subscribers:view: %s", state(d))
	}
	if v := view(); v.Status != "paused" || !strings.Contains(v.PausedReason, "View contacts") {
		t.Fatalf("paused for the permission: %+v", v)
	}

	// Removed from the workspace, she sends nothing, campaigns included.
	linh.Do("POST", ws+"/webhooks/"+made.Webhook.ID+"/resume", nil).MustStatus(t, 204)
	ivy.Do("PUT", ws+"/webhooks/"+made.Webhook.ID, map[string]any{"name": "CRM", "url": srv.URL + "/crm", "events": []string{"campaign.completed"}}).MustStatus(t, 200)
	before := received()
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM workspace_members m USING users u WHERE u.id = m.user_id AND u.email = 'ivy@acme.io' AND m.workspace_id = $1`, w.ID); err != nil {
		t.Fatal(err)
	}
	if d := emit("campaign.completed"); state(d) != "held" || received() != before {
		t.Fatalf("Ivy removed: %s", state(d))
	}
	if v := view(); v.Status != "paused" || !strings.Contains(v.PausedReason, "no longer an active member") {
		t.Fatalf("paused because she left: %+v", v)
	}
	var paused int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'webhook.paused' AND target_id = $1`, made.Webhook.ID).Scan(&paused); err != nil || paused != 3 {
		t.Fatalf("one entry per pause: %d %v", paused, err)
	}

	// An endpoint that sends for nobody (from before people were recorded)
	// goes on as it did.
	legacy := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO webhooks (id, workspace_id, url, events, secret_enc) VALUES ($1, $2, $3, '{contact.updated}', $4)`,
		legacy, w.ID, srv.URL+"/legacy", sealed(t, env, legacy)); err != nil {
		t.Fatal(err)
	}
	id = legacy
	before = received()
	if d := emit("contact.updated"); state(d) != "delivered" || received() != before+1 {
		t.Fatalf("an endpoint without a creator: %s", state(d))
	}
}

// member adds a person with a custom role holding perms, signed in.
func member(t *testing.T, env *apptest.Env, w store.Workspace, email, name string, perms []string) *apptest.Client {
	t.Helper()
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: email, Name: name})
	r := customRole(t, env, w, name, perms)
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: w.ID, UserID: u.ID, RoleID: r, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo(email)
	return c
}

// role gives a member a new custom role holding perms.
func role(t *testing.T, env *apptest.Env, w store.Workspace, email string, perms []string) {
	t.Helper()
	r := customRole(t, env, w, email+" "+strings.Join(perms, " "), perms)
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE workspace_members m SET role_id = $3 FROM users u WHERE u.id = m.user_id AND u.email = $1 AND m.workspace_id = $2`, email, w.ID, r); err != nil {
		t.Fatal(err)
	}
}

func customRole(t *testing.T, env *apptest.Env, w store.Workspace, name string, perms []string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	// A v7 id starts with its time: the whole id keeps keys apart.
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO roles (id, workspace_id, key, name, builtin, permissions) VALUES ($1, $2, $3, $4, false, $5)`,
		id, w.ID, "r"+strings.ReplaceAll(id.String(), "-", ""), name, perms); err != nil {
		t.Fatal(err)
	}
	return id
}

// scope limits a member to one list ("" for every list).
func scope(t *testing.T, env *apptest.Env, email, listID string) {
	t.Helper()
	s := `{}`
	if listID != "" {
		s = `{"lists":["` + listID + `"]}`
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE workspace_members m SET scope = $2 FROM users u WHERE u.id = m.user_id AND u.email = $1`, email, s); err != nil {
		t.Fatal(err)
	}
}

func list(t *testing.T, env *apptest.Env, w store.Workspace, name string) string {
	t.Helper()
	var id string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id::text FROM lists WHERE workspace_id = $1 AND name = $2`, w.ID, name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// sealed is a signing secret for an endpoint inserted by hand.
func sealed(t *testing.T, env *apptest.Env, id uuid.UUID) []byte {
	t.Helper()
	b, err := env.App.Secrets.SealString(webhooks.NewSecret(), webhooks.SecretAD(id))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
