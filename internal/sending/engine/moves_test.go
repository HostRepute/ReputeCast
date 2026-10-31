package engine_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testMoveKeepsHoldouts: a person moves a held share from the providers
// screen. The target is vetted (a sandboxed account is refused with the
// reason), the A/B test's holdouts stay held for the winner on the new
// provider, the incident resolves, and neither Retry nor the move sends a
// campaign a person paused (review GS-5).
func testMoveKeepsHoldouts(t *testing.T, f *sendtest.Fixture) {
	f.AddContacts(sendtest.ContactSpec{N: 400, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	sendtest.Fault.Reset(sendtest.FaultRefuse)
	pid := f.UseFaultProvider(false)
	f.StartSend(engine.StartOptions{})
	at, _ := f.Drive(6 * time.Minute)
	if st, _ := routeOf(t, f, pid); st != "held" {
		f.Dump()
		t.Fatalf("route %s, want held", st)
	}
	holdouts := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held'`, f.Campaign)
	if holdouts == 0 {
		t.Fatal("no A/B holdouts to keep")
	}
	setClock(t, f, at)
	// A person pauses the send while its share waits for a decision.
	if err := f.E.Pause(f.Ctx, f.Campaign, f.Sofia, "checking the provider"); err != nil {
		t.Fatal(err)
	}
	if err := f.E.RetryRoute(f.Ctx, f.Campaign, f.Sofia, pid); err != nil {
		t.Fatal(err)
	}
	if s := f.Status(); s != "paused" {
		t.Fatalf("campaign %s after Retry, want the person's pause kept", s)
	}
	linh := f.Env.Client()
	linh.SignInDemo("linh@acme.io")
	path := "/workspaces/acme-newsletter/providers/" + pid.String() + "/held/" + f.Campaign.String()
	// A sandboxed account can't take the share.
	sandbox := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1, $2, 'faultesp', 'Sandboxed ESP', 'test', '{}', NULL, $3, 'sandbox', '', 'shared', '{}',
		'{"bulk":"allowed","webhooks":"Registered","batchSize":10}', '{}', '{}', '{}', '{}', now())`, sandbox, f.WS, "fault-"+sandbox.String())
	linh.Do("POST", path, map[string]string{"action": "move", "targetProviderId": sandbox.String()}).MustStatus(t, 409)
	// SES can.
	ses := f.ProviderID("ses")
	var res struct{ Moved int64 }
	linh.Do("POST", path, map[string]string{"action": "move", "targetProviderId": ses.String()}).MustStatus(t, 200).JSON(t, &res)
	if res.Moved == 0 {
		t.Fatal("nothing moved")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held' AND provider_id = $2`, f.Campaign, ses); n != holdouts {
		t.Fatalf("%d holdouts held on SES, want all %d", n, holdouts)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND status IN ('queued', 'held')`, f.Campaign, pid); n != 0 {
		t.Fatalf("%d left on the failed provider", n)
	}
	if s := f.Status(); s != "paused" {
		t.Fatalf("campaign %s after the move, want the person's pause kept", s)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND resolved_at IS NULL`, f.Campaign); n != 0 {
		t.Fatalf("%d incidents still open after the move", n)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE action IN ('campaign.held_move', 'campaign.share_moved') AND (target_id = $1 OR metadata->>'campaign_id' = $1)`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d audit entries for the move, want one", n)
	}
	// The person resumes: the test group goes out on SES; the holdouts
	// wait for the A/B winner.
	if err := f.E.Resume(f.Ctx, f.Campaign, f.Sofia); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL AND provider_id = $2`, f.Campaign, ses); n == 0 {
		t.Fatal("nothing went out on SES")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held'`, f.Campaign); n != holdouts {
		t.Fatalf("%d holdouts held, want %d until the winner", n, holdouts)
	}
}
