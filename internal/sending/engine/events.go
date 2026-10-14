package engine

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Who records an action in the audit log, and the outgoing webhook events
// of a send.

// ownAPIKey marks a request served by the engine's own API.
type ownAPIKey struct{}

// ownAPI marks ctx as a request of the engine's own API (the live
// monitor's actions): the engine records those actions itself.
func ownAPI(ctx context.Context) context.Context { return context.WithValue(ctx, ownAPIKey{}, true) }

// records reports whether the engine writes the audit entry of an action it
// was asked to take. It does for what happens with no request behind it (a
// scheduled start, a held send started again, the demo command) and for
// its own API's actions. The other callers of the Starter contract (Review
// & send, the MCP tools, the providers screen) record the person's or the
// agent's action themselves, so the same action isn't logged twice. What
// the engine does on its own (auto-pause, holds, failover, the A/B winner,
// the end of a send) is always recorded.
func records(ctx context.Context) bool {
	return audit.PrincipalFrom(ctx) == nil || ctx.Value(ownAPIKey{}) != nil
}

// emit queues an outgoing webhook event of a campaign (the catalog in
// internal/webhooks) in the transaction that makes the change. The
// dispatcher never calls the endpoints the demo seeded.
func emit(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID, event string, data map[string]any) error {
	ws, err := store.New(tx).GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	_, err = webhooks.Emit(ctx, tx, workspaceID, ws.Slug, event, data)
	return err
}

// campaignData starts an event's data with the campaign.
func campaignData(id uuid.UUID, name string, kv ...any) map[string]any {
	d := map[string]any{"campaign_id": id.String(), "name": name}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			d[k] = kv[i+1]
		}
	}
	return d
}
