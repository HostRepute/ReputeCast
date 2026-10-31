package engine_test

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testSMTPInfersDelivery: a provider that reports no deliveries (plain
// SMTP) has them inferred from its acceptance (review GS-7): every message
// it took counts as delivered once, in the event log as "delivered
// (inferred)", so delivered-based rates have a base.
func testSMTPInfersDelivery(t *testing.T, f *sendtest.Fixture) {
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.AddContacts(sendtest.ContactSpec{N: 40, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	sendtest.Fault.Reset(sendtest.FaultOK)
	pid := f.UseFaultSMTP()
	f.StartSend(engine.StartOptions{})
	if _, _, err := f.E.Drive(f.Ctx, f.Campaign, f.Start, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT coalesce(sum(delivered), 0)::bigint FROM campaign_counters WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, pid); n != 40 {
		f.Dump()
		t.Fatalf("%d delivered, want the 40 the server accepted", n)
	}
	if n := f.Count(`SELECT count(*) FROM delivery_events WHERE campaign_id = $1 AND type = 'delivered' AND original_type LIKE '%inferred%'`, f.Campaign); n != 40 {
		t.Fatalf("%d inferred deliveries in the event log, want 40", n)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range snap.Providers {
		if p.ID == pid.String() && (!p.InfersDelivery || p.DeliveredRate == nil || *p.DeliveredRate != 100) {
			t.Fatalf("provider row %+v, want 100%% delivered (inferred)", p)
		}
	}
}
