package segments_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestClickedRuleCountsPeopleOnly: "Clicked in the last N days" and the
// events view's human column count a person's clicks only (review AN-2):
// a scanner's click filed as a bot's, and a click still waiting to settle,
// don't make a contact a clicker. Clicks recorded before clicks were
// classified (no class) count as they always did.
func TestClickedRuleCountsPeopleOnly(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct {
		name, class string
		machine     bool
	}{{"pending", "pending", false}, {"bot", "bot", true}, {"human", "human", false}, {"legacy", "", false}} {
		id := uuid.Must(uuid.NewV7())
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name) VALUES ($1, $2, $3, 'Clicker')`,
			id, wsID, fmt.Sprintf("%s@clicks.example", x.name)); err != nil {
			t.Fatal(err)
		}
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, contact_id, type, occurred_at, machine, class, url)
			VALUES ($1, $2, $3, 'click', $4, $5, $6, 'https://acme.io/docs')`, uuid.Must(uuid.NewV7()), wsID, id, time.Now().Add(-24*time.Hour), x.machine, x.class); err != nil {
			t.Fatal(err)
		}
	}
	preview := func(op string) (int64, string) {
		t.Helper()
		def := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{
			{"field": "email_domain", "op": "is", "value": "clicks.example"},
			{"field": "clicked", "op": op, "amount": 30, "unit": "days"},
		}}}}
		var p struct {
			Count        int64
			GeneratedSQL string
		}
		c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": def}).MustStatus(t, http.StatusOK).JSON(t, &p)
		return p.Count, p.GeneratedSQL
	}
	n, sql := preview("in_last")
	if n != 2 {
		t.Fatalf("clicked in the last 30 days: %d contacts, want the person and the earlier clicker (2)\n%s", n, sql)
	}
	if !strings.Contains(sql, "e.type = 'click' AND e.human = true") {
		t.Fatalf("the generated SQL doesn't say it counts people's clicks:\n%s", sql)
	}
	if n, _ := preview("not_in_last"); n != 2 {
		t.Fatalf("didn't click in the last 30 days: %d contacts, want the scanned and the unsettled one (2)", n)
	}
	// A SQL segment reads the same events view: human is a person's.
	r := c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "sql",
		"sql": "SELECT c.id FROM contacts c WHERE c.email LIKE '%@clicks.example' AND EXISTS (SELECT 1 FROM events e WHERE e.contact_id = c.id AND e.type = 'click' AND e.human)"})
	if r.Status != http.StatusOK && strings.Contains(string(r.Body), "turned off") {
		return // SQL segments are turned off on this database
	}
	var p struct{ Count int64 }
	r.MustStatus(t, http.StatusOK).JSON(t, &p)
	if p.Count != 2 {
		t.Fatalf("SQL segment over events.human: %d contacts, want 2", p.Count)
	}
}
