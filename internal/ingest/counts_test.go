package ingest_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/providers"
)

// countsFixture is a message of a demo campaign with a counters row of
// its own (variant "gs9"), a way to apply provider events to it, and its
// deferred and delivered counts.
func countsFixture(t *testing.T) (apply func(providers.EventType, string), counts func() (int64, int64), events func() int64) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID, providerID, campaignID, contactID uuid.UUID
	var email string
	if err := env.App.Pool.QueryRow(ctx, `SELECT m.workspace_id, m.provider_id, m.campaign_id, m.contact_id, m.email FROM messages m
		JOIN workspaces w ON w.id = m.workspace_id AND w.slug = 'acme-newsletter'
		WHERE m.campaign_id IS NOT NULL AND m.provider_id IS NOT NULL AND m.contact_id IS NOT NULL ORDER BY m.id LIMIT 1`).
		Scan(&wsID, &providerID, &campaignID, &contactID, &email); err != nil {
		t.Fatal(err)
	}
	msgID := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, provider_id, mailbox_provider, variant, status, queued_at, sent_at)
		VALUES ($1, $2, 'broadcast', $3, $4, $5, $6, 'yahoo', 'gs9', 'sent', now(), now())`, msgID, wsID, campaignID, contactID, email, providerID); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, providerID)
	if err != nil {
		t.Fatal(err)
	}
	n := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q}
	at := time.Now().Add(-time.Hour)
	apply = func(typ providers.EventType, code string) {
		t.Helper()
		at = at.Add(time.Minute)
		ev := providers.Event{Type: typ, MessageID: &msgID, Recipient: email, OccurredAt: at, OriginalType: "DSN · " + code, OriginalCode: code}
		if err := db.Tx(ctx, env.App.Pool, func(tx pgx.Tx) error {
			_, err := n.Apply(ctx, tx, p, nil, "webhook", at, ev)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	counts = func() (deferred, delivered int64) {
		t.Helper()
		if err := env.App.Pool.QueryRow(ctx, `SELECT coalesce(sum(deferred), 0)::bigint, coalesce(sum(delivered), 0)::bigint FROM campaign_counters
			WHERE campaign_id = $1 AND provider_id = $2 AND mailbox_provider = 'yahoo' AND variant = 'gs9'`, campaignID, providerID).Scan(&deferred, &delivered); err != nil {
			t.Fatal(err)
		}
		return
	}
	events = func() int64 {
		t.Helper()
		var n int64
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events WHERE message_id = $1`, msgID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	return apply, counts, events
}

// A message counts once as delivered: a success DSN confirming a delivery
// SMTP inferred at acceptance stays in the event log without counting it
// again (review GS-7).
func TestDeliveredCountsOnce(t *testing.T) {
	apply, counts, events := countsFixture(t)
	apply(providers.EventDelivered, "250")
	apply(providers.EventDelivered, "2.0.0")
	if _, del := counts(); del != 1 {
		t.Fatalf("delivered %d, want 1", del)
	}
	if n := events(); n != 2 {
		t.Fatalf("%d events in the log, want both kept", n)
	}
}
