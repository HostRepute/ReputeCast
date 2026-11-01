package approvals_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestInboxAtAThousandRequests times the approvals inbox with 1,000 open
// requests (review 2026-11 AP-3): opening it re-checks only the stale ones,
// so with every request fresh it is one light page, and after ten
// campaigns change it re-checks those ten and no others.
func TestInboxAtAThousandRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds 1,000 approval requests")
	}
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	// Open the inbox once, so the demo's own requests are checked.
	c.Do("GET", ws+"/approvals?status=all&days=0", nil).MustStatus(t, 200)

	var campaign, request uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT r.campaign_id, r.id FROM approval_requests r JOIN campaigns c ON c.id = r.campaign_id
		WHERE c.name = 'November Product Update' AND r.status = 'pending'`).Scan(&campaign, &request); err != nil {
		t.Fatal(err)
	}
	conn, err := env.App.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	for _, q := range []string{
		`CREATE TEMP TABLE tc AS SELECT c.* FROM campaigns c CROSS JOIN generate_series(1, 1000) WHERE c.id = '` + campaign.String() + `'`,
		`UPDATE tc SET id = gen_random_uuid(), updated_at = now() - interval '1 hour'`,
		`UPDATE tc SET name = name || ' · ' || left(id::text, 8)`,
		`INSERT INTO campaigns SELECT * FROM tc`,
		`CREATE TEMP TABLE tr AS SELECT r.* FROM approval_requests r CROSS JOIN generate_series(1, 1000) WHERE r.id = '` + request.String() + `'`,
		`ALTER TABLE tr ADD COLUMN n bigserial`,
		`ALTER TABLE tc ADD COLUMN n bigserial`,
		`UPDATE tr SET id = gen_random_uuid(), revalidated_at = now(), campaign_id = tc.id FROM tc WHERE tc.n = tr.n`,
		`ALTER TABLE tr DROP COLUMN n`,
		`INSERT INTO approval_requests SELECT * FROM tr`,
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v: %.80s", err, q)
		}
	}
	checkedSince := func(at time.Time) int {
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM approval_requests WHERE revalidated_at > $1`, at).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	start := time.Now()
	var list struct {
		Items      []struct{ ID string }
		NextCursor string
	}
	c.Do("GET", ws+"/approvals?status=all&days=0", nil).MustStatus(t, 200).JSON(t, &list)
	fresh := time.Since(start)
	if len(list.Items) != 50 || list.NextCursor == "" || checkedSince(start) != 0 {
		t.Fatalf("a fresh inbox: %d items, next %q, %d re-checked", len(list.Items), list.NextCursor, checkedSince(start))
	}

	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET updated_at = now() WHERE id IN (
		SELECT r.campaign_id FROM approval_requests r WHERE r.revalidated_at IS NOT NULL AND r.campaign_id <> $1
		  AND r.campaign_id IN (SELECT id FROM campaigns WHERE name LIKE 'November Product Update · %') LIMIT 10)`, campaign); err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	c.Do("GET", ws+"/approvals?status=all&days=0", nil).MustStatus(t, 200)
	changed := time.Since(start)
	if n := checkedSince(start); n != 10 {
		t.Fatalf("re-checked %d requests, want the 10 whose campaign changed", n)
	}
	t.Logf("inbox with 1,000 open requests: %v fresh, %v with 10 changed", fresh, changed)
	// Generous bounds: they catch a re-check of every request, not a slow machine.
	if fresh > 3*time.Second || changed > 10*time.Second {
		t.Fatalf("too slow: %v fresh, %v with 10 changed", fresh, changed)
	}
}
