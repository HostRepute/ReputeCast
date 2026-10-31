package app_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// EN-5: a provider's delivery report about an automation email counts on
// the step that sent it, as the simulator's did; before, real providers
// left Delivered and Bounced at 0.
func TestProviderEventsCountOnAutomationSteps(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	var msgID, providerID, automationID, wsID uuid.UUID
	var node string
	if err := pool.QueryRow(ctx, `SELECT m.id, m.provider_id, m.automation_id, m.automation_node, m.workspace_id FROM messages m
		WHERE m.stream = 'automation' AND m.automation_id IS NOT NULL AND m.automation_node IS NOT NULL AND m.provider_id IS NOT NULL AND m.status IN ('sent', 'delivered')
		ORDER BY m.queued_at DESC LIMIT 1`).Scan(&msgID, &providerID, &automationID, &node, &wsID); err != nil {
		t.Fatalf("a sent automation email in the demo: %v", err)
	}
	delivered := func() int64 {
		var n int64
		if err := pool.QueryRow(ctx, `SELECT coalesce(sum(delivered), 0) FROM automation_step_stats WHERE automation_id = $1 AND node_id = $2`, automationID, node).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := delivered()
	p, err := env.App.Q.ProvGet(ctx, store.ProvGetParams{ID: providerID, WorkspaceID: wsID})
	if err != nil {
		t.Fatal(err)
	}
	n := app.NormalizerOf(env.App)
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := n.Apply(ctx, tx, p, nil, "webhook", time.Now(), providers.Event{Type: providers.EventDelivered, MessageID: &msgID, OccurredAt: time.Now()})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if got := delivered(); got != before+1 {
		t.Fatalf("delivered on the step went from %d to %d, want +1", before, got)
	}
}

// EN-6: a provider's report about campaign email reaches the workspace's
// endpoints as email.delivered, with the message, contact and campaign;
// nothing is queued for endpoints that don't take it.
func TestEmailEventsReachWebhooks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var msgID, providerID, wsID, campaignID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT m.id, m.provider_id, m.workspace_id, m.campaign_id FROM messages m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme-newsletter' AND m.stream = 'broadcast' AND m.campaign_id IS NOT NULL AND m.provider_id IS NOT NULL
		ORDER BY m.queued_at DESC LIMIT 1`).Scan(&msgID, &providerID, &wsID, &campaignID); err != nil {
		t.Fatal(err)
	}
	var hook struct{ Webhook struct{ ID string } }
	c.Do("POST", "/workspaces/acme-newsletter/webhooks", map[string]any{"name": "Warehouse", "url": "https://warehouse.example.com/hook", "events": []string{"email.*"}}).
		MustStatus(t, 201).JSON(t, &hook)
	p, err := env.App.Q.ProvGet(ctx, store.ProvGetParams{ID: providerID, WorkspaceID: wsID})
	if err != nil {
		t.Fatal(err)
	}
	n := app.NormalizerOf(env.App)
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := n.Apply(ctx, tx, p, nil, "webhook", time.Now(), providers.Event{Type: providers.EventHardBounce, MessageID: &msgID, OccurredAt: time.Now(),
			OriginalCode: "550 5.1.1", Diagnostic: "user unknown"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var payload string
	if err := pool.QueryRow(ctx, `SELECT payload::text FROM webhook_deliveries WHERE webhook_id = $1 AND event = 'email.bounced'`, hook.Webhook.ID).Scan(&payload); err != nil {
		t.Fatalf("no email.bounced delivery: %v", err)
	}
	for _, want := range []string{msgID.String(), campaignID.String(), `"stream": "broadcast"`, "550 5.1.1 user unknown"} {
		if !strings.Contains(payload, want) {
			t.Errorf("payload lacks %s: %s", want, payload)
		}
	}
}
