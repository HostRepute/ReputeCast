package audience_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
)

type pageOf struct {
	Items []struct {
		ID    string
		Email string
	}
	Total       int64
	TotalCapped bool
	NextCursor  string
	PrevCursor  string
}

// pagingWorkspace is a workspace of 61 contacts whose sort keys repeat and
// are often empty: ties and NULLs are where keysets go wrong.
func pagingWorkspace(t *testing.T) (*apptest.Env, *apptest.Client, string) {
	t.Helper()
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@paging.example", Name: "Pat Paging"})
	ws := env.CreateWorkspace("Paging Weekly", owner.ID)
	c := env.Client()
	c.SignIn("owner@paging.example", "")
	ctx := context.Background()
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for i := range 61 {
		id := uuid.Must(uuid.NewV7())
		var activity, open *time.Time
		if i%3 != 0 {
			a := base.Add(time.Duration(i%5) * time.Hour) // five values, many ties
			activity = &a
		}
		if i%4 == 0 {
			o := base.Add(time.Duration(i%2) * time.Hour)
			open = &o
		}
		joined := base.AddDate(0, 0, -(i % 7))
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name, created_at, last_activity_at, last_open_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, ws.ID, fmt.Sprintf("reader%02d@paging.example", i), fmt.Sprintf("Reader %d", i), joined, activity, open); err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contact_scores (contact_id, workspace_id, score, events_part, fields_part, version) VALUES ($1, $2, $3, 0, 0, 1)`,
				id, ws.ID, (i%4)*10); err != nil {
				t.Fatal(err)
			}
		}
	}
	return env, c, "/workspaces/" + ws.Slug
}

// Review 2026-11 AU-1, AU-9: every order pages forward and back with
// cursors, reaching each contact once, in the order one query gives.
func TestContactsPageWithCursors(t *testing.T) {
	_, c, base := pagingWorkspace(t)
	get := func(q url.Values) pageOf {
		t.Helper()
		var p pageOf
		c.Do(http.MethodGet, base+"/contacts?"+q.Encode(), nil).MustStatus(t, 200).JSON(t, &p)
		return p
	}
	for _, sort := range []string{"", "activity", "score", "joined", "last_open", "email"} {
		t.Run("sort="+sort, func(t *testing.T) {
			whole := get(url.Values{"sort": {sort}, "pageSize": {"100"}})
			if len(whole.Items) != 61 || whole.Total != 61 || whole.NextCursor != "" || whole.PrevCursor != "" {
				t.Fatalf("one page: %d items, total %d, cursors %q %q", len(whole.Items), whole.Total, whole.NextCursor, whole.PrevCursor)
			}
			want := make([]string, len(whole.Items))
			for i, it := range whole.Items {
				want[i] = it.ID
			}
			// Forward in pages of 7.
			var got []string
			var pages []pageOf
			q := url.Values{"sort": {sort}, "pageSize": {"7"}}
			for {
				p := get(q)
				pages = append(pages, p)
				for _, it := range p.Items {
					got = append(got, it.ID)
				}
				if len(pages) > 1 && p.PrevCursor == "" {
					t.Fatalf("page %d has no way back", len(pages))
				}
				if p.NextCursor == "" {
					break
				}
				q.Set("cursor", p.NextCursor)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("forward:\n got %v\nwant %v", got, want)
			}
			if len(pages) != 9 || pages[0].PrevCursor != "" {
				t.Fatalf("%d pages, first page's prev %q", len(pages), pages[0].PrevCursor)
			}
			// Back from the last page, each page as it was going forward.
			back := pages[len(pages)-1]
			for i := len(pages) - 2; i >= 0; i-- {
				q.Set("cursor", back.PrevCursor)
				back = get(q)
				if fmt.Sprint(back.Items) != fmt.Sprint(pages[i].Items) {
					t.Fatalf("back to page %d:\n got %v\nwant %v", i+1, back.Items, pages[i].Items)
				}
			}
			if back.PrevCursor != "" {
				t.Fatalf("the first page reached going back still has a prev cursor")
			}
		})
	}
	// A cursor of another order, or a broken one, asks to start again.
	first := get(url.Values{"sort": {"email"}, "pageSize": {"7"}})
	c.Do(http.MethodGet, base+"/contacts?sort=joined&cursor="+first.NextCursor, nil).MustStatus(t, 422)
	c.Do(http.MethodGet, base+"/contacts?cursor=not-a-cursor", nil).MustStatus(t, 422)
	// The page number still works for the API's first pages.
	if p := get(url.Values{"sort": {"email"}, "pageSize": {"7"}, "page": {"2"}}); len(p.Items) != 7 || p.Items[0].Email != "reader07@paging.example" || p.PrevCursor == "" {
		t.Fatalf("page 2 by number: %+v", p)
	}
}

// Review 2026-11 D5: counts stop at 10,000 ("10,000+") unless the exact
// one is asked for.
func TestContactCountStopsAtTenThousand(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	var ws uuid.UUID
	ctx := context.Background()
	if err := env.App.Pool.QueryRow(ctx, `SELECT workspace_id FROM contacts WHERE email = 'reader00@paging.example'`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email)
		SELECT gen_random_uuid(), $1, 'bulk' || g || '@paging.example' FROM generate_series(1, 10000) g`, ws); err != nil {
		t.Fatal(err)
	}
	var p pageOf
	c.Do(http.MethodGet, base+"/contacts?pageSize=5", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Total != 10000 || !p.TotalCapped || len(p.Items) != 5 {
		t.Fatalf("capped: %d %v", p.Total, p.TotalCapped)
	}
	p = pageOf{}
	c.Do(http.MethodGet, base+"/contacts?pageSize=5&exact=true", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Total != 10061 || p.TotalCapped {
		t.Fatalf("exact: %d %v", p.Total, p.TotalCapped)
	}
	p = pageOf{}
	c.Do(http.MethodGet, base+"/contacts?pageSize=5&q=reader1", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Total != 10 || p.TotalCapped {
		t.Fatalf("a narrow filter counts exactly: %d %v", p.Total, p.TotalCapped)
	}
	// A search reads its matches once and pages through them by id
	// (review 2026-11 OP-13): the next page holds the other five, in order,
	// with the same count, and a page back returns to the first.
	first := p
	p = pageOf{}
	c.Do(http.MethodGet, base+"/contacts?pageSize=5&q=reader1&cursor="+url.QueryEscape(first.NextCursor), nil).MustStatus(t, 200).JSON(t, &p)
	seen := map[string]bool{}
	for _, it := range append(first.Items, p.Items...) {
		if !strings.HasPrefix(it.Email, "reader1") {
			t.Fatalf("%s doesn't match the search", it.Email)
		}
		seen[it.Email] = true
	}
	if len(seen) != 10 || p.Total != 10 || p.NextCursor != "" || p.PrevCursor == "" {
		t.Fatalf("second page: %d distinct, total %d, next %q", len(seen), p.Total, p.NextCursor)
	}
	back := pageOf{}
	c.Do(http.MethodGet, base+"/contacts?pageSize=5&q=reader1&cursor="+url.QueryEscape(p.PrevCursor), nil).MustStatus(t, 200).JSON(t, &back)
	if len(back.Items) != 5 || back.Items[0].Email != first.Items[0].Email {
		t.Fatalf("back to the first page: %+v", back.Items)
	}
	// A search past the cap reads "10,000+" and still shows its first page.
	p = pageOf{}
	c.Do(http.MethodGet, base+"/contacts?pageSize=5&q=paging", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Total != 10000 || !p.TotalCapped || len(p.Items) != 5 || p.NextCursor == "" {
		t.Fatalf("a broad search: %d %v, %d rows", p.Total, p.TotalCapped, len(p.Items))
	}
	// A search with no match answers an empty page.
	p = pageOf{}
	c.Do(http.MethodGet, base+"/contacts?pageSize=5&q=nobody-here", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Total != 0 || p.TotalCapped || len(p.Items) != 0 {
		t.Fatalf("no match: %d %v, %d rows", p.Total, p.TotalCapped, len(p.Items))
	}
}

// Review 2026-11 AU-2, AU-3, AU-9: each order and the search can use an
// index. The test data is small, so sequential scans are switched off to
// ask whether the planner could use one at all.
func TestContactQueriesUseIndexes(t *testing.T) {
	env, _, _ := pagingWorkspace(t)
	ctx := context.Background()
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT workspace_id FROM contacts WHERE email = 'reader00@paging.example'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	ws, err := env.App.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		t.Fatal(err)
	}
	// Ten thousand more contacts and fresh statistics, so the planner
	// weighs a filter the way it would on a large audience.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name, last_name, source, created_at, last_activity_at)
		SELECT gen_random_uuid(), $1, 'bulk' || g || '@paging.example', 'Bulk', 'Reader ' || g, 'form:' || (g % 40), now() - g * interval '1 minute',
		       now() - (g % 500) * interval '1 hour'
		FROM generate_series(1, 10000) g`, wsID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `ANALYZE contacts`); err != nil {
		t.Fatal(err)
	}
	svc := env.App.Audience()
	plan := func(f audience.ContactFilter, count, alone bool) string {
		t.Helper()
		sql, args, err := audience.ListSQL(ctx, svc, &ws, f, audience.ContactPage{PageSize: 25})
		if err != nil {
			t.Fatal(err)
		}
		if count {
			// The filter alone, as the count line reads it.
			from := "ON cs.contact_id = c.id WHERE "
			where := sql[strings.Index(sql, from)+len(from) : strings.LastIndex(sql, "ORDER BY")]
			if alone {
				// The search's own condition, without the workspace's (on
				// a large audience it is the narrower one).
				where = strings.Replace(where, "c.workspace_id = $1 AND c.erased_at IS NULL", "$1::uuid IS NOT NULL", 1)
			}
			sql = "SELECT count(*) FROM contacts c WHERE " + where
			args = args[:len(args)-1] // the page's LIMIT
		}
		var out []string
		err = pgxTx(ctx, env, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
				return err
			}
			rows, err := tx.Query(ctx, "EXPLAIN "+sql, args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var line string
				if err := rows.Scan(&line); err != nil {
					return err
				}
				out = append(out, line)
			}
			return rows.Err()
		})
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(out, "\n")
	}
	for _, c := range []struct {
		name         string
		f            audience.ContactFilter
		count, alone bool
		index        string
	}{
		{"activity", audience.ContactFilter{}, false, false, "contacts_activity_order_idx"},
		{"joined", audience.ContactFilter{ScoreGrowthFilter: audience.ScoreGrowthFilter{Sort: "joined"}}, false, false, "contacts_joined_order_idx"},
		{"last open", audience.ContactFilter{ScoreGrowthFilter: audience.ScoreGrowthFilter{Sort: "last_open"}}, false, false, "contacts_last_open_order_idx"},
		{"email", audience.ContactFilter{ScoreGrowthFilter: audience.ScoreGrowthFilter{Sort: "email"}}, false, false, "contacts_email_key"},
		{"search", audience.ContactFilter{Q: "rivera"}, true, true, "contacts_email_trgm_idx"},
		{"search names", audience.ContactFilter{Q: "rivera"}, true, true, "contacts_name_trgm_idx"},
		{"search tags", audience.ContactFilter{Q: "rivera"}, true, true, "contacts_tags_idx"},
		{"partial address", audience.ContactFilter{Q: "reader07@paging"}, true, true, "contacts_email_trgm_idx"},
		{"whole address", audience.ContactFilter{Q: "Reader07@Paging.example"}, true, false, "contacts_email_key"},
		{"source", audience.ContactFilter{Source: "import:csv"}, true, false, "contacts_source_idx"},
	} {
		if p := plan(c.f, c.count, c.alone); !strings.Contains(p, c.index) {
			t.Errorf("%s doesn't use %s:\n%s", c.name, c.index, p)
		}
	}
}

func pgxTx(ctx context.Context, env *apptest.Env, fn func(pgx.Tx) error) error {
	tx, err := env.App.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return fn(tx)
}

// Review 2026-11 AU-3, AU-8, AU-9: saved views keep their counts until
// their filters change; they can be renamed, given columns (custom fields
// included) and deleted, each change audited.
func TestSavedViewsKeepCountsAndChange(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	c.Do(http.MethodPost, base+"/fields", map[string]any{"key": "plan", "label": "Plan", "type": "text"}).MustStatus(t, 201)
	type view struct {
		ID      string
		Name    string
		Count   int64
		Columns []string
		Filters struct{ Q, Sort string }
	}
	var v view
	c.Do(http.MethodPost, base+"/contacts/views", map[string]any{"name": "Readers 1x", "filters": map[string]any{"q": "reader1"}, "columns": []string{"tier", "field:plan"}}).
		MustStatus(t, 201).JSON(t, &v)
	if len(v.Columns) != 2 || v.Columns[1] != "field:plan" {
		t.Fatalf("created: %+v", v)
	}
	c.Do(http.MethodPost, base+"/contacts/views", map[string]any{"name": "Bad", "columns": []string{"field:nope"}}).MustStatus(t, 422)
	c.Do(http.MethodPost, base+"/contacts/views", map[string]any{"name": "Bad", "columns": []string{"password"}}).MustStatus(t, 422)
	counts := func() map[string]int64 {
		t.Helper()
		var vs struct{ Items []view }
		c.Do(http.MethodGet, base+"/contacts/views", nil).MustStatus(t, 200).JSON(t, &vs)
		out := map[string]int64{}
		for _, x := range vs.Items {
			out[x.Name] = x.Count
		}
		return out
	}
	if n := counts()["Readers 1x"]; n != 10 {
		t.Fatalf("count %d, want 10", n)
	}
	// Kept for ten minutes: a new reader1x contact doesn't show yet…
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email) SELECT gen_random_uuid(), workspace_id, 'reader1extra@paging.example'
		FROM contacts WHERE email = 'reader00@paging.example'`); err != nil {
		t.Fatal(err)
	}
	if n := counts()["Readers 1x"]; n != 10 {
		t.Fatalf("a kept count moved: %d", n)
	}
	// …and new filters are counted at once.
	c.Do(http.MethodPatch, base+"/contacts/views/"+v.ID, map[string]any{"name": "Readers 2x", "filters": map[string]any{"q": "reader2", "sort": "email"}}).
		MustStatus(t, 200).JSON(t, &v)
	if v.Name != "Readers 2x" || v.Filters.Sort != "email" || len(v.Columns) != 2 {
		t.Fatalf("updated: %+v", v)
	}
	if n := counts()["Readers 2x"]; n != 10 {
		t.Fatalf("new filters counted %d, want 10", n)
	}
	c.Do(http.MethodPatch, base+"/contacts/views/"+v.ID, map[string]any{"columns": []string{}}).MustStatus(t, 200).JSON(t, &v)
	if len(v.Columns) != 0 || v.Name != "Readers 2x" {
		t.Fatalf("columns cleared: %+v", v)
	}
	c.Do(http.MethodPatch, base+"/contacts/views/"+v.ID, map[string]any{"name": "  "}).MustStatus(t, 422)
	var audited int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'contact_view.update' AND target_id = $1`, v.ID).Scan(&audited); err != nil || audited != 2 {
		t.Fatalf("audited %d (%v)", audited, err)
	}
	c.Do(http.MethodDelete, base+"/contacts/views/"+v.ID, nil).MustStatus(t, 204)
	if _, ok := counts()["Readers 2x"]; ok {
		t.Fatal("deleted view still listed")
	}
	// The custom field's values come with the rows that ask for them.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET attributes = '{"plan": "team", "secret": "x"}' WHERE email = 'reader05@paging.example'`); err != nil {
		t.Fatal(err)
	}
	var p struct {
		Items []struct {
			Email  string
			Fields map[string]any
		}
	}
	c.Do(http.MethodGet, base+"/contacts?q=reader05&fields=plan", nil).MustStatus(t, 200).JSON(t, &p)
	if len(p.Items) != 1 || p.Items[0].Fields["plan"] != "team" || p.Items[0].Fields["secret"] != nil {
		t.Fatalf("fields: %+v", p.Items)
	}
}

// Review 2026-11 AU-3: a segment filters the table as a subquery; its
// members are the segment's whether it is static, rule-based or SQL.
func TestSegmentFilterIsASubquery(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE contacts SET tier = 'sunset' WHERE email LIKE 'reader1%'`); err != nil {
		t.Fatal(err)
	}
	var seg struct{ ID string }
	c.Do(http.MethodPost, base+"/segments", map[string]any{"name": "Sunset", "mode": "builder", "refresh": "dynamic",
		"definition": map[string]any{"groups": []any{map[string]any{"match": "all", "conditions": []any{map[string]any{"field": "tier", "op": "is", "value": "sunset"}}}}}}).
		MustStatus(t, 201).JSON(t, &seg)
	var p pageOf
	c.Do(http.MethodGet, base+"/contacts?pageSize=100&segment="+seg.ID, nil).MustStatus(t, 200).JSON(t, &p)
	if p.Total != 10 || len(p.Items) != 10 {
		t.Fatalf("segment members: %d", p.Total)
	}
	for _, it := range p.Items {
		if !strings.HasPrefix(it.Email, "reader1") {
			t.Fatalf("not a member: %s", it.Email)
		}
	}
	// Paged with cursors inside the segment.
	var first pageOf
	c.Do(http.MethodGet, base+"/contacts?pageSize=4&segment="+seg.ID, nil).MustStatus(t, 200).JSON(t, &first)
	var second pageOf
	c.Do(http.MethodGet, base+"/contacts?pageSize=4&segment="+seg.ID+"&cursor="+first.NextCursor, nil).MustStatus(t, 200).JSON(t, &second)
	if len(second.Items) != 4 || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("second page in the segment: %+v", second.Items)
	}
}
