package app_test

import (
	"context"
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
