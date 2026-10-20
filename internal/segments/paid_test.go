package segments_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Paid subscription conditions (Earn, EarnInAudience): "Trialing now"
// counts the trials with access, "Insider and above" every paid member,
// the top tier only its own, and the generated SQL runs as a SQL segment
// over the paid_members and paid_tiers views. Test paid members never count.
func TestPaidSubscriptionConditions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	one := func(sql string) int64 {
		t.Helper()
		var n int64
		if err := env.App.Pool.QueryRow(ctx, sql).Scan(&n); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}
	const live = `FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id AND c.erased_at IS NULL
		AND NOT EXISTS (SELECT 1 FROM suppressions x WHERE x.workspace_id = c.workspace_id AND x.email = c.email)
		WHERE s.workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter') AND s.mode = 'live' AND s.ended_at IS NULL `
	trials := one(`SELECT count(DISTINCT s.contact_id) ` + live + `AND s.status = 'trialing'`)
	paid := one(`SELECT count(DISTINCT s.contact_id) ` + live + `AND (s.status IN ('trialing','active','complimentary','gift') OR (s.status = 'past_due' AND NOT s.access_paused))`)

	var cat struct {
		Fields []struct {
			Key, Label string
			Options    []struct{ Value, Label string }
		}
	}
	c.Do(http.MethodGet, ws+"/segments/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	tiers := map[string]string{}
	labels := map[string]string{}
	for _, f := range cat.Fields {
		labels[f.Key] = f.Label
		if f.Key == "paid_tier" {
			for _, o := range f.Options {
				tiers[o.Label] = o.Value
			}
		}
	}
	if labels["paid_status"] != "Paid subscription · status" || labels["paid_ltv"] != "Paid subscription · lifetime value" || tiers["Insider and above"] == "" || tiers["Founding member"] == "" {
		t.Fatalf("catalog: %v %v", labels, tiers)
	}

	type preview struct {
		Count        int64
		GeneratedSQL string
	}
	count := func(conds ...map[string]any) preview {
		t.Helper()
		var p preview
		def := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": conds}}}
		c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": def}).MustStatus(t, 200).JSON(t, &p)
		return p
	}
	trialing := map[string]any{"field": "paid_status", "op": "is", "value": "trialing"}
	insider := map[string]any{"field": "paid_tier", "op": "is", "value": tiers["Insider and above"]}
	founding := map[string]any{"field": "paid_tier", "op": "is", "value": tiers["Founding member"]}
	if p := count(trialing, insider); p.Count != trials || trials == 0 {
		t.Errorf("Trialing now: %d, want %d", p.Count, trials)
	}
	if p := count(insider); p.Count != paid {
		t.Errorf("Insider and above: %d, want every paid member, %d", p.Count, paid)
	}
	top := count(founding)
	if top.Count == 0 || top.Count >= paid {
		t.Errorf("Founding member: %d of %d", top.Count, paid)
	}
	none := count(map[string]any{"field": "paid_status", "op": "is", "value": "none"})
	everyone := count(map[string]any{"field": "paid_status", "op": "is_not", "value": "none"})
	if none.Count+everyone.Count == 0 || everyone.Count != paid {
		t.Errorf("no paid access %d, paid access %d (want %d)", none.Count, everyone.Count, paid)
	}
	if p := count(map[string]any{"field": "paid_ltv", "op": "at_least", "value": "100"}); p.Count == 0 || p.Count >= paid {
		t.Errorf("lifetime value at least $100: %d", p.Count)
	}
	if p := count(map[string]any{"field": "paid_ended", "op": "in_last", "amount": 30, "unit": "days"}); p.Count == 0 {
		t.Error("nobody's access ended in the last 30 days")
	}
	if p := count(map[string]any{"field": "paid_period", "op": "is", "value": "year"}, map[string]any{"field": "paid_since", "op": "after", "value": "2026-01-01"}); p.Count == 0 {
		t.Error("no yearly member since this year")
	}
	c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": map[string]any{"groups": []map[string]any{{"match": "all",
		"conditions": []map[string]any{{"field": "paid_tier", "op": "is", "value": "not-a-tier"}}}}}}).MustStatus(t, 422)

	// The generated SQL runs as a SQL segment over the same views.
	r := c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "sql", "sql": strings.TrimSuffix(top.GeneratedSQL, ";")})
	if r.Status != 200 && strings.Contains(string(r.Body), "turned off") {
		t.Skip("SQL segments are turned off on this database")
	}
	var p preview
	r.MustStatus(t, 200).JSON(t, &p)
	if p.Count != top.Count {
		t.Errorf("as SQL: %d, builder %d", p.Count, top.Count)
	}
}
