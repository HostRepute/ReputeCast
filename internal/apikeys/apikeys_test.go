package apikeys_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apikeys"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

type keyList struct {
	Items []struct {
		ID, Name, Masked, Status, LastUsedIP, LastUsedClient string
		Permissions                                          []string
		Scope                                                struct{ Label string }
		CreatedBy                                            struct{ Name string }
		Rejections                                           *struct {
			Calls  int64
			IP     string
			Client string
		}
	}
	Lists  []struct{ ID, Name string }
	Locked []struct{ Key, Reason string }
}

type created struct {
	Key struct {
		ID, Masked  string
		Permissions []string
	}
	Secret string
}

func TestKeysAndAgents(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var keys keyList
	c.Do("GET", ws+"/api-keys", nil).MustStatus(t, 200).JSON(t, &keys)
	// The SettingsKeys board: Acme Cloud's transactional keys first, as
	// they were created, then the others by last use, the expired one last.
	// Earn's conversions key (X17, EarnSettingsStates) is among the others.
	names := []string{}
	for _, k := range keys.Items {
		names = append(names, k.Name)
	}
	if strings.Join(names, ", ") != "Acme Cloud · production, Acme Cloud · staging, Billing service (SMTP), Website signup form, Acme Cloud · conversions, Data warehouse sync, Old CI key" {
		t.Fatalf("keys: %v", names)
	}
	if keys.Items[0].Masked != "rc_live_4c1e…d07b" || strings.Join(keys.Items[0].Permissions, ",") != "transactional:send" {
		t.Fatalf("production key: %+v", keys.Items[0])
	}
	if keys.Items[3].Scope.Label != "Weekly Digest only" || strings.Join(keys.Items[4].Permissions, ",") != "conversions:create" ||
		strings.Join(keys.Items[5].Permissions, ",") != "reports:view,subscribers:view" || keys.Items[5].LastUsedClient != "nightly job" {
		t.Fatalf("keys: %+v", keys.Items)
	}
	old := keys.Items[6]
	if old.Status != "expired" || old.Rejections == nil || old.Rejections.Calls != 212 || old.Rejections.Client != "GitHub Actions runner" {
		t.Fatalf("expired key: %+v", old)
	}
	var agents keyList
	c.Do("GET", ws+"/mcp-agents", nil).MustStatus(t, 200).JSON(t, &agents)
	if len(agents.Items) != 2 || agents.Items[0].Masked != "rc_agent_e0…44" && agents.Items[0].Masked != "rc_agent_7a…c2" {
		t.Fatalf("agents: %+v", agents.Items)
	}

	// Create: the secret is shown once and never listed again.
	var digest string
	for _, l := range keys.Lists {
		if l.Name == "Weekly Digest" {
			digest = l.ID
		}
	}
	var k created
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Campaign reader", "permissions": []string{"campaigns:view"}}).MustStatus(t, 201).JSON(t, &k)
	if !strings.HasPrefix(k.Secret, "rc_live_") || len(k.Secret) != len("rc_live_")+64 || !strings.HasPrefix(k.Key.Masked, k.Secret[:12]) {
		t.Fatalf("created: %+v", k)
	}
	r := c.Do("GET", ws+"/api-keys", nil).MustStatus(t, 200)
	if strings.Contains(string(r.Body), k.Secret) || strings.Contains(string(r.Body), k.Secret[12:40]) {
		t.Fatal("the secret must never be listed")
	}
	// The key works against the API, within its permissions.
	b := env.Client()
	b.Bearer = k.Secret
	b.Do("GET", ws+"/campaigns", nil).MustStatus(t, 200)
	b.Do("GET", ws+"/members", nil).MustStatus(t, 403)

	// Rotate: a new secret, the old one stops working.
	var rot created
	c.Do("POST", ws+"/api-keys/"+k.Key.ID+"/rotate", nil).MustStatus(t, 200).JSON(t, &rot)
	if rot.Secret == k.Secret || rot.Key.ID == k.Key.ID {
		t.Fatalf("rotate: %+v", rot)
	}
	b.Do("GET", ws+"/campaigns", nil).MustStatus(t, 401)
	b.Bearer = rot.Secret
	b.Do("GET", ws+"/campaigns", nil).MustStatus(t, 200)
	c.Do("POST", ws+"/api-keys/"+rot.Key.ID+"/revoke", nil).MustStatus(t, 204)
	b.Do("GET", ws+"/campaigns", nil).MustStatus(t, 401)

	// Scoped key and agent.
	var scoped created
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Signup", "permissions": []string{"subscribers:create"}, "lists": []string{digest}}).MustStatus(t, 201).JSON(t, &scoped)
	var agent created
	c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Writer bot", "description": "Claude Code", "permissions": []string{"campaigns:edit"}}).MustStatus(t, 201).JSON(t, &agent)
	if !strings.HasPrefix(agent.Secret, "rc_agent_") {
		t.Fatalf("agent token: %q", agent.Secret)
	}

	// Expired keys can be replaced or deleted; active ones can't be deleted.
	var repl created
	c.Do("POST", ws+"/api-keys/"+old.ID+"/rotate", nil).MustStatus(t, 200).JSON(t, &repl)
	c.Do("DELETE", ws+"/api-keys/"+scoped.Key.ID, nil).MustStatus(t, 409)

	var n int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('apikey.created','apikey.rotated','apikey.revoked','apikey.replaced','agent.created')`).Scan(&n)
	if n != 10 { // the seeded apikey.created and Acme Cloud's three, 2 created + rotated + revoked + replaced + agent
		t.Fatalf("audit entries: %d", n)
	}
	var meta []byte
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT after FROM audit_events WHERE action = 'apikey.created' ORDER BY occurred_at DESC LIMIT 1`).Scan(&meta)
	if strings.Contains(string(meta), "rc_live_") && strings.Contains(string(meta), scoped.Secret) {
		t.Fatal("secrets never go to the audit log")
	}
}

func TestKeyEscalation(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("POST", ws+"/roles", map[string]any{"name": "Integrations", "permissions": []string{"apikeys:manage", "reports:view", "subscribers:view"}}).MustStatus(t, 201)
	ctx := context.Background()
	w, _ := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	role, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: w.ID, Key: "integrations"})
	if err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "ivy@acme.io", Name: "Ivy Integrations"})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: w.ID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	ic := env.Client()
	ic.SignInDemo("ivy@acme.io")
	var l keyList
	ic.Do("GET", ws+"/api-keys", nil).MustStatus(t, 200).JSON(t, &l)
	found := false
	for _, lk := range l.Locked {
		if lk.Key == "subscribers:export" && strings.Contains(lk.Reason, "Ask Linh Nguyen or Marcus Chen") {
			found = true
		}
	}
	if !found {
		t.Fatalf("locked: %+v", l.Locked)
	}
	r := ic.Do("POST", ws+"/api-keys", map[string]any{"name": "Analytics export", "permissions": []string{"reports:view", "subscribers:export"}}).MustStatus(t, 403)
	if p := r.Problem(t); p["code"] != "escalation_denied" || p["permission"] != "subscribers:export" {
		t.Fatalf("escalation: %v", p)
	}
	var k created
	ic.Do("POST", ws+"/api-keys", map[string]any{"name": "Analytics export", "permissions": []string{"reports:view", "subscribers:view"}}).MustStatus(t, 201).JSON(t, &k)
	ic.Do("GET", ws+"/mcp-agents", nil).MustStatus(t, 403)
	// The key never does more than its creator can now: nothing once Ivy is
	// deactivated in the workspace.
	b := env.Client()
	b.Bearer = k.Secret
	b.Do("GET", ws+"/contacts?pageSize=1", nil).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE workspace_members SET status = 'deactivated' WHERE user_id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	b.Do("GET", ws+"/contacts?pageSize=1", nil).MustStatus(t, 403)
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/api-keys", nil).MustStatus(t, 403)
}

// A key's lists are its own scope narrowed to its creator's as it is now:
// Eve made a key, then was narrowed to Beta Program only, and the key reads
// Beta Program only. Narrowed to a list the key never had, it reads no
// contact at all, not every contact.
func TestKeyScopeFollowsItsCreator(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	linh.Do("POST", ws+"/roles", map[string]any{"name": "Key maker", "permissions": []string{"apikeys:manage", "subscribers:view"}}).MustStatus(t, 201)
	role, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: w.ID, Key: "key-maker"})
	if err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "eve@acme.io", Name: "Eve Keys"})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: w.ID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	lists := map[string]string{}
	rows, err := env.App.Q.TeamWorkspaceLists(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range rows {
		lists[l.Name] = l.ID.String()
	}
	beta, digest := lists["Beta Program"], lists["Weekly Digest"]
	if beta == "" || digest == "" {
		t.Fatalf("lists: %v", lists)
	}
	narrow := func(list string) {
		t.Helper()
		scope := `{}`
		if list != "" {
			scope = `{"lists":["` + list + `"]}`
		}
		if _, err := env.App.Pool.Exec(ctx, `UPDATE workspace_members SET scope = $2 WHERE user_id = $1`, u.ID, scope); err != nil {
			t.Fatal(err)
		}
	}
	type page struct {
		Total int64
		Items []struct{ Email string }
	}
	total := func(c *apptest.Client, query string) int64 {
		t.Helper()
		var p page
		c.Do("GET", ws+"/contacts?pageSize=1&exact=true"+query, nil).MustStatus(t, 200).JSON(t, &p)
		return p.Total
	}
	all := total(linh, "")
	eve := env.Client()
	eve.SignInDemo("eve@acme.io")
	var k, scoped created
	eve.Do("POST", ws+"/api-keys", map[string]any{"name": "Eve's reader", "permissions": []string{"subscribers:view"}}).MustStatus(t, 201).JSON(t, &k)
	eve.Do("POST", ws+"/api-keys", map[string]any{"name": "Eve's digest reader", "permissions": []string{"subscribers:view"}, "lists": []string{digest}}).MustStatus(t, 201).JSON(t, &scoped)
	key, digestKey := env.Client(), env.Client()
	key.Bearer, digestKey.Bearer = k.Secret, scoped.Secret
	if got := total(key, ""); got != all {
		t.Fatalf("before: the key reads %d contacts, want %d", got, all)
	}

	// Narrowed, Eve reads Beta Program's contacts, and so does her key.
	narrow(beta)
	inBeta := total(eve, "")
	if inBeta == 0 || inBeta >= all {
		t.Fatalf("Eve narrowed to Beta Program reads %d of %d contacts", inBeta, all)
	}
	if got := total(key, ""); got != inBeta {
		t.Fatalf("Eve narrowed to Beta Program: the key reads %d contacts, want %d", got, inBeta)
	}
	// The digest key and Eve share no list now: it reads nothing.
	if got := total(digestKey, ""); got != 0 {
		t.Fatalf("no list in common: the key reads %d contacts, want 0", got)
	}
	// Widening Eve again gives the keys back what they had.
	narrow("")
	if got := total(key, ""); got != all {
		t.Fatalf("Eve widened again: %d, want %d", got, all)
	}
	if got := total(digestKey, ""); got == 0 || got >= all {
		t.Fatalf("Eve widened again: the digest key reads %d of %d contacts", got, all)
	}
}

func TestClientLabel(t *testing.T) {
	for ua, want := range map[string]string{
		"GitHub-Actions-Runner/2.319": "GitHub Actions runner",
		"curl/8.7.1":                  "curl",
		"python-requests/2.32.3":      "python-requests",
		"":                            "",
		"Claude-Desktop/1.4":          "Claude Desktop",
	} {
		if got := apikeys.ClientLabel(ua); got != want {
			t.Errorf("ClientLabel(%q) = %q, want %q", ua, got, want)
		}
	}
}

// A call with a revoked key is refused and counted, so the keys screen can
// show what still uses it.
func TestRejectedKeyCallsAreCounted(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var k created
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Old sync", "permissions": []string{"campaigns:view"}}).MustStatus(t, 201).JSON(t, &k)
	c.Do("POST", ws+"/api-keys/"+k.Key.ID+"/revoke", nil).MustStatus(t, 204)

	b := env.Client()
	b.Bearer = k.Secret
	b.Do("GET", ws+"/campaigns", nil).MustStatus(t, 401)

	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM api_key_rejections WHERE api_key_id = $1 AND reason = 'revoked'`, uuid.MustParse(k.Key.ID)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rejections recorded: %d, want 1", n)
	}
}

func TestPeopleOnlyPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	r := c.Do("POST", ws+"/api-keys", map[string]any{"name": "Zapier", "permissions": []string{"subscribers:view", "integrations:manage"}}).MustStatus(t, 403)
	if p := r.Problem(t); p["code"] != "escalation_denied" || p["permission"] != "integrations:manage" ||
		!strings.Contains(p["detail"].(string), "Only people can hold integrations:manage") {
		t.Fatalf("people-only on a key: %v", p)
	}
	r = c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Claude Desktop", "permissions": []string{"subscribers:verify"}}).MustStatus(t, 403)
	if p := r.Problem(t); p["permission"] != "subscribers:verify" {
		t.Fatalf("no-agents on an agent: %v", p)
	}
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Verifier", "permissions": []string{"subscribers:verify"}}).MustStatus(t, 201)
	var agents keyList
	c.Do("GET", ws+"/mcp-agents", nil).MustStatus(t, 200).JSON(t, &agents)
	reasons := map[string]string{}
	for _, l := range agents.Locked {
		reasons[l.Key] = l.Reason
	}
	if reasons["ai:use"] != "AI features are for people. Agents bring their own model." || reasons["placement:manage"] == "" {
		t.Fatalf("agent locks: %v", reasons)
	}
}
