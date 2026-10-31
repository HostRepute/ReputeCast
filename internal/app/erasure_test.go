package app_test

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
)

const erasePath = "/workspaces/acme-newsletter/contacts/erase"

// livePayer is the SQL condition of a contact who still pays through Stripe.
const livePayer = `EXISTS (SELECT 1 FROM paid_subscriptions ps WHERE ps.contact_id = c.id AND ps.kind = 'stripe' AND ps.mode = 'live'
	AND ps.ended_at IS NULL AND ps.status IN ('trialing','active','past_due'))`

// DP-1: erasing contacts who have data in every registered table leaves no
// trace of their address or phone number anywhere outside the exempt
// tables, and every declared column of their rows is emptied.
func TestEraseLeavesNothingPersonal(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	pool := env.App.Pool
	var ws uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	columns := tableColumns(t, env)

	// One contact from every registered table that links rows to contacts,
	// none who still pays through Stripe.
	picked := map[uuid.UUID]bool{}
	var sampled, unsampled []string
	for _, src := range app.ErasureSources() {
		for _, tb := range src.Tables {
			link := ""
			switch {
			case slices.Contains(columns[tb.Name], "contact_id"):
				link = "t.contact_id"
			case slices.Contains(columns[tb.Name], "contact_ids"):
				link = "t.contact_ids[1]"
			case slices.Contains(columns[tb.Name], "redeemed_contact"):
				link = "t.redeemed_contact"
			}
			if link == "" {
				continue
			}
			var id uuid.UUID
			err := pool.QueryRow(ctx, fmt.Sprintf(`SELECT c.id FROM %s t JOIN contacts c ON c.id = %s
				WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND NOT %s ORDER BY c.id LIMIT 1`, pgx.Identifier{tb.Name}.Sanitize(), link, livePayer), ws).Scan(&id)
			if err == pgx.ErrNoRows {
				unsampled = append(unsampled, tb.Name)
				continue
			}
			if err != nil {
				t.Fatalf("%s: %v", tb.Name, err)
			}
			picked[id] = true
			sampled = append(sampled, tb.Name)
		}
	}
	t.Logf("contacts from %d tables; no demo rows in %v", len(sampled), unsampled)
	if len(sampled) < 25 {
		t.Fatalf("only %d registered tables have demo rows to erase: %v", len(sampled), sampled)
	}
	type who struct{ email, phone string }
	people := map[uuid.UUID]who{}
	ids := make([]string, 0, len(picked))
	for id := range picked {
		var w who
		if err := pool.QueryRow(ctx, `SELECT email, phone FROM contacts WHERE id = $1`, id).Scan(&w.email, &w.phone); err != nil {
			t.Fatal(err)
		}
		people[id] = w
		ids = append(ids, id.String())
	}

	var res struct{ Matched, Done int64 }
	c.Do("POST", erasePath, map[string]any{"ids": ids, "confirm": len(ids)}).MustStatus(t, 200).JSON(t, &res)
	if res.Done != int64(len(ids)) {
		t.Fatalf("erased %d of %d", res.Done, len(ids))
	}

	uids := make([]uuid.UUID, 0, len(picked))
	for id := range picked {
		uids = append(uids, id)
	}
	// Every declared column of the contacts' rows is empty; deleted tables
	// have no rows left.
	for _, src := range app.ErasureSources() {
		for _, tb := range src.Tables {
			if !slices.Contains(columns[tb.Name], "contact_id") || tb.Action == "keep" {
				continue
			}
			name := pgx.Identifier{tb.Name}.Sanitize()
			if tb.Action == "delete" {
				var n int
				if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+name+` WHERE contact_id = ANY ($1)`, uids).Scan(&n); err != nil {
					t.Fatal(err)
				}
				if n > 0 {
					t.Errorf("%s still has %d rows of erased contacts", tb.Name, n)
				}
				continue
			}
			for _, col := range tb.Columns {
				if col == "contact_id" || tb.Kept[col] != "" {
					continue
				}
				var n int
				q := fmt.Sprintf(`SELECT count(*) FROM %s WHERE contact_id = ANY ($1) AND %s IS NOT NULL
					AND %s::text NOT IN ('', '{}', '[]', '\x', 'unknown') AND %s::text NOT LIKE 'erased%%'`, name, col, col, col)
				if err := pool.QueryRow(ctx, q, uids).Scan(&n); err != nil {
					t.Fatalf("%s.%s: %v", tb.Name, col, err)
				}
				if n > 0 {
					t.Errorf("%s.%s still holds data for %d rows of erased contacts", tb.Name, col, n)
				}
			}
		}
	}

	// No address or phone number of theirs is left anywhere in the workspace
	// but in the exempt tables (the suppression list keeps the address on
	// purpose, and so do bounce events of a suppressed address).
	exempt := app.ErasureExempt()
	var all, unsuppressed []string
	for _, w := range people {
		var suppressed bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM suppressions WHERE workspace_id = $1 AND email = lower($2))`, ws, w.email).Scan(&suppressed); err != nil {
			t.Fatal(err)
		}
		for _, re := range []string{emailPattern(w.email), phonePattern(w.phone)} {
			if re == "" {
				continue
			}
			all = append(all, re)
			if !suppressed || re != emailPattern(w.email) {
				unsuppressed = append(unsuppressed, re)
			}
		}
	}
	for tbl, cols := range columns {
		if _, ok := exempt[tbl]; ok || strings.HasPrefix(tbl, "river_") || tbl == "goose_db_version" {
			continue
		}
		scope := ""
		switch {
		case slices.Contains(cols, "workspace_id"):
			scope = "t.workspace_id = $2 AND "
		case tbl == "import_rows":
			scope = "t.import_id IN (SELECT id FROM imports WHERE workspace_id = $2) AND "
		}
		pats := all
		if tbl == "delivery_events" {
			pats = unsuppressed
		}
		args := []any{"(" + strings.Join(pats, ")|(") + ")"}
		if scope != "" {
			args = append(args, ws)
		}
		var sample string
		err := pool.QueryRow(ctx, `SELECT left(t::text, 300) FROM `+pgx.Identifier{tbl}.Sanitize()+` t WHERE `+scope+`t::text ~* $1 LIMIT 1`, args...).Scan(&sample)
		if err == nil {
			t.Errorf("%s still mentions an erased contact's address or phone number: %s", tbl, sample)
		} else if err != pgx.ErrNoRows {
			t.Fatalf("%s: %v", tbl, err)
		}
	}
}

// emailPattern matches the address on its own, not inside a longer one
// (a.meyer@gmail.com is not jana.meyer@gmail.com).
func emailPattern(email string) string {
	if email == "" {
		return ""
	}
	return `(^|[^a-z0-9._%+-])` + regexp.QuoteMeta(strings.ToLower(email)) + `($|[^a-z0-9._%+-])`
}

// phonePattern matches the number on its own.
func phonePattern(phone string) string {
	if phone == "" {
		return ""
	}
	return `(^|[^0-9])` + regexp.QuoteMeta(phone) + `($|[^0-9])`
}

// DP-1: a contact who still pays through Stripe can't be erased; the whole
// erase is refused and nothing changes. A complimentary member is erased
// and their access ends.
func TestEraseRefusesLiveStripePayers(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	pool := env.App.Pool
	var payer, other uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT c.id FROM contacts c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.erased_at IS NULL AND `+livePayer+` ORDER BY c.id LIMIT 1`).Scan(&payer); err != nil {
		t.Fatalf("a live Stripe payer in the demo: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT c.id FROM contacts c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.erased_at IS NULL AND NOT `+livePayer+` ORDER BY c.id LIMIT 1`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	r := c.Do("POST", erasePath, map[string]any{"ids": []string{payer.String(), other.String()}, "confirm": 2})
	if r.Status != 409 {
		t.Fatalf("status %d, want 409: %s", r.Status, r.Body)
	}
	p := r.Problem(t)
	meta, _ := p["meta"].(map[string]any)
	if p["code"] != "paid_members" || meta["refused"] != float64(1) || !strings.Contains(fmt.Sprint(p["detail"]), "1 contact still pays through Stripe") {
		t.Fatalf("problem = %v", p)
	}
	var erased int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE id = ANY ($1) AND erased_at IS NOT NULL`, []uuid.UUID{payer, other}).Scan(&erased); err != nil || erased != 0 {
		t.Fatalf("a refused erase erased %d contacts (%v)", erased, err)
	}

	var comp uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT ps.contact_id FROM paid_subscriptions ps JOIN contacts c ON c.id = ps.contact_id
		WHERE ps.kind = 'complimentary' AND ps.ended_at IS NULL AND ps.mode = 'live' AND c.erased_at IS NULL AND NOT `+livePayer+` LIMIT 1`).Scan(&comp); err != nil {
		t.Skipf("no complimentary member in the demo: %v", err)
	}
	c.Do("POST", erasePath, map[string]any{"ids": []string{comp.String()}, "confirm": 1}).MustStatus(t, 200)
	var open int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paid_subscriptions WHERE contact_id = $1 AND ended_at IS NULL`, comp).Scan(&open); err != nil || open != 0 {
		t.Fatalf("the erased complimentary member keeps %d open subscriptions (%v)", open, err)
	}
}

// D2: erasing members whose subscriptions ended leaves revenue, MRR and
// churn exactly as they were.
func TestErasureKeepsRevenueNumbers(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	pool := env.App.Pool
	rows, err := pool.Query(ctx, `SELECT DISTINCT c.id FROM paid_subscriptions ps JOIN contacts c ON c.id = ps.contact_id JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND ps.mode = 'live' AND ps.ended_at IS NOT NULL AND c.erased_at IS NULL AND NOT `+livePayer+` LIMIT 20`)
	if err != nil {
		t.Fatal(err)
	}
	members, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil || len(members) == 0 {
		t.Fatalf("former paid members in the demo: %d (%v)", len(members), err)
	}
	snapshot := func() (string, int) {
		r := c.Do("GET", "/workspaces/acme-newsletter/earn/revenue", nil).MustStatus(t, 200)
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM earn_mrr_changes`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return string(r.Body), n
	}
	beforeBody, beforeChanges := snapshot()
	ids := make([]string, len(members))
	for i, id := range members {
		ids[i] = id.String()
	}
	c.Do("POST", erasePath, map[string]any{"ids": ids, "confirm": len(ids)}).MustStatus(t, 200)
	afterBody, afterChanges := snapshot()
	if afterChanges != beforeChanges {
		t.Fatalf("earn_mrr_changes went from %d to %d rows", beforeChanges, afterChanges)
	}
	if b, a := stripVolatile(beforeBody), stripVolatile(afterBody); a != b {
		i := 0
		for i < len(a) && i < len(b) && a[i] == b[i] {
			i++
		}
		from := max(0, i-200)
		t.Fatalf("revenue changed after erasing %d former members, at byte %d:\nbefore …%s\nafter  …%s", len(members), i, b[from:min(len(b), i+200)], a[from:min(len(a), i+200)])
	}
}

// stripVolatile drops the times the overview takes from the clock ("to",
// the chart's last point) so only the numbers are compared.
func stripVolatile(body string) string {
	return volatileTo.ReplaceAllString(body, `"$1":"now"`)
}

var volatileTo = regexp.MustCompile(`"(to|at)":"[^"]*\.[0-9]+Z"`)

// tableColumns lists every base table's columns.
func tableColumns(t *testing.T, env *apptest.Env) map[string][]string {
	t.Helper()
	rows, err := env.App.Pool.Query(context.Background(), `SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name IN (SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var tbl, col string
		if err := rows.Scan(&tbl, &col); err != nil {
			t.Fatal(err)
		}
		out[tbl] = append(out[tbl], col)
	}
	return out
}

// DP-2: a contact's data export streams everything with no cap (10,050
// emails, past the old 10,000), has a section for every registered export,
// says it is complete, and is logged.
func TestContactExportIsComplete(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	pool := env.App.Pool
	var ws, contact uuid.UUID
	var email string
	if err := pool.QueryRow(ctx, `SELECT c.workspace_id, c.id, c.email FROM contacts c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.erased_at IS NULL ORDER BY c.id LIMIT 1`).Scan(&ws, &contact, &email); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, contact_id, stream, email, status, queued_at)
		SELECT gen_random_uuid(), $1, $2, 'broadcast', $3, 'delivered', now() - make_interval(mins => g) FROM generate_series(1, 10050) g`, ws, contact, email); err != nil {
		t.Fatal(err)
	}
	var want int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE contact_id = $1`, contact).Scan(&want); err != nil {
		t.Fatal(err)
	}
	r := c.Do("GET", "/workspaces/acme-newsletter/contacts/"+contact.String()+"/export", nil).MustStatus(t, 200)
	if !strings.Contains(r.Header.Get("Content-Disposition"), "contact-") {
		t.Fatalf("Content-Disposition = %q", r.Header.Get("Content-Disposition"))
	}
	var doc struct {
		Contact  map[string]any   `json:"contact"`
		Messages []map[string]any `json:"messages"`
		Events   []map[string]any `json:"events"`
		Sections map[string]any   `json:"sections"`
		Cut      []map[string]any `json:"cut"`
		Complete bool             `json:"complete"`
	}
	r.JSON(t, &doc)
	if len(doc.Messages) != want || want <= 10000 {
		t.Fatalf("exported %d of %d messages", len(doc.Messages), want)
	}
	if !doc.Complete || len(doc.Cut) != 0 {
		t.Fatalf("complete = %v, cut = %v", doc.Complete, doc.Cut)
	}
	for _, src := range app.ErasureSources() {
		for _, ex := range src.Exports {
			if _, ok := doc.Sections[ex.Section]; !ok {
				t.Errorf("the export has no %q section (source %s)", ex.Section, src.Name)
			}
		}
	}
	var logged int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'contact.export' AND target_id = $1`, contact.String()).Scan(&logged); err != nil || logged != 1 {
		t.Fatalf("contact.export logged %d times (%v)", logged, err)
	}

	// A missing contact is a 404 before anything streams.
	c.Do("GET", "/workspaces/acme-newsletter/contacts/"+uuid.New().String()+"/export", nil).MustStatus(t, 404)
}
