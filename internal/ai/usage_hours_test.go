package ai_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// The monthly caps read hourly totals that a trigger keeps from the
// activity log (ai_usage_hours, 03652), so reserving doesn't sum the
// month's requests. Whatever writes the log, through the service or not,
// at once or not, in this hour or earlier ones, the totals stay its sums.
func TestAIUsageHours(t *testing.T) {
	first, mine := newModelServer(t), newModelServer(t)
	p := newPersonal(t, first, map[string]any{"capKind": "usd", "capUsd": 1000})
	p.enable(1000)
	_, model := p.addOwn(mine.srv.URL, "", 1, 2)
	p.pick("draft", model)
	ctx := context.Background()
	gone := p.env.CreateUser(apptest.UserOpts{Email: "gone@registry.test"})
	p.env.AddMember(p.ws, gone.ID, authz.RoleEditor)
	other := p.env.CreateWorkspace("Elsewhere", p.owner)
	now := time.Now()

	var wg sync.WaitGroup
	errs := make(chan error, 64)
	// Drafts on the workspace's key and on the member's own, at once.
	for i := range 8 {
		c := p.c
		if i%2 == 1 {
			c = p.mc
		}
		wg.Go(func() {
			if res := draftBy(c, p.base); res.Status != 200 {
				errs <- fmt.Errorf("draft %d: %d %s", i, res.Status, res.Body)
			}
		})
	}
	// Meanwhile rows written, changed and deleted directly: both budgets,
	// a person who goes, another workspace, this hour and earlier ones.
	for i := range 48 {
		wg.Go(func() {
			id := uuid.Must(uuid.NewV7())
			ws, user, personal := p.ws.ID, p.owner, i%3 == 0
			switch {
			case i%5 == 0:
				ws = other.ID
			case personal && i%4 == 3:
				user = gone.ID
			case personal:
				user = p.member
			}
			at := now.Add(-time.Duration(i*97) * time.Minute)
			steps := []string{
				`INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, input_tokens, output_tokens, cost_micros, outcome, created_at, personal)
				 VALUES ($1, $2, $3, 'rewrite', 'compatible', 'model-a', $4, $5, $6, 'accepted', $7, $8)`,
			}
			if i%4 == 0 {
				steps = append(steps, `UPDATE ai_requests SET cost_micros = cost_micros * 3 + 1, output_tokens = output_tokens + 7 WHERE id = $1`)
			}
			if i%7 == 0 {
				steps = append(steps, `UPDATE ai_requests SET created_at = created_at - interval '90 minutes' WHERE id = $1`)
			}
			if i%6 == 0 {
				steps = append(steps, `DELETE FROM ai_requests WHERE id = $1`)
			}
			for n, q := range steps {
				args := []any{id}
				if n == 0 {
					args = append(args, ws, user, 100+i, 10+i, int64(1000+i*13), at, personal)
				}
				if _, err := p.env.App.Pool.Exec(ctx, q, args...); err != nil {
					errs <- fmt.Errorf("row %d: %w", i, err)
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	// Requests in one hour on both sides of a time within it, which the
	// totals since then count only after it.
	hour := now.Add(-5 * time.Hour).Truncate(time.Hour)
	since := hour.Add(40 * time.Minute)
	for i, at := range []time.Time{hour.Add(10 * time.Minute), hour.Add(50 * time.Minute), hour.Add(10 * time.Minute), hour.Add(50 * time.Minute)} {
		user, personal := p.owner, i >= 2
		if personal {
			user = p.member
		}
		p.exec(`INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, input_tokens, output_tokens, cost_micros, outcome, created_at, personal)
			VALUES ($1, $2, $3, 'rewrite', 'compatible', 'model-a', 50, 5, 500, 'accepted', $4, $5)`, uuid.Must(uuid.NewV7()), p.ws.ID, user, at, personal)
	}
	// The person's rows lose their owner, as when their account is
	// deleted; the oldest rows go, as the activity log's purge does.
	p.exec(`UPDATE ai_requests SET user_id = NULL WHERE user_id = $1`, gone.ID)
	p.exec(`DELETE FROM ai_requests WHERE created_at < $1`, now.Add(-48*time.Hour))

	sameAsLog(t, p.env.App.Pool, 20)

	// And the totals the caps read, from times on the hour and between.
	q := store.New(p.env.App.Pool)
	for _, since := range []time.Time{now.Add(-37 * time.Minute), since, now.Add(-3 * time.Hour).Truncate(time.Hour), now.Add(-29*time.Hour - 13*time.Minute), now.Add(-72 * time.Hour)} {
		for _, ws := range []uuid.UUID{p.ws.ID, other.ID} {
			got, err := q.AIBudgetTotals(ctx, store.AIBudgetTotalsParams{WorkspaceID: ws, Since: since})
			if err != nil {
				t.Fatal(err)
			}
			got.HeldTokens, got.HeldCostMicros, got.HeldUncertain = 0, 0, 0
			var want store.AIBudgetTotalsRow
			if err := p.env.App.Pool.QueryRow(ctx, `SELECT coalesce(sum(cost_micros), 0), coalesce(sum(input_tokens), 0), coalesce(sum(output_tokens), 0), count(*)
				FROM ai_requests WHERE workspace_id = $1 AND created_at >= $2 AND NOT personal`, ws, since).
				Scan(&want.CostMicros, &want.InputTokens, &want.OutputTokens, &want.Requests); err != nil {
				t.Fatal(err)
			}
			if got != want || (ws == p.ws.ID && want.Requests == 0) {
				t.Fatalf("the workspace's use since %s: %+v, the log has %+v", since, got, want)
			}
		}
		got, err := q.AIPersonalBudgetTotals(ctx, store.AIPersonalBudgetTotalsParams{WorkspaceID: p.ws.ID, UserID: p.member, Since: since})
		if err != nil {
			t.Fatal(err)
		}
		got.HeldCostMicros, got.HeldUncertain = 0, 0
		var want store.AIPersonalBudgetTotalsRow
		if err := p.env.App.Pool.QueryRow(ctx, `SELECT coalesce(sum(cost_micros), 0), count(*) FROM ai_requests
			WHERE workspace_id = $1 AND user_id = $2 AND created_at >= $3 AND personal`, p.ws.ID, p.member, since).Scan(&want.CostMicros, &want.Requests); err != nil {
			t.Fatal(err)
		}
		if got != want || want.Requests == 0 {
			t.Fatalf("the member's own use since %s: %+v, the log has %+v", since, got, want)
		}
	}
}

// An upgrade fills the totals from the log it finds: this month's caps
// don't start again at zero.
func TestAIUsageHoursFilledOnUpgrade(t *testing.T) {
	env := apptest.New(t)
	user := env.CreateUser(apptest.UserOpts{Email: "owner@upgrade.test"})
	ws := env.CreateWorkspace("Upgrade", user.ID)
	ctx := context.Background()
	pool := env.App.Pool
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	for {
		st, err := db.Status(ctx, pool, log)
		if err != nil {
			t.Fatal(err)
		}
		latest := int64(0)
		for _, m := range st {
			if m.Applied {
				latest = max(latest, m.Version)
			}
		}
		if latest < 3652 {
			break
		}
		if err := db.MigrateDown(ctx, pool, log); err != nil {
			t.Fatal(err)
		}
	}
	// Written by the release before, without the trigger.
	for i := range 6 {
		if _, err := pool.Exec(ctx, `INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, input_tokens, output_tokens, cost_micros, outcome, created_at, personal)
			VALUES ($1, $2, $3, 'rewrite', 'compatible', 'model-a', 100, 10, $4, 'accepted', now() - $5 * interval '1 hour', $6)`,
			uuid.Must(uuid.NewV7()), ws.ID, user.ID, 1000+i, i, i%2 == 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.MigrateUp(ctx, pool, log); err != nil {
		t.Fatal(err)
	}
	sameAsLog(t, pool, 4)
}

// sameAsLog fails unless the hourly totals are what summing the activity
// log by budget and hour gives, in at least atLeast rows.
func sameAsLog(t *testing.T, pool *pgxpool.Pool, atLeast int) {
	t.Helper()
	ctx := context.Background()
	// One line a row.
	read := func(query string) string {
		t.Helper()
		rows, err := pool.Query(ctx, query)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var b strings.Builder
		for rows.Next() {
			v, err := rows.Values()
			if err != nil {
				t.Fatal(err)
			}
			for i, x := range v {
				if id, ok := x.([16]byte); ok {
					v[i] = uuid.UUID(id)
				}
			}
			fmt.Fprintln(&b, v...)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	kept := read(`SELECT workspace_id, personal, user_id, hour, requests, input_tokens, output_tokens, cost_micros FROM ai_usage_hours ORDER BY 1, 2, 3, 4`)
	summed := read(`SELECT workspace_id, personal, CASE WHEN personal THEN user_id ELSE '00000000-0000-0000-0000-000000000000' END,
			date_bin('1 hour', created_at, '2000-01-01 00:00:00+00'), count(*), sum(input_tokens)::bigint, sum(output_tokens)::bigint, sum(cost_micros)::bigint
		FROM ai_requests WHERE NOT personal OR user_id IS NOT NULL GROUP BY 1, 2, 3, 4 ORDER BY 1, 2, 3, 4`)
	if kept != summed || strings.Count(kept, "\n") < atLeast {
		t.Fatalf("the hourly totals aren't the log's:\nkept\n%s\nsummed\n%s", kept, summed)
	}
}
