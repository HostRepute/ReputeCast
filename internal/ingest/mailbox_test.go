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

// Reports read from the bounce mailbox carry no signature, and anyone who
// got a newsletter can email its return path. A hard bounce or complaint
// from there only suppresses the recipient of a message ReputeCast sent;
// one naming other addresses is kept as an event and changes nothing.
// Signed webhooks are unchanged.
func TestMailboxReportsOnlyCountForSentMessages(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID, providerID, msgID uuid.UUID
	var sentTo string
	if err := env.App.Pool.QueryRow(ctx, `SELECT m.workspace_id, m.provider_id, m.id, m.email FROM messages m
		JOIN workspaces w ON w.id = m.workspace_id AND w.slug = 'acme-newsletter'
		JOIN contacts c ON c.id = m.contact_id AND c.status = 'active'
		WHERE m.provider_id IS NOT NULL AND m.kind <> 'placement'
		  AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = m.workspace_id AND s.email = m.email)
		ORDER BY m.id LIMIT 1`).Scan(&wsID, &providerID, &msgID, &sentTo); err != nil {
		t.Fatal(err)
	}
	rows, err := env.App.Pool.Query(ctx, `SELECT c.email FROM contacts c WHERE c.workspace_id = $1 AND c.status = 'active' AND c.email <> $2
		AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email) ORDER BY c.email LIMIT 3`, wsID, sentTo)
	if err != nil {
		t.Fatal(err)
	}
	others, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil || len(others) != 3 {
		t.Fatalf("other contacts: %v %v", others, err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, providerID)
	if err != nil {
		t.Fatal(err)
	}
	n := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q}
	apply := func(source string, ev providers.Event) int {
		t.Helper()
		ev.OccurredAt = time.Now()
		var added int
		if err := db.Tx(ctx, env.App.Pool, func(tx pgx.Tx) error {
			added, err = n.Apply(ctx, tx, p, nil, source, time.Now(), ev)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return added
	}
	state := func(email string) (suppressed bool, status string, events int) {
		t.Helper()
		if err := env.App.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM suppressions WHERE workspace_id = $1 AND email = $2),
			(SELECT status FROM contacts WHERE workspace_id = $1 AND email = $2),
			(SELECT count(*) FROM delivery_events WHERE workspace_id = $1 AND recipient = $2)`, wsID, email).Scan(&suppressed, &status, &events); err != nil {
			t.Fatal(err)
		}
		return
	}
	bounce := func(to string, msg *uuid.UUID) providers.Event {
		return providers.Event{Type: providers.EventHardBounce, MessageID: msg, Recipient: to, OriginalType: "DSN · 5.1.1", OriginalCode: "5.1.1"}
	}

	// A report about no message ReputeCast sent, and one citing a real
	// message but naming someone else: kept, nothing suppressed.
	forged, cited := others[0], others[1]
	if added := apply("bounce_mailbox", bounce(forged, nil)); added != 0 {
		t.Fatalf("forged bounce suppressed %d", added)
	}
	complaint := providers.Event{Type: providers.EventComplaint, MessageID: &msgID, Recipient: cited, OriginalType: "ARF · abuse"}
	if added := apply("bounce_mailbox", complaint); added != 0 {
		t.Fatalf("complaint naming another address suppressed %d", added)
	}
	for _, email := range []string{forged, cited} {
		if suppressed, status, events := state(email); suppressed || status != "active" || events != 1 {
			t.Fatalf("%s: suppressed %v, status %s, %d events", email, suppressed, status, events)
		}
	}

	// The message's own recipient bouncing is a real report.
	if added := apply("bounce_mailbox", bounce(sentTo, &msgID)); added != 1 {
		t.Fatalf("matched bounce suppressed %d", added)
	}
	if suppressed, status, _ := state(sentTo); !suppressed || status != "bounced" {
		t.Fatalf("%s: suppressed %v, status %s", sentTo, suppressed, status)
	}

	// A signed webhook needs no matched message.
	if added := apply("api", bounce(others[2], nil)); added != 1 {
		t.Fatalf("webhook bounce suppressed %d", added)
	}
}
