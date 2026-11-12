package holdout_test

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
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
	// The demo tracks Acme Merch (M62), so site visitors make a fourth metric.
	if res.Period == nil || res.Verdict == "none" || res.Verdict == "pending" || len(res.Metrics) != 4 || !res.Tracking || res.Holdout.Contacts == 0 {
		t.Fatalf("results: %s", r.Body)
	}
	if r := viewer.Do(http.MethodPost, ws+"/holdout/measure", nil); r.Status != http.StatusAccepted {
		t.Fatalf("measure: %d %s", r.Status, r.Body)
	}
	if r := viewer.Do(http.MethodPost, ws+"/holdout/measure", nil); r.Status != http.StatusTooManyRequests || r.Header.Get("Retry-After") == "" {
		t.Errorf("second measure: %d %v", r.Status, r.Header)
	}
}

// The demo's holdout group (demo.SeedHoldout, as `reputecast seed` adds
// it): measured from the demo's orders, conversions and paid
// subscriptions, marketing added 18% more revenue per contact with 96%
// confidence, and the eight Weekly Digests sent since included the group.
func TestMeasureOnTheDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	tx, err := env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := demo.SeedHoldout(ctx, tx, wsID, time.Now()); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo("aisha@acme.io")
	r := c.Do(http.MethodGet, ws+"/holdout/results", nil)
	if r.Status != http.StatusOK {
		t.Fatalf("results: %d %s", r.Status, r.Body)
	}
	var res holdout.HoldoutResults
	r.JSON(t, &res)
	if res.Verdict != holdout.VerdictAdded || res.Period == nil || res.Period.BasisPoints != 500 || !res.Period.Campaigns || !res.Period.Automations {
		t.Fatalf("results: %s", r.Body)
	}
	var rev *holdout.HoldoutMetric
	for i := range res.Metrics {
		if res.Metrics[i].Key == holdout.MetricRevenue {
			rev = &res.Metrics[i]
		}
	}
	if rev == nil || math.Round(rev.Lift) != 18 || math.Floor(rev.Confidence*100) != 96 || !rev.Significant {
		t.Fatalf("revenue per contact: %+v", rev)
	}
	if share := float64(res.Holdout.Contacts) / float64(res.Holdout.Contacts+res.Marketing.Contacts) * 100; share < 4.5 || share > 5.5 {
		t.Errorf("the holdout group is %.2f%% of the population", share)
	}
	if res.CampaignsIncluded != 8 {
		t.Errorf("%d campaigns included the holdout group, want the 8 digests", res.CampaignsIncluded)
	}
	// Site visitors (M62's tracker is in the demo): each group's count is
	// its contacts seen on a tracked site in the window, and marketing
	// brings more of them to the store.
	if !res.Tracking {
		t.Fatalf("the demo tracks Acme Merch, results say no tracking: %s", r.Body)
	}
	var visitors *holdout.HoldoutMetric
	for i := range res.Metrics {
		if res.Metrics[i].Key == holdout.MetricVisitors {
			visitors = &res.Metrics[i]
		}
	}
	var seed int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT seed FROM holdout_periods WHERE workspace_id = $1 AND ended_at IS NULL`, wsID).Scan(&seed); err != nil {
		t.Fatal(err)
	}
	var heldVisitors, otherVisitors int64
	if err := env.App.Pool.QueryRow(ctx, `
		WITH vis AS (SELECT DISTINCT e.contact_id FROM tracker_events e
			WHERE e.workspace_id = $1 AND e.contact_id IS NOT NULL AND e.occurred_at >= $2 AND e.occurred_at < $3)
		SELECT count(*) FILTER (WHERE rc_holdout_bucket(c.id, $4) < 500), count(*) FILTER (WHERE rc_holdout_bucket(c.id, $4) >= 500)
		FROM vis JOIN contacts c ON c.id = vis.contact_id
		WHERE c.erased_at IS NULL AND c.created_at < $3
		  AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email AND s.created_at < $2)`,
		wsID, res.From, res.To, seed).Scan(&heldVisitors, &otherVisitors); err != nil {
		t.Fatal(err)
	}
	if res.Holdout.Visitors != heldVisitors || res.Marketing.Visitors != otherVisitors || heldVisitors == 0 {
		t.Fatalf("visitors %d / %d, the tracker says %d / %d", res.Holdout.Visitors, res.Marketing.Visitors, heldVisitors, otherVisitors)
	}
	if visitors == nil || !visitors.Tested || visitors.Lift <= 0 || visitors.Marketing <= visitors.Holdout {
		t.Fatalf("site visitors: %+v", visitors)
	}
	// A second run adds nothing: the open period stays as it is.
	tx, err = env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := demo.SeedHoldout(ctx, tx, wsID, time.Now()); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	_ = tx.Commit(ctx)
	var periods int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM holdout_periods WHERE workspace_id = $1`, wsID).Scan(&periods); err != nil || periods != 1 {
		t.Errorf("%d periods after seeding twice (%v)", periods, err)
	}
}

// Review & send's "Include the holdout group": saved in the campaign's
// audience and audited on a draft, refused while the send is scheduled.
func TestIncludeTheHoldoutGroup(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := func(name string) string {
		var v uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT c.id FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = 'acme-newsletter' AND c.name = $1`, name).Scan(&v); err != nil {
			t.Fatal(name, err)
		}
		return v.String()
	}
	draft := id("Flash sale: 48 hours only")
	c.Do(http.MethodPatch, ws+"/campaigns/"+draft+"/send-settings", map[string]any{"includeHoldout": true}).MustStatus(t, http.StatusNoContent)
	var included bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT coalesce((audience->>'include_holdout')::boolean, false) FROM campaigns WHERE id = $1`, draft).Scan(&included); err != nil || !included {
		t.Fatalf("include_holdout %v (%v)", included, err)
	}
	var audits int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'campaign.send_settings_updated' AND target_id = $1 AND (after->>'include_holdout')::boolean`, draft).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("%d audit entries (%v)", audits, err)
	}
	c.Do(http.MethodPatch, ws+"/campaigns/"+draft+"/send-settings", map[string]any{"includeHoldout": false}).MustStatus(t, http.StatusNoContent)
	if err := env.App.Pool.QueryRow(ctx, `SELECT audience ? 'include_holdout' FROM campaigns WHERE id = $1`, draft).Scan(&included); err != nil || included {
		t.Fatalf("include_holdout still set (%v)", err)
	}
	// Weekly Digest #142 is scheduled: who gets it can't change.
	r := c.Do(http.MethodPatch, ws+"/campaigns/"+id("Weekly Digest #142")+"/send-settings", map[string]any{"includeHoldout": false})
	if r.Status != http.StatusConflict || r.Problem(t)["detail"] != "The campaign is scheduled. Unschedule it to change who gets it." {
		t.Fatalf("scheduled: %d %s", r.Status, r.Body)
	}
}

// Without a tracked site there are no site visitors: the results say so
// and leave the metric out.
func TestSiteVisitorsNeedTracking(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE tracker_sites SET deleted_at = now() WHERE workspace_id = $1`, wsID); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do(http.MethodPut, ws+"/holdout", map[string]any{"on": true, "basisPoints": 500, "campaigns": true, "automations": true}).MustStatus(t, http.StatusOK)
	if err := env.App.Holdout().Measure(ctx, wsID); err != nil {
		t.Fatal(err)
	}
	var res holdout.HoldoutResults
	c.Do(http.MethodGet, ws+"/holdout/results", nil).MustStatus(t, http.StatusOK).JSON(t, &res)
	if res.Tracking || res.Holdout.Visitors != 0 || res.Marketing.Visitors != 0 {
		t.Fatalf("no tracked site, results: tracking %v, visitors %d / %d", res.Tracking, res.Holdout.Visitors, res.Marketing.Visitors)
	}
	for _, m := range res.Metrics {
		if m.Key == holdout.MetricVisitors {
			t.Fatalf("site visitors without a tracked site: %+v", m)
		}
	}
	if len(res.Metrics) != 3 {
		t.Fatalf("%d metrics, want the three without site visitors", len(res.Metrics))
	}
}
