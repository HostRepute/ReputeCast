package app_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/push"
	"github.com/hostrepute/reputecast/internal/whatsapp"
)

// Review 2026-11 CH-2: the push and WhatsApp sweeps remove what is past its
// 90 days (WhatsApp replies and raw webhooks, push clicks and removed
// subscriptions) and keep what is newer.
func TestChannelRetentionSweeps(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	count := func(q string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, q).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return n
	}
	before := map[string]int{
		"inbound": count(`SELECT count(*) FROM whatsapp_inbound`),
		"clicks":  count(`SELECT count(*) FROM push_clicks`),
		"removed": count(`SELECT count(*) FROM push_subscriptions WHERE removed_at IS NOT NULL`),
	}
	if before["inbound"] == 0 || before["clicks"] < 2 || before["removed"] < 2 {
		t.Fatalf("the demo's rows: %v", before)
	}
	for _, q := range []string{
		`UPDATE whatsapp_inbound SET received_at = now() - interval '91 days'`,
		`UPDATE push_clicks SET occurred_at = now() - interval '91 days' WHERE id = (SELECT id FROM push_clicks ORDER BY occurred_at LIMIT 1)`,
		`UPDATE push_subscriptions SET removed_at = now() - interval '91 days' WHERE id = (SELECT id FROM push_subscriptions WHERE removed_at IS NOT NULL ORDER BY id LIMIT 1)`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	if err := (&whatsapp.SweepWorker{S: env.App.WhatsApp()}).Work(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := (&push.SweepWorker{S: env.App.Push()}).Work(ctx, nil); err != nil {
		t.Fatal(err)
	}
	after := map[string]int{
		"inbound": count(`SELECT count(*) FROM whatsapp_inbound`),
		"clicks":  count(`SELECT count(*) FROM push_clicks`),
		"removed": count(`SELECT count(*) FROM push_subscriptions WHERE removed_at IS NOT NULL`),
	}
	if after["inbound"] != 0 || after["clicks"] != before["clicks"]-1 || after["removed"] != before["removed"]-1 {
		t.Fatalf("after the sweeps: %v, before %v", after, before)
	}
}
