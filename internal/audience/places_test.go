package audience_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
)

// G-12: the country and city a campaign report keeps per message come from
// the reader's IP, so erasing the contact and the retention pass that
// clears old IPs clear them too.
func TestPlacesFollowTheIP(t *testing.T) {
	env, c := demoEnv(t)
	ctx := context.Background()
	pool := env.App.Pool
	var ws uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	// Two campaign messages to different contacts (with no paid
	// subscription, which erasing would refuse), each opened by a person
	// whose IP is kept.
	rows, err := pool.Query(ctx, `SELECT DISTINCT ON (m.contact_id) m.id, m.contact_id FROM messages m
		WHERE m.workspace_id = $1 AND m.stream = 'broadcast' AND m.campaign_id IS NOT NULL AND m.contact_id IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM paid_subscriptions s WHERE s.contact_id = m.contact_id)
		ORDER BY m.contact_id, m.id LIMIT 2`, ws)
	if err != nil {
		t.Fatal(err)
	}
	var msgs, contacts []uuid.UUID
	for rows.Next() {
		var m, ct uuid.UUID
		if err := rows.Scan(&m, &ct); err != nil {
			t.Fatal(err)
		}
		msgs, contacts = append(msgs, m), append(contacts, ct)
	}
	rows.Close()
	if len(msgs) != 2 {
		t.Fatalf("%d campaign messages", len(msgs))
	}
	for _, m := range msgs {
		if _, err := pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, provider_id, type, occurred_at, mailbox_provider,
			user_agent, ip, class)
			SELECT $1, workspace_id, id, campaign_id, contact_id, provider_id, 'open', now() - interval '1 day', mailbox_provider,
			'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148', '198.51.100.7', 'human'
			FROM messages WHERE id = $2`, uuid.Must(uuid.NewV7()), m); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE messages SET human_client = 'apple_mail', human_country = 'VN', human_city = 'Hanoi' WHERE id = ANY ($1)`, msgs); err != nil {
		t.Fatal(err)
	}
	placed := func(m uuid.UUID) bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE id = $1 AND (human_country <> '' OR human_city <> '')`, m).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n == 1
	}

	c.Do("POST", "/workspaces/acme-newsletter/contacts/erase", map[string]any{"ids": []string{contacts[0].String()}, "confirm": 1}).MustStatus(t, 200)
	if placed(msgs[0]) {
		t.Error("erasing the contact left the place of their open")
	}
	if !placed(msgs[1]) {
		t.Fatal("erasing one contact cleared another's place")
	}

	svc := &audience.Service{Pool: pool, Q: env.App.Q, Audit: env.App.Audit, Now: func() time.Time { return time.Now().AddDate(2, 0, 0) }}
	if _, err := svc.RetainWorkspace(ctx, ws, 0, 30); err != nil {
		t.Fatal(err)
	}
	if placed(msgs[1]) {
		t.Error("clearing old IPs left the place derived from one")
	}
	var client string
	if err := pool.QueryRow(ctx, `SELECT human_client FROM messages WHERE id = $1`, msgs[1]).Scan(&client); err != nil || client != "apple_mail" {
		t.Errorf("retention cleared the mail client too: %q %v", client, err)
	}
}
