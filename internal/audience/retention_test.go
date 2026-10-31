package audience_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
)

// DP-6: clearing the IP address and user agent of old opens and clicks in
// campaigns and automations leaves every campaign report exactly as it was,
// and leaves transactional events alone.
func TestRetentionClearsDetailKeepsReports(t *testing.T) {
	env, c := demoEnv(t)
	ctx := context.Background()
	pool := env.App.Pool
	var ws, campaign uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT d.workspace_id, d.campaign_id FROM delivery_events d JOIN workspaces w ON w.id = d.workspace_id
		WHERE w.slug = 'acme-newsletter' AND d.type IN ('open', 'click') AND (d.ip IS NOT NULL OR d.user_agent <> '') AND d.campaign_id IS NOT NULL
		GROUP BY 1, 2 ORDER BY count(*) DESC LIMIT 1`).Scan(&ws, &campaign); err != nil {
		t.Fatalf("a campaign with tracked opens: %v", err)
	}
	report := func() string {
		return string(c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+campaign.String()+"/report", nil).MustStatus(t, 200).Body)
	}
	detail := func(stream string) int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events d LEFT JOIN messages m ON m.id = d.message_id
			WHERE d.workspace_id = $1 AND d.type IN ('open', 'click') AND (d.ip IS NOT NULL OR d.user_agent <> '')
			AND coalesce(m.stream, 'broadcast') = ANY ($2)`, ws, []string{stream}).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before, txBefore := report(), detail("transactional")
	svc := &audience.Service{Pool: pool, Q: env.App.Q, Audit: env.App.Audit, Now: func() time.Time { return time.Now().AddDate(2, 0, 0) }}
	res, err := svc.RetainWorkspace(ctx, ws, 0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if res.DetailCleared == 0 {
		t.Fatal("nothing cleared")
	}
	if n := detail("broadcast") + detail("automation"); n != 0 {
		t.Fatalf("%d marketing opens and clicks still keep an IP or user agent", n)
	}
	if n := detail("transactional"); n != txBefore {
		t.Fatalf("transactional events with detail went from %d to %d", txBefore, n)
	}
	if after := report(); after != before {
		t.Fatalf("the report changed:\nbefore %.600s\nafter  %.600s", before, after)
	}
}

// DP-4: a double opt-in signup never confirmed is erased once the setting's
// days have passed; contacts the team added or imported, recent signups and
// anyone who once confirmed are not.
func TestRetentionErasesUnconfirmedSignups(t *testing.T) {
	env, _ := demoEnv(t)
	ctx := context.Background()
	pool := env.App.Pool
	var ws, list uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT w.id, l.id FROM workspaces w JOIN lists l ON l.workspace_id = w.id WHERE w.slug = 'acme-newsletter' ORDER BY l.created_at LIMIT 1`).Scan(&ws, &list); err != nil {
		t.Fatal(err)
	}
	add := func(email, source string, age time.Duration, confirmedOnce bool) uuid.UUID {
		t.Helper()
		id := uuid.Must(uuid.NewV7())
		at := time.Now().Add(-age)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO contacts (id, workspace_id, email, consent, source, created_at) VALUES ($1, $2, $3, 'pending', $4, $5)`, []any{id, ws, email, source, at}},
			{`INSERT INTO list_memberships (list_id, contact_id, status, source, created_at) VALUES ($1, $2, 'pending', $3, $4)`, []any{list, id, source, at}},
			{`INSERT INTO consent_events (id, contact_id, list_id, action, source, occurred_at) VALUES ($1, $2, $3, 'subscribed', $4, $5)`, []any{uuid.Must(uuid.NewV7()), id, list, source, at}},
		} {
			if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
				t.Fatal(err)
			}
		}
		if confirmedOnce {
			if _, err := pool.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, occurred_at) VALUES ($1, $2, $3, 'confirmed', 'confirm_link', $4)`,
				uuid.Must(uuid.NewV7()), id, list, at.Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
		}
		return id
	}
	day := 24 * time.Hour
	due := add("never.clicked@fastmail.com", "form:footer", 40*day, false)
	keep := map[string]uuid.UUID{
		"recent signup":  add("recent.signup@fastmail.com", "form:footer", 10*day, false),
		"added by team":  add("added.by.team@fastmail.com", "manual", 40*day, false),
		"imported":       add("imported.one@fastmail.com", "import:csv", 40*day, false),
		"confirmed once": add("confirmed.once@fastmail.com", "form:footer", 40*day, true),
	}
	svc := &audience.Service{Pool: pool, Q: env.App.Q, Audit: env.App.Audit}
	res, err := svc.RetainWorkspace(ctx, ws, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	erased := func(id uuid.UUID) bool {
		var at *time.Time
		if err := pool.QueryRow(ctx, `SELECT erased_at FROM contacts WHERE id = $1`, id).Scan(&at); err != nil {
			t.Fatal(err)
		}
		return at != nil
	}
	if !erased(due) || res.Erased < 1 {
		t.Fatalf("the signup never confirmed for 40 days wasn't erased (%+v)", res)
	}
	for name, id := range keep {
		if erased(id) {
			t.Errorf("%s was erased", name)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'contacts.erase' AND metadata->>'reason' = 'unconfirmed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("%d audit entries for the purge (%v)", n, err)
	}
}
