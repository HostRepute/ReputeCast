package engine_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/migrations"
)

// testCanaryNewIP (review GS-10): Traffic rules' "The sending domain or IP
// is new" covers a dedicated sending IP that never sent for the workspace.
// Postal lists its IP, so a route through it starts a canary until mail
// went out from that IP; SES sends from a shared pool, the provider's to
// keep warm, which isn't checked. Mail that goes out records the IP.
func testCanaryNewIP(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	// Without the seed mailboxes, as the other canary scenarios.
	f.Exec(`INSERT INTO traffic_rules (workspace_id, canary) VALUES ($1, '{"seeds":false}')
		ON CONFLICT (workspace_id) DO UPDATE SET canary = traffic_rules.canary || '{"seeds":false}'`, f.WS)
	t.Cleanup(func() { f.Exec(`UPDATE traffic_rules SET canary = canary - 'seeds' WHERE workspace_id = $1`, f.WS) })
	f.AddContacts(sendtest.ContactSpec{N: 1500, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	ses, postal := f.ProviderID("ses"), f.ProviderID("postal")
	var ips []string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT ips FROM providers WHERE id = $1 AND ip_mode = 'dedicated'`, postal).Scan(&ips); err != nil || len(ips) == 0 {
		t.Fatalf("Postal's dedicated IPs %v (%v)", ips, err)
	}
	// Send Readiness counted the audience and planned the split: SES 90%,
	// Postal 10%.
	f.Exec(`UPDATE campaigns SET audience_count = 1500 WHERE id = $1`, f.Campaign)
	f.Exec(`INSERT INTO campaign_routes (campaign_id, provider_id, planned, status) VALUES ($1, $2, 1350, 'pending'), ($1, $3, 150, 'pending')
		ON CONFLICT (campaign_id, provider_id) DO UPDATE SET planned = EXCLUDED.planned`, f.Campaign, ses, postal)
	// Postal's IP has never sent for this workspace.
	f.Exec(`DELETE FROM workspace_sending_ips WHERE workspace_id = $1`, f.WS)
	plan, err := f.E.CanaryPlanFor(f.Ctx, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	// Only Postal's: SES's shared pool isn't the workspace's to warm.
	if !slices.Contains(plan.Reasons, "new_ip") || len(plan.NewIPs) != len(ips) || plan.NewIPs[0].IP != ips[0] || plan.NewIPs[0].Provider != "Postal" {
		t.Fatalf("plan %+v, want Postal's IP %s new", plan, ips[0])
	}
	if !plan.Auto {
		t.Fatalf("plan %+v, want a canary by itself for 1,500 recipients and a new IP", plan)
	}
	// A send left to Traffic rules ("auto") starts with it, and says why.
	f.StartSend(engine.StartOptions{Canary: engine.CanaryAuto})
	if c := canaryState(t, f); c["phase"] != "sending" || !slices.Contains(toStrings(c["reasons"]), "new_ip") {
		t.Fatalf("canary %v, want one for the new IP", c)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Canary == nil || len(snap.Canary.NewIPs) != len(ips) || snap.Canary.NewIPs[0].Provider != "Postal" {
		t.Fatalf("live canary %+v, want Postal's new IP", snap.Canary)
	}
	// Mail that went out through Postal records its IP: it isn't new any
	// more.
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND sent_at IS NOT NULL`, f.Campaign, postal); n == 0 {
		f.Dump()
		t.Fatal("nothing went out through Postal")
	}
	if n := f.Count(`SELECT count(*) FROM workspace_sending_ips WHERE workspace_id = $1 AND ip = ANY($2) AND provider_id = $3`, f.WS, ips, postal); n != int64(len(ips)) {
		t.Fatalf("%d of Postal's %d IPs recorded", n, len(ips))
	}
	if plan, err = f.E.CanaryPlanFor(f.Ctx, f.Campaign); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(plan.Reasons, "new_ip") || len(plan.NewIPs) != 0 {
		t.Fatalf("plan %+v after Postal sent, want its IP known", plan)
	}
}

// testCanaryReportedIP (review GS-10, F-4): the new-IP trigger works on
// what production writes, not on a seeded list of IPs. A provider whose
// API reports a dedicated IP gets it when it's connected (the IP check
// providers.ip_mode and ips come from); the first send through it starts
// a canary for that IP, and once mail went out from it, the next send to
// the same readers doesn't start one.
func testCanaryReportedIP(t *testing.T, s *sendtest.Suite) {
	f := s.Fixture(t, "GS-10 reported IP")
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.Exec(`INSERT INTO traffic_rules (workspace_id, canary) VALUES ($1, '{"seeds":false}')
		ON CONFLICT (workspace_id) DO UPDATE SET canary = traffic_rules.canary || '{"seeds":false}'`, f.WS)
	t.Cleanup(func() { f.Exec(`UPDATE traffic_rules SET canary = canary - 'seeds' WHERE workspace_id = $1`, f.WS) })
	f.AddContacts(sendtest.ContactSpec{N: 1200, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	sendtest.Fault.Reset(sendtest.FaultOK)
	t.Cleanup(func() { sendtest.Fault.Reset(sendtest.FaultOK) })
	id := f.UseReportingProvider()
	const ip = "198.51.100.77"
	sendtest.Fault.SetIPs(ip)

	// Connecting asks the provider for its IPs, as the daily check does.
	svc := &manage.Service{Pool: f.Env.App.Pool, Q: f.Env.App.Q, Audit: f.Env.App.Audit, Factory: &manage.Factory{Box: f.Env.App.Secrets}}
	p, err := f.Env.App.Q.ProvGetAny(f.Ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if p, err = svc.RefreshIPs(f.Ctx, p, manage.IPsOnConnect); err != nil {
		t.Fatal(err)
	}
	if p.IpMode != "dedicated" || !slices.Equal(p.Ips, []string{ip}) {
		t.Fatalf("after the IP check: %s %q, want dedicated %s", p.IpMode, p.Ips, ip)
	}

	// The first send through it: a canary, for the new IP among its reasons.
	f.StartSend(engine.StartOptions{Canary: engine.CanaryAuto})
	c := canaryState(t, f)
	if c["phase"] != "sending" || !slices.Contains(toStrings(c["reasons"]), "new_ip") {
		f.Dump()
		t.Fatalf("canary %v, want one for the reported IP", c)
	}
	if ips, _ := c["newIps"].([]any); len(ips) != 1 || ips[0].(map[string]any)["ip"] != ip {
		t.Fatalf("canary's new IPs %v, want %s", c["newIps"], ip)
	}
	if _, done := f.Drive(3 * time.Hour); !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("the first send didn't finish (%s)", f.Status())
	}
	if n := f.Count(`SELECT count(*) FROM workspace_sending_ips WHERE workspace_id = $1 AND ip = $2 AND provider_id = $3`, f.WS, ip, id); n != 1 {
		t.Fatalf("the IP mail went out from isn't recorded (%d)", n)
	}

	// The second send through it, to the same readers: the IP has sent for
	// the workspace and nobody is new to the domain, so no canary at all.
	// The domain has sent for months (the first send moved back; the demo's
	// own sends from it carry no start), so only the IP is under test.
	var started time.Time
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT started_at FROM campaigns WHERE id = $1`, f.Campaign).Scan(&started); err != nil {
		t.Fatal(err)
	}
	f.Exec(`UPDATE campaigns SET started_at = $2 WHERE id = $1`, f.Campaign, started.AddDate(0, -2, 0))
	t.Cleanup(func() { f.Exec(`UPDATE campaigns SET started_at = $2 WHERE id = $1`, f.Campaign, started) })
	g := s.Fixture(t, "GS-10 reported IP again")
	g.Exec(`UPDATE campaigns SET ab_test = NULL, audience = $2, route_override = $3 WHERE id = $1`,
		g.Campaign, fmt.Sprintf(`{"lists":[%q]}`, f.List), `{"provider_id":"`+id.String()+`"}`)
	g.StartSend(engine.StartOptions{Canary: engine.CanaryAuto})
	if c := canaryState(t, g); c["phase"] != nil && c["phase"] != "" || len(toStrings(c["reasons"])) > 0 {
		t.Fatalf("canary %v on the second send, want none", c)
	}
	plan, err := g.E.CanaryPlanFor(g.Ctx, g.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(plan.Reasons, "new_ip") || len(plan.NewIPs) != 0 {
		t.Fatalf("plan %+v, want the reported IP known", plan)
	}
}

// Migration 03313 records the dedicated IPs that already sent, from the
// sends through their providers, so an existing IP doesn't start a canary
// as if it were new (review GS-10). A dedicated IP whose provider never
// sent stays new; a shared pool has no IPs to record. Going down drops the
// table, and up again rebuilds it.
func TestSendingIPsBackfill(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var ws uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	count := func(sql string, args ...any) int64 {
		t.Helper()
		var n int64
		if err := env.App.Pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}
	// A dedicated provider connected today, whose IP never sent.
	fresh := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1, $2, 'postal', 'Postal 2', 'test', '{}', NULL, $3, 'healthy', '', 'dedicated', '{198.51.100.7}',
		'{}', '{}', '{}', '{}', '{}', now())`, fresh, ws, "fresh-"+fresh.String())
	sent := count(`SELECT count(*) FROM providers p CROSS JOIN LATERAL unnest(p.ips) AS a (ip)
		WHERE p.workspace_id = $1 AND p.ip_mode = 'dedicated' AND EXISTS (SELECT 1 FROM campaign_counters cc WHERE cc.provider_id = p.id AND cc.sent > 0)`, ws)
	if sent == 0 {
		t.Fatal("the demo has no dedicated IP that sent")
	}
	p, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(env.App.Pool), migrations.FS, goose.WithAllowOutofOrder(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	const version = 3313
	for range 2 {
		if _, err := p.ApplyVersion(ctx, version, false); err != nil {
			t.Fatal(err)
		}
		if n := count(`SELECT count(*) FROM pg_tables WHERE tablename = 'workspace_sending_ips'`); n != 0 {
			t.Fatal("going down kept workspace_sending_ips")
		}
		if _, err := p.ApplyVersion(ctx, version, true); err != nil {
			t.Fatal(err)
		}
		if n := count(`SELECT count(*) FROM workspace_sending_ips WHERE workspace_id = $1`, ws); n != sent {
			t.Fatalf("%d IPs recorded, want the %d dedicated IPs that sent", n, sent)
		}
		if n := count(`SELECT count(*) FROM workspace_sending_ips s JOIN providers p ON p.id = s.provider_id
			WHERE s.workspace_id = $1 AND p.ip_mode = 'dedicated' AND s.ip = ANY(p.ips) AND s.first_sent_at <= s.last_sent_at`, ws); n != sent {
			t.Fatalf("%d IPs recorded with their provider and times, want %d", n, sent)
		}
		if n := count(`SELECT count(*) FROM workspace_sending_ips WHERE ip = '198.51.100.7' OR provider_id = $1`, fresh); n != 0 {
			t.Fatal("an IP that never sent was recorded")
		}
	}
}

// toStrings reads a JSON list of strings.
func toStrings(v any) []string {
	var out []string
	l, _ := v.([]any)
	for _, x := range l {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
