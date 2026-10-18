package integrations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

const ws = "/workspaces/acme-newsletter"

// call sends a request with a bearer key and extra headers.
func call(t *testing.T, env *apptest.Env, key, method, path string, body any, headers map[string]string) *apptest.Response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, env.Server.URL+"/api/v1"+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return &apptest.Response{Status: resp.StatusCode, Header: resp.Header, Body: b}
}

func code(t *testing.T, r *apptest.Response) string {
	t.Helper()
	c, _ := r.Problem(t)["code"].(string)
	return c
}

type secret struct {
	IntegrationID string
	Kind          string
	Secret        string
	InstanceURL   string
	OldKeyUntil   *time.Time
	Key           struct {
		ID, Name, Masked string
		Permissions      []string
	}
}

// publicURL gives the test instance a public address, so Zapier and Make
// may connect (the test server itself listens on loopback).
func publicURL(env *apptest.Env) {
	env.App.PublicURL.Set("https://203.0.113.10")
}

func connect(t *testing.T, c *apptest.Client, kind string, body map[string]any) secret {
	t.Helper()
	var out secret
	c.Do("POST", ws+"/integrations/apps/"+kind+"/connect", body).MustStatus(t, 201).JSON(t, &out)
	if !strings.HasPrefix(out.Secret, "rc_live_") || out.IntegrationID == "" {
		t.Fatalf("connected: %+v", out)
	}
	return out
}

func audited(t *testing.T, env *apptest.Env, action, actorType string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND actor_type = $2`, action, actorType).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func exec(t *testing.T, env *apptest.Env, sql string, args ...any) {
	t.Helper()
	if _, err := env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

type catalog struct {
	Groups []struct {
		Key     string
		Entries []struct {
			Key, Status     string
			UpdateAvailable bool
			ConnectionID    string
			Reasons         []struct{ Code, Host string }
			Facts           struct {
				Workflows, Endpoints, Failing, Agents int64
				Sites, SilentSites, Updates           int
				Team                                  string
			}
		}
	}
	Alerts    []struct{ Entry string }
	CanManage bool
	Helpers   []string
}

func (c catalog) entry(key string) (string, int64, []string) {
	for _, g := range c.Groups {
		for _, e := range g.Entries {
			if e.Key == key {
				codes := []string{}
				for _, r := range e.Reasons {
					codes = append(codes, r.Code)
				}
				return e.Status, e.Facts.Workflows, codes
			}
		}
	}
	return "", 0, nil
}

func TestCatalogAndReadOnly(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var cat catalog
	c.Do("GET", ws+"/integrations/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	keys := []string{}
	for _, g := range cat.Groups {
		keys = append(keys, g.Key)
	}
	if strings.Join(keys, ",") != "automation,websites,deliverability,developer,verification" {
		t.Fatalf("groups with nothing in them aren't shown: %v", keys)
	}
	// Verification lists every address verification service (X14).
	services := []string{}
	for _, e := range cat.Groups[4].Entries {
		services = append(services, e.Key)
	}
	if strings.Join(services, ",") != "zerobounce,neverbounce,kickbox,bouncer,emailable" {
		t.Fatalf("verification cards = %v", services)
	}
	if st, wf, _ := cat.entry("zapier"); st != "connected" || wf != 3 {
		t.Errorf("Zapier: %s, %d Zaps", st, wf)
	}
	if st, _, _ := cat.entry("make"); st != "not_connected" {
		t.Errorf("Make: %s", st)
	}
	if st, wf, _ := cat.entry("n8n"); st != "connected" || wf != 3 {
		t.Errorf("n8n: %s, %d workflows", st, wf)
	}
	if st, _, reasons := cat.entry("wordpress"); st != "attention" || strings.Join(reasons, ",") != "site_silent" {
		t.Errorf("WordPress: %s %v", st, reasons)
	}
	if len(cat.Alerts) != 1 || cat.Alerts[0].Entry != "wordpress" || !cat.CanManage {
		t.Errorf("banners: %+v", cat.Alerts)
	}
	// Aisha, a Viewer, sees everything and connects nothing.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	var ro catalog
	v.Do("GET", ws+"/integrations/catalog", nil).MustStatus(t, 200).JSON(t, &ro)
	if ro.CanManage || strings.Join(ro.Helpers, ",") != "Marcus Chen,Linh Nguyen" {
		t.Errorf("read-only: %v %v", ro.CanManage, ro.Helpers)
	}
	v.Do("GET", ws+"/integrations/apps/zapier", nil).MustStatus(t, 200)
	if r := v.Do("POST", ws+"/integrations/apps/make/connect", map[string]any{"items": []string{"upsert_contact"}}); r.Status != 403 || code(t, r) != "permission_denied" {
		t.Errorf("a Viewer connects: %d %s", r.Status, r.Body)
	}
}

func TestKeyWorksOnlyForItsOperations(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	publicURL(env)
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	var info struct {
		Name  string
		Items []struct {
			Key       string
			On        bool
			Locked    bool
			Sensitive bool
		}
		Reachability struct{ Reachable bool }
	}
	c.Do("GET", ws+"/integrations/apps/make/connect", nil).MustStatus(t, 200).JSON(t, &info)
	if info.Name != "Make · Acme ops" || !info.Reachability.Reachable {
		t.Fatalf("connect info: %+v", info)
	}
	for _, it := range info.Items {
		if it.Key == "send_transactional" && (it.On || !it.Sensitive) {
			t.Error("Send a transactional email starts off: it needs ⚑ transactional:send")
		}
	}
	made := connect(t, c, "make", map[string]any{"name": "Make · Acme ops",
		"items": []string{"contact.subscribed", "contact.unsubscribed", "upsert_contact", "add_to_list", "add_tag", "record_event"}})
	if made.Key.Name != "Make · Acme ops" || strings.Join(made.Key.Permissions, ",") != "subscribers:create,subscribers:edit,webhooks:manage" {
		t.Errorf("key: %+v", made.Key)
	}
	if audited(t, env, "integration.connected", "user") != 5 {
		t.Error("connecting is audited")
	}
	var status struct{ Status string }
	c.Do("GET", ws+"/integrations/connections/"+made.IntegrationID+"/status", nil).MustStatus(t, 200).JSON(t, &status)
	if status.Status != "waiting" {
		t.Errorf("before the first call: %s", status.Status)
	}
	// Test connection needs only the key.
	r := call(t, env, made.Secret, "GET", "/integration", nil, map[string]string{"User-Agent": "ReputeCast-Make/1.0.4"})
	var me struct {
		Workspace, Integration string
		Triggers, Actions      []string
	}
	r.MustStatus(t, 200).JSON(t, &me)
	if me.Workspace != "acme-newsletter" || me.Integration != "make" || len(me.Triggers) != 2 || len(me.Actions) != 4 {
		t.Errorf("me: %+v", me)
	}
	c.Do("GET", ws+"/integrations/connections/"+made.IntegrationID+"/status", nil).MustStatus(t, 200).JSON(t, &status)
	if status.Status != "connected" {
		t.Errorf("after the first call: %s", status.Status)
	}
	// Its actions work.
	beta := lists(t, env)["Beta Program"]
	wf := map[string]string{"X-ReputeCast-Workflow": "Beta signups; id=42"}
	call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "new.person@fastmail.com", "firstName": "New",
		"lists": []string{beta}, "consent": "single", "tags": []string{"Beta-Applicant"}}, wf).MustStatus(t, 200)
	// Everything else is refused, even where the key's permissions would allow it.
	for _, p := range []struct{ method, path string }{
		{"GET", ws + "/contacts"},
		{"GET", ws + "/webhooks"},
		{"POST", ws + "/webhooks"},
		{"GET", ws + "/lists"},
		{"GET", ws + "/integrations/catalog"},
	} {
		r := call(t, env, made.Secret, p.method, p.path, map[string]any{}, nil)
		if r.Status != 403 || code(t, r) != "integration_operation_denied" {
			t.Errorf("%s %s with Make's key: %d %s", p.method, p.path, r.Status, r.Body)
		}
	}
	// A plain API key can't hold integrations:manage, and keys can't connect.
	if r := call(t, env, made.Secret, "POST", ws+"/integrations/apps/n8n/connect", map[string]any{"items": []string{"add_tag"}}, nil); r.Status != 403 {
		t.Errorf("a key connects an integration: %d", r.Status)
	}
	// The call log has them, with the workflow.
	var log struct {
		Total, Errors int64
		Items         []struct {
			Workflow, Operation, Code string
			Status                    int
			Target                    struct{ Kind, Email, List string }
		}
	}
	c.Do("GET", ws+"/integrations/connections/"+made.IntegrationID+"/calls", nil).MustStatus(t, 200).JSON(t, &log)
	if log.Total != 8 || log.Errors != 6 {
		t.Fatalf("call log: %d calls, %d errors: %+v", log.Total, log.Errors, log.Items)
	}
	var upsert bool
	for _, it := range log.Items {
		if it.Operation == "upsert-contact" {
			upsert = it.Workflow == "Beta signups" && it.Status == 200 && it.Target.Email == "new.person@fastmail.com" && it.Target.List == "Beta Program"
		}
	}
	if !upsert {
		t.Errorf("the upsert call: %+v", log.Items)
	}
	// Aisha (Viewer, subscribers:view) sees addresses; a member without it sees "a contact".
	exec(t, env, `UPDATE roles SET permissions = array_remove(permissions, 'subscribers:view') WHERE key = 'content-writer'`)
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	var hidden struct {
		Items []struct {
			Who    string
			Target struct{ Email string }
		}
	}
	d.Do("GET", ws+"/integrations/connections/"+made.IntegrationID+"/calls", nil).MustStatus(t, 200).JSON(t, &hidden)
	for _, it := range hidden.Items {
		if it.Target.Email != "" {
			t.Errorf("an address shown to a member without subscribers:view: %+v", it)
		}
	}
	// Change what Make can do: Add a tag turned off stops its operation.
	c.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/change", map[string]any{"items": []string{"upsert_contact", "record_event"}}).MustStatus(t, 200)
	if r := call(t, env, made.Secret, "POST", ws+"/contacts/tags", map[string]any{"email": "new.person@fastmail.com", "tags": []string{"x"}}, nil); r.Status != 403 ||
		code(t, r) != "integration_operation_denied" {
		t.Errorf("a turned-off action: %d %s", r.Status, r.Body)
	}
	if audited(t, env, "integration.permissions_changed", "user") != 1 {
		t.Error("changing what it can do is audited")
	}
}

func lists(t *testing.T, env *apptest.Env) map[string]string {
	t.Helper()
	rows, err := env.App.Pool.Query(context.Background(), `SELECT l.id, l.name FROM lists l JOIN workspaces w ON w.id = l.workspace_id WHERE w.slug = 'acme-newsletter'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		_ = rows.Scan(&id, &name)
		out[name] = id.String()
	}
	return out
}

func TestEscalationGuard(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	publicURL(env)
	// Daniel's custom role can manage integrations but can't send transactional email.
	exec(t, env, `UPDATE roles SET permissions = permissions || '{integrations:manage,subscribers:create,subscribers:edit,webhooks:manage}' WHERE key = 'content-writer'`)
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	var info struct {
		Items []struct {
			Key, Missing string
			Locked       bool
			Helpers      []string
		}
	}
	d.Do("GET", ws+"/integrations/apps/make/connect", nil).MustStatus(t, 200).JSON(t, &info)
	for _, it := range info.Items {
		if it.Key == "send_transactional" && (!it.Locked || it.Missing != "transactional:send" || strings.Join(it.Helpers, ",") != "Linh Nguyen,Marcus Chen") {
			t.Errorf("the locked row: %+v", it)
		}
		if it.Key == "upsert_contact" && it.Locked {
			t.Error("what Daniel holds stays available")
		}
	}
	r := d.Do("POST", ws+"/integrations/apps/make/connect", map[string]any{"items": []string{"upsert_contact", "send_transactional"}})
	if r.Status != 403 || code(t, r) != "escalation_denied" || !strings.Contains(string(r.Body), "You can't grant transactional:send because you don't hold it. Ask Linh Nguyen or Marcus Chen.") {
		t.Fatalf("escalation: %d %s", r.Status, r.Body)
	}
	connect(t, d, "make", map[string]any{"items": []string{"upsert_contact"}})
}

func TestRotationKeepsTheOldKeyForADay(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	publicURL(env)
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	made := connect(t, c, "make", map[string]any{"items": []string{"upsert_contact"}})
	var rotated secret
	c.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/rotate", nil).MustStatus(t, 200).JSON(t, &rotated)
	if rotated.Secret == made.Secret || rotated.OldKeyUntil == nil || time.Until(*rotated.OldKeyUntil) < 23*time.Hour {
		t.Fatalf("rotated: %+v", rotated)
	}
	call(t, env, rotated.Secret, "GET", "/integration", nil, nil).MustStatus(t, 200)
	call(t, env, made.Secret, "GET", "/integration", nil, nil).MustStatus(t, 200) // still works for 24 hours
	var app struct {
		Key    struct{ Masked string }
		OldKey *struct {
			Status     string
			GraceUntil time.Time
		}
	}
	c.Do("GET", ws+"/integrations/apps/make", nil).MustStatus(t, 200).JSON(t, &app)
	if app.OldKey == nil || app.OldKey.Status != "grace" || app.Key.Masked != rotated.Key.Masked {
		t.Errorf("the page shows both keys: %+v", app)
	}
	// After 24 hours the old key stops.
	exec(t, env, `UPDATE api_keys SET grace_until = now() - interval '1 second' WHERE id = $1`, made.Key.ID)
	if r := call(t, env, made.Secret, "GET", "/integration", nil, nil); r.Status != 401 || code(t, r) != "key_revoked" {
		t.Errorf("the old key after its grace period: %d %s", r.Status, r.Body)
	}
	// "Stop the old key now" ends it at once.
	var again secret
	c.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/rotate", nil).MustStatus(t, 200).JSON(t, &again)
	call(t, env, rotated.Secret, "GET", "/integration", nil, nil).MustStatus(t, 200)
	c.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/stop-old-key", nil).MustStatus(t, 204)
	if r := call(t, env, rotated.Secret, "GET", "/integration", nil, nil); r.Status != 401 {
		t.Errorf("stopped key: %d", r.Status)
	}
	call(t, env, again.Secret, "GET", "/integration", nil, nil).MustStatus(t, 200)
	if audited(t, env, "integration.key_rotated", "user") != 2 || audited(t, env, "integration.key_revoked", "user") != 1 {
		t.Error("rotating and stopping are audited")
	}
	// Keys & MCP lists it as managed and leaves rotation to Integrations.
	l := env.Client()
	l.SignInDemo("linh@acme.io")
	var plain struct{ Items []struct{ ID, Name string } }
	l.Do("GET", ws+"/api-keys", nil).MustStatus(t, 200).JSON(t, &plain)
	for _, k := range plain.Items {
		if strings.HasPrefix(k.Name, "Make · ") || strings.HasPrefix(k.Name, "Zapier · ") {
			t.Errorf("an integration's key in the plain table: %s", k.Name)
		}
	}
	var managed struct {
		Items []struct {
			ID, Kind, Name string
			Workflows      []struct{ Name string }
		}
	}
	l.Do("GET", ws+"/integrations/keys", nil).MustStatus(t, 200).JSON(t, &managed)
	if len(managed.Items) != 5 {
		t.Fatalf("managed keys: %+v", managed.Items)
	}
	for _, k := range managed.Items {
		if k.Kind == "zapier" && len(k.Workflows) != 3 {
			t.Errorf("revoking names the Zaps that stop: %+v", k)
		}
		if k.Kind == "make" {
			if r := l.Do("POST", ws+"/api-keys/"+k.ID+"/rotate", nil); r.Status != 409 || code(t, r) != "managed_by_integration" {
				t.Errorf("rotating an integration's key in Keys & MCP: %d %s", r.Status, r.Body)
			}
			// Revoke stays possible there.
			l.Do("POST", ws+"/api-keys/"+k.ID+"/revoke", nil).MustStatus(t, 204)
		}
	}
	if r := call(t, env, again.Secret, "GET", "/integration", nil, nil); r.Status != 401 || code(t, r) != "key_revoked" {
		t.Errorf("revoked in Keys & MCP: %d %s", r.Status, r.Body)
	}
	c.Do("GET", ws+"/integrations/apps/make", nil).MustStatus(t, 200)
	var cat catalog
	c.Do("GET", ws+"/integrations/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	if st, _, reasons := cat.entry("make"); st != "attention" || strings.Join(reasons, ",") != "key_revoked" {
		t.Errorf("after a revoke: %s %v", st, reasons)
	}
	// Reconnect gives a new key with what it could do.
	var re secret
	c.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/reconnect", map[string]any{}).MustStatus(t, 200).JSON(t, &re)
	call(t, env, re.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "after.reconnect@fastmail.com"}, nil).MustStatus(t, 200)
}

func TestTakeOver(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	publicURL(env)
	m := env.Client()
	m.SignInDemo("marcus@acme.io")
	made := connect(t, m, "make", map[string]any{"items": []string{"upsert_contact", "record_event", "send_transactional"}})
	// Marcus leaves: the key loses its permissions and calls fail with 403.
	exec(t, env, `UPDATE workspace_members SET status = 'deactivated', deactivated_at = now() WHERE user_id = (SELECT id FROM users WHERE email = 'marcus@acme.io')`)
	if r := call(t, env, made.Secret, "GET", "/integration", nil, nil); r.Status != 403 || code(t, r) != "owner_deactivated" {
		t.Fatalf("owner left: %d %s", r.Status, r.Body)
	}
	// Daniel's custom role holds integrations:manage but not transactional:send.
	exec(t, env, `UPDATE roles SET permissions = permissions || '{integrations:manage,subscribers:create,subscribers:edit}' WHERE key = 'content-writer'`)
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	var prev struct {
		Held, Missing, Stops []string
		Key                  struct{ OwnerStatus string }
	}
	d.Do("GET", ws+"/integrations/connections/"+made.IntegrationID+"/take-over", nil).MustStatus(t, 200).JSON(t, &prev)
	if strings.Join(prev.Missing, ",") != "transactional:send" || strings.Join(prev.Stops, ",") != "send_transactional" || prev.Key.OwnerStatus != "deactivated" {
		t.Fatalf("take-over preview: %+v", prev)
	}
	d.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/take-over", nil).MustStatus(t, 200)
	// The key keeps its value and works again, limited by Daniel's permissions.
	call(t, env, made.Secret, "GET", "/integration", nil, nil).MustStatus(t, 200)
	var app struct {
		Items []struct {
			Key, OffReason, Missing string
			Available               bool
		}
		Key struct{ Owner struct{ Name string } }
	}
	d.Do("GET", ws+"/integrations/apps/make", nil).MustStatus(t, 200).JSON(t, &app)
	if app.Key.Owner.Name != "Daniel Park" {
		t.Errorf("owner: %+v", app.Key)
	}
	for _, it := range app.Items {
		if it.Key == "send_transactional" && (it.Available || it.OffReason != "owner_lacks" || it.Missing != "transactional:send") {
			t.Errorf("an action the new owner can't do: %+v", it)
		}
	}
	if audited(t, env, "integration.taken_over", "user") != 1 {
		t.Error("taking over is audited")
	}
	// Only members with integrations:manage take over.
	s := env.Client()
	s.SignInDemo("sofia@acme.io")
	if r := s.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/take-over", nil); r.Status != 403 {
		t.Errorf("an Editor takes over: %d", r.Status)
	}
}

func TestActionsKeepConsent(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	publicURL(env)
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	made := connect(t, c, "make", map[string]any{"items": []string{"upsert_contact", "add_to_list", "add_tag"}})
	beta := lists(t, env)["Beta Program"]
	var res struct {
		ContactID string
		Created   bool
		Lists     []struct{ Name, Status string }
	}
	call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "Rob@Example.net", "lists": []string{beta}, "consent": "single", "language": "es_MX"}, nil).
		MustStatus(t, 200).JSON(t, &res)
	if !res.Created || len(res.Lists) != 1 || res.Lists[0].Status != "subscribed" {
		t.Fatalf("created: %+v", res)
	}
	var lang, langSource, langBy string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT language, language_source, language_source_name FROM contacts WHERE id = $1`, res.ContactID).Scan(&lang, &langSource, &langBy)
	if lang != "es-MX" || langSource != "api" || langBy != "Make" {
		t.Errorf("language normalized and set by Make: %q %q %q", lang, langSource, langBy)
	}
	// A value that isn't a language answers 422 with the closest match.
	r := call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "rob@example.net", "language": "Spanosh"}, nil)
	if r.Status != 422 || !strings.Contains(string(r.Body), "Did you mean Spanish (es)?") || !strings.Contains(string(r.Body), `"closest":"es"`) {
		t.Fatalf("invalid language: %d %s", r.Status, r.Body)
	}
	// Rob leaves Beta Program.
	exec(t, env, `UPDATE list_memberships SET status = 'unsubscribed', unsubscribed_at = now() - interval '13 days' WHERE contact_id = $1`, res.ContactID)
	r = call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "rob@example.net", "lists": []string{beta}}, nil)
	if r.Status != 409 || code(t, r) != "unsubscribed" || !strings.Contains(string(r.Body), "rob@example.net left Beta Program on") ||
		!strings.Contains(string(r.Body), "ReputeCast doesn't add people back to a list they left.") {
		t.Fatalf("back to a list they left: %d %s", r.Status, r.Body)
	}
	r = call(t, env, made.Secret, "POST", ws+"/lists/"+beta+"/contacts", map[string]any{"email": "rob@example.net"}, nil)
	if r.Status != 409 || code(t, r) != "unsubscribed" {
		t.Errorf("add to a list they left: %d %s", r.Status, r.Body)
	}
	var status string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT status FROM list_memberships WHERE contact_id = $1`, res.ContactID).Scan(&status)
	if status != "unsubscribed" {
		t.Errorf("membership changed to %s", status)
	}
	// Updating Rob without lists still works.
	call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "rob@example.net", "firstName": "Rob"}, nil).MustStatus(t, 200)
	// A suppressed address is never added.
	exec(t, env, `INSERT INTO suppressions (id, workspace_id, email, reason) SELECT $1, id, 'bounced.before@fastmail.com', 'hard_bounce' FROM workspaces WHERE slug = 'acme-newsletter'`, uuid.Must(uuid.NewV7()))
	r = call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "bounced.before@fastmail.com", "lists": []string{beta}}, nil)
	if r.Status != 409 || code(t, r) != "suppressed" || !strings.Contains(string(r.Body), "hard bounce") {
		t.Fatalf("suppressed: %d %s", r.Status, r.Body)
	}
	// Tags merge; unknown contacts and fields are refused with what to do.
	call(t, env, made.Secret, "POST", ws+"/contacts/tags", map[string]any{"email": "rob@example.net", "tags": []string{"Customer"}}, nil).MustStatus(t, 200)
	if r := call(t, env, made.Secret, "POST", ws+"/contacts/tags", map[string]any{"email": "nobody@fastmail.com", "tags": []string{"x"}}, nil); r.Status != 404 {
		t.Errorf("tagging an unknown contact: %d", r.Status)
	}
	if r := call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "rob@example.net", "fields": map[string]any{"nope": 1}}, nil); r.Status != 422 {
		t.Errorf("an unknown field: %d", r.Status)
	}
	if audited(t, env, "contact.create", "api_key") != 1 || audited(t, env, "contact.update", "api_key") < 2 {
		t.Error("changes are audited with the key as actor")
	}
	var log struct{ Errors int64 }
	c.Do("GET", ws+"/integrations/connections/"+made.IntegrationID+"/calls?errors=true", nil).MustStatus(t, 200).JSON(t, &log)
	if log.Errors < 3 {
		t.Errorf("the refusals are in the call log: %d", log.Errors)
	}
}

// hookServer records deliveries and answers with status.
type hookServer struct {
	*httptest.Server
	status atomic.Int32
	hits   atomic.Int32
}

func newHookServer(t *testing.T) *hookServer {
	h := &hookServer{}
	h.status.Store(200)
	h.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.hits.Add(1)
		w.WriteHeader(int(h.status.Load()))
	}))
	t.Cleanup(h.Close)
	return h
}

func webhookService(env *apptest.Env) *webhooks.Service {
	return &webhooks.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return env.App.Cfg.Webhooks.AllowPrivateNetworks }, Managed: hubService(env)}
}

func hubService(env *apptest.Env) *integrations.Service {
	return &integrations.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Cfg: env.App.Cfg,
		PublicURL: env.App.URLs.Base, AllowPrivate: func() bool { return env.App.Cfg.Webhooks.AllowPrivateNetworks }}
}

func TestRESTHooks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	publicURL(env)
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	made := connect(t, c, "make", map[string]any{"items": []string{"contact.unsubscribed", "upsert_contact"}})
	srv := newHookServer(t)
	wf := map[string]string{"X-ReputeCast-Workflow": "Unsubscribes %E2%86%92 CRM; id=s-77"}
	// The SSRF guard refuses a private address until an instance admin allows it.
	r := call(t, env, made.Secret, "POST", ws+"/integration/hooks", map[string]any{"event": "contact.unsubscribed", "url": srv.URL + "/hook"}, wf)
	if r.Status != 422 || code(t, r) != "private_address" || !strings.Contains(string(r.Body), "Linh Nguyen") {
		t.Fatalf("private address: %d %s", r.Status, r.Body)
	}
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	// A trigger that isn't on can't be subscribed to.
	if r := call(t, env, made.Secret, "POST", ws+"/integration/hooks", map[string]any{"event": "contact.updated", "url": srv.URL + "/hook"}, wf); r.Status != 403 {
		t.Errorf("a trigger turned off: %d %s", r.Status, r.Body)
	}
	var hook struct{ ID, Secret, Event string }
	call(t, env, made.Secret, "POST", ws+"/integration/hooks", map[string]any{"event": "contact.unsubscribed", "url": srv.URL + "/hook"}, wf).MustStatus(t, 201).JSON(t, &hook)
	if !strings.HasPrefix(hook.Secret, "whsec_") {
		t.Fatalf("the secret goes once to the integration: %+v", hook)
	}
	if audited(t, env, "webhook.created", "api_key") != 2 {
		t.Error("a managed webhook created is audited with the key as actor")
	}
	// People see it as managed and can't change it by hand.
	l := env.Client()
	l.SignInDemo("linh@acme.io")
	var list struct {
		Items []struct {
			ID        string
			ManagedBy *struct{ Kind, Workflow string }
		}
	}
	l.Do("GET", ws+"/webhooks", nil).MustStatus(t, 200).JSON(t, &list)
	found := false
	for _, it := range list.Items {
		if it.ID == hook.ID {
			found = it.ManagedBy != nil && it.ManagedBy.Kind == "make" && it.ManagedBy.Workflow == "Unsubscribes → CRM"
		}
	}
	if !found {
		t.Errorf("managed endpoint on the Webhooks page: %+v", list.Items)
	}
	for _, p := range []struct{ method, path string }{{"PUT", "/webhooks/" + hook.ID}, {"DELETE", "/webhooks/" + hook.ID}, {"POST", "/webhooks/" + hook.ID + "/rotate-secret"}} {
		if r := l.Do(p.method, ws+p.path, map[string]any{"name": "x", "url": "https://example.com/x", "events": []string{"*"}}); r.Status != 409 || code(t, r) != "managed_by_integration" {
			t.Errorf("%s %s by hand: %d %s", p.method, p.path, r.Status, r.Body)
		}
	}
	// Other keys don't see it.
	var plain struct{ Secret string }
	l.Do("POST", ws+"/api-keys", map[string]any{"name": "ops script", "permissions": []string{"webhooks:manage"}}).MustStatus(t, 201).JSON(t, &plain)
	var keyList struct{ Items []struct{ ID string } }
	call(t, env, plain.Secret, "GET", ws+"/webhooks", nil, nil).MustStatus(t, 200).JSON(t, &keyList)
	for _, it := range keyList.Items {
		if it.ID == hook.ID {
			t.Error("another key sees the managed endpoint")
		}
	}
	// Deliveries go out signed; a 410 Gone removes the webhook, the system as actor.
	ctx := context.Background()
	wsRow, _ := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if _, err := webhooks.Emit(ctx, env.App.Pool, wsRow.ID, wsRow.Slug, "contact.unsubscribed", map[string]any{"email": "rob@example.net"}); err != nil {
		t.Fatal(err)
	}
	var delivery uuid.UUID
	hookID := uuid.MustParse(hook.ID)
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM webhook_deliveries WHERE webhook_id = $1`, hookID).Scan(&delivery); err != nil {
		t.Fatal(err)
	}
	srv.status.Store(410)
	if _, err := webhookService(env).Attempt(ctx, delivery); err != nil {
		t.Fatal(err)
	}
	if srv.hits.Load() != 1 {
		t.Fatalf("delivered %d times", srv.hits.Load())
	}
	var left int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhooks WHERE id = $1`, hookID).Scan(&left)
	if left != 0 || audited(t, env, "webhook.deleted", "system") != 1 {
		t.Errorf("410 Gone: %d left, system audit %d", left, audited(t, env, "webhook.deleted", "system"))
	}
	// Unsubscribing removes a hook, the key as actor.
	call(t, env, made.Secret, "POST", ws+"/integration/hooks", map[string]any{"event": "contact.unsubscribed", "url": srv.URL + "/hook2"}, wf).MustStatus(t, 201).JSON(t, &hook)
	call(t, env, made.Secret, "DELETE", ws+"/integration/hooks/"+hook.ID, nil, nil).MustStatus(t, 204)
	if audited(t, env, "webhook.deleted", "api_key") != 1 {
		t.Error("unsubscribing is audited with the key as actor")
	}
	// Samples for field mapping.
	var sample struct {
		Event   string
		Payload struct {
			Type string
			Data map[string]any
		}
	}
	call(t, env, made.Secret, "GET", ws+"/integration/samples/contact.unsubscribed", nil, nil).MustStatus(t, 200).JSON(t, &sample)
	if sample.Payload.Type != "contact.unsubscribed" || sample.Payload.Data["email"] == nil {
		t.Errorf("sample: %+v", sample)
	}
}

func TestDisconnect(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	publicURL(env)
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	made := connect(t, c, "make", map[string]any{"items": []string{"contact.updated", "upsert_contact"}})
	srv := newHookServer(t)
	var hook struct{ ID string }
	call(t, env, made.Secret, "POST", ws+"/integration/hooks", map[string]any{"event": "contact.updated", "url": srv.URL + "/h"},
		map[string]string{"X-ReputeCast-Workflow": "Sync to CRM; id=9"}).MustStatus(t, 201).JSON(t, &hook)
	call(t, env, made.Secret, "POST", ws+"/contacts/upsert", map[string]any{"email": "made.by.make@fastmail.com"}, map[string]string{"X-ReputeCast-Workflow": "Sync to CRM; id=9"}).MustStatus(t, 200)
	ctx := context.Background()
	wsRow, _ := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	_, _ = webhooks.Emit(ctx, env.App.Pool, wsRow.ID, wsRow.Slug, "contact.updated", map[string]any{"email": "x@y.z"})
	var prev struct {
		Workflows []struct{ Name string }
		Webhooks  int64
		Queued    int64
	}
	c.Do("GET", ws+"/integrations/connections/"+made.IntegrationID+"/disconnect", nil).MustStatus(t, 200).JSON(t, &prev)
	if len(prev.Workflows) != 1 || prev.Workflows[0].Name != "Sync to CRM" || prev.Webhooks != 1 || prev.Queued != 1 {
		t.Fatalf("what stops: %+v", prev)
	}
	c.Do("POST", ws+"/integrations/connections/"+made.IntegrationID+"/disconnect", map[string]any{"reason": "moved to n8n"}).MustStatus(t, 204)
	if r := call(t, env, made.Secret, "GET", "/integration", nil, nil); r.Status != 401 || code(t, r) != "key_revoked" {
		t.Errorf("the key after disconnecting: %d %s", r.Status, r.Body)
	}
	var hooks, deliveries, contacts int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhooks WHERE id = $1`, hook.ID).Scan(&hooks)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1`, hook.ID).Scan(&deliveries)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE email = 'made.by.make@fastmail.com'`).Scan(&contacts)
	if hooks != 0 || deliveries != 0 || contacts != 1 {
		t.Errorf("after disconnecting: %d webhooks, %d deliveries, %d contacts", hooks, deliveries, contacts)
	}
	var reason string
	_ = env.App.Pool.QueryRow(ctx, `SELECT reason FROM audit_events WHERE action = 'integration.disconnected'`).Scan(&reason)
	if reason != "moved to n8n" {
		t.Errorf("disconnect audited with its reason: %q", reason)
	}
	var cat catalog
	c.Do("GET", ws+"/integrations/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	if st, _, _ := cat.entry("make"); st != "not_connected" {
		t.Errorf("after disconnecting: %s", st)
	}
	// It can be connected again.
	connect(t, c, "make", map[string]any{"items": []string{"upsert_contact"}})
}

func TestZapierAndMakeNeedAPublicAddress(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	// The test instance listens on 127.0.0.1.
	var app struct {
		Reachability struct {
			Private, Reachable bool
			DNS                struct{ Status, Detail string }
		}
	}
	c.Do("GET", ws+"/integrations/apps/make", nil).MustStatus(t, 200).JSON(t, &app)
	if !app.Reachability.Private || app.Reachability.Reachable || !strings.Contains(app.Reachability.DNS.Detail, "private address") {
		t.Fatalf("reachability: %+v", app.Reachability)
	}
	r := c.Do("POST", ws+"/integrations/apps/make/connect", map[string]any{"items": []string{"upsert_contact"}})
	if r.Status != 409 || code(t, r) != "instance_unreachable" || !strings.Contains(string(r.Body), "use n8n on your own network") {
		t.Fatalf("Make on a private address: %d %s", r.Status, r.Body)
	}
	// n8n can, on the same network.
	c.Do("POST", ws+"/integrations/connections/"+connectionID(t, c, "n8n")+"/disconnect", map[string]any{}).MustStatus(t, 204)
	connect(t, c, "n8n", map[string]any{"items": []string{"upsert_contact"}, "address": "http://n8n.acme.internal:5678"})
	var check struct{ Private, Allowed bool }
	c.Do("POST", ws+"/integrations/address-check", map[string]any{"address": "http://n8n.acme.internal:5678"}).MustStatus(t, 200).JSON(t, &check)
	if !check.Private || check.Allowed {
		t.Errorf("a private n8n: %+v", check)
	}
	// The dialog's second step keeps n8n's address: checked, saved, audited.
	n8n := connectionID(t, c, "n8n")
	var set struct {
		Address string
		Private bool
	}
	c.Do("PUT", ws+"/integrations/connections/"+n8n+"/address", map[string]any{"address": "n8n.acme.internal:5679/"}).MustStatus(t, 200).JSON(t, &set)
	if set.Address != "https://n8n.acme.internal:5679" || !set.Private {
		t.Errorf("address: %+v", set)
	}
	var page struct{ Address string }
	c.Do("GET", ws+"/integrations/apps/n8n", nil).MustStatus(t, 200).JSON(t, &page)
	if page.Address != "https://n8n.acme.internal:5679" || audited(t, env, "integration.address_changed", "user") != 1 {
		t.Errorf("n8n's page address: %q", page.Address)
	}
	// The same address again changes nothing; only n8n has an address.
	c.Do("PUT", ws+"/integrations/connections/"+n8n+"/address", map[string]any{"address": "https://n8n.acme.internal:5679"}).MustStatus(t, 200)
	if audited(t, env, "integration.address_changed", "user") != 1 {
		t.Error("an unchanged address isn't audited")
	}
	c.Do("PUT", ws+"/integrations/connections/"+connectionID(t, c, "zapier")+"/address", map[string]any{"address": "https://x.example"}).MustStatus(t, 422)
}

func connectionID(t *testing.T, c *apptest.Client, kind string) string {
	t.Helper()
	var app struct{ ConnectionID string }
	c.Do("GET", ws+"/integrations/apps/"+kind, nil).MustStatus(t, 200).JSON(t, &app)
	return app.ConnectionID
}

func TestWordPressSiteAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	site := connect(t, c, "wordpress", map[string]any{"items": []string{"list_forms", "create_drafts"}, "address": "news.example.org"})
	if site.Key.Name != "WordPress · news.example.org" {
		t.Errorf("key name: %s", site.Key.Name)
	}
	var me struct{ Integration string }
	call(t, env, site.Secret, "GET", "/integration", nil, nil).MustStatus(t, 200).JSON(t, &me)
	if me.Integration != "wordpress" {
		t.Errorf("me: %+v", me)
	}
	report := map[string]any{"siteUrl": "https://news.example.org", "name": "Example News", "pluginVersion": "1.2.0", "wordpressVersion": "6.9.2", "phpVersion": "8.3",
		"feedUrl": "https://news.example.org/feed/", "categories": []map[string]string{{"id": "2", "name": "News", "slug": "news"}, {"id": "3", "name": "Jobs", "slug": "jobs"}},
		"forms": []map[string]any{{"form": "frm_8c2e1a", "kind": "block", "places": []map[string]string{{"title": "Sidebar"}}}}}
	var rep struct{ Workspace, Site, LatestPlugin string }
	call(t, env, site.Secret, "POST", ws+"/wordpress/report", report, map[string]string{"User-Agent": "ReputeCast-WordPress/1.2.0"}).MustStatus(t, 200).JSON(t, &rep)
	if rep.Workspace != "acme-newsletter" || rep.Site != "news.example.org" || rep.LatestPlugin != integrations.KnownPluginVersion {
		t.Errorf("report: %+v", rep)
	}
	if audited(t, env, "wordpress.site_connected", "api_key") != 1 {
		t.Error("a site connecting is audited, the site's key as actor")
	}
	var forms struct {
		Items []struct {
			ID, Shortcode string
			SiteAllowed   bool
		}
	}
	call(t, env, site.Secret, "GET", ws+"/wordpress/forms", nil, nil).MustStatus(t, 200).JSON(t, &forms)
	if len(forms.Items) != 3 || forms.Items[0].ID != "frm_8c2e1a" || forms.Items[0].SiteAllowed || forms.Items[0].Shortcode != `[reputecast_form id="frm_8c2e1a"]` {
		t.Errorf("forms: %+v", forms.Items)
	}
	// The key can't read contacts or open campaigns.
	for _, p := range []string{ws + "/contacts", ws + "/campaigns", ws + "/integration/lists"} {
		if r := call(t, env, site.Secret, "GET", p, nil, nil); r.Status != 403 || code(t, r) != "integration_operation_denied" {
			t.Errorf("GET %s with a site's key: %d %s", p, r.Status, r.Body)
		}
	}
	// People and other keys don't call the site API.
	if r := c.Do("POST", ws+"/wordpress/report", report); r.Status != 403 {
		t.Errorf("a person reports in: %d", r.Status)
	}
	post := map[string]any{"post": map[string]any{"id": "77", "url": "https://news.example.org/hello/", "title": "Hello & welcome", "excerpt": "Our first post.",
		"imageUrl": "https://news.example.org/hello.png", "categories": []string{"news"}, "publishedAt": time.Now().UTC()}}
	var draft struct {
		Created    bool
		Reason     string
		CampaignID string
	}
	call(t, env, site.Secret, "POST", ws+"/wordpress/drafts", post, nil).MustStatus(t, 200).JSON(t, &draft)
	if draft.Created || draft.Reason != "no_rule" {
		t.Errorf("without a rule: %+v", draft)
	}
	// Marcus sets the rule: posts in News become drafts for Product Updates.
	ids := lists(t, env)
	var tpl, sender, sofia string
	ctx := context.Background()
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM templates WHERE name = 'Blog post'`).Scan(&tpl)
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM senders WHERE email = 'product@news.acme.io' LIMIT 1`).Scan(&sender)
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'sofia@acme.io'`).Scan(&sofia)
	c.Do("PUT", ws+"/integrations/connections/"+site.IntegrationID+"/rule", map[string]any{"enabled": true, "categories": []string{"news"},
		"listIds": []string{ids["Product Updates"]}, "templateId": tpl, "senderId": sender, "notifyUserId": sofia}).MustStatus(t, 200)
	if audited(t, env, "wordpress.rule_changed", "user") != 1 {
		t.Error("the rule change is audited")
	}
	other := map[string]any{"post": map[string]any{"id": "78", "title": "We're hiring", "categories": []string{"jobs"}}}
	call(t, env, site.Secret, "POST", ws+"/wordpress/drafts", other, nil).MustStatus(t, 200).JSON(t, &draft)
	if draft.Created || draft.Reason != "category" {
		t.Errorf("a post outside the rule's categories: %+v", draft)
	}
	call(t, env, site.Secret, "POST", ws+"/wordpress/drafts", post, nil).MustStatus(t, 200).JSON(t, &draft)
	if !draft.Created || draft.CampaignID == "" {
		t.Fatalf("draft: %+v", draft)
	}
	var subject, preheader, status, html string
	_ = env.App.Pool.QueryRow(ctx, `SELECT subject, preheader, status, html FROM campaigns WHERE id = $1`, draft.CampaignID).Scan(&subject, &preheader, &status, &html)
	if subject != "Hello & welcome" || preheader != "Our first post." || status != "draft" || !strings.Contains(html, "Read more") || !strings.Contains(html, "Hello &amp; welcome") {
		t.Errorf("the draft: %q %q %q", subject, preheader, status)
	}
	var notified int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'post_draft_created' AND user_id = $1 AND body LIKE '%Hello & welcome%'`, sofia).Scan(&notified)
	if notified != 1 || audited(t, env, "wordpress.draft_created", "api_key") != 2 {
		t.Errorf("Sofia is notified (%d) and the draft audited", notified)
	}
	call(t, env, site.Secret, "POST", ws+"/wordpress/drafts", post, nil).MustStatus(t, 200).JSON(t, &draft)
	if draft.Created || draft.Reason != "exists" {
		t.Errorf("the same post again: %+v", draft)
	}
	// The sites page lists it with its rule and draft.
	var page struct {
		Sites []struct {
			Host, Status    string
			UpdateAvailable bool
			Forms           []struct{ Allowed bool }
			Drafts          []struct{ Title string }
			Rule            struct {
				Enabled bool
				List    struct{ Name string }
			}
		}
	}
	c.Do("GET", ws+"/integrations/wordpress", nil).MustStatus(t, 200).JSON(t, &page)
	var mine bool
	for _, s := range page.Sites {
		if s.Host == "news.example.org" {
			mine = s.Status == "connected" && s.UpdateAvailable && len(s.Forms) == 1 && !s.Forms[0].Allowed && len(s.Drafts) == 1 && s.Rule.List.Name == "Product Updates"
		}
	}
	if !mine || len(page.Sites) != 3 {
		t.Errorf("sites page: %+v", page.Sites)
	}
}

func TestCheckNowThroughTheSSRFGuard(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("marcus@acme.io")
	var slow atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slow.Load() {
			time.Sleep(400 * time.Millisecond)
		}
		if r.URL.Path != "/wp-json/reputecast/v1/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"plugin":"1.3.0"}`))
	}))
	defer srv.Close()
	site := connect(t, c, "wordpress", map[string]any{"items": []string{"list_forms"}, "address": srv.URL})
	var check struct {
		OK      bool
		Outcome string
		Ms      int64
		Detail  string
	}
	c.Do("POST", ws+"/integrations/connections/"+site.IntegrationID+"/check", nil).MustStatus(t, 200).JSON(t, &check)
	if check.OK || check.Outcome != "blocked" || !strings.HasPrefix(check.Detail, "Blocked:") {
		t.Fatalf("a loopback site without private addresses allowed: %+v", check)
	}
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c.Do("POST", ws+"/integrations/connections/"+site.IntegrationID+"/check", nil).MustStatus(t, 200).JSON(t, &check)
	if !check.OK || check.Outcome != "answered" || !strings.HasPrefix(check.Detail, "Answered in") {
		t.Fatalf("check now: %+v", check)
	}
	// A site that doesn't answer in time.
	s := hubService(env)
	s.CheckTimeout = 100 * time.Millisecond
	slow.Store(true)
	wsRow, _ := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	got, err := s.CheckSite(context.Background(), &wsRow, uuid.MustParse(site.IntegrationID))
	if err != nil || got.OK || got.Outcome != "timeout" || got.Seconds != 0.1 || got.Detail != "Timed out after 0.1 s" {
		t.Fatalf("timeout: %+v %v", got, err)
	}
	// The page keeps the last check, outcome and all.
	var page struct {
		Sites []struct {
			ID        string
			LastCheck *struct {
				Outcome string
				Seconds float64
			}
		}
	}
	c.Do("GET", ws+"/integrations/wordpress", nil).MustStatus(t, 200).JSON(t, &page)
	for _, st := range page.Sites {
		if st.ID == site.IntegrationID && (st.LastCheck == nil || st.LastCheck.Outcome != "timeout" || st.LastCheck.Seconds != 0.1) {
			t.Fatalf("last check: %+v", st.LastCheck)
		}
	}
	// The cloud metadata address is never reached, even with private networks allowed.
	meta := connect(t, c, "wordpress", map[string]any{"items": []string{"list_forms"}, "address": "http://169.254.169.254"})
	c.Do("POST", ws+"/integrations/connections/"+meta.IntegrationID+"/check", nil).MustStatus(t, 200).JSON(t, &check)
	if check.OK || !strings.HasPrefix(check.Detail, "Blocked:") {
		t.Errorf("metadata address: %+v", check)
	}
}

func TestHealthNotifiesOnce(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	s := hubService(env)
	count := func() int {
		var n int
		_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'integration_attention'`).Scan(&n)
		return n
	}
	before := count()
	if err := s.CheckHealth(ctx); err != nil {
		t.Fatal(err)
	}
	if count() != before {
		t.Error("staging.acme.io was already notified")
	}
	// blog.acme.io goes quiet.
	exec(t, env, `UPDATE wordpress_sites SET last_report_at = now() - interval '30 hours' WHERE address = 'blog.acme.io'`)
	if err := s.CheckHealth(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'integration_attention' AND body LIKE 'WordPress: blog.acme.io hasn''t reported in since%'`).Scan(&n)
	if n != 2 {
		t.Errorf("Linh and Marcus are told: %d", n)
	}
	if err := s.CheckHealth(ctx); err != nil {
		t.Fatal(err)
	}
	if count() != before+2 {
		t.Errorf("told once: %d", count()-before)
	}
}

func TestContactEventsBecomeDeliveries(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	l := env.Client()
	l.SignInDemo("linh@acme.io")
	var created struct{ Webhook struct{ ID string } }
	l.Do("POST", ws+"/webhooks", map[string]any{"name": "CRM", "url": "https://crm.example.com/hook", "events": []string{"contact.updated", "contact.unsubscribed"}}).
		MustStatus(t, 201).JSON(t, &created)
	exec(t, env, `UPDATE contacts SET first_name = 'Samuel' WHERE email = 'sam.rivera@outlook.com'`)
	exec(t, env, `UPDATE list_memberships SET status = 'unsubscribed' WHERE contact_id = (SELECT id FROM contacts WHERE email = 'sam.rivera@outlook.com') AND status = 'subscribed'`)
	n, err := hubService(env).DeliverContactEvents(ctx, 500)
	if err != nil || n < 2 {
		t.Fatalf("taken %d: %v", n, err)
	}
	var updated int
	var payload []byte
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND event = 'contact.updated'`, created.Webhook.ID).Scan(&updated)
	_ = env.App.Pool.QueryRow(ctx, `SELECT payload FROM webhook_deliveries WHERE webhook_id = $1 AND event = 'contact.unsubscribed' LIMIT 1`, created.Webhook.ID).Scan(&payload)
	var env1 struct {
		Type string
		Data map[string]any
	}
	_ = json.Unmarshal(payload, &env1)
	if updated != 1 || env1.Type != "contact.unsubscribed" || env1.Data["email"] != "sam.rivera@outlook.com" || env1.Data["list_name"] == nil {
		t.Errorf("deliveries: %d updated, payload %s", updated, payload)
	}
	// Nothing is queued while no endpoint listens.
	exec(t, env, `DELETE FROM webhooks`)
	exec(t, env, `UPDATE contacts SET first_name = 'Sam' WHERE email = 'sam.rivera@outlook.com'`)
	var queued int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_contact_events`).Scan(&queued)
	if queued != 0 {
		t.Errorf("queued without listeners: %d", queued)
	}
}

func TestMemberIntegrations(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	l := env.Client()
	l.SignInDemo("linh@acme.io")
	var marcus string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT id FROM users WHERE email = 'marcus@acme.io'`).Scan(&marcus)
	var owned struct {
		Items []struct {
			Kind, KeyName string
			Workflows     int64
		}
		CanManage bool
	}
	l.Do("GET", ws+"/integrations/owned-by/"+marcus, nil).MustStatus(t, 200).JSON(t, &owned)
	names := []string{}
	for _, it := range owned.Items {
		names = append(names, it.KeyName)
	}
	if strings.Join(names, ",") != "Zapier · Acme ops,WordPress · blog.acme.io,WordPress · staging.acme.io" || !owned.CanManage {
		t.Errorf("Marcus's integrations: %v", names)
	}
	a := env.Client()
	a.SignInDemo("aisha@acme.io")
	if r := a.Do("GET", ws+"/integrations/owned-by/"+marcus, nil); r.Status != 403 {
		t.Errorf("a Viewer: %d", r.Status)
	}
	// His member page lists his own API keys; the integrations' keys are
	// handed over in the dialogs, so "Revoke them now" leaves them alone.
	var member string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE u.email = 'marcus@acme.io'`).Scan(&member)
	var detail struct {
		APIKeys []struct{ Name string }
	}
	l.Do("GET", ws+"/members/"+member, nil).MustStatus(t, 200).JSON(t, &detail)
	for _, k := range detail.APIKeys {
		if strings.HasPrefix(k.Name, "Zapier · ") || strings.HasPrefix(k.Name, "WordPress · ") {
			t.Errorf("an integration's key among his API keys: %s", k.Name)
		}
	}
	l.Do("POST", ws+"/members/"+member+"/deactivate", map[string]any{"revokeKeys": true}).MustStatus(t, 200)
	var revoked int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM api_keys WHERE integration_id IS NOT NULL AND revoked_at IS NOT NULL`).Scan(&revoked)
	if revoked != 0 {
		t.Errorf("deactivating revoked %d integration keys", revoked)
	}
}

func TestDemoPages(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var zap struct {
		Status  string
		Numbers struct {
			Calls7d, Errors7d, Fixed7d, ConsentKept7d int64
			LastCallWorkflow                          string
		}
		Workflows []struct {
			Name              string
			Calls7d, Errors7d int64
			ConsentKept7d     int64
			FixedAt           *time.Time
		}
		Items []struct {
			Key       string
			Available bool
		}
	}
	c.Do("GET", ws+"/integrations/apps/zapier", nil).MustStatus(t, 200).JSON(t, &zap)
	if zap.Status != "connected" || zap.Numbers.Calls7d != 169 || zap.Numbers.Errors7d != 3 || zap.Numbers.Fixed7d != 2 || zap.Numbers.ConsentKept7d != 1 ||
		zap.Numbers.LastCallWorkflow != "Beta application (Typeform) → ReputeCast contact" {
		t.Errorf("Zapier's numbers: %+v", zap.Numbers)
	}
	if len(zap.Workflows) != 3 || zap.Workflows[0].Calls7d != 128 {
		t.Errorf("Zaps: %+v", zap.Workflows)
	}
	available := 0
	for _, it := range zap.Items {
		if it.Available {
			available++
		}
	}
	if available != 11 {
		t.Errorf("11 of 11 available: %d", available)
	}
	var calls struct {
		Total, Errors int64
		Items         []struct {
			Code       string
			Resolution *struct {
				FixedBy   string
				LaterRuns []time.Time
			}
		}
	}
	c.Do("GET", ws+"/integrations/connections/"+connectionID(t, c, "zapier")+"/calls?errors=true", nil).MustStatus(t, 200).JSON(t, &calls)
	if len(calls.Items) != 3 || calls.Items[0].Resolution == nil || calls.Items[0].Resolution.FixedBy != "Marcus Chen" || calls.Items[2].Code != "unsubscribed" {
		t.Errorf("errors: %+v", calls.Items)
	}
	var n8n struct {
		Webhooks []struct {
			Event, Endpoint, Status string
			Delivered24h            int64
		}
		Workflows []struct {
			Name         string
			Deliveries7d int64
			LastRunAt    time.Time
			Uses         []struct {
				Item, Event    string
				Starts, GoalOf []string
			}
		}
	}
	c.Do("GET", ws+"/integrations/apps/n8n", nil).MustStatus(t, 200).JSON(t, &n8n)
	if len(n8n.Webhooks) != 1 || n8n.Webhooks[0].Delivered24h != 14 || n8n.Webhooks[0].Endpoint != "https://n8n.acme.io/webhook/7c1e…/reputecast" {
		t.Errorf("n8n's webhook: %+v", n8n.Webhooks)
	}
	// "Record an event workspace.created, which starts the Onboarding
	// automation"; teammate.invited is its goal.
	if len(n8n.Workflows) != 3 || len(n8n.Workflows[0].Uses) != 1 || n8n.Workflows[0].Uses[0].Event != "workspace.created" ||
		strings.Join(n8n.Workflows[0].Uses[0].Starts, ",") != "Onboarding · new workspace" ||
		strings.Join(n8n.Workflows[1].Uses[0].GoalOf, ",") != "Onboarding · new workspace" || n8n.Workflows[2].Deliveries7d != 14 ||
		time.Since(n8n.Workflows[2].LastRunAt) > time.Hour {
		t.Errorf("n8n's workflows: %+v", n8n.Workflows)
	}
	// The delivery log shows n8n's endpoint masked, as its page does.
	var log struct {
		Items []struct{ Endpoint string }
	}
	c.Do("GET", ws+"/webhooks/deliveries?limit=200", nil).MustStatus(t, 200).JSON(t, &log)
	masked := 0
	for _, d := range log.Items {
		if strings.Contains(d.Endpoint, "7c1e9d42") {
			t.Fatalf("delivery shows the full n8n address: %s", d.Endpoint)
		}
		if strings.Contains(d.Endpoint, "n8n.acme.io/webhook/7c1e…/reputecast") {
			masked++
		}
	}
	if masked == 0 {
		t.Errorf("no n8n deliveries in the log: %+v", log.Items)
	}
	// SettingsAudit's "Integrations" filter: the hub's actions, by group.
	var audit struct {
		Items []struct{ Action string }
	}
	c.Do("GET", ws+"/audit?category=integrations&range=all&limit=200", nil).MustStatus(t, 200).JSON(t, &audit)
	groups := map[string]bool{}
	for _, it := range audit.Items {
		g, _, _ := strings.Cut(it.Action, ".")
		groups[g] = true
	}
	if !groups["integration"] || !groups["wordpress"] || groups["webhook"] || groups["campaign"] {
		t.Errorf("integrations category: %v", groups)
	}
	var beta struct {
		Workflows []struct {
			Uses []struct {
				Item, OptIn string
				Lists, Tags []string
			}
		}
	}
	c.Do("GET", ws+"/integrations/apps/zapier", nil).MustStatus(t, 200).JSON(t, &beta)
	if u := beta.Workflows[0].Uses; len(u) != 1 || u[0].Item != "upsert_contact" || u[0].OptIn != "double" || strings.Join(u[0].Lists, ",") != "Beta Program" || strings.Join(u[0].Tags, ",") != "beta-applicant" {
		t.Errorf("the Beta Zap's use: %+v", u)
	}
}

func TestSendTransactionalEmail(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	// A published template with a sensitive variable (the starter's
	// reset_url), next to the demo's own templates.
	var tpl struct{ DraftRevision int }
	c.Do("POST", ws+"/transactional/templates", map[string]any{"name": "Zap reset", "key": "zap-reset", "starter": "password-reset"}).MustStatus(t, 201).JSON(t, &tpl)
	c.Do("POST", ws+"/transactional/templates/zap-reset/publish", map[string]any{"baseRevision": tpl.DraftRevision}).MustStatus(t, 200)

	// n8n, reconnected with "Send a transactional email" only.
	c.Do("POST", ws+"/integrations/connections/"+connectionID(t, c, "n8n")+"/disconnect", map[string]any{}).MustStatus(t, 204)
	key := connect(t, c, "n8n", map[string]any{"items": []string{"send_transactional"}})

	// The app's dropdown: published templates and their variables.
	var list struct {
		Items []struct {
			Key       string
			Version   int
			Variables []struct {
				Name      string
				Sensitive bool
			}
		}
	}
	call(t, env, key.Secret, "GET", ws+"/integration/templates", nil, nil).MustStatus(t, 200).JSON(t, &list)
	mine := -1
	for i, it := range list.Items {
		if it.Key == "zap-reset" {
			mine = i
		}
	}
	if mine < 0 || list.Items[mine].Version != 1 {
		t.Fatalf("templates: %+v", list.Items)
	}
	sensitive := false
	for _, v := range list.Items[mine].Variables {
		if v.Name == "reset_url" && v.Sensitive {
			sensitive = true
		}
	}
	if !sensitive {
		t.Errorf("reset_url is sensitive: %+v", list.Items[mine].Variables)
	}

	// Sending goes through X02's send API; whatever it answers, the call
	// log keeps the request with the sensitive variable masked.
	r := call(t, env, key.Secret, "POST", ws+"/transactional/messages",
		map[string]any{"template": "zap-reset", "to": map[string]any{"email": "sam.rivera@outlook.com"},
			"data": map[string]any{"first_name": "Sam", "reset_url": "https://app.example.com/reset/secret-token", "expires_in": "30 minutes"}},
		map[string]string{"X-ReputeCast-Workflow": "Acme Cloud: password reset; id=wf_r1", "Idempotency-Key": "reset-1"})
	if r.Status == http.StatusForbidden && code(t, r) == "integration_operation_denied" {
		t.Fatalf("send refused for an integration key with the action on: %s", r.Body)
	}
	var stored, item string
	if err := env.App.Pool.QueryRow(ctx, `SELECT request::text, item FROM integration_calls WHERE operation = 'send-transactional-message'`).Scan(&stored, &item); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "secret-token") || !strings.Contains(stored, integrations.Masked) || item != "send_transactional" ||
		strings.Index(stored, `"template"`) > strings.Index(stored, `"data"`) {
		t.Errorf("logged request: %s (%s)", stored, item)
	}

	// With the action turned off, the same key can't list templates.
	c.Do("POST", ws+"/integrations/connections/"+key.IntegrationID+"/change", map[string]any{"items": []string{"add_tag"}}).MustStatus(t, 200)
	r = call(t, env, key.Secret, "GET", ws+"/integration/templates", nil, nil)
	if r.Status != http.StatusForbidden || code(t, r) != "integration_operation_denied" {
		t.Errorf("templates without the action: %d %s", r.Status, r.Body)
	}
}
