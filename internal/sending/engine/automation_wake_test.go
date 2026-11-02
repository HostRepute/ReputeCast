package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testAutomationOutboxWake: an automation email queued inside a db.Tx (the
// automations engine's tick, a referral reward) doesn't insert the outbox's
// unique job in that transaction, which would hold the job's row until it
// commits and queue concurrent automation workers behind it; the outbox is
// woken once the transaction has committed. A transaction begun some other
// way still wakes it inside (review 2026-11 OP-13).
func testAutomationOutboxWake(t *testing.T, f *sendtest.Fixture) {
	email := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO automation_emails (id, workspace_id, name, latest_version) VALUES ($1, $2, 'Wake test', 1)`, email, f.WS)
	f.Exec(`INSERT INTO automation_email_versions (email_id, version, subject, html, plain_text) VALUES ($1, 1, 'Welcome', $2, 'Welcome aboard')`, email,
		`<html><body><p>Welcome aboard</p><p><a href="{{ unsubscribe_url }}">Unsubscribe</a> · Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105</p></body></html>`)
	f.AddContacts(sendtest.ContactSpec{N: 2, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	var contacts []automations.OutgoingEmail
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT c.id, c.email FROM contacts c JOIN list_memberships m ON m.contact_id = c.id WHERE m.list_id = $1 ORDER BY c.email LIMIT 2`, f.List)
	if err != nil {
		t.Fatal(err)
	}
	var flow uuid.UUID
	for rows.Next() {
		var o automations.OutgoingEmail
		if err := rows.Scan(&o.ContactID, &o.To); err != nil {
			t.Fatal(err)
		}
		contacts = append(contacts, o)
	}
	rows.Close()
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT id FROM automations WHERE workspace_id = $1 ORDER BY created_at LIMIT 1`, f.WS).Scan(&flow); err != nil {
		t.Fatal(err)
	}
	for i := range contacts {
		contacts[i].WorkspaceID, contacts[i].AutomationID, contacts[i].NodeID = f.WS, flow, "m"
		contacts[i].EmailID, contacts[i].EmailVersion, contacts[i].Subject = email, 1, "Welcome"
	}
	m := engine.AutomationMailer{E: f.E}
	jobs := func(ctx context.Context, q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	) int {
		t.Helper()
		var n int
		if err := q.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'sending.outbox'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	f.Exec(`DELETE FROM river_job WHERE kind = 'sending.outbox'`)
	if err := db.Tx(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
		out, err := m.QueueAutomationEmail(f.Ctx, tx, contacts[0])
		if err != nil {
			return err
		}
		if out.Held {
			t.Fatal("the email was held: the fixture's route has no provider")
		}
		if n := jobs(f.Ctx, tx); n != 0 {
			t.Errorf("the outbox job was inserted inside the sender's transaction (%d)", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	woken := 0
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if woken = jobs(f.Ctx, f.Env.App.Pool); woken > 0 {
			break
		}
	}
	if woken != 1 {
		t.Fatalf("outbox wake jobs after commit %d, want 1", woken)
	}

	// A transaction that isn't a db.Tx wakes the outbox inside it, as before.
	f.Exec(`DELETE FROM river_job WHERE kind = 'sending.outbox'`)
	tx, err := f.Env.App.Pool.Begin(f.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(f.Ctx) //nolint:errcheck
	if _, err := m.QueueAutomationEmail(f.Ctx, tx, contacts[1]); err != nil {
		t.Fatal(err)
	}
	if n := jobs(f.Ctx, tx); n != 1 {
		t.Fatalf("outside a db.Tx the outbox job inside the transaction %d, want 1", n)
	}
}
