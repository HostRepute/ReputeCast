package ingest_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// TestSenderBlocksAreNeverSuppressed: a bounce for the sending IP's or
// domain's reputation says nothing about the address, so it is never
// suppressed; an unknown user still is (review AN-7).
func TestSenderBlocksAreNeverSuppressed(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "Blocked bounces")
	ids := f.HandedOver(sendtest.ContactSpec{N: 3, Mailbox: "outlook", Domain: "outlook.com"})
	at := f.Start.Add(time.Minute)
	bounced := func(id uuid.UUID, class, code, diag string) {
		t.Helper()
		var pid uuid.UUID
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT provider_id FROM messages WHERE id = $1`, id).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		p, err := f.Env.App.Q.ProvGetAny(f.Ctx, pid)
		if err != nil {
			t.Fatal(err)
		}
		mid := id
		if err := sending.Tx(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
			_, err := f.E.Normalizer.Apply(f.Ctx, tx, p, nil, "webhook", at, providers.Event{Type: providers.EventHardBounce, MessageID: &mid, OccurredAt: at,
				OriginalType: "Bounce/Permanent", OriginalCode: code, Diagnostic: diag, BounceClass: class})
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	bounced(ids[0], bounce.Reputation, "550 5.7.1", "Messages from 203.0.113.25 weren't sent: the sending IP's reputation (S3150)")
	bounced(ids[1], bounce.SpamBlock, "554 5.7.1", "Client host [203.0.113.25] blocked using Spamhaus")
	bounced(ids[2], bounce.MailboxUnknown, "550 5.1.1", "User unknown")
	suppressed := func(id uuid.UUID) bool {
		t.Helper()
		return f.Count(`SELECT count(*) FROM suppressions s JOIN messages m ON lower(m.email) = s.email AND m.workspace_id = s.workspace_id WHERE m.id = $1`, id) > 0
	}
	if suppressed(ids[0]) || suppressed(ids[1]) {
		t.Fatal("an address blocked for the sender's reputation was suppressed")
	}
	if !suppressed(ids[2]) {
		t.Fatal("an unknown user wasn't suppressed")
	}
	if n := f.Count(`SELECT count(*) FROM contacts c JOIN messages m ON m.contact_id = c.id WHERE m.id = ANY($1) AND c.status = 'bounced'`, []uuid.UUID{ids[0], ids[1]}); n != 0 {
		t.Fatalf("%d contacts blocked for the sender's reputation marked bounced", n)
	}
	// The bounces still count as bounces.
	if c := f.Counts(); c["hard_bounced"] != 3 {
		t.Fatalf("hard bounces: %v", c)
	}
}
