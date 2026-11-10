package apikeys_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apikeys"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

type keyView struct {
	ID         string
	AllowedIPs []string `json:"allowedIps"`
}

func errorsAt(t *testing.T, r *apptest.Response) []string {
	t.Helper()
	var p struct {
		Errors []struct{ Location string }
	}
	r.JSON(t, &p)
	out := []string{}
	for _, e := range p.Errors {
		out = append(out, e.Location)
	}
	return out
}

// A key limited to allowed IPs answers only calls from them (the test
// client calls from 127.0.0.1, no proxy trusted), and the refusals are
// counted like those of expired keys.
func TestKeyAllowedIPs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var list struct {
		YourIP       string `json:"yourIp"`
		ProxyWarning bool   `json:"proxyWarning"`
	}
	c.Do("GET", ws+"/api-keys", nil).MustStatus(t, 200).JSON(t, &list)
	if list.YourIP != "127.0.0.1" || list.ProxyWarning {
		t.Fatalf("your address: %+v", list)
	}
	// A proxy ReputeCast doesn't trust forwards requests: the addresses it
	// sees may be the proxy's.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO untrusted_proxies (peer, header, requests, first_seen, last_seen) VALUES ('10.0.0.5', 'X-Forwarded-For', 40, now(), now())`); err != nil {
		t.Fatal(err)
	}
	c.Do("GET", ws+"/mcp-agents", nil).MustStatus(t, 200).JSON(t, &list)
	if !list.ProxyWarning {
		t.Fatal("no proxy warning while an untrusted proxy forwards requests")
	}

	var k created
	var kv struct{ Key keyView }
	r := c.Do("POST", ws+"/api-keys", map[string]any{"name": "Office sync", "permissions": []string{"campaigns:view"}, "allowedIps": []string{"203.0.113.9/24", " 203.0.113.0/24"}}).MustStatus(t, 201)
	r.JSON(t, &k)
	r.JSON(t, &kv)
	if !slices.Equal(kv.Key.AllowedIPs, []string{"203.0.113.0/24"}) {
		t.Fatalf("created with allowed IPs %v", kv.Key.AllowedIPs)
	}
	b := env.Client()
	b.Bearer = k.Secret
	if p := b.Do("GET", ws+"/campaigns", nil).MustStatus(t, 403).Problem(t); p["code"] != "ip_not_allowed" {
		t.Fatalf("a call from outside the allowed IPs: %v", p)
	}
	var reason, ip string
	if err := env.App.Pool.QueryRow(ctx, `SELECT reason, ip FROM api_key_rejections WHERE api_key_id = $1`, uuid.MustParse(k.Key.ID)).Scan(&reason, &ip); err != nil || reason != "ip_not_allowed" || ip != "127.0.0.1" {
		t.Fatalf("rejection: %q %q %v", reason, ip, err)
	}

	// Invalid entries are refused, each at its own index.
	for bad, want := range map[string][]string{
		`["not-an-ip"]`:                          {"body.allowedIps[0]"},
		`["127.0.0.1", "::ffff:203.0.113.0/90"]`: {"body.allowedIps[1]"},
		`["fe80::1%eth0", "127.0.0.1", "x/24"]`:  {"body.allowedIps[0]", "body.allowedIps[2]"},
	} {
		var list []string
		_ = json.Unmarshal([]byte(bad), &list)
		r := c.Do("PUT", ws+"/api-keys/"+k.Key.ID+"/allowed-ips", map[string]any{"allowedIps": list}).MustStatus(t, 422)
		if at := errorsAt(t, r); !slices.Equal(at, want) {
			t.Errorf("%s: errors at %v, want %v", bad, at, want)
		}
	}
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Bad", "permissions": []string{"campaigns:view"}, "allowedIps": []string{"203.0.113.300"}}).MustStatus(t, 422)

	// Allowing this address lets the key in, audited with before and after.
	var v keyView
	c.Do("PUT", ws+"/api-keys/"+k.Key.ID+"/allowed-ips", map[string]any{"allowedIps": []string{"127.0.0.1", "2001:db8::/48"}}).MustStatus(t, 200).JSON(t, &v)
	if !slices.Equal(v.AllowedIPs, []string{"127.0.0.1", "2001:db8::/48"}) {
		t.Fatalf("saved: %v", v.AllowedIPs)
	}
	b.Do("GET", ws+"/campaigns", nil).MustStatus(t, 200)
	var before, after []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT before, after FROM audit_events WHERE action = 'apikey.updated' AND target_id = $1`, k.Key.ID).Scan(&before, &after); err != nil {
		t.Fatal(err)
	}
	var bm, am struct {
		AllowedIPs []string `json:"allowed_ips"`
	}
	_ = json.Unmarshal(before, &bm)
	_ = json.Unmarshal(after, &am)
	if !slices.Equal(bm.AllowedIPs, []string{"203.0.113.0/24"}) || !slices.Equal(am.AllowedIPs, v.AllowedIPs) {
		t.Fatalf("audit: before %s after %s", before, after)
	}

	// Rotating keeps the list.
	var rot struct{ Key keyView }
	c.Do("POST", ws+"/api-keys/"+k.Key.ID+"/rotate", nil).MustStatus(t, 200).JSON(t, &rot)
	if !slices.Equal(rot.Key.AllowedIPs, v.AllowedIPs) {
		t.Fatalf("rotated: %v", rot.Key.AllowedIPs)
	}
	var stored []netip.Prefix
	if err := env.App.Pool.QueryRow(ctx, `SELECT allowed_ips FROM api_keys WHERE id = $1`, uuid.MustParse(rot.Key.ID)).Scan(&stored); err != nil || len(stored) != 2 {
		t.Fatalf("rotated key's stored list: %v %v", stored, err)
	}

	// A revoked key keeps its list, an agent isn't an API key, an
	// integration's key never gets a list.
	c.Do("PUT", ws+"/api-keys/"+k.Key.ID+"/allowed-ips", map[string]any{"allowedIps": []string{}}).MustStatus(t, 409)
	c.Do("PUT", ws+"/mcp-agents/"+rot.Key.ID+"/allowed-ips", map[string]any{"allowedIps": []string{}}).MustStatus(t, 404)
	var integ uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT k.id FROM api_keys k JOIN workspaces w ON w.id = k.workspace_id
		WHERE w.slug = 'acme-newsletter' AND k.integration_id IS NOT NULL AND k.revoked_at IS NULL LIMIT 1`).Scan(&integ); err != nil {
		t.Fatalf("the demo's integration keys: %v", err)
	}
	if p := c.Do("PUT", ws+"/api-keys/"+integ.String()+"/allowed-ips", map[string]any{"allowedIps": []string{"127.0.0.1"}}).MustStatus(t, 409).Problem(t); p["code"] != "managed_by_integration" {
		t.Fatalf("integration key: %v", p)
	}

	// Emptying the list allows any address again.
	c.Do("PUT", ws+"/api-keys/"+rot.Key.ID+"/allowed-ips", map[string]any{"allowedIps": []string{}}).MustStatus(t, 200).JSON(t, &v)
	if v.AllowedIPs == nil || len(v.AllowedIPs) != 0 {
		t.Fatalf("emptied: %#v", v.AllowedIPs)
	}

	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("PUT", ws+"/api-keys/"+rot.Key.ID+"/allowed-ips", map[string]any{"allowedIps": []string{"127.0.0.1"}}).MustStatus(t, 403)
	viewer.Do("PUT", ws+"/mcp-agents/"+rot.Key.ID+"/allowed-ips", map[string]any{"allowedIps": []string{"127.0.0.1"}}).MustStatus(t, 403)
}

// Narrowing a key's allowed IPs needs only apikeys:manage; letting it be
// called from more addresses needs what rotating it needs.
func TestAllowedIPsEscalation(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	linh.Do("POST", ws+"/roles", map[string]any{"name": "Integrations", "permissions": []string{"apikeys:manage", "reports:view"}}).MustStatus(t, 201)
	w, _ := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	role, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: w.ID, Key: "integrations"})
	if err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "ivy@acme.io", Name: "Ivy Integrations"})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: w.ID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	var k created
	linh.Do("POST", ws+"/api-keys", map[string]any{"name": "Exporter", "permissions": []string{"subscribers:view"},
		"allowedIps": []string{"203.0.113.0/24", "198.51.100.0/24"}}).MustStatus(t, 201).JSON(t, &k)
	ivy := env.Client()
	ivy.SignInDemo("ivy@acme.io")
	path := ws + "/api-keys/" + k.Key.ID + "/allowed-ips"
	ivy.Do("PUT", path, map[string]any{"allowedIps": []string{"203.0.113.0/24"}}).MustStatus(t, 200)
	ivy.Do("PUT", path, map[string]any{"allowedIps": []string{"203.0.113.128/25"}}).MustStatus(t, 200)
	for _, wider := range [][]string{{}, {"203.0.113.0/24"}, {"203.0.113.128/25", "192.0.2.7"}} {
		if p := ivy.Do("PUT", path, map[string]any{"allowedIps": wider}).MustStatus(t, 403).Problem(t); p["code"] != "escalation_denied" || p["permission"] != "subscribers:view" {
			t.Fatalf("widening to %v: %v", wider, p)
		}
	}
	linh.Do("PUT", path, map[string]any{"allowedIps": []string{}}).MustStatus(t, 200)
}

// A key keeps at most 1,000 refused addresses a day, IPv6 ones as their
// /64; an address already counted keeps counting.
func TestRejectionsAreCapped(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var k created
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "Flooded", "permissions": []string{"campaigns:view"}}).MustStatus(t, 201).JSON(t, &k)
	id := uuid.MustParse(k.Key.ID)
	first := netip.MustParseAddr("198.51.100.4")
	apikeys.RecordRejection(ctx, env.App.Q, id, &first, "curl/8.7.1", "expired")
	for i := range 1000 {
		ip := netip.MustParseAddr(fmt.Sprintf("2001:db8:%x::%x", i, i+1))
		apikeys.RecordRejection(ctx, env.App.Q, id, &ip, "", "ip_not_allowed")
	}
	apikeys.RecordRejection(ctx, env.App.Q, id, &first, "", "ip_not_allowed")
	var rows, calls int
	var reason, client string
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM api_key_rejections WHERE api_key_id = $1`, id).Scan(&rows); err != nil || rows != 1000 {
		t.Fatalf("rows: %d %v", rows, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT calls, reason, client FROM api_key_rejections WHERE api_key_id = $1 AND ip = '198.51.100.4'`, id).Scan(&calls, &reason, &client); err != nil ||
		calls != 2 || reason != "ip_not_allowed" || client != "curl" {
		t.Fatalf("repeat: %d %q %q %v", calls, reason, client, err)
	}
	var masked int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM api_key_rejections WHERE api_key_id = $1 AND ip = '2001:db8:1::/64'`, id).Scan(&masked); err != nil || masked != 1 {
		t.Fatalf("IPv6 kept as its /64: %d %v", masked, err)
	}
}
