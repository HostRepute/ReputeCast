// Package loadtest builds the large data set the load tests and the
// benchmarks run against (review 2026-11 OP-9): a "Load test" workspace
// with a million contacts (by default), three lists, three sent campaigns
// with a message per recipient and their delivery events spread over five
// months. Rows are generated in Postgres, so a million contacts take
// minutes, not hours.
package loadtest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// Slug is the load test workspace's slug.
const Slug = "load-test"

// Result says what Seed made.
type Result struct {
	Workspace store.Workspace
	Contacts  int64
	Messages  int64
	Events    int64
	Took      time.Duration
}

// batch is how many contacts one statement inserts.
const batch = 100_000

// Seed makes the workspace and its data. It refuses when the workspace
// exists already; an instance user owns it.
func Seed(ctx context.Context, pool *pgxpool.Pool, contacts int, log *slog.Logger) (Result, error) {
	start := time.Now()
	var res Result
	if contacts < 1 {
		return res, errors.New("loadtest: at least one contact")
	}
	var owner uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE status = 'active' ORDER BY is_instance_admin DESC, created_at LIMIT 1`).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, errors.New("loadtest: no user to own the workspace: run `reputecast install` or `reputecast seed` first")
	}
	if err != nil {
		return res, err
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = $1)`, Slug).Scan(&exists); err != nil || exists {
		if err == nil {
			err = fmt.Errorf("loadtest: the %s workspace exists already", Slug)
		}
		return res, err
	}
	if err := db.Tx(ctx, pool, func(tx pgx.Tx) error {
		var err error
		res.Workspace, err = workspaces.Create(ctx, tx, workspaces.CreateParams{Name: "Load test", Owner: owner, Source: "admin"})
		return err
	}); err != nil {
		return res, err
	}
	ws := res.Workspace.ID
	exec := func(what, sql string, args ...any) (int64, error) {
		t := time.Now()
		tag, err := pool.Exec(ctx, sql, args...)
		if err != nil {
			return 0, fmt.Errorf("loadtest: %s: %w", what, err)
		}
		log.Info("load test data", "step", what, "rows", tag.RowsAffected(), "took", time.Since(t).Round(time.Millisecond))
		return tag.RowsAffected(), nil
	}

	// Contacts, in batches: mostly active, some unsubscribed or bounced,
	// joined over two years, at six mailbox providers.
	for from := 1; from <= contacts; from += batch {
		to := min(from+batch-1, contacts)
		n, err := exec(fmt.Sprintf("contacts %d–%d", from, to), `
			INSERT INTO contacts (id, workspace_id, email, first_name, last_name, status, consent, source, mailbox_provider, country, tags,
			                      created_at, updated_at)
			SELECT gen_random_uuid(), $1,
			       'load' || i || '@' || (ARRAY['gmail.com', 'outlook.com', 'yahoo.com', 'icloud.com', 'proton.me', 'acme.io'])[1 + i % 6],
			       (ARRAY['Linh', 'Sam', 'Priya', 'Jordan', 'Maya', 'Kenji', 'Sofia', 'Marcus'])[1 + i % 8],
			       (ARRAY['Nguyen', 'Rivera', 'Shah', 'Lee', 'Chen', 'Tanaka', 'Rossi', 'Brown'])[1 + (i / 8) % 8],
			       CASE WHEN i % 50 = 0 THEN 'unsubscribed' WHEN i % 199 = 0 THEN 'bounced' ELSE 'active' END,
			       'double', 'import',
			       (ARRAY['gmail', 'outlook', 'yahoo', 'icloud', 'other', 'other'])[1 + i % 6],
			       (ARRAY['US', 'VN', 'DE', 'GB', 'FR', 'BR'])[1 + i % 6],
			       CASE WHEN i % 10 = 0 THEN ARRAY['vip'] WHEN i % 7 = 0 THEN ARRAY['customer'] ELSE ARRAY[]::text[] END,
			       now() - (i % 730) * interval '1 day' - (i % 86400) * interval '1 second', now()
			FROM generate_series($2::int, $3::int) i`, ws, from, to)
		if err != nil {
			return res, err
		}
		res.Contacts += n
	}

	// Lists: everyone, every third contact, every twentieth.
	for _, l := range []struct {
		name  string
		every int
	}{{"Newsletter", 1}, {"Product updates", 3}, {"Beta program", 20}} {
		id := uuid.Must(uuid.NewV7())
		if _, err := pool.Exec(ctx, `INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, $3)`, id, ws, l.name); err != nil {
			return res, err
		}
		if _, err := exec("list "+l.name, `
			INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at, created_at)
			SELECT $1, c.id, CASE WHEN c.status = 'active' THEN 'subscribed' ELSE 'unsubscribed' END, 'import', c.created_at, c.created_at
			FROM contacts c
			WHERE c.workspace_id = $2 AND (substring(c.email FROM '^load([0-9]+)@')::int % $3) = 0`, id, ws, l.every); err != nil {
			return res, err
		}
	}

	// Three sent campaigns, five, three and one month ago: a message per
	// active recipient (the second to every other one), delivered, with
	// opens for a quarter and clicks for one in twenty-five. The last one
	// recorded mail clients, places and times (campaign_dims, M57 G-12).
	for _, c := range []struct {
		name      string
		ago       time.Duration
		modulo    int
		breakdown bool
	}{
		{"June product update", 150 * 24 * time.Hour, 1, false},
		{"August digest", 90 * 24 * time.Hour, 2, false},
		{"October product update", 30 * 24 * time.Hour, 1, true},
	} {
		id := uuid.Must(uuid.NewV7())
		sent := time.Now().Add(-c.ago).UTC().Truncate(time.Hour)
		if _, err := pool.Exec(ctx, `
			INSERT INTO campaigns (id, workspace_id, name, status, subject, created_at, updated_at, started_at, completed_at, scheduled_at)
			VALUES ($1, $2, $3, 'sent', $3, $4::timestamptz, $4::timestamptz, $4::timestamptz, $4::timestamptz + interval '40 minutes', $4::timestamptz)`, id, ws, c.name, sent); err != nil {
			return res, err
		}
		n, err := exec("messages "+c.name, `
			INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, mailbox_provider, status, queued_at, sent_at, delivered_at, opened_at, clicked_at)
			SELECT gen_random_uuid(), $1, 'broadcast', $2, c.id, c.email, c.mailbox_provider, 'delivered',
			       $3::timestamptz, $3::timestamptz + (k % 2400) * interval '1 second', $3::timestamptz + (k % 2400) * interval '1 second' + interval '3 seconds',
			       CASE WHEN k % 4 = 0 THEN $3::timestamptz + (k % 86400) * interval '1 second' + interval '10 minutes' END,
			       CASE WHEN k % 25 = 0 THEN $3::timestamptz + (k % 86400) * interval '1 second' + interval '12 minutes' END
			FROM (SELECT c.*, substring(c.email FROM '^load([0-9]+)@')::int AS k FROM contacts c WHERE c.workspace_id = $1 AND c.status = 'active') c
			WHERE c.k % $4 = 0`, ws, id, sent, c.modulo)
		if err != nil {
			return res, err
		}
		res.Messages += n
		for _, ev := range []struct{ typ, at, where string }{
			{"delivered", "m.delivered_at", "true"},
			{"open", "m.opened_at", "m.opened_at IS NOT NULL"},
			{"click", "m.clicked_at", "m.clicked_at IS NOT NULL"},
		} {
			n, err := exec(ev.typ+" events "+c.name, `
				INSERT INTO delivery_events (id, workspace_id, message_id, campaign_id, contact_id, type, occurred_at, received_at, mailbox_provider,
				                             recipient, url, user_agent, ip)
				SELECT gen_random_uuid(), m.workspace_id, m.id, m.campaign_id, m.contact_id, '`+ev.typ+`', `+ev.at+`, `+ev.at+` + interval '2 seconds',
				       m.mailbox_provider, m.email,
				       CASE WHEN '`+ev.typ+`' = 'click' THEN 'https://acme.io/blog/october' ELSE '' END,
				       CASE WHEN '`+ev.typ+`' IN ('open', 'click') THEN 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X)' ELSE '' END,
				       CASE WHEN '`+ev.typ+`' IN ('open', 'click') THEN '198.51.100.7'::inet END
				FROM messages m WHERE m.campaign_id = $1 AND `+ev.where, id)
			if err != nil {
				return res, err
			}
			res.Events += n
		}
		if c.breakdown {
			if _, err := exec("breakdown "+c.name, breakdownSQL, id); err != nil {
				return res, err
			}
		}
	}
	if _, err := exec("analyze", `ANALYZE contacts, list_memberships, messages, delivery_events`); err != nil {
		return res, err
	}
	res.Took = time.Since(start)
	return res, nil
}

// breakdownSQL fills a campaign's campaign_dims from its messages' opens and
// clicks, spread over ten mail apps, seven systems, four devices, 60
// countries, 2,000 cities (as many rows as a large send reaches), 24 hours
// and 7 weekdays, so the report reads as many rows as a real one would.
const breakdownSQL = `
	WITH t AS (
	    SELECT count(*) FILTER (WHERE opened_at IS NOT NULL)::float8 AS opened, count(*) FILTER (WHERE clicked_at IS NOT NULL)::float8 AS clicked
	    FROM messages WHERE campaign_id = $1
	), d AS (
	    SELECT 'client' AS dim, u.k AS key, u.w FROM unnest(ARRAY['gmail','apple_mail','outlook','yahoo_mail','chrome','thunderbird','safari','firefox','edge','other'],
	                                                     ARRAY[35,30,15,6,4,1,3,1,1,4]::float8[]) AS u(k, w)
	    UNION ALL SELECT 'os', u.k, u.w FROM unnest(ARRAY['unknown','ios','windows','macos','android','linux','chromeos'], ARRAY[40,25,15,9,8,2,1]::float8[]) AS u(k, w)
	    UNION ALL SELECT 'device', u.k, u.w FROM unnest(ARRAY['unknown','mobile','desktop','tablet'], ARRAY[40,30,27,3]::float8[]) AS u(k, w)
	    UNION ALL SELECT 'country', chr(65 + i / 26) || chr(65 + i % 26), 1.0 / (i + 1) FROM generate_series(0, 59) i
	    UNION ALL SELECT 'city', chr(65 + (i % 60) / 26) || chr(65 + (i % 60) % 26) || '|City ' || i, 1.0 / (i + 1) FROM generate_series(0, 1999) i
	    UNION ALL SELECT 'hour', i::text, 1 + 4 * exp(-((i - 10) ^ 2) / 8.0) FROM generate_series(0, 23) i
	    UNION ALL SELECT 'weekday', i::text, CASE WHEN i < 6 THEN 3 ELSE 1 END FROM generate_series(1, 7) i
	), s AS (
	    SELECT dim, sum(w) AS total FROM d GROUP BY dim
	), ins AS (
	    INSERT INTO campaign_dims (campaign_id, dim, key, opened_human, clicked)
	    SELECT $1, d.dim, d.key, round(t.opened * d.w / s.total)::bigint, round(t.clicked * d.w / s.total)::bigint
	    FROM d JOIN s USING (dim) CROSS JOIN t
	)
	UPDATE campaigns SET breakdown = true WHERE id = $1`
