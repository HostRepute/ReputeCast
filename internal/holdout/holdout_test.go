package holdout_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/holdout"
)

const ws = "/workspaces/acme-newsletter"

// TestBuckets: 100,000 random ids at 500 basis points put 5.0% ± 0.3% in
// the holdout group, and every id held at 500 is held at 1,000 (a larger
// share keeps everyone already held out).
func TestBuckets(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	var at500, at1000, lost int64
	err := env.App.Pool.QueryRow(ctx, `
		WITH ids AS (SELECT gen_random_uuid() AS id FROM generate_series(1, 100000))
		SELECT count(*) FILTER (WHERE rc_holdout_bucket(id, 42) < 500),
		       count(*) FILTER (WHERE rc_holdout_bucket(id, 42) < 1000),
		       count(*) FILTER (WHERE rc_holdout_bucket(id, 42) < 500 AND NOT rc_holdout_bucket(id, 42) < 1000)
		FROM ids`).Scan(&at500, &at1000, &lost)
	if err != nil {
		t.Fatal(err)
	}
	if share := float64(at500) / 1000; share < 4.7 || share > 5.3 {
		t.Errorf("500 bp held %.2f%%", share)
	}
	if share := float64(at1000) / 1000; share < 9.6 || share > 10.4 {
		t.Errorf("1,000 bp held %.2f%%", share)
	}
	if lost != 0 {
		t.Errorf("%d held at 500 bp aren't at 1,000 bp", lost)
	}
	// Another seed draws other people.
	var same int64
	if err := env.App.Pool.QueryRow(ctx, `
		WITH ids AS (SELECT gen_random_uuid() AS id FROM generate_series(1, 20000))
		SELECT count(*) FILTER (WHERE rc_holdout_bucket(id, 42) < 500 AND rc_holdout_bucket(id, 43) < 500) FROM ids`).Scan(&same); err != nil {
		t.Fatal(err)
	}
	if same > 120 { // ~0.25% of 20,000 = 50 by chance
		t.Errorf("seeds 42 and 43 share %d contacts", same)
	}
}

type settings struct {
	Holdout struct {
		On          bool    `json:"on"`
		BasisPoints int     `json:"basisPoints"`
		Percent     float64 `json:"percent"`
		Campaigns   bool    `json:"campaigns"`
		Automations bool    `json:"automations"`
		HeldOut     int64   `json:"heldOut"`
		Measured    bool    `json:"measured"`
	} `json:"holdout"`
}

func TestSettingsAndResults(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	// On, then off, then on again with the same people.
	if r := c.Do(http.MethodPut, ws+"/holdout", map[string]any{"on": true, "basisPoints": 500, "campaigns": true, "automations": true}); r.Status != http.StatusOK {
		t.Fatalf("first on: %d %s", r.Status, r.Body)
	}
	r := c.Do(http.MethodPut, ws+"/holdout", map[string]any{"on": false, "basisPoints": 500, "campaigns": true, "automations": true})
	if r.Status != http.StatusOK {
		t.Fatalf("off: %d %s", r.Status, r.Body)
	}
	var v settings
	r.JSON(t, &v)
	if v.Holdout.On || !v.Holdout.Measured {
		t.Fatalf("off: %+v", v.Holdout)
	}
	r = c.Do(http.MethodPut, ws+"/holdout", map[string]any{"on": true, "basisPoints": 1000, "campaigns": true, "automations": false})
	if r.Status != http.StatusOK {
		t.Fatalf("on: %d %s", r.Status, r.Body)
	}
	r.JSON(t, &v)
	if !v.Holdout.On || v.Holdout.Percent != 10 || v.Holdout.Automations || v.Holdout.HeldOut == 0 {
		t.Fatalf("on: %+v", v.Holdout)
	}
	var seeds []int64
	rows, err := env.App.Pool.Query(ctx, `SELECT p.seed FROM holdout_periods p JOIN workspaces w ON w.id = p.workspace_id WHERE w.slug = 'acme-newsletter' ORDER BY p.started_at DESC, p.id DESC`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s int64
		_ = rows.Scan(&s)
		seeds = append(seeds, s)
	}
	rows.Close()
	if len(seeds) < 2 || seeds[0] != seeds[1] {
		t.Errorf("turned on again with another seed: %v", seeds)
	}
	// Validation.
	if r := c.Do(http.MethodPut, ws+"/holdout", map[string]any{"on": true, "basisPoints": 5, "campaigns": true, "automations": true}); r.Status != http.StatusUnprocessableEntity {
		t.Errorf("5 bp: %d", r.Status)
	}
	if r := c.Do(http.MethodPut, ws+"/holdout", map[string]any{"on": true, "basisPoints": 500, "campaigns": false, "automations": false}); r.Status != http.StatusUnprocessableEntity {
		t.Errorf("no scope: %d", r.Status)
	}
	// Start over: a new seed, audited with the reason.
	if r := c.Do(http.MethodPost, ws+"/holdout/restart", map[string]any{"reason": "New season"}); r.Status != http.StatusOK {
		t.Fatalf("restart: %d %s", r.Status, r.Body)
	}
	var latest int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT seed FROM holdout_periods p JOIN workspaces w ON w.id = p.workspace_id WHERE w.slug = 'acme-newsletter' AND ended_at IS NULL`).Scan(&latest); err != nil {
		t.Fatal(err)
	}
	if latest == seeds[0] {
		t.Error("start over kept the seed")
	}
	var audits int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('holdout.updated', 'holdout.restarted')`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits < 3 {
		t.Errorf("%d audit entries", audits)
	}
	// The Viewer reads results but can't change the settings.
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	if r := viewer.Do(http.MethodPut, ws+"/holdout", map[string]any{"on": false, "basisPoints": 500, "campaigns": true, "automations": true}); r.Status != http.StatusForbidden {
		t.Errorf("viewer: %d", r.Status)
	}
	// Measure, read, and refuse a second recompute within 10 minutes.
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Holdout().Measure(ctx, wsID); err != nil {
		t.Fatal(err)
	}
	r = viewer.Do(http.MethodGet, ws+"/holdout/results", nil)
	if r.Status != http.StatusOK {
		t.Fatalf("results: %d %s", r.Status, r.Body)
	}
	var res holdout.HoldoutResults
	r.JSON(t, &res)
	if res.Period == nil || res.Verdict == "none" || res.Verdict == "pending" || len(res.Metrics) != 3 || res.Holdout.Contacts == 0 {
		t.Fatalf("results: %s", r.Body)
	}
	if r := viewer.Do(http.MethodPost, ws+"/holdout/measure", nil); r.Status != http.StatusAccepted {
		t.Fatalf("measure: %d %s", r.Status, r.Body)
	}
	if r := viewer.Do(http.MethodPost, ws+"/holdout/measure", nil); r.Status != http.StatusTooManyRequests || r.Header.Get("Retry-After") == "" {
		t.Errorf("second measure: %d %v", r.Status, r.Header)
	}
}
