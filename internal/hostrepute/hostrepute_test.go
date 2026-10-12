package hostrepute_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/hostrepute/hrtest"
)

type connView struct {
	Status          string
	Method          string
	Demo            bool
	TeamName        string
	ConnectedBy     string
	MonitorLimit    *int
	MonitorUsed     int
	TokenRejectedAt *time.Time
	SyncError       string
	Scopes          []struct {
		Key     string
		Granted bool
	}
	Assets []struct {
		Asset, Role, Provider string
		Monitored             bool
		Listings              []string
		DMARCPolicy           string `json:"dmarcPolicy"`
	}
	Shared struct{ Domains, IPs []string }
}

func ws(env *apptest.Env) uuid.UUID {
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&id); err != nil {
		env.T.Fatal(err)
	}
	return id
}

func auditCount(env *apptest.Env, action, result string) int {
	var n int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND result = $2`, action, result).Scan(&n)
	return n
}

func TestIntegrationsOverview(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var out struct {
		HostRepute connView
		Postmaster struct{ Via string }
	}
	r := c.Do("GET", "/workspaces/acme-newsletter/integrations", nil).MustStatus(t, 200)
	r.JSON(t, &out)
	hr := out.HostRepute
	if hr.Status != "connected" || hr.TeamName != "Acme Inc" || hr.ConnectedBy != "Linh Nguyen" || *hr.MonitorLimit != 3 || hr.MonitorUsed != 3 || !hr.Demo {
		t.Fatalf("connection: %+v", hr)
	}
	if len(hr.Scopes) != 3 || !hr.Scopes[0].Granted || hr.Scopes[0].Key != "monitor:read" {
		t.Fatalf("scopes: %+v", hr.Scopes)
	}
	want := []string{"news.acme.io", "acme.io", "203.0.113.25", "mta.acme.io"}
	for i, a := range hr.Assets {
		if a.Asset != want[i] {
			t.Fatalf("asset order: %+v", hr.Assets)
		}
	}
	if !hr.Assets[0].Monitored || hr.Assets[3].Monitored || hr.Assets[3].Role != "server_hostname" || hr.Assets[3].Provider != "Postal" {
		t.Fatalf("monitoring: %+v", hr.Assets)
	}
	if len(hr.Assets[2].Listings) != 1 || hr.Assets[2].Listings[0] != "UCEPROTECT Level 3" || hr.Assets[1].DMARCPolicy != "quarantine" {
		t.Fatalf("statuses: %+v", hr.Assets)
	}
	if strings.Join(hr.Shared.Domains, ",") != "news.acme.io,acme.io" || strings.Join(hr.Shared.IPs, ",") != "203.0.113.25" {
		t.Fatalf("shared: %+v", hr.Shared)
	}
	if out.Postmaster.Via != "hostrepute" {
		t.Fatalf("postmaster via: %+v", out.Postmaster)
	}
	// Tokens are write-only.
	if strings.Contains(string(r.Body), "hr_demo") || strings.Contains(strings.ToLower(string(r.Body)), "token\"") {
		t.Fatalf("token leaked: %s", r.Body)
	}

	// Viewers can see the page but not change the connection.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", "/workspaces/acme-newsletter/integrations", nil).MustStatus(t, 200)
	p := v.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/assets", map[string]any{"asset": "mta.acme.io"}).MustStatus(t, 403).Problem(t)
	if p["permission"] != "integrations:hostrepute" {
		t.Fatalf("denial: %+v", p)
	}
	v.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 403)
	v.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/oauth", nil).MustStatus(t, 403)
}

func TestMonitoringLimitAndSwap(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	p := c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/assets", map[string]any{"asset": "mta.acme.io"}).MustStatus(t, 409).Problem(t)
	meta, _ := p["meta"].(map[string]any)
	if p["code"] != "monitor_limit_reached" || meta["limit"] != float64(3) || meta["used"] != float64(3) || meta["swap"] != "acme.io" || meta["team"] != "Acme Inc" {
		t.Fatalf("limit problem: %+v", p)
	}
	if !strings.Contains(p["detail"].(string), "This is a HostRepute limit, not a ReputeCast one") {
		t.Fatalf("detail: %v", p["detail"])
	}
	if auditCount(env, "hostrepute.asset_add", "error") != 1 {
		t.Fatal("refused add is audited")
	}
	// Only the workspace's own assets can be sent to HostRepute.
	c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/assets", map[string]any{"asset": "example.org"}).MustStatus(t, 422)

	var hr connView
	c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/assets/swap", map[string]any{"remove": "acme.io", "add": "mta.acme.io"}).MustStatus(t, 200).JSON(t, &hr)
	monitored := map[string]bool{}
	for _, a := range hr.Assets {
		monitored[a.Asset] = a.Monitored
	}
	if !monitored["mta.acme.io"] || monitored["acme.io"] || hr.MonitorUsed != 3 {
		t.Fatalf("after swap: %+v", hr.Assets)
	}
	if auditCount(env, "hostrepute.asset_swap", "allowed") != 1 {
		t.Fatal("swap is audited")
	}
	var jobs int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM river_job WHERE kind = 'hostrepute.sync'`).Scan(&jobs)
	if jobs == 0 {
		t.Fatal("a sync is queued after a monitoring change")
	}
}

func TestDisconnect(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)
	var out struct{ HostRepute connView }
	c.Do("GET", "/workspaces/acme-newsletter/integrations", nil).MustStatus(t, 200).JSON(t, &out)
	if out.HostRepute.Status != "not_connected" || len(out.HostRepute.Assets) != 4 || out.HostRepute.Assets[0].Monitored {
		t.Fatalf("after disconnect: %+v", out.HostRepute)
	}
	ctx := context.Background()
	var synced, builtin, runs int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM blocklist_results WHERE via = 'hostrepute'`).Scan(&synced)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM blocklist_results WHERE via = 'builtin'`).Scan(&builtin)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM deliverability_check_runs WHERE via = 'builtin' AND trigger = 'fallback'`).Scan(&runs)
	if synced != 0 || builtin == 0 || runs != 1 {
		t.Fatalf("HostRepute data forgotten (%d), built-in kept (%d) and queued (%d)", synced, builtin, runs)
	}
	var before string
	_ = env.App.Pool.QueryRow(ctx, `SELECT before::text FROM audit_events WHERE action = 'hostrepute.disconnect'`).Scan(&before)
	if !strings.Contains(before, "Acme Inc") || strings.Contains(before, "hr_demo") {
		t.Fatalf("disconnect audit: %s", before)
	}
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 404)
}

func TestConnectWithAPIKey(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	fake := hrtest.New(t, "hr_live_9f8e7d")
	fake.Monitored = nil // a new HostRepute team monitors nothing yet
	env.App.Cfg.HostRepute.APIURL = fake.APIURL()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)

	p := c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/api-key", map[string]any{"apiKey": "hr_live_wrong"}).MustStatus(t, 422).Problem(t)
	if !strings.Contains(p["detail"].(string), "didn't accept this key") {
		t.Fatalf("wrong key: %+v", p)
	}
	r := c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/api-key", map[string]any{"apiKey": "hr_live_9f8e7d"}).MustStatus(t, 200)
	if strings.Contains(string(r.Body), "hr_live_9f8e7d") {
		t.Fatal("the key is never returned")
	}
	var hr connView
	r.JSON(t, &hr)
	if hr.Status != "connected" || hr.Method != "api_key" || hr.Demo || hr.MonitorUsed != 3 {
		t.Fatalf("connected: %+v", hr)
	}
	// Connecting shares exactly the sending domains, organization domain and dedicated IP.
	var shared []string
	for _, b := range fake.Bodies {
		shared = append(shared, b)
	}
	got := strings.Join(shared, " ")
	for _, want := range []string{`"asset":"news.acme.io"`, `"asset":"acme.io"`, `"asset":"203.0.113.25"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %s to be added to monitoring, got %s", want, got)
		}
	}
	if strings.Contains(got, "@") || strings.Contains(got, "mta.acme.io") {
		t.Fatalf("only domains and IPs are shared: %s", got)
	}
	var enc []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT token_enc FROM hostrepute_connections`).Scan(&enc); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(enc), "hr_live_9f8e7d") {
		t.Fatal("the key is encrypted at rest")
	}
	if auditCount(env, "hostrepute.connect", "allowed") != 1 {
		t.Fatal("connect is audited")
	}

	fake.Down = true
	p = c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/api-key", map[string]any{"apiKey": "hr_live_9f8e7d"}).MustStatus(t, 503).Problem(t)
	if !strings.Contains(p["detail"].(string), "Couldn't reach 127.0.0.1") || !strings.Contains(p["detail"].(string), "3 attempts") {
		t.Fatalf("unreachable: %+v", p)
	}
}

func TestOAuthHandoff(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	fake := hrtest.New(t, "oauth-access-1")
	cfg := &env.App.Cfg.HostRepute
	cfg.APIURL, cfg.AuthURL, cfg.TokenURL, cfg.RevokeURL = fake.APIURL(), fake.URL+"/oauth/authorize", fake.URL+"/oauth/token", fake.URL+"/oauth/revoke"
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)

	var start struct{ AuthorizeURL string }
	c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/oauth", nil).MustStatus(t, 200).JSON(t, &start)
	if !strings.HasPrefix(start.AuthorizeURL, fake.URL+"/oauth/authorize?") {
		t.Fatalf("authorize URL: %s", start.AuthorizeURL)
	}
	u, _ := url.Parse(start.AuthorizeURL)
	if u.Query().Get("redirect_uri") != env.Server.URL+hostrepute.CallbackPath {
		t.Fatalf("redirect_uri: %s", u.Query().Get("redirect_uri"))
	}
	code, state := fake.Consent(t, start.AuthorizeURL)

	// Someone else can't finish the handoff with the state.
	other := env.Client()
	other.SignInDemo("marcus@acme.io")
	r := other.Do("GET", hostrepute.CallbackPath+"?code="+code+"&state="+state, nil).MustStatus(t, http.StatusFound)
	if loc := r.Header.Get("Location"); loc != "/w/acme-newsletter/deliverability/connect?result=error&reason=wrong_user" {
		t.Fatalf("wrong user: %s", loc)
	}
	// The state was consumed: start again as Linh.
	c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/oauth", nil).MustStatus(t, 200).JSON(t, &start)
	code, state = fake.Consent(t, start.AuthorizeURL)
	r = c.Do("GET", hostrepute.CallbackPath+"?code="+code+"&state="+state, nil).MustStatus(t, http.StatusFound)
	if loc := r.Header.Get("Location"); loc != "/w/acme-newsletter/deliverability/connect?result=connected" {
		t.Fatalf("callback: %s", loc)
	}
	var hr connView
	c.Do("GET", "/workspaces/acme-newsletter/integrations", nil).MustStatus(t, 200).JSON(t, &struct{ HostRepute *connView }{&hr})
	if hr.Status != "connected" || hr.Method != "oauth" || hr.TeamName != "Acme Inc" {
		t.Fatalf("after OAuth: %+v", hr)
	}
	// Reusing the state fails.
	r = c.Do("GET", hostrepute.CallbackPath+"?code="+code+"&state="+state, nil).MustStatus(t, http.StatusFound)
	if loc := r.Header.Get("Location"); loc != "/?hostrepute=expired" {
		t.Fatalf("replayed state: %s", loc)
	}
	// Declining on HostRepute comes back as an error and changes nothing.
	c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/oauth", nil).MustStatus(t, 200).JSON(t, &start)
	_, state = fake.Consent(t, start.AuthorizeURL)
	r = c.Do("GET", hostrepute.CallbackPath+"?error=access_denied&state="+state, nil).MustStatus(t, http.StatusFound)
	if loc := r.Header.Get("Location"); loc != "/w/acme-newsletter/deliverability/connect?result=error&reason=declined" {
		t.Fatalf("declined: %s", loc)
	}
	// Signed out: sign in first, then come back.
	anon := env.Client()
	r = anon.Do("GET", hostrepute.CallbackPath+"?code=x&state=y", nil).MustStatus(t, http.StatusFound)
	if loc := r.Header.Get("Location"); !strings.HasPrefix(loc, "/sign-in?next=") {
		t.Fatalf("anonymous: %s", loc)
	}
	// Disconnecting revokes the OAuth token at HostRepute.
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)
	if fake.Revocations != 1 {
		t.Fatalf("revocations: %d", fake.Revocations)
	}
}

func service(env *apptest.Env) *hostrepute.Service {
	return &hostrepute.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Cfg: env.App.Cfg,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestSyncAlertsAndRevokedToken(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	fake := hrtest.New(t, "hr_live_sync")
	env.App.Cfg.HostRepute.APIURL = fake.APIURL()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)
	c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/api-key", map[string]any{"apiKey": "hr_live_sync"}).MustStatus(t, 200)
	ctx := context.Background()
	id := ws(env)
	s := service(env)

	fake.Alerts = []hostrepute.Alert{{ID: "a-new", CreatedAt: time.Now().Add(-10 * time.Minute), Kind: "listed", Severity: "warn", Asset: "203.0.113.25",
		Source: "Mailspike", Title: "203.0.113.25 listed on Mailspike", Detail: "Low impact.", DelistURL: "https://mailspike.org"}}
	if err := s.Sync(ctx, hostrepute.SyncArgs{WorkspaceID: id}, true); err != nil {
		t.Fatal(err)
	}
	var listed, notes, incidents int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM blocklist_results WHERE via = 'hostrepute' AND listed`).Scan(&listed)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'hostrepute_alert' AND title = '203.0.113.25 listed on Mailspike'`).Scan(&notes)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM incidents WHERE external_id = 'a-new'`).Scan(&incidents)
	// Members who can see Deliverability: everyone active in the demo but the deactivated Tom.
	if listed != 1 || incidents != 1 || notes < 4 {
		t.Fatalf("sync wrote listed=%d incidents=%d notifications=%d", listed, incidents, notes)
	}
	// The same alert again is not stored or notified twice.
	if err := s.Sync(ctx, hostrepute.SyncArgs{WorkspaceID: id}, true); err != nil {
		t.Fatal(err)
	}
	var again int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE title = '203.0.113.25 listed on Mailspike'`).Scan(&again)
	if again != notes {
		t.Fatalf("notified twice: %d then %d", notes, again)
	}

	// HostRepute down: the error is recorded for the Deliverability error state.
	fake.Down = true
	if err := s.Sync(ctx, hostrepute.SyncArgs{WorkspaceID: id}, false); err == nil {
		t.Fatal("a failed attempt returns the error so River retries")
	}
	var syncErr string
	_ = env.App.Pool.QueryRow(ctx, `SELECT last_sync_error FROM hostrepute_connections`).Scan(&syncErr)
	if !strings.HasPrefix(syncErr, "Couldn't reach 127.0.0.1") {
		t.Fatalf("sync error: %q", syncErr)
	}
	fake.Down = false

	// Revoked on the HostRepute side: status revoked, audited, the people who can reconnect are told.
	fake.Revoked = true
	if err := s.Sync(ctx, hostrepute.SyncArgs{WorkspaceID: id}, true); err != nil {
		t.Fatal(err)
	}
	var out struct{ HostRepute connView }
	c.Do("GET", "/workspaces/acme-newsletter/integrations", nil).MustStatus(t, 200).JSON(t, &out)
	if out.HostRepute.Status != "revoked" || out.HostRepute.TokenRejectedAt == nil {
		t.Fatalf("revoked: %+v", out.HostRepute)
	}
	if auditCount(env, "hostrepute.token_rejected", "allowed") != 1 {
		t.Fatal("token rejection is audited")
	}
	var reconnect int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE title = 'HostRepute needs reconnecting'`).Scan(&reconnect)
	if reconnect != 2 { // Linh (Owner) and Marcus (Admin)
		t.Fatalf("reconnect notifications: %d", reconnect)
	}
	p := c.Do("POST", "/workspaces/acme-newsletter/integrations/hostrepute/assets", map[string]any{"asset": "mta.acme.io"}).MustStatus(t, 409).Problem(t)
	if p["code"] != "hostrepute_reconnect" {
		t.Fatalf("acting on a revoked connection: %+v", p)
	}
}
